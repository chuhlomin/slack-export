package main

import (
	"context"
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

var (
	cfg            config
	errBadResponse = fmt.Errorf("bad response")
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func run() error {
	if _, err := flags.Parse(&cfg); err != nil {
		return fmt.Errorf("could not parse flags: %w", err)
	}

	var data structs.Data
	content, err := os.ReadFile(cfg.Input)
	if err != nil {
		return fmt.Errorf("could not read file: %w", err)
	}

	if err := json.Unmarshal(content, &data); err != nil {
		return fmt.Errorf("could not unmarshal messages: %w", err)
	}

	for _, user := range data.Users {
		if user.Profile.Image512 != "" {
			err := downloadFile(user.ID, user.Profile.Image512, cfg.Output)
			if err != nil {
				return fmt.Errorf("could not download file: %w", err)
			}
		}
	}

	return nil
}

func downloadFile(id, fileURL, output string) error {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, fileURL, http.NoBody)
	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: %d", errBadResponse, resp.StatusCode)
	}

	filename := filepath.Join(output, "avatars", id+".png")
	file, err := os.Create(filename)
	if err != nil {
		return err
	}

	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return err
	}

	return nil
}
