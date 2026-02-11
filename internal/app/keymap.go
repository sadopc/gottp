package app

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all application keybindings.
type KeyMap struct {
	// Global
	Quit          key.Binding
	SendRequest   key.Binding
	CommandPalette key.Binding
	Help          key.Binding
	NewRequest    key.Binding
	CloseTab      key.Binding
	SaveRequest   key.Binding
	SwitchEnv     key.Binding

	// Panel navigation
	CycleFocus    key.Binding
	CycleFocusRev key.Binding
	FocusSidebar  key.Binding
	FocusEditor   key.Binding
	FocusResponse key.Binding
	ToggleSidebar key.Binding

	// Tab navigation
	PrevTab key.Binding
	NextTab key.Binding
}

// DefaultKeyMap returns the default keybinding configuration.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "quit"),
		),
		SendRequest: key.NewBinding(
			key.WithKeys("ctrl+r"),
			key.WithHelp("ctrl+r", "send request"),
		),
		CommandPalette: key.NewBinding(
			key.WithKeys("ctrl+k"),
			key.WithHelp("ctrl+k", "command palette"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		NewRequest: key.NewBinding(
			key.WithKeys("ctrl+n"),
			key.WithHelp("ctrl+n", "new request"),
		),
		CloseTab: key.NewBinding(
			key.WithKeys("ctrl+w"),
			key.WithHelp("ctrl+w", "close tab"),
		),
		SaveRequest: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("ctrl+s", "save"),
		),
		SwitchEnv: key.NewBinding(
			key.WithKeys("ctrl+e"),
			key.WithHelp("ctrl+e", "switch env"),
		),
		CycleFocus: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next panel"),
		),
		CycleFocusRev: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev panel"),
		),
		FocusSidebar: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "sidebar"),
		),
		FocusEditor: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "editor"),
		),
		FocusResponse: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "response"),
		),
		ToggleSidebar: key.NewBinding(
			key.WithKeys("b"),
			key.WithHelp("b", "toggle sidebar"),
		),
		PrevTab: key.NewBinding(
			key.WithKeys("["),
			key.WithHelp("[", "prev tab"),
		),
		NextTab: key.NewBinding(
			key.WithKeys("]"),
			key.WithHelp("]", "next tab"),
		),
	}
}
