package protocol

import (
	"context"
	"fmt"
)

// Registry manages protocol implementations.
type Registry struct {
	protocols map[string]Protocol
}

// NewRegistry creates a new protocol registry.
func NewRegistry() *Registry {
	return &Registry{
		protocols: make(map[string]Protocol),
	}
}

// Register adds a protocol implementation.
func (r *Registry) Register(p Protocol) {
	r.protocols[p.Name()] = p
}

// Get returns a protocol by name.
func (r *Registry) Get(name string) (Protocol, bool) {
	p, ok := r.protocols[name]
	return p, ok
}

// Execute dispatches a request to the appropriate protocol handler.
func (r *Registry) Execute(ctx context.Context, req *Request) (*Response, error) {
	proto := req.Protocol
	if proto == "" {
		proto = "http"
	}
	p, ok := r.protocols[proto]
	if !ok {
		return nil, fmt.Errorf("unknown protocol: %s", proto)
	}
	if err := p.Validate(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	return p.Execute(ctx, req)
}

// Names returns all registered protocol names.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.protocols))
	for name := range r.protocols {
		names = append(names, name)
	}
	return names
}
