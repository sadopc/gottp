package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sadopc/gottp/internal/core/collection"
	"github.com/sadopc/gottp/internal/core/environment"
)

func validateCmd() {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gottp validate <file.gottp.yaml> [files...]\n\n")
		fmt.Fprintf(os.Stderr, "Validate collection and environment YAML files.\n\n")
		fmt.Fprintf(os.Stderr, "If an environments.yaml exists next to the collection, it is also validated.\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  gottp validate api.gottp.yaml\n")
		fmt.Fprintf(os.Stderr, "  gottp validate *.gottp.yaml\n")
	}

	if err := fs.Parse(os.Args[2:]); err != nil {
		os.Exit(1)
	}

	if fs.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Error: at least one file path is required\n\n")
		fs.Usage()
		os.Exit(1)
	}

	hasErrors := false
	for _, path := range fs.Args() {
		if err := validateFile(path); err != nil {
			fmt.Fprintf(os.Stderr, "FAIL %s: %v\n", path, err)
			hasErrors = true
		} else {
			fmt.Printf("OK   %s\n", path)
		}
	}

	if hasErrors {
		os.Exit(1)
	}
}

func validateFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	if len(data) == 0 {
		return fmt.Errorf("file is empty")
	}

	// Check if it's an environment file
	if filepath.Base(path) == "environments.yaml" {
		return validateEnvironment(path)
	}

	// Validate as collection
	col, err := collection.LoadFromBytes(data)
	if err != nil {
		return err
	}

	// Structural checks
	var warnings []string

	if col.Name == "" {
		warnings = append(warnings, "missing collection name")
	}

	requestCount := countRequests(col.Items)
	if requestCount == 0 {
		warnings = append(warnings, "collection contains no requests")
	}

	// Check for duplicate request IDs
	ids := make(map[string]string)
	duplicates := checkDuplicateIDs(col.Items, ids)
	for _, dup := range duplicates {
		warnings = append(warnings, fmt.Sprintf("duplicate request ID: %s", dup))
	}

	// Check for empty URLs
	emptyURLs := checkEmptyURLs(col.Items)
	for _, name := range emptyURLs {
		warnings = append(warnings, fmt.Sprintf("request %q has empty URL", name))
	}

	if len(warnings) > 0 {
		return fmt.Errorf("validation warnings:\n  - %s", strings.Join(warnings, "\n  - "))
	}

	// Also validate environments.yaml if present
	dir := filepath.Dir(path)
	envPath := filepath.Join(dir, "environments.yaml")
	if _, err := os.Stat(envPath); err == nil {
		if err := validateEnvironment(envPath); err != nil {
			fmt.Fprintf(os.Stderr, "WARN %s: %v\n", envPath, err)
		} else {
			fmt.Printf("OK   %s\n", envPath)
		}
	}

	return nil
}

func validateEnvironment(path string) error {
	ef, err := environment.LoadEnvironments(path)
	if err != nil {
		return err
	}

	if len(ef.Environments) == 0 {
		return fmt.Errorf("no environments defined")
	}

	// Check for duplicate environment names
	names := make(map[string]bool)
	for _, env := range ef.Environments {
		if env.Name == "" {
			return fmt.Errorf("environment has empty name")
		}
		if names[env.Name] {
			return fmt.Errorf("duplicate environment name: %s", env.Name)
		}
		names[env.Name] = true
	}

	return nil
}

func countRequests(items []collection.Item) int {
	count := 0
	for _, item := range items {
		if item.Request != nil {
			count++
		}
		if item.Folder != nil {
			count += countRequests(item.Folder.Items)
		}
	}
	return count
}

func checkDuplicateIDs(items []collection.Item, seen map[string]string) []string {
	var duplicates []string
	for _, item := range items {
		if item.Request != nil && item.Request.ID != "" {
			if prevName, exists := seen[item.Request.ID]; exists {
				duplicates = append(duplicates, fmt.Sprintf("%s (in %q and %q)", item.Request.ID, prevName, item.Request.Name))
			}
			seen[item.Request.ID] = item.Request.Name
		}
		if item.Folder != nil {
			duplicates = append(duplicates, checkDuplicateIDs(item.Folder.Items, seen)...)
		}
	}
	return duplicates
}

func checkEmptyURLs(items []collection.Item) []string {
	var empty []string
	for _, item := range items {
		if item.Request != nil && item.Request.URL == "" {
			empty = append(empty, item.Request.Name)
		}
		if item.Folder != nil {
			empty = append(empty, checkEmptyURLs(item.Folder.Items)...)
		}
	}
	return empty
}
