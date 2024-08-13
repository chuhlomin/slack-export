package structs

import "github.com/slack-go/slack"

type Message struct {
	Message slack.Message
	Replies []Message `json:"replies,omitempty"`
}
