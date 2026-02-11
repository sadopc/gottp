package environment

import (
	"os"
	"regexp"
	"strings"
)

var varPattern = regexp.MustCompile(`\{\{(\w+)\}\}`)

// Resolve replaces {{variable}} placeholders in a string using the provided variable map.
// It checks environment variables, then collection variables, then OS env vars.
func Resolve(input string, envVars, colVars map[string]string) string {
	return varPattern.ReplaceAllStringFunc(input, func(match string) string {
		key := strings.TrimPrefix(strings.TrimSuffix(match, "}}"), "{{")
		// Priority: environment vars > collection vars > OS env
		if v, ok := envVars[key]; ok {
			return v
		}
		if v, ok := colVars[key]; ok {
			return v
		}
		if v := os.Getenv(key); v != "" {
			return v
		}
		return match // leave unreplaced
	})
}

// ResolveKVPairs resolves variables in key-value pairs.
func ResolveKVPairs(pairs []KVPair, envVars, colVars map[string]string) []KVPair {
	resolved := make([]KVPair, len(pairs))
	for i, p := range pairs {
		resolved[i] = KVPair{
			Key:     Resolve(p.Key, envVars, colVars),
			Value:   Resolve(p.Value, envVars, colVars),
			Enabled: p.Enabled,
		}
	}
	return resolved
}

// KVPair is copied here to avoid circular imports.
type KVPair struct {
	Key     string
	Value   string
	Enabled bool
}
