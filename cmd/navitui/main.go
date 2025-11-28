package main

import (
	"fmt"
	"os"

	"navitui/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	os.Exit(run(os.Args))
}

func run(args []string) int {
	if len(args) < 2 {
		runTUI()
		return 0
	}
	fmt.Printf("nice command bro\n")
	return 0
}

func runTUI() error {
	p := tea.NewProgram(ui.InitialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		return err
	}

	return nil
}
