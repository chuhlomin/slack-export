package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"soelke.de/slack-channel-export/src/clients"
)

func main() {
	currentWD, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error getting current working directory. Exiting...\nError: %v", err)
	}

	err = godotenv.Load(fmt.Sprintf("%s/%s", currentWD, ".env"))
	if err != nil {
		log.Fatalf("Error loading .env file. Exiting...\nError: %v", err)
	}

	slackChannelId, exists := os.LookupEnv("SLACK_CHANNEL_ID")
	if !exists {
		log.Fatal("SLACK_CHANNEL_ID is not set in .env file. Exiting...")
	}

	slackToken, exists := os.LookupEnv("SLACK_API_TOKEN")
	if !exists {
		log.Fatal("SLACK_API_TOKEN is not set in .env file. Exiting...")
	}

	slackClient := clients.NewSlack(slackToken)
	msgs, err := slackClient.GetMessages(slackChannelId)
	if err != nil {
		log.Fatalf("Could not get messages from Slack: %v", err)
	}

	// Save messages to a file
	content, err := json.Marshal(msgs)
	if err != nil {
		log.Fatal(err)
	}

	err = os.WriteFile("messages.json", content, 0644)
	if err != nil {
		log.Fatal(err)
	}
}
