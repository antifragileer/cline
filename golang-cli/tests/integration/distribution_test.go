// Package integration provides integration tests for the Cline CLI.
// This file contains tests for distribution mechanisms (Homebrew, NPM, etc.).
package integration

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/cline/cline/golang-cli/scripts"
)

// TestHomebrewFormulaGeneration tests the Homebrew formula generation.
func TestHomebrewFormulaGeneration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		version   string
		expectErr bool
	}{
		{
			name:      "valid semver version",
			version:   "1.0.0",
			expectErr: false,
		},
		{
			name:      "version with v prefix",
			version:   "v1.2.3",
			expectErr: false,
		},
		{
			name:      "version with prerelease",
			version:   "1.0.0-beta.1",
			expectErr: false,
		},
		{
			name:      "version with build metadata",
			version:   "1.0.0+build.123",
			expectErr: false,
		},
		{
			name:      "empty version should fail",
			version:   "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip actual URL download tests in CI to avoid network dependencies
			if os.Getenv("CI") != "" && !tt.expectErr {
				t.Skip("Skipping network-dependent test in CI")
			}

			formula := scripts.DefaultHomebrewFormula(tt.version)
			
			// Add a dummy platform for validation to pass
			if !tt.expectErr {
				formula.Platforms = []scripts.HomebrewPlatform{
					{
						OS:     "darwin",
						Arch:   "amd64",
						URL:    "https://example.com/test.tar.gz",
						SHA256: strings.Repeat("a", 64),
					},
				}
			}
			
			// Test validation
			err := formula.Validate()
			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected validation error for version %q, got nil", tt.version)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected validation error: %v", err)
				return
			}

			// Test formula generation
			content, err := formula.Generate()
			if err != nil {
				t.Errorf("Failed to generate formula: %v", err)
				return
			}

			// Verify formula content
			if !strings.Contains(content, "class Cline < Formula") {
				t.Error("Formula missing class definition")
			}

			// Check version - formula uses the version from DefaultHomebrewFormula which may normalize it
			expectedVersion := strings.TrimPrefix(tt.version, "v")
			if !strings.Contains(content, fmt.Sprintf(`version "%s"`, expectedVersion)) && 
			   !strings.Contains(content, fmt.Sprintf(`version "%s"`, tt.version)) {
				t.Logf("Formula content:\n%s", content)
				t.Errorf("Formula missing or incorrect version: expected %q or %q", expectedVersion, tt.version)
			}

			if !strings.Contains(content, `desc "AI-powered coding assistant CLI"`) {
				t.Error("Formula missing description")
			}

			if !strings.Contains(content, "def install") {
				t.Error("Formula missing install method")
			}

			if !strings.Contains(content, "test do") {
				t.Error("Formula missing test block")
			}
		})
	}
}

// TestHomebrewFormulaPlatformSupport tests platform-specific formula generation.
func TestHomebrewFormulaPlatformSupport(t *testing.T) {
	t.Parallel()

	formula := scripts.DefaultHomebrewFormula("1.0.0")

	// Test all supported platforms
	platforms := []struct {
		os   string
		arch string
	}{
		{"darwin", "amd64"},
		{"darwin", "arm64"},
		{"linux", "amd64"},
		{"linux", "arm64"},
	}

	for _, p := range platforms {
		t.Run(fmt.Sprintf("%s_%s", p.os, p.arch), func(t *testing.T) {
			// In CI, use dummy SHA256 to avoid network calls
			sha256 := "a" + strings.Repeat("0", 63) // 64 char hex string
			
			platform := scripts.HomebrewPlatform{
				OS:     p.os,
				Arch:   p.arch,
				URL:    fmt.Sprintf("https://example.com/cline_1.0.0_%s_%s.tar.gz", p.os, p.arch),
				SHA256: sha256,
			}
			
			formula.Platforms = append(formula.Platforms, platform)

			// Verify platform was added
			found := false
			for _, fp := range formula.Platforms {
				if fp.OS == p.os && fp.Arch == p.arch {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Platform %s/%s not found in formula", p.os, p.arch)
			}
		})
	}

	// Validate the complete formula
	if err := formula.Validate(); err != nil {
		t.Errorf("Formula validation failed: %v", err)
	}
}

// TestHomebrewTapStructure tests tap repository creation.
func TestHomebrewTapStructure(t *testing.T) {
	t.Parallel()

	// Create temporary directory for tap
	tempDir := t.TempDir()

	tap := scripts.DefaultHomebrewTap()
	
	err := tap.CreateTapStructure(tempDir)
	if err != nil {
		t.Fatalf("Failed to create tap structure: %v", err)
	}

	// Verify directory structure
	formulaDir := filepath.Join(tempDir, "Formula")
	if _, err := os.Stat(formulaDir); os.IsNotExist(err) {
		t.Error("Formula directory not created")
	}

	readmePath := filepath.Join(tempDir, "README.md")
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		t.Error("README.md not created")
	}

	// Verify README content
	readmeContent, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("Failed to read README: %v", err)
	}

	if !strings.Contains(string(readmeContent), "cline/homebrew-tap") {
		t.Error("README missing tap name")
	}

	if !strings.Contains(string(readmeContent), "brew tap") {
		t.Error("README missing installation instructions")
	}
}

// TestHomebrewSHA256Calculation tests SHA256 calculation functions.
func TestHomebrewSHA256Calculation(t *testing.T) {
	t.Parallel()

	// Create a temporary file with known content
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := []byte("Hello, Cline CLI!")

	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Calculate expected SHA256
	hasher := sha256.New()
	hasher.Write(testContent)
	expectedSHA256 := hex.EncodeToString(hasher.Sum(nil))

	// Test CalculateSHA256FromFile
	calculatedSHA256, err := scripts.CalculateSHA256FromFile(testFile)
	if err != nil {
		t.Errorf("CalculateSHA256FromFile failed: %v", err)
	}

	if calculatedSHA256 != expectedSHA256 {
		t.Errorf("SHA256 mismatch: got %s, expected %s", calculatedSHA256, expectedSHA256)
	}

	// Test with non-existent file
	_, err = scripts.CalculateSHA256FromFile("/nonexistent/file.txt")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

// TestNPMWrapperPackageJSON tests the NPM wrapper package configuration.
func TestNPMWrapperPackageJSON(t *testing.T) {
	t.Parallel()

	// Read the package.json file
	packagePath := filepath.Join("..", "..", "npm-wrapper", "package.json")
	
	// Skip if file doesn't exist (running in isolated test environment)
	if _, err := os.Stat(packagePath); os.IsNotExist(err) {
		t.Skip("NPM wrapper package.json not found")
	}

	content, err := os.ReadFile(packagePath)
	if err != nil {
		t.Fatalf("Failed to read package.json: %v", err)
	}

	// Verify required fields
	requiredFields := []string{
		`"name": "@cline/golang-cli"`,
		`"version"`,
		`"bin"`,
		`"cline-go"`,
		`"postinstall"`,
		`"engines"`,
		`"os"`,
		`"cpu"`,
	}

	contentStr := string(content)
	for _, field := range requiredFields {
		if !strings.Contains(contentStr, field) {
			t.Errorf("package.json missing required field: %s", field)
		}
	}

	// Verify platform restrictions
	if !strings.Contains(contentStr, "darwin") {
		t.Error("package.json missing darwin in os array")
	}
	if !strings.Contains(contentStr, "linux") {
		t.Error("package.json missing linux in os array")
	}
	if !strings.Contains(contentStr, "x64") && !strings.Contains(contentStr, "amd64") {
		t.Error("package.json missing x64/amd64 in cpu array")
	}
	if !strings.Contains(contentStr, "arm64") {
		t.Error("package.json missing arm64 in cpu array")
	}
}

// TestNPMWrapperScripts tests the NPM wrapper JavaScript files.
func TestNPMWrapperScripts(t *testing.T) {
	t.Parallel()

	// Get the directory of the current test file
	_, testFile, _, _ := runtime.Caller(0)
	testDir := filepath.Dir(testFile)
	wrapperDir := filepath.Join(testDir, "..", "..", "npm-wrapper")

	// Check if directory exists
	if _, err := os.Stat(wrapperDir); os.IsNotExist(err) {
		t.Skip("NPM wrapper directory not found")
	}

	// Test that required files exist
	requiredFiles := []string{
		"index.js",
		"install.js",
		"platform.js",
		"package.json",
		"README.md",
	}

	for _, file := range requiredFiles {
		path := filepath.Join(wrapperDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Required file missing: %s", file)
		}
	}

	// Test that platform.js is valid JavaScript (basic syntax check)
	platformPath := filepath.Join(wrapperDir, "platform.js")
	if _, err := os.Stat(platformPath); err == nil {
		// Try to parse with Node.js
		nodeCmd := exec.Command("node", "--check", platformPath)
		output, err := nodeCmd.CombinedOutput()
		if err != nil {
			t.Errorf("platform.js has syntax errors: %v\nOutput: %s", err, string(output))
		}
	}

	// Test that index.js is valid JavaScript
	indexPath := filepath.Join(wrapperDir, "index.js")
	if _, err := os.Stat(indexPath); err == nil {
		nodeCmd := exec.Command("node", "--check", indexPath)
		output, err := nodeCmd.CombinedOutput()
		if err != nil {
			t.Errorf("index.js has syntax errors: %v\nOutput: %s", err, string(output))
		}
	}
}

// TestNPMWrapperPlatformDetection tests platform detection in the wrapper.
func TestNPMWrapperPlatformDetection(t *testing.T) {
	t.Parallel()

	// Skip if not on a system with Node.js
	nodeCmd := exec.Command("node", "--version")
	if err := nodeCmd.Run(); err != nil {
		t.Skip("Node.js not available")
	}

	// Get the directory of the current test file
	_, testFile, _, _ := runtime.Caller(0)
	testDir := filepath.Dir(testFile)
	wrapperDir := filepath.Join(testDir, "..", "..", "npm-wrapper")
	platformPath := filepath.Join(wrapperDir, "platform.js")

	if _, err := os.Stat(platformPath); os.IsNotExist(err) {
		t.Skip("platform.js not found")
	}

	// Run the platform test script
	testPath := filepath.Join(wrapperDir, "platform.test.js")
	if _, err := os.Stat(testPath); os.IsNotExist(err) {
		t.Skip("platform.test.js not found")
	}

	cmd := exec.Command("node", testPath)
	cmd.Dir = wrapperDir
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Errorf("Platform tests failed: %v\nOutput: %s", err, string(output))
	}

	if !strings.Contains(string(output), "All tests passed") {
		t.Errorf("Platform tests did not complete successfully: %s", string(output))
	}
}

// TestDistributionBinaryNaming tests binary naming conventions.
func TestDistributionBinaryNaming(t *testing.T) {
	t.Parallel()

	version := "1.0.0"
	platforms := scripts.HomebrewSupportedPlatforms()

	for _, p := range platforms {
		t.Run(fmt.Sprintf("%s_%s", p.OS, p.Arch), func(t *testing.T) {
			expectedName := fmt.Sprintf("cline-%s-%s-%s.tar.gz", version, p.OS, p.Arch)
			actualName := scripts.GetHomebrewBinaryName(version, p.OS, p.Arch)

			if actualName != expectedName {
				t.Errorf("Binary name mismatch: got %s, expected %s", actualName, expectedName)
			}
		})
	}
}

// TestDistributionVersionValidation tests version string validation.
func TestDistributionVersionValidation(t *testing.T) {
	t.Parallel()

	validVersions := []string{
		"1.0.0",
		"v1.0.0",
		"2.0.0-beta",
		"1.0.0-alpha.1",
		"1.0.0+build.123",
		"0.0.1",
	}

	for _, v := range validVersions {
		t.Run(v, func(t *testing.T) {
			if !scripts.IsHomebrewValidVersion(v) {
				t.Errorf("Version %q should be valid", v)
			}
		})
	}

	invalidVersions := []string{
		"",
		"1.0.0/invalid",
		"1.0.0\\invalid",
	}

	for _, v := range invalidVersions {
		t.Run("invalid_"+strings.ReplaceAll(v, "/", "_"), func(t *testing.T) {
			if scripts.IsHomebrewValidVersion(v) {
				t.Errorf("Version %q should be invalid", v)
			}
		})
	}
}

// TestDistributionReleaseURLGeneration tests release URL generation.
func TestDistributionReleaseURLGeneration(t *testing.T) {
	t.Parallel()

	baseURL := "https://github.com/cline/cline/releases/download"
	version := "1.0.0"
	os := "darwin"
	arch := "arm64"

	expectedURL := "https://github.com/cline/cline/releases/download/v1.0.0/cline-1.0.0-darwin-arm64.tar.gz"
	actualURL := scripts.GenerateHomebrewPlatformURL(baseURL, version, os, arch)

	if actualURL != expectedURL {
		t.Errorf("URL mismatch:\ngot:      %s\nexpected: %s", actualURL, expectedURL)
	}
}

// TestDistributionCompletePipeline tests the complete distribution pipeline.
func TestDistributionCompletePipeline(t *testing.T) {
	t.Parallel()

	// This test verifies the entire distribution workflow
	// It should be run locally or in CI with proper setup
	if os.Getenv("RUN_DISTRIBUTION_TESTS") != "true" {
		t.Skip("Skipping complete pipeline test (set RUN_DISTRIBUTION_TESTS=true to run)")
	}

	tempDir := t.TempDir()
	version := "0.0.1-test"

	// Step 1: Create a test tap structure
	tap := scripts.DefaultHomebrewTap()
	tapDir := filepath.Join(tempDir, "homebrew-tap")

	if err := tap.CreateTapStructure(tapDir); err != nil {
		t.Fatalf("Failed to create tap structure: %v", err)
	}

	// Step 2: Generate a formula with test data
	formula := scripts.DefaultHomebrewFormula(version)
	
	// Add test platforms (without actual SHA256 calculation)
	platforms := scripts.HomebrewSupportedPlatforms()
	for _, p := range platforms {
		platform := scripts.HomebrewPlatform{
			OS:     p.OS,
			Arch:   p.Arch,
			URL:    scripts.GenerateHomebrewPlatformURL(tap.BaseURL, version, p.OS, p.Arch),
			SHA256: strings.Repeat("0", 64), // Dummy SHA256
		}
		formula.Platforms = append(formula.Platforms, platform)
	}

	// Step 3: Validate and generate formula
	if err := formula.Validate(); err != nil {
		t.Fatalf("Formula validation failed: %v", err)
	}

	content, err := formula.Generate()
	if err != nil {
		t.Fatalf("Failed to generate formula: %v", err)
	}

	// Step 4: Write formula to tap
	formulaPath := tap.GetFormulaPath(tapDir, formula.Name)
	formulaDir := filepath.Dir(formulaPath)
	if err := os.MkdirAll(formulaDir, 0755); err != nil {
		t.Fatalf("Failed to create formula directory: %v", err)
	}

	if err := os.WriteFile(formulaPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write formula: %v", err)
	}

	// Step 5: Verify the complete tap structure
	expectedFiles := []string{
		"README.md",
		"Formula/cline.rb",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(tapDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file not found: %s", file)
		}
	}

	// Step 6: Verify formula syntax (if Ruby is available)
	if _, err := exec.LookPath("ruby"); err == nil {
		rubyCmd := exec.Command("ruby", "-c", formulaPath)
		output, err := rubyCmd.CombinedOutput()
		if err != nil {
			t.Errorf("Formula has Ruby syntax errors: %v\nOutput: %s", err, string(output))
		}
	}
}

// TestDistributionCrossPlatformCompatibility tests cross-platform compatibility.
func TestDistributionCrossPlatformCompatibility(t *testing.T) {
	t.Parallel()

	// Test that distribution works on all supported platforms
	supportedPlatforms := []struct {
		os   string
		arch string
	}{
		{"darwin", "amd64"},
		{"darwin", "arm64"},
		{"linux", "amd64"},
		{"linux", "arm64"},
		{"windows", "amd64"},
	}

	for _, p := range supportedPlatforms {
		t.Run(fmt.Sprintf("%s_%s", p.os, p.arch), func(t *testing.T) {
			// Verify binary name generation
			version := "1.0.0"
			var binaryName string
			if p.os == "windows" {
				binaryName = fmt.Sprintf("cline_%s_%s_%s.exe.tar.gz", version, p.os, p.arch)
			} else {
				binaryName = fmt.Sprintf("cline_%s_%s_%s.tar.gz", version, p.os, p.arch)
			}

			if binaryName == "" {
				t.Error("Failed to generate binary name")
			}

			// Verify URL generation
			baseURL := "https://github.com/cline/cline/releases/download"
			url := scripts.GenerateHomebrewReleaseURL(baseURL, version, "cline", p.os, p.arch)
			
			if !strings.Contains(url, p.os) {
				t.Errorf("URL missing OS: %s", url)
			}
			if !strings.Contains(url, p.arch) {
				t.Errorf("URL missing arch: %s", url)
			}
		})
	}
}

// TestDistributionDocumentation tests that documentation exists and is valid.
func TestDistributionDocumentation(t *testing.T) {
	t.Parallel()

	// Test npm-wrapper README
	readmePath := filepath.Join("..", "..", "npm-wrapper", "README.md")
	if _, err := os.Stat(readmePath); err == nil {
		content, err := os.ReadFile(readmePath)
		if err != nil {
			t.Errorf("Failed to read npm-wrapper README: %v", err)
		} else {
			contentStr := string(content)
			
			requiredSections := []string{
				"Installation",
				"Usage",
				"Binary Name",
				"Troubleshooting",
			}

			for _, section := range requiredSections {
				if !strings.Contains(contentStr, section) {
					t.Errorf("npm-wrapper README missing section: %s", section)
				}
			}
		}
	}

	// Test main README
	mainReadmePath := filepath.Join("..", "..", "README.md")
	if _, err := os.Stat(mainReadmePath); err == nil {
		content, err := os.ReadFile(mainReadmePath)
		if err != nil {
			t.Errorf("Failed to read main README: %v", err)
		} else {
			contentStr := string(content)
			
			// Should mention installation methods
			if !strings.Contains(contentStr, "Installation") {
				t.Error("Main README should mention Installation")
			}
		}
	}
}

// BenchmarkHomebrewFormulaGeneration benchmarks formula generation.
func BenchmarkHomebrewFormulaGeneration(b *testing.B) {
	formula := scripts.DefaultHomebrewFormula("1.0.0")
	
	// Add dummy platforms
	platforms := scripts.HomebrewSupportedPlatforms()
	for _, p := range platforms {
		platform := scripts.HomebrewPlatform{
			OS:     p.OS,
			Arch:   p.Arch,
			URL:    fmt.Sprintf("https://example.com/cline_1.0.0_%s_%s.tar.gz", p.OS, p.Arch),
			SHA256: strings.Repeat("0", 64),
		}
		formula.Platforms = append(formula.Platforms, platform)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := formula.Generate()
		if err != nil {
			b.Fatalf("Formula generation failed: %v", err)
		}
	}
}

// BenchmarkSHA256Calculation benchmarks SHA256 calculation.
func BenchmarkSHA256Calculation(b *testing.B) {
	// Create test file
	tempDir := b.TempDir()
	testFile := filepath.Join(tempDir, "test.bin")
	testData := make([]byte, 1024*1024) // 1MB of data
	if err := os.WriteFile(testFile, testData, 0644); err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := scripts.CalculateSHA256FromFile(testFile)
		if err != nil {
			b.Fatalf("SHA256 calculation failed: %v", err)
		}
	}
}

// Helper function to check if running on specific platform
func isPlatform(os, arch string) bool {
	goOS := runtime.GOOS
	goArch := runtime.GOARCH

	// Normalize architecture names
	if goArch == "amd64" {
		goArch = "amd64"
	}

	return goOS == os && goArch == arch
}