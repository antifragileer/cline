// Package exit provides tests for enhanced error handling.
package exit

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorCategory_String(t *testing.T) {
	tests := []struct {
		category ErrorCategory
		expected string
	}{
		{ErrorCategoryUnknown, "unknown"},
		{ErrorCategoryValidation, "validation"},
		{ErrorCategoryConfiguration, "configuration"},
		{ErrorCategoryConnection, "connection"},
		{ErrorCategoryAuthentication, "authentication"},
		{ErrorCategoryAuthorization, "authorization"},
		{ErrorCategoryTimeout, "timeout"},
		{ErrorCategoryCancelled, "cancelled"},
		{ErrorCategoryTaskFailure, "task_failure"},
		{ErrorCategorySystem, "system"},
		{ErrorCategoryUser, "user"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.category.String())
		})
	}
}

func TestCategorizedError_Error(t *testing.T) {
	t.Run("returns message when set", func(t *testing.T) {
		ce := &CategorizedError{
			Err:     errors.New("original error"),
			Message: "user-friendly message",
		}
		assert.Equal(t, "user-friendly message", ce.Error())
	})

	t.Run("returns original error when no message", func(t *testing.T) {
		ce := &CategorizedError{
			Err: errors.New("original error"),
		}
		assert.Equal(t, "original error", ce.Error())
	})

	t.Run("returns unknown when no error or message", func(t *testing.T) {
		ce := &CategorizedError{}
		assert.Equal(t, "unknown error", ce.Error())
	})
}

func TestCategorizedError_Unwrap(t *testing.T) {
	originalErr := errors.New("original")
	ce := &CategorizedError{
		Err: originalErr,
	}

	assert.Equal(t, originalErr, ce.Unwrap())
}

func TestCategorizedError_Format(t *testing.T) {
	ce := &CategorizedError{
		Err:         errors.New("original error"),
		Category:    ErrorCategoryValidation,
		ExitCode:    InvalidArguments,
		Message:     "Invalid input",
		Suggestion:  "Check your arguments",
		Context:     map[string]interface{}{"field": "username"},
		Timestamp:   time.Now(),
		Recoverable: true,
		Retryable:   false,
	}

	t.Run("formats with suggestion", func(t *testing.T) {
		formatted := ce.Format(false)
		assert.Contains(t, formatted, "Error: Invalid input")
		assert.Contains(t, formatted, "Suggestion: Check your arguments")
	})

	t.Run("formats verbose", func(t *testing.T) {
		formatted := ce.Format(true)
		assert.Contains(t, formatted, "Error: Invalid input")
		assert.Contains(t, formatted, "Category: validation")
		assert.Contains(t, formatted, "Exit Code: 2")
		assert.Contains(t, formatted, "Context:")
	})
}

func TestNewErrorClassifier(t *testing.T) {
	classifier := NewErrorClassifier()
	require.NotNil(t, classifier)
	assert.NotEmpty(t, classifier.classifiers)
}

func TestErrorClassifier_AddClassifier(t *testing.T) {
	classifier := NewErrorClassifier()

	customClassifier := func(err error) (*CategorizedError, bool) {
		if err.Error() == "custom error" {
			return &CategorizedError{
				Err:      err,
				Category: ErrorCategoryUser,
				ExitCode: GeneralError,
			}, true
		}
		return nil, false
	}

	classifier.AddClassifier(customClassifier)

	ce := classifier.Classify(errors.New("custom error"))
	require.NotNil(t, ce)
	assert.Equal(t, ErrorCategoryUser, ce.Category)
}

func TestErrorClassifier_Classify(t *testing.T) {
	classifier := NewErrorClassifier()

	tests := []struct {
		name          string
		err           error
		wantCategory  ErrorCategory
		wantExitCode  Code
		wantRetryable bool
	}{
		{
			name:          "validation error",
			err:           errors.New("invalid argument provided"),
			wantCategory:  ErrorCategoryValidation,
			wantExitCode:  InvalidArguments,
			wantRetryable: false,
		},
		{
			name:          "connection error",
			err:           errors.New("connection refused"),
			wantCategory:  ErrorCategoryConnection,
			wantExitCode:  ConnectionError,
			wantRetryable: true,
		},
		{
			name:          "authentication error",
			err:           errors.New("authentication failed"),
			wantCategory:  ErrorCategoryAuthentication,
			wantExitCode:  PermissionDenied,
			wantRetryable: false,
		},
		{
			name:          "timeout error",
			err:           errors.New("operation timeout"),
			wantCategory:  ErrorCategoryTimeout,
			wantExitCode:  Timeout,
			wantRetryable: true,
		},
		{
			name:          "configuration error",
			err:           errors.New("configuration invalid"),
			wantCategory:  ErrorCategoryConfiguration,
			wantExitCode:  ConfigurationError,
			wantRetryable: false,
		},
		{
			name:          "permission error",
			err:           errors.New("permission denied"),
			wantCategory:  ErrorCategoryAuthorization,
			wantExitCode:  PermissionDenied,
			wantRetryable: false,
		},
		{
			name:          "unknown error",
			err:           errors.New("something unexpected"),
			wantCategory:  ErrorCategoryUnknown,
			wantExitCode:  GeneralError,
			wantRetryable: false,
		},
		{
			name:          "nil error",
			err:           nil,
			wantCategory:  ErrorCategoryUnknown,
			wantExitCode:  Success,
			wantRetryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ce := classifier.Classify(tt.err)
			if tt.err == nil {
				assert.Nil(t, ce)
				return
			}

			require.NotNil(t, ce)
			assert.Equal(t, tt.wantCategory, ce.Category)
			assert.Equal(t, tt.wantExitCode, ce.ExitCode)
			assert.Equal(t, tt.wantRetryable, ce.Retryable)
		})
	}
}

func TestErrorClassifier_Classify_AlreadyCategorized(t *testing.T) {
	classifier := NewErrorClassifier()

	original := &CategorizedError{
		Err:      errors.New("test"),
		Category: ErrorCategorySystem,
		ExitCode: TaskFailed,
	}

	result := classifier.Classify(original)
	assert.Equal(t, original, result)
}

func TestNewErrorRecovery(t *testing.T) {
	recovery := NewErrorRecovery()
	require.NotNil(t, recovery)
	assert.Equal(t, 3, recovery.maxRetries)
	assert.Equal(t, 1*time.Second, recovery.retryDelay)
}

func TestErrorRecovery_ExecuteWithRetry(t *testing.T) {
	t.Run("succeeds on first attempt", func(t *testing.T) {
		recovery := NewErrorRecovery()
		attempts := 0

		ctx := context.Background()
		err := recovery.ExecuteWithRetry(ctx, func() error {
			attempts++
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, 1, attempts)
	})

	t.Run("retries on retryable error", func(t *testing.T) {
		recovery := NewErrorRecovery()
		recovery.retryDelay = 1 * time.Millisecond // Speed up test
		attempts := 0

		ctx := context.Background()
		err := recovery.ExecuteWithRetry(ctx, func() error {
			attempts++
			if attempts < 3 {
				return errors.New("connection refused")
			}
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, 3, attempts)
	})

	t.Run("fails on non-retryable error", func(t *testing.T) {
		recovery := NewErrorRecovery()
		attempts := 0

		ctx := context.Background()
		err := recovery.ExecuteWithRetry(ctx, func() error {
			attempts++
			return errors.New("invalid argument") // Not retryable
		})

		assert.Error(t, err)
		assert.Equal(t, 1, attempts)
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		recovery := NewErrorRecovery()
		recovery.retryDelay = 1 * time.Hour // Will be cancelled before retry

		ctx, cancel := context.WithCancel(context.Background())

		attempts := 0
		go func() {
			time.Sleep(10 * time.Millisecond)
			cancel()
		}()

		err := recovery.ExecuteWithRetry(ctx, func() error {
			attempts++
			return errors.New("connection refused")
		})

		assert.Error(t, err)
		assert.Equal(t, 1, attempts) // Only one attempt before cancellation
	})

	t.Run("exhausts max retries", func(t *testing.T) {
		recovery := NewErrorRecovery()
		recovery.retryDelay = 1 * time.Millisecond
		recovery.maxRetries = 2
		attempts := 0

		ctx := context.Background()
		err := recovery.ExecuteWithRetry(ctx, func() error {
			attempts++
			return errors.New("connection refused")
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "max retries exceeded")
		assert.Equal(t, 3, attempts) // Initial + 2 retries
	})
}

func TestErrorRecovery_HandleError(t *testing.T) {
	recovery := NewErrorRecovery()
	var buf strings.Builder

	t.Run("handles nil error", func(t *testing.T) {
		code := recovery.HandleError(nil, &buf, false)
		assert.Equal(t, Code(Success), code)
	})

	t.Run("handles categorized error", func(t *testing.T) {
		buf.Reset()
		err := errors.New("connection refused")
		code := recovery.HandleError(err, &buf, false)

		assert.Equal(t, Code(ConnectionError), code)
		output := buf.String()
		assert.Contains(t, output, "Error:")
	})

	t.Run("includes suggestion when available", func(t *testing.T) {
		buf.Reset()
		err := errors.New("permission denied")
		code := recovery.HandleError(err, &buf, false)

		assert.Equal(t, Code(PermissionDenied), code)
		output := buf.String()
		assert.Contains(t, output, "Suggestion:")
	})
}

func TestNewEnhancedExitHandler(t *testing.T) {
	handler := NewEnhancedExitHandler()
	require.NotNil(t, handler)
	require.NotNil(t, handler.Handler)
	require.NotNil(t, handler.classifier)
	require.NotNil(t, handler.recovery)
	assert.NotNil(t, handler.errorWriter)
	assert.Equal(t, 100, handler.maxErrorLog)
}

func TestEnhancedExitHandler_SetVerbose(t *testing.T) {
	handler := NewEnhancedExitHandler()
	assert.False(t, handler.verbose)

	handler.SetVerbose(true)
	assert.True(t, handler.verbose)
}

func TestEnhancedExitHandler_SetErrorWriter(t *testing.T) {
	handler := NewEnhancedExitHandler()
	var buf strings.Builder

	handler.SetErrorWriter(&buf)
	assert.Equal(t, &buf, handler.errorWriter)
}

func TestEnhancedExitHandler_HandleError(t *testing.T) {
	handler := NewEnhancedExitHandler()
	var buf strings.Builder
	handler.SetErrorWriter(&buf)

	t.Run("handles error and logs it", func(t *testing.T) {
		buf.Reset()
		testErr := errors.New("test error")
		code := handler.HandleError(testErr)

		assert.Equal(t, Code(GeneralError), code)
		assert.Equal(t, Code(GeneralError), handler.GetExitCode())

		log := handler.GetErrorLog()
		require.Len(t, log, 1)
		assert.Contains(t, log[0].Error, "test error")
	})
}

func TestEnhancedExitHandler_HandleErrorWithRecovery(t *testing.T) {
	handler := NewEnhancedExitHandler()
	handler.recovery.retryDelay = 1 * time.Millisecond

	t.Run("succeeds after retry", func(t *testing.T) {
		handler.ClearErrorLog()
		attempts := 0

		ctx := context.Background()
		err := handler.HandleErrorWithRecovery(ctx, func() error {
			attempts++
			if attempts < 2 {
				return errors.New("connection refused")
			}
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, 2, attempts)
	})

	t.Run("fails and handles error", func(t *testing.T) {
		handler.ClearErrorLog()
		var buf strings.Builder
		handler.SetErrorWriter(&buf)

		ctx := context.Background()
		err := handler.HandleErrorWithRecovery(ctx, func() error {
			return errors.New("invalid argument")
		})

		assert.Error(t, err)
		assert.Equal(t, Code(InvalidArguments), handler.GetExitCode())
	})
}

func TestEnhancedExitHandler_GetErrorLog(t *testing.T) {
	handler := NewEnhancedExitHandler()

	// Add some errors
	for i := 0; i < 5; i++ {
		ce := &CategorizedError{
			Err:      errors.New("test"),
			Category: ErrorCategoryValidation,
			ExitCode: InvalidArguments,
		}
		handler.logError(ce.Err, ce, InvalidArguments, false)
	}

	log := handler.GetErrorLog()
	require.Len(t, log, 5)
}

func TestEnhancedExitHandler_ClearErrorLog(t *testing.T) {
	handler := NewEnhancedExitHandler()

	handler.logError(errors.New("test"), &CategorizedError{}, GeneralError, false)
	assert.Len(t, handler.GetErrorLog(), 1)

	handler.ClearErrorLog()
	assert.Len(t, handler.GetErrorLog(), 0)
}

func TestEnhancedExitHandler_GetErrorStats(t *testing.T) {
	handler := NewEnhancedExitHandler()

	// Add errors of different categories
	handler.logError(errors.New("val"), &CategorizedError{Category: ErrorCategoryValidation}, InvalidArguments, false)
	handler.logError(errors.New("conn"), &CategorizedError{Category: ErrorCategoryConnection}, ConnectionError, true)
	handler.logError(errors.New("conn2"), &CategorizedError{Category: ErrorCategoryConnection}, ConnectionError, false)

	stats := handler.GetErrorStats()
	assert.Equal(t, 3, stats.TotalErrors)
	assert.Equal(t, 2, stats.ByCategory["connection"])
	assert.Equal(t, 1, stats.ByCategory["validation"])
	assert.Equal(t, 2, stats.ByExitCode[ConnectionError])
	assert.Equal(t, 1, stats.RecoveredErrors)
}

func TestWrapError(t *testing.T) {
	t.Run("returns nil for nil error", func(t *testing.T) {
		result := WrapError(nil, ErrorCategoryValidation, "message")
		assert.Nil(t, result)
	})

	t.Run("wraps error with category", func(t *testing.T) {
		original := errors.New("original")
		wrapped := WrapError(original, ErrorCategoryValidation, "Invalid input")

		require.NotNil(t, wrapped)
		assert.Equal(t, original, wrapped.Err)
		assert.Equal(t, ErrorCategoryValidation, wrapped.Category)
		assert.Equal(t, "Invalid input", wrapped.Message)
		assert.Equal(t, Code(InvalidArguments), wrapped.ExitCode)
	})

	t.Run("maps categories to exit codes", func(t *testing.T) {
		tests := []struct {
			category ErrorCategory
			wantCode Code
		}{
			{ErrorCategoryValidation, InvalidArguments},
			{ErrorCategoryConfiguration, ConfigurationError},
			{ErrorCategoryConnection, ConnectionError},
			{ErrorCategoryAuthentication, PermissionDenied},
			{ErrorCategoryAuthorization, PermissionDenied},
			{ErrorCategoryTimeout, Timeout},
			{ErrorCategoryTaskFailure, TaskFailed},
			{ErrorCategorySystem, GeneralError},
		}

		for _, tt := range tests {
			err := errors.New("test")
			wrapped := WrapError(err, tt.category, "test")
			assert.Equal(t, tt.wantCode, wrapped.ExitCode, "Category %s", tt.category)
		}
	})
}

func TestIsCategorizedError(t *testing.T) {
	t.Run("returns true for categorized error", func(t *testing.T) {
		ce := &CategorizedError{
			Err:      errors.New("test"),
			Category: ErrorCategoryValidation,
		}

		result, ok := IsCategorizedError(ce)
		assert.True(t, ok)
		assert.Equal(t, ce, result)
	})

	t.Run("returns false for regular error", func(t *testing.T) {
		err := errors.New("regular error")
		result, ok := IsCategorizedError(err)
		assert.False(t, ok)
		assert.Nil(t, result)
	})

	t.Run("unwraps wrapped errors", func(t *testing.T) {
		ce := &CategorizedError{
			Err:      errors.New("inner"),
			Category: ErrorCategoryValidation,
		}
		wrapped := fmt.Errorf("outer: %w", ce)

		result, ok := IsCategorizedError(wrapped)
		assert.True(t, ok)
		assert.Equal(t, ce, result)
	})
}

func TestCategorizeError(t *testing.T) {
	t.Run("returns nil for nil error", func(t *testing.T) {
		result := CategorizeError(nil)
		assert.Nil(t, result)
	})

	t.Run("returns existing categorized error", func(t *testing.T) {
		ce := &CategorizedError{
			Err:      errors.New("test"),
			Category: ErrorCategorySystem,
		}

		result := CategorizeError(ce)
		assert.Equal(t, ce, result)
	})

	t.Run("classifies regular errors", func(t *testing.T) {
		err := errors.New("invalid argument")
		result := CategorizeError(err)

		require.NotNil(t, result)
		assert.Equal(t, ErrorCategoryValidation, result.Category)
		assert.Equal(t, Code(InvalidArguments), result.ExitCode)
	})
}

func TestRecoverFromPanic(t *testing.T) {
	t.Run("returns nil for nil", func(t *testing.T) {
		err := RecoverFromPanic(nil)
		assert.Nil(t, err)
	})

	t.Run("recovers from error", func(t *testing.T) {
		original := errors.New("panic error")
		err := RecoverFromPanic(original)

		require.NotNil(t, err)
		assert.Contains(t, err.Error(), "panic error")
	})

	t.Run("recovers from string", func(t *testing.T) {
		err := RecoverFromPanic("string panic")

		require.NotNil(t, err)
		assert.Contains(t, err.Error(), "string panic")
	})

	t.Run("recovers from other types", func(t *testing.T) {
		err := RecoverFromPanic(42)

		require.NotNil(t, err)
		assert.Contains(t, err.Error(), "panic: 42")
	})
}

func TestSafeExecute(t *testing.T) {
	t.Run("executes successfully", func(t *testing.T) {
		called := false
		err := SafeExecute(func() error {
			called = true
			return nil
		})

		assert.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("returns error", func(t *testing.T) {
		expectedErr := errors.New("test error")
		err := SafeExecute(func() error {
			return expectedErr
		})

		assert.Equal(t, expectedErr, err)
	})

	t.Run("recovers from panic", func(t *testing.T) {
		err := SafeExecute(func() error {
			panic("oh no")
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "oh no")
		assert.Contains(t, err.Error(), "Internal error occurred")
	})

	t.Run("recovers from panic with error", func(t *testing.T) {
		expectedErr := errors.New("panic error")
		err := SafeExecute(func() error {
			panic(expectedErr)
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "panic error")
	})
}

func BenchmarkClassifyError(b *testing.B) {
	classifier := NewErrorClassifier()
	err := errors.New("connection refused")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		classifier.Classify(err)
	}
}

func BenchmarkWrapError(b *testing.B) {
	err := errors.New("test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		WrapError(err, ErrorCategoryValidation, "message")
	}
}
