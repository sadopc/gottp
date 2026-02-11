package awsv4

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestSign(t *testing.T) {
	cfg := AWSConfig{
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		Region:          "us-east-1",
		Service:         "s3",
	}

	req, err := http.NewRequest("GET", "https://examplebucket.s3.amazonaws.com/test.txt", nil)
	if err != nil {
		t.Fatal(err)
	}

	ts := time.Date(2013, 5, 24, 0, 0, 0, 0, time.UTC)
	err = Sign(req, nil, cfg, ts)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	auth := req.Header.Get("Authorization")
	if auth == "" {
		t.Fatal("missing Authorization header")
	}
	if !strings.HasPrefix(auth, "AWS4-HMAC-SHA256") {
		t.Errorf("expected AWS4-HMAC-SHA256 prefix, got: %s", auth)
	}
	if !strings.Contains(auth, "Credential=AKIAIOSFODNN7EXAMPLE/20130524/us-east-1/s3/aws4_request") {
		t.Errorf("wrong credential scope in: %s", auth)
	}

	amzDate := req.Header.Get("X-Amz-Date")
	if amzDate != "20130524T000000Z" {
		t.Errorf("expected 20130524T000000Z, got %s", amzDate)
	}
}

func TestSignWithSessionToken(t *testing.T) {
	cfg := AWSConfig{
		AccessKeyID:     "ASIAACCESSKEY",
		SecretAccessKey: "secretkey",
		SessionToken:    "session-token-123",
		Region:          "eu-west-1",
		Service:         "execute-api",
	}

	req, err := http.NewRequest("POST", "https://api.example.com/test", strings.NewReader(`{"key":"value"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	err = Sign(req, []byte(`{"key":"value"}`), cfg, time.Now())
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	if req.Header.Get("X-Amz-Security-Token") != "session-token-123" {
		t.Error("missing session token header")
	}
}

func TestSignMissingCredentials(t *testing.T) {
	cfg := AWSConfig{
		Region:  "us-east-1",
		Service: "s3",
	}

	req, _ := http.NewRequest("GET", "https://example.com", nil)
	err := Sign(req, nil, cfg, time.Now())
	if err == nil {
		t.Fatal("expected error for missing credentials")
	}
}

func TestDeriveSigningKey(t *testing.T) {
	key := deriveSigningKey("wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", "20130524", "us-east-1", "s3")
	if len(key) != 32 {
		t.Errorf("expected 32 byte signing key, got %d", len(key))
	}
}

func TestHashPayload(t *testing.T) {
	// Empty payload should produce the known SHA-256 of empty string
	hash := hashPayload(nil)
	expected := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if hash != expected {
		t.Errorf("expected %s, got %s", expected, hash)
	}
}

func TestCanonicalRequest(t *testing.T) {
	req, _ := http.NewRequest("GET", "https://example.com/path?foo=bar&baz=qux", nil)
	req.Header.Set("Host", "example.com")
	req.Header.Set("X-Amz-Date", "20130524T000000Z")

	signedHeaders := getSignedHeaders(req)
	cr := canonicalRequest(req, nil, signedHeaders)

	if !strings.Contains(cr, "GET") {
		t.Error("canonical request should contain method")
	}
	if !strings.Contains(cr, "/path") {
		t.Error("canonical request should contain path")
	}
}
