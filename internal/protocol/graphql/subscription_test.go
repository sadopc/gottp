package graphql

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/sadopc/gottp/internal/protocol"
)

// --- isSubscription detection tests ---

func TestIsSubscription(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  bool
	}{
		{
			name:  "simple subscription",
			query: `subscription { messageAdded { text } }`,
			want:  true,
		},
		{
			name:  "subscription with name",
			query: `subscription OnMessage { messageAdded { text } }`,
			want:  true,
		},
		{
			name:  "subscription with leading whitespace",
			query: `   subscription { messageAdded { text } }`,
			want:  true,
		},
		{
			name:  "subscription with leading newlines",
			query: "\n\n  subscription { messageAdded { text } }",
			want:  true,
		},
		{
			name:  "subscription with comment",
			query: "# my subscription\nsubscription { messageAdded { text } }",
			want:  true,
		},
		{
			name:  "subscription with multiple comments",
			query: "# comment 1\n# comment 2\nsubscription { x }",
			want:  true,
		},
		{
			name:  "query operation",
			query: `query { users { name } }`,
			want:  false,
		},
		{
			name:  "mutation operation",
			query: `mutation { addUser(name: "test") { id } }`,
			want:  false,
		},
		{
			name:  "shorthand query",
			query: `{ users { name } }`,
			want:  false,
		},
		{
			name:  "empty query",
			query: ``,
			want:  false,
		},
		{
			name:  "only comment",
			query: `# just a comment`,
			want:  false,
		},
		{
			name:  "case insensitive",
			query: `Subscription { x }`,
			want:  true,
		},
		{
			name:  "subscription in body but not operation",
			query: `query { subscription }`,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSubscription(tt.query)
			if got != tt.want {
				t.Errorf("isSubscription(%q) = %v, want %v", tt.query, got, tt.want)
			}
		})
	}
}

// --- Protocol message format tests ---

func TestConnectionInitMessageFormat(t *testing.T) {
	msg := gqlWSMessage{
		Type:    msgConnectionInit,
		Payload: json.RawMessage(`{}`),
	}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed["type"] != "connection_init" {
		t.Errorf("expected type connection_init, got %v", parsed["type"])
	}
	if _, ok := parsed["id"]; ok {
		t.Error("connection_init should not have an id field")
	}
}

func TestSubscribeMessageFormat(t *testing.T) {
	payload := subscribePayload{
		Query:     "subscription { messageAdded { text } }",
		Variables: map[string]interface{}{"channel": "general"},
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := gqlWSMessage{
		ID:      "1",
		Type:    msgSubscribe,
		Payload: json.RawMessage(payloadBytes),
	}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed["type"] != "subscribe" {
		t.Errorf("expected type subscribe, got %v", parsed["type"])
	}
	if parsed["id"] != "1" {
		t.Errorf("expected id 1, got %v", parsed["id"])
	}

	payloadMap, ok := parsed["payload"].(map[string]interface{})
	if !ok {
		t.Fatal("expected payload to be a map")
	}
	if payloadMap["query"] != "subscription { messageAdded { text } }" {
		t.Errorf("unexpected query in payload: %v", payloadMap["query"])
	}
	vars, ok := payloadMap["variables"].(map[string]interface{})
	if !ok {
		t.Fatal("expected variables in payload")
	}
	if vars["channel"] != "general" {
		t.Errorf("expected channel=general, got %v", vars["channel"])
	}
}

func TestCompleteMessageFormat(t *testing.T) {
	msg := gqlWSMessage{
		ID:   "1",
		Type: msgComplete,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed["type"] != "complete" {
		t.Errorf("expected type complete, got %v", parsed["type"])
	}
	if parsed["id"] != "1" {
		t.Errorf("expected id 1, got %v", parsed["id"])
	}
}

// --- toWebSocketURL tests ---

func TestToWebSocketURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"http://localhost:4000/graphql", "ws://localhost:4000/graphql"},
		{"https://api.example.com/graphql", "wss://api.example.com/graphql"},
		{"ws://localhost:4000/graphql", "ws://localhost:4000/graphql"},
		{"wss://api.example.com/graphql", "wss://api.example.com/graphql"},
	}
	for _, tt := range tests {
		got := toWebSocketURL(tt.input)
		if got != tt.want {
			t.Errorf("toWebSocketURL(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- graphql-ws server helper ---

// newGraphQLWSServer creates a test server that speaks the graphql-ws protocol.
// After accepting a subscription, it sends `count` next messages with a short
// delay between each, then sends a complete message.
func newGraphQLWSServer(t *testing.T, count int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			InsecureSkipVerify: true,
			Subprotocols:       []string{graphqlWSSubprotocol},
		})
		if err != nil {
			t.Logf("accept error: %v", err)
			return
		}
		defer conn.CloseNow()

		// Use a bounded context so server goroutines don't leak in tests.
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		// Read connection_init.
		_, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		var initMsg gqlWSMessage
		json.Unmarshal(data, &initMsg)
		if initMsg.Type != msgConnectionInit {
			return
		}

		// Send connection_ack.
		ackBytes, _ := json.Marshal(gqlWSMessage{Type: msgConnectionAck})
		if err := conn.Write(ctx, websocket.MessageText, ackBytes); err != nil {
			return
		}

		// Read subscribe (may fail if client disconnects early).
		_, data, err = conn.Read(ctx)
		if err != nil {
			return
		}
		var subMsg gqlWSMessage
		json.Unmarshal(data, &subMsg)
		if subMsg.Type != msgSubscribe {
			return
		}

		// Send `count` next messages.
		for i := 0; i < count; i++ {
			payload, _ := json.Marshal(map[string]interface{}{
				"data": map[string]interface{}{
					"messageAdded": map[string]interface{}{
						"text": "hello",
						"seq":  i + 1,
					},
				},
			})
			nextMsg := gqlWSMessage{
				ID:      subMsg.ID,
				Type:    msgNext,
				Payload: json.RawMessage(payload),
			}
			nextBytes, _ := json.Marshal(nextMsg)
			if err := conn.Write(ctx, websocket.MessageText, nextBytes); err != nil {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}

		// Send complete.
		completeBytes, _ := json.Marshal(gqlWSMessage{
			ID:   subMsg.ID,
			Type: msgComplete,
		})
		conn.Write(ctx, websocket.MessageText, completeBytes)

		// Keep connection open briefly so client can process.
		time.Sleep(50 * time.Millisecond)
	}))
}

func wsURLFromHTTP(srv *httptest.Server) string {
	return "ws" + strings.TrimPrefix(srv.URL, "http")
}

// --- Connection lifecycle tests ---

func TestSubscriptionConnectAndClose(t *testing.T) {
	srv := newGraphQLWSServer(t, 0)
	defer srv.Close()

	sc := NewSubscriptionClient()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if sc.IsConnected() {
		t.Error("expected not connected before Connect")
	}

	err := sc.Connect(ctx, wsURLFromHTTP(srv), nil)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	if !sc.IsConnected() {
		t.Error("expected connected after Connect")
	}

	err = sc.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if sc.IsConnected() {
		t.Error("expected not connected after Close")
	}
}

func TestSubscriptionDoubleConnect(t *testing.T) {
	srv := newGraphQLWSServer(t, 0)
	defer srv.Close()

	sc := NewSubscriptionClient()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := sc.Connect(ctx, wsURLFromHTTP(srv), nil)
	if err != nil {
		t.Fatalf("first Connect failed: %v", err)
	}
	defer sc.Close()

	err = sc.Connect(ctx, wsURLFromHTTP(srv), nil)
	if err == nil {
		t.Error("expected error on double connect")
	}
}

func TestSubscriptionCloseWhenNotConnected(t *testing.T) {
	sc := NewSubscriptionClient()
	if err := sc.Close(); err != nil {
		t.Errorf("unexpected error closing unconnected client: %v", err)
	}
}

// --- Subscription streaming test ---

func TestSubscriptionReceivesMessages(t *testing.T) {
	messageCount := 3
	srv := newGraphQLWSServer(t, messageCount)
	defer srv.Close()

	sc := NewSubscriptionClient()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := sc.Connect(ctx, wsURLFromHTTP(srv), nil)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer sc.Close()

	msgChan := make(chan protocol.StreamMessage, 20)
	go func() {
		if err := sc.Subscribe(ctx, "subscription { messageAdded { text } }", "", msgChan); err != nil {
			t.Logf("Subscribe returned: %v", err)
		}
	}()

	// First message should be the "sent" subscribe payload.
	select {
	case msg := <-msgChan:
		if msg.Direction != "sent" {
			t.Errorf("first message direction: expected sent, got %s", msg.Direction)
		}
		if !msg.IsJSON {
			t.Error("expected sent message to be JSON")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for sent message")
	}

	// Receive the `next` messages.
	received := 0
	for received < messageCount {
		select {
		case msg := <-msgChan:
			if msg.Err != nil {
				t.Fatalf("received error: %v", msg.Err)
			}
			if msg.Direction != "received" {
				t.Errorf("expected direction received, got %s", msg.Direction)
			}
			if !msg.IsJSON {
				t.Error("expected next message to be JSON")
			}
			// Verify the payload contains subscription data.
			var payload map[string]interface{}
			if err := json.Unmarshal([]byte(msg.Content), &payload); err != nil {
				t.Errorf("failed to parse message content: %v", err)
			}
			if _, ok := payload["data"]; !ok {
				t.Error("expected data field in message")
			}
			received++
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out after %d messages", received)
		}
	}

	if received != messageCount {
		t.Errorf("expected %d messages, got %d", messageCount, received)
	}

	// Channel should be closed after complete.
	select {
	case <-msgChan:
		// Might get one more msg before close, drain it.
	case <-time.After(3 * time.Second):
		t.Log("channel not closed yet, which is OK if still draining")
	}
}

func TestSubscribeWithoutConnect(t *testing.T) {
	sc := NewSubscriptionClient()
	msgChan := make(chan protocol.StreamMessage, 1)
	err := sc.Subscribe(context.Background(), "subscription { x }", "", msgChan)
	if err == nil {
		t.Error("expected error subscribing without connection")
	}
}

// --- Client integration tests ---

func TestClientConnectSubscription(t *testing.T) {
	srv := newGraphQLWSServer(t, 1)
	defer srv.Close()

	client := New()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if client.IsSubscriptionConnected() {
		t.Error("should not be connected initially")
	}

	err := client.ConnectSubscription(ctx, wsURLFromHTTP(srv), nil)
	if err != nil {
		t.Fatalf("ConnectSubscription failed: %v", err)
	}

	if !client.IsSubscriptionConnected() {
		t.Error("should be connected after ConnectSubscription")
	}

	err = client.CloseSubscription()
	if err != nil {
		t.Fatalf("CloseSubscription failed: %v", err)
	}

	if client.IsSubscriptionConnected() {
		t.Error("should not be connected after CloseSubscription")
	}
}

func TestClientIsSubscriptionQuery(t *testing.T) {
	client := New()
	if !client.IsSubscriptionQuery("subscription { x }") {
		t.Error("expected true for subscription query")
	}
	if client.IsSubscriptionQuery("query { x }") {
		t.Error("expected false for query")
	}
}

func TestExecuteDetectsSubscription(t *testing.T) {
	client := New()
	ctx := context.Background()

	resp, err := client.Execute(ctx, &protocol.Request{
		Protocol:     "graphql",
		URL:          "http://localhost:4000/graphql",
		GraphQLQuery: "subscription { messageAdded { text } }",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 101 {
		t.Errorf("expected 101, got %d", resp.StatusCode)
	}
	if resp.Proto != "graphql-ws" {
		t.Errorf("expected proto graphql-ws, got %s", resp.Proto)
	}
}

func TestClientSubscribeWithoutConnect(t *testing.T) {
	client := New()
	msgChan := make(chan protocol.StreamMessage, 1)
	err := client.Subscribe(context.Background(), "subscription { x }", "", msgChan)
	if err == nil {
		t.Error("expected error subscribing without connection")
	}
}

// --- Error message test ---

func TestSubscriptionServerError(t *testing.T) {
	// Server that sends an error after subscribe.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			InsecureSkipVerify: true,
			Subprotocols:       []string{graphqlWSSubprotocol},
		})
		if err != nil {
			return
		}
		defer conn.CloseNow()

		ctx := r.Context()

		// Read connection_init.
		conn.Read(ctx)

		// Send connection_ack.
		ackBytes, _ := json.Marshal(gqlWSMessage{Type: msgConnectionAck})
		conn.Write(ctx, websocket.MessageText, ackBytes)

		// Read subscribe.
		_, data, _ := conn.Read(ctx)
		var subMsg gqlWSMessage
		json.Unmarshal(data, &subMsg)

		// Send error.
		errPayload, _ := json.Marshal([]map[string]string{
			{"message": "subscription not supported"},
		})
		errMsg := gqlWSMessage{
			ID:      subMsg.ID,
			Type:    msgError,
			Payload: json.RawMessage(errPayload),
		}
		errBytes, _ := json.Marshal(errMsg)
		conn.Write(ctx, websocket.MessageText, errBytes)

		time.Sleep(50 * time.Millisecond)
	}))
	defer srv.Close()

	sc := NewSubscriptionClient()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := sc.Connect(ctx, wsURLFromHTTP(srv), nil)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer sc.Close()

	msgChan := make(chan protocol.StreamMessage, 10)
	err = sc.Subscribe(ctx, "subscription { x }", "", msgChan)
	if err == nil {
		t.Error("expected error from server error message")
	}
	if !strings.Contains(err.Error(), "subscription error") {
		t.Errorf("expected subscription error, got: %v", err)
	}
}

// --- Subscribe with variables test ---

func TestSubscribePayloadWithVariables(t *testing.T) {
	payload := subscribePayload{
		Query: "subscription ($ch: String!) { messages(channel: $ch) { text } }",
	}

	vars := `{"ch": "general"}`
	var parsedVars map[string]interface{}
	if err := json.Unmarshal([]byte(vars), &parsedVars); err == nil {
		payload.Variables = parsedVars
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var result map[string]interface{}
	json.Unmarshal(data, &result)

	if result["query"] != payload.Query {
		t.Errorf("query mismatch")
	}
	variables, ok := result["variables"].(map[string]interface{})
	if !ok {
		t.Fatal("expected variables map")
	}
	if variables["ch"] != "general" {
		t.Errorf("expected ch=general, got %v", variables["ch"])
	}
}
