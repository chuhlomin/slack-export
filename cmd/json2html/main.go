package main

import (
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"log"
	"os"
	"slices"
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
	Input  string `long:"input" description:"Input JSON file" required:"true"`
	Output string `long:"output" description:"Output HTML file" required:"true"`
}

//go:embed template.html
var tmpl string
var (
	cfg config
	fm  = template.FuncMap{
		"lookupUser": lookupUser,
		"username":   username,
		"avatar": func(user slack.User) string {
			return user.Profile.Image512
		},
		"sameMessage": func(a, b structs.Message) bool {
			return a.SameContext(b)
		},
		"sameSlackMessage": func(a, b slack.Message) bool {
			ma := structs.Message{Message: a}
			return ma.SameContext(structs.Message{Message: b})
		},
		"formatTime": func(t string) string {
			unixPart := t[:strings.Index(t, ".")]
			sec, err := strconv.ParseInt(unixPart, 10, 64)
			if err != nil {
				log.Printf("could not parse time: %v", err)
				return t
			}

			return time.Unix(sec, 0).Format(time.ANSIC)
		},
		"emoji": emojiParse,
		"format": func(blocks slack.Blocks, users []slack.User) template.HTML {
			sb := &strings.Builder{}
			for _, block := range blocks.BlockSet {
				switch block.BlockType() {
				case slack.MBTRichText:
					sb.WriteString(
						processRichTextElements(block.(*slack.RichTextBlock).Elements, users),
					)
				}
			}

			return template.HTML(sb.String())
		},
	}
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func run() error {
	if _, err := flags.Parse(&cfg); err != nil {
		return fmt.Errorf("could not parse flags: %v", err)
	}

	var data structs.Data
	content, err := os.ReadFile(cfg.Input)
	if err != nil {
		return fmt.Errorf("could not read file: %v", err)
	}

	if err := json.Unmarshal(content, &data); err != nil {
		return fmt.Errorf("could not unmarshal messages: %v", err)
	}

	t, err := template.New("template").Funcs(fm).Parse(tmpl)
	if err != nil {
		return fmt.Errorf("could not parse template: %v", err)
	}

	o, err := os.Create(cfg.Output)
	if err != nil {
		return fmt.Errorf("could not create file: %v", err)
	}

	slices.Reverse(data.Messages)

	if err := t.Execute(o, data); err != nil {
		return fmt.Errorf("could not execute template: %v", err)
	}

	return nil
}

func lookupUser(id string, users []slack.User) slack.User {
	for _, user := range users {
		if user.ID == id {
			return user
		}
	}

	log.Printf("User not found: %s", id)
	return slack.User{}
}

func username(user slack.User) string {
	return first(
		user.Profile.RealNameNormalized,
		user.RealName,
		user.Profile.DisplayNameNormalized,
		user.Name,
	)
}

func emojiParse(s string) string {
	return emoji.Parse(":" + s + ":")
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
	users []slack.User,
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
					te := rtEelement.(*slack.RichTextSectionTextElement)
					text := html.EscapeString(te.Text)
					sb.WriteString(text)
				case slack.RTSELink:
					if rtEelement.(*slack.RichTextSectionLinkElement).Text != "" {
						sb.WriteString(fmt.Sprintf("<a href=\"%s\">%s</a>", rtEelement.(*slack.RichTextSectionLinkElement).URL, rtEelement.(*slack.RichTextSectionLinkElement).Text))
					} else {
						sb.WriteString(fmt.Sprintf("<a href=\"%s\">%s</a>", rtEelement.(*slack.RichTextSectionLinkElement).URL, rtEelement.(*slack.RichTextSectionLinkElement).URL))
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

func processRichTextSectionElements(elements []slack.RichTextSectionElement, users []slack.User) string {
	sb := strings.Builder{}
	var code bool

	for _, rtEelement := range elements {
		switch rtEelement.RichTextSectionElementType() {
		case slack.RTSEText:
			te := rtEelement.(*slack.RichTextSectionTextElement)
			text := html.EscapeString(te.Text)

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
					} else {
						text = fmt.Sprintf("%s", text)
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
			sb.WriteString(emojiParse(rtEelement.(*slack.RichTextSectionEmojiElement).Name))
		case slack.RTSELink:
			if rtEelement.(*slack.RichTextSectionLinkElement).Text != "" {
				sb.WriteString(fmt.Sprintf("<a href=\"%s\">%s</a>", rtEelement.(*slack.RichTextSectionLinkElement).URL, rtEelement.(*slack.RichTextSectionLinkElement).Text))
			} else {
				sb.WriteString(fmt.Sprintf("<a href=\"%s\">%s</a>", rtEelement.(*slack.RichTextSectionLinkElement).URL, rtEelement.(*slack.RichTextSectionLinkElement).URL))
			}
		}
	}

	if code {
		sb.WriteString("</code>")
	}

	return sb.String()
}
