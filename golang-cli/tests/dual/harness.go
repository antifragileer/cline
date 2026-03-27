// Package dual provides a comprehensive dual testing framework for comparing
// Go CLI and TypeScript CLI behavior to ensure 100% parity.
//
// This is a test helper package, not a regular test file. It provides
// utilities for comparing Go and TypeScript CLI implementations.
package dual

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

// CLIBinary represents a CLI binary under test
type CLIBinary struct {
	Name       string
	Path       string
	WorkingDir string
	Env        map[string]string
}

// DualTest represents a test case for dual CLI comparison
type DualTest struct {
	Name           string
	Description    string
	Args           []string
	Env            map[string]string
	Stdin          string
	Timeout        time.Duration
	SkipOutput     bool   // Skip output comparison
	SkipExitCode   bool   // Skip exit code comparison
	ExpectFailure  bool   // Both should fail
	OutputMatchers []OutputMatcher
	PreHook        func() error
	PostHook       func() error
}

// OutputMatcher defines custom output validation
type OutputMatcher struct {
	Name      string
	Pattern   *regexp.Regexp
	Required  bool
	InBoth    bool // Must match in both outputs
	Normalize bool // Apply normalization before matching
}

// DualTestResult represents the result of a dual test
type DualTestResult struct {
	TestName     string
	Description  string
	Passed       bool
	GoResult     *ExecutionResult
	TSResult     *ExecutionResult
	Differences  []Difference
	Warnings     []string
	Duration     time.Duration
	Timestamp    time.Time
}

// ExecutionResult represents the result of executing a CLI command
type ExecutionResult struct {
	Binary      string
	Args        []string
	ExitCode    int
	Stdout      string
	Stderr      string
	Combined    string
	Duration    time.Duration
	StartTime   time.Time
	EndTime     time.Time
}

// Difference represents a difference between Go and TS outputs
type Difference struct {
	Type        string
	Field       string
	GoValue     string
	TSValue     string
	Description string
	Severity    DifferenceSeverity
}

// DifferenceSeverity indicates how serious a difference is
type DifferenceSeverity int

const (
	SeverityInfo DifferenceSeverity = iota
	SeverityWarning
	SeverityError
	SeverityCritical
)

func (s DifferenceSeverity) String() string {
	switch s {
	case SeverityInfo:
		return "INFO"
	case SeverityWarning:
		return "WARNING"
	case SeverityError:
		return "ERROR"
	case SeverityCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// DualTestReport contains all test results
type DualTestReport struct {
	Success       bool
	Results       []DualTestResult
	Summary       string
	ExitCode      int
	TotalTests    int
	PassedTests   int
	FailedTests   int
	SkippedTests  int
	TotalDuration time.Duration
	StartTime     time.Time
	EndTime       time.Time
	GoBinary      string
	TSBinary      string
}

// DualTestHarness orchestrates dual testing
type DualTestHarness struct {
	GoBinary *CLIBinary
	TSBinary *CLIBinary
	Results  []DualTestResult
	mu       sync.RWMutex
}

// NewDualTestHarness creates a new test harness
func NewDualTestHarness(goPath, tsPath string) *DualTestHarness {
	return &DualTestHarness{
		GoBinary: &CLIBinary{
			Name: "Go CLI",
			Path: goPath,
			Env:  make(map[string]string),
		},
		TSBinary: &CLIBinary{
			Name: "TypeScript CLI",
			Path: tsPath,
			Env:  make(map[string]string),
		},
		Results: make([]DualTestResult, 0),
	}
}

// RunTests executes all dual tests and generates a report
func (h *DualTestHarness) RunTests(ctx context.Context, tests []DualTest) (*DualTestReport, error) {
	startTime := time.Now()
	h.Results = make([]DualTestResult, 0)

	// Validate binaries
	if err := h.validateBinaries(); err != nil {
		return nil, fmt.Errorf("binary validation failed: %w", err)
	}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 4) // Limit concurrent tests

	for _, test := range tests {
		wg.Add(1)
		semaphore <- struct{}{}

		go func(t DualTest) {
			defer wg.Done()
			defer func() { <-semaphore }()

			result := h.runTest(ctx, t)
			h.mu.Lock()
			h.Results = append(h.Results, result)
			h.mu.Unlock()
		}(test)
	}

	wg.Wait()

	return h.generateReport(startTime), nil
}

// RunTestsSequential executes tests sequentially (for tests that can't run in parallel)
func (h *DualTestHarness) RunTestsSequential(ctx context.Context, tests []DualTest) (*DualTestReport, error) {
	startTime := time.Now()
	h.Results = make([]DualTestResult, 0)

	if err := h.validateBinaries(); err != nil {
		return nil, fmt.Errorf("binary validation failed: %w", err)
	}

	for _, test := range tests {
		result := h.runTest(ctx, test)
		h.Results = append(h.Results, result)
	}

	return h.generateReport(startTime), nil
}

// validateBinaries checks that both binaries exist and are executable
func (h *DualTestHarness) validateBinaries() error {
	for _, binary := range []*CLIBinary{h.GoBinary, h.TSBinary} {
		if binary.Path == "" {
			return fmt.Errorf("%s path not set", binary.Name)
		}

		info, err := os.Stat(binary.Path)
		if err != nil {
			return fmt.Errorf("%s not found at %s: %w", binary.Name, binary.Path, err)
		}

		// Check if executable (skip on Windows)
		if runtime.GOOS != "windows" {
			if info.Mode()&0111 == 0 {
				return fmt.Errorf("%s at %s is not executable", binary.Name, binary.Path)
			}
		}
	}

	return nil
}

// runTest executes a single dual test
func (h *DualTestHarness) runTest(ctx context.Context, test DualTest) DualTestResult {
	startTime := time.Now()
	result := DualTestResult{
		TestName:    test.Name,
		Description: test.Description,
		Timestamp:   startTime,
		Passed:      false,
		Differences: make([]Difference, 0),
		Warnings:    make([]string, 0),
	}

	timeout := test.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	testCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Run pre-hook if defined
	if test.PreHook != nil {
		if err := test.PreHook(); err != nil {
			result.Differences = append(result.Differences, Difference{
				Type:        "PreHook",
				Description: fmt.Sprintf("Pre-hook failed: %v", err),
				Severity:    SeverityCritical,
			})
			result.Duration = time.Since(startTime)
			return result
		}
	}

	// Execute both binaries
	var wg sync.WaitGroup
	var goResult, tsResult *ExecutionResult

	wg.Add(2)
	go func() {
		defer wg.Done()
		goResult = h.executeBinary(testCtx, h.GoBinary, test)
	}()
	go func() {
		defer wg.Done()
		tsResult = h.executeBinary(testCtx, h.TSBinary, test)
	}()
	wg.Wait()

	result.GoResult = goResult
	result.TSResult = tsResult

	// Run post-hook if defined
	if test.PostHook != nil {
		if err := test.PostHook(); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Post-hook failed: %v", err))
		}
	}

	// Compare results
	result.Differences = h.compareResults(goResult, tsResult, test)

	// Determine pass/fail
	result.Passed = len(result.Differences) == 0 ||
		!h.hasCriticalDifferences(result.Differences)

	result.Duration = time.Since(startTime)
	return result
}

// executeBinary runs a single binary with the test configuration
func (h *DualTestHarness) executeBinary(ctx context.Context, binary *CLIBinary, test DualTest) *ExecutionResult {
	result := &ExecutionResult{
		Binary:    binary.Name,
		Args:      test.Args,
		StartTime: time.Now(),
	}

	if binary.Path == "" {
		result.EndTime = time.Now()
		result.Stderr = "Binary path not set"
		result.ExitCode = -1
		return result
	}

	cmd := exec.CommandContext(ctx, binary.Path, test.Args...)
	cmd.Dir = binary.WorkingDir

	// Set up environment
	cmd.Env = os.Environ()
	for k, v := range binary.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	for k, v := range test.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Set up stdin
	if test.Stdin != "" {
		cmd.Stdin = strings.NewReader(test.Stdin)
	}

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute
	err := cmd.Run()
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	result.Stdout = stdout.String()
	result.Stderr = stderr.String()
	result.Combined = result.Stdout + result.Stderr

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else if ctx.Err() == context.DeadlineExceeded {
			result.ExitCode = -2 // Timeout
			result.Stderr += "\n[TIMEOUT]"
		} else {
			result.ExitCode = -1
			result.Stderr += fmt.Sprintf("\n[ERROR: %v]", err)
		}
	}

	return result
}

// compareResults compares execution results and returns differences
func (h *DualTestHarness) compareResults(goResult, tsResult *ExecutionResult, test DualTest) []Difference {
	diffs := make([]Difference, 0)

	// Compare exit codes
	if !test.SkipExitCode && goResult.ExitCode != tsResult.ExitCode {
		diffs = append(diffs, Difference{
			Type:        "ExitCode",
			Field:       "exit_code",
			GoValue:     fmt.Sprintf("%d", goResult.ExitCode),
			TSValue:     fmt.Sprintf("%d", tsResult.ExitCode),
			Description: "Exit codes differ between Go and TypeScript CLI",
			Severity:    SeverityError,
		})
	}

	// Skip output comparison if requested
	if test.SkipOutput {
		return diffs
	}

	// Normalize outputs
	goNorm := normalizeOutput(goResult.Combined)
	tsNorm := normalizeOutput(tsResult.Combined)

	// Compare normalized outputs
	if goNorm != tsNorm {
		// Find specific line differences
		lineDiffs := compareLines(goResult.Combined, tsResult.Combined)
		diffs = append(diffs, lineDiffs...)
	}

	// Apply custom matchers
	for _, matcher := range test.OutputMatchers {
		goMatches := matcher.Pattern.MatchString(goResult.Combined)
		tsMatches := matcher.Pattern.MatchString(tsResult.Combined)

		if matcher.InBoth {
			if goMatches != tsMatches {
				diffs = append(diffs, Difference{
					Type:        "Matcher",
					Field:       matcher.Name,
					GoValue:     fmt.Sprintf("%v", goMatches),
					TSValue:     fmt.Sprintf("%v", tsMatches),
					Description: fmt.Sprintf("Pattern '%s' match differs", matcher.Name),
					Severity:    SeverityWarning,
				})
			}
		} else if matcher.Required {
			if !goMatches {
				diffs = append(diffs, Difference{
					Type:        "Matcher",
					Field:       matcher.Name,
					GoValue:     "false",
					TSValue:     "N/A",
					Description: fmt.Sprintf("Required pattern '%s' not found in Go output", matcher.Name),
					Severity:    SeverityError,
				})
			}
			if !tsMatches {
				diffs = append(diffs, Difference{
					Type:        "Matcher",
					Field:       matcher.Name,
					GoValue:     "N/A",
					TSValue:     "false",
					Description: fmt.Sprintf("Required pattern '%s' not found in TS output", matcher.Name),
					Severity:    SeverityError,
				})
			}
		}
	}

	// Compare durations (warning only)
	durationDiff := goResult.Duration - tsResult.Duration
	if durationDiff < 0 {
		durationDiff = -durationDiff
	}
	if durationDiff > 5*time.Second {
		diffs = append(diffs, Difference{
			Type:        "Performance",
			Field:       "duration",
			GoValue:     goResult.Duration.String(),
			TSValue:     tsResult.Duration.String(),
			Description: fmt.Sprintf("Significant duration difference: %v", durationDiff),
			Severity:    SeverityInfo,
		})
	}

	return diffs
}

// hasCriticalDifferences checks if any differences are critical
func (h *DualTestHarness) hasCriticalDifferences(diffs []Difference) bool {
	for _, d := range diffs {
		if d.Severity == SeverityCritical || d.Severity == SeverityError {
			return true
		}
	}
	return false
}

// generateReport creates a test report from results
func (h *DualTestHarness) generateReport(startTime time.Time) *DualTestReport {
	endTime := time.Now()
	totalDuration := endTime.Sub(startTime)

	passed := 0
	failed := 0
	skipped := 0

	for _, r := range h.Results {
		if r.Passed {
			passed++
		} else {
			failed++
		}
	}

	success := failed == 0
	summary := fmt.Sprintf("Dual Test Results: %d/%d passed", passed, len(h.Results))
	if !success {
		summary = fmt.Sprintf("Dual Test Results: %d/%d failed", failed, len(h.Results))
	}

	exitCode := 0
	if !success {
		exitCode = 1
	}

	return &DualTestReport{
		Success:       success,
		Results:       h.Results,
		Summary:       summary,
		ExitCode:      exitCode,
		TotalTests:    len(h.Results),
		PassedTests:   passed,
		FailedTests:   failed,
		SkippedTests:  skipped,
		TotalDuration: totalDuration,
		StartTime:     startTime,
		EndTime:       endTime,
		GoBinary:      h.GoBinary.Path,
		TSBinary:      h.TSBinary.Path,
	}
}

// normalizeOutput normalizes output for comparison
func normalizeOutput(output string) string {
	// Normalize line endings
	output = strings.ReplaceAll(output, "\r\n", "\n")

	// Remove timestamps (various formats)
	timestampPatterns := []string{
		`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})?`,
		`\d{2}:\d{2}:\d{2}`,
		`\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}`,
		`\d{2}:\d{2}:\d{2}\.\d+`,
	}

	for _, pattern := range timestampPatterns {
		re := regexp.MustCompile(pattern)
		output = re.ReplaceAllString(output, "[TIMESTAMP]")
	}

	// Remove UUIDs
	uuidPattern := `[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`
	re := regexp.MustCompile(uuidPattern)
	output = re.ReplaceAllString(output, "[UUID]")

	// Remove temporary paths
	output = strings.ReplaceAll(output, os.TempDir(), "[TEMPDIR]")

	// Normalize whitespace
	output = strings.TrimSpace(output)
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	output = strings.Join(lines, "\n")

	return output
}

// compareLines compares outputs line by line
func compareLines(goOutput, tsOutput string) []Difference {
	diffs := make([]Difference, 0)

	goLines := strings.Split(normalizeOutput(goOutput), "\n")
	tsLines := strings.Split(normalizeOutput(tsOutput), "\n")

	maxLines := len(goLines)
	if len(tsLines) > maxLines {
		maxLines = len(tsLines)
	}

	for i := 0; i < maxLines; i++ {
		goLine := ""
		tsLine := ""

		if i < len(goLines) {
			goLine = goLines[i]
		}
		if i < len(tsLines) {
			tsLine = tsLines[i]
		}

		if goLine != tsLine {
			diffs = append(diffs, Difference{
				Type:        "Line",
				Field:       fmt.Sprintf("line_%d", i+1),
				GoValue:     truncate(goLine, 100),
				TSValue:     truncate(tsLine, 100),
				Description: fmt.Sprintf("Line %d differs", i+1),
				Severity:    SeverityError,
			})
		}
	}

	return diffs
}

// truncate truncates a string to max length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// FindGoBinary attempts to find the Go CLI binary
func FindGoBinary() string {
	binaryName := "cline"
	if runtime.GOOS == "windows" {
		binaryName = "cline.exe"
	}

	locations := []string{
		filepath.Join(".", binaryName),
		filepath.Join("..", binaryName),
		filepath.Join("..", "..", binaryName),
		filepath.Join("cmd", "cline", binaryName),
		filepath.Join("..", "cmd", "cline", binaryName),
		filepath.Join("..", "..", "cmd", "cline", binaryName),
		filepath.Join("build", binaryName),
		filepath.Join("dist", binaryName),
		filepath.Join("bin", binaryName),
	}

	for _, loc := range locations {
		if absPath, err := filepath.Abs(loc); err == nil {
			if _, err := os.Stat(absPath); err == nil {
				return absPath
			}
		}
	}

	// Try PATH
	if path, err := exec.LookPath(binaryName); err == nil {
		return path
	}

	return ""
}

// FindTSBinary attempts to find the TypeScript CLI binary
func FindTSBinary() string {
	locations := []string{
		filepath.Join("..", "..", "..", "cli", "bin", "cline"),
		filepath.Join("..", "..", "cli", "bin", "cline"),
		filepath.Join("..", "cli", "bin", "cline"),
		filepath.Join("cli", "bin", "cline"),
		filepath.Join("..", "..", "..", "cli", "dist", "index.js"),
		filepath.Join("..", "..", "cli", "dist", "index.js"),
		filepath.Join("..", "cli", "dist", "index.js"),
		filepath.Join("cli", "dist", "index.js"),
	}

	for _, loc := range locations {
		if absPath, err := filepath.Abs(loc); err == nil {
			if _, err := os.Stat(absPath); err == nil {
				return absPath
			}
		}
	}

	// Try npm global
	if path, err := exec.LookPath("cline"); err == nil {
		// Check if it's the TS version
		cmd := exec.Command(path, "version", "--json")
		output, _ := cmd.CombinedOutput()
		if strings.Contains(string(output), "nodeVersion") {
			return path
		}
	}

	return ""
}

// StandardDualTests returns the standard set of dual tests
func StandardDualTests() []DualTest {
	return []DualTest{
		{
			Name:        "version_short",
			Description: "Short version output",
			Args:        []string{"version", "--short"},
			Timeout:     5 * time.Second,
		},
		{
			Name:        "version_json",
			Description: "JSON version output",
			Args:        []string{"version", "--json"},
			Timeout:     5 * time.Second,
			OutputMatchers: []OutputMatcher{
				{
					Name:     "version_field",
					Pattern:  regexp.MustCompile(`"version":`),
					Required: true,
					InBoth:   true,
				},
			},
		},
		{
			Name:        "help_root",
			Description: "Root help command",
			Args:        []string{"--help"},
			Timeout:     5 * time.Second,
		},
		{
			Name:        "config_list",
			Description: "List configuration",
			Args:        []string{"config", "list"},
			Timeout:     5 * time.Second,
		},
		{
			Name:        "history_json",
			Description: "History in JSON format",
			Args:        []string{"history", "--json"},
			Timeout:     5 * time.Second,
		},
		{
			Name:        "mcp_help",
			Description: "MCP help command",
			Args:        []string{"mcp", "--help"},
			Timeout:     5 * time.Second,
		},
		{
			Name:        "invalid_command",
			Description: "Invalid command handling",
			Args:        []string{"invalid-command-that-does-not-exist"},
			Timeout:     5 * time.Second,
			ExpectFailure: true,
		},
	}
}

// PrintReport prints a human-readable test report
func (r *DualTestReport) PrintReport(w io.Writer) {
	fmt.Fprintln(w, strings.Repeat("=", 80))
	fmt.Fprintln(w, "  DUAL CLI TEST REPORT")
	fmt.Fprintln(w, strings.Repeat("=", 80))
	fmt.Fprintf(w, "Go Binary:    %s\n", r.GoBinary)
	fmt.Fprintf(w, "TS Binary:    %s\n", r.TSBinary)
	fmt.Fprintf(w, "Duration:     %v\n", r.TotalDuration)
	fmt.Fprintf(w, "Start Time:   %s\n", r.StartTime.Format(time.RFC3339))
	fmt.Fprintf(w, "End Time:     %s\n", r.EndTime.Format(time.RFC3339))
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Results: %d/%d passed, %d failed\n", r.PassedTests, r.TotalTests, r.FailedTests)
	fmt.Fprintln(w, strings.Repeat("-", 80))

	for _, result := range r.Results {
		status := "✓ PASS"
		if !result.Passed {
			status = "✗ FAIL"
		}

		fmt.Fprintf(w, "\n[%s] %s\n", status, result.TestName)
		if result.Description != "" {
			fmt.Fprintf(w, "      Description: %s\n", result.Description)
		}
		fmt.Fprintf(w, "      Duration: %v\n", result.Duration)

		if len(result.Differences) > 0 {
			fmt.Fprintln(w, "      Differences:")
			for _, diff := range result.Differences {
				fmt.Fprintf(w, "        [%s] %s: %s\n", diff.Severity, diff.Type, diff.Description)
				if diff.GoValue != "" && diff.GoValue != "N/A" {
					fmt.Fprintf(w, "          Go: %s\n", diff.GoValue)
				}
				if diff.TSValue != "" && diff.TSValue != "N/A" {
					fmt.Fprintf(w, "          TS: %s\n", diff.TSValue)
				}
			}
		}

		if len(result.Warnings) > 0 {
			fmt.Fprintln(w, "      Warnings:")
			for _, warning := range result.Warnings {
				fmt.Fprintf(w, "        ⚠ %s\n", warning)
			}
		}
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, strings.Repeat("=", 80))
	fmt.Fprintf(w, "Summary: %s\n", r.Summary)
	fmt.Fprintf(w, "Exit Code: %d\n", r.ExitCode)
}

// ToJSON returns the report as JSON
func (r *DualTestReport) ToJSON() (string, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FilterResults returns only results matching a predicate
func (r *DualTestReport) FilterResults(predicate func(DualTestResult) bool) []DualTestResult {
	filtered := make([]DualTestResult, 0)
	for _, result := range r.Results {
		if predicate(result) {
			filtered = append(filtered, result)
		}
	}
	return filtered
}

// GetFailedResults returns all failed test results
func (r *DualTestReport) GetFailedResults() []DualTestResult {
	return r.FilterResults(func(r DualTestResult) bool {
		return !r.Passed
	})
}

// GetResultsBySeverity returns results with differences of a specific severity
func (r *DualTestReport) GetResultsBySeverity(severity DifferenceSeverity) []DualTestResult {
	return r.FilterResults(func(result DualTestResult) bool {
		for _, diff := range result.Differences {
			if diff.Severity == severity {
				return true
			}
		}
		return false
	})
}