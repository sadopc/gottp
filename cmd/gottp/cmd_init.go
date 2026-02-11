package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/serdar/gottp/internal/core/collection"
)

func initCmd() {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	nameFlag := fs.String("name", "", "Collection name (default: prompt interactively)")
	outputFlag := fs.String("output", "", "Output file path (default: <name>.gottp.yaml)")
	withEnvFlag := fs.Bool("with-env", false, "Also create an environments.yaml file")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gottp init [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Create a new .gottp.yaml collection file.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  gottp init\n")
		fmt.Fprintf(os.Stderr, "  gottp init --name \"My API\"\n")
		fmt.Fprintf(os.Stderr, "  gottp init --name \"My API\" --with-env\n")
		fmt.Fprintf(os.Stderr, "  gottp init --output api.gottp.yaml\n")
	}

	if err := fs.Parse(os.Args[2:]); err != nil {
		os.Exit(1)
	}

	reader := bufio.NewReader(os.Stdin)
	name := *nameFlag
	if name == "" {
		fmt.Print("Collection name: ")
		input, _ := reader.ReadString('\n')
		name = strings.TrimSpace(input)
		if name == "" {
			name = "My API"
		}
	}

	// Prompt for a base URL
	fmt.Print("Base URL (e.g. https://api.example.com, leave empty to skip): ")
	baseURL, _ := reader.ReadString('\n')
	baseURL = strings.TrimSpace(baseURL)

	col := &collection.Collection{
		Name:    name,
		Version: "1",
	}

	if baseURL != "" {
		col.Variables = map[string]string{
			"base_url": baseURL,
		}
	}

	// Add a sample request
	sampleURL := "https://httpbin.org/get"
	if baseURL != "" {
		sampleURL = "{{base_url}}/health"
	}
	col.Items = []collection.Item{
		{
			Folder: &collection.Folder{
				Name: "General",
				Items: []collection.Item{
					{
						Request: collection.NewRequest("Health Check", "GET", sampleURL),
					},
				},
			},
		},
	}

	outputPath := *outputFlag
	if outputPath == "" {
		// Generate filename from collection name
		safeName := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
		outputPath = safeName + ".gottp.yaml"
	}

	// Check if file already exists
	if _, err := os.Stat(outputPath); err == nil {
		fmt.Fprintf(os.Stderr, "Error: file %q already exists\n", outputPath)
		os.Exit(1)
	}

	if err := collection.SaveToFile(col, outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Created %s\n", outputPath)

	// Optionally create environments.yaml
	if *withEnvFlag {
		envPath := "environments.yaml"
		if _, err := os.Stat(envPath); err == nil {
			fmt.Fprintf(os.Stderr, "Warning: %s already exists, skipping\n", envPath)
		} else {
			envContent := `environments:
  - name: Development
    variables:
      base_url:
        value: "http://localhost:8080"
  - name: Production
    variables:
      base_url:
        value: "https://api.example.com"
`
			if baseURL != "" {
				envContent = fmt.Sprintf(`environments:
  - name: Development
    variables:
      base_url:
        value: "http://localhost:8080"
  - name: Production
    variables:
      base_url:
        value: %q
`, baseURL)
			}
			if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating environments.yaml: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Created %s\n", envPath)
		}
	}
}
