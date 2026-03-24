// Package tests provides CI/CD pipeline tests for the Go CLI.
// This package ensures the Go CLI can be built and verified in CI/CD environments.
package tests

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// CICDStage represents a CI/CD pipeline stage
type CICDStage struct {
	Name     string        `json:"name"`
	Duration time.Duration `json:"duration"`
	Success  bool          `json:"success"`
	Message  string        `json:"message"`
}

// CICDPipelineReport contains the results of CI/CD pipeline tests
type CICDPipelineReport struct {
	Success    bool        `json:"success"`
	Stages     []CICDStage `json:"stages"`
	Summary    string      `json:"summary"`
	ExitCode   int         `json:"exitCode"`
	Duration   time.Duration `json:"totalDuration"`
	BuildInfo  BuildInfo   `json:"buildInfo"`
}

// BuildInfo contains build metadata
type BuildInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	Branch    string `json:"branch"`
	BuildDate string `json:"buildDate"`
	GoVersion string `json:"goVersion"`
}

// CICDTester performs CI/CD pipeline tests
type CICDTester struct {
	ProjectRoot string
	BinaryPath  string
	Stages      []CICDStage
}

// NewCICDTester creates a new CI/CD tester instance
func NewCICDTester(projectRoot, binaryPath string) *CICDTester {
	return &CICDTester{
		ProjectRoot: projectRoot,
		BinaryPath:  binaryPath,
		Stages:      make([]CICDStage, 0),
	}
}

// RunAllStages runs all CI/CD pipeline stages
func (c *CICDTester) RunAllStages() (*CICDPipelineReport, error) {
	c.Stages = make([]CICDStage, 0)
	startTime := time.Now()

	// Define all stages
	stages := []struct {
		name string
		fn   func() (bool, string)
	}{
		{"build", c.stageBuild},
		{"unit_tests", c.stageUnitTests},
		{"smoke_tests", c.stageSmokeTests},
		{"independence_check", c.stageIndependenceCheck},
		{"cross_platform_build", c.stageCrossPlatformBuild},
		{"binary_size_check", c.stageBinarySizeCheck},
	}

	allPassed := true
	for _, stage := range stages {
		stageStart := time.Now()
		success, message := stage.fn()
		duration := time.Since(stageStart)

		c.Stages = append(c.Stages, CICDStage{
			Name:     stage.name,
			Duration: duration,
			Success:    success,
			Message:  message,
		})

		if !success {
			allPassed = false
		}
	}

	totalDuration := time.Since(startTime)

	exitCode := 0
	summary := "All CI/CD pipeline stages passed"
	if !allPassed {
		exitCode = 1
		summary = "Some CI/CD pipeline stages failed"
	}

	// Get build info
	buildInfo := c.getBuildInfo()

	report := &CICDPipelineReport{
		Success:   allPassed,
		Stages:    c.Stages,
		Summary:   summary,
		ExitCode:  exitCode,
		Duration:  totalDuration,
		BuildInfo: buildInfo,
	}

	return report, nil
}

// stageBuild tests the build stage
func (c *CICDTester) stageBuild() (bool, string) {
	// Check if binary exists or build it
	if c.BinaryPath != "" {
		if _, err := os.Stat(c.BinaryPath); err == nil {
			return true, "Binary already exists"
		}
	}

	// Build the binary
	cmd := exec.Command("go", "build", "-o", "cline", "./cmd/cline")
	cmd.Dir = c.ProjectRoot
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Sprintf("Build failed: %v\nOutput: %s", err, string(output))
	}

	// Update binary path
	c.BinaryPath = filepath.Join(c.ProjectRoot, "cline")
	return true, "Build successful"
}

// stageUnitTests runs unit tests
func (c *CICDTester) stageUnitTests() (bool, string) {
	cmd := exec.Command("go", "test", "./...", "-v", "-race", "-count=1")
	cmd.Dir = c.ProjectRoot
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if it's just test failures vs execution error
		outputStr := string(output)
		if strings.Contains(outputStr, "FAIL") {
			return false, fmt.Sprintf("Unit tests failed\nOutput: %s", outputStr)
		}
		return false, fmt.Sprintf("Unit test execution failed: %v\nOutput: %s", err, outputStr)
	}

	// Count tests passed
	outputStr := string(output)
	passCount := strings.Count(outputStr, "PASS")
	return true, fmt.Sprintf("Unit tests passed (%d packages)", passCount)
}

// stageSmokeTests runs smoke tests
func (c *CICDTester) stageSmokeTests() (bool, string) {
	if c.BinaryPath == "" {
		return false, "Binary path not set"
	}

	tester := NewSmokeTester(c.BinaryPath, false)
	report, err := tester.Run()
	if err != nil {
		return false, fmt.Sprintf("Smoke test error: %v", err)
	}

	if !report.Success {
		return false, fmt.Sprintf("Smoke tests failed: %s", report.Summary)
	}

	return true, fmt.Sprintf("Smoke tests passed (%d/%d)", len(report.Results), len(report.Results))
}

// stageIndependenceCheck runs independence verification
func (c *CICDTester) stageIndependenceCheck() (bool, string) {
	verifier := NewIndependenceVerifier(c.ProjectRoot, c.BinaryPath, false)
	report, err := verifier.Verify()
	if err != nil {
		return false, fmt.Sprintf("Independence check error: %v", err)
	}

	if !report.Success {
		failedCount := 0
		for _, result := range report.Results {
			if !result.Passed {
				failedCount++
			}
		}
		return false, fmt.Sprintf("Independence check failed (%d checks failed)", failedCount)
	}

	return true, fmt.Sprintf("Independence check passed (%d/%d checks)", len(report.Results), len(report.Results))
}

// stageCrossPlatformBuild tests cross-platform builds
func (c *CICDTester) stageCrossPlatformBuild() (bool, string) {
	buildDir := filepath.Join(c.ProjectRoot, "build", "ci-cd")
	builder := NewCrossPlatformBuilder(c.ProjectRoot, buildDir, false)
	
	report, err := builder.BuildAll()
	if err != nil {
		return false, fmt.Sprintf("Cross-platform build error: %v", err)
	}

	if !report.Success {
		failedCount := 0
		for _, result := range report.Results {
			if !result.Success {
				failedCount++
			}
		}
		return false, fmt.Sprintf("Cross-platform builds failed (%d/%d)", failedCount, len(report.Results))
	}

	return true, fmt.Sprintf("Cross-platform builds passed (%d platforms)", len(report.Results))
}

// stageBinarySizeCheck verifies binary size
func (c *CICDTester) stageBinarySizeCheck() (bool, string) {
	if c.BinaryPath == "" {
		return false, "Binary path not set"
	}

	info, err := os.Stat(c.BinaryPath)
	if err != nil {
		return false, fmt.Sprintf("Failed to stat binary: %v", err)
	}

	sizeMB := float64(info.Size()) / (1024 * 1024)
	if sizeMB > BinarySizeLimit {
		return false, fmt.Sprintf("Binary size %.2f MB exceeds limit %d MB", sizeMB, BinarySizeLimit)
	}

	return true, fmt.Sprintf("Binary size OK (%.2f MB)", sizeMB)
}

// getBuildInfo retrieves build information
func (c *CICDTester) getBuildInfo() BuildInfo {
	// Try to get info from binary
	if c.BinaryPath != "" {
		cmd := exec.Command(c.BinaryPath, "version", "--json")
		output, err := cmd.CombinedOutput()
		if err == nil {
			var info map[string]interface{}
			if err := json.Unmarshal(output, &info); err == nil {
				return BuildInfo{
					Version:   getString(info, "version"),
					Commit:    getString(info, "gitCommit"),
					Branch:    getString(info, "gitBranch"),
					BuildDate: getString(info, "buildDate"),
					GoVersion: getString(info, "goVersion"),
				}
			}
		}
	}

	// Fallback to defaults
	return BuildInfo{
		Version:   "unknown",
		Commit:    "unknown",
		Branch:    "unknown",
		BuildDate: time.Now().Format(time.RFC3339),
		GoVersion: "unknown",
	}
}

// getString safely gets a string from a map
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return "unknown"
}

// PrintReport prints a human-readable CI/CD pipeline report
func (r *CICDPipelineReport) PrintReport() {
	fmt.Println("=" + strings.Repeat("=", 70))
	fmt.Println("  CI/CD Pipeline Test Report")
	fmt.Println("=" + strings.Repeat("=", 70))
	fmt.Printf("Total Duration: %v\n", r.Duration)
	fmt.Printf("Build Version: %s\n", r.BuildInfo.Version)
	fmt.Printf("Git Commit: %s\n", r.BuildInfo.Commit)
	fmt.Println()

	for _, stage := range r.Stages {
		status := "✓ PASS"
		if !stage.Success {
			status = "✗ FAIL"
		}
		fmt.Printf("[%s] %s (%v)\n", status, stage.Name, stage.Duration)
		if stage.Message != "" {
			fmt.Printf("       %s\n", stage.Message)
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

// ToJSON returns the report as JSON
func (r *CICDPipelineReport) ToJSON() (string, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// RunCICDPipelineCommand executes CI/CD pipeline tests as a CLI command
func RunCICDPipelineCommand(projectRoot, binaryPath string, jsonOutput bool) int {
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

	tester := NewCICDTester(projectRoot, binaryPath)
	report, err := tester.RunAllStages()
	if err != nil {
		fmt.Fprintf(os.Stderr, "CI/CD pipeline error: %v\n", err)
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

// ==================== Unit Tests ====================

func TestNewCICDTester(t *testing.T) {
	t.Run("creates tester with correct settings", func(t *testing.T) {
		tester := NewCICDTester("/project", "/binary")

		if tester.ProjectRoot != "/project" {
			t.Errorf("ProjectRoot = %s, want /project", tester.ProjectRoot)
		}
		if tester.BinaryPath != "/binary" {
			t.Errorf("BinaryPath = %s, want /binary", tester.BinaryPath)
		}
		if tester.Stages == nil {
			t.Error("Stages should be initialized")
		}
	})
}

func TestCICDPipelineReport(t *testing.T) {
	t.Run("JSON marshaling", func(t *testing.T) {
		report := &CICDPipelineReport{
			Success: true,
			Stages: []CICDStage{
				{Name: "build", Duration: time.Second, Success: true, Message: "OK"},
			},
			Summary:  "All good",
			ExitCode: 0,
			Duration: time.Minute,
			BuildInfo: BuildInfo{
				Version: "1.0.0",
			},
		}

		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var decoded CICDPipelineReport
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if decoded.Success != report.Success {
			t.Errorf("Success mismatch")
		}
		if len(decoded.Stages) != len(report.Stages) {
			t.Errorf("Stages length mismatch")
		}
	})

	t.Run("report formatting", func(t *testing.T) {
		report := &CICDPipelineReport{
			Success: true,
			Stages: []CICDStage{
				{Name: "build", Duration: time.Second, Success: true, Message: "OK"},
			},
			Summary:  "All good",
			ExitCode: 0,
			Duration: time.Minute,
			BuildInfo: BuildInfo{
				Version: "1.0.0",
			},
		}

		// Just verify it doesn't panic
		report.PrintReport()
	})
}

func TestGetString(t *testing.T) {
	t.Run("returns string value", func(t *testing.T) {
		m := map[string]interface{}{
			"key": "value",
		}
		result := getString(m, "key")
		if result != "value" {
			t.Errorf("Expected 'value', got '%s'", result)
		}
	})

	t.Run("returns unknown for missing key", func(t *testing.T) {
		m := map[string]interface{}{}
		result := getString(m, "missing")
		if result != "unknown" {
			t.Errorf("Expected 'unknown', got '%s'", result)
		}
	})

	t.Run("returns unknown for non-string value", func(t *testing.T) {
		m := map[string]interface{}{
			"key": 123,
		}
		result := getString(m, "key")
		if result != "unknown" {
			t.Errorf("Expected 'unknown', got '%s'", result)
		}
	})
}

func TestRunCICDPipelineCommand(t *testing.T) {
	t.Run("fails without go.mod", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "ci-cd-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		exitCode := RunCICDPipelineCommand(tempDir, "", false)
		if exitCode == 0 {
			t.Error("Expected non-zero exit code without go.mod")
		}
	})
}