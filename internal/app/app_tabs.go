package app

import (
	"encoding/json"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/serdar/gottp/internal/core/collection"
	"github.com/serdar/gottp/internal/ui/components"
	"github.com/serdar/gottp/internal/ui/msgs"
	"github.com/serdar/gottp/internal/ui/panels/sidebar"
)

func (a *App) syncTabs() {
	tabs := make([]components.TabItem, len(a.store.Tabs))
	for i, t := range a.store.Tabs {
		tabs[i] = components.TabItem{
			Name:   t.Request.Name,
			Method: t.Request.Method,
		}
	}
	a.tabBar.SetTabs(tabs)
	a.tabBar.SetActive(a.store.ActiveTab)
}

func (a *App) loadActiveRequest() {
	req := a.store.ActiveRequest()
	if req != nil {
		a.editor.LoadRequest(req)
	}
}

func (a *App) loadHistory() {
	if a.history == nil {
		return
	}
	entries, err := a.history.List(20, 0)
	if err != nil {
		return
	}
	items := make([]sidebar.HistoryItem, len(entries))
	for i, e := range entries {
		items[i] = sidebar.HistoryItem{
			ID:         e.ID,
			Method:     e.Method,
			URL:        e.URL,
			StatusCode: e.StatusCode,
			Duration:   e.Duration,
			Timestamp:  e.Timestamp,
		}
	}
	a.sidebar.SetHistory(items)
}

func (a App) handleRequestSelected(msg msgs.RequestSelectedMsg) (tea.Model, tea.Cmd) {
	if a.store.Collection == nil {
		return a, nil
	}
	req := findRequest(a.store.Collection.Items, msg.RequestID)
	if req != nil {
		a.store.OpenRequest(req)
		a.syncTabs()
		a.editor.LoadRequest(req)
		a.focus = msgs.FocusEditor
		a.updateFocus()
	}
	return a, nil
}

func (a App) handleHistorySelected(msg msgs.HistorySelectedMsg) (tea.Model, tea.Cmd) {
	if a.history == nil {
		return a, nil
	}
	entries, err := a.history.List(100, 0)
	if err != nil {
		return a, nil
	}
	for _, e := range entries {
		if e.ID == msg.ID {
			// Create a new tab with the history entry
			colReq := collection.NewRequest("History", e.Method, e.URL)
			if e.RequestBody != "" {
				colReq.Body = &collection.Body{Type: "json", Content: e.RequestBody}
			}
			if e.Headers != "" {
				var headers map[string]string
				if json.Unmarshal([]byte(e.Headers), &headers) == nil {
					for k, v := range headers {
						colReq.Headers = append(colReq.Headers, collection.KVPair{Key: k, Value: v, Enabled: true})
					}
				}
			}
			a.store.OpenRequest(colReq)
			a.syncTabs()
			a.editor.LoadRequest(colReq)
			a.focus = msgs.FocusEditor
			a.updateFocus()
			break
		}
	}
	return a, nil
}
