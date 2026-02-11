package app

import (
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/serdar/gottp/internal/config"
	"github.com/serdar/gottp/internal/core/collection"
	"github.com/serdar/gottp/internal/core/cookies"
	"github.com/serdar/gottp/internal/core/environment"
	"github.com/serdar/gottp/internal/core/history"
	"github.com/serdar/gottp/internal/core/state"
	gotls "github.com/serdar/gottp/internal/core/tls"
	"github.com/serdar/gottp/internal/protocol"
	"github.com/serdar/gottp/internal/protocol/graphql"
	grpcclient "github.com/serdar/gottp/internal/protocol/grpc"
	httpclient "github.com/serdar/gottp/internal/protocol/http"
	wsclient "github.com/serdar/gottp/internal/protocol/websocket"
	"github.com/serdar/gottp/internal/scripting"
	"github.com/serdar/gottp/internal/ui/components"
	"github.com/serdar/gottp/internal/ui/layout"
	"github.com/serdar/gottp/internal/ui/msgs"
	"github.com/serdar/gottp/internal/ui/panels/editor"
	"github.com/serdar/gottp/internal/ui/panels/response"
	"github.com/serdar/gottp/internal/ui/panels/sidebar"
	"github.com/serdar/gottp/internal/ui/theme"
)

// App is the root Bubble Tea model.
type App struct {
	sidebar  sidebar.Model
	editor   editor.Model
	response response.Model

	tabBar         components.TabBar
	statusBar      components.StatusBar
	commandPalette components.CommandPalette
	help           components.Help
	toast          components.Toast
	modal          components.Modal
	jump           components.JumpOverlay

	store        *state.Store
	protocols    *protocol.Registry
	scriptEngine *scripting.Engine
	envFile      *environment.EnvironmentFile
	cfg          config.Config
	history      *history.Store

	mode           msgs.AppMode
	focus          msgs.PanelFocus
	sidebarVisible bool
	layout         layout.PanelLayout
	keys           KeyMap

	theme  theme.Theme
	styles theme.Styles

	width  int
	height int
	ready  bool
}

// New creates a new App model.
func New(col *collection.Collection, colPath string, cfg config.Config) App {
	t := theme.Resolve(cfg.Theme)
	s := theme.NewStyles(t)

	store := state.NewStore()
	store.Collection = col
	store.CollectionPath = colPath
	store.NewTab()

	// Set up protocol registry
	registry := protocol.NewRegistry()
	httpClient := httpclient.New()
	if cfg.DefaultTimeout > 0 {
		httpClient.SetTimeout(cfg.DefaultTimeout)
	}
	if cfg.ProxyURL != "" {
		httpClient.SetProxy(cfg.ProxyURL, cfg.NoProxy)
	}
	if !cfg.TLS.IsEmpty() {
		tlsCfg, err := (&gotls.Config{
			CertFile:           cfg.TLS.CertFile,
			KeyFile:            cfg.TLS.KeyFile,
			CAFile:             cfg.TLS.CAFile,
			InsecureSkipVerify: cfg.TLS.InsecureSkipVerify,
		}).BuildTLSConfig()
		if err == nil && tlsCfg != nil {
			httpClient.SetTLSConfig(tlsCfg)
		}
	}
	cookieJar := cookies.New()
	httpClient.SetCookieJar(cookieJar)
	registry.Register(httpClient)
	registry.Register(graphql.New())
	registry.Register(wsclient.New())
	registry.Register(grpcclient.New())

	// Init scripting engine
	scriptTimeout := cfg.ScriptTimeout
	if scriptTimeout == 0 {
		scriptTimeout = 5 * time.Second
	}
	scriptEngine := scripting.NewEngine(scriptTimeout)

	// Load environments from environments.yaml next to the collection file
	var envFile *environment.EnvironmentFile
	if colPath != "" {
		dir := filepath.Dir(colPath)
		ef, err := environment.LoadEnvironments(filepath.Join(dir, "environments.yaml"))
		if err == nil && len(ef.Environments) > 0 {
			envFile = ef
			// Auto-select first environment
			store.ActiveEnv = ef.Environments[0].Name
			store.EnvVars = ef.GetVariables(store.ActiveEnv)
		}
	}

	// Init history store
	var histStore *history.Store
	dataDir := filepath.Join(os.Getenv("HOME"), ".local", "share", "gottp")
	os.MkdirAll(dataDir, 0755)
	if hs, err := history.NewStore(filepath.Join(dataDir, "history.db")); err == nil {
		histStore = hs
	}

	a := App{
		sidebar:  sidebar.New(t, s),
		editor:   editor.New(t, s),
		response: response.New(t, s),

		tabBar:         components.NewTabBar(t, s),
		statusBar:      components.NewStatusBar(t, s),
		commandPalette: components.NewCommandPalette(t, s),
		help:           components.NewHelp(t, s),
		toast:          components.NewToast(t, s),
		modal:          components.NewModal(t, s),
		jump:           components.NewJumpOverlay(t, s),

		store:        store,
		protocols:    registry,
		scriptEngine: scriptEngine,
		envFile:      envFile,
		cfg:          cfg,
		history:      histStore,

		mode:           msgs.ModeNormal,
		focus:          msgs.FocusEditor,
		sidebarVisible: true,
		keys:           DefaultKeyMap(),

		theme:  t,
		styles: s,
	}

	if col != nil {
		items := collection.FlattenItems(col.Items, 0, "")
		a.sidebar.SetItems(items)
	}

	if store.ActiveEnv != "" {
		a.statusBar.SetEnv(store.ActiveEnv)
	}

	// Load recent history into sidebar
	a.loadHistory()

	a.syncTabs()
	return a
}

func (a App) Init() tea.Cmd {
	return a.response.Init()
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.layout = layout.HandleResize(msg, a.sidebarVisible)
		a.resizePanels()
		a.ready = true
		return a, nil

	case tea.KeyMsg:
		if a.commandPalette.Visible {
			var cmd tea.Cmd
			a.commandPalette, cmd = a.commandPalette.Update(msg)
			return a, cmd
		}
		if a.help.Visible {
			var cmd tea.Cmd
			a.help, cmd = a.help.Update(msg)
			return a, cmd
		}
		if a.modal.Visible {
			var cmd tea.Cmd
			a.modal, cmd = a.modal.Update(msg)
			return a, cmd
		}
		if a.jump.Visible {
			var cmd tea.Cmd
			a.jump, cmd = a.jump.Update(msg)
			return a, cmd
		}

		if a.focus == msgs.FocusEditor && a.editor.Editing() {
			return a.updateEditorInsert(msg)
		}

		cmd := a.handleGlobalKey(msg)
		if cmd != nil {
			return a, cmd
		}
		return a.handlePanelKey(msg)

	case msgs.SendRequestMsg:
		return a.sendRequest()

	case msgs.RequestSentMsg:
		return a.handleRequestSent(msg)

	case msgs.NewRequestMsg:
		a.store.NewTab()
		a.syncTabs()
		a.loadActiveRequest()
		return a, nil

	case msgs.CloseTabMsg:
		a.store.CloseTab()
		a.syncTabs()
		a.loadActiveRequest()
		return a, nil

	case msgs.NextTabMsg:
		a.store.NextTab()
		a.syncTabs()
		a.loadActiveRequest()
		return a, nil

	case msgs.PrevTabMsg:
		a.store.PrevTab()
		a.syncTabs()
		a.loadActiveRequest()
		return a, nil

	case msgs.SwitchTabMsg:
		if msg.Index >= 0 && msg.Index < len(a.store.Tabs) {
			a.store.ActiveTab = msg.Index
			a.syncTabs()
			a.loadActiveRequest()
		}
		return a, nil

	case msgs.SwitchEnvMsg:
		if msg.Name != "" && a.envFile != nil {
			a.store.ActiveEnv = msg.Name
			a.store.EnvVars = a.envFile.GetVariables(msg.Name)
			a.statusBar.SetEnv(msg.Name)
			cmd := a.toast.Show("Environment: "+msg.Name, false, 2*time.Second)
			return a, cmd
		}
		// If Name is empty, open env picker via command palette
		if a.envFile != nil && len(a.envFile.Environments) > 0 {
			names := a.envFile.Names()
			a.commandPalette.OpenEnvPicker(names)
			a.mode = msgs.ModeCommandPalette
			return a, nil
		}
		cmd := a.toast.Show("No environments found", true, 2*time.Second)
		return a, cmd

	case msgs.SwitchThemeMsg:
		return a.handleSwitchTheme(msg)

	case msgs.ToggleSidebarMsg:
		a.sidebarVisible = !a.sidebarVisible
		a.layout = layout.Calculate(a.width, a.height, a.sidebarVisible)
		a.resizePanels()
		return a, nil

	case msgs.OpenCommandPaletteMsg:
		a.mode = msgs.ModeCommandPalette
		a.commandPalette.Open()
		return a, nil

	case msgs.ShowHelpMsg:
		a.mode = msgs.ModeModal
		a.help.Toggle()
		return a, nil

	case msgs.SetModeMsg:
		a.mode = msg.Mode
		a.statusBar.SetMode(msg.Mode)
		return a, nil

	case msgs.SaveRequestMsg:
		return a.saveCollection()

	case msgs.RequestSelectedMsg:
		return a.handleRequestSelected(msg)

	case msgs.StatusMsg:
		a.statusBar.SetMessage(msg.Text)
		if msg.Duration > 0 {
			cmds = append(cmds, tea.Tick(msg.Duration, func(time.Time) tea.Msg {
				return msgs.StatusMsg{Text: ""}
			}))
		}
		return a, tea.Batch(cmds...)

	case msgs.ToastMsg:
		cmd := a.toast.Show(msg.Text, msg.IsError, msg.Duration)
		return a, cmd

	case msgs.CopyAsCurlMsg:
		return a.copyAsCurl()

	case msgs.ImportCurlMsg:
		return a.importCurl()

	case msgs.ImportFileMsg:
		return a.handleImportFile(msg)

	case msgs.ImportCompleteMsg:
		return a.handleImportComplete(msg)

	case msgs.SetBaselineMsg:
		return a.handleSetBaseline()

	case msgs.ClearBaselineMsg:
		a.response.ClearBaseline()
		cmd := a.toast.Show("Baseline cleared", false, 2*time.Second)
		return a, cmd

	case msgs.OAuth2TokenMsg:
		return a.handleOAuth2Token(msg)

	case msgs.HistorySelectedMsg:
		return a.handleHistorySelected(msg)

	case msgs.FocusPanelMsg:
		a.focus = msg.Panel
		a.updateFocus()
		return a, nil

	case msgs.OpenEditorMsg:
		return a.openExternalEditor()

	case msgs.EditorDoneMsg:
		if msg.Content != "" {
			a.editor.SetBody(msg.Content)
			cmd := a.toast.Show("Body updated from editor", false, 2*time.Second)
			return a, cmd
		}
		return a, nil

	case msgs.SwitchProtocolMsg:
		a.editor.SetProtocol(msg.Protocol)
		a.response.SetMode(msg.Protocol)
		return a, nil

	case msgs.IntrospectMsg:
		return a.handleIntrospect()

	case msgs.IntrospectionResultMsg:
		return a.handleIntrospectionResult(msg)

	case msgs.ScriptResultMsg:
		return a.handleScriptResult(msg)

	case msgs.WSConnectedMsg:
		if msg.Err != nil {
			cmd := a.toast.Show("WebSocket error: "+msg.Err.Error(), true, 5*time.Second)
			return a, cmd
		}
		cmd := a.toast.Show("WebSocket connected", false, 2*time.Second)
		return a, cmd

	case msgs.WSDisconnectedMsg:
		if msg.Err != nil {
			cmd := a.toast.Show("WebSocket closed: "+msg.Err.Error(), true, 3*time.Second)
			return a, cmd
		}
		cmd := a.toast.Show("WebSocket disconnected", false, 2*time.Second)
		return a, cmd

	case msgs.WSMessageReceivedMsg:
		a.response.AddWSMessage(response.WSMessage{
			Direction: "received",
			Content:   msg.Content,
			Timestamp: msg.Timestamp,
			IsJSON:    msg.IsJSON,
		})
		return a, nil

	case msgs.GenerateCodeMsg:
		return a.handleGenerateCode(msg)

	case msgs.GRPCReflectMsg:
		return a.handleGRPCReflect()

	case msgs.GRPCReflectionResultMsg:
		return a.handleGRPCReflectionResult(msg)

	case msgs.InsertTemplateMsg:
		return a.handleInsertTemplate(msg)
	}

	var cmd tea.Cmd
	a.toast, cmd = a.toast.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	a.statusBar, cmd = a.statusBar.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	a.response, cmd = a.response.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return a, tea.Batch(cmds...)
}

func (a App) View() string {
	if !a.ready {
		return "Loading..."
	}

	tabBar := a.tabBar.View()

	var panels string
	if a.layout.SinglePanel {
		switch a.focus {
		case msgs.FocusSidebar:
			panels = a.sidebar.View()
		case msgs.FocusEditor:
			panels = a.editor.View()
		case msgs.FocusResponse:
			panels = a.response.View()
		}
	} else {
		var panelViews []string
		if a.sidebarVisible && !a.layout.TwoPanelMode {
			panelViews = append(panelViews, a.sidebar.View())
		}
		panelViews = append(panelViews, a.editor.View())
		panelViews = append(panelViews, a.response.View())
		panels = lipgloss.JoinHorizontal(lipgloss.Top, panelViews...)
	}

	statusBar := a.statusBar.View()
	main := lipgloss.JoinVertical(lipgloss.Left, tabBar, panels, statusBar)

	if a.commandPalette.Visible {
		main = overlayCenter(main, a.commandPalette.View(), a.width, a.height)
	}
	if a.help.Visible {
		main = overlayCenter(main, a.help.View(), a.width, a.height)
	}
	if a.modal.Visible {
		main = overlayCenter(main, a.modal.View(), a.width, a.height)
	}
	if a.jump.Visible {
		main = overlayCenter(main, a.jump.View(), a.width, a.height)
	}
	if a.toast.Visible {
		toastView := a.toast.View()
		main = overlayTopRight(main, toastView, a.width)
	}

	return main
}

func overlayCenter(_, overlay string, width, height int) string {
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, overlay,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("#1e1e2e")),
	)
}

func overlayTopRight(bg, overlay string, width int) string {
	overlayWidth := lipgloss.Width(overlay)
	gap := width - overlayWidth - 2
	if gap < 0 {
		gap = 0
	}
	positioned := lipgloss.NewStyle().MarginLeft(gap).Render(overlay)
	return positioned + "\n" + bg
}

func (a *App) resizePanels() {
	l := a.layout
	a.sidebar.SetSize(l.SidebarWidth, l.ContentHeight)
	a.editor.SetSize(l.EditorWidth, l.ContentHeight)
	a.response.SetSize(l.ResponseWidth, l.ContentHeight)
	a.tabBar.SetWidth(a.width)
	a.statusBar.SetWidth(a.width)
	a.help.SetSize(a.width, a.height)
	a.updateFocus()
}

func findRequest(items []collection.Item, id string) *collection.Request {
	for i := range items {
		if items[i].Request != nil && items[i].Request.ID == id {
			return items[i].Request
		}
		if items[i].Folder != nil {
			if req := findRequest(items[i].Folder.Items, id); req != nil {
				return req
			}
		}
	}
	return nil
}
