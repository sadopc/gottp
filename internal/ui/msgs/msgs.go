package msgs

import (
	"net/http"
	"time"

	"github.com/sadopc/gottp/internal/core/collection"
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

	// Post-script results (attached if script ran)
	ScriptResult *ScriptResultMsg
	ScriptErr    *string
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

// CopyAsCurlMsg triggers copying the current request as cURL.
type CopyAsCurlMsg struct{}

// ImportCurlMsg triggers importing a request from clipboard cURL.
type ImportCurlMsg struct{}

// OpenEditorMsg triggers opening $EDITOR for body editing.
type OpenEditorMsg struct{}

// EditorDoneMsg is emitted when $EDITOR exits with new content.
type EditorDoneMsg struct {
	Content string
}

// HistorySelectedMsg is emitted when a history entry is selected.
type HistorySelectedMsg struct {
	ID int64
}

// --- Phase 3A: Theme switching ---

// SwitchThemeMsg requests switching to a named theme.
type SwitchThemeMsg struct {
	Name string
}

// --- Phase 3B: OAuth2 ---

// OAuth2TokenMsg is emitted when an OAuth2 token is acquired.
type OAuth2TokenMsg struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
	Err          error
}

// OAuth2BrowserMsg requests opening the browser for OAuth2 auth code flow.
type OAuth2BrowserMsg struct {
	URL string
}

// --- Phase 3D: Importers ---

// ImportFileMsg triggers importing a collection from a file path.
type ImportFileMsg struct {
	Path string
}

// ImportCompleteMsg is emitted when an import finishes.
type ImportCompleteMsg struct {
	Collection *collection.Collection
	Err        error
}

// --- Phase 3E: Response Diffing ---

// SetBaselineMsg saves the current response body as the diff baseline.
type SetBaselineMsg struct{}

// ClearBaselineMsg removes the saved diff baseline.
type ClearBaselineMsg struct{}

// --- Phase 4: Multi-Protocol ---

// SwitchProtocolMsg requests switching the editor protocol form.
type SwitchProtocolMsg struct {
	Protocol string
}

// IntrospectMsg triggers GraphQL introspection.
type IntrospectMsg struct{}

// IntrospectionResultMsg carries GraphQL introspection results.
type IntrospectionResultMsg struct {
	Types []SchemaType
	Err   error
}

// SchemaType is a simplified GraphQL type for display.
type SchemaType struct {
	Name   string
	Fields []SchemaField
}

// SchemaField is a simplified GraphQL field for display.
type SchemaField struct {
	Name string
	Type string
}

// --- Phase 5: WebSocket ---

// WSConnectMsg requests a WebSocket connection.
type WSConnectMsg struct{}

// WSDisconnectMsg requests disconnecting WebSocket.
type WSDisconnectMsg struct{}

// WSSendMsg sends a message over WebSocket.
type WSSendMsg struct {
	Content string
}

// WSConnectedMsg is emitted when WebSocket connects.
type WSConnectedMsg struct {
	Err error
}

// WSDisconnectedMsg is emitted when WebSocket disconnects.
type WSDisconnectedMsg struct {
	Err error
}

// WSMessageReceivedMsg is emitted when a WebSocket message arrives.
type WSMessageReceivedMsg struct {
	Content   string
	IsJSON    bool
	Timestamp time.Time
}

// --- Phase 6: gRPC ---

// GRPCReflectMsg triggers gRPC server reflection.
type GRPCReflectMsg struct{}

// GRPCReflectionResultMsg carries gRPC reflection results.
type GRPCReflectionResultMsg struct {
	Services []GRPCServiceInfo
	Err      error
}

// GRPCServiceInfo holds discovered gRPC service metadata.
type GRPCServiceInfo struct {
	Name    string
	Methods []GRPCMethodInfo
}

// GRPCMethodInfo holds discovered gRPC method metadata.
type GRPCMethodInfo struct {
	Name           string
	FullName       string
	InputType      string
	OutputType     string
	IsClientStream bool
	IsServerStream bool
}

// --- Code Generation ---

// GenerateCodeMsg triggers code generation for the current request.
type GenerateCodeMsg struct {
	Language string // go, python, javascript, curl, ruby, java, rust, php
}

// --- Phase 7: Scripting ---

// ScriptResultMsg carries script execution results.
type ScriptResultMsg struct {
	Logs        []string
	TestResults []ScriptTestResult
	EnvChanges  map[string]string
	Err         error
}

// ScriptTestResult holds a single test assertion result.
type ScriptTestResult struct {
	Name   string
	Passed bool
	Error  string
}

// InsertTemplateMsg inserts a request template into a new tab.
type InsertTemplateMsg struct {
	TemplateName string
}
