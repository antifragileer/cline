// Package e2e provides end-to-end tests for the Go CLI.
// This package tests complete user workflows.
package e2e

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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// E2ETest represents an end-to-end test
type E2ETest struct {
	Name        string
	Setup       func() error
	Steps       []E2EStep
	Teardown    func() error
	Timeout     time.Duration
}

// E2EStep represents a single step in an E2E test
type E2EStep struct {
	Name        string
	Action      func() (string, int, error)
	ExpectedExit int
	Validate    func(output string) error
}

// E2EResult represents the result of an E2E test
type E2EResult struct {
	TestName    string
	Passed      bool
	Steps       []E2EStepResult
	Duration    time.Duration
	Error       string
}

// E2EStepResult represents the result of a step
type E2EStepResult struct {
	StepName    string
	Passed      bool
	Output      string
	ExitCode    int
	Duration    time.Duration
	Error       string
}

// E2ERunner runs E2E tests
type E2ERunner struct {
	BinaryPath  string
	WorkingDir  string
	Results     []E2EResult
}

// NewE2ERunner creates a new E2E runner
func NewE2ERunner(binaryPath, workingDir string) *E2ERunner {
	return &E2ERunner{
		BinaryPath: binaryPath,
		WorkingDir: workingDir,
		Results:    make([]E2EResult, 0),
	}
}

// RunTest runs a single E2E test
func (r *E2ERunner) RunTest(test E2ETest) E2EResult {
	result := E2EResult{
		TestName: test.Name,
		Steps:    make([]E2EStepResult, 0),
	}

	timeout := test.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	startTime := time.Now()

	// Run setup
	if test.Setup != nil {
		if err := test.Setup(); err != nil {
			result.Error = fmt.Sprintf("Setup failed: %v", err)
			return result
		}
	}

	// Run teardown after test
	defer func() {
		if test.Teardown != nil {
			test.Teardown()
		}
		result.Duration = time.Since(startTime)
		r.Results = append(r.Results, result)
	}()

	// Run steps
	allPassed := true
	for _, step := range test.Steps {
		stepStart := time.Now()
		output, exitCode, err := step.Action()
		stepDuration := time.Since(stepStart)

		stepResult := E2EStepResult{
			StepName: step.Name,
			Output:   output,
			ExitCode: exitCode,
			Duration: stepDuration,
		}

		if err != nil {
			stepResult.Error = err.Error()
			stepResult.Passed = false
			allPassed = false
		} else if exitCode != step.ExpectedExit {
			stepResult.Error = fmt.Sprintf("Expected exit code %d, got %d", step.ExpectedExit, exitCode)
			stepResult.Passed = false
			allPassed = false
		} else if step.Validate != nil {
			if err := step.Validate(output); err != nil {
				stepResult.Error = fmt.Sprintf("Validation failed: %v", err)
				stepResult.Passed = false
				allPassed = false
			} else {
				stepResult.Passed = true
			}
		} else {
			stepResult.Passed = true
		}

		result.Steps = append(result.Steps, stepResult)

		// Stop on first failure
		if !stepResult.Passed {
			allPassed = false
			break
		}
	}

	result.Passed = allPassed
	return result
}

// FindBinary finds the CLI binary
func FindBinary() string {
	binaryName := "cline"
	if runtime.GOOS == "windows" {
		binaryName = "cline.exe"
	}

	locations := []string{
		filepath.Join("..", "..", binaryName),
		filepath.Join("..", "..", "cmd", "cline", binaryName),
		filepath.Join("..", binaryName),
		binaryName,
	}

	for _, loc := range locations {
		if path, err := filepath.Abs(loc); err == nil {
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
	}

	return ""
}

// ==================== E2E Test Cases ====================

func TestE2EBasicWorkflow(t *testing.T) {
	binary := FindBinary()
	if binary == "" {
		t.Skip("CLI binary not found")
	}

	runner := NewE2ERunner(binary, "")

	test := E2ETest{
		Name: "Basic CLI Workflow",
		Steps: []E2EStep{
			{
				Name:         "Check version",
				ExpectedExit: 0,
				Action: func() (string, int, error) {
					cmd := exec.Command(binary, "version", "--short")
					output, err := cmd.CombinedOutput()
					exitCode := 0
					if err != nil {
						if exitErr, ok := err.(*exec.ExitError); ok {
							exitCode = exitErr.ExitCode()
						} else {
							exitCode = -1
						}
					}
					return string(output), exitCode, nil
				},
				Validate: func(output string) error {
					if strings.TrimSpace(output) == "" {
						return fmt.Errorf("version output is empty")
					}
					return nil
				},
			},
			{
				Name:         "Show help",
				ExpectedExit: 0,
				Action: func() (string, int, error) {
					cmd := exec.Command(binary, "--help")
					output, err := cmd.CombinedOutput()
					exitCode := 0
					if err != nil {
						if exitErr, ok := err.(*exec.ExitError); ok {
							exitCode = exitErr.ExitCode()
						} else {
							exitCode = -1
						}
					}
					return string(output), exitCode, nil
				},
				Validate: func(output string) error {
					if !strings.Contains(output, "Usage:") {
						return fmt.Errorf("help output missing Usage section")
					}
					return nil
				},
			},
			{
				Name:         "List config",
				ExpectedExit: 0,
				Action: func() (string, int, error) {
					cmd := exec.Command(binary, "config", "list")
					output, err := cmd.CombinedOutput()
					exitCode := 0
					if err != nil {
						if exitErr, ok := err.(*exec.ExitError); ok {
							exitCode = exitErr.ExitCode()
						} else {
							exitCode = -1
						}
					}
					return string(output), exitCode, nil
				},
			},
		},
	}

	result := runner.RunTest(test)
	require.True(t, result.Passed, "E2E test failed: %s", result.Error)

	for _, step := range result.Steps {
		t.Logf("Step '%s': %v (%.2fs)", step.StepName, step.Passed, step.Duration.Seconds())
		if step.Error != "" {
			t.Logf("  Error: %s", step.Error)
		}
	}
}

func TestE2EConfigWorkflow(t *testing.T) {
	binary := FindBinary()
	if binary == "" {
		t.Skip("CLI binary not found")
	}

	tempDir, err := os.MkdirTemp("", "e2e-config-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	runner := NewE2ERunner(binary, tempDir)

	test := E2ETest{
		Name: "Config Management Workflow",
		Setup: func() error {
			// Set config directory to temp dir
			os.Setenv("CLINE_CONFIG_DIR", tempDir)
			return nil
		},
		Teardown: func() error {
			os.Unsetenv("CLINE_CONFIG_DIR")
			return nil
		},
		Steps: []E2EStep{
			{
				Name:         "Initial config list",
				ExpectedExit: 0,
				Action: func() (string, int, error) {
					cmd := exec.Command(binary, "config", "list")
					cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)
					output, err := cmd.CombinedOutput()
					exitCode := 0
					if err != nil {
						if exitErr, ok := err.(*exec.ExitError); ok {
							exitCode = exitErr.ExitCode()
						}
					}
					return string(output), exitCode, nil
				},
			},
			{
				Name:         "Show config help",
				ExpectedExit: 0,
				Action: func() (string, int, error) {
					cmd := exec.Command(binary, "config", "--help")
					output, err := cmd.CombinedOutput()
					exitCode := 0
					if err != nil {
						if exitErr, ok := err.(*exec.ExitError); ok {
							exitCode = exitErr.ExitCode()
						}
					}
					return string(output), exitCode, nil
				},
				Validate: func(output string) error {
					required := []string{"Available Commands", "Flags", "config"}
					for _, r := range required {
						if !strings.Contains(output, r) {
							return fmt.Errorf("missing required content: %s", r)
						}
					}
					return nil
				},
			},
		},
	}

	result := runner.RunTest(test)
	require.True(t, result.Passed, "E2E config workflow failed")

	for _, step := range result.Steps {
		assert.True(t, step.Passed, "Step '%s' failed: %s", step.StepName, step.Error)
	}
}

func TestE2EHistoryWorkflow(t *testing.T) {
	binary := FindBinary()
	if binary == "" {
		t.Skip("CLI binary not found")
	}

	tempDir, err := os.MkdirTemp("", "e2e-history-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	runner := NewE2ERunner(binary, tempDir)

	test := E2ETest{
		Name: "History Management Workflow",
		Setup: func() error {
			os.Setenv("CLINE_DATA_DIR", tempDir)
			return nil
		},
		Teardown: func() error {
			os.Unsetenv("CLINE_DATA_DIR")
			return nil
		},
		Steps: []E2EStep{
			{
				Name:         "List empty history",
				ExpectedExit: 0,
				Action: func() (string, int, error) {
					cmd := exec.Command(binary, "history")
					cmd.Env = append(os.Environ(), "CLINE_DATA_DIR="+tempDir)
					output, err := cmd.CombinedOutput()
					exitCode := 0
					if err != nil {
						if exitErr, ok := err.(*exec.ExitError); ok {
							exitCode = exitErr.ExitCode()
						}
					}
					return string(output), exitCode, nil
				},
			},
			{
				Name:         "Show history in JSON format",
				ExpectedExit: 0,
				Action: func() (string, int, error) {
					cmd := exec.Command(binary, "history", "--json")
					cmd.Env = append(os.Environ(), "CLINE_DATA_DIR="+tempDir)
					output, err := cmd.CombinedOutput()
					exitCode := 0
					if err != nil {
						if exitErr, ok := err.(*exec.ExitError); ok {
							exitCode = exitErr.ExitCode()
						}
					}
					return string(output), exitCode, nil
				},
				Validate: func(output string) error {
					// Verify valid JSON
					var data interface{}
					if err := json.Unmarshal([]byte(output), &data); err != nil {
						return fmt.Errorf("invalid JSON output: %v", err)
					}
					return nil
				},
			},
		},
	}

	result := runner.RunTest(test)
	require.True(t, result.Passed, "E2E history workflow failed")
}

func TestE2EErrorHandling(t *testing.T) {
	binary := FindBinary()
	if binary == "" {
		t.Skip("CLI binary not found")
	}

	runner := NewE2ERunner(binary, "")

	test := E2ETest{
		Name: "Error Handling",
		Steps: []E2EStep{
			{
				Name:         "Invalid command",
				ExpectedExit: 1,
				Action: func() (string, int, error) {
					cmd := exec.Command(binary, "invalid-command-that-does-not-exist")
					output, err := cmd.CombinedOutput()
					exitCode := 0
					if err != nil {
						if exitErr, ok := err.(*exec.ExitError); ok {
							exitCode = exitErr.ExitCode()
						} else {
							exitCode = -1
						}
					}
					return string(output), exitCode, nil
				},
				Validate: func(output string) error {
					// Should show error message
					if !strings.Contains(strings.ToLower(output), "unknown") &&
						!strings.Contains(strings.ToLower(output), "error") &&
						!strings.Contains(strings.ToLower(output), "not found") {
						return fmt.Errorf("expected error message in output")
					}
					return nil
				},
			},
			{
				Name:         "Invalid flag",
				ExpectedExit: 1,
				Action: func() (string, int, error) {
					cmd := exec.Command(binary, "--invalid-flag")
					output, err := cmd.CombinedOutput()
					exitCode := 0
					if err != nil {
						if exitErr, ok := err.(*exec.ExitError); ok {
							exitCode = exitErr.ExitCode()
						} else {
							exitCode = -1
						}
					}
					return string(output), exitCode, nil
				},
			},
		},
	}

	result := runner.RunTest(test)
	require.True(t, result.Passed, "E2E error handling test failed")
}

// TestE2EConcurrentUsage tests concurrent CLI usage
func TestE2EConcurrentUsage(t *testing.T) {
	binary := FindBinary()
	if binary == "" {
		t.Skip("CLI binary not found")
	}

	numConcurrent := 10
	results := make(chan bool, numConcurrent)

	for i := 0; i < numConcurrent; i++ {
		go func(n int) {
			cmd := exec.Command(binary, "version", "--short")
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Logf("Concurrent call %d failed: %v", n, err)
				results <- false
				return
			}
			if strings.TrimSpace(string(output)) == "" {
				results <- false
				return
			}
			results <- true
		}(i)
	}

	// Wait for all goroutines
	successCount := 0
	for i := 0; i < numConcurrent; i++ {
		if <-results {
			successCount++
		}
	}

	assert.Equal(t, numConcurrent, successCount, "All concurrent calls should succeed")
}

// BenchmarkE2EWorkflow benchmarks the complete workflow
func BenchmarkE2EWorkflow(b *testing.B) {
	binary := FindBinary()
	if binary == "" {
		b.Skip("CLI binary not found")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := exec.Command(binary, "version")
		cmd.Run()
	}
}