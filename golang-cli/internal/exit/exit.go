// Package exit provides exit code definitions and signal handling for the Cline CLI.
// It ensures consistent exit behavior matching the TypeScript CLI.
package exit

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// Exit codes matching the TypeScript CLI behavior
const (
	// Success indicates successful execution
	Success = 0
	// GeneralError indicates a general error occurred
	GeneralError = 1
	// InvalidArguments indicates invalid command-line arguments
	InvalidArguments = 2
	// CommandNotFound indicates the specified command was not found
	CommandNotFound = 127
	// Interrupted indicates the process was interrupted (SIGINT)
	Interrupted = 130
	// Timeout indicates the operation timed out
	Timeout = 124
	// PermissionDenied indicates permission was denied
	PermissionDenied = 126
	// ConfigurationError indicates a configuration error
	ConfigurationError = 78
	// ConnectionError indicates a connection error (gRPC, API)
	ConnectionError = 69
	// TaskFailed indicates the task execution failed
	TaskFailed = 70
)

// Code represents an exit code
type Code int

// String returns the human-readable description of the exit code
func (c Code) String() string {
	switch c {
	case Success:
		return "success"
	case GeneralError:
		return "general error"
	case InvalidArguments:
		return "invalid arguments"
	case CommandNotFound:
		return "command not found"
	case Interrupted:
		return "interrupted"
	case Timeout:
		return "timeout"
	case PermissionDenied:
		return "permission denied"
	case ConfigurationError:
		return "configuration error"
	case ConnectionError:
		return "connection error"
	case TaskFailed:
		return "task failed"
	default:
		return "unknown"
	}
}

// Handler manages exit codes and signal handling
type Handler struct {
	mu          sync.RWMutex
	exitCode    Code
	interrupted bool
	cancelFunc  context.CancelFunc
	sigChan     chan os.Signal
	cleanupFns  []func()
	once        sync.Once
}

// NewHandler creates a new exit handler
func NewHandler() *Handler {
	return &Handler{
		exitCode:   Success,
		sigChan:    make(chan os.Signal, 1),
		cleanupFns: make([]func(), 0),
	}
}

// SetExitCode sets the exit code
func (h *Handler) SetExitCode(code Code) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.exitCode = code
}

// GetExitCode returns the current exit code
func (h *Handler) GetExitCode() Code {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.exitCode
}

// Exit performs cleanup and exits with the current exit code
func (h *Handler) Exit() {
	h.once.Do(func() {
		h.runCleanup()
		os.Exit(int(h.GetExitCode()))
	})
}

// ExitWithCode sets the exit code and exits immediately
func (h *Handler) ExitWithCode(code Code) {
	h.SetExitCode(code)
	h.Exit()
}

// Run executes a function and handles exit codes and signals
func (h *Handler) Run(ctx context.Context, fn func(context.Context) error) error {
	// Create a cancellable context
	ctx, cancel := context.WithCancel(ctx)
	h.cancelFunc = cancel
	defer cancel()

	// Setup signal handling
	h.setupSignals()

	// Run the function in a goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- fn(ctx)
	}()

	// Wait for completion or signal
	select {
	case err := <-errChan:
		if err != nil {
			h.SetExitCode(MapErrorToCode(err))
			return err
		}
		return nil
	case sig := <-h.sigChan:
		h.handleSignal(sig)
		return nil
	case <-ctx.Done():
		h.SetExitCode(Timeout)
		return ctx.Err()
	}
}

// setupSignals configures signal handling
func (h *Handler) setupSignals() {
	signal.Notify(h.sigChan, syscall.SIGINT, syscall.SIGTERM)
}

// handleSignal processes received signals
func (h *Handler) handleSignal(sig os.Signal) {
	switch sig {
	case syscall.SIGINT:
		h.interrupted = true
		h.SetExitCode(Interrupted)
		if h.cancelFunc != nil {
			h.cancelFunc()
		}
	case syscall.SIGTERM:
		h.SetExitCode(Interrupted)
		if h.cancelFunc != nil {
			h.cancelFunc()
		}
	}
}

// RegisterCleanup registers a cleanup function to run on exit
func (h *Handler) RegisterCleanup(fn func()) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cleanupFns = append(h.cleanupFns, fn)
}

// runCleanup executes all registered cleanup functions
func (h *Handler) runCleanup() {
	h.mu.RLock()
	fns := make([]func(), len(h.cleanupFns))
	copy(fns, h.cleanupFns)
	h.mu.RUnlock()

	// Run cleanup functions in reverse order (LIFO)
	for i := len(fns) - 1; i >= 0; i-- {
		if fns[i] != nil {
			fns[i]()
		}
	}
}

// IsInterrupted returns true if the process was interrupted
func (h *Handler) IsInterrupted() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.interrupted
}

// ContextWithHandler creates a context with the exit handler
func ContextWithHandler(ctx context.Context, handler *Handler) context.Context {
	return context.WithValue(ctx, handlerKey{}, handler)
}

// HandlerFromContext retrieves the exit handler from context
func HandlerFromContext(ctx context.Context) *Handler {
	if handler, ok := ctx.Value(handlerKey{}).(*Handler); ok {
		return handler
	}
	return nil
}

type handlerKey struct{}

// MapErrorToCode maps errors to appropriate exit codes
func MapErrorToCode(err error) Code {
	if err == nil {
		return Success
	}

	errStr := err.Error()

	// Check for specific error patterns
	switch {
	case contains(errStr, "permission denied"):
		return PermissionDenied
	case contains(errStr, "connection refused"), contains(errStr, "connection reset"):
		return ConnectionError
	case contains(errStr, "timeout"), contains(errStr, "deadline exceeded"):
		return Timeout
	case contains(errStr, "configuration"), contains(errStr, "config"):
		return ConfigurationError
	case contains(errStr, "task failed"), contains(errStr, "execution failed"):
		return TaskFailed
	case contains(errStr, "command not found"), contains(errStr, "executable file not found"):
		return CommandNotFound
	case contains(errStr, "invalid argument"), contains(errStr, "flag provided but not defined"):
		return InvalidArguments
	default:
		return GeneralError
	}
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(s) > len(substr) && (containsAt(s, substr, 0) ||
			containsAt(s, substr, 1) ||
			containsAt(s, substr, len(s)-len(substr))))
}

// containsAt checks if substr appears at position i in s
func containsAt(s, substr string, i int) bool {
	if i < 0 || i+len(substr) > len(s) {
		return false
	}
	for j := 0; j < len(substr); j++ {
		if toLower(s[i+j]) != toLower(substr[j]) {
			return false
		}
	}
	return true
}

// toLower converts a byte to lowercase
func toLower(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + ('a' - 'A')
	}
	return b
}

// IsSuccess returns true if the exit code indicates success
func IsSuccess(code Code) bool {
	return code == Success
}

// IsError returns true if the exit code indicates an error
func IsError(code Code) bool {
	return code != Success
}

// IsSignal returns true if the exit code indicates a signal
func IsSignal(code Code) bool {
	return code >= 128 && code <= 165 // Signal range
}