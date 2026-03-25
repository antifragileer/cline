// Package auth provides OAuth 2.0 authentication flows with PKCE support
// for the Cline CLI.
package auth

import (
	"testing"
	"time"
)

func TestNewProviderRegistry(t *testing.T) {
	registry := NewProviderRegistry()

	// Check that all default providers are registered
	expectedProviders := []string{"github", "google", "cline", "openai-codex"}
	for _, provider := range expectedProviders {
		if !registry.IsRegistered(provider) {
			t.Errorf("Provider %s should be registered", provider)
		}
	}
}

func TestProviderRegistry_Get(t *testing.T) {
	registry := NewProviderRegistry()

	t.Run("existing provider", func(t *testing.T) {
		config, err := registry.Get("github")
		if err != nil {
			t.Errorf("Get(github) returned error: %v", err)
		}
		if config.Provider != "github" {
			t.Errorf("Provider = %v, want github", config.Provider)
		}
		if config.DisplayName != "GitHub" {
			t.Errorf("DisplayName = %v, want GitHub", config.DisplayName)
		}
		if config.AuthURL == "" {
			t.Error("AuthURL should not be empty")
		}
		if config.TokenURL == "" {
			t.Error("TokenURL should not be empty")
		}
	})

	t.Run("non-existent provider", func(t *testing.T) {
		_, err := registry.Get("nonexistent")
		if err == nil {
			t.Error("Get(nonexistent) should return error")
		}
	})
}

func TestProviderRegistry_Register(t *testing.T) {
	registry := NewProviderRegistry()

	newConfig := OAuthProviderConfig{
		Provider:    "custom",
		DisplayName: "Custom Provider",
		AuthURL:     "https://custom.com/auth",
		TokenURL:    "https://custom.com/token",
		Scopes:      []string{"read"},
		Timeout:     5 * time.Minute,
	}

	registry.Register(newConfig)

	if !registry.IsRegistered("custom") {
		t.Error("Custom provider should be registered")
	}

	config, err := registry.Get("custom")
	if err != nil {
		t.Errorf("Get(custom) returned error: %v", err)
	}
	if config.Provider != "custom" {
		t.Errorf("Provider = %v, want custom", config.Provider)
	}
}

func TestProviderRegistry_Unregister(t *testing.T) {
	registry := NewProviderRegistry()

	// Register a test provider
	registry.Register(OAuthProviderConfig{
		Provider: "test-unregister",
		AuthURL:  "https://test.com/auth",
		TokenURL: "https://test.com/token",
	})

	if !registry.IsRegistered("test-unregister") {
		t.Error("Provider should be registered before unregister")
	}

	registry.Unregister("test-unregister")

	if registry.IsRegistered("test-unregister") {
		t.Error("Provider should not be registered after unregister")
	}
}

func TestProviderRegistry_GetAll(t *testing.T) {
	registry := NewProviderRegistry()

	all := registry.GetAll()
	if len(all) < 4 {
		t.Errorf("GetAll() returned %d providers, expected at least 4", len(all))
	}
}

func TestProviderRegistry_GetAvailableProviders(t *testing.T) {
	registry := NewProviderRegistry()

	providers := registry.GetAvailableProviders()
	if len(providers) < 4 {
		t.Errorf("GetAvailableProviders() returned %d providers, expected at least 4", len(providers))
	}

	// Check that all returned providers are valid
	for _, provider := range providers {
		if !registry.IsRegistered(provider) {
			t.Errorf("Provider %s is returned but not registered", provider)
		}
	}
}

func TestProviderRegistry_CreateFlow(t *testing.T) {
	registry := NewProviderRegistry()

	t.Run("valid provider", func(t *testing.T) {
		flow, err := registry.CreateFlow("github", "test-client-id", "test-client-secret")
		if err != nil {
			t.Errorf("CreateFlow() returned error: %v", err)
		}
		if flow == nil {
			t.Error("CreateFlow() returned nil flow")
			return
		}

		// Check that config was set correctly
		if flow.config.ClientID != "test-client-id" {
			t.Errorf("ClientID = %v, want test-client-id", flow.config.ClientID)
		}
		if flow.config.ClientSecret != "test-client-secret" {
			t.Errorf("ClientSecret = %v, want test-client-secret", flow.config.ClientSecret)
		}
	})

	t.Run("invalid provider", func(t *testing.T) {
		_, err := registry.CreateFlow("nonexistent", "client-id", "")
		if err == nil {
			t.Error("CreateFlow() with invalid provider should return error")
		}
	})
}

func TestProviderRegistry_CreateFlowWithCredentials(t *testing.T) {
	registry := NewProviderRegistry()

	// This will fail because environment variables are not set
	// But we can verify the error message
	_, err := registry.CreateFlowWithCredentials("github")
	if err == nil {
		t.Error("CreateFlowWithCredentials() should fail without environment variables")
	}
}

func TestProviderEnvPrefix(t *testing.T) {
	tests := []struct {
		provider string
		expected string
	}{
		{"github", "GITHUB"},
		{"google", "GOOGLE"},
		{"cline", "CLINE"},
		{"openai-codex", "OPENAI_CODEX"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		result := providerEnvPrefix(tt.provider)
		if result != tt.expected {
			t.Errorf("providerEnvPrefix(%q) = %v, want %v", tt.provider, result, tt.expected)
		}
	}
}

func TestNewOAuthManager(t *testing.T) {
	manager := NewOAuthManager(nil)

	if manager == nil {
		t.Error("NewOAuthManager() returned nil")
	}

	if manager.registry == nil {
		t.Error("OAuthManager registry should not be nil")
	}

	if manager.flows == nil {
		t.Error("OAuthManager flows map should not be nil")
	}
}

func TestOAuthManager_GetRegistry(t *testing.T) {
	manager := NewOAuthManager(nil)
	registry := manager.GetRegistry()

	if registry == nil {
		t.Error("GetRegistry() returned nil")
	}

	// Verify it's a valid registry
	if !registry.IsRegistered("github") {
		t.Error("Registry should have github provider")
	}
}

func TestOAuthManager_GetToken_NoStorage(t *testing.T) {
	manager := NewOAuthManager(nil)
	_, err := manager.GetToken()
	if err == nil {
		t.Error("GetToken() should fail when no storage configured")
	}
}

func TestOAuthManager_Logout_NoStorage(t *testing.T) {
	manager := NewOAuthManager(nil)
	err := manager.Logout()
	if err == nil {
		t.Error("Logout() should fail when no storage configured")
	}
}

func TestOAuthManager_Stop(t *testing.T) {
	manager := NewOAuthManager(nil)

	// Should not panic with no flows
	manager.Stop()

	// Verify flows map is cleared
	if len(manager.flows) != 0 {
		t.Errorf("flows map should be empty after Stop(), got %d flows", len(manager.flows))
	}
}

func TestOAuthManager_IsAuthenticated_NoStorage(t *testing.T) {
	manager := NewOAuthManager(nil)
	authenticated, err := manager.IsAuthenticated()
	if err != nil {
		t.Errorf("IsAuthenticated() returned error: %v", err)
	}
	if authenticated {
		t.Error("IsAuthenticated() should return false with no storage")
	}
}

func TestPredefinedConfigs(t *testing.T) {
	t.Run("GitHubOAuthConfig", func(t *testing.T) {
		if GitHubOAuthConfig.Provider != "github" {
			t.Errorf("Provider = %v, want github", GitHubOAuthConfig.Provider)
		}
		if GitHubOAuthConfig.AuthURL == "" {
			t.Error("AuthURL should not be empty")
		}
		if GitHubOAuthConfig.TokenURL == "" {
			t.Error("TokenURL should not be empty")
		}
		if len(GitHubOAuthConfig.Scopes) == 0 {
			t.Error("Scopes should not be empty")
		}
	})

	t.Run("GoogleOAuthConfig", func(t *testing.T) {
		if GoogleOAuthConfig.Provider != "google" {
			t.Errorf("Provider = %v, want google", GoogleOAuthConfig.Provider)
		}
		if GoogleOAuthConfig.AuthURL == "" {
			t.Error("AuthURL should not be empty")
		}
		if len(GoogleOAuthConfig.Scopes) == 0 {
			t.Error("Scopes should not be empty")
		}
	})

	t.Run("ClineOAuthConfig", func(t *testing.T) {
		if ClineOAuthConfig.Provider != "cline" {
			t.Errorf("Provider = %v, want cline", ClineOAuthConfig.Provider)
		}
		if ClineOAuthConfig.AuthURL == "" {
			t.Error("AuthURL should not be empty")
		}
	})

	t.Run("OpenAICodexOAuthConfig", func(t *testing.T) {
		if OpenAICodexOAuthConfig.Provider != "openai-codex" {
			t.Errorf("Provider = %v, want openai-codex", OpenAICodexOAuthConfig.Provider)
		}
		if OpenAICodexOAuthConfig.AdditionalParams["audience"] == "" {
			t.Error("AdditionalParams should include audience")
		}
	})
}

func TestOAuthProviderConfig_Fields(t *testing.T) {
	config := OAuthProviderConfig{
		Provider:         "test",
		DisplayName:      "Test Provider",
		AuthURL:          "https://test.com/auth",
		TokenURL:         "https://test.com/token",
		ClientID:         "client-id",
		ClientSecret:     "client-secret",
		Scopes:           []string{"read", "write"},
		AdditionalParams: map[string]string{"param": "value"},
		Timeout:          10 * time.Minute,
	}

	if config.Provider != "test" {
		t.Errorf("Provider = %v, want test", config.Provider)
	}
	if config.DisplayName != "Test Provider" {
		t.Errorf("DisplayName = %v, want Test Provider", config.DisplayName)
	}
	if config.Timeout != 10*time.Minute {
		t.Errorf("Timeout = %v, want 10m", config.Timeout)
	}
}