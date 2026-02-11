package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/serdar/gottp/internal/ui/theme"
)

// KVPair represents a key-value pair with an enabled toggle.
type KVPair struct {
	Key     string
	Value   string
	Enabled bool
}

// Column identifies which column is focused.
type Column int

const (
	ColKey Column = iota
	ColValue
)

// KVTable is a reusable key-value pair editor component.
type KVTable struct {
	pairs   []KVPair
	cursor  int
	column  Column
	editing bool
	input   textinput.Model
	width   int
	styles  theme.Styles
}

// NewKVTable creates a new KVTable.
func NewKVTable(styles theme.Styles) KVTable {
	ti := textinput.New()
	ti.CharLimit = 256

	return KVTable{
		pairs:  []KVPair{{Key: "", Value: "", Enabled: true}},
		styles: styles,
		input:  ti,
		width:  60,
	}
}

// SetPairs replaces all pairs.
func (m *KVTable) SetPairs(pairs []KVPair) {
	m.pairs = pairs
	if len(m.pairs) == 0 {
		m.pairs = []KVPair{{Key: "", Value: "", Enabled: true}}
	}
	if m.cursor >= len(m.pairs) {
		m.cursor = len(m.pairs) - 1
	}
}

// GetPairs returns a copy of all pairs.
func (m KVTable) GetPairs() []KVPair {
	out := make([]KVPair, len(m.pairs))
	copy(out, m.pairs)
	return out
}

// SetSize sets the table width.
func (m *KVTable) SetSize(w int) {
	m.width = w
}

// Editing returns whether the table is in edit mode.
func (m KVTable) Editing() bool {
	return m.editing
}

// Init implements tea.Model.
func (m KVTable) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m KVTable) Update(msg tea.Msg) (KVTable, tea.Cmd) {
	if m.editing {
		return m.updateEditing(msg)
	}
	return m.updateNormal(msg)
}

func (m KVTable) updateNormal(msg tea.Msg) (KVTable, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if m.cursor < len(m.pairs)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "tab":
			if m.column == ColKey {
				m.column = ColValue
			} else {
				m.column = ColKey
			}
		case "enter":
			m.startEditing()
			return m, textinput.Blink
		case "a":
			m.pairs = append(m.pairs, KVPair{Key: "", Value: "", Enabled: true})
			m.cursor = len(m.pairs) - 1
			m.column = ColKey
			m.startEditing()
			return m, textinput.Blink
		case "d":
			if len(m.pairs) > 1 {
				m.pairs = append(m.pairs[:m.cursor], m.pairs[m.cursor+1:]...)
				if m.cursor >= len(m.pairs) {
					m.cursor = len(m.pairs) - 1
				}
			} else {
				m.pairs[0] = KVPair{Key: "", Value: "", Enabled: true}
			}
		case " ":
			m.pairs[m.cursor].Enabled = !m.pairs[m.cursor].Enabled
		}
	}
	return m, nil
}

func (m KVTable) updateEditing(msg tea.Msg) (KVTable, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.commitEdit()
			m.editing = false
			return m, nil
		case "enter":
			m.commitEdit()
			m.editing = false
			return m, nil
		case "tab":
			m.commitEdit()
			if m.column == ColKey {
				m.column = ColValue
			} else {
				m.column = ColKey
			}
			m.startEditing()
			return m, textinput.Blink
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *KVTable) startEditing() {
	m.editing = true
	if m.column == ColKey {
		m.input.SetValue(m.pairs[m.cursor].Key)
	} else {
		m.input.SetValue(m.pairs[m.cursor].Value)
	}
	m.input.Focus()
	m.input.CursorEnd()
}

func (m *KVTable) commitEdit() {
	if m.cursor >= len(m.pairs) {
		return
	}
	if m.column == ColKey {
		m.pairs[m.cursor].Key = m.input.Value()
	} else {
		m.pairs[m.cursor].Value = m.input.Value()
	}
	m.input.Blur()
}

// View implements tea.Model.
func (m KVTable) View() string {
	if len(m.pairs) == 0 {
		return m.styles.Muted.Render("  No entries")
	}

	// Calculate column widths: [x] key | value
	// Checkbox takes 4 chars: "[x] "
	// Separator takes 3 chars: " | "
	checkboxW := 4
	separatorW := 3
	available := m.width - checkboxW - separatorW
	if available < 10 {
		available = 10
	}
	keyW := available / 2
	valW := available - keyW

	m.input.Width = keyW - 1
	if m.column == ColValue {
		m.input.Width = valW - 1
	}

	var rows []string
	for i, pair := range m.pairs {
		isCursor := i == m.cursor

		// Cursor prefix
		prefix := "  "
		if isCursor {
			prefix = "> "
		}

		// Checkbox
		check := "[ ] "
		if pair.Enabled {
			check = "[x] "
		}

		var keyStr, valStr string

		if isCursor && m.editing && m.column == ColKey {
			keyStr = m.input.View()
		} else {
			keyStr = truncate(pair.Key, keyW)
			if keyStr == "" {
				keyStr = "key"
			}
		}

		if isCursor && m.editing && m.column == ColValue {
			valStr = m.input.View()
		} else {
			valStr = truncate(pair.Value, valW)
			if valStr == "" {
				valStr = "value"
			}
		}

		// Apply styles
		sep := m.styles.KVSeparator.Render(" | ")

		if !pair.Enabled {
			check = m.styles.KVDisabled.Render(check)
			keyStr = m.styles.KVDisabled.Render(padRight(keyStr, keyW))
			valStr = m.styles.KVDisabled.Render(padRight(valStr, valW))
		} else if isCursor {
			check = m.styles.Normal.Render(check)
			if !(m.editing && m.column == ColKey) {
				if m.column == ColKey {
					keyStr = m.styles.Cursor.Render(padRight(keyStr, keyW))
				} else {
					keyStr = m.styles.KVKey.Render(padRight(keyStr, keyW))
				}
			} else {
				keyStr = padRight(keyStr, keyW)
			}
			if !(m.editing && m.column == ColValue) {
				if m.column == ColValue {
					valStr = m.styles.Cursor.Render(padRight(valStr, valW))
				} else {
					valStr = m.styles.KVValue.Render(padRight(valStr, valW))
				}
			} else {
				valStr = padRight(valStr, valW)
			}
		} else {
			check = m.styles.Muted.Render(check)
			if pair.Key == "" {
				keyStr = m.styles.Muted.Render(padRight(keyStr, keyW))
			} else {
				keyStr = m.styles.KVKey.Render(padRight(keyStr, keyW))
			}
			if pair.Value == "" {
				valStr = m.styles.Muted.Render(padRight(valStr, valW))
			} else {
				valStr = m.styles.KVValue.Render(padRight(valStr, valW))
			}
		}

		row := prefix + check + keyStr + sep + valStr
		rows = append(rows, row)
	}

	return strings.Join(rows, "\n")
}

func truncate(s string, maxW int) string {
	if maxW <= 0 {
		return ""
	}
	if len(s) > maxW {
		if maxW > 3 {
			return s[:maxW-3] + "..."
		}
		return s[:maxW]
	}
	return s
}

func padRight(s string, width int) string {
	// Use lipgloss to handle ANSI-aware width
	return lipgloss.NewStyle().Width(width).Render(s)
}
