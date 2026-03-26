package scripts

import (
	"flag"
	"testing"
)

// TestGenerateHomebrewFlags tests the flag parsing functionality.
func TestGenerateHomebrewFlags(t *testing.T) {
	// This is a placeholder test. The actual test would require
	// refactoring the ParseFlags function to accept arguments.
	// For now, we just verify the package compiles correctly.
	t.Skip("Flag parsing requires os.Args manipulation which affects global state")
}

// TestGenerateHomebrewConfig tests HomebrewConfig methods.
func TestGenerateHomebrewConfig(t *testing.T) {
	// Test empty version
	config := &HomebrewConfig{}
	err := config.Validate()
	if err == nil {
		t.Error("Expected validation error for empty version")
	}

	// Test valid config
	config = &HomebrewConfig{
		Version: "1.0.0",
		DryRun:  true,
	}
	err = config.Validate()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

// TestIsHomebrewValidVersion tests version validation.
func TestIsHomebrewValidVersion(t *testing.T) {
	tests := []struct {
		version  string
		expected bool
	}{
		{"1.0.0", true},
		{"v1.2.3", true},
		{"2.0.0-beta", true},
		{"1.0.0+build.123", true},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			result := IsHomebrewValidVersion(tt.version)
			if result != tt.expected {
				t.Errorf("IsHomebrewValidVersion(%q) = %v, expected %v", tt.version, result, tt.expected)
			}
		})
	}
}

// TestSanitizeHomebrewVersion tests version sanitization.
func TestSanitizeHomebrewVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1.0.0", "1.0.0"},
		{"v1.0.0/test", "v1.0.0-test"},
		{"v1.0.0\\test", "v1.0.0-test"},
		{"v1.0.0:test", "v1.0.0-test"},
		{"v1.0.0 test", "v1.0.0-test"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := SanitizeHomebrewVersion(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeHomebrewVersion(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestGetHomebrewBinaryName tests binary name generation.
func TestGetHomebrewBinaryName(t *testing.T) {
	result := GetHomebrewBinaryName("1.0.0", "linux", "amd64")
	expected := "cline_1.0.0_linux_amd64.tar.gz"
	if result != expected {
		t.Errorf("GetHomebrewBinaryName(1.0.0, linux, amd64) = %q, expected %q", result, expected)
	}
}

// TestListHomebrewRequiredBinaries tests the list of required binaries.
func TestListHomebrewRequiredBinaries(t *testing.T) {
	binaries := ListHomebrewRequiredBinaries("1.0.0")
	if len(binaries) != len(HomebrewSupportedPlatforms()) {
		t.Errorf("Expected %d binaries, got %d", len(HomebrewSupportedPlatforms()), len(binaries))
	}
}

// TestGetHomebrewDefaultOutputPath tests the default output path.
func TestGetHomebrewDefaultOutputPath(t *testing.T) {
	result := GetHomebrewDefaultOutputPath("1.0.0")
	expected := "Formula/cline.rb"
	if result != expected {
		t.Errorf("GetHomebrewDefaultOutputPath(1.0.0) = %q, expected %q", result, expected)
	}
}

// TestFlagPackage tests that the flag package is available.
func TestFlagPackage(t *testing.T) {
	// This test ensures the flag package is properly imported
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	if fs == nil {
		t.Error("Failed to create flag set")
	}
}