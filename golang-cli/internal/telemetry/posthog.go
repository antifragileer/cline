// Package telemetry provides PostHog telemetry provider implementation
// Reference: src/services/telemetry/providers/posthog/PostHogTelemetryProvider.ts
package telemetry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
)

// PostHogConfig contains PostHog configuration
// Reference: src/services/telemetry/providers/posthog/PostHogClientProvider.ts
type PostHogConfig struct {
	APIKey string
	Host   string
}

// PostHogProvider implements the Provider interface for PostHog
// Reference: src/services/telemetry/providers/posthog/PostHogTelemetryProvider.ts
type PostHogProvider struct {
	mu         sync.RWMutex
	client     *http.Client
	config     PostHogConfig
	distinctID string
	enabled    bool
	logger     *slog.Logger
	eventQueue []PostHogEvent
	queueMu    sync.Mutex
}

// PostHogEvent represents a single event to be sent to PostHog
// Reference: https://posthog.com/docs/api/capture
type PostHogEvent struct {
	Event      string                 `json:"event"`
	DistinctID string                 `json:"distinct_id"`
	Properties map[string]interface{} `json:"properties"`
	Timestamp  time.Time              `json:"timestamp"`
}

// PostHogCaptureRequest is the request body for the capture API
type PostHogCaptureRequest struct {
	APIKey string         `json:"api_key"`
	Event  string         `json:"event,omitempty"`
	Batch  []PostHogEvent `json:"batch,omitempty"`
}

// NewPostHogProvider creates a new PostHog telemetry provider
// Reference: src/services/telemetry/providers/posthog/PostHogTelemetryProvider.ts:24-47
func NewPostHogProvider(logger *slog.Logger) (*PostHogProvider, error) {
	config := getPostHogConfig()

	// Generate distinct ID (same logic as distinctId.ts)
	distinctID := getDistinctID()

	provider := &PostHogProvider{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		config:     config,
		distinctID: distinctID,
		enabled:    config.APIKey != "",
		logger:     logger,
		eventQueue: make([]PostHogEvent, 0),
	}

	logger.Info("PostHog provider initialized", "enabled", provider.enabled)

	return provider, nil
}

// Log captures a telemetry event
// Reference: src/services/telemetry/providers/posthog/PostHogTelemetryProvider.ts:74-91
func (p *PostHogProvider) Log(event string, properties map[string]interface{}) error {
	if !p.enabled {
		return nil
	}

	// Add required properties
	props := map[string]interface{}{
		"distinct_id": p.distinctID,
		"$lib":        "cline-cli-go",
	}
	for k, v := range properties {
		props[k] = v
	}

	posthogEvent := PostHogEvent{
		Event:      event,
		DistinctID: p.distinctID,
		Properties: props,
		Timestamp:  time.Now().UTC(),
	}

	// Queue the event for batching
	p.queueMu.Lock()
	p.eventQueue = append(p.eventQueue, posthogEvent)
	shouldFlush := len(p.eventQueue) >= 10 // Flush after 10 events
	p.queueMu.Unlock()

	if shouldFlush {
		return p.Flush()
	}

	return nil
}

// LogRequired captures a required telemetry event (always sent)
// Reference: src/services/telemetry/providers/posthog/PostHogTelemetryProvider.ts:93-102
func (p *PostHogProvider) LogRequired(event string, properties map[string]interface{}) error {
	// Required events bypass the enabled check but still need API key
	if p.config.APIKey == "" {
		return nil
	}

	props := map[string]interface{}{
		"distinct_id": p.distinctID,
		"$lib":        "cline-cli-go",
		"_required":   true,
	}
	for k, v := range properties {
		props[k] = v
	}

	posthogEvent := PostHogEvent{
		Event:      event,
		DistinctID: p.distinctID,
		Properties: props,
		Timestamp:  time.Now().UTC(),
	}

	// Send immediately for required events
	return p.sendEvent(posthogEvent)
}

// Flush sends all queued events to PostHog
// Reference: src/services/telemetry/providers/posthog/PostHogTelemetryProvider.ts:71-73
func (p *PostHogProvider) Flush() error {
	if !p.enabled {
		return nil
	}

	p.queueMu.Lock()
	events := make([]PostHogEvent, len(p.eventQueue))
	copy(events, p.eventQueue)
	p.eventQueue = p.eventQueue[:0] // Clear queue
	p.queueMu.Unlock()

	if len(events) == 0 {
		return nil
	}

	return p.sendBatch(events)
}

// Dispose cleans up provider resources
// Reference: src/services/telemetry/providers/posthog/PostHogTelemetryProvider.ts:204-213
func (p *PostHogProvider) Dispose() error {
	// Flush any pending events
	if err := p.Flush(); err != nil {
		p.logger.Warn("Failed to flush events during dispose", "error", err)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.enabled = false
	return nil
}

// IsEnabled returns whether the provider is enabled
// Reference: src/services/telemetry/providers/posthog/PostHogTelemetryProvider.ts:122-138
func (p *PostHogProvider) IsEnabled() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.enabled
}

// sendEvent sends a single event to PostHog
func (p *PostHogProvider) sendEvent(event PostHogEvent) error {
	return p.sendBatch([]PostHogEvent{event})
}

// sendBatch sends a batch of events to PostHog
// Reference: https://posthog.com/docs/api/capture
func (p *PostHogProvider) sendBatch(events []PostHogEvent) error {
	if len(events) == 0 {
		return nil
	}

	reqBody := PostHogCaptureRequest{
		APIKey: p.config.APIKey,
		Batch:  events,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal events: %w", err)
	}

	// Use the standard PostHog capture endpoint
	url := fmt.Sprintf("%s/capture/", p.config.Host)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "cline-cli-go/1.0")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send events: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("PostHog returned status %d", resp.StatusCode)
	}

	p.logger.Debug("Sent events to PostHog", "count", len(events))
	return nil
}

// Helper functions

func getPostHogConfig() PostHogConfig {
	apiKey := os.Getenv("POSTHOG_API_KEY")
	if apiKey == "" {
		// Use the default PostHog API key for Cline
		apiKey = "phc_7PdjBaqBRnKiuRjR2HczzXy5Y9fM4e9xG8V7tJ3K4N5" // This is a placeholder - use actual key
	}

	host := os.Getenv("POSTHOG_HOST")
	if host == "" {
		host = "https://app.posthog.com"
	}

	return PostHogConfig{
		APIKey: apiKey,
		Host:   host,
	}
}

// getDistinctID generates or retrieves a distinct ID for the user
// Reference: src/services/logging/distinctId.ts
func getDistinctID() string {
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
