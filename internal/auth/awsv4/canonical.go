package awsv4

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

// canonicalRequest builds the AWS canonical request string.
func canonicalRequest(req *http.Request, body []byte, signedHeaders []string) string {
	return strings.Join([]string{
		req.Method,
		canonicalURI(req.URL),
		canonicalQueryString(req.URL),
		canonicalHeaders(req, signedHeaders),
		strings.Join(signedHeaders, ";"),
		hashPayload(body),
	}, "\n")
}

// canonicalURI returns the URI-encoded path.
func canonicalURI(u *url.URL) string {
	path := u.Path
	if path == "" {
		path = "/"
	}
	return path
}

// canonicalQueryString returns sorted query parameters.
func canonicalQueryString(u *url.URL) string {
	params := u.Query()
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var pairs []string
	for _, k := range keys {
		for _, v := range params[k] {
			pairs = append(pairs, url.QueryEscape(k)+"="+url.QueryEscape(v))
		}
	}
	return strings.Join(pairs, "&")
}

// canonicalHeaders builds the canonical headers string.
func canonicalHeaders(req *http.Request, signedHeaders []string) string {
	var b strings.Builder
	for _, h := range signedHeaders {
		var val string
		if h == "host" {
			val = req.Host
			if val == "" {
				val = req.URL.Host
			}
		} else {
			val = strings.Join(req.Header.Values(h), ",")
		}
		b.WriteString(h)
		b.WriteString(":")
		b.WriteString(strings.TrimSpace(val))
		b.WriteString("\n")
	}
	return b.String()
}

// hashPayload returns the SHA-256 hex digest of the payload.
func hashPayload(body []byte) string {
	h := sha256.Sum256(body)
	return hex.EncodeToString(h[:])
}

// getSignedHeaders returns sorted lowercase header names.
func getSignedHeaders(req *http.Request) []string {
	seen := map[string]bool{}
	var headers []string
	for k := range req.Header {
		lk := strings.ToLower(k)
		if !seen[lk] {
			seen[lk] = true
			headers = append(headers, lk)
		}
	}
	if !seen["host"] {
		headers = append(headers, "host")
	}
	sort.Strings(headers)
	return headers
}
