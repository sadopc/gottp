package response

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/serdar/gottp/internal/diff"
	"github.com/serdar/gottp/internal/protocol"
	"github.com/serdar/gottp/internal/ui/theme"
)

func newResponseModelForTest() Model {
	th := theme.Default()
	m := New(th, theme.NewStyles(th))
	m.SetSize(100, 24)
	return m
}

func TestResponseModel_ModeTabsAndSetResponse(t *testing.T) {
	m := newResponseModelForTest()
	if got := len(m.tabLabels()); got != 6 {
		t.Fatalf("http tab count = %d, want 6", got)
	}

	m.SetMode("websocket")
	if got := len(m.tabLabels()); got != 3 {
		t.Fatalf("ws tab count = %d, want 3", got)
	}
	if m.active != wsTabMessages {
		t.Fatalf("active tab in ws mode = %d, want %d", m.active, wsTabMessages)
	}

	m.SetMode("http")
	resp := &protocol.Response{
		StatusCode:  200,
		Status:      "200 OK",
		Body:        []byte(`{"ok":true}`),
		ContentType: "application/json",
		Headers:     http.Header{"Content-Type": {"application/json"}},
		Duration:    120 * time.Millisecond,
		Size:        128,
		Proto:       "HTTP/1.1",
		TLS:         true,
	}
	m.SetBaseline([]byte(`{"ok":false}`))
	m.SetResponse(resp)

	if !m.hasResp {
		t.Fatal("expected hasResp true")
	}
	if m.code != 200 || m.status != "200 OK" {
		t.Fatalf("unexpected status state code=%d status=%q", m.code, m.status)
	}
	if !m.diff.HasDiff() {
		t.Fatal("expected diff to be computed when baseline exists")
	}
	if !m.HasBaseline() {
		t.Fatal("expected baseline to be present")
	}
	if got := string(m.ResponseBody()); !strings.Contains(got, "ok") {
		t.Fatalf("unexpected response body: %q", got)
	}

	m.ClearBaseline()
	if m.HasBaseline() {
		t.Fatal("expected baseline to be cleared")
	}
}

func TestResponseModel_UpdateAndViewStates(t *testing.T) {
	m := newResponseModelForTest()
	if got := m.View(); !strings.Contains(got, "Send a request") {
		t.Fatalf("empty view missing prompt: %q", got)
	}

	m.SetLoading(true)
	updated, cmd := m.Update(spinner.TickMsg{})
	if cmd == nil {
		t.Fatal("expected spinner tick command while loading")
	}
	if got := updated.View(); !strings.Contains(got, "Sending request") {
		t.Fatalf("loading view missing text: %q", got)
	}

	updated.loading = false
	updated.hasResp = true
	updated.status = "204 No Content"

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	if updated.active != 1 {
		t.Fatalf("active tab = %d, want 1", updated.active)
	}
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyTab})
	if updated.active != 2 {
		t.Fatalf("active tab after tab = %d, want 2", updated.active)
	}
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	if updated.active != 1 {
		t.Fatalf("active tab after shift+tab = %d, want 1", updated.active)
	}

	updated.SetMode("websocket")
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}})
	if updated.active > wsTabTiming {
		t.Fatalf("unexpected ws active tab after '4': %d", updated.active)
	}
}

func TestResponseSubmodels_BodyHeadersCookiesTimingDiffConsoleWS(t *testing.T) {
	th := theme.Default()
	styles := theme.NewStyles(th)

	body := NewBodyModel(styles)
	body.SetSize(40, 8)
	body.SetContent([]byte("line1\nline2\nline1"), "text/plain")
	if !strings.Contains(body.View(), "line1") {
		t.Fatalf("body view missing content: %q", body.View())
	}
	body, _ = body.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !body.Searching() {
		t.Fatal("expected body searching mode after '/'")
	}
	body.search.SetMatches([]int{0, 2})
	body.search.query = "line1"
	body.search.input.Blur()
	body, _ = body.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if body.search.current != 1 {
		t.Fatalf("expected current match index 1, got %d", body.search.current)
	}
	body, _ = body.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if body.Searching() {
		t.Fatal("expected search mode to close on esc")
	}

	if got := detectLexer("application/json; charset=utf-8"); got != "json" {
		t.Fatalf("detectLexer json = %q", got)
	}
	if got := detectLexer("text/unknown"); got != "text" {
		t.Fatalf("detectLexer default = %q", got)
	}
	if _, matches := HighlightMatches("Abc\nxxxabc", "abc"); len(matches) != 2 {
		t.Fatalf("highlight matches len = %d, want 2", len(matches))
	}

	headers := NewHeadersModel(styles)
	headers.SetSize(60, 8)
	headers.SetHeaders(http.Header{"X-Beta": {"2"}, "X-Alpha": {"1"}})
	if !strings.Contains(headers.View(), "X-Alpha") {
		t.Fatalf("headers view missing X-Alpha: %q", headers.View())
	}

	cookies := NewCookiesModel(styles)
	cookies.SetSize(80, 8)
	cookies.SetHeaders(http.Header{"Set-Cookie": {"sid=abc; Path=/; HttpOnly", "lang=en; Secure"}})
	if !strings.Contains(cookies.View(), "sid") {
		t.Fatalf("cookies view missing sid: %q", cookies.View())
	}
	if got := truncateCookie("abcdef", 4); got != "a..." {
		t.Fatalf("truncateCookie = %q, want a...", got)
	}

	timing := NewTimingModel(th, styles)
	timing.SetSize(100, 12)
	timing.SetResponse(&protocol.Response{
		Duration: 250 * time.Millisecond,
		Size:     1536,
		Proto:    "HTTP/2",
		TLS:      true,
		Timing: &protocol.TimingDetail{
			DNSLookup:    10 * time.Millisecond,
			TCPConnect:   20 * time.Millisecond,
			TLSHandshake: 30 * time.Millisecond,
			TTFB:         40 * time.Millisecond,
			Transfer:     50 * time.Millisecond,
			Total:        150 * time.Millisecond,
		},
	})
	if !strings.Contains(timing.View(), "Waterfall") {
		t.Fatalf("timing view missing Waterfall: %q", timing.View())
	}
	if got := formatSize(2048); got != "2.0 KB" {
		t.Fatalf("formatSize(2048) = %q", got)
	}
	if got := formatDuration(999 * time.Microsecond); got != "999µs" {
		t.Fatalf("formatDuration(999us) = %q", got)
	}

	dm := NewDiffModel(th, styles)
	dm.SetSize(80, 8)
	dm.SetDiff([]byte("hello world"), []byte("hello brave world"))
	if !dm.HasDiff() {
		t.Fatal("expected diff model to have diff")
	}
	if !strings.Contains(dm.View(), "Diff:") {
		t.Fatalf("diff view missing header: %q", dm.View())
	}
	var wb strings.Builder
	dm.renderWordDiffAdded(&wb, []diff.WordDiff{{Type: diff.Same, Content: "x"}, {Type: diff.Added, Content: "y"}})
	if wb.Len() == 0 {
		t.Fatal("expected added word diff rendering output")
	}
	dm.Clear()
	if dm.HasDiff() {
		t.Fatal("expected diff model clear to reset state")
	}

	console := NewConsoleModel(th, styles)
	console.SetSize(80, 8)
	console.SetResults([]string{"log1"}, []ScriptTestResult{{Name: "t1", Passed: false, Error: "boom"}}, "script failed")
	if !console.HasContent() {
		t.Fatal("expected console to have content")
	}
	if !strings.Contains(console.View(), "Error:") {
		t.Fatalf("console view missing error: %q", console.View())
	}
	console.Clear()
	if console.HasContent() {
		t.Fatal("expected console to be empty after clear")
	}

	ws := NewWSLogModel(th, styles)
	ws.SetSize(80, 8)
	ws.AddMessage(WSMessage{Direction: "sent", Content: "hello", Timestamp: time.Now()})
	if ws.MessageCount() != 1 {
		t.Fatalf("ws message count = %d, want 1", ws.MessageCount())
	}
	if !strings.Contains(ws.View(), "messages") {
		t.Fatalf("ws view missing messages header: %q", ws.View())
	}
	ws.Clear()
	if ws.MessageCount() != 0 {
		t.Fatalf("ws message count after clear = %d, want 0", ws.MessageCount())
	}
}

func TestResponseModel_WebSocketStatusAndScriptResults(t *testing.T) {
	m := newResponseModelForTest()
	m.SetMode("websocket")
	m.SetScriptResults([]string{"a"}, []ScriptTestResult{{Name: "ok", Passed: true}}, "")
	m.AddWSMessage(WSMessage{Direction: "received", Content: "pong", Timestamp: time.Now()})

	if !m.hasResp {
		t.Fatal("expected hasResp true after ws message")
	}
	if got := m.renderStatus(80); !strings.Contains(got, "WebSocket Connected") {
		t.Fatalf("status missing connected label: %q", got)
	}

	m.ClearWSLog()
	if got := m.renderStatus(80); !strings.Contains(got, "WebSocket") {
		t.Fatalf("status missing websocket label: %q", got)
	}

	m.SetResponse(nil)
	if m.hasResp {
		t.Fatal("expected hasResp false after nil response")
	}
}
