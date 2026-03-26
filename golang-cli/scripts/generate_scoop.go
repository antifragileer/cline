// Package scripts provides build and distribution utilities for the Cline CLI.
// This file implements the CLI tool for generating Scoop manifests.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

// ScoopConfig holds the configuration for generating a Scoop manifest.
type ScoopConfig struct {
	// Version is the release version
	Version string

	// OutputPath is where to write the manifest file
	OutputPath string

	// BaseURL is the base URL for release downloads
	BaseURL string

	// BinaryDir is the directory containing pre-built binaries
	BinaryDir string

	// ManifestName is the name of the manifest
	ManifestName string

	// BucketDir is the directory to create bucket structure in (optional)
	BucketDir string

	// LocalMode uses local files for SHA256 calculation instead of downloading
	LocalMode bool

	// DryRun prints the manifest without writing to file
	DryRun bool
}

// ParseFlags parses command-line flags and returns a ScoopConfig.
func ParseScoopFlags() *ScoopConfig {
	config := &ScoopConfig{}

	flag.StringVar(&config.Version, "version", "", "Release version (required)")
	flag.StringVar(&config.OutputPath, "output", "", "Output path for manifest file")
	flag.StringVar(&config.BaseURL, "base-url", "https://github.com/cline/cline/releases/download", "Base URL for release downloads")
	flag.StringVar(&config.BinaryDir, "binary-dir", "", "Directory containing pre-built binaries for local SHA256 calculation")
	flag.StringVar(&config.ManifestName, "name", "cline", "Manifest name")
	flag.StringVar(&config.BucketDir, "bucket-dir", "", "Create bucket structure in this directory")
	flag.BoolVar(&config.LocalMode, "local", false, "Use local files for SHA256 calculation")
	flag.BoolVar(&config.DryRun, "dry-run", false, "Print manifest without writing to file")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Generate Scoop manifest for Cline CLI Windows distribution.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Generate manifest for a release\n")
		fmt.Fprintf(os.Stderr, "  %s -version 1.0.0 -output bucket/cline.json\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Generate manifest using local binaries\n")
		fmt.Fprintf(os.Stderr, "  %s -version 1.0.0 -local -binary-dir ./dist -output bucket/cline.json\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Create complete bucket structure\n")
		fmt.Fprintf(os.Stderr, "  %s -version 1.0.0 -bucket-dir ./scoop-bucket\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Dry run to preview manifest\n")
		fmt.Fprintf(os.Stderr, "  %s -version 1.0.0 -dry-run\n", os.Args[0])
	}

	flag.Parse()

	return config
}

// Validate checks if the configuration is valid.
func (c *ScoopConfig) Validate() error {
	if c.Version == "" {
		return fmt.Errorf("version is required (use -version flag)")
	}

	if !c.DryRun && c.OutputPath == "" && c.BucketDir == "" {
		return fmt.Errorf("either -output, -bucket-dir, or -dry-run must be specified")
	}

	if c.LocalMode && c.BinaryDir == "" {
		return fmt.Errorf("binary-dir is required when using local mode")
	}

	return nil
}

// RunScoop executes the Scoop manifest generation.
func RunScoop(config *ScoopConfig) error {
	if err := config.Validate(); err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	// Create bucket structure if requested
	if config.BucketDir != "" {
		bucket := DefaultScoopBucket()
		fmt.Fprintf(os.Stderr, "Creating bucket structure in %s...\n", config.BucketDir)
		if err := bucket.CreateBucketStructure(config.BucketDir); err != nil {
			return fmt.Errorf("failed to create bucket structure: %w", err)
		}
	}

	// Generate the manifest
	fmt.Fprintf(os.Stderr, "Generating manifest for version %s...\n", config.Version)
	manifest := DefaultScoopManifest(config.Version)
	if config.ManifestName != "" {
		// Note: Scoop uses the binary name from the manifest, not a custom name field
		_ = config.ManifestName // Reserved for future use
	}

	// Generate URLs and SHA256 for each architecture
	architectures := ScoopSupportedArchitectures()

	for _, arch := range architectures {
		url := GenerateScoopReleaseURL(config.BaseURL, config.Version, "cline", arch)

		if config.LocalMode {
			// Use local file for SHA256 calculation
			// Note: Scoop uses .zip format for Windows
			binaryName := fmt.Sprintf("cline_%s_windows_%s.zip", config.Version, arch)
			localPath := filepath.Join(config.BinaryDir, binaryName)

			if _, err := os.Stat(localPath); os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "Warning: local binary not found: %s (skipping)\n", localPath)
				continue
			}

			if err := manifest.AddPlatformWithLocalFile(arch, url, localPath); err != nil {
				return fmt.Errorf("failed to add architecture %s: %w", arch, err)
			}
		} else {
			// Download and calculate SHA256
			fmt.Fprintf(os.Stderr, "Downloading windows/%s binary for SHA256 calculation...\n", arch)
			if err := manifest.AddPlatform(arch, url); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to add architecture %s: %v (skipping)\n", arch, err)
				continue
			}
		}
	}

	// Validate the manifest
	if err := manifest.Validate(); err != nil {
		return fmt.Errorf("manifest validation failed: %w", err)
	}

	// Generate manifest content
	content, err := manifest.Generate()
	if err != nil {
		return fmt.Errorf("failed to generate manifest: %w", err)
	}

	// Output the result
	if config.DryRun {
		fmt.Println(content)
		return nil
	}

	// Determine output path
	outputPath := config.OutputPath
	if outputPath == "" && config.BucketDir != "" {
		bucket := DefaultScoopBucket()
		outputPath = bucket.GetManifestPath(config.BucketDir, manifest.Bin[:len(manifest.Bin)-4]) // Remove .exe
	}

	// Write to file
	fmt.Fprintf(os.Stderr, "Writing manifest to %s...\n", outputPath)
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Successfully generated Scoop manifest: %s\n", outputPath)
	return nil
}

func main() {
	config := ParseScoopFlags()

	if err := RunScoop(config); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// GenerateScoopManifestFromRelease generates a manifest for a specific release version.
func GenerateScoopManifestFromRelease(version, outputPath, baseURL string) error {
	config := &ScoopConfig{
		Version:    version,
		OutputPath: outputPath,
		BaseURL:    baseURL,
	}

	return RunScoop(config)
}

// GenerateScoopManifestFromLocalBinaries generates a manifest using local pre-built binaries.
func GenerateScoopManifestFromLocalBinaries(version, binaryDir, outputPath string) error {
	config := &ScoopConfig{
		Version:    version,
		OutputPath: outputPath,
		BinaryDir:  binaryDir,
		LocalMode:  true,
	}

	return RunScoop(config)
}

// CreateScoopBucket creates a complete Scoop bucket structure.
func CreateScoopBucket(bucketDir, version string, localMode bool, binaryDir string) error {
	config := &ScoopConfig{
		Version:   version,
		BucketDir: bucketDir,
		LocalMode: localMode,
		BinaryDir: binaryDir,
	}

	return RunScoop(config)
}

// PrintScoopManifest prints the manifest to stdout without writing to file.
func PrintScoopManifest(version string, baseURL string) error {
	config := &ScoopConfig{
		Version: version,
		BaseURL: baseURL,
		DryRun:  true,
	}

	return RunScoop(config)
}

// GetScoopDefaultOutputPath returns the default output path for the manifest.
func GetScoopDefaultOutputPath(manifestName string) string {
	return filepath.Join("bucket", manifestName+".json")
}

// IsScoopValidVersion checks if a version string is valid.
func IsScoopValidVersion(version string) bool {
	if version == "" {
		return false
	}
	// Simple version validation - must not be empty
	return true
}

// GetScoopBinaryName generates the expected binary name for Windows.
func GetScoopBinaryName(version, arch string) string {
	return fmt.Sprintf("cline_%s_windows_%s.zip", version, arch)
}

// GetScoopBinaryPath generates the full path to a binary.
func GetScoopBinaryPath(binaryDir, version, arch string) string {
	return filepath.Join(binaryDir, GetScoopBinaryName(version, arch))
}

// CheckScoopBinariesExist checks if all required binaries exist.
func CheckScoopBinariesExist(binaryDir, version string) error {
	architectures := ScoopSupportedArchitectures()
	var missing []string

	for _, arch := range architectures {
		path := GetScoopBinaryPath(binaryDir, version, arch)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			missing = append(missing, GetScoopBinaryName(version, arch))
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing binaries: %s", fmt.Sprintf("%v", missing))
	}

	return nil
}

// ListScoopRequiredBinaries returns the list of required binary names.
func ListScoopRequiredBinaries(version string) []string {
	architectures := ScoopSupportedArchitectures()
	binaries := make([]string, len(architectures))

	for i, arch := range architectures {
		binaries[i] = GetScoopBinaryName(version, arch)
	}

	return binaries
}