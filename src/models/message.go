package models

type Message struct {
	Id      string    `json:"id"`
	Time    string    `json:"time"`
	Text    string    `json:"text"`
	Replies []Message `json:"replies,omitempty"`
}
