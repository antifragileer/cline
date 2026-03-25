// Package parity provides functional parity tests between Go and TypeScript CLIs.
// This package ensures both CLIs produce identical behavior.
package parity

import (
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

// CLIBinary represents a CLI binary to test
type CLIBinary struct {
	Name       string
	Path       string
	IsGo       bool
	WorkingDir string
}

// ParityTest represents a parity test case
type ParityTest struct {
	Name        string
	Args        []string
	Env         map[string]string
	Timeout     time.Duration
	SkipOutput  bool // Skip output comparison (for commands with timestamps/IDs)
}

// ParityResult represents the result of a parity comparison
type ParityResult struct {
	TestName     string
	Passed       bool
	GoOutput     string
	TSOutput     string
	GoExitCode   int
	TSExitCode   int
	Differences  []string
	Duration     time.Duration
}

// ParityReport contains all parity test results
type ParityReport struct {
	Success      bool
	Results      []ParityResult
	Summary      string
	ExitCode     int
	TotalTests   int
	PassedTests  int
	FailedTests  int
	Duration     time.Duration
}

// CLIParityTester performs parity tests between Go and TypeScript CLIs
type CLIParityTester struct {
	GoBinary *CLIBinary
	TSBinary *CLIBinary
	Results  []ParityResult
}

// NewCLIParityTester creates a new parity tester
func NewCLIParityTester(goPath, tsPath string) *CLIParityTester {
	return &CLIParityTester{
		GoBinary: &CLIBinary{
			Name: "Go CLI",
			Path: goPath,
			IsGo: true,
		},
		TSBinary: &CLIBinary{
			Name: "TypeScript CLI",
			Path: tsPath,
			IsGo: false,
		},
		Results: make([]ParityResult, 0),
	}
}

// RunTests runs all parity tests
func (p *CLIParityTester) RunTests(tests []ParityTest) (*ParityReport, error) {
	startTime := time.Now()
	p.Results = make([]ParityResult, 0)

	allPassed := true
	for _, test := range tests {
		result := p.runTest(test)
		p.Results = append(p.Results, result)
		if !result.Passed {
			allPassed = false
		}
	}

	duration := time.Since(startTime)
	passedCount := 0
	failedCount := 0
	for _, r := range p.Results {
		if r.Passed {
			passedCount++
		} else {
			failedCount++
		}
	}

	exitCode := 0
	summary := fmt.Sprintf("All %d parity tests passed", len(tests))
	if !allPassed {
		exitCode = 1
		summary = fmt.Sprintf("Parity tests failed: %d/%d passed", passedCount, len(tests))
	}

	return &ParityReport{
		Success:     allPassed,
		Results:     p.Results,
		Summary:     summary,
		ExitCode:    exitCode,
		TotalTests:  len(tests),
		PassedTests: passedCount,
		FailedTests: failedCount,
		Duration:    duration,
	}, nil
}

// runTest runs a single parity test
func (p *CLIParityTester) runTest(test ParityTest) ParityResult {
	result := ParityResult{
		TestName: test.Name,
		Passed:   false,
	}

	timeout := test.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	// Run Go CLI
	goStart := time.Now()
	goOutput, goExit := p.runBinary(p.GoBinary, test.Args, test.Env, timeout)
	result.GoOutput = goOutput
	result.GoExitCode = goExit
	goDuration := time.Since(goStart)

	// Run TypeScript CLI
	tsStart := time.Now()
	tsOutput, tsExit := p.runBinary(p.TSBinary, test.Args, test.Env, timeout)
	result.TSOutput = tsOutput
	result.TSExitCode = tsExit
	tsDuration := time.Since(tsStart)

	result.Duration = goDuration + tsDuration

	// Compare exit codes
	if goExit != tsExit {
		result.Differences = append(result.Differences,
			fmt.Sprintf("Exit codes differ: Go=%d, TS=%d", goExit, tsExit))
	}

	// Compare outputs (unless skipped)
	if !test.SkipOutput {
		diffs := compareOutputs(goOutput, tsOutput)
		result.Differences = append(result.Differences, diffs...)
	}

	// Test passes if no differences
	result.Passed = len(result.Differences) == 0

	return result
}

// runBinary executes a CLI binary with given arguments
func (p *CLIParityTester) runBinary(binary *CLIBinary, args []string, env map[string]string, timeout time.Duration) (string, int) {
	if binary.Path == "" {
		return "Binary not found", -1
	}

	cmd := exec.Command(binary.Path, args...)

	// Set environment
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	output, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	return string(output), exitCode
}

// compareOutputs compares two outputs and returns differences
func compareOutputs(goOutput, tsOutput string) []string {
	differences := make([]string, 0)

	// Normalize line endings
	goOutput = strings.ReplaceAll(goOutput, "\r\n", "\n")
	tsOutput = strings.ReplaceAll(tsOutput, "\r\n", "\n")

	goLines := strings.Split(goOutput, "\n")
	tsLines := strings.Split(tsOutput, "\n")

	// Compare line by line
	maxLines := len(goLines)
	if len(tsLines) > maxLines {
		maxLines = len(tsLines)
	}

	for i := 0; i < maxLines; i++ {
		goLine := ""
		tsLine := ""

		if i < len(goLines) {
			goLine = normalizeLine(goLines[i])
		}
		if i < len(tsLines) {
			tsLine = normalizeLine(tsLines[i])
		}

		if goLine != tsLine {
			differences = append(differences, fmt.Sprintf("Line %d differs:\n  Go: %s\n  TS: %s", i+1, goLine, tsLine))
		}
	}

	return differences
}

// normalizeLine normalizes a line for comparison (removes timestamps, IDs, etc.)
func normalizeLine(line string) string {
	// Remove timestamps (various formats)
	timestampPatterns := []string{
		`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})?`,
		`\d{2}:\d{2}:\d{2}`,
		`\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}`,
	}

	for _, pattern := range timestampPatterns {
		re := regexp.MustCompile(pattern)
		line = re.ReplaceAllString(line, "[TIMESTAMP]")
	}

	// Remove UUIDs
	uuidPattern := `[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`
	re := regexp.MustCompile(uuidPattern)
	line = re.ReplaceAllString(line, "[UUID]")

	// Remove temporary file paths
	line = strings.ReplaceAll(line, os.TempDir(), "[TEMPDIR]")

	return line
}

// StandardParityTests returns the standard set of parity tests
func StandardParityTests() []ParityTest {
	return []ParityTest{
		{
			Name:    "version command",
			Args:    []string{"version", "--short"},
			Timeout: 5 * time.Second,
		},
		{
			Name:    "help command",
			Args:    []string{"--help"},
			Timeout: 5 * time.Second,
		},
		{
			Name:    "config list",
			Args:    []string{"config", "list"},
			Timeout: 5 * time.Second,
		},
		{
			Name:    "history command",
			Args:    []string{"history", "--json"},
			Timeout: 5 * time.Second,
			SkipOutput: true, // May have different history state
		},
		{
			Name:    "version JSON",
			Args:    []string{"version", "--json"},
			Timeout: 5 * time.Second,
			SkipOutput: true, // Version info may differ
		},
	}
}

// ==================== Test Functions ====================

func TestCLIParity(t *testing.T) {
	// Skip if binaries not available
	goPath := findGoBinary()
	tsPath := findTSBinary()

	if goPath == "" {
		t.Skip("Go CLI binary not found")
	}
	if tsPath == "" {
		t.Skip("TypeScript CLI binary not found")
	}

	tester := NewCLIParityTester(goPath, tsPath)
	tests := StandardParityTests()

	report, err := tester.RunTests(tests)
	require.NoError(t, err)

	// Log results
	for _, result := range report.Results {
		status := "✓ PASS"
		if !result.Passed {
			status = "✗ FAIL"
		}
		t.Logf("[%s] %s", status, result.TestName)
		for _, diff := range result.Differences {
			t.Logf("  Diff: %s", diff)
		}
	}

	// Assert all tests passed
	assert.True(t, report.Success, "Some parity tests failed: %s", report.Summary)
}

func TestCommandStructureParity(t *testing.T) {
	goPath := findGoBinary()
	tsPath := findTSBinary()

	if goPath == "" || tsPath == "" {
		t.Skip("CLI binaries not found")
	}

	t.Run("commands match", func(t *testing.T) {
		goCmds := getCommands(goPath)
		tsCmds := getCommands(tsPath)

		// Compare command lists
		for _, cmd := range goCmds {
			assert.Contains(t, tsCmds, cmd, "Go command %s not found in TS CLI", cmd)
		}

		for _, cmd := range tsCmds {
			assert.Contains(t, goCmds, cmd, "TS command %s not found in Go CLI", cmd)
		}
	})
}

func TestFlagParity(t *testing.T) {
	goPath := findGoBinary()
	tsPath := findTSBinary()

	if goPath == "" || tsPath == "" {
		t.Skip("CLI binaries not found")
	}

	t.Run("root flags match", func(t *testing.T) {
		goFlags := getFlags(goPath, []string{"--help"})
		tsFlags := getFlags(tsPath, []string{"--help"})

		// Check that critical flags exist in both
		criticalFlags := []string{"--help", "--version", "--config"}
		for _, flag := range criticalFlags {
			goHas := contains(goFlags, flag)
			tsHas := contains(tsFlags, flag)
			assert.Equal(t, goHas, tsHas, "Flag %s availability differs", flag)
		}
	})
}

// ==================== Helper Functions ====================

func findGoBinary() string {
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

func findTSBinary() string {
	// Look for TypeScript CLI in parent directory
	locations := []string{
		filepath.Join("..", "..", "..", "cli", "bin", "cline"),
		filepath.Join("..", "..", "..", "cli", "dist", "index.js"),
		filepath.Join("..", "..", "cli", "bin", "cline"),
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

func getCommands(binary string) []string {
	cmd := exec.Command(binary, "--help")
	output, _ := cmd.CombinedOutput()
	
	// Parse commands from help text
	// This is a simplified parser
	commands := make([]string, 0)
	lines := strings.Split(string(output), "\n")
	inCommands := false
	
	for _, line := range lines {
		if strings.Contains(line, "Available Commands:") {
			inCommands = true
			continue
		}
		if inCommands && strings.TrimSpace(line) == "" {
			break
		}
		if inCommands {
			parts := strings.Fields(line)
			if len(parts) > 0 && !strings.HasPrefix(parts[0], "-") {
				commands = append(commands, parts[0])
			}
		}
	}
	
	return commands
}

func getFlags(binary string, args []string) []string {
	cmd := exec.Command(binary, args...)
	output, _ := cmd.CombinedOutput()
	
	flags := make([]string, 0)
	lines := strings.Split(string(output), "\n")
	
	for _, line := range lines {
		if strings.Contains(line, "--") {
			parts := strings.Fields(line)
			for _, part := range parts {
				if strings.HasPrefix(part, "--") || strings.HasPrefix(part, "-") {
					flags = append(flags, part)
				}
			}
		}
	}
	
	return flags
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// BenchmarkParity benchmarks command execution speed
func BenchmarkGoCLI(b *testing.B) {
	goPath := findGoBinary()
	if goPath == "" {
		b.Skip("Go CLI not found")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := exec.Command(goPath, "version", "--short")
		cmd.Run()
	}
}

func BenchmarkTSCLI(b *testing.B) {
	tsPath := findTSBinary()
	if tsPath == "" {
		b.Skip("TS CLI not found")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := exec.Command(tsPath, "version", "--short")
		cmd.Run()
	}
}