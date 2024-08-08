package clients

import (
	"errors"
	"fmt"

	"github.com/slack-go/slack"
	"soelke.de/slack-channel-export/src/models"
	"soelke.de/slack-channel-export/src/utils"
)

type Client struct {
	api *slack.Client
}

func NewSlack(token string) *Client {
	return &Client{
		api: slack.New(token, slack.OptionDebug(true)),
	}
}

func (c *Client) GetMessages(channel string) ([]models.Message, error) {
	if channel == "" {
		return nil, errors.New("argument 'channel' is required")
	}

	var allMessages []slack.Message

	cursor := ""
	for {
		resp, err := c.api.GetConversationHistory(&slack.GetConversationHistoryParameters{
			ChannelID: channel,
			Limit:     100,
			Cursor:    cursor,
		})
		if err != nil {
			return nil, err
		}

		allMessages = append(allMessages, resp.Messages...)

		if resp.ResponseMetaData.NextCursor == "" {
			break
		}

		cursor = resp.ResponseMetaData.NextCursor
	}

	var convertedMessages []models.Message
	for _, msg := range allMessages {
		var replies []models.Message
		var err error

		if msg.ReplyCount > 0 {
			replies, err = c.getReplies(channel, msg.Timestamp)
			if err != nil {
				fmt.Printf("Could not get replies for message '%s': %v", msg.Timestamp, err)
			}
		}

		convertedMsg := c.convertToMsg(msg)
		convertedMsg.Replies = replies
		convertedMessages = append(convertedMessages, convertedMsg)
	}

	return convertedMessages, nil
}

func (c *Client) getReplies(channel, messageId string) ([]models.Message, error) {
	if channel == "" {
		return nil, errors.New("argument 'channel' is required")
	}

	var allReplies []slack.Message

	cursor := ""
	for {
		msgs, _, nextCursor, err := c.api.GetConversationReplies(&slack.GetConversationRepliesParameters{
			ChannelID: channel,
			Limit:     100,
			Cursor:    cursor,
			Timestamp: messageId,
		})
		if err != nil {
			return nil, err
		}

		allReplies = append(allReplies, msgs...)

		if nextCursor == "" {
			break
		}

		cursor = nextCursor
	}

	// Filter out reply which matches the parent message
	filterFn := func(replies []slack.Message, parentId string) (ret []slack.Message) {
		for _, r := range replies {
			if r.Timestamp != parentId {
				ret = append(ret, r)
			}
		}
		return
	}
	filteredReplies := filterFn(allReplies, messageId)

	convertedReplies := make([]models.Message, 0, len(filteredReplies))
	for _, message := range filteredReplies {
		convertedReplies = append(convertedReplies, c.convertToMsg(message))
	}

	return convertedReplies, nil
}

func (c *Client) convertToMsg(message slack.Message) models.Message {
	time, err := utils.EpochToRFC1123(message.Timestamp)
	if err != nil {
		fmt.Printf("could not convert message timestamp '%s': %v", message.Timestamp, err)
	}

	return models.Message{
		Id:   message.Timestamp,
		Time: time,
		Text: message.Text,
	}
}
