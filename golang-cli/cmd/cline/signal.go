// Package main provides signal handling for the Cline CLI
// Reference: cli/src/index.ts lines 403-475
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/cline/cline/golang-cli/internal/errorservice"
	"github.com/cline/cline/golang-cli/internal/telemetry"
)

// SignalHandler manages signal handling and graceful shutdown
// Reference: cli/src/index.ts:403-475
type SignalHandler struct {
	telemetry    telemetry.Service
	errorService errorservice.Service
	logger       *slog.Logger
	cancelFunc   context.CancelFunc
	shutdownChan chan os.Signal
}

// NewSignalHandler creates a new signal handler
func NewSignalHandler(
	telemetry telemetry.Service,
	errorService errorservice.Service,
	logger *slog.Logger,
	cancelFunc context.CancelFunc,
) *SignalHandler {
	return &SignalHandler{
		telemetry:    telemetry,
		errorService: errorService,
		logger:       logger,
		cancelFunc:   cancelFunc,
		shutdownChan: make(chan os.Signal, 1),
	}
}

// Setup sets up signal handling
// Reference: cli/src/index.ts:403-475
func (h *SignalHandler) Setup() {
	// Create signal channel
	sigChan := make(chan os.Signal, 1)

	// Register for SIGINT and SIGTERM
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start signal handling goroutine
	go h.handleSignals(sigChan)
}

// handleSignals handles incoming signals
func (h *SignalHandler) handleSignals(sigChan chan os.Signal) {
	// Wait for signal
	sig := <-sigChan

	h.logger.Info("Received signal, shutting down...", "signal", sig.String())

	// Call cancel function to stop ongoing operations
	if h.cancelFunc != nil {
		h.cancelFunc()
	}

	// Dispose telemetry service
	if h.telemetry != nil {
		h.logger.Debug("Disposing telemetry service")
		if err := h.telemetry.Dispose(); err != nil {
			h.logger.Warn("Failed to dispose telemetry service", "error", err)
		}
	}

	// Dispose error service
	if h.errorService != nil {
		h.logger.Debug("Disposing error service")
		if err := h.errorService.Dispose(); err != nil {
			h.logger.Warn("Failed to dispose error service", "error", err)
		}
	}

	h.logger.Info("Shutdown complete")

	// Notify that shutdown is complete
	if h.shutdownChan != nil {
		h.shutdownChan <- sig
		close(h.shutdownChan)
	}

	// Exit with appropriate code
	switch sig {
	case syscall.SIGINT:
		os.Exit(130) // 128 + SIGINT (2)
	case syscall.SIGTERM:
		os.Exit(143) // 128 + SIGTERM (15)
	default:
		os.Exit(1)
	}
}

// WaitForShutdown blocks until shutdown is complete
func (h *SignalHandler) WaitForShutdown() {
	if h.shutdownChan != nil {
		<-h.shutdownChan
	}
}

// Cleanup performs cleanup without signal handling
// Useful for testing or when signals are handled elsewhere
func (h *SignalHandler) Cleanup() {
	// Dispose telemetry service
	if h.telemetry != nil {
		if err := h.telemetry.Dispose(); err != nil {
			h.logger.Warn("Failed to dispose telemetry service", "error", err)
		}
	}

	// Dispose error service
	if h.errorService != nil {
		if err := h.errorService.Dispose(); err != nil {
			h.logger.Warn("Failed to dispose error service", "error", err)
		}
	}
}

// SetupSignalHandling is a convenience function to set up signal handling
// Returns a cleanup function that should be called on normal exit
func SetupSignalHandling(
	telemetry telemetry.Service,
	errorService errorservice.Service,
	logger *slog.Logger,
	cancelFunc context.CancelFunc,
) func() {
	handler := NewSignalHandler(telemetry, errorService, logger, cancelFunc)
	handler.Setup()

	// Return cleanup function for normal exit
	return func() {
		handler.Cleanup()
	}
}