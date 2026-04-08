// Package main implements the Scoop manifest generator CLI tool.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// ScoopPlatform represents a supported platform for the Scoop manifest.
type ScoopPlatform struct {
	URL  string `json:"url"`
	Hash string `json:"hash"`
	Bin  string `json:"bin,omitempty"`
}

// ScoopAutoupdate represents the autoupdate configuration for Scoop.
type ScoopAutoupdate struct {
	Architecture ScoopArchitecture `json:"architecture"`
}

// ScoopArchitecture represents architecture-specific autoupdate URLs.
type ScoopArchitecture struct {
	AMD64 ScoopPlatformUpdate `json:"64bit"`
	ARM64 ScoopPlatformUpdate `json:"arm64,omitempty"`
}

// ScoopPlatformUpdate represents the update configuration for a specific platform.
type ScoopPlatformUpdate struct {
	URL  string          `json:"url"`
	Hash ScoopHashUpdate `json:"hash"`
}

// ScoopHashUpdate represents hash extraction configuration.
type ScoopHashUpdate struct {
	URL string `json:"url,omitempty"`
}

// ScoopManifest represents a Scoop manifest configuration.
type ScoopManifest struct {
	Version      string                    `json:"version"`
	Description  string                    `json:"description"`
	Homepage     string                    `json:"homepage"`
	License      string                    `json:"license"`
	Architecture ScoopArchitectureManifest `json:"architecture"`
	Bin          string                    `json:"bin"`
	Checkver     *ScoopCheckver            `json:"checkver,omitempty"`
	Autoupdate   *ScoopAutoupdate          `json:"autoupdate,omitempty"`
	Notes        string                    `json:"notes,omitempty"`
	Suggest      map[string][]string       `json:"suggest,omitempty"`
}

// ScoopArchitectureManifest contains platform-specific binaries in Scoop format.
type ScoopArchitectureManifest struct {
	AMD64 *ScoopPlatform `json:"64bit,omitempty"`
	ARM64 *ScoopPlatform `json:"arm64,omitempty"`
}

// ScoopCheckver represents version checking configuration.
type ScoopCheckver struct {
	Github string `json:"github,omitempty"`
}

// DefaultScoopManifest returns a ScoopManifest with default values for Cline CLI.
func DefaultScoopManifest(version string) *ScoopManifest {
	return &ScoopManifest{
		Version:     version,
		Description: "AI-powered coding assistant CLI",
		Homepage:    "https://github.com/cline/cline",
		License:     "Apache-2.0",
		Bin:         "cline.exe",
		Checkver: &ScoopCheckver{
			Github: "https://github.com/cline/cline",
		},
		Autoupdate: &ScoopAutoupdate{
			Architecture: ScoopArchitecture{
				AMD64: ScoopPlatformUpdate{
					URL: "https://github.com/cline/cline/releases/download/v$version/cline-$version-windows-amd64.zip",
					Hash: ScoopHashUpdate{
						URL: "$url.sha256",
					},
				},
				ARM64: ScoopPlatformUpdate{
					URL: "https://github.com/cline/cline/releases/download/v$version/cline-$version-windows-arm64.zip",
					Hash: ScoopHashUpdate{
						URL: "$url.sha256",
					},
				},
			},
		},
		Notes: "Run 'cline --help' to get started with the Cline CLI.",
		Suggest: map[string][]string{
			"git": {"git"},
		},
	}
}

// AddPlatform adds a platform to the manifest with SHA256 calculation.
func (m *ScoopManifest) AddPlatform(arch, url string) error {
	sha256, err := calculateSHA256FromURL(url)
	if err != nil {
		return fmt.Errorf("failed to calculate SHA256 for %s: %w", arch, err)
	}

	platform := &ScoopPlatform{
		URL:  url,
		Hash: "sha256:" + sha256,
		Bin:  "cline.exe",
	}

	switch arch {
	case "amd64":
		m.Architecture.AMD64 = platform
	case "arm64":
		m.Architecture.ARM64 = platform
	default:
		return fmt.Errorf("unsupported architecture: %s", arch)
	}

	return nil
}

// AddPlatformWithLocalFile adds a platform using a local file path for SHA256 calculation.
func (m *ScoopManifest) AddPlatformWithLocalFile(arch, url, localPath string) error {
	sha256, err := calculateSHA256FromFile(localPath)
	if err != nil {
		return fmt.Errorf("failed to calculate SHA256 for %s from file %s: %w", arch, localPath, err)
	}

	platform := &ScoopPlatform{
		URL:  url,
		Hash: "sha256:" + sha256,
		Bin:  "cline.exe",
	}

	switch arch {
	case "amd64":
		m.Architecture.AMD64 = platform
	case "arm64":
		m.Architecture.ARM64 = platform
	default:
		return fmt.Errorf("unsupported architecture: %s", arch)
	}

	return nil
}

// Validate checks if the manifest configuration is valid.
func (m *ScoopManifest) Validate() error {
	if m.Version == "" {
		return fmt.Errorf("manifest version is required")
	}
	if m.Description == "" {
		return fmt.Errorf("manifest description is required")
	}
	if m.Homepage == "" {
		return fmt.Errorf("manifest homepage is required")
	}
	if m.Bin == "" {
		return fmt.Errorf("manifest bin is required")
	}
	if m.Architecture.AMD64 == nil && m.Architecture.ARM64 == nil {
		return fmt.Errorf("at least one platform is required")
	}
	return nil
}

// Generate generates the Scoop manifest JSON.
func (m *ScoopManifest) Generate() (string, error) {
	if err := m.Validate(); err != nil {
		return "", fmt.Errorf("manifest validation failed: %w", err)
	}

	data, err := json.MarshalIndent(m, "", "    ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal manifest: %w", err)
	}

	return string(data), nil
}

// calculateSHA256FromURL downloads a file and calculates its SHA256 checksum.
func calculateSHA256FromURL(url string) (string, error) {
	client := &http.Client{Timeout: 5 * time.Minute}

	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code %d for %s", resp.StatusCode, url)
	}

	hasher := sha256.New()
	if _, err := io.Copy(hasher, resp.Body); err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// calculateSHA256FromFile calculates the SHA256 checksum of a local file.
func calculateSHA256FromFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file %s: %w", path, err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", path, err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// ScoopConfig holds the configuration for generating a Scoop manifest.
type ScoopConfig struct {
	Version      string
	OutputPath   string
	BaseURL      string
	BinaryDir    string
	ManifestName string
	BucketDir    string
	LocalMode    bool
	DryRun       bool
}

// parseScoopFlags parses command-line flags and returns a ScoopConfig.
func parseScoopFlags() *ScoopConfig {
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
		fmt.Fprintf(os.Stderr, "Generate Scoop manifest for Cline CLI distribution.\n\n")
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

// validate checks if the configuration is valid.
func (c *ScoopConfig) validate() error {
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

// generateScoopReleaseURL generates a download URL for a release asset.
func generateScoopReleaseURL(baseURL, version, binaryName, arch string) string {
	assetName := fmt.Sprintf("%s-%s-windows-%s.zip", binaryName, version, arch)
	return fmt.Sprintf("%s/v%s/%s", baseURL, version, assetName)
}

// supportedArchitectures returns the list of supported architectures for Scoop.
func supportedArchitectures() []string {
	return []string{"amd64", "arm64"}
}

// createBucketStructure creates the directory structure for a Scoop bucket.
func createBucketStructure(rootPath string) error {
	manifestPath := filepath.Join(rootPath, "bucket")
	if err := os.MkdirAll(manifestPath, 0755); err != nil {
		return fmt.Errorf("failed to create manifest directory: %w", err)
	}

	readmePath := filepath.Join(rootPath, "README.md")
	readmeContent := `# Scoop Bucket for Cline

This is the official Scoop bucket for [Cline](https://github.com/cline/cline).

## Installation

` + "```powershell" + `
scoop bucket add cline https://github.com/cline/scoop-cline
scoop install cline
` + "```" + `

## Updating

` + "```powershell" + `
scoop update
scoop update cline
` + "```" + `

## Documentation

For more information about Cline, visit the [official documentation](https://docs.cline.bot).

## License

Apache-2.0
`
	if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
		return fmt.Errorf("failed to create README: %w", err)
	}

	return nil
}

// runScoop executes the Scoop manifest generation.
func runScoop(config *ScoopConfig) error {
	if err := config.validate(); err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	// Create bucket structure if requested
	if config.BucketDir != "" {
		fmt.Fprintf(os.Stderr, "Creating bucket structure in %s...\n", config.BucketDir)
		if err := createBucketStructure(config.BucketDir); err != nil {
			return fmt.Errorf("failed to create bucket structure: %w", err)
		}
	}

	// Generate the manifest
	fmt.Fprintf(os.Stderr, "Generating manifest for version %s...\n", config.Version)
	manifest := DefaultScoopManifest(config.Version)
	if config.ManifestName != "" {
		manifest.Bin = config.ManifestName + ".exe"
	}

	architectures := supportedArchitectures()
	atLeastOne := false

	for _, arch := range architectures {
		url := generateScoopReleaseURL(config.BaseURL, config.Version, "cline", arch)

		if config.LocalMode {
			binaryName := fmt.Sprintf("cline-%s-windows-%s.zip", config.Version, arch)
			localPath := filepath.Join(config.BinaryDir, binaryName)

			if _, err := os.Stat(localPath); os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "Warning: local binary not found: %s, skipping %s\n", localPath, arch)
				continue
			}

			if err := manifest.AddPlatformWithLocalFile(arch, url, localPath); err != nil {
				return fmt.Errorf("failed to add architecture %s: %w", arch, err)
			}
			atLeastOne = true
		} else {
			fmt.Fprintf(os.Stderr, "Downloading windows/%s binary for SHA256 calculation...\n", arch)
			if err := manifest.AddPlatform(arch, url); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to download %s: %v, skipping\n", arch, err)
				continue
			}
			atLeastOne = true
		}
	}

	if !atLeastOne {
		return fmt.Errorf("no architectures available for manifest")
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
		outputPath = filepath.Join(config.BucketDir, "bucket", config.ManifestName+".json")
	}

	// Write to file
	fmt.Fprintf(os.Stderr, "Writing manifest to %s...\n", outputPath)
	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Successfully generated Scoop manifest: %s\n", outputPath)
	return nil
}

func main() {
	config := parseScoopFlags()
	if err := runScoop(config); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
