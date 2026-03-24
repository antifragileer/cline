// Package tui provides authentication wizard components for the Cline CLI.
// This implements a multi-step authentication wizard with support for
// provider selection, authentication methods, API key input, model selection,
// validation, and configuration saving.
package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cline/cline/golang-cli/internal/auth"
	"github.com/cline/cline/golang-cli/internal/storage"
)

// AuthStep represents the current step in the authentication wizard.
type AuthStep int

const (
	// AuthStepMenu shows the main authentication menu.
	AuthStepMenu AuthStep = iota
	// AuthStepProvider shows provider selection.
	AuthStepProvider
	// AuthStepAuthMethod shows authentication method selection.
	AuthStepAuthMethod
	// AuthStepAPIKey shows API key input.
	AuthStepAPIKey
	// AuthStepModel shows model selection.
	AuthStepModel
	// AuthStepBaseURL shows optional base URL input.
	AuthStepBaseURL
	// AuthStepValidating shows validation in progress.
	AuthStepValidating
	// AuthStepSuccess shows success confirmation.
	AuthStepSuccess
	// AuthStepError shows error state.
	AuthStepError
	// AuthStepStatus shows current authentication status.
	AuthStepStatus
)

// AuthWizardConfig configures the authentication wizard.
type AuthWizardConfig struct {
	// StorageContext provides access to persistent storage.
	StorageContext *storage.StorageContext

	// APIKeyManager handles API key operations.
	APIKeyManager *auth.APIKeyManager

	// Wizard handles the backend wizard logic.
	Wizard *auth.Wizard

	// Flags for non-interactive mode.
	Flags auth.WizardFlags

	// ForceReauth forces re-authentication even if already authenticated.
	ForceReauth bool

	// Title is the wizard title.
	Title string
}

// DefaultAuthWizardConfig returns a default configuration.
func DefaultAuthWizardConfig() AuthWizardConfig {
	return AuthWizardConfig{
		Title: "Cline Authentication",
	}
}

// AuthWizardKeyMap defines key bindings for the auth wizard.
type AuthWizardKeyMap struct {
	Up      key.Binding
	Down    key.Binding
	Select  key.Binding
	Back    key.Binding
	Quit    key.Binding
	Submit  key.Binding
	Confirm key.Binding
}

// DefaultAuthWizardKeyMap returns the default key bindings.
func DefaultAuthWizardKeyMap() AuthWizardKeyMap {
	return AuthWizardKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc", "left"),
			key.WithHelp("esc", "back"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c", "q"),
			key.WithHelp("ctrl+c/q", "quit"),
		),
		Submit: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "submit"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("y", "enter"),
			key.WithHelp("y/enter", "confirm"),
		),
	}
}

// AuthResult contains the outcome of the authentication wizard.
type AuthResult struct {
	Provider   string
	AuthMethod auth.AuthMethod
	APIKey     string
	Model      string
	BaseURL    string
	Validated  bool
	Saved      bool
	Step       AuthStep
	Error      error
}

// AuthStatusInfo represents the current authentication status.
type AuthStatusInfo struct {
	Provider        string
	DisplayName     string
	IsAuthenticated bool
	Model           string
	AuthMethod      string
	APIKeyMasked    string
}

// MenuItem represents a menu item.
type MenuItem struct {
	Label       string
	Value       string
	Description string
}

// AuthWizardModel is the Bubble Tea model for the authentication wizard.
type AuthWizardModel struct {
	config AuthWizardConfig
	keyMap AuthWizardKeyMap

	// State
	step          AuthStep
	previousSteps []AuthStep
	width         int
	height        int

	// Data
	providers      []auth.ProviderInfo
	selectedProvider auth.ProviderInfo
	authMethod     auth.AuthMethod
	apiKey         string
	model          string
	baseURL        string
	errorMessage   string
	authStatus     AuthStatusInfo

	// Menu navigation
	menuItems      []MenuItem
	menuIndex      int
	providerIndex  int
	authMethodIndex int
	modelIndex     int

	// Input components
	apiKeyInput    textinput.Model
	baseURLInput   textinput.Model
	spinner        spinner.Model

	// Validation
	validating     bool
	validationErr  error

	// Result
	result AuthResult
	quit   bool
}

// NewAuthWizardModel creates a new authentication wizard model.
func NewAuthWizardModel(config AuthWizardConfig) *AuthWizardModel {
	// Initialize API key input with masking
	apiKeyInput := textinput.New()
	apiKeyInput.Placeholder = "Enter your API key"
	apiKeyInput.EchoMode = textinput.EchoPassword
	apiKeyInput.Focus()

	// Initialize base URL input
	baseURLInput := textinput.New()
	baseURLInput.Placeholder = "https://api.example.com/v1 (optional)"
	baseURLInput.Focus()

	// Initialize spinner
	s := spinner.New()
	s.Spinner = spinner.Dot

	return &AuthWizardModel{
		config:         config,
		keyMap:         DefaultAuthWizardKeyMap(),
		step:           AuthStepMenu,
		previousSteps:  make([]AuthStep, 0),
		width:          80,
		height:         24,
		menuItems:      getMainMenuItems(),
		providers:      make([]auth.ProviderInfo, 0),
		apiKeyInput:    apiKeyInput,
		baseURLInput:   baseURLInput,
		spinner:        s,
	}
}

// getMainMenuItems returns the main menu items.
func getMainMenuItems() []MenuItem {
	return []MenuItem{
		{
			Label:       "Configure new provider",
			Value:       "configure",
			Description: "Set up a new API provider",
		},
		{
			Label:       "View authentication status",
			Value:       "status",
			Description: "Check current authentication status",
		},
		{
			Label:       "Re-authenticate",
			Value:       "reauth",
			Description: "Update existing authentication",
		},
		{
			Label:       "Exit",
			Value:       "exit",
			Description: "Exit without changes",
		},
	}
}

// Init implements tea.Model.
func (m *AuthWizardModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.loadAuthStatusCmd(),
	)
}

// loadAuthStatusCmd returns a command to load authentication status.
func (m *AuthWizardModel) loadAuthStatusCmd() tea.Cmd {
	return func() tea.Msg {
		return m.loadAuthStatus()
	}
}

// authStatusLoadedMsg is sent when auth status is loaded.
type authStatusLoadedMsg struct {
	status AuthStatusInfo
}

// loadAuthStatus loads the current authentication status.
func (m *AuthWizardModel) loadAuthStatus() tea.Msg {
	status := AuthStatusInfo{
		IsAuthenticated: false,
	}

	if m.config.StorageContext == nil {
		return authStatusLoadedMsg{status: status}
	}

	// Get provider
	if providerVal, ok := m.config.StorageContext.GlobalState.Get("apiProvider"); ok {
		if provider, ok := providerVal.(string); ok {
			status.Provider = provider
			// Get display name if wizard is available
			if m.config.Wizard != nil {
				if info, ok := m.config.Wizard.GetProviderInfo(provider); ok {
					status.DisplayName = info.DisplayName
				}
			}
			if status.DisplayName == "" {
				status.DisplayName = provider
			}
		}
	}

	// Get model
	if modelVal, ok := m.config.StorageContext.GlobalState.Get("defaultModel"); ok {
		if model, ok := modelVal.(string); ok {
			status.Model = model
		}
	}

	// Check if API key exists
	if status.Provider != "" && m.config.APIKeyManager != nil {
		if key, err := m.config.APIKeyManager.GetKey(status.Provider); err == nil && key != "" {
			status.IsAuthenticated = true
			status.APIKeyMasked = maskAPIKey(key)
		}
	}

	return authStatusLoadedMsg{status: status}
}

// maskAPIKey masks an API key for display.
func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "..." + key[len(key)-4:]
}

// loadProviders loads available providers from the wizard.
func (m *AuthWizardModel) loadProviders() {
	if m.config.Wizard == nil {
		return
	}

	providerNames := m.config.Wizard.ListProviders()
	for _, name := range providerNames {
		if info, ok := m.config.Wizard.GetProviderInfo(name); ok {
			m.providers = append(m.providers, info)
		}
	}
}

// Update implements tea.Model.
func (m *AuthWizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateInputWidths()

	case authStatusLoadedMsg:
		m.authStatus = msg.status

	case spinner.TickMsg:
		if m.validating {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case validationResultMsg:
		m.validating = false
		if msg.err != nil {
			m.validationErr = msg.err
			m.errorMessage = fmt.Sprintf("Validation failed: %v", msg.err)
			m.step = AuthStepError
		} else {
			m.result.Validated = true
			m.step = AuthStepSuccess
		}
		return m, nil
	}

	// Update input components based on current step
	switch m.step {
	case AuthStepAPIKey:
		var cmd tea.Cmd
		m.apiKeyInput, cmd = m.apiKeyInput.Update(msg)
		cmds = append(cmds, cmd)
	case AuthStepBaseURL:
		var cmd tea.Cmd
		m.baseURLInput, cmd = m.baseURLInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// updateInputWidths updates input component widths.
func (m *AuthWizardModel) updateInputWidths() {
	width := m.width - 4
	if width < 20 {
		width = 20
	}
	m.apiKeyInput.Width = width
	m.baseURLInput.Width = width
}

// handleKeyMsg handles keyboard input.
func (m *AuthWizardModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle quit
	if key.Matches(msg, m.keyMap.Quit) {
		m.quit = true
		return m, tea.Quit
	}

	// Handle back
	if key.Matches(msg, m.keyMap.Back) {
		if m.canGoBack() {
			m.goBack()
		}
		return m, nil
	}

	// Handle step-specific input
	switch m.step {
	case AuthStepMenu:
		return m.handleMenuStep(msg)
	case AuthStepProvider:
		return m.handleProviderStep(msg)
	case AuthStepAuthMethod:
		return m.handleAuthMethodStep(msg)
	case AuthStepAPIKey:
		return m.handleAPIKeyStep(msg)
	case AuthStepModel:
		return m.handleModelStep(msg)
	case AuthStepBaseURL:
		return m.handleBaseURLStep(msg)
	case AuthStepError:
		return m.handleErrorStep(msg)
	case AuthStepStatus:
		return m.handleStatusStep(msg)
	}

	return m, nil
}

// handleMenuStep handles the main menu step.
func (m *AuthWizardModel) handleMenuStep(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keyMap.Up):
		if m.menuIndex > 0 {
			m.menuIndex--
		} else {
			m.menuIndex = len(m.menuItems) - 1
		}
	case key.Matches(msg, m.keyMap.Down):
		if m.menuIndex < len(m.menuItems)-1 {
			m.menuIndex++
		} else {
			m.menuIndex = 0
		}
	case key.Matches(msg, m.keyMap.Select):
		return m.handleMenuSelection()
	}
	return m, nil
}

// handleMenuSelection handles menu item selection.
func (m *AuthWizardModel) handleMenuSelection() (tea.Model, tea.Cmd) {
	if m.menuIndex >= len(m.menuItems) {
		return m, nil
	}

	item := m.menuItems[m.menuIndex]
	switch item.Value {
	case "configure":
		m.loadProviders()
		m.pushStep(AuthStepProvider)
	case "status":
		m.pushStep(AuthStepStatus)
	case "reauth":
		m.loadProviders()
		if m.authStatus.Provider != "" {
			// Pre-select current provider
			for i, p := range m.providers {
				if p.Name == m.authStatus.Provider {
					m.providerIndex = i
					m.selectedProvider = p
					break
				}
			}
		}
		m.pushStep(AuthStepProvider)
	case "exit":
		m.quit = true
		return m, tea.Quit
	}
	return m, nil
}

// handleProviderStep handles provider selection.
func (m *AuthWizardModel) handleProviderStep(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keyMap.Up):
		if m.providerIndex > 0 {
			m.providerIndex--
		} else {
			m.providerIndex = len(m.providers) - 1
		}
	case key.Matches(msg, m.keyMap.Down):
		if m.providerIndex < len(m.providers)-1 {
			m.providerIndex++
		} else {
			m.providerIndex = 0
		}
	case key.Matches(msg, m.keyMap.Select):
		if m.providerIndex < len(m.providers) {
			m.selectedProvider = m.providers[m.providerIndex]
			return m.selectProvider(m.selectedProvider)
		}
	}
	return m, nil
}

// selectProvider handles provider selection and determines next step.
func (m *AuthWizardModel) selectProvider(provider auth.ProviderInfo) (tea.Model, tea.Cmd) {
	// Check for non-interactive mode flags
	if m.config.Flags.NonInteractive {
		return m.handleNonInteractive()
	}

	// Determine next step based on provider
	if len(provider.AuthMethods) == 1 {
		// Only one auth method, use it automatically
		m.authMethod = provider.AuthMethods[0]
		return m.proceedToAuthStep()
	}

	// Multiple auth methods, show selection
	m.authMethodIndex = 0
	m.pushStep(AuthStepAuthMethod)
	return m, nil
}

// proceedToAuthStep proceeds to the appropriate auth step based on method.
func (m *AuthWizardModel) proceedToAuthStep() (tea.Model, tea.Cmd) {
	switch m.authMethod {
	case auth.AuthMethodAPIKey:
		m.apiKeyInput.Reset()
		m.apiKeyInput.Focus()
		m.pushStep(AuthStepAPIKey)
	case auth.AuthMethodOAuth:
		// OAuth would be handled separately
		m.errorMessage = "OAuth authentication not yet implemented in TUI"
		m.pushStep(AuthStepError)
	case auth.AuthMethodNone:
		// No auth needed, skip to model selection
		m.pushStep(AuthStepModel)
	}
	return m, nil
}

// handleAuthMethodStep handles auth method selection.
func (m *AuthWizardModel) handleAuthMethodStep(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	methods := m.selectedProvider.AuthMethods

	switch {
	case key.Matches(msg, m.keyMap.Up):
		if m.authMethodIndex > 0 {
			m.authMethodIndex--
		} else {
			m.authMethodIndex = len(methods) - 1
		}
	case key.Matches(msg, m.keyMap.Down):
		if m.authMethodIndex < len(methods)-1 {
			m.authMethodIndex++
		} else {
			m.authMethodIndex = 0
		}
	case key.Matches(msg, m.keyMap.Select):
		if m.authMethodIndex < len(methods) {
			m.authMethod = methods[m.authMethodIndex]
			return m.proceedToAuthStep()
		}
	}
	return m, nil
}

// handleAPIKeyStep handles API key input.
func (m *AuthWizardModel) handleAPIKeyStep(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keyMap.Submit) {
		key := strings.TrimSpace(m.apiKeyInput.Value())
		if key == "" {
			return m, nil // Don't allow empty keys
		}

		// Validate key format
		if m.config.APIKeyManager != nil {
			if err := m.config.APIKeyManager.ValidateKey(m.selectedProvider.Name, key); err != nil {
				m.errorMessage = fmt.Sprintf("Invalid API key: %v", err)
				return m, nil
			}
		}

		m.apiKey = key
		m.pushStep(AuthStepModel)
		return m, nil
	}

	// Let the input component handle the key
	var cmd tea.Cmd
	m.apiKeyInput, cmd = m.apiKeyInput.Update(msg)
	return m, cmd
}

// handleModelStep handles model selection.
func (m *AuthWizardModel) handleModelStep(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	models := m.selectedProvider.Models

	switch {
	case key.Matches(msg, m.keyMap.Up):
		if m.modelIndex > 0 {
			m.modelIndex--
		} else {
			m.modelIndex = len(models) - 1
		}
	case key.Matches(msg, m.keyMap.Down):
		if m.modelIndex < len(models)-1 {
			m.modelIndex++
		} else {
			m.modelIndex = 0
		}
	case key.Matches(msg, m.keyMap.Select):
		if m.modelIndex < len(models) {
			m.model = models[m.modelIndex].ID
			// Check if base URL is needed
			if m.selectedProvider.Name == "openai" || m.selectedProvider.Name == "openai-native" {
				m.pushStep(AuthStepBaseURL)
			} else {
				return m.startValidation()
			}
		}
	}
	return m, nil
}

// handleBaseURLStep handles base URL input.
func (m *AuthWizardModel) handleBaseURLStep(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keyMap.Submit) {
		m.baseURL = strings.TrimSpace(m.baseURLInput.Value())
		return m.startValidation()
	}

	// Let the input component handle the key
	var cmd tea.Cmd
	m.baseURLInput, cmd = m.baseURLInput.Update(msg)
	return m, cmd
}

// handleErrorStep handles the error step.
func (m *AuthWizardModel) handleErrorStep(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keyMap.Select) || key.Matches(msg, m.keyMap.Back) {
		// Go back to appropriate step or menu
		if m.canGoBack() {
			m.goBack()
		} else {
			m.step = AuthStepMenu
			m.previousSteps = m.previousSteps[:0]
		}
	}
	return m, nil
}

// handleStatusStep handles the status step.
func (m *AuthWizardModel) handleStatusStep(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keyMap.Select) || key.Matches(msg, m.keyMap.Back) {
		m.goBack()
	}
	return m, nil
}

// startValidation starts the validation process.
func (m *AuthWizardModel) startValidation() (tea.Model, tea.Cmd) {
	m.validating = true
	m.validationErr = nil
	m.pushStep(AuthStepValidating)

	return m, m.validateCmd()
}

// validationResultMsg is sent when validation completes.
type validationResultMsg struct {
	err error
}

// validateCmd returns a command to validate the configuration.
func (m *AuthWizardModel) validateCmd() tea.Cmd {
	return func() tea.Msg {
		// Create context for validation
		ctx := context.Background()

	// Use wizard to validate if available
		if m.config.Wizard != nil {
			// Create temporary flags for validation
			flags := auth.WizardFlags{
				Provider:       m.selectedProvider.Name,
				AuthMethod:     string(m.authMethod),
				APIKey:         m.apiKey,
				Model:          m.model,
				NonInteractive: true,
				Force:          true,
			}

			// Run wizard in non-interactive mode to validate and save
			wizardResult, err := m.config.Wizard.Run(ctx, flags)
			if err != nil {
				return validationResultMsg{err: err}
			}

			// Update result
			m.result.Provider = wizardResult.Provider
			m.result.AuthMethod = wizardResult.AuthMethod
			m.result.APIKey = wizardResult.APIKey
			m.result.Model = wizardResult.Model
			m.result.Validated = wizardResult.Validated
			m.result.Saved = true
		} else {
			// Fallback: just test the API key
			if m.config.APIKeyManager != nil && m.apiKey != "" {
				if err := m.config.APIKeyManager.TestKey(ctx, m.selectedProvider.Name, m.apiKey); err != nil {
					return validationResultMsg{err: err}
				}
			}
			m.result.Validated = true
		}

		return validationResultMsg{err: nil}
	}
}

// handleNonInteractive handles non-interactive mode.
func (m *AuthWizardModel) handleNonInteractive() (tea.Model, tea.Cmd) {
	ctx := context.Background()

	wizardRes, err := m.config.Wizard.Run(ctx, m.config.Flags)
	if err != nil {
		m.errorMessage = fmt.Sprintf("Configuration failed: %v", err)
		m.result.Error = err
		m.pushStep(AuthStepError)
		return m, nil
	}

	m.result.Provider = wizardRes.Provider
	m.result.AuthMethod = wizardRes.AuthMethod
	m.result.APIKey = wizardRes.APIKey
	m.result.Model = wizardRes.Model
	m.result.Validated = wizardRes.Validated
	m.result.Saved = true
	m.pushStep(AuthStepSuccess)
	return m, nil
}

// pushStep pushes a new step onto the stack.
func (m *AuthWizardModel) pushStep(step AuthStep) {
	m.previousSteps = append(m.previousSteps, m.step)
	m.step = step
}

// canGoBack returns true if we can go back to a previous step.
func (m *AuthWizardModel) canGoBack() bool {
	return len(m.previousSteps) > 0
}

// goBack goes back to the previous step.
func (m *AuthWizardModel) goBack() {
	if len(m.previousSteps) == 0 {
		return
	}

	// Pop the last step
	lastIdx := len(m.previousSteps) - 1
	m.step = m.previousSteps[lastIdx]
	m.previousSteps = m.previousSteps[:lastIdx]

	// Clear step-specific state
	switch m.step {
	case AuthStepAPIKey:
		m.apiKey = ""
		m.apiKeyInput.Reset()
	case AuthStepModel:
		m.model = ""
		m.modelIndex = 0
	case AuthStepBaseURL:
		m.baseURL = ""
		m.baseURLInput.Reset()
	}
}

// View implements tea.Model.
func (m *AuthWizardModel) View() string {
	switch m.step {
	case AuthStepMenu:
		return m.renderMenu()
	case AuthStepProvider:
		return m.renderProviderSelection()
	case AuthStepAuthMethod:
		return m.renderAuthMethodSelection()
	case AuthStepAPIKey:
		return m.renderAPIKeyInput()
	case AuthStepModel:
		return m.renderModelSelection()
	case AuthStepBaseURL:
		return m.renderBaseURLInput()
	case AuthStepValidating:
		return m.renderValidating()
	case AuthStepSuccess:
		return m.renderSuccess()
	case AuthStepError:
		return m.renderError()
	case AuthStepStatus:
		return m.renderStatus()
	default:
		return "Unknown step"
	}
}

// renderMenu renders the main menu.
func (m *AuthWizardModel) renderMenu() string {
	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		Width(m.width).
		Align(lipgloss.Center)
	b.WriteString(titleStyle.Render(m.config.Title))
	b.WriteString("\n\n")

	// Welcome message
	welcomeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Width(m.width).
		Align(lipgloss.Center)
	b.WriteString(welcomeStyle.Render("Welcome to Cline"))
	b.WriteString("\n\n")

	// Menu items
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#404040")).
		Padding(1).
		Width(m.width - 4)

	var menuContent strings.Builder
	for i, item := range m.menuItems {
		if i == m.menuIndex {
			selectedStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7D56F4")).
				Bold(true)
			menuContent.WriteString(selectedStyle.Render("❯ " + item.Label))
			if item.Description != "" {
				descStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color("#808080"))
				menuContent.WriteString("\n  " + descStyle.Render(item.Description))
			}
		} else {
			normalStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#E0E0E0"))
			menuContent.WriteString(normalStyle.Render("  " + item.Label))
		}
		if i < len(m.menuItems)-1 {
			menuContent.WriteString("\n\n")
		}
	}

	b.WriteString(boxStyle.Render(menuContent.String()))
	b.WriteString("\n\n")

	// Help
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#808080")).
		Width(m.width).
		Align(lipgloss.Center)
	b.WriteString(helpStyle.Render("↑/↓ to navigate • Enter to select • Ctrl+C to quit"))

	return b.String()
}

// renderProviderSelection renders the provider selection step.
func (m *AuthWizardModel) renderProviderSelection() string {
	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4"))
	b.WriteString(titleStyle.Render("Select a Provider"))
	b.WriteString("\n\n")

	// Provider list
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#404040")).
		Padding(1).
		Width(m.width - 4)

	var listContent strings.Builder
	for i, provider := range m.providers {
		if i == m.providerIndex {
			selectedStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7D56F4")).
				Bold(true)
			listContent.WriteString(selectedStyle.Render("❯ " + provider.DisplayName))
			if provider.Description != "" {
				descStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color("#808080"))
				listContent.WriteString("\n  " + descStyle.Render(provider.Description))
			}
		} else {
			normalStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#E0E0E0"))
			listContent.WriteString(normalStyle.Render("  " + provider.DisplayName))
		}
		if i < len(m.providers)-1 {
			listContent.WriteString("\n")
		}
	}

	b.WriteString(boxStyle.Render(listContent.String()))
	b.WriteString("\n\n")

	// Help
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#808080"))
	b.WriteString(helpStyle.Render("↑/↓ to navigate • Enter to select • Esc to go back"))

	return b.String()
}

// renderAuthMethodSelection renders the auth method selection step.
func (m *AuthWizardModel) renderAuthMethodSelection() string {
	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4"))
	b.WriteString(titleStyle.Render("Select Authentication Method"))
	b.WriteString("\n\n")

	// Provider info
	infoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#A0A0A0"))
	b.WriteString(infoStyle.Render(fmt.Sprintf("Provider: %s", m.selectedProvider.DisplayName)))
	b.WriteString("\n\n")

	// Auth methods
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#404040")).
		Padding(1).
		Width(m.width - 4)

	methods := m.selectedProvider.AuthMethods
	var listContent strings.Builder

	for i, method := range methods {
		label := string(method)
		switch method {
		case auth.AuthMethodAPIKey:
			label = "API Key"
		case auth.AuthMethodOAuth:
			label = "OAuth 2.0"
		case auth.AuthMethodNone:
			label = "None (Local Provider)"
		}

		if i == m.authMethodIndex {
			selectedStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7D56F4")).
				Bold(true)
			listContent.WriteString(selectedStyle.Render("❯ " + label))
		} else {
			normalStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#E0E0E0"))
			listContent.WriteString(normalStyle.Render("  " + label))
		}
		if i < len(methods)-1 {
			listContent.WriteString("\n")
		}
	}

	b.WriteString(boxStyle.Render(listContent.String()))
	b.WriteString("\n\n")

	// Help
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#808080"))
	b.WriteString(helpStyle.Render("↑/↓ to navigate • Enter to select • Esc to go back"))

	return b.String()
}

// renderAPIKeyInput renders the API key input step.
func (m *AuthWizardModel) renderAPIKeyInput() string {
	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4"))
	b.WriteString(titleStyle.Render("Enter API Key"))
	b.WriteString("\n\n")

	// Provider info
	infoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#A0A0A0"))
	b.WriteString(infoStyle.Render(fmt.Sprintf("Provider: %s", m.selectedProvider.DisplayName)))
	b.WriteString("\n")

	// Help text
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#808080"))
	b.WriteString(helpStyle.Render("Your API key will be securely stored."))
	b.WriteString("\n\n")

	// Input box
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#404040")).
		Padding(1).
		Width(m.width - 4)

	b.WriteString(boxStyle.Render(m.apiKeyInput.View()))
	b.WriteString("\n\n")

	// Help
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#808080"))
	b.WriteString(footerStyle.Render("Enter to submit • Esc to go back"))

	return b.String()
}

// renderModelSelection renders the model selection step.
func (m *AuthWizardModel) renderModelSelection() string {
	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4"))
	b.WriteString(titleStyle.Render("Select a Model"))
	b.WriteString("\n\n")

	// Provider info
	infoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#A0A0A0"))
	b.WriteString(infoStyle.Render(fmt.Sprintf("Provider: %s", m.selectedProvider.DisplayName)))
	b.WriteString("\n\n")

	// Model list
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#404040")).
		Padding(1).
		Width(m.width - 4)

	var listContent strings.Builder
	for i, model := range m.selectedProvider.Models {
		if i == m.modelIndex {
			selectedStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7D56F4")).
				Bold(true)
			listContent.WriteString(selectedStyle.Render("❯ " + model.Name))
			if model.Description != "" {
				descStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color("#808080"))
				listContent.WriteString("\n  " + descStyle.Render(model.Description))
			}
			if model.ContextSize > 0 {
				ctxStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color("#606060"))
				listContent.WriteString(fmt.Sprintf("\n  %s", ctxStyle.Render(fmt.Sprintf("Context: %d tokens", model.ContextSize))))
			}
		} else {
			normalStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#E0E0E0"))
			listContent.WriteString(normalStyle.Render("  " + model.Name))
		}
		if i < len(m.selectedProvider.Models)-1 {
			listContent.WriteString("\n")
		}
	}

	b.WriteString(boxStyle.Render(listContent.String()))
	b.WriteString("\n\n")

	// Help
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#808080"))
	b.WriteString(helpStyle.Render("↑/↓ to navigate • Enter to select • Esc to go back"))

	return b.String()
}

// renderBaseURLInput renders the base URL input step.
func (m *AuthWizardModel) renderBaseURLInput() string {
	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4"))
	b.WriteString(titleStyle.Render("Base URL (Optional)"))
	b.WriteString("\n\n")

	// Help text
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#808080"))
	b.WriteString(helpStyle.Render("For self-hosted or proxy endpoints. Leave empty to use default."))
	b.WriteString("\n\n")

	// Input box
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#404040")).
		Padding(1).
		Width(m.width - 4)

	b.WriteString(boxStyle.Render(m.baseURLInput.View()))
	b.WriteString("\n\n")

	// Help
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#808080"))
	b.WriteString(footerStyle.Render("Enter to continue • Esc to go back"))

	return b.String()
}

// renderValidating renders the validation step.
func (m *AuthWizardModel) renderValidating() string {
	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		Width(m.width).
		Align(lipgloss.Center)
	b.WriteString(titleStyle.Render("Validating Configuration"))
	b.WriteString("\n\n")

	// Spinner
	spinnerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7D56F4")).
		Width(m.width).
		Align(lipgloss.Center)
	b.WriteString(spinnerStyle.Render(m.spinner.View()))
	b.WriteString("\n")

	// Status
	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#A0A0A0")).
		Width(m.width).
		Align(lipgloss.Center)
	b.WriteString(statusStyle.Render("Testing API connection..."))

	return b.String()
}

// renderSuccess renders the success step.
func (m *AuthWizardModel) renderSuccess() string {
	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#4CAF50")).
		Width(m.width).
		Align(lipgloss.Center)
	b.WriteString(titleStyle.Render("✓ Authentication Successful"))
	b.WriteString("\n\n")

	// Details box
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#4CAF50")).
		Padding(1).
		Width(m.width - 4)

	var details strings.Builder
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#808080"))
	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF"))

	if m.result.Provider != "" {
		details.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Provider:"), valueStyle.Render(m.result.Provider)))
	}
	if m.result.Model != "" {
		details.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Model:"), valueStyle.Render(m.result.Model)))
	}
	if m.result.AuthMethod != "" {
		details.WriteString(fmt.Sprintf("%s %s", labelStyle.Render("Method:"), valueStyle.Render(string(m.result.AuthMethod))))
	}

	b.WriteString(boxStyle.Render(details.String()))
	b.WriteString("\n\n")

	// Help
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#808080")).
		Width(m.width).
		Align(lipgloss.Center)
	b.WriteString(helpStyle.Render("Press Enter to continue"))

	return b.String()
}

// renderError renders the error step.
func (m *AuthWizardModel) renderError() string {
	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#F44336")).
		Width(m.width).
		Align(lipgloss.Center)
	b.WriteString(titleStyle.Render("✗ Authentication Failed"))
	b.WriteString("\n\n")

	// Error box
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#F44336")).
		Padding(1).
		Width(m.width - 4)

	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFC107"))
	b.WriteString(boxStyle.Render(errorStyle.Render(m.errorMessage)))
	b.WriteString("\n\n")

	// Help
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#808080")).
		Width(m.width).
		Align(lipgloss.Center)
	b.WriteString(helpStyle.Render("Press Enter or Esc to go back"))

	return b.String()
}

// renderStatus renders the authentication status step.
func (m *AuthWizardModel) renderStatus() string {
	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4"))
	b.WriteString(titleStyle.Render("Authentication Status"))
	b.WriteString("\n\n")

	// Status box
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#404040")).
		Padding(1).
		Width(m.width - 4)

	var status strings.Builder
	if m.authStatus.IsAuthenticated {
		successStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4CAF50"))
		status.WriteString(successStyle.Render("✓ Authenticated"))
		status.WriteString("\n\n")

		labelStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#808080"))
		valueStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF"))

		status.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Provider:"), valueStyle.Render(m.authStatus.DisplayName)))
		if m.authStatus.Model != "" {
			status.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Model:"), valueStyle.Render(m.authStatus.Model)))
		}
		if m.authStatus.APIKeyMasked != "" {
			status.WriteString(fmt.Sprintf("%s %s", labelStyle.Render("API Key:"), valueStyle.Render(m.authStatus.APIKeyMasked)))
		}
	} else {
		warningStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFC107"))
		status.WriteString(warningStyle.Render("⚠ Not Authenticated"))
		status.WriteString("\n\n")

		infoStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A0A0A0"))
		status.WriteString(infoStyle.Render("Run 'cline auth' to configure authentication"))
	}

	b.WriteString(boxStyle.Render(status.String()))
	b.WriteString("\n\n")

	// Help
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#808080"))
	b.WriteString(helpStyle.Render("Press Enter or Esc to go back"))

	return b.String()
}

// Result returns the authentication result.
func (m *AuthWizardModel) Result() AuthResult {
	authResult := m.result
	authResult.Step = m.step
	return authResult
}

// Run executes the auth wizard and returns the result.
func (m *AuthWizardModel) Run() (AuthResult, error) {
	p := tea.NewProgram(m, tea.WithAltScreen())
	model, err := p.Run()
	if err != nil {
		return AuthResult{}, err
	}

	am, ok := model.(*AuthWizardModel)
	if !ok {
		return AuthResult{}, errors.New("unexpected model type")
	}

	return am.Result(), nil
}

// RunAuthWizard runs the authentication wizard with the given configuration.
func RunAuthWizard(config AuthWizardConfig) (AuthResult, error) {
	// Check if non-interactive mode
	if config.Flags.NonInteractive {
		return runNonInteractive(config)
	}

	model := NewAuthWizardModel(config)
	return model.Run()
}

// runNonInteractive runs the wizard in non-interactive mode.
func runNonInteractive(config AuthWizardConfig) (AuthResult, error) {
	ctx := context.Background()

	if config.Wizard == nil {
		return AuthResult{}, errors.New("wizard not configured for non-interactive mode")
	}

	wizardResult, err := config.Wizard.Run(ctx, config.Flags)
	if err != nil {
		return AuthResult{
			Validated: false,
			Error:     err,
		}, err
	}

	return AuthResult{
		Provider:   wizardResult.Provider,
		AuthMethod: wizardResult.AuthMethod,
		APIKey:     wizardResult.APIKey,
		Model:      wizardResult.Model,
		Validated:  wizardResult.Validated,
		Saved:      true,
		Step:       AuthStepSuccess,
	}, nil
}

// ShowAuthStatus displays the current authentication status.
func ShowAuthStatus(storageCtx *storage.StorageContext, apiKeyManager *auth.APIKeyManager) error {
	if storageCtx == nil {
		return errors.New("storage context is required")
	}

	config := DefaultAuthWizardConfig()
	config.StorageContext = storageCtx
	config.APIKeyManager = apiKeyManager

	model := NewAuthWizardModel(config)
	model.step = AuthStepStatus

	// Load auth status
	msg := model.loadAuthStatus()
	if statusMsg, ok := msg.(authStatusLoadedMsg); ok {
		model.authStatus = statusMsg.status
	}

	// Just display and wait for key press
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// CheckAuth checks if the user is authenticated.
func CheckAuth(storageCtx *storage.StorageContext, apiKeyManager *auth.APIKeyManager) (bool, error) {
	if storageCtx == nil {
		return false, errors.New("storage context is required")
	}

	// Get provider
	providerVal, ok := storageCtx.GlobalState.Get("apiProvider")
	if !ok {
		return false, nil
	}

	provider, ok := providerVal.(string)
	if !ok || provider == "" {
		return false, nil
	}

	// Check if API key exists
	if apiKeyManager != nil {
		if key, err := apiKeyManager.GetKey(provider); err == nil && key != "" {
			return true, nil
		}
	}

	return false, nil
}

// RequireAuth ensures the user is authenticated, running the wizard if needed.
func RequireAuth(storageCtx *storage.StorageContext, apiKeyManager *auth.APIKeyManager, wizard *auth.Wizard, force bool) (AuthResult, error) {
	isAuth, err := CheckAuth(storageCtx, apiKeyManager)
	if err != nil {
		return AuthResult{}, err
	}

	if isAuth && !force {
		// Already authenticated
		return AuthResult{
			Validated: true,
			Saved:     true,
		}, nil
	}

	// Run the wizard
	config := DefaultAuthWizardConfig()
	config.StorageContext = storageCtx
	config.APIKeyManager = apiKeyManager
	config.Wizard = wizard

	return RunAuthWizard(config)
}