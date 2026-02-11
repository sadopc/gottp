package protocol

import (
	"context"
	"net/http"
	"time"
)

// Protocol defines the interface all protocol clients must implement.
type Protocol interface {
	Name() string
	Execute(ctx context.Context, req *Request) (*Response, error)
	Validate(req *Request) error
}

// Request is the unified request type across all protocols.
type Request struct {
	ID       string
	Protocol string // http, graphql, grpc, websocket
	Method   string
	URL      string
	Headers  map[string]string
	Params   map[string]string
	Body     []byte
	Auth     *AuthConfig

	// GraphQL-specific
	GraphQLQuery     string
	GraphQLVariables string

	// gRPC-specific
	GRPCService string
	GRPCMethod  string
	Metadata    map[string]string

	// Scripting
	PreScript  string
	PostScript string

	// Timeout
	Timeout time.Duration
}

// AuthConfig holds authentication settings.
type AuthConfig struct {
	Type     string // none, basic, bearer, apikey, oauth2, awsv4
	Username string
	Password string
	Token    string
	APIKey   string
	APIValue string
	APIIn    string // header, query

	// OAuth2
	OAuth2 *OAuth2AuthConfig

	// AWS Signature v4
	AWSAuth *AWSAuthConfig
}

// OAuth2AuthConfig holds OAuth2-specific auth settings.
type OAuth2AuthConfig struct {
	GrantType    string // authorization_code, client_credentials, password
	AuthURL      string
	TokenURL     string
	ClientID     string
	ClientSecret string
	Scope        string
	Username     string
	Password     string
	UsePKCE      bool
	AccessToken  string
	RefreshToken string
	TokenExpiry  time.Time
}

// AWSAuthConfig holds AWS Signature v4 auth settings.
type AWSAuthConfig struct {
	AccessKeyID    string
	SecretAccessKey string
	SessionToken   string
	Region         string
	Service        string
}

// Response is the unified response type.
type Response struct {
	StatusCode  int
	Status      string
	Headers     http.Header
	Body        []byte
	ContentType string
	Duration    time.Duration
	Size        int64
	Proto       string
	TLS         bool
}
