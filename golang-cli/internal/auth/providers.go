// Package auth provides OAuth 2.0 authentication flows with PKCE support
// for the Cline CLI.
package auth

import (
	"fmt"
	"os"
	"time"
)

// OAuthProviderConfig holds OAuth configuration for a specific provider
type OAuthProviderConfig struct {
	// Provider ID
	Provider string
	// Display name for the provider
	DisplayName string
	// OAuth endpoints
	AuthURL  string
	TokenURL string
	// OAuth client credentials
	ClientID     string
	ClientSecret string
	// Scopes to request
	Scopes []string
	// Additional authorization parameters
	AdditionalParams map[string]string
	// Default timeout for the OAuth flow
	Timeout time.Duration
}

// Predefined OAuth provider configurations
var (
	// GitHubOAuthConfig is the OAuth configuration for GitHub
	GitHubOAuthConfig = OAuthProviderConfig{
		Provider:    "github",
		DisplayName: "GitHub",
		AuthURL:     "https://github.com/login/oauth/authorize",
		TokenURL:    "https://github.com/login/oauth/access_token",
		Scopes:      []string{"read:user", "user:email"},
		Timeout:     5 * time.Minute,
	}

	// GoogleOAuthConfig is the OAuth configuration for Google
	GoogleOAuthConfig = OAuthProviderConfig{
		Provider:    "google",
		DisplayName: "Google",
		AuthURL:     "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:    "https://oauth2.googleapis.com/token",
		Scopes: []string{
			"openid",
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Timeout: 5 * time.Minute,
	}

	// ClineOAuthConfig is the OAuth configuration for Cline authentication
	ClineOAuthConfig = OAuthProviderConfig{
		Provider:    "cline",
		DisplayName: "Cline",
		AuthURL:     "https://api.cline.bot/v1/auth/authorize",
		TokenURL:    "https://api.cline.bot/v1/auth/token",
		Scopes:      []string{"profile", "api"},
		Timeout:     5 * time.Minute,
	}

	// OpenAICodexOAuthConfig is the OAuth configuration for OpenAI Codex
	OpenAICodexOAuthConfig = OAuthProviderConfig{
		Provider:    "openai-codex",
		DisplayName: "OpenAI Codex",
		AuthURL:     "https://auth.openai.com/authorize",
		TokenURL:    "https://auth.openai.com/token",
		Scopes:      []string{"codex", "profile"},
		AdditionalParams: map[string]string{
			"audience": "https://api.openai.com/v1",
		},
		Timeout: 5 * time.Minute,
	}
)

// ProviderRegistry holds OAuth configurations for all supported providers
type ProviderRegistry struct {
	providers map[string]OAuthProviderConfig
}

// NewProviderRegistry creates a new provider registry with default configurations
func NewProviderRegistry() *ProviderRegistry {
	registry := &ProviderRegistry{
		providers: make(map[string]OAuthProviderConfig),
	}

	// Register default providers
	registry.Register(GitHubOAuthConfig)
	registry.Register(GoogleOAuthConfig)
	registry.Register(ClineOAuthConfig)
	registry.Register(OpenAICodexOAuthConfig)

	return registry
}

// Register adds or updates a provider configuration
func (r *ProviderRegistry) Register(config OAuthProviderConfig) {
	r.providers[config.Provider] = config
}

// Get retrieves a provider configuration by ID
func (r *ProviderRegistry) Get(provider string) (OAuthProviderConfig, error) {
	config, ok := r.providers[provider]
	if !ok {
		return OAuthProviderConfig{}, fmt.Errorf("unknown OAuth provider: %s", provider)
	}
	return config, nil
}

// GetAll returns all registered provider configurations
func (r *ProviderRegistry) GetAll() []OAuthProviderConfig {
	configs := make([]OAuthProviderConfig, 0, len(r.providers))
	for _, config := range r.providers {
		configs = append(configs, config)
	}
	return configs
}

// GetAvailableProviders returns a list of available provider IDs
func (r *ProviderRegistry) GetAvailableProviders() []string {
	providers := make([]string, 0, len(r.providers))
	for provider := range r.providers {
		providers = append(providers, provider)
	}
	return providers
}

// IsRegistered checks if a provider is registered
func (r *ProviderRegistry) IsRegistered(provider string) bool {
	_, ok := r.providers[provider]
	return ok
}

// Unregister removes a provider from the registry
func (r *ProviderRegistry) Unregister(provider string) {
	delete(r.providers, provider)
}

// CreateFlow creates an OAuth flow for the specified provider
func (r *ProviderRegistry) CreateFlow(provider, clientID, clientSecret string) (*Flow, error) {
	config, err := r.Get(provider)
	if err != nil {
		return nil, err
	}

	flowConfig := &Config{
		ClientID:         clientID,
		ClientSecret:     clientSecret,
		AuthURL:          config.AuthURL,
		TokenURL:         config.TokenURL,
		Scopes:           config.Scopes,
		Timeout:          config.Timeout,
		AdditionalParams: config.AdditionalParams,
	}

	return NewFlow(flowConfig)
}

// CreateFlowWithCredentials creates an OAuth flow using credentials from environment or config
func (r *ProviderRegistry) CreateFlowWithCredentials(provider string) (*Flow, error) {
	config, err := r.Get(provider)
	if err != nil {
		return nil, err
	}

	// Get credentials from environment variables
	clientID := getEnvVar(provider, "CLIENT_ID")
	clientSecret := getEnvVar(provider, "CLIENT_SECRET")

	if clientID == "" {
		return nil, fmt.Errorf("OAuth client ID not found for provider %s. Set %s_CLIENT_ID environment variable", provider, providerEnvPrefix(provider))
	}

	flowConfig := &Config{
		ClientID:         clientID,
		ClientSecret:     clientSecret,
		AuthURL:          config.AuthURL,
		TokenURL:         config.TokenURL,
		Scopes:           config.Scopes,
		Timeout:          config.Timeout,
		AdditionalParams: config.AdditionalParams,
	}

	return NewFlow(flowConfig)
}

// Helper functions for environment variables
func providerEnvPrefix(provider string) string {
	switch provider {
	case "github":
		return "GITHUB"
	case "google":
		return "GOOGLE"
	case "cline":
		return "CLINE"
	case "openai-codex":
		return "OPENAI_CODEX"
	default:
		return provider
	}
}

func getEnvVar(provider, suffix string) string {
	prefix := providerEnvPrefix(provider)
	envVar := prefix + "_" + suffix
	return os.Getenv(envVar)
}

// OAuthManager manages OAuth authentication for multiple providers
type OAuthManager struct {
	registry     *ProviderRegistry
	tokenStorage TokenStorage
	flows        map[string]*Flow
}

// NewOAuthManager creates a new OAuth manager
func NewOAuthManager(tokenStorage TokenStorage) *OAuthManager {
	return &OAuthManager{
		registry:     NewProviderRegistry(),
		tokenStorage: tokenStorage,
		flows:        make(map[string]*Flow),
	}
}

// GetRegistry returns the provider registry
func (m *OAuthManager) GetRegistry() *ProviderRegistry {
	return m.registry
}

// Authenticate performs OAuth authentication for a provider
func (m *OAuthManager) Authenticate(provider string, openBrowser bool) (*Token, error) {
	flow, err := m.registry.CreateFlowWithCredentials(provider)
	if err != nil {
		return nil, fmt.Errorf("failed to create OAuth flow: %w", err)
	}

	// Store the flow for later reference
	m.flows[provider] = flow

	// Execute the OAuth flow
	token, err := flow.Execute(openBrowser)
	if err != nil {
		return nil, fmt.Errorf("OAuth authentication failed: %w", err)
	}

	// Store the token if storage is available
	if m.tokenStorage != nil {
		if err := m.tokenStorage.Save(token); err != nil {
			// Log but don't fail - token is still returned to caller
			fmt.Printf("Warning: failed to save token: %v\n", err)
		}
	}

	return token, nil
}

// GetToken retrieves a stored token
func (m *OAuthManager) GetToken() (*Token, error) {
	if m.tokenStorage == nil {
		return nil, fmt.Errorf("no token storage configured")
	}
	return m.tokenStorage.Load()
}

// RefreshToken refreshes the access token
func (m *OAuthManager) RefreshToken(provider, refreshToken string) (*Token, error) {
	flow, ok := m.flows[provider]
	if !ok {
		// Create a new flow for refresh
		newFlow, err := m.registry.CreateFlowWithCredentials(provider)
		if err != nil {
			return nil, err
		}
		flow = newFlow
		m.flows[provider] = flow
	}

	return flow.RefreshToken(refreshToken)
}

// Logout clears the stored token
func (m *OAuthManager) Logout() error {
	if m.tokenStorage == nil {
		return fmt.Errorf("no token storage configured")
	}
	return m.tokenStorage.Delete()
}

// Stop stops all active OAuth flows
func (m *OAuthManager) Stop() {
	for provider, flow := range m.flows {
		if err := flow.Stop(); err != nil {
			fmt.Printf("Warning: failed to stop flow for %s: %v\n", provider, err)
		}
	}
	m.flows = make(map[string]*Flow)
}

// IsAuthenticated checks if a valid token exists
func (m *OAuthManager) IsAuthenticated() (bool, error) {
	if m.tokenStorage == nil {
		return false, nil
	}

	token, err := m.tokenStorage.Load()
	if err != nil {
		return false, err
	}

	return !token.IsExpired(), nil
}