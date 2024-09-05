package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	_ "embed"

	"github.com/jessevdk/go-flags"

	"github.com/chuhlomin/slack-export/pkg/structs"
)

type config struct {
	Input  string `long:"input" description:"Input JSON file" required:"true"`
	Output string `long:"output" description:"Output directory file" required:"true"`
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

	var data structs.Data
	content, err := os.ReadFile(cfg.Input)
	if err != nil {
		return fmt.Errorf("could not read file: %v", err)
	}

	if err := json.Unmarshal(content, &data); err != nil {
		return fmt.Errorf("could not unmarshal messages: %v", err)
	}

	for _, user := range data.Users {
		if user.Profile.Image512 != "" {
			downloadFile(user.ID, user.Profile.Image512, cfg.Output)
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
