package curl

import (
	"fmt"
	"strings"

	"github.com/sadopc/gottp/internal/protocol"
)

// ParseCurl parses a curl command string into a protocol.Request.
func ParseCurl(input string) (*protocol.Request, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("empty input")
	}

	// Handle line continuations
	input = strings.ReplaceAll(input, "\\\n", " ")
	input = strings.ReplaceAll(input, "\\\r\n", " ")

	args := tokenize(input)
	if len(args) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	// Strip leading "curl" if present
	if strings.ToLower(args[0]) == "curl" {
		args = args[1:]
	}

	req := &protocol.Request{
		Protocol: "http",
		Method:   "GET",
		Headers:  make(map[string]string),
		Params:   make(map[string]string),
	}

	i := 0
	for i < len(args) {
		arg := args[i]
		switch {
		case arg == "-X" || arg == "--request":
			i++
			if i < len(args) {
				req.Method = strings.ToUpper(args[i])
			}
		case arg == "-H" || arg == "--header":
			i++
			if i < len(args) {
				key, val := parseHeader(args[i])
				if key != "" {
					req.Headers[key] = val
				}
			}
		case arg == "-d" || arg == "--data" || arg == "--data-raw" || arg == "--data-binary":
			i++
			if i < len(args) {
				req.Body = []byte(args[i])
				if req.Method == "GET" {
					req.Method = "POST"
				}
			}
		case arg == "-u" || arg == "--user":
			i++
			if i < len(args) {
				parts := strings.SplitN(args[i], ":", 2)
				req.Auth = &protocol.AuthConfig{Type: "basic", Username: parts[0]}
				if len(parts) > 1 {
					req.Auth.Password = parts[1]
				}
			}
		case arg == "-A" || arg == "--user-agent":
			i++
			if i < len(args) {
				req.Headers["User-Agent"] = args[i]
			}
		case arg == "--compressed" || arg == "-k" || arg == "--insecure" ||
			arg == "-v" || arg == "--verbose" || arg == "-s" || arg == "--silent" ||
			arg == "-S" || arg == "--show-error" || arg == "-L" || arg == "--location" ||
			arg == "-i" || arg == "--include" || arg == "-o" || arg == "--output":
			// Skip known flags without values (except -o which takes a value)
			if arg == "-o" || arg == "--output" {
				i++ // skip the output filename
			}
		case !strings.HasPrefix(arg, "-"):
			// Positional argument = URL
			if req.URL == "" {
				req.URL = arg
			}
		}
		i++
	}

	if req.URL == "" {
		return nil, fmt.Errorf("no URL found in curl command")
	}

	return req, nil
}

// tokenize splits a shell command into tokens, handling single and double quotes.
func tokenize(input string) []string {
	var tokens []string
	var current strings.Builder
	inSingle := false
	inDouble := false
	escaped := false

	for _, r := range input {
		if escaped {
			current.WriteRune(r)
			escaped = false
			continue
		}

		if r == '\\' && !inSingle {
			escaped = true
			continue
		}

		if r == '\'' && !inDouble {
			inSingle = !inSingle
			continue
		}

		if r == '"' && !inSingle {
			inDouble = !inDouble
			continue
		}

		if (r == ' ' || r == '\t' || r == '\n') && !inSingle && !inDouble {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			continue
		}

		current.WriteRune(r)
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

// parseHeader parses "Key: Value" into key and value.
func parseHeader(s string) (string, string) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return s, ""
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}
