// Package tests provides cross-platform build verification for the Go CLI.
// This package ensures the Go CLI binary can be built for multiple platforms.
package tests

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// CrossPlatformTarget represents a build target platform
type CrossPlatformTarget struct {
	GOOS   string
	GOARCH string
}

// CrossPlatformBuildResult represents the result of a cross-platform build
type CrossPlatformBuildResult struct {
	Target  CrossPlatformTarget `json:"target"`
	Success bool                `json:"success"`
	Message string              `json:"message"`
	Output  string              `json:"output,omitempty"`
}

// CrossPlatformBuildReport contains all cross-platform build results
type CrossPlatformBuildReport struct {
	Success   bool                     `json:"success"`
	Results   []CrossPlatformBuildResult `json:"results"`
	Summary   string                   `json:"summary"`
	ExitCode  int                      `json:"exitCode"`
	BuildDir  string                   `json:"buildDir"`
}

// Standard cross-platform targets to test
var StandardTargets = []CrossPlatformTarget{
	{GOOS: "linux", GOARCH: "amd64"},
	{GOOS: "linux", GOARCH: "arm64"},
	{GOOS: "darwin", GOARCH: "amd64"},
	{GOOS: "darwin", GOARCH: "arm64"},
	{GOOS: "windows", GOARCH: "amd64"},
}

// CrossPlatformBuilder performs cross-platform builds
type CrossPlatformBuilder struct {
	ProjectRoot string
	BuildDir    string
	Verbose     bool
	Results     []CrossPlatformBuildResult
}

// NewCrossPlatformBuilder creates a new builder instance
func NewCrossPlatformBuilder(projectRoot, buildDir string, verbose bool) *CrossPlatformBuilder {
	return &CrossPlatformBuilder{
		ProjectRoot: projectRoot,
		BuildDir:    buildDir,
		Verbose:     verbose,
		Results:     make([]CrossPlatformBuildResult, 0),
	}
}

// BuildAll builds for all standard targets
func (b *CrossPlatformBuilder) BuildAll() (*CrossPlatformBuildReport, error) {
	b.Results = make([]CrossPlatformBuildResult, 0)

	// Ensure build directory exists
	if err := os.MkdirAll(b.BuildDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create build directory: %w", err)
	}

	allPassed := true
	for _, target := range StandardTargets {
		result := b.buildTarget(target)
		b.Results = append(b.Results, result)
		if !result.Success {
			allPassed = false
		}
	}

	exitCode := 0
	summary := "All cross-platform builds succeeded"
	if !allPassed {
		exitCode = 1
		summary = "Some cross-platform builds failed"
	}

	report := &CrossPlatformBuildReport{
		Success:  allPassed,
		Results:  b.Results,
		Summary:  summary,
		ExitCode: exitCode,
		BuildDir: b.BuildDir,
	}

	return report, nil
}

// BuildTarget builds for a specific target
func (b *CrossPlatformBuilder) BuildTarget(target CrossPlatformTarget) (*CrossPlatformBuildReport, error) {
	b.Results = make([]CrossPlatformBuildResult, 0)

	// Ensure build directory exists
	if err := os.MkdirAll(b.BuildDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create build directory: %w", err)
	}

	result := b.buildTarget(target)
	b.Results = append(b.Results, result)

	exitCode := 0
	summary := "Cross-platform build succeeded"
	if !result.Success {
		exitCode = 1
		summary = "Cross-platform build failed"
	}

	report := &CrossPlatformBuildReport{
		Success:  result.Success,
		Results:  b.Results,
		Summary:  summary,
		ExitCode: exitCode,
		BuildDir: b.BuildDir,
	}

	return report, nil
}

// buildTarget builds for a single target
func (b *CrossPlatformBuilder) buildTarget(target CrossPlatformTarget) CrossPlatformBuildResult {
	result := CrossPlatformBuildResult{
		Target:  target,
		Success: false,
	}

	outputPath := filepath.Join(b.BuildDir, fmt.Sprintf("cline-%s-%s", target.GOOS, target.GOARCH))
	if target.GOOS == "windows" {
		outputPath += ".exe"
	}

	// Set up environment
	env := os.Environ()
	env = setEnvVar(env, "GOOS", target.GOOS)
	env = setEnvVar(env, "GOARCH", target.GOARCH)
	env = setEnvVar(env, "CGO_ENABLED", "0")

	// Build command
	cmd := exec.Command("go", "build", "-ldflags", "-s -w", "-o", outputPath, "./cmd/cline")
	cmd.Dir = b.ProjectRoot
	cmd.Env = env

	// Capture output
	output, err := cmd.CombinedOutput()
	result.Output = string(output)

	if err != nil {
		result.Message = fmt.Sprintf("Build failed for %s/%s", target.GOOS, target.GOARCH)
		return result
	}

	// Verify binary was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		result.Message = fmt.Sprintf("Binary not created for %s/%s", target.GOOS, target.GOARCH)
		return result
	}

	// Verify binary is not empty
	info, err := os.Stat(outputPath)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to stat binary for %s/%s", target.GOOS, target.GOARCH)
		return result
	}

	if info.Size() == 0 {
		result.Message = fmt.Sprintf("Binary is empty for %s/%s", target.GOOS, target.GOARCH)
		return result
	}

	result.Success = true
	result.Message = fmt.Sprintf("Build successful for %s/%s (%.2f MB)", target.GOOS, target.GOARCH, float64(info.Size())/(1024*1024))
	return result
}

// setEnvVar sets or replaces an environment variable
func setEnvVar(env []string, key, value string) []string {
	prefix := key + "="
	for i, e := range env {
		if strings.HasPrefix(e, prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

// PrintReport prints a human-readable cross-platform build report
func (r *CrossPlatformBuildReport) PrintReport() {
	fmt.Println("=" + strings.Repeat("=", 70))
	fmt.Println("  Cross-Platform Build Report")
	fmt.Println("=" + strings.Repeat("=", 70))
	fmt.Printf("Build Directory: %s\n", r.BuildDir)
	fmt.Println()

	for _, result := range r.Results {
		status := "✓ PASS"
		if !result.Success {
			status = "✗ FAIL"
		}
		fmt.Printf("[%s] %s/%s\n", status, result.Target.GOOS, result.Target.GOARCH)
		fmt.Printf("       %s\n", result.Message)
		if result.Output != "" && !result.Success {
			// Show last few lines of output on failure
			lines := strings.Split(result.Output, "\n")
			start := len(lines) - 5
			if start < 0 {
				start = 0
			}
			fmt.Printf("       Output: %s\n", strings.Join(lines[start:], "\n"))
		}
		fmt.Println()
	}

	fmt.Println(strings.Repeat("-", 72))
	if r.Success {
		fmt.Printf("Summary: %s\n", r.Summary)
	} else {
		fmt.Printf("Summary: %s\n", r.Summary)
	}
	fmt.Printf("Exit Code: %d\n", r.ExitCode)
}

// RunCrossPlatformBuildsCommand executes cross-platform builds as a CLI command
func RunCrossPlatformBuildsCommand(projectRoot, buildDir string, verbose bool) int {
	// Auto-detect project root if not provided
	if projectRoot == "" {
		var err error
		projectRoot, err = os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting working directory: %v\n", err)
			return 1
		}
	}

	// Verify project root is valid
	if _, err := os.Stat(filepath.Join(projectRoot, "go.mod")); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: go.mod not found in %s\n", projectRoot)
		return 1
	}

	// Set default build directory
	if buildDir == "" {
		buildDir = filepath.Join(projectRoot, "build", "cross-platform")
	}

	builder := NewCrossPlatformBuilder(projectRoot, buildDir, verbose)
	report, err := builder.BuildAll()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cross-platform build error: %v\n", err)
		return 1
	}

	report.PrintReport()

	return report.ExitCode
}

// ==================== Unit Tests ====================

func TestNewCrossPlatformBuilder(t *testing.T) {
	t.Run("creates builder with correct settings", func(t *testing.T) {
		builder := NewCrossPlatformBuilder("/project", "/build", true)

		if builder.ProjectRoot != "/project" {
			t.Errorf("ProjectRoot = %s, want /project", builder.ProjectRoot)
		}
		if builder.BuildDir != "/build" {
			t.Errorf("BuildDir = %s, want /build", builder.BuildDir)
		}
		if !builder.Verbose {
			t.Error("Verbose should be true")
		}
		if builder.Results == nil {
			t.Error("Results should be initialized")
		}
	})
}

func TestCrossPlatformBuilder_buildTarget(t *testing.T) {
	// Skip if not in a Go module
	projectRoot := os.Getenv("PROJECT_ROOT")
	if projectRoot == "" {
		// Try to find project root
		cwd, err := os.Getwd()
		if err != nil {
			t.Skip("Cannot determine working directory")
		}
		// Check if we're in the tests directory
		if strings.HasSuffix(cwd, "/tests") {
			projectRoot = filepath.Dir(cwd)
		} else {
			projectRoot = cwd
		}
	}

	// Verify go.mod exists
	if _, err := os.Stat(filepath.Join(projectRoot, "go.mod")); os.IsNotExist(err) {
		t.Skip("Not in a Go module directory, skipping build test")
	}

	tempDir, err := os.MkdirTemp("", "cross-platform-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	builder := NewCrossPlatformBuilder(projectRoot, tempDir, false)

	t.Run("builds for current platform", func(t *testing.T) {
		target := CrossPlatformTarget{
			GOOS:   runtime.GOOS,
			GOARCH: runtime.GOARCH,
		}

		result := builder.buildTarget(target)

		if !result.Success {
			t.Errorf("Expected build to succeed, got: %s\nOutput: %s", result.Message, result.Output)
		}

		// Verify binary exists
		expectedBinary := filepath.Join(tempDir, fmt.Sprintf("cline-%s-%s", target.GOOS, target.GOARCH))
		if target.GOOS == "windows" {
			expectedBinary += ".exe"
		}

		if _, err := os.Stat(expectedBinary); os.IsNotExist(err) {
			t.Errorf("Expected binary at %s does not exist", expectedBinary)
		}
	})

	t.Run("fails for invalid target", func(t *testing.T) {
		target := CrossPlatformTarget{
			GOOS:   "invalid",
			GOARCH: "invalid",
		}

		result := builder.buildTarget(target)

		if result.Success {
			t.Error("Expected build to fail for invalid target")
		}
	})
}

func TestSetEnvVar(t *testing.T) {
	t.Run("adds new variable", func(t *testing.T) {
		env := []string{"PATH=/usr/bin", "HOME=/home/user"}
		env = setEnvVar(env, "NEW_VAR", "value")

		found := false
		for _, e := range env {
			if e == "NEW_VAR=value" {
				found = true
				break
			}
		}

		if !found {
			t.Error("NEW_VAR not found in environment")
		}
	})

	t.Run("replaces existing variable", func(t *testing.T) {
		env := []string{"PATH=/usr/bin", "HOME=/home/user"}
		env = setEnvVar(env, "PATH", "/new/path")

		found := false
		for _, e := range env {
			if e == "PATH=/new/path" {
				found = true
				break
			}
			if e == "PATH=/usr/bin" {
				t.Error("Old PATH value still exists")
			}
		}

		if !found {
			t.Error("New PATH value not found")
		}
	})
}

func TestStandardTargets(t *testing.T) {
	t.Run("has expected targets", func(t *testing.T) {
		expectedTargets := map[string]bool{
			"linux/amd64":   false,
			"linux/arm64":   false,
			"darwin/amd64":  false,
			"darwin/arm64":  false,
			"windows/amd64": false,
		}

		for _, target := range StandardTargets {
			key := fmt.Sprintf("%s/%s", target.GOOS, target.GOARCH)
			if _, exists := expectedTargets[key]; exists {
				expectedTargets[key] = true
			}
		}

		for key, found := range expectedTargets {
			if !found {
				t.Errorf("Expected target %s not found", key)
			}
		}
	})
}

func TestCrossPlatformBuildReport(t *testing.T) {
	t.Run("report formatting", func(t *testing.T) {
		report := &CrossPlatformBuildReport{
			Success:  true,
			BuildDir: "/build",
			Results: []CrossPlatformBuildResult{
				{
					Target:  CrossPlatformTarget{GOOS: "linux", GOARCH: "amd64"},
					Success: true,
					Message: "Build successful",
				},
			},
			Summary:  "All builds succeeded",
			ExitCode: 0,
		}

		// Just verify it doesn't panic
		report.PrintReport()
	})
}