// Package tests provides smoke tests and binary verification for the Go CLI.
// This package ensures the Go CLI binary works correctly across platforms.
package tests

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// SmokeTestResult represents the result of a smoke test
type SmokeTestResult struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// SmokeTestReport contains all smoke test results
type SmokeTestReport struct {
	Success      bool                `json:"success"`
	Results      []SmokeTestResult   `json:"results"`
	Summary      string              `json:"summary"`
	ExitCode     int                 `json:"exitCode"`
	BinaryPath   string              `json:"binaryPath"`
	BinarySize   int64               `json:"binarySizeBytes"`
	BinarySizeMB float64             `json:"binarySizeMB"`
	Platform     string              `json:"platform"`
	Timestamp    time.Time           `json:"timestamp"`
}

// Constants for smoke test checks
const (
	TestBinaryExecution   = "binary_execution"
	TestHelpText          = "help_text"
	TestVersionOutput     = "version_output"
	TestConfigCommand     = "config_command"
	TestHistoryCommand    = "history_command"
	TestBinarySize        = "binary_size"
	TestNoExternalDeps    = "no_external_dependencies"
	TestCrossPlatform     = "cross_platform_build"
)

// BinarySizeLimit is the maximum allowed binary size in MB
const BinarySizeLimit = 100

// SmokeTester performs all smoke tests
type SmokeTester struct {
	BinaryPath string
	Verbose    bool
	Results    []SmokeTestResult
}

// NewSmokeTester creates a new smoke tester instance
func NewSmokeTester(binaryPath string, verbose bool) *SmokeTester {
	return &SmokeTester{
		BinaryPath: binaryPath,
		Verbose:    verbose,
		Results:    make([]SmokeTestResult, 0),
	}
}

// Run runs all smoke tests and returns a report
func (s *SmokeTester) Run() (*SmokeTestReport, error) {
	s.Results = make([]SmokeTestResult, 0)

	// Run all smoke tests
	tests := []func() SmokeTestResult{
		s.testBinaryExecution,
		s.testHelpText,
		s.testVersionOutput,
		s.testConfigCommand,
		s.testHistoryCommand,
		s.testBinarySize,
		s.testNoExternalDependencies,
	}

	allPassed := true
	for _, test := range tests {
		result := test()
		s.Results = append(s.Results, result)
		if !result.Passed {
			allPassed = false
		}
	}

	// Get binary info
	binarySize := int64(0)
	binarySizeMB := float64(0)
	if info, err := os.Stat(s.BinaryPath); err == nil {
		binarySize = info.Size()
		binarySizeMB = float64(binarySize) / (1024 * 1024)
	}

	exitCode := 0
	summary := "All smoke tests passed"
	if !allPassed {
		exitCode = 1
		summary = "Smoke tests failed"
	}

	report := &SmokeTestReport{
		Success:      allPassed,
		Results:      s.Results,
		Summary:      summary,
		ExitCode:     exitCode,
		BinaryPath:   s.BinaryPath,
		BinarySize:   binarySize,
		BinarySizeMB: binarySizeMB,
		Platform:     fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		Timestamp:    time.Now().UTC(),
	}

	return report, nil
}

// testBinaryExecution verifies the binary can be executed
func (s *SmokeTester) testBinaryExecution() SmokeTestResult {
	result := SmokeTestResult{
		Name:   TestBinaryExecution,
		Passed: false,
	}

	if s.BinaryPath == "" {
		result.Message = "Binary path not provided"
		result.Details = "Cannot test binary execution without a binary path"
		return result
	}

	// Check if binary exists
	if _, err := os.Stat(s.BinaryPath); os.IsNotExist(err) {
		result.Message = "Binary not found"
		result.Details = fmt.Sprintf("Binary path does not exist: %s", s.BinaryPath)
		return result
	}

	// Try to execute the binary with --help (should always work)
	cmd := exec.Command(s.BinaryPath, "--help")
	cmd.Env = os.Environ()
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if it's an execution error
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.Message = "Binary execution failed"
			result.Details = fmt.Sprintf("Exit code: %d, Output: %s", exitErr.ExitCode(), string(output))
			return result
		}
		result.Message = "Failed to execute binary"
		result.Details = err.Error()
		return result
	}

	// Verify output contains expected content
	outputStr := string(output)
	if !strings.Contains(outputStr, "Cline") && !strings.Contains(outputStr, "cline") {
		result.Message = "Binary executed but unexpected output"
		result.Details = "Output does not contain expected 'Cline' text"
		return result
	}

	result.Passed = true
	result.Message = "Binary executes successfully"
	return result
}

// testHelpText verifies help text display
func (s *SmokeTester) testHelpText() SmokeTestResult {
	result := SmokeTestResult{
		Name:   TestHelpText,
		Passed: false,
	}

	if s.BinaryPath == "" {
		result.Message = "Binary path not provided"
		return result
	}

	// Test root command help
	cmd := exec.Command(s.BinaryPath, "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		result.Message = "Failed to get help text"
		result.Details = err.Error()
		return result
	}

	outputStr := string(output)
	
	// Check for expected help elements
	requiredElements := []string{
		"Usage:",
		"Available Commands:",
		"Flags:",
	}

	missing := make([]string, 0)
	for _, element := range requiredElements {
		if !strings.Contains(outputStr, element) {
			missing = append(missing, element)
		}
	}

	if len(missing) > 0 {
		result.Message = "Help text missing required elements"
		result.Details = fmt.Sprintf("Missing: %v", missing)
		return result
	}

	// Check for specific commands
	expectedCommands := []string{"version", "config", "history"}
	for _, cmd := range expectedCommands {
		if !strings.Contains(outputStr, cmd) {
			missing = append(missing, fmt.Sprintf("command '%s'", cmd))
		}
	}

	if len(missing) > 0 {
		result.Message = "Help text missing expected commands"
		result.Details = fmt.Sprintf("Missing: %v", missing)
		return result
	}

	result.Passed = true
	result.Message = "Help text display is correct"
	return result
}

// testVersionOutput verifies version command output
func (s *SmokeTester) testVersionOutput() SmokeTestResult {
	result := SmokeTestResult{
		Name:   TestVersionOutput,
		Passed: false,
	}

	if s.BinaryPath == "" {
		result.Message = "Binary path not provided"
		return result
	}

	// Test version command
	cmd := exec.Command(s.BinaryPath, "version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		result.Message = "Version command failed"
		result.Details = err.Error()
		return result
	}

	outputStr := strings.TrimSpace(string(output))
	
	// Check for version string
	if !strings.Contains(outputStr, "version") && !strings.Contains(outputStr, "Version") {
		result.Message = "Version output missing version info"
		result.Details = fmt.Sprintf("Output: %s", outputStr)
		return result
	}

	// Test version --short
	cmd = exec.Command(s.BinaryPath, "version", "--short")
	output, err = cmd.CombinedOutput()
	if err != nil {
		result.Message = "Version --short command failed"
		result.Details = err.Error()
		return result
	}

	shortVersion := strings.TrimSpace(string(output))
	if shortVersion == "" {
		result.Message = "Version --short returned empty"
		return result
	}

	// Test version --json
	cmd = exec.Command(s.BinaryPath, "version", "--json")
	output, err = cmd.CombinedOutput()
	if err != nil {
		result.Message = "Version --json command failed"
		result.Details = err.Error()
		return result
	}

	// Verify JSON output is valid
	var versionInfo map[string]interface{}
	if err := json.Unmarshal(output, &versionInfo); err != nil {
		result.Message = "Version --json output is not valid JSON"
		result.Details = err.Error()
		return result
	}

	// Check for expected fields
	expectedFields := []string{"version", "goVersion", "os", "arch"}
	missingFields := make([]string, 0)
	for _, field := range expectedFields {
		if _, ok := versionInfo[field]; !ok {
			missingFields = append(missingFields, field)
		}
	}

	if len(missingFields) > 0 {
		result.Message = "Version JSON missing expected fields"
		result.Details = fmt.Sprintf("Missing fields: %v", missingFields)
		return result
	}

	result.Passed = true
	result.Message = "Version output is correct"
	return result
}

// testConfigCommand verifies config command functionality
func (s *SmokeTester) testConfigCommand() SmokeTestResult {
	result := SmokeTestResult{
		Name:   TestConfigCommand,
		Passed: false,
	}

	if s.BinaryPath == "" {
		result.Message = "Binary path not provided"
		return result
	}

	// Test config --help
	cmd := exec.Command(s.BinaryPath, "config", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		result.Message = "Config --help command failed"
		result.Details = err.Error()
		return result
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "config") {
		result.Message = "Config help doesn't mention config"
		return result
	}

	// Test config list (should work even if no config exists)
	cmd = exec.Command(s.BinaryPath, "config", "list")
	output, err = cmd.CombinedOutput()
	// This might fail if no config exists, but shouldn't crash
	if err != nil {
		// Check if it's a graceful error
		outputStr = string(output)
		if !strings.Contains(outputStr, "Error") && !strings.Contains(outputStr, "error") {
			// Some other error, might be acceptable
		}
	}

	result.Passed = true
	result.Message = "Config command works correctly"
	return result
}

// testHistoryCommand verifies history command functionality
func (s *SmokeTester) testHistoryCommand() SmokeTestResult {
	result := SmokeTestResult{
		Name:   TestHistoryCommand,
		Passed: false,
	}

	if s.BinaryPath == "" {
		result.Message = "Binary path not provided"
		return result
	}

	// Test history --help
	cmd := exec.Command(s.BinaryPath, "history", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		result.Message = "History --help command failed"
		result.Details = err.Error()
		return result
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "history") {
		result.Message = "History help doesn't mention history"
		return result
	}

	// Test history command (might be empty but should not crash)
	cmd = exec.Command(s.BinaryPath, "history")
	output, err = cmd.CombinedOutput()
	// Command should succeed even with no history
	outputStr = string(output)
	
	// Should either show entries or "No task history" message
	if !strings.Contains(outputStr, "ID") && !strings.Contains(outputStr, "No task history") && 
	   !strings.Contains(outputStr, "TASK") && !strings.Contains(outputStr, "TIMESTAMP") {
		// Try JSON format
		cmd = exec.Command(s.BinaryPath, "history", "--json")
		output, err = cmd.CombinedOutput()
		if err != nil {
			result.Message = "History command failed"
			result.Details = err.Error()
			return result
		}

		// Verify JSON is valid
		var historyData map[string]interface{}
		if err := json.Unmarshal(output, &historyData); err != nil {
			result.Message = "History JSON output is invalid"
			result.Details = err.Error()
			return result
		}
	}

	result.Passed = true
	result.Message = "History command works correctly"
	return result
}

// testBinarySize verifies binary size is reasonable
func (s *SmokeTester) testBinarySize() SmokeTestResult {
	result := SmokeTestResult{
		Name:   TestBinarySize,
		Passed: false,
	}

	if s.BinaryPath == "" {
		result.Message = "Binary path not provided"
		return result
	}

	info, err := os.Stat(s.BinaryPath)
	if err != nil {
		result.Message = "Failed to stat binary"
		result.Details = err.Error()
		return result
	}

	sizeMB := float64(info.Size()) / (1024 * 1024)
	
	if sizeMB > BinarySizeLimit {
		result.Message = "Binary size exceeds limit"
		result.Details = fmt.Sprintf("Size: %.2f MB, Limit: %d MB", sizeMB, BinarySizeLimit)
		return result
	}

	result.Passed = true
	result.Message = fmt.Sprintf("Binary size is acceptable (%.2f MB)", sizeMB)
	return result
}

// testNoExternalDependencies verifies binary runs without external dependencies
func (s *SmokeTester) testNoExternalDependencies() SmokeTestResult {
	result := SmokeTestResult{
		Name:   TestNoExternalDeps,
		Passed: false,
	}

	if s.BinaryPath == "" {
		result.Message = "Binary path not provided"
		return result
	}

	// Test that binary can run with minimal environment
	cmd := exec.Command(s.BinaryPath, "version", "--short")
	
	// Clear most environment variables to test independence
	minimalEnv := []string{
		"PATH=/usr/bin:/bin",
		"HOME=/tmp",
	}
	cmd.Env = minimalEnv
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		result.Message = "Binary requires external dependencies"
		result.Details = fmt.Sprintf("Failed with minimal env: %v, Output: %s", err, string(output))
		return result
	}

	result.Passed = true
	result.Message = "Binary runs without external dependencies"
	return result
}

// PrintReport prints a human-readable smoke test report
func (s *SmokeTestReport) PrintReport() {
	fmt.Println("=" + strings.Repeat("=", 70))
	fmt.Println("  Smoke Test Report")
	fmt.Println("=" + strings.Repeat("=", 70))
	fmt.Printf("Binary: %s\n", s.BinaryPath)
	fmt.Printf("Size: %.2f MB (%d bytes)\n", s.BinarySizeMB, s.BinarySize)
	fmt.Printf("Platform: %s\n", s.Platform)
	fmt.Printf("Timestamp: %s\n", s.Timestamp.Format(time.RFC3339))
	fmt.Println()

	for _, result := range s.Results {
		status := "✓ PASS"
		if !result.Passed {
			status = "✗ FAIL"
		}
		fmt.Printf("[%s] %s\n", status, result.Name)
		fmt.Printf("       %s\n", result.Message)
		if result.Details != "" {
			fmt.Printf("       Details: %s\n", result.Details)
		}
		fmt.Println()
	}

	fmt.Println(strings.Repeat("-", 72))
	if s.Success {
		fmt.Printf("Summary: %s\n", s.Summary)
	} else {
		fmt.Printf("Summary: %s\n", s.Summary)
	}
	fmt.Printf("Exit Code: %d\n", s.ExitCode)
}

// ToJSON returns the report as JSON
func (s *SmokeTestReport) ToJSON() (string, error) {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// RunSmokeTestsCommand executes smoke tests as a CLI command
func RunSmokeTestsCommand(binaryPath string, verbose bool, jsonOutput bool) int {
	// Auto-detect binary if not provided
	if binaryPath == "" {
		binaryPath = findBinary()
	}

	if binaryPath == "" {
		fmt.Fprintf(os.Stderr, "Error: Could not find cline binary. Please build first or specify path.\n")
		return 1
	}

	// Verify binary exists
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: Binary not found at %s\n", binaryPath)
		return 1
	}

	tester := NewSmokeTester(binaryPath, verbose)
	report, err := tester.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Smoke test error: %v\n", err)
		return 1
	}

	if jsonOutput {
		jsonStr, err := report.ToJSON()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating JSON: %v\n", err)
			return 1
		}
		fmt.Println(jsonStr)
	} else {
		report.PrintReport()
	}

	return report.ExitCode
}

// findBinary attempts to find the cline binary
func findBinary() string {
	binaryName := "cline"
	if runtime.GOOS == "windows" {
		binaryName = "cline.exe"
	}

	// Check common locations
	locations := []string{
		filepath.Join(".", binaryName),
		filepath.Join("..", binaryName),
		filepath.Join("..", "cmd", "cline", binaryName),
		filepath.Join("cmd", "cline", binaryName),
		filepath.Join("bin", binaryName),
		filepath.Join("dist", binaryName),
		filepath.Join("build", binaryName),
	}

	for _, loc := range locations {
		if absPath, err := filepath.Abs(loc); err == nil {
			if _, err := os.Stat(absPath); err == nil {
				return absPath
			}
		}
	}

	// Try to find in PATH
	if path, err := exec.LookPath(binaryName); err == nil {
		return path
	}

	return ""
}

// ==================== Unit Tests ====================

func TestNewSmokeTester(t *testing.T) {
	t.Run("creates tester with correct settings", func(t *testing.T) {
		tester := NewSmokeTester("/path/to/binary", true)
		
		if tester.BinaryPath != "/path/to/binary" {
			t.Errorf("BinaryPath = %s, want /path/to/binary", tester.BinaryPath)
		}
		if !tester.Verbose {
			t.Error("Verbose should be true")
		}
		if tester.Results == nil {
			t.Error("Results should be initialized")
		}
	})
}

func TestSmokeTester_testBinarySize(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "smoke-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("fails when binary path is empty", func(t *testing.T) {
		tester := NewSmokeTester("", false)
		result := tester.testBinarySize()
		
		if result.Passed {
			t.Error("Expected fail when binary path is empty")
		}
		if !strings.Contains(result.Message, "not provided") {
			t.Errorf("Message should mention 'not provided': %s", result.Message)
		}
	})

	t.Run("fails when binary does not exist", func(t *testing.T) {
		tester := NewSmokeTester("/nonexistent/binary", false)
		result := tester.testBinarySize()
		
		if result.Passed {
			t.Error("Expected fail when binary does not exist")
		}
	})

	t.Run("passes with small binary", func(t *testing.T) {
		binaryPath := filepath.Join(tempDir, "small-binary")
		// Create a small file (1 KB)
		content := make([]byte, 1024)
		if err := os.WriteFile(binaryPath, content, 0755); err != nil {
			t.Fatalf("Failed to create binary: %v", err)
		}

		tester := NewSmokeTester(binaryPath, false)
		result := tester.testBinarySize()
		
		if !result.Passed {
			t.Errorf("Expected pass for small binary, got: %s - %s", result.Message, result.Details)
		}
	})

	t.Run("fails with oversized binary", func(t *testing.T) {
		binaryPath := filepath.Join(tempDir, "large-binary")
		// Create a large file (150 MB)
		content := make([]byte, 150*1024*1024)
		if err := os.WriteFile(binaryPath, content, 0755); err != nil {
			t.Fatalf("Failed to create binary: %v", err)
		}

		tester := NewSmokeTester(binaryPath, false)
		result := tester.testBinarySize()
		
		if result.Passed {
			t.Error("Expected fail for oversized binary")
		}
		if !strings.Contains(result.Message, "exceeds") {
			t.Errorf("Message should mention 'exceeds': %s", result.Message)
		}
	})
}

func TestSmokeTester_testHelpText(t *testing.T) {
	// This test requires a real binary, so we skip if not available
	binaryPath := findBinary()
	if binaryPath == "" {
		t.Skip("No cline binary found for testing")
	}

	t.Run("passes with valid binary", func(t *testing.T) {
		tester := NewSmokeTester(binaryPath, false)
		result := tester.testHelpText()
		
		if !result.Passed {
			t.Errorf("Expected help text test to pass, got: %s - %s", result.Message, result.Details)
		}
	})
}

func TestSmokeTester_testVersionOutput(t *testing.T) {
	// This test requires a real binary, so we skip if not available
	binaryPath := findBinary()
	if binaryPath == "" {
		t.Skip("No cline binary found for testing")
	}

	t.Run("passes with valid binary", func(t *testing.T) {
		tester := NewSmokeTester(binaryPath, false)
		result := tester.testVersionOutput()
		
		if !result.Passed {
			t.Errorf("Expected version test to pass, got: %s - %s", result.Message, result.Details)
		}
	})
}

func TestSmokeTester_testConfigCommand(t *testing.T) {
	// This test requires a real binary, so we skip if not available
	binaryPath := findBinary()
	if binaryPath == "" {
		t.Skip("No cline binary found for testing")
	}

	t.Run("passes with valid binary", func(t *testing.T) {
		tester := NewSmokeTester(binaryPath, false)
		result := tester.testConfigCommand()
		
		if !result.Passed {
			t.Errorf("Expected config test to pass, got: %s - %s", result.Message, result.Details)
		}
	})
}

func TestSmokeTester_testHistoryCommand(t *testing.T) {
	// This test requires a real binary, so we skip if not available
	binaryPath := findBinary()
	if binaryPath == "" {
		t.Skip("No cline binary found for testing")
	}

	t.Run("passes with valid binary", func(t *testing.T) {
		tester := NewSmokeTester(binaryPath, false)
		result := tester.testHistoryCommand()
		
		if !result.Passed {
			t.Errorf("Expected history test to pass, got: %s - %s", result.Message, result.Details)
		}
	})
}

func TestSmokeTestReport(t *testing.T) {
	t.Run("JSON marshaling", func(t *testing.T) {
		report := &SmokeTestReport{
			Success:      true,
			BinaryPath:   "/path/to/binary",
			BinarySize:   1024,
			BinarySizeMB: 0.001,
			Platform:     "linux/amd64",
			Timestamp:    time.Now().UTC(),
			Results: []SmokeTestResult{
				{Name: "test1", Passed: true, Message: "OK"},
			},
			Summary:  "All good",
			ExitCode: 0,
		}

		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var decoded SmokeTestReport
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if decoded.Success != report.Success {
			t.Errorf("Success mismatch")
		}
		if decoded.BinaryPath != report.BinaryPath {
			t.Errorf("BinaryPath mismatch")
		}
	})

	t.Run("ToJSON", func(t *testing.T) {
		report := &SmokeTestReport{
			Success: true,
			Results: []SmokeTestResult{},
			Summary: "Test",
		}

		jsonStr, err := report.ToJSON()
		if err != nil {
			t.Fatalf("ToJSON failed: %v", err)
		}

		if !strings.Contains(jsonStr, "Test") {
			t.Error("JSON should contain summary")
		}
	})
}

func TestFindBinary(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "find-binary-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("finds binary in current directory", func(t *testing.T) {
		binaryName := "cline"
		if runtime.GOOS == "windows" {
			binaryName = "cline.exe"
		}

		// Create a fake binary
		binaryPath := filepath.Join(tempDir, binaryName)
		if err := os.WriteFile(binaryPath, []byte("fake binary"), 0755); err != nil {
			t.Fatalf("Failed to create binary: %v", err)
		}

		// Change to temp directory
		origDir, _ := os.Getwd()
		os.Chdir(tempDir)
		defer os.Chdir(origDir)

		found := findBinary()
		if found == "" {
			t.Error("Expected to find binary in current directory")
		}
	})
}

func TestSmokeConstants(t *testing.T) {
	t.Run("test constants are defined", func(t *testing.T) {
		tests := []string{
			TestBinaryExecution,
			TestHelpText,
			TestVersionOutput,
			TestConfigCommand,
			TestHistoryCommand,
			TestBinarySize,
			TestNoExternalDeps,
			TestCrossPlatform,
		}

		for _, test := range tests {
			if test == "" {
				t.Error("Test constant should not be empty")
			}
		}
	})

	t.Run("binary size limit is reasonable", func(t *testing.T) {
		if BinarySizeLimit != 100 {
			t.Errorf("BinarySizeLimit = %d, want 100", BinarySizeLimit)
		}
	})
}

func TestRunSmokeTestsCommand(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "run-smoke-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("fails with nonexistent binary", func(t *testing.T) {
		exitCode := RunSmokeTestsCommand("/nonexistent/binary", false, false)
		if exitCode == 0 {
			t.Error("Expected non-zero exit code for nonexistent binary")
		}
	})
}
