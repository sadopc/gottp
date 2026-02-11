package websocket

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/serdar/gottp/internal/protocol"
)

// WSClientMessage represents a message received from a WebSocket server.
type WSClientMessage struct {
	Content   string
	IsJSON    bool
	Timestamp time.Time
	Err       error
}

// Client implements the WebSocket protocol. It is stateful: once connected,
// the underlying connection persists across calls until explicitly closed.
type Client struct {
	mu        sync.Mutex
	conn      *websocket.Conn
	connected bool
}

// New creates a new WebSocket client.
func New() *Client {
	return &Client{}
}

func (c *Client) Name() string { return "websocket" }

func (c *Client) Validate(req *protocol.Request) error {
	if req.URL == "" {
		return fmt.Errorf("URL is required")
	}
	return nil
}

// Execute implements protocol.Protocol. If the client is not yet connected it
// dials the server, stores the connection, and returns a synthetic 101
// Switching Protocols response. If already connected and the request body is
// non-empty, it sends the body as a text message and returns a synthetic 200.
func (c *Client) Execute(ctx context.Context, req *protocol.Request) (*protocol.Response, error) {
	if err := c.Validate(req); err != nil {
		return nil, err
	}

	c.mu.Lock()
	alreadyConnected := c.connected
	c.mu.Unlock()

	if !alreadyConnected {
		start := time.Now()
		if err := c.Connect(ctx, req.URL, req.Headers, req.Auth); err != nil {
			return nil, fmt.Errorf("websocket connect: %w", err)
		}
		duration := time.Since(start)

		return &protocol.Response{
			StatusCode:  101,
			Status:      "101 Switching Protocols",
			Headers:     http.Header{"Upgrade": []string{"websocket"}},
			Body:        []byte("WebSocket connection established"),
			ContentType: "text/plain",
			Duration:    duration,
			Size:        int64(len("WebSocket connection established")),
			Proto:       "websocket",
			TLS:         len(req.URL) > 3 && req.URL[:4] == "wss:",
		}, nil
	}

	// Already connected -- send the request body as a text message.
	if len(req.Body) > 0 {
		start := time.Now()
		if err := c.Send(ctx, string(req.Body)); err != nil {
			return nil, fmt.Errorf("websocket send: %w", err)
		}
		duration := time.Since(start)

		return &protocol.Response{
			StatusCode:  200,
			Status:      "200 Message Sent",
			Headers:     http.Header{},
			Body:        req.Body,
			ContentType: "text/plain",
			Duration:    duration,
			Size:        int64(len(req.Body)),
			Proto:       "websocket",
		}, nil
	}

	return &protocol.Response{
		StatusCode:  200,
		Status:      "200 Already Connected",
		Headers:     http.Header{},
		Body:        []byte("WebSocket already connected"),
		ContentType: "text/plain",
		Proto:       "websocket",
	}, nil
}

// Connect establishes a WebSocket connection to the given URL. Custom headers
// and auth configuration are applied to the initial HTTP handshake.
func (c *Client) Connect(ctx context.Context, url string, headers map[string]string, auth *protocol.AuthConfig) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return fmt.Errorf("already connected")
	}

	httpHeaders := make(http.Header)
	for k, v := range headers {
		httpHeaders.Set(k, v)
	}

	applyAuth(httpHeaders, auth)

	conn, _, err := websocket.Dial(ctx, url, &websocket.DialOptions{
		HTTPHeader: httpHeaders,
	})
	if err != nil {
		return fmt.Errorf("dialing %s: %w", url, err)
	}

	c.conn = conn
	c.connected = true
	return nil
}

// Send writes a text message on the open WebSocket connection.
func (c *Client) Send(ctx context.Context, content string) error {
	c.mu.Lock()
	conn := c.conn
	connected := c.connected
	c.mu.Unlock()

	if !connected || conn == nil {
		return fmt.Errorf("not connected")
	}

	return conn.Write(ctx, websocket.MessageText, []byte(content))
}

// Close gracefully closes the WebSocket connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected || c.conn == nil {
		return nil
	}

	err := c.conn.Close(websocket.StatusNormalClosure, "client closed")
	c.conn = nil
	c.connected = false
	return err
}

// IsConnected returns whether the client currently holds an open connection.
func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}

// ReadMessages reads messages from the WebSocket connection in a loop and
// sends each message to msgChan. It should be called as a goroutine after a
// successful Connect. The loop exits when the context is cancelled, the
// connection is closed, or a read error occurs (the final message on the
// channel will carry the error).
func (c *Client) ReadMessages(ctx context.Context, msgChan chan<- WSClientMessage) {
	defer close(msgChan)

	for {
		c.mu.Lock()
		conn := c.conn
		connected := c.connected
		c.mu.Unlock()

		if !connected || conn == nil {
			return
		}

		typ, reader, err := conn.Reader(ctx)
		if err != nil {
			// Check for normal closure -- don't treat it as an error.
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
				return
			}
			select {
			case msgChan <- WSClientMessage{Err: err, Timestamp: time.Now()}:
			case <-ctx.Done():
			}
			return
		}

		data, err := io.ReadAll(reader)
		if err != nil {
			select {
			case msgChan <- WSClientMessage{Err: fmt.Errorf("reading message body: %w", err), Timestamp: time.Now()}:
			case <-ctx.Done():
			}
			return
		}

		content := string(data)
		isJSON := false
		if typ == websocket.MessageText && len(data) > 0 {
			first := data[0]
			isJSON = first == '{' || first == '['
		}

		select {
		case msgChan <- WSClientMessage{
			Content:   content,
			IsJSON:    isJSON,
			Timestamp: time.Now(),
		}:
		case <-ctx.Done():
			return
		}
	}
}

// applyAuth sets authentication headers for the WebSocket handshake.
func applyAuth(headers http.Header, auth *protocol.AuthConfig) {
	if auth == nil || auth.Type == "none" || auth.Type == "" {
		return
	}
	switch auth.Type {
	case "basic":
		encoded := base64.StdEncoding.EncodeToString(
			[]byte(auth.Username + ":" + auth.Password),
		)
		headers.Set("Authorization", "Basic "+encoded)
	case "bearer":
		headers.Set("Authorization", "Bearer "+auth.Token)
	case "apikey":
		if auth.APIKey != "" {
			headers.Set(auth.APIKey, auth.APIValue)
		}
	case "oauth2":
		if auth.OAuth2 != nil && auth.OAuth2.AccessToken != "" {
			headers.Set("Authorization", "Bearer "+auth.OAuth2.AccessToken)
		}
	}
}
