// Package grpc provides error handling and retry logic for gRPC connections.
package grpc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RetryPolicy defines the retry behavior
type RetryPolicy struct {
	// MaxRetries is the maximum number of retry attempts
	MaxRetries int
	// InitialBackoff is the initial backoff duration
	InitialBackoff time.Duration
	// MaxBackoff is the maximum backoff duration
	MaxBackoff time.Duration
	// BackoffMultiplier is the multiplier for exponential backoff
	BackoffMultiplier float64
	// RetryableCodes are the gRPC status codes that should trigger a retry
	RetryableCodes []codes.Code
	// NonRetryableErrors are errors that should never be retried
	NonRetryableErrors []error
	// Jitter adds random jitter to backoff to prevent thundering herd
	Jitter bool
	// JitterFactor is the maximum jitter as a fraction of the backoff (0.0-1.0)
	JitterFactor float64
}

// DefaultRetryPolicy returns a default retry policy
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxRetries:        5,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        30 * time.Second,
		BackoffMultiplier: 2.0,
		RetryableCodes: []codes.Code{
			codes.Unavailable,
			codes.ResourceExhausted,
			codes.Aborted,
			codes.DeadlineExceeded,
		},
		Jitter:       true,
		JitterFactor: 0.2,
	}
}

// IsRetryable checks if an error should be retried
func (p *RetryPolicy) IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Check non-retryable errors
	for _, nonRetryable := range p.NonRetryableErrors {
		if errors.Is(err, nonRetryable) {
			return false
		}
	}

	// Check if it's an EOF (stream closed)
	if err == io.EOF {
		return true
	}

	// Check context cancellation
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Check gRPC status codes
	st, ok := status.FromError(err)
	if !ok {
		// Not a gRPC error, retry if it's a network error
		return isNetworkError(err)
	}

	for _, code := range p.RetryableCodes {
		if st.Code() == code {
			return true
		}
	}

	return false
}

// CalculateBackoff calculates the backoff duration for a given attempt
func (p *RetryPolicy) CalculateBackoff(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}

	// Calculate exponential backoff
	backoff := float64(p.InitialBackoff) * math.Pow(p.BackoffMultiplier, float64(attempt-1))

	// Cap at max backoff
	if backoff > float64(p.MaxBackoff) {
		backoff = float64(p.MaxBackoff)
	}

	// Add jitter
	if p.Jitter {
		jitter := backoff * p.JitterFactor * (rand.Float64()*2 - 1) // Random between -factor and +factor
		backoff += jitter
	}

	return time.Duration(backoff)
}

// ErrorHandler handles errors with retry logic
type ErrorHandler struct {
	policy         *RetryPolicy
	attempts       map[string]int
	attemptsMu     sync.RWMutex
	onRetry        func(attempt int, err error, backoff time.Duration)
	onMaxRetries   func(err error)
	onSuccess      func(attempts int)
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(policy *RetryPolicy) *ErrorHandler {
	if policy == nil {
		policy = DefaultRetryPolicy()
	}

	return &ErrorHandler{
		policy:   policy,
		attempts: make(map[string]int),
	}
}

// SetRetryCallback sets a callback for retry attempts
func (h *ErrorHandler) SetRetryCallback(cb func(attempt int, err error, backoff time.Duration)) {
	h.onRetry = cb
}

// SetMaxRetriesCallback sets a callback for when max retries are exceeded
func (h *ErrorHandler) SetMaxRetriesCallback(cb func(err error)) {
	h.onMaxRetries = cb
}

// SetSuccessCallback sets a callback for successful operations
func (h *ErrorHandler) SetSuccessCallback(cb func(attempts int)) {
	h.onSuccess = cb
}

// Execute executes a function with retry logic
func (h *ErrorHandler) Execute(ctx context.Context, operationID string, fn func() error) error {
	h.attemptsMu.Lock()
	h.attempts[operationID] = 0
	h.attemptsMu.Unlock()

	for {
		err := fn()

		if err == nil {
			// Success
			h.attemptsMu.RLock()
			attempts := h.attempts[operationID]
			h.attemptsMu.RUnlock()

			if h.onSuccess != nil {
				h.onSuccess(attempts)
			}

			h.attemptsMu.Lock()
			delete(h.attempts, operationID)
			h.attemptsMu.Unlock()

			return nil
		}

		// Check if we should retry
		if !h.policy.IsRetryable(err) {
			return err // Not retryable
		}

		h.attemptsMu.Lock()
		h.attempts[operationID]++
		attempt := h.attempts[operationID]
		h.attemptsMu.Unlock()

		// Check max retries
		if attempt > h.policy.MaxRetries {
			if h.onMaxRetries != nil {
				h.onMaxRetries(err)
			}

			h.attemptsMu.Lock()
			delete(h.attempts, operationID)
			h.attemptsMu.Unlock()

			return fmt.Errorf("max retries (%d) exceeded: %w", h.policy.MaxRetries, err)
		}

		// Calculate backoff
		backoff := h.policy.CalculateBackoff(attempt)

		if h.onRetry != nil {
			h.onRetry(attempt, err, backoff)
		}

		// Wait for backoff or context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
			// Continue to next attempt
		}
	}
}

// ExecuteWithResult executes a function that returns a result with retry logic
func (h *ErrorHandler) ExecuteWithResult(ctx context.Context, operationID string, fn func() (interface{}, error)) (interface{}, error) {
	var result interface{}
	err := h.Execute(ctx, operationID, func() error {
		var err error
		result, err = fn()
		return err
	})
	return result, err
}

// GetAttempts returns the number of attempts for an operation
func (h *ErrorHandler) GetAttempts(operationID string) int {
	h.attemptsMu.RLock()
	defer h.attemptsMu.RUnlock()
	return h.attempts[operationID]
}

// ResetAttempts resets the attempt counter for an operation
func (h *ErrorHandler) ResetAttempts(operationID string) {
	h.attemptsMu.Lock()
	defer h.attemptsMu.Unlock()
	delete(h.attempts, operationID)
}

// StreamErrorHandler handles errors in streaming contexts
type StreamErrorHandler struct {
	errorHandler *ErrorHandler
	reconnectFn  func() error
	onReconnect  func()
}

// NewStreamErrorHandler creates a new stream error handler
func NewStreamErrorHandler(errorHandler *ErrorHandler, reconnectFn func() error) *StreamErrorHandler {
	return &StreamErrorHandler{
		errorHandler: errorHandler,
		reconnectFn:  reconnectFn,
	}
}

// SetReconnectCallback sets a callback for successful reconnections
func (h *StreamErrorHandler) SetReconnectCallback(cb func()) {
	h.onReconnect = cb
}

// HandleRecvError handles errors from stream.Recv()
func (h *StreamErrorHandler) HandleRecvError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}

	// EOF is expected when stream closes normally
	if err == io.EOF {
		return err
	}

	// Check if we should try to reconnect
	if !h.errorHandler.policy.IsRetryable(err) {
		return err
	}

	// Attempt to reconnect
	reconnectErr := h.errorHandler.Execute(ctx, "stream_reconnect", h.reconnectFn)
	if reconnectErr != nil {
		return fmt.Errorf("failed to reconnect after error (%v): %w", err, reconnectErr)
	}

	if h.onReconnect != nil {
		h.onReconnect()
	}

	return nil
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	// Threshold is the number of failures before opening the circuit
	Threshold int
	// ResetTimeout is the duration before attempting to close the circuit
	ResetTimeout time.Duration

	failures     int
	lastFailure  time.Time
	state        CircuitState
	stateMu      sync.RWMutex
	onStateChange func(oldState, newState CircuitState)
}

// CircuitState represents the state of the circuit breaker
type CircuitState int

const (
	// CircuitClosed means the circuit is closed and requests pass through
	CircuitClosed CircuitState = iota
	// CircuitOpen means the circuit is open and requests fail fast
	CircuitOpen
	// CircuitHalfOpen means the circuit is testing if it can be closed
	CircuitHalfOpen
)

func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(threshold int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		Threshold:    threshold,
		ResetTimeout: resetTimeout,
		state:        CircuitClosed,
	}
}

// SetStateChangeCallback sets a callback for state changes
func (cb *CircuitBreaker) SetStateChangeCallback(fn func(oldState, newState CircuitState)) {
	cb.onStateChange = fn
}

// CanExecute returns true if the request can be executed
func (cb *CircuitBreaker) CanExecute() bool {
	cb.stateMu.Lock()
	defer cb.stateMu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		// Check if we should transition to half-open
		if time.Since(cb.lastFailure) > cb.ResetTimeout {
			cb.transitionTo(CircuitHalfOpen)
			return true
		}
		return false
	case CircuitHalfOpen:
		return true
	default:
		return false
	}
}

// RecordSuccess records a successful execution
func (cb *CircuitBreaker) RecordSuccess() {
	cb.stateMu.Lock()
	defer cb.stateMu.Unlock()

	if cb.state == CircuitHalfOpen {
		cb.failures = 0
		cb.transitionTo(CircuitClosed)
	} else {
		cb.failures = 0
	}
}

// RecordFailure records a failed execution
func (cb *CircuitBreaker) RecordFailure() {
	cb.stateMu.Lock()
	defer cb.stateMu.Unlock()

	cb.failures++
	cb.lastFailure = time.Now()

	if cb.state == CircuitHalfOpen {
		cb.transitionTo(CircuitOpen)
	} else if cb.failures >= cb.Threshold {
		cb.transitionTo(CircuitOpen)
	}
}

// GetState returns the current circuit state
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.stateMu.RLock()
	defer cb.stateMu.RUnlock()
	return cb.state
}

func (cb *CircuitBreaker) transitionTo(newState CircuitState) {
	if cb.state != newState && cb.onStateChange != nil {
		cb.onStateChange(cb.state, newState)
	}
	cb.state = newState
}

// Execute executes a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(fn func() error) error {
	if !cb.CanExecute() {
		return errors.New("circuit breaker is open")
	}

	err := fn()
	if err != nil {
		cb.RecordFailure()
		return err
	}

	cb.RecordSuccess()
	return nil
}

// RecoveryManager manages recovery from errors
type RecoveryManager struct {
	errorHandler    *ErrorHandler
	circuitBreaker  *CircuitBreaker
	healthMonitor   *HealthMonitor
	onRecoveryStart func()
	onRecoveryEnd   func(success bool, err error)
}

// NewRecoveryManager creates a new recovery manager
func NewRecoveryManager(errorHandler *ErrorHandler, circuitBreaker *CircuitBreaker, healthMonitor *HealthMonitor) *RecoveryManager {
	return &RecoveryManager{
		errorHandler:   errorHandler,
		circuitBreaker: circuitBreaker,
		healthMonitor:  healthMonitor,
	}
}

// SetRecoveryCallbacks sets callbacks for recovery events
func (rm *RecoveryManager) SetRecoveryCallbacks(onStart func(), onEnd func(success bool, err error)) {
	rm.onRecoveryStart = onStart
	rm.onRecoveryEnd = onEnd
}

// Recover attempts to recover from an error
func (rm *RecoveryManager) Recover(ctx context.Context, operationID string, recoverFn func() error) error {
	if rm.onRecoveryStart != nil {
		rm.onRecoveryStart()
	}

	// Check circuit breaker
	if !rm.circuitBreaker.CanExecute() {
		err := errors.New("circuit breaker is open, cannot recover")
		if rm.onRecoveryEnd != nil {
			rm.onRecoveryEnd(false, err)
		}
		return err
	}

	// Wait for healthy connection if health monitor is available
	if rm.healthMonitor != nil {
		err := rm.healthMonitor.WaitForHealthy(ctx)
		if err != nil {
			if rm.onRecoveryEnd != nil {
				rm.onRecoveryEnd(false, err)
			}
			return err
		}
	}

	// Attempt recovery with retries
	err := rm.errorHandler.Execute(ctx, operationID, recoverFn)
	if err != nil {
		rm.circuitBreaker.RecordFailure()
		if rm.onRecoveryEnd != nil {
			rm.onRecoveryEnd(false, err)
		}
		return err
	}

	rm.circuitBreaker.RecordSuccess()
	if rm.onRecoveryEnd != nil {
		rm.onRecoveryEnd(true, nil)
	}
	return nil
}

// isNetworkError checks if an error is a network-related error
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	networkErrors := []string{
		"connection refused",
		"connection reset",
		"no such host",
		"timeout",
		"deadline exceeded",
		"broken pipe",
		"network is unreachable",
		"connection timed out",
	}

	for _, ne := range networkErrors {
		if contains(errStr, ne) {
			return true
		}
	}

	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// GrpcErrorClassifier provides error classification for gRPC errors
type GrpcErrorClassifier struct{}

// Classify classifies a gRPC error
func (c GrpcErrorClassifier) Classify(err error) ErrorClassification {
	if err == nil {
		return ErrorClassification{Category: ErrorCategoryNone, Severity: ErrorSeverityInfo}
	}

	// Check for context cancellation
	if errors.Is(err, context.Canceled) {
		return ErrorClassification{Category: ErrorCategoryCancellation, Severity: ErrorSeverityInfo}
	}

	// Check for timeout
	if errors.Is(err, context.DeadlineExceeded) {
		return ErrorClassification{Category: ErrorCategoryTimeout, Severity: ErrorSeverityWarning}
	}

	// Check gRPC status
	st, ok := status.FromError(err)
	if !ok {
		return ErrorClassification{Category: ErrorCategoryUnknown, Severity: ErrorSeverityError}
	}

	category := ErrorCategoryUnknown
	severity := ErrorSeverityError

	switch st.Code() {
	case codes.OK:
		category = ErrorCategoryNone
		severity = ErrorSeverityInfo
	case codes.Canceled:
		category = ErrorCategoryCancellation
		severity = ErrorSeverityInfo
	case codes.Unknown, codes.Internal, codes.DataLoss:
		category = ErrorCategoryInternal
		severity = ErrorSeverityError
	case codes.InvalidArgument, codes.FailedPrecondition, codes.OutOfRange:
		category = ErrorCategoryValidation
		severity = ErrorSeverityWarning
	case codes.DeadlineExceeded:
		category = ErrorCategoryTimeout
		severity = ErrorSeverityWarning
	case codes.NotFound:
		category = ErrorCategoryNotFound
		severity = ErrorSeverityWarning
	case codes.AlreadyExists, codes.Aborted:
		category = ErrorCategoryConflict
		severity = ErrorSeverityWarning
	case codes.PermissionDenied:
		category = ErrorCategoryAuth
		severity = ErrorSeverityError
	case codes.ResourceExhausted:
		category = ErrorCategoryResource
		severity = ErrorSeverityWarning
	case codes.Unimplemented:
		category = ErrorCategoryNotSupported
		severity = ErrorSeverityError
	case codes.Unavailable:
		category = ErrorCategoryNetwork
		severity = ErrorSeverityError
	case codes.Unauthenticated:
		category = ErrorCategoryAuth
		severity = ErrorSeverityError
	}

	return ErrorClassification{
		Category: category,
		Severity: severity,
		Code:     st.Code(),
		Message:  st.Message(),
	}
}

// ErrorCategory represents the category of an error
type ErrorCategory int

const (
	ErrorCategoryNone ErrorCategory = iota
	ErrorCategoryNetwork
	ErrorCategoryTimeout
	ErrorCategoryAuth
	ErrorCategoryValidation
	ErrorCategoryNotFound
	ErrorCategoryConflict
	ErrorCategoryResource
	ErrorCategoryInternal
	ErrorCategoryCancellation
	ErrorCategoryNotSupported
	ErrorCategoryUnknown
)

// ErrorSeverity represents the severity of an error
type ErrorSeverity int

const (
	ErrorSeverityInfo ErrorSeverity = iota
	ErrorSeverityWarning
	ErrorSeverityError
	ErrorSeverityFatal
)

// ErrorClassification contains detailed error classification
type ErrorClassification struct {
	Category ErrorCategory
	Severity ErrorSeverity
	Code     codes.Code
	Message  string
}