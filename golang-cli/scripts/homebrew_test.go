package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDefaultHomebrewFormula(t *testing.T) {
	version := "1.0.0"
	formula := DefaultHomebrewFormula(version)

	if formula.Name != "cline" {
		t.Errorf("Expected name 'cline', got '%s'", formula.Name)
	}

	if formula.Version != version {
		t.Errorf("Expected version '%s', got '%s'", version, formula.Version)
	}

	if formula.Homepage != "https://github.com/cline/cline" {
		t.Errorf("Expected homepage 'https://github.com/cline/cline', got '%s'", formula.Homepage)
	}

	if formula.License != "Apache-2.0" {
		t.Errorf("Expected license 'Apache-2.0', got '%s'", formula.License)
	}

	if formula.BinaryName != "cline" {
		t.Errorf("Expected binary name 'cline', got '%s'", formula.BinaryName)
	}

	if len(formula.Platforms) != 0 {
		t.Errorf("Expected empty platforms, got %d", len(formula.Platforms))
	}
}

func TestHomebrewFormulaAddPlatform(t *testing.T) {
	// Create test content and calculate expected SHA256
	testContent := []byte("test binary content for sha256 calculation")
	hasher := sha256.New()
	hasher.Write(testContent)
	expectedSHA256 := hex.EncodeToString(hasher.Sum(nil))

	// Create a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(testContent)
	}))
	defer server.Close()

	formula := DefaultHomebrewFormula("1.0.0")
	url := server.URL + "/test.tar.gz"

	err := formula.AddPlatform("darwin", "amd64", url)
	if err != nil {
		t.Fatalf("AddPlatform failed: %v", err)
	}

	if len(formula.Platforms) != 1 {
		t.Fatalf("Expected 1 platform, got %d", len(formula.Platforms))
	}

	p := formula.Platforms[0]
	if p.OS != "darwin" {
		t.Errorf("Expected OS 'darwin', got '%s'", p.OS)
	}

	if p.Arch != "amd64" {
		t.Errorf("Expected Arch 'amd64', got '%s'", p.Arch)
	}

	if p.URL != url {
		t.Errorf("Expected URL '%s', got '%s'", url, p.URL)
	}

	if p.SHA256 != expectedSHA256 {
		t.Errorf("Expected SHA256 '%s', got '%s'", expectedSHA256, p.SHA256)
	}
}

func TestHomebrewFormulaAddPlatformWithLocalFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "cline_1.0.0_darwin_amd64.tar.gz")
	testContent := []byte("test binary content")
	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Calculate expected SHA256
	hasher := sha256.New()
	hasher.Write(testContent)
	expectedSHA256 := hex.EncodeToString(hasher.Sum(nil))

	formula := DefaultHomebrewFormula("1.0.0")
	url := "https://example.com/cline_1.0.0_darwin_amd64.tar.gz"

	err := formula.AddPlatformWithLocalFile("darwin", "amd64", url, testFile)
	if err != nil {
		t.Fatalf("AddPlatformWithLocalFile failed: %v", err)
	}

	if len(formula.Platforms) != 1 {
		t.Fatalf("Expected 1 platform, got %d", len(formula.Platforms))
	}

	p := formula.Platforms[0]
	if p.SHA256 != expectedSHA256 {
		t.Errorf("Expected SHA256 '%s', got '%s'", expectedSHA256, p.SHA256)
	}
}

func TestHomebrewFormulaValidate(t *testing.T) {
	tests := []struct {
		name      string
		formula   *HomebrewFormula
		wantError bool
		errorMsg  string
	}{
		{
			name:      "empty formula",
			formula:   &HomebrewFormula{},
			wantError: true,
			errorMsg:  "formula name is required",
		},
		{
			name: "missing version",
			formula: &HomebrewFormula{
				Name: "cline",
			},
			wantError: true,
			errorMsg:  "formula version is required",
		},
		{
			name: "missing homepage",
			formula: &HomebrewFormula{
				Name:    "cline",
				Version: "1.0.0",
			},
			wantError: true,
			errorMsg:  "formula homepage is required",
		},
		{
			name: "no platforms",
			formula: &HomebrewFormula{
				Name:     "cline",
				Version:  "1.0.0",
				Homepage: "https://example.com",
			},
			wantError: true,
			errorMsg:  "at least one platform is required",
		},
		{
			name: "missing platform OS",
			formula: &HomebrewFormula{
				Name:     "cline",
				Version:  "1.0.0",
				Homepage: "https://example.com",
				Platforms: []HomebrewPlatform{
					{Arch: "amd64", URL: "https://example.com", SHA256: strings.Repeat("a", 64)},
				},
			},
			wantError: true,
			errorMsg:  "platform 0: OS is required",
		},
		{
			name: "missing platform Arch",
			formula: &HomebrewFormula{
				Name:     "cline",
				Version:  "1.0.0",
				Homepage: "https://example.com",
				Platforms: []HomebrewPlatform{
					{OS: "darwin", URL: "https://example.com", SHA256: strings.Repeat("a", 64)},
				},
			},
			wantError: true,
			errorMsg:  "platform 0: Arch is required",
		},
		{
			name: "missing platform URL",
			formula: &HomebrewFormula{
				Name:     "cline",
				Version:  "1.0.0",
				Homepage: "https://example.com",
				Platforms: []HomebrewPlatform{
					{OS: "darwin", Arch: "amd64", SHA256: strings.Repeat("a", 64)},
				},
			},
			wantError: true,
			errorMsg:  "platform 0: URL is required",
		},
		{
			name: "missing platform SHA256",
			formula: &HomebrewFormula{
				Name:     "cline",
				Version:  "1.0.0",
				Homepage: "https://example.com",
				Platforms: []HomebrewPlatform{
					{OS: "darwin", Arch: "amd64", URL: "https://example.com"},
				},
			},
			wantError: true,
			errorMsg:  "platform 0: SHA256 is required",
		},
		{
			name: "invalid SHA256 length",
			formula: &HomebrewFormula{
				Name:     "cline",
				Version:  "1.0.0",
				Homepage: "https://example.com",
				Platforms: []HomebrewPlatform{
					{OS: "darwin", Arch: "amd64", URL: "https://example.com", SHA256: "tooshort"},
				},
			},
			wantError: true,
			errorMsg:  "platform 0: SHA256 must be 64 characters",
		},
		{
			name: "valid formula",
			formula: &HomebrewFormula{
				Name:     "cline",
				Version:  "1.0.0",
				Homepage: "https://example.com",
				Platforms: []HomebrewPlatform{
					{OS: "darwin", Arch: "amd64", URL: "https://example.com", SHA256: strings.Repeat("a", 64)},
				},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.formula.Validate()
			if tt.wantError {
				if err == nil {
					t.Errorf("Validate() expected error but got nil")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Validate() error = %v, want error containing %q", err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestHomebrewFormulaGenerate(t *testing.T) {
	formula := &HomebrewFormula{
		Name:        "cline",
		Desc:        "AI-powered coding assistant CLI",
		Homepage:    "https://github.com/cline/cline",
		Version:     "1.0.0",
		License:     "Apache-2.0",
		BinaryName:  "cline",
		InstallDir:  "bin",
		TestCommand: "cline version",
		Platforms: []HomebrewPlatform{
			{
				OS:     "darwin",
				Arch:   "amd64",
				URL:    "https://github.com/cline/cline/releases/download/v1.0.0/cline_1.0.0_darwin_amd64.tar.gz",
				SHA256: "a" + strings.Repeat("0", 63),
			},
			{
				OS:     "darwin",
				Arch:   "arm64",
				URL:    "https://github.com/cline/cline/releases/download/v1.0.0/cline_1.0.0_darwin_arm64.tar.gz",
				SHA256: "b" + strings.Repeat("0", 63),
			},
			{
				OS:     "linux",
				Arch:   "amd64",
				URL:    "https://github.com/cline/cline/releases/download/v1.0.0/cline_1.0.0_linux_amd64.tar.gz",
				SHA256: "c" + strings.Repeat("0", 63),
			},
			{
				OS:     "linux",
				Arch:   "arm64",
				URL:    "https://github.com/cline/cline/releases/download/v1.0.0/cline_1.0.0_linux_arm64.tar.gz",
				SHA256: "d" + strings.Repeat("0", 63),
			},
		},
	}

	content, err := formula.Generate()
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Check that the content contains expected elements
	expectedElements := []string{
		"class Cline < Formula",
		`desc "AI-powered coding assistant CLI"`,
		`homepage "https://github.com/cline/cline"`,
		`version "1.0.0"`,
		`license "Apache-2.0"`,
		`url "https://github.com/cline/cline/releases/download/v1.0.0/cline_1.0.0_darwin_amd64.tar.gz"`,
		"if OS.mac? && Hardware::CPU.intel?",
		"if OS.mac? && Hardware::CPU.arm?",
		"if OS.linux? && Hardware::CPU.intel?",
		"if OS.linux? && Hardware::CPU.arm?",
		`bin.install "cline"`,
		`system "#{bin}/cline", "version"`,
	}

	for _, elem := range expectedElements {
		if !strings.Contains(content, elem) {
			t.Errorf("Generated formula missing expected element: %s", elem)
		}
	}
}

func TestHomebrewFormulaGenerateToFile(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "Formula", "cline.rb")

	formula := &HomebrewFormula{
		Name:     "cline",
		Version:  "1.0.0",
		Homepage: "https://example.com",
		Platforms: []HomebrewPlatform{
			{OS: "darwin", Arch: "amd64", URL: "https://example.com", SHA256: strings.Repeat("a", 64)},
		},
	}

	err := formula.GenerateToFile(outputPath)
	if err != nil {
		t.Fatalf("GenerateToFile() failed: %v", err)
	}

	// Check file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Errorf("Formula file was not created at %s", outputPath)
	}

	// Check content
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	if !strings.Contains(string(content), "class Cline < Formula") {
		t.Error("Generated file does not contain expected formula class")
	}
}

func TestCalculateSHA256FromFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := []byte("test content for sha256")

	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Calculate expected SHA256
	hasher := sha256.New()
	hasher.Write(testContent)
	expected := hex.EncodeToString(hasher.Sum(nil))

	// Test function
	result, err := CalculateSHA256FromFile(testFile)
	if err != nil {
		t.Fatalf("CalculateSHA256FromFile() failed: %v", err)
	}

	if result != expected {
		t.Errorf("CalculateSHA256FromFile() = %s, want %s", result, expected)
	}
}

func TestCalculateSHA256FromFileNotFound(t *testing.T) {
	_, err := CalculateSHA256FromFile("/nonexistent/path/to/file")
	if err == nil {
		t.Error("CalculateSHA256FromFile() expected error for nonexistent file")
	}
}

func TestCalculateSHA256FromBytes(t *testing.T) {
	testContent := []byte("test content")
	expectedHasher := sha256.New()
	expectedHasher.Write(testContent)
	expected := hex.EncodeToString(expectedHasher.Sum(nil))

	result := CalculateSHA256FromBytes(testContent)

	if result != expected {
		t.Errorf("CalculateSHA256FromBytes() = %s, want %s", result, expected)
	}
}

func TestDefaultHomebrewTap(t *testing.T) {
	tap := DefaultHomebrewTap()

	if tap.Name != "cline/homebrew-tap" {
		t.Errorf("Expected tap name 'cline/homebrew-tap', got '%s'", tap.Name)
	}

	if tap.FormulaDir != "Formula" {
		t.Errorf("Expected formula dir 'Formula', got '%s'", tap.FormulaDir)
	}

	if tap.BaseURL != "https://github.com/cline/cline/releases/download" {
		t.Errorf("Expected base URL 'https://github.com/cline/cline/releases/download', got '%s'", tap.BaseURL)
	}
}

func TestHomebrewTapCreateTapStructure(t *testing.T) {
	tmpDir := t.TempDir()
	tap := DefaultHomebrewTap()

	err := tap.CreateTapStructure(tmpDir)
	if err != nil {
		t.Fatalf("CreateTapStructure() failed: %v", err)
	}

	// Check Formula directory was created
	formulaDir := filepath.Join(tmpDir, "Formula")
	if _, err := os.Stat(formulaDir); os.IsNotExist(err) {
		t.Errorf("Formula directory was not created")
	}

	// Check README was created
	readmePath := filepath.Join(tmpDir, "README.md")
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		t.Errorf("README.md was not created")
	}

	// Check README content
	readmeContent, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("Failed to read README: %v", err)
	}

	if !strings.Contains(string(readmeContent), "Homebrew Tap for Cline") {
		t.Error("README does not contain expected title")
	}
}

func TestHomebrewTapGetFormulaPath(t *testing.T) {
	tap := DefaultHomebrewTap()
	rootPath := "/tmp/tap"
	formulaName := "cline"

	expected := filepath.Join(rootPath, "Formula", "cline.rb")
	result := tap.GetFormulaPath(rootPath, formulaName)

	if result != expected {
		t.Errorf("GetFormulaPath() = %s, want %s", result, expected)
	}
}

func TestGenerateHomebrewReleaseURL(t *testing.T) {
	baseURL := "https://github.com/cline/cline/releases/download"
	version := "1.0.0"
	binaryName := "cline"
	os := "darwin"
	arch := "amd64"

	expected := "https://github.com/cline/cline/releases/download/v1.0.0/cline_1.0.0_darwin_amd64.tar.gz"
	result := GenerateHomebrewReleaseURL(baseURL, version, binaryName, os, arch)

	if result != expected {
		t.Errorf("GenerateHomebrewReleaseURL() = %s, want %s", result, expected)
	}
}

func TestGenerateHomebrewPlatformURL(t *testing.T) {
	baseURL := "https://github.com/cline/cline/releases/download"
	version := "1.0.0"
	os := "linux"
	arch := "arm64"

	expected := "https://github.com/cline/cline/releases/download/v1.0.0/cline_1.0.0_linux_arm64.tar.gz"
	result := GenerateHomebrewPlatformURL(baseURL, version, os, arch)

	if result != expected {
		t.Errorf("GenerateHomebrewPlatformURL() = %s, want %s", result, expected)
	}
}

func TestHomebrewSupportedPlatforms(t *testing.T) {
	platforms := HomebrewSupportedPlatforms()

	expected := []struct {
		OS   string
		Arch string
	}{
		{"darwin", "amd64"},
		{"darwin", "arm64"},
		{"linux", "amd64"},
		{"linux", "arm64"},
	}

	if len(platforms) != len(expected) {
		t.Errorf("HomebrewSupportedPlatforms() returned %d platforms, want %d", len(platforms), len(expected))
	}

	for i, p := range platforms {
		if p.OS != expected[i].OS || p.Arch != expected[i].Arch {
			t.Errorf("Platform %d: got %s/%s, want %s/%s", i, p.OS, p.Arch, expected[i].OS, expected[i].Arch)
		}
	}
}

func TestDetectHomebrewPlatform(t *testing.T) {
	os, arch := DetectHomebrewPlatform()

	if os == "" {
		t.Error("DetectHomebrewPlatform() returned empty OS")
	}

	if arch == "" {
		t.Error("DetectHomebrewPlatform() returned empty Arch")
	}

	// Should match runtime values
	if os != runtime.GOOS {
		t.Errorf("DetectHomebrewPlatform() OS = %s, want %s", os, runtime.GOOS)
	}

	if arch != runtime.GOARCH {
		t.Errorf("DetectHomebrewPlatform() Arch = %s, want %s", arch, runtime.GOARCH)
	}
}

func TestIsHomebrewSupportedPlatform(t *testing.T) {
	tests := []struct {
		os      string
		arch    string
		support bool
	}{
		{"darwin", "amd64", true},
		{"darwin", "arm64", true},
		{"linux", "amd64", true},
		{"linux", "arm64", true},
		{"windows", "amd64", false},
		{"darwin", "386", false},
		{"freebsd", "amd64", false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s/%s", tt.os, tt.arch), func(t *testing.T) {
			result := IsHomebrewSupportedPlatform(tt.os, tt.arch)
			if result != tt.support {
				t.Errorf("IsHomebrewSupportedPlatform(%s, %s) = %v, want %v", tt.os, tt.arch, result, tt.support)
			}
		})
	}
}

func TestGetHomebrewPlatformMap(t *testing.T) {
	tests := []struct {
		os       string
		arch     string
		expected string
	}{
		{"darwin", "amd64", ":x86_64_darwin"},
		{"darwin", "arm64", ":arm64_darwin"},
		{"linux", "amd64", ":x86_64_linux"},
		{"linux", "arm64", ":arm64_linux"},
		{"windows", "amd64", ":amd64_windows"},
		{"freebsd", "arm64", ":arm64_freebsd"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s-%s", tt.os, tt.arch), func(t *testing.T) {
			result := getHomebrewPlatformMap(tt.os, tt.arch)
			if result != tt.expected {
				t.Errorf("getHomebrewPlatformMap(%s, %s) = %s, want %s", tt.os, tt.arch, result, tt.expected)
			}
		})
	}
}