package digest

import (
	"strings"
	"testing"
)

func TestParseChallenge(t *testing.T) {
	header := `Digest realm="testrealm@host.com", nonce="dcd98b7102dd2f0e8b11d0f600bfb0c093", opaque="5ccc069c403ebaf9f0171e9517f40e41", qop="auth"`

	ch, err := ParseChallenge(header)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ch.Realm != "testrealm@host.com" {
		t.Errorf("realm = %q, want %q", ch.Realm, "testrealm@host.com")
	}
	if ch.Nonce != "dcd98b7102dd2f0e8b11d0f600bfb0c093" {
		t.Errorf("nonce = %q, want %q", ch.Nonce, "dcd98b7102dd2f0e8b11d0f600bfb0c093")
	}
	if ch.Opaque != "5ccc069c403ebaf9f0171e9517f40e41" {
		t.Errorf("opaque = %q, want %q", ch.Opaque, "5ccc069c403ebaf9f0171e9517f40e41")
	}
	if ch.QOP != "auth" {
		t.Errorf("qop = %q, want %q", ch.QOP, "auth")
	}
	if ch.Algorithm != "MD5" {
		t.Errorf("algorithm = %q, want %q (default)", ch.Algorithm, "MD5")
	}
}

func TestParseChallenge_SHA256(t *testing.T) {
	header := `Digest realm="api.example.com", nonce="abc123", algorithm=SHA-256, qop="auth"`

	ch, err := ParseChallenge(header)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ch.Algorithm != "SHA-256" {
		t.Errorf("algorithm = %q, want %q", ch.Algorithm, "SHA-256")
	}
	if ch.Realm != "api.example.com" {
		t.Errorf("realm = %q, want %q", ch.Realm, "api.example.com")
	}
}

func TestParseChallenge_NoQOP(t *testing.T) {
	header := `Digest realm="legacy", nonce="legacynonce123"`

	ch, err := ParseChallenge(header)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ch.QOP != "" {
		t.Errorf("qop = %q, want empty (legacy mode)", ch.QOP)
	}
	if ch.Realm != "legacy" {
		t.Errorf("realm = %q, want %q", ch.Realm, "legacy")
	}
	if ch.Algorithm != "MD5" {
		t.Errorf("algorithm = %q, want %q", ch.Algorithm, "MD5")
	}
}

func TestParseChallenge_Invalid(t *testing.T) {
	tests := []struct {
		name   string
		header string
	}{
		{"not digest", "Basic realm=test"},
		{"missing realm", `Digest nonce="abc123"`},
		{"missing nonce", `Digest realm="test"`},
		{"empty", ""},
		{"just prefix", "Digest "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseChallenge(tt.header)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestAuthorize_MD5(t *testing.T) {
	ch := &Challenge{
		Realm:     "testrealm@host.com",
		Nonce:     "dcd98b7102dd2f0e8b11d0f600bfb0c093",
		Opaque:    "5ccc069c403ebaf9f0171e9517f40e41",
		Algorithm: "MD5",
		QOP:       "auth",
	}

	result := Authorize("Mufasa", "Circle Of Life", "GET", "/dir/index.html", ch)

	// Verify the result starts with "Digest "
	if !strings.HasPrefix(result, "Digest ") {
		t.Errorf("result should start with 'Digest ', got: %s", result)
	}

	// Verify key components are present
	mustContain := []string{
		`username="Mufasa"`,
		`realm="testrealm@host.com"`,
		`nonce="dcd98b7102dd2f0e8b11d0f600bfb0c093"`,
		`uri="/dir/index.html"`,
		`algorithm=MD5`,
		`qop=auth`,
		`nc=00000001`,
		`opaque="5ccc069c403ebaf9f0171e9517f40e41"`,
	}
	for _, s := range mustContain {
		if !strings.Contains(result, s) {
			t.Errorf("result missing %q, got: %s", s, result)
		}
	}

	// Verify response hash is present and non-empty
	if !strings.Contains(result, `response="`) {
		t.Errorf("result missing response hash, got: %s", result)
	}

	// Verify known HA1 and HA2 for these specific values
	// HA1 = MD5("Mufasa:testrealm@host.com:Circle Of Life") = 939e7578ed9e3c518a452acee763bce9
	ha1 := hashMD5("Mufasa:testrealm@host.com:Circle Of Life")
	if ha1 != "939e7578ed9e3c518a452acee763bce9" {
		t.Errorf("HA1 = %s, want 939e7578ed9e3c518a452acee763bce9", ha1)
	}

	// HA2 = MD5("GET:/dir/index.html") = 39aff3a2bab6126f332b942af96d3366
	ha2 := hashMD5("GET:/dir/index.html")
	if ha2 != "39aff3a2bab6126f332b942af96d3366" {
		t.Errorf("HA2 = %s, want 39aff3a2bab6126f332b942af96d3366", ha2)
	}
}

func TestAuthorize_SHA256(t *testing.T) {
	ch := &Challenge{
		Realm:     "example.com",
		Nonce:     "7ypf/xlj9XXwfDPEoM4URrv/xwf94BcCAzFZH4GiTo0v",
		Algorithm: "SHA-256",
		QOP:       "auth",
	}

	result := Authorize("user", "pass", "GET", "/resource", ch)

	if !strings.HasPrefix(result, "Digest ") {
		t.Errorf("result should start with 'Digest ', got: %s", result)
	}
	if !strings.Contains(result, "algorithm=SHA-256") {
		t.Errorf("result missing algorithm=SHA-256, got: %s", result)
	}
	if !strings.Contains(result, `username="user"`) {
		t.Errorf("result missing username, got: %s", result)
	}

	// Verify SHA-256 hash is 64 hex chars (SHA-256 = 32 bytes = 64 hex)
	// Extract response value
	idx := strings.Index(result, `response="`)
	if idx < 0 {
		t.Fatal("response field not found")
	}
	respStart := idx + len(`response="`)
	respEnd := strings.Index(result[respStart:], `"`)
	if respEnd < 0 {
		t.Fatal("response field not terminated")
	}
	resp := result[respStart : respStart+respEnd]
	if len(resp) != 64 {
		t.Errorf("SHA-256 response hash length = %d, want 64", len(resp))
	}
}

func TestAuthorize_NoQOP(t *testing.T) {
	// Legacy mode without qop
	ch := &Challenge{
		Realm:     "legacy",
		Nonce:     "legacynonce",
		Algorithm: "MD5",
		QOP:       "", // no qop = legacy
	}

	result := Authorize("admin", "secret", "POST", "/api", ch)

	if !strings.HasPrefix(result, "Digest ") {
		t.Errorf("result should start with 'Digest ', got: %s", result)
	}

	// Without qop, nc and cnonce should NOT be present
	if strings.Contains(result, "nc=") {
		t.Errorf("legacy mode should not include nc, got: %s", result)
	}
	if strings.Contains(result, "cnonce=") {
		t.Errorf("legacy mode should not include cnonce, got: %s", result)
	}
	if strings.Contains(result, "qop=") {
		t.Errorf("legacy mode should not include qop, got: %s", result)
	}

	// Verify the response is computed correctly for legacy mode
	// response = MD5(HA1:nonce:HA2)
	ha1 := hashMD5("admin:legacy:secret")
	ha2 := hashMD5("POST:/api")
	expected := hashMD5(ha1 + ":legacynonce:" + ha2)

	if !strings.Contains(result, `response="`+expected+`"`) {
		t.Errorf("response hash mismatch, expected %s in: %s", expected, result)
	}
}

func TestSplitParams(t *testing.T) {
	input := `realm="test,realm", nonce="abc", qop="auth"`
	parts := splitParams(input)
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts, got %d: %v", len(parts), parts)
	}
	// The comma inside quotes should not split
	if !strings.Contains(parts[0], "test,realm") {
		t.Errorf("first part should contain quoted comma: %s", parts[0])
	}
}

func TestFirstQOP(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"auth", "auth"},
		{"auth,auth-int", "auth"},
		{"auth-int,auth", "auth"},
		{"auth-int", "auth-int"},
	}
	for _, tt := range tests {
		got := firstQOP(tt.input)
		if got != tt.want {
			t.Errorf("firstQOP(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
