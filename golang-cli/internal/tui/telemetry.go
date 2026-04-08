// Package tui provides terminal UI components for the Cline CLI.
package tui

import (
	"encoding/json"
	"fmt"
	"time"
)

// Telemetry tracks usage metrics and events for the CLI
type Telemetry struct {
	// Event tracking
	events []TelemetryEvent

	// Auto-approval tracking
	autoApprovals []AutoApprovalEvent

	// Session info
	sessionStart time.Time
}

// TelemetryEvent represents a generic telemetry event
type TelemetryEvent struct {
	Type      string                 `json:"type"`
	Timestamp int64                  `json:"ts"`
	Data      map[string]interface{} `json:"data"`
}

// AutoApprovalEvent tracks YOLO mode auto-approvals
type AutoApprovalEvent struct {
	Timestamp   int64  `json:"ts"`
	AskType     string `json:"askType"`
	Description string `json:"description"`
	TaskID      string `json:"taskId,omitempty"`
}

// NewTelemetry creates a new telemetry tracker
func NewTelemetry() *Telemetry {
	return &Telemetry{
		events:        make([]TelemetryEvent, 0),
		autoApprovals: make([]AutoApprovalEvent, 0),
		sessionStart:  time.Now(),
	}
}

// RecordEvent records a generic telemetry event
func (t *Telemetry) RecordEvent(eventType string, data map[string]interface{}) {
	event := TelemetryEvent{
		Type:      eventType,
		Timestamp: time.Now().UnixMilli(),
		Data:      data,
	}
	t.events = append(t.events, event)
}

// RecordAutoApproval records a YOLO mode auto-approval
func (t *Telemetry) RecordAutoApproval(askType, description string) {
	approval := AutoApprovalEvent{
		Timestamp:   time.Now().UnixMilli(),
		AskType:     askType,
		Description: truncate(description, 100),
	}
	t.autoApprovals = append(t.autoApprovals, approval)
}

// RecordYOLOModeActivation records when YOLO mode is enabled
func (t *Telemetry) RecordYOLOModeActivation(taskID string) {
	t.RecordEvent("yolo_mode_activated", map[string]interface{}{
		"taskId": taskID,
	})
}

// RecordToolUsage records tool usage
func (t *Telemetry) RecordToolUsage(toolName string, params map[string]interface{}) {
	t.RecordEvent("tool_usage", map[string]interface{}{
		"tool":   toolName,
		"params": params,
	})
}

// RecordCommandExecution records command execution
func (t *Telemetry) RecordCommandExecution(command string, approved bool) {
	t.RecordEvent("command_execution", map[string]interface{}{
		"command":  command,
		"approved": approved,
	})
}

// RecordError records an error
func (t *Telemetry) RecordError(errType string, err error) {
	t.RecordEvent("error", map[string]interface{}{
		"errorType": errType,
		"message":   err.Error(),
	})
}

// GetAutoApprovalCount returns the number of auto-approvals
func (t *Telemetry) GetAutoApprovalCount() int {
	return len(t.autoApprovals)
}

// GetEventCount returns the total number of events
func (t *Telemetry) GetEventCount() int {
	return len(t.events)
}

// GetSessionDuration returns the session duration
func (t *Telemetry) GetSessionDuration() time.Duration {
	return time.Since(t.sessionStart)
}

// Export exports all telemetry data as JSON
func (t *Telemetry) Export() (string, error) {
	data := map[string]interface{}{
		"sessionStart":    t.sessionStart.UnixMilli(),
		"sessionDuration": t.GetSessionDuration().Milliseconds(),
		"events":          t.events,
		"autoApprovals":   t.autoApprovals,
		"summary": map[string]interface{}{
			"totalEvents":        len(t.events),
			"totalAutoApprovals": len(t.autoApprovals),
		},
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal telemetry: %w", err)
	}

	return string(jsonData), nil
}

// ExportYOLOSummary exports a summary of YOLO mode usage
func (t *Telemetry) ExportYOLOSummary() (string, error) {
	// Count approvals by type
	approvalCounts := make(map[string]int)
	for _, approval := range t.autoApprovals {
		approvalCounts[approval.AskType]++
	}

	summary := map[string]interface{}{
		"yoloModeEnabled":   true,
		"totalApprovals":    len(t.autoApprovals),
		"approvalBreakdown": approvalCounts,
		"sessionDuration":   t.GetSessionDuration().Seconds(),
	}

	jsonData, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal YOLO summary: %w", err)
	}

	return string(jsonData), nil
}

// truncate truncates a string to the specified length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// ShouldCaptureTelemetry returns true if telemetry should be captured
func ShouldCaptureTelemetry() bool {
	// Check environment variable or config setting
	// Default to false for privacy
	return false
}

// TelemetryConfig holds telemetry configuration
type TelemetryConfig struct {
	Enabled        bool   `json:"enabled"`
	Endpoint       string `json:"endpoint,omitempty"`
	IncludePII     bool   `json:"includePii,omitempty"`
	AnonymousOnly  bool   `json:"anonymousOnly,omitempty"`
}

// DefaultTelemetryConfig returns the default telemetry configuration
func DefaultTelemetryConfig() TelemetryConfig {
	return TelemetryConfig{
		Enabled:       false,
		AnonymousOnly: true,
	}
}