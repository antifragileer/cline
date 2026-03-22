// Package scripts provides build and distribution utilities for the Cline CLI.
// This file implements Homebrew formula generation for distributing the CLI
// via Homebrew taps.
package main

import (
	"crypto/sha256"
	"encoding/hex"
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
	// OS is the operating system (darwin or linux)
	OS string

	// Arch is the architecture (amd64 or arm64)
	Arch string

	// URL is the download URL for the binary
	URL string

	// SHA256 is the SHA256 checksum of the binary
	SHA256 string
}

// HomebrewFormula represents a Homebrew formula configuration.
type HomebrewFormula struct {
	// Name is the formula name
	Name string

	// Desc is a short description of the formula
	Desc string

	// Homepage is the project homepage URL
	Homepage string

	// Version is the formula version
	Version string

	// License is the software license
	License string

	// Platforms contains all supported platforms
	Platforms []HomebrewPlatform

	// InstallDir is the directory where binaries are installed
	InstallDir string

	// BinaryName is the name of the binary to install
	BinaryName string

	// Dependencies are Homebrew dependencies
	Dependencies []string

	// Caveats are optional post-installation messages
	Caveats string

	// TestCommand is the command to run for `brew test`
	TestCommand string
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
	sha256, err := CalculateSHA256FromURL(url)
	if err != nil {
		return fmt.Errorf("failed to calculate SHA256 for %s/%s: %w", os, arch, err)
	}

	platform := HomebrewPlatform{
		OS:     os,
		Arch:   arch,
		URL:    url,
		SHA256: sha256,
	}

	f.Platforms = append(f.Platforms, platform)
	return nil
}

// AddPlatformWithLocalFile adds a platform using a local file path for SHA256 calculation.
func (f *HomebrewFormula) AddPlatformWithLocalFile(os, arch, url, localPath string) error {
	sha256, err := CalculateSHA256FromFile(localPath)
	if err != nil {
		return fmt.Errorf("failed to calculate SHA256 for %s/%s from file %s: %w", os, arch, localPath, err)
	}

	platform := HomebrewPlatform{
		OS:     os,
		Arch:   arch,
		URL:    url,
		SHA256: sha256,
	}

	f.Platforms = append(f.Platforms, platform)
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

	for i, p := range f.Platforms {
		if p.OS == "" {
			return fmt.Errorf("platform %d: OS is required", i)
		}
		if p.Arch == "" {
			return fmt.Errorf("platform %d: Arch is required", i)
		}
		if p.URL == "" {
			return fmt.Errorf("platform %d: URL is required", i)
		}
		if p.SHA256 == "" {
			return fmt.Errorf("platform %d: SHA256 is required", i)
		}
		if len(p.SHA256) != 64 {
			return fmt.Errorf("platform %d: SHA256 must be 64 characters", i)
		}
	}

	return nil
}

// Generate generates the Homebrew formula Ruby code.
func (f *HomebrewFormula) Generate() (string, error) {
	if err := f.Validate(); err != nil {
		return "", fmt.Errorf("formula validation failed: %w", err)
	}

	tmpl, err := template.New("formula").Funcs(template.FuncMap{
		"title":       strings.Title,
		"upper":       strings.ToUpper,
		"platformMap": getHomebrewPlatformMap,
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

// GenerateToFile generates the formula and writes it to a file.
func (f *HomebrewFormula) GenerateToFile(path string) error {
	content, err := f.Generate()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write formula to %s: %w", path, err)
	}

	return nil
}

// getHomebrewPlatformMap returns the platform identifier for Homebrew.
func getHomebrewPlatformMap(os, arch string) string {
	// Map to Homebrew platform identifiers
	platformMap := map[string]string{
		"darwin-amd64": ":x86_64_darwin",
		"darwin-arm64": ":arm64_darwin",
		"linux-amd64":  ":x86_64_linux",
		"linux-arm64":  ":arm64_linux",
	}

	key := fmt.Sprintf("%s-%s", os, arch)
	if mapped, ok := platformMap[key]; ok {
		return mapped
	}
	return fmt.Sprintf(":%s_%s", arch, os)
}

// CalculateSHA256FromURL downloads a file and calculates its SHA256 checksum.
func CalculateSHA256FromURL(url string) (string, error) {
	client := &http.Client{
		Timeout: 5 * time.Minute,
	}

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

// CalculateSHA256FromFile calculates the SHA256 checksum of a local file.
func CalculateSHA256FromFile(path string) (string, error) {
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

// CalculateSHA256FromBytes calculates the SHA256 checksum of byte data.
func CalculateSHA256FromBytes(data []byte) string {
	hasher := sha256.New()
	hasher.Write(data)
	return hex.EncodeToString(hasher.Sum(nil))
}

// HomebrewTap represents a Homebrew tap repository structure.
type HomebrewTap struct {
	// Name is the tap name (e.g., "cline/homebrew-tap")
	Name string

	// FormulaDir is the directory containing formula files (usually "Formula")
	FormulaDir string

	// BaseURL is the base URL for formula downloads
	BaseURL string
}

// DefaultHomebrewTap returns a HomebrewTap with default values for Cline.
func DefaultHomebrewTap() *HomebrewTap {
	return &HomebrewTap{
		Name:       "cline/homebrew-tap",
		FormulaDir: "Formula",
		BaseURL:    "https://github.com/cline/cline/releases/download",
	}
}

// CreateTapStructure creates the directory structure for a Homebrew tap.
func (t *HomebrewTap) CreateTapStructure(rootPath string) error {
	formulaPath := filepath.Join(rootPath, t.FormulaDir)
	if err := os.MkdirAll(formulaPath, 0755); err != nil {
		return fmt.Errorf("failed to create formula directory: %w", err)
	}

	// Create a basic README
	readmePath := filepath.Join(rootPath, "README.md")
	readmeContent := fmt.Sprintf(homebrewTapReadmeTemplate, t.Name)
	if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
		return fmt.Errorf("failed to create README: %w", err)
	}

	return nil
}

// GetFormulaPath returns the path where a formula should be stored.
func (t *HomebrewTap) GetFormulaPath(rootPath, formulaName string) string {
	return filepath.Join(rootPath, t.FormulaDir, formulaName+".rb")
}

// HomebrewVersionInfo holds version information for generating formula URLs.
type HomebrewVersionInfo struct {
	// Version is the release version
	Version string

	// GitCommit is the git commit hash
	GitCommit string
}

// GenerateHomebrewReleaseURL generates a download URL for a release asset.
func GenerateHomebrewReleaseURL(baseURL, version, binaryName, os, arch string) string {
	// Construct the asset name
	// Format: cline_1.0.0_darwin_amd64.tar.gz
	assetName := fmt.Sprintf("%s_%s_%s_%s.tar.gz", binaryName, version, os, arch)
	return fmt.Sprintf("%s/v%s/%s", baseURL, version, assetName)
}

// GenerateHomebrewPlatformURL generates a download URL for a specific platform.
func GenerateHomebrewPlatformURL(baseURL, version, os, arch string) string {
	return GenerateHomebrewReleaseURL(baseURL, version, "cline", os, arch)
}

// HomebrewSupportedPlatforms returns the list of supported platforms.
func HomebrewSupportedPlatforms() []struct {
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

// DetectHomebrewPlatform detects the current platform OS and architecture.
func DetectHomebrewPlatform() (string, string) {
	os := runtime.GOOS
	arch := runtime.GOARCH

	// Normalize architecture names
	switch arch {
	case "amd64":
		arch = "amd64"
	case "arm64":
		arch = "arm64"
	case "386":
		arch = "386"
	default:
		arch = arch
	}

	return os, arch
}

// IsHomebrewSupportedPlatform checks if a platform is supported.
func IsHomebrewSupportedPlatform(os, arch string) bool {
	for _, p := range HomebrewSupportedPlatforms() {
		if p.OS == os && p.Arch == arch {
			return true
		}
	}
	return false
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

  {{- range .Dependencies}}
  depends_on "{{.}}"
  {{- end}}

  def install
    bin.install "{{.BinaryName}}"
  end

  test do
    system "#{bin}/{{.BinaryName}}", "version"
  end
end
`

const homebrewTapReadmeTemplate = `# Homebrew Tap for Cline

This is the official Homebrew tap for [Cline](https://github.com/cline/cline).

## Installation

` + "```bash" + `
brew tap %s
brew install cline
` + "```" + `

## Updating

` + "```bash" + `
brew update
brew upgrade cline
` + "```" + `

## Available Formulae

- ` + "`cline`" + ` - AI-powered coding assistant CLI

## Documentation

For more information about Cline, visit the [official documentation](https://docs.cline.bot).

## License

Apache-2.0
`