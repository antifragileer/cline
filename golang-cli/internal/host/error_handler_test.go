// Package host provides gRPC client functionality for communicating with the Cline extension.
// This file contains tests for the error handler.
package host

import (
	"errors"
	"strconv"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"github.com/stretchr/testify/assert"
)

func TestNewErrorHandler(t *testing.T) {
	handler := NewErrorHandler()
	assert.NotNil(t, handler)
}

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()
	assert.Equal(t, 5, config.MaxRetries)
	assert.Equal(t, 500*time.Millisecond, config.InitialBackoff)
	assert.Equal(t, 30*time.Second, config.MaxBackoff)
	assert.Equal(t, 2.0, config.BackoffMultiplier)
}

func TestDefaultReconnectConfig(t *testing.T) {
	config := DefaultReconnectConfig()
	assert.Equal(t, 5, config.MaxReconnectAttempts)
	assert.Equal(t, 2*time.Second, config.ReconnectDelay)
	assert.Equal(t, 30*time.Second, config.HealthCheckInterval)
}

func TestErrorHandler_IsRetryableError(t *testing.T) {
	handler := NewErrorHandler()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error is not retryable",
			err:      nil,
			expected: false,
		},
		{
			name:     "gRPC Unavailable is retryable",
			err:      status.Error(codes.Unavailable, "service unavailable"),
			expected: true,
		},
		{
			name:     "gRPC ResourceExhausted is retryable",
			err:      status.Error(codes.ResourceExhausted, "rate limited"),
			expected: true,
		},
		{
			name:     "gRPC DeadlineExceeded is retryable",
			err:      status.Error(codes.DeadlineExceeded, "timeout"),
			expected: true,
		},
		{
			name:     "gRPC InvalidArgument is not retryable",
			err:      status.Error(codes.InvalidArgument, "bad request"),
			expected: false,
		},
		{
			name:     "gRPC NotFound is not retryable",
			err:      status.Error(codes.NotFound, "not found"),
			expected: false,
		},
		{
			name:     "connection refused is retryable",
			err:      errors.New("connection refused"),
			expected: true,
		},
		{
			name:     "connection reset is retryable",
			err:      errors.New("connection reset by peer"),
			expected: true,
		},
		{
			name:     "broken pipe is retryable",
			err:      errors.New("broken pipe"),
			expected: true,
		},
		{
			name:     "timeout is retryable",
			err:      errors.New("operation timeout"),
			expected: true,
		},
		{
			name:     "deadline exceeded is retryable",
			err:      errors.New("context deadline exceeded"),
			expected: true,
		},
		{
			name:     "stream closed is retryable",
			err:      errors.New("stream closed unexpectedly"),
			expected: true,
		},
		{
			name:     "transport closing is retryable",
			err:      errors.New("transport is closing"),
			expected: true,
		},
		{
			name:     "generic error is not retryable",
			err:      errors.New("some random error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.IsRetryableError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestErrorHandler_CalculateBackoff(t *testing.T) {
	handler := NewErrorHandler()

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 500 * time.Millisecond},
		{1, 1 * time.Second},
		{2, 2 * time.Second},
		{3, 4 * time.Second},
		{4, 8 * time.Second},
		{5, 16 * time.Second},
		{6, 30 * time.Second}, // capped at max
		{10, 30 * time.Second}, // capped at max
	}

	for _, tt := range tests {
		t.Run(strconv.Itoa(tt.attempt), func(t *testing.T) {
			result := handler.CalculateBackoff(tt.attempt)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestErrorHandler_HandleError(t *testing.T) {
	handler := NewErrorHandler()

	t.Run("retryable error under max retries", func(t *testing.T) {
		err := status.Error(codes.Unavailable, "service unavailable")
		shouldRetry, ctx := handler.HandleError(err, "test_operation", 2, StateConnected)

		assert.True(t, shouldRetry)
		assert.Equal(t, "test_operation", ctx.Operation)
		assert.Equal(t, 2, ctx.RetryCount)
		assert.True(t, ctx.IsRetryable)
		assert.Equal(t, StateConnected, ctx.ConnState)
	})

	t.Run("non-retryable error", func(t *testing.T) {
		err := status.Error(codes.InvalidArgument, "bad request")
		shouldRetry, ctx := handler.HandleError(err, "test_operation", 0, StateConnected)

		assert.False(t, shouldRetry)
		assert.False(t, ctx.IsRetryable)
	})

	t.Run("max retries exceeded", func(t *testing.T) {
		err := status.Error(codes.Unavailable, "service unavailable")
		shouldRetry, _ := handler.HandleError(err, "test_operation", 5, StateConnected)

		assert.False(t, shouldRetry)
	})
}

func TestErrorHandler_Callbacks(t *testing.T) {
	handler := NewErrorHandler()

	var receivedError error
	var receivedCtx ErrorContext

	callback := func(err error, ctx ErrorContext) {
		receivedError = err
		receivedCtx = ctx
	}

	handler.RegisterErrorCallback(callback)

	testErr := errors.New("test error")
	handler.HandleError(testErr, "test_op", 1, StateConnecting)

	assert.Equal(t, testErr, receivedError)
	assert.Equal(t, "test_op", receivedCtx.Operation)
	assert.Equal(t, 1, receivedCtx.RetryCount)
}

func TestErrorHandler_MultipleCallbacks(t *testing.T) {
	handler := NewErrorHandler()

	callCount := 0
	callback1 := func(err error, ctx ErrorContext) {
		callCount++
	}
	callback2 := func(err error, ctx ErrorContext) {
		callCount++
	}

	handler.RegisterErrorCallback(callback1)
	handler.RegisterErrorCallback(callback2)

	handler.HandleError(errors.New("test"), "op", 0, StateConnected)

	assert.Equal(t, 2, callCount)
}

func TestErrorHandler_GetMetrics(t *testing.T) {
	handler := NewErrorHandler()

	// Generate some errors
	handler.HandleError(errors.New("connection refused"), "op", 0, StateConnected) // retryable (string match)
	handler.HandleError(status.Error(codes.Unavailable, "unavailable"), "op", 0, StateConnected) // retryable (gRPC code)
	handler.HandleError(status.Error(codes.InvalidArgument, "invalid"), "op", 0, StateConnected) // non-retryable (gRPC code)

	metrics := handler.GetMetrics()

	assert.Equal(t, int64(3), metrics.TotalErrors)
	// Both "connection refused" (string match) and Unavailable (gRPC code) are retryable
	assert.Equal(t, int64(2), metrics.RetryableErrors)
	// InvalidArgument is non-retryable
	assert.Equal(t, int64(1), metrics.FatalErrors)
	assert.False(t, metrics.LastError.IsZero())
	assert.Greater(t, len(metrics.ErrorCounts), 0)
}

func TestErrorHandler_ResetMetrics(t *testing.T) {
	handler := NewErrorHandler()

	// Generate an error
	handler.HandleError(errors.New("test"), "op", 0, StateConnected)

	// Reset metrics
	handler.ResetMetrics()

	metrics := handler.GetMetrics()
	assert.Equal(t, int64(0), metrics.TotalErrors)
	assert.Equal(t, int64(0), metrics.RetryableErrors)
	assert.Equal(t, int64(0), metrics.FatalErrors)
	assert.True(t, metrics.LastError.IsZero())
}

func TestErrorHandler_ShouldReconnect(t *testing.T) {
	handler := NewErrorHandler()

	t.Run("should reconnect with retryable error under max attempts", func(t *testing.T) {
		err := status.Error(codes.Unavailable, "unavailable")
		assert.True(t, handler.ShouldReconnect(0, err))
		assert.True(t, handler.ShouldReconnect(4, err))
	})

	t.Run("should not reconnect at max attempts", func(t *testing.T) {
		err := status.Error(codes.Unavailable, "unavailable")
		assert.False(t, handler.ShouldReconnect(5, err))
	})

	t.Run("should not reconnect with non-retryable error", func(t *testing.T) {
		err := status.Error(codes.InvalidArgument, "invalid")
		assert.False(t, handler.ShouldReconnect(0, err))
	})
}

func TestErrorHandler_SetRetryConfig(t *testing.T) {
	handler := NewErrorHandler()

	newConfig := RetryConfig{
		MaxRetries:        10,
		InitialBackoff:    1 * time.Second,
		MaxBackoff:        60 * time.Second,
		BackoffMultiplier: 3.0,
	}

	handler.SetRetryConfig(newConfig)

	backoff := handler.CalculateBackoff(0)
	assert.Equal(t, 1*time.Second, backoff)
}

func TestErrorHandler_SetReconnectConfig(t *testing.T) {
	handler := NewErrorHandler()

	newConfig := ReconnectConfig{
		MaxReconnectAttempts: 10,
		ReconnectDelay:       5 * time.Second,
		HealthCheckInterval:  60 * time.Second,
	}

	handler.SetReconnectConfig(newConfig)
	assert.Equal(t, 5*time.Second, handler.GetReconnectDelay())
}

func TestContainsString(t *testing.T) {
	tests := []struct {
		s        string
		substr   string
		expected bool
	}{
		{"Hello World", "world", true},
		{"Hello World", "World", true},
		{"Hello World", "Hello", true},
		{"Hello World", "foo", false},
		{"", "foo", false},
		{"foo", "", true},
		{"foo", "foobar", false},
	}

	for _, tt := range tests {
		t.Run(tt.s+"_"+tt.substr, func(t *testing.T) {
			result := containsString(tt.s, tt.substr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToLower(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "hello world"},
		{"HELLO", "hello"},
		{"hello", "hello"},
		{"123ABC", "123abc"},
		{"", ""},
		{"!@#$%", "!@#$%"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toLower(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClassifyError(t *testing.T) {
	tests := []struct {
		err      error
		expected string
	}{
		{nil, "none"},
		{status.Error(codes.Unavailable, "test"), "grpc_Unavailable"},
		{status.Error(codes.Internal, "test"), "grpc_Internal"},
		{errors.New("connection refused"), "connection"},
		{errors.New("timeout"), "timeout"},
		{errors.New("stream closed"), "stream"},
		{errors.New("context cancelled"), "context"},
		{errors.New("something else"), "other"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := classifyError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewGRPCError(t *testing.T) {
	t.Run("creates from gRPC status error", func(t *testing.T) {
		err := status.Error(codes.Unavailable, "service unavailable")
		grpcErr := NewGRPCError(err)

		assert.NotNil(t, grpcErr)
		assert.Equal(t, codes.Unavailable, grpcErr.Code)
		assert.Equal(t, "service unavailable", grpcErr.Message)
		assert.True(t, grpcErr.Retryable)
	})

	t.Run("creates from generic error", func(t *testing.T) {
		err := errors.New("generic error")
		grpcErr := NewGRPCError(err)

		assert.NotNil(t, grpcErr)
		assert.Equal(t, codes.Unknown, grpcErr.Code)
		assert.Equal(t, "generic error", grpcErr.Message)
		assert.True(t, grpcErr.Retryable)
	})

	t.Run("returns nil for nil error", func(t *testing.T) {
		grpcErr := NewGRPCError(nil)
		assert.Nil(t, grpcErr)
	})

	t.Run("identifies non-retryable codes", func(t *testing.T) {
		err := status.Error(codes.InvalidArgument, "bad request")
		grpcErr := NewGRPCError(err)

		assert.False(t, grpcErr.Retryable)
	})
}

func TestGRPCError_Error(t *testing.T) {
	grpcErr := &GRPCError{
		Code:    codes.Unavailable,
		Message: "service unavailable",
	}

	expected := "gRPC error [Unavailable]: service unavailable"
	assert.Equal(t, expected, grpcErr.Error())
}

func BenchmarkErrorHandler_IsRetryableError(b *testing.B) {
	handler := NewErrorHandler()
	err := status.Error(codes.Unavailable, "service unavailable")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.IsRetryableError(err)
	}
}

func BenchmarkContainsString(b *testing.B) {
	s := "This is a test string with connection refused error message"
	substr := "connection refused"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		containsString(s, substr)
	}
}