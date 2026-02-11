package msgs

import (
	"net/http"
	"time"
)

// Panel focus targets
type PanelFocus int

const (
	FocusSidebar PanelFocus = iota
	FocusEditor
	FocusResponse
)

// AppMode represents the current input mode.
type AppMode int

const (
	ModeNormal AppMode = iota
	ModeInsert
	ModeCommandPalette
	ModeJump
	ModeModal
	ModeSearch
)

func (m AppMode) String() string {
	switch m {
	case ModeNormal:
		return "NORMAL"
	case ModeInsert:
		return "INSERT"
	case ModeCommandPalette:
		return "COMMAND"
	case ModeJump:
		return "JUMP"
	case ModeModal:
		return "MODAL"
	case ModeSearch:
		return "SEARCH"
	default:
		return "UNKNOWN"
	}
}

// FocusPanelMsg requests focus change to a specific panel.
type FocusPanelMsg struct {
	Panel PanelFocus
}

// CycleFocusMsg cycles focus to the next/previous panel.
type CycleFocusMsg struct {
	Reverse bool
}

// ToggleSidebarMsg toggles sidebar visibility.
type ToggleSidebarMsg struct{}

// SendRequestMsg triggers sending the current request.
type SendRequestMsg struct{}

// RequestSentMsg is emitted when a request completes.
type RequestSentMsg struct {
	StatusCode  int
	Status      string
	Headers     http.Header
	Body        []byte
	ContentType string
	Duration    time.Duration
	Size        int64
	Err         error
}

// NewRequestMsg opens a new empty request tab.
type NewRequestMsg struct{}

// CloseTabMsg closes the current tab.
type CloseTabMsg struct{}

// SwitchTabMsg switches to a specific tab.
type SwitchTabMsg struct {
	Index int
}

// NextTabMsg / PrevTabMsg for tab navigation.
type NextTabMsg struct{}
type PrevTabMsg struct{}

// SaveRequestMsg saves the current request.
type SaveRequestMsg struct{}

// OpenCommandPaletteMsg opens the command palette.
type OpenCommandPaletteMsg struct{}

// ShowHelpMsg toggles the help overlay.
type ShowHelpMsg struct{}

// SetModeMsg changes the app mode.
type SetModeMsg struct {
	Mode AppMode
}

// StatusMsg sets a temporary status bar message.
type StatusMsg struct {
	Text     string
	Duration time.Duration
}

// ToastMsg shows a toast notification.
type ToastMsg struct {
	Text     string
	Duration time.Duration
	IsError  bool
}

// RequestSelectedMsg is emitted when a request is selected from the sidebar.
type RequestSelectedMsg struct {
	RequestID string
}

// CollectionLoadedMsg is emitted when a collection is loaded.
type CollectionLoadedMsg struct {
	Err error
}

// SwitchEnvMsg switches the active environment.
type SwitchEnvMsg struct {
	Name string
}
