package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type choice struct {
	value     string
	label     string
	isChannel bool
}

type modelChoices struct {
	cursor   int
	choices  []choice
	selected map[int]struct{}
}

const (
	downloadAvatarsIndex = 4
	downloadFilesIndex   = 5
	includeArchivedIndex = 6
)

func initialModelChoices(downloadAvatars, downloadFiles, includeArchived bool) modelChoices {
	selected := make(map[int]struct{})
	if downloadAvatars {
		selected[downloadAvatarsIndex] = struct{}{}
	}
	if downloadFiles {
		selected[downloadFilesIndex] = struct{}{}
	}
	if includeArchived {
		selected[includeArchivedIndex] = struct{}{}
	}

	return modelChoices{
		choices: []choice{
			{"public_channel", "Public channels", true},
			{"private_channel", "Private channels", true},
			{"im", "DM", true},
			{"mpim", "Group DM", true},
			{"downloadAvatars", "Download avatars", false},
			{"downloadFiles", "Download files", false},
			{"includeArchived", "Include archived channels", false},
		},
		selected: selected,
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
			cursor = "▶︎"
		}

		checked := " "
		if _, ok := mc.selected[i]; ok {
			checked = "x"
		}

		if choice.value == "downloadAvatars" {
			s += "\n"
		}

		s += fmt.Sprintf("%s [%s] %s\n", cursor, checked, choice.label)
	}

	s += "\nPress q to quit. Press enter to continue.\n"

	return s
}
