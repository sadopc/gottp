package export

import (
	"fmt"
	"strings"

	"github.com/sadopc/gottp/internal/protocol"
)

// AsCurl converts a request to a curl command string.
func AsCurl(req *protocol.Request) string {
	var parts []string
	parts = append(parts, "curl")

	// Method
	if req.Method != "GET" {
		parts = append(parts, "-X", req.Method)
	}

	// Headers
	for k, v := range req.Headers {
		parts = append(parts, "-H", fmt.Sprintf("'%s: %s'", k, v))
	}

	// Auth
	if req.Auth != nil {
		switch req.Auth.Type {
		case "basic":
			parts = append(parts, "-u", fmt.Sprintf("'%s:%s'", req.Auth.Username, req.Auth.Password))
		case "bearer":
			parts = append(parts, "-H", fmt.Sprintf("'Authorization: Bearer %s'", req.Auth.Token))
		case "apikey":
			if req.Auth.APIIn == "header" {
				parts = append(parts, "-H", fmt.Sprintf("'%s: %s'", req.Auth.APIKey, req.Auth.APIValue))
			}
		}
	}

	// Body
	if len(req.Body) > 0 {
		body := strings.ReplaceAll(string(req.Body), "'", "'\\''")
		parts = append(parts, "-d", fmt.Sprintf("'%s'", body))
	}

	// URL with params
	url := req.URL
	if len(req.Params) > 0 {
		var params []string
		for k, v := range req.Params {
			params = append(params, fmt.Sprintf("%s=%s", k, v))
		}
		url += "?" + strings.Join(params, "&")
	}
	parts = append(parts, fmt.Sprintf("'%s'", url))

	return strings.Join(parts, " ")
}
