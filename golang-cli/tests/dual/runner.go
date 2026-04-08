// Package dual provides test runner and comparison tools for the dual testing framework.
package dual

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// TestRunnerConfig configures the test runner
type TestRunnerConfig struct {
	GoBinaryPath  string
	TSBinaryPath  string
	OutputFormat  string // "text", "json", "junit"
	OutputPath    string
	Parallel      bool
	Verbose       bool
	IncludeTests  []string
	ExcludeTests  []string
	StopOnFailure bool
	Timeout       time.Duration
}

// TestRunner executes dual tests
type TestRunner struct {
	config  *TestRunnerConfig
	harness *DualTestHarness
}

// NewTestRunner creates a new test runner
func NewTestRunner(config *TestRunnerConfig) *TestRunner {
	return &TestRunner{
		config: config,
	}
}

// Run executes all configured tests
func (r *TestRunner) Run(ctx context.Context) (*DualTestReport, error) {
	// Auto-detect binaries if not provided
	if r.config.GoBinaryPath == "" {
		r.config.GoBinaryPath = FindGoBinary()
	}
	if r.config.TSBinaryPath == "" {
		r.config.TSBinaryPath = FindTSBinary()
	}

	// Validate binaries
	if r.config.GoBinaryPath == "" {
		return nil, fmt.Errorf("Go CLI binary not found")
	}
	if r.config.TSBinaryPath == "" {
		return nil, fmt.Errorf("TypeScript CLI binary not found")
	}

	// Create harness
	r.harness = NewDualTestHarness(r.config.GoBinaryPath, r.config.TSBinaryPath)

	// Get tests
	tests := r.getTests()

	// Filter tests
	tests = r.filterTests(tests)

	if len(tests) == 0 {
		return nil, fmt.Errorf("no tests to run")
	}

	// Run tests
	var report *DualTestReport
	var err error

	if r.config.Parallel {
		report, err = r.harness.RunTests(ctx, tests)
	} else {
		report, err = r.harness.RunTestsSequential(ctx, tests)
	}

	if err != nil {
		return nil, err
	}

	// Write output
	if err := r.writeOutput(report); err != nil {
		return nil, fmt.Errorf("failed to write output: %w", err)
	}

	return report, nil
}

// getTests returns the list of tests to run
func (r *TestRunner) getTests() []DualTest {
	// Start with standard tests
	tests := StandardDualTests()

	// Add format parity tests
	tests = append(tests, r.getFormatParityTests()...)

	// Add edge case tests (only for Go CLI since TS may not handle all edge cases)
	// These are handled separately in edge_case_test.go

	return tests
}

// getFormatParityTests returns format-specific parity tests
func (r *TestRunner) getFormatParityTests() []DualTest {
	return []DualTest{
		{
			Name:        "json_deterministic",
			Description: "JSON output is deterministic",
			Args:        []string{"version", "--json"},
			Timeout:     5 * time.Second,
		},
		{
			Name:        "text_format_consistency",
			Description: "Text format is consistent",
			Args:        []string{"version"},
			Timeout:     5 * time.Second,
		},
		{
			Name:          "error_format_consistency",
			Description:   "Error format is consistent",
			Args:          []string{"invalid-command"},
			Timeout:       5 * time.Second,
			ExpectFailure: true,
		},
	}
}

// filterTests filters tests based on include/exclude patterns
func (r *TestRunner) filterTests(tests []DualTest) []DualTest {
	if len(r.config.IncludeTests) == 0 && len(r.config.ExcludeTests) == 0 {
		return tests
	}

	filtered := make([]DualTest, 0)

	for _, test := range tests {
		// Check exclude patterns
		excluded := false
		for _, pattern := range r.config.ExcludeTests {
			if matched, _ := filepath.Match(pattern, test.Name); matched {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}

		// Check include patterns
		if len(r.config.IncludeTests) > 0 {
			included := false
			for _, pattern := range r.config.IncludeTests {
				if matched, _ := filepath.Match(pattern, test.Name); matched {
					included = true
					break
				}
			}
			if !included {
				continue
			}
		}

		filtered = append(filtered, test)
	}

	return filtered
}

// writeOutput writes the test report in the configured format
func (r *TestRunner) writeOutput(report *DualTestReport) error {
	var writer io.Writer = os.Stdout

	// Open output file if specified
	if r.config.OutputPath != "" {
		file, err := os.Create(r.config.OutputPath)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer file.Close()
		writer = file
	}

	switch r.config.OutputFormat {
	case "json":
		return r.writeJSON(writer, report)
	case "junit":
		return r.writeJUnit(writer, report)
	default:
		return r.writeText(writer, report)
	}
}

// writeText writes a text format report
func (r *TestRunner) writeText(w io.Writer, report *DualTestReport) error {
	report.PrintReport(w)
	return nil
}

// writeJSON writes a JSON format report
func (r *TestRunner) writeJSON(w io.Writer, report *DualTestReport) error {
	jsonStr, err := report.ToJSON()
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, jsonStr)
	return err
}

// writeJUnit writes a JUnit XML format report
func (r *TestRunner) writeJUnit(w io.Writer, report *DualTestReport) error {
	// JUnit XML format
	fmt.Fprintln(w, `<?xml version="1.0" encoding="UTF-8"?>`)
	fmt.Fprintln(w, `<testsuites>`)
	fmt.Fprintf(w, `  <testsuite name="dual-tests" tests="%d" failures="%d" time="%f">`,
		report.TotalTests, report.FailedTests, report.TotalDuration.Seconds())
	fmt.Fprintln(w)

	for _, result := range report.Results {
		fmt.Fprintf(w, `    <testcase name="%s" time="%f">`,
			escapeXML(result.TestName), result.Duration.Seconds())
		fmt.Fprintln(w)

		if !result.Passed {
			fmt.Fprintf(w, `      <failure message="%s">`,
				escapeXML(result.Differences[0].Description))
			fmt.Fprintln(w)
			for _, diff := range result.Differences {
				fmt.Fprintf(w, "        %s: %s\n", diff.Type, diff.Description)
				if diff.GoValue != "" {
					fmt.Fprintf(w, "        Go: %s\n", diff.GoValue)
				}
				if diff.TSValue != "" {
					fmt.Fprintf(w, "        TS: %s\n", diff.TSValue)
				}
			}
			fmt.Fprintln(w, `      </failure>`)
		}

		fmt.Fprintln(w, `    </testcase>`)
	}

	fmt.Fprintln(w, `  </testsuite>`)
	fmt.Fprintln(w, `</testsuites>`)

	return nil
}

// escapeXML escapes special XML characters
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&#38;")
	s = strings.ReplaceAll(s, "<", "&#60;")
	s = strings.ReplaceAll(s, ">", "&#62;")
	s = strings.ReplaceAll(s, "\"", "&#34;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}

// ComparisonTool provides detailed comparison between Go and TS CLI outputs
type ComparisonTool struct {
	GoBinary string
	TSBinary string
	Verbose  bool
	DiffOnly bool
}

// NewComparisonTool creates a new comparison tool
func NewComparisonTool(goBinary, tsBinary string) *ComparisonTool {
	return &ComparisonTool{
		GoBinary: goBinary,
		TSBinary: tsBinary,
	}
}

// CompareCommand compares a single command between Go and TS CLI
func (c *ComparisonTool) CompareCommand(ctx context.Context, args []string) (*CommandComparison, error) {
	harness := NewDualTestHarness(c.GoBinary, c.TSBinary)

	test := DualTest{
		Name:    "comparison",
		Args:    args,
		Timeout: 30 * time.Second,
	}

	result := harness.runTest(ctx, test)

	return &CommandComparison{
		Args:        args,
		GoResult:    result.GoResult,
		TSResult:    result.TSResult,
		Differences: result.Differences,
		Match:       len(result.Differences) == 0,
	}, nil
}

// CommandComparison represents a comparison of a single command
type CommandComparison struct {
	Args        []string
	GoResult    *ExecutionResult
	TSResult    *ExecutionResult
	Differences []Difference
	Match       bool
}

// Print prints the comparison result
func (c *CommandComparison) Print(w io.Writer, diffOnly bool) {
	fmt.Fprintln(w, strings.Repeat("=", 80))
	fmt.Fprintf(w, "Command: cline %s\n", strings.Join(c.Args, " "))
	fmt.Fprintln(w, strings.Repeat("=", 80))

	if c.Match {
		fmt.Fprintln(w, "✓ OUTPUTS MATCH")
	} else {
		fmt.Fprintln(w, "✗ OUTPUTS DIFFER")
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "--- Go CLI ---")
	fmt.Fprintf(w, "Exit Code: %d\n", c.GoResult.ExitCode)
	fmt.Fprintf(w, "Duration: %v\n", c.GoResult.Duration)
	if !diffOnly {
		fmt.Fprintln(w, "Output:")
		fmt.Fprintln(w, c.GoResult.Combined)
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "--- TypeScript CLI ---")
	fmt.Fprintf(w, "Exit Code: %d\n", c.TSResult.ExitCode)
	fmt.Fprintf(w, "Duration: %v\n", c.TSResult.Duration)
	if !diffOnly {
		fmt.Fprintln(w, "Output:")
		fmt.Fprintln(w, c.TSResult.Combined)
	}

	if len(c.Differences) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "--- Differences ---")
		for i, diff := range c.Differences {
			fmt.Fprintf(w, "%d. [%s] %s\n", i+1, diff.Severity, diff.Type)
			fmt.Fprintf(w, "   %s\n", diff.Description)
			if diff.GoValue != "" {
				fmt.Fprintf(w, "   Go: %s\n", diff.GoValue)
			}
			if diff.TSValue != "" {
				fmt.Fprintf(w, "   TS: %s\n", diff.TSValue)
			}
			fmt.Fprintln(w)
		}
	}
}

// RegressionDetector detects potential regressions
type RegressionDetector struct {
	harness  *DualTestHarness
	baseline *DualTestReport
}

// NewRegressionDetector creates a new regression detector
func NewRegressionDetector(harness *DualTestHarness) *RegressionDetector {
	return &RegressionDetector{
		harness: harness,
	}
}

// LoadBaseline loads a baseline report for comparison
func (r *RegressionDetector) LoadBaseline(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read baseline: %w", err)
	}

	var report DualTestReport
	if err := json.Unmarshal(data, &report); err != nil {
		return fmt.Errorf("failed to parse baseline: %w", err)
	}

	r.baseline = &report
	return nil
}

// DetectRegressions compares current results against baseline
func (r *RegressionDetector) DetectRegressions(ctx context.Context, tests []DualTest) (*RegressionReport, error) {
	if r.baseline == nil {
		return nil, fmt.Errorf("no baseline loaded")
	}

	current, err := r.harness.RunTests(ctx, tests)
	if err != nil {
		return nil, fmt.Errorf("failed to run tests: %w", err)
	}

	report := &RegressionReport{
		Timestamp:    time.Now(),
		Baseline:     r.baseline,
		Current:      current,
		Regressions:  make([]Regression, 0),
		Improvements: make([]Improvement, 0),
	}

	// Compare results
	baselineResults := make(map[string]DualTestResult)
	for _, result := range r.baseline.Results {
		baselineResults[result.TestName] = result
	}

	for _, currentResult := range current.Results {
		baselineResult, ok := baselineResults[currentResult.TestName]
		if !ok {
			// New test
			continue
		}

		if baselineResult.Passed && !currentResult.Passed {
			// Regression
			report.Regressions = append(report.Regressions, Regression{
				TestName:     currentResult.TestName,
				BaselineDiff: len(baselineResult.Differences),
				CurrentDiff:  len(currentResult.Differences),
				Description:  "Test was passing but now fails",
			})
		} else if !baselineResult.Passed && currentResult.Passed {
			// Improvement
			report.Improvements = append(report.Improvements, Improvement{
				TestName:    currentResult.TestName,
				Description: "Test was failing but now passes",
			})
		}
	}

	return report, nil
}

// RegressionReport contains regression detection results
type RegressionReport struct {
	Timestamp    time.Time
	Baseline     *DualTestReport
	Current      *DualTestReport
	Regressions  []Regression
	Improvements []Improvement
}

// Regression represents a detected regression
type Regression struct {
	TestName     string
	BaselineDiff int
	CurrentDiff  int
	Description  string
}

// Improvement represents an improvement
type Improvement struct {
	TestName    string
	Description string
}

// Print prints the regression report
func (r *RegressionReport) Print(w io.Writer) {
	fmt.Fprintln(w, strings.Repeat("=", 80))
	fmt.Fprintln(w, "REGRESSION DETECTION REPORT")
	fmt.Fprintln(w, strings.Repeat("=", 80))
	fmt.Fprintf(w, "Timestamp: %s\n", r.Timestamp.Format(time.RFC3339))
	fmt.Fprintf(w, "Baseline:  %s\n", r.Baseline.StartTime.Format(time.RFC3339))
	fmt.Fprintf(w, "Current:   %s\n", r.Current.StartTime.Format(time.RFC3339))
	fmt.Fprintln(w)

	if len(r.Regressions) > 0 {
		fmt.Fprintf(w, "⚠ REGRESSIONS DETECTED: %d\n", len(r.Regressions))
		for _, reg := range r.Regressions {
			fmt.Fprintf(w, "  - %s: %s\n", reg.TestName, reg.Description)
		}
		fmt.Fprintln(w)
	}

	if len(r.Improvements) > 0 {
		fmt.Fprintf(w, "✓ IMPROVEMENTS: %d\n", len(r.Improvements))
		for _, imp := range r.Improvements {
			fmt.Fprintf(w, "  - %s: %s\n", imp.TestName, imp.Description)
		}
		fmt.Fprintln(w)
	}

	if len(r.Regressions) == 0 && len(r.Improvements) == 0 {
		fmt.Fprintln(w, "No changes detected")
	}
}

// ExitCode returns the exit code for CI integration
func (r *RegressionReport) ExitCode() int {
	if len(r.Regressions) > 0 {
		return 1
	}
	return 0
}
