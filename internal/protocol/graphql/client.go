package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/serdar/gottp/internal/protocol"
)

// Client implements the GraphQL protocol via HTTP POST. It also supports
// GraphQL subscriptions over WebSocket using the graphql-ws protocol.
type Client struct {
	subscription *SubscriptionClient
}

// New creates a new GraphQL client.
func New() *Client {
	return &Client{}
}

func (c *Client) Name() string { return "graphql" }

func (c *Client) Validate(req *protocol.Request) error {
	if req.URL == "" {
		return fmt.Errorf("URL is required")
	}
	if req.GraphQLQuery == "" {
		return fmt.Errorf("GraphQL query is required")
	}
	return nil
}

func (c *Client) Execute(ctx context.Context, req *protocol.Request) (*protocol.Response, error) {
	if err := c.Validate(req); err != nil {
		return nil, err
	}

	// Detect subscription queries and return a hint response — the caller
	// (TUI or runner) should use ConnectSubscription / Subscribe for the
	// actual streaming flow.
	if isSubscription(req.GraphQLQuery) {
		return &protocol.Response{
			StatusCode:  101,
			Status:      "101 Subscription Detected",
			Headers:     http.Header{},
			Body:        []byte("GraphQL subscription detected. Use ConnectSubscription/Subscribe for streaming."),
			ContentType: "text/plain",
			Proto:       "graphql-ws",
		}, nil
	}

	// Build GraphQL request body
	gqlBody := map[string]interface{}{
		"query": req.GraphQLQuery,
	}
	if req.GraphQLVariables != "" {
		var vars map[string]interface{}
		if err := json.Unmarshal([]byte(req.GraphQLVariables), &vars); err == nil {
			gqlBody["variables"] = vars
		}
	}

	bodyBytes, err := json.Marshal(gqlBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling GraphQL body: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", req.URL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Set default headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	// Override with user headers
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	// Apply auth
	if req.Auth != nil {
		applyAuth(httpReq, req.Auth)
	}

	timeout := req.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	client := &http.Client{Timeout: timeout}

	start := time.Now()
	resp, err := client.Do(httpReq)
	duration := time.Since(start)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	return &protocol.Response{
		StatusCode:  resp.StatusCode,
		Status:      resp.Status,
		Headers:     resp.Header,
		Body:        respBody,
		ContentType: resp.Header.Get("Content-Type"),
		Duration:    duration,
		Size:        int64(len(respBody)),
		Proto:       resp.Proto,
		TLS:         resp.TLS != nil,
	}, nil
}

// ConnectSubscription establishes a WebSocket connection for GraphQL
// subscriptions using the graphql-ws protocol. Headers from the request are
// forwarded to the WebSocket handshake.
func (c *Client) ConnectSubscription(ctx context.Context, url string, headers map[string]string) error {
	if c.subscription != nil && c.subscription.IsConnected() {
		return fmt.Errorf("subscription already connected")
	}
	c.subscription = NewSubscriptionClient()
	return c.subscription.Connect(ctx, url, headers)
}

// Subscribe starts a GraphQL subscription and sends events to msgChan. The
// connection must have been established via ConnectSubscription first. This
// method blocks and should be called as a goroutine.
func (c *Client) Subscribe(ctx context.Context, query string, variables string, msgChan chan<- protocol.StreamMessage) error {
	if c.subscription == nil {
		return fmt.Errorf("subscription not connected")
	}
	return c.subscription.Subscribe(ctx, query, variables, msgChan)
}

// CloseSubscription closes the active subscription and its WebSocket
// connection.
func (c *Client) CloseSubscription() error {
	if c.subscription == nil {
		return nil
	}
	err := c.subscription.Close()
	c.subscription = nil
	return err
}

// IsSubscriptionConnected returns whether the subscription client holds an
// open connection.
func (c *Client) IsSubscriptionConnected() bool {
	if c.subscription == nil {
		return false
	}
	return c.subscription.IsConnected()
}

// IsSubscriptionQuery returns true if the given query is a GraphQL
// subscription operation. This is a convenience wrapper for the package-level
// isSubscription function.
func (c *Client) IsSubscriptionQuery(query string) bool {
	return isSubscription(query)
}

func applyAuth(req *http.Request, auth *protocol.AuthConfig) {
	if auth == nil || auth.Type == "none" {
		return
	}
	switch auth.Type {
	case "bearer":
		req.Header.Set("Authorization", "Bearer "+auth.Token)
	case "basic":
		req.SetBasicAuth(auth.Username, auth.Password)
	case "apikey":
		if auth.APIIn == "query" {
			q := req.URL.Query()
			q.Set(auth.APIKey, auth.APIValue)
			req.URL.RawQuery = q.Encode()
		} else {
			req.Header.Set(auth.APIKey, auth.APIValue)
		}
	case "oauth2":
		if auth.OAuth2 != nil && auth.OAuth2.AccessToken != "" {
			req.Header.Set("Authorization", "Bearer "+auth.OAuth2.AccessToken)
		}
	}
}
