package protocol

import (
	"context"
	"errors"
	"sort"
	"strings"
	"testing"
)

type stubProtocol struct {
	name        string
	validateErr error
	executeResp *Response
	executeErr  error

	validateCalls int
	executeCalls  int
}

func (s *stubProtocol) Name() string { return s.name }

func (s *stubProtocol) Execute(context.Context, *Request) (*Response, error) {
	s.executeCalls++
	return s.executeResp, s.executeErr
}

func (s *stubProtocol) Validate(*Request) error {
	s.validateCalls++
	return s.validateErr
}

func TestRegistryRegisterGetAndNames(t *testing.T) {
	r := NewRegistry()
	httpProtocol := &stubProtocol{name: "http"}
	grpcProtocol := &stubProtocol{name: "grpc"}

	r.Register(httpProtocol)
	r.Register(grpcProtocol)

	if got, ok := r.Get("http"); !ok || got != httpProtocol {
		t.Fatalf("Get(http) = (%v, %v), want registered protocol", got, ok)
	}

	names := r.Names()
	sort.Strings(names)
	if len(names) != 2 || names[0] != "grpc" || names[1] != "http" {
		t.Fatalf("Names() = %v, want [grpc http]", names)
	}
}

func TestRegistryRegisterOverridesExistingProtocol(t *testing.T) {
	r := NewRegistry()
	first := &stubProtocol{name: "http"}
	second := &stubProtocol{name: "http"}

	r.Register(first)
	r.Register(second)

	got, ok := r.Get("http")
	if !ok {
		t.Fatal("expected protocol to be present")
	}
	if got != second {
		t.Fatalf("expected second registration to overwrite first, got %T", got)
	}
}

func TestRegistryExecuteUsesDefaultHTTPProtocol(t *testing.T) {
	r := NewRegistry()
	httpProtocol := &stubProtocol{name: "http", executeResp: &Response{StatusCode: 204}}
	r.Register(httpProtocol)

	resp, err := r.Execute(context.Background(), &Request{Method: "GET", URL: "https://example.com"})
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
	if resp == nil || resp.StatusCode != 204 {
		t.Fatalf("Execute() response = %#v, want status 204", resp)
	}
	if httpProtocol.validateCalls != 1 {
		t.Fatalf("validate called %d times, want 1", httpProtocol.validateCalls)
	}
	if httpProtocol.executeCalls != 1 {
		t.Fatalf("execute called %d times, want 1", httpProtocol.executeCalls)
	}
}

func TestRegistryExecuteUnknownProtocol(t *testing.T) {
	r := NewRegistry()

	_, err := r.Execute(context.Background(), &Request{Protocol: "smtp"})
	if err == nil {
		t.Fatal("expected error for unknown protocol")
	}
	if !strings.Contains(err.Error(), "unknown protocol: smtp") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRegistryExecuteReturnsValidationErrors(t *testing.T) {
	r := NewRegistry()
	httpProtocol := &stubProtocol{name: "http", validateErr: errors.New("missing url")}
	r.Register(httpProtocol)

	_, err := r.Execute(context.Background(), &Request{Protocol: "http"})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "validation failed: missing url") {
		t.Fatalf("unexpected validation error: %v", err)
	}
	if httpProtocol.executeCalls != 0 {
		t.Fatalf("execute should not be called when validation fails; got %d", httpProtocol.executeCalls)
	}
}

func TestRegistryExecuteReturnsProtocolExecutionError(t *testing.T) {
	r := NewRegistry()
	wantErr := errors.New("network down")
	httpProtocol := &stubProtocol{name: "http", executeErr: wantErr}
	r.Register(httpProtocol)

	_, err := r.Execute(context.Background(), &Request{Protocol: "http", Method: "GET", URL: "https://example.com"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}
	if httpProtocol.validateCalls != 1 || httpProtocol.executeCalls != 1 {
		t.Fatalf("unexpected calls validate=%d execute=%d", httpProtocol.validateCalls, httpProtocol.executeCalls)
	}
}
