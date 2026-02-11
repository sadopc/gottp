package app

import (
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/sadopc/gottp/internal/core/collection"
	importutil "github.com/sadopc/gottp/internal/import"
	"github.com/sadopc/gottp/internal/import/insomnia"
	"github.com/sadopc/gottp/internal/import/openapi"
	"github.com/sadopc/gottp/internal/import/postman"
	"github.com/sadopc/gottp/internal/ui/components"
	"github.com/sadopc/gottp/internal/ui/msgs"
	"github.com/sadopc/gottp/internal/ui/panels/response"
	"github.com/sadopc/gottp/internal/ui/panels/sidebar"
	"github.com/sadopc/gottp/internal/ui/theme"
)

func (a App) handleSwitchTheme(msg msgs.SwitchThemeMsg) (tea.Model, tea.Cmd) {
	if msg.Name == "" {
		// Open theme picker
		names := theme.Names()
		a.commandPalette.OpenThemePicker(names)
		a.mode = msgs.ModeCommandPalette
		return a, nil
	}

	t := theme.Resolve(msg.Name)
	s := theme.NewStyles(t)
	a.theme = t
	a.styles = s

	// Rebuild all components with new styles
	a.sidebar = sidebar.New(t, s)
	a.response = response.New(t, s)
	a.tabBar = components.NewTabBar(t, s)
	a.statusBar = components.NewStatusBar(t, s)
	a.commandPalette = components.NewCommandPalette(t, s)
	a.help = components.NewHelp(t, s)
	a.toast = components.NewToast(t, s)
	a.modal = components.NewModal(t, s)
	a.jump = components.NewJumpOverlay(t, s)

	// Re-set state
	if a.store.Collection != nil {
		items := collection.FlattenItems(a.store.Collection.Items, 0, "")
		a.sidebar.SetItems(items)
	}
	a.loadHistory()
	if a.store.ActiveEnv != "" {
		a.statusBar.SetEnv(a.store.ActiveEnv)
	}
	a.statusBar.SetMode(a.mode)
	a.syncTabs()
	a.resizePanels()

	cmd := a.toast.Show("Theme: "+t.Name, false, 2*time.Second)
	return a, cmd
}

func (a App) handleImportFile(msg msgs.ImportFileMsg) (tea.Model, tea.Cmd) {
	// For file-based import, we'd need a file picker. For now, use clipboard content.
	text, err := clipboard.ReadAll()
	if err != nil {
		cmd := a.toast.Show("Clipboard error: "+err.Error(), true, 3*time.Second)
		return a, cmd
	}
	text = strings.TrimSpace(text)
	if text == "" {
		cmd := a.toast.Show("Clipboard is empty. Copy file content first.", true, 2*time.Second)
		return a, cmd
	}

	data := []byte(text)
	cmd := func() tea.Msg {
		format := msg.Path // hint from command
		if format == "" {
			format = importutil.DetectFormat(data)
		}

		var col *collection.Collection
		var parseErr error

		switch format {
		case "postman":
			col, parseErr = postman.ParsePostman(data)
		case "insomnia":
			col, parseErr = insomnia.ParseInsomnia(data)
		case "openapi":
			col, parseErr = openapi.ParseOpenAPI(data)
		default:
			// Try auto-detection
			detected := importutil.DetectFormat(data)
			switch detected {
			case "postman":
				col, parseErr = postman.ParsePostman(data)
			case "insomnia":
				col, parseErr = insomnia.ParseInsomnia(data)
			case "openapi":
				col, parseErr = openapi.ParseOpenAPI(data)
			default:
				return msgs.ImportCompleteMsg{Err: os.ErrInvalid}
			}
		}

		return msgs.ImportCompleteMsg{Collection: col, Err: parseErr}
	}

	return a, cmd
}

func (a App) handleImportComplete(msg msgs.ImportCompleteMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		cmd := a.toast.Show("Import failed: "+msg.Err.Error(), true, 3*time.Second)
		return a, cmd
	}

	if msg.Collection == nil {
		cmd := a.toast.Show("No data imported", true, 2*time.Second)
		return a, cmd
	}

	// Merge into current collection or set as new
	if a.store.Collection == nil {
		a.store.Collection = msg.Collection
	} else {
		a.store.Collection.Items = append(a.store.Collection.Items, msg.Collection.Items...)
	}

	items := collection.FlattenItems(a.store.Collection.Items, 0, "")
	a.sidebar.SetItems(items)

	cmd := a.toast.Show("Imported "+msg.Collection.Name, false, 2*time.Second)
	return a, cmd
}

func (a App) handleSetBaseline() (tea.Model, tea.Cmd) {
	body := a.response.ResponseBody()
	if len(body) == 0 {
		cmd := a.toast.Show("No response to use as baseline", true, 2*time.Second)
		return a, cmd
	}
	a.response.SetBaseline(body)
	cmd := a.toast.Show("Baseline set", false, 2*time.Second)
	return a, cmd
}

func (a App) openExternalEditor() (tea.Model, tea.Cmd) {
	editorCmd := a.cfg.Editor
	if editorCmd == "" {
		editorCmd = os.Getenv("EDITOR")
	}
	if editorCmd == "" {
		editorCmd = "vi"
	}

	// Write body to temp file
	tmpFile, err := os.CreateTemp("", "gottp-body-*.txt")
	if err != nil {
		cmd := a.toast.Show("Failed to create temp file: "+err.Error(), true, 3*time.Second)
		return a, cmd
	}

	bodyContent := a.editor.GetBodyContent()
	if bodyContent != "" {
		_, _ = tmpFile.WriteString(bodyContent)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()

	c := exec.Command(editorCmd, tmpPath)
	return a, tea.ExecProcess(c, func(err error) tea.Msg {
		defer os.Remove(tmpPath)
		if err != nil {
			return msgs.EditorDoneMsg{}
		}
		data, err := os.ReadFile(tmpPath)
		if err != nil {
			return msgs.EditorDoneMsg{}
		}
		return msgs.EditorDoneMsg{Content: string(data)}
	})
}
