package sidebar

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sadopc/gottp/internal/core/collection"
	"github.com/sadopc/gottp/internal/ui/msgs"
	"github.com/sadopc/gottp/internal/ui/theme"
)

func newSidebarModelForTest() Model {
	th := theme.Default()
	m := New(th, theme.NewStyles(th))
	m.SetSize(80, 20)
	return m
}

func TestSidebar_FilterToggleAndRequestSelection(t *testing.T) {
	m := newSidebarModelForTest()

	items := []collection.FlatItem{
		{IsFolder: true, Expanded: true, Depth: 0, Folder: &collection.Folder{Name: "Folder"}},
		{Depth: 1, Request: &collection.Request{ID: "r1", Name: "Child", Method: "GET"}},
		{Depth: 0, Request: &collection.Request{ID: "r2", Name: "Root", Method: "POST"}},
	}
	m.SetItems(items)
	if len(m.filtered) != 3 {
		t.Fatalf("filtered len = %d, want 3", len(m.filtered))
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !updated.filtering {
		t.Fatal("expected filtering mode enabled")
	}

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	if got := len(updated.filtered); got != 1 {
		t.Fatalf("filtered len after query = %d, want 1", got)
	}

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if updated.filtering {
		t.Fatal("expected filtering mode disabled on esc")
	}
	if got := updated.filterInput.Value(); got != "" {
		t.Fatalf("filter input = %q, want empty", got)
	}

	m = updated
	m.cursor = 1 // select child request
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected RequestSelected command")
	}
	msg := cmd()
	sel, ok := msg.(msgs.RequestSelectedMsg)
	if !ok {
		t.Fatalf("expected RequestSelectedMsg, got %T", msg)
	}
	if sel.RequestID != "r1" {
		t.Fatalf("request id = %q, want r1", sel.RequestID)
	}

	_ = updated
}

func TestSidebar_FolderToggleAndHistorySelection(t *testing.T) {
	m := newSidebarModelForTest()
	m.SetItems([]collection.FlatItem{
		{IsFolder: true, Expanded: true, Depth: 0, Folder: &collection.Folder{Name: "Folder"}},
		{Depth: 1, Request: &collection.Request{ID: "r1", Name: "Child", Method: "GET"}},
	})
	m.SetHistory([]HistoryItem{
		{ID: 11, Method: "GET", URL: "https://api.example.com/users", Timestamp: time.Now().Add(-2 * time.Minute)},
		{ID: 22, Method: "POST", URL: "https://api.example.com/items", Timestamp: time.Now().Add(-2 * time.Hour)},
	})

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if updated.items[0].Expanded {
		t.Fatal("expected folder to collapse on enter")
	}
	if got := len(updated.filtered); got != 1 {
		t.Fatalf("filtered len after collapse = %d, want 1", got)
	}

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	if !updated.inHistory || updated.historyCursor != 1 {
		t.Fatalf("expected in history at last item, got inHistory=%v cursor=%d", updated.inHistory, updated.historyCursor)
	}

	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected HistorySelected command")
	}
	msg := cmd()
	hsel, ok := msg.(msgs.HistorySelectedMsg)
	if !ok {
		t.Fatalf("expected HistorySelectedMsg, got %T", msg)
	}
	if hsel.ID != 22 {
		t.Fatalf("history id = %d, want 22", hsel.ID)
	}
}

func TestSidebar_ViewAndHelpers(t *testing.T) {
	m := newSidebarModelForTest()
	m.SetFocused(true)
	m.SetItems([]collection.FlatItem{{Depth: 0, Request: &collection.Request{Name: "Health", Method: "GET"}}})
	m.SetHistory([]HistoryItem{{Method: "GET", URL: "https://example.com", Timestamp: time.Now().Add(-30 * time.Second)}})

	v := m.View()
	if !strings.Contains(v, "Collections") {
		t.Fatalf("view missing Collections header: %q", v)
	}
	if !strings.Contains(v, "History") {
		t.Fatalf("view missing History header: %q", v)
	}

	if got := padMethod("GET"); got != "GET   " {
		t.Fatalf("padMethod(GET) = %q", got)
	}
	if got := padMethod("OPTIONS"); got != "OPTION" {
		t.Fatalf("padMethod(OPTIONS) = %q", got)
	}

	if got := formatTimeAgo(time.Now().Add(-10 * time.Second)); got != "now" {
		t.Fatalf("formatTimeAgo(<1m) = %q, want now", got)
	}
	if got := formatTimeAgo(time.Now().Add(-3 * time.Minute)); got != "3m" {
		t.Fatalf("formatTimeAgo(3m) = %q", got)
	}

	if got := stripForWidth("abcdef", 3); got != "abc" {
		t.Fatalf("stripForWidth = %q, want abc", got)
	}
	if got := m.fitHeight("a\nb\nc", 2); got != "a\nb" {
		t.Fatalf("fitHeight truncate = %q", got)
	}
}
