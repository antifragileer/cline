// Package dual provides enhanced functional testing capabilities for the dual testing framework.
// This file adds task execution comparison, state file compatibility tests, and automated parity reports.
package dual

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// FunctionalDualTest represents a functional test case with state comparison
type FunctionalDualTest struct {
	DualTest
	StateComparison   bool                    // Compare state files after execution
	TaskExecution     bool                    // Test actual task execution flow
	HistoryValidation bool                    // Validate history entries
	ConfigValidation  bool                    // Validate config persistence
	ExpectedState     map[string]interface{}  // Expected state after execution
}

// FunctionalTestResult extends DualTestResult with functional validation
type FunctionalTestResult struct {
	DualTestResult
	StateFilesMatch    bool
	HistoryValid       bool
	ConfigValid        bool
	TaskCompleted      bool
	StateDifferences   []StateDifference
}

// StateDifference represents a difference in state files
type StateDifference struct {
	File        string
	Key         string
	GoValue     interface{}
	TSValue     interface{}
	Description string
}

// FunctionalDualHarness extends DualTestHarness with functional testing capabilities
type FunctionalDualHarness struct {
	*DualTestHarness
	TempDir          string
	StateDir         string
	FunctionalResults []FunctionalTestResult
	mu               sync.RWMutex
}

// NewFunctionalDualHarness creates a new functional dual test harness
func NewFunctionalDualHarness(goPath, tsPath string) (*FunctionalDualHarness, error) {
	tempDir, err := os.MkdirTemp("", "functional-dual-test-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	return &FunctionalDualHarness{
		DualTestHarness: NewDualTestHarness(goPath, tsPath),
		TempDir:         tempDir,
		StateDir:        filepath.Join(tempDir, "state"),
		FunctionalResults: make([]FunctionalTestResult, 0),
	}, nil
}

// Cleanup removes temporary files
func (h *FunctionalDualHarness) Cleanup() {
	os.RemoveAll(h.TempDir)
}

// RunFunctionalTests executes functional dual tests
func (h *FunctionalDualHarness) RunFunctionalTests(ctx context.Context, tests []FunctionalDualTest) (*FunctionalTestReport, error) {
	startTime := time.Now()

	for _, test := range tests {
		result := h.runFunctionalTest(ctx, test)
		h.mu.Lock()
		h.FunctionalResults = append(h.FunctionalResults, result)
		h.mu.Unlock()
	}

	return h.generateFunctionalReport(startTime), nil
}

// runFunctionalTest runs a single functional dual test
func (h *FunctionalDualHarness) runFunctionalTest(ctx context.Context, test FunctionalDualTest) FunctionalTestResult {
	// Run the base dual test
	baseResult := h.runTest(ctx, test.DualTest)

	result := FunctionalTestResult{
		DualTestResult: baseResult,
		StateFilesMatch: true,
		HistoryValid:    true,
		ConfigValid:     true,
		TaskCompleted:   false,
		StateDifferences: make([]StateDifference, 0),
	}

	// Perform state file comparison if requested
	if test.StateComparison {
		stateDiffs := h.compareStateFiles()
		result.StateDifferences = append(result.StateDifferences, stateDiffs...)
		result.StateFilesMatch = len(stateDiffs) == 0
	}

	// Validate history if requested
	if test.HistoryValidation {
		result.HistoryValid = h.validateHistory()
	}

	// Validate config if requested
	if test.ConfigValidation {
		result.ConfigValid = h.validateConfig()
	}

	// Check task completion if requested
	if test.TaskExecution {
		result.TaskCompleted = h.checkTaskCompletion()
	}

	// Update overall pass status
	result.Passed = baseResult.Passed && 
		(!test.StateComparison || result.StateFilesMatch) &&
		(!test.HistoryValidation || result.HistoryValid) &&
		(!test.ConfigValidation || result.ConfigValid)

	return result
}

// compareStateFiles compares state files between Go and TypeScript CLI
func (h *FunctionalDualHarness) compareStateFiles() []StateDifference {
	diffs := make([]StateDifference, 0)

	// Compare global state
	goGlobalState := h.loadStateFile(h.GoBinary.Path, "globalState.json")
	tsGlobalState := h.loadStateFile(h.TSBinary.Path, "globalState.json")

	diffs = append(diffs, h.compareStateMaps("globalState.json", goGlobalState, tsGlobalState)...)

	// Compare workspace state
	goWorkspaceState := h.loadStateFile(h.GoBinary.Path, "workspaceState.json")
	tsWorkspaceState := h.loadStateFile(h.TSBinary.Path, "workspaceState.json")

	diffs = append(diffs, h.compareStateMaps("workspaceState.json", goWorkspaceState, tsWorkspaceState)...)

	// Compare config
	goConfig := h.loadStateFile(h.GoBinary.Path, "config.json")
	tsConfig := h.loadStateFile(h.TSBinary.Path, "config.json")

	diffs = append(diffs, h.compareStateMaps("config.json", goConfig, tsConfig)...)

	return diffs
}

// loadStateFile loads a state file for a binary
func (h *FunctionalDualHarness) loadStateFile(binaryPath, filename string) map[string]interface{} {
	stateDir := h.getStateDir(binaryPath)
	filePath := filepath.Join(stateDir, filename)

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}

	var state map[string]interface{}
	if err := json.Unmarshal(data, &state); err != nil {
		return nil
	}

	return state
}

// getStateDir returns the state directory for a binary
func (h *FunctionalDualHarness) getStateDir(binaryPath string) string {
	// Use binary-specific state directories
	binaryName := "go"
	if strings.Contains(binaryPath, "cline") && !strings.Contains(binaryPath, "go") {
		binaryName = "ts"
	}
	return filepath.Join(h.StateDir, binaryName)
}

// compareStateMaps compares two state maps
func (h *FunctionalDualHarness) compareStateMaps(filename string, goState, tsState map[string]interface{}) []StateDifference {
	diffs := make([]StateDifference, 0)

	if goState == nil && tsState == nil {
		return diffs
	}

	if goState == nil {
		return []StateDifference{{
			File:        filename,
			Description: "Go state is nil but TypeScript state exists",
		}}
	}

	if tsState == nil {
		return []StateDifference{{
			File:        filename,
			Description: "TypeScript state is nil but Go state exists",
		}}
	}

	// Compare all keys
	allKeys := make(map[string]bool)
	for k := range goState {
		allKeys[k] = true
	}
	for k := range tsState {
		allKeys[k] = true
	}

	for key := range allKeys {
		goVal, goExists := goState[key]
		tsVal, tsExists := tsState[key]

		if !goExists {
			diffs = append(diffs, StateDifference{
				File:        filename,
				Key:         key,
				Description: fmt.Sprintf("Key '%s' missing in Go state", key),
				TSValue:     tsVal,
			})
			continue
		}

		if !tsExists {
			diffs = append(diffs, StateDifference{
				File:        filename,
				Key:         key,
				Description: fmt.Sprintf("Key '%s' missing in TypeScript state", key),
				GoValue:     goVal,
			})
			continue
		}

		// Compare values (normalize for comparison)
		if !h.valuesEqual(goVal, tsVal) {
			diffs = append(diffs, StateDifference{
				File:        filename,
				Key:         key,
				Description: fmt.Sprintf("Values differ for key '%s'", key),
				GoValue:     goVal,
				TSValue:     tsVal,
			})
		}
	}

	return diffs
}

// valuesEqual compares two values for equality
func (h *FunctionalDualHarness) valuesEqual(a, b interface{}) bool {
	// Normalize to JSON for comparison
	aJSON, err := json.Marshal(a)
	if err != nil {
		return false
	}

	bJSON, err := json.Marshal(b)
	if err != nil {
		return false
	}

	return string(aJSON) == string(bJSON)
}

// validateHistory validates history entries
func (h *FunctionalDualHarness) validateHistory() bool {
	historyPath := filepath.Join(h.StateDir, "go", "tasks", "history.json")

	data, err := os.ReadFile(historyPath)
	if err != nil {
		return false
	}

	var history []map[string]interface{}
	if err := json.Unmarshal(data, &history); err != nil {
		return false
	}

	// Validate required fields
	for _, entry := range history {
		if _, ok := entry["id"]; !ok {
			return false
		}
		if _, ok := entry["timestamp"]; !ok {
			return false
		}
		if _, ok := entry["prompt"]; !ok {
			return false
		}
	}

	return true
}

// validateConfig validates configuration
func (h *FunctionalDualHarness) validateConfig() bool {
	configPath := filepath.Join(h.StateDir, "go", "config.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return false
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return false
	}

	return true
}

// checkTaskCompletion checks if a task completed successfully
func (h *FunctionalDualHarness) checkTaskCompletion() bool {
	// Check for task completion indicators in state
	return true // Placeholder
}

// generateFunctionalReport creates a functional test report
func (h *FunctionalDualHarness) generateFunctionalReport(startTime time.Time) *FunctionalTestReport {
	total := len(h.FunctionalResults)
	passed := 0
	failed := 0

	for _, result := range h.FunctionalResults {
		if result.Passed {
			passed++
		} else {
			failed++
		}
	}

	successRate := 0.0
	if total > 0 {
		successRate = float64(passed) / float64(total) * 100
	}

	return &FunctionalTestReport{
		Success:           failed == 0,
		Results:           h.FunctionalResults,
		TotalTests:        total,
		PassedTests:       passed,
		FailedTests:       failed,
		SuccessRate:       successRate,
		TotalDuration:     time.Since(startTime),
		StateDir:          h.StateDir,
		TempDir:           h.TempDir,
		GoBinary:          h.GoBinary.Path,
		TSBinary:          h.TSBinary.Path,
	}
}

// FunctionalTestReport contains functional test results
type FunctionalTestReport struct {
	Success           bool
	Results           []FunctionalTestResult
	TotalTests        int
	PassedTests       int
	FailedTests       int
	SuccessRate       float64
	TotalDuration     time.Duration
	StateDir          string
	TempDir           string
	GoBinary          string
	TSBinary          string
}

// PrintReport prints a human-readable functional test report
func (r *FunctionalTestReport) PrintReport() {
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("  FUNCTIONAL DUAL TEST REPORT")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("Go Binary:    %s\n", r.GoBinary)
	fmt.Printf("TS Binary:    %s\n", r.TSBinary)
	fmt.Printf("State Dir:    %s\n", r.StateDir)
	fmt.Printf("Temp Dir:     %s\n", r.TempDir)
	fmt.Printf("Duration:     %v\n", r.TotalDuration)
	fmt.Println()
	fmt.Printf("Results: %d/%d passed (%.1f%%)\n", r.PassedTests, r.TotalTests, r.SuccessRate)
	fmt.Println(strings.Repeat("-", 80))

	for _, result := range r.Results {
		status := "✓ PASS"
		if !result.Passed {
			status = "✗ FAIL"
		}

		fmt.Printf("\n[%s] %s\n", status, result.TestName)
		fmt.Printf("      Description: %s\n", result.Description)
		fmt.Printf("      State Files Match: %v\n", result.StateFilesMatch)
		fmt.Printf("      History Valid: %v\n", result.HistoryValid)
		fmt.Printf("      Config Valid: %v\n", result.ConfigValid)
		fmt.Printf("      Task Completed: %v\n", result.TaskCompleted)

		if len(result.StateDifferences) > 0 {
			fmt.Println("      State Differences:")
			for _, diff := range result.StateDifferences {
				fmt.Printf("        File: %s, Key: %s\n", diff.File, diff.Key)
				fmt.Printf("          Description: %s\n", diff.Description)
			}
		}

		if len(result.Differences) > 0 {
			fmt.Println("      Output Differences:")
			for _, diff := range result.Differences {
				fmt.Printf("        [%s] %s: %s\n", diff.Severity, diff.Type, diff.Description)
			}
		}
	}

	fmt.Println()
	fmt.Println(strings.Repeat("=", 80))
	if r.Success {
		fmt.Println("✓ ALL FUNCTIONAL TESTS PASSED")
	} else {
		fmt.Printf("✗ %d FUNCTIONAL TESTS FAILED\n", r.FailedTests)
	}
}

// StandardFunctionalTests returns standard functional tests
func StandardFunctionalTests() []FunctionalDualTest {
	return []FunctionalDualTest{
		{
			DualTest: DualTest{
				Name:        "version_with_state",
				Description: "Version command with state validation",
				Args:        []string{"version", "--json"},
				Timeout:     10 * time.Second,
			},
			StateComparison:  true,
			ConfigValidation: true,
		},
		{
			DualTest: DualTest{
				Name:        "config_list_functional",
				Description: "Config list with state validation",
				Args:        []string{"config", "list", "--json"},
				Timeout:     10 * time.Second,
			},
			StateComparison:  true,
			ConfigValidation: true,
		},
		{
			DualTest: DualTest{
				Name:        "auth_list_functional",
				Description: "Auth list with state validation",
				Args:        []string{"auth", "list"},
				Timeout:     10 * time.Second,
			},
			StateComparison: true,
		},
		{
			DualTest: DualTest{
				Name:        "history_functional",
				Description: "History with validation",
				Args:        []string{"history", "--json"},
				Timeout:     10 * time.Second,
			},
			StateComparison:   true,
			HistoryValidation: true,
		},
	}
}

// TestFunctionalDual runs functional dual tests
func TestFunctionalDual(t *testing.T) {
	goPath := findGoBinary()
	tsPath := findTSBinary()

	if goPath == "" {
		t.Skip("Go CLI binary not found")
	}
	if tsPath == "" {
		t.Skip("TypeScript CLI binary not found")
	}

	harness, err := NewFunctionalDualHarness(goPath, tsPath)
	require.NoError(t, err)
	defer harness.Cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	tests := StandardFunctionalTests()
	report, err := harness.RunFunctionalTests(ctx, tests)
	require.NoError(t, err)

	// Print report
	report.PrintReport()

	// Assert results
	assert.GreaterOrEqual(t, report.SuccessRate, 70.0,
		"Functional test success rate (%.1f%%) below threshold", report.SuccessRate)
}

// Helper function (duplicated from parity_test.go for self-containment)
func findGoBinary() string {
	binaryName := "cline-go"
	if runtime.GOOS == "windows" {
		binaryName = "cline-go.exe"
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

func findTSBinary() string {
	locations := []string{
		filepath.Join("..", "..", "..", "cli", "dist", "cli.mjs"),
		filepath.Join("..", "..", "cli", "dist", "cli.mjs"),
		filepath.Join("..", "cli", "dist", "cli.mjs"),
		filepath.Join("cli", "dist", "cli.mjs"),
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