package state

import (
	"github.com/serdar/gottp/internal/core/collection"
)

// OpenTab represents an open request tab.
type OpenTab struct {
	Request  *collection.Request
	Modified bool
}

// Store holds the central application state.
type Store struct {
	Collection    *collection.Collection
	CollectionPath string

	ActiveEnv     string
	EnvVars       map[string]string

	Tabs          []OpenTab
	ActiveTab     int
}

// NewStore creates a new state store.
func NewStore() *Store {
	return &Store{
		EnvVars: make(map[string]string),
	}
}

// ActiveRequest returns the currently active request, or nil.
func (s *Store) ActiveRequest() *collection.Request {
	if s.ActiveTab >= 0 && s.ActiveTab < len(s.Tabs) {
		return s.Tabs[s.ActiveTab].Request
	}
	return nil
}

// OpenRequest opens a request in a new tab or focuses it if already open.
func (s *Store) OpenRequest(req *collection.Request) {
	// Check if already open
	for i, tab := range s.Tabs {
		if tab.Request.ID == req.ID {
			s.ActiveTab = i
			return
		}
	}
	// Open new tab
	s.Tabs = append(s.Tabs, OpenTab{Request: req})
	s.ActiveTab = len(s.Tabs) - 1
}

// CloseTab closes the active tab.
func (s *Store) CloseTab() {
	if len(s.Tabs) == 0 {
		return
	}
	s.Tabs = append(s.Tabs[:s.ActiveTab], s.Tabs[s.ActiveTab+1:]...)
	if s.ActiveTab >= len(s.Tabs) {
		s.ActiveTab = len(s.Tabs) - 1
	}
}

// NewTab creates a new empty request tab.
func (s *Store) NewTab() {
	req := collection.NewRequest("New Request", "GET", "")
	s.Tabs = append(s.Tabs, OpenTab{Request: req})
	s.ActiveTab = len(s.Tabs) - 1
}

// NextTab switches to the next tab.
func (s *Store) NextTab() {
	if len(s.Tabs) == 0 {
		return
	}
	s.ActiveTab = (s.ActiveTab + 1) % len(s.Tabs)
}

// PrevTab switches to the previous tab.
func (s *Store) PrevTab() {
	if len(s.Tabs) == 0 {
		return
	}
	s.ActiveTab = (s.ActiveTab - 1 + len(s.Tabs)) % len(s.Tabs)
}
