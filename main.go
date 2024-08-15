package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/chuhlomin/slack-export/pkg/structs"
	"github.com/jessevdk/go-flags"
)

type config struct {
	Channel         string `env:"CHANNEL" long:"channel" description:"Slack channel ID" required:"true"`
	APIToken        string `env:"API_TOKEN" long:"api-token" description:"Slack API Token"`
	AppClientID     string `env:"APP_CLIENT_ID" long:"app-client-id" description:"Slack App Client ID"`
	AppClientSecret string `env:"APP_CLIENT_SECRET" long:"app-client-secret" description:"Slack App Client Secret"`
	Address         string `env:"ADDRESS" long:"address" description:"Server address" default:"localhost"`
	Port            string `env:"PORT" long:"port" description:"Server port" default:"8079"`
}

var cfg config

func main() {
	// log.Println("Starting...")
	if err := run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
	// log.Println("Done")
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

	channelInfo, err := c.GetChannelInfo(cfg.Channel)
	if err != nil {
		return fmt.Errorf("could not get channel info: %v", err)
	}

	msgs, err := c.GetMessages(cfg.Channel)
	if err != nil {
		return fmt.Errorf("could not get messages: %v", err)
	}

	files, err := c.DownloadFiles(cfg.Channel)
	if err != nil {
		return fmt.Errorf("could not download files: %v", err)
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

	err = os.WriteFile(cfg.Channel+".json", content, 0644)
	if err != nil {
		return fmt.Errorf("could not write messages to file: %v", err)
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
