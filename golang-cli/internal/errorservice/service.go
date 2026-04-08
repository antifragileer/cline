// Package errorservice provides error tracking and reporting for the Cline CLI
// Reference: cli/src/index.ts lines 20, 94, 367-386, 543, 577, src/services/error/
package errorservice

import (
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"
	"time"

	"github.com/cline/cline/golang-cli/internal/storage"
)

// Service defines the error service interface
// Reference: cli/src/index.ts:20, 94, 367-386, 543, 577
type Service interface {
	// Initialize sets up the error service
	// Reference: cli/src/index.ts:94
	Initialize() error

	// CaptureException captures an exception with context
	// Reference: cli/src/index.ts:367-386
	CaptureException(err error, context map[string]string) error

	// LogException logs an exception without capturing
	LogException(err error, context map[string]string)

	// Dispose flushes pending errors before shutdown
	// Reference: cli/src/index.ts:543, 577
	Dispose() error
}

// DefaultService implements the Service interface
type DefaultService struct {
	storage     *storage.StorageContext
	logger      *slog.Logger
	enabled     bool
	initialized bool
}

// NewService creates a new error service
// Reference: cli/src/index.ts:20, 94
func NewService(storageCtx *storage.StorageContext, logger *slog.Logger) Service {
	return &DefaultService{
		storage: storageCtx,
		logger:  logger,
		enabled: isErrorReportingEnabled(storageCtx),
	}
}

// Initialize sets up the error service
// Reference: cli/src/index.ts:94
func (s *DefaultService) Initialize() error {
	if !s.enabled {
		s.logger.Debug("Error reporting is disabled")
		return nil
	}

	s.initialized = true
	s.logger.Debug("Error service initialized")

	return nil
}

// CaptureException captures an exception with context
// Reference: cli/src/index.ts:367-386
func (s *DefaultService) CaptureException(err error, context map[string]string) error {
	if !s.enabled || !s.initialized {
		// Log to stderr if not initialized or disabled
		s.logger.Error("Error captured", "error", err, "context", context)
		return nil
	}

	// Build error report
	report := map[string]interface{}{
		"error":     err.Error(),
		"stack":     string(debug.Stack()),
		"context":   context,
		"timestamp": fmt.Sprintf("%d", getTimestamp()),
	}

	// Log the error
	s.logger.Error("Exception captured",
		"error", err.Error(),
		"context", context,
	)

	// In a production implementation, this would send to Sentry or similar
	// For now, we just log it
	s.logger.Debug("Error report", "report", report)

	return nil
}

// LogException logs an exception without capturing
func (s *DefaultService) LogException(err error, context map[string]string) {
	s.logger.Error("Exception logged",
		"error", err.Error(),
		"context", context,
	)
}

// Dispose flushes pending errors before shutdown
// Reference: cli/src/index.ts:543, 577
func (s *DefaultService) Dispose() error {
	if !s.enabled || !s.initialized {
		return nil
	}

	s.logger.Debug("Disposing error service")

	// In a production implementation, this would flush any pending errors
	// to the error tracking service (Sentry, etc.)

	return nil
}

// getTimestamp returns the current timestamp in milliseconds
func getTimestamp() int64 {
	// Simple timestamp implementation - return current time in milliseconds
	return time.Now().UnixMilli()
}

// isErrorReportingEnabled checks if error reporting is enabled
func isErrorReportingEnabled(storageCtx *storage.StorageContext) bool {
	// Check if error reporting is disabled via environment variable
	if os.Getenv("CLINE_ERROR_REPORTING_DISABLED") != "" {
		return false
	}

	// Check if we're in dev mode
	if os.Getenv("CLINE_DEV") != "" || os.Getenv("IS_DEV") != "" {
		return false
	}

	return true
}

// NoopService is a no-op implementation of Service for testing
type NoopService struct{}

// Initialize implements Service
func (n *NoopService) Initialize() error { return nil }

// CaptureException implements Service
func (n *NoopService) CaptureException(err error, context map[string]string) error { return nil }

// LogException implements Service
func (n *NoopService) LogException(err error, context map[string]string) {}

// Dispose implements Service
func (n *NoopService) Dispose() error { return nil }

// Ensure DefaultService implements Service
var _ Service = (*DefaultService)(nil)
var _ Service = (*NoopService)(nil)
