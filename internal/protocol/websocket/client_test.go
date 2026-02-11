package websocket

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/serdar/gottp/internal/protocol"
)

// newTestServer creates an httptest server that upgrades to WebSocket and
// echoes every text message back to the client.
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			InsecureSkipVerify: true,
		})
		if err != nil {
			t.Logf("accept error: %v", err)
			return
		}
		defer conn.CloseNow()

		for {
			typ, data, err := conn.Read(r.Context())
			if err != nil {
				return
			}
			if err := conn.Write(r.Context(), typ, data); err != nil {
				return
			}
		}
	}))
}

// wsURL converts an http:// test server URL to ws://.
func wsURL(s *httptest.Server) string {
	return "ws" + strings.TrimPrefix(s.URL, "http")
}

func TestName(t *testing.T) {
	c := New()
	if c.Name() != "websocket" {
		t.Errorf("expected websocket, got %s", c.Name())
	}
}

func TestValidate(t *testing.T) {
	c := New()

	if err := c.Validate(&protocol.Request{URL: ""}); err == nil {
		t.Error("expected error for empty URL")
	}
	if err := c.Validate(&protocol.Request{URL: "ws://localhost:8080"}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestConnectAndSend(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	c := New()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Connect via Execute (should return 101).
	resp, err := c.Execute(ctx, &protocol.Request{
		Protocol: "websocket",
		URL:      wsURL(srv),
		Headers:  map[string]string{},
	})
	if err != nil {
		t.Fatalf("Execute (connect) failed: %v", err)
	}
	if resp.StatusCode != 101 {
		t.Errorf("expected 101, got %d", resp.StatusCode)
	}
	if !c.IsConnected() {
		t.Fatal("expected IsConnected to be true after connect")
	}

	// Start reading messages in background.
	msgChan := make(chan WSClientMessage, 10)
	go c.ReadMessages(ctx, msgChan)

	// Send a message.
	if err := c.Send(ctx, "hello"); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// Wait for echo response.
	select {
	case msg := <-msgChan:
		if msg.Err != nil {
			t.Fatalf("received error: %v", msg.Err)
		}
		if msg.Content != "hello" {
			t.Errorf("expected 'hello', got %q", msg.Content)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for echo")
	}

	// Close.
	if err := c.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	if c.IsConnected() {
		t.Error("expected IsConnected to be false after close")
	}
}

func TestExecuteSendMessage(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	c := New()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Connect first.
	err := c.Connect(ctx, wsURL(srv), nil, nil)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	msgChan := make(chan WSClientMessage, 10)
	go c.ReadMessages(ctx, msgChan)

	// Execute with body on an already connected client sends the message.
	resp, err := c.Execute(ctx, &protocol.Request{
		Protocol: "websocket",
		URL:      wsURL(srv),
		Body:     []byte("world"),
	})
	if err != nil {
		t.Fatalf("Execute (send) failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	select {
	case msg := <-msgChan:
		if msg.Err != nil {
			t.Fatalf("received error: %v", msg.Err)
		}
		if msg.Content != "world" {
			t.Errorf("expected 'world', got %q", msg.Content)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for echo")
	}

	c.Close()
}

func TestSendWithoutConnect(t *testing.T) {
	c := New()
	ctx := context.Background()
	if err := c.Send(ctx, "test"); err == nil {
		t.Error("expected error sending without connection")
	}
}

func TestJSONDetection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			InsecureSkipVerify: true,
		})
		if err != nil {
			return
		}
		defer conn.CloseNow()

		// Send a JSON object back.
		conn.Write(r.Context(), websocket.MessageText, []byte(`{"status":"ok"}`))
		// Send a JSON array.
		conn.Write(r.Context(), websocket.MessageText, []byte(`[1,2,3]`))
		// Send plain text.
		conn.Write(r.Context(), websocket.MessageText, []byte(`hello world`))

		// Wait for client to read all messages before closing.
		conn.Read(r.Context())
	}))
	defer srv.Close()

	c := New()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	if err := c.Connect(ctx, url, nil, nil); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	msgChan := make(chan WSClientMessage, 10)
	go c.ReadMessages(ctx, msgChan)

	expected := []struct {
		content string
		isJSON  bool
	}{
		{`{"status":"ok"}`, true},
		{`[1,2,3]`, true},
		{`hello world`, false},
	}

	for _, exp := range expected {
		select {
		case msg := <-msgChan:
			if msg.Err != nil {
				t.Fatalf("received error: %v", msg.Err)
			}
			if msg.Content != exp.content {
				t.Errorf("expected %q, got %q", exp.content, msg.Content)
			}
			if msg.IsJSON != exp.isJSON {
				t.Errorf("for %q: expected IsJSON=%v, got %v", exp.content, exp.isJSON, msg.IsJSON)
			}
		case <-time.After(3 * time.Second):
			t.Fatalf("timed out waiting for message %q", exp.content)
		}
	}

	c.Close()
}

func TestConnectWithBearerAuth(t *testing.T) {
	var receivedAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			InsecureSkipVerify: true,
		})
		if err != nil {
			return
		}
		conn.Close(websocket.StatusNormalClosure, "done")
	}))
	defer srv.Close()

	c := New()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	err := c.Connect(ctx, url, nil, &protocol.AuthConfig{
		Type:  "bearer",
		Token: "my-secret-token",
	})
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer c.Close()

	if receivedAuth != "Bearer my-secret-token" {
		t.Errorf("expected 'Bearer my-secret-token', got %q", receivedAuth)
	}
}

func TestConnectWithBasicAuth(t *testing.T) {
	var receivedAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			InsecureSkipVerify: true,
		})
		if err != nil {
			return
		}
		conn.Close(websocket.StatusNormalClosure, "done")
	}))
	defer srv.Close()

	c := New()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	err := c.Connect(ctx, url, nil, &protocol.AuthConfig{
		Type:     "basic",
		Username: "admin",
		Password: "pass123",
	})
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer c.Close()

	if !strings.HasPrefix(receivedAuth, "Basic ") {
		t.Errorf("expected Basic auth header, got %q", receivedAuth)
	}
}

func TestConnectWithAPIKeyAuth(t *testing.T) {
	var receivedKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedKey = r.Header.Get("X-API-Key")
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			InsecureSkipVerify: true,
		})
		if err != nil {
			return
		}
		conn.Close(websocket.StatusNormalClosure, "done")
	}))
	defer srv.Close()

	c := New()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	err := c.Connect(ctx, url, nil, &protocol.AuthConfig{
		Type:     "apikey",
		APIKey:   "X-API-Key",
		APIValue: "secret-key-123",
	})
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer c.Close()

	if receivedKey != "secret-key-123" {
		t.Errorf("expected 'secret-key-123', got %q", receivedKey)
	}
}

func TestDoubleConnect(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	c := New()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := c.Connect(ctx, wsURL(srv), nil, nil); err != nil {
		t.Fatalf("first Connect failed: %v", err)
	}
	defer c.Close()

	if err := c.Connect(ctx, wsURL(srv), nil, nil); err == nil {
		t.Error("expected error on double connect")
	}
}

func TestCloseWhenNotConnected(t *testing.T) {
	c := New()
	// Close on a client that was never connected should be a no-op.
	if err := c.Close(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
