package components

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/serdar/gottp/internal/ui/msgs"
	"github.com/serdar/gottp/internal/ui/theme"
)

// helpers

func testStyles() theme.Styles {
	return theme.NewStyles(theme.Default())
}

func testTheme() theme.Theme {
	return theme.Default()
}

func keyMsg(key string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
}

func specialKeyMsg(t tea.KeyType) tea.KeyMsg {
	return tea.KeyMsg{Type: t}
}

// ─────────────────────────────────────────────────────────────────────────────
// KVTable tests
// ─────────────────────────────────────────────────────────────────────────────

func TestKVTable_NewDefault(t *testing.T) {
	kv := NewKVTable(testStyles())
	pairs := kv.GetPairs()
	if len(pairs) != 1 {
		t.Fatalf("expected 1 default pair, got %d", len(pairs))
	}
	if pairs[0].Key != "" || pairs[0].Value != "" {
		t.Fatalf("expected empty default pair, got %+v", pairs[0])
	}
	if !pairs[0].Enabled {
		t.Fatal("default pair should be enabled")
	}
	if kv.Editing() {
		t.Fatal("should not start in editing mode")
	}
}

func TestKVTable_SetPairsGetPairs_RoundTrip(t *testing.T) {
	kv := NewKVTable(testStyles())

	input := []KVPair{
		{Key: "Content-Type", Value: "application/json", Enabled: true},
		{Key: "Authorization", Value: "Bearer token123", Enabled: false},
		{Key: "Accept", Value: "*/*", Enabled: true},
	}
	kv.SetPairs(input)

	got := kv.GetPairs()
	if len(got) != len(input) {
		t.Fatalf("expected %d pairs, got %d", len(input), len(got))
	}
	for i := range input {
		if got[i].Key != input[i].Key || got[i].Value != input[i].Value || got[i].Enabled != input[i].Enabled {
			t.Errorf("pair %d mismatch: want %+v, got %+v", i, input[i], got[i])
		}
	}
}

func TestKVTable_SetPairs_EmptySlice(t *testing.T) {
	kv := NewKVTable(testStyles())
	kv.SetPairs([]KVPair{})
	pairs := kv.GetPairs()
	if len(pairs) != 1 {
		t.Fatalf("expected 1 fallback pair for empty input, got %d", len(pairs))
	}
	if pairs[0].Key != "" || pairs[0].Value != "" {
		t.Fatal("fallback pair should be empty")
	}
}

func TestKVTable_SetPairs_CursorClamp(t *testing.T) {
	kv := NewKVTable(testStyles())
	kv.SetPairs([]KVPair{
		{Key: "a", Value: "1", Enabled: true},
		{Key: "b", Value: "2", Enabled: true},
		{Key: "c", Value: "3", Enabled: true},
	})
	// Move cursor to the last element
	kv, _ = kv.Update(keyMsg("j"))
	kv, _ = kv.Update(keyMsg("j"))
	// Now replace with fewer items
	kv.SetPairs([]KVPair{
		{Key: "x", Value: "9", Enabled: true},
	})
	// Cursor should be clamped
	pairs := kv.GetPairs()
	if len(pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(pairs))
	}
}

func TestKVTable_GetPairs_ReturnsCopy(t *testing.T) {
	kv := NewKVTable(testStyles())
	kv.SetPairs([]KVPair{
		{Key: "a", Value: "1", Enabled: true},
	})
	got := kv.GetPairs()
	got[0].Key = "modified"
	// Original should be unchanged
	original := kv.GetPairs()
	if original[0].Key == "modified" {
		t.Fatal("GetPairs should return a copy, not a reference")
	}
}

func TestKVTable_Navigation_JK(t *testing.T) {
	kv := NewKVTable(testStyles())
	kv.SetPairs([]KVPair{
		{Key: "a", Value: "1", Enabled: true},
		{Key: "b", Value: "2", Enabled: true},
		{Key: "c", Value: "3", Enabled: true},
	})

	// Start at index 0
	if kv.cursor != 0 {
		t.Fatalf("expected cursor at 0, got %d", kv.cursor)
	}

	// j moves down
	kv, _ = kv.Update(keyMsg("j"))
	if kv.cursor != 1 {
		t.Fatalf("after j: expected cursor at 1, got %d", kv.cursor)
	}

	kv, _ = kv.Update(keyMsg("j"))
	if kv.cursor != 2 {
		t.Fatalf("after j j: expected cursor at 2, got %d", kv.cursor)
	}

	// j at bottom stays at bottom
	kv, _ = kv.Update(keyMsg("j"))
	if kv.cursor != 2 {
		t.Fatalf("j at bottom: expected cursor at 2, got %d", kv.cursor)
	}

	// k moves up
	kv, _ = kv.Update(keyMsg("k"))
	if kv.cursor != 1 {
		t.Fatalf("after k: expected cursor at 1, got %d", kv.cursor)
	}

	// k at top stays at top
	kv, _ = kv.Update(keyMsg("k"))
	kv, _ = kv.Update(keyMsg("k"))
	if kv.cursor != 0 {
		t.Fatalf("k at top: expected cursor at 0, got %d", kv.cursor)
	}
}

func TestKVTable_Navigation_ArrowKeys(t *testing.T) {
	kv := NewKVTable(testStyles())
	kv.SetPairs([]KVPair{
		{Key: "a", Value: "1", Enabled: true},
		{Key: "b", Value: "2", Enabled: true},
	})

	kv, _ = kv.Update(specialKeyMsg(tea.KeyDown))
	if kv.cursor != 1 {
		t.Fatalf("after down: expected cursor at 1, got %d", kv.cursor)
	}

	kv, _ = kv.Update(specialKeyMsg(tea.KeyUp))
	if kv.cursor != 0 {
		t.Fatalf("after up: expected cursor at 0, got %d", kv.cursor)
	}
}

func TestKVTable_ColumnSwitching(t *testing.T) {
	kv := NewKVTable(testStyles())
	if kv.column != ColKey {
		t.Fatalf("expected initial column ColKey, got %d", kv.column)
	}

	kv, _ = kv.Update(specialKeyMsg(tea.KeyTab))
	if kv.column != ColValue {
		t.Fatalf("after tab: expected ColValue, got %d", kv.column)
	}

	kv, _ = kv.Update(specialKeyMsg(tea.KeyTab))
	if kv.column != ColKey {
		t.Fatalf("after second tab: expected ColKey, got %d", kv.column)
	}
}

func TestKVTable_ToggleEnabled(t *testing.T) {
	kv := NewKVTable(testStyles())
	kv.SetPairs([]KVPair{
		{Key: "a", Value: "1", Enabled: true},
	})

	kv, _ = kv.Update(keyMsg(" "))
	pairs := kv.GetPairs()
	if pairs[0].Enabled {
		t.Fatal("after space: pair should be disabled")
	}

	kv, _ = kv.Update(keyMsg(" "))
	pairs = kv.GetPairs()
	if !pairs[0].Enabled {
		t.Fatal("after second space: pair should be enabled again")
	}
}

func TestKVTable_EnterStartsEditing(t *testing.T) {
	kv := NewKVTable(testStyles())
	kv.SetPairs([]KVPair{
		{Key: "mykey", Value: "myval", Enabled: true},
	})

	if kv.Editing() {
		t.Fatal("should not be editing before enter")
	}

	kv, cmd := kv.Update(specialKeyMsg(tea.KeyEnter))
	if !kv.Editing() {
		t.Fatal("should be editing after enter")
	}
	if cmd == nil {
		t.Fatal("enter should return a blink cmd")
	}
}

func TestKVTable_EditKey_CommitWithEsc(t *testing.T) {
	kv := NewKVTable(testStyles())
	kv.SetPairs([]KVPair{
		{Key: "old", Value: "val", Enabled: true},
	})

	// Start editing key column
	kv.column = ColKey
	kv, _ = kv.Update(specialKeyMsg(tea.KeyEnter))
	if !kv.Editing() {
		t.Fatal("should be editing")
	}

	// The input should have the old value loaded
	if kv.input.Value() != "old" {
		t.Fatalf("expected input value 'old', got '%s'", kv.input.Value())
	}

	// Type new value by setting it directly (simulating typing)
	kv.input.SetValue("newkey")

	// Esc commits and exits editing
	kv, _ = kv.Update(specialKeyMsg(tea.KeyEscape))
	if kv.Editing() {
		t.Fatal("should not be editing after esc")
	}

	pairs := kv.GetPairs()
	if pairs[0].Key != "newkey" {
		t.Fatalf("expected key 'newkey', got '%s'", pairs[0].Key)
	}
}

func TestKVTable_EditValue_CommitWithEnter(t *testing.T) {
	kv := NewKVTable(testStyles())
	kv.SetPairs([]KVPair{
		{Key: "k", Value: "oldval", Enabled: true},
	})

	// Switch to value column and start editing
	kv.column = ColValue
	kv, _ = kv.Update(specialKeyMsg(tea.KeyEnter))

	if kv.input.Value() != "oldval" {
		t.Fatalf("expected input value 'oldval', got '%s'", kv.input.Value())
	}

	kv.input.SetValue("newval")

	kv, _ = kv.Update(specialKeyMsg(tea.KeyEnter))
	if kv.Editing() {
		t.Fatal("should not be editing after enter")
	}

	pairs := kv.GetPairs()
	if pairs[0].Value != "newval" {
		t.Fatalf("expected value 'newval', got '%s'", pairs[0].Value)
	}
}

func TestKVTable_TabDuringEditing_SwitchesColumn(t *testing.T) {
	kv := NewKVTable(testStyles())
	kv.SetPairs([]KVPair{
		{Key: "k", Value: "v", Enabled: true},
	})

	kv.column = ColKey
	kv, _ = kv.Update(specialKeyMsg(tea.KeyEnter))
	if !kv.Editing() {
		t.Fatal("should be editing")
	}

	// Tab during editing: commits current, switches column, starts editing again
	kv, cmd := kv.Update(specialKeyMsg(tea.KeyTab))
	if !kv.Editing() {
		t.Fatal("should still be editing after tab in edit mode")
	}
	if kv.column != ColValue {
		t.Fatalf("expected column to switch to ColValue, got %d", kv.column)
	}
	if cmd == nil {
		t.Fatal("tab during editing should return a blink cmd")
	}
}

func TestKVTable_AddPair(t *testing.T) {
	kv := NewKVTable(testStyles())
	kv.SetPairs([]KVPair{
		{Key: "existing", Value: "val", Enabled: true},
	})

	kv, cmd := kv.Update(keyMsg("a"))
	pairs := kv.GetPairs()
	if len(pairs) != 2 {
		t.Fatalf("expected 2 pairs after add, got %d", len(pairs))
	}
	if pairs[1].Key != "" || pairs[1].Value != "" {
		t.Fatalf("new pair should be empty, got %+v", pairs[1])
	}
	if !pairs[1].Enabled {
		t.Fatal("new pair should be enabled")
	}
	if kv.cursor != 1 {
		t.Fatalf("cursor should be on new pair (1), got %d", kv.cursor)
	}
	if !kv.Editing() {
		t.Fatal("should be editing after add")
	}
	if kv.column != ColKey {
		t.Fatalf("should focus key column after add, got %d", kv.column)
	}
	if cmd == nil {
		t.Fatal("add should return blink cmd")
	}
}

func TestKVTable_DeletePair(t *testing.T) {
	kv := NewKVTable(testStyles())
	kv.SetPairs([]KVPair{
		{Key: "a", Value: "1", Enabled: true},
		{Key: "b", Value: "2", Enabled: true},
		{Key: "c", Value: "3", Enabled: true},
	})

	// Move to middle item and delete
	kv, _ = kv.Update(keyMsg("j"))
	kv, _ = kv.Update(keyMsg("d"))

	pairs := kv.GetPairs()
	if len(pairs) != 2 {
		t.Fatalf("expected 2 pairs after delete, got %d", len(pairs))
	}
	if pairs[0].Key != "a" || pairs[1].Key != "c" {
		t.Fatalf("expected [a, c], got [%s, %s]", pairs[0].Key, pairs[1].Key)
	}
}

func TestKVTable_DeleteLastPair_ResetToEmpty(t *testing.T) {
	kv := NewKVTable(testStyles())
	kv.SetPairs([]KVPair{
		{Key: "only", Value: "one", Enabled: false},
	})

	kv, _ = kv.Update(keyMsg("d"))

	pairs := kv.GetPairs()
	if len(pairs) != 1 {
		t.Fatalf("expected 1 pair (reset), got %d", len(pairs))
	}
	if pairs[0].Key != "" || pairs[0].Value != "" {
		t.Fatalf("reset pair should be empty, got %+v", pairs[0])
	}
	if !pairs[0].Enabled {
		t.Fatal("reset pair should be enabled")
	}
}

func TestKVTable_DeleteAtEnd_CursorClamps(t *testing.T) {
	kv := NewKVTable(testStyles())
	kv.SetPairs([]KVPair{
		{Key: "a", Value: "1", Enabled: true},
		{Key: "b", Value: "2", Enabled: true},
	})

	// Move to last item and delete
	kv, _ = kv.Update(keyMsg("j"))
	if kv.cursor != 1 {
		t.Fatalf("expected cursor at 1, got %d", kv.cursor)
	}
	kv, _ = kv.Update(keyMsg("d"))

	if kv.cursor != 0 {
		t.Fatalf("cursor should clamp to 0 after deleting last item, got %d", kv.cursor)
	}
}

func TestKVTable_View_NotEmpty(t *testing.T) {
	kv := NewKVTable(testStyles())
	kv.SetPairs([]KVPair{
		{Key: "Content-Type", Value: "application/json", Enabled: true},
		{Key: "X-Custom", Value: "hello", Enabled: false},
	})
	kv.SetSize(80)

	view := kv.View()
	if view == "" {
		t.Fatal("view should not be empty")
	}
	// The view should contain our key names somewhere (possibly styled)
	if !strings.Contains(view, "Content-Type") {
		t.Error("view should contain 'Content-Type'")
	}
}

func TestKVTable_View_EmptyPairs(t *testing.T) {
	kv := NewKVTable(testStyles())
	// Force empty pairs by directly setting (bypassing SetPairs guard)
	kv.pairs = []KVPair{}

	view := kv.View()
	if !strings.Contains(view, "No entries") {
		t.Error("empty pairs view should show 'No entries'")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// TabBar tests
// ─────────────────────────────────────────────────────────────────────────────

func TestTabBar_NewDefault(t *testing.T) {
	tb := NewTabBar(testTheme(), testStyles())
	if tb.active != 0 {
		t.Fatalf("expected initial active 0, got %d", tb.active)
	}
}

func TestTabBar_SetTabs(t *testing.T) {
	tb := NewTabBar(testTheme(), testStyles())
	tabs := []TabItem{
		{Name: "Request 1", Method: "GET"},
		{Name: "Request 2", Method: "POST"},
		{Name: "Request 3", Method: "DELETE"},
	}
	tb.SetTabs(tabs)
	if len(tb.tabs) != 3 {
		t.Fatalf("expected 3 tabs, got %d", len(tb.tabs))
	}
}

func TestTabBar_SetActive(t *testing.T) {
	tb := NewTabBar(testTheme(), testStyles())
	tb.SetTabs([]TabItem{
		{Name: "A", Method: "GET"},
		{Name: "B", Method: "POST"},
	})

	tb.SetActive(1)
	if tb.active != 1 {
		t.Fatalf("expected active 1, got %d", tb.active)
	}

	// Out-of-bounds values should be ignored
	tb.SetActive(10)
	if tb.active != 1 {
		t.Fatalf("out of bounds should be ignored; expected active 1, got %d", tb.active)
	}

	tb.SetActive(-1)
	if tb.active != 1 {
		t.Fatalf("negative should be ignored; expected active 1, got %d", tb.active)
	}
}

func TestTabBar_SetTabs_ClampActive(t *testing.T) {
	tb := NewTabBar(testTheme(), testStyles())
	tb.SetTabs([]TabItem{
		{Name: "A", Method: "GET"},
		{Name: "B", Method: "POST"},
		{Name: "C", Method: "PUT"},
	})
	tb.SetActive(2) // last tab

	// Replace with fewer tabs
	tb.SetTabs([]TabItem{
		{Name: "X", Method: "GET"},
	})
	if tb.active != 0 {
		t.Fatalf("active should be clamped to 0, got %d", tb.active)
	}
}

func TestTabBar_View_Empty(t *testing.T) {
	tb := NewTabBar(testTheme(), testStyles())
	view := tb.View()
	if view != "" {
		t.Fatalf("empty tab bar should render empty string, got: %q", view)
	}
}

func TestTabBar_View_WithTabs(t *testing.T) {
	tb := NewTabBar(testTheme(), testStyles())
	tb.SetTabs([]TabItem{
		{Name: "Users", Method: "GET"},
		{Name: "Create", Method: "POST"},
	})
	tb.SetWidth(100)

	view := tb.View()
	if view == "" {
		t.Fatal("tab bar view should not be empty")
	}
	// Should contain the [+] button
	if !strings.Contains(view, "[+]") {
		t.Error("tab bar should contain [+] button")
	}
	// Should contain tab names
	if !strings.Contains(view, "Users") {
		t.Error("tab bar should contain 'Users'")
	}
	if !strings.Contains(view, "Create") {
		t.Error("tab bar should contain 'Create'")
	}
}

func TestTabBar_Update_BracketKeys(t *testing.T) {
	tb := NewTabBar(testTheme(), testStyles())
	tb.SetTabs([]TabItem{
		{Name: "A", Method: "GET"},
		{Name: "B", Method: "POST"},
	})

	// [ should produce PrevTabMsg
	_, cmd := tb.Update(keyMsg("["))
	if cmd == nil {
		t.Fatal("[ should produce a cmd")
	}
	msg := cmd()
	if _, ok := msg.(msgs.PrevTabMsg); !ok {
		t.Fatalf("[ should emit PrevTabMsg, got %T", msg)
	}

	// ] should produce NextTabMsg
	_, cmd = tb.Update(keyMsg("]"))
	if cmd == nil {
		t.Fatal("] should produce a cmd")
	}
	msg = cmd()
	if _, ok := msg.(msgs.NextTabMsg); !ok {
		t.Fatalf("] should emit NextTabMsg, got %T", msg)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// StatusBar tests
// ─────────────────────────────────────────────────────────────────────────────

func TestStatusBar_NewDefault(t *testing.T) {
	sb := NewStatusBar(testTheme(), testStyles())
	if sb.mode != msgs.ModeNormal {
		t.Fatalf("expected initial mode ModeNormal, got %d", sb.mode)
	}
}

func TestStatusBar_SetStatus(t *testing.T) {
	sb := NewStatusBar(testTheme(), testStyles())
	sb.SetStatus(200, 150*time.Millisecond, 1024, "application/json")

	if sb.statusCode != 200 {
		t.Fatalf("expected statusCode 200, got %d", sb.statusCode)
	}
	if sb.duration != 150*time.Millisecond {
		t.Fatalf("expected duration 150ms, got %v", sb.duration)
	}
	if sb.size != 1024 {
		t.Fatalf("expected size 1024, got %d", sb.size)
	}
	if sb.contentType != "application/json" {
		t.Fatalf("expected contentType application/json, got %s", sb.contentType)
	}
}

func TestStatusBar_SetMode(t *testing.T) {
	sb := NewStatusBar(testTheme(), testStyles())
	sb.SetMode(msgs.ModeInsert)
	if sb.mode != msgs.ModeInsert {
		t.Fatalf("expected ModeInsert, got %d", sb.mode)
	}
}

func TestStatusBar_SetMessage(t *testing.T) {
	sb := NewStatusBar(testTheme(), testStyles())
	sb.SetMessage("Request sent!")
	if sb.message != "Request sent!" {
		t.Fatalf("expected message 'Request sent!', got '%s'", sb.message)
	}
}

func TestStatusBar_SetEnv(t *testing.T) {
	sb := NewStatusBar(testTheme(), testStyles())
	sb.SetEnv("Production")
	if sb.envName != "Production" {
		t.Fatalf("expected envName 'Production', got '%s'", sb.envName)
	}
}

func TestStatusBar_UpdateClearsMessage(t *testing.T) {
	sb := NewStatusBar(testTheme(), testStyles())
	sb.SetMessage("temporary")

	sb, _ = sb.Update(clearStatusMsg{})
	if sb.message != "" {
		t.Fatalf("expected empty message after clearStatusMsg, got '%s'", sb.message)
	}
}

func TestStatusBar_View_ContainsModeIndicator(t *testing.T) {
	tests := []struct {
		mode     msgs.AppMode
		expected string
	}{
		{msgs.ModeNormal, "NORMAL"},
		{msgs.ModeInsert, "INSERT"},
		{msgs.ModeCommandPalette, "COMMAND"},
		{msgs.ModeJump, "JUMP"},
		{msgs.ModeSearch, "SEARCH"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			sb := NewStatusBar(testTheme(), testStyles())
			sb.SetMode(tt.mode)
			sb.SetWidth(120)

			view := sb.View()
			if !strings.Contains(view, tt.expected) {
				t.Errorf("view should contain mode indicator '%s'", tt.expected)
			}
		})
	}
}

func TestStatusBar_View_ContainsEnv(t *testing.T) {
	sb := NewStatusBar(testTheme(), testStyles())
	sb.SetEnv("Staging")
	sb.SetWidth(120)

	view := sb.View()
	if !strings.Contains(view, "Staging") {
		t.Error("view should contain environment name 'Staging'")
	}
}

func TestStatusBar_View_ContainsMessage(t *testing.T) {
	sb := NewStatusBar(testTheme(), testStyles())
	sb.SetMessage("Saved!")
	sb.SetWidth(120)

	view := sb.View()
	if !strings.Contains(view, "Saved!") {
		t.Error("view should contain the message 'Saved!'")
	}
}

func TestStatusBar_View_ContainsHelpHint(t *testing.T) {
	sb := NewStatusBar(testTheme(), testStyles())
	sb.SetWidth(120)

	view := sb.View()
	if !strings.Contains(view, "?:help") {
		t.Error("view should contain help hint")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Toast tests
// ─────────────────────────────────────────────────────────────────────────────

func TestToast_NewDefault(t *testing.T) {
	toast := NewToast(testTheme(), testStyles())
	if toast.Visible {
		t.Fatal("toast should start hidden")
	}
}

func TestToast_Show(t *testing.T) {
	toast := NewToast(testTheme(), testStyles())

	cmd := toast.Show("Request sent!", false, 2*time.Second)
	if !toast.Visible {
		t.Fatal("toast should be visible after Show")
	}
	if toast.text != "Request sent!" {
		t.Fatalf("expected text 'Request sent!', got '%s'", toast.text)
	}
	if toast.isError {
		t.Fatal("toast should not be error")
	}
	if toast.duration != 2*time.Second {
		t.Fatalf("expected duration 2s, got %v", toast.duration)
	}
	if cmd == nil {
		t.Fatal("Show should return a tick cmd for auto-dismiss")
	}
}

func TestToast_Show_ErrorState(t *testing.T) {
	toast := NewToast(testTheme(), testStyles())
	toast.Show("Failed!", true, 0)
	if !toast.isError {
		t.Fatal("toast should be in error state")
	}
	// Zero duration should default to 3s
	if toast.duration != 3*time.Second {
		t.Fatalf("expected default 3s duration, got %v", toast.duration)
	}
}

func TestToast_Update_DismissMsg(t *testing.T) {
	toast := NewToast(testTheme(), testStyles())
	toast.Show("hello", false, time.Second)

	toast, _ = toast.Update(toastDismissMsg{})
	if toast.Visible {
		t.Fatal("toast should be hidden after dismiss")
	}
	if toast.text != "" {
		t.Fatalf("toast text should be empty after dismiss, got '%s'", toast.text)
	}
}

func TestToast_View_WhenHidden(t *testing.T) {
	toast := NewToast(testTheme(), testStyles())
	view := toast.View()
	if view != "" {
		t.Fatalf("hidden toast should render empty string, got: %q", view)
	}
}

func TestToast_View_WhenVisible(t *testing.T) {
	toast := NewToast(testTheme(), testStyles())
	toast.Show("Success!", false, time.Second)
	view := toast.View()
	if view == "" {
		t.Fatal("visible toast should not render empty")
	}
	if !strings.Contains(view, "Success!") {
		t.Error("toast view should contain the message text")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Modal tests
// ─────────────────────────────────────────────────────────────────────────────

func TestModal_NewDefault(t *testing.T) {
	m := NewModal(testTheme(), testStyles())
	if m.Visible {
		t.Fatal("modal should start hidden")
	}
	if !m.focusOK {
		t.Fatal("modal should default to focusOK = true")
	}
}

func TestModal_Show(t *testing.T) {
	m := NewModal(testTheme(), testStyles())
	confirmMsg := msgs.SaveRequestMsg{}
	m.Show("Confirm", "Are you sure?", confirmMsg)

	if !m.Visible {
		t.Fatal("modal should be visible after Show")
	}
	if m.Title != "Confirm" {
		t.Fatalf("expected title 'Confirm', got '%s'", m.Title)
	}
	if m.Message != "Are you sure?" {
		t.Fatalf("expected message 'Are you sure?', got '%s'", m.Message)
	}
	if !m.focusOK {
		t.Fatal("Show should reset focusOK to true")
	}
}

func TestModal_Esc_Closes(t *testing.T) {
	m := NewModal(testTheme(), testStyles())
	m.Show("Test", "msg", msgs.SaveRequestMsg{})

	m, cmd := m.Update(specialKeyMsg(tea.KeyEscape))
	if m.Visible {
		t.Fatal("modal should be hidden after esc")
	}
	if cmd == nil {
		t.Fatal("esc should emit SetModeMsg")
	}
	msg := cmd()
	if setMode, ok := msg.(msgs.SetModeMsg); ok {
		if setMode.Mode != msgs.ModeNormal {
			t.Fatalf("expected ModeNormal, got %d", setMode.Mode)
		}
	} else {
		t.Fatalf("expected SetModeMsg, got %T", msg)
	}
}

func TestModal_Tab_TogglesFocus(t *testing.T) {
	m := NewModal(testTheme(), testStyles())
	m.Show("Test", "msg", msgs.SaveRequestMsg{})

	if !m.focusOK {
		t.Fatal("initial focus should be on OK")
	}

	m, _ = m.Update(specialKeyMsg(tea.KeyTab))
	if m.focusOK {
		t.Fatal("after tab: focus should be on Cancel")
	}

	m, _ = m.Update(specialKeyMsg(tea.KeyTab))
	if !m.focusOK {
		t.Fatal("after second tab: focus should be back on OK")
	}
}

func TestModal_Enter_FocusOK_Confirms(t *testing.T) {
	m := NewModal(testTheme(), testStyles())
	confirmMsg := msgs.SaveRequestMsg{}
	m.Show("Test", "msg", confirmMsg)
	m.focusOK = true

	m, cmd := m.Update(specialKeyMsg(tea.KeyEnter))
	if m.Visible {
		t.Fatal("modal should be hidden after confirm")
	}
	if cmd == nil {
		t.Fatal("enter with focusOK should emit cmd")
	}
}

func TestModal_Enter_FocusCancel_NoConfirm(t *testing.T) {
	m := NewModal(testTheme(), testStyles())
	m.Show("Test", "msg", msgs.SaveRequestMsg{})
	m.focusOK = false

	m, cmd := m.Update(specialKeyMsg(tea.KeyEnter))
	if m.Visible {
		t.Fatal("modal should be hidden after cancel-enter")
	}
	// Should still get a SetModeMsg (not the confirm message)
	if cmd == nil {
		t.Fatal("enter should still emit SetModeMsg")
	}
	msg := cmd()
	if _, ok := msg.(msgs.SetModeMsg); !ok {
		t.Fatalf("expected SetModeMsg for cancel-enter, got %T", msg)
	}
}

func TestModal_IgnoresInputWhenHidden(t *testing.T) {
	m := NewModal(testTheme(), testStyles())
	// Not visible, should do nothing
	m, cmd := m.Update(specialKeyMsg(tea.KeyEnter))
	if cmd != nil {
		t.Fatal("hidden modal should not produce cmds")
	}
}

func TestModal_View_WhenHidden(t *testing.T) {
	m := NewModal(testTheme(), testStyles())
	view := m.View()
	if view != "" {
		t.Fatalf("hidden modal should render empty string, got: %q", view)
	}
}

func TestModal_View_WhenVisible(t *testing.T) {
	m := NewModal(testTheme(), testStyles())
	m.Show("Delete?", "This cannot be undone.", msgs.SaveRequestMsg{})

	view := m.View()
	if view == "" {
		t.Fatal("visible modal should not render empty")
	}
	if !strings.Contains(view, "Delete?") {
		t.Error("modal view should contain title")
	}
	if !strings.Contains(view, "This cannot be undone.") {
		t.Error("modal view should contain message")
	}
	if !strings.Contains(view, "OK") {
		t.Error("modal view should contain OK button")
	}
	if !strings.Contains(view, "Cancel") {
		t.Error("modal view should contain Cancel button")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// JumpOverlay tests
// ─────────────────────────────────────────────────────────────────────────────

func TestGenerateLabel(t *testing.T) {
	tests := []struct {
		idx      int
		expected string
	}{
		{0, "a"},
		{1, "b"},
		{25, "z"},
		{26, "aa"},
		{27, "ab"},
		{51, "az"},
		{52, "ba"},
		{53, "bb"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := generateLabel(tt.idx)
			if got != tt.expected {
				t.Fatalf("generateLabel(%d) = %q, want %q", tt.idx, got, tt.expected)
			}
		})
	}
}

func TestJumpOverlay_NewDefault(t *testing.T) {
	j := NewJumpOverlay(testTheme(), testStyles())
	if j.Visible {
		t.Fatal("jump overlay should start hidden")
	}
}

func TestJumpOverlay_Open(t *testing.T) {
	j := NewJumpOverlay(testTheme(), testStyles())
	targets := []JumpTarget{
		{Name: "URL", Panel: msgs.FocusEditor, Action: msgs.FocusPanelMsg{Panel: msgs.FocusEditor}},
		{Name: "Sidebar", Panel: msgs.FocusSidebar, Action: msgs.FocusPanelMsg{Panel: msgs.FocusSidebar}},
		{Name: "Response", Panel: msgs.FocusResponse, Action: msgs.FocusPanelMsg{Panel: msgs.FocusResponse}},
	}
	j.Open(targets)

	if !j.Visible {
		t.Fatal("should be visible after Open")
	}
	if len(j.targets) != 3 {
		t.Fatalf("expected 3 targets, got %d", len(j.targets))
	}
	// Labels should be auto-generated
	if j.targets[0].Label != "a" {
		t.Fatalf("first label should be 'a', got '%s'", j.targets[0].Label)
	}
	if j.targets[1].Label != "b" {
		t.Fatalf("second label should be 'b', got '%s'", j.targets[1].Label)
	}
	if j.targets[2].Label != "c" {
		t.Fatalf("third label should be 'c', got '%s'", j.targets[2].Label)
	}
	if j.typed != "" {
		t.Fatal("typed should be empty on open")
	}
}

func TestJumpOverlay_Open_ManyTargets(t *testing.T) {
	j := NewJumpOverlay(testTheme(), testStyles())

	// Create 30 targets (more than 26 letters)
	targets := make([]JumpTarget, 30)
	for i := range targets {
		targets[i] = JumpTarget{
			Name:   "target",
			Panel:  msgs.FocusEditor,
			Action: msgs.FocusPanelMsg{Panel: msgs.FocusEditor},
		}
	}
	j.Open(targets)

	if j.targets[25].Label != "z" {
		t.Fatalf("26th label should be 'z', got '%s'", j.targets[25].Label)
	}
	if j.targets[26].Label != "aa" {
		t.Fatalf("27th label should be 'aa', got '%s'", j.targets[26].Label)
	}
	if j.targets[29].Label != "ad" {
		t.Fatalf("30th label should be 'ad', got '%s'", j.targets[29].Label)
	}
}

func TestJumpOverlay_TypeSingleChar_UniqueMatch(t *testing.T) {
	j := NewJumpOverlay(testTheme(), testStyles())
	actionMsg := msgs.FocusPanelMsg{Panel: msgs.FocusEditor}
	targets := []JumpTarget{
		{Name: "URL", Panel: msgs.FocusEditor, Action: actionMsg},
		{Name: "Sidebar", Panel: msgs.FocusSidebar, Action: msgs.FocusPanelMsg{Panel: msgs.FocusSidebar}},
	}
	j.Open(targets)

	// Type 'a' — matches only "a" (first target), should select it
	j, cmd := j.Update(keyMsg("a"))
	if j.Visible {
		t.Fatal("overlay should close after unique match")
	}
	if cmd == nil {
		t.Fatal("unique match should produce a cmd")
	}
}

func TestJumpOverlay_TypeChar_NoMatch(t *testing.T) {
	j := NewJumpOverlay(testTheme(), testStyles())
	targets := []JumpTarget{
		{Name: "URL", Panel: msgs.FocusEditor, Action: msgs.FocusPanelMsg{Panel: msgs.FocusEditor}},
	}
	j.Open(targets)

	// Type 'z' — no match for single target with label 'a'
	j, cmd := j.Update(keyMsg("z"))
	if j.Visible {
		t.Fatal("overlay should close on no match")
	}
	if cmd == nil {
		t.Fatal("no match should still emit SetModeMsg")
	}
}

func TestJumpOverlay_Esc_Closes(t *testing.T) {
	j := NewJumpOverlay(testTheme(), testStyles())
	j.Open([]JumpTarget{
		{Name: "URL", Panel: msgs.FocusEditor, Action: msgs.FocusPanelMsg{Panel: msgs.FocusEditor}},
	})

	j, cmd := j.Update(specialKeyMsg(tea.KeyEscape))
	if j.Visible {
		t.Fatal("overlay should close on esc")
	}
	if cmd == nil {
		t.Fatal("esc should emit SetModeMsg")
	}
}

func TestJumpOverlay_IgnoresInputWhenHidden(t *testing.T) {
	j := NewJumpOverlay(testTheme(), testStyles())
	j, cmd := j.Update(keyMsg("a"))
	if cmd != nil {
		t.Fatal("hidden overlay should not produce cmds")
	}
}

func TestJumpOverlay_Close(t *testing.T) {
	j := NewJumpOverlay(testTheme(), testStyles())
	j.Open([]JumpTarget{
		{Name: "URL", Panel: msgs.FocusEditor, Action: msgs.FocusPanelMsg{Panel: msgs.FocusEditor}},
	})
	j.Close()

	if j.Visible {
		t.Fatal("should not be visible after Close")
	}
	if j.targets != nil {
		t.Fatal("targets should be nil after Close")
	}
	if j.typed != "" {
		t.Fatal("typed should be empty after Close")
	}
}

func TestJumpOverlay_View_WhenHidden(t *testing.T) {
	j := NewJumpOverlay(testTheme(), testStyles())
	view := j.View()
	if view != "" {
		t.Fatalf("hidden overlay should render empty, got: %q", view)
	}
}

func TestJumpOverlay_View_WhenVisible(t *testing.T) {
	j := NewJumpOverlay(testTheme(), testStyles())
	j.Open([]JumpTarget{
		{Name: "URL field", Panel: msgs.FocusEditor, Action: msgs.FocusPanelMsg{Panel: msgs.FocusEditor}},
		{Name: "Sidebar", Panel: msgs.FocusSidebar, Action: msgs.FocusPanelMsg{Panel: msgs.FocusSidebar}},
	})

	view := j.View()
	if view == "" {
		t.Fatal("visible overlay should not render empty")
	}
	if !strings.Contains(view, "Jump to:") {
		t.Error("view should contain 'Jump to:' title")
	}
	if !strings.Contains(view, "URL field") {
		t.Error("view should contain target name 'URL field'")
	}
}

func TestJumpOverlay_PartialMatch_NarrowsDown(t *testing.T) {
	j := NewJumpOverlay(testTheme(), testStyles())
	// Create 27+ targets so labels go into two-char territory (aa, ab, ...)
	targets := make([]JumpTarget, 28)
	for i := range targets {
		targets[i] = JumpTarget{
			Name:   "target",
			Panel:  msgs.FocusEditor,
			Action: msgs.FocusPanelMsg{Panel: msgs.FocusEditor},
		}
	}
	j.Open(targets)

	// Type 'a' — matches "a" (index 0) and "aa" (index 26) and "ab" (index 27)
	// That's multiple matches, so overlay stays open
	j, _ = j.Update(keyMsg("a"))
	// After typing 'a', we have "a" as exact prefix match for label "a" + "aa" + "ab"
	// len(matching) > 1, so it should remain visible
	// Actually: target[0].Label = "a", has prefix "a" -> yes
	//           target[26].Label = "aa", has prefix "a" -> yes
	//           target[27].Label = "ab", has prefix "a" -> yes
	// So multiple matches, stays open
	if !j.Visible {
		t.Fatal("overlay should remain visible with multiple partial matches")
	}

	// Type 'b' -> typed is now "ab", only matches target[27] (label "ab")
	j, cmd := j.Update(keyMsg("b"))
	if j.Visible {
		t.Fatal("overlay should close after unique two-char match")
	}
	if cmd == nil {
		t.Fatal("unique match should produce a cmd")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// CommandPalette tests
// ─────────────────────────────────────────────────────────────────────────────

func TestCommandPalette_NewDefault(t *testing.T) {
	cp := NewCommandPalette(testTheme(), testStyles())
	if cp.Visible {
		t.Fatal("palette should start hidden")
	}
	if len(cp.commands) == 0 {
		t.Fatal("should have default commands")
	}
	if len(cp.filtered) == 0 {
		t.Fatal("filtered should have default commands")
	}
}

func TestCommandPalette_OpenClose(t *testing.T) {
	cp := NewCommandPalette(testTheme(), testStyles())

	cp.Open()
	if !cp.Visible {
		t.Fatal("should be visible after Open")
	}
	if cp.cursor != 0 {
		t.Fatalf("cursor should reset to 0, got %d", cp.cursor)
	}

	cp.Close()
	if cp.Visible {
		t.Fatal("should be hidden after Close")
	}
}

func TestCommandPalette_Esc_ClosesAndResets(t *testing.T) {
	cp := NewCommandPalette(testTheme(), testStyles())
	cp.Open()

	cp, cmd := cp.Update(specialKeyMsg(tea.KeyEscape))
	if cp.Visible {
		t.Fatal("palette should close on esc")
	}
	if cmd == nil {
		t.Fatal("esc should emit SetModeMsg")
	}
	// Verify commands are reset to defaults
	if cp.commands[0].Name != defaultCommands[0].Name {
		t.Fatal("commands should be reset after esc")
	}
}

func TestCommandPalette_Enter_SelectsItem(t *testing.T) {
	cp := NewCommandPalette(testTheme(), testStyles())
	cp.Open()

	// First item is "Send Request"
	cp, cmd := cp.Update(specialKeyMsg(tea.KeyEnter))
	if cp.Visible {
		t.Fatal("palette should close after selection")
	}
	if cmd == nil {
		t.Fatal("enter should produce a cmd")
	}
}

func TestCommandPalette_Navigation_JK(t *testing.T) {
	cp := NewCommandPalette(testTheme(), testStyles())
	cp.Open()

	if cp.cursor != 0 {
		t.Fatalf("initial cursor should be 0, got %d", cp.cursor)
	}

	// j / down moves cursor down
	cp, _ = cp.Update(keyMsg("j"))
	if cp.cursor != 1 {
		t.Fatalf("after j: expected cursor 1, got %d", cp.cursor)
	}

	cp, _ = cp.Update(specialKeyMsg(tea.KeyDown))
	if cp.cursor != 2 {
		t.Fatalf("after down: expected cursor 2, got %d", cp.cursor)
	}

	// k / up moves cursor up
	cp, _ = cp.Update(keyMsg("k"))
	if cp.cursor != 1 {
		t.Fatalf("after k: expected cursor 1, got %d", cp.cursor)
	}

	cp, _ = cp.Update(specialKeyMsg(tea.KeyUp))
	if cp.cursor != 0 {
		t.Fatalf("after up: expected cursor 0, got %d", cp.cursor)
	}

	// Can't go above 0
	cp, _ = cp.Update(keyMsg("k"))
	if cp.cursor != 0 {
		t.Fatalf("k at top: expected cursor 0, got %d", cp.cursor)
	}
}

func TestCommandPalette_Navigation_CantGoPastEnd(t *testing.T) {
	cp := NewCommandPalette(testTheme(), testStyles())
	cp.Open()

	lastIdx := len(cp.filtered) - 1
	// Move cursor to last item
	for i := 0; i < lastIdx+5; i++ {
		cp, _ = cp.Update(keyMsg("j"))
	}
	if cp.cursor != lastIdx {
		t.Fatalf("cursor should stop at last index %d, got %d", lastIdx, cp.cursor)
	}
}

func TestCommandPalette_OpenEnvPicker(t *testing.T) {
	cp := NewCommandPalette(testTheme(), testStyles())
	envs := []string{"Development", "Staging", "Production"}
	cp.OpenEnvPicker(envs)

	if !cp.Visible {
		t.Fatal("env picker should be visible")
	}
	if len(cp.commands) != 3 {
		t.Fatalf("expected 3 env commands, got %d", len(cp.commands))
	}
	if cp.commands[0].Name != "Development" {
		t.Fatalf("first env command should be 'Development', got '%s'", cp.commands[0].Name)
	}
	if cp.input.Placeholder != "Select environment..." {
		t.Fatalf("placeholder should be 'Select environment...', got '%s'", cp.input.Placeholder)
	}
}

func TestCommandPalette_OpenThemePicker(t *testing.T) {
	cp := NewCommandPalette(testTheme(), testStyles())
	themes := []string{"Catppuccin Mocha", "Nord", "Dracula"}
	cp.OpenThemePicker(themes)

	if !cp.Visible {
		t.Fatal("theme picker should be visible")
	}
	if len(cp.commands) != 3 {
		t.Fatalf("expected 3 theme commands, got %d", len(cp.commands))
	}
	if cp.commands[1].Name != "Nord" {
		t.Fatalf("second theme command should be 'Nord', got '%s'", cp.commands[1].Name)
	}
	if cp.input.Placeholder != "Select theme..." {
		t.Fatalf("placeholder should be 'Select theme...', got '%s'", cp.input.Placeholder)
	}
}

func TestCommandPalette_ResetCommands(t *testing.T) {
	cp := NewCommandPalette(testTheme(), testStyles())
	cp.OpenEnvPicker([]string{"Dev"})
	cp.ResetCommands()

	if len(cp.commands) != len(defaultCommands) {
		t.Fatalf("expected %d default commands, got %d", len(defaultCommands), len(cp.commands))
	}
	if cp.input.Placeholder != "Type a command..." {
		t.Fatalf("placeholder should be reset, got '%s'", cp.input.Placeholder)
	}
}

func TestCommandPalette_IgnoresInputWhenHidden(t *testing.T) {
	cp := NewCommandPalette(testTheme(), testStyles())
	cp, cmd := cp.Update(specialKeyMsg(tea.KeyEnter))
	if cmd != nil {
		t.Fatal("hidden palette should not produce cmds")
	}
}

func TestCommandPalette_View_WhenHidden(t *testing.T) {
	cp := NewCommandPalette(testTheme(), testStyles())
	view := cp.View()
	if view != "" {
		t.Fatalf("hidden palette should render empty, got: %q", view)
	}
}

func TestCommandPalette_View_WhenVisible(t *testing.T) {
	cp := NewCommandPalette(testTheme(), testStyles())
	cp.Open()

	view := cp.View()
	if view == "" {
		t.Fatal("visible palette should not render empty")
	}
	if !strings.Contains(view, "Command Palette") {
		t.Error("view should contain 'Command Palette' title")
	}
	// Should contain at least one command name
	if !strings.Contains(view, "Send Request") {
		t.Error("view should contain 'Send Request' command")
	}
}

func TestCommandPalette_EnvPicker_Enter_EmitsEnvMsg(t *testing.T) {
	cp := NewCommandPalette(testTheme(), testStyles())
	cp.OpenEnvPicker([]string{"Production", "Staging"})

	// Move to second item
	cp, _ = cp.Update(keyMsg("j"))
	// Select it
	cp, cmd := cp.Update(specialKeyMsg(tea.KeyEnter))
	if cp.Visible {
		t.Fatal("should close after selection")
	}
	if cmd == nil {
		t.Fatal("enter should produce cmd")
	}
}

func TestCommandPalette_ThemePicker_Enter_EmitsThemeMsg(t *testing.T) {
	cp := NewCommandPalette(testTheme(), testStyles())
	cp.OpenThemePicker([]string{"Nord", "Dracula"})

	// Select first item
	cp, cmd := cp.Update(specialKeyMsg(tea.KeyEnter))
	if cp.Visible {
		t.Fatal("should close after selection")
	}
	if cmd == nil {
		t.Fatal("enter should produce cmd")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Help tests
// ─────────────────────────────────────────────────────────────────────────────

func TestHelp_NewDefault(t *testing.T) {
	h := NewHelp(testTheme(), testStyles())
	if h.Visible {
		t.Fatal("help should start hidden")
	}
}

func TestHelp_Toggle(t *testing.T) {
	h := NewHelp(testTheme(), testStyles())
	h.SetSize(120, 40)

	h.Toggle()
	if !h.Visible {
		t.Fatal("should be visible after first toggle")
	}

	h.Toggle()
	if h.Visible {
		t.Fatal("should be hidden after second toggle")
	}
}

func TestHelp_Esc_Closes(t *testing.T) {
	h := NewHelp(testTheme(), testStyles())
	h.SetSize(120, 40)
	h.Toggle()

	h, cmd := h.Update(specialKeyMsg(tea.KeyEscape))
	if h.Visible {
		t.Fatal("help should close on esc")
	}
	if cmd == nil {
		t.Fatal("esc should emit SetModeMsg")
	}
	msg := cmd()
	if setMode, ok := msg.(msgs.SetModeMsg); ok {
		if setMode.Mode != msgs.ModeNormal {
			t.Fatalf("expected ModeNormal, got %d", setMode.Mode)
		}
	} else {
		t.Fatalf("expected SetModeMsg, got %T", msg)
	}
}

func TestHelp_QuestionMark_Closes(t *testing.T) {
	h := NewHelp(testTheme(), testStyles())
	h.SetSize(120, 40)
	h.Toggle()

	h, cmd := h.Update(keyMsg("?"))
	if h.Visible {
		t.Fatal("help should close on ?")
	}
	if cmd == nil {
		t.Fatal("? should emit SetModeMsg")
	}
}

func TestHelp_IgnoresInputWhenHidden(t *testing.T) {
	h := NewHelp(testTheme(), testStyles())
	h, cmd := h.Update(specialKeyMsg(tea.KeyEscape))
	if cmd != nil {
		t.Fatal("hidden help should not produce cmds")
	}
}

func TestHelp_View_WhenHidden(t *testing.T) {
	h := NewHelp(testTheme(), testStyles())
	view := h.View()
	if view != "" {
		t.Fatalf("hidden help should render empty, got length: %d", len(view))
	}
}

func TestHelp_View_WhenVisible(t *testing.T) {
	h := NewHelp(testTheme(), testStyles())
	h.SetSize(120, 40)
	h.Toggle()

	view := h.View()
	if view == "" {
		t.Fatal("visible help should not render empty")
	}
	if !strings.Contains(view, "Keyboard Shortcuts") {
		t.Error("help view should contain 'Keyboard Shortcuts' title")
	}
	// Should contain section titles
	if !strings.Contains(view, "General") {
		t.Error("help view should contain 'General' section")
	}
	if !strings.Contains(view, "Sidebar") {
		t.Error("help view should contain 'Sidebar' section")
	}
	if !strings.Contains(view, "Editor") {
		t.Error("help view should contain 'Editor' section")
	}
	if !strings.Contains(view, "Response") {
		t.Error("help view should contain 'Response' section")
	}
}

func TestHelp_SetSize(t *testing.T) {
	h := NewHelp(testTheme(), testStyles())
	h.SetSize(200, 50)
	if h.width != 200 {
		t.Fatalf("expected width 200, got %d", h.width)
	}
	if h.height != 50 {
		t.Fatalf("expected height 50, got %d", h.height)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// formatDuration tests (StatusBar helper)
// ─────────────────────────────────────────────────────────────────────────────

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		dur      time.Duration
		expected string
	}{
		{"microseconds", 500 * time.Microsecond, "500µs"},
		{"milliseconds", 150 * time.Millisecond, "150ms"},
		{"seconds", 2500 * time.Millisecond, "2.50s"},
		{"sub-millisecond", 999 * time.Microsecond, "999µs"},
		{"exactly 1ms", time.Millisecond, "1ms"},
		{"exactly 1s", time.Second, "1.00s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDuration(tt.dur)
			if got != tt.expected {
				t.Fatalf("formatDuration(%v) = %q, want %q", tt.dur, got, tt.expected)
			}
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// truncate helper tests
// ─────────────────────────────────────────────────────────────────────────────

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxW     int
		expected string
	}{
		{"short string", "abc", 10, "abc"},
		{"exact fit", "abcde", 5, "abcde"},
		{"needs truncation", "hello world", 8, "hello..."},
		{"very short max", "hello", 3, "hel"},
		{"zero max", "hello", 0, ""},
		{"empty string", "", 10, ""},
		{"negative max", "hello", -1, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.input, tt.maxW)
			if got != tt.expected {
				t.Fatalf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxW, got, tt.expected)
			}
		})
	}
}
