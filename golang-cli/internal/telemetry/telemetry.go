// Package telemetry provides telemetry event tracking for the Cline CLI
// Matches Node.js CLI implementation from cli/src/index.ts and src/services/telemetry/
package telemetry

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/cline/cline/golang-cli/internal/storage"
)

// Service defines the telemetry service interface
// Reference: cli/src/index.ts lines 82-88, 147-163, 182-184, 191-193, 198-199, 204-205, 211-212, 217-218, 311, 317, 548-549
type Service interface {
	// CaptureHostEvent captures a host-specific event with properties
	// Reference: cli/src/index.ts:147-163, 182-184, 191-193, 198-199, 204-205, 211-212, 217-218
	CaptureHostEvent(event string, properties map[string]interface{}) error

	// CaptureExtensionActivated captures when the CLI extension is activated
	// Reference: cli/src/index.ts:548
	CaptureExtensionActivated() error

	// CapturePlainTextMode captures when plain text mode is used
	// Reference: cli/src/index.ts:310
	CapturePlainTextMode(reason string) error

	// CaptureCommand captures command execution events
	// Reference: cli/src/index.ts:600, 657, 691, 769, 788
	CaptureCommand(command string, details string) error

	// CaptureAuth captures authentication events
	CaptureAuth(status string, provider string) error

	// CaptureAuthQuickSetup captures auth quick_setup command
	CaptureAuthQuickSetup() error

	// CaptureAuthInteractive captures auth interactive command
	CaptureAuthInteractive() error

	// CaptureTaskCreated captures when a task is created
	CaptureTaskCreated(taskID string, apiProvider string) error

	// CaptureModeFlag captures mode flag usage
	CaptureModeFlag(mode string) error

	// CaptureModelFlag captures model flag usage
	CaptureModelFlag(model string) error

	// CaptureThinkingFlag captures thinking flag usage
	CaptureThinkingFlag() error

	// CaptureReasoningEffortFlag captures reasoning effort flag usage
	CaptureReasoningEffortFlag(effort string) error

	// CaptureMaxConsecutiveMistakesFlag captures max consecutive mistakes flag usage
	CaptureMaxConsecutiveMistakesFlag(count int) error

	// CaptureYoloFlag captures yolo flag usage
	CaptureYoloFlag() error

	// CaptureAutoApproveAllFlag captures auto-approve-all flag usage
	CaptureAutoApproveAllFlag() error

	// CaptureDoubleCheckCompletionFlag captures double-check-completion flag usage
	CaptureDoubleCheckCompletionFlag() error

	// CapturePiped captures piped input detection
	CapturePiped() error

	// CaptureResumeTask captures task resumption
	CaptureResumeTask(withPrompt bool) error

	// Dispose flushes pending events before shutdown
	// Reference: cli/src/index.ts:82-88
	Dispose() error
}

// TelemetryMetadata contains standard metadata for all events
// Reference: src/services/telemetry/TelemetryService.ts:70-94
type TelemetryMetadata struct {
	ExtensionVersion string `json:"extension_version"`
	ClineType        string `json:"cline_type"`
	Platform         string `json:"platform"`
	PlatformVersion  string `json:"platform_version"`
	OSType           string `json:"os_type"`
	OSVersion        string `json:"os_version"`
	IsDev            string `json:"is_dev,omitempty"`
}

// Event represents a telemetry event
// Reference: src/services/telemetry/TelemetryService.ts:442-448
type Event struct {
	Event      string                 `json:"event"`
	Properties map[string]interface{} `json:"properties"`
	Timestamp  time.Time              `json:"timestamp"`
}

// DefaultService implements the Service interface
type DefaultService struct {
	mu       sync.RWMutex
	provider Provider
	metadata TelemetryMetadata
	storage  *storage.StorageContext
	logger   *slog.Logger
	enabled  bool
}

// Provider defines the telemetry provider interface (e.g., PostHog)
// Reference: src/services/telemetry/providers/ITelemetryProvider.ts
type Provider interface {
	// Log captures a telemetry event
	Log(event string, properties map[string]interface{}) error

	// LogRequired captures a required telemetry event (always sent)
	LogRequired(event string, properties map[string]interface{}) error

	// Flush forces any buffered events to be sent
	Flush() error

	// Dispose cleans up provider resources
	Dispose() error

	// IsEnabled returns whether the provider is enabled
	IsEnabled() bool
}

// NewService creates a new telemetry service
// Reference: src/services/telemetry/TelemetryService.ts:353-366
func NewService(storageCtx *storage.StorageContext, version string, logger *slog.Logger) (*DefaultService, error) {
	metadata := TelemetryMetadata{
		ExtensionVersion: version,
		ClineType:        "cli",
		Platform:         getPlatformName(),
		PlatformVersion:  runtime.Version(),
		OSType:           runtime.GOOS,
		OSVersion:        getOSVersion(),
		IsDev:            getIsDev(),
	}

	service := &DefaultService{
		metadata: metadata,
		storage:  storageCtx,
		logger:   logger,
		enabled:  isTelemetryEnabled(storageCtx),
	}

	// Initialize PostHog provider if telemetry is enabled
	if service.enabled {
		provider, err := NewPostHogProvider(logger)
		if err != nil {
			logger.Warn("Failed to initialize PostHog provider", "error", err)
			// Continue without telemetry - don't fail startup
		} else {
			service.provider = provider
		}
	}

	return service, nil
}

// CaptureHostEvent captures a host-specific event
// Reference: cli/src/index.ts:147-163, 182-184, 191-193, 198-199, 204-205, 211-212, 217-218
func (s *DefaultService) CaptureHostEvent(event string, properties map[string]interface{}) error {
	if !s.enabled || s.provider == nil {
		return nil
	}

	// Merge with metadata
	allProperties := s.getStandardAttributes(properties)

	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.provider.Log(event, allProperties)
}

// CaptureExtensionActivated captures when the CLI is activated
// Reference: cli/src/index.ts:548-549
func (s *DefaultService) CaptureExtensionActivated() error {
	return s.CaptureHostEvent("user.extension_activated", map[string]interface{}{
		"source": "cline_cli",
	})
}

// CapturePlainTextMode captures when plain text mode is used
// Reference: cli/src/index.ts:310
func (s *DefaultService) CapturePlainTextMode(reason string) error {
	return s.CaptureHostEvent("plain_text_mode", map[string]interface{}{
		"reason": reason,
	})
}

// CaptureCommand captures command execution events
// Reference: cli/src/index.ts:600, 657, 691, 769, 788, 802, 805, 940
func (s *DefaultService) CaptureCommand(command string, details string) error {
	eventName := fmt.Sprintf("%s_command", command)
	return s.CaptureHostEvent(eventName, map[string]interface{}{
		"details": details,
	})
}

// CaptureAuth captures authentication events
// Reference: cli/src/index.ts:769, 788, 802, 805
func (s *DefaultService) CaptureAuth(status string, provider string) error {
	eventName := fmt.Sprintf("user.auth_%s", status)
	properties := map[string]interface{}{
		"status": status,
	}
	if provider != "" {
		properties["provider"] = provider
	}
	return s.CaptureHostEvent(eventName, properties)
}

// CaptureAuthQuickSetup captures auth quick_setup command
// Reference: cli/src/index.ts:769
func (s *DefaultService) CaptureAuthQuickSetup() error {
	return s.CaptureCommand("auth", "quick_setup")
}

// CaptureAuthInteractive captures auth interactive command
// Reference: cli/src/index.ts:788
func (s *DefaultService) CaptureAuthInteractive() error {
	return s.CaptureCommand("auth", "interactive")
}

// CaptureTaskCreated captures when a task is created
// Reference: src/services/telemetry/TelemetryService.ts:674-680
func (s *DefaultService) CaptureTaskCreated(taskID string, apiProvider string) error {
	return s.CaptureHostEvent("task.created", map[string]interface{}{
		"ulid":        taskID,
		"apiProvider": apiProvider,
	})
}

// CaptureModeFlag captures mode flag usage
// Reference: cli/src/index.ts:147-151
func (s *DefaultService) CaptureModeFlag(mode string) error {
	return s.CaptureHostEvent("mode_flag", map[string]interface{}{
		"mode": mode,
	})
}

// CaptureModelFlag captures model flag usage
// Reference: cli/src/index.ts:162
func (s *DefaultService) CaptureModelFlag(model string) error {
	return s.CaptureHostEvent("model_flag", map[string]interface{}{
		"model": model,
	})
}

// CaptureThinkingFlag captures thinking flag usage
// Reference: cli/src/index.ts:183
func (s *DefaultService) CaptureThinkingFlag() error {
	return s.CaptureHostEvent("thinking_flag", map[string]interface{}{
		"enabled": true,
	})
}

// CaptureReasoningEffortFlag captures reasoning effort flag usage
// Reference: cli/src/index.ts:192
func (s *DefaultService) CaptureReasoningEffortFlag(effort string) error {
	return s.CaptureHostEvent("reasoning_effort_flag", map[string]interface{}{
		"effort": effort,
	})
}

// CaptureMaxConsecutiveMistakesFlag captures max consecutive mistakes flag usage
// Reference: cli/src/index.ts:198
func (s *DefaultService) CaptureMaxConsecutiveMistakesFlag(count int) error {
	return s.CaptureHostEvent("max_consecutive_mistakes_flag", map[string]interface{}{
		"count": count,
	})
}

// CaptureYoloFlag captures yolo flag usage
// Reference: cli/src/index.ts:205
func (s *DefaultService) CaptureYoloFlag() error {
	return s.CaptureHostEvent("yolo_flag", map[string]interface{}{
		"enabled": true,
	})
}

// CaptureAutoApproveAllFlag captures auto-approve-all flag usage
// Reference: cli/src/index.ts:212
func (s *DefaultService) CaptureAutoApproveAllFlag() error {
	return s.CaptureHostEvent("auto_approve_all_flag", map[string]interface{}{
		"enabled": true,
	})
}

// CaptureDoubleCheckCompletionFlag captures double-check-completion flag usage
// Reference: cli/src/index.ts:218
func (s *DefaultService) CaptureDoubleCheckCompletionFlag() error {
	return s.CaptureHostEvent("double_check_completion_flag", map[string]interface{}{
		"enabled": true,
	})
}

// CapturePiped captures piped input detection
// Reference: cli/src/index.ts:604, 944
func (s *DefaultService) CapturePiped() error {
	return s.CaptureHostEvent("piped", map[string]interface{}{
		"source": "detached",
	})
}

// CaptureResumeTask captures task resumption
// Reference: cli/src/index.ts:940
func (s *DefaultService) CaptureResumeTask(withPrompt bool) error {
	details := "interactive"
	if withPrompt {
		details = "with_prompt"
	}
	return s.CaptureCommand("resume_task", details)
}

// Dispose flushes pending events and cleans up resources
// Reference: cli/src/index.ts:82-88
func (s *DefaultService) Dispose() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.provider != nil {
		// Flush any pending events
		if err := s.provider.Flush(); err != nil {
			s.logger.Warn("Failed to flush telemetry events", "error", err)
		}
		return s.provider.Dispose()
	}
	return nil
}

// getStandardAttributes returns standard attributes merged with extra properties
// Reference: src/services/telemetry/TelemetryService.ts:483-490
func (s *DefaultService) getStandardAttributes(extra map[string]interface{}) map[string]interface{} {
	attrs := map[string]interface{}{
		"extension_version": s.metadata.ExtensionVersion,
		"cline_type":        s.metadata.ClineType,
		"platform":          s.metadata.Platform,
		"platform_version":  s.metadata.PlatformVersion,
		"os_type":           s.metadata.OSType,
		"os_version":        s.metadata.OSVersion,
		"timestamp":         time.Now().UTC().Format(time.RFC3339),
	}

	if s.metadata.IsDev != "" {
		attrs["is_dev"] = s.metadata.IsDev
	}

	// Merge extra properties
	for k, v := range extra {
		attrs[k] = v
	}

	return attrs
}

// IsEnabled returns whether telemetry is enabled
func (s *DefaultService) IsEnabled() bool {
	return s.enabled && s.provider != nil && s.provider.IsEnabled()
}

// Helper functions

func getPlatformName() string {
	// Try to detect the specific platform (e.g., VSCode, Cursor, etc.)
	// For CLI, we return the terminal emulator if detectable
	if term := os.Getenv("TERM_PROGRAM"); term != "" {
		return term
	}
	if os.Getenv("VSCODE_PID") != "" || os.Getenv("VSCODE_CWD") != "" {
		return "vscode-terminal"
	}
	return "terminal"
}

func getOSVersion() string {
	// Try to get more specific OS version info
	switch runtime.GOOS {
	case "darwin":
		// Could use sw_vers command on macOS
		return runtime.GOOS + " " + runtime.GOARCH
	case "linux":
		// Could read /etc/os-release
		return runtime.GOOS + " " + runtime.GOARCH
	case "windows":
		return runtime.GOOS + " " + runtime.GOARCH
	default:
		return runtime.GOOS + " " + runtime.GOARCH
	}
}

func getIsDev() string {
	if os.Getenv("CLINE_DEV") != "" || os.Getenv("IS_DEV") != "" {
		return "true"
	}
	return ""
}

func isTelemetryEnabled(storageCtx *storage.StorageContext) bool {
	// Check if telemetry is disabled via environment variable
	if os.Getenv("CLINE_TELEMETRY_DISABLED") != "" {
		return false
	}

	// Check storage for telemetry setting
	if storageCtx != nil {
		if val, ok := storageCtx.GlobalState.Get("telemetrySetting"); ok {
			if str, ok := val.(string); ok && str == "disabled" {
				return false
			}
		}
	}

	return true
}

// NoOpService is a no-op implementation of the Service interface
// for use in tests and when telemetry is disabled
type NoOpService struct{}

// NewNoOpService creates a new no-op telemetry service
func NewNoOpService() *NoOpService {
	return &NoOpService{}
}

// CaptureHostEvent implements Service
func (n *NoOpService) CaptureHostEvent(event string, properties map[string]interface{}) error {
	return nil
}

// CaptureExtensionActivated implements Service
func (n *NoOpService) CaptureExtensionActivated() error {
	return nil
}

// CapturePlainTextMode implements Service
func (n *NoOpService) CapturePlainTextMode(reason string) error {
	return nil
}

// CaptureCommand implements Service
func (n *NoOpService) CaptureCommand(command string, details string) error {
	return nil
}

// CaptureAuth implements Service
func (n *NoOpService) CaptureAuth(status string, provider string) error {
	return nil
}

// CaptureAuthQuickSetup implements Service
func (n *NoOpService) CaptureAuthQuickSetup() error {
	return nil
}

// CaptureAuthInteractive implements Service
func (n *NoOpService) CaptureAuthInteractive() error {
	return nil
}

// CaptureTaskCreated implements Service
func (n *NoOpService) CaptureTaskCreated(taskID string, apiProvider string) error {
	return nil
}

// CaptureModeFlag implements Service
func (n *NoOpService) CaptureModeFlag(mode string) error {
	return nil
}

// CaptureModelFlag implements Service
func (n *NoOpService) CaptureModelFlag(model string) error {
	return nil
}

// CaptureThinkingFlag implements Service
func (n *NoOpService) CaptureThinkingFlag() error {
	return nil
}

// CaptureReasoningEffortFlag implements Service
func (n *NoOpService) CaptureReasoningEffortFlag(effort string) error {
	return nil
}

// CaptureMaxConsecutiveMistakesFlag implements Service
func (n *NoOpService) CaptureMaxConsecutiveMistakesFlag(count int) error {
	return nil
}

// CaptureYoloFlag implements Service
func (n *NoOpService) CaptureYoloFlag() error {
	return nil
}

// CaptureAutoApproveAllFlag implements Service
func (n *NoOpService) CaptureAutoApproveAllFlag() error {
	return nil
}

// CaptureDoubleCheckCompletionFlag implements Service
func (n *NoOpService) CaptureDoubleCheckCompletionFlag() error {
	return nil
}

// CapturePiped implements Service
func (n *NoOpService) CapturePiped() error {
	return nil
}

// CaptureResumeTask implements Service
func (n *NoOpService) CaptureResumeTask(withPrompt bool) error {
	return nil
}

// Dispose implements Service
func (n *NoOpService) Dispose() error {
	return nil
}

// MarshalJSON implements json.Marshaler for safe serialization
func (e *Event) MarshalJSON() ([]byte, error) {
	type Alias Event
	return json.Marshal(&struct {
		*Alias
		Timestamp string `json:"timestamp"`
	}{
		Alias:     (*Alias)(e),
		Timestamp: e.Timestamp.Format(time.RFC3339),
	})
}
