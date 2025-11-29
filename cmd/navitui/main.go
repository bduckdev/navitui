package main

import (
	"fmt"
	"os"

	"navitui/internal/navidrome"
	"navitui/internal/tui"
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

func runTUI() {
	tui.Run(navidrome.Tracks)
}
