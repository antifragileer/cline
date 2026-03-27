package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/cline/cline/golang-cli/internal/auth"
	"github.com/cline/cline/golang-cli/internal/storage"
	"github.com/spf13/cobra"
)

// OAuthProviderInfo holds information about OAuth-enabled providers
var OAuthProviderInfo = map[string]struct {
	Name         string
	Description  string
	RequiresOAuth bool
	AuthType     string // "oauth" or "apikey"
}{
	"openai-codex": {
		Name:         "openai-codex",
		Description:  "OpenAI Codex (OAuth)",
		RequiresOAuth: true,
		AuthType:     "oauth",
	},
	"github": {
		Name:         "github",
		Description:  "GitHub (OAuth)",
		RequiresOAuth: true,
		AuthType:     "oauth",
	},
	"google": {
		Name:         "google",
		Description:  "Google (OAuth)",
		RequiresOAuth: true,
		AuthType:     "oauth",
	},
}

// runOAuthFlow executes the OAuth authentication flow
func runOAuthFlow(cmd *cobra.Command, ctx *storage.StorageContext, provider string) error {
	// Get OAuth configuration for the provider
	registry := auth.NewProviderRegistry()
	
	// Check if provider is supported
	if !registry.IsRegistered(provider) {
		return fmt.Errorf("OAuth provider '%s' is not supported. Supported providers: %s", 
			provider, strings.Join(registry.GetAvailableProviders(), ", "))
	}

	// Get provider config
	config, err := registry.Get(provider)
	if err != nil {
		return fmt.Errorf("failed to get OAuth config: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\n🔐 OAuth Authentication: %s\n", config.DisplayName)
	fmt.Fprintln(cmd.OutOrStdout(), strings.Repeat("=", 50))
	fmt.Fprintf(cmd.OutOrStdout(), "\nThis will open your browser to authenticate with %s.\n", config.DisplayName)
	fmt.Fprintln(cmd.OutOrStdout(), "The CLI will start a local server to receive the authentication callback.")
	fmt.Fprintln(cmd.OutOrStdout())

	// Confirm with user
	if !confirmContinue(cmd.InOrStdin(), cmd.OutOrStdout(), "Continue with OAuth authentication?") {
		fmt.Fprintln(cmd.OutOrStdout(), "\nAuthentication cancelled.")
		return nil
	}

	// Create token storage
	tokenDir := filepath.Join(getClineDataDir(), "tokens")
	if err := os.MkdirAll(tokenDir, 0700); err != nil {
		return fmt.Errorf("failed to create token directory: %w", err)
	}
	
	tokenPath := filepath.Join(tokenDir, provider+"_token.json")
	tokenStorage := auth.NewJSONTokenStorage(tokenPath)

	// Create OAuth manager
	manager := auth.NewOAuthManager(tokenStorage)

	fmt.Fprintln(cmd.OutOrStdout(), "\n🌐 Opening browser for authentication...")
	fmt.Fprintln(cmd.OutOrStdout(), "Waiting for authorization... (timeout: 5 minutes)")
	fmt.Fprintln(cmd.OutOrStdout())

	// Execute OAuth flow
	token, err := manager.Authenticate(provider, true)
	if err != nil {
		return fmt.Errorf("OAuth authentication failed: %w", err)
	}

	// Save the token and update configuration
	if err := saveOAuthConfig(ctx, provider, token); err != nil {
		return fmt.Errorf("failed to save OAuth configuration: %w", err)
	}

	// Mark onboarding as complete
	if err := ctx.GlobalState.Set("welcomeViewCompleted", true); err != nil {
		return fmt.Errorf("failed to update onboarding status: %w", err)
	}

	// Display success
	fmt.Fprintln(cmd.OutOrStdout(), "\n✓ OAuth authentication successful!")
	fmt.Fprintf(cmd.OutOrStdout(), "  Provider: %s\n", provider)
	fmt.Fprintf(cmd.OutOrStdout(), "  Token expires: %s\n", formatExpiry(token.Expiry))
	
	return nil
}

// saveOAuthConfig saves OAuth configuration to storage
func saveOAuthConfig(ctx *storage.StorageContext, provider string, token *auth.Token) error {
	// Save provider setting
	if err := ctx.GlobalState.Set("apiProvider", provider); err != nil {
		return fmt.Errorf("failed to save provider: %w", err)
	}

	// Save OAuth token indicator
	if err := ctx.GlobalState.Set(provider+"OAuthEnabled", true); err != nil {
		return fmt.Errorf("failed to save OAuth status: %w", err)
	}

	// Save token expiry for reference
	if !token.Expiry.IsZero() {
		if err := ctx.GlobalState.Set(provider+"TokenExpiry", token.Expiry.Format("2006-01-02T15:04:05Z")); err != nil {
			return fmt.Errorf("failed to save token expiry: %w", err)
		}
	}

	// Store the actual token in secrets
	tokenData := map[string]interface{}{
		"access_token":  token.AccessToken,
		"refresh_token": token.RefreshToken,
		"token_type":    token.TokenType,
		"expiry":        token.Expiry.Format("2006-01-02T15:04:05Z"),
	}
	
	if err := ctx.Secrets.Set(provider+"OAuthToken", tokenData); err != nil {
		return fmt.Errorf("failed to save OAuth token: %w", err)
	}

	return nil
}

// confirmContinue prompts the user to confirm an action
func confirmContinue(input io.Reader, output io.Writer, prompt string) bool {
	fmt.Fprintf(output, "%s [Y/n]: ", prompt)
	
	reader := bufio.NewReader(input)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "" || response == "y" || response == "yes"
}

// formatExpiry formats the token expiry time for display
func formatExpiry(expiry interface{}) string {
	if expiry == nil {
		return "never"
	}
	
	// Handle different expiry formats
	switch v := expiry.(type) {
	case string:
		return v
	default:
		return "unknown"
	}
}

// getClineDataDir returns the Cline data directory
func getClineDataDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	
	// Use .cline directory in home
	return filepath.Join(homeDir, ".cline", "data")
}

// IsOAuthProvider checks if a provider uses OAuth authentication
func IsOAuthProvider(provider string) bool {
	_, ok := OAuthProviderInfo[provider]
	return ok
}

// GetOAuthProviders returns a list of OAuth-enabled providers
func GetOAuthProviders() []string {
	providers := make([]string, 0, len(OAuthProviderInfo))
	for name := range OAuthProviderInfo {
		providers = append(providers, name)
	}
	return providers
}

// RefreshOAuthToken refreshes an OAuth token for a provider
func RefreshOAuthToken(ctx *storage.StorageContext, provider string) (*auth.Token, error) {
	// Get stored token
	tokenData, ok := ctx.Secrets.Get(provider + "OAuthToken")
	if !ok {
		return nil, fmt.Errorf("no OAuth token found for provider: %s", provider)
	}

	// Parse token data
	tokenMap, ok := tokenData.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid token data format")
	}

	refreshToken, _ := tokenMap["refresh_token"].(string)
	if refreshToken == "" {
		return nil, fmt.Errorf("no refresh token available")
	}

	// Create OAuth manager
	tokenDir := filepath.Join(getClineDataDir(), "tokens")
	tokenPath := filepath.Join(tokenDir, provider+"_token.json")
	tokenStorage := auth.NewJSONTokenStorage(tokenPath)
	manager := auth.NewOAuthManager(tokenStorage)

	// Refresh token
	newToken, err := manager.RefreshToken(provider, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	// Save new token
	if err := saveOAuthConfig(ctx, provider, newToken); err != nil {
		return nil, fmt.Errorf("failed to save refreshed token: %w", err)
	}

	return newToken, nil
}

// LogoutOAuth clears OAuth authentication for a provider
func LogoutOAuth(ctx *storage.StorageContext, provider string) error {
	// Remove OAuth settings
	if err := ctx.GlobalState.Set(provider+"OAuthEnabled", false); err != nil {
		return fmt.Errorf("failed to update OAuth status: %w", err)
	}

	// Remove token
	if err := ctx.Secrets.Delete(provider + "OAuthToken"); err != nil {
		return fmt.Errorf("failed to remove OAuth token: %w", err)
	}

	// Remove token file
	tokenDir := filepath.Join(getClineDataDir(), "tokens")
	tokenPath := filepath.Join(tokenDir, provider+"_token.json")
	os.Remove(tokenPath)

	return nil
}