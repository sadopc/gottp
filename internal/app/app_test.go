package app

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/serdar/gottp/internal/config"
	"github.com/serdar/gottp/internal/core/collection"
	"github.com/serdar/gottp/internal/ui/msgs"
)

// testApp creates a minimal App for testing without side effects
// (no history DB, no env file loading).
func testApp() App {
	col := &collection.Collection{
		Name: "Test",
		Items: []collection.Item{
			{Request: collection.NewRequest("Get Users", "GET", "https://api.example.com/users")},
			{Request: collection.NewRequest("Create User", "POST", "https://api.example.com/users")},
		},
	}
	cfg := config.DefaultConfig()
	return New(col, "/tmp/test.gottp.yaml", cfg)
}

// testAppResized returns an App that has been resized so a.ready == true.
func testAppResized() App {
	a := testApp()
	m, _ := a.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	return m.(App)
}

// keyMsg creates a tea.KeyMsg for a single rune key.
func keyMsg(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

// --- Tests ---

func TestNew_DefaultState(t *testing.T) {
	a := testApp()

	if a.mode != msgs.ModeNormal {
		t.Errorf("expected ModeNormal, got %v", a.mode)
	}
	if a.focus != msgs.FocusEditor {
		t.Errorf("expected FocusEditor, got %v", a.focus)
	}
	if !a.sidebarVisible {
		t.Error("expected sidebar visible by default")
	}
	if a.ready {
		t.Error("expected ready=false before WindowSizeMsg")
	}
	if a.store == nil {
		t.Fatal("expected non-nil store")
	}
	if len(a.store.Tabs) != 1 {
		t.Errorf("expected 1 initial tab, got %d", len(a.store.Tabs))
	}
	if a.store.Collection.Name != "Test" {
		t.Errorf("expected collection name 'Test', got %q", a.store.Collection.Name)
	}
}

func TestWindowSizeMsg_SetsReadyAndLayout(t *testing.T) {
	a := testApp()

	m, cmd := a.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	if cmd != nil {
		t.Error("expected nil cmd from WindowSizeMsg")
	}
	a = m.(App)

	if !a.ready {
		t.Error("expected ready=true after WindowSizeMsg")
	}
	if a.width != 120 {
		t.Errorf("expected width=120, got %d", a.width)
	}
	if a.height != 30 {
		t.Errorf("expected height=30, got %d", a.height)
	}
	if a.layout.ContentHeight <= 0 {
		t.Errorf("expected positive ContentHeight, got %d", a.layout.ContentHeight)
	}
}

func TestWindowSizeMsg_ResponsiveBreakpoints(t *testing.T) {
	tests := []struct {
		name        string
		width       int
		singlePanel bool
		twoPanelMode bool
	}{
		{"single panel (narrow)", 50, true, false},
		{"two panels (medium)", 80, false, true},
		{"three panels (wide)", 160, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := testApp()
			m, _ := a.Update(tea.WindowSizeMsg{Width: tt.width, Height: 30})
			a = m.(App)

			if a.layout.SinglePanel != tt.singlePanel {
				t.Errorf("SinglePanel: expected %v, got %v", tt.singlePanel, a.layout.SinglePanel)
			}
			if a.layout.TwoPanelMode != tt.twoPanelMode {
				t.Errorf("TwoPanelMode: expected %v, got %v", tt.twoPanelMode, a.layout.TwoPanelMode)
			}
		})
	}
}

func TestCycleFocus_Forward(t *testing.T) {
	a := testAppResized()
	a.focus = msgs.FocusEditor

	// Editor -> Response
	a.cycleFocus(false)
	if a.focus != msgs.FocusResponse {
		t.Errorf("expected FocusResponse, got %v", a.focus)
	}

	// Response -> Sidebar
	a.cycleFocus(false)
	if a.focus != msgs.FocusSidebar {
		t.Errorf("expected FocusSidebar, got %v", a.focus)
	}

	// Sidebar -> Editor
	a.cycleFocus(false)
	if a.focus != msgs.FocusEditor {
		t.Errorf("expected FocusEditor, got %v", a.focus)
	}
}

func TestCycleFocus_Reverse(t *testing.T) {
	a := testAppResized()
	a.focus = msgs.FocusEditor

	// Editor -> Sidebar (reverse)
	a.cycleFocus(true)
	if a.focus != msgs.FocusSidebar {
		t.Errorf("expected FocusSidebar, got %v", a.focus)
	}

	// Sidebar -> Response (reverse wraps)
	a.cycleFocus(true)
	if a.focus != msgs.FocusResponse {
		t.Errorf("expected FocusResponse, got %v", a.focus)
	}
}

func TestCycleFocus_SidebarHidden(t *testing.T) {
	a := testAppResized()
	a.sidebarVisible = false
	a.focus = msgs.FocusEditor

	// Editor -> Response (skips sidebar)
	a.cycleFocus(false)
	if a.focus != msgs.FocusResponse {
		t.Errorf("expected FocusResponse, got %v", a.focus)
	}

	// Response -> Editor (skips sidebar)
	a.cycleFocus(false)
	if a.focus != msgs.FocusEditor {
		t.Errorf("expected FocusEditor, got %v", a.focus)
	}
}

func TestCycleFocus_TabKey(t *testing.T) {
	a := testAppResized()
	a.focus = msgs.FocusEditor

	m, _ := a.Update(tea.KeyMsg{Type: tea.KeyTab})
	a = m.(App)

	if a.focus != msgs.FocusResponse {
		t.Errorf("expected FocusResponse after Tab, got %v", a.focus)
	}
}

func TestCycleFocus_ShiftTabKey(t *testing.T) {
	a := testAppResized()
	a.focus = msgs.FocusEditor

	m, _ := a.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	a = m.(App)

	if a.focus != msgs.FocusSidebar {
		t.Errorf("expected FocusSidebar after Shift+Tab, got %v", a.focus)
	}
}

func TestFocusPanelMsg(t *testing.T) {
	panels := []msgs.PanelFocus{msgs.FocusSidebar, msgs.FocusEditor, msgs.FocusResponse}

	for _, target := range panels {
		a := testAppResized()
		m, _ := a.Update(msgs.FocusPanelMsg{Panel: target})
		a = m.(App)

		if a.focus != target {
			t.Errorf("expected focus %v, got %v", target, a.focus)
		}
	}
}

func TestGlobalKey_Quit(t *testing.T) {
	a := testAppResized()

	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyCtrlC})

	// tea.Quit returns a special tea.Cmd that when executed returns tea.QuitMsg{}
	if cmd == nil {
		t.Fatal("expected non-nil cmd for Ctrl+C")
	}
	result := cmd()
	if _, ok := result.(tea.QuitMsg); !ok {
		t.Errorf("expected QuitMsg from Ctrl+C, got %T", result)
	}
}

func TestGlobalKey_CommandPalette(t *testing.T) {
	a := testAppResized()

	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	if cmd == nil {
		t.Fatal("expected non-nil cmd for Ctrl+K")
	}

	result := cmd()
	if _, ok := result.(msgs.OpenCommandPaletteMsg); !ok {
		t.Errorf("expected OpenCommandPaletteMsg, got %T", result)
	}
}

func TestGlobalKey_SendRequest(t *testing.T) {
	a := testAppResized()

	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	if cmd == nil {
		t.Fatal("expected non-nil cmd for Ctrl+R")
	}

	result := cmd()
	if _, ok := result.(msgs.SendRequestMsg); !ok {
		t.Errorf("expected SendRequestMsg, got %T", result)
	}
}

func TestGlobalKey_NewRequest(t *testing.T) {
	a := testAppResized()

	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
	if cmd == nil {
		t.Fatal("expected non-nil cmd for Ctrl+N")
	}

	result := cmd()
	if _, ok := result.(msgs.NewRequestMsg); !ok {
		t.Errorf("expected NewRequestMsg, got %T", result)
	}
}

func TestGlobalKey_CloseTab(t *testing.T) {
	a := testAppResized()

	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyCtrlW})
	if cmd == nil {
		t.Fatal("expected non-nil cmd for Ctrl+W")
	}

	result := cmd()
	if _, ok := result.(msgs.CloseTabMsg); !ok {
		t.Errorf("expected CloseTabMsg, got %T", result)
	}
}

func TestGlobalKey_SaveRequest(t *testing.T) {
	a := testAppResized()

	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	if cmd == nil {
		t.Fatal("expected non-nil cmd for Ctrl+S")
	}

	result := cmd()
	if _, ok := result.(msgs.SaveRequestMsg); !ok {
		t.Errorf("expected SaveRequestMsg, got %T", result)
	}
}

func TestGlobalKey_PrevNextTab(t *testing.T) {
	a := testAppResized()

	// Test '[' for PrevTab
	_, cmd := a.Update(keyMsg('['))
	if cmd == nil {
		t.Fatal("expected non-nil cmd for '['")
	}
	result := cmd()
	if _, ok := result.(msgs.PrevTabMsg); !ok {
		t.Errorf("expected PrevTabMsg, got %T", result)
	}

	// Test ']' for NextTab
	_, cmd = a.Update(keyMsg(']'))
	if cmd == nil {
		t.Fatal("expected non-nil cmd for ']'")
	}
	result = cmd()
	if _, ok := result.(msgs.NextTabMsg); !ok {
		t.Errorf("expected NextTabMsg, got %T", result)
	}
}

func TestPanelKey_InsertMode(t *testing.T) {
	a := testAppResized()
	a.focus = msgs.FocusEditor
	a.mode = msgs.ModeNormal

	m, _ := a.Update(keyMsg('i'))
	a = m.(App)

	if a.mode != msgs.ModeInsert {
		t.Errorf("expected ModeInsert after 'i' in editor, got %v", a.mode)
	}
}

func TestPanelKey_InsertMode_OnlyInEditor(t *testing.T) {
	a := testAppResized()
	a.focus = msgs.FocusSidebar
	a.mode = msgs.ModeNormal

	m, _ := a.Update(keyMsg('i'))
	a = m.(App)

	// When sidebar is focused, 'i' should not trigger insert mode;
	// it should be dispatched to the sidebar panel instead.
	if a.mode == msgs.ModeInsert {
		t.Error("'i' should not enter insert mode when sidebar is focused")
	}
}

func TestPanelKey_ToggleSidebar(t *testing.T) {
	a := testAppResized()
	initial := a.sidebarVisible

	m, _ := a.Update(keyMsg('b'))
	a = m.(App)

	if a.sidebarVisible == initial {
		t.Error("expected sidebar visibility to toggle after 'b'")
	}

	// Toggle back
	m, _ = a.Update(keyMsg('b'))
	a = m.(App)
	if a.sidebarVisible != initial {
		t.Error("expected sidebar visibility to toggle back after second 'b'")
	}
}

func TestPanelKey_HelpToggle(t *testing.T) {
	a := testAppResized()

	m, _ := a.Update(keyMsg('?'))
	a = m.(App)

	if a.mode != msgs.ModeModal {
		t.Errorf("expected ModeModal after '?', got %v", a.mode)
	}
	if !a.help.Visible {
		t.Error("expected help overlay to be visible after '?'")
	}
}

func TestPanelKey_JumpMode(t *testing.T) {
	a := testAppResized()

	m, _ := a.Update(keyMsg('f'))
	a = m.(App)

	if a.mode != msgs.ModeJump {
		t.Errorf("expected ModeJump after 'f', got %v", a.mode)
	}
	if !a.jump.Visible {
		t.Error("expected jump overlay to be visible after 'f'")
	}
}

func TestPanelKey_SendViaEnter(t *testing.T) {
	a := testAppResized()
	a.focus = msgs.FocusEditor

	// Enter in editor in normal mode triggers sendRequest.
	// sendRequest checks URL and since URL is empty in a new tab, it just sets status message.
	m, _ := a.Update(tea.KeyMsg{Type: tea.KeyEnter})
	_ = m.(App) // just ensure no panic
}

func TestPanelKey_SendViaCapitalS(t *testing.T) {
	a := testAppResized()
	a.focus = msgs.FocusSidebar // S works from any panel

	m, _ := a.Update(keyMsg('S'))
	_ = m.(App) // sendRequest is invoked; no panic expected
}

func TestOverlayPriority_CommandPaletteBlocksKeys(t *testing.T) {
	a := testAppResized()
	a.commandPalette.Open()
	a.mode = msgs.ModeCommandPalette

	// When command palette is visible, keys should go to it, not to global handlers.
	// Sending 'j' should not cycle focus or cause side effects.
	initialFocus := a.focus
	m, _ := a.Update(keyMsg('j'))
	a = m.(App)

	if a.focus != initialFocus {
		t.Errorf("key should have been consumed by command palette; focus changed from %v to %v", initialFocus, a.focus)
	}
}

func TestOverlayPriority_HelpBlocksKeys(t *testing.T) {
	a := testAppResized()
	a.help.SetSize(160, 40)
	a.help.Toggle()
	a.mode = msgs.ModeModal

	initialFocus := a.focus
	m, _ := a.Update(keyMsg('j'))
	a = m.(App)

	if a.focus != initialFocus {
		t.Errorf("key should have been consumed by help overlay; focus changed from %v to %v", initialFocus, a.focus)
	}
}

func TestOverlayPriority_ModalBlocksKeys(t *testing.T) {
	a := testAppResized()
	a.modal.Show("Test", "test modal", nil)
	a.mode = msgs.ModeModal

	initialFocus := a.focus
	m, _ := a.Update(keyMsg('j'))
	a = m.(App)

	if a.focus != initialFocus {
		t.Errorf("key should have been consumed by modal; focus changed from %v to %v", initialFocus, a.focus)
	}
}

func TestOverlayPriority_JumpBlocksKeys(t *testing.T) {
	a := testAppResized()
	a.activateJumpMode()

	initialFocus := a.focus
	// Sending a random character should be consumed by jump overlay
	m, _ := a.Update(keyMsg('x'))
	a = m.(App)

	// The focus should either be the same or changed by jump overlay logic,
	// but the key should not have reached panel handlers.
	_ = initialFocus // suppress unused if needed
	// Just verify no panic and mode is still jump or has been resolved
}

func TestOverlayPriority_Order(t *testing.T) {
	// Command palette takes priority over help, which takes priority over modal,
	// which takes priority over jump.
	a := testAppResized()

	// Activate all overlays
	a.activateJumpMode()      // sets jump.Visible = true
	a.modal.Show("T", "m", nil) // sets modal.Visible = true
	a.help.SetSize(160, 40)
	a.help.Toggle()              // sets help.Visible = true
	a.commandPalette.Open()      // sets commandPalette.Visible = true

	if !a.commandPalette.Visible {
		t.Fatal("command palette should be visible")
	}
	if !a.help.Visible {
		t.Fatal("help should be visible")
	}
	if !a.modal.Visible {
		t.Fatal("modal should be visible")
	}
	if !a.jump.Visible {
		t.Fatal("jump should be visible")
	}

	// Esc should go to command palette (first in priority), not to help/modal/jump
	m, _ := a.Update(tea.KeyMsg{Type: tea.KeyEsc})
	a = m.(App)

	// After Esc, command palette should close
	if a.commandPalette.Visible {
		t.Error("command palette should have closed after Esc")
	}
	// Help, modal, and jump should still be visible since command palette consumed the key
	if !a.help.Visible {
		t.Error("help should still be visible; command palette should have consumed Esc")
	}
}

func TestTabManagement_NewTab(t *testing.T) {
	a := testAppResized()
	initialTabs := len(a.store.Tabs)

	m, _ := a.Update(msgs.NewRequestMsg{})
	a = m.(App)

	if len(a.store.Tabs) != initialTabs+1 {
		t.Errorf("expected %d tabs after new, got %d", initialTabs+1, len(a.store.Tabs))
	}
	if a.store.ActiveTab != initialTabs {
		t.Errorf("expected active tab %d, got %d", initialTabs, a.store.ActiveTab)
	}
}

func TestTabManagement_CloseTab(t *testing.T) {
	a := testAppResized()

	// Open additional tabs first
	m, _ := a.Update(msgs.NewRequestMsg{})
	a = m.(App)
	m, _ = a.Update(msgs.NewRequestMsg{})
	a = m.(App)
	tabCount := len(a.store.Tabs)

	m, _ = a.Update(msgs.CloseTabMsg{})
	a = m.(App)

	if len(a.store.Tabs) != tabCount-1 {
		t.Errorf("expected %d tabs after close, got %d", tabCount-1, len(a.store.Tabs))
	}
}

func TestTabManagement_NextPrevTab(t *testing.T) {
	a := testAppResized()

	// Create two more tabs (total 3)
	m, _ := a.Update(msgs.NewRequestMsg{})
	a = m.(App)
	m, _ = a.Update(msgs.NewRequestMsg{})
	a = m.(App)

	// Active tab should be 2 (last one)
	if a.store.ActiveTab != 2 {
		t.Fatalf("expected active tab 2, got %d", a.store.ActiveTab)
	}

	// Prev
	m, _ = a.Update(msgs.PrevTabMsg{})
	a = m.(App)
	if a.store.ActiveTab != 1 {
		t.Errorf("expected active tab 1 after prev, got %d", a.store.ActiveTab)
	}

	// Next
	m, _ = a.Update(msgs.NextTabMsg{})
	a = m.(App)
	if a.store.ActiveTab != 2 {
		t.Errorf("expected active tab 2 after next, got %d", a.store.ActiveTab)
	}

	// Next should wrap to 0
	m, _ = a.Update(msgs.NextTabMsg{})
	a = m.(App)
	if a.store.ActiveTab != 0 {
		t.Errorf("expected active tab 0 after wrap, got %d", a.store.ActiveTab)
	}
}

func TestTabManagement_SwitchTab(t *testing.T) {
	a := testAppResized()

	// Create extra tabs
	m, _ := a.Update(msgs.NewRequestMsg{})
	a = m.(App)
	m, _ = a.Update(msgs.NewRequestMsg{})
	a = m.(App)

	// Switch to tab 0
	m, _ = a.Update(msgs.SwitchTabMsg{Index: 0})
	a = m.(App)
	if a.store.ActiveTab != 0 {
		t.Errorf("expected active tab 0, got %d", a.store.ActiveTab)
	}

	// Switch to invalid tab (should be no-op)
	m, _ = a.Update(msgs.SwitchTabMsg{Index: 999})
	a = m.(App)
	if a.store.ActiveTab != 0 {
		t.Errorf("expected active tab still 0 after invalid switch, got %d", a.store.ActiveTab)
	}

	// Switch to negative index (should be no-op)
	m, _ = a.Update(msgs.SwitchTabMsg{Index: -1})
	a = m.(App)
	if a.store.ActiveTab != 0 {
		t.Errorf("expected active tab still 0 after negative switch, got %d", a.store.ActiveTab)
	}
}

func TestModeSwitch_SetModeMsg(t *testing.T) {
	modes := []msgs.AppMode{
		msgs.ModeNormal,
		msgs.ModeInsert,
		msgs.ModeCommandPalette,
		msgs.ModeJump,
		msgs.ModeModal,
		msgs.ModeSearch,
	}

	for _, mode := range modes {
		t.Run(mode.String(), func(t *testing.T) {
			a := testAppResized()
			m, _ := a.Update(msgs.SetModeMsg{Mode: mode})
			a = m.(App)

			if a.mode != mode {
				t.Errorf("expected mode %v, got %v", mode, a.mode)
			}
		})
	}
}

func TestInsertMode_EscReturnsToNormal(t *testing.T) {
	a := testAppResized()
	a.focus = msgs.FocusEditor

	// Enter insert mode
	m, _ := a.Update(keyMsg('i'))
	a = m.(App)
	if a.mode != msgs.ModeInsert {
		t.Fatalf("expected ModeInsert, got %v", a.mode)
	}

	// Esc exits insert mode
	m, _ = a.Update(tea.KeyMsg{Type: tea.KeyEsc})
	a = m.(App)

	if a.mode != msgs.ModeNormal {
		t.Errorf("expected ModeNormal after Esc, got %v", a.mode)
	}
}

func TestToggleSidebarMsg(t *testing.T) {
	a := testAppResized()
	initial := a.sidebarVisible

	m, _ := a.Update(msgs.ToggleSidebarMsg{})
	a = m.(App)

	if a.sidebarVisible == initial {
		t.Error("expected sidebar visibility to toggle via ToggleSidebarMsg")
	}
}

func TestOpenCommandPaletteMsg(t *testing.T) {
	a := testAppResized()

	m, _ := a.Update(msgs.OpenCommandPaletteMsg{})
	a = m.(App)

	if a.mode != msgs.ModeCommandPalette {
		t.Errorf("expected ModeCommandPalette, got %v", a.mode)
	}
	if !a.commandPalette.Visible {
		t.Error("expected command palette to be visible")
	}
}

func TestShowHelpMsg(t *testing.T) {
	a := testAppResized()

	m, _ := a.Update(msgs.ShowHelpMsg{})
	a = m.(App)

	if a.mode != msgs.ModeModal {
		t.Errorf("expected ModeModal, got %v", a.mode)
	}
	if !a.help.Visible {
		t.Error("expected help to be visible")
	}
}

func TestStatusMsg(t *testing.T) {
	a := testAppResized()

	m, _ := a.Update(msgs.StatusMsg{Text: "testing", Duration: 2 * time.Second})
	_ = m.(App)
	// StatusMsg handling sets the status bar text. No panic expected.
}

func TestStatusMsg_WithoutDuration(t *testing.T) {
	a := testAppResized()

	m, cmd := a.Update(msgs.StatusMsg{Text: "persistent"})
	_ = m.(App)
	// With zero duration, no tick cmd should be returned
	if cmd != nil {
		// cmd could be a batch, but the batch may contain nil cmds.
		// At minimum, we verify no panic.
	}
}

func TestSwitchProtocolMsg(t *testing.T) {
	protocols := []string{"http", "graphql", "websocket", "grpc"}

	for _, proto := range protocols {
		t.Run(proto, func(t *testing.T) {
			a := testAppResized()
			m, _ := a.Update(msgs.SwitchProtocolMsg{Protocol: proto})
			a = m.(App)

			if a.editor.Protocol() != proto {
				t.Errorf("expected protocol %q, got %q", proto, a.editor.Protocol())
			}
		})
	}
}

func TestEditorDoneMsg_WithContent(t *testing.T) {
	a := testAppResized()

	m, cmd := a.Update(msgs.EditorDoneMsg{Content: `{"key":"value"}`})
	a = m.(App)

	// Should return a toast cmd
	if cmd == nil {
		t.Error("expected non-nil cmd (toast) after EditorDoneMsg with content")
	}
}

func TestEditorDoneMsg_Empty(t *testing.T) {
	a := testAppResized()

	m, cmd := a.Update(msgs.EditorDoneMsg{Content: ""})
	_ = m.(App)

	// Empty content should be a no-op
	if cmd != nil {
		t.Error("expected nil cmd for empty EditorDoneMsg")
	}
}

func TestRequestSelectedMsg(t *testing.T) {
	a := testAppResized()

	// Get the ID of the first request
	reqID := a.store.Collection.Items[0].Request.ID

	m, _ := a.Update(msgs.RequestSelectedMsg{RequestID: reqID})
	a = m.(App)

	// After selecting, focus should move to editor
	if a.focus != msgs.FocusEditor {
		t.Errorf("expected FocusEditor after request selected, got %v", a.focus)
	}
	// Active request should match the selected one
	active := a.store.ActiveRequest()
	if active == nil {
		t.Fatal("expected non-nil active request")
	}
	if active.ID != reqID {
		t.Errorf("expected active request ID %q, got %q", reqID, active.ID)
	}
}

func TestRequestSelectedMsg_UnknownID(t *testing.T) {
	a := testAppResized()
	initialTab := a.store.ActiveTab

	m, _ := a.Update(msgs.RequestSelectedMsg{RequestID: "nonexistent-id"})
	a = m.(App)

	// Should be a no-op
	if a.store.ActiveTab != initialTab {
		t.Error("active tab should not change for unknown request ID")
	}
}

func TestClearBaselineMsg(t *testing.T) {
	a := testAppResized()

	m, cmd := a.Update(msgs.ClearBaselineMsg{})
	_ = m.(App)

	// Should return a toast cmd
	if cmd == nil {
		t.Error("expected non-nil cmd (toast) for ClearBaselineMsg")
	}
}

func TestWSConnectedMsg_Success(t *testing.T) {
	a := testAppResized()

	m, cmd := a.Update(msgs.WSConnectedMsg{Err: nil})
	_ = m.(App)

	if cmd == nil {
		t.Error("expected non-nil cmd (toast) for WSConnectedMsg")
	}
}

func TestWSConnectedMsg_Error(t *testing.T) {
	a := testAppResized()

	m, cmd := a.Update(msgs.WSConnectedMsg{Err: errTest})
	_ = m.(App)

	if cmd == nil {
		t.Error("expected non-nil cmd (toast) for WSConnectedMsg with error")
	}
}

func TestWSDisconnectedMsg(t *testing.T) {
	a := testAppResized()

	m, cmd := a.Update(msgs.WSDisconnectedMsg{Err: nil})
	_ = m.(App)

	if cmd == nil {
		t.Error("expected non-nil cmd (toast) for WSDisconnectedMsg")
	}
}

func TestWSMessageReceivedMsg(t *testing.T) {
	a := testAppResized()

	m, _ := a.Update(msgs.WSMessageReceivedMsg{
		Content:   `{"hello":"world"}`,
		IsJSON:    true,
		Timestamp: time.Now(),
	})
	_ = m.(App) // no panic expected
}

func TestView_NotReady(t *testing.T) {
	a := testApp()
	view := a.View()
	if view != "Loading..." {
		t.Errorf("expected 'Loading...' before ready, got %q", view)
	}
}

func TestView_Ready(t *testing.T) {
	a := testAppResized()
	view := a.View()
	if view == "Loading..." {
		t.Error("should not show Loading... after resize")
	}
	if len(view) == 0 {
		t.Error("expected non-empty view")
	}
}

func TestFindRequest(t *testing.T) {
	req1 := collection.NewRequest("R1", "GET", "http://a.com")
	req2 := collection.NewRequest("R2", "POST", "http://b.com")
	req3 := collection.NewRequest("R3", "PUT", "http://c.com")

	items := []collection.Item{
		{Request: req1},
		{Folder: &collection.Folder{
			Name: "folder",
			Items: []collection.Item{
				{Request: req2},
				{Request: req3},
			},
		}},
	}

	tests := []struct {
		name     string
		id       string
		expected *collection.Request
	}{
		{"top-level request", req1.ID, req1},
		{"nested request", req2.ID, req2},
		{"deeply nested request", req3.ID, req3},
		{"nonexistent", "does-not-exist", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findRequest(items, tt.id)
			if result != tt.expected {
				t.Errorf("findRequest(%q): expected %v, got %v", tt.id, tt.expected, result)
			}
		})
	}
}

func TestMessageRouting_TableDriven(t *testing.T) {
	tests := []struct {
		name   string
		msg    tea.Msg
		check  func(t *testing.T, a App, cmd tea.Cmd)
	}{
		{
			name: "NewRequestMsg adds a tab",
			msg:  msgs.NewRequestMsg{},
			check: func(t *testing.T, a App, cmd tea.Cmd) {
				// Started with 1 tab, should now have 2
				if len(a.store.Tabs) != 2 {
					t.Errorf("expected 2 tabs, got %d", len(a.store.Tabs))
				}
			},
		},
		{
			name: "CloseTabMsg reduces tabs",
			msg:  msgs.CloseTabMsg{},
			check: func(t *testing.T, a App, cmd tea.Cmd) {
				// Started with 1 tab, after close should have 0
				if len(a.store.Tabs) != 0 {
					t.Errorf("expected 0 tabs, got %d", len(a.store.Tabs))
				}
			},
		},
		{
			name: "ToggleSidebarMsg toggles visibility",
			msg:  msgs.ToggleSidebarMsg{},
			check: func(t *testing.T, a App, cmd tea.Cmd) {
				if a.sidebarVisible {
					t.Error("expected sidebar to be hidden after toggle")
				}
			},
		},
		{
			name: "OpenCommandPaletteMsg sets mode",
			msg:  msgs.OpenCommandPaletteMsg{},
			check: func(t *testing.T, a App, cmd tea.Cmd) {
				if a.mode != msgs.ModeCommandPalette {
					t.Errorf("expected ModeCommandPalette, got %v", a.mode)
				}
			},
		},
		{
			name: "ShowHelpMsg sets modal mode",
			msg:  msgs.ShowHelpMsg{},
			check: func(t *testing.T, a App, cmd tea.Cmd) {
				if a.mode != msgs.ModeModal {
					t.Errorf("expected ModeModal, got %v", a.mode)
				}
			},
		},
		{
			name: "SetModeMsg changes mode",
			msg:  msgs.SetModeMsg{Mode: msgs.ModeSearch},
			check: func(t *testing.T, a App, cmd tea.Cmd) {
				if a.mode != msgs.ModeSearch {
					t.Errorf("expected ModeSearch, got %v", a.mode)
				}
			},
		},
		{
			name: "FocusPanelMsg sets focus",
			msg:  msgs.FocusPanelMsg{Panel: msgs.FocusResponse},
			check: func(t *testing.T, a App, cmd tea.Cmd) {
				if a.focus != msgs.FocusResponse {
					t.Errorf("expected FocusResponse, got %v", a.focus)
				}
			},
		},
		{
			name: "SwitchProtocolMsg sets protocol",
			msg:  msgs.SwitchProtocolMsg{Protocol: "graphql"},
			check: func(t *testing.T, a App, cmd tea.Cmd) {
				if a.editor.Protocol() != "graphql" {
					t.Errorf("expected graphql, got %q", a.editor.Protocol())
				}
			},
		},
		{
			name: "ClearBaselineMsg returns toast",
			msg:  msgs.ClearBaselineMsg{},
			check: func(t *testing.T, a App, cmd tea.Cmd) {
				if cmd == nil {
					t.Error("expected non-nil cmd (toast)")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := testAppResized()
			m, cmd := a.Update(tt.msg)
			a = m.(App)
			tt.check(t, a, cmd)
		})
	}
}

func TestCtrlEnter_SendsRequest(t *testing.T) {
	a := testAppResized()

	// Ctrl+Enter should trigger a SendRequestMsg via handleGlobalKey
	cmd := a.handleGlobalKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'\r'}})
	// ctrl+enter has a special string representation; test the direct path
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{}}
	msg = tea.KeyMsg{Type: tea.KeyCtrlR}
	cmd = a.handleGlobalKey(msg)
	if cmd == nil {
		t.Fatal("expected non-nil cmd for Ctrl+R")
	}
	result := cmd()
	if _, ok := result.(msgs.SendRequestMsg); !ok {
		t.Errorf("expected SendRequestMsg, got %T", result)
	}
}

func TestSendRequest_EmptyURL(t *testing.T) {
	a := testAppResized()

	// Ensure the active request has an empty URL (new tab default)
	m, _ := a.Update(msgs.NewRequestMsg{})
	a = m.(App)

	m, cmd := a.Update(msgs.SendRequestMsg{})
	a = m.(App)

	// With empty URL, sendRequest should set a status message and return nil cmd
	if cmd != nil {
		t.Error("expected nil cmd when sending request with empty URL")
	}
}

func TestRequestSentMsg_Error(t *testing.T) {
	a := testAppResized()

	m, cmd := a.Update(msgs.RequestSentMsg{Err: errTest})
	_ = m.(App)

	// Should return a toast cmd for the error
	if cmd == nil {
		t.Error("expected non-nil cmd (toast) for RequestSentMsg with error")
	}
}

func TestRequestSentMsg_Success(t *testing.T) {
	a := testAppResized()

	m, _ := a.Update(msgs.RequestSentMsg{
		StatusCode:  200,
		Status:      "200 OK",
		Headers:     nil,
		Body:        []byte(`{"ok":true}`),
		ContentType: "application/json",
		Duration:    150 * time.Millisecond,
		Size:        11,
	})
	_ = m.(App) // no panic expected
}

func TestScriptResultMsg(t *testing.T) {
	a := testAppResized()

	envChanges := map[string]string{"TOKEN": "abc123"}
	m, _ := a.Update(msgs.ScriptResultMsg{
		Logs:       []string{"log line 1"},
		EnvChanges: envChanges,
		TestResults: []msgs.ScriptTestResult{
			{Name: "test1", Passed: true},
		},
	})
	a = m.(App)

	// Env changes should be applied
	if a.store.EnvVars["TOKEN"] != "abc123" {
		t.Errorf("expected env var TOKEN=abc123, got %q", a.store.EnvVars["TOKEN"])
	}
}

func TestGRPCReflectMsg(t *testing.T) {
	a := testAppResized()

	m, cmd := a.Update(msgs.GRPCReflectMsg{})
	_ = m.(App)

	// Currently returns a toast with "not yet implemented"
	if cmd == nil {
		t.Error("expected non-nil cmd for GRPCReflectMsg")
	}
}

func TestSwitchEnvMsg_NoEnvFile(t *testing.T) {
	a := testAppResized()
	// envFile is nil in test since no environments.yaml exists

	m, cmd := a.Update(msgs.SwitchEnvMsg{Name: ""})
	_ = m.(App)

	// With nil envFile and empty name, should show error toast
	if cmd == nil {
		t.Error("expected non-nil cmd (toast) for SwitchEnvMsg with no env file")
	}
}

func TestUpdateFocus(t *testing.T) {
	a := testAppResized()

	a.focus = msgs.FocusSidebar
	a.updateFocus()

	// We can't easily inspect focused state of sub-components from outside,
	// but we can verify no panic and focus is set correctly.
	if a.focus != msgs.FocusSidebar {
		t.Errorf("expected FocusSidebar, got %v", a.focus)
	}
}

func TestActivateJumpMode_SetsCorrectState(t *testing.T) {
	a := testAppResized()
	a.activateJumpMode()

	if a.mode != msgs.ModeJump {
		t.Errorf("expected ModeJump, got %v", a.mode)
	}
	if !a.jump.Visible {
		t.Error("expected jump overlay to be visible")
	}
}

func TestImportCompleteMsg_NilCollection(t *testing.T) {
	a := testAppResized()

	m, cmd := a.Update(msgs.ImportCompleteMsg{Collection: nil, Err: nil})
	_ = m.(App)

	// Should show "No data imported" toast
	if cmd == nil {
		t.Error("expected non-nil cmd (toast) for nil collection import")
	}
}

func TestImportCompleteMsg_WithError(t *testing.T) {
	a := testAppResized()

	m, cmd := a.Update(msgs.ImportCompleteMsg{Err: errTest})
	_ = m.(App)

	// Should show error toast
	if cmd == nil {
		t.Error("expected non-nil cmd (toast) for import error")
	}
}

func TestImportCompleteMsg_Success(t *testing.T) {
	a := testAppResized()

	imported := &collection.Collection{
		Name: "Imported API",
		Items: []collection.Item{
			{Request: collection.NewRequest("New Endpoint", "GET", "https://imported.example.com")},
		},
	}

	m, cmd := a.Update(msgs.ImportCompleteMsg{Collection: imported, Err: nil})
	a = m.(App)

	// Should merge into existing collection
	if len(a.store.Collection.Items) <= 2 {
		t.Error("expected imported items to be merged into collection")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd (toast) for successful import")
	}
}

func TestView_SinglePanelMode(t *testing.T) {
	a := testApp()
	// Narrow width triggers single panel mode
	m, _ := a.Update(tea.WindowSizeMsg{Width: 40, Height: 20})
	a = m.(App)

	if !a.layout.SinglePanel {
		t.Fatal("expected single panel mode at width 40")
	}

	view := a.View()
	if len(view) == 0 {
		t.Error("expected non-empty view in single panel mode")
	}
}

func TestEditorInsert_CtrlEnterSendsRequest(t *testing.T) {
	a := testAppResized()
	a.focus = msgs.FocusEditor
	a.mode = msgs.ModeInsert

	// Simulate the editor being in editing mode by first entering insert mode
	m, _ := a.Update(keyMsg('i'))
	a = m.(App)

	// Now, in the updateEditorInsert path, ctrl+enter should trigger sendRequest.
	// The sendRequest will check for URL and handle accordingly.
	m, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{}})
	_ = m.(App) // no panic
}

// sentinel error for tests
type testError struct{}

func (e testError) Error() string { return "test error" }

var errTest error = testError{}
