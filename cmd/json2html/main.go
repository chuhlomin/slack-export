package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"html/template"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "embed"

	"github.com/enescakir/emoji"
	"github.com/jessevdk/go-flags"
	"github.com/slack-go/slack"

	"github.com/chuhlomin/slack-export/pkg/structs"
)

type config struct {
	Input        string `long:"input" short:"i" description:"Input JSON file or directory" required:"true"`
	Output       string `long:"output" short:"o" description:"Output HTML file or directory" required:"true"`
	EmojiDir     string `long:"emoji" description:"Directory with emoji"`
	SkipArchived bool   `long:"skip-archived" description:"Skip archived channels"`
}

var errChannelIsArchived = fmt.Errorf("channel is archived")

//go:embed template.html
var tmpl string

//go:embed index.html
var index string

var (
	cfg config
	fm  = template.FuncMap{
		"lookupUser": lookupUser,
		"username":   username,
		"avatar": func(user *slack.User) string {
			if user == nil {
				return ""
			}
			return filepath.Join("avatars", user.ID+".png")
			// return user.Profile.Image512
		},
		"sameMessage": func(a, b structs.Message) bool {
			return a.SameContext(b)
		},
		"usersList": func(ids []string, users map[string]*slack.User) string {
			names := make([]string, 0, len(ids))

			for _, id := range ids {
				names = append(names, username((lookupUser(id, users))))
			}

			return strings.Join(names, ", ")
		},
		"sameSlackMessage": func(a, b slack.Message) bool {
			ma := structs.Message{Message: a}
			return ma.SameContext(structs.Message{Message: b})
		},
		"formatTime": func(t string) string {
			dotIndex := strings.Index(t, ".")
			if dotIndex == -1 {
				return t
			}
			unixPart := t[:dotIndex]
			sec, err := strconv.ParseInt(unixPart, 10, 64)
			if err != nil {
				log.Printf("could not parse time: %v", err)
				return t
			}

			return time.Unix(sec, 0).Format(time.ANSIC)
		},
		"emoji":   emojiParse,
		"replace": strings.ReplaceAll,
		"format": func(blocks slack.Blocks, users map[string]*slack.User) template.HTML {
			sb := &strings.Builder{}
			for _, block := range blocks.BlockSet {
				switch block.BlockType() {
				case slack.MBTRichText:
					sb.WriteString(
						processRichTextElements(block.(*slack.RichTextBlock).Elements, users),
					)
				}
			}

			return template.HTML(sb.String()) // #nosec G203
		},
		"attachment": func(file slack.File, files map[string]string, channel slack.Channel) template.HTML {
			filename, ok := files[file.ID]
			if !ok {
				url := file.URLPrivateDownload
				if url == "" {
					url = file.URLPrivate
				}
				return template.HTML(fmt.Sprintf("<a href=%q>%s</a>", url, file.Title)) // #nosec G203
			}

			// url-encode filename (account for \u202f symbol)
			filename = url.PathEscape(filename)

			switch file.Filetype {
			case "png", "jpg", "gif":
				w, h := maxLength(file.OriginalW, file.OriginalH, 550, 550)
				return template.HTML( // #nosec G203
					fmt.Sprintf(
						"<img loading=\"lazy\" src=%q alt=%q class=\"attachment\" width=\"%d\" height=\"%d\"/>",
						filepath.Join(channel.ID, file.ID+"-"+filename),
						file.Title,
						w, h,
					),
				)
			case "mov", "mp4":
				return template.HTML( // #nosec G203
					fmt.Sprintf(
						"<video controls preload=\"none\" src=%q alt=%q class=\"attachment\"/>",
						filepath.Join(channel.ID, file.ID+"-"+filename),
						file.Title,
					),
				)

			default:
				return template.HTML( // #nosec G203
					fmt.Sprintf(
						"<a href=%q download=%q>%s</a>",
						filepath.Join(channel.ID, file.ID+"-"+filename),
						file.Name,
						file.Title,
					),
				)
			}
		},
	}
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

var slackEmoji emojiMap

func run() error {
	if _, err := flags.Parse(&cfg); err != nil {
		return fmt.Errorf("could not parse flags: %w", err)
	}

	if cfg.EmojiDir != "" {
		var err error
		slackEmoji, err = loadSlackEmoji(filepath.Join(cfg.EmojiDir, "emoji.json"))
		if err != nil {
			return fmt.Errorf("could not load emoji: %w", err)
		}
	}

	t, err := template.New("template").Funcs(fm).Parse(tmpl)
	if err != nil {
		return fmt.Errorf("could not parse template: %w", err)
	}

	// check if input is a file or a directory
	info, err := os.Stat(cfg.Input)
	if err != nil {
		return fmt.Errorf("could not get file info: %w", err)
	}

	if !info.IsDir() {
		_, err := processFile(cfg.Input, cfg.Output, t)
		if err != nil {
			return fmt.Errorf("could not process file %q: %w", cfg.Input, err)
		}
		return nil
	}

	return processDirectory(cfg.Input, cfg.Output, t)
}

func processDirectory(input, output string, t *template.Template) error {
	it, err := template.New("index").Funcs(fm).Parse(index)
	if err != nil {
		return fmt.Errorf("could not parse index template: %w", err)
	}

	if err := os.MkdirAll(output, 0o755); err != nil {
		return fmt.Errorf("could not create output directory: %w", err)
	}

	files, err := os.ReadDir(input)
	if err != nil {
		return fmt.Errorf("could not read directory: %w", err)
	}

	var channels []*slack.Channel

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if filepath.Ext(file.Name()) != ".json" {
			continue
		}

		outputFilename := strings.TrimSuffix(file.Name(), ".json") + ".html"

		log.Printf("Processing file %q", file.Name())
		channel, err := processFile(
			filepath.Join(input, file.Name()),
			filepath.Join(output, outputFilename),
			t,
		)
		if err != nil {
			if errors.Is(err, errChannelIsArchived) {
				log.Printf("Channel is archived, skipping")
				continue
			}

			return fmt.Errorf("could not process file %q: %w", file.Name(), err)
		}

		channels = append(channels, channel)
	}

	log.Printf("Generating index")
	return generateIndex(output, channels, it)
}

func processFile(input, output string, t *template.Template) (*slack.Channel, error) {
	var data structs.Data
	content, err := os.ReadFile(input)
	if err != nil {
		return nil, fmt.Errorf("could not read file: %w", err)
	}

	if err := json.Unmarshal(content, &data); err != nil {
		return nil, fmt.Errorf("could not unmarshal messages: %w", err)
	}

	if data.Channel.IsArchived && cfg.SkipArchived {
		return nil, errChannelIsArchived
	}

	o, err := os.Create(output)
	if err != nil {
		return nil, fmt.Errorf("could not create file: %w", err)
	}

	slices.Reverse(data.Messages)

	if err := t.Execute(o, data); err != nil {
		return nil, fmt.Errorf("could not execute template: %w", err)
	}

	return &data.Channel, nil
}

func generateIndex(output string, channels []*slack.Channel, t *template.Template) error {
	o, err := os.Create(filepath.Join(output, "index.html"))
	if err != nil {
		return fmt.Errorf("could not create index file: %w", err)
	}

	// sort alphabetically
	sort.Slice(channels, func(i, j int) bool {
		return channels[i].Name < channels[j].Name
	})

	if err := t.Execute(o, struct {
		Channels []*slack.Channel
	}{
		Channels: channels,
	}); err != nil {
		return fmt.Errorf("could not execute index template: %w", err)
	}

	return nil
}

func lookupUser(id string, users map[string]*slack.User) *slack.User {
	if id == "" {
		return nil
	}

	if user, ok := users[id]; ok {
		return user
	}

	log.Printf("User not found: %s", id)
	return nil
}

func username(user *slack.User) string {
	if user == nil {
		return "unknown"
	}

	return first(
		user.Profile.RealNameNormalized,
		user.RealName,
		user.Profile.DisplayNameNormalized,
		user.Name,
	)
}

var emojiSkinTone = regexp.MustCompile(`:skin-tone-(\d)`)

func emojiParse(s string) template.HTML {
	if emojiSkinTone.MatchString(s) {
		matches := emojiSkinTone.FindStringSubmatch(s)
		tone := matches[1]
		suffix := ""

		switch tone {
		case "2":
			suffix = emoji.Light.String()
		case "3":
			suffix = emoji.MediumLight.String()
		case "4":
			suffix = emoji.Medium.String()
		case "5":
			suffix = emoji.MediumDark.String()
		case "6":
			suffix = emoji.Dark.String()
		}

		// remove skin tone suffix
		s = strings.Split(s, "::skin-tone-")[0]
		return template.HTML(emoji.Parse(":"+s+":") + suffix) // #nosec G203
	}

	alias, filename := slackEmoji.Get(s)
	if alias != "" {
		return template.HTML(emoji.Parse(":" + alias + ":")) // #nosec G203
	}

	if filename != "" {
		return template.HTML( // #nosec G203
			fmt.Sprintf("<img class=\"emoji\" src=\"emoji/%s\" alt=\":%s:\" />", filename, s),
		)
	}

	return template.HTML(emoji.Parse(":" + s + ":")) // #nosec G203
}

func first(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}

	return ""
}

func processRichTextElements(
	elements []slack.RichTextElement,
	users map[string]*slack.User,
	transforms ...func(string) string,
) string {
	result := strings.Builder{}

	for _, element := range elements {
		sb := strings.Builder{}
		switch element.RichTextElementType() {
		case slack.RTESection:
			sb.WriteString(
				processRichTextSectionElements(element.(*slack.RichTextSection).Elements, users),
			)
		case slack.RTEQuote:
			sb.WriteString(
				fmt.Sprintf(
					"<blockquote>%s</blockquote>",
					processRichTextSectionElements(element.(*slack.RichTextQuote).Elements, users),
				),
			)
		case slack.RTEPreformatted:
			sb.WriteString("<pre>")
			for _, rtEelement := range element.(*slack.RichTextPreformatted).Elements {
				switch rtEelement.RichTextSectionElementType() {
				case slack.RTSEText:
					te, ok := rtEelement.(*slack.RichTextSectionTextElement)
					if !ok {
						log.Printf("could not cast to RichTextSectionTextElement")
						continue
					}
					text := html.EscapeString(te.Text)
					sb.WriteString(text)
				case slack.RTSELink:
					if rtEelement.(*slack.RichTextSectionLinkElement).Text != "" {
						sb.WriteString(fmt.Sprintf("<a href=%q>%s</a>", rtEelement.(*slack.RichTextSectionLinkElement).URL, rtEelement.(*slack.RichTextSectionLinkElement).Text))
					} else {
						sb.WriteString(fmt.Sprintf("<a href=%q>%s</a>", rtEelement.(*slack.RichTextSectionLinkElement).URL, rtEelement.(*slack.RichTextSectionLinkElement).URL))
					}
				}
			}
			sb.WriteString("</pre>")
		case slack.RTEList:
			var tag string
			switch element.(*slack.RichTextList).Style {
			case slack.RTEListBullet:
				tag = "ul"
			case slack.RTEListOrdered:
				tag = "ol"
			}

			sb.WriteString(fmt.Sprintf("<%s>", tag))

			sb.WriteString(
				processRichTextElements(
					element.(*slack.RichTextList).Elements,
					users,
					func(s string) string {
						return fmt.Sprintf("<li>%s</li>", s)
					},
				),
			)

			sb.WriteString(fmt.Sprintf("</%s>", tag))
		}

		if len(transforms) > 0 {
			for _, transform := range transforms {
				result.WriteString(transform(sb.String()))
			}
		} else {
			result.WriteString(sb.String())
		}
	}

	return result.String()
}

func processRichTextSectionElements(elements []slack.RichTextSectionElement, users map[string]*slack.User) string {
	sb := strings.Builder{}
	var code bool

	for _, rtEelement := range elements {
		switch rtEelement.RichTextSectionElementType() {
		case slack.RTSEText:
			te, ok := rtEelement.(*slack.RichTextSectionTextElement)
			if !ok {
				log.Printf("could not cast to RichTextSectionTextElement")
				continue
			}
			text := html.EscapeString(te.Text)
			text = strings.ReplaceAll(text, "\n", "<br>")

			if code && (te.Style == nil || !te.Style.Code) {
				code = false
				sb.WriteString("</code>")
			}

			if te.Style != nil {
				if te.Style.Bold {
					text = fmt.Sprintf("<b>%s</b>", text)
				}
				if te.Style.Italic {
					text = fmt.Sprintf("<i>%s</i>", text)
				}
				if te.Style.Strike {
					text = fmt.Sprintf("<s>%s</s>", text)
				}
				if te.Style.Code {
					if !code {
						code = true
						text = fmt.Sprintf("<code>%s", text)
					}
				}
			}

			sb.WriteString(text)
		case slack.RTSEUser:
			sb.WriteString(
				"<span class=\"user\">" +
					username(lookupUser(rtEelement.(*slack.RichTextSectionUserElement).UserID, users)) +
					"</span>",
			)
		case slack.RTSEEmoji:
			sb.WriteString(
				string(emojiParse(rtEelement.(*slack.RichTextSectionEmojiElement).Name)),
			)
		case slack.RTSELink:
			if rtEelement.(*slack.RichTextSectionLinkElement).Text != "" {
				sb.WriteString(fmt.Sprintf("<a href=%q>%s</a>", rtEelement.(*slack.RichTextSectionLinkElement).URL, rtEelement.(*slack.RichTextSectionLinkElement).Text))
			} else {
				sb.WriteString(fmt.Sprintf("<a href=%q>%s</a>", rtEelement.(*slack.RichTextSectionLinkElement).URL, rtEelement.(*slack.RichTextSectionLinkElement).URL))
			}
		}
	}

	if code {
		sb.WriteString("</code>")
	}

	return sb.String()
}

func maxLength(w, h, maxW, maxH int) (width, height int) {
	if w > maxW {
		h = h * maxW / w
		w = maxW
	}

	if h > maxH {
		w = w * maxH / h
		h = maxH
	}

	return w, h
}
