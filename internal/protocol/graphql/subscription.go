package graphql

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/sadopc/gottp/internal/protocol"
)

// graphql-ws protocol message types.
// See: https://github.com/enisdenjo/graphql-ws/blob/master/PROTOCOL.md
const (
	msgConnectionInit = "connection_init"
	msgConnectionAck  = "connection_ack"
	msgSubscribe      = "subscribe"
	msgNext           = "next"
	msgError          = "error"
	msgComplete       = "complete"
	msgPing           = "ping"
	msgPong           = "pong"

	// graphql-ws sub-protocol identifier sent during the WebSocket handshake.
	graphqlWSSubprotocol = "graphql-transport-ws"
)

// gqlWSMessage is the envelope for all graphql-ws protocol messages.
type gqlWSMessage struct {
	ID      string          `json:"id,omitempty"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// subscribePayload is the payload for a subscribe message.
type subscribePayload struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

// SubscriptionClient manages a graphql-ws subscription over WebSocket.
type SubscriptionClient struct {
	conn      *websocket.Conn
	connected bool
	subID     string
	mu        sync.Mutex
}

// NewSubscriptionClient creates a new SubscriptionClient.
func NewSubscriptionClient() *SubscriptionClient {
	return &SubscriptionClient{
		subID: "1",
	}
}

// Connect establishes the WebSocket connection and performs the graphql-ws
// handshake (connection_init / connection_ack).
func (c *SubscriptionClient) Connect(ctx context.Context, url string, headers map[string]string) error {
	c.mu.Lock()
	if c.connected {
		c.mu.Unlock()
		return fmt.Errorf("already connected")
	}
	c.mu.Unlock()

	// Convert http(s) URL to ws(s) if needed.
	wsURL := toWebSocketURL(url)

	httpHeaders := make(map[string][]string)
	for k, v := range headers {
		httpHeaders[k] = []string{v}
	}

	conn, _, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		HTTPHeader:   httpHeaders,
		Subprotocols: []string{graphqlWSSubprotocol},
	})
	if err != nil {
		return fmt.Errorf("dialing %s: %w", wsURL, err)
	}

	// Store the connection under lock so other methods can see it.
	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	// Send connection_init.
	initMsg := gqlWSMessage{
		Type:    msgConnectionInit,
		Payload: json.RawMessage(`{}`),
	}
	if err := c.writeJSON(ctx, initMsg); err != nil {
		_ = conn.CloseNow()
		c.mu.Lock()
		c.conn = nil
		c.mu.Unlock()
		return fmt.Errorf("sending connection_init: %w", err)
	}

	// Wait for connection_ack.
	if err := c.waitForAck(ctx); err != nil {
		_ = conn.CloseNow()
		c.mu.Lock()
		c.conn = nil
		c.mu.Unlock()
		return fmt.Errorf("waiting for connection_ack: %w", err)
	}

	c.mu.Lock()
	c.connected = true
	c.mu.Unlock()
	return nil
}

// Subscribe sends a subscription query and starts receiving events on msgChan.
// This method blocks until the subscription completes, the context is cancelled,
// or an error occurs. It should typically be called as a goroutine.
// The caller must not close msgChan; it is closed by Subscribe when done.
func (c *SubscriptionClient) Subscribe(ctx context.Context, query string, variables string, msgChan chan<- protocol.StreamMessage) error {
	defer close(msgChan)

	c.mu.Lock()
	if !c.connected || c.conn == nil {
		c.mu.Unlock()
		return fmt.Errorf("not connected")
	}
	conn := c.conn
	subID := c.subID
	c.mu.Unlock()

	// Build the subscribe payload.
	payload := subscribePayload{
		Query: query,
	}
	if variables != "" {
		var vars map[string]interface{}
		if err := json.Unmarshal([]byte(variables), &vars); err == nil {
			payload.Variables = vars
		}
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling subscribe payload: %w", err)
	}

	subMsg := gqlWSMessage{
		ID:      subID,
		Type:    msgSubscribe,
		Payload: json.RawMessage(payloadBytes),
	}
	if err := c.writeJSON(ctx, subMsg); err != nil {
		return fmt.Errorf("sending subscribe: %w", err)
	}

	// Send the subscribe message itself as a "sent" stream message.
	select {
	case msgChan <- protocol.StreamMessage{
		Content:   string(payloadBytes),
		IsJSON:    true,
		Timestamp: time.Now(),
		Direction: "sent",
	}:
	case <-ctx.Done():
		return ctx.Err()
	}

	// Read loop: process server messages until complete/error/disconnect.
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
				return nil
			}
			select {
			case msgChan <- protocol.StreamMessage{
				Err:       err,
				Timestamp: time.Now(),
				Direction: "received",
			}:
			case <-ctx.Done():
			}
			return err
		}

		var msg gqlWSMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			select {
			case msgChan <- protocol.StreamMessage{
				Err:       fmt.Errorf("invalid server message: %w", err),
				Timestamp: time.Now(),
				Direction: "received",
			}:
			case <-ctx.Done():
			}
			continue
		}

		switch msg.Type {
		case msgNext:
			content := string(msg.Payload)
			isJSON := len(msg.Payload) > 0 && (msg.Payload[0] == '{' || msg.Payload[0] == '[')
			select {
			case msgChan <- protocol.StreamMessage{
				Content:   content,
				IsJSON:    isJSON,
				Timestamp: time.Now(),
				Direction: "received",
			}:
			case <-ctx.Done():
				return ctx.Err()
			}

		case msgError:
			errContent := string(msg.Payload)
			select {
			case msgChan <- protocol.StreamMessage{
				Content:   errContent,
				IsJSON:    true,
				Timestamp: time.Now(),
				Direction: "received",
				Err:       fmt.Errorf("subscription error: %s", errContent),
			}:
			case <-ctx.Done():
			}
			return fmt.Errorf("subscription error: %s", errContent)

		case msgComplete:
			return nil

		case msgPing:
			// Respond to server pings with a pong.
			pong := gqlWSMessage{Type: msgPong}
			_ = c.writeJSON(ctx, pong)

		default:
			// Ignore unknown message types.
		}
	}
}

// Close gracefully closes the subscription and the underlying WebSocket
// connection.
func (c *SubscriptionClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected || c.conn == nil {
		return nil
	}

	// Send complete for the active subscription.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	completeMsg := gqlWSMessage{
		ID:   c.subID,
		Type: msgComplete,
	}
	// Best-effort: don't fail Close if sending complete fails.
	_ = c.writeJSONLocked(ctx, completeMsg)

	err := c.conn.Close(websocket.StatusNormalClosure, "client closed")
	c.conn = nil
	c.connected = false

	// Ignore errors that indicate the connection is already closed. This is
	// common when the server completes the subscription and drops the conn.
	if isAlreadyClosedErr(err) {
		return nil
	}
	return err
}

// IsConnected returns whether the subscription client holds an open connection.
func (c *SubscriptionClient) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}

// waitForAck reads messages until it receives a connection_ack. Must be called
// with c.mu NOT held (it reads from the connection).
func (c *SubscriptionClient) waitForAck(ctx context.Context) error {
	// Use a timeout for the ack to avoid blocking forever.
	ackCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	for {
		_, data, err := c.conn.Read(ackCtx)
		if err != nil {
			return fmt.Errorf("reading ack: %w", err)
		}

		var msg gqlWSMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return fmt.Errorf("decoding ack message: %w", err)
		}

		switch msg.Type {
		case msgConnectionAck:
			return nil
		case msgPing:
			// Respond to ping during handshake.
			pong := gqlWSMessage{Type: msgPong}
			_ = c.writeJSONLocked(ackCtx, pong)
		default:
			return fmt.Errorf("expected connection_ack, got %q", msg.Type)
		}
	}
}

// writeJSON marshals and sends a JSON message on the connection. It acquires
// the mutex.
func (c *SubscriptionClient) writeJSON(ctx context.Context, msg gqlWSMessage) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.writeJSONLocked(ctx, msg)
}

// writeJSONLocked marshals and sends a JSON message. Caller must hold c.mu.
func (c *SubscriptionClient) writeJSONLocked(ctx context.Context, msg gqlWSMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return c.conn.Write(ctx, websocket.MessageText, data)
}

// toWebSocketURL converts an http/https URL to ws/wss.
func toWebSocketURL(url string) string {
	if strings.HasPrefix(url, "http://") {
		return "ws://" + url[7:]
	}
	if strings.HasPrefix(url, "https://") {
		return "wss://" + url[8:]
	}
	return url
}

// isSubscription checks whether the given GraphQL operation is a subscription.
// It trims whitespace and line comments, then checks if the operation starts
// with the "subscription" keyword.
func isSubscription(query string) bool {
	q := strings.TrimSpace(query)

	// Strip leading single-line comments.
	for strings.HasPrefix(q, "#") {
		idx := strings.IndexByte(q, '\n')
		if idx == -1 {
			return false // only comments, no query
		}
		q = strings.TrimSpace(q[idx+1:])
	}

	return strings.HasPrefix(strings.ToLower(q), "subscription")
}

// isAlreadyClosedErr returns true if the error indicates the WebSocket
// connection was already closed by the peer (EOF, closed network connection,
// or a WebSocket normal closure status).
func isAlreadyClosedErr(err error) bool {
	if err == nil {
		return false
	}
	if err == io.EOF {
		return true
	}
	if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
		return true
	}
	// Check the error message as a fallback for wrapped errors.
	msg := err.Error()
	return strings.Contains(msg, "EOF") ||
		strings.Contains(msg, "use of closed network connection") ||
		strings.Contains(msg, "connection reset")
}
