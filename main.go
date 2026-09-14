package main

import (
	"log"

	tea "charm.land/bubbletea/v2"

	"clocky/internal/ui"
)

func main() {
	program := tea.NewProgram(ui.NewModel())
	if _, err := program.Run(); err != nil {
		log.Fatal(err)
	}
}
