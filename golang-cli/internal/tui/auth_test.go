// Package tui provides tests for the authentication wizard components.
package tui

import (
	"errors"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/cline/cline/golang-cli/internal/auth"
	"github.com/cline/cline/golang-cli/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockAuthKeyManager is a mock implementation of APIKeyManager for testing.
type MockAuthKeyManager struct {
	keys map[string]string
}

func NewMockAuthKeyManager() *MockAuthKeyManager {
	return &MockAuthKeyManager{
		keys: make(map[string]string),
	}
}

func (m *MockAuthKeyManager) GetKey(provider string) (string, error) {
	if key, ok := m.keys[provider]; ok {
		return key, nil
	}
	return "", auth.ErrKeyNotFound
}

func (m *MockAuthKeyManager) StoreKey(provider, key string, metadata *auth.KeyMetadata) error {
	m.keys[provider] = key
	return nil
}

func (m *MockAuthKeyManager) ValidateKey(provider, key string) error {
	if key == "" {
		return auth.ErrInvalidKeyFormat
	}
	return nil
}

func (m *MockAuthKeyManager) TestKey(ctx interface{}, provider, key string) error {
	return nil
}

func (m *MockAuthKeyManager) KeyExists(provider string) (bool, error) {
	_, ok := m.keys[provider]
	return ok, nil
}

// MockWizard is a mock implementation of auth.Wizard for testing.
type MockWizard struct {
	providers map[string]auth.ProviderInfo
}

func NewMockWizard() *MockWizard {
	w := &MockWizard{
		providers: make(map[string]auth.ProviderInfo),
	}
	w.registerMockProviders()
	return w
}

func (w *MockWizard) registerMockProviders() {
	w.providers["anthropic"] = auth.ProviderInfo{
		Name:         "anthropic",
		DisplayName:  "Anthropic",
		Description:  "Claude models by Anthropic",
		AuthMethods:  []auth.AuthMethod{auth.AuthMethodAPIKey},
		DefaultModel: "claude-3-sonnet-20240229",
		Models: []auth.ModelInfo{
			{
				ID:          "claude-3-opus-20240229",
				Name:        "Claude 3 Opus",
				Description: "Most powerful model",
				ContextSize: 200000,
			},
			{
				ID:          "claude-3-sonnet-20240229",
				Name:        "Claude 3 Sonnet",
				Description: "Balanced performance",
				ContextSize: 200000,
			},
		},
	}

	w.providers["openai"] = auth.ProviderInfo{
		Name:         "openai",
		DisplayName:  "OpenAI",
		Description:  "GPT models by OpenAI",
		AuthMethods:  []auth.AuthMethod{auth.AuthMethodAPIKey},
		DefaultModel: "gpt-4",
		Models: []auth.ModelInfo{
			{
				ID:          "gpt-4",
				Name:        "GPT-4",
				Description: "Most capable GPT-4 model",
				ContextSize: 8192,
			},
		},
	}

	w.providers["ollama"] = auth.ProviderInfo{
		Name:         "ollama",
		DisplayName:  "Ollama",
		Description:  "Local models via Ollama",
		AuthMethods:  []auth.AuthMethod{auth.AuthMethodNone},
		DefaultModel: "llama3",
		Models: []auth.ModelInfo{
			{
				ID:          "llama3",
				Name:        "Llama 3",
				Description: "Meta's Llama 3 model",
				ContextSize: 8192,
			},
		},
	}
}

func (w *MockWizard) GetProviderInfo(name string) (auth.ProviderInfo, bool) {
	info, ok := w.providers[name]
	return info, ok
}

func (w *MockWizard) ListProviders() []string {
	names := make([]string, 0, len(w.providers))
	for name := range w.providers {
		names = append(names, name)
	}
	return names
}

func TestNewAuthWizardModel(t *testing.T) {
	config := DefaultAuthWizardConfig()
	model := NewAuthWizardModel(config)

	assert.NotNil(t, model)
	assert.Equal(t, AuthStepMenu, model.step)
	assert.Equal(t, "Cline Authentication", model.config.Title)
	assert.NotNil(t, model.apiKeyInput)
	assert.NotNil(t, model.baseURLInput)
	assert.NotNil(t, model.spinner)
	assert.Equal(t, 4, len(model.menuItems))
}

func TestAuthWizardModel_Init(t *testing.T) {
	config := DefaultAuthWizardConfig()
	model := NewAuthWizardModel(config)

	cmd := model.Init()
	assert.NotNil(t, cmd)
}

func TestAuthWizardModel_handleMenuStep(t *testing.T) {
	config := DefaultAuthWizardConfig()
	model := NewAuthWizardModel(config)

	// Test navigating down
	msg := tea.KeyMsg{Type: tea.KeyDown}
	newModel, _ := model.handleMenuStep(msg)
	m, ok := newModel.(*AuthWizardModel)
	require.True(t, ok)
	assert.Equal(t, 1, m.menuIndex)

	// Test navigating up
	msg = tea.KeyMsg{Type: tea.KeyUp}
	newModel, _ = m.handleMenuStep(msg)
	m, ok = newModel.(*AuthWizardModel)
	require.True(t, ok)
	assert.Equal(t, 0, m.menuIndex)

	// Test wrap around down
	m.menuIndex = len(m.menuItems) - 1
	msg = tea.KeyMsg{Type: tea.KeyDown}
	newModel, _ = m.handleMenuStep(msg)
	m, ok = newModel.(*AuthWizardModel)
	require.True(t, ok)
	assert.Equal(t, 0, m.menuIndex)

	// Test wrap around up
	m.menuIndex = 0
	msg = tea.KeyMsg{Type: tea.KeyUp}
	newModel, _ = m.handleMenuStep(msg)
	m, ok = newModel.(*AuthWizardModel)
	require.True(t, ok)
	assert.Equal(t, len(m.menuItems)-1, m.menuIndex)
}

func TestAuthWizardModel_handleMenuSelection(t *testing.T) {
	config := DefaultAuthWizardConfig()
	config.Wizard = NewMockWizard()
	model := NewAuthWizardModel(config)

	// Test configure selection
	model.menuIndex = 0 // "Configure new provider"
	newModel, _ := model.handleMenuSelection()
	m, ok := newModel.(*AuthWizardModel)
	require.True(t, ok)
	assert.Equal(t, AuthStepProvider, m.step)
	assert.True(t, len(m.providers) > 0)

	// Test status selection
	model = NewAuthWizardModel(config)
	model.menuIndex = 1 // "View authentication status"
	newModel, _ = model.handleMenuSelection()
	m, ok = newModel.(*AuthWizardModel)
	require.True(t, ok)
	assert.Equal(t, AuthStepStatus, m.step)

	// Test exit selection
	model = NewAuthWizardModel(config)
	model.menuIndex = 3 // "Exit"
	newModel, cmd := model.handleMenuSelection()
	m, ok = newModel.(*AuthWizardModel)
	require.True(t, ok)
	assert.True(t, m.quit)
	assert.NotNil(t, cmd)
}

func TestAuthWizardModel_handleProviderStep(t *testing.T) {
	config := DefaultAuthWizardConfig()
	config.Wizard = NewMockWizard()
	model := NewAuthWizardModel(config)
	model.loadProviders()

	// Test navigating down
	msg := tea.KeyMsg{Type: tea.KeyDown}
	newModel, _ := model.handleProviderStep(msg)
	m, ok := newModel.(*AuthWizardModel)
	require.True(t, ok)
	assert.Equal(t, 1, m.providerIndex)

	// Test navigating up
	msg = tea.KeyMsg{Type: tea.KeyUp}
	newModel, _ = m.handleProviderStep(msg)
	m, ok = newModel.(*AuthWizardModel)
	require.True(t, ok)
	assert.Equal(t, 0, m.providerIndex)

	// Test selection
	msg = tea.KeyMsg{Type: tea.KeyEnter}
	newModel, _ = m.handleProviderStep(msg)
	m, ok = newModel.(*AuthWizardModel)
	require.True(t, ok)
	assert.NotEmpty(t, m.selectedProvider.Name)
}

func TestAuthWizardModel_handleAuthMethodStep(t *testing.T) {
	config := DefaultAuthWizardConfig()
	config.Wizard = NewMockWizard()
	model := NewAuthWizardModel(config)
	model.loadProviders()
	model.selectedProvider = model.providers[0] // Anthropic with API key only

	// For single auth method, it should auto-select
	newModel, _ := model.selectProvider(model.selectedProvider)
	m, ok := newModel.(*AuthWizardModel)
	require.True(t, ok)
	assert.Equal(t, auth.AuthMethodAPIKey, m.authMethod)
	assert.Equal(t, AuthStepAPIKey, m.step)
}

func TestAuthWizardModel_handleAPIKeyStep(t *testing.T) {
	config := DefaultAuthWizardConfig()
	config.APIKeyManager = NewMockAuthKeyManager()
	config.Wizard = NewMockWizard()
	model := NewAuthWizardModel(config)
	model.selectedProvider = auth.ProviderInfo{
		Name: "anthropic",
		AuthMethods: []auth.AuthMethod{auth.AuthMethodAPIKey},
	}

	// Test empty key (should not proceed)
	model.apiKeyInput.SetValue("")
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, _ := model.handleAPIKeyStep(msg)
	m, ok := newModel.(*AuthWizardModel)
	require.True(t, ok)
	assert.Equal(t, AuthStepAPIKey, m.step) // Should stay on same step

	// Test valid key
	model.apiKeyInput.SetValue("sk-ant-test123")
	msg = tea.KeyMsg{Type: tea.KeyEnter}
	newModel, _ = model.handleAPIKeyStep(msg)
	m, ok = newModel.(*AuthWizardModel)
	require.True(t, ok)
	assert.Equal(t, "sk-ant-test123", m.apiKey)
	assert.Equal(t, AuthStepModel, m.step)
}

func TestAuthWizardModel_handleModelStep(t *testing.T) {
	config := DefaultAuthWizardConfig()
	config.Wizard = NewMockWizard()
	model := NewAuthWizardModel(config)
	model.loadProviders()
	model.selectedProvider = model.providers[0] // Anthropic

	// Test navigating down
	msg := tea.KeyMsg{Type: tea.KeyDown}
	newModel, _ := model.handleModelStep(msg)
	m, ok := newModel.(*AuthWizardModel)
	require.True(t, ok)
	assert.Equal(t, 1, m.modelIndex)

	// Test navigating up
	msg = tea.KeyMsg{Type: tea.KeyUp}
	newModel, _ = m.handleModelStep(msg)
	m, ok = newModel.(*AuthWizardModel)
	require.True(t, ok)
	assert.Equal(t, 0, m.modelIndex)

	// Test selection (should go to validation for non-OpenAI)
	msg = tea.KeyMsg{Type: tea.KeyEnter}
	newModel, cmd := m.handleModelStep(msg)
	m, ok = newModel.(*AuthWizardModel)
	require.True(t, ok)
	assert.Equal(t, AuthStepValidating, m.step)
	assert.NotNil(t, cmd)
}

func TestAuthWizardModel_pushStepAndGoBack(t *testing.T) {
	config := DefaultAuthWizardConfig()
	model := NewAuthWizardModel(config)

	// Push steps
	model.pushStep(AuthStepProvider)
	assert.Equal(t, AuthStepProvider, model.step)
	assert.Equal(t, 1, len(model.previousSteps))

	model.pushStep(AuthStepAPIKey)
	assert.Equal(t, AuthStepAPIKey, model.step)
	assert.Equal(t, 2, len(model.previousSteps))

	// Go back
	model.goBack()
	assert.Equal(t, AuthStepProvider, model.step)
	assert.Equal(t, 1, len(model.previousSteps))

	model.goBack()
	assert.Equal(t, AuthStepMenu, model.step)
	assert.Equal(t, 0, len(model.previousSteps))

	// Go back when no history
	model.goBack() // Should not panic
	assert.Equal(t, AuthStepMenu, model.step)
}

func TestAuthWizardModel_canGoBack(t *testing.T) {
	config := DefaultAuthWizardConfig()
	model := NewAuthWizardModel(config)

	assert.False(t, model.canGoBack())

	model.pushStep(AuthStepProvider)
	assert.True(t, model.canGoBack())
}

func TestMaskAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{
			name:     "short key",
			key:      "abc",
			expected: "****",
		},
		{
			name:     "exactly 8 chars",
			key:      "abcdefgh",
			expected: "****",
		},
		{
			name:     "long key",
			key:      "sk-ant-api123456789",
			expected: "sk-a...6789",
		},
		{
			name:     "empty key",
			key:      "",
			expected: "****",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskAPIKey(tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRunNonInteractive(t *testing.T) {
	// Test with nil wizard
	config := DefaultAuthWizardConfig()
	config.Flags = auth.WizardFlags{
		NonInteractive: true,
		Provider:       "anthropic",
		APIKey:         "sk-ant-test",
		Model:          "claude-3-sonnet",
	}

	result, err := runNonInteractive(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "wizard not configured")
	assert.False(t, result.Validated)

	// Test with valid wizard
	config.Wizard = NewMockWizard()
	result, err = runNonInteractive(config)
	// Will fail because MockWizard doesn't have a Run method
	// This tests the error handling path
	assert.Error(t, err)
}

func TestCheckAuth(t *testing.T) {
	// Test with nil storage
	authenticated, err := CheckAuth(nil, nil)
	assert.Error(t, err)
	assert.False(t, authenticated)

	// Note: Full integration tests with storage would require
	// setting up a full storage context, which is better done
	// in integration tests.
}

func TestShowAuthStatus(t *testing.T) {
	// Test with nil storage
	err := ShowAuthStatus(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "storage context is required")
}

func TestRequireAuth(t *testing.T) {
	// Test with nil storage
	result, err := RequireAuth(nil, nil, nil, false)
	assert.Error(t, err)

	// Note: Full integration tests would require storage setup
	_ = result
}

func TestAuthWizardModel_RenderMenu(t *testing.T) {
	config := DefaultAuthWizardConfig()
	model := NewAuthWizardModel(config)
	model.width = 80

	view := model.renderMenu()
	assert.Contains(t, view, "Cline Authentication")
	assert.Contains(t, view, "Configure new provider")
	assert.Contains(t, view, "View authentication status")
	assert.Contains(t, view, "Re-authenticate")
	assert.Contains(t, view, "Exit")
}

func TestAuthWizardModel_RenderProviderSelection(t *testing.T) {
	config := DefaultAuthWizardConfig()
	config.Wizard = NewMockWizard()
	model := NewAuthWizardModel(config)
	model.loadProviders()
	model.width = 80

	view := model.renderProviderSelection()
	assert.Contains(t, view, "Select a Provider")
	assert.Contains(t, view, "Anthropic")
	assert.Contains(t, view, "OpenAI")
}

func TestAuthWizardModel_RenderAPIKeyInput(t *testing.T) {
	config := DefaultAuthWizardConfig()
	model := NewAuthWizardModel(config)
	model.selectedProvider = auth.ProviderInfo{
		Name:        "anthropic",
		DisplayName: "Anthropic",
	}
	model.width = 80

	view := model.renderAPIKeyInput()
	assert.Contains(t, view, "Enter API Key")
	assert.Contains(t, view, "Anthropic")
	assert.Contains(t, view, "securely stored")
}

func TestAuthWizardModel_RenderModelSelection(t *testing.T) {
	config := DefaultAuthWizardConfig()
	config.Wizard = NewMockWizard()
	model := NewAuthWizardModel(config)
	model.loadProviders()
	model.selectedProvider = model.providers[0]
	model.width = 80

	view := model.renderModelSelection()
	assert.Contains(t, view, "Select a Model")
	assert.Contains(t, view, "Claude 3 Opus")
	assert.Contains(t, view, "Claude 3 Sonnet")
}

func TestAuthWizardModel_RenderSuccess(t *testing.T) {
	config := DefaultAuthWizardConfig()
	model := NewAuthWizardModel(config)
	model.result = AuthResult{
		Provider:   "anthropic",
		Model:      "claude-3-sonnet",
		AuthMethod: auth.AuthMethodAPIKey,
		Validated:  true,
	}
	model.width = 80

	view := model.renderSuccess()
	assert.Contains(t, view, "Authentication Successful")
	assert.Contains(t, view, "anthropic")
	assert.Contains(t, view, "claude-3-sonnet")
}

func TestAuthWizardModel_RenderError(t *testing.T) {
	config := DefaultAuthWizardConfig()
	model := NewAuthWizardModel(config)
	model.errorMessage = "Invalid API key"
	model.width = 80

	view := model.renderError()
	assert.Contains(t, view, "Authentication Failed")
	assert.Contains(t, view, "Invalid API key")
}

func TestAuthWizardModel_RenderStatus_Authenticated(t *testing.T) {
	config := DefaultAuthWizardConfig()
	model := NewAuthWizardModel(config)
	model.authStatus = AuthStatusInfo{
		Provider:        "anthropic",
		DisplayName:     "Anthropic",
		IsAuthenticated: true,
		Model:           "claude-3-sonnet",
		APIKeyMasked:    "sk-a...6789",
	}
	model.width = 80

	view := model.renderStatus()
	assert.Contains(t, view, "Authentication Status")
	assert.Contains(t, view, "Authenticated")
	assert.Contains(t, view, "Anthropic")
	assert.Contains(t, view, "claude-3-sonnet")
	assert.Contains(t, view, "sk-a...6789")
}

func TestAuthWizardModel_RenderStatus_NotAuthenticated(t *testing.T) {
	config := DefaultAuthWizardConfig()
	model := NewAuthWizardModel(config)
	model.authStatus = AuthStatusInfo{
		IsAuthenticated: false,
	}
	model.width = 80

	view := model.renderStatus()
	assert.Contains(t, view, "Authentication Status")
	assert.Contains(t, view, "Not Authenticated")
	assert.Contains(t, view, "cline auth")
}

func TestAuthWizardModel_RenderValidating(t *testing.T) {
	config := DefaultAuthWizardConfig()
	model := NewAuthWizardModel(config)
	model.width = 80

	view := model.renderValidating()
	assert.Contains(t, view, "Validating Configuration")
	assert.Contains(t, view, "Testing API connection")
}

func TestAuthWizardModel_UpdateInputWidths(t *testing.T) {
	config := DefaultAuthWizardConfig()
	model := NewAuthWizardModel(config)

	// Test with normal width
	model.width = 80
	model.updateInputWidths()
	assert.Equal(t, 76, model.apiKeyInput.Width)
	assert.Equal(t, 76, model.baseURLInput.Width)

	// Test with small width
	model.width = 10
	model.updateInputWidths()
	assert.Equal(t, 20, model.apiKeyInput.Width) // Minimum width
	assert.Equal(t, 20, model.baseURLInput.Width)
}

func TestAuthWizardModel_HandleQuit(t *testing.T) {
	config := DefaultAuthWizardConfig()
	model := NewAuthWizardModel(config)

	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	newModel, cmd := model.handleKeyMsg(msg)
	m, ok := newModel.(*AuthWizardModel)
	require.True(t, ok)
	assert.True(t, m.quit)
	assert.NotNil(t, cmd)
}

func TestAuthWizardModel_HandleBack(t *testing.T) {
	config := DefaultAuthWizardConfig()
	model := NewAuthWizardModel(config)
	model.pushStep(AuthStepProvider)

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	newModel, _ := model.handleKeyMsg(msg)
	m, ok := newModel.(*AuthWizardModel)
	require.True(t, ok)
	assert.Equal(t, AuthStepMenu, m.step)
}

func TestAuthWizardKeyMap(t *testing.T) {
	km := DefaultAuthWizardKeyMap()

	assert.NotNil(t, km.Up)
	assert.NotNil(t, km.Down)
	assert.NotNil(t, km.Select)
	assert.NotNil(t, km.Back)
	assert.NotNil(t, km.Quit)
	assert.NotNil(t, km.Submit)
	assert.NotNil(t, km.Confirm)
}

func TestGetMainMenuItems(t *testing.T) {
	items := getMainMenuItems()
	assert.Equal(t, 4, len(items))

	assert.Equal(t, "configure", items[0].Value)
	assert.Equal(t, "Configure new provider", items[0].Label)

	assert.Equal(t, "status", items[1].Value)
	assert.Equal(t, "View authentication status", items[1].Label)

	assert.Equal(t, "reauth", items[2].Value)
	assert.Equal(t, "Re-authenticate", items[2].Label)

	assert.Equal(t, "exit", items[3].Value)
	assert.Equal(t, "Exit", items[3].Label)
}

func TestAuthWizardModel_LoadAuthStatus(t *testing.T) {
	config := DefaultAuthWizardConfig()
	model := NewAuthWizardModel(config)

	// Test with nil storage
	msg := model.loadAuthStatus()
	statusMsg, ok := msg.(authStatusLoadedMsg)
	require.True(t, ok)
	assert.False(t, statusMsg.status.IsAuthenticated)

	// Note: Full integration tests with storage would require
	// setting up a full storage context with actual data.
}

func TestDefaultAuthWizardConfig(t *testing.T) {
	config := DefaultAuthWizardConfig()
	assert.Equal(t, "Cline Authentication", config.Title)
	assert.False(t, config.ForceReauth)
	assert.Nil(t, config.StorageContext)
	assert.Nil(t, config.APIKeyManager)
	assert.Nil(t, config.Wizard)
}

func TestAuthResult(t *testing.T) {
	config := DefaultAuthWizardConfig()
	model := NewAuthWizardModel(config)
	model.result = AuthResult{
		Provider:   "anthropic",
		AuthMethod: auth.AuthMethodAPIKey,
		APIKey:     "sk-test",
		Model:      "claude-3",
		Validated:  true,
		Saved:      true,
	}

	result := model.Result()
	assert.Equal(t, "anthropic", result.Provider)
	assert.Equal(t, auth.AuthMethodAPIKey, result.AuthMethod)
	assert.Equal(t, "sk-test", result.APIKey)
	assert.Equal(t, "claude-3", result.Model)
	assert.True(t, result.Validated)
	assert.True(t, result.Saved)
}

func TestAuthWizardModel_LoadProviders(t *testing.T) {
	config := DefaultAuthWizardConfig()
	
	// Test with nil wizard
	model := NewAuthWizardModel(config)
	model.loadProviders()
	assert.Equal(t, 0, len(model.providers))

	// Test with mock wizard
	config.Wizard = NewMockWizard()
	model = NewAuthWizardModel(config)
	model.loadProviders()
	assert.True(t, len(model.providers) > 0)
}

func TestAuthWizardModel_SelectProvider_WithNonInteractive(t *testing.T) {
	config := DefaultAuthWizardConfig()
	config.Flags = auth.WizardFlags{
		NonInteractive: true,
	}
	config.Wizard = NewMockWizard()
	model := NewAuthWizardModel(config)
	model.selectedProvider = auth.ProviderInfo{
		Name:        "anthropic",
		AuthMethods: []auth.AuthMethod{auth.AuthMethodAPIKey},
	}

	// This should trigger non-interactive handling
	// which will fail because we don't have a real wizard Run method
	newModel, _ := model.selectProvider(model.selectedProvider)
	m, ok := newModel.(*AuthWizardModel)
	require.True(t, ok)
	// The error handling will set step to Error
	assert.Equal(t, AuthStepError, m.step)
}

func TestAuthWizardModel_ProceedToAuthStep(t *testing.T) {
	config := DefaultAuthWizardConfig()
	config.APIKeyManager = NewMockAuthKeyManager()

	// Test API key method
	model := NewAuthWizardModel(config)
	model.authMethod = auth.AuthMethodAPIKey
	newModel, _ := model.proceedToAuthStep()
	m, ok := newModel.(*AuthWizardModel)
	require.True(t, ok)
	assert.Equal(t, AuthStepAPIKey, m.step)

	// Test OAuth method
	model = NewAuthWizardModel(config)
	model.authMethod = auth.AuthMethodOAuth
	newModel, _ = model.proceedToAuthStep()
	m, ok = newModel.(*AuthWizardModel)
	require.True(t, ok)
	assert.Equal(t, AuthStepError, m.step) // OAuth not implemented
	assert.Contains(t, m.errorMessage, "OAuth")

	// Test None method
	model = NewAuthWizardModel(config)
	model.authMethod = auth.AuthMethodNone
	newModel, _ = model.proceedToAuthStep()
	m, ok = newModel.(*AuthWizardModel)
	require.True(t, ok)
	assert.Equal(t, AuthStepModel, m.step)
}

func TestAuthWizardModel_HandleBaseURLStep(t *testing.T) {
	config := DefaultAuthWizardConfig()
	config.Wizard = NewMockWizard()
	model := NewAuthWizardModel(config)

	// Test entering base URL
	model.baseURLInput.SetValue("https://api.example.com")
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, cmd := model.handleBaseURLStep(msg)
	m, ok := newModel.(*AuthWizardModel)
	require.True(t, ok)
	assert.Equal(t, "https://api.example.com", m.baseURL)
	assert.Equal(t, AuthStepValidating, m.step)
	assert.NotNil(t, cmd)

	// Test empty base URL
	model = NewAuthWizardModel(config)
	model.baseURLInput.SetValue("")
	msg = tea.KeyMsg{Type: tea.KeyEnter}
	newModel, cmd = model.handleBaseURLStep(msg)
	m, ok = newModel.(*AuthWizardModel)
	require.True(t, ok)
	assert.Equal(t, "", m.baseURL)
	assert.Equal(t, AuthStepValidating, m.step)
	assert.NotNil(t, cmd)
}

func TestAuthWizardModel_HandleErrorStep(t *testing.T) {
	config := DefaultAuthWizardConfig()
	model := NewAuthWizardModel(config)
	model.step = AuthStepError
	model.previousSteps = []AuthStep{AuthStepMenu}

	// Test Enter key
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, _ := model.handleErrorStep(msg)
	m, ok := newModel.(*AuthWizardModel)
	require.True(t, ok)
	assert.Equal(t, AuthStepMenu, m.step)

	// Test Esc key
	model.step = AuthStepError
	model.previousSteps = []AuthStep{AuthStepMenu}
	msg = tea.KeyMsg{Type: tea.KeyEsc}
	newModel, _ = model.handleErrorStep(msg)
	m, ok = newModel.(*AuthWizardModel)
	require.True(t, ok)
	assert.Equal(t, AuthStepMenu, m.step)

	// Test with no previous steps
	model.step = AuthStepError
	model.previousSteps = []AuthStep{}
	msg = tea.KeyMsg{Type: tea.KeyEnter}
	newModel, _ = model.handleErrorStep(msg)
	m, ok = newModel.(*AuthWizardModel)
	require.True(t, ok)
	assert.Equal(t, AuthStepMenu, m.step)
}

func TestAuthWizardModel_HandleStatusStep(t *testing.T) {
	config := DefaultAuthWizardConfig()
	model := NewAuthWizardModel(config)
	model.step = AuthStepStatus
	model.previousSteps = []AuthStep{AuthStepMenu}

	// Test Enter key
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, _ := model.handleStatusStep(msg)
	m, ok := newModel.(*AuthWizardModel)
	require.True(t, ok)
	assert.Equal(t, AuthStepMenu, m.step)

	// Reset and test Esc key
	model.step = AuthStepStatus
	model.previousSteps = []AuthStep{AuthStepMenu}
	msg = tea.KeyMsg{Type: tea.KeyEsc}
	newModel, _ = model.handleStatusStep(msg)
	m, ok = newModel.(*AuthWizardModel)
	require.True(t, ok)
	assert.Equal(t, AuthStepMenu, m.step)
}

func TestAuthWizardModel_RenderBaseURLInput(t *testing.T) {
	config := DefaultAuthWizardConfig()
	model := NewAuthWizardModel(config)
	model.width = 80

	view := model.renderBaseURLInput()
	assert.Contains(t, view, "Base URL (Optional)")
	assert.Contains(t, view, "self-hosted")
	assert.Contains(t, view, "proxy")
}

func TestAuthWizardModel_RenderAuthMethodSelection(t *testing.T) {
	config := DefaultAuthWizardConfig()
	model := NewAuthWizardModel(config)
	model.selectedProvider = auth.ProviderInfo{
		Name:        "test-provider",
		DisplayName: "Test Provider",
		AuthMethods: []auth.AuthMethod{
			auth.AuthMethodAPIKey,
			auth.AuthMethodOAuth,
			auth.AuthMethodNone,
		},
	}
	model.width = 80

	view := model.renderAuthMethodSelection()
	assert.Contains(t, view, "Select Authentication Method")
	assert.Contains(t, view, "Test Provider")
	assert.Contains(t, view, "API Key")
	assert.Contains(t, view, "OAuth 2.0")
	assert.Contains(t, view, "None (Local Provider)")
}

func TestAuthWizardModel_GoBackClearsState(t *testing.T) {
	config := DefaultAuthWizardConfig()
	model := NewAuthWizardModel(config)

	// Test API key state clearing
	model.pushStep(AuthStepAPIKey)
	model.apiKey = "test-key"
	model.apiKeyInput.SetValue("test-key")
	model.goBack()
	assert.Equal(t, "", model.apiKey)
	assert.Equal(t, "", model.apiKeyInput.Value())

	// Test model state clearing
	model.pushStep(AuthStepModel)
	model.model = "test-model"
	model.modelIndex = 5
	model.goBack()
	assert.Equal(t, "", model.model)
	assert.Equal(t, 0, model.modelIndex)

	// Test base URL state clearing
	model.pushStep(AuthStepBaseURL)
	model.baseURL = "https://test.com"
	model.baseURLInput.SetValue("https://test.com")
	model.goBack()
	assert.Equal(t, "", model.baseURL)
	assert.Equal(t, "", model.baseURLInput.Value())
}

func TestAuthWizardModel_WindowSizeMsg(t *testing.T) {
	config := DefaultAuthWizardConfig()
	model := NewAuthWizardModel(config)

	msg := tea.WindowSizeMsg{Width: 100, Height: 30}
	newModel, _ := model.Update(msg)
	m, ok := newModel.(*AuthWizardModel)
	require.True(t, ok)
	assert.Equal(t, 100, m.width)
	assert.Equal(t, 30, m.height)
}

func TestAuthWizardModel_SpinnerTickDuringValidation(t *testing.T) {
	config := DefaultAuthWizardConfig()
	model := NewAuthWizardModel(config)
	model.validating = true

	// Simulate a spinner tick
	msg := spinner.TickMsg{}
	newModel, _ := model.Update(msg)
	m, ok := newModel.(*AuthWizardModel)
	require.True(t, ok)
	// The spinner should still be validating
	assert.True(t, m.validating)
}

func TestAuthWizardModel_ValidationResultMsg_Success(t *testing.T) {
	config := DefaultAuthWizardConfig()
	model := NewAuthWizardModel(config)
	model.validating = true
	model.step = AuthStepValidating

	msg := validationResultMsg{err: nil}
	newModel, _ := model.Update(msg)
	m, ok := newModel.(*AuthWizardModel)
	require.True(t, ok)
	assert.False(t, m.validating)
	assert.True(t, m.result.Validated)
	assert.Equal(t, AuthStepSuccess, m.step)
}

func TestAuthWizardModel_ValidationResultMsg_Error(t *testing.T) {
	config := DefaultAuthWizardConfig()
	model := NewAuthWizardModel(config)
	model.validating = true
	model.step = AuthStepValidating

	testErr := errors.New("test validation error")
	msg := validationResultMsg{err: testErr}
	newModel, _ := model.Update(msg)
	m, ok := newModel.(*AuthWizardModel)
	require.True(t, ok)
	assert.False(t, m.validating)
	assert.Equal(t, testErr, m.validationErr)
	assert.Contains(t, m.errorMessage, "test validation error")
	assert.Equal(t, AuthStepError, m.step)
}

func TestAuthWizardModel_AuthStatusLoadedMsg(t *testing.T) {
	config := DefaultAuthWizardConfig()
	model := NewAuthWizardModel(config)

	status := AuthStatusInfo{
		Provider:        "anthropic",
		IsAuthenticated: true,
		Model:           "claude-3",
	}
	msg := authStatusLoadedMsg{status: status}
	newModel, _ := model.Update(msg)
	m, ok := newModel.(*AuthWizardModel)
	require.True(t, ok)
	assert.Equal(t, "anthropic", m.authStatus.Provider)
	assert.True(t, m.authStatus.IsAuthenticated)
	assert.Equal(t, "claude-3", m.authStatus.Model)
}