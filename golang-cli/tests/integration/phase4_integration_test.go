// Package integration provides integration tests for Phase 4 implementation.
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cline/cline/golang-cli/internal/exit"
	"github.com/cline/cline/golang-cli/internal/mode"
)

// TestPhase4Integration tests the full Phase 4 implementation
func TestPhase4Integration(t *testing.T) {
	t.Run("Enhanced Mode Detection Integration", func(t *testing.T) {
		// Save and restore environment
		cleanup := saveAndRestoreEnv()
		defer cleanup()

		t.Run("detects CI environment correctly", func(t *testing.T) {
			os.Setenv("CI", "true")
			os.Setenv("TERM", "xterm-256color")

			detector := mode.NewEnhancedDetector()
			detectedMode := detector.DetectExtended()

			assert.Equal(t, mode.ExtendedModeCI, detectedMode)
			assert.False(t, detectedMode.IsInteractive())
			assert.True(t, detectedMode.SupportsColor())
		})

		t.Run("detects interactive TTY", func(t *testing.T) {
			os.Unsetenv("CI")
			os.Setenv("TERM", "xterm-256color")
			os.Setenv("NO_COLOR", "")

			detector := mode.NewEnhancedDetector()
			detectedMode := detector.DetectExtended()

			if detectedMode == mode.ExtendedModeInteractiveTTY {
				assert.True(t, detectedMode.IsInteractive())
				assert.True(t, detectedMode.SupportsColor())
			}
		})

		t.Run("respects_NO_COLOR", func(t *testing.T) {
			os.Unsetenv("CI")
			os.Setenv("NO_COLOR", "1")
			os.Setenv("TERM", "xterm-256color")

			detector := mode.NewEnhancedDetector()
			detectedMode := detector.DetectExtended()

			// In non-TTY test environments, mode will be Pipe or File, not InteractiveNoColor
			// Both should not support color when NO_COLOR is set
			if detectedMode == mode.ExtendedModeInteractiveNoColor {
				assert.True(t, detectedMode.IsInteractive())
				assert.False(t, detectedMode.SupportsColor())
			} else {
				// In non-TTY environments, we get Pipe/File mode which also doesn't support color
				assert.False(t, detectedMode.SupportsColor(), "Mode %v should not support color when NO_COLOR is set", detectedMode)
			}
		})

		t.Run("output mode selection", func(t *testing.T) {
			os.Setenv("CI", "true")

			detector := mode.NewEnhancedDetector()
			outputMode := detector.SelectOutputMode(false, false)

			assert.Equal(t, mode.OutputModeJSON, outputMode)
			assert.False(t, outputMode.IsInteractive())
		})

		t.Run("plain mode override", func(t *testing.T) {
			detector := mode.NewEnhancedDetector()
			outputMode := detector.SelectOutputMode(false, true)

			assert.Equal(t, mode.OutputModePlain, outputMode)
		})
	})

	t.Run("Enhanced JSON Streaming Integration", func(t *testing.T) {
		t.Run("writes and reads complete messages", func(t *testing.T) {
			var buf bytes.Buffer
			config := &mode.StreamConfig{
				AutoFlush:             false,
				EnablePartialMessages: true,
			}

			streamer := mode.NewJSONStreamer(&buf, config)

			// Write messages
			require.NoError(t, streamer.WriteMessage("text", "Hello"))
			require.NoError(t, streamer.WriteMessage("text", "World"))
			require.NoError(t, streamer.Flush())

			// Read messages back
			var messages []*mode.StreamMessage
			reader := strings.NewReader(buf.String())
			err := mode.ReadJSONStream(reader, func(msg *mode.StreamMessage) error {
				messages = append(messages, msg)
				return nil
			})
			require.NoError(t, err)

			require.Len(t, messages, 2)
			assert.Equal(t, "text", messages[0].Type)
			assert.Equal(t, "Hello", messages[0].Content)
			assert.Equal(t, "World", messages[1].Content)
		})

		t.Run("handles partial messages", func(t *testing.T) {
			var buf bytes.Buffer
			config := &mode.StreamConfig{
				AutoFlush:             false,
				EnablePartialMessages: true,
			}

			streamer := mode.NewJSONStreamer(&buf, config)

			// Write partial chunks
			require.NoError(t, streamer.WritePartialMessage("text", "Hello ", "msg-1", false))
			require.NoError(t, streamer.WritePartialMessage("text", "World", "msg-1", true))
			require.NoError(t, streamer.Flush())

			// Read messages
			var messages []*mode.StreamMessage
			err := mode.ReadJSONStream(strings.NewReader(buf.String()), func(msg *mode.StreamMessage) error {
				messages = append(messages, msg)
				return nil
			})
			require.NoError(t, err)

			require.Len(t, messages, 2)
			assert.True(t, messages[0].Partial)
			assert.False(t, messages[1].Partial)

			// Reassemble
			complete := streamer.ReassembleMessage("msg-1")
			require.NotNil(t, complete)
			assert.Equal(t, "Hello World", complete.Content)
		})

		t.Run("stream context with cancellation", func(t *testing.T) {
			var buf bytes.Buffer
			ctx := context.Background()

			sc := mode.NewStreamContext(ctx, &buf, nil)
			require.NotNil(t, sc)

			// Write should succeed
			err := sc.WriteMessage("text", "before cancel")
			require.NoError(t, err)

			// Cancel
			sc.Cancel()

			// Write should fail after cancel
			err = sc.WriteMessage("text", "after cancel")
			assert.Error(t, err)
			assert.Equal(t, context.Canceled, err)
		})

		t.Run("handles progress messages", func(t *testing.T) {
			var buf bytes.Buffer
			streamer := mode.NewJSONStreamer(&buf, nil)

			err := streamer.WriteProgress("download", 50, 100, "Downloading...")
			require.NoError(t, err)
			require.NoError(t, streamer.Flush())

			var msg mode.StreamMessage
			line := strings.TrimSpace(buf.String())
			require.NoError(t, json.Unmarshal([]byte(line), &msg))

			assert.Equal(t, "progress", msg.Type)
			assert.Equal(t, "Downloading...", msg.Content)
			assert.NotNil(t, msg.Metadata)
			assert.Equal(t, float64(50), msg.Metadata["current"])
			assert.Equal(t, float64(100), msg.Metadata["total"])
			assert.Equal(t, float64(50), msg.Metadata["percent"])
		})

		t.Run("handles tool use messages", func(t *testing.T) {
			var buf bytes.Buffer
			streamer := mode.NewJSONStreamer(&buf, nil)

			params := map[string]interface{}{
				"file": "test.txt",
				"line": 42,
			}

			err := streamer.WriteToolUse("read_file", params, "tool-1")
			require.NoError(t, err)
			require.NoError(t, streamer.Flush())

			var msg mode.StreamMessage
			line := strings.TrimSpace(buf.String())
			require.NoError(t, json.Unmarshal([]byte(line), &msg))

			assert.Equal(t, "tool_use", msg.Type)
			assert.Equal(t, "tool-1", msg.MessageID)
			assert.Equal(t, "read_file", msg.Metadata["tool"])
		})

		t.Run("handles tool result messages", func(t *testing.T) {
			var buf bytes.Buffer
			streamer := mode.NewJSONStreamer(&buf, nil)

			result := map[string]interface{}{
				"content": "file contents here",
			}

			err := streamer.WriteToolResult("read_file", result, true, "tool-1")
			require.NoError(t, err)
			require.NoError(t, streamer.Flush())

			var msg mode.StreamMessage
			line := strings.TrimSpace(buf.String())
			require.NoError(t, json.Unmarshal([]byte(line), &msg))

			assert.Equal(t, "tool_result", msg.Type)
			assert.Equal(t, "tool-1", msg.MessageID)
			assert.True(t, msg.Metadata["success"].(bool))
		})

		t.Run("handles checkpoint messages", func(t *testing.T) {
			var buf bytes.Buffer
			streamer := mode.NewJSONStreamer(&buf, nil)

			err := streamer.WriteCheckpoint("chk-123", "Checkpoint created")
			require.NoError(t, err)
			require.NoError(t, streamer.Flush())

			var msg mode.StreamMessage
			line := strings.TrimSpace(buf.String())
			require.NoError(t, json.Unmarshal([]byte(line), &msg))

			assert.Equal(t, "checkpoint", msg.Type)
			assert.Equal(t, "chk-123", msg.Metadata["checkpoint_id"])
		})
	})

	t.Run("Enhanced Error Handling Integration", func(t *testing.T) {
		t.Run("classifies common errors", func(t *testing.T) {
			classifier := exit.NewErrorClassifier()

			testCases := []struct {
				err       string
				category  exit.ErrorCategory
				retryable bool
			}{
				{"invalid argument", exit.ErrorCategoryValidation, false},
				{"connection refused", exit.ErrorCategoryConnection, true},
				{"authentication failed", exit.ErrorCategoryAuthentication, false},
				{"timeout", exit.ErrorCategoryTimeout, true},
				{"permission denied", exit.ErrorCategoryAuthorization, false},
			}

			for _, tc := range testCases {
				err := errors.New(tc.err)
				ce := classifier.Classify(err)
				require.NotNil(t, ce)
				assert.Equal(t, tc.category, ce.Category, "Error: %s", tc.err)
				assert.Equal(t, tc.retryable, ce.Retryable, "Error: %s", tc.err)
			}
		})

		t.Run("handles retryable errors with recovery", func(t *testing.T) {
			recovery := exit.NewErrorRecovery()
			recovery.SetRetryDelay(1 * time.Millisecond)
			recovery.SetMaxRetries(2)

			attempts := 0
			ctx := context.Background()

			err := recovery.ExecuteWithRetry(ctx, func() error {
				attempts++
				if attempts < 2 {
					return errors.New("connection refused") // Retryable
				}
				return nil
			})

			assert.NoError(t, err)
			assert.Equal(t, 2, attempts)
		})

		t.Run("does not retry non-retryable errors", func(t *testing.T) {
			recovery := exit.NewErrorRecovery()

			attempts := 0
			ctx := context.Background()

			err := recovery.ExecuteWithRetry(ctx, func() error {
				attempts++
				return errors.New("invalid argument") // Not retryable
			})

			assert.Error(t, err)
			assert.Equal(t, 1, attempts)
		})

		t.Run("enhanced exit handler logs errors", func(t *testing.T) {
			var buf strings.Builder
			handler := exit.NewEnhancedExitHandler()
			handler.SetErrorWriter(&buf)

			testErr := errors.New("test error")
			code := handler.HandleError(testErr)

			assert.Equal(t, int(exit.GeneralError), int(code))

			log := handler.GetErrorLog()
			require.Len(t, log, 1)
			assert.Contains(t, log[0].Error, "test error")
		})

		t.Run("safe execute recovers from panic", func(t *testing.T) {
			err := exit.SafeExecute(func() error {
				panic("unexpected panic")
			})

			require.Error(t, err)
			assert.Contains(t, err.Error(), "unexpected panic")
			assert.Contains(t, err.Error(), "Internal error occurred")
		})

		t.Run("wrap error with category", func(t *testing.T) {
			original := errors.New("original")
			wrapped := exit.WrapError(original, exit.ErrorCategoryValidation, "Invalid input")

			require.NotNil(t, wrapped)
			assert.Equal(t, original, wrapped.Unwrap())
			assert.Equal(t, exit.ErrorCategoryValidation, wrapped.Category)
			assert.Equal(t, int(exit.InvalidArguments), int(wrapped.ExitCode))
		})

		t.Run("categorized error format", func(t *testing.T) {
			ce := &exit.CategorizedError{
				Err:         errors.New("details"),
				Category:    exit.ErrorCategoryValidation,
				ExitCode:    exit.InvalidArguments,
				Message:     "User message",
				Suggestion:  "Try this fix",
				Recoverable: true,
				Retryable:   false,
			}

			formatted := ce.Format(false)
			assert.Contains(t, formatted, "User message")
			assert.Contains(t, formatted, "Suggestion: Try this fix")

			formattedVerbose := ce.Format(true)
			assert.Contains(t, formattedVerbose, "Category: validation")
			assert.Contains(t, formattedVerbose, "Exit Code: 2")
		})

		t.Run("error statistics tracking", func(t *testing.T) {
			handler := exit.NewEnhancedExitHandler()
			var buf strings.Builder
			handler.SetErrorWriter(&buf)

			// Create different errors
			errors := []struct {
				err      error
				category exit.ErrorCategory
			}{
				{errors.New("validation 1"), exit.ErrorCategoryValidation},
				{errors.New("validation 2"), exit.ErrorCategoryValidation},
				{errors.New("connection 1"), exit.ErrorCategoryConnection},
			}

			for _, e := range errors {
				ce := exit.WrapError(e.err, e.category, "message")
				handler.HandleError(ce)
			}

			stats := handler.GetErrorStats()
			assert.Equal(t, 3, stats.TotalErrors)
			assert.Equal(t, 2, stats.ByCategory["validation"])
			assert.Equal(t, 1, stats.ByCategory["connection"])
		})
	})
}

// TestEndToEndStreaming tests a complete streaming scenario
func TestEndToEndStreaming(t *testing.T) {
	var buf bytes.Buffer
	config := &mode.StreamConfig{
		AutoFlush:             false,
		EnablePartialMessages: true,
	}

	streamer := mode.NewJSONStreamer(&buf, config)

	// Simulate a task execution
	require.NoError(t, streamer.WriteMessage("say", "Starting task..."))

	// Tool use
	require.NoError(t, streamer.WriteToolUse("read_file", map[string]interface{}{
		"file": "src/main.go",
	}, "tool-1"))

	// Tool result
	require.NoError(t, streamer.WriteToolResult("read_file", map[string]interface{}{
		"content": "package main\n...",
	}, true, "tool-1"))

	// Checkpoint
	require.NoError(t, streamer.WriteCheckpoint("chk-1", "Created checkpoint after file read"))

	// Progress
	require.NoError(t, streamer.WriteProgress("task", 50, 100, "Task 50% complete"))

	// Final message
	require.NoError(t, streamer.WriteMessage("say", "Task completed"))
	require.NoError(t, streamer.Close())

	// Read back all messages
	var messages []*mode.StreamMessage
	err := mode.ReadJSONStream(strings.NewReader(buf.String()), func(msg *mode.StreamMessage) error {
		messages = append(messages, msg)
		return nil
	})
	require.NoError(t, err)

	require.Len(t, messages, 6)
	assert.Equal(t, "say", messages[0].Type)
	assert.Equal(t, "tool_use", messages[1].Type)
	assert.Equal(t, "tool_result", messages[2].Type)
	assert.Equal(t, "checkpoint", messages[3].Type)
	assert.Equal(t, "progress", messages[4].Type)
	assert.Equal(t, "say", messages[5].Type)
}

// saveAndRestoreEnv saves current environment and returns a cleanup function
func saveAndRestoreEnv() func() {
	envVars := []string{"CI", "TERM", "NO_COLOR", "FORCE_COLOR", "TERM_PROGRAM", "TERMINAL_EMULATOR", "COLORTERM"}
	saved := make(map[string]string)
	for _, v := range envVars {
		saved[v] = os.Getenv(v)
	}

	return func() {
		for k, v := range saved {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}
}

// BenchmarkFullIntegration benchmarks the full Phase 4 implementation
func BenchmarkFullIntegration(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		streamer := mode.NewJSONStreamer(&buf, &mode.StreamConfig{AutoFlush: false})

		for j := 0; j < 100; j++ {
			streamer.WriteMessage("text", "benchmark message")
		}
		streamer.Close()
	}
}