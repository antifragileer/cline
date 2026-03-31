package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestGetVersionInfo(t *testing.T) {
	info := GetVersionInfo()

	if info.Version != Version {
		t.Errorf("Expected version %s, got %s", Version, info.Version)
	}
	if info.GoVersion != runtime.Version() {
		t.Errorf("Expected Go version %s, got %s", runtime.Version(), info.GoVersion)
	}
	if info.OS != runtime.GOOS {
		t.Errorf("Expected OS %s, got %s", runtime.GOOS, info.OS)
	}
	if info.Arch != runtime.GOARCH {
		t.Errorf("Expected arch %s, got %s", runtime.GOARCH, info.Arch)
	}
}

func TestOutputVersionJSON(t *testing.T) {
	info := &VersionInfo{
		Version:   "1.0.0",
		BuildDate: "2024-01-15T10:30:00Z",
		GitCommit: "abc123def456",
		GitBranch: "main",
		BuildHost: "build-server",
		GoVersion: "go1.21.0",
		OS:        "linux",
		Arch:      "amd64",
		Compiler:  "gc",
	}

	var buf bytes.Buffer
	err := outputVersionJSON(&buf, info)
	if err != nil {
		t.Errorf("outputVersionJSON returned error: %v", err)
	}

	// Verify JSON output
	var decoded VersionInfo
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Errorf("Failed to parse JSON output: %v", err)
	}

	if decoded.Version != info.Version {
		t.Errorf("Expected version %s, got %s", info.Version, decoded.Version)
	}
}

func TestOutputVersionStandard(t *testing.T) {
	info := &VersionInfo{
		Version:   "1.0.0",
		BuildDate: "2024-01-15T10:30:00Z",
		GitCommit: "abc123def456789",
		GoVersion: "go1.21.0",
		OS:        "linux",
		Arch:      "amd64",
	}

	var buf bytes.Buffer
	err := outputVersionStandard(&buf, info)
	if err != nil {
		t.Errorf("outputVersionStandard returned error: %v", err)
	}

	output := buf.String()

	// Check expected content
	if !strings.Contains(output, "Cline CLI version:") {
		t.Error("Output should contain version")
	}
	if !strings.Contains(output, "1.0.0") {
		t.Error("Output should contain version number")
	}
	if !strings.Contains(output, "abc123d") { // Short commit hash
		t.Error("Output should contain shortened commit hash")
	}
	if !strings.Contains(output, "go1.21.0") {
		t.Error("Output should contain Go version")
	}
	if !strings.Contains(output, "linux/amd64") {
		t.Error("Output should contain OS/Arch")
	}
}

func TestOutputVersionVerbose(t *testing.T) {
	info := &VersionInfo{
		Version:   "1.0.0",
		BuildDate: "2024-01-15T10:30:00Z",
		GitCommit: "abc123",
		GitBranch: "main",
		BuildHost: "build-server",
		GoVersion: "go1.21.0",
		OS:        "linux",
		Arch:      "amd64",
		Compiler:  "gc",
	}

	var buf bytes.Buffer
	err := outputVersionVerbose(&buf, info)
	if err != nil {
		t.Errorf("outputVersionVerbose returned error: %v", err)
	}

	output := buf.String()

	// Check sections
	if !strings.Contains(output, "Version Information") {
		t.Error("Output should contain 'Version Information'")
	}
	if !strings.Contains(output, "Build Information") {
		t.Error("Output should contain 'Build Information'")
	}
	if !strings.Contains(output, "Runtime Information") {
		t.Error("Output should contain 'Runtime Information'")
	}
	if !strings.Contains(output, "NumCPU") {
		t.Error("Output should contain NumCPU")
	}
}

func TestFormatCommit(t *testing.T) {
	tests := []struct {
		commit   string
		expected string
	}{
		{"abc123def456789", "abc123d"},
		{"short", "short"},
		{"", ""},
		{"abcdefg", "abcdefg"},
		{"abcdefgh", "abcdefg"},
	}

	for _, tt := range tests {
		t.Run(tt.commit, func(t *testing.T) {
			result := formatCommit(tt.commit)
			if result != tt.expected {
				t.Errorf("formatCommit(%q) = %q, want %q", tt.commit, result, tt.expected)
			}
		})
	}
}

func TestFormatBuildDate(t *testing.T) {
	tests := []struct {
		date     string
		expected string
	}{
		{"2024-01-15T10:30:00Z", "Jan 15, 2024 10:30:00 UTC"},
		{"2024-01-15T10:30:00.000Z", "Jan 15, 2024 10:30:00 UTC"},
		{"2024-01-15 10:30:00", "Jan 15, 2024 10:30:00 UTC"},
		{"invalid", "invalid"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.date, func(t *testing.T) {
			result := formatBuildDate(tt.date)
			if result != tt.expected && tt.date != "invalid" && tt.date != "" {
				t.Errorf("formatBuildDate(%q) = %q, want %q", tt.date, result, tt.expected)
			}
		})
	}
}

func TestPrintVersion(t *testing.T) {
	var buf bytes.Buffer
	PrintVersion(&buf)

	output := buf.String()
	if !strings.Contains(output, "Cline CLI version") {
		t.Error("Output should contain 'Cline CLI version'")
	}
	if !strings.Contains(output, Version) {
		t.Error("Output should contain version number")
	}
}

func TestGetShortVersion(t *testing.T) {
	version := GetShortVersion()
	if version != Version {
		t.Errorf("GetShortVersion() = %q, want %q", version, Version)
	}
}

func TestGetBuildInfo(t *testing.T) {
	info := GetBuildInfo()

	if info["version"] != Version {
		t.Errorf("Expected version %s, got %s", Version, info["version"])
	}
	if info["os"] != runtime.GOOS {
		t.Errorf("Expected OS %s, got %s", runtime.GOOS, info["os"])
	}
	if info["arch"] != runtime.GOARCH {
		t.Errorf("Expected arch %s, got %s", runtime.GOARCH, info["arch"])
	}
}

func TestIsDevelopmentBuild(t *testing.T) {
	// Save original values
	origCommit := GitCommit
	origDate := BuildDate
	defer func() {
		GitCommit = origCommit
		BuildDate = origDate
	}()

	// Test with unknown values
	GitCommit = "unknown"
	BuildDate = "unknown"
	if !IsDevelopmentBuild() {
		t.Error("Should be development build with unknown values")
	}

	// Test with known commit but unknown date
	GitCommit = "abc123"
	BuildDate = "unknown"
	if !IsDevelopmentBuild() {
		t.Error("Should be development build with unknown date")
	}

	// Test with known values
	GitCommit = "abc123"
	BuildDate = "2024-01-15T10:30:00Z"
	if IsDevelopmentBuild() {
		t.Error("Should not be development build with known values")
	}
}

func TestCheckVersionCompatibility(t *testing.T) {
	tests := []struct {
		current  string
		minimum  string
		expected bool
	}{
		{"1.0.0", "0.9.0", true},
		{"1.0.0", "1.0.0", true},
		{"1.0.0", "1.0.1", false},
		{"1.0.0", "1.1.0", false},
		{"1.0.0", "2.0.0", false},
		{"2.0.0", "1.0.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.current+"_"+tt.minimum, func(t *testing.T) {
			result := CheckVersionCompatibilityWith(tt.current, tt.minimum)
			if result != tt.expected {
				t.Errorf("CheckVersionCompatibility(%q) with current %q = %v, want %v",
					tt.minimum, tt.current, result, tt.expected)
			}
		})
	}
}

func TestParseVersionString(t *testing.T) {
	tests := []struct {
		version  string
		expected [3]int
	}{
		{"1.2.3", [3]int{1, 2, 3}},
		{"v1.2.3", [3]int{1, 2, 3}},
		{"1.2.3-beta", [3]int{1, 2, 3}},
		{"2.0", [3]int{2, 0, 0}},
		{"3", [3]int{3, 0, 0}},
		{"", [3]int{0, 0, 0}},
		{"invalid", [3]int{0, 0, 0}},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			result := parseVersionString(tt.version)
			if result != tt.expected {
				t.Errorf("parseVersionString(%q) = %v, want %v", tt.version, result, tt.expected)
			}
		})
	}
}

func TestWriteVersionInfo(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cline-version-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	path := filepath.Join(tempDir, "version.json")
	err = WriteVersionInfo(path)
	if err != nil {
		t.Errorf("WriteVersionInfo returned error: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("Version file was not created")
	}

	// Read and verify content
	info, err := ReadVersionInfo(path)
	if err != nil {
		t.Errorf("ReadVersionInfo returned error: %v", err)
	}

	if info.Version != Version {
		t.Errorf("Expected version %s, got %s", Version, info.Version)
	}
}

func TestReadVersionInfoInvalidFile(t *testing.T) {
	// Test with non-existent file
	_, err := ReadVersionInfo("/nonexistent/path/version.json")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}

	// Test with invalid JSON
	tempFile, err := os.CreateTemp("", "invalid-version-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	tempFile.WriteString("invalid json")
	tempFile.Close()

	_, err = ReadVersionInfo(tempFile.Name())
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestVersionInfoStructure(t *testing.T) {
	info := &VersionInfo{
		Version:   "1.0.0",
		BuildDate: time.Now().Format(time.RFC3339),
		GitCommit: "abc123",
		GitBranch: "main",
		BuildHost: "ci-server",
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		Compiler:  runtime.Compiler,
	}

	// Verify all fields are set
	if info.Version == "" {
		t.Error("Version should not be empty")
	}
	if info.BuildDate == "" {
		t.Error("BuildDate should not be empty")
	}
	if info.GitCommit == "" {
		t.Error("GitCommit should not be empty")
	}
	if info.GitBranch == "" {
		t.Error("GitBranch should not be empty")
	}
	if info.BuildHost == "" {
		t.Error("BuildHost should not be empty")
	}
	if info.GoVersion == "" {
		t.Error("GoVersion should not be empty")
	}
	if info.OS == "" {
		t.Error("OS should not be empty")
	}
	if info.Arch == "" {
		t.Error("Arch should not be empty")
	}
	if info.Compiler == "" {
		t.Error("Compiler should not be empty")
	}
}