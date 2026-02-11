package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/sadopc/gottp/internal/core/collection"
)

func fmtCmd() {
	fs := flag.NewFlagSet("fmt", flag.ExitOnError)
	writeFlag := fs.Bool("w", false, "Write result to file instead of stdout")
	checkFlag := fs.Bool("check", false, "Check if files are formatted (exit 1 if not)")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gottp fmt [flags] <file.gottp.yaml> [files...]\n\n")
		fmt.Fprintf(os.Stderr, "Format and normalize collection YAML files.\n\n")
		fmt.Fprintf(os.Stderr, "By default, formatted output is written to stdout.\n")
		fmt.Fprintf(os.Stderr, "Use -w to write back to the source file.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  gottp fmt api.gottp.yaml           # print formatted to stdout\n")
		fmt.Fprintf(os.Stderr, "  gottp fmt -w api.gottp.yaml        # overwrite file in-place\n")
		fmt.Fprintf(os.Stderr, "  gottp fmt --check *.gottp.yaml     # check formatting (CI)\n")
	}

	if err := fs.Parse(os.Args[2:]); err != nil {
		os.Exit(1)
	}

	if fs.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Error: at least one file path is required\n\n")
		fs.Usage()
		os.Exit(1)
	}

	hasUnformatted := false
	for _, path := range fs.Args() {
		if err := formatFile(path, *writeFlag, *checkFlag, &hasUnformatted); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting %s: %v\n", path, err)
			os.Exit(1)
		}
	}

	if *checkFlag && hasUnformatted {
		os.Exit(1)
	}
}

func formatFile(path string, write, check bool, hasUnformatted *bool) error {
	original, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	// Parse and re-serialize to normalize
	col, err := collection.LoadFromBytes(original)
	if err != nil {
		return fmt.Errorf("parsing: %w", err)
	}

	// Normalize: ensure all requests have IDs, version is set
	if col.Version == "" {
		col.Version = "1"
	}

	// Re-serialize
	err = collection.SaveToFile(col, path+".tmp")
	if err != nil {
		return fmt.Errorf("serializing: %w", err)
	}
	formatted, err := os.ReadFile(path + ".tmp")
	os.Remove(path + ".tmp")
	if err != nil {
		return fmt.Errorf("reading formatted: %w", err)
	}

	if check {
		if string(original) != string(formatted) {
			fmt.Fprintf(os.Stderr, "UNFORMATTED %s\n", path)
			*hasUnformatted = true
		} else {
			fmt.Printf("OK          %s\n", path)
		}
		return nil
	}

	if write {
		if err := os.WriteFile(path, formatted, 0644); err != nil {
			return fmt.Errorf("writing: %w", err)
		}
		fmt.Printf("Formatted %s\n", path)
	} else {
		os.Stdout.Write(formatted)
	}

	return nil
}
