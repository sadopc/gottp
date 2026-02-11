package collection

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// SaveToFile saves a collection to a YAML file.
func SaveToFile(col *Collection, path string) error {
	data, err := yaml.Marshal(col)
	if err != nil {
		return fmt.Errorf("marshaling collection: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing collection file: %w", err)
	}
	return nil
}
