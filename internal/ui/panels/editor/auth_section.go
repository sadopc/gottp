package editor

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/serdar/gottp/internal/core/collection"
	"github.com/serdar/gottp/internal/protocol"
	"github.com/serdar/gottp/internal/ui/theme"
)

var authTypes = []string{"none", "basic", "bearer", "apikey", "oauth2", "awsv4", "digest"}

// AuthSection manages auth configuration with type selector and field inputs.
type AuthSection struct {
	authType    string // none, basic, bearer, apikey, oauth2, awsv4
	typeIndex   int
	cursor      int // 0=type, 1+=fields
	editing     bool
	activeInput int // which input is active for editing

	// Basic auth
	username textinput.Model
	password textinput.Model

	// Bearer
	token textinput.Model

	// API Key
	apiKeyName  textinput.Model
	apiKeyValue textinput.Model
	apiKeyIn    string // header, query
	apiKeyInIdx int    // 0=header, 1=query

	// OAuth2
	oauth2GrantType    string
	oauth2GrantTypeIdx int
	oauth2AuthURL      textinput.Model
	oauth2TokenURL     textinput.Model
	oauth2ClientID     textinput.Model
	oauth2ClientSecret textinput.Model
	oauth2Scope        textinput.Model
	oauth2Username     textinput.Model
	oauth2Password     textinput.Model
	oauth2UsePKCE      bool

	// AWS Sig v4
	awsAccessKey    textinput.Model
	awsSecretKey    textinput.Model
	awsSessionToken textinput.Model
	awsRegion       textinput.Model
	awsService      textinput.Model

	// Digest
	digestUsername textinput.Model
	digestPassword textinput.Model

	width  int
	styles theme.Styles
}

var oauth2GrantTypes = []string{"client_credentials", "authorization_code", "password"}

// NewAuthSection creates a new auth section.
func NewAuthSection(styles theme.Styles) AuthSection {
	mkInput := func(placeholder string) textinput.Model {
		ti := textinput.New()
		ti.Placeholder = placeholder
		ti.CharLimit = 512
		ti.Width = 40
		return ti
	}

	return AuthSection{
		authType:           "none",
		username:           mkInput("Username"),
		password:           mkInput("Password"),
		token:              mkInput("Bearer token"),
		apiKeyName:         mkInput("Key name (e.g. X-API-Key)"),
		apiKeyValue:        mkInput("Key value"),
		apiKeyIn:           "header",
		oauth2GrantType:    "client_credentials",
		oauth2AuthURL:      mkInput("Authorization URL"),
		oauth2TokenURL:     mkInput("Token URL"),
		oauth2ClientID:     mkInput("Client ID"),
		oauth2ClientSecret: mkInput("Client Secret"),
		oauth2Scope:        mkInput("Scope (space separated)"),
		oauth2Username:     mkInput("Username"),
		oauth2Password:     mkInput("Password"),
		awsAccessKey:       mkInput("Access Key ID"),
		awsSecretKey:       mkInput("Secret Access Key"),
		awsSessionToken:    mkInput("Session Token (optional)"),
		awsRegion:          mkInput("Region (e.g. us-east-1)"),
		awsService:         mkInput("Service (e.g. execute-api)"),
		digestUsername:     mkInput("Username"),
		digestPassword:     mkInput("Password"),
		styles:             styles,
	}
}

// SetSize updates the section width.
func (m *AuthSection) SetSize(w int) {
	m.width = w
	inputW := w - 16
	if inputW < 10 {
		inputW = 10
	}
	m.username.Width = inputW
	m.password.Width = inputW
	m.token.Width = inputW
	m.apiKeyName.Width = inputW
	m.apiKeyValue.Width = inputW
	m.oauth2AuthURL.Width = inputW
	m.oauth2TokenURL.Width = inputW
	m.oauth2ClientID.Width = inputW
	m.oauth2ClientSecret.Width = inputW
	m.oauth2Scope.Width = inputW
	m.oauth2Username.Width = inputW
	m.oauth2Password.Width = inputW
	m.awsAccessKey.Width = inputW
	m.awsSecretKey.Width = inputW
	m.awsSessionToken.Width = inputW
	m.awsRegion.Width = inputW
	m.awsService.Width = inputW
	m.digestUsername.Width = inputW
	m.digestPassword.Width = inputW
}

// Editing returns whether any field is being edited.
func (m AuthSection) Editing() bool {
	return m.editing
}

// BuildAuth returns a protocol.AuthConfig from the current state.
func (m AuthSection) BuildAuth() *protocol.AuthConfig {
	switch m.authType {
	case "basic":
		return &protocol.AuthConfig{
			Type:     "basic",
			Username: m.username.Value(),
			Password: m.password.Value(),
		}
	case "bearer":
		return &protocol.AuthConfig{
			Type:  "bearer",
			Token: m.token.Value(),
		}
	case "apikey":
		return &protocol.AuthConfig{
			Type:     "apikey",
			APIKey:   m.apiKeyName.Value(),
			APIValue: m.apiKeyValue.Value(),
			APIIn:    m.apiKeyIn,
		}
	case "oauth2":
		return &protocol.AuthConfig{
			Type: "oauth2",
			OAuth2: &protocol.OAuth2AuthConfig{
				GrantType:    m.oauth2GrantType,
				AuthURL:      m.oauth2AuthURL.Value(),
				TokenURL:     m.oauth2TokenURL.Value(),
				ClientID:     m.oauth2ClientID.Value(),
				ClientSecret: m.oauth2ClientSecret.Value(),
				Scope:        m.oauth2Scope.Value(),
				Username:     m.oauth2Username.Value(),
				Password:     m.oauth2Password.Value(),
				UsePKCE:      m.oauth2UsePKCE,
			},
		}
	case "awsv4":
		return &protocol.AuthConfig{
			Type: "awsv4",
			AWSAuth: &protocol.AWSAuthConfig{
				AccessKeyID:    m.awsAccessKey.Value(),
				SecretAccessKey: m.awsSecretKey.Value(),
				SessionToken:   m.awsSessionToken.Value(),
				Region:         m.awsRegion.Value(),
				Service:        m.awsService.Value(),
			},
		}
	case "digest":
		return &protocol.AuthConfig{
			Type:           "digest",
			DigestUsername: m.digestUsername.Value(),
			DigestPassword: m.digestPassword.Value(),
		}
	default:
		return nil
	}
}

// LoadAuth loads auth configuration from a collection auth.
func (m *AuthSection) LoadAuth(auth *collection.Auth) {
	if auth == nil {
		m.authType = "none"
		m.typeIndex = 0
		return
	}
	m.authType = auth.Type
	for i, t := range authTypes {
		if t == auth.Type {
			m.typeIndex = i
			break
		}
	}
	switch auth.Type {
	case "basic":
		if auth.Basic != nil {
			m.username.SetValue(auth.Basic.Username)
			m.password.SetValue(auth.Basic.Password)
		}
	case "bearer":
		if auth.Bearer != nil {
			m.token.SetValue(auth.Bearer.Token)
		}
	case "apikey":
		if auth.APIKey != nil {
			m.apiKeyName.SetValue(auth.APIKey.Key)
			m.apiKeyValue.SetValue(auth.APIKey.Value)
			m.apiKeyIn = auth.APIKey.In
			if m.apiKeyIn == "query" {
				m.apiKeyInIdx = 1
			} else {
				m.apiKeyInIdx = 0
				m.apiKeyIn = "header"
			}
		}
	case "oauth2":
		if auth.OAuth2 != nil {
			m.oauth2GrantType = auth.OAuth2.GrantType
			for i, gt := range oauth2GrantTypes {
				if gt == auth.OAuth2.GrantType {
					m.oauth2GrantTypeIdx = i
					break
				}
			}
			m.oauth2AuthURL.SetValue(auth.OAuth2.AuthURL)
			m.oauth2TokenURL.SetValue(auth.OAuth2.TokenURL)
			m.oauth2ClientID.SetValue(auth.OAuth2.ClientID)
			m.oauth2ClientSecret.SetValue(auth.OAuth2.ClientSecret)
			m.oauth2Scope.SetValue(auth.OAuth2.Scope)
			m.oauth2Username.SetValue(auth.OAuth2.Username)
			m.oauth2Password.SetValue(auth.OAuth2.Password)
			m.oauth2UsePKCE = auth.OAuth2.UsePKCE
		}
	case "awsv4":
		if auth.AWSAuth != nil {
			m.awsAccessKey.SetValue(auth.AWSAuth.AccessKeyID)
			m.awsSecretKey.SetValue(auth.AWSAuth.SecretAccessKey)
			m.awsSessionToken.SetValue(auth.AWSAuth.SessionToken)
			m.awsRegion.SetValue(auth.AWSAuth.Region)
			m.awsService.SetValue(auth.AWSAuth.Service)
		}
	case "digest":
		if auth.Digest != nil {
			m.digestUsername.SetValue(auth.Digest.Username)
			m.digestPassword.SetValue(auth.Digest.Password)
		}
	}
}

// Update handles input messages.
func (m AuthSection) Update(msg tea.Msg) (AuthSection, tea.Cmd) {
	if m.editing {
		return m.updateEditing(msg)
	}
	return m.updateNormal(msg)
}

func (m AuthSection) updateNormal(msg tea.Msg) (AuthSection, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			maxCursor := m.maxCursor()
			if m.cursor < maxCursor {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "enter", " ":
			if m.cursor == 0 {
				// Cycle auth type
				m.typeIndex = (m.typeIndex + 1) % len(authTypes)
				m.authType = authTypes[m.typeIndex]
				m.cursor = 0
			} else if m.isToggleField() {
				m.handleToggle()
			} else {
				// Start editing the focused field
				m.startEditing()
				return m, textinput.Blink
			}
		case "h", "left":
			if m.cursor == 0 {
				m.typeIndex = (m.typeIndex - 1 + len(authTypes)) % len(authTypes)
				m.authType = authTypes[m.typeIndex]
			}
			m.handleLeftRight()
		case "l", "right":
			if m.cursor == 0 {
				m.typeIndex = (m.typeIndex + 1) % len(authTypes)
				m.authType = authTypes[m.typeIndex]
			}
			m.handleLeftRight()
		}
	}
	return m, nil
}

func (m AuthSection) updateEditing(msg tea.Msg) (AuthSection, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "enter":
			m.blurAll()
			m.editing = false
			return m, nil
		}
	}

	var cmd tea.Cmd
	switch m.authType {
	case "basic":
		if m.cursor == 1 {
			m.username, cmd = m.username.Update(msg)
		} else if m.cursor == 2 {
			m.password, cmd = m.password.Update(msg)
		}
	case "bearer":
		if m.cursor == 1 {
			m.token, cmd = m.token.Update(msg)
		}
	case "apikey":
		if m.cursor == 1 {
			m.apiKeyName, cmd = m.apiKeyName.Update(msg)
		} else if m.cursor == 2 {
			m.apiKeyValue, cmd = m.apiKeyValue.Update(msg)
		}
	case "oauth2":
		cmd = m.updateOAuth2Editing(msg)
	case "awsv4":
		cmd = m.updateAWSEditing(msg)
	case "digest":
		cmd = m.updateDigestEditing(msg)
	}
	return m, cmd
}

func (m *AuthSection) updateOAuth2Editing(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	switch m.cursor {
	case 2:
		m.oauth2AuthURL, cmd = m.oauth2AuthURL.Update(msg)
	case 3:
		m.oauth2TokenURL, cmd = m.oauth2TokenURL.Update(msg)
	case 4:
		m.oauth2ClientID, cmd = m.oauth2ClientID.Update(msg)
	case 5:
		m.oauth2ClientSecret, cmd = m.oauth2ClientSecret.Update(msg)
	case 6:
		m.oauth2Scope, cmd = m.oauth2Scope.Update(msg)
	case 7:
		m.oauth2Username, cmd = m.oauth2Username.Update(msg)
	case 8:
		m.oauth2Password, cmd = m.oauth2Password.Update(msg)
	}
	return cmd
}

func (m *AuthSection) updateAWSEditing(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	switch m.cursor {
	case 1:
		m.awsAccessKey, cmd = m.awsAccessKey.Update(msg)
	case 2:
		m.awsSecretKey, cmd = m.awsSecretKey.Update(msg)
	case 3:
		m.awsSessionToken, cmd = m.awsSessionToken.Update(msg)
	case 4:
		m.awsRegion, cmd = m.awsRegion.Update(msg)
	case 5:
		m.awsService, cmd = m.awsService.Update(msg)
	}
	return cmd
}

func (m *AuthSection) updateDigestEditing(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	switch m.cursor {
	case 1:
		m.digestUsername, cmd = m.digestUsername.Update(msg)
	case 2:
		m.digestPassword, cmd = m.digestPassword.Update(msg)
	}
	return cmd
}

func (m *AuthSection) startEditing() {
	m.editing = true
	switch m.authType {
	case "basic":
		if m.cursor == 1 {
			m.username.Focus()
			m.username.CursorEnd()
		} else if m.cursor == 2 {
			m.password.Focus()
			m.password.CursorEnd()
		}
	case "bearer":
		if m.cursor == 1 {
			m.token.Focus()
			m.token.CursorEnd()
		}
	case "apikey":
		if m.cursor == 1 {
			m.apiKeyName.Focus()
			m.apiKeyName.CursorEnd()
		} else if m.cursor == 2 {
			m.apiKeyValue.Focus()
			m.apiKeyValue.CursorEnd()
		}
	case "oauth2":
		m.startOAuth2Editing()
	case "awsv4":
		m.startAWSEditing()
	case "digest":
		m.startDigestEditing()
	}
}

func (m *AuthSection) startOAuth2Editing() {
	switch m.cursor {
	case 2:
		m.oauth2AuthURL.Focus()
		m.oauth2AuthURL.CursorEnd()
	case 3:
		m.oauth2TokenURL.Focus()
		m.oauth2TokenURL.CursorEnd()
	case 4:
		m.oauth2ClientID.Focus()
		m.oauth2ClientID.CursorEnd()
	case 5:
		m.oauth2ClientSecret.Focus()
		m.oauth2ClientSecret.CursorEnd()
	case 6:
		m.oauth2Scope.Focus()
		m.oauth2Scope.CursorEnd()
	case 7:
		m.oauth2Username.Focus()
		m.oauth2Username.CursorEnd()
	case 8:
		m.oauth2Password.Focus()
		m.oauth2Password.CursorEnd()
	}
}

func (m *AuthSection) startAWSEditing() {
	switch m.cursor {
	case 1:
		m.awsAccessKey.Focus()
		m.awsAccessKey.CursorEnd()
	case 2:
		m.awsSecretKey.Focus()
		m.awsSecretKey.CursorEnd()
	case 3:
		m.awsSessionToken.Focus()
		m.awsSessionToken.CursorEnd()
	case 4:
		m.awsRegion.Focus()
		m.awsRegion.CursorEnd()
	case 5:
		m.awsService.Focus()
		m.awsService.CursorEnd()
	}
}

func (m *AuthSection) startDigestEditing() {
	switch m.cursor {
	case 1:
		m.digestUsername.Focus()
		m.digestUsername.CursorEnd()
	case 2:
		m.digestPassword.Focus()
		m.digestPassword.CursorEnd()
	}
}

func (m *AuthSection) blurAll() {
	m.username.Blur()
	m.password.Blur()
	m.token.Blur()
	m.apiKeyName.Blur()
	m.apiKeyValue.Blur()
	m.oauth2AuthURL.Blur()
	m.oauth2TokenURL.Blur()
	m.oauth2ClientID.Blur()
	m.oauth2ClientSecret.Blur()
	m.oauth2Scope.Blur()
	m.oauth2Username.Blur()
	m.oauth2Password.Blur()
	m.awsAccessKey.Blur()
	m.awsSecretKey.Blur()
	m.awsSessionToken.Blur()
	m.awsRegion.Blur()
	m.awsService.Blur()
	m.digestUsername.Blur()
	m.digestPassword.Blur()
}

func (m AuthSection) isToggleField() bool {
	switch m.authType {
	case "apikey":
		return m.cursor == 3 // Send In selector
	case "oauth2":
		return m.cursor == 1 || m.cursor == 9 // grant type selector, PKCE toggle
	}
	return false
}

func (m *AuthSection) handleToggle() {
	switch m.authType {
	case "apikey":
		if m.cursor == 3 {
			m.apiKeyInIdx = (m.apiKeyInIdx + 1) % 2
			if m.apiKeyInIdx == 0 {
				m.apiKeyIn = "header"
			} else {
				m.apiKeyIn = "query"
			}
		}
	case "oauth2":
		if m.cursor == 1 {
			m.oauth2GrantTypeIdx = (m.oauth2GrantTypeIdx + 1) % len(oauth2GrantTypes)
			m.oauth2GrantType = oauth2GrantTypes[m.oauth2GrantTypeIdx]
		} else if m.cursor == 9 {
			m.oauth2UsePKCE = !m.oauth2UsePKCE
		}
	}
}

func (m *AuthSection) handleLeftRight() {
	switch m.authType {
	case "apikey":
		if m.cursor == 3 {
			m.apiKeyInIdx = (m.apiKeyInIdx + 1) % 2
			if m.apiKeyInIdx == 0 {
				m.apiKeyIn = "header"
			} else {
				m.apiKeyIn = "query"
			}
		}
	case "oauth2":
		if m.cursor == 1 {
			m.oauth2GrantTypeIdx = (m.oauth2GrantTypeIdx + 1) % len(oauth2GrantTypes)
			m.oauth2GrantType = oauth2GrantTypes[m.oauth2GrantTypeIdx]
		}
	}
}

func (m AuthSection) maxCursor() int {
	switch m.authType {
	case "basic":
		return 2 // type, username, password
	case "bearer":
		return 1 // type, token
	case "apikey":
		return 3 // type, key, value, in
	case "oauth2":
		return 9 // type, grant_type, auth_url, token_url, client_id, client_secret, scope, username, password, pkce
	case "awsv4":
		return 5 // type, access_key, secret_key, session_token, region, service
	case "digest":
		return 2 // type, username, password
	default:
		return 0 // none: just type
	}
}

// View renders the auth section.
func (m AuthSection) View() string {
	var lines []string

	// Type selector row
	typeLabel := "  Type: "
	if m.cursor == 0 {
		typeLabel = "> Type: "
	}

	var typeParts []string
	for i, t := range authTypes {
		if i == m.typeIndex {
			typeParts = append(typeParts, m.styles.TabActive.Render(t))
		} else {
			typeParts = append(typeParts, m.styles.TabInactive.Render(t))
		}
	}
	lines = append(lines, typeLabel+strings.Join(typeParts, " "))

	switch m.authType {
	case "none":
		lines = append(lines, "")
		lines = append(lines, m.styles.Muted.Render("  No authentication"))

	case "basic":
		lines = append(lines, "")
		lines = append(lines, m.renderField("Username", m.username, 1))
		lines = append(lines, m.renderField("Password", m.password, 2))

	case "bearer":
		lines = append(lines, "")
		lines = append(lines, m.renderField("Token", m.token, 1))

	case "apikey":
		lines = append(lines, "")
		lines = append(lines, m.renderField("Key", m.apiKeyName, 1))
		lines = append(lines, m.renderField("Value", m.apiKeyValue, 2))
		// In selector
		prefix := "  "
		if m.cursor == 3 {
			prefix = "> "
		}
		inLabel := prefix + m.styles.Key.Render(lipgloss.NewStyle().Width(10).Render("Send In")) + " "
		var inParts []string
		inOptions := []string{"header", "query"}
		for i, opt := range inOptions {
			if i == m.apiKeyInIdx {
				inParts = append(inParts, m.styles.TabActive.Render(opt))
			} else {
				inParts = append(inParts, m.styles.TabInactive.Render(opt))
			}
		}
		lines = append(lines, inLabel+strings.Join(inParts, " "))

	case "oauth2":
		lines = append(lines, "")
		// Grant type selector
		prefix := "  "
		if m.cursor == 1 {
			prefix = "> "
		}
		gtLabel := prefix + m.styles.Key.Render(lipgloss.NewStyle().Width(10).Render("Grant")) + " "
		var gtParts []string
		for i, gt := range oauth2GrantTypes {
			if i == m.oauth2GrantTypeIdx {
				gtParts = append(gtParts, m.styles.TabActive.Render(gt))
			} else {
				gtParts = append(gtParts, m.styles.TabInactive.Render(gt))
			}
		}
		lines = append(lines, gtLabel+strings.Join(gtParts, " "))
		lines = append(lines, m.renderField("Auth URL", m.oauth2AuthURL, 2))
		lines = append(lines, m.renderField("Token URL", m.oauth2TokenURL, 3))
		lines = append(lines, m.renderField("Client ID", m.oauth2ClientID, 4))
		lines = append(lines, m.renderField("Secret", m.oauth2ClientSecret, 5))
		lines = append(lines, m.renderField("Scope", m.oauth2Scope, 6))
		lines = append(lines, m.renderField("Username", m.oauth2Username, 7))
		lines = append(lines, m.renderField("Password", m.oauth2Password, 8))
		// PKCE toggle
		pkcePrefix := "  "
		if m.cursor == 9 {
			pkcePrefix = "> "
		}
		pkceVal := "off"
		if m.oauth2UsePKCE {
			pkceVal = "on"
		}
		pkceLabel := pkcePrefix + m.styles.Key.Render(lipgloss.NewStyle().Width(10).Render("PKCE")) + " "
		lines = append(lines, pkceLabel+m.styles.TabActive.Render(pkceVal))

	case "awsv4":
		lines = append(lines, "")
		lines = append(lines, m.renderField("Access Key", m.awsAccessKey, 1))
		lines = append(lines, m.renderField("Secret Key", m.awsSecretKey, 2))
		lines = append(lines, m.renderField("Session", m.awsSessionToken, 3))
		lines = append(lines, m.renderField("Region", m.awsRegion, 4))
		lines = append(lines, m.renderField("Service", m.awsService, 5))

	case "digest":
		lines = append(lines, "")
		lines = append(lines, m.renderField("Username", m.digestUsername, 1))
		lines = append(lines, m.renderField("Password", m.digestPassword, 2))
	}

	return strings.Join(lines, "\n")
}

func (m AuthSection) renderField(label string, input textinput.Model, fieldIdx int) string {
	prefix := "  "
	if m.cursor == fieldIdx {
		prefix = "> "
	}

	labelStr := m.styles.Key.Render(lipgloss.NewStyle().Width(10).Render(label))

	if m.cursor == fieldIdx && m.editing {
		return prefix + labelStr + " " + input.View()
	}

	val := input.Value()
	if val == "" {
		val = input.Placeholder
		return prefix + labelStr + " " + m.styles.Muted.Render(val)
	}
	return prefix + labelStr + " " + m.styles.Normal.Render(val)
}
