// Package tests provides test coverage analysis and reporting for the Go CLI.
// This package ensures all packages achieve >80% test coverage.
package tests

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// CoverageThreshold is the minimum required coverage percentage
const CoverageThreshold = 80.0

// CoverageResult represents the coverage for a single package
type CoverageResult struct {
	Package     string  `json:"package"`
	Coverage    float64 `json:"coverage"`
	Statements  int     `json:"statements"`
	Covered     int     `json:"covered"`
	Uncovered   int     `json:"uncovered"`
	Passes      bool    `json:"passes"`
}

// CoverageReport contains coverage results for all packages
type CoverageReport struct {
	Success      bool             `json:"success"`
	Results      []CoverageResult `json:"results"`
	Overall      float64          `json:"overallCoverage"`
	Threshold    float64          `json:"threshold"`
	Summary      string           `json:"summary"`
	ExitCode     int              `json:"exitCode"`
	TotalPackages int             `json:"totalPackages"`
	PassedPackages int            `json:"passedPackages"`
	FailedPackages int            `json:"failedPackages"`
}

// CoverageAnalyzer analyzes test coverage
type CoverageAnalyzer struct {
	ProjectRoot string
	Threshold   float64
	Results     []CoverageResult
}

// NewCoverageAnalyzer creates a new coverage analyzer
func NewCoverageAnalyzer(projectRoot string, threshold float64) *CoverageAnalyzer {
	return &CoverageAnalyzer{
		ProjectRoot: projectRoot,
		Threshold:   threshold,
		Results:     make([]CoverageResult, 0),
	}
}

// Analyze runs coverage analysis on all packages
func (c *CoverageAnalyzer) Analyze() (*CoverageReport, error) {
	c.Results = make([]CoverageResult, 0)

	// Run go test with coverage
	cmd := exec.Command("go", "test", "./...", "-coverprofile=coverage.out", "-covermode=set")
	cmd.Dir = c.ProjectRoot
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Tests might fail, but coverage file could still be generated
		// Continue to parse coverage
		_ = output
	}

	// Parse coverage output
	cmd = exec.Command("go", "tool", "cover", "-func=coverage.out")
	cmd.Dir = c.ProjectRoot
	output, err = cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to parse coverage: %w\nOutput: %s", err, string(output))
	}

	// Parse the coverage results
	c.parseCoverageOutput(string(output))

	// Calculate overall coverage
	totalStatements := 0
	totalCovered := 0
	passedCount := 0
	failedCount := 0

	for _, result := range c.Results {
		totalStatements += result.Statements
		totalCovered += result.Covered
		if result.Passes {
			passedCount++
		} else {
			failedCount++
		}
	}

	overall := 0.0
	if totalStatements > 0 {
		overall = float64(totalCovered) / float64(totalStatements) * 100
	}

	success := failedCount == 0 && overall >= c.Threshold
	exitCode := 0
	summary := fmt.Sprintf("All packages meet %.1f%% coverage threshold (overall: %.2f%%)", c.Threshold, overall)
	if !success {
		exitCode = 1
		summary = fmt.Sprintf("Coverage check failed: %d packages below %.1f%% threshold (overall: %.2f%%)", failedCount, c.Threshold, overall)
	}

	report := &CoverageReport{
		Success:         success,
		Results:         c.Results,
		Overall:         overall,
		Threshold:       c.Threshold,
		Summary:         summary,
		ExitCode:        exitCode,
		TotalPackages:   len(c.Results),
		PassedPackages:  passedCount,
		FailedPackages:  failedCount,
	}

	return report, nil
}

// parseCoverageOutput parses the output of go tool cover -func
func (c *CoverageAnalyzer) parseCoverageOutput(output string) {
	lines := strings.Split(output, "\n")
	
	// Map to aggregate coverage by package
	packageStats := make(map[string]*struct {
		statements int
		covered    int
	})

	// Regex to match coverage lines
	// Format: <file>:<function> <statements> <coverage>%
	lineRegex := regexp.MustCompile(`^([^:]+):([^ ]+) (\d+) (\d+\.\d+)%$`)

	for _, line := range lines {
		matches := lineRegex.FindStringSubmatch(line)
		if len(matches) != 5 {
			continue
		}

		file := matches[1]
		statements, _ := strconv.Atoi(matches[3])
		coverage, _ := strconv.ParseFloat(matches[4], 64)

		// Extract package from file path
		pkg := c.extractPackage(file)
		if pkg == "" {
			continue
		}

		if _, ok := packageStats[pkg]; !ok {
			packageStats[pkg] = &struct {
				statements int
				covered    int
			}{}
		}

		packageStats[pkg].statements += statements
		packageStats[pkg].covered += int(float64(statements) * coverage / 100)
	}

	// Convert to results
	for pkg, stats := range packageStats {
		coverage := 0.0
		if stats.statements > 0 {
			coverage = float64(stats.covered) / float64(stats.statements) * 100
		}

		result := CoverageResult{
			Package:    pkg,
			Coverage:   coverage,
			Statements: stats.statements,
			Covered:    stats.covered,
			Uncovered:  stats.statements - stats.covered,
			Passes:     coverage >= c.Threshold,
		}
		c.Results = append(c.Results, result)
	}

	// Sort by coverage (lowest first)
	sort.Slice(c.Results, func(i, j int) bool {
		return c.Results[i].Coverage < c.Results[j].Coverage
	})
}

// extractPackage extracts the package name from a file path
func (c *CoverageAnalyzer) extractPackage(file string) string {
	// File path is relative to project root
	// We want the package path (directory containing the file)
	dir := filepath.Dir(file)
	
	// Remove leading ./ if present
	dir = strings.TrimPrefix(dir, "./")
	
	// Skip test files and vendor
	if strings.Contains(dir, "vendor") || strings.Contains(dir, "testdata") {
		return ""
	}

	// Convert to package import path
	pkg := filepath.Join("github.com/cline/cline/golang-cli", dir)
	pkg = strings.ReplaceAll(pkg, string(filepath.Separator), "/")
	
	return pkg
}

// PrintReport prints a human-readable coverage report
func (r *CoverageReport) PrintReport() {
	fmt.Println("=" + strings.Repeat("=", 70))
	fmt.Println("  Test Coverage Report")
	fmt.Println("=" + strings.Repeat("=", 70))
	fmt.Printf("Overall Coverage: %.2f%%\n", r.Overall)
	fmt.Printf("Threshold: %.1f%%\n", r.Threshold)
	fmt.Printf("Total Packages: %d\n", r.TotalPackages)
	fmt.Printf("Passed: %d | Failed: %d\n", r.PassedPackages, r.FailedPackages)
	fmt.Println()

	// Print failed packages first
	fmt.Println("Package Coverage Details:")
	fmt.Println(strings.Repeat("-", 72))
	
	for _, result := range r.Results {
		status := "✓ PASS"
		if !result.Passes {
			status = "✗ FAIL"
		}
		fmt.Printf("[%s] %-50s %6.2f%% (%d/%d statements)\n", 
			status, result.Package, result.Coverage, result.Covered, result.Statements)
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
func (r *CoverageReport) ToJSON() (string, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// RunCoverageCheckCommand executes coverage check as a CLI command
func RunCoverageCheckCommand(projectRoot string, threshold float64, jsonOutput bool) int {
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

	// Use default threshold if not specified
	if threshold <= 0 {
		threshold = CoverageThreshold
	}

	analyzer := NewCoverageAnalyzer(projectRoot, threshold)
	report, err := analyzer.Analyze()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Coverage analysis error: %v\n", err)
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

// GetPackageCoverage returns coverage for a specific package
func GetPackageCoverage(projectRoot, pkg string) (float64, error) {
	analyzer := NewCoverageAnalyzer(projectRoot, CoverageThreshold)
	report, err := analyzer.Analyze()
	if err != nil {
		return 0, err
	}

	for _, result := range report.Results {
		if result.Package == pkg || strings.HasSuffix(result.Package, "/"+pkg) {
			return result.Coverage, nil
		}
	}

	return 0, fmt.Errorf("package %s not found", pkg)
}

// ==================== Unit Tests ====================

func TestNewCoverageAnalyzer(t *testing.T) {
	t.Run("creates analyzer with correct settings", func(t *testing.T) {
		analyzer := NewCoverageAnalyzer("/project", 85.0)

		if analyzer.ProjectRoot != "/project" {
			t.Errorf("ProjectRoot = %s, want /project", analyzer.ProjectRoot)
		}
		if analyzer.Threshold != 85.0 {
			t.Errorf("Threshold = %f, want 85.0", analyzer.Threshold)
		}
		if analyzer.Results == nil {
			t.Error("Results should be initialized")
		}
	})
}

func TestCoverageResult(t *testing.T) {
	t.Run("JSON marshaling", func(t *testing.T) {
		result := CoverageResult{
			Package:    "github.com/example/test",
			Coverage:   85.5,
			Statements: 100,
			Covered:    85,
			Uncovered:  15,
			Passes:     true,
		}

		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var decoded CoverageResult
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if decoded.Package != result.Package {
			t.Errorf("Package mismatch")
		}
		if decoded.Coverage != result.Coverage {
			t.Errorf("Coverage mismatch")
		}
	})
}

func TestCoverageReport(t *testing.T) {
	t.Run("report formatting", func(t *testing.T) {
		report := &CoverageReport{
			Success:         true,
			Overall:         85.5,
			Threshold:       80.0,
			Results: []CoverageResult{
				{Package: "pkg1", Coverage: 90.0, Passes: true},
				{Package: "pkg2", Coverage: 75.0, Passes: false},
			},
			TotalPackages:   2,
			PassedPackages:  1,
			FailedPackages:  1,
			Summary:         "Test summary",
			ExitCode:        0,
		}

		// Just verify it doesn't panic
		report.PrintReport()
	})

	t.Run("ToJSON", func(t *testing.T) {
		report := &CoverageReport{
			Success: true,
			Results: []CoverageResult{},
			Summary: "Test",
		}

		jsonStr, err := report.ToJSON()
		if err != nil {
			t.Fatalf("ToJSON failed: %v", err)
		}

		if !strings.Contains(jsonStr, "Test") {
			t.Error("JSON should contain summary")
		}
	})
}

func TestExtractPackage(t *testing.T) {
	analyzer := NewCoverageAnalyzer("/project", 80.0)

	tests := []struct {
		file     string
		expected string
	}{
		{"./internal/api/provider.go", "github.com/cline/cline/golang-cli/internal/api"},
		{"./cmd/cline/main.go", "github.com/cline/cline/golang-cli/cmd/cline"},
		{"./internal/tui/chat.go", "github.com/cline/cline/golang-cli/internal/tui"},
		{"vendor/something.go", ""},
		{"./testdata/test.go", ""},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			result := analyzer.extractPackage(tt.file)
			if result != tt.expected {
				t.Errorf("extractPackage(%s) = %s, want %s", tt.file, result, tt.expected)
			}
		})
	}
}

func TestRunCoverageCheckCommand(t *testing.T) {
	t.Run("fails without go.mod", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "coverage-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		exitCode := RunCoverageCheckCommand(tempDir, 80.0, false)
		if exitCode == 0 {
			t.Error("Expected non-zero exit code without go.mod")
		}
	})
}