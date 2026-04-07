// Package errorservice provides PostHog error provider implementation
// Reference: src/services/error/providers/PostHogErrorProvider.ts
package errorservice

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"github.com/google/uuid"
)

// PostHogEvent represents an event to be sent to PostHog
type PostHogEvent struct {
	Event      string                 `json:"event"`
	DistinctID string                 `json:"distinct_id"`
	Properties map[string]interface{} `json:"properties"`
	Timestamp  time.Time              `json:"timestamp"`
}

// PostHogCaptureRequest represents a capture request to PostHog
type PostHogCaptureRequest struct {
	APIKey string         `json:"api_key"`
	Event  string         `json:"event"`
	Batch  []PostHogEvent `json:"batch"`
}

// PostHogErrorProvider implements the ErrorProvider interface using PostHog
// Reference: src/services/error/providers/PostHogErrorProvider.ts:25-164
type PostHogErrorProvider struct {
	mu         sync.RWMutex
	client     *http.Client
	config     PostHogErrorConfig
	distinctID string
	enabled    bool
	logger     *slog.Logger
}

// PostHogErrorConfig contains configuration for PostHog error tracking
type PostHogErrorConfig struct {
	APIKey                     string
	Host                       string
	EnableExceptionAutocapture bool
	Version                    string
}

// NewPostHogErrorProvider creates a new PostHog error provider
// Reference: src/services/error/providers/PostHogErrorProvider.ts:31-44
func NewPostHogErrorProvider(logger *slog.Logger) (*PostHogErrorProvider, error) {
	config := getPostHogErrorConfig()

	// Generate distinct ID
	distinctID := getDistinctIDForErrors()

	provider := &PostHogErrorProvider{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		config:     config,
		distinctID: distinctID,
		enabled:    config.APIKey != "",
		logger:     logger,
	}

	logger.Info("PostHog error provider initialized", "enabled", provider.enabled)

	return provider, nil
}

// CaptureException captures an error with context and sends to PostHog
// Reference: src/services/error/providers/PostHogErrorProvider.ts:68-81
func (p *PostHogErrorProvider) CaptureException(err error, context map[string]string) error {
	if !p.enabled {
		return nil
	}

	if !p.shouldCaptureError() {
		return nil
	}

	// Build error details
	errorDetails := map[string]interface{}{
		"name":              "Error",
		"extension_version": p.config.Version,
		"is_dev":            getIsDevForErrors(),
		"timestamp":         time.Now().UTC().Format(time.RFC3339),
	}

	// Add error-specific details
	if err != nil {
		errorDetails["message"] = err.Error()
		errorDetails["type"] = fmt.Sprintf("%T", err)
	}

	// Add context
	for k, v := range context {
		errorDetails[k] = v
	}

	// Add stack trace if available
	if clineErr, ok := err.(*ClineError); ok {
		errorDetails["stack"] = clineErr.Stack
		errorDetails["model_id"] = clineErr.ModelID
		errorDetails["provider_id"] = clineErr.ProviderID
	} else {
		errorDetails["stack"] = string(debug.Stack())
	}

	// Send as exception event
	return p.sendEvent("extension.exception", errorDetails)
}

// LogException logs an error locally and to PostHog
// Reference: src/services/error/providers/PostHogErrorProvider.ts:83-116
func (p *PostHogErrorProvider) LogException(err error, context map[string]string) {
	if !p.enabled {
		return
	}

	// Build error details
	errorDetails := map[string]interface{}{
		"message":           "",
		"stack":             "",
		"name":              "Error",
		"extension_version": p.config.Version,
		"is_dev":            getIsDevForErrors(),
		"timestamp":         time.Now().UTC().Format(time.RFC3339),
		"error_type":        "exception",
	}

	if err != nil {
		errorDetails["message"] = err.Error()
		errorDetails["name"] = fmt.Sprintf("%T", err)
	}

	// Add stack trace
	if clineErr, ok := err.(*ClineError); ok {
		errorDetails["stack"] = clineErr.Stack
		errorDetails["model_id"] = clineErr.ModelID
		errorDetails["provider_id"] = clineErr.ProviderID
		if clineErr.Original != nil {
			errorDetails["original_error"] = clineErr.Original.Error()
		}
	} else {
		errorDetails["stack"] = string(debug.Stack())
	}

	// Merge context
	for k, v := range context {
		errorDetails[k] = v
	}

	// Send to PostHog
	_ = p.sendEvent("extension.error", errorDetails)
}

// LogMessage logs a message with level to PostHog
// Reference: src/services/error/providers/PostHogErrorProvider.ts:118-144
func (p *PostHogErrorProvider) LogMessage(message string, level string, context map[string]string) {
	if !p.enabled {
		return
	}

	if !p.shouldLogMessage(level) {
		return
	}

	// Truncate long messages
	if len(message) > 500 {
		message = message[:500] + "..."
	}

	properties := map[string]interface{}{
		"message":           message,
		"level":             level,
		"extension_version": p.config.Version,
		"is_dev":            getIsDevForErrors(),
		"timestamp":         time.Now().UTC().Format(time.RFC3339),
	}

	// Merge context
	for k, v := range context {
		properties[k] = v
	}

	_ = p.sendEvent("extension.message", properties)
}

// IsEnabled returns whether the provider is enabled
// Reference: src/services/error/providers/PostHogErrorProvider.ts:146-148
func (p *PostHogErrorProvider) IsEnabled() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.enabled
}

// GetSettings returns current error settings
// Reference: src/services/error/providers/PostHogErrorProvider.ts:150-152
func (p *PostHogErrorProvider) GetSettings() Settings {
	return Settings{
		Enabled:     p.enabled,
		HostEnabled: true,
		Level:       ErrorLevelAll,
	}
}

// Dispose cleans up provider resources
// Reference: src/services/error/providers/PostHogErrorProvider.ts:158-163
func (p *PostHogErrorProvider) Dispose() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.enabled = false
	return nil
}

// shouldCaptureError determines if errors should be captured based on settings
func (p *PostHogErrorProvider) shouldCaptureError() bool {
	// Could be extended to check error level settings
	return true
}

// shouldLogMessage determines if a message should be logged based on level
func (p *PostHogErrorProvider) shouldLogMessage(level string) bool {
	// Could be extended to filter by level
	return true
}

// sendEvent sends an event to PostHog
func (p *PostHogErrorProvider) sendEvent(eventName string, properties map[string]interface{}) error {
	p.mu.RLock()
	apiKey := p.config.APIKey
	host := p.config.Host
	distinctID := p.distinctID
	p.mu.RUnlock()

	if apiKey == "" {
		return nil
	}

	event := PostHogEvent{
		Event:      eventName,
		DistinctID: distinctID,
		Properties: properties,
		Timestamp:  time.Now().UTC(),
	}

	reqBody := PostHogCaptureRequest{
		APIKey: apiKey,
		Event:  eventName,
		Batch:  []PostHogEvent{event},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal error event: %w", err)
	}

	url := fmt.Sprintf("%s/capture/", host)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create error request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "cline-cli-go-error/1.0")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send error event: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("PostHog error endpoint returned status %d", resp.StatusCode)
	}

	return nil
}

// Helper functions

func getPostHogErrorConfig() PostHogErrorConfig {
	apiKey := os.Getenv("POSTHOG_ERROR_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("POSTHOG_API_KEY")
	}
	if apiKey == "" {
		// Use the default PostHog API key for Cline
		apiKey = "phc_7PdjBaqBRnKiuRjR2HczzXy5Y9fM4e9xG8V7tJ3K4N5" // This is a placeholder
	}

	host := os.Getenv("POSTHOG_HOST")
	if host == "" {
		host = "https://app.posthog.com"
	}

	version := os.Getenv("CLINE_VERSION")
	if version == "" {
		version = "0.0.0-dev"
	}

	return PostHogErrorConfig{
		APIKey:                     apiKey,
		Host:                       host,
		EnableExceptionAutocapture: true,
		Version:                    version,
	}
}

func getDistinctIDForErrors() string {
	// Check environment variable first
	if id := os.Getenv("CLINE_DISTINCT_ID"); id != "" {
		return id
	}

	// Check for existing distinct ID in temp file
	tempDir := os.TempDir()
	distinctIDFile := tempDir + "/cline_distinct_id"

	if data, err := os.ReadFile(distinctIDFile); err == nil && len(data) > 0 {
		return string(data)
	}

	// Generate new distinct ID
	newID := uuid.New().String()

	// Save for future use
	_ = os.WriteFile(distinctIDFile, []byte(newID), 0600)

	return newID
}

func getIsDevForErrors() bool {
	return os.Getenv("CLINE_DEV") != "" || os.Getenv("IS_DEV") != ""
}