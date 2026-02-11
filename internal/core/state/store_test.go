package state

import (
	"testing"

	"github.com/sadopc/gottp/internal/core/collection"
)

func TestNewStoreInitialState(t *testing.T) {
	s := NewStore()

	if s == nil {
		t.Fatal("NewStore() returned nil")
	}
	if s.EnvVars == nil {
		t.Fatal("EnvVars should be initialized")
	}
	if len(s.Tabs) != 0 {
		t.Fatalf("expected 0 tabs, got %d", len(s.Tabs))
	}
	if got := s.ActiveRequest(); got != nil {
		t.Fatalf("ActiveRequest() = %#v, want nil", got)
	}
}

func TestOpenRequestAddsAndActivatesTab(t *testing.T) {
	s := NewStore()
	req := &collection.Request{ID: "req-1", Name: "List Users"}

	s.OpenRequest(req)

	if len(s.Tabs) != 1 {
		t.Fatalf("expected 1 tab, got %d", len(s.Tabs))
	}
	if s.ActiveTab != 0 {
		t.Fatalf("ActiveTab = %d, want 0", s.ActiveTab)
	}
	if got := s.ActiveRequest(); got != req {
		t.Fatalf("ActiveRequest() = %#v, want %#v", got, req)
	}
}

func TestOpenRequestFocusesExistingTab(t *testing.T) {
	s := NewStore()
	req1 := &collection.Request{ID: "req-1", Name: "List Users"}
	req2 := &collection.Request{ID: "req-2", Name: "Create User"}

	s.OpenRequest(req1)
	s.OpenRequest(req2)
	s.OpenRequest(req1)

	if len(s.Tabs) != 2 {
		t.Fatalf("expected 2 tabs without duplicates, got %d", len(s.Tabs))
	}
	if s.ActiveTab != 0 {
		t.Fatalf("ActiveTab = %d, want 0 (existing req1 tab)", s.ActiveTab)
	}
}

func TestCloseTabNoopWhenEmpty(t *testing.T) {
	s := NewStore()

	s.CloseTab()

	if len(s.Tabs) != 0 {
		t.Fatalf("expected no tabs, got %d", len(s.Tabs))
	}
}

func TestCloseTabRemovesActiveAndAdjustsActiveIndex(t *testing.T) {
	s := NewStore()
	req1 := &collection.Request{ID: "req-1", Name: "One"}
	req2 := &collection.Request{ID: "req-2", Name: "Two"}
	req3 := &collection.Request{ID: "req-3", Name: "Three"}

	s.OpenRequest(req1)
	s.OpenRequest(req2)
	s.OpenRequest(req3)

	s.ActiveTab = 1
	s.CloseTab()

	if len(s.Tabs) != 2 {
		t.Fatalf("expected 2 tabs after close, got %d", len(s.Tabs))
	}
	if s.ActiveTab != 1 {
		t.Fatalf("ActiveTab = %d, want 1", s.ActiveTab)
	}
	if got := s.ActiveRequest(); got != req3 {
		t.Fatalf("ActiveRequest() = %#v, want req3", got)
	}

	s.CloseTab() // close req3
	if len(s.Tabs) != 1 {
		t.Fatalf("expected 1 tab, got %d", len(s.Tabs))
	}
	if s.ActiveTab != 0 {
		t.Fatalf("ActiveTab = %d, want 0", s.ActiveTab)
	}

	s.CloseTab() // close req1
	if len(s.Tabs) != 0 {
		t.Fatalf("expected 0 tabs, got %d", len(s.Tabs))
	}
	if s.ActiveTab != -1 {
		t.Fatalf("ActiveTab = %d, want -1 when no tabs remain", s.ActiveTab)
	}
}

func TestNewTabCreatesDefaultRequest(t *testing.T) {
	s := NewStore()

	s.NewTab()

	req := s.ActiveRequest()
	if req == nil {
		t.Fatal("ActiveRequest() is nil after NewTab()")
	}
	if req.ID == "" {
		t.Fatal("new request ID should not be empty")
	}
	if req.Name != "New Request" {
		t.Fatalf("Name = %q, want New Request", req.Name)
	}
	if req.Method != "GET" {
		t.Fatalf("Method = %q, want GET", req.Method)
	}
	if req.Protocol != "http" {
		t.Fatalf("Protocol = %q, want http", req.Protocol)
	}
}

func TestNextPrevTabWrapAround(t *testing.T) {
	s := NewStore()
	req1 := &collection.Request{ID: "req-1"}
	req2 := &collection.Request{ID: "req-2"}
	req3 := &collection.Request{ID: "req-3"}

	s.OpenRequest(req1)
	s.OpenRequest(req2)
	s.OpenRequest(req3)

	if s.ActiveTab != 2 {
		t.Fatalf("ActiveTab = %d, want 2", s.ActiveTab)
	}

	s.NextTab()
	if s.ActiveTab != 0 {
		t.Fatalf("NextTab() ActiveTab = %d, want 0", s.ActiveTab)
	}

	s.PrevTab()
	if s.ActiveTab != 2 {
		t.Fatalf("PrevTab() ActiveTab = %d, want 2", s.ActiveTab)
	}
}

func TestNextPrevTabNoopWhenEmpty(t *testing.T) {
	s := NewStore()

	s.NextTab()
	s.PrevTab()

	if s.ActiveTab != 0 {
		t.Fatalf("ActiveTab = %d, want unchanged zero value", s.ActiveTab)
	}
}
