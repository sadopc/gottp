package layout

import tea "github.com/charmbracelet/bubbletea"

// HandleResize processes a WindowSizeMsg and returns the updated layout.
func HandleResize(msg tea.WindowSizeMsg, sidebarVisible bool) PanelLayout {
	return Calculate(msg.Width, msg.Height, sidebarVisible)
}
