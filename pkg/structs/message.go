// Package structs contains the data structures shared between
// the main app that exports Slack messages and the JSON to HTML converter.
package structs

import (
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/slack-go/slack"
)

// Message is a wrapper for slack.Message with replies.
type Message struct {
	slack.Message
	Replies []slack.Message `json:"replies,omitempty"`
}

func (m *Message) SameContext(m2 Message) bool {
	if m.User != m2.User {
		return false
	}

	// timestamp difference is less than 15 minutes
	mTs := m.Timestamp[:strings.Index(m.Timestamp, ".")]
	m2Ts := m2.Timestamp[:strings.Index(m2.Timestamp, ".")]
	mSec, err := strconv.ParseInt(mTs, 10, 64)
	if err != nil {
		log.Printf("could not parse time: %v", err)
		return false
	}

	m2Sec, err := strconv.ParseInt(m2Ts, 10, 64)
	if err != nil {
		log.Printf("could not parse time: %v", err)
		return false
	}

	if time.Duration(math.Abs(float64(mSec-m2Sec))) > 15*time.Minute {
		return false
	}

	return true
}

// Data struct used to marshal/unmarshal JSON data.
type Data struct {
	Channel  slack.Channel `json:"channel"`
	Messages []Message     `json:"messages"`
	Users    []slack.User  `json:"users"`
}
