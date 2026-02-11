// Package digest implements HTTP Digest Authentication (RFC 7616).
package digest

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// Challenge represents a parsed WWW-Authenticate digest challenge.
type Challenge struct {
	Realm     string
	Nonce     string
	Opaque    string
	Algorithm string // MD5, SHA-256, MD5-sess, SHA-256-sess
	QOP       string // auth, auth-int
}

// ParseChallenge extracts digest parameters from a WWW-Authenticate header value.
// The header should start with "Digest " followed by comma-separated key=value pairs.
func ParseChallenge(header string) (*Challenge, error) {
	header = strings.TrimSpace(header)

	// Must start with "Digest "
	if !strings.HasPrefix(header, "Digest ") && !strings.HasPrefix(header, "digest ") {
		return nil, fmt.Errorf("not a Digest challenge: %q", header)
	}

	params := header[7:] // strip "Digest "
	ch := &Challenge{
		Algorithm: "MD5", // default per RFC
	}

	for _, part := range splitParams(params) {
		part = strings.TrimSpace(part)
		idx := strings.IndexByte(part, '=')
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(strings.ToLower(part[:idx]))
		val := strings.TrimSpace(part[idx+1:])
		val = unquote(val)

		switch key {
		case "realm":
			ch.Realm = val
		case "nonce":
			ch.Nonce = val
		case "opaque":
			ch.Opaque = val
		case "algorithm":
			ch.Algorithm = val
		case "qop":
			ch.QOP = val
		}
	}

	if ch.Realm == "" {
		return nil, fmt.Errorf("digest challenge missing realm")
	}
	if ch.Nonce == "" {
		return nil, fmt.Errorf("digest challenge missing nonce")
	}

	return ch, nil
}

// Authorize creates the Authorization header value for a digest auth response.
// method is the HTTP method (GET, POST, etc.) and uri is the request URI path.
func Authorize(username, password, method, uri string, ch *Challenge) string {
	cnonce := generateCNonce()
	nc := "00000001"

	ha1 := computeHA1(ch.Algorithm, username, ch.Realm, password, ch.Nonce, cnonce)
	ha2 := computeHA2(ch.Algorithm, method, uri)
	response := computeResponse(ha1, ch.Nonce, nc, cnonce, ch.QOP, ha2, ch.Algorithm)

	// Build the Authorization header value
	parts := []string{
		fmt.Sprintf(`username="%s"`, username),
		fmt.Sprintf(`realm="%s"`, ch.Realm),
		fmt.Sprintf(`nonce="%s"`, ch.Nonce),
		fmt.Sprintf(`uri="%s"`, uri),
		fmt.Sprintf(`algorithm=%s`, ch.Algorithm),
		fmt.Sprintf(`response="%s"`, response),
	}

	if ch.QOP != "" {
		parts = append(parts, fmt.Sprintf(`qop=%s`, firstQOP(ch.QOP)))
		parts = append(parts, fmt.Sprintf(`nc=%s`, nc))
		parts = append(parts, fmt.Sprintf(`cnonce="%s"`, cnonce))
	}

	if ch.Opaque != "" {
		parts = append(parts, fmt.Sprintf(`opaque="%s"`, ch.Opaque))
	}

	return "Digest " + strings.Join(parts, ", ")
}

// computeHA1 computes the HA1 hash based on the algorithm.
func computeHA1(algorithm, username, realm, password, nonce, cnonce string) string {
	base := hashFn(algorithm, username+":"+realm+":"+password)
	// For -sess variants, HA1 = H(H(username:realm:password):nonce:cnonce)
	alg := strings.ToUpper(algorithm)
	if strings.HasSuffix(alg, "-SESS") {
		return hashFn(algorithm, base+":"+nonce+":"+cnonce)
	}
	return base
}

// computeHA2 computes the HA2 hash: H(method:uri).
// For qop=auth-int the entity body would be included, but we only support qop=auth.
func computeHA2(algorithm, method, uri string) string {
	return hashFn(algorithm, method+":"+uri)
}

// computeResponse computes the final response hash.
func computeResponse(ha1, nonce, nc, cnonce, qop, ha2, algorithm string) string {
	if qop == "" {
		// Legacy: response = H(HA1:nonce:HA2)
		return hashFn(algorithm, ha1+":"+nonce+":"+ha2)
	}
	q := firstQOP(qop)
	return hashFn(algorithm, ha1+":"+nonce+":"+nc+":"+cnonce+":"+q+":"+ha2)
}

// hashFn selects MD5 or SHA-256 based on the algorithm string.
func hashFn(algorithm, data string) string {
	alg := strings.ToUpper(algorithm)
	// Strip -sess suffix for hash selection
	alg = strings.TrimSuffix(alg, "-SESS")
	switch alg {
	case "SHA-256":
		return hashSHA256(data)
	default:
		return hashMD5(data)
	}
}

func hashMD5(data string) string {
	h := md5.Sum([]byte(data))
	return hex.EncodeToString(h[:])
}

func hashSHA256(data string) string {
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])
}

// generateCNonce creates a random client nonce.
func generateCNonce() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback: should never happen in practice
		return "0000000000000000"
	}
	return hex.EncodeToString(b[:8])
}

// firstQOP returns the first qop option from a potentially comma-separated list.
// Servers may offer "auth,auth-int"; we pick the first supported one (prefer "auth").
func firstQOP(qop string) string {
	for _, q := range strings.Split(qop, ",") {
		q = strings.TrimSpace(q)
		if q == "auth" {
			return "auth"
		}
	}
	// If no "auth" found, return the first one
	parts := strings.SplitN(qop, ",", 2)
	return strings.TrimSpace(parts[0])
}

// splitParams splits a comma-separated parameter string, respecting quoted values.
func splitParams(s string) []string {
	var parts []string
	var current strings.Builder
	inQuotes := false

	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch {
		case ch == '"' && (i == 0 || s[i-1] != '\\'):
			inQuotes = !inQuotes
			current.WriteByte(ch)
		case ch == ',' && !inQuotes:
			parts = append(parts, current.String())
			current.Reset()
		default:
			current.WriteByte(ch)
		}
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

// unquote removes surrounding double quotes if present.
func unquote(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}
