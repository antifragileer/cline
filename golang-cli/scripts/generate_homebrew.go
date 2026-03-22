// Package scripts provides build and distribution utilities for the Cline CLI.
// This file implements the CLI tool for generating Homebrew formulas.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// HomebrewConfig holds the configuration for generating a Homebrew formula.
type HomebrewConfig struct {
	// Version is the release version
	Version string

	// OutputPath is where to write the formula file
	OutputPath string

	// BaseURL is the base URL for release downloads
	BaseURL string

	// BinaryDir is the directory containing pre-built binaries
	BinaryDir string

	// FormulaName is the name of the formula
	FormulaName string

	// TapDir is the directory to create tap structure in (optional)
	TapDir string

	// LocalMode uses local files for SHA256 calculation instead of downloading
	LocalMode bool

	// DryRun prints the formula without writing to file
	DryRun bool
}

// ParseFlags parses command-line flags and returns a HomebrewConfig.
func ParseFlags() *HomebrewConfig {
	config := &HomebrewConfig{}

	flag.StringVar(&config.Version, "version", "", "Release version (required)")
	flag.StringVar(&config.OutputPath, "output", "", "Output path for formula file")
	flag.StringVar(&config.BaseURL, "base-url", "https://github.com/cline/cline/releases/download", "Base URL for release downloads")
	flag.StringVar(&config.BinaryDir, "binary-dir", "", "Directory containing pre-built binaries for local SHA256 calculation")
	flag.StringVar(&config.FormulaName, "name", "cline", "Formula name")
	flag.StringVar(&config.TapDir, "tap-dir", "", "Create tap structure in this directory")
	flag.BoolVar(&config.LocalMode, "local", false, "Use local files for SHA256 calculation")
	flag.BoolVar(&config.DryRun, "dry-run", false, "Print formula without writing to file")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Generate Homebrew formula for Cline CLI distribution.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Generate formula for a release\n")
		fmt.Fprintf(os.Stderr, "  %s -version 1.0.0 -output Formula/cline.rb\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Generate formula using local binaries\n")
		fmt.Fprintf(os.Stderr, "  %s -version 1.0.0 -local -binary-dir ./dist -output Formula/cline.rb\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Create complete tap structure\n")
		fmt.Fprintf(os.Stderr, "  %s -version 1.0.0 -tap-dir ./homebrew-tap\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Dry run to preview formula\n")
		fmt.Fprintf(os.Stderr, "  %s -version 1.0.0 -dry-run\n", os.Args[0])
	}

	flag.Parse()

	return config
}

// Validate checks if the configuration is valid.
func (c *HomebrewConfig) Validate() error {
	if c.Version == "" {
		return fmt.Errorf("version is required (use -version flag)")
	}

	if !c.DryRun && c.OutputPath == "" && c.TapDir == "" {
		return fmt.Errorf("either -output, -tap-dir, or -dry-run must be specified")
	}

	if c.LocalMode && c.BinaryDir == "" {
		return fmt.Errorf("binary-dir is required when using local mode")
	}

	return nil
}

// GenerateHomebrew generates the Homebrew formula based on configuration.
func GenerateHomebrew(config *HomebrewConfig) (*HomebrewFormula, error) {
	formula := DefaultHomebrewFormula(config.Version)
	if config.FormulaName != "" {
		formula.Name = config.FormulaName
	}

	// Generate URLs and SHA256 for each platform
	platforms := HomebrewSupportedPlatforms()

	for _, p := range platforms {
		url := GenerateHomebrewPlatformURL(config.BaseURL, config.Version, p.OS, p.Arch)

		if config.LocalMode {
			// Use local file for SHA256 calculation
			binaryName := fmt.Sprintf("cline_%s_%s_%s.tar.gz", config.Version, p.OS, p.Arch)
			localPath := filepath.Join(config.BinaryDir, binaryName)

			if _, err := os.Stat(localPath); os.IsNotExist(err) {
				return nil, fmt.Errorf("local binary not found: %s", localPath)
			}

			if err := formula.AddPlatformWithLocalFile(p.OS, p.Arch, url, localPath); err != nil {
				return nil, fmt.Errorf("failed to add platform %s/%s: %w", p.OS, p.Arch, err)
			}
		} else {
			// Download and calculate SHA256
			fmt.Fprintf(os.Stderr, "Downloading %s/%s binary for SHA256 calculation...\n", p.OS, p.Arch)
			if err := formula.AddPlatform(p.OS, p.Arch, url); err != nil {
				return nil, fmt.Errorf("failed to add platform %s/%s: %w", p.OS, p.Arch, err)
			}
		}
	}

	return formula, nil
}

// RunHomebrew executes the Homebrew formula generation.
func RunHomebrew(config *HomebrewConfig) error {
	if err := config.Validate(); err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	// Create tap structure if requested
	if config.TapDir != "" {
		tap := DefaultHomebrewTap()
		fmt.Fprintf(os.Stderr, "Creating tap structure in %s...\n", config.TapDir)
		if err := tap.CreateTapStructure(config.TapDir); err != nil {
			return fmt.Errorf("failed to create tap structure: %w", err)
		}
	}

	// Generate the formula
	fmt.Fprintf(os.Stderr, "Generating formula for version %s...\n", config.Version)
	formula, err := GenerateHomebrew(config)
	if err != nil {
		return err
	}

	// Validate the formula
	if err := formula.Validate(); err != nil {
		return fmt.Errorf("formula validation failed: %w", err)
	}

	// Generate formula content
	content, err := formula.Generate()
	if err != nil {
		return fmt.Errorf("failed to generate formula: %w", err)
	}

	// Output the result
	if config.DryRun {
		fmt.Println(content)
		return nil
	}

	// Determine output path
	outputPath := config.OutputPath
	if outputPath == "" && config.TapDir != "" {
		tap := DefaultHomebrewTap()
		outputPath = tap.GetFormulaPath(config.TapDir, formula.Name)
	}

	// Write to file
	fmt.Fprintf(os.Stderr, "Writing formula to %s...\n", outputPath)
	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write formula: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Successfully generated Homebrew formula: %s\n", outputPath)
	return nil
}

func main() {
	config := ParseFlags()

	if err := RunHomebrew(config); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// GenerateHomebrewFormulaFromRelease generates a formula for a specific release version.
// This is a convenience function for programmatic use.
func GenerateHomebrewFormulaFromRelease(version, outputPath, baseURL string) error {
	config := &HomebrewConfig{
		Version:    version,
		OutputPath: outputPath,
		BaseURL:    baseURL,
	}

	return RunHomebrew(config)
}

// GenerateHomebrewFormulaFromLocalBinaries generates a formula using local pre-built binaries.
// This is useful in CI/CD pipelines where binaries are already built.
func GenerateHomebrewFormulaFromLocalBinaries(version, binaryDir, outputPath string) error {
	config := &HomebrewConfig{
		Version:    version,
		OutputPath: outputPath,
		BinaryDir:  binaryDir,
		LocalMode:  true,
	}

	return RunHomebrew(config)
}

// CreateHomebrewTap creates a complete Homebrew tap structure.
func CreateHomebrewTap(tapDir, version string, localMode bool, binaryDir string) error {
	config := &HomebrewConfig{
		Version:   version,
		TapDir:    tapDir,
		LocalMode: localMode,
		BinaryDir: binaryDir,
	}

	return RunHomebrew(config)
}

// PrintHomebrewFormula prints the formula to stdout without writing to file.
func PrintHomebrewFormula(version string, baseURL string) error {
	config := &HomebrewConfig{
		Version: version,
		BaseURL: baseURL,
		DryRun:  true,
	}

	return RunHomebrew(config)
}

// GetHomebrewDefaultOutputPath returns the default output path for the formula.
func GetHomebrewDefaultOutputPath(version string) string {
	return filepath.Join("Formula", "cline.rb")
}

// IsHomebrewValidVersion checks if a version string is valid.
func IsHomebrewValidVersion(version string) bool {
	// Simple version validation - must not be empty and should follow semver-like format
	if version == "" {
		return false
	}

	// Check for valid characters
	for _, c := range version {
		if !isHomebrewValidVersionChar(c) {
			return false
		}
	}

	return true
}

func isHomebrewValidVersionChar(c rune) bool {
	return (c >= '0' && c <= '9') || c == '.' || c == '-' || c == '+' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// SanitizeHomebrewVersion sanitizes a version string for use in filenames.
func SanitizeHomebrewVersion(version string) string {
	// Replace characters that might be problematic in filenames
	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "-",
		" ", "-",
	)
	return replacer.Replace(version)
}

// GetHomebrewBinaryName generates the expected binary name for a platform.
func GetHomebrewBinaryName(version, os, arch string) string {
	return fmt.Sprintf("cline_%s_%s_%s.tar.gz", version, os, arch)
}

// GetHomebrewBinaryPath generates the full path to a binary.
func GetHomebrewBinaryPath(binaryDir, version, os, arch string) string {
	return filepath.Join(binaryDir, GetHomebrewBinaryName(version, os, arch))
}

// CheckHomebrewBinariesExist checks if all required binaries exist in the binary directory.
func CheckHomebrewBinariesExist(binaryDir, version string) error {
	platforms := HomebrewSupportedPlatforms()
	var missing []string

	for _, p := range platforms {
		path := GetHomebrewBinaryPath(binaryDir, version, p.OS, p.Arch)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			missing = append(missing, GetHomebrewBinaryName(version, p.OS, p.Arch))
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing binaries: %s", strings.Join(missing, ", "))
	}

	return nil
}

// ListHomebrewRequiredBinaries returns the list of required binary names for a version.
func ListHomebrewRequiredBinaries(version string) []string {
	platforms := HomebrewSupportedPlatforms()
	binaries := make([]string, len(platforms))

	for i, p := range platforms {
		binaries[i] = GetHomebrewBinaryName(version, p.OS, p.Arch)
	}

	return binaries
}