package main

import (
	"log"
	"os"

	"navitui/internal/app"
)

func main() {
	err := app.Init()
	if err != nil {
		log.Fatalf("Failed to initialize navitui: %v", err)
	}

	os.Exit(0)
}
