package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestDefaultBuildConfig(t *testing.T) {
	config := DefaultBuildConfig()

	if config.BinaryName != "cline" {
		t.Errorf("Expected BinaryName to be 'cline', got '%s'", config.BinaryName)
	}

	if config.Version != "dev" {
		t.Errorf("Expected Version to be 'dev', got '%s'", config.Version)
	}

	if config.GitCommit != "unknown" {
		t.Errorf("Expected GitCommit to be 'unknown', got '%s'", config.GitCommit)
	}

	if config.OutputDir != "build" {
		t.Errorf("Expected OutputDir to be 'build', got '%s'", config.OutputDir)
	}

	if config.SourcePath != "./cmd/cline" {
		t.Errorf("Expected SourcePath to be './cmd/cline', got '%s'", config.SourcePath)
	}

	if config.CGOEnabled != "0" {
		t.Errorf("Expected CGOEnabled to be '0', got '%s'", config.CGOEnabled)
	}

	if config.GOOS != runtime.GOOS {
		t.Errorf("Expected GOOS to be '%s', got '%s'", runtime.GOOS, config.GOOS)
	}

	if config.GOARCH != runtime.GOARCH {
		t.Errorf("Expected GOARCH to be '%s', got '%s'", runtime.GOARCH, config.GOARCH)
	}

	// BuildTime should be set to current time
	if config.BuildTime == "" {
		t.Error("Expected BuildTime to be set")
	}
}

func TestBuildPlatformString(t *testing.T) {
	tests := []struct {
		platform BuildPlatform
		expected string
	}{
		{BuildPlatform{OS: "linux", Arch: "amd64"}, "linux/amd64"},
		{BuildPlatform{OS: "darwin", Arch: "arm64"}, "darwin/arm64"},
		{BuildPlatform{OS: "windows", Arch: "amd64"}, "windows/amd64"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.platform.String()
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestBuildPlatformBinaryName(t *testing.T) {
	tests := []struct {
		platform BuildPlatform
		baseName string
		expected string
	}{
		{BuildPlatform{OS: "linux", Arch: "amd64"}, "cline", "cline"},
		{BuildPlatform{OS: "darwin", Arch: "arm64"}, "cline", "cline"},
		{BuildPlatform{OS: "windows", Arch: "amd64"}, "cline", "cline.exe"},
		{BuildPlatform{OS: "windows", Arch: "amd64"}, "myapp", "myapp.exe"},
	}

	for _, tt := range tests {
		t.Run(tt.platform.String(), func(t *testing.T) {
			result := tt.platform.BinaryName(tt.baseName)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestBuildPlatformArchiveExtension(t *testing.T) {
	tests := []struct {
		platform BuildPlatform
		expected string
	}{
		{BuildPlatform{OS: "linux", Arch: "amd64"}, ".tar.gz"},
		{BuildPlatform{OS: "darwin", Arch: "arm64"}, ".tar.gz"},
		{BuildPlatform{OS: "windows", Arch: "amd64"}, ".zip"},
	}

	for _, tt := range tests {
		t.Run(tt.platform.String(), func(t *testing.T) {
			result := tt.platform.ArchiveExtension()
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestBuildSupportedPlatforms(t *testing.T) {
	platforms := BuildSupportedPlatforms()

	expectedPlatforms := []BuildPlatform{
		{OS: "linux", Arch: "amd64"},
		{OS: "linux", Arch: "arm64"},
		{OS: "darwin", Arch: "amd64"},
		{OS: "darwin", Arch: "arm64"},
		{OS: "windows", Arch: "amd64"},
	}

	if len(platforms) != len(expectedPlatforms) {
		t.Errorf("Expected %d platforms, got %d", len(expectedPlatforms), len(platforms))
	}

	for i, expected := range expectedPlatforms {
		if platforms[i] != expected {
			t.Errorf("Platform %d: expected %+v, got %+v", i, expected, platforms[i])
		}
	}
}

func TestValidateBuildPlatform(t *testing.T) {
	tests := []struct {
		os       string
		arch     string
		wantErr  bool
		errMsg   string
	}{
		{"linux", "amd64", false, ""},
		{"linux", "arm64", false, ""},
		{"darwin", "amd64", false, ""},
		{"darwin", "arm64", false, ""},
		{"windows", "amd64", false, ""},
		{"freebsd", "amd64", true, "unsupported platform: freebsd/amd64"},
		{"linux", "386", true, "unsupported platform: linux/386"},
		{"", "amd64", true, "unsupported platform: /amd64"},
	}

	for _, tt := range tests {
		t.Run(tt.os+"_"+tt.arch, func(t *testing.T) {
			err := ValidateBuildPlatform(tt.os, tt.arch)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got '%s'", err.Error())
				}
			}
		})
	}
}

func TestParseBuildPlatform(t *testing.T) {
	tests := []struct {
		input       string
		expectedOS  string
		expectedArch string
		wantErr     bool
	}{
		{"linux/amd64", "linux", "amd64", false},
		{"darwin/arm64", "darwin", "arm64", false},
		{"windows/amd64", "windows", "amd64", false},
		{"invalid", "", "", true},
		{"too/many/parts", "", "", true},
		{"freebsd/amd64", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			platform, err := ParseBuildPlatform(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got '%s'", err.Error())
				}
				if platform.OS != tt.expectedOS {
					t.Errorf("Expected OS '%s', got '%s'", tt.expectedOS, platform.OS)
				}
				if platform.Arch != tt.expectedArch {
					t.Errorf("Expected Arch '%s', got '%s'", tt.expectedArch, platform.Arch)
				}
			}
		})
	}
}

func TestBuildConfigDefaultLDFlags(t *testing.T) {
	config := &BuildConfig{
		Version:   "1.0.0",
		GitCommit: "abc123",
		BuildTime: "2024-01-01T00:00:00Z",
	}

	ldflags := config.DefaultLDFlags()

	expectedParts := []string{
		"-s -w",
		"-X github.com/cline/cline/golang-cli/cmd/cline.Version=1.0.0",
		"-X github.com/cline/cline/golang-cli/cmd/cline.BuildTime=2024-01-01T00:00:00Z",
		"-X github.com/cline/cline/golang-cli/cmd/cline.GitCommit=abc123",
	}

	for _, part := range expectedParts {
		if !strings.Contains(ldflags, part) {
			t.Errorf("Expected ldflags to contain '%s', got '%s'", part, ldflags)
		}
	}
}

func TestBuildConfigBuildValidation(t *testing.T) {
	config := &BuildConfig{
		BinaryName: "",
	}

	err := config.Build()
	if err == nil {
		t.Error("Expected error for empty BinaryName, got nil")
	}

	if !strings.Contains(err.Error(), "binary name cannot be empty") {
		t.Errorf("Expected error about empty binary name, got '%s'", err.Error())
	}
}

func TestGenerateChecksum(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("hello world")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Generate checksum
	checksum, err := GenerateChecksum(testFile)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Expected SHA256 for "hello world"
	expected := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	if checksum != expected {
		t.Errorf("Expected checksum '%s', got '%s'", expected, checksum)
	}

	// Test non-existent file
	_, err = GenerateChecksum(filepath.Join(tmpDir, "nonexistent.txt"))
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestWriteChecksum(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	checksum := "abc123"

	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err := WriteChecksum(testFile, checksum)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	checksumFile := testFile + ".sha256"
	data, err := os.ReadFile(checksumFile)
	if err != nil {
		t.Errorf("Failed to read checksum file: %v", err)
	}

	expected := checksum + "\n"
	if string(data) != expected {
		t.Errorf("Expected '%s', got '%s'", expected, string(data))
	}
}

func TestVerifyChecksum(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("hello world")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Correct checksum
	expectedChecksum := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	err := VerifyChecksum(testFile, expectedChecksum)
	if err != nil {
		t.Errorf("Unexpected error for valid checksum: %v", err)
	}

	// Incorrect checksum
	err = VerifyChecksum(testFile, "wrongchecksum")
	if err == nil {
		t.Error("Expected error for invalid checksum, got nil")
	}

	// Non-existent file
	err = VerifyChecksum(filepath.Join(tmpDir, "nonexistent.txt"), expectedChecksum)
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestGetVersionInfo(t *testing.T) {
	// Set environment variables
	t.Setenv("VERSION", "1.2.3")
	t.Setenv("GIT_COMMIT", "def456")
	t.Setenv("BUILD_TIME", "2024-06-15T12:00:00Z")

	info := GetVersionInfo()

	if info.Version != "1.2.3" {
		t.Errorf("Expected Version '1.2.3', got '%s'", info.Version)
	}

	if info.GitCommit != "def456" {
		t.Errorf("Expected GitCommit 'def456', got '%s'", info.GitCommit)
	}

	if info.BuildTime != "2024-06-15T12:00:00Z" {
		t.Errorf("Expected BuildTime '2024-06-15T12:00:00Z', got '%s'", info.BuildTime)
	}

	if len(info.Platforms) != len(BuildSupportedPlatforms()) {
		t.Errorf("Expected %d platforms, got %d", len(BuildSupportedPlatforms()), len(info.Platforms))
	}
}

func TestGetVersionInfoDefaults(t *testing.T) {
	// Unset environment variables
	t.Setenv("VERSION", "")
	t.Setenv("GIT_COMMIT", "")
	t.Setenv("BUILD_TIME", "")

	info := GetVersionInfo()

	if info.Version != "dev" {
		t.Errorf("Expected default Version 'dev', got '%s'", info.Version)
	}

	if info.GitCommit != "unknown" {
		t.Errorf("Expected default GitCommit 'unknown', got '%s'", info.GitCommit)
	}

	if info.BuildTime == "" {
		t.Error("Expected BuildTime to be set to current time")
	}
}

func TestClean(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "build")

	// Create the directory and a file
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("Failed to create output dir: %v", err)
	}
	testFile := filepath.Join(outputDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Verify directory exists
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		t.Fatal("Output dir should exist")
	}

	// Clean
	err := Clean(outputDir)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify directory is removed
	if _, err := os.Stat(outputDir); !os.IsNotExist(err) {
		t.Error("Output dir should be removed")
	}
}

func TestArchiveName(t *testing.T) {
	tests := []struct {
		binaryName string
		version    string
		platform   BuildPlatform
		expected   string
	}{
		{"cline", "1.0.0", BuildPlatform{OS: "linux", Arch: "amd64"}, "cline-1.0.0-linux-amd64.tar.gz"},
		{"cline", "1.0.0", BuildPlatform{OS: "darwin", Arch: "arm64"}, "cline-1.0.0-darwin-arm64.tar.gz"},
		{"cline", "1.0.0", BuildPlatform{OS: "windows", Arch: "amd64"}, "cline-1.0.0-windows-amd64.zip"},
		{"myapp", "2.0.0", BuildPlatform{OS: "linux", Arch: "arm64"}, "myapp-2.0.0-linux-arm64.tar.gz"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := ArchiveName(tt.binaryName, tt.version, tt.platform)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestIsSupportedOS(t *testing.T) {
	tests := []struct {
		os       string
		expected bool
	}{
		{"linux", true},
		{"darwin", true},
		{"windows", true},
		{"freebsd", false},
		{"openbsd", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.os, func(t *testing.T) {
			result := IsSupportedOS(tt.os)
			if result != tt.expected {
				t.Errorf("Expected %v for OS '%s', got %v", tt.expected, tt.os, result)
			}
		})
	}
}

func TestIsSupportedArch(t *testing.T) {
	tests := []struct {
		os       string
		arch     string
		expected bool
	}{
		{"linux", "amd64", true},
		{"linux", "arm64", true},
		{"darwin", "amd64", true},
		{"darwin", "arm64", true},
		{"windows", "amd64", true},
		{"linux", "386", false},
		{"darwin", "386", false},
		{"freebsd", "amd64", false},
		{"", "amd64", false},
	}

	for _, tt := range tests {
		t.Run(tt.os+"_"+tt.arch, func(t *testing.T) {
			result := IsSupportedArch(tt.os, tt.arch)
			if result != tt.expected {
				t.Errorf("Expected %v for %s/%s, got %v", tt.expected, tt.os, tt.arch, result)
			}
		})
	}
}

func TestVersionInfoStruct(t *testing.T) {
	info := &VersionInfo{
		Version:   "1.0.0",
		GitCommit: "abc123",
		BuildTime: time.Now().UTC().Format(time.RFC3339),
		Platforms: []string{"linux/amd64", "darwin/arm64"},
	}

	if info.Version != "1.0.0" {
		t.Errorf("Expected Version '1.0.0', got '%s'", info.Version)
	}

	if len(info.Platforms) != 2 {
		t.Errorf("Expected 2 platforms, got %d", len(info.Platforms))
	}
}

func TestCrossCompileEmptyConfig(t *testing.T) {
	config := &BuildConfig{
		BinaryName: "test",
		Version:    "1.0.0",
		GitCommit:  "abc123",
		OutputDir:  t.TempDir(),
		SourcePath: "./nonexistent", // This will cause build to fail
	}

	// This should return errors for each platform since source path doesn't exist
	built, err := CrossCompile(config)

	// We expect some errors since the source path is invalid
	if err == nil {
		t.Error("Expected errors for invalid source path")
	}

	// Some platforms might succeed (unlikely with invalid source), but most should fail
	if len(built) == len(BuildSupportedPlatforms()) {
		t.Error("Expected some build failures with invalid source path")
	}
}

func TestBuildConfigWithLDFlags(t *testing.T) {
	config := &BuildConfig{
		BinaryName: "cline",
		Version:    "1.0.0",
		LDFlags:    "-custom-flag",
	}

	ldflags := config.LDFlags
	if ldflags != "-custom-flag" {
		t.Errorf("Expected custom ldflags '-custom-flag', got '%s'", ldflags)
	}

	// When LDFlags is empty, DefaultLDFlags should be used
	config.LDFlags = ""
	ldflags = config.DefaultLDFlags()
	if !strings.Contains(ldflags, "Version=1.0.0") {
		t.Errorf("Expected DefaultLDFlags to include version info, got '%s'", ldflags)
	}
}

func TestBuildCreatesOutputDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "nested", "build", "dir")

	config := &BuildConfig{
		BinaryName: "test",
		OutputDir:  outputDir,
		GOOS:       runtime.GOOS,
		GOARCH:     runtime.GOARCH,
		SourcePath: "./nonexistent", // Will fail but should create dir first
	}

	// Try to build (will fail due to invalid source)
	_ = config.Build()

	// But the directory should have been created
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		t.Error("Expected output directory to be created")
	}
}

func TestBuildPlatformMethods(t *testing.T) {
	p := BuildPlatform{OS: "linux", Arch: "amd64"}

	// Test String()
	if p.String() != "linux/amd64" {
		t.Errorf("Expected String() to return 'linux/amd64', got '%s'", p.String())
	}

	// Test BinaryName()
	if p.BinaryName("app") != "app" {
		t.Errorf("Expected BinaryName('app') to return 'app', got '%s'", p.BinaryName("app"))
	}

	// Test ArchiveExtension()
	if p.ArchiveExtension() != ".tar.gz" {
		t.Errorf("Expected ArchiveExtension() to return '.tar.gz', got '%s'", p.ArchiveExtension())
	}
}

func TestChecksumRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("round trip test content")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Generate checksum
	checksum, err := GenerateChecksum(testFile)
	if err != nil {
		t.Fatalf("Failed to generate checksum: %v", err)
	}

	// Write checksum to file
	if err := WriteChecksum(testFile, checksum); err != nil {
		t.Fatalf("Failed to write checksum: %v", err)
	}

	// Verify checksum
	if err := VerifyChecksum(testFile, checksum); err != nil {
		t.Errorf("Failed to verify checksum: %v", err)
	}

	// Verify with whitespace
	if err := VerifyChecksum(testFile, checksum+"\n"); err != nil {
		t.Errorf("Failed to verify checksum with whitespace: %v", err)
	}
}

func TestBuildPlatformCount(t *testing.T) {
	platforms := BuildSupportedPlatforms()
	expectedCount := 5 // linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64

	if len(platforms) != expectedCount {
		t.Errorf("Expected %d supported platforms, got %d", expectedCount, len(platforms))
	}

	// Ensure all expected platforms are present
	expectedPlatforms := map[string]bool{
		"linux/amd64":   false,
		"linux/arm64":   false,
		"darwin/amd64":  false,
		"darwin/arm64":  false,
		"windows/amd64": false,
	}

	for _, p := range platforms {
		key := p.String()
		if _, exists := expectedPlatforms[key]; !exists {
			t.Errorf("Unexpected platform: %s", key)
		}
		expectedPlatforms[key] = true
	}

	for key, found := range expectedPlatforms {
		if !found {
			t.Errorf("Missing expected platform: %s", key)
		}
	}
}

func TestBuildConfigCopy(t *testing.T) {
	original := DefaultBuildConfig()
	copy := *original

	// Modify copy
	copy.Version = "modified"

	// Original should be unchanged
	if original.Version != "dev" {
		t.Errorf("Original config was modified: expected 'dev', got '%s'", original.Version)
	}

	if copy.Version != "modified" {
		t.Errorf("Copy was not modified: expected 'modified', got '%s'", copy.Version)
	}
}

func TestEmptyBuildPlatformParse(t *testing.T) {
	_, err := ParseBuildPlatform("")
	if err == nil {
		t.Error("Expected error for empty platform string")
	}
}

func TestInvalidBuildPlatformFormat(t *testing.T) {
	invalidFormats := []string{
		"linux",
		"linux/amd64/extra",
		"/amd64",
		"linux/",
	}

	for _, format := range invalidFormats {
		t.Run(format, func(t *testing.T) {
			_, err := ParseBuildPlatform(format)
			if err == nil {
				t.Errorf("Expected error for invalid format '%s'", format)
			}
		})
	}
}

func TestWindowsBinaryExtension(t *testing.T) {
	win := BuildPlatform{OS: "windows", Arch: "amd64"}
	linux := BuildPlatform{OS: "linux", Arch: "amd64"}
	darwin := BuildPlatform{OS: "darwin", Arch: "arm64"}

	if win.BinaryName("app") != "app.exe" {
		t.Errorf("Expected Windows binary to have .exe extension")
	}

	if linux.BinaryName("app") != "app" {
		t.Errorf("Expected Linux binary to have no extension")
	}

	if darwin.BinaryName("app") != "app" {
		t.Errorf("Expected Darwin binary to have no extension")
	}
}

func TestVersionInfoDefaultTime(t *testing.T) {
	before := time.Now().UTC().Format(time.RFC3339)
	info := GetVersionInfo()
	after := time.Now().UTC().Format(time.RFC3339)

	// Check that BuildTime is a valid RFC3339 time between before and after
	infoTime, err := time.Parse(time.RFC3339, info.BuildTime)
	if err != nil {
		t.Errorf("BuildTime is not valid RFC3339: %v", err)
	}

	beforeTime, _ := time.Parse(time.RFC3339, before)
	afterTime, _ := time.Parse(time.RFC3339, after)

	if infoTime.Before(beforeTime) || infoTime.After(afterTime) {
		t.Error("BuildTime is not within expected time range")
	}
}