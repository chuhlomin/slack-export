package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/chuhlomin/slack-export/pkg/structs"
	"github.com/jessevdk/go-flags"
)

type config struct {
	Channel         string `env:"CHANNEL" long:"channel" description:"Slack channel ID; pass \"public\" to export all public channels" required:"true"`
	Output          string `env:"OUTPUT" long:"output" description:"Output directory" default:"output"`
	APIToken        string `env:"API_TOKEN" long:"api-token" description:"Slack API Token"`
	AppClientID     string `env:"APP_CLIENT_ID" long:"app-client-id" description:"Slack App Client ID"`
	AppClientSecret string `env:"APP_CLIENT_SECRET" long:"app-client-secret" description:"Slack App Client Secret"`
	Address         string `env:"ADDRESS" long:"address" description:"Server address" default:"localhost"`
	Port            string `env:"PORT" long:"port" description:"Server port" default:"8079"`
	DownloadFiles   bool   `env:"DOWNLOAD_FILES" long:"download-files" description:"Download files"`
	DownloadAvatars bool   `env:"DOWNLOAD_AVATARS" long:"download-avatars" description:"Download avatars"`
	SkipArchived    bool   `env:"SKIP_ARCHIVED" long:"skip-archived" description:"Skip archived channels"`
}

var cfg config

func main() {
	if err := run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func run() error {
	if _, err := flags.Parse(&cfg); err != nil {
		return fmt.Errorf("could not parse flags: %v", err)
	}

	c := NewSlackClient(cfg.AppClientID, cfg.AppClientSecret)

	if cfg.APIToken == "" {
		err := getToken(c)
		if err != nil {
			return fmt.Errorf("could not get token: %v", err)
		}
	} else {
		c.SetToken(cfg.APIToken)
	}

	// make sure the output directory exists
	if err := os.MkdirAll(cfg.Output, 0755); err != nil {
		return fmt.Errorf("could not create output directory: %v", err)
	}

	switch cfg.Channel {
	case "public":
		err := exportPublicChannels(c)
		if err != nil {
			return fmt.Errorf("could not export public channels: %v", err)
		}
	default:
		err := exportChannel(c, cfg.Channel)
		if err != nil {
			return fmt.Errorf("could not export channel %q: %v", cfg.Channel, err)
		}
	}

	if cfg.DownloadAvatars {
		log.Println("Downloading avatars")
		if err := downloadAvatars(c); err != nil {
			return fmt.Errorf("could not download avatars: %v", err)
		}
	}

	return nil
}

func getToken(c *SlackClient) error {
	state := RandStringBytesMaskImprSrcSB(16)
	code := make(chan string)

	s := NewServer(cfg.Address, cfg.Port, state, code)

	go func() {
		err := s.Start()
		if err != nil && err != http.ErrServerClosed {
			log.Printf("could not start server: %v", err)
		}
	}()

	defer func() {
		err := s.Stop()
		if err != nil {
			log.Printf("could not stop server: %v", err)
		}
	}()

	log.Printf("App authorization URL: %s", c.GetAuthorizeURL(state))

	err := c.GetToken(<-code)
	if err != nil {
		return err
	}

	return nil
}

func exportChannel(c *SlackClient, channelID string) error {
	channelInfo, err := c.GetChannelInfo(channelID)
	if err != nil {
		return fmt.Errorf("could not get channel info: %v", err)
	}

	if channelInfo.IsArchived && cfg.SkipArchived {
		log.Printf("Channel %q is archived, skipping", channelInfo.Name)
		return nil
	}

	outputFilename := filepath.Join(cfg.Output, channelID+".json")

	// check if the file already exists
	if _, err := os.Stat(outputFilename); err == nil {
		log.Printf("File %q already exists", outputFilename)
		// read the file to pull users
		data, err := os.ReadFile(outputFilename)
		if err != nil {
			return fmt.Errorf("could not read file: %v", err)
		}

		var d structs.Data
		if err = json.Unmarshal(data, &d); err != nil {
			return fmt.Errorf("could not unmarshal data: %v", err)
		}

		for id, user := range d.Users {
			c.UsersCache[id] = user
		}
		return nil
	}

	msgs, err := c.GetMessages(channelID)
	if err != nil {
		return fmt.Errorf("could not get messages: %v", err)
	}

	var files map[string]string
	if cfg.DownloadFiles {
		files, err = c.DownloadFiles(channelID)
		if err != nil {
			return fmt.Errorf("could not download files: %v", err)
		}
	}

	users, err := c.GetUsers()
	if err != nil {
		return fmt.Errorf("could not get users: %v", err)
	}

	data := structs.Data{
		Channel:  *channelInfo,
		Messages: msgs,
		Users:    users,
		Files:    files,
	}

	// Save to a file
	content, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("could not marshal messages: %v", err)
	}

	if err = os.WriteFile(outputFilename, content, 0644); err != nil {
		return fmt.Errorf("could not write messages to file: %v", err)
	}

	return nil
}

func exportPublicChannels(c *SlackClient) error {
	channels, err := c.GetPublicChannels()
	if err != nil {
		return fmt.Errorf("could not get public channels: %v", err)
	}

	for _, channel := range channels {
		log.Printf("Exporting channel %q (%s)", channel.Name, channel.ID)
		err := exportChannel(c, channel.ID)
		if err != nil {
			return fmt.Errorf("could not export channel %q: %v", channel.Name, err)
		}
	}

	return nil
}

func downloadAvatars(c *SlackClient) error {
	err := os.MkdirAll(filepath.Join(cfg.Output, "avatars"), 0755)
	if err != nil {
		return fmt.Errorf("could not create avatars directory: %v", err)
	}

	for _, user := range c.UsersCache {
		if user.Profile.Image512 != "" {
			err := downloadFile(user.ID, user.Profile.Image512, cfg.Output)
			if err != nil {
				return fmt.Errorf("could not download avatar: %v", err)
			}
		}
	}

	return nil
}

func downloadFile(id, url, output string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("could not send request: %v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	filename := filepath.Join(output, "avatars", id+".png")
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("could not create file: %v", err)
	}

	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("could not write file: %v", err)
	}

	return nil
}
