package environment

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// EnvironmentFile holds all environments.
type EnvironmentFile struct {
	Environments []Environment `yaml:"environments"`
}

// Environment represents a named set of variables.
type Environment struct {
	Name      string              `yaml:"name"`
	Variables map[string]Variable `yaml:"variables"`
}

// Variable represents an environment variable value.
type Variable struct {
	Value  string `yaml:"value"`
	Secret bool   `yaml:"secret,omitempty"`
}

// LoadEnvironments loads environments from a YAML file.
func LoadEnvironments(path string) (*EnvironmentFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &EnvironmentFile{}, nil
		}
		return nil, fmt.Errorf("reading environments: %w", err)
	}
	var ef EnvironmentFile
	if err := yaml.Unmarshal(data, &ef); err != nil {
		return nil, fmt.Errorf("parsing environments: %w", err)
	}
	return &ef, nil
}

// GetVariables returns a flat map of variable name -> value for the given environment.
func (ef *EnvironmentFile) GetVariables(envName string) map[string]string {
	result := make(map[string]string)
	for _, env := range ef.Environments {
		if env.Name == envName {
			for k, v := range env.Variables {
				result[k] = v.Value
			}
			break
		}
	}
	return result
}

// Names returns all environment names.
func (ef *EnvironmentFile) Names() []string {
	names := make([]string, len(ef.Environments))
	for i, e := range ef.Environments {
		names[i] = e.Name
	}
	return names
}
