package collection

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// LoadFromFile loads a collection from a YAML file.
func LoadFromFile(path string) (*Collection, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading collection file: %w", err)
	}
	return LoadFromBytes(data)
}

// LoadFromBytes parses a collection from YAML bytes.
func LoadFromBytes(data []byte) (*Collection, error) {
	var col Collection
	if err := yaml.Unmarshal(data, &col); err != nil {
		return nil, fmt.Errorf("parsing collection: %w", err)
	}
	if col.Version == "" {
		col.Version = "1"
	}
	// Ensure all requests have IDs
	assignIDs(col.Items)
	return &col, nil
}

// LoadFromDir loads all .gottp.yaml files from a directory.
func LoadFromDir(dir string) ([]*Collection, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "*.gottp.yaml"))
	if err != nil {
		return nil, fmt.Errorf("globbing collection files: %w", err)
	}
	var collections []*Collection
	for _, path := range matches {
		col, err := LoadFromFile(path)
		if err != nil {
			return nil, fmt.Errorf("loading %s: %w", path, err)
		}
		collections = append(collections, col)
	}
	return collections, nil
}

func assignIDs(items []Item) {
	for i := range items {
		if items[i].Request != nil && items[i].Request.ID == "" {
			items[i].Request.ID = uuid.New().String()
		}
		if items[i].Folder != nil {
			assignIDs(items[i].Folder.Items)
		}
	}
}
