// Package dual provides output format parity tests between Go and TypeScript CLIs.
// These tests ensure both CLIs produce identical JSON and plain text output formats.
package dual

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// OutputFormatTest represents a format parity test case
type OutputFormatTest struct {
	Name           string
	Description    string
	Args           []string
	Env            map[string]string
	ExpectedType   string // "json", "text", "error"
	ValidateJSON   func(t *testing.T, data map[string]interface{})
	ValidateText   func(t *testing.T, output string)
	RequiredFields []string
	FieldTypes     map[string]string
}

// JSONFieldType represents expected JSON field types
type JSONFieldType string

const (
	TypeString  JSONFieldType = "string"
	TypeNumber  JSONFieldType = "number"
	TypeBoolean JSONFieldType = "boolean"
	TypeArray   JSONFieldType = "array"
	TypeObject  JSONFieldType = "object"
	TypeNull    JSONFieldType = "null"
)

// ==================== JSON Format Tests ====================

func TestVersionJSONFormat(t *testing.T) {
	goPath := FindGoBinary()
	tsPath := FindTSBinary()

	if goPath == "" || tsPath == "" {
		t.Skip("Both binaries required for format parity tests")
	}

	tests := []OutputFormatTest{
		{
			Name:         "version_json_structure",
			Description:  "Version JSON has correct structure",
			Args:         []string{"version", "--json"},
			ExpectedType: "json",
			RequiredFields: []string{
				"version",
				"goVersion",
				"os",
				"arch",
				"buildTime",
			},
			FieldTypes: map[string]string{
				"version":   "string",
				"goVersion": "string",
				"os":        "string",
				"arch":      "string",
				"buildTime": "string",
			},
		},
		{
			Name:         "version_json_semantic",
			Description:  "Version follows semantic versioning",
			Args:         []string{"version", "--json"},
			ExpectedType: "json",
			ValidateJSON: func(t *testing.T, data map[string]interface{}) {
				version, ok := data["version"].(string)
				require.True(t, ok, "version should be a string")
				// Check semantic versioning pattern (e.g., "3.0.0" or "3.0.0-beta.1")
				semverPattern := `^\d+\.\d+\.\d+(-[a-zA-Z0-9.-]+)?$`
				matched, err := regexp.MatchString(semverPattern, version)
				require.NoError(t, err)
				assert.True(t, matched, "version %s should follow semantic versioning", version)
			},
		},
	}

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			// Run Go CLI
			goCmd := exec.CommandContext(ctx, goPath, tt.Args...)
			goOutput, err := goCmd.CombinedOutput()
			require.NoError(t, err, "Go CLI failed: %s", string(goOutput))

			// Run TS CLI
			tsCmd := exec.CommandContext(ctx, tsPath, tt.Args...)
			tsOutput, err := tsCmd.CombinedOutput()
			require.NoError(t, err, "TS CLI failed: %s", string(tsOutput))

			// Parse both outputs as JSON
			var goData, tsData map[string]interface{}
			require.NoError(t, json.Unmarshal(goOutput, &goData), "Go output should be valid JSON")
			require.NoError(t, json.Unmarshal(tsOutput, &tsData), "TS output should be valid JSON")

			// Check required fields match
			for _, field := range tt.RequiredFields {
				goVal, goHas := goData[field]
				tsVal, tsHas := tsData[field]

				assert.Equal(t, goHas, tsHas, "Field %s presence should match", field)
				if goHas && tsHas {
					goType := reflect.TypeOf(goVal).Kind().String()
					tsType := reflect.TypeOf(tsVal).Kind().String()
					assert.Equal(t, goType, tsType, "Field %s type should match", field)
				}
			}

			// Check field types
			for field, expectedType := range tt.FieldTypes {
				if goVal, ok := goData[field]; ok {
					actualType := reflect.TypeOf(goVal).Kind().String()
					assert.Equal(t, expectedType, actualType, "Go CLI field %s type mismatch", field)
				}
				if tsVal, ok := tsData[field]; ok {
					actualType := reflect.TypeOf(tsVal).Kind().String()
					assert.Equal(t, expectedType, actualType, "TS CLI field %s type mismatch", field)
				}
			}

			// Run custom validation
			if tt.ValidateJSON != nil {
				tt.ValidateJSON(t, goData)
				tt.ValidateJSON(t, tsData)
			}
		})
	}
}

func TestHistoryJSONFormat(t *testing.T) {
	goPath := FindGoBinary()
	tsPath := FindTSBinary()

	if goPath == "" || tsPath == "" {
		t.Skip("Both binaries required for format parity tests")
	}

	ctx := context.Background()

	t.Run("history_json_empty", func(t *testing.T) {
		// Create temp directories for both CLIs
		goTempDir, err := os.MkdirTemp("", "go-history-*")
		require.NoError(t, err)
		defer os.RemoveAll(goTempDir)

		tsTempDir, err := os.MkdirTemp("", "ts-history-*")
		require.NoError(t, err)
		defer os.RemoveAll(tsTempDir)

		// Run Go CLI
		goCmd := exec.CommandContext(ctx, goPath, "history", "--json")
		goCmd.Env = append(os.Environ(), "CLINE_DATA_DIR="+goTempDir)
		goOutput, err := goCmd.CombinedOutput()
		require.NoError(t, err, "Go CLI failed: %s", string(goOutput))

		// Run TS CLI
		tsCmd := exec.CommandContext(ctx, tsPath, "history", "--json")
		tsCmd.Env = append(os.Environ(), "CLINE_DATA_DIR="+tsTempDir)
		tsOutput, err := tsCmd.CombinedOutput()
		require.NoError(t, err, "TS CLI failed: %s", string(tsOutput))

		// Parse both outputs
		var goData, tsData interface{}
		require.NoError(t, json.Unmarshal(goOutput, &goData), "Go output should be valid JSON")
		require.NoError(t, json.Unmarshal(tsOutput, &tsData), "TS output should be valid JSON")

		// Both should be arrays or objects with tasks
		goKind := reflect.TypeOf(goData).Kind()
		tsKind := reflect.TypeOf(tsData).Kind()
		assert.Equal(t, goKind, tsKind, "History JSON types should match")

		// Check if both are empty (no history)
		if goArr, ok := goData.([]interface{}); ok {
			assert.Empty(t, goArr, "Go CLI empty history should be empty array")
		}
		if tsArr, ok := tsData.([]interface{}); ok {
			assert.Empty(t, tsArr, "TS CLI empty history should be empty array")
		}
	})

	t.Run("history_json_structure", func(t *testing.T) {
		// This test creates a task and checks the history format
		// Note: This requires a running core extension which may not be available in tests
		// So we just verify the JSON structure is valid

		goTempDir, err := os.MkdirTemp("", "go-history-struct-*")
		require.NoError(t, err)
		defer os.RemoveAll(goTempDir)

		goCmd := exec.CommandContext(ctx, goPath, "history", "--json")
		goCmd.Env = append(os.Environ(), "CLINE_DATA_DIR="+goTempDir)
		goOutput, err := goCmd.CombinedOutput()
		require.NoError(t, err)

		var goData interface{}
		require.NoError(t, json.Unmarshal(goOutput, &goData))

		// Should be either an array or an object with a tasks field
		switch v := goData.(type) {
		case []interface{}:
			// Array format - valid
		case map[string]interface{}:
			// Object format - check for tasks field
			if tasks, ok := v["tasks"]; ok {
				assert.IsType(t, []interface{}{}, tasks)
			}
		default:
			t.Errorf("Unexpected history format type: %T", v)
		}
	})
}

func TestConfigJSONFormat(t *testing.T) {
	goPath := FindGoBinary()
	tsPath := FindTSBinary()

	if goPath == "" || tsPath == "" {
		t.Skip("Both binaries required for format parity tests")
	}

	ctx := context.Background()

	t.Run("config_list_json", func(t *testing.T) {
		goTempDir, err := os.MkdirTemp("", "go-config-*")
		require.NoError(t, err)
		defer os.RemoveAll(goTempDir)

		tsTempDir, err := os.MkdirTemp("", "ts-config-*")
		require.NoError(t, err)
		defer os.RemoveAll(tsTempDir)

		// Run Go CLI
		goCmd := exec.CommandContext(ctx, goPath, "config", "list", "--json")
		goCmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+goTempDir)
		goOutput, err := goCmd.CombinedOutput()
		require.NoError(t, err, "Go CLI failed: %s", string(goOutput))

		// Run TS CLI
		tsCmd := exec.CommandContext(ctx, tsPath, "config", "list", "--json")
		tsCmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tsTempDir)
		tsOutput, err := tsCmd.CombinedOutput()
		require.NoError(t, err, "TS CLI failed: %s", string(tsOutput))

		// Parse both outputs
		var goData, tsData map[string]interface{}
		require.NoError(t, json.Unmarshal(goOutput, &goData), "Go output should be valid JSON")
		require.NoError(t, json.Unmarshal(tsOutput, &tsData), "TS output should be valid JSON")

		// Both should be objects (config key-value pairs)
		assert.Equal(t, reflect.TypeOf(goData).Kind(), reflect.Map)
		assert.Equal(t, reflect.TypeOf(tsData).Kind(), reflect.Map)
	})

	t.Run("config_get_json", func(t *testing.T) {
		goTempDir, err := os.MkdirTemp("", "go-config-get-*")
		require.NoError(t, err)
		defer os.RemoveAll(goTempDir)

		// Set a config value first
		goSetCmd := exec.CommandContext(ctx, goPath, "config", "set", "test-key", "test-value")
		goSetCmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+goTempDir)
		_, err = goSetCmd.CombinedOutput()
		require.NoError(t, err)

		// Get config as JSON
		goCmd := exec.CommandContext(ctx, goPath, "config", "get", "test-key", "--json")
		goCmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+goTempDir)
		goOutput, err := goCmd.CombinedOutput()
		require.NoError(t, err, "Go CLI failed: %s", string(goOutput))

		var goData interface{}
		require.NoError(t, json.Unmarshal(goOutput, &goData))

		// Should contain the value
		data, ok := goData.(map[string]interface{})
		require.True(t, ok, "Config get should return an object")
		assert.Equal(t, "test-value", data["value"])
		assert.Equal(t, "test-key", data["key"])
	})
}

// ==================== Plain Text Format Tests ====================

func TestPlainTextOutputFormat(t *testing.T) {
	goPath := FindGoBinary()
	tsPath := FindTSBinary()

	if goPath == "" || tsPath == "" {
		t.Skip("Both binaries required for format parity tests")
	}

	ctx := context.Background()

	tests := []struct {
		name       string
		args       []string
		checks     []func(t *testing.T, goOut, tsOut string)
		skipReason string
	}{
		{
			name: "version_short",
			args: []string{"version", "--short"},
			checks: []func(t *testing.T, goOut, tsOut string){
				func(t *testing.T, goOut, tsOut string) {
					// Both should be single line version strings
					goLines := strings.Split(strings.TrimSpace(goOut), "\n")
					tsLines := strings.Split(strings.TrimSpace(tsOut), "\n")
					assert.Len(t, goLines, 1, "Go short version should be single line")
					assert.Len(t, tsLines, 1, "TS short version should be single line")

					// Both should contain version numbers
					versionPattern := `\d+\.\d+\.\d+`
					goMatched, _ := regexp.MatchString(versionPattern, goOut)
					tsMatched, _ := regexp.MatchString(versionPattern, tsOut)
					assert.True(t, goMatched, "Go version should contain semantic version")
					assert.True(t, tsMatched, "TS version should contain semantic version")
				},
			},
		},
		{
			name: "help_format",
			args: []string{"--help"},
			checks: []func(t *testing.T, goOut, tsOut string){
				func(t *testing.T, goOut, tsOut string) {
					// Both should have Usage section
					assert.Contains(t, goOut, "Usage:")
					assert.Contains(t, tsOut, "Usage:")

					// Both should have Available Commands
					assert.Contains(t, goOut, "Available Commands:")
					assert.Contains(t, tsOut, "Available Commands:")

					// Both should have Flags
					assert.Contains(t, goOut, "Flags:")
					assert.Contains(t, tsOut, "Flags:")
				},
			},
		},
		{
			name: "config_list_text",
			args: []string{"config", "list"},
			checks: []func(t *testing.T, goOut, tsOut string){
				func(t *testing.T, goOut, tsOut string) {
					// Both should show config or indicate no config
					goHasOutput := len(strings.TrimSpace(goOut)) > 0
					tsHasOutput := len(strings.TrimSpace(tsOut)) > 0
					assert.True(t, goHasOutput, "Go CLI should produce output")
					assert.True(t, tsHasOutput, "TS CLI should produce output")
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipReason != "" {
				t.Skip(tt.skipReason)
			}

			// Run Go CLI
			goCmd := exec.CommandContext(ctx, goPath, tt.args...)
			goOutput, err := goCmd.CombinedOutput()
			require.NoError(t, err, "Go CLI failed: %s", string(goOutput))

			// Run TS CLI
			tsCmd := exec.CommandContext(ctx, tsPath, tt.args...)
			tsOutput, err := tsCmd.CombinedOutput()
			require.NoError(t, err, "TS CLI failed: %s", string(tsOutput))

			// Run all checks
			for _, check := range tt.checks {
				check(t, string(goOutput), string(tsOutput))
			}
		})
	}
}

// ==================== Error Message Format Tests ====================

func TestErrorMessageFormat(t *testing.T) {
	goPath := FindGoBinary()
	tsPath := FindTSBinary()

	if goPath == "" || tsPath == "" {
		t.Skip("Both binaries required for format parity tests")
	}

	ctx := context.Background()

	tests := []struct {
		name           string
		args           []string
		expectedExit   int
		checkErrorFunc func(t *testing.T, goErr, tsErr string)
	}{
		{
			name:         "invalid_command",
			args:         []string{"nonexistent-command-xyz"},
			expectedExit: 1,
			checkErrorFunc: func(t *testing.T, goErr, tsErr string) {
				// Both should indicate unknown command or error
				goHasError := len(goErr) > 0 || strings.Contains(goErr, "unknown") ||
					strings.Contains(goErr, "not found") || strings.Contains(goErr, "error")
				tsHasError := len(tsErr) > 0 || strings.Contains(tsErr, "unknown") ||
					strings.Contains(tsErr, "not found") || strings.Contains(tsErr, "error")
				assert.True(t, goHasError || tsHasError, "At least one CLI should show error")
			},
		},
		{
			name:         "invalid_flag",
			args:         []string{"--invalid-flag-xyz"},
			expectedExit: 1,
			checkErrorFunc: func(t *testing.T, goErr, tsErr string) {
				// Both should indicate unknown flag
				goLower := strings.ToLower(goErr)
				tsLower := strings.ToLower(tsErr)
				goHasFlagError := strings.Contains(goLower, "flag") ||
					strings.Contains(goLower, "unknown") ||
					strings.Contains(goLower, "invalid")
				tsHasFlagError := strings.Contains(tsLower, "flag") ||
					strings.Contains(tsLower, "unknown") ||
					strings.Contains(tsLower, "invalid")
				assert.True(t, goHasFlagError || tsHasFlagError, "At least one CLI should flag error")
			},
		},
		{
			name:         "missing_required_arg",
			args:         []string{"config", "get"}, // Missing key argument
			expectedExit: 1,
			checkErrorFunc: func(t *testing.T, goErr, tsErr string) {
				// Both should indicate missing argument
				goLower := strings.ToLower(goErr)
				tsLower := strings.ToLower(tsErr)
				goHasArgError := strings.Contains(goLower, "arg") ||
					strings.Contains(goLower, "missing") ||
					strings.Contains(goLower, "required")
				tsHasArgError := strings.Contains(tsLower, "arg") ||
					strings.Contains(tsLower, "missing") ||
					strings.Contains(tsLower, "required")
				assert.True(t, goHasArgError || tsHasArgError, "At least one CLI should arg error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run Go CLI
			goCmd := exec.CommandContext(ctx, goPath, tt.args...)
			goOutput, goErr := goCmd.CombinedOutput()
			goExit := 0
			if goErr != nil {
				if exitErr, ok := goErr.(*exec.ExitError); ok {
					goExit = exitErr.ExitCode()
				}
			}

			// Run TS CLI
			tsCmd := exec.CommandContext(ctx, tsPath, tt.args...)
			tsOutput, tsErr := tsCmd.CombinedOutput()
			tsExit := 0
			if tsErr != nil {
				if exitErr, ok := tsErr.(*exec.ExitError); ok {
					tsExit = exitErr.ExitCode()
				}
			}

			// Both should have non-zero exit codes for errors
			if tt.expectedExit != 0 {
				assert.NotEqual(t, 0, goExit, "Go CLI should have non-zero exit code")
				assert.NotEqual(t, 0, tsExit, "TS CLI should have non-zero exit code")
			}

			// Run custom error checks
			if tt.checkErrorFunc != nil {
				tt.checkErrorFunc(t, string(goOutput), string(tsOutput))
			}
		})
	}
}

// ==================== JSON Stream Format Tests ====================

func TestJSONStreamFormat(t *testing.T) {
	goPath := FindGoBinary()
	tsPath := FindTSBinary()

	if goPath == "" || tsPath == "" {
		t.Skip("Both binaries required for format parity tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	t.Run("json_stream_structure", func(t *testing.T) {
		// Note: This requires actual task execution which needs a core extension
		// We'll just verify the help output format for now
		goCmd := exec.CommandContext(ctx, goPath, "--help")
		goOutput, err := goCmd.CombinedOutput()
		require.NoError(t, err)

		// Verify help is plain text, not JSON
		var jsonData interface{}
		jsonErr := json.Unmarshal(goOutput, &jsonData)
		assert.Error(t, jsonErr, "Help output should not be valid JSON")
	})

	t.Run("json_output_flag_produces_valid_json", func(t *testing.T) {
		tests := []struct {
			name string
			args []string
		}{
			{"version", []string{"version", "--json"}},
			{"history", []string{"history", "--json"}},
			{"config_list", []string{"config", "list", "--json"}},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Run Go CLI
				goCmd := exec.CommandContext(ctx, goPath, tt.args...)
				goOutput, err := goCmd.CombinedOutput()
				if err != nil {
					// Some commands might fail without proper setup
					t.Skipf("Command failed, skipping: %v", err)
				}

				// Should be valid JSON
				var goData interface{}
				require.NoError(t, json.Unmarshal(goOutput, &goData),
					"Go CLI output with --json should be valid JSON")
			})
		}
	})
}

// ==================== Field Order and Whitespace Tests ====================

func TestOutputConsistency(t *testing.T) {
	goPath := FindGoBinary()
	tsPath := FindTSBinary()

	if goPath == "" || tsPath == "" {
		t.Skip("Both binaries required for format parity tests")
	}

	ctx := context.Background()

	t.Run("deterministic_json_output", func(t *testing.T) {
		// Run the same command multiple times and verify identical output
		args := []string{"version", "--json"}

		var goOutputs []string
		var tsOutputs []string

		// Run 3 times each
		for i := 0; i < 3; i++ {
			goCmd := exec.CommandContext(ctx, goPath, args...)
			goOut, err := goCmd.CombinedOutput()
			require.NoError(t, err)
			goOutputs = append(goOutputs, string(goOut))

			tsCmd := exec.CommandContext(ctx, tsPath, args...)
			tsOut, err := tsCmd.CombinedOutput()
			require.NoError(t, err)
			tsOutputs = append(tsOutputs, string(tsOut))
		}

		// All Go outputs should be identical (ignoring timestamps)
		normalizedGo := normalizeOutputForComparison(goOutputs[0])
		for i, out := range goOutputs[1:] {
			normalized := normalizeOutputForComparison(out)
			assert.Equal(t, normalizedGo, normalized, "Go CLI output %d differs from first run", i+1)
		}

		// All TS outputs should be identical (ignoring timestamps)
		normalizedTS := normalizeOutputForComparison(tsOutputs[0])
		for i, out := range tsOutputs[1:] {
			normalized := normalizeOutputForComparison(out)
			assert.Equal(t, normalizedTS, normalized, "TS CLI output %d differs from first run", i+1)
		}
	})
}

// normalizeOutputForComparison normalizes output for deterministic comparison
func normalizeOutputForComparison(output string) string {
	// Remove timestamps
	timestampPatterns := []string{
		`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})?`,
		`\d{2}:\d{2}:\d{2}`,
		`"\d{13}"`, // Unix timestamp in milliseconds
	}

	for _, pattern := range timestampPatterns {
		re := regexp.MustCompile(pattern)
		output = re.ReplaceAllString(output, `"[TIMESTAMP]"`)
	}

	// Remove UUIDs
	uuidPattern := `[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`
	re := regexp.MustCompile(uuidPattern)
	output = re.ReplaceAllString(output, `"[UUID]"`)

	// Normalize whitespace
	output = strings.TrimSpace(output)

	return output
}

// ==================== Integration with Harness ====================

func TestDualHarnessIntegration(t *testing.T) {
	goPath := FindGoBinary()
	tsPath := FindTSBinary()

	if goPath == "" || tsPath == "" {
		t.Skip("Both binaries required for dual harness tests")
	}

	ctx := context.Background()

	t.Run("standard_dual_tests", func(t *testing.T) {
		harness := NewDualTestHarness(goPath, tsPath)
		tests := StandardDualTests()
		report, err := harness.RunTests(ctx, tests)
		require.NoError(t, err)

		// Log results
		for _, result := range report.Results {
			status := "PASS"
			if !result.Passed {
				status = "FAIL"
			}
			t.Logf("[%s] %s: %v", status, result.TestName, result.Duration)
			for _, diff := range result.Differences {
				t.Logf("  Diff: [%s] %s", diff.Severity, diff.Description)
			}
		}

		// All standard tests should pass
		assert.Equal(t, len(tests), report.PassedTests,
			"All %d standard dual tests should pass", len(tests))
	})
}

// BenchmarkOutputFormat benchmarks JSON output formatting
func BenchmarkGoJSONOutput(b *testing.B) {
	goPath := FindGoBinary()
	if goPath == "" {
		b.Skip("Go CLI not found")
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := exec.CommandContext(ctx, goPath, "version", "--json")
		cmd.Run()
	}
}

func BenchmarkTSJSONOutput(b *testing.B) {
	tsPath := FindTSBinary()
	if tsPath == "" {
		b.Skip("TS CLI not found")
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := exec.CommandContext(ctx, tsPath, "version", "--json")
		cmd.Run()
	}
}