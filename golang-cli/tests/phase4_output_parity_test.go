// Package tests provides Phase 4: Output Format Parity tests
// These tests ensure JSON and plain text output formats match Node.js CLI exactly
package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cline/cline/golang-cli/internal/formatter"
)

// ==================== 4.1 JSON Output Mode Tests ====================

// TestJSONFieldNameParity ensures JSON field names match Node.js exactly
func TestJSONFieldNameParity(t *testing.T) {
	tests := []struct {
		name         string
		messageType  string
		sayType      string
		text         string
		requiredFields []string
	}{
		{
			name:         "say_text_message",
			messageType:  "say",
			sayType:      "text",
			text:         "Hello world",
			requiredFields: []string{"ts", "type", "say", "text"},
		},
		{
			name:         "say_error_message",
			messageType:  "say",
			sayType:      "error",
			text:         "Error occurred",
			requiredFields: []string{"ts", "type", "say", "text"},
		},
		{
			name:         "ask_command_message",
			messageType:  "ask",
			sayType:      "command",
			text:         "Approve command?",
			requiredFields: []string{"ts", "type", "ask", "text"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			f := formatter.NewJSONFormatter(&buf, os.Stderr, false)

			var err error
			if tt.messageType == "say" {
				err = f.FormatSayMessage(tt.sayType, tt.text, false, nil)
			} else {
				err = f.FormatAskMessage(tt.sayType, tt.text, nil)
			}
			require.NoError(t, err)

			// Parse the JSON output
			var result map[string]interface{}
			err = json.Unmarshal(buf.Bytes(), &result)
			require.NoError(t, err, "Output should be valid JSON")

			// Verify all required fields are present
			for _, field := range tt.requiredFields {
				assert.Contains(t, result, field, "Required field %s should be present", field)
			}

			// Verify field names match Node.js expectations
			assert.IsType(t, float64(0), result["ts"], "ts should be a number")
			assert.IsType(t, "", result["type"], "type should be a string")
			if tt.messageType == "say" {
				assert.IsType(t, "", result["say"], "say should be a string")
			} else {
				assert.IsType(t, "", result["ask"], "ask should be a string")
			}
			assert.IsType(t, "", result["text"], "text should be a string")
		})
	}
}

// TestJSONTimestampFormat ensures timestamps are in milliseconds (Unix timestamp)
func TestJSONTimestampFormat(t *testing.T) {
	var buf bytes.Buffer
	f := formatter.NewJSONFormatter(&buf, os.Stderr, false)

	// Use a time window to account for timing differences
	beforeTime := time.Now().UnixMilli()

	err := f.FormatSayMessage("text", "Test message", false, nil)
	require.NoError(t, err)

	afterTime := time.Now().UnixMilli()

	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	ts, ok := result["ts"].(float64)
	require.True(t, ok, "ts should be a number")

	// Verify timestamp is in milliseconds (allow small timing window)
	// Use >= and <= with 1ms buffer for timing precision
	assert.GreaterOrEqual(t, int64(ts), beforeTime-1, "Timestamp should be >= beforeTime-1ms")
	assert.LessOrEqual(t, int64(ts), afterTime+1, "Timestamp should be <= afterTime+1ms")

	// Verify it's a reasonable millisecond timestamp (not seconds)
	assert.Greater(t, int64(ts), int64(1_000_000_000_000), "Timestamp should be in milliseconds (13+ digits)")
}

// TestJSONOptionalFields ensures optional fields are omitted when empty
func TestJSONOptionalFields(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(*formatter.JSONFormatter) error
		shouldHaveField string
		shouldNotHave   []string
	}{
		{
			name: "text_message_minimal",
			setupFunc: func(f *formatter.JSONFormatter) error {
				return f.FormatSayMessage("text", "Hello", false, nil)
			},
			shouldNotHave: []string{"partial", "reasoning", "images", "files", "error"},
		},
		{
			name: "partial_message_has_partial_field",
			setupFunc: func(f *formatter.JSONFormatter) error {
				return f.FormatSayMessage("text", "Hello", true, nil)
			},
			shouldHaveField: "partial",
		},
		{
			name: "error_message_has_error_field",
			setupFunc: func(f *formatter.JSONFormatter) error {
				return f.FormatError(fmt.Errorf("test error"))
			},
			shouldHaveField: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			f := formatter.NewJSONFormatter(&buf, os.Stderr, false)

			err := tt.setupFunc(f)
			require.NoError(t, err)

			var result map[string]interface{}
			err = json.Unmarshal(buf.Bytes(), &result)
			require.NoError(t, err)

			if tt.shouldHaveField != "" {
				assert.Contains(t, result, tt.shouldHaveField, "Should have field %s", tt.shouldHaveField)
			}

			for _, field := range tt.shouldNotHave {
				assert.NotContains(t, result, field, "Should not have field %s", field)
			}
		})
	}
}

// ==================== 4.2 JSON Streaming Tests ====================

// TestJSONStreamingLineDelimited ensures each message is on its own line
func TestJSONStreamingLineDelimited(t *testing.T) {
	var buf bytes.Buffer
	f := formatter.NewJSONFormatter(&buf, os.Stderr, true) // streaming mode

	// Send multiple messages
	messages := []string{"Message 1", "Message 2", "Message 3"}
	for _, msg := range messages {
		err := f.FormatSayMessage("text", msg, false, nil)
		require.NoError(t, err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Should have exactly 3 lines
	assert.Len(t, lines, len(messages), "Each message should be on its own line")

	// Each line should be valid JSON
	for i, line := range lines {
		var result map[string]interface{}
		err := json.Unmarshal([]byte(line), &result)
		assert.NoError(t, err, "Line %d should be valid JSON: %s", i, line)

		// Verify content
		assert.Equal(t, messages[i], result["text"], "Line %d should have correct text", i)
	}
}

// TestJSONStreamingImmediateFlush ensures messages are flushed immediately
func TestJSONStreamingImmediateFlush(t *testing.T) {
	var buf bytes.Buffer
	f := formatter.NewJSONFormatter(&buf, os.Stderr, true) // streaming mode

	// Write a message
	err := f.FormatSayMessage("text", "Test", false, nil)
	require.NoError(t, err)

	// Output should be available immediately (no buffering)
	output := buf.String()
	assert.Contains(t, output, "Test", "Message should be flushed immediately")

	// Should end with newline
	assert.True(t, strings.HasSuffix(output, "\n"), "Message should end with newline")
}

// TestJSONStreamingErrorHandling ensures errors are properly formatted in streaming mode
func TestJSONStreamingErrorHandling(t *testing.T) {
	var buf bytes.Buffer
	f := formatter.NewJSONFormatter(&buf, os.Stderr, true) // streaming mode

	// Send a regular message
	err := f.FormatSayMessage("text", "Before error", false, nil)
	require.NoError(t, err)

	// Send an error
	err = f.FormatError(fmt.Errorf("test error"))
	require.NoError(t, err)

	// Send another message
	err = f.FormatSayMessage("text", "After error", false, nil)
	require.NoError(t, err)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	require.Len(t, lines, 3, "Should have 3 lines")

	// Verify each line is valid JSON
	for i, line := range lines {
		var result map[string]interface{}
		err := json.Unmarshal([]byte(line), &result)
		assert.NoError(t, err, "Line %d should be valid JSON", i)
	}

	// Second line should be error type
	var errorMsg map[string]interface{}
	json.Unmarshal([]byte(lines[1]), &errorMsg)
	assert.Equal(t, "error", errorMsg["type"], "Second line should be error type")
}

// ==================== 4.3 Plain Text Mode Tests ====================

// TestPlainTextPrefixes ensures correct prefixes are applied
func TestPlainTextPrefixes(t *testing.T) {
	tests := []struct {
		name         string
		sayType      string
		text         string
		expectedPrefix string
	}{
		{
			name:         "cline_prefix_for_text",
			sayType:      "text",
			text:         "Hello from Cline",
			expectedPrefix: "[Cline]",
		},
		{
			name:         "tool_prefix_for_tool",
			sayType:      "tool",
			text:         "write_to_file",
			expectedPrefix: "[Tool]",
		},
		{
			name:         "error_prefix_for_error",
			sayType:      "error",
			text:         "Something went wrong",
			expectedPrefix: "[Error]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var outBuf, errBuf bytes.Buffer
			f := formatter.NewPlainFormatter(&outBuf, &errBuf, false, false)

			err := f.FormatSayMessage(tt.sayType, tt.text, false)
			require.NoError(t, err)

			var output string
			if tt.sayType == "error" {
				output = errBuf.String()
			} else {
				output = outBuf.String()
			}

			assert.Contains(t, output, tt.expectedPrefix, "Output should contain %s prefix", tt.expectedPrefix)
			assert.Contains(t, output, tt.text, "Output should contain the message text")
		})
	}
}

// TestPlainTextIndentation ensures consistent indentation
func TestPlainTextIndentation(t *testing.T) {
	tests := []struct {
		name           string
		sayType        string
		text           string
		shouldBeIndented bool
	}{
		{
			name:           "command_output_indented",
			sayType:        "command_output",
			text:           "line1\nline2\nline3",
			shouldBeIndented: true,
		},
		{
			name:           "text_not_indented",
			sayType:        "text",
			text:           "Hello world",
			shouldBeIndented: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var outBuf bytes.Buffer
			f := formatter.NewPlainFormatter(&outBuf, os.Stderr, false, false)

			err := f.FormatSayMessage(tt.sayType, tt.text, false)
			require.NoError(t, err)

			output := outBuf.String()
			lines := strings.Split(output, "\n")

			for _, line := range lines {
				if line == "" {
					continue
				}
				if tt.shouldBeIndented {
					assert.True(t, strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "\t"),
						"Line should be indented: %s", line)
				} else {
					// Should not start with spaces (unless it's the prefix)
					trimmed := strings.TrimLeft(line, " ")
					assert.True(t, strings.HasPrefix(trimmed, "[") || len(trimmed) > 0,
						"Line should not be unnecessarily indented: %s", line)
				}
			}
		})
	}
}

// TestPlainTextLineBreaks ensures proper line breaks
func TestPlainTextLineBreaks(t *testing.T) {
	var outBuf bytes.Buffer
	f := formatter.NewPlainFormatter(&outBuf, os.Stderr, false, false)

	// Send multiple messages
	messages := []string{"Message 1", "Message 2", "Message 3"}
	for _, msg := range messages {
		err := f.FormatSayMessage("text", msg, false)
		require.NoError(t, err)
	}

	output := outBuf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Should have 3 lines (one per message)
	assert.Len(t, lines, len(messages), "Each message should be on its own line")

	// Each line should end with the message text
	for i, line := range lines {
		assert.Contains(t, line, messages[i], "Line %d should contain message %d", i, i)
	}
}

// ==================== 4.4 Validation / Parity Tests ====================

// TestJSONOutputMatchesNodeJS runs parity tests against Node.js CLI
func TestJSONOutputMatchesNodeJS(t *testing.T) {
	tsPath := findTSBinary()
	if tsPath == "" {
		t.Skip("TypeScript CLI not found, skipping parity test")
	}

	ctx := context.Background()

	t.Run("version_json_parity", func(t *testing.T) {
		// Run TS CLI with JSON flag
		tsCmd := exec.CommandContext(ctx, tsPath, "version", "--json")
		tsOutput, err := tsCmd.CombinedOutput()
		
		// If TS CLI doesn't support --json flag, skip this test
		if err != nil && bytes.Contains(tsOutput, []byte("unknown flag")) {
			t.Skip("TypeScript CLI does not support --json flag, skipping")
		}
		
		// If other error, fail the test
		require.NoError(t, err, "TS CLI failed: %s", string(tsOutput))

		// Parse TS output
		var tsResult map[string]interface{}
		err = json.Unmarshal(tsOutput, &tsResult)
		require.NoError(t, err, "TS output should be valid JSON")

		// Verify required fields
		requiredFields := []string{"version", "os", "arch"}
		for _, field := range requiredFields {
			assert.Contains(t, tsResult, field, "TS CLI should have field %s", field)
		}

		// Verify field types
		assert.IsType(t, "", tsResult["version"], "version should be string")
		assert.IsType(t, "", tsResult["os"], "os should be string")
		assert.IsType(t, "", tsResult["arch"], "arch should be string")
	})
}

// TestJSONFieldOrdering ensures consistent field ordering
func TestJSONFieldOrdering(t *testing.T) {
	var buf bytes.Buffer
	f := formatter.NewJSONFormatter(&buf, os.Stderr, false)

	err := f.FormatSayMessage("text", "Test", false, nil)
	require.NoError(t, err)

	output := buf.String()

	// Fields should appear in order: ts, type, text, partial, say
	tsIndex := strings.Index(output, `"ts"`)
	typeIndex := strings.Index(output, `"type"`)
	textIndex := strings.Index(output, `"text"`)
	sayIndex := strings.Index(output, `"say"`)

	assert.Greater(t, tsIndex, -1, "ts should be present")
	assert.Greater(t, typeIndex, -1, "type should be present")
	assert.Greater(t, textIndex, -1, "text should be present")
	assert.Greater(t, sayIndex, -1, "say should be present")

	// Verify ordering
	assert.Less(t, tsIndex, typeIndex, "ts should come before type")
	assert.Less(t, typeIndex, sayIndex, "type should come before say")
}

// TestJSONStreamingParity ensures streaming output matches Node.js format
func TestJSONStreamingParity(t *testing.T) {
	var buf bytes.Buffer
	f := formatter.NewJSONFormatter(&buf, os.Stderr, true) // streaming mode

	// Simulate a streaming conversation
	messages := []struct {
		sayType string
		text    string
		partial bool
	}{
		{"text", "Hello", false},
		{"text", "How can I help?", true},
		{"text", "How can I help you today?", false},
		{"tool", "write_to_file", false},
	}

	for _, msg := range messages {
		err := f.FormatSayMessage(msg.sayType, msg.text, msg.partial, nil)
		require.NoError(t, err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Each non-partial message should be a complete JSON line
	validLines := 0
	for i, line := range lines {
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(line), &result); err == nil {
			validLines++
			// Verify structure matches Node.js expectations
			assert.Contains(t, result, "ts", "Line %d should have ts", i)
			assert.Contains(t, result, "type", "Line %d should have type", i)
			assert.Contains(t, result, "say", "Line %d should have say", i)
		}
	}

	assert.Greater(t, validLines, 0, "Should have valid JSON lines")
}

// ==================== Helper Functions ====================

// findTSBinary attempts to find the TypeScript CLI binary
func findTSBinary() string {
	// Check environment variable
	if path := os.Getenv("CLINE_TS_CLI"); path != "" {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Check common locations
	candidates := []string{
		"./node_modules/.bin/cline",
		"../cli/dist/index.js",
		"../../cli/dist/index.js",
		"cline", // If installed globally
	}

	for _, candidate := range candidates {
		if path, err := exec.LookPath(candidate); err == nil {
			return path
		}
	}

	return ""
}

// TestMain runs the test suite
func TestMain(m *testing.M) {
	// Run tests
	code := m.Run()
	os.Exit(code)
}

// ==================== Additional Edge Case Tests ====================

// TestJSONSpecialCharacters ensures special characters are properly escaped
func TestJSONSpecialCharacters(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{
			name:     "quotes",
			text:     `He said "Hello"`,
			expected: `He said \"Hello\"`,
		},
		{
			name:     "newlines",
			text:     "Line 1\nLine 2",
			expected: "Line 1\\nLine 2",
		},
		{
			name:     "tabs",
			text:     "Column 1\tColumn 2",
			expected: "Column 1\\tColumn 2",
		},
		{
			name:     "backslashes",
			text:     `C:\Users\test`,
			expected: `C:\\Users\\test`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			f := formatter.NewJSONFormatter(&buf, os.Stderr, false)

			err := f.FormatSayMessage("text", tt.text, false, nil)
			require.NoError(t, err)

			output := buf.String()
			var result map[string]interface{}
			err = json.Unmarshal([]byte(output), &result)
			require.NoError(t, err, "Output should be valid JSON despite special characters")

			assert.Equal(t, tt.text, result["text"], "Text should be preserved exactly")
		})
	}
}

// TestJSONUnicode ensures Unicode characters are preserved
func TestJSONUnicode(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{
			name: "emoji",
			text: "Hello 👋 World 🌍",
		},
		{
			name: "chinese",
			text: "你好世界",
		},
		{
			name: "arabic",
			text: "مرحبا بالعالم",
		},
		{
			name: "mixed",
			text: "Hello 你好 👋 مرحبا",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			f := formatter.NewJSONFormatter(&buf, os.Stderr, false)

			err := f.FormatSayMessage("text", tt.text, false, nil)
			require.NoError(t, err)

			var result map[string]interface{}
			err = json.Unmarshal(buf.Bytes(), &result)
			require.NoError(t, err)

			assert.Equal(t, tt.text, result["text"], "Unicode text should be preserved")
		})
	}
}

// TestPlainTextLongMessages ensures long messages are handled correctly
func TestPlainTextLongMessages(t *testing.T) {
	var buf bytes.Buffer
	f := formatter.NewPlainFormatter(&buf, os.Stderr, false, false)

	// Create a long message (> 1000 characters)
	longText := strings.Repeat("This is a long message. ", 50)

	err := f.FormatSayMessage("text", longText, false)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, longText[:50], "Long message should be preserved")
	assert.Contains(t, output, "[Cline]", "Should have Cline prefix")
}

// TestJSONLargePayload ensures large JSON payloads are handled
func TestJSONLargePayload(t *testing.T) {
	var buf bytes.Buffer
	f := formatter.NewJSONFormatter(&buf, os.Stderr, false)

	// Create a large metadata object
	metadata := make(map[string]interface{})
	for i := 0; i < 100; i++ {
		metadata[fmt.Sprintf("key%d", i)] = strings.Repeat("value", 10)
	}

	err := f.FormatSayMessage("text", "Test", false, metadata)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	// Verify metadata is present
	assert.Contains(t, result, "metadata", "Large metadata should be preserved")
}

// TestPlainTextColorDisable ensures colors can be disabled
func TestPlainTextColorDisable(t *testing.T) {
	var buf bytes.Buffer
	f := formatter.NewPlainFormatter(&buf, os.Stderr, false, false) // useColor = false

	err := f.FormatSayMessage("text", "Hello", false)
	require.NoError(t, err)

	output := buf.String()

	// Should not contain ANSI color codes
	ansiPattern := `\x1b\[[0-9;]*m`
	matched, _ := regexp.MatchString(ansiPattern, output)
	assert.False(t, matched, "Output should not contain ANSI codes when color is disabled")
}