// Package grpc provides error handling functionality.
package grpc

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestDefaultRetryPolicy(t *testing.T) {
	policy := DefaultRetryPolicy()
	if policy == nil {
		t.Fatal("Expected non-nil policy")
	}
	if policy.MaxRetries != 5 {
		t.Errorf("Expected MaxRetries to be 5, got %d", policy.MaxRetries)
	}
	if policy.InitialBackoff != 100*time.Millisecond {
		t.Errorf("Expected InitialBackoff to be 100ms, got %v", policy.InitialBackoff)
	}
	if policy.MaxBackoff != 30*time.Second {
		t.Errorf("Expected MaxBackoff to be 30s, got %v", policy.MaxBackoff)
	}
	if policy.BackoffMultiplier != 2.0 {
		t.Errorf("Expected BackoffMultiplier to be 2.0, got %f", policy.BackoffMultiplier)
	}
	if !policy.Jitter {
		t.Error("Expected Jitter to be true")
	}
	if policy.JitterFactor != 0.2 {
		t.Errorf("Expected JitterFactor to be 0.2, got %f", policy.JitterFactor)
	}
}

func TestRetryPolicy_IsRetryable(t *testing.T) {
	policy := DefaultRetryPolicy()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"EOF", io.EOF, true},
		{"context canceled", context.Canceled, false},
		{"context deadline exceeded", context.DeadlineExceeded, false},
		{"unavailable", status.Error(codes.Unavailable, "service unavailable"), true},
		{"resource exhausted", status.Error(codes.ResourceExhausted, "rate limited"), true},
		{"aborted", status.Error(codes.Aborted, "operation aborted"), true},
		{"deadline exceeded", status.Error(codes.DeadlineExceeded, "timeout"), true},
		{"invalid argument", status.Error(codes.InvalidArgument, "bad request"), false},
		{"not found", status.Error(codes.NotFound, "not found"), false},
		{"already exists", status.Error(codes.AlreadyExists, "exists"), false},
		{"permission denied", status.Error(codes.PermissionDenied, "denied"), false},
		{"unauthenticated", status.Error(codes.Unauthenticated, "unauthenticated"), false},
		{"network error", errors.New("connection refused"), true},
		{"timeout error", errors.New("connection timed out"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := policy.IsRetryable(tt.err)
			if got != tt.expected {
				t.Errorf("IsRetryable(%v) = %v, want %v", tt.err, got, tt.expected)
			}
		})
	}
}

func TestRetryPolicy_IsRetryable_NonRetryable(t *testing.T) {
	policy := &RetryPolicy{
		MaxRetries:        5,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        30 * time.Second,
		BackoffMultiplier: 2.0,
		RetryableCodes:    []codes.Code{codes.Unavailable},
		NonRetryableErrors: []error{
			errors.New("permanent error"),
		},
		Jitter:       true,
		JitterFactor: 0.2,
	}

	// Should not retry permanent errors
	err := errors.New("permanent error")
	if policy.IsRetryable(err) {
		t.Error("Expected permanent error to not be retryable")
	}
}

func TestRetryPolicy_CalculateBackoff(t *testing.T) {
	policy := DefaultRetryPolicy()

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 0},
		{1, 100 * time.Millisecond},
		{2, 200 * time.Millisecond},
		{3, 400 * time.Millisecond},
		{5, 1600 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run("attempt_"+string(rune(tt.attempt)), func(t *testing.T) {
			got := policy.CalculateBackoff(tt.attempt)
			// Allow some tolerance due to jitter
			if tt.attempt == 0 {
				if got != 0 {
					t.Errorf("Expected 0 for attempt 0, got %v", got)
				}
				return
			}
			// Without jitter, backoff should match expected
			// With jitter, it could be ±20%
			minExpected := time.Duration(float64(tt.expected) * 0.8)
			maxExpected := time.Duration(float64(tt.expected) * 1.2)
			if got < minExpected || got > maxExpected {
				t.Errorf("Expected backoff between %v and %v, got %v", minExpected, maxExpected, got)
			}
		})
	}
}

func TestRetryPolicy_CalculateBackoff_MaxBackoff(t *testing.T) {
	policy := &RetryPolicy{
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        1 * time.Second,
		BackoffMultiplier: 10.0,
		Jitter:            false,
	}

	// 100ms * 10^10 = way more than 1 second, should be capped
	backoff := policy.CalculateBackoff(10)
	if backoff > policy.MaxBackoff {
		t.Errorf("Expected backoff to be capped at %v, got %v", policy.MaxBackoff, backoff)
	}
}

func TestRetryPolicy_CalculateBackoff_NoJitter(t *testing.T) {
	policy := &RetryPolicy{
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        30 * time.Second,
		BackoffMultiplier: 2.0,
		Jitter:            false,
		JitterFactor:      0.0,
	}

	backoff1 := policy.CalculateBackoff(1)
	backoff2 := policy.CalculateBackoff(1)

	if backoff1 != backoff2 {
		t.Errorf("Without jitter, backoff should be deterministic: %v vs %v", backoff1, backoff2)
	}

	expected := 100 * time.Millisecond
	if backoff1 != expected {
		t.Errorf("Expected %v, got %v", expected, backoff1)
	}
}

func TestNewErrorHandler(t *testing.T) {
	policy := DefaultRetryPolicy()
	handler := NewErrorHandler(policy)
	if handler == nil {
		t.Fatal("Expected non-nil handler")
	}
	if handler.policy != policy {
		t.Error("Handler policy should match provided policy")
	}
}

func TestNewErrorHandler_DefaultPolicy(t *testing.T) {
	handler := NewErrorHandler(nil)
	if handler == nil {
		t.Fatal("Expected non-nil handler")
	}
	if handler.policy == nil {
		t.Error("Handler should have default policy")
	}
}

func TestErrorHandler_Execute_Success(t *testing.T) {
	handler := NewErrorHandler(DefaultRetryPolicy())
	handler.SetSuccessCallback(func(attempts int) {
		// Success callback called
	})

	calls := 0
	err := handler.Execute(context.Background(), "test-op", func() error {
		calls++
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if calls != 1 {
		t.Errorf("Expected function to be called once, got %d", calls)
	}
}

func TestErrorHandler_Execute_RetryThenSuccess(t *testing.T) {
	policy := &RetryPolicy{
		MaxRetries:        3,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        100 * time.Millisecond,
		BackoffMultiplier: 2.0,
		RetryableCodes:    []codes.Code{codes.Unavailable},
		Jitter:            false,
	}

	handler := NewErrorHandler(policy)
	retryCount := 0
	handler.SetRetryCallback(func(attempt int, err error, backoff time.Duration) {
		retryCount++
	})

	calls := 0
	err := handler.Execute(context.Background(), "test-op", func() error {
		calls++
		if calls < 3 {
			return status.Error(codes.Unavailable, "temporarily unavailable")
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected success after retries, got %v", err)
	}
	if calls != 3 {
		t.Errorf("Expected 3 calls, got %d", calls)
	}
	if retryCount != 2 {
		t.Errorf("Expected 2 retries, got %d", retryCount)
	}
}

func TestErrorHandler_Execute_MaxRetriesExceeded(t *testing.T) {
	policy := &RetryPolicy{
		MaxRetries:        2,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        100 * time.Millisecond,
		BackoffMultiplier: 2.0,
		RetryableCodes:    []codes.Code{codes.Unavailable},
		Jitter:            false,
	}

	handler := NewErrorHandler(policy)
	maxRetriesCalled := false
	handler.SetMaxRetriesCallback(func(err error) {
		maxRetriesCalled = true
	})

	expectedErr := status.Error(codes.Unavailable, "always unavailable")
	err := handler.Execute(context.Background(), "test-op", func() error {
		return expectedErr
	})

	if err == nil {
		t.Error("Expected error after max retries")
	}
	if !maxRetriesCalled {
		t.Error("Max retries callback should have been called")
	}
	if !errors.Is(err, expectedErr) && err.Error() != "max retries (2) exceeded: rpc error: code = Unavailable desc = always unavailable" {
		t.Errorf("Expected wrapped error, got %v", err)
	}
}

func TestErrorHandler_Execute_ContextCancellation(t *testing.T) {
	policy := &RetryPolicy{
		MaxRetries:        5,
		InitialBackoff:    1 * time.Second, // Long backoff
		MaxBackoff:        5 * time.Second,
		BackoffMultiplier: 2.0,
		RetryableCodes:    []codes.Code{codes.Unavailable},
		Jitter:            false,
	}

	handler := NewErrorHandler(policy)
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context after short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := handler.Execute(ctx, "test-op", func() error {
		return status.Error(codes.Unavailable, "unavailable")
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context canceled error, got %v", err)
	}
}

func TestErrorHandler_ExecuteWithResult(t *testing.T) {
	handler := NewErrorHandler(DefaultRetryPolicy())

	result, err := handler.ExecuteWithResult(context.Background(), "test-op", func() (interface{}, error) {
		return "success", nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != "success" {
		t.Errorf("Expected 'success', got %v", result)
	}
}

func TestErrorHandler_GetAttempts(t *testing.T) {
	handler := NewErrorHandler(DefaultRetryPolicy())

	// No attempts yet
	if handler.GetAttempts("unknown-op") != 0 {
		t.Error("Expected 0 attempts for unknown operation")
	}

	// Execute an operation
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	handler.Execute(ctx, "test-op", func() error {
		return nil
	})

	// After successful execution, attempts should be cleared
	if handler.GetAttempts("test-op") != 0 {
		t.Error("Expected 0 attempts after success")
	}
}

func TestErrorHandler_ResetAttempts(t *testing.T) {
	handler := NewErrorHandler(DefaultRetryPolicy())

	// Execute a failing operation
	ctx := context.Background()
	handler.Execute(ctx, "test-op", func() error {
		return errors.New("error")
	})

	handler.ResetAttempts("test-op")

	if handler.GetAttempts("test-op") != 0 {
		t.Error("Expected 0 attempts after reset")
	}
}

func TestNewStreamErrorHandler(t *testing.T) {
	errorHandler := NewErrorHandler(DefaultRetryPolicy())
	reconnectFn := func() error { return nil }

	streamHandler := NewStreamErrorHandler(errorHandler, reconnectFn)
	if streamHandler == nil {
		t.Fatal("Expected non-nil stream handler")
	}
}

func TestStreamErrorHandler_HandleRecvError_Nil(t *testing.T) {
	errorHandler := NewErrorHandler(DefaultRetryPolicy())
	streamHandler := NewStreamErrorHandler(errorHandler, func() error { return nil })

	err := streamHandler.HandleRecvError(context.Background(), nil)
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
}

func TestStreamErrorHandler_HandleRecvError_EOF(t *testing.T) {
	errorHandler := NewErrorHandler(DefaultRetryPolicy())
	streamHandler := NewStreamErrorHandler(errorHandler, func() error { return nil })

	err := streamHandler.HandleRecvError(context.Background(), io.EOF)
	if err != io.EOF {
		t.Errorf("Expected io.EOF, got %v", err)
	}
}

func TestStreamErrorHandler_HandleRecvError_NonRetryable(t *testing.T) {
	errorHandler := NewErrorHandler(DefaultRetryPolicy())
	streamHandler := NewStreamErrorHandler(errorHandler, func() error { return nil })

	expectedErr := errors.New("permanent error")
	err := streamHandler.HandleRecvError(context.Background(), expectedErr)
	if err != expectedErr {
		t.Errorf("Expected %v, got %v", expectedErr, err)
	}
}

func TestStreamErrorHandler_HandleRecvError_Recovers(t *testing.T) {
	policy := &RetryPolicy{
		MaxRetries:        3,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        100 * time.Millisecond,
		BackoffMultiplier: 2.0,
		RetryableCodes:    []codes.Code{codes.Unavailable},
		Jitter:            false,
	}

	errorHandler := NewErrorHandler(policy)
	reconnectCalled := false
	reconnectFn := func() error {
		reconnectCalled = true
		return nil
	}

	streamHandler := NewStreamErrorHandler(errorHandler, reconnectFn)
	streamHandler.SetReconnectCallback(func() {
		// Reconnected callback
	})

	err := streamHandler.HandleRecvError(context.Background(), status.Error(codes.Unavailable, "stream closed"))

	if err != nil {
		t.Errorf("Expected successful reconnect, got %v", err)
	}
	if !reconnectCalled {
		t.Error("Reconnect function should have been called")
	}
}

func TestStreamErrorHandler_HandleRecvError_RecoverFails(t *testing.T) {
	policy := &RetryPolicy{
		MaxRetries:        1,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        100 * time.Millisecond,
		BackoffMultiplier: 2.0,
		RetryableCodes:    []codes.Code{codes.Unavailable},
		Jitter:            false,
	}

	errorHandler := NewErrorHandler(policy)
	reconnectFn := func() error {
		return errors.New("reconnect failed")
	}

	streamHandler := NewStreamErrorHandler(errorHandler, reconnectFn)

	originalErr := status.Error(codes.Unavailable, "stream closed")
	err := streamHandler.HandleRecvError(context.Background(), originalErr)

	if err == nil {
		t.Error("Expected error when reconnect fails")
	}
}

func TestNewCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker(5, 30*time.Second)
	if cb == nil {
		t.Fatal("Expected non-nil circuit breaker")
	}
	if cb.Threshold != 5 {
		t.Errorf("Expected threshold 5, got %d", cb.Threshold)
	}
	if cb.ResetTimeout != 30*time.Second {
		t.Errorf("Expected reset timeout 30s, got %v", cb.ResetTimeout)
	}
	if cb.GetState() != CircuitClosed {
		t.Errorf("Expected initial state Closed, got %v", cb.GetState())
	}
}

func TestCircuitState_String(t *testing.T) {
	tests := []struct {
		state    CircuitState
		expected string
	}{
		{CircuitClosed, "closed"},
		{CircuitOpen, "open"},
		{CircuitHalfOpen, "half-open"},
		{CircuitState(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.state.String(); got != tt.expected {
				t.Errorf("String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCircuitBreaker_CanExecute(t *testing.T) {
	cb := NewCircuitBreaker(3, 50*time.Millisecond)

	// Initially closed, can execute
	if !cb.CanExecute() {
		t.Error("Expected to be able to execute when closed")
	}

	// Record failures to open circuit
	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordFailure()

	// Should be open now
	if cb.CanExecute() {
		t.Error("Expected not to be able to execute when open")
	}

	// Wait for reset timeout
	time.Sleep(60 * time.Millisecond)

	// Should be half-open now
	if !cb.CanExecute() {
		t.Error("Expected to be able to execute when half-open")
	}

	// Recording success should close it
	cb.RecordSuccess()
	if !cb.CanExecute() {
		t.Error("Expected to be able to execute after success")
	}
}

func TestCircuitBreaker_RecordSuccess(t *testing.T) {
	cb := NewCircuitBreaker(3, 30*time.Second)

	cb.RecordSuccess()
	// Should remain closed
	if cb.GetState() != CircuitClosed {
		t.Errorf("Expected state Closed, got %v", cb.GetState())
	}
}

func TestCircuitBreaker_RecordFailure_OpensCircuit(t *testing.T) {
	cb := NewCircuitBreaker(2, 30*time.Second)

	cb.RecordFailure()
	if cb.GetState() != CircuitClosed {
		t.Errorf("Expected state Closed after 1 failure, got %v", cb.GetState())
	}

	cb.RecordFailure()
	if cb.GetState() != CircuitOpen {
		t.Errorf("Expected state Open after 2 failures, got %v", cb.GetState())
	}
}

func TestCircuitBreaker_RecordFailure_HalfOpenToOpen(t *testing.T) {
	cb := NewCircuitBreaker(2, 50*time.Millisecond)

	// Open the circuit
	cb.RecordFailure()
	cb.RecordFailure()

	// Wait for reset
	time.Sleep(60 * time.Millisecond)

	// Should be half-open
	cb.CanExecute()

	// Failure in half-open should reopen
	cb.RecordFailure()
	if cb.GetState() != CircuitOpen {
		t.Errorf("Expected state Open after failure in half-open, got %v", cb.GetState())
	}
}

func TestCircuitBreaker_SetStateChangeCallback(t *testing.T) {
	cb := NewCircuitBreaker(1, 30*time.Second)

	stateChanges := []struct{ old, new CircuitState }{}
	cb.SetStateChangeCallback(func(old, new CircuitState) {
		stateChanges = append(stateChanges, struct{ old, new CircuitState }{old, new})
	})

	cb.RecordFailure()
	cb.RecordFailure()

	if len(stateChanges) != 1 {
		t.Errorf("Expected 1 state change, got %d", len(stateChanges))
	}
}

func TestCircuitBreaker_Execute(t *testing.T) {
	cb := NewCircuitBreaker(3, 30*time.Second)

	// Successful execution
	err := cb.Execute(func() error {
		return nil
	})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Failed execution
	expectedErr := errors.New("operation failed")
	err = cb.Execute(func() error {
		return expectedErr
	})
	if err != expectedErr {
		t.Errorf("Expected %v, got %v", expectedErr, err)
	}
}

func TestCircuitBreaker_Execute_OpenCircuit(t *testing.T) {
	cb := NewCircuitBreaker(1, 30*time.Second)

	// Fail once to open circuit
	cb.Execute(func() error {
		return errors.New("fail")
	})

	// Next execution should fail immediately
	err := cb.Execute(func() error {
		return nil
	})
	if err == nil || err.Error() != "circuit breaker is open" {
		t.Errorf("Expected 'circuit breaker is open' error, got %v", err)
	}
}

func TestNewRecoveryManager(t *testing.T) {
	errorHandler := NewErrorHandler(DefaultRetryPolicy())
	circuitBreaker := NewCircuitBreaker(5, 30*time.Second)
	healthMonitor := &HealthMonitor{}

	rm := NewRecoveryManager(errorHandler, circuitBreaker, healthMonitor)
	if rm == nil {
		t.Fatal("Expected non-nil recovery manager")
	}
}

func TestRecoveryManager_SetRecoveryCallbacks(t *testing.T) {
	rm := NewRecoveryManager(NewErrorHandler(nil), NewCircuitBreaker(5, 30*time.Second), nil)

	callbackSet := false

	rm.SetRecoveryCallbacks(
		func() { callbackSet = true },
		func(success bool, err error) { callbackSet = callbackSet || success },
	)

	if rm.onRecoveryStart == nil || rm.onRecoveryEnd == nil {
		t.Error("Callbacks should be set")
	}
	
	// Silence unused warning - callbacks were set
	_ = callbackSet
}

func TestRecoveryManager_Recover_CircuitOpen(t *testing.T) {
	cb := NewCircuitBreaker(1, 30*time.Second)
	cb.RecordFailure()
	cb.RecordFailure()

	rm := NewRecoveryManager(NewErrorHandler(nil), cb, nil)

	err := rm.Recover(context.Background(), "test", func() error {
		return nil
	})

	if err == nil || err.Error() != "circuit breaker is open, cannot recover" {
		t.Errorf("Expected circuit breaker error, got %v", err)
	}
}

func TestRecoveryManager_Recover_Success(t *testing.T) {
	policy := &RetryPolicy{
		MaxRetries:        3,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        100 * time.Millisecond,
		BackoffMultiplier: 2.0,
		RetryableCodes:    []codes.Code{codes.Unavailable},
		Jitter:            false,
	}

	cb := NewCircuitBreaker(5, 30*time.Second)
	rm := NewRecoveryManager(NewErrorHandler(policy), cb, nil)

	callbackTriggered := false

	rm.SetRecoveryCallbacks(
		func() { callbackTriggered = true },
		func(s bool, err error) {
			callbackTriggered = callbackTriggered || s
		},
	)

	err := rm.Recover(context.Background(), "test", func() error {
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	_ = callbackTriggered // Silence unused warning
}

func TestRecoveryManager_Recover_Failure(t *testing.T) {
	policy := &RetryPolicy{
		MaxRetries:        1,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        100 * time.Millisecond,
		BackoffMultiplier: 2.0,
		RetryableCodes:    []codes.Code{codes.Unavailable},
		Jitter:            false,
	}

	cb := NewCircuitBreaker(5, 30*time.Second)
	rm := NewRecoveryManager(NewErrorHandler(policy), cb, nil)

	err := rm.Recover(context.Background(), "test", func() error {
		return errors.New("always fails")
	})

	if err == nil {
		t.Error("Expected error from recovery")
	}
}

func TestGrpcErrorClassifier_Classify(t *testing.T) {
	classifier := GrpcErrorClassifier{}

	tests := []struct {
		name             string
		err              error
		expectedCategory ErrorCategory
		expectedSeverity ErrorSeverity
	}{
		{"nil", nil, ErrorCategoryNone, ErrorSeverityInfo},
		{"context canceled", context.Canceled, ErrorCategoryCancellation, ErrorSeverityInfo},
		{"context deadline exceeded", context.DeadlineExceeded, ErrorCategoryTimeout, ErrorSeverityWarning},
		{"unknown", status.Error(codes.Unknown, "unknown"), ErrorCategoryInternal, ErrorSeverityError},
		{"internal", status.Error(codes.Internal, "internal"), ErrorCategoryInternal, ErrorSeverityError},
		{"data loss", status.Error(codes.DataLoss, "data loss"), ErrorCategoryInternal, ErrorSeverityError},
		{"invalid argument", status.Error(codes.InvalidArgument, "invalid"), ErrorCategoryValidation, ErrorSeverityWarning},
		{"failed precondition", status.Error(codes.FailedPrecondition, "precondition"), ErrorCategoryValidation, ErrorSeverityWarning},
		{"out of range", status.Error(codes.OutOfRange, "range"), ErrorCategoryValidation, ErrorSeverityWarning},
		{"deadline exceeded", status.Error(codes.DeadlineExceeded, "deadline"), ErrorCategoryTimeout, ErrorSeverityWarning},
		{"not found", status.Error(codes.NotFound, "not found"), ErrorCategoryNotFound, ErrorSeverityWarning},
		{"already exists", status.Error(codes.AlreadyExists, "exists"), ErrorCategoryConflict, ErrorSeverityWarning},
		{"aborted", status.Error(codes.Aborted, "aborted"), ErrorCategoryConflict, ErrorSeverityWarning},
		{"permission denied", status.Error(codes.PermissionDenied, "denied"), ErrorCategoryAuth, ErrorSeverityError},
		{"resource exhausted", status.Error(codes.ResourceExhausted, "exhausted"), ErrorCategoryResource, ErrorSeverityWarning},
		{"unimplemented", status.Error(codes.Unimplemented, "unimplemented"), ErrorCategoryNotSupported, ErrorSeverityError},
		{"unavailable", status.Error(codes.Unavailable, "unavailable"), ErrorCategoryNetwork, ErrorSeverityError},
		{"unauthenticated", status.Error(codes.Unauthenticated, "unauthenticated"), ErrorCategoryAuth, ErrorSeverityError},
		{"canceled", status.Error(codes.Canceled, "canceled"), ErrorCategoryCancellation, ErrorSeverityInfo},
		{"OK", status.Error(codes.OK, "ok"), ErrorCategoryNone, ErrorSeverityInfo},
		{"non-grpc error", errors.New("random error"), ErrorCategoryUnknown, ErrorSeverityError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.Classify(tt.err)
			if result.Category != tt.expectedCategory {
				t.Errorf("Expected category %v, got %v", tt.expectedCategory, result.Category)
			}
			if result.Severity != tt.expectedSeverity {
				t.Errorf("Expected severity %v, got %v", tt.expectedSeverity, result.Severity)
			}
		})
	}
}

func TestIsNetworkError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil", nil, false},
		{"connection refused", errors.New("connection refused"), true},
		{"connection reset", errors.New("connection reset by peer"), true},
		{"no such host", errors.New("no such host"), true},
		{"timeout", errors.New("i/o timeout"), true},
		{"deadline exceeded", errors.New("context deadline exceeded"), true},
		{"broken pipe", errors.New("broken pipe"), true},
		{"network unreachable", errors.New("network is unreachable"), true},
		{"connection timed out", errors.New("connection timed out"), true},
		{"other error", errors.New("some other error"), false},
		{"empty", errors.New(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isNetworkError(tt.err)
			if got != tt.expected {
				t.Errorf("isNetworkError(%v) = %v, want %v", tt.err, got, tt.expected)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		s        string
		substr   string
		expected bool
	}{
		{"hello world", "world", true},
		{"hello world", "hello", true},
		{"hello world", "foo", false},
		{"", "", true},
		{"a", "a", true},
		{"a", "ab", false},
	}

	for _, tt := range tests {
		t.Run(tt.s+"_"+tt.substr, func(t *testing.T) {
			got := contains(tt.s, tt.substr)
			if got != tt.expected {
				t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.expected)
			}
		})
	}
}

func BenchmarkErrorHandler_Execute(b *testing.B) {
	handler := NewErrorHandler(&RetryPolicy{
		MaxRetries:        1,
		InitialBackoff:    1 * time.Millisecond,
		MaxBackoff:        10 * time.Millisecond,
		BackoffMultiplier: 2.0,
		Jitter:            false,
	})

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.Execute(ctx, "benchmark", func() error {
			return nil
		})
	}
}

func BenchmarkRetryPolicy_CalculateBackoff(b *testing.B) {
	policy := DefaultRetryPolicy()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		policy.CalculateBackoff(5)
	}
}

func BenchmarkCircuitBreaker_CanExecute(b *testing.B) {
	cb := NewCircuitBreaker(100, 30*time.Second)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cb.CanExecute()
	}
}

func BenchmarkGrpcErrorClassifier_Classify(b *testing.B) {
	classifier := GrpcErrorClassifier{}
	err := status.Error(codes.Unavailable, "service unavailable")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		classifier.Classify(err)
	}
}