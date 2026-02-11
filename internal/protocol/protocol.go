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

	// Timeout
	Timeout time.Duration
}

// AuthConfig holds authentication settings.
type AuthConfig struct {
	Type     string // none, basic, bearer, apikey
	Username string
	Password string
	Token    string
	APIKey   string
	APIValue string
	APIIn    string // header, query
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
