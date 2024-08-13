package slack

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"

	"github.com/chuhlomin/slack-export/pkg/structs"
	"github.com/slack-go/slack"
)

type TokenResponse struct {
	Ok          bool   `json:"ok"`
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	BotUserID   string `json:"bot_user_id"`
	AppID       string `json:"app_id"`
	Team        struct {
		Name string `json:"name"`
		ID   string `json:"id"`
	} `json:"team"`
	Enterprise struct {
		Name string `json:"name"`
		ID   string `json:"id"`
	} `json:"enterprise"`
	AuthedUser struct {
		ID          string `json:"id"`
		Scope       string `json:"scope"`
		TokenType   string `json:"token_type"`
		AccessToken string `json:"access_token"`
	} `json:"authed_user"`
}

type Client struct {
	clientID     string
	clientSecret string
	api          *slack.Client
	seenUsers    map[string]interface{}
}

func NewClient(id, secret string) *Client {
	return &Client{
		clientID:     id,
		clientSecret: secret,
		seenUsers:    make(map[string]interface{}),
	}
}

func (c *Client) GetAuthorizeURL(state string) string {
	url := url.URL{
		Scheme: "https",
		Host:   "slack.com",
		Path:   "/oauth/v2/authorize",
	}

	vals := url.Query()
	vals.Add("scope", "")
	vals.Add("user_scope", strings.Join(
		[]string{
			"channels:history",
			"groups:history",
			"im:history",
			"mpim:history",
			"users:read",
			"channels:read",
		},
		",",
	))
	vals.Add("redirect_uri", "https://exporter.local")
	vals.Add("client_id", c.clientID)

	if state != "" {
		vals.Add("state", state)
	}

	url.RawQuery = vals.Encode()

	return url.String()
}

func (c *Client) SetToken(token string) {
	c.api = slack.New(token)
}

func (c *Client) GetToken(code string) error {
	if code == "" {
		return errors.New("argument 'code' is required")
	}

	// set multipart/form-data values
	multipartData := &bytes.Buffer{}
	writer := multipart.NewWriter(multipartData)
	writer.WriteField("client_id", c.clientID)
	writer.WriteField("client_secret", c.clientSecret)
	writer.WriteField("code", code)
	writer.Close()

	req, err := http.NewRequest("POST", "https://slack.com/api/oauth.v2.access", multipartData)
	if err != nil {
		return fmt.Errorf("could not create request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("could not send request: %v", err)
	}

	defer resp.Body.Close()

	var token TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return fmt.Errorf("could not decode response: %v", err)
	}

	if !token.Ok {
		return fmt.Errorf("error response: %#v", token)
	}

	log.Printf("Token received: %s", token.AuthedUser.AccessToken)

	c.api = slack.New(token.AuthedUser.AccessToken)
	return nil
}

func (c *Client) GetUsers() ([]slack.User, error) {
	users := make([]slack.User, 0, len(c.seenUsers))
	for user := range c.seenUsers {
		if user == "" {
			continue
		}
		u, err := c.api.GetUserInfo(user)
		if err != nil {
			return nil, fmt.Errorf("%q: %v", user, err)
		}

		users = append(users, *u)
	}

	return users, nil
}

func (c *Client) GetMessages(channel string) ([]structs.Message, error) {
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

	var convertedMessages []structs.Message
	for _, msg := range allMessages {
		var replies []structs.Message
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

func (c *Client) getReplies(channel, messageId string) ([]structs.Message, error) {
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

	convertedReplies := make([]structs.Message, 0, len(filteredReplies))
	for _, message := range filteredReplies {
		convertedReplies = append(convertedReplies, c.convertToMsg(message))
	}

	return convertedReplies, nil
}

func (c *Client) convertToMsg(message slack.Message) structs.Message {
	c.seenUsers[message.User] = nil

	return structs.Message{
		Message: message,
	}
}
