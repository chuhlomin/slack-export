package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/jessevdk/go-flags"

	"github.com/chuhlomin/slack-export/pkg/rand"
	"github.com/chuhlomin/slack-export/pkg/server"
	"github.com/chuhlomin/slack-export/pkg/slack"
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

	c := slack.NewClient(cfg.AppClientID, cfg.AppClientSecret)

	if cfg.APIToken == "" {
		err := getToken(c)
		if err != nil {
			return fmt.Errorf("could not get token: %v", err)
		}
	} else {
		c.SetToken(cfg.APIToken)
	}

	msgs, err := c.GetMessages(cfg.Channel)
	if err != nil {
		return fmt.Errorf("could not get messages: %v", err)
	}

	// Save messages to a file
	content, err := json.Marshal(msgs)
	if err != nil {
		return fmt.Errorf("could not marshal messages: %v", err)
	}

	err = os.WriteFile(cfg.Channel+".json", content, 0644)
	if err != nil {
		return fmt.Errorf("could not write messages to file: %v", err)
	}

	// Save users info
	users, err := c.GetUsers()
	if err != nil {
		return fmt.Errorf("could not get users: %v", err)
	}

	usersContent, err := json.Marshal(users)
	if err != nil {
		return fmt.Errorf("could not marshal users: %v", err)
	}

	err = os.WriteFile(cfg.Channel+"_users.json", usersContent, 0644)
	if err != nil {
		return fmt.Errorf("could not write users to file: %v", err)
	}

	return nil
}

func getToken(c *slack.Client) error {
	state := rand.RandStringBytesMaskImprSrcSB(16)
	code := make(chan string)

	s := server.NewServer(cfg.Address, cfg.Port, state, code)

	// log.Println("Starting server on " + cfg.Address + ":" + cfg.Port)
	go s.Start()

	log.Printf("App authorization URL: %s", c.GetAuthorizeURL(state))

	err := c.GetToken(<-code)
	if err != nil {
		return err
	}

	return nil
}
