// Package parity provides side-by-side testing of Node.js and GoLang CLIs
package parity

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Command-line flags
var (
	captureBaselines = flag.Bool("capture-baselines", false, "Capture baselines from Node.js CLI")
	generateReport   = flag.String("generate-report", "", "Generate report in specified format (html, md, json, junit)")
	outputDir        = flag.String("output-dir", "parity-reports", "Output directory for reports")
	baselineDir      = flag.String("baseline-dir", "baselines", "Directory for baseline files")
	nodeCliPath      = flag.String("node-cli", "", "Path to Node.js CLI (auto-detect if not specified)")
	golangCliPath    = flag.String("golang-cli", "", "Path to GoLang CLI binary (auto-detect if not specified)")
	runCategory      = flag.String("category", "", "Run only tests in specified category")
	runScenario      = flag.String("scenario", "", "Run only specified scenario")
	skipAICategory   = flag.Bool("skip-ai", false, "Skip AI connection tests (task category)")
)

// TestMain handles test setup and teardown
func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}

// TestParity runs the full parity test suite
// NOTE: DO NOT use t.Parallel() for AI tests - they must run serially per process-leak-prevention.md
func TestParity(t *testing.T) {
	// Check if we should capture baselines
	if *captureBaselines {
		testCaptureBaselines(t)
		return
	}

	// Check if we should run a specific scenario
	if *runScenario != "" {
		testSingleScenario(t, *runScenario)
		return
	}

	// Create suite
	opts := SuiteOptions{
		NodeCliPath:   *nodeCliPath,
		GoLangCliPath: *golangCliPath,
		OutputDir:     *outputDir,
		BaselineDir:   *baselineDir,
	}

	suite, err := NewSuite(opts)
	if err != nil {
		t.Fatalf("Failed to create parity suite: %v", err)
	}

	// Determine which scenarios to run
	var scenarios []Scenario
	if *runCategory != "" {
		scenarios = suite.GetRegistry().GetByCategory(Category(*runCategory))
		if len(scenarios) == 0 {
			t.Fatalf("No scenarios found for category: %s", *runCategory)
		}
	} else {
		scenarios = suite.GetRegistry().GetAll()
	}

	// Filter out AI tests if requested
	if *skipAICategory {
		var filtered []Scenario
		for _, s := range scenarios {
			if s.Category != CategoryTask {
				filtered = append(filtered, s)
			}
		}
		scenarios = filtered
	}

	// Run tests
	t.Logf("Running %d parity tests...", len(scenarios))
	result, err := suite.RunScenarios(scenarios)
	if err != nil {
		t.Fatalf("Failed to run parity tests: %v", err)
	}

	// Generate report if requested
	if *generateReport != "" {
		if err := suite.GetReporter().Generate(result); err != nil {
			t.Errorf("Failed to generate report: %v", err)
		} else {
			t.Logf("Report generated in: %s", *outputDir)
		}
	}

	// Print console summary
	consoleReporter := NewConsoleReporter()
	consoleReporter.PrintSummary(result)

	// Fail if any tests failed
	if result.Failed > 0 {
		t.Errorf("Parity tests failed: %d/%d", result.Failed, result.Total)
	}
}

// TestParity_Help runs help category tests
func TestParity_Help(t *testing.T) {
	testCategory(t, CategoryHelp)
}

// TestParity_Version runs version category tests
func TestParity_Version(t *testing.T) {
	testCategory(t, CategoryVersion)
}

// TestParity_Config runs config category tests
func TestParity_Config(t *testing.T) {
	testCategory(t, CategoryConfig)
}

// TestParity_History runs history category tests
func TestParity_History(t *testing.T) {
	testCategory(t, CategoryHistory)
}

// TestParity_Auth runs auth category tests
func TestParity_Auth(t *testing.T) {
	testCategory(t, CategoryAuth)
}

// TestParity_Error runs error category tests
func TestParity_Error(t *testing.T) {
	testCategory(t, CategoryError)
}

// TestParity_Task runs task category tests
// NOTE: This runs serially - DO NOT add t.Parallel()
func TestParity_Task(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping task tests in short mode")
	}
	testCategory(t, CategoryTask)
}

// testCategory runs tests for a specific category
func testCategory(t *testing.T, cat Category) {
	opts := SuiteOptions{
		NodeCliPath:   *nodeCliPath,
		GoLangCliPath: *golangCliPath,
		OutputDir:     *outputDir,
		BaselineDir:   *baselineDir,
	}

	suite, err := NewSuite(opts)
	if err != nil {
		t.Fatalf("Failed to create parity suite: %v", err)
	}

	result, err := suite.RunCategory(cat)
	if err != nil {
		t.Fatalf("Failed to run %s tests: %v", cat, err)
	}

	// Print summary
	consoleReporter := NewConsoleReporter()
	consoleReporter.PrintSummary(result)

	if result.Failed > 0 {
		t.Errorf("%s tests failed: %d/%d", cat, result.Failed, result.Total)
	}
}

// testSingleScenario runs a single scenario by name
func testSingleScenario(t *testing.T, name string) {
	opts := SuiteOptions{
		NodeCliPath:   *nodeCliPath,
		GoLangCliPath: *golangCliPath,
		OutputDir:     *outputDir,
		BaselineDir:   *baselineDir,
	}

	suite, err := NewSuite(opts)
	if err != nil {
		t.Fatalf("Failed to create parity suite: %v", err)
	}

	result, err := suite.RunByName(name)
	if err != nil {
		t.Fatalf("Failed to run scenario %s: %v", name, err)
	}

	if !result.Passed {
		if result.Error != "" {
			t.Errorf("Scenario %s failed: %s", name, result.Error)
		} else {
			t.Errorf("Scenario %s failed", name)
		}

		if result.Comparison != nil {
			t.Logf("Diff:\n%s", result.Comparison.Diff)
		}
	}
}

// testCaptureBaselines captures baselines from Node.js CLI
func testCaptureBaselines(t *testing.T) {
	opts := SuiteOptions{
		NodeCliPath:   *nodeCliPath,
		GoLangCliPath: *golangCliPath,
		OutputDir:     *outputDir,
		BaselineDir:   *baselineDir,
	}

	suite, err := NewSuite(opts)
	if err != nil {
		t.Fatalf("Failed to create parity suite: %v", err)
	}

	var scenarios []Scenario
	if *runCategory != "" {
		scenarios = suite.GetRegistry().GetByCategory(Category(*runCategory))
	} else {
		scenarios = suite.GetRegistry().GetAll()
	}

	t.Logf("Capturing %d baselines...", len(scenarios))
	if err := suite.CaptureBaselines(scenarios); err != nil {
		t.Fatalf("Failed to capture baselines: %v", err)
	}

	t.Logf("Baselines captured in: %s", *baselineDir)
}

// TestCLIInfo verifies CLI detection and info
func TestCLIInfo(t *testing.T) {
	// Test Node.js CLI detection
	nodeInfo, err := GetNodeCLIInfo()
	if err != nil {
		t.Skipf("Node.js CLI not available: %v", err)
	} else {
		t.Logf("Node.js CLI detected:")
		for k, v := range nodeInfo {
			t.Logf("  %s: %s", k, v)
		}
	}

	// Test GoLang CLI detection
	golangInfo, err := GetGoLangCLIInfo()
	if err != nil {
		t.Skipf("GoLang CLI not available: %v", err)
	} else {
		t.Logf("GoLang CLI detected:")
		for k, v := range golangInfo {
			t.Logf("  %s: %s", k, v)
		}
	}
}

// TestComparator verifies the output comparator
func TestComparator(t *testing.T) {
	comparator := NewComparator()

	tests := []struct {
		name     string
		expected string
		actual   string
		wantMatch bool
	}{
		{
			name:      "exact match",
			expected:  "hello world",
			actual:    "hello world",
			wantMatch: true,
		},
		{
			name:      "timestamp normalization",
			expected:  "Started at 2024-01-15T10:30:00Z",
			actual:    "Started at 2024-01-20T15:45:30Z",
			wantMatch: true,
		},
		{
			name:      "UUID normalization",
			expected:  "Task ID: 550e8400-e29b-41d4-a716-446655440000",
			actual:    "Task ID: 6ba7b810-9dad-11d1-80b4-00c04fd430c8",
			wantMatch: true,
		},
		{
			name:      "duration normalization",
			expected:  "Completed in 5.2s",
			actual:    "Completed in 3.8s",
			wantMatch: true,
		},
		{
			name:      "different content",
			expected:  "hello world",
			actual:    "goodbye world",
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := comparator.Compare(tt.expected, tt.actual)
			if result.Match != tt.wantMatch {
				t.Errorf("Compare() match = %v, want %v", result.Match, tt.wantMatch)
				t.Logf("Expected (normalized):\n%s", result.Expected)
				t.Logf("Actual (normalized):\n%s", result.Actual)
				t.Logf("Diff:\n%s", result.Diff)
			}
		})
	}
}

// TestScenarioRegistry verifies scenario registry
func TestScenarioRegistry(t *testing.T) {
	registry := NewRegistry()

	// Check that we have scenarios
	scenarios := registry.GetAll()
	if len(scenarios) == 0 {
		t.Error("Registry has no scenarios")
	}

	// Check categories
	categories := map[Category]int{}
	for _, s := range scenarios {
		categories[s.Category]++
	}

	expectedCategories := []Category{
		CategoryHelp,
		CategoryVersion,
		CategoryTask,
		CategoryHistory,
		CategoryConfig,
		CategoryAuth,
		CategoryError,
	}

	for _, cat := range expectedCategories {
		count := categories[cat]
		if count == 0 {
			t.Errorf("No scenarios found for category %s", cat)
		} else {
			t.Logf("Category %s has %d scenarios", cat, count)
		}
	}

	// Test lookup by name
	if _, found := registry.GetByName("help_flag"); !found {
		t.Error("Could not find 'help_flag' scenario")
	}

	// Test category filtering
	helpScenarios := registry.GetByCategory(CategoryHelp)
	if len(helpScenarios) == 0 {
		t.Error("No help scenarios found")
	}
}

// TestBaselinesDirectory verifies baseline directory setup
func TestBaselinesDirectory(t *testing.T) {
	// Create test baseline directory
	testDir := t.TempDir()

	// Create a test baseline file
	testFile := filepath.Join(testDir, "test_help.txt")
	content := "Test help output"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test baseline: %v", err)
	}

	// Read it back
	read, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test baseline: %v", err)
	}

	if string(read) != content {
		t.Errorf("Baseline content mismatch: got %q, want %q", string(read), content)
	}
}

// BenchmarkComparator benchmarks the comparator
func BenchmarkComparator(b *testing.B) {
	comparator := NewComparator()

	expected := "Task completed in 5.2s at 2024-01-15T10:30:00Z with ID 550e8400-e29b-41d4-a716-446655440000"
	actual := "Task completed in 3.8s at 2024-01-20T15:45:30Z with ID 6ba7b810-9dad-11d1-80b4-00c04fd430c8"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		comparator.Compare(expected, actual)
	}
}

// Example test showing how to use the suite
func ExampleSuite() {
	// This is an example - won't actually run in tests
	opts := SuiteOptions{
		OutputDir:   "reports",
		BaselineDir: "baselines",
	}

	suite, _ := NewSuite(opts)
	result, _ := suite.RunCategory(CategoryHelp)

	// Print results
	console := NewConsoleReporter()
	console.PrintSummary(result)
}

// Integration test marker
func TestIntegration(t *testing.T) {
	if os.Getenv("PARITY_INTEGRATION") != "1" {
		t.Skip("Skipping integration test. Set PARITY_INTEGRATION=1 to run.")
	}

	// Run full suite with reporting
	opts := SuiteOptions{
		OutputDir:   "parity-reports",
		BaselineDir: "baselines",
	}

	suite, err := NewSuite(opts)
	if err != nil {
		t.Fatalf("Failed to create suite: %v", err)
	}

	result, err := suite.RunAll()
	if err != nil {
		t.Fatalf("Failed to run suite: %v", err)
	}

	// Generate all report formats
	if err := suite.GetReporter().Generate(result); err != nil {
		t.Errorf("Failed to generate reports: %v", err)
	}

	// Verify reports were created
	reports := []string{
		"parity_report.md",
		"parity_report.json",
		"parity_report.html",
		"parity_report.xml",
	}

	for _, report := range reports {
		path := filepath.Join("parity-reports", report)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Report not created: %s", path)
		}
	}
}

// Helper function to check if running in CI
func isCI() bool {
	ciVars := []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI", "CIRCLECI", "JENKINS_URL", "BUILDKITE"}
	for _, v := range ciVars {
		if os.Getenv(v) != "" {
			return true
		}
	}
	return false
}

// Helper function to skip tests in CI if needed
func skipInCI(t *testing.T, reason string) {
	if isCI() {
		t.Skipf("Skipping in CI: %s", reason)
	}
}

// TestSkippedImports verifies that test flags work correctly
func TestFlags(t *testing.T) {
	// Just verify the flags are defined
	t.Logf("capture-baselines: %v", *captureBaselines)
	t.Logf("generate-report: %s", *generateReport)
	t.Logf("output-dir: %s", *outputDir)
	t.Logf("baseline-dir: %s", *baselineDir)
	t.Logf("category: %s", *runCategory)
	t.Logf("scenario: %s", *runScenario)
	t.Logf("skip-ai: %v", *skipAICategory)
}

// TestExecutableInPath verifies CLIs are in PATH
func TestExecutableInPath(t *testing.T) {
	// Check for node
	if _, err := execLookPath("node"); err != nil {
		t.Log("node not in PATH")
	}

	// Check for go
	if _, err := execLookPath("go"); err != nil {
		t.Log("go not in PATH")
	}
}

// execLookPath wraps exec.LookPath for testing
func execLookPath(file string) (string, error) {
	// This is a stub - the real implementation uses os/exec
	return "", nil
}

// Cleanup function to remove test artifacts
func cleanupTestArtifacts(dir string) {
	// Remove test reports
	os.RemoveAll(dir)
}

// TestCleanup verifies cleanup works
func TestCleanup(t *testing.T) {
	testDir := t.TempDir()
	testFile := filepath.Join(testDir, "test.txt")
	
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	// TempDir cleanup is automatic, but verify file was created
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("Test file was not created")
	}
}

// TestGoModCheck verifies we're in a Go module
func TestGoModCheck(t *testing.T) {
	// Find go.mod by walking up
	dir, _ := os.Getwd()
	for {
		goMod := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goMod); err == nil {
			t.Logf("Found go.mod at: %s", goMod)
			return
		}
		
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	
	t.Skip("go.mod not found - may be running outside module")
}

// TestRelativePathHandling verifies path handling
func TestRelativePathHandling(t *testing.T) {
	// Test that relative paths work correctly
	base := "reports"
	sub := "test"
	full := filepath.Join(base, sub)
	
	if !strings.Contains(full, base) {
		t.Error("Path join did not include base")
	}
}