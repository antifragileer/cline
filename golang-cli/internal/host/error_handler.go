// Package host provides gRPC client functionality for communicating with the Cline extension.
// This file contains error handling and reconnection logic for gRPC connections.
package host

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrorHandler provides centralized error handling for gRPC operations
type ErrorHandler struct {
	mu              sync.RWMutex
	retryConfig     RetryConfig
	errorCallbacks  []ErrorCallback
	reconnectConfig ReconnectConfig
	metrics         ErrorMetrics
}

// RetryConfig defines retry behavior
type RetryConfig struct {
	MaxRetries      int
	InitialBackoff  time.Duration
	MaxBackoff      time.Duration
	BackoffMultiplier float64
}

// ReconnectConfig defines reconnection behavior
type ReconnectConfig struct {
	MaxReconnectAttempts int
	ReconnectDelay       time.Duration
	HealthCheckInterval  time.Duration
}

// ErrorCallback is called when an error occurs
type ErrorCallback func(error, ErrorContext)

// ErrorContext provides context about an error
type ErrorContext struct {
	Operation   string
	RetryCount  int
	IsRetryable bool
	ConnState   ConnectionState
}

// ErrorMetrics tracks error statistics
type ErrorMetrics struct {
	mu              sync.RWMutex
	TotalErrors     int64
	RetryableErrors int64
	FatalErrors     int64
	Reconnections   int64
	LastError       time.Time
	ErrorCounts     map[string]int64
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:        5,
		InitialBackoff:    500 * time.Millisecond,
		MaxBackoff:        30 * time.Second,
		BackoffMultiplier: 2.0,
	}
}

// DefaultReconnectConfig returns default reconnection configuration
func DefaultReconnectConfig() ReconnectConfig {
	return ReconnectConfig{
		MaxReconnectAttempts: 5,
		ReconnectDelay:       2 * time.Second,
		HealthCheckInterval:  30 * time.Second,
	}
}

// NewErrorHandler creates a new error handler
func NewErrorHandler() *ErrorHandler {
	return &ErrorHandler{
		retryConfig:     DefaultRetryConfig(),
		reconnectConfig: DefaultReconnectConfig(),
		errorCallbacks:  make([]ErrorCallback, 0),
		metrics: ErrorMetrics{
			ErrorCounts: make(map[string]int64),
		},
	}
}

// SetRetryConfig sets the retry configuration
func (h *ErrorHandler) SetRetryConfig(config RetryConfig) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.retryConfig = config
}

// SetReconnectConfig sets the reconnection configuration
func (h *ErrorHandler) SetReconnectConfig(config ReconnectConfig) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.reconnectConfig = config
}

// RegisterErrorCallback registers a callback for error events
func (h *ErrorHandler) RegisterErrorCallback(callback ErrorCallback) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.errorCallbacks = append(h.errorCallbacks, callback)
}

// HandleError handles an error and returns whether it should be retried
func (h *ErrorHandler) HandleError(err error, operation string, retryCount int, connState ConnectionState) (bool, ErrorContext) {
	h.mu.RLock()
	config := h.retryConfig
	h.mu.RUnlock()

	ctx := ErrorContext{
		Operation:   operation,
		RetryCount:  retryCount,
		IsRetryable: h.IsRetryableError(err),
		ConnState:   connState,
	}

	// Update metrics
	h.updateMetrics(err, ctx.IsRetryable)

	// Notify callbacks
	h.notifyCallbacks(err, ctx)

	return ctx.IsRetryable && retryCount < config.MaxRetries, ctx
}

// IsRetryableError determines if an error is retryable
func (h *ErrorHandler) IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for known retryable errors
	if errors.Is(err, ErrStreamNotReady) || errors.Is(err, ErrBackpressureExceeded) {
		return true
	}

	// Check for non-retryable errors
	if errors.Is(err, ErrStreamClosed) {
		return false
	}

	// Check for gRPC status codes
	if st, ok := status.FromError(err); ok {
		switch st.Code() {
		case codes.Unavailable,
			codes.ResourceExhausted,
			codes.Aborted,
			codes.DeadlineExceeded:
			return true
		case codes.InvalidArgument,
			codes.Unauthenticated,
			codes.PermissionDenied,
			codes.NotFound,
			codes.AlreadyExists,
			codes.FailedPrecondition,
			codes.OutOfRange,
			codes.Unimplemented,
			codes.Internal,
			codes.DataLoss:
			return false
		default:
			// Unknown codes may be retryable
			return true
		}
	}

	// Check for specific error strings
	errStr := err.Error()
	retryableErrors := []string{
		"connection refused",
		"connection reset",
		"broken pipe",
		"no such host",
		"timeout",
		"deadline exceeded",
		"temporarily unavailable",
		"try again",
		"stream closed",
		"transport is closing",
	}

	for _, retryable := range retryableErrors {
		if containsString(errStr, retryable) {
			return true
		}
	}

	return false
}

// CalculateBackoff calculates the backoff duration for a retry attempt
func (h *ErrorHandler) CalculateBackoff(attempt int) time.Duration {
	h.mu.RLock()
	config := h.retryConfig
	h.mu.RUnlock()

	backoff := config.InitialBackoff
	for i := 0; i < attempt; i++ {
		backoff = time.Duration(float64(backoff) * config.BackoffMultiplier)
		if backoff > config.MaxBackoff {
			backoff = config.MaxBackoff
			break
		}
	}

	return backoff
}

// ShouldReconnect determines if a reconnection should be attempted
func (h *ErrorHandler) ShouldReconnect(reconnectCount int, err error) bool {
	h.mu.RLock()
	config := h.reconnectConfig
	h.mu.RUnlock()

	if reconnectCount >= config.MaxReconnectAttempts {
		return false
	}

	// Only reconnect for certain errors
	return h.IsRetryableError(err)
}

// GetReconnectDelay returns the delay before reconnection
func (h *ErrorHandler) GetReconnectDelay() time.Duration {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.reconnectConfig.ReconnectDelay
}

// GetMetrics returns current error metrics
func (h *ErrorHandler) GetMetrics() ErrorMetrics {
	h.metrics.mu.RLock()
	defer h.metrics.mu.RUnlock()
	return ErrorMetrics{
		TotalErrors:     h.metrics.TotalErrors,
		RetryableErrors: h.metrics.RetryableErrors,
		FatalErrors:     h.metrics.FatalErrors,
		Reconnections:   h.metrics.Reconnections,
		LastError:       h.metrics.LastError,
		ErrorCounts:     copyErrorCounts(h.metrics.ErrorCounts),
	}
}

// ResetMetrics resets error metrics
func (h *ErrorHandler) ResetMetrics() {
	h.metrics.mu.Lock()
	defer h.metrics.mu.Unlock()
	
	// Reset individual fields without replacing the struct (which would replace the mutex)
	h.metrics.TotalErrors = 0
	h.metrics.RetryableErrors = 0
	h.metrics.FatalErrors = 0
	h.metrics.Reconnections = 0
	h.metrics.LastError = time.Time{}
	h.metrics.ErrorCounts = make(map[string]int64)
}

// updateMetrics updates error metrics
func (h *ErrorHandler) updateMetrics(err error, isRetryable bool) {
	h.metrics.mu.Lock()
	defer h.metrics.mu.Unlock()

	h.metrics.TotalErrors++
	h.metrics.LastError = time.Now()

	if isRetryable {
		h.metrics.RetryableErrors++
	} else {
		h.metrics.FatalErrors++
	}

	// Track error type counts
	errorType := classifyError(err)
	h.metrics.ErrorCounts[errorType]++
}

// notifyCallbacks notifies all registered callbacks
func (h *ErrorHandler) notifyCallbacks(err error, ctx ErrorContext) {
	h.mu.RLock()
	callbacks := make([]ErrorCallback, len(h.errorCallbacks))
	copy(callbacks, h.errorCallbacks)
	h.mu.RUnlock()

	for _, callback := range callbacks {
		callback(err, ctx)
	}
}

// classifyError classifies an error type
func classifyError(err error) string {
	if err == nil {
		return "none"
	}

	if st, ok := status.FromError(err); ok {
		return fmt.Sprintf("grpc_%s", st.Code().String())
	}

	errStr := err.Error()
	if containsString(errStr, "connection") {
		return "connection"
	}
	if containsString(errStr, "timeout") || containsString(errStr, "deadline") {
		return "timeout"
	}
	if containsString(errStr, "stream") {
		return "stream"
	}
	if containsString(errStr, "context") {
		return "context"
	}

	return "other"
}

// containsString checks if a string contains a substring (case-insensitive)
func containsString(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	if s == substr {
		return true
	}

	// Simple case-insensitive check
	lowerS := toLower(s)
	lowerSubstr := toLower(substr)

	for i := 0; i <= len(lowerS)-len(lowerSubstr); i++ {
		if lowerS[i:i+len(lowerSubstr)] == lowerSubstr {
			return true
		}
	}
	return false
}

// toLower converts a string to lowercase (ASCII only)
func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c = c + ('a' - 'A')
		}
		result[i] = c
	}
	return string(result)
}

func copyErrorCounts(src map[string]int64) map[string]int64 {
	dst := make(map[string]int64, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// GRPCError represents a structured gRPC error
type GRPCError struct {
	Code    codes.Code
	Message string
	Details string
	Retryable bool
}

// NewGRPCError creates a new GRPCError from an error
func NewGRPCError(err error) *GRPCError {
	if err == nil {
		return nil
	}

	if st, ok := status.FromError(err); ok {
		return &GRPCError{
			Code:      st.Code(),
			Message:   st.Message(),
			Details:   fmt.Sprintf("%v", st.Details()),
			Retryable: isRetryableCode(st.Code()),
		}
	}

	return &GRPCError{
		Code:      codes.Unknown,
		Message:   err.Error(),
		Details:   "",
		Retryable: true,
	}
}

// Error implements the error interface
func (e *GRPCError) Error() string {
	return fmt.Sprintf("gRPC error [%s]: %s", e.Code.String(), e.Message)
}

// isRetryableCode determines if a gRPC code is retryable
func isRetryableCode(code codes.Code) bool {
	switch code {
	case codes.Unavailable,
		codes.ResourceExhausted,
		codes.Aborted,
		codes.DeadlineExceeded:
		return true
	default:
		return false
	}
}

// ConnectionHealthMonitor monitors connection health
type ConnectionHealthMonitor struct {
	mu             sync.RWMutex
	conn           *grpc.ClientConn
	interval       time.Duration
	healthCallback func(bool)
	stopChan       chan struct{}
	running        bool
}

// NewConnectionHealthMonitor creates a new health monitor
func NewConnectionHealthMonitor(conn *grpc.ClientConn, interval time.Duration) *ConnectionHealthMonitor {
	return &ConnectionHealthMonitor{
		conn:     conn,
		interval: interval,
		stopChan: make(chan struct{}),
	}
}

// SetHealthCallback sets the callback for health status changes
func (m *ConnectionHealthMonitor) SetHealthCallback(callback func(bool)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.healthCallback = callback
}

// Start starts the health monitoring
func (m *ConnectionHealthMonitor) Start() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return
	}

	m.running = true
	go m.monitor()
}

// Stop stops the health monitoring
func (m *ConnectionHealthMonitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}

	m.running = false
	close(m.stopChan)
}

// monitor runs the health check loop
func (m *ConnectionHealthMonitor) monitor() {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopChan:
			return
		case <-ticker.C:
			healthy := m.checkHealth()
			m.mu.RLock()
			callback := m.healthCallback
			m.mu.RUnlock()

			if callback != nil {
				callback(healthy)
			}
		}
	}
}

// checkHealth performs a health check
func (m *ConnectionHealthMonitor) checkHealth() bool {
	m.mu.RLock()
	conn := m.conn
	m.mu.RUnlock()

	if conn == nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try to get connection state
	state := conn.GetState()
	if state.String() == "READY" {
		return true
	}

	// Wait for state change with timeout
	return conn.WaitForStateChange(ctx, state)
}

// IsRunning returns whether the monitor is running
func (m *ConnectionHealthMonitor) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}