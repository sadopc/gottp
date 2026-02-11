package curl

import "testing"

func FuzzParseCurl(f *testing.F) {
	// Seed corpus: valid curl commands covering various features
	f.Add(`curl https://api.example.com/users`)
	f.Add(`curl -X POST -H 'Content-Type: application/json' -d '{"name":"test"}' https://api.example.com/users`)
	f.Add(`curl -u admin:secret https://api.example.com/private`)
	f.Add(`curl -H "Accept: application/json" -H "Authorization: Bearer token123" https://api.example.com`)
	f.Add(`curl -d 'data=value' https://api.example.com`)
	f.Add("curl \\\n  -X PUT \\\n  -H 'Content-Type: text/plain' \\\n  -d 'hello' \\\n  https://example.com")
	f.Add(`curl -X DELETE https://api.example.com/users/42`)
	f.Add(`curl --request PATCH --header "Content-Type: application/json" --data-raw '{"active":true}' https://api.example.com/users/1`)
	f.Add(`curl -A "Mozilla/5.0" https://example.com`)
	f.Add(`curl --compressed -k -v -s -S -L https://example.com`)
	f.Add(`curl -o output.txt https://example.com/file`)
	f.Add(`curl --data-binary @file.txt https://example.com/upload`)
	f.Add(`curl --insecure --silent --show-error --location --include https://example.com`)
	f.Add(`curl -u user:pass -X POST -H "X-Custom: value" -d "body content" https://example.com/api`)

	// Edge cases
	f.Add(``)
	f.Add(`curl`)
	f.Add(`curl -H 'Accept: */*'`)
	f.Add(`not a curl command at all`)
	f.Add(`CURL https://example.com`)
	f.Add(`curl ''`)
	f.Add(`curl -X`)
	f.Add(`curl -H`)
	f.Add(`curl -d`)

	f.Fuzz(func(t *testing.T, input string) {
		// ParseCurl must not panic on any input.
		// It may return an error, which is fine.
		req, err := ParseCurl(input)
		if err != nil {
			// Error is expected for many fuzzed inputs
			return
		}
		// If no error, the result should have basic fields populated
		if req == nil {
			t.Fatal("ParseCurl returned nil request without error")
		}
		if req.URL == "" {
			t.Fatal("ParseCurl returned request with empty URL without error")
		}
		if req.Method == "" {
			t.Fatal("ParseCurl returned request with empty Method without error")
		}
	})
}

func FuzzTokenize(f *testing.F) {
	f.Add(`curl -H 'Content-Type: application/json' -d '{"key":"val"}' "https://example.com"`)
	f.Add(`simple words here`)
	f.Add(`"double quoted string"`)
	f.Add(`'single quoted string'`)
	f.Add(`escaped\ space`)
	f.Add(`mixed 'single' "double" plain`)
	f.Add(``)
	f.Add(`"unclosed double quote`)
	f.Add(`'unclosed single quote`)
	f.Add(`backslash at end\`)
	f.Add("tabs\there\tand\tthere")
	f.Add("newlines\nand\nmore")

	f.Fuzz(func(t *testing.T, input string) {
		// tokenize must not panic on any input
		tokens := tokenize(input)
		// tokens should be a valid (possibly empty) slice
		_ = tokens
	})
}
