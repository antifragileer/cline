// Package api provides API provider implementations for the Cline CLI.
package api

import (
	"context"
	"testing"
	"time"
)

func TestOAuthToken_IsExpired(t *testing.T) {
	tests := []struct {
		name     string
		expiry   time.Time
		expected bool
	}{
		{
			name:     "token expired",
			expiry:   time.Now().Add(-time.Hour),
			expected: true,
		},
		{
			name:     "token expiring soon (less than 1 minute)",
			expiry:   time.Now().Add(30 * time.Second),
			expected: true,
		},
		{
			name:     "token valid",
			expiry:   time.Now().Add(time.Hour),
			expected: false,
		},
		{
			name:     "token with zero expiry",
			expiry:   time.Time{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := &OAuthToken{
				AccessToken: "test-token",
				Expiry:      tt.expiry,
			}

			if got := token.IsExpired(); got != tt.expected {
				t.Errorf("IsExpired() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestOAuthToken_String(t *testing.T) {
	t.Run("with token type", func(t *testing.T) {
		token := &OAuthToken{
			AccessToken: "access-token",
			TokenType:   "Bearer",
		}
		expected := "Bearer access-token"
		if got := token.String(); got != expected {
			t.Errorf("String() = %v, want %v", got, expected)
		}
	})

	t.Run("without token type", func(t *testing.T) {
		token := &OAuthToken{
			AccessToken: "access-token",
			TokenType:   "",
		}
		expected := "Bearer access-token"
		if got := token.String(); got != expected {
			t.Errorf("String() = %v, want %v", got, expected)
		}
	})
}

func TestBaseOAuthProvider(t *testing.T) {
	t.Run("Set and Get OAuthToken", func(t *testing.T) {
		provider := &BaseOAuthProvider{}

		token := &OAuthToken{
			AccessToken: "test-token",
			TokenType:   "Bearer",
		}

		provider.SetOAuthToken(token)

		got := provider.GetOAuthToken()
		if got == nil {
			t.Fatal("GetOAuthToken() returned nil")
		}
		if got.AccessToken != "test-token" {
			t.Errorf("AccessToken = %v, want test-token", got.AccessToken)
		}
	})

	t.Run("IsOAuthAuthenticated with no token", func(t *testing.T) {
		provider := &BaseOAuthProvider{}
		if provider.IsOAuthAuthenticated() {
			t.Error("IsOAuthAuthenticated() should return false with no token")
		}
	})

	t.Run("IsOAuthAuthenticated with valid token", func(t *testing.T) {
		provider := &BaseOAuthProvider{}
		token := &OAuthToken{
			AccessToken: "test-token",
			Expiry:      time.Now().Add(time.Hour),
		}
		provider.SetOAuthToken(token)

		if !provider.IsOAuthAuthenticated() {
			t.Error("IsOAuthAuthenticated() should return true with valid token")
		}
	})

	t.Run("IsOAuthAuthenticated with expired token", func(t *testing.T) {
		provider := &BaseOAuthProvider{}
		token := &OAuthToken{
			AccessToken: "test-token",
			Expiry:      time.Now().Add(-time.Hour),
		}
		provider.SetOAuthToken(token)

		if provider.IsOAuthAuthenticated() {
			t.Error("IsOAuthAuthenticated() should return false with expired token")
		}
	})

	t.Run("NeedsTokenRefresh with no token", func(t *testing.T) {
		provider := &BaseOAuthProvider{}
		if provider.NeedsTokenRefresh() {
			t.Error("NeedsTokenRefresh() should return false with no token")
		}
	})

	t.Run("NeedsTokenRefresh with token expiring soon", func(t *testing.T) {
		provider := &BaseOAuthProvider{}
		token := &OAuthToken{
			AccessToken:  "test-token",
			RefreshToken: "refresh-token",
			Expiry:       time.Now().Add(3 * time.Minute), // Less than 5 minute buffer
		}
		provider.SetOAuthToken(token)

		if !provider.NeedsTokenRefresh() {
			t.Error("NeedsTokenRefresh() should return true for token expiring within buffer")
		}
	})

	t.Run("GetAuthHeader with no token", func(t *testing.T) {
		provider := &BaseOAuthProvider{}
		if got := provider.GetAuthHeader(); got != "" {
			t.Errorf("GetAuthHeader() = %v, want empty string", got)
		}
	})

	t.Run("GetAuthHeader with token", func(t *testing.T) {
		provider := &BaseOAuthProvider{}
		token := &OAuthToken{
			AccessToken: "test-token",
			TokenType:   "Bearer",
		}
		provider.SetOAuthToken(token)

		expected := "Bearer test-token"
		if got := provider.GetAuthHeader(); got != expected {
			t.Errorf("GetAuthHeader() = %v, want %v", got, expected)
		}
	})

	t.Run("RefreshOAuthToken with no token", func(t *testing.T) {
		provider := &BaseOAuthProvider{}
		err := provider.RefreshOAuthToken(context.Background())
		if err == nil {
			t.Error("RefreshOAuthToken() should return error with no token")
		}
	})

	t.Run("RefreshOAuthToken with no refresh token", func(t *testing.T) {
		provider := &BaseOAuthProvider{}
		token := &OAuthToken{
			AccessToken: "test-token",
			// No refresh token
		}
		provider.SetOAuthToken(token)

		err := provider.RefreshOAuthToken(context.Background())
		if err == nil {
			t.Error("RefreshOAuthToken() should return error with no refresh token")
		}
	})
}

func TestOAuthProviderFactory(t *testing.T) {
	factory := NewOAuthProviderFactory()

	t.Run("Register and Get Config", func(t *testing.T) {
		config := &OAuthConfig{
			ClientID:     "test-client",
			ClientSecret: "test-secret",
			AuthURL:      "https://test.com/auth",
			TokenURL:     "https://test.com/token",
		}

		factory.RegisterConfig("test-provider", config)

		got, ok := factory.GetConfig("test-provider")
		if !ok {
			t.Error("GetConfig() should return true for registered provider")
		}
		if got.ClientID != "test-client" {
			t.Errorf("ClientID = %v, want test-client", got.ClientID)
		}
	})

	t.Run("Get Config for unregistered provider", func(t *testing.T) {
		_, ok := factory.GetConfig("unknown-provider")
		if ok {
			t.Error("GetConfig() should return false for unregistered provider")
		}
	})

	t.Run("Store and Get Token", func(t *testing.T) {
		token := &OAuthToken{
			AccessToken: "access-token",
			TokenType:   "Bearer",
		}

		factory.StoreToken("test-provider", token)

		got, ok := factory.GetToken("test-provider")
		if !ok {
			t.Error("GetToken() should return true for stored token")
		}
		if got.AccessToken != "access-token" {
			t.Errorf("AccessToken = %v, want access-token", got.AccessToken)
		}
	})

	t.Run("Clear Token", func(t *testing.T) {
		token := &OAuthToken{AccessToken: "token"}
		factory.StoreToken("clear-test", token)

		factory.ClearToken("clear-test")

		_, ok := factory.GetToken("clear-test")
		if ok {
			t.Error("GetToken() should return false after ClearToken()")
		}
	})
}

func TestProviderAuthType(t *testing.T) {
	tests := []struct {
		provider string
		supports bool
	}{
		{"openai", true},
		{"gemini", true},
		{"bedrock", true},
		{"anthropic", false},
		{"openrouter", false},
		{"ollama", false},
		{"lmstudio", false},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			if got := SupportsOAuth(tt.provider); got != tt.supports {
				t.Errorf("SupportsOAuth(%q) = %v, want %v", tt.provider, got, tt.supports)
			}
		})
	}
}

func TestGetDefaultAuthType(t *testing.T) {
	tests := []struct {
		provider string
		expected ProviderAuthType
	}{
		{"anthropic", AuthTypeAPIKey},
		{"openai", AuthTypeAPIKey},
		{"ollama", AuthTypeNone},
		{"lmstudio", AuthTypeNone},
		{"unknown", AuthTypeAPIKey}, // default fallback
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			if got := GetDefaultAuthType(tt.provider); got != tt.expected {
				t.Errorf("GetDefaultAuthType(%q) = %v, want %v", tt.provider, got, tt.expected)
			}
		})
	}
}

func TestGetProviderAuthInfo(t *testing.T) {
	info := GetProviderAuthInfo()

	// Check that all expected providers are present
	expectedProviders := map[string]bool{
		"anthropic":  false,
		"openai":     false,
		"openrouter": false,
		"gemini":     false,
		"bedrock":    false,
		"ollama":     false,
		"lmstudio":   false,
	}

	for _, p := range info {
		if _, ok := expectedProviders[p.Provider]; ok {
			expectedProviders[p.Provider] = true
		}
	}

	for provider, found := range expectedProviders {
		if !found {
			t.Errorf("Provider %s not found in GetProviderAuthInfo()", provider)
		}
	}
}

func TestNewOpenAIOAuthProvider(t *testing.T) {
	config := OpenAIConfig{
		APIKey: "test-key",
	}
	oauthConfig := &OAuthConfig{
		ClientID: "oauth-client",
	}

	provider, err := NewOpenAIOAuthProvider(config, oauthConfig)
	if err != nil {
		t.Fatalf("NewOpenAIOAuthProvider() error = %v", err)
	}

	if provider == nil {
		t.Fatal("NewOpenAIOAuthProvider() returned nil")
	}

	// Check that OAuth config was set
	if provider.oauthConfig == nil {
		t.Error("OAuth config should be set")
	}
	if provider.oauthConfig.ClientID != "oauth-client" {
		t.Errorf("ClientID = %v, want oauth-client", provider.oauthConfig.ClientID)
	}
}

func TestNewOpenAIOAuthProvider_InvalidConfig(t *testing.T) {
	config := OpenAIConfig{
		APIKey: "", // Invalid - no API key
	}
	oauthConfig := &OAuthConfig{}

	_, err := NewOpenAIOAuthProvider(config, oauthConfig)
	if err == nil {
		t.Error("NewOpenAIOAuthProvider() should return error with invalid config")
	}
}

func TestNewGeminiOAuthProvider(t *testing.T) {
	config := GeminiConfig{
		APIKey: "test-key",
	}
	oauthConfig := &OAuthConfig{
		ClientID: "oauth-client",
	}

	provider, err := NewGeminiOAuthProvider(config, oauthConfig)
	if err != nil {
		t.Fatalf("NewGeminiOAuthProvider() error = %v", err)
	}

	if provider == nil {
		t.Fatal("NewGeminiOAuthProvider() returned nil")
	}

	// Check that OAuth config was set
	if provider.oauthConfig == nil {
		t.Error("OAuth config should be set")
	}
}

func TestNewGeminiOAuthProvider_InvalidConfig(t *testing.T) {
	config := GeminiConfig{
		APIKey: "", // Invalid - no API key
	}
	oauthConfig := &OAuthConfig{}

	_, err := NewGeminiOAuthProvider(config, oauthConfig)
	if err == nil {
		t.Error("NewGeminiOAuthProvider() should return error with invalid config")
	}
}
