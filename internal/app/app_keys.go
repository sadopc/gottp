package app

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/serdar/gottp/internal/ui/components"
	"github.com/serdar/gottp/internal/ui/layout"
	"github.com/serdar/gottp/internal/ui/msgs"
)

func (a App) handleGlobalKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, a.keys.Quit):
		return tea.Quit
	case key.Matches(msg, a.keys.SendRequest):
		return func() tea.Msg { return msgs.SendRequestMsg{} }
	case msg.String() == "ctrl+enter":
		return func() tea.Msg { return msgs.SendRequestMsg{} }
	case key.Matches(msg, a.keys.CommandPalette):
		return func() tea.Msg { return msgs.OpenCommandPaletteMsg{} }
	case key.Matches(msg, a.keys.NewRequest):
		return func() tea.Msg { return msgs.NewRequestMsg{} }
	case key.Matches(msg, a.keys.CloseTab):
		return func() tea.Msg { return msgs.CloseTabMsg{} }
	case key.Matches(msg, a.keys.SaveRequest):
		return func() tea.Msg { return msgs.SaveRequestMsg{} }
	case key.Matches(msg, a.keys.PrevTab):
		return func() tea.Msg { return msgs.PrevTabMsg{} }
	case key.Matches(msg, a.keys.NextTab):
		return func() tea.Msg { return msgs.NextTabMsg{} }
	}
	return nil
}

func (a App) handlePanelKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		a.cycleFocus(false)
		return a, nil
	case "shift+tab":
		a.cycleFocus(true)
		return a, nil
	case "b":
		a.sidebarVisible = !a.sidebarVisible
		a.layout = layout.Calculate(a.width, a.height, a.sidebarVisible)
		a.resizePanels()
		return a, nil
	case "?":
		a.mode = msgs.ModeModal
		a.help.SetSize(a.width, a.height)
		a.help.Toggle()
		return a, nil
	case "i":
		// Enter insert mode: focus URL input in editor
		if a.focus == msgs.FocusEditor {
			a.mode = msgs.ModeInsert
			a.statusBar.SetMode(msgs.ModeInsert)
			a.editor.FocusURL()
			return a, nil
		}
	case "enter":
		// Send request when editor is focused in normal mode
		if a.focus == msgs.FocusEditor {
			return a.sendRequest()
		}
	case "S":
		// Capital S as alternative send shortcut (always works)
		return a.sendRequest()
	case "f":
		// Activate jump mode
		a.activateJumpMode()
		return a, nil
	case "E":
		// Open body in $EDITOR
		return a.openExternalEditor()
	}

	var cmd tea.Cmd
	switch a.focus {
	case msgs.FocusSidebar:
		a.sidebar, cmd = a.sidebar.Update(msg)
	case msgs.FocusEditor:
		a.editor, cmd = a.editor.Update(msg)
	case msgs.FocusResponse:
		a.response, cmd = a.response.Update(msg)
	}

	return a, cmd
}

func (a App) updateEditorInsert(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		a.mode = msgs.ModeNormal
		a.statusBar.SetMode(msgs.ModeNormal)
	}
	if msg.String() == "ctrl+enter" {
		return a.sendRequest()
	}

	var cmd tea.Cmd
	a.editor, cmd = a.editor.Update(msg)

	if a.editor.Editing() {
		a.mode = msgs.ModeInsert
		a.statusBar.SetMode(msgs.ModeInsert)
	} else {
		a.mode = msgs.ModeNormal
		a.statusBar.SetMode(msgs.ModeNormal)
	}

	return a, cmd
}

func (a *App) cycleFocus(reverse bool) {
	panels := []msgs.PanelFocus{msgs.FocusSidebar, msgs.FocusEditor, msgs.FocusResponse}
	if !a.sidebarVisible {
		panels = []msgs.PanelFocus{msgs.FocusEditor, msgs.FocusResponse}
	}

	idx := 0
	for i, p := range panels {
		if p == a.focus {
			idx = i
			break
		}
	}

	if reverse {
		idx = (idx - 1 + len(panels)) % len(panels)
	} else {
		idx = (idx + 1) % len(panels)
	}

	a.focus = panels[idx]
	a.updateFocus()
}

func (a *App) updateFocus() {
	a.sidebar.SetFocused(a.focus == msgs.FocusSidebar)
	a.editor.SetFocused(a.focus == msgs.FocusEditor)
	a.response.SetFocused(a.focus == msgs.FocusResponse)
}

func (a *App) activateJumpMode() {
	targets := []components.JumpTarget{
		{Name: "Sidebar", Panel: msgs.FocusSidebar, Action: msgs.FocusPanelMsg{Panel: msgs.FocusSidebar}},
		{Name: "Editor", Panel: msgs.FocusEditor, Action: msgs.FocusPanelMsg{Panel: msgs.FocusEditor}},
		{Name: "Response", Panel: msgs.FocusResponse, Action: msgs.FocusPanelMsg{Panel: msgs.FocusResponse}},
		{Name: "Send Request", Action: msgs.SendRequestMsg{}},
		{Name: "New Request", Action: msgs.NewRequestMsg{}},
		{Name: "Command Palette", Action: msgs.OpenCommandPaletteMsg{}},
		{Name: "Save", Action: msgs.SaveRequestMsg{}},
		{Name: "Toggle Sidebar", Action: msgs.ToggleSidebarMsg{}},
	}
	a.jump.Open(targets)
	a.mode = msgs.ModeJump
	a.statusBar.SetMode(msgs.ModeJump)
}
