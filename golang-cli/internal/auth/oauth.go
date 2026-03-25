// Package auth provides OAuth 2.0 authentication flows with PKCE support
// for the Cline CLI.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"golang.org/x/oauth2"
)

const (
	// DefaultCallbackPort is the default port for the local callback server
	DefaultCallbackPort = 0 // 0 means any available port
	// DefaultCallbackPath is the path for the OAuth callback
	DefaultCallbackPath = "/oauth/callback"
	// StateLength is the length of the random state parameter in bytes
	StateLength = 32
	// PKCEVerifierLength is the length of the PKCE verifier in bytes
	PKCEVerifierLength = 32
)

// Provider identifiers
const (
	// ProviderAnthropic is the Anthropic provider
	ProviderAnthropic = "anthropic"
	// ProviderOpenAI is the OpenAI provider
	ProviderOpenAI = "openai"
	// ProviderOpenRouter is the OpenRouter provider
	ProviderOpenRouter = "openrouter"
	// ProviderGemini is the Google Gemini provider
	ProviderGemini = "gemini"
	// ProviderBedrock is the AWS Bedrock provider
	ProviderBedrock = "bedrock"
	// ProviderOllama is the Ollama local provider
	ProviderOllama = "ollama"
	// ProviderLMStudio is the LM Studio local provider
	ProviderLMStudio = "lmstudio"
)

var (
	// ErrAuthorizationDenied is returned when the user denies authorization
	ErrAuthorizationDenied = errors.New("authorization denied by user")
	// ErrAuthorizationTimeout is returned when the authorization flow times out
	ErrAuthorizationTimeout = errors.New("authorization timeout")
	// ErrInvalidState is returned when the state parameter doesn't match
	ErrInvalidState = errors.New("invalid state parameter")
	// ErrTokenExchangeFailed is returned when token exchange fails
	ErrTokenExchangeFailed = errors.New("token exchange failed")
	// ErrNoAuthorizationCode is returned when no authorization code is received
	ErrNoAuthorizationCode = errors.New("no authorization code received")
	// ErrCallbackServerFailed is returned when the callback server fails to start
	ErrCallbackServerFailed = errors.New("callback server failed")
)

// Token represents an OAuth 2.0 token with additional metadata
type Token struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	Expiry       time.Time `json:"expiry"`
}

// IsExpired returns true if the token is expired or about to expire
func (t *Token) IsExpired() bool {
	if t.Expiry.IsZero() {
		return false
	}
	// Consider token expired 1 minute before actual expiry
	return time.Until(t.Expiry) < time.Minute
}

// ToOAuth2Token converts to golang.org/x/oauth2 Token
func (t *Token) ToOAuth2Token() *oauth2.Token {
	return &oauth2.Token{
		AccessToken:  t.AccessToken,
		RefreshToken: t.RefreshToken,
		TokenType:    t.TokenType,
		Expiry:       t.Expiry,
	}
}

// TokenFromOAuth2 creates a Token from golang.org/x/oauth2 Token
func TokenFromOAuth2(tok *oauth2.Token) *Token {
	return &Token{
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		TokenType:    tok.TokenType,
		Expiry:       tok.Expiry,
	}
}

// Config holds the OAuth 2.0 configuration
type Config struct {
	// ClientID is the OAuth client ID
	ClientID string
	// ClientSecret is the OAuth client secret (optional for PKCE)
	ClientSecret string
	// AuthURL is the authorization endpoint URL
	AuthURL string
	// TokenURL is the token endpoint URL
	TokenURL string
	// RedirectURL is the callback URL (automatically set if empty)
	RedirectURL string
	// Scopes are the OAuth scopes to request
	Scopes []string
	// CallbackPort is the port for the local callback server (0 = any)
	CallbackPort int
	// CallbackPath is the path for the callback endpoint
	CallbackPath string
	// Timeout is the total timeout for the authorization flow
	Timeout time.Duration
	// AdditionalParams are additional query parameters for the auth URL
	AdditionalParams map[string]string
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.ClientID == "" {
		return errors.New("client ID is required")
	}
	if c.AuthURL == "" {
		return errors.New("auth URL is required")
	}
	if c.TokenURL == "" {
		return errors.New("token URL is required")
	}
	return nil
}

// Flow manages the OAuth 2.0 authorization flow
type Flow struct {
	config       *Config
	oauth2Config *oauth2.Config
	state        string
	codeVerifier string
	codeChallenge string
	token        *Token
	server       *callbackServer
	resultChan   chan *authResult
}

// authResult holds the result of the authorization flow
type authResult struct {
	token *Token
	err   error
}

// NewFlow creates a new OAuth flow with the given configuration
func NewFlow(config *Config) (*Flow, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Set defaults
	if config.CallbackPort == 0 {
		config.CallbackPort = DefaultCallbackPort
	}
	if config.CallbackPath == "" {
		config.CallbackPath = DefaultCallbackPath
	}
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Minute
	}

	flow := &Flow{
		config:     config,
		resultChan: make(chan *authResult, 1),
	}

	// Generate PKCE parameters
	if err := flow.generatePKCE(); err != nil {
		return nil, fmt.Errorf("failed to generate PKCE: %w", err)
	}

	// Generate state parameter
	if err := flow.generateState(); err != nil {
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}

	return flow, nil
}

// generatePKCE generates PKCE code verifier and challenge
func (f *Flow) generatePKCE() error {
	// Generate code verifier
	verifier := make([]byte, PKCEVerifierLength)
	if _, err := rand.Read(verifier); err != nil {
		return fmt.Errorf("failed to generate verifier: %w", err)
	}
	f.codeVerifier = base64.RawURLEncoding.EncodeToString(verifier)

	// Generate code challenge (SHA256 hash of verifier)
	hash := sha256.Sum256([]byte(f.codeVerifier))
	f.codeChallenge = base64.RawURLEncoding.EncodeToString(hash[:])

	return nil
}

// generateState generates a random state parameter
func (f *Flow) generateState() error {
	stateBytes := make([]byte, StateLength)
	if _, err := rand.Read(stateBytes); err != nil {
		return fmt.Errorf("failed to generate state: %w", err)
	}
	f.state = base64.RawURLEncoding.EncodeToString(stateBytes)
	return nil
}

// GetAuthURL returns the authorization URL to open in the browser
func (f *Flow) GetAuthURL() (string, error) {
	if f.server == nil {
		return "", errors.New("callback server not started")
	}

	redirectURL := f.server.GetURL() + f.config.CallbackPath

	f.oauth2Config = &oauth2.Config{
		ClientID:     f.config.ClientID,
		ClientSecret: f.config.ClientSecret,
		RedirectURL:  redirectURL,
		Scopes:       f.config.Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  f.config.AuthURL,
			TokenURL: f.config.TokenURL,
		},
	}

	// Build auth URL with PKCE
	opts := []oauth2.AuthCodeOption{
		oauth2.SetAuthURLParam("code_challenge", f.codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	}

	// Add any additional parameters
	for key, value := range f.config.AdditionalParams {
		opts = append(opts, oauth2.SetAuthURLParam(key, value))
	}

	authURL := f.oauth2Config.AuthCodeURL(f.state, opts...)
	return authURL, nil
}

// StartCallbackServer starts the local HTTP callback server
func (f *Flow) StartCallbackServer() error {
	if f.server != nil {
		return errors.New("callback server already started")
	}

	server, err := newCallbackServer(callbackServerConfig{
		Port:     f.config.CallbackPort,
		Path:     f.config.CallbackPath,
		OnCode:   f.handleAuthorizationCode,
		OnError:  f.handleAuthorizationError,
		Timeout:  f.config.Timeout,
	})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCallbackServerFailed, err)
	}

	f.server = server
	return server.Start()
}

// handleAuthorizationCode handles the authorization code from the callback
func (f *Flow) handleAuthorizationCode(code, state string) {
	// Verify state parameter
	if state != f.state {
		f.resultChan <- &authResult{
			err: fmt.Errorf("%w: state mismatch", ErrInvalidState),
		}
		return
	}

	// Exchange code for token
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	opts := []oauth2.AuthCodeOption{
		oauth2.SetAuthURLParam("code_verifier", f.codeVerifier),
	}

	tok, err := f.oauth2Config.Exchange(ctx, code, opts...)
	if err != nil {
		f.resultChan <- &authResult{
			err: fmt.Errorf("%w: %v", ErrTokenExchangeFailed, err),
		}
		return
	}

	f.token = TokenFromOAuth2(tok)
	f.resultChan <- &authResult{token: f.token}
}

// handleAuthorizationError handles authorization errors from the callback
func (f *Flow) handleAuthorizationError(errMsg, description string) {
	var err error
	if errMsg == "access_denied" {
		err = ErrAuthorizationDenied
	} else {
		err = fmt.Errorf("authorization error: %s - %s", errMsg, description)
	}
	f.resultChan <- &authResult{err: err}
}

// OpenBrowser opens the system default browser with the given URL
func (f *Flow) OpenBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	default: // Linux and others
		cmd = "xdg-open"
		args = []string{url}
	}

	if err := exec.Command(cmd, args...).Start(); err != nil {
		return fmt.Errorf("failed to open browser: %w", err)
	}

	return nil
}

// WaitForAuthorization waits for the authorization to complete
// This blocks until authorization is complete, times out, or fails
func (f *Flow) WaitForAuthorization() (*Token, error) {
	if f.server == nil {
		return nil, errors.New("callback server not started")
	}

	ctx, cancel := context.WithTimeout(context.Background(), f.config.Timeout)
	defer cancel()

	select {
	case result := <-f.resultChan:
		if result.err != nil {
			return nil, result.err
		}
		return result.token, nil
	case <-ctx.Done():
		return nil, ErrAuthorizationTimeout
	}
}

// Execute runs the complete OAuth flow: start server, open browser, wait for result
func (f *Flow) Execute(openBrowser bool) (*Token, error) {
	// Start callback server
	if err := f.StartCallbackServer(); err != nil {
		return nil, err
	}
	defer f.Stop()

	// Get authorization URL
	authURL, err := f.GetAuthURL()
	if err != nil {
		return nil, err
	}

	// Open browser if requested
	if openBrowser {
		if err := f.OpenBrowser(authURL); err != nil {
			// Don't fail if browser can't be opened, user can manually navigate
			fmt.Printf("Please open this URL in your browser:\n%s\n", authURL)
		}
	} else {
		fmt.Printf("Please open this URL in your browser:\n%s\n", authURL)
	}

	// Wait for authorization
	return f.WaitForAuthorization()
}

// Stop stops the callback server
func (f *Flow) Stop() error {
	if f.server != nil {
		return f.server.Stop()
	}
	return nil
}

// GetToken returns the current token (may be nil if not yet authorized)
func (f *Flow) GetToken() *Token {
	return f.token
}

// RefreshToken refreshes the access token using the refresh token
func (f *Flow) RefreshToken(refreshToken string) (*Token, error) {
	if refreshToken == "" {
		return nil, errors.New("refresh token is required")
	}

	if f.oauth2Config == nil {
		return nil, errors.New("OAuth config not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tokenSource := f.oauth2Config.TokenSource(ctx, &oauth2.Token{
		RefreshToken: refreshToken,
	})

	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	f.token = TokenFromOAuth2(newToken)
	return f.token, nil
}

// GetState returns the state parameter (for testing purposes)
func (f *Flow) GetState() string {
	return f.state
}

// GetCodeChallenge returns the PKCE code challenge (for testing purposes)
func (f *Flow) GetCodeChallenge() string {
	return f.codeChallenge
}

// TokenStorage defines the interface for token storage
type TokenStorage interface {
	// Save stores the token
	Save(token *Token) error
	// Load retrieves the token
	Load() (*Token, error)
	// Delete removes the stored token
	Delete() error
}

// JSONTokenStorage implements TokenStorage using JSON file storage
type JSONTokenStorage struct {
	filePath string
}

// NewJSONTokenStorage creates a new JSON token storage
func NewJSONTokenStorage(filePath string) *JSONTokenStorage {
	return &JSONTokenStorage{
		filePath: filePath,
	}
}

// Save stores the token to a JSON file
func (s *JSONTokenStorage) Save(token *Token) error {
	if token == nil {
		return errors.New("token is nil")
	}

	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	// Write with restricted permissions (owner only)
	if err := writeFileRestricted(s.filePath, data); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	return nil
}

// Load retrieves the token from a JSON file
func (s *JSONTokenStorage) Load() (*Token, error) {
	data, err := readFileRestricted(s.filePath)
	if err != nil {
		if errors.Is(err, ErrFileNotFound) {
			return nil, errors.New("token not found")
		}
		return nil, fmt.Errorf("failed to read token file: %w", err)
	}

	var token Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}

	return &token, nil
}

// Delete removes the stored token file
func (s *JSONTokenStorage) Delete() error {
	return deleteFileRestricted(s.filePath)
}

// Helper function to write file with restricted permissions (0600 - owner read/write only)
func writeFileRestricted(path string, data []byte) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file with restricted permissions (0600)
	// Use os.WriteFile with 0600 permissions (owner read/write only)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// Helper function to read file
func readFileRestricted(path string) ([]byte, error) {
	// Check if file exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrFileNotFound
		}
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Verify file is a regular file
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("not a regular file")
	}

	// Read file contents
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return data, nil
}

// Helper function to delete file
func deleteFileRestricted(path string) error {
	// Check if file exists
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist, consider deletion successful
		}
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Delete the file
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

var ErrFileNotFound = errors.New("file not found")
