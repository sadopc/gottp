package oauth2

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
)

// GenerateCodeVerifier creates a cryptographically random PKCE code verifier.
func GenerateCodeVerifier() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generating code verifier: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// GenerateCodeChallenge computes the S256 code challenge from a verifier.
func GenerateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// BuildAuthURL constructs the authorization URL for the auth code flow.
func BuildAuthURL(cfg OAuth2Config, state, codeChallenge string) string {
	params := url.Values{
		"response_type": {"code"},
		"client_id":     {cfg.ClientID},
		"redirect_uri":  {cfg.RedirectURI},
		"state":         {state},
	}
	if cfg.Scope != "" {
		params.Set("scope", cfg.Scope)
	}
	if codeChallenge != "" {
		params.Set("code_challenge", codeChallenge)
		params.Set("code_challenge_method", "S256")
	}

	u, _ := url.Parse(cfg.AuthURL)
	u.RawQuery = params.Encode()
	return u.String()
}
