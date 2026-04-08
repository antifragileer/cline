// Package host provides gRPC client functionality for communicating with the Cline extension.
package host

import (
	"context"
	"errors"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCErrorCategory classifies gRPC errors for appropriate handling
type GRPCErrorCategory int

const (
	// CategoryOK indicates no error
	CategoryOK GRPCErrorCategory = iota
	// CategoryTransient indicates a transient error that may succeed on retry
	CategoryTransient
	// CategoryTimeout indicates a timeout error
	CategoryTimeout
	// CategoryUnavailable indicates the service is unavailable
	CategoryUnavailable
	// CategoryInvalidArgument indicates invalid arguments were provided
	CategoryInvalidArgument
	// CategoryNotFound indicates the requested resource was not found
	CategoryNotFound
	// CategoryAlreadyExists indicates the resource already exists
	CategoryAlreadyExists
	// CategoryPermissionDenied indicates insufficient permissions
	CategoryPermissionDenied
	// CategoryUnauthenticated indicates authentication failed
	CategoryUnauthenticated
	// CategoryResourceExhausted indicates resource limits were hit
	CategoryResourceExhausted
	// CategoryFailedPrecondition indicates a precondition was not met
	CategoryFailedPrecondition
	// CategoryAborted indicates the operation was aborted
	CategoryAborted
	// CategoryOutOfRange indicates a value is out of range
	CategoryOutOfRange
	// CategoryUnimplemented indicates the operation is not implemented
	CategoryUnimplemented
	// CategoryInternal indicates an internal server error
	CategoryInternal
	// CategoryUnknown indicates an unknown error
	CategoryUnknown
	// CategoryConnectionClosed indicates the connection was closed
	CategoryConnectionClosed
)

func (c GRPCErrorCategory) String() string {
	switch c {
	case CategoryOK:
		return "ok"
	case CategoryTransient:
		return "transient"
	case CategoryTimeout:
		return "timeout"
	case CategoryUnavailable:
		return "unavailable"
	case CategoryInvalidArgument:
		return "invalid_argument"
	case CategoryNotFound:
		return "not_found"
	case CategoryAlreadyExists:
		return "already_exists"
	case CategoryPermissionDenied:
		return "permission_denied"
	case CategoryUnauthenticated:
		return "unauthenticated"
	case CategoryResourceExhausted:
		return "resource_exhausted"
	case CategoryFailedPrecondition:
		return "failed_precondition"
	case CategoryAborted:
		return "aborted"
	case CategoryOutOfRange:
		return "out_of_range"
	case CategoryUnimplemented:
		return "unimplemented"
	case CategoryInternal:
		return "internal"
	case CategoryUnknown:
		return "unknown"
	case CategoryConnectionClosed:
		return "connection_closed"
	default:
		return "unknown"
	}
}

// IsRetryable returns true if the error category indicates a retryable error
func (c GRPCErrorCategory) IsRetryable() bool {
	switch c {
	case CategoryTransient, CategoryTimeout, CategoryUnavailable,
		CategoryAborted, CategoryResourceExhausted:
		return true
	default:
		return false
	}
}

// ClassifyGRPCError categorizes a gRPC error
func ClassifyGRPCError(err error) GRPCErrorCategory {
	if err == nil {
		return CategoryOK
	}

	// Check for context errors
	if errors.Is(err, context.DeadlineExceeded) {
		return CategoryTimeout
	}
	if errors.Is(err, context.Canceled) {
		return CategoryAborted
	}

	// Extract gRPC status
	st, ok := status.FromError(err)
	if !ok {
		// Check for connection closed errors
		if errors.Is(err, context.Canceled) || err.Error() == "transport is closing" || err.Error() == "grpc: the client connection is closing" {
			return CategoryConnectionClosed
		}
		return CategoryUnknown
	}

	switch st.Code() {
	case codes.OK:
		return CategoryOK
	case codes.Canceled:
		return CategoryAborted
	case codes.Unknown:
		return CategoryUnknown
	case codes.InvalidArgument:
		return CategoryInvalidArgument
	case codes.DeadlineExceeded:
		return CategoryTimeout
	case codes.NotFound:
		return CategoryNotFound
	case codes.AlreadyExists:
		return CategoryAlreadyExists
	case codes.PermissionDenied:
		return CategoryPermissionDenied
	case codes.ResourceExhausted:
		return CategoryResourceExhausted
	case codes.FailedPrecondition:
		return CategoryFailedPrecondition
	case codes.Aborted:
		return CategoryAborted
	case codes.OutOfRange:
		return CategoryOutOfRange
	case codes.Unimplemented:
		return CategoryUnimplemented
	case codes.Internal:
		return CategoryInternal
	case codes.Unavailable:
		return CategoryUnavailable
	case codes.DataLoss:
		return CategoryInternal
	case codes.Unauthenticated:
		return CategoryUnauthenticated
	default:
		return CategoryUnknown
	}
}

// GRPCErrorInfo provides detailed information about a gRPC error
type GRPCErrorInfo struct {
	// Original error
	Error error

	// Category of the error
	Category GRPCErrorCategory

	// gRPC status code (if applicable)
	Code codes.Code

	// Error message
	Message string

	// Retryable indicates if the operation can be retried
	Retryable bool

	// SuggestedRetryDelay is the recommended delay before retrying
	SuggestedRetryDelay time.Duration
}

// AnalyzeGRPCError provides detailed analysis of a gRPC error
func AnalyzeGRPCError(err error) GRPCErrorInfo {
	if err == nil {
		return GRPCErrorInfo{
			Category: CategoryOK,
			Retryable: false,
		}
	}

	category := ClassifyGRPCError(err)
	
	// Extract gRPC status
	var code codes.Code
	var message string
	if st, ok := status.FromError(err); ok {
		code = st.Code()
		message = st.Message()
	} else {
		code = codes.Unknown
		message = err.Error()
	}

	// Calculate suggested retry delay
	retryDelay := calculateRetryDelay(category, err)

	return GRPCErrorInfo{
		Error:               err,
		Category:            category,
		Code:                code,
		Message:             message,
		Retryable:           category.IsRetryable(),
		SuggestedRetryDelay: retryDelay,
	}
}

// calculateRetryDelay calculates a suggested retry delay based on error category
func calculateRetryDelay(category GRPCErrorCategory, err error) time.Duration {
	switch category {
	case CategoryTimeout:
		return 5 * time.Second
	case CategoryUnavailable:
		// Use exponential backoff for unavailable
		return 2 * time.Second
	case CategoryResourceExhausted:
		// Longer delay for rate limiting
		return 10 * time.Second
	case CategoryTransient:
		return 1 * time.Second
	case CategoryAborted:
		return 500 * time.Millisecond
	default:
		return 0 // Not retryable
	}
}

// GRPCErrorHandler handles gRPC errors with appropriate actions
type GRPCErrorHandler struct {
	// Callbacks
	onRetryableError func(error, time.Duration)
	onFatalError     func(error)
	onReconnect      func()

	// Retry configuration
	maxRetries      int
	initialBackoff  time.Duration
	maxBackoff      time.Duration
	backoffMultiplier float64
}

// GRPCErrorHandlerConfig contains configuration for the error handler
type GRPCErrorHandlerConfig struct {
	MaxRetries        int
	InitialBackoff    time.Duration
	MaxBackoff        time.Duration
	BackoffMultiplier float64
}

// DefaultGRPCErrorHandlerConfig returns default configuration
func DefaultGRPCErrorHandlerConfig() GRPCErrorHandlerConfig {
	return GRPCErrorHandlerConfig{
		MaxRetries:        5,
		InitialBackoff:    1 * time.Second,
		MaxBackoff:        60 * time.Second,
		BackoffMultiplier: 2.0,
	}
}

// NewGRPCErrorHandler creates a new gRPC error handler
func NewGRPCErrorHandler(config GRPCErrorHandlerConfig) *GRPCErrorHandler {
	if config.MaxRetries <= 0 {
		config.MaxRetries = DefaultGRPCErrorHandlerConfig().MaxRetries
	}
	if config.InitialBackoff <= 0 {
		config.InitialBackoff = DefaultGRPCErrorHandlerConfig().InitialBackoff
	}
	if config.MaxBackoff <= 0 {
		config.MaxBackoff = DefaultGRPCErrorHandlerConfig().MaxBackoff
	}
	if config.BackoffMultiplier <= 0 {
		config.BackoffMultiplier = DefaultGRPCErrorHandlerConfig().BackoffMultiplier
	}

	return &GRPCErrorHandler{
		maxRetries:        config.MaxRetries,
		initialBackoff:    config.InitialBackoff,
		maxBackoff:        config.MaxBackoff,
		backoffMultiplier: config.BackoffMultiplier,
	}
}

// SetOnRetryableError sets the callback for retryable errors
func (h *GRPCErrorHandler) SetOnRetryableError(fn func(error, time.Duration)) {
	h.onRetryableError = fn
}

// SetOnFatalError sets the callback for fatal errors
func (h *GRPCErrorHandler) SetOnFatalError(fn func(error)) {
	h.onFatalError = fn
}

// SetOnReconnect sets the callback for reconnect events
func (h *GRPCErrorHandler) SetOnReconnect(fn func()) {
	h.onReconnect = fn
}

// HandleError handles a gRPC error and returns true if it was handled successfully
func (h *GRPCErrorHandler) HandleError(err error, retryCount int) (shouldRetry bool, retryDelay time.Duration) {
	info := AnalyzeGRPCError(err)

	// Not an error
	if info.Category == CategoryOK {
		return false, 0
	}

	// Non-retryable errors
	if !info.Retryable {
		if h.onFatalError != nil {
			h.onFatalError(err)
		}
		return false, 0
	}

	// Check if we've exceeded max retries
	if retryCount >= h.maxRetries {
		if h.onFatalError != nil {
			h.onFatalError(fmt.Errorf("max retries (%d) exceeded: %w", h.maxRetries, err))
		}
		return false, 0
	}

	// Calculate backoff
	retryDelay = h.calculateBackoff(retryCount, info.SuggestedRetryDelay)

	// Notify of retryable error
	if h.onRetryableError != nil {
		h.onRetryableError(err, retryDelay)
	}

	return true, retryDelay
}

// calculateBackoff calculates exponential backoff
func (h *GRPCErrorHandler) calculateBackoff(retryCount int, suggestedDelay time.Duration) time.Duration {
	// Use the larger of suggested delay or calculated backoff
	backoff := h.initialBackoff
	for i := 0; i < retryCount; i++ {
		backoff = time.Duration(float64(backoff) * h.backoffMultiplier)
		if backoff > h.maxBackoff {
			backoff = h.maxBackoff
			break
		}
	}

	if suggestedDelay > backoff {
		return suggestedDelay
	}
	return backoff
}

// IsRetryableError returns true if the error is retryable
func IsRetryableError(err error) bool {
	return ClassifyGRPCError(err).IsRetryable()
}

// FormatGRPCError returns a human-readable error message
func FormatGRPCError(err error) string {
	if err == nil {
		return "no error"
	}

	info := AnalyzeGRPCError(err)

	switch info.Category {
	case CategoryTimeout:
		return fmt.Sprintf("Request timed out: %s", info.Message)
	case CategoryUnavailable:
		return fmt.Sprintf("Service unavailable: %s. Please check if the Cline extension is running.", info.Message)
	case CategoryConnectionClosed:
		return fmt.Sprintf("Connection closed: %s. Attempting to reconnect...", info.Message)
	case CategoryPermissionDenied:
		return fmt.Sprintf("Permission denied: %s", info.Message)
	case CategoryUnauthenticated:
		return fmt.Sprintf("Authentication failed: %s", info.Message)
	case CategoryResourceExhausted:
		return fmt.Sprintf("Rate limit exceeded: %s. Please wait before retrying.", info.Message)
	case CategoryNotFound:
		return fmt.Sprintf("Not found: %s", info.Message)
	case CategoryInvalidArgument:
		return fmt.Sprintf("Invalid argument: %s", info.Message)
	case CategoryInternal:
		return fmt.Sprintf("Internal error: %s", info.Message)
	default:
		return fmt.Sprintf("Error (%s): %s", info.Category, info.Message)
	}
}

// ErrorHandlerWithContext wraps error handling with context awareness
type ErrorHandlerWithContext struct {
	handler *GRPCErrorHandler
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewErrorHandlerWithContext creates a new error handler with context
func NewErrorHandlerWithContext(parent context.Context, handler *GRPCErrorHandler) *ErrorHandlerWithContext {
	ctx, cancel := context.WithCancel(parent)
	return &ErrorHandlerWithContext{
		handler: handler,
		ctx:     ctx,
		cancel:  cancel,
	}
}

// HandleError handles an error with context awareness
func (ehc *ErrorHandlerWithContext) HandleError(err error, retryCount int) (shouldRetry bool, retryDelay time.Duration) {
	// Check if context is cancelled
	select {
	case <-ehc.ctx.Done():
		return false, 0
	default:
	}

	return ehc.handler.HandleError(err, retryCount)
}

// Stop stops the error handler context
func (ehc *ErrorHandlerWithContext) Stop() {
	ehc.cancel()
}

// GetContext returns the handler context
func (ehc *ErrorHandlerWithContext) GetContext() context.Context {
	return ehc.ctx
}