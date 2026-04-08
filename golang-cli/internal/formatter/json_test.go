// Package formatter provides JSON output formatting for the Cline CLI.
package formatter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// TestJSONFieldOrdering tests that JSON fields appear in the correct order
func TestJSONFieldOrdering(t *testing.T) {
	tests := []struct {
		name     string
		message  JSONMessage
		expected string
	}{
		{
			name: "basic message with ts and type first",
			message: JSONMessage{
				Ts:   1234567890,
				Type: "say",
				Text: "Hello world",
			},
			expected: `{"ts":1234567890,"type":"say","text":"Hello world"}`,
		},
		{
			name: "say message with fields in correct order",
			message: JSONMessage{
				Ts:   1234567890,
				Type: "say",
				Say:  "text",
				Text: "Hello",
			},
			expected: `{"ts":1234567890,"type":"say","text":"Hello","say":"text"}`,
		},
		{
			name: "partial message ordering",
			message: JSONMessage{
				Ts:      1234567890,
				Type:    "say",
				Text:    "Hello",
				Partial: true,
			},
			expected: `{"ts":1234567890,"type":"say","text":"Hello","partial":true}`,
		},
		{
			name: "tool use message",
			message: JSONMessage{
				Ts:       1234567890,
				Type:     "say",
				Say:      "tool_use",
				ToolName: "read_file",
				ToolInput: map[string]interface{}{
					"path": "/test.txt",
				},
			},
			expected: `{"ts":1234567890,"type":"say","say":"tool_use","toolName":"read_file","toolInput":{"path":"/test.txt"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := NewOrderedJSONOutput()
			err := output.WriteMessage(tt.message)
			if err != nil {
				t.Fatalf("WriteMessage failed: %v", err)
			}

			result := output.String()
			if result != tt.expected {
				t.Errorf("Field ordering mismatch:\nGot:      %s\nExpected: %s", result, tt.expected)
			}
		})
	}
}

// TestByteForByteComparison verifies JSON output matches expected Node.js format
func TestByteForByteComparison(t *testing.T) {
	testCases := []struct {
		name     string
		message  JSONMessage
		expected string
	}{
		{
			name: "simple text message (omits false partial)",
			message: JSONMessage{
				Ts:      1699900000000,
				Type:    "say",
				Say:     "text",
				Text:    "Hello, how can I help you?",
				Partial: false,
			},
			expected: `{"ts":1699900000000,"type":"say","text":"Hello, how can I help you?","say":"text"}`,
		},
		{
			name: "API request started",
			message: JSONMessage{
				Ts:   1699900000001,
				Type: "say",
				Say:  "api_req_started",
				APIRequestStarted: &APIRequestInfo{
					RequestID: "req-001",
					Model:     "claude-sonnet-4",
					TokensIn:  1000,
				},
			},
			expected: `{"ts":1699900000001,"type":"say","say":"api_req_started","apiRequestStarted":{"requestId":"req-001","model":"claude-sonnet-4","tokensIn":1000}}`,
		},
		{
			name: "error message",
			message: JSONMessage{
				Ts:    1699900000002,
				Type:  "error",
				Error: "Something went wrong",
				Details: map[string]interface{}{
					"code":    "E001",
					"message": "Something went wrong",
				},
			},
			expected: `{"ts":1699900000002,"type":"error","error":"Something went wrong","details":{"code":"E001","message":"Something went wrong"}}`,
		},
		{
			name: "tool result (omits false partial)",
			message: JSONMessage{
				Ts:         1699900000003,
				Type:       "say",
				Say:        "tool_result",
				ToolName:   "read_file",
				ToolResult: "File contents here",
				Partial:    false,
			},
			expected: `{"ts":1699900000003,"type":"say","say":"tool_result","toolName":"read_file","toolResult":"File contents here"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output := NewOrderedJSONOutput()
			err := output.WriteMessage(tc.message)
			if err != nil {
				t.Fatalf("Failed to write message: %v", err)
			}

			result := output.String()

			// Byte-for-byte comparison
			if result != tc.expected {
				t.Errorf("Byte mismatch:\nGot:      %s (%d bytes)\nExpected: %s (%d bytes)", 
					result, len(result), tc.expected, len(tc.expected))
				
				// Show first difference
				for i := 0; i < min(len(result), len(tc.expected)); i++ {
					if result[i] != tc.expected[i] {
						t.Errorf("First difference at position %d: got %q, expected %q", 
							i, result[i], tc.expected[i])
						break
					}
				}
			}
		})
	}
}

// TestJSONLinesFormat verifies streaming JSON lines format
func TestJSONLinesFormat(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewJSONFormatter(&buf, &buf, true)

	messages := []struct {
		text string
		typ  string
	}{
		{"Hello", "say"},
		{"How are you?", "say"},
		{"Good!", "say"},
	}

	for _, msg := range messages {
		err := formatter.FormatMessage(msg.typ, msg.text, false)
		if err != nil {
			t.Fatalf("FormatMessage failed: %v", err)
		}
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != len(messages) {
		t.Errorf("Expected %d lines, got %d", len(messages), len(lines))
	}

	// Verify each line is valid JSON
	for i, line := range lines {
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(line), &parsed); err != nil {
			t.Errorf("Line %d is not valid JSON: %v", i+1, err)
		}
	}
}

// TestErrorFormatting verifies error messages are properly formatted
func TestErrorFormatting(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewJSONFormatter(&buf, &buf, true)

	testErr := fmt.Errorf("test error message")
	err := formatter.FormatErrorOutput(testErr)
	if err != nil {
		t.Fatalf("FormatErrorOutput failed: %v", err)
	}

	output := strings.TrimSpace(buf.String())

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	// Check required fields
	if parsed["type"] != "error" {
		t.Errorf("Expected type 'error', got %v", parsed["type"])
	}
	if parsed["error"] != "test error message" {
		t.Errorf("Expected error 'test error message', got %v", parsed["error"])
	}
}

// TestJSONStructureValidation validates JSON message structure matches TypeScript
func TestJSONStructureValidation(t *testing.T) {
	tests := []struct {
		name        string
		message     JSONMessage
		shouldHave  []string
		shouldNotHave []string
	}{
		{
			name: "complete say message",
			message: JSONMessage{
				Ts:      1,
				Type:    "say",
				Say:     "text",
				Text:    "test",
				Partial: false,
			},
			shouldHave:    []string{"ts", "type", "say", "text"},
			shouldNotHave: []string{"ask", "error", "details", "partial"},
		},
		{
			name: "ask message",
			message: JSONMessage{
				Ts:   2,
				Type: "ask",
				Ask:  "tool",
				Text: "Approve this?",
			},
			shouldHave:    []string{"ts", "type", "ask", "text"},
			shouldNotHave: []string{"say", "partial"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := NewOrderedJSONOutput()
			output.WriteMessage(tt.message)
			result := output.String()

			for _, field := range tt.shouldHave {
				if !strings.Contains(result, fmt.Sprintf(`"%s":`, field)) {
					t.Errorf("Expected field %s to be present in: %s", field, result)
				}
			}

			for _, field := range tt.shouldNotHave {
				if strings.Contains(result, fmt.Sprintf(`"%s":`, field)) {
					t.Errorf("Field %s should not be present in: %s", field, result)
				}
			}
		})
	}
}

// TestStreamingPartialMessages verifies partial message handling
func TestStreamingPartialMessages(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewJSONFormatter(&buf, &buf, true)

	// Send partial message
	err := formatter.FormatMessage("say", "Hello wo", true)
	if err != nil {
		t.Fatalf("FormatMessage failed: %v", err)
	}

	// Send final message (false partial is omitted in JSON)
	err = formatter.FormatMessage("say", "Hello world", false)
	if err != nil {
		t.Fatalf("FormatMessage failed: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(lines))
	}

	// Verify first line has partial: true
	var first map[string]interface{}
	json.Unmarshal([]byte(lines[0]), &first)
	if first["partial"] != true {
		t.Errorf("First message should be partial: %v", first["partial"])
	}

	// Verify second line omits partial (false values are omitted for compactness)
	var second map[string]interface{}
	json.Unmarshal([]byte(lines[1]), &second)
	if _, hasPartial := second["partial"]; hasPartial {
		t.Errorf("Second message should omit partial field (false values are omitted), got: %v", second["partial"])
	}
}

// TestNodeJSCompatibility tests compatibility with Node.js output format
func TestNodeJSCompatibility(t *testing.T) {
	// These tests verify our output matches what the Node.js CLI produces

	compatibilityTests := []struct {
		name        string
		description string
		message     JSONMessage
		nodeOutput  string
	}{
		{
			name:        "simple text say",
			description: "Basic text message from assistant",
			message: JSONMessage{
				Ts:      1699900000000,
				Type:    "say",
				Say:     "text",
				Text:    "I'll help you with that.",
				Partial: false,
			},
			// Note: false values are omitted for compactness
			nodeOutput: `{"ts":1699900000000,"type":"say","text":"I'll help you with that.","say":"text"}`,
		},
		{
			name:        "tool use notification",
			description: "Tool use announcement",
			message: JSONMessage{
				Ts:       1699900000001,
				Type:     "say",
				Say:      "tool",
				Text:     "Using read_file tool...",
				ToolName: "read_file",
			},
			nodeOutput: `{"ts":1699900000001,"type":"say","text":"Using read_file tool...","say":"tool","toolName":"read_file"}`,
		},
		{
			name:        "API request completed",
			description: "API request finished message",
			message: JSONMessage{
				Ts:   1699900000002,
				Type: "say",
				Say:  "api_req_finished",
				APIRequestFinished: &APIRequestInfo{
					RequestID: "req-123",
					Model:     "claude-sonnet-4-20250514",
					TokensIn:  1500,
					TokensOut: 500,
				},
			},
			nodeOutput: `{"ts":1699900000002,"type":"say","say":"api_req_finished","apiRequestFinished":{"requestId":"req-123","model":"claude-sonnet-4-20250514","tokensIn":1500,"tokensOut":500}}`,
		},
	}

	for _, tc := range compatibilityTests {
		t.Run(tc.name, func(t *testing.T) {
			output := NewOrderedJSONOutput()
			err := output.WriteMessage(tc.message)
			if err != nil {
				t.Fatalf("Failed to write message: %v", err)
			}

			result := output.String()

			// Verify exact match with Node.js output
			if result != tc.nodeOutput {
				t.Errorf("Node.js compatibility failure:\nGot:      %s\nExpected: %s", result, tc.nodeOutput)
			}
		})
	}
}

// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}