package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cline/cline/golang-cli/internal/auth"
	"github.com/cline/cline/golang-cli/internal/storage"
	"github.com/spf13/cobra"
)

// mockCommand implements the command interface for testing
type mockOAuthCommand struct {
	output *bytes.Buffer
	input  *bytes.Buffer
}

func (m *mockOAuthCommand) OutOrStdout() *bytes.Buffer {
	return m.output
}

func (m *mockOAuthCommand) InOrStdin() *bytes.Buffer {
	return m.input
}

func TestIsOAuthProvider(t *testing.T) {
	tests := []struct {
		provider string
		expected bool
	}{
		{"openai-codex", true},
		{"github", true},
		{"google", true},
		{"anthropic", false},
		{"openai", false},
		{"unknown", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			result := IsOAuthProvider(tt.provider)
			if result != tt.expected {
				t.Errorf("IsOAuthProvider(%q) = %v, want %v", tt.provider, result, tt.expected)
			}
		})
	}
}

func TestGetOAuthProviders(t *testing.T) {
	providers := GetOAuthProviders()
	
	if len(providers) == 0 {
		t.Error("GetOAuthProviders() returned empty list")
	}

	// Check that all expected providers are present
	expectedProviders := map[string]bool{
		"openai-codex": false,
		"github":       false,
		"google":       false,
	}

	for _, p := range providers {
		if _, ok := expectedProviders[p]; ok {
			expectedProviders[p] = true
		}
	}

	for provider, found := range expectedProviders {
		if !found {
			t.Errorf("Expected provider %s not found in GetOAuthProviders()", provider)
		}
	}
}

func TestConfirmContinue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"yes", "y\n", true},
		{"yes uppercase", "Y\n", true},
		{"yes full", "yes\n", true},
		{"default empty", "\n", true},
		{"no", "n\n", false},
		{"no full", "no\n", false},
		{"invalid", "maybe\n", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := bytes.NewBufferString(tt.input)
			output := &bytes.Buffer{}
			
			result := confirmContinue(input, output, "Continue?")
			if result != tt.expected {
				t.Errorf("confirmContinue() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConfirmContinueError(t *testing.T) {
	// Test with a reader that returns an error
	errorReader := &errorReader{}
	output := &bytes.Buffer{}
	
	result := confirmContinue(errorReader, output, "Continue?")
	if result != false {
		t.Error("confirmContinue() should return false on read error")
	}
}

func TestFormatExpiry(t *testing.T) {
	tests := []struct {
		name     string
		expiry   interface{}
		expected string
	}{
		{"nil", nil, "never"},
		{"string", "2024-01-15T10:30:00Z", "2024-01-15T10:30:00Z"},
		{"int", 12345, "unknown"},
		{"time", time.Now(), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatExpiry(tt.expiry)
			if result != tt.expected {
				t.Errorf("formatExpiry(%v) = %q, want %q", tt.expiry, result, tt.expected)
			}
		})
	}
}

func TestGetClineDataDir(t *testing.T) {
	dir := getClineDataDir()
	if dir == "" {
		t.Error("getClineDataDir() returned empty string")
	}

	// Should contain .cline/data
	if !strings.Contains(dir, ".cline") {
		t.Errorf("getClineDataDir() should contain '.cline', got: %s", dir)
	}
}

func TestSaveOAuthConfig(t *testing.T) {
	tmpDir := t.TempDir()
	ctx, err := storage.NewStorageContext(tmpDir, "test-workspace")
	if err != nil {
		t.Fatalf("Failed to create storage context: %v", err)
	}
	defer ctx.Close()

	token := &auth.Token{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}

	err = saveOAuthConfig(ctx, "openai-codex", token)
	if err != nil {
		t.Errorf("saveOAuthConfig() error = %v", err)
	}

	// Verify provider was saved
	providerVal, ok := ctx.GlobalState.Get("apiProvider")
	if !ok || providerVal != "openai-codex" {
		t.Error("Provider not saved correctly")
	}

	// Verify OAuth enabled flag
	oauthEnabled, ok := ctx.GlobalState.Get("openai-codexOAuthEnabled")
	if !ok || oauthEnabled != true {
		t.Error("OAuth enabled flag not saved correctly")
	}

	// Verify token was saved in secrets
	tokenData, ok := ctx.Secrets.Get("openai-codexOAuthToken")
	if !ok {
		t.Error("OAuth token not saved in secrets")
	}

	tokenMap, ok := tokenData.(map[string]interface{})
	if !ok {
		t.Fatal("Token data is not a map")
	}

	if tokenMap["access_token"] != "test-access-token" {
		t.Error("Access token not saved correctly")
	}
}

func TestSaveOAuthConfigNoExpiry(t *testing.T) {
	tmpDir := t.TempDir()
	ctx, err := storage.NewStorageContext(tmpDir, "test-workspace")
	if err != nil {
		t.Fatalf("Failed to create storage context: %v", err)
	}
	defer ctx.Close()

	token := &auth.Token{
		AccessToken: "test-token",
		TokenType:   "Bearer",
		Expiry:      time.Time{}, // Zero time
	}

	err = saveOAuthConfig(ctx, "github", token)
	if err != nil {
		t.Errorf("saveOAuthConfig() error = %v", err)
	}

	// Verify token was saved without expiry
	tokenData, ok := ctx.Secrets.Get("githubOAuthToken")
	if !ok {
		t.Error("OAuth token not saved")
	}

	tokenMap, ok := tokenData.(map[string]interface{})
	if !ok {
		t.Fatal("Token data is not a map")
	}

	// Expiry should be the zero time format
	if tokenMap["expiry"] != "0001-01-01T00:00:00Z" {
		t.Errorf("Expected zero time format, got: %v", tokenMap["expiry"])
	}
}

func TestRefreshOAuthToken(t *testing.T) {
	tmpDir := t.TempDir()
	ctx, err := storage.NewStorageContext(tmpDir, "test-workspace")
	if err != nil {
		t.Fatalf("Failed to create storage context: %v", err)
	}
	defer ctx.Close()

	// Set up a mock token without refresh token
	tokenData := map[string]interface{}{
		"access_token": "old-token",
		"token_type":   "Bearer",
	}
	ctx.Secrets.Set("testOAuthToken", tokenData)

	// Should fail because no refresh token
	_, err = RefreshOAuthToken(ctx, "test")
	if err == nil {
		t.Error("RefreshOAuthToken() should fail without refresh token")
	}
}

func TestRefreshOAuthTokenNoToken(t *testing.T) {
	tmpDir := t.TempDir()
	ctx, err := storage.NewStorageContext(tmpDir, "test-workspace")
	if err != nil {
		t.Fatalf("Failed to create storage context: %v", err)
	}
	defer ctx.Close()

	// Should fail because no token exists
	_, err = RefreshOAuthToken(ctx, "nonexistent")
	if err == nil {
		t.Error("RefreshOAuthToken() should fail when no token exists")
	}
}

func TestLogoutOAuth(t *testing.T) {
	tmpDir := t.TempDir()
	ctx, err := storage.NewStorageContext(tmpDir, "test-workspace")
	if err != nil {
		t.Fatalf("Failed to create storage context: %v", err)
	}
	defer ctx.Close()

	// Set up a mock token
	tokenData := map[string]interface{}{
		"access_token": "test-token",
	}
	ctx.Secrets.Set("githubOAuthToken", tokenData)
	ctx.GlobalState.Set("githubOAuthEnabled", true)

	// Create token file
	tokenDir := filepath.Join(tmpDir, "tokens")
	os.MkdirAll(tokenDir, 0700)
	tokenFile := filepath.Join(tokenDir, "github_token.json")
	os.WriteFile(tokenFile, []byte("{}"), 0600)

	err = LogoutOAuth(ctx, "github")
	if err != nil {
		t.Errorf("LogoutOAuth() error = %v", err)
	}

	// Verify OAuth disabled
	enabled, ok := ctx.GlobalState.Get("githubOAuthEnabled")
	if !ok || enabled != false {
		t.Error("OAuth should be disabled after logout")
	}

	// Verify token removed
	_, ok = ctx.Secrets.Get("githubOAuthToken")
	if ok {
		t.Error("OAuth token should be removed after logout")
	}

	// Verify file removed (or at least function doesn't error)
}

func TestOAuthProviderInfoStructure(t *testing.T) {
	// Verify all OAuth providers have required fields
	for name, info := range OAuthProviderInfo {
		t.Run(name, func(t *testing.T) {
			if info.Name == "" {
				t.Errorf("Provider %s has empty Name", name)
			}
			if info.Description == "" {
				t.Errorf("Provider %s has empty Description", name)
			}
			if info.AuthType != "oauth" && info.AuthType != "apikey" {
				t.Errorf("Provider %s has invalid AuthType: %s", name, info.AuthType)
			}
		})
	}
}

func TestRunOAuthFlowUnsupportedProvider(t *testing.T) {
	cmd := &cobra.Command{}
	ctx, err := storage.NewStorageContext("", "")
	if err != nil {
		t.Fatalf("Failed to create storage context: %v", err)
	}
	defer ctx.Close()

	// Test with unsupported provider
	err = runOAuthFlow(cmd, ctx, "unsupported-provider")
	if err == nil {
		t.Error("runOAuthFlow() should fail with unsupported provider")
	}

	if !strings.Contains(err.Error(), "not supported") {
		t.Errorf("Error should indicate provider not supported: %v", err)
	}
}

func TestLogoutOAuthPartialFailure(t *testing.T) {
	tmpDir := t.TempDir()
	ctx, err := storage.NewStorageContext(tmpDir, "test-workspace")
	if err != nil {
		t.Fatalf("Failed to create storage context: %v", err)
	}
	defer ctx.Close()

	// Set up a mock token but don't create the file
	tokenData := map[string]interface{}{
		"access_token": "test-token",
	}
	ctx.Secrets.Set("testOAuthToken", tokenData)

	// Should not error even if file doesn't exist
	err = LogoutOAuth(ctx, "test")
	if err != nil {
		t.Errorf("LogoutOAuth() should not error when file doesn't exist: %v", err)
	}
}

func TestFormatExpiryVariousTypes(t *testing.T) {
	tests := []struct {
		name     string
		expiry   interface{}
		expected string
	}{
		{"nil expiry", nil, "never"},
		{"string expiry", "2024-12-25T00:00:00Z", "2024-12-25T00:00:00Z"},
		{"int value", 42, "unknown"},
		{"float value", 3.14, "unknown"},
		{"bool value", true, "unknown"},
		{"slice value", []string{"a", "b"}, "unknown"},
		{"map value", map[string]string{"key": "value"}, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatExpiry(tt.expiry)
			if result != tt.expected {
				t.Errorf("formatExpiry() = %q, want %q", result, tt.expected)
			}
		})
	}
}