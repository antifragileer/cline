// Package api provides API provider implementations for the Cline CLI.
package api

import (
	"context"
	"fmt"
	"time"
)

// OAuthToken represents an OAuth token for API authentication
type OAuthToken struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	Expiry       time.Time `json:"expiry"`
}

// IsExpired returns true if the token is expired or about to expire
func (t *OAuthToken) IsExpired() bool {
	if t.Expiry.IsZero() {
		return false
	}
	// Consider token expired 1 minute before actual expiry
	return time.Until(t.Expiry) < time.Minute
}

// String returns the token as an Authorization header value
func (t *OAuthToken) String() string {
	if t.TokenType == "" {
		return fmt.Sprintf("Bearer %s", t.AccessToken)
	}
	return fmt.Sprintf("%s %s", t.TokenType, t.AccessToken)
}

// OAuthProvider interface for providers that support OAuth authentication
type OAuthProvider interface {
	// SetOAuthToken sets the OAuth token for authentication
	SetOAuthToken(token *OAuthToken)
	// GetOAuthToken returns the current OAuth token
	GetOAuthToken() *OAuthToken
	// RefreshOAuthToken refreshes the OAuth token
	RefreshOAuthToken(ctx context.Context) error
	// IsOAuthAuthenticated returns true if the provider has a valid OAuth token
	IsOAuthAuthenticated() bool
}

// OAuthConfig holds OAuth configuration for API providers
type OAuthConfig struct {
	// ClientID is the OAuth client ID
	ClientID string
	// ClientSecret is the OAuth client secret
	ClientSecret string
	// AuthURL is the authorization endpoint
	AuthURL string
	// TokenURL is the token endpoint
	TokenURL string
	// Scopes are the OAuth scopes
	Scopes []string
	// TokenRefreshBuffer is the time buffer before expiry to refresh (default 5 minutes)
	TokenRefreshBuffer time.Duration
}

// BaseOAuthProvider provides a base implementation of OAuthProvider
type BaseOAuthProvider struct {
	oauthToken *OAuthToken
	oauthConfig *OAuthConfig
}

// SetOAuthToken sets the OAuth token
func (p *BaseOAuthProvider) SetOAuthToken(token *OAuthToken) {
	p.oauthToken = token
}

// GetOAuthToken returns the current OAuth token
func (p *BaseOAuthProvider) GetOAuthToken() *OAuthToken {
	return p.oauthToken
}

// IsOAuthAuthenticated returns true if there's a valid non-expired token
func (p *BaseOAuthProvider) IsOAuthAuthenticated() bool {
	if p.oauthToken == nil {
		return false
	}
	return !p.oauthToken.IsExpired()
}

// NeedsTokenRefresh returns true if the token needs to be refreshed
func (p *BaseOAuthProvider) NeedsTokenRefresh() bool {
	if p.oauthToken == nil || p.oauthToken.RefreshToken == "" {
		return false
	}
	
	buffer := 5 * time.Minute
	if p.oauthConfig != nil && p.oauthConfig.TokenRefreshBuffer > 0 {
		buffer = p.oauthConfig.TokenRefreshBuffer
	}
	
	return time.Until(p.oauthToken.Expiry) < buffer
}

// RefreshOAuthToken performs token refresh - should be implemented by specific providers
func (p *BaseOAuthProvider) RefreshOAuthToken(ctx context.Context) error {
	if p.oauthToken == nil || p.oauthToken.RefreshToken == "" {
		return fmt.Errorf("no refresh token available")
	}
	
	// Base implementation - specific providers should override
	return fmt.Errorf("token refresh not implemented for this provider")
}

// GetAuthHeader returns the Authorization header value
func (p *BaseOAuthProvider) GetAuthHeader() string {
	if p.oauthToken == nil {
		return ""
	}
	return p.oauthToken.String()
}

// OAuthProviderFactory creates OAuth-enabled providers
type OAuthProviderFactory struct {
	configs map[string]*OAuthConfig
	tokens  map[string]*OAuthToken
}

// NewOAuthProviderFactory creates a new OAuth provider factory
func NewOAuthProviderFactory() *OAuthProviderFactory {
	return &OAuthProviderFactory{
		configs: make(map[string]*OAuthConfig),
		tokens:  make(map[string]*OAuthToken),
	}
}

// RegisterConfig registers OAuth configuration for a provider
func (f *OAuthProviderFactory) RegisterConfig(provider string, config *OAuthConfig) {
	f.configs[provider] = config
}

// GetConfig retrieves OAuth configuration for a provider
func (f *OAuthProviderFactory) GetConfig(provider string) (*OAuthConfig, bool) {
	config, ok := f.configs[provider]
	return config, ok
}

// StoreToken stores an OAuth token for a provider
func (f *OAuthProviderFactory) StoreToken(provider string, token *OAuthToken) {
	f.tokens[provider] = token
}

// GetToken retrieves an OAuth token for a provider
func (f *OAuthProviderFactory) GetToken(provider string) (*OAuthToken, bool) {
	token, ok := f.tokens[provider]
	return token, ok
}

// ClearToken removes the OAuth token for a provider
func (f *OAuthProviderFactory) ClearToken(provider string) {
	delete(f.tokens, provider)
}

// OpenAIOAuthProvider extends OpenAIProvider with OAuth support
type OpenAIOAuthProvider struct {
	*OpenAIProvider
	BaseOAuthProvider
}

// NewOpenAIOAuthProvider creates a new OpenAI provider with OAuth support
func NewOpenAIOAuthProvider(config OpenAIConfig, oauthConfig *OAuthConfig) (*OpenAIOAuthProvider, error) {
	baseProvider, err := NewOpenAIProvider(config)
	if err != nil {
		return nil, err
	}

	return &OpenAIOAuthProvider{
		OpenAIProvider: baseProvider,
		BaseOAuthProvider: BaseOAuthProvider{
			oauthConfig: oauthConfig,
		},
	}, nil
}

// RefreshOAuthToken refreshes the OpenAI OAuth token
func (p *OpenAIOAuthProvider) RefreshOAuthToken(ctx context.Context) error {
	// OpenAI uses standard OAuth2 token refresh
	// This would need to be implemented with the actual OAuth2 flow
	return p.BaseOAuthProvider.RefreshOAuthToken(ctx)
}

// GeminiOAuthProvider extends GeminiProvider with OAuth support
type GeminiOAuthProvider struct {
	*GeminiProvider
	BaseOAuthProvider
}

// NewGeminiOAuthProvider creates a new Gemini provider with OAuth support
func NewGeminiOAuthProvider(config GeminiConfig, oauthConfig *OAuthConfig) (*GeminiOAuthProvider, error) {
	baseProvider, err := NewGeminiProvider(config)
	if err != nil {
		return nil, err
	}

	return &GeminiOAuthProvider{
		GeminiProvider: baseProvider,
		BaseOAuthProvider: BaseOAuthProvider{
			oauthConfig: oauthConfig,
		},
	}, nil
}

// RefreshOAuthToken refreshes the Gemini OAuth token
func (p *GeminiOAuthProvider) RefreshOAuthToken(ctx context.Context) error {
	// Google uses standard OAuth2 token refresh
	return p.BaseOAuthProvider.RefreshOAuthToken(ctx)
}

// ProviderAuthType represents the type of authentication a provider supports
type ProviderAuthType string

const (
	// AuthTypeAPIKey uses API key authentication
	AuthTypeAPIKey ProviderAuthType = "api_key"
	// AuthTypeOAuth uses OAuth authentication
	AuthTypeOAuth ProviderAuthType = "oauth"
	// AuthTypeBoth supports both API key and OAuth
	AuthTypeBoth ProviderAuthType = "both"
	// AuthTypeNone requires no authentication
	AuthTypeNone ProviderAuthType = "none"
)

// ProviderAuthInfo contains authentication information for a provider
type ProviderAuthInfo struct {
	Provider    string
	AuthTypes   []ProviderAuthType
	DefaultType ProviderAuthType
}

// GetProviderAuthInfo returns authentication information for all supported providers
func GetProviderAuthInfo() []ProviderAuthInfo {
	return []ProviderAuthInfo{
		{
			Provider:    "anthropic",
			AuthTypes:   []ProviderAuthType{AuthTypeAPIKey},
			DefaultType: AuthTypeAPIKey,
		},
		{
			Provider:    "openai",
			AuthTypes:   []ProviderAuthType{AuthTypeAPIKey, AuthTypeOAuth},
			DefaultType: AuthTypeAPIKey,
		},
		{
			Provider:    "openrouter",
			AuthTypes:   []ProviderAuthType{AuthTypeAPIKey},
			DefaultType: AuthTypeAPIKey,
		},
		{
			Provider:    "gemini",
			AuthTypes:   []ProviderAuthType{AuthTypeAPIKey, AuthTypeOAuth},
			DefaultType: AuthTypeAPIKey,
		},
		{
			Provider:    "bedrock",
			AuthTypes:   []ProviderAuthType{AuthTypeAPIKey, AuthTypeOAuth},
			DefaultType: AuthTypeAPIKey,
		},
		{
			Provider:    "ollama",
			AuthTypes:   []ProviderAuthType{AuthTypeNone},
			DefaultType: AuthTypeNone,
		},
		{
			Provider:    "lmstudio",
			AuthTypes:   []ProviderAuthType{AuthTypeNone, AuthTypeAPIKey},
			DefaultType: AuthTypeNone,
		},
	}
}

// SupportsOAuth returns true if the provider supports OAuth authentication
func SupportsOAuth(provider string) bool {
	info := GetProviderAuthInfo()
	for _, p := range info {
		if p.Provider == provider {
			for _, authType := range p.AuthTypes {
				if authType == AuthTypeOAuth || authType == AuthTypeBoth {
					return true
				}
			}
		}
	}
	return false
}

// GetDefaultAuthType returns the default authentication type for a provider
func GetDefaultAuthType(provider string) ProviderAuthType {
	info := GetProviderAuthInfo()
	for _, p := range info {
		if p.Provider == provider {
			return p.DefaultType
		}
	}
	return AuthTypeAPIKey
}