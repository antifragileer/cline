// Package main implements the Homebrew formula generator CLI tool.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
	"time"
)

// HomebrewPlatform represents a supported platform for the Homebrew formula.
type HomebrewPlatform struct {
	OS     string
	Arch   string
	URL    string
	SHA256 string
}

// HomebrewFormula represents a Homebrew formula configuration.
type HomebrewFormula struct {
	Name         string
	Desc         string
	Homepage     string
	Version      string
	License      string
	Platforms    []HomebrewPlatform
	InstallDir   string
	BinaryName   string
	Dependencies []string
	Caveats      string
	TestCommand  string
}

// DefaultHomebrewFormula returns a HomebrewFormula with default values for Cline CLI.
func DefaultHomebrewFormula(version string) *HomebrewFormula {
	return &HomebrewFormula{
		Name:         "cline",
		Desc:         "AI-powered coding assistant CLI",
		Homepage:     "https://github.com/cline/cline",
		Version:      version,
		License:      "Apache-2.0",
		InstallDir:   "bin",
		BinaryName:   "cline",
		TestCommand:  "cline version",
		Dependencies: []string{},
		Platforms:    make([]HomebrewPlatform, 0, 4),
	}
}

// AddPlatform adds a platform to the formula with SHA256 calculation.
func (f *HomebrewFormula) AddPlatform(os, arch, url string) error {
	sha256, err := calculateSHA256FromURL(url)
	if err != nil {
		return fmt.Errorf("failed to calculate SHA256 for %s/%s: %w", os, arch, err)
	}

	f.Platforms = append(f.Platforms, HomebrewPlatform{
		OS:     os,
		Arch:   arch,
		URL:    url,
		SHA256: sha256,
	})
	return nil
}

// AddPlatformWithLocalFile adds a platform using a local file path for SHA256 calculation.
func (f *HomebrewFormula) AddPlatformWithLocalFile(os, arch, url, localPath string) error {
	sha256, err := calculateSHA256FromFile(localPath)
	if err != nil {
		return fmt.Errorf("failed to calculate SHA256 for %s/%s from file %s: %w", os, arch, localPath, err)
	}

	f.Platforms = append(f.Platforms, HomebrewPlatform{
		OS:     os,
		Arch:   arch,
		URL:    url,
		SHA256: sha256,
	})
	return nil
}

// Validate checks if the formula configuration is valid.
func (f *HomebrewFormula) Validate() error {
	if f.Name == "" {
		return fmt.Errorf("formula name is required")
	}
	if f.Version == "" {
		return fmt.Errorf("formula version is required")
	}
	if f.Homepage == "" {
		return fmt.Errorf("formula homepage is required")
	}
	if len(f.Platforms) == 0 {
		return fmt.Errorf("at least one platform is required")
	}
	return nil
}

// Generate generates the Homebrew formula Ruby code.
func (f *HomebrewFormula) Generate() (string, error) {
	if err := f.Validate(); err != nil {
		return "", fmt.Errorf("formula validation failed: %w", err)
	}

	tmpl, err := template.New("formula").Funcs(template.FuncMap{
		"title": strings.Title,
	}).Parse(homebrewFormulaTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse formula template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, f); err != nil {
		return "", fmt.Errorf("failed to execute formula template: %w", err)
	}

	return buf.String(), nil
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

// HomebrewConfig holds the configuration for generating a Homebrew formula.
type HomebrewConfig struct {
	Version     string
	OutputPath  string
	BaseURL     string
	BinaryDir   string
	FormulaName string
	TapDir      string
	LocalMode   bool
	DryRun      bool
}

// parseFlags parses command-line flags and returns a HomebrewConfig.
func parseFlags() *HomebrewConfig {
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

// validate checks if the configuration is valid.
func (c *HomebrewConfig) validate() error {
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

// supportedPlatforms returns the list of supported platforms.
func supportedPlatforms() []struct {
	OS   string
	Arch string
} {
	return []struct {
		OS   string
		Arch string
	}{
		{"darwin", "amd64"},
		{"darwin", "arm64"},
		{"linux", "amd64"},
		{"linux", "arm64"},
	}
}

// generatePlatformURL generates a download URL for a specific platform.
func generatePlatformURL(baseURL, version, os, arch string) string {
	assetName := fmt.Sprintf("cline_%s_%s_%s.tar.gz", version, os, arch)
	return fmt.Sprintf("%s/v%s/%s", baseURL, version, assetName)
}

// createTapStructure creates the directory structure for a Homebrew tap.
func createTapStructure(rootPath string) error {
	formulaPath := filepath.Join(rootPath, "Formula")
	if err := os.MkdirAll(formulaPath, 0755); err != nil {
		return fmt.Errorf("failed to create formula directory: %w", err)
	}

	readmePath := filepath.Join(rootPath, "README.md")
	readmeContent := `# Homebrew Tap for Cline

This is the official Homebrew tap for [Cline](https://github.com/cline/cline).

## Installation

` + "```bash" + `
brew tap cline/tap
brew install cline
` + "```" + `

## Updating

` + "```bash" + `
brew update
brew upgrade cline
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

// run executes the Homebrew formula generation.
func run(config *HomebrewConfig) error {
	if err := config.validate(); err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	// Create tap structure if requested
	if config.TapDir != "" {
		fmt.Fprintf(os.Stderr, "Creating tap structure in %s...\n", config.TapDir)
		if err := createTapStructure(config.TapDir); err != nil {
			return fmt.Errorf("failed to create tap structure: %w", err)
		}
	}

	// Generate the formula
	fmt.Fprintf(os.Stderr, "Generating formula for version %s...\n", config.Version)
	formula := DefaultHomebrewFormula(config.Version)
	if config.FormulaName != "" {
		formula.Name = config.FormulaName
	}

	platforms := supportedPlatforms()

	for _, p := range platforms {
		url := generatePlatformURL(config.BaseURL, config.Version, p.OS, p.Arch)

		if config.LocalMode {
			binaryName := fmt.Sprintf("cline_%s_%s_%s.tar.gz", config.Version, p.OS, p.Arch)
			localPath := filepath.Join(config.BinaryDir, binaryName)

			if _, err := os.Stat(localPath); os.IsNotExist(err) {
				return fmt.Errorf("local binary not found: %s", localPath)
			}

			if err := formula.AddPlatformWithLocalFile(p.OS, p.Arch, url, localPath); err != nil {
				return fmt.Errorf("failed to add platform %s/%s: %w", p.OS, p.Arch, err)
			}
		} else {
			fmt.Fprintf(os.Stderr, "Downloading %s/%s binary for SHA256 calculation...\n", p.OS, p.Arch)
			if err := formula.AddPlatform(p.OS, p.Arch, url); err != nil {
				return fmt.Errorf("failed to add platform %s/%s: %w", p.OS, p.Arch, err)
			}
		}
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
		outputPath = filepath.Join(config.TapDir, "Formula", formula.Name+".rb")
	}

	// Write to file
	fmt.Fprintf(os.Stderr, "Writing formula to %s...\n", outputPath)
	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write formula: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Successfully generated Homebrew formula: %s\n", outputPath)
	return nil
}

const homebrewFormulaTemplate = `# typed: false
# frozen_string_literal: true

class {{title .Name}} < Formula
  desc "{{.Desc}}"
  homepage "{{.Homepage}}"
  version "{{.Version}}"
  license "{{.License}}"

  {{- range .Platforms}}
  {{- if eq .OS "darwin"}}
  {{- if eq .Arch "amd64"}}
  if OS.mac? && Hardware::CPU.intel?
    url "{{.URL}}"
    sha256 "{{.SHA256}}"
  end
  {{- end}}
  {{- if eq .Arch "arm64"}}
  if OS.mac? && Hardware::CPU.arm?
    url "{{.URL}}"
    sha256 "{{.SHA256}}"
  end
  {{- end}}
  {{- end}}
  {{- if eq .OS "linux"}}
  {{- if eq .Arch "amd64"}}
  if OS.linux? && Hardware::CPU.intel?
    url "{{.URL}}"
    sha256 "{{.SHA256}}"
  end
  {{- end}}
  {{- if eq .Arch "arm64"}}
  if OS.linux? && Hardware::CPU.arm?
    url "{{.URL}}"
    sha256 "{{.SHA256}}"
  end
  {{- end}}
  {{- end}}
  {{- end}}

  def install
    bin.install "{{.BinaryName}}"
  end

  test do
    system "#{bin}/{{.BinaryName}}", "version"
  end
end
`

func main() {
	config := parseFlags()
	if err := run(config); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// Suppress unused import error for runtime
var _ = runtime.GOOS