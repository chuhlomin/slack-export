package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type modelChoices struct {
	cursor   int
	choices  []string
	selected map[int]struct{}
}

func initialModelChoices() modelChoices {
	return modelChoices{
		choices:  []string{"public_channel", "private_channel", "im", "mpim"},
		selected: make(map[int]struct{}),
	}
}

func (mc modelChoices) Init() tea.Cmd {
	return tea.SetWindowTitle("Select channels to export")
}

func (mc modelChoices) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			for i := range mc.selected {
				delete(mc.selected, i)
			}
			return mc, tea.Quit
		case "up", "k":
			if mc.cursor > 0 {
				mc.cursor--
			}
		case "down", "j":
			if mc.cursor < len(mc.choices)-1 {
				mc.cursor++
			}
		case " ":
			_, ok := mc.selected[mc.cursor]
			if ok {
				delete(mc.selected, mc.cursor)
			} else {
				mc.selected[mc.cursor] = struct{}{}
			}
		case "enter":
			return mc, tea.Quit
		}
	}

	return mc, nil
}

func (mc modelChoices) View() string {
	s := "What channels do you want to export?\n\n"

	for i, choice := range mc.choices {
		cursor := " "
		if mc.cursor == i {
			cursor = ">"
		}

		checked := " "
		if _, ok := mc.selected[i]; ok {
			checked = "x"
		}

		s += fmt.Sprintf("%s [%s] %s\n", cursor, checked, choice)
	}

	s += "\nPress q to quit. Press enter to continue.\n"

	return s
}
