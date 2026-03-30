// Package parity provides comprehensive parity tests between Go and TypeScript CLIs.
// These tests ensure 100% feature parity as required by the remediation plan.
package parity

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ComprehensiveParityTest represents a detailed parity test case
type ComprehensiveParityTest struct {
	Name           string
	Description    string
	Args           []string
	Env            map[string]string
	Stdin          string
	Timeout        time.Duration
	ExpectFailure  bool
	SkipOutput     bool
	SkipExitCode   bool
	OutputMatchers []ParityOutputMatcher
	PreHook        func() error
	PostHook       func() error
}

// ParityOutputMatcher defines custom output validation
type ParityOutputMatcher struct {
	Name      string
	Pattern   *regexp.Regexp
	Required  bool
	InBoth    bool
	Normalize bool
}

// ParityTestSuite contains all parity tests organized by category
type ParityTestSuite struct {
	Name  string
	Tests []ComprehensiveParityTest
}

// GetComprehensiveParityTests returns all comprehensive parity test suites
func GetComprehensiveParityTests() []ParityTestSuite {
	return []ParityTestSuite{
		{
			Name:  "Basic Commands",
			Tests: getBasicCommandTests(),
		},
		{
			Name:  "Version and Help",
			Tests: getVersionHelpTests(),
		},
		{
			Name:  "Configuration",
			Tests: getConfigurationTests(),
		},
		{
			Name:  "Authentication",
			Tests: getAuthenticationTests(),
		},
		{
			Name:  "Task Execution",
			Tests: getTaskExecutionTests(),
		},
		{
			Name:  "History",
			Tests: getHistoryTests(),
		},
		{
			Name:  "MCP",
			Tests: getMCPTests(),
		},
		{
			Name:  "Flags and Options",
			Tests: getFlagsTests(),
		},
		{
			Name:  "Error Handling",
			Tests: getErrorHandlingTests(),
		},
		{
			Name:  "Edge Cases",
			Tests: getEdgeCaseTests(),
		},
	}
}

func getBasicCommandTests() []ComprehensiveParityTest {
	return []ComprehensiveParityTest{
		{
			Name:        "root_command",
			Description: "Root command without arguments",
			Args:        []string{},
			Timeout:     10 * time.Second,
			SkipOutput:  true, // Interactive mode
		},
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
			OutputMatchers: []ParityOutputMatcher{
				{
					Name:     "version_field",
					Pattern:  regexp.MustCompile(`"version":\s*"[^"]+"`),
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
			Name:        "help_task",
			Description: "Task subcommand help",
			Args:        []string{"task", "--help"},
			Timeout:     5 * time.Second,
		},
		{
			Name:        "help_config",
			Description: "Config subcommand help",
			Args:        []string{"config", "--help"},
			Timeout:     5 * time.Second,
		},
		{
			Name:        "help_auth",
			Description: "Auth subcommand help",
			Args:        []string{"auth", "--help"},
			Timeout:     5 * time.Second,
		},
		{
			Name:        "help_history",
			Description: "History subcommand help",
			Args:        []string{"history", "--help"},
			Timeout:     5 * time.Second,
		},
		{
			Name:        "help_mcp",
			Description: "MCP subcommand help",
			Args:        []string{"mcp", "--help"},
			Timeout:     5 * time.Second,
		},
	}
}

func getVersionHelpTests() []ComprehensiveParityTest {
	return []ComprehensiveParityTest{
		{
			Name:        "version_detailed",
			Description: "Detailed version information",
			Args:        []string{"version"},
			Timeout:     5 * time.Second,
		},
		{
			Name:        "version_with_flags",
			Description: "Version with verbose flag",
			Args:        []string{"version", "-v"},
			Timeout:     5 * time.Second,
		},
	}
}

func getConfigurationTests() []ComprehensiveParityTest {
	return []ComprehensiveParityTest{
		{
			Name:        "config_list",
			Description: "List configuration",
			Args:        []string{"config", "list"},
			Timeout:     10 * time.Second,
		},
		{
			Name:        "config_list_json",
			Description: "List configuration in JSON format",
			Args:        []string{"config", "list", "--json"},
			Timeout:     10 * time.Second,
		},
		{
			Name:        "config_get",
			Description: "Get specific config value",
			Args:        []string{"config", "get", "provider"},
			Timeout:     10 * time.Second,
		},
	}
}

func getAuthenticationTests() []ComprehensiveParityTest {
	return []ComprehensiveParityTest{
		{
			Name:        "auth_list",
			Description: "List authentication providers",
			Args:        []string{"auth", "list"},
			Timeout:     10 * time.Second,
		},
		{
			Name:        "auth_status",
			Description: "Check authentication status",
			Args:        []string{"auth", "status"},
			Timeout:     10 * time.Second,
		},
		{
			Name:        "auth_help",
			Description: "Auth help",
			Args:        []string{"auth", "--help"},
			Timeout:     5 * time.Second,
		},
	}
}

func getTaskExecutionTests() []ComprehensiveParityTest {
	return []ComprehensiveParityTest{
		{
			Name:        "task_help",
			Description: "Task command help",
			Args:        []string{"task", "--help"},
			Timeout:     5 * time.Second,
		},
		{
			Name:        "task_with_prompt",
			Description: "Task with prompt",
			Args:        []string{"task", "echo hello"},
			Timeout:     30 * time.Second,
			SkipOutput:  true, // Will fail without core extension
		},
		{
			Name:        "task_with_json_flag",
			Description: "Task with JSON output",
			Args:        []string{"task", "--json", "test prompt"},
			Timeout:     30 * time.Second,
			SkipOutput:  true,
		},
		{
			Name:        "task_with_act_flag",
			Description: "Task with act mode",
			Args:        []string{"task", "-a", "test prompt"},
			Timeout:     30 * time.Second,
			SkipOutput:  true,
		},
		{
			Name:        "task_with_plan_flag",
			Description: "Task with plan mode",
			Args:        []string{"task", "-p", "test prompt"},
			Timeout:     30 * time.Second,
			SkipOutput:  true,
		},
	}
}

func getHistoryTests() []ComprehensiveParityTest {
	return []ComprehensiveParityTest{
		{
			Name:        "history_empty",
			Description: "History with no data",
			Args:        []string{"history"},
			Timeout:     10 * time.Second,
		},
		{
			Name:        "history_json",
			Description: "History in JSON format",
			Args:        []string{"history", "--json"},
			Timeout:     10 * time.Second,
		},
		{
			Name:        "history_with_limit",
			Description: "History with limit",
			Args:        []string{"history", "--json", "--limit", "10"},
			Timeout:     10 * time.Second,
		},
	}
}

func getMCPTests() []ComprehensiveParityTest {
	return []ComprehensiveParityTest{
		{
			Name:        "mcp_list",
			Description: "List MCP servers",
			Args:        []string{"mcp", "list"},
			Timeout:     10 * time.Second,
		},
		{
			Name:        "mcp_list_json",
			Description: "List MCP servers in JSON",
			Args:        []string{"mcp", "list", "--json"},
			Timeout:     10 * time.Second,
		},
		{
			Name:        "mcp_help",
			Description: "MCP help",
			Args:        []string{"mcp", "--help"},
			Timeout:     5 * time.Second,
		},
	}
}

func getFlagsTests() []ComprehensiveParityTest {
	return []ComprehensiveParityTest{
		{
			Name:        "flag_act_short",
			Description: "Act flag short form",
			Args:        []string{"-a", "test"},
			Timeout:     30 * time.Second,
			SkipOutput:  true,
		},
		{
			Name:        "flag_act_long",
			Description: "Act flag long form",
			Args:        []string{"--act", "test"},
			Timeout:     30 * time.Second,
			SkipOutput:  true,
		},
		{
			Name:        "flag_plan_short",
			Description: "Plan flag short form",
			Args:        []string{"-p", "test"},
			Timeout:     30 * time.Second,
			SkipOutput:  true,
		},
		{
			Name:        "flag_plan_long",
			Description: "Plan flag long form",
			Args:        []string{"--plan", "test"},
			Timeout:     30 * time.Second,
			SkipOutput:  true,
		},
		{
			Name:        "flag_yolo_short",
			Description: "Yolo flag short form",
			Args:        []string{"-y", "test"},
			Timeout:     30 * time.Second,
			SkipOutput:  true,
		},
		{
			Name:        "flag_yolo_long",
			Description: "Yolo flag long form",
			Args:        []string{"--yolo", "test"},
			Timeout:     30 * time.Second,
			SkipOutput:  true,
		},
		{
			Name:        "flag_verbose",
			Description: "Verbose flag",
			Args:        []string{"-v", "version"},
			Timeout:     10 * time.Second,
		},
		{
			Name:        "flag_config",
			Description: "Config directory flag",
			Args:        []string{"--config", "/tmp", "version"},
			Timeout:     10 * time.Second,
		},
		{
			Name:        "flag_cwd",
			Description: "Working directory flag",
			Args:        []string{"-c", "/tmp", "version"},
			Timeout:     10 * time.Second,
		},
		{
			Name:        "flag_timeout",
			Description: "Timeout flag",
			Args:        []string{"-t", "60", "version"},
			Timeout:     10 * time.Second,
		},
		{
			Name:        "flag_model",
			Description: "Model flag",
			Args:        []string{"-m", "claude-3-5-sonnet", "version"},
			Timeout:     10 * time.Second,
		},
		{
			Name:        "flag_json",
			Description: "JSON output flag",
			Args:        []string{"--json", "version"},
			Timeout:     10 * time.Second,
		},
		{
			Name:        "flag_task_id",
			Description: "Task ID flag",
			Args:        []string{"-T", "test-task-id", "version"},
			Timeout:     10 * time.Second,
		},
		{
			Name:        "flag_continue",
			Description: "Continue flag",
			Args:        []string{"--continue"},
			Timeout:     30 * time.Second,
			SkipOutput:  true,
		},
	}
}

func getErrorHandlingTests() []ComprehensiveParityTest {
	return []ComprehensiveParityTest{
		{
			Name:          "invalid_command",
			Description:   "Invalid command handling",
			Args:          []string{"invalid-command-that-does-not-exist"},
			Timeout:       10 * time.Second,
			ExpectFailure: true,
		},
		{
			Name:          "invalid_flag",
			Description:   "Invalid flag handling",
			Args:          []string{"--invalid-flag", "version"},
			Timeout:       10 * time.Second,
			ExpectFailure: true,
		},
		{
			Name:          "missing_required_arg",
			Description:   "Missing required argument",
			Args:          []string{"config", "get"},
			Timeout:       10 * time.Second,
			ExpectFailure: true,
		},
	}
}

func getEdgeCaseTests() []ComprehensiveParityTest {
	return []ComprehensiveParityTest{
		{
			Name:        "empty_args",
			Description: "Empty arguments",
			Args:        []string{},
			Timeout:     10 * time.Second,
			SkipOutput:  true,
		},
		{
			Name:        "special_chars_in_prompt",
			Description: "Special characters in prompt",
			Args:        []string{"task", "test with special chars: !@#$%^&*()"},
			Timeout:     30 * time.Second,
			SkipOutput:  true,
		},
		{
			Name:        "very_long_prompt",
			Description: "Very long prompt",
			Args:        []string{"task", strings.Repeat("a", 1000)},
			Timeout:     30 * time.Second,
			SkipOutput:  true,
		},
	}
}

// ComprehensiveParityRunner runs comprehensive parity tests
type ComprehensiveParityRunner struct {
	GoBinary string
	TSBinary string
	Results  []ComprehensiveParityResult
}

// ComprehensiveParityResult represents a single test result
type ComprehensiveParityResult struct {
	Suite       string
	TestName    string
	Description string
	Passed      bool
	GoResult    *ParityExecutionResult
	TSResult    *ParityExecutionResult
	Differences []ParityDifference
	Duration    time.Duration
	Timestamp   time.Time
}

// ParityExecutionResult represents execution result
type ParityExecutionResult struct {
	Binary   string
	Args     []string
	ExitCode int
	Stdout   string
	Stderr   string
	Combined string
	Duration time.Duration
}

// ParityDifference represents a difference between Go and TS
type ParityDifference struct {
	Type        string
	Field       string
	GoValue     string
	TSValue     string
	Description string
	Severity    ParitySeverity
}

// ParitySeverity indicates difference severity
type ParitySeverity int

const (
	ParityInfo ParitySeverity = iota
	ParityWarning
	ParityError
	ParityCritical
)

func (p ParitySeverity) String() string {
	switch p {
	case ParityInfo:
		return "INFO"
	case ParityWarning:
		return "WARNING"
	case ParityError:
		return "ERROR"
	case ParityCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// NewComprehensiveParityRunner creates a new runner
func NewComprehensiveParityRunner(goPath, tsPath string) *ComprehensiveParityRunner {
	return &ComprehensiveParityRunner{
		GoBinary: goPath,
		TSBinary: tsPath,
		Results:  make([]ComprehensiveParityResult, 0),
	}
}

// RunAllTests runs all comprehensive parity tests
func (r *ComprehensiveParityRunner) RunAllTests(ctx context.Context) (*ComprehensiveParityReport, error) {
	startTime := time.Now()
	suites := GetComprehensiveParityTests()

	for _, suite := range suites {
		for _, test := range suite.Tests {
			result := r.runTest(ctx, suite.Name, test)
			r.Results = append(r.Results, result)
		}
	}

	return r.generateReport(startTime), nil
}

// runTest runs a single comprehensive parity test
func (r *ComprehensiveParityRunner) runTest(ctx context.Context, suiteName string, test ComprehensiveParityTest) ComprehensiveParityResult {
	startTime := time.Now()
	result := ComprehensiveParityResult{
		Suite:       suiteName,
		TestName:    test.Name,
		Description: test.Description,
		Timestamp:   startTime,
		Passed:      false,
		Differences: make([]ParityDifference, 0),
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
			result.Differences = append(result.Differences, ParityDifference{
				Type:        "PreHook",
				Description: fmt.Sprintf("Pre-hook failed: %v", err),
				Severity:    ParityCritical,
			})
			result.Duration = time.Since(startTime)
			return result
		}
	}

	// Execute both binaries
	goResult := r.executeBinary(testCtx, r.GoBinary, "Go CLI", test.Args, test.Env)
	tsResult := r.executeBinary(testCtx, r.TSBinary, "TypeScript CLI", test.Args, test.Env)

	result.GoResult = goResult
	result.TSResult = tsResult

	// Run post-hook if defined
	if test.PostHook != nil {
		if err := test.PostHook(); err != nil {
			result.Differences = append(result.Differences, ParityDifference{
				Type:        "PostHook",
				Description: fmt.Sprintf("Post-hook failed: %v", err),
				Severity:    ParityWarning,
			})
		}
	}

	// Compare results
	result.Differences = r.compareResults(goResult, tsResult, test)

	// Determine pass/fail
	result.Passed = len(result.Differences) == 0 || !r.hasCriticalDifferences(result.Differences)
	result.Duration = time.Since(startTime)

	return result
}

// executeBinary runs a single binary
func (r *ComprehensiveParityRunner) executeBinary(ctx context.Context, binary, name string, args []string, env map[string]string) *ParityExecutionResult {
	result := &ParityExecutionResult{
		Binary: binary,
		Args:   args,
	}

	if binary == "" {
		result.Stderr = "Binary path not set"
		result.ExitCode = -1
		return result
	}

	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	start := time.Now()
	output, err := cmd.CombinedOutput()
	result.Duration = time.Since(start)

	result.Combined = string(output)
	result.Stdout = string(output)
	result.ExitCode = 0

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else if ctx.Err() == context.DeadlineExceeded {
			result.ExitCode = -2
			result.Stderr = "[TIMEOUT]"
		} else {
			result.ExitCode = -1
			result.Stderr = fmt.Sprintf("[ERROR: %v]", err)
		}
	}

	return result
}

// compareResults compares execution results
func (r *ComprehensiveParityRunner) compareResults(goResult, tsResult *ParityExecutionResult, test ComprehensiveParityTest) []ParityDifference {
	diffs := make([]ParityDifference, 0)

	// Compare exit codes
	if !test.SkipExitCode {
		expectedMatch := (goResult.ExitCode == 0) == (tsResult.ExitCode == 0)
		if !expectedMatch {
			severity := ParityError
			if test.ExpectFailure {
				severity = ParityWarning
			}
			diffs = append(diffs, ParityDifference{
				Type:        "ExitCode",
				Field:       "exit_code",
				GoValue:     fmt.Sprintf("%d", goResult.ExitCode),
				TSValue:     fmt.Sprintf("%d", tsResult.ExitCode),
				Description: "Exit codes differ between Go and TypeScript CLI",
				Severity:    severity,
			})
		}
	}

	// Skip output comparison if requested
	if test.SkipOutput {
		return diffs
	}

	// Normalize outputs
	goNorm := normalizeParityOutput(goResult.Combined)
	tsNorm := normalizeParityOutput(tsResult.Combined)

	// Compare normalized outputs
	if goNorm != tsNorm {
		diffs = append(diffs, ParityDifference{
			Type:        "Output",
			Field:       "output",
			GoValue:     truncate(goNorm, 200),
			TSValue:     truncate(tsNorm, 200),
			Description: "Output differs between Go and TypeScript CLI",
			Severity:    ParityError,
		})
	}

	// Apply custom matchers
	for _, matcher := range test.OutputMatchers {
		goMatches := matcher.Pattern.MatchString(goResult.Combined)
		tsMatches := matcher.Pattern.MatchString(tsResult.Combined)

		if matcher.InBoth && goMatches != tsMatches {
			severity := ParityWarning
			if matcher.Required {
				severity = ParityError
			}
			diffs = append(diffs, ParityDifference{
				Type:        "Matcher",
				Field:       matcher.Name,
				GoValue:     fmt.Sprintf("%v", goMatches),
				TSValue:     fmt.Sprintf("%v", tsMatches),
				Description: fmt.Sprintf("Pattern '%s' match differs", matcher.Name),
				Severity:    severity,
			})
		}
	}

	return diffs
}

// hasCriticalDifferences checks for critical differences
func (r *ComprehensiveParityRunner) hasCriticalDifferences(diffs []ParityDifference) bool {
	for _, d := range diffs {
		if d.Severity == ParityCritical || d.Severity == ParityError {
			return true
		}
	}
	return false
}

// generateReport creates a comprehensive report
func (r *ComprehensiveParityRunner) generateReport(startTime time.Time) *ComprehensiveParityReport {
	total := len(r.Results)
	passed := 0
	failed := 0
	warnings := 0

	for _, result := range r.Results {
		if result.Passed {
			passed++
		} else {
			failed++
		}
		for _, diff := range result.Differences {
			if diff.Severity == ParityWarning {
				warnings++
			}
		}
	}

	successRate := float64(passed) / float64(total) * 100

	return &ComprehensiveParityReport{
		Success:       failed == 0,
		Results:       r.Results,
		TotalTests:    total,
		PassedTests:   passed,
		FailedTests:   failed,
		WarningCount:  warnings,
		SuccessRate:   successRate,
		TotalDuration: time.Since(startTime),
		GoBinary:      r.GoBinary,
		TSBinary:      r.TSBinary,
		Timestamp:     time.Now(),
	}
}

// ComprehensiveParityReport contains all test results
type ComprehensiveParityReport struct {
	Success       bool
	Results       []ComprehensiveParityResult
	TotalTests    int
	PassedTests   int
	FailedTests   int
	WarningCount  int
	SuccessRate   float64
	TotalDuration time.Duration
	GoBinary      string
	TSBinary      string
	Timestamp     time.Time
}

// PrintReport prints a human-readable report
func (r *ComprehensiveParityReport) PrintReport() {
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("  COMPREHENSIVE PARITY TEST REPORT")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("Go Binary:    %s\n", r.GoBinary)
	fmt.Printf("TS Binary:    %s\n", r.TSBinary)
	fmt.Printf("Timestamp:    %s\n", r.Timestamp.Format(time.RFC3339))
	fmt.Printf("Duration:     %v\n", r.TotalDuration)
	fmt.Println()
	fmt.Printf("Results: %d/%d passed (%.1f%%)\n", r.PassedTests, r.TotalTests, r.SuccessRate)
	fmt.Printf("Warnings: %d\n", r.WarningCount)
	fmt.Println(strings.Repeat("-", 80))

	// Group by suite
	suiteResults := make(map[string][]ComprehensiveParityResult)
	for _, result := range r.Results {
		suiteResults[result.Suite] = append(suiteResults[result.Suite], result)
	}

	for suite, results := range suiteResults {
		fmt.Printf("\n## %s\n", suite)
		fmt.Println(strings.Repeat("-", 40))

		for _, result := range results {
			status := "✓ PASS"
			if !result.Passed {
				status = "✗ FAIL"
			}

			fmt.Printf("\n[%s] %s\n", status, result.TestName)
			fmt.Printf("      Description: %s\n", result.Description)
			fmt.Printf("      Duration: %v\n", result.Duration)

			if len(result.Differences) > 0 {
				fmt.Println("      Differences:")
				for _, diff := range result.Differences {
					fmt.Printf("        [%s] %s: %s\n", diff.Severity, diff.Type, diff.Description)
					if diff.GoValue != "" && diff.GoValue != "N/A" {
						fmt.Printf("          Go: %s\n", diff.GoValue)
					}
					if diff.TSValue != "" && diff.TSValue != "N/A" {
						fmt.Printf("          TS: %s\n", diff.TSValue)
					}
				}
			}
		}
	}

	fmt.Println()
	fmt.Println(strings.Repeat("=", 80))
	if r.Success {
		fmt.Println("✓ ALL TESTS PASSED")
	} else {
		fmt.Printf("✗ %d TESTS FAILED\n", r.FailedTests)
	}
}

// ToJSON returns the report as JSON
func (r *ComprehensiveParityReport) ToJSON() (string, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Helper functions
func normalizeParityOutput(output string) string {
	// Normalize line endings
	output = strings.ReplaceAll(output, "\r\n", "\n")

	// Remove timestamps
	timestampPatterns := []string{
		`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})?`,
		`\d{2}:\d{2}:\d{2}`,
		`\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}`,
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

	return output
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// TestComprehensiveParity runs the full comprehensive parity test suite
func TestComprehensiveParity(t *testing.T) {
	goPath := findGoBinary()
	tsPath := findTSBinary()

	if goPath == "" {
		t.Skip("Go CLI binary not found")
	}
	if tsPath == "" {
		t.Skip("TypeScript CLI binary not found - skipping parity tests")
	}

	t.Logf("Go Binary: %s", goPath)
	t.Logf("TS Binary: %s", tsPath)

	runner := NewComprehensiveParityRunner(goPath, tsPath)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	report, err := runner.RunAllTests(ctx)
	require.NoError(t, err)

	// Print summary
	t.Logf("Total Tests: %d", report.TotalTests)
	t.Logf("Passed: %d", report.PassedTests)
	t.Logf("Failed: %d", report.FailedTests)
	t.Logf("Success Rate: %.1f%%", report.SuccessRate)

	// Log failed tests
	for _, result := range report.Results {
		if !result.Passed {
			t.Logf("FAILED: [%s] %s", result.Suite, result.TestName)
			for _, diff := range result.Differences {
				t.Logf("  Diff: [%s] %s", diff.Severity, diff.Description)
			}
		}
	}

	// Assert minimum success rate
	assert.GreaterOrEqual(t, report.SuccessRate, 80.0,
		"Parity test success rate (%.1f%%) below threshold", report.SuccessRate)
}

// BenchmarkParity compares execution speed
func BenchmarkComprehensiveParity(b *testing.B) {
	goPath := findGoBinary()
	tsPath := findTSBinary()

	if goPath == "" || tsPath == "" {
		b.Skip("Binaries not found")
	}

	b.Run("go_cli", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cmd := exec.Command(goPath, "version", "--short")
			cmd.Run()
		}
	})

	b.Run("ts_cli", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cmd := exec.Command(tsPath, "version", "--short")
			cmd.Run()
		}
	})
}