package importutil

import "testing"

func FuzzDetectFormat(f *testing.F) {
	// Seed: known format examples
	f.Add([]byte(`curl https://api.example.com/users`))
	f.Add([]byte("curl\thttps://api.example.com/users"))
	f.Add([]byte(`{"info":{"name":"Sample","schema":"https://schema.getpostman.com/json/collection/v2.1.0/collection.json"},"item":[]}`))
	f.Add([]byte(`{"_type":"export","resources":[]}`))
	f.Add([]byte(`{"openapi":"3.1.0","paths":{}}`))
	f.Add([]byte("openapi: 3.0.3\npaths:\n  /users:\n    get:\n      summary: list users\n"))
	f.Add([]byte(`{"log":{"version":"1.2","entries":[{"request":{"method":"GET","url":"http://example.com"}}]}}`))

	// Seed: unknown/edge cases
	f.Add([]byte(`GET /users HTTP/1.1`))
	f.Add([]byte(``))
	f.Add([]byte(`{}`))
	f.Add([]byte(`null`))
	f.Add([]byte(`[]`))
	f.Add([]byte(`not a format at all`))
	f.Add([]byte(`  curl https://example.com`)) // leading whitespace before curl
	f.Add([]byte(`{"_type":"request"}`))         // _type but not "export"
	f.Add([]byte("openapi: 3.0.0\ncomponents:\n  schemas: {}\n")) // openapi without paths

	// Seed: precedence tests
	f.Add([]byte(`curl https://api.example.com -d '{"openapi":"3.0.0"}'`))
	f.Add([]byte(`{"info":{"name":"n"},"item":[],"openapi":"3.0.0"}`))

	// Seed: large-ish JSON structures
	f.Add([]byte(`{"log":{"entries":[]}}`))                            // log without version (still HAR)
	f.Add([]byte(`{"info":{"name":"test"}}`))                          // info without item (not postman)
	f.Add([]byte(`{"item":[]}`))                                       // item without info (not postman)
	f.Add([]byte(`{"_type":"export"}`))                                // insomnia missing resources
	f.Add([]byte(`{"openapi":"3.0.0","info":{"title":"t"},"paths":{}}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		format := DetectFormat(data)

		// DetectFormat must not panic and must return one of the known values
		validFormats := map[string]bool{
			"curl":    true,
			"postman": true,
			"insomnia": true,
			"openapi": true,
			"har":     true,
			"unknown": true,
		}
		if !validFormats[format] {
			t.Fatalf("DetectFormat returned unexpected format: %q", format)
		}
	})
}
