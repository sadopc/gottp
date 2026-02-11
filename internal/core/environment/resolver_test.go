package environment

import "testing"

func TestResolve(t *testing.T) {
	envVars := map[string]string{
		"base_url":   "https://api.example.com",
		"auth_token": "secret123",
	}
	colVars := map[string]string{
		"version": "v1",
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"{{base_url}}/users", "https://api.example.com/users"},
		{"Bearer {{auth_token}}", "Bearer secret123"},
		{"{{base_url}}/{{version}}/users", "https://api.example.com/v1/users"},
		{"no variables here", "no variables here"},
		{"{{unknown}}", "{{unknown}}"}, // unreplaced
		{"", ""},
	}

	for _, tc := range tests {
		result := Resolve(tc.input, envVars, colVars)
		if result != tc.expected {
			t.Errorf("Resolve(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestResolveKVPairs(t *testing.T) {
	envVars := map[string]string{"token": "abc"}
	colVars := map[string]string{}

	pairs := []KVPair{
		{Key: "Authorization", Value: "Bearer {{token}}", Enabled: true},
		{Key: "X-Custom", Value: "static", Enabled: false},
	}

	resolved := ResolveKVPairs(pairs, envVars, colVars)
	if resolved[0].Value != "Bearer abc" {
		t.Errorf("expected 'Bearer abc', got %q", resolved[0].Value)
	}
	if resolved[1].Enabled {
		t.Error("second pair should still be disabled")
	}
}
