package importutil

import (
	"encoding/json"
	"strings"
)

// DetectFormat inspects the data and returns the detected import format.
func DetectFormat(data []byte) string {
	s := strings.TrimSpace(string(data))

	// Check for curl command
	if strings.HasPrefix(s, "curl ") || strings.HasPrefix(s, "curl\t") {
		return "curl"
	}

	// Try JSON
	var obj map[string]json.RawMessage
	if json.Unmarshal(data, &obj) == nil {
		// Postman: has "info" with "schema" field
		if _, ok := obj["info"]; ok {
			if _, ok := obj["item"]; ok {
				return "postman"
			}
		}
		// Insomnia: has "_type": "export"
		if typeRaw, ok := obj["_type"]; ok {
			var t string
			if json.Unmarshal(typeRaw, &t) == nil && t == "export" {
				return "insomnia"
			}
		}
		// HAR: has "log" with "entries" inside
		if logRaw, ok := obj["log"]; ok {
			var logObj map[string]json.RawMessage
			if json.Unmarshal(logRaw, &logObj) == nil {
				if _, ok := logObj["entries"]; ok {
					return "har"
				}
			}
		}
		// OpenAPI JSON: has "openapi" field
		if _, ok := obj["openapi"]; ok {
			return "openapi"
		}
	}

	// OpenAPI YAML: check for "openapi:" at start of lines
	if strings.Contains(s, "openapi:") && strings.Contains(s, "paths:") {
		return "openapi"
	}

	return "unknown"
}
