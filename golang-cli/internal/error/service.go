// Package errorservice provides error tracking and logging for the Cline CLI
// Reference: src/services/error/ErrorService.ts, cli/src/index.ts lines 20, 94, 367-386, 543, 577
package errorservice

import (
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"runtime/debug"
	"sync"

	"github.com/cline/cline/golang-cli/internal/storage"
)

// Service defines the error service interface
// Reference: src/services/error/ErrorService.ts:11-96
type Service interface {
	// Initialize sets up the error service
	// Reference: cli/src/index.ts:20, src/services/error/ErrorService.ts:19-27
	Initialize() error

	// CaptureException captures an error with context
	// Reference: src/services/error/ErrorService.ts:43-45
	CaptureException(err error, context map[string]string) error

	// LogException logs an error locally
	// Reference: src/services/error/ErrorService.ts:47-50
	LogException(err error, context map[string]string)

	// LogMessage logs a message with level
	// Reference: src/services/error/ErrorService.ts:52-58
	LogMessage(message string, level string, context map[string]string)

	// IsEnabled returns whether error logging is enabled
	// Reference: src/services/error/ErrorService.ts:70-72
	IsEnabled() bool

	// Dispose flushes pending errors before shutdown
	// Reference: cli/src/index.ts:82-88, src/services/error/ErrorService.ts:93-95
	Dispose() error
}

// ErrorLevel defines the level of error logging
type ErrorLevel string

const (
	ErrorLevelAll   ErrorLevel = "all"
	ErrorLevelOff   ErrorLevel = "off"
	ErrorLevelError ErrorLevel = "error"
	ErrorLevelCrash ErrorLevel = "crash"
)

// Settings contains error service settings
// Reference: src/services/error/providers/IErrorProvider.ts:11-18
type Settings struct {
	Enabled     bool
	HostEnabled bool
	Level       ErrorLevel
}

// DefaultService implements the Service interface
type DefaultService struct {
	mu       sync.RWMutex
	enabled  bool
	settings Settings
	storage  *storage.StorageContext
	logger   *slog.Logger
	provider ErrorProvider
}

// ErrorProvider defines the interface for error tracking providers
// Reference: src/services/error/providers/IErrorProvider.ts:24-65
type ErrorProvider interface {
	CaptureException(err error, context map[string]string) error
	LogException(err error, context map[string]string)
	LogMessage(message string, level string, context map[string]string)
	IsEnabled() bool
	GetSettings() Settings
	Dispose() error
}

// NewService creates a new error service
// Reference: src/services/error/ErrorService.ts:19-27
func NewService(storageCtx *storage.StorageContext, logger *slog.Logger) *DefaultService {
	return &DefaultService{
		storage: storageCtx,
		logger:  logger,
		settings: Settings{
			Enabled:     true,
			HostEnabled: true,
			Level:       ErrorLevelAll,
		},
	}
}

// Initialize sets up the error service and provider
// Reference: cli/src/index.ts:20, src/services/error/ErrorService.ts:19-27
func (s *DefaultService) Initialize() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if error tracking is enabled
	s.enabled = s.isErrorTrackingEnabled()

	if s.enabled {
		// Initialize PostHog error provider
		provider, err := NewPostHogErrorProvider(s.logger)
		if err != nil {
			s.logger.Warn("Failed to initialize PostHog error provider", "error", err)
			// Continue without error tracking - don't fail startup
		} else {
			s.provider = provider
		}

		// Set up panic recovery
		s.setupPanicRecovery()
	}

	s.logger.Info("Error service initialized", "enabled", s.enabled)
	return nil
}

// CaptureException captures an error with context
// Reference: src/services/error/ErrorService.ts:43-45
func (s *DefaultService) CaptureException(err error, context map[string]string) error {
	if !s.enabled || s.provider == nil {
		return nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.provider.CaptureException(err, context)
}

// LogException logs an error locally
// Reference: src/services/error/ErrorService.ts:47-50
func (s *DefaultService) LogException(err error, context map[string]string) {
	if !s.enabled {
		return
	}

	// Always log to local logger
	s.logger.Error("Exception logged", "error", err, "context", context)

	if s.provider != nil {
		s.provider.LogException(err, context)
	}
}

// LogMessage logs a message with level
// Reference: src/services/error/ErrorService.ts:52-58
func (s *DefaultService) LogMessage(message string, level string, context map[string]string) {
	if !s.enabled {
		return
	}

	// Map level to slog level
	switch level {
	case "error":
		s.logger.Error(message, "context", context)
	case "warning":
		s.logger.Warn(message, "context", context)
	case "info":
		s.logger.Info(message, "context", context)
	case "debug":
		s.logger.Debug(message, "context", context)
	default:
		s.logger.Info(message, "context", context)
	}

	if s.provider != nil {
		s.provider.LogMessage(message, level, context)
	}
}

// IsEnabled returns whether error logging is enabled
// Reference: src/services/error/ErrorService.ts:70-72
func (s *DefaultService) IsEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.enabled && s.provider != nil && s.provider.IsEnabled()
}

// GetSettings returns current error settings
// Reference: src/services/error/ErrorService.ts:78-80
func (s *DefaultService) GetSettings() Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.provider != nil {
		return s.provider.GetSettings()
	}
	return s.settings
}

// Dispose flushes pending errors and cleans up
// Reference: cli/src/index.ts:82-88, src/services/error/ErrorService.ts:93-95
func (s *DefaultService) Dispose() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.provider != nil {
		return s.provider.Dispose()
	}
	return nil
}

// isErrorTrackingEnabled checks if error tracking is enabled
func (s *DefaultService) isErrorTrackingEnabled() bool {
	// Check environment variable
	if os.Getenv("CLINE_ERROR_TRACKING_DISABLED") != "" {
		return false
	}

	// Check storage setting
	if s.storage != nil {
		if val, ok := s.storage.GlobalState.Get("telemetrySetting"); ok {
			if str, ok := val.(string); ok && str == "disabled" {
				return false
			}
		}
	}

	return true
}

// setupPanicRecovery sets up panic recovery handlers
// Reference: cli/src/index.ts:367-386
func (s *DefaultService) setupPanicRecovery() {
	// Set up Go's panic recovery for the main goroutine
	// Note: In Go, we can't catch all panics globally like in Node.js
	// This is a best-effort approach for the main thread
}

// HandlePanic should be deferred in main functions to catch panics
// Usage: defer errorService.HandlePanic()
// Reference: cli/src/index.ts:367-386
func (s *DefaultService) HandlePanic() {
	if r := recover(); r != nil {
		stack := string(debug.Stack())
		err := fmt.Errorf("panic: %v\n%s", r, stack)

		// Log the panic
		s.LogException(err, map[string]string{
			"type":   "panic",
			"stack":  stack,
			"source": "runtime",
		})

		// Try to capture it remotely
		_ = s.CaptureException(err, map[string]string{
			"type":   "panic",
			"source": "runtime",
		})

		// Re-panic to maintain normal Go behavior
		panic(r)
	}
}

// ClineError represents a Cline-specific error
// Reference: src/services/error/ClineError.ts
type ClineError struct {
	Message    string
	Stack      string
	Name       string
	ModelID    string
	ProviderID string
	Original   error
}

// Error implements the error interface
func (e *ClineError) Error() string {
	return e.Message
}

// NewClineError creates a new Cline error
func NewClineError(message string, original error) *ClineError {
	stack := string(debug.Stack())
	return &ClineError{
		Message:  message,
		Stack:    stack,
		Name:     "ClineError",
		Original: original,
	}
}

// WrapError wraps an error with context
func WrapError(err error, context string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", context, err)
}

// IsCriticalError determines if an error is critical
func IsCriticalError(err error) bool {
	if err == nil {
		return false
	}

	// Check for critical error types
	// This can be extended based on specific error types
	switch err.(type) {
	case *ClineError:
		return true
	default:
		// Check error message for critical indicators
		msg := err.Error()
		criticalIndicators := []string{
			"panic",
			"fatal",
			"crash",
			"out of memory",
		}
		for _, indicator := range criticalIndicators {
			if len(msg) >= len(indicator) && msg[:len(indicator)] == indicator {
				return true
			}
		}
		return false
	}
}

// GetErrorStack returns the stack trace for an error
func GetErrorStack(err error) string {
	if err == nil {
		return ""
	}

	// If it's a ClineError, return its stack
	if clineErr, ok := err.(*ClineError); ok {
		return clineErr.Stack
	}

	// Otherwise, get current stack
	return string(debug.Stack())
}

// GetRuntimeInfo returns runtime information for error context
func GetRuntimeInfo() map[string]string {
	return map[string]string{
		"go_version":    runtime.Version(),
		"go_os":         runtime.GOOS,
		"go_arch":       runtime.GOARCH,
		"num_cpu":       fmt.Sprintf("%d", runtime.NumCPU()),
		"num_goroutine": fmt.Sprintf("%d", runtime.NumGoroutine()),
	}
}

// SafeExecute executes a function with error recovery
// Returns the error if one occurred, nil otherwise
func SafeExecute(fn func() error, errorService Service) error {
	defer func() {
		if r := recover(); r != nil {
			stack := string(debug.Stack())
			err := fmt.Errorf("panic in SafeExecute: %v\n%s", r, stack)

			if errorService != nil {
				errorService.LogException(err, map[string]string{
					"type":   "panic",
					"source": "SafeExecute",
				})
				_ = errorService.CaptureException(err, map[string]string{
					"type":   "panic",
					"source": "SafeExecute",
				})
			}

			// Re-panic
			panic(r)
		}
	}()

	return fn()
}
