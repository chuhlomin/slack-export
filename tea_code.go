package main

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type modelCode struct {
	code textinput.Model
}

func initialModelCode() modelCode {
	t := textinput.New()
	t.Cursor.Style = cursorStyle
	t.Prompt = "Code ▶︎ "
	t.Placeholder = "1234567890.1234567890.0A1B2C3D4E5F6G7H8I9J0K"
	t.Focus()

	return modelCode{code: t}
}

func (mc modelCode) Init() tea.Cmd {
	return textinput.Blink
}

func (mc modelCode) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return mc, tea.Quit
		case "enter":
			return mc, tea.Quit
		}
	}

	cmd := mc.update(msg)

	return mc, cmd
}

func (mc *modelCode) update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd

	mc.code, cmd = mc.code.Update(msg)

	return cmd
}

func (mc modelCode) View() string {
	return fmt.Sprintf(
		"%s\n\n%s",
		mc.code.View(),
		"Press Enter to submit, or Ctrl+C to quit.",
	)
}
