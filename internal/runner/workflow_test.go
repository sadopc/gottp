package runner

import (
	"testing"
)

func TestExtractValue(t *testing.T) {
	body := []byte(`{"id": 123, "name": "John", "address": {"city": "NYC"}, "tags": ["go", "api"]}`)

	tests := []struct {
		expr     string
		expected string
	}{
		{"$.id", "123"},
		{"$.name", "John"},
		{"$.address.city", "NYC"},
		{"$.tags[0]", "go"},
		{"$.tags[1]", "api"},
		{"id", "123"},
		{"$.nonexistent", ""},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result := extractValue(body, tt.expr)
			if result != tt.expected {
				t.Errorf("extractValue(%q) = %q, want %q", tt.expr, result, tt.expected)
			}
		})
	}
}

func TestExtractValueInvalidJSON(t *testing.T) {
	result := extractValue([]byte("not json"), "$.id")
	if result != "" {
		t.Errorf("expected empty string for invalid JSON, got %q", result)
	}
}

func TestEvaluateCondition(t *testing.T) {
	result := Result{StatusCode: 200}

	tests := []struct {
		condition string
		expected  bool
	}{
		{"status == 200", true},
		{"status == 404", false},
		{"status < 400", true},
		{"status >= 200", true},
		{"status > 200", false},
		{"status != 200", false},
		{"status != 404", true},
		{"success", true},
	}

	for _, tt := range tests {
		t.Run(tt.condition, func(t *testing.T) {
			got := evaluateCondition(tt.condition, result)
			if got != tt.expected {
				t.Errorf("evaluateCondition(%q) = %v, want %v", tt.condition, got, tt.expected)
			}
		})
	}
}

func TestEvaluateConditionFailedRequest(t *testing.T) {
	result := Result{StatusCode: 500}
	if evaluateCondition("success", result) {
		t.Error("expected 'success' to be false for 500 status")
	}
}
