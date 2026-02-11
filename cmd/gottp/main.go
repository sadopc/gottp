package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/serdar/gottp/internal/app"
	"github.com/serdar/gottp/internal/core/collection"
	"github.com/serdar/gottp/pkg/version"
)

func main() {
	versionFlag := flag.Bool("version", false, "Print version and exit")
	collectionFlag := flag.String("collection", "", "Path to a .gottp.yaml collection file")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("gottp %s (%s) built %s\n", version.Version, version.Commit, version.Date)
		os.Exit(0)
	}

	// Load collection
	var col *collection.Collection
	var colPath string

	if *collectionFlag != "" {
		c, err := collection.LoadFromFile(*collectionFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading collection: %v\n", err)
			os.Exit(1)
		}
		col = c
		colPath = *collectionFlag
	} else {
		// Try to find a .gottp.yaml file in the current directory
		cwd, _ := os.Getwd()
		matches, _ := filepath.Glob(filepath.Join(cwd, "*.gottp.yaml"))
		if len(matches) > 0 {
			c, err := collection.LoadFromFile(matches[0])
			if err == nil {
				col = c
				colPath = matches[0]
			}
		}
	}

	model := app.New(col, colPath)
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
