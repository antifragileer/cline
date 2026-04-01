// Package parity provides functional parity tests between Go and TypeScript CLIs.
// This package ensures both CLIs produce identical behavior.
package parity

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
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
// These tests compare commands that exist in both CLIs with compatible interfaces
// IMPORTANT: Only use commands that complete without user interaction
func StandardParityTests() []ParityTest {
	return []ParityTest{
		{
			Name:       "version command",
			Args:       []string{"version"},
			Timeout:    10 * time.Second,
			SkipOutput: true, // Version output format differs (expected)
		},
		{
			Name:       "help command",
			Args:       []string{"--help"},
			Timeout:    10 * time.Second,
			SkipOutput: true, // Help output format differs between implementations
		},
		{
			Name:       "mcp command help",
			Args:       []string{"mcp", "--help"},
			Timeout:    10 * time.Second,
			SkipOutput: true, // Help output format differs
		},
		{
			Name:       "auth command help",
			Args:       []string{"auth", "--help"},
			Timeout:    10 * time.Second,
			SkipOutput: true, // Help output format differs
		},
		{
			Name:       "history command help",
			Args:       []string{"history", "--help"},
			Timeout:    10 * time.Second,
			SkipOutput: true, // Help output format differs
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

	// These are the commands that exist in both CLIs
	// Note: Go CLI has 'completion' and 'help' which TypeScript doesn't expose as commands
	// TypeScript CLI doesn't have 'completion' command
	expectedCommonCommands := []string{
		"task",
		"history",
		"config",
		"auth",
		"version",
		"update",
		"dev",
	}

	t.Run("common commands exist", func(t *testing.T) {
		goCmds := getCommands(goPath)
		tsCmds := getCommands(tsPath)

		// Verify all expected common commands exist in both CLIs
		for _, cmd := range expectedCommonCommands {
			assert.Contains(t, goCmds, cmd, "Go CLI missing expected command: %s", cmd)
			assert.Contains(t, tsCmds, cmd, "TypeScript CLI missing expected command: %s", cmd)
		}
	})

	t.Run("go specific commands", func(t *testing.T) {
		goCmds := getCommands(goPath)
		
		// These commands only exist in Go CLI
		goSpecific := []string{"completion", "help"}
		for _, cmd := range goSpecific {
			assert.Contains(t, goCmds, cmd, "Go CLI should have %s command", cmd)
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

// Note: findGoBinary() and findTSBinary() are defined in comprehensive_test.go
// in the same package, so they are available for use here.

func getCommands(binary string) []string {
	cmd := exec.Command(binary, "--help")
	output, _ := cmd.CombinedOutput()
	
	// Parse commands from help text
	// Supports both formats:
	// - Go CLI: "Available Commands:" followed by "  cmdname    description"
	// - TypeScript CLI: "Commands:" followed by "  cmd|alias [options]    description"
	commands := make([]string, 0)
	lines := strings.Split(string(output), "\n")
	inCommands := false
	
	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		
		// Detect start of commands section (both formats)
		if strings.Contains(line, "Available Commands:") || strings.Contains(line, "Commands:") {
			inCommands = true
			continue
		}
		
		if !inCommands {
			continue
		}
		
		// End of commands section detection
		// Go CLI has "Flags:" after commands, TypeScript CLI just ends
		if strings.HasSuffix(trimmedLine, ":") && !strings.Contains(line, "  ") {
			// This is a new section header like "Flags:", "Options:", "Usage:", "Arguments:"
			break
		}
		
		// Skip empty lines
		if trimmedLine == "" {
			continue
		}
		
		// Parse command line
		// Commands are indented (start with spaces)
		if !strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "\t") {
			// Not indented - might be continuation of previous description or end of section
			continue
		}
		
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}
		
		// Skip flag lines (start with -)
		if strings.HasPrefix(parts[0], "-") {
			continue
		}
		
		// Extract command name
		cmdName := parts[0]
		
		// Handle aliases: "task|t" -> "task"
		if idx := strings.Index(cmdName, "|"); idx != -1 {
			cmdName = cmdName[:idx]
		}
		
		// Remove bracketed parts: "[options]", "<prompt>", etc.
		cmdName = strings.TrimSuffix(cmdName, "[options]")
		cmdName = strings.TrimSuffix(cmdName, "[option]")
		cmdName = strings.TrimSuffix(cmdName, "<prompt>")
		cmdName = strings.Trim(cmdName, "[]<>")
		
		// Validate: command names are lowercase, single word, no spaces or special chars
		if cmdName == "" || strings.ContainsAny(cmdName, " \t\n\r") {
			continue
		}
		
		// Check it looks like a command name (lowercase letters, maybe digits, hyphens)
		if !regexp.MustCompile(`^[a-z][a-z0-9-]*$`).MatchString(cmdName) {
			continue
		}
		
		// Check next line - if it's more indented, this was a command with wrapped description
		// If less or same indentation, we've moved on
		if i+1 < len(lines) {
			nextLine := lines[i+1]
			nextTrimmed := strings.TrimSpace(nextLine)
			if nextTrimmed != "" && !strings.HasPrefix(nextLine, "  ") && !strings.HasPrefix(nextLine, "\t") {
				// Next line not indented, might be end of section
				if !strings.HasPrefix(nextTrimmed, "-") && !strings.HasSuffix(nextTrimmed, ":") {
					// Could be a continuation, skip
				}
			}
		}
		
		// Avoid duplicates
		alreadyHave := false
		for _, existing := range commands {
			if existing == cmdName {
				alreadyHave = true
				break
			}
		}
		
		if !alreadyHave {
			commands = append(commands, cmdName)
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