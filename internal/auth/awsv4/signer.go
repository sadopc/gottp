package awsv4

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// AWSConfig holds AWS credential and service information.
type AWSConfig struct {
	AccessKeyID    string
	SecretAccessKey string
	SessionToken   string
	Region         string
	Service        string
}

// Sign signs an HTTP request with AWS Signature Version 4.
func Sign(req *http.Request, body []byte, cfg AWSConfig, t time.Time) error {
	if cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" {
		return fmt.Errorf("AWS access key and secret key are required")
	}
	if cfg.Region == "" {
		return fmt.Errorf("AWS region is required")
	}
	if cfg.Service == "" {
		return fmt.Errorf("AWS service is required")
	}

	// Format timestamps
	amzDate := t.UTC().Format("20060102T150405Z")
	dateStamp := t.UTC().Format("20060102")

	// Set required headers
	req.Header.Set("X-Amz-Date", amzDate)
	if cfg.SessionToken != "" {
		req.Header.Set("X-Amz-Security-Token", cfg.SessionToken)
	}
	req.Header.Set("X-Amz-Content-Sha256", hashPayload(body))

	// Build canonical request
	signedHeaders := getSignedHeaders(req)
	canonReq := canonicalRequest(req, body, signedHeaders)

	// Build string to sign
	credentialScope := fmt.Sprintf("%s/%s/%s/aws4_request", dateStamp, cfg.Region, cfg.Service)
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		credentialScope,
		hashString(canonReq),
	}, "\n")

	// Derive signing key
	signingKey := deriveSigningKey(cfg.SecretAccessKey, dateStamp, cfg.Region, cfg.Service)

	// Calculate signature
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	// Build Authorization header
	authHeader := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		cfg.AccessKeyID, credentialScope, strings.Join(signedHeaders, ";"), signature)
	req.Header.Set("Authorization", authHeader)

	return nil
}

// deriveSigningKey derives the signing key for AWS Sig V4.
func deriveSigningKey(secret, dateStamp, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secret), []byte(dateStamp))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	kSigning := hmacSHA256(kService, []byte("aws4_request"))
	return kSigning
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func hashString(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
