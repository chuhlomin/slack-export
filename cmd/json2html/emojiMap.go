package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type emojiMap map[string]string

func loadSlackEmoji(path string) (emojiMap, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var m emojiMap
	err = json.NewDecoder(f).Decode(&m)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func (m emojiMap) Get(needle string) (alias string, filename string) {
	e, ok := m[needle]
	if !ok {
		return "", ""
	}

	if strings.HasPrefix(e, "alias:") {
		return strings.TrimPrefix(e, "alias:"), ""
	}

	ext := filepath.Ext(e)
	return "", needle + ext
}
