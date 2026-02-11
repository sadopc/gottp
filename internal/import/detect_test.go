package importutil

import "testing"

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		name string
		data string
		want string
	}{
		{
			name: "detects curl command with space",
			data: "curl https://api.example.com/users",
			want: "curl",
		},
		{
			name: "detects curl command with tab",
			data: "curl\thttps://api.example.com/users",
			want: "curl",
		},
		{
			name: "detects postman collection json",
			data: `{"info":{"name":"Sample","schema":"https://schema.getpostman.com/json/collection/v2.1.0/collection.json"},"item":[]}`,
			want: "postman",
		},
		{
			name: "detects insomnia export json",
			data: `{"_type":"export","resources":[]}`,
			want: "insomnia",
		},
		{
			name: "detects openapi json",
			data: `{"openapi":"3.1.0","paths":{}}`,
			want: "openapi",
		},
		{
			name: "detects openapi yaml",
			data: "openapi: 3.0.3\npaths:\n  /users:\n    get:\n      summary: list users\n",
			want: "openapi",
		},
		{
			name: "detects har json",
			data: `{"log":{"version":"1.2","entries":[{"request":{"method":"GET","url":"http://example.com"}}]}}`,
			want: "har",
		},
		{
			name: "returns unknown for plain text",
			data: "GET /users HTTP/1.1",
			want: "unknown",
		},
		{
			name: "returns unknown for empty input",
			data: "",
			want: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectFormat([]byte(tt.data))
			if got != tt.want {
				t.Fatalf("DetectFormat() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDetectFormatPrecedence(t *testing.T) {
	tests := []struct {
		name string
		data string
		want string
	}{
		{
			name: "curl takes precedence over json-like body",
			data: "curl https://api.example.com -d '{\"openapi\":\"3.0.0\"}'",
			want: "curl",
		},
		{
			name: "postman takes precedence when info and item exist",
			data: `{"info":{"name":"n"},"item":[],"openapi":"3.0.0"}`,
			want: "postman",
		},
		{
			name: "insomnia requires export type",
			data: `{"_type":"request"}`,
			want: "unknown",
		},
		{
			name: "openapi yaml needs paths marker",
			data: "openapi: 3.0.0\ncomponents:\n  schemas: {}\n",
			want: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectFormat([]byte(tt.data))
			if got != tt.want {
				t.Fatalf("DetectFormat() = %q, want %q", got, tt.want)
			}
		})
	}
}
