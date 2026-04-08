package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cline/cline/golang-cli/internal/storage"
)

func TestGetLogPath(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get home directory")
	}

	path := getLogPath()
	expected := filepath.Join(homeDir, ".cline", "logs", "cline.log")
	if path != expected {
		t.Errorf("getLogPath() = %q, want %q", path, expected)
	}
}

func TestGetAlternativeLogPaths(t *testing.T) {
	paths := getAlternativeLogPaths()

	if len(paths) == 0 {
		t.Error("getAlternativeLogPaths() should return at least one path")
	}

	// Check that paths are valid absolute paths or contain expected components
	for _, path := range paths {
		if !strings.Contains(path, "cline") {
			t.Errorf("Path %q should contain 'cline'", path)
		}
	}
}

func TestShowLogLines(t *testing.T) {
	// Create a temporary log file
	tempDir, err := os.MkdirTemp("", "cline-log-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logFile := filepath.Join(tempDir, "test.log")
	content := "line1\nline2\nline3\nline4\nline5\n"
	if err := os.WriteFile(logFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write log file: %v", err)
	}

	file, err := os.Open(logFile)
	if err != nil {
		t.Fatalf("Failed to open log file: %v", err)
	}
	defer file.Close()

	var buf bytes.Buffer
	err = showLogLines(file, &buf, 3, false)
	if err != nil {
		t.Errorf("showLogLines returned error: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(lines))
	}

	// Check last 3 lines
	if !strings.Contains(output, "line3") {
		t.Error("Output should contain line3")
	}
	if !strings.Contains(output, "line5") {
		t.Error("Output should contain line5")
	}
}

func TestShowLogLinesAll(t *testing.T) {
	// Create a temporary log file
	tempDir, err := os.MkdirTemp("", "cline-log-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logFile := filepath.Join(tempDir, "test.log")
	content := "line1\nline2\nline3\n"
	if err := os.WriteFile(logFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write log file: %v", err)
	}

	file, err := os.Open(logFile)
	if err != nil {
		t.Fatalf("Failed to open log file: %v", err)
	}
	defer file.Close()

	var buf bytes.Buffer
	err = showLogLines(file, &buf, 10, true) // all=true
	if err != nil {
		t.Errorf("showLogLines returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "line1") {
		t.Error("Output should contain line1")
	}
	if !strings.Contains(output, "line3") {
		t.Error("Output should contain line3")
	}
}

func TestParseLogEntry(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected struct {
			level   string
			message string
		}
	}{
		{
			name: "standard format",
			line: "2024-01-15T10:30:00.000Z [INFO] test message",
			expected: struct {
				level   string
				message string
			}{
				level:   "INFO",
				message: "test message",
			},
		},
		{
			name: "no level",
			line: "2024-01-15T10:30:00.000Z plain message",
			expected: struct {
				level   string
				message string
			}{
				level:   "",
				message: "plain message",
			},
		},
		{
			name: "simple message",
			line: "simple message",
			expected: struct {
				level   string
				message string
			}{
				level:   "",
				message: "message", // Parser extracts last word as message
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := ParseLogEntry(tt.line)
			if err != nil {
				t.Errorf("ParseLogEntry returned error: %v", err)
			}

			if entry.Level != tt.expected.level {
				t.Errorf("Expected level %q, got %q", tt.expected.level, entry.Level)
			}
			if entry.Message != tt.expected.message {
				t.Errorf("Expected message %q, got %q", tt.expected.message, entry.Message)
			}
		})
	}
}

func TestGetStatusIcon(t *testing.T) {
	tests := []struct {
		status   string
		expected string
	}{
		{"pass", "✓"},
		{"fail", "✗"},
		{"warn", "⚠"},
		{"skip", "⊘"},
		{"unknown", "?"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := getStatusIcon(tt.status)
			if result != tt.expected {
				t.Errorf("getStatusIcon(%q) = %q, want %q", tt.status, result, tt.expected)
			}
		})
	}
}

func TestOutputDoctorJSON(t *testing.T) {
	report := &DoctorReport{
		OS:           "linux",
		Arch:         "amd64",
		GoVersion:    "go1.21.0",
		Version:      "1.0.0",
		PassedCount:  5,
		FailedCount:  1,
		WarningCount: 2,
		Results: []DiagnosticResult{
			{Name: "Test", Status: "pass", Message: "All good"},
		},
	}

	var buf bytes.Buffer
	err := outputDoctorJSON(&buf, report)
	if err != nil {
		t.Errorf("outputDoctorJSON returned error: %v", err)
	}

	// Verify JSON output
	var decoded DoctorReport
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Errorf("Failed to parse JSON: %v", err)
	}

	if decoded.OS != report.OS {
		t.Errorf("Expected OS %s, got %s", report.OS, decoded.OS)
	}
	if decoded.PassedCount != report.PassedCount {
		t.Errorf("Expected passed count %d, got %d", report.PassedCount, decoded.PassedCount)
	}
}

func TestOutputDoctorHuman(t *testing.T) {
	report := &DoctorReport{
		OS:           "linux",
		Arch:         "amd64",
		GoVersion:    "go1.21.0",
		Version:      "1.0.0",
		PassedCount:  5,
		FailedCount:  1,
		WarningCount: 2,
		Results: []DiagnosticResult{
			{Name: "Test", Status: "pass", Message: "All good", Details: "Details here"},
		},
	}

	var buf bytes.Buffer
	err := outputDoctorHuman(&buf, report)
	if err != nil {
		t.Errorf("outputDoctorHuman returned error: %v", err)
	}

	output := buf.String()

	// Check sections
	if !strings.Contains(output, "Cline CLI Diagnostics") {
		t.Error("Output should contain 'Cline CLI Diagnostics'")
	}
	if !strings.Contains(output, "System Information") {
		t.Error("Output should contain 'System Information'")
	}
	if !strings.Contains(output, "Checks:") {
		t.Error("Output should contain 'Checks:'")
	}
	if !strings.Contains(output, "Summary:") {
		t.Error("Output should contain 'Summary:'")
	}
	if !strings.Contains(output, "Passed:") {
		t.Error("Output should contain 'Passed:'")
	}
}

func TestDiagnosticChecks(t *testing.T) {
	// Create temp storage for tests
	tempDir, err := os.MkdirTemp("", "cline-doctor-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create storage context pointing to temp dir
	ctx, err := storage.NewStorageContext(tempDir, "test-workspace")
	if err != nil {
		t.Fatalf("Failed to create storage context: %v", err)
	}
	defer ctx.Close()

	// Test checkStorage
	t.Run("checkStorage", func(t *testing.T) {
		result := checkStorage(ctx, false)
		if result.Status != "pass" {
			t.Errorf("checkStorage should pass, got: %s - %s", result.Status, result.Message)
		}
	})

	// Test checkConfig (no provider configured)
	t.Run("checkConfig_no_provider", func(t *testing.T) {
		result := checkConfig(ctx, false)
		if result.Status != "warn" {
			t.Errorf("checkConfig should warn when no provider, got: %s", result.Status)
		}
	})

	// Set up a provider and test again
	ctx.GlobalState.Set("apiProvider", "anthropic")
	t.Run("checkConfig_with_provider", func(t *testing.T) {
		result := checkConfig(ctx, false)
		if result.Status != "pass" {
			t.Errorf("checkConfig should pass with provider, got: %s - %s", result.Status, result.Message)
		}
	})

	// Test checkSecrets
	t.Run("checkSecrets", func(t *testing.T) {
		result := checkSecrets(ctx, false)
		if result.Status != "pass" {
			t.Errorf("checkSecrets should pass, got: %s", result.Status)
		}
	})

	// Test checkAPIKey (no key configured)
	t.Run("checkAPIKey_no_key", func(t *testing.T) {
		result := checkAPIKey(ctx, false)
		if result.Status != "fail" {
			t.Errorf("checkAPIKey should fail when no key, got: %s", result.Status)
		}
	})

	// Set up API key and test again
	ctx.Secrets.Set("anthropicApiKey", "test-key")
	t.Run("checkAPIKey_with_key", func(t *testing.T) {
		result := checkAPIKey(ctx, false)
		if result.Status != "pass" {
			t.Errorf("checkAPIKey should pass with key, got: %s - %s", result.Status, result.Message)
		}
	})

	// Test checkEditor
	t.Run("checkEditor", func(t *testing.T) {
		result := checkEditor(ctx, false)
		// Should pass or warn depending on environment
		if result.Status != "pass" && result.Status != "warn" {
			t.Errorf("checkEditor should pass or warn, got: %s", result.Status)
		}
	})

	// Test checkNetwork
	t.Run("checkNetwork", func(t *testing.T) {
		result := checkNetwork(ctx, false)
		if result.Status != "pass" {
			t.Errorf("checkNetwork should pass, got: %s", result.Status)
		}
	})

	// Test checkDiskSpace
	t.Run("checkDiskSpace", func(t *testing.T) {
		result := checkDiskSpace(ctx, false)
		if result.Status != "pass" {
			t.Errorf("checkDiskSpace should pass, got: %s - %s", result.Status, result.Message)
		}
	})

	// Test checkPermissions
	t.Run("checkPermissions", func(t *testing.T) {
		result := checkPermissions(ctx, false)
		if result.Status != "pass" {
			t.Errorf("checkPermissions should pass, got: %s", result.Status)
		}
	})
}

func TestDoctorReportStructure(t *testing.T) {
	report := &DoctorReport{
		OS:           "linux",
		Arch:         "amd64",
		GoVersion:    "go1.21.0",
		Version:      "1.0.0",
		PassedCount:  3,
		FailedCount:  1,
		WarningCount: 1,
		Results: []DiagnosticResult{
			{
				Name:    "Test Check",
				Status:  "pass",
				Message: "Everything is fine",
				Details: "Additional details",
			},
		},
	}

	// Verify JSON marshaling
	data, err := json.Marshal(report)
	if err != nil {
		t.Errorf("Failed to marshal report: %v", err)
	}

	var decoded DoctorReport
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Errorf("Failed to unmarshal report: %v", err)
	}

	if decoded.OS != report.OS {
		t.Errorf("Expected OS %s, got %s", report.OS, decoded.OS)
	}
	if len(decoded.Results) != len(report.Results) {
		t.Errorf("Expected %d results, got %d", len(report.Results), len(decoded.Results))
	}
}

func TestLogEntryStructure(t *testing.T) {
	entry := &LogEntry{
		Level:   "INFO",
		Message: "Test message",
		Source:  "test",
	}

	// Verify JSON marshaling
	data, err := json.Marshal(entry)
	if err != nil {
		t.Errorf("Failed to marshal entry: %v", err)
	}

	var decoded LogEntry
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Errorf("Failed to unmarshal entry: %v", err)
	}

	if decoded.Level != entry.Level {
		t.Errorf("Expected level %s, got %s", entry.Level, decoded.Level)
	}
	if decoded.Message != entry.Message {
		t.Errorf("Expected message %s, got %s", entry.Message, decoded.Message)
	}
}

func TestDiagnosticResultStatus(t *testing.T) {
	statuses := []string{"pass", "fail", "warn", "skip", "unknown"}

	for _, status := range statuses {
		result := DiagnosticResult{
			Name:    "Test",
			Status:  status,
			Message: "Test message",
		}

		// Verify the result can be marshaled
		data, err := json.Marshal(result)
		if err != nil {
			t.Errorf("Failed to marshal result with status %s: %v", status, err)
		}

		var decoded DiagnosticResult
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Errorf("Failed to unmarshal result with status %s: %v", status, err)
		}

		if decoded.Status != status {
			t.Errorf("Expected status %s, got %s", status, decoded.Status)
		}
	}
}
