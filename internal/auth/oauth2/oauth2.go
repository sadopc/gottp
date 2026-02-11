package oauth2

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// OAuth2Config holds OAuth2 configuration.
type OAuth2Config struct {
	GrantType    string // authorization_code, client_credentials, password
	AuthURL      string
	TokenURL     string
	ClientID     string
	ClientSecret string
	Scope        string
	Username     string // for password grant
	Password     string // for password grant
	UsePKCE      bool
	RedirectURI  string
}

// TokenResponse holds the OAuth2 token response.
type TokenResponse struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	RefreshToken string    `json:"refresh_token"`
	Scope        string    `json:"scope"`
	ObtainedAt   time.Time `json:"-"`
}

// IsExpired checks whether the token has expired.
func (t *TokenResponse) IsExpired() bool {
	if t.ExpiresIn == 0 {
		return false
	}
	return time.Since(t.ObtainedAt) > time.Duration(t.ExpiresIn)*time.Second
}

// ClientCredentials performs the client_credentials grant.
func ClientCredentials(ctx context.Context, cfg OAuth2Config) (*TokenResponse, error) {
	data := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {cfg.ClientID},
		"client_secret": {cfg.ClientSecret},
	}
	if cfg.Scope != "" {
		data.Set("scope", cfg.Scope)
	}
	return tokenRequest(ctx, cfg.TokenURL, data)
}

// PasswordGrant performs the resource owner password grant.
func PasswordGrant(ctx context.Context, cfg OAuth2Config) (*TokenResponse, error) {
	data := url.Values{
		"grant_type":    {"password"},
		"client_id":     {cfg.ClientID},
		"client_secret": {cfg.ClientSecret},
		"username":      {cfg.Username},
		"password":      {cfg.Password},
	}
	if cfg.Scope != "" {
		data.Set("scope", cfg.Scope)
	}
	return tokenRequest(ctx, cfg.TokenURL, data)
}

// ExchangeAuthCode exchanges an authorization code for tokens.
func ExchangeAuthCode(ctx context.Context, cfg OAuth2Config, code, codeVerifier string) (*TokenResponse, error) {
	data := url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"client_id":    {cfg.ClientID},
		"redirect_uri": {cfg.RedirectURI},
	}
	if cfg.ClientSecret != "" {
		data.Set("client_secret", cfg.ClientSecret)
	}
	if codeVerifier != "" {
		data.Set("code_verifier", codeVerifier)
	}
	return tokenRequest(ctx, cfg.TokenURL, data)
}

// RefreshAccessToken refreshes an expired access token.
func RefreshAccessToken(ctx context.Context, tokenURL, clientID, clientSecret, refreshToken string) (*TokenResponse, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {clientID},
	}
	if clientSecret != "" {
		data.Set("client_secret", clientSecret)
	}
	return tokenRequest(ctx, tokenURL, data)
}

func tokenRequest(ctx context.Context, tokenURL string, data url.Values) (*TokenResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var token TokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("parsing token response: %w", err)
	}
	token.ObtainedAt = time.Now()

	return &token, nil
}
