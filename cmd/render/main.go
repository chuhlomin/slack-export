package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"os"

	_ "embed"

	"github.com/jessevdk/go-flags"

	"github.com/chuhlomin/slack-export/pkg/structs"
)

type config struct {
	Input  string `long:"input" description:"Input JSON file" required:"true"`
	Output string `long:"output" description:"Output HTML file" required:"true"`
}

//go:embed template.html
var tmpl string
var cfg config

var fm = template.FuncMap{}

func main() {
	if err := run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func run() error {
	if _, err := flags.Parse(&cfg); err != nil {
		return fmt.Errorf("could not parse flags: %v", err)
	}

	var msgs []structs.Message
	content, err := os.ReadFile(cfg.Input)
	if err != nil {
		return fmt.Errorf("could not read file: %v", err)
	}

	if err := json.Unmarshal(content, &msgs); err != nil {
		return fmt.Errorf("could not unmarshal messages: %v", err)
	}

	t, err := template.New("template").Funcs(fm).Parse(tmpl)
	if err != nil {
		return fmt.Errorf("could not parse template: %v", err)
	}

	o, err := os.Create(cfg.Output)
	if err != nil {
		return fmt.Errorf("could not create file: %v", err)
	}

	if err := t.Execute(o, struct {
		Messages []structs.Message
	}{
		Messages: msgs,
	}); err != nil {
		return fmt.Errorf("could not execute template: %v", err)
	}

	return nil
}
