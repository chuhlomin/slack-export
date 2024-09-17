package main

// A simple example demonstrating the use of multiple text input components
// from the Bubbles component library.

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	focusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	blurredStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	cursorStyle  = focusedStyle
	noStyle      = lipgloss.NewStyle()

	focusedButton = focusedStyle.Render("[ Submit ]")
	blurredButton = fmt.Sprintf("[ %s ]", blurredStyle.Render("Submit"))
)

type modelInputs struct {
	focusIndex int
	inputs     []textinput.Model
}

func initialModelInputs(clientID, clientSecret string) modelInputs {
	mi := modelInputs{
		inputs: make([]textinput.Model, 3),
	}

	var t textinput.Model
	for i := range mi.inputs {
		t = textinput.New()
		t.Cursor.Style = cursorStyle

		switch i {
		case 0:
			t.Prompt = "Slack App OAuth Client ID > "
			t.Placeholder = "1234567890.1234567890123"
			if clientID != "" {
				t.SetValue(clientID)
			}
			t.CharLimit = 100
			t.Focus()
			t.PromptStyle = focusedStyle
			t.TextStyle = focusedStyle
		case 1:
			t.Prompt = "Slack App OAuth Client Secret > "
			t.Placeholder = "12345678901234567890123456789012"
			if clientSecret != "" {
				t.SetValue(clientSecret)
			}
			t.CharLimit = 32
		case 2:
			t.Prompt = "Slack User Token (optional) > "
			t.Placeholder = "xoxp-"
			t.CharLimit = 76
		}

		mi.inputs[i] = t
	}

	return mi
}

func (mi modelInputs) Init() tea.Cmd {
	return textinput.Blink
}

func (mi modelInputs) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return mi, tea.Quit

		// Set focus to next input
		case "tab", "shift+tab", "enter", "up", "down":
			s := msg.String()

			// Did the user press enter while the submit button was focused?
			// If so, exit.
			if s == "enter" && mi.focusIndex == len(mi.inputs) {
				return mi, tea.Quit
			}

			// Cycle indexes
			if s == "up" || s == "shift+tab" {
				mi.focusIndex--
			} else {
				mi.focusIndex++
			}

			if mi.focusIndex > len(mi.inputs) {
				mi.focusIndex = 0
			} else if mi.focusIndex < 0 {
				mi.focusIndex = len(mi.inputs)
			}

			cmds := make([]tea.Cmd, len(mi.inputs))
			for i := 0; i <= len(mi.inputs)-1; i++ {
				if i == mi.focusIndex {
					// Set focused state
					cmds[i] = mi.inputs[i].Focus()
					mi.inputs[i].PromptStyle = focusedStyle
					mi.inputs[i].TextStyle = focusedStyle
					continue
				}
				// Remove focused state
				mi.inputs[i].Blur()
				mi.inputs[i].PromptStyle = noStyle
				mi.inputs[i].TextStyle = noStyle
			}

			return mi, tea.Batch(cmds...)
		}
	}

	// Handle character input and blinking
	cmd := mi.updateInputs(msg)

	return mi, cmd
}

func (mi *modelInputs) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(mi.inputs))

	// Only text inputs with Focus() set will respond, so it's safe to simply
	// update all of them here without any further logic.
	for i := range mi.inputs {
		mi.inputs[i], cmds[i] = mi.inputs[i].Update(msg)
	}

	return tea.Batch(cmds...)
}

func (mi modelInputs) View() string {
	var b strings.Builder

	for i := range mi.inputs {
		b.WriteString(mi.inputs[i].View())
		if i < len(mi.inputs)-1 {
			b.WriteRune('\n')
		}
	}

	button := &blurredButton
	if mi.focusIndex == len(mi.inputs) {
		button = &focusedButton
	}
	fmt.Fprintf(&b, "\n\n%s\n\n", *button)

	return b.String()
}
