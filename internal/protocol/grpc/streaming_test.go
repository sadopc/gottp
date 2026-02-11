package grpc

import (
	"bytes"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/sadopc/gottp/internal/protocol"
)

func TestStreamMessageType(t *testing.T) {
	msg := protocol.StreamMessage{
		Content:   `{"name": "test"}`,
		IsJSON:    true,
		Timestamp: time.Now(),
		Direction: "received",
	}

	if msg.Content != `{"name": "test"}` {
		t.Errorf("expected content to be preserved, got %q", msg.Content)
	}
	if !msg.IsJSON {
		t.Error("expected IsJSON to be true")
	}
	if msg.Direction != "received" {
		t.Errorf("expected direction 'received', got %q", msg.Direction)
	}
	if msg.Err != nil {
		t.Errorf("expected no error, got %v", msg.Err)
	}
}

func TestStreamMessageWithError(t *testing.T) {
	msg := protocol.StreamMessage{
		Timestamp: time.Now(),
		Direction: "received",
		Err:       status.Error(codes.Unavailable, "connection lost"),
	}

	if msg.Err == nil {
		t.Fatal("expected error to be set")
	}
	if msg.Content != "" {
		t.Errorf("expected empty content with error, got %q", msg.Content)
	}
}

func TestStreamMessageDirections(t *testing.T) {
	sent := protocol.StreamMessage{
		Content:   `{"msg": "hello"}`,
		IsJSON:    true,
		Timestamp: time.Now(),
		Direction: "sent",
	}
	received := protocol.StreamMessage{
		Content:   `{"msg": "world"}`,
		IsJSON:    true,
		Timestamp: time.Now(),
		Direction: "received",
	}

	if sent.Direction != "sent" {
		t.Errorf("expected 'sent', got %q", sent.Direction)
	}
	if received.Direction != "received" {
		t.Errorf("expected 'received', got %q", received.Direction)
	}
}

// fakeMessage implements proto.Message for testing the response handler.
type fakeMessage struct {
	data string
}

func (f *fakeMessage) ProtoMessage()            {}
func (f *fakeMessage) Reset()                   {}
func (f *fakeMessage) String() string           { return f.data }
func (f *fakeMessage) Marshal() ([]byte, error) { return []byte(f.data), nil }

func TestResponseHandlerNonStreaming(t *testing.T) {
	var buf bytes.Buffer
	handler := &responseHandler{
		out: &buf,
		formatter: func(msg proto.Message) (string, error) {
			return `{"result": "ok"}`, nil
		},
		streaming: false,
	}

	handler.OnReceiveResponse(&fakeMessage{data: "test"})

	if handler.numResponses != 1 {
		t.Errorf("expected 1 response, got %d", handler.numResponses)
	}
	if buf.String() != `{"result": "ok"}` {
		t.Errorf("expected buffered output, got %q", buf.String())
	}
}

func TestResponseHandlerStreaming(t *testing.T) {
	msgChan := make(chan protocol.StreamMessage, 10)
	handler := &responseHandler{
		out: &bytes.Buffer{},
		formatter: func(msg proto.Message) (string, error) {
			return `{"value": 42}`, nil
		},
		streaming: true,
		msgChan:   msgChan,
	}

	// Simulate three streamed responses.
	handler.OnReceiveResponse(&fakeMessage{})
	handler.OnReceiveResponse(&fakeMessage{})
	handler.OnReceiveResponse(&fakeMessage{})

	if handler.numResponses != 3 {
		t.Errorf("expected 3 responses, got %d", handler.numResponses)
	}

	// Check that all three messages were sent to the channel.
	for i := 0; i < 3; i++ {
		select {
		case sm := <-msgChan:
			if sm.Content != `{"value": 42}` {
				t.Errorf("message %d: expected content %q, got %q", i, `{"value": 42}`, sm.Content)
			}
			if !sm.IsJSON {
				t.Errorf("message %d: expected IsJSON true", i)
			}
			if sm.Direction != "received" {
				t.Errorf("message %d: expected direction 'received', got %q", i, sm.Direction)
			}
			if sm.Err != nil {
				t.Errorf("message %d: unexpected error: %v", i, sm.Err)
			}
		default:
			t.Errorf("expected message %d on channel", i)
		}
	}
}

func TestResponseHandlerStreamingNonJSON(t *testing.T) {
	msgChan := make(chan protocol.StreamMessage, 10)
	handler := &responseHandler{
		out: &bytes.Buffer{},
		formatter: func(msg proto.Message) (string, error) {
			return "plain text response", nil
		},
		streaming: true,
		msgChan:   msgChan,
	}

	handler.OnReceiveResponse(&fakeMessage{})

	select {
	case sm := <-msgChan:
		if sm.IsJSON {
			t.Error("expected IsJSON false for plain text")
		}
		if sm.Content != "plain text response" {
			t.Errorf("unexpected content: %q", sm.Content)
		}
	default:
		t.Error("expected message on channel")
	}
}

func TestResponseHandlerTrailers(t *testing.T) {
	handler := &responseHandler{
		out: &bytes.Buffer{},
		formatter: func(msg proto.Message) (string, error) {
			return "{}", nil
		},
	}

	trailers := metadata.MD{"x-trailer": []string{"value"}}
	stat := status.New(codes.OK, "")
	handler.OnReceiveTrailers(stat, trailers)

	if handler.status.Code() != codes.OK {
		t.Errorf("expected OK status, got %v", handler.status.Code())
	}
	if vals := handler.responseTrailers.Get("x-trailer"); len(vals) == 0 || vals[0] != "value" {
		t.Errorf("expected trailer x-trailer=value, got %v", handler.responseTrailers)
	}
}

func TestResponseHandlerHeaders(t *testing.T) {
	handler := &responseHandler{
		out: &bytes.Buffer{},
		formatter: func(msg proto.Message) (string, error) {
			return "{}", nil
		},
	}

	headers := metadata.MD{"content-type": []string{"application/grpc"}}
	handler.OnReceiveHeaders(headers)

	if vals := handler.responseHeaders.Get("content-type"); len(vals) == 0 || vals[0] != "application/grpc" {
		t.Errorf("expected content-type header, got %v", handler.responseHeaders)
	}
}

func TestSendStreamMessageNoActiveStream(t *testing.T) {
	client := New()
	err := client.SendStreamMessage(`{"test": true}`)
	if err == nil {
		t.Fatal("expected error when no active stream")
	}
	if err.Error() != "no active client stream" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCloseStreamNoActiveStream(t *testing.T) {
	client := New()
	err := client.CloseStream()
	if err == nil {
		t.Fatal("expected error when no active stream")
	}
	if err.Error() != "no active client stream" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSendStreamMessageWithChannel(t *testing.T) {
	client := New()

	// Simulate an active stream by setting up the input channel.
	client.streamMu.Lock()
	client.streamInput = make(chan string, 16)
	client.streamDone = make(chan struct{})
	client.streamMu.Unlock()

	err := client.SendStreamMessage(`{"message": "hello"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the message was sent to the channel.
	select {
	case msg := <-client.streamInput:
		if msg != `{"message": "hello"}` {
			t.Errorf("unexpected message: %q", msg)
		}
	default:
		t.Fatal("expected message on streamInput channel")
	}

	// Clean up.
	close(client.streamInput)
	close(client.streamDone)
	client.streamMu.Lock()
	client.streamInput = nil
	client.streamDone = nil
	client.streamMu.Unlock()
}

func TestResponseHandlerStreamingMultipleMessages(t *testing.T) {
	msgChan := make(chan protocol.StreamMessage, 100)
	handler := &responseHandler{
		out: &bytes.Buffer{},
		formatter: func(msg proto.Message) (string, error) {
			return `{"seq": 1}`, nil
		},
		streaming: true,
		msgChan:   msgChan,
	}

	// Simulate a server-streaming RPC sending many responses.
	const messageCount = 50
	for i := 0; i < messageCount; i++ {
		handler.OnReceiveResponse(&fakeMessage{})
	}

	if handler.numResponses != messageCount {
		t.Errorf("expected %d responses, got %d", messageCount, handler.numResponses)
	}

	// Drain and count channel messages.
	count := 0
	for count < messageCount {
		select {
		case <-msgChan:
			count++
		default:
			t.Fatalf("expected %d messages, only got %d", messageCount, count)
		}
	}
}
