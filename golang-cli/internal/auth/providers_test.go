// Package auth provides OAuth 2.0 authentication flows with PKCE support
// for the Cline CLI.
package auth

import (
	"os"
	"testing"
	"time"
)

func TestNewProviderRegistry(t *testing.T) {
	registry := NewProviderRegistry()

	// Check that default providers are registered
	providers := registry.GetAvailableProviders()
	if len(providers) != 4 {
		t.Errorf("Expected 4 default providers, got %d", len(providers))
	}

	// Check that specific providers exist
	for _, provider := range []string{"github", "google", "cline", "openai-codex"} {
		if !registry.IsRegistered(provider) {
			t.Errorf("Expected %s to be registered", provider)
		}
	}
}

func TestProviderRegistry_Get(t *testing.T) {
	registry := NewProviderRegistry()

	t.Run("gets existing provider", func(t *testing.T) {
		config, err := registry.Get("github")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if config.Provider != "github" {
			t.Errorf("Expected provider 'github', got %s", config.Provider)
		}
		if config.AuthURL != "https://github.com/login/oauth/authorize" {
			t.Errorf("Unexpected auth URL: %s", config.AuthURL)
		}
	})

	t.Run("returns error for unknown provider", func(t *testing.T) {
		_, err := registry.Get("unknown")
		if err == nil {
			t.Error("Expected error for unknown provider")
		}
	})
}

func TestProviderRegistry_Register(t *testing.T) {
	registry := NewProviderRegistry()

	config := OAuthProviderConfig{
		Provider:    "custom",
		DisplayName: "Custom Provider",
		AuthURL:     "https://custom.example.com/auth",
		TokenURL:    "https://custom.example.com/token",
		Scopes:      []string{"read", "write"},
		Timeout:     5 * time.Minute,
	}

	registry.Register(config)

	if !registry.IsRegistered("custom") {
		t.Error("Expected custom provider to be registered")
	}

	retrieved, err := registry.Get("custom")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if retrieved.Provider != "custom" {
		t.Errorf("Expected provider 'custom', got %s", retrieved.Provider)
	}
}

func TestProviderRegistry_Unregister(t *testing.T) {
	registry := NewProviderRegistry()

	// Unregister a default provider
	registry.Unregister("github")

	if registry.IsRegistered("github") {
		t.Error("Expected github to be unregistered")
	}
}

func TestProviderRegistry_GetAll(t *testing.T) {
	registry := NewProviderRegistry()

	configs := registry.GetAll()
	if len(configs) != 4 {
		t.Errorf("Expected 4 configs, got %d", len(configs))
	}
}

func TestProviderRegistry_CreateFlow(t *testing.T) {
	registry := NewProviderRegistry()

	t.Run("creates flow with valid credentials", func(t *testing.T) {
		flow, err := registry.CreateFlow("github", "test-client-id", "test-secret")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if flow == nil {
			t.Error("Expected flow to be created")
		}
		if flow.config.ClientID != "test-client-id" {
			t.Errorf("Expected client ID 'test-client-id', got %s", flow.config.ClientID)
		}
	})

	t.Run("returns error for unknown provider", func(t *testing.T) {
		_, err := registry.CreateFlow("unknown", "id", "secret")
		if err == nil {
			t.Error("Expected error for unknown provider")
		}
	})
}

func TestProviderRegistry_CreateFlowWithCredentials(t *testing.T) {
	// Set environment variables for testing
	os.Setenv("GITHUB_CLIENT_ID", "env-client-id")
	os.Setenv("GITHUB_CLIENT_SECRET", "env-secret")
	defer func() {
		os.Unsetenv("GITHUB_CLIENT_ID")
		os.Unsetenv("GITHUB_CLIENT_SECRET")
	}()

	registry := NewProviderRegistry()

	t.Run("creates flow with environment credentials", func(t *testing.T) {
		flow, err := registry.CreateFlowWithCredentials("github")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if flow == nil {
			t.Error("Expected flow to be created")
		}
		if flow.config.ClientID != "env-client-id" {
			t.Errorf("Expected client ID from env, got %s", flow.config.ClientID)
		}
	})

	t.Run("returns error when credentials not found", func(t *testing.T) {
		// Unset env var temporarily
		os.Unsetenv("GITHUB_CLIENT_ID")
		defer os.Setenv("GITHUB_CLIENT_ID", "env-client-id")

		_, err := registry.CreateFlowWithCredentials("github")
		if err == nil {
			t.Error("Expected error when credentials not found")
		}
	})
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
		{"custom", "custom"},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			result := providerEnvPrefix(tt.provider)
			if result != tt.expected {
				t.Errorf("providerEnvPrefix(%s) = %s, want %s", tt.provider, result, tt.expected)
			}
		})
	}
}

func TestGetEnvVar(t *testing.T) {
	// Set test environment variable
	os.Setenv("TEST_PROVIDER_CLIENT_ID", "test-value")
	defer os.Unsetenv("TEST_PROVIDER_CLIENT_ID")

	t.Run("reads environment variable", func(t *testing.T) {
		// Use a custom prefix by modifying the providerEnvPrefix function behavior
		// This test verifies the getEnvVar function works with actual env vars
		os.Setenv("GITHUB_CLIENT_ID", "github-test-id")
		defer os.Unsetenv("GITHUB_CLIENT_ID")

		result := getEnvVar("github", "CLIENT_ID")
		if result != "github-test-id" {
			t.Errorf("getEnvVar() = %s, want %s", result, "github-test-id")
		}
	})
}

func TestNewOAuthManager(t *testing.T) {
	storage := NewJSONTokenStorage("/tmp/test-token.json")
	manager := NewOAuthManager(storage)

	if manager == nil {
		t.Error("Expected manager to be created")
	}
	if manager.registry == nil {
		t.Error("Expected registry to be initialized")
	}
	if manager.tokenStorage != storage {
		t.Error("Expected token storage to be set")
	}
}

func TestOAuthManager_GetRegistry(t *testing.T) {
	storage := NewJSONTokenStorage("/tmp/test-token.json")
	manager := NewOAuthManager(storage)

	registry := manager.GetRegistry()
	if registry == nil {
		t.Error("Expected registry to be returned")
	}
}

func TestOAuthManager_GetToken(t *testing.T) {
	t.Run("returns error when no storage", func(t *testing.T) {
		manager := NewOAuthManager(nil)
		_, err := manager.GetToken()
		if err == nil {
			t.Error("Expected error when no storage configured")
		}
	})
}

func TestOAuthManager_RefreshToken(t *testing.T) {
	t.Run("creates new flow when not cached", func(t *testing.T) {
		// Set environment variables
		os.Setenv("GITHUB_CLIENT_ID", "test-id")
		defer os.Unsetenv("GITHUB_CLIENT_ID")

		storage := NewJSONTokenStorage("/tmp/test-token.json")
		manager := NewOAuthManager(storage)

		// This will fail because there's no mock server, but it tests flow creation
		_, err := manager.RefreshToken("github", "refresh-token")
		// Expect error because oauth2Config is not initialized
		if err == nil {
			t.Error("Expected error when oauth2Config not initialized")
		}
	})
}

func TestOAuthManager_Logout(t *testing.T) {
	t.Run("returns error when no storage", func(t *testing.T) {
		manager := NewOAuthManager(nil)
		err := manager.Logout()
		if err == nil {
			t.Error("Expected error when no storage configured")
		}
	})
}

func TestOAuthManager_Stop(t *testing.T) {
	storage := NewJSONTokenStorage("/tmp/test-token.json")
	manager := NewOAuthManager(storage)

	// Stop with no flows - should not panic
	manager.Stop()

	// Verify flows map is cleared
	if len(manager.flows) != 0 {
		t.Error("Expected flows map to be cleared")
	}
}

func TestOAuthManager_IsAuthenticated(t *testing.T) {
	t.Run("returns false when no storage", func(t *testing.T) {
		manager := NewOAuthManager(nil)
		authenticated, err := manager.IsAuthenticated()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if authenticated {
			t.Error("Expected not authenticated when no storage")
		}
	})
}

func TestGitHubOAuthConfig(t *testing.T) {
	config := GitHubOAuthConfig
	if config.Provider != "github" {
		t.Errorf("Expected provider 'github', got %s", config.Provider)
	}
	if config.AuthURL != "https://github.com/login/oauth/authorize" {
		t.Errorf("Unexpected auth URL: %s", config.AuthURL)
	}
	if config.TokenURL != "https://github.com/login/oauth/access_token" {
		t.Errorf("Unexpected token URL: %s", config.TokenURL)
	}
}

func TestGoogleOAuthConfig(t *testing.T) {
	config := GoogleOAuthConfig
	if config.Provider != "google" {
		t.Errorf("Expected provider 'google', got %s", config.Provider)
	}
	if len(config.Scopes) != 3 {
		t.Errorf("Expected 3 scopes, got %d", len(config.Scopes))
	}
}

func TestClineOAuthConfig(t *testing.T) {
	config := ClineOAuthConfig
	if config.Provider != "cline" {
		t.Errorf("Expected provider 'cline', got %s", config.Provider)
	}
	if config.AuthURL != "https://api.cline.bot/v1/auth/authorize" {
		t.Errorf("Unexpected auth URL: %s", config.AuthURL)
	}
}

func TestOpenAICodexOAuthConfig(t *testing.T) {
	config := OpenAICodexOAuthConfig
	if config.Provider != "openai-codex" {
		t.Errorf("Expected provider 'openai-codex', got %s", config.Provider)
	}
	if config.AdditionalParams == nil {
		t.Error("Expected additional params")
	}
	if config.AdditionalParams["audience"] != "https://api.openai.com/v1" {
		t.Errorf("Unexpected audience: %s", config.AdditionalParams["audience"])
	}
}