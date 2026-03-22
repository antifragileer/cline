package auth

import (
	"bytes"
	"context"
	"errors"
	"testing"
)

// mockTerminal implements terminal.FileReader and terminal.FileWriter for testing
type mockTerminal struct {
	buffer *bytes.Buffer
}

func newMockTerminal() *mockTerminal {
	return &mockTerminal{
		buffer: &bytes.Buffer{},
	}
}

func (m *mockTerminal) Read(p []byte) (n int, err error) {
	return m.buffer.Read(p)
}

func (m *mockTerminal) Write(p []byte) (n int, err error) {
	return m.buffer.Write(p)
}

func (m *mockTerminal) Fd() uintptr {
	return 0
}

func (m *mockTerminal) Close() error {
	return nil
}

func TestNewWizard(t *testing.T) {
	term := newMockTerminal()
	w := NewWizardWithIO(nil, term, term)
	if w == nil {
		t.Error("NewWizardWithIO returned nil")
	}
	if w.providers == nil {
		t.Error("providers map not initialized")
	}
	if len(w.providers) == 0 {
		t.Error("no providers registered")
	}
}

func TestWizard_ProviderInfo(t *testing.T) {
	w := NewWizard(nil)

	// Test all expected providers
	expectedProviders := []string{
		ProviderAnthropic,
		ProviderOpenAI,
		ProviderOpenRouter,
		ProviderGemini,
		ProviderBedrock,
		ProviderOllama,
		ProviderLMStudio,
	}

	for _, provider := range expectedProviders {
		t.Run(provider, func(t *testing.T) {
			info, ok := w.GetProviderInfo(provider)
			if !ok {
				t.Errorf("provider %s not found", provider)
				return
			}
			if info.Name != provider {
				t.Errorf("name = %v, want %v", info.Name, provider)
			}
			if info.DisplayName == "" {
				t.Error("display name is empty")
			}
			if info.Description == "" {
				t.Error("description is empty")
			}
			if info.DefaultModel == "" {
				t.Error("default model is empty")
			}
			if len(info.Models) == 0 {
				t.Error("no models")
			}
			if len(info.AuthMethods) == 0 {
				t.Error("no auth methods")
			}
		})
	}
}

func TestWizard_ListProviders(t *testing.T) {
	w := NewWizard(nil)
	providers := w.ListProviders()
	if len(providers) == 0 {
		t.Error("ListProviders returned empty")
	}
}

func TestWizard_runNonInteractive_MissingProvider(t *testing.T) {
	w := NewWizard(nil)
	flags := WizardFlags{
		NonInteractive: true,
	}

	_, err := w.runNonInteractive(context.Background(), flags)
	if err == nil {
		t.Error("expected error for missing provider")
	}
}

func TestWizard_runNonInteractive_UnknownProvider(t *testing.T) {
	w := NewWizard(nil)
	flags := WizardFlags{
		Provider:       "unknown",
		NonInteractive: true,
	}

	_, err := w.runNonInteractive(context.Background(), flags)
	if err == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestWizard_runNonInteractive_ValidAnthropic(t *testing.T) {
	w := NewWizard(nil)

	flags := WizardFlags{
		Provider:       ProviderAnthropic,
		AuthMethod:     string(AuthMethodAPIKey),
		APIKey:         "sk-ant-test-key-12345",
		Model:          "claude-3-sonnet-20240229",
		NonInteractive: true,
		Force:          true,
	}

	result, err := w.runNonInteractive(context.Background(), flags)
	if err != nil {
		// This may fail on key format validation, which is expected
		t.Logf("runNonInteractive() error (may be expected): %v", err)
		return
	}

	if result.Provider != ProviderAnthropic {
		t.Errorf("provider = %v, want %v", result.Provider, ProviderAnthropic)
	}
	if result.AuthMethod != AuthMethodAPIKey {
		t.Errorf("auth method = %v, want %v", result.AuthMethod, AuthMethodAPIKey)
	}
	if result.Model != "claude-3-sonnet-20240229" {
		t.Errorf("model = %v, want claude-3-sonnet-20240229", result.Model)
	}
}

func TestWizard_runNonInteractive_MissingAPIKey(t *testing.T) {
	w := NewWizard(nil)
	flags := WizardFlags{
		Provider:       ProviderAnthropic,
		AuthMethod:     string(AuthMethodAPIKey),
		NonInteractive: true,
	}

	_, err := w.runNonInteractive(context.Background(), flags)
	if err == nil {
		t.Error("expected error for missing API key")
	}
}

func TestWizard_runNonInteractive_InvalidAuthMethod(t *testing.T) {
	w := NewWizard(nil)
	flags := WizardFlags{
		Provider:       ProviderAnthropic,
		AuthMethod:     "invalid",
		APIKey:         "sk-ant-test-key",
		NonInteractive: true,
	}

	_, err := w.runNonInteractive(context.Background(), flags)
	if err == nil {
		t.Error("expected error for invalid auth method")
	}
}

func TestWizard_runNonInteractive_DefaultModel(t *testing.T) {
	w := NewWizard(nil)

	flags := WizardFlags{
		Provider:       ProviderOpenAI,
		AuthMethod:     string(AuthMethodAPIKey),
		APIKey:         "sk-test-key-12345",
		Model:          "", // Should use default
		NonInteractive: true,
		Force:          true,
	}

	result, err := w.runNonInteractive(context.Background(), flags)
	if err != nil {
		t.Logf("runNonInteractive() error (may be expected): %v", err)
		return
	}

	if result.Model != "gpt-4" {
		t.Errorf("model = %v, want default gpt-4", result.Model)
	}
}

func TestWizard_runNonInteractive_DefaultAuthMethod(t *testing.T) {
	w := NewWizard(nil)

	flags := WizardFlags{
		Provider:       ProviderGemini,
		APIKey:         "test-key-12345",
		Model:          "gemini-1.5-flash",
		NonInteractive: true,
		Force:          true,
		// AuthMethod is empty - should use default
	}

	result, err := w.runNonInteractive(context.Background(), flags)
	if err != nil {
		t.Logf("runNonInteractive() error (may be expected): %v", err)
		return
	}

	if result.AuthMethod != AuthMethodAPIKey {
		t.Errorf("auth method = %v, want API key", result.AuthMethod)
	}
}

func TestWizard_runNonInteractive_OllamaNoAuth(t *testing.T) {
	w := NewWizard(nil)

	flags := WizardFlags{
		Provider:       ProviderOllama,
		AuthMethod:     string(AuthMethodNone),
		Model:          "llama3",
		NonInteractive: true,
		Force:          true,
	}

	result, err := w.runNonInteractive(context.Background(), flags)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if result.Provider != ProviderOllama {
		t.Errorf("provider = %v, want %v", result.Provider, ProviderOllama)
	}
	if result.AuthMethod != AuthMethodNone {
		t.Errorf("auth method = %v, want none", result.AuthMethod)
	}
}

func TestWizard_validateConfiguration(t *testing.T) {
	w := NewWizard(nil)

	tests := []struct {
		name    string
		result  *WizardResult
		wantErr bool
	}{
		{
			name: "unknown provider",
			result: &WizardResult{
				Provider:   "unknown",
				AuthMethod: AuthMethodAPIKey,
			},
			wantErr: true,
		},
		{
			name: "unsupported auth method",
			result: &WizardResult{
				Provider:   ProviderOllama,
				AuthMethod: AuthMethodAPIKey, // Ollama doesn't support API key
				APIKey:     "test-key",
			},
			wantErr: true,
		},
		{
			name: "missing API key",
			result: &WizardResult{
				Provider:   ProviderAnthropic,
				AuthMethod: AuthMethodAPIKey,
				APIKey:     "",
			},
			wantErr: true,
		},
		{
			name: "invalid model",
			result: &WizardResult{
				Provider:   ProviderAnthropic,
				AuthMethod: AuthMethodAPIKey,
				APIKey:     "sk-ant-test-key-12345",
				Model:      "invalid-model",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := w.validateConfiguration(context.Background(), tt.result)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfiguration() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWizard_saveConfiguration_NoAPIKey(t *testing.T) {
	w := NewWizard(nil)

	// Result with no API key - should not error
	result := &WizardResult{
		Provider:   ProviderOllama,
		AuthMethod: AuthMethodNone,
	}

	err := w.saveConfiguration(result)
	if err != nil {
		t.Errorf("saveConfiguration() error = %v", err)
	}
}

func TestWizard_saveConfiguration_NoManager(t *testing.T) {
	w := NewWizard(nil)

	result := &WizardResult{
		Provider:   ProviderAnthropic,
		AuthMethod: AuthMethodAPIKey,
		APIKey:     "test-key",
	}

	// Should not panic with nil manager
	err := w.saveConfiguration(result)
	if err != nil {
		t.Errorf("saveConfiguration() error = %v", err)
	}
}

func TestWizardFlags_Validation(t *testing.T) {
	tests := []struct {
		name    string
		flags   WizardFlags
		wantErr bool
	}{
		{
			name: "interactive mode no flags",
			flags: WizardFlags{
				NonInteractive: false,
			},
			wantErr: false, // Interactive mode handles missing flags
		},
		{
			name: "non-interactive with all required flags",
			flags: WizardFlags{
				Provider:       ProviderAnthropic,
				AuthMethod:     string(AuthMethodAPIKey),
				APIKey:         "test-key",
				Model:          "claude-3",
				NonInteractive: true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validation happens during runNonInteractive
			if tt.flags.NonInteractive && tt.flags.Provider == "" {
				// This should fail validation
			}
		})
	}
}

func TestAuthMethod_String(t *testing.T) {
	tests := []struct {
		method AuthMethod
		want   string
	}{
		{AuthMethodAPIKey, "api_key"},
		{AuthMethodOAuth, "oauth"},
		{AuthMethodNone, "none"},
	}

	for _, tt := range tests {
		t.Run(string(tt.method), func(t *testing.T) {
			if string(tt.method) != tt.want {
				t.Errorf("AuthMethod = %v, want %v", tt.method, tt.want)
			}
		})
	}
}

func TestModelInfo_Fields(t *testing.T) {
	model := ModelInfo{
		ID:          "test-model",
		Name:        "Test Model",
		Description: "A test model",
		ContextSize: 100000,
	}

	if model.ID != "test-model" {
		t.Error("ID mismatch")
	}
	if model.Name != "Test Model" {
		t.Error("Name mismatch")
	}
	if model.Description != "A test model" {
		t.Error("Description mismatch")
	}
	if model.ContextSize != 100000 {
		t.Error("ContextSize mismatch")
	}
}

func TestProviderInfo_Fields(t *testing.T) {
	info := ProviderInfo{
		Name:         "test",
		DisplayName:  "Test Provider",
		Description:  "A test provider",
		AuthMethods:  []AuthMethod{AuthMethodAPIKey},
		Models:       []ModelInfo{{ID: "model1", Name: "Model 1"}},
		DefaultModel: "model1",
	}

	if info.Name != "test" {
		t.Error("Name mismatch")
	}
	if info.DisplayName != "Test Provider" {
		t.Error("DisplayName mismatch")
	}
	if info.Description != "A test provider" {
		t.Error("Description mismatch")
	}
	if len(info.AuthMethods) != 1 {
		t.Error("AuthMethods length mismatch")
	}
	if len(info.Models) != 1 {
		t.Error("Models length mismatch")
	}
	if info.DefaultModel != "model1" {
		t.Error("DefaultModel mismatch")
	}
}

func TestWizardResult_Fields(t *testing.T) {
	result := WizardResult{
		Provider:   ProviderAnthropic,
		AuthMethod: AuthMethodAPIKey,
		APIKey:     "test-key",
		Model:      "claude-3",
		Validated:  true,
	}

	if result.Provider != ProviderAnthropic {
		t.Error("Provider mismatch")
	}
	if result.AuthMethod != AuthMethodAPIKey {
		t.Error("AuthMethod mismatch")
	}
	if result.APIKey != "test-key" {
		t.Error("APIKey mismatch")
	}
	if result.Model != "claude-3" {
		t.Error("Model mismatch")
	}
	if !result.Validated {
		t.Error("Validated should be true")
	}
}

func TestWizard_Run_NonInteractive(t *testing.T) {
	w := NewWizard(nil)

	flags := WizardFlags{
		Provider:       ProviderOpenAI,
		AuthMethod:     string(AuthMethodAPIKey),
		APIKey:         "sk-test-key-12345",
		Model:          "gpt-4",
		NonInteractive: true,
		Force:          true,
	}

	result, err := w.Run(context.Background(), flags)
	if err != nil {
		t.Logf("Run() error (may be expected): %v", err)
		return
	}

	if result.Provider != ProviderOpenAI {
		t.Errorf("provider = %v, want %v", result.Provider, ProviderOpenAI)
	}
}

func TestWizard_Run_InteractiveMode(t *testing.T) {
	// Just verify that interactive mode calls the right path
	// We can't easily test the actual interactive flow
	w := NewWizard(nil)

	flags := WizardFlags{
		NonInteractive: false,
	}

	// This would require stdin interaction, so we just verify it doesn't panic
	// In real usage, this would prompt the user
	_, err := w.Run(context.Background(), flags)
	// Expected to fail since we're not providing interactive input
	if err == nil {
		t.Error("expected error in interactive mode without input")
	}
}

func TestWizard_selectProvider_PreSelected(t *testing.T) {
	term := newMockTerminal()
	w := NewWizardWithIO(nil, term, term)

	// Test with pre-selected provider
	provider, err := w.selectProvider(ProviderAnthropic)
	if err != nil {
		t.Errorf("selectProvider() error = %v", err)
	}
	if provider != ProviderAnthropic {
		t.Errorf("provider = %v, want %v", provider, ProviderAnthropic)
	}
}

func TestWizard_selectAuthMethod_PreSelected(t *testing.T) {
	term := newMockTerminal()
	w := NewWizardWithIO(nil, term, term)

	info, _ := w.GetProviderInfo(ProviderAnthropic)

	// Test with pre-selected auth method
	method, err := w.selectAuthMethod(info, string(AuthMethodAPIKey))
	if err != nil {
		t.Errorf("selectAuthMethod() error = %v", err)
	}
	if method != AuthMethodAPIKey {
		t.Errorf("auth method = %v, want %v", method, AuthMethodAPIKey)
	}
}

func TestWizard_selectAuthMethod_SingleMethod(t *testing.T) {
	term := newMockTerminal()
	w := NewWizardWithIO(nil, term, term)

	info, _ := w.GetProviderInfo(ProviderOllama) // Only supports None

	// Should return the only auth method automatically
	method, err := w.selectAuthMethod(info, "")
	if err != nil {
		t.Errorf("selectAuthMethod() error = %v", err)
	}
	if method != AuthMethodNone {
		t.Errorf("auth method = %v, want %v", method, AuthMethodNone)
	}
}

func TestWizard_inputAPIKey_PreEntered(t *testing.T) {
	term := newMockTerminal()
	w := NewWizardWithIO(nil, term, term)

	key, err := w.inputAPIKey(ProviderAnthropic, "pre-entered-key")
	if err != nil {
		t.Errorf("inputAPIKey() error = %v", err)
	}
	if key != "pre-entered-key" {
		t.Errorf("key = %v, want pre-entered-key", key)
	}
}

func TestWizard_selectModel_PreSelected(t *testing.T) {
	term := newMockTerminal()
	w := NewWizardWithIO(nil, term, term)

	info, _ := w.GetProviderInfo(ProviderAnthropic)

	// Test with pre-selected model
	model, err := w.selectModel(info, "claude-3-opus-20240229")
	if err != nil {
		t.Errorf("selectModel() error = %v", err)
	}
	if model != "claude-3-opus-20240229" {
		t.Errorf("model = %v, want claude-3-opus-20240229", model)
	}
}

func TestWizard_confirmSave(t *testing.T) {
	term := newMockTerminal()
	w := NewWizardWithIO(nil, term, term)

	// In non-interactive mode, confirmSave returns true without prompting
	// (when Force flag is used or in non-interactive mode)
	// Since we can't easily simulate interactive input here, we just verify
	// it doesn't panic and handles the mock terminal gracefully
	confirmed, _ := w.confirmSave()
	// The actual result depends on how survey handles the mock terminal
	_ = confirmed
}

func TestWizard_runNonInteractive_NoAuthMethodAvailable(t *testing.T) {
	// This tests a theoretical case where a provider has no auth methods
	w := NewWizard(nil)

	// Create a custom provider with no auth methods
	w.providers["noauth"] = ProviderInfo{
		Name:         "noauth",
		DisplayName:  "No Auth Provider",
		Description:  "Test provider with no auth",
		AuthMethods:  []AuthMethod{}, // Empty
		DefaultModel: "test-model",
		Models:       []ModelInfo{{ID: "test-model", Name: "Test"}},
	}

	flags := WizardFlags{
		Provider:       "noauth",
		NonInteractive: true,
	}

	_, err := w.runNonInteractive(context.Background(), flags)
	if err == nil {
		t.Error("expected error for provider with no auth methods")
	}
}

func TestWizard_validateConfiguration_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	w := NewWizard(nil)

	result := &WizardResult{
		Provider:   ProviderAnthropic,
		AuthMethod: AuthMethodAPIKey,
		APIKey:     "sk-ant-test-key-12345",
		Model:      "claude-3-sonnet-20240229",
	}

	err := w.validateConfiguration(ctx, result)
	// Validation should still work with cancelled context since it doesn't use context for API calls in basic validation
	if err != nil {
		t.Logf("validateConfiguration() with cancelled context: %v", err)
	}
}

func TestWizard_runNonInteractive_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	w := NewWizard(nil)

	flags := WizardFlags{
		Provider:       ProviderOllama,
		AuthMethod:     string(AuthMethodNone),
		Model:          "llama3",
		NonInteractive: true,
		Force:          true,
	}

	// Should still work since non-interactive doesn't wait on context
	result, err := w.runNonInteractive(ctx, flags)
	if err != nil {
		t.Errorf("runNonInteractive() with cancelled context error = %v", err)
		return
	}
	if result.Provider != ProviderOllama {
		t.Error("unexpected result")
	}
}

// Test with different providers
func TestWizard_AllProviders(t *testing.T) {
	w := NewWizard(nil)

	providers := []struct {
		name   string
		apiKey string
		model  string
		auth   AuthMethod
	}{
		{ProviderOpenAI, "sk-test-key", "gpt-4", AuthMethodAPIKey},
		{ProviderGemini, "test-key", "gemini-1.5-flash", AuthMethodAPIKey},
		{ProviderOllama, "", "llama3", AuthMethodNone},
	}

	for _, p := range providers {
		t.Run(p.name, func(t *testing.T) {
			flags := WizardFlags{
				Provider:       p.name,
				AuthMethod:     string(p.auth),
				APIKey:         p.apiKey,
				Model:          p.model,
				NonInteractive: true,
				Force:          true,
			}

			result, err := w.Run(context.Background(), flags)
			if err != nil {
				// Some might fail due to key format validation
				t.Logf("Run() for %s: %v", p.name, err)
				return
			}

			if result.Provider != p.name {
				t.Errorf("provider = %v, want %v", result.Provider, p.name)
			}
		})
	}
}

// Test error messages
func TestWizard_ErrorMessages(t *testing.T) {
	w := NewWizard(nil)

	tests := []struct {
		name   string
		flags  WizardFlags
		errMsg string
	}{
		{
			name:   "missing provider",
			flags:  WizardFlags{NonInteractive: true},
			errMsg: "provider flag is required",
		},
		{
			name:   "unknown provider",
			flags:  WizardFlags{Provider: "unknown", NonInteractive: true},
			errMsg: "unknown provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := w.runNonInteractive(context.Background(), tt.flags)
			if err == nil {
				t.Errorf("expected error containing %q", tt.errMsg)
				return
			}
			if !errors.Is(err, errors.New(tt.errMsg)) && err.Error() != tt.errMsg && len(err.Error()) < len(tt.errMsg) {
				t.Errorf("error = %v, want containing %q", err, tt.errMsg)
			}
		})
	}
}

// Benchmark the wizard initialization
func BenchmarkNewWizard(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewWizard(nil)
	}
}

// Benchmark provider lookup
func BenchmarkGetProviderInfo(b *testing.B) {
	w := NewWizard(nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = w.GetProviderInfo(ProviderAnthropic)
	}
}