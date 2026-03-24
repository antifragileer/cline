// Package tests provides a unified test runner for all verification tests.
// This package can be used to run smoke tests, independence checks, cross-platform builds,
// and CI/CD pipeline tests from a single entry point.
package tests

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// TestRunnerConfig holds configuration for the test runner
type TestRunnerConfig struct {
	// Test types to run
	SmokeTests       bool
	IndependenceCheck bool
	CrossPlatform    bool
	CICDPipeline     bool
	All              bool

	// Paths
	ProjectRoot string
	BinaryPath  string
	BuildDir    string

	// Output
	Verbose    bool
	JSONOutput bool
}

// ParseFlags parses command-line flags and returns the configuration
func ParseFlags() *TestRunnerConfig {
	config := &TestRunnerConfig{}

	// Test type flags
	flag.BoolVar(&config.SmokeTests, "smoke", false, "Run smoke tests")
	flag.BoolVar(&config.IndependenceCheck, "independence", false, "Run independence verification")
	flag.BoolVar(&config.CrossPlatform, "cross-platform", false, "Run cross-platform builds")
	flag.BoolVar(&config.CICDPipeline, "ci-cd", false, "Run CI/CD pipeline tests")
	flag.BoolVar(&config.All, "all", false, "Run all tests")

	// Path flags
	flag.StringVar(&config.ProjectRoot, "project", "", "Path to project root (auto-detected if not provided)")
	flag.StringVar(&config.BinaryPath, "binary", "", "Path to binary (auto-detected if not provided)")
	flag.StringVar(&config.BuildDir, "build-dir", "", "Build directory for cross-platform builds")

	// Output flags
	flag.BoolVar(&config.Verbose, "verbose", false, "Enable verbose output")
	flag.BoolVar(&config.JSONOutput, "json", false, "Output results in JSON format")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Test Runner for Cline CLI - runs smoke tests, independence checks,\n")
		fmt.Fprintf(os.Stderr, "cross-platform builds, and CI/CD pipeline tests.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Run smoke tests\n")
		fmt.Fprintf(os.Stderr, "  %s -smoke\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Run independence check\n")
		fmt.Fprintf(os.Stderr, "  %s -independence\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Run all tests\n")
		fmt.Fprintf(os.Stderr, "  %s -all\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Run smoke tests with JSON output\n")
		fmt.Fprintf(os.Stderr, "  %s -smoke -json\n\n", os.Args[0])
	}

	flag.Parse()

	// If no specific test type is selected, default to all
	if !config.SmokeTests && !config.IndependenceCheck && !config.CrossPlatform && !config.CICDPipeline && !config.All {
		config.All = true
	}

	// If all is selected, enable all test types
	if config.All {
		config.SmokeTests = true
		config.IndependenceCheck = true
		config.CrossPlatform = true
		config.CICDPipeline = true
	}

	return config
}

// RunTests runs the selected tests based on configuration
func RunTests(config *TestRunnerConfig) int {
	// Auto-detect project root if not provided
	if config.ProjectRoot == "" {
		var err error
		config.ProjectRoot, err = detectProjectRoot()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Could not detect project root: %v\n", err)
			return 1
		}
	}

	// Auto-detect binary if not provided
	if config.BinaryPath == "" {
		config.BinaryPath = detectBinary(config.ProjectRoot)
	}

	// Print header
	if !config.JSONOutput {
		printHeader(config)
	}

	exitCode := 0

	// Run smoke tests
	if config.SmokeTests {
		if !config.JSONOutput {
			fmt.Println("\n" + strings.Repeat("=", 70))
			fmt.Println("Running Smoke Tests")
			fmt.Println(strings.Repeat("=", 70))
		}

		code := RunSmokeTestsCommand(config.BinaryPath, config.Verbose, config.JSONOutput)
		if code != 0 {
			exitCode = code
		}
	}

	// Run independence check
	if config.IndependenceCheck {
		if !config.JSONOutput {
			fmt.Println("\n" + strings.Repeat("=", 70))
			fmt.Println("Running Independence Check")
			fmt.Println(strings.Repeat("=", 70))
		}

		verifier := NewIndependenceVerifier(config.ProjectRoot, config.BinaryPath, config.Verbose)
		report, err := verifier.Verify()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Independence verification error: %v\n", err)
			exitCode = 1
		} else {
			if config.JSONOutput {
				// JSON output is handled by the report
				data, _ := report.ToJSON()
				fmt.Println(data)
			} else {
				printVerificationReport(report)
			}

			if !report.Success {
				exitCode = 1
			}
		}
	}

	// Run cross-platform builds
	if config.CrossPlatform {
		if !config.JSONOutput {
			fmt.Println("\n" + strings.Repeat("=", 70))
			fmt.Println("Running Cross-Platform Builds")
			fmt.Println(strings.Repeat("=", 70))
		}

		buildDir := config.BuildDir
		if buildDir == "" {
			buildDir = filepath.Join(config.ProjectRoot, "build", "cross-platform")
		}

		code := RunCrossPlatformBuildsCommand(config.ProjectRoot, buildDir, config.Verbose)
		if code != 0 {
			exitCode = code
		}
	}

	// Run CI/CD pipeline tests
	if config.CICDPipeline {
		if !config.JSONOutput {
			fmt.Println("\n" + strings.Repeat("=", 70))
			fmt.Println("Running CI/CD Pipeline Tests")
			fmt.Println(strings.Repeat("=", 70))
		}

		code := RunCICDPipelineCommand(config.ProjectRoot, config.BinaryPath, config.JSONOutput)
		if code != 0 {
			exitCode = code
		}
	}

	// Print footer
	if !config.JSONOutput {
		printFooter(exitCode)
	}

	return exitCode
}

// detectProjectRoot attempts to detect the project root
func detectProjectRoot() (string, error) {
	// Try current directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Check if go.mod exists in current directory
	if _, err := os.Stat(filepath.Join(cwd, "go.mod")); err == nil {
		return cwd, nil
	}

	// Check parent directory (in case we're in tests/)
	parent := filepath.Dir(cwd)
	if _, err := os.Stat(filepath.Join(parent, "go.mod")); err == nil {
		return parent, nil
	}

	return "", fmt.Errorf("go.mod not found in current or parent directory")
}

// detectBinary attempts to detect the binary path
func detectBinary(projectRoot string) string {
	binaryName := "cline"
	if os.Getenv("GOOS") == "windows" || isWindows() {
		binaryName = "cline.exe"
	}

	// Common locations
	locations := []string{
		filepath.Join(projectRoot, binaryName),
		filepath.Join(projectRoot, "cmd", "cline", binaryName),
		filepath.Join(projectRoot, "bin", binaryName),
		filepath.Join(projectRoot, "dist", binaryName),
		filepath.Join(projectRoot, "build", binaryName),
	}

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return loc
		}
	}

	// Try to find in PATH
	if path, err := exec.LookPath(binaryName); err == nil {
		return path
	}

	return ""
}

// isWindows checks if running on Windows
func isWindows() bool {
	return os.PathSeparator == '\\' || os.Getenv("OS") == "Windows_NT"
}

// printHeader prints the test runner header
func printHeader(config *TestRunnerConfig) {
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println("  Cline CLI Test Runner")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Printf("Project Root: %s\n", config.ProjectRoot)
	fmt.Printf("Binary Path:  %s\n", config.BinaryPath)
	fmt.Println()
	fmt.Println("Selected Tests:")
	if config.SmokeTests {
		fmt.Println("  ✓ Smoke Tests")
	}
	if config.IndependenceCheck {
		fmt.Println("  ✓ Independence Check")
	}
	if config.CrossPlatform {
		fmt.Println("  ✓ Cross-Platform Builds")
	}
	if config.CICDPipeline {
		fmt.Println("  ✓ CI/CD Pipeline Tests")
	}
}

// printFooter prints the test runner footer
func printFooter(exitCode int) {
	fmt.Println("\n" + strings.Repeat("=", 70))
	if exitCode == 0 {
		fmt.Println("  All tests PASSED")
	} else {
		fmt.Println("  Some tests FAILED")
	}
	fmt.Println(strings.Repeat("=", 70))
	fmt.Printf("Exit Code: %d\n", exitCode)
}

// printVerificationReport prints a human-readable verification report
func printVerificationReport(report *VerificationReport) {
	fmt.Println("============================================================")
	fmt.Println("  Independence Verification Report")
	fmt.Println("============================================================")
	fmt.Println()

	for _, result := range report.Results {
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

	fmt.Println("------------------------------------------------------------")
	if report.Success {
		fmt.Printf("Summary: %s\n", report.Summary)
	} else {
		fmt.Printf("Summary: %s\n", report.Summary)
	}
	fmt.Printf("Exit Code: %d\n", report.ExitCode)
	fmt.Println("------------------------------------------------------------")
}

// ToJSON returns the verification report as JSON
func (r *VerificationReport) ToJSON() (string, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}