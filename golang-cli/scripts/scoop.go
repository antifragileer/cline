// Package scripts provides build and distribution utilities for the Cline CLI.
// This file implements Scoop manifest generation for distributing the CLI
// via Scoop (Windows package manager).
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// ScoopPlatform represents a supported platform for the Scoop manifest.
type ScoopPlatform struct {
	// URL is the download URL for the binary
	URL string `json:"url"`

	// Hash is the SHA256 hash of the binary (format: "sha256:...")
	Hash string `json:"hash"`

	// ExtractDir is the directory to extract from the archive
	ExtractDir string `json:"extract_dir,omitempty"`

	// Bin is the binary path within the archive
	Bin string `json:"bin,omitempty"`
}

// ScoopAutoupdate represents the autoupdate configuration for Scoop.
type ScoopAutoupdate struct {
	// Architecture-specific autoupdate URLs
	Architecture ScoopArchitecture `json:"architecture"`
}

// ScoopArchitecture represents architecture-specific autoupdate URLs.
type ScoopArchitecture struct {
	// AMD64 autoupdate configuration
	AMD64 ScoopPlatformUpdate `json:"64bit"`

	// ARM64 autoupdate configuration
	ARM64 ScoopPlatformUpdate `json:"arm64,omitempty"`
}

// ScoopPlatformUpdate represents the update configuration for a specific platform.
type ScoopPlatformUpdate struct {
	// URL template for updates
	URL string `json:"url"`

	// Hash extraction configuration
	Hash ScoopHashUpdate `json:"hash"`
}

// ScoopHashUpdate represents hash extraction configuration.
type ScoopHashUpdate struct {
	// URL template for hash file
	URL string `json:"url,omitempty"`

	// Regex to extract hash from file
	Regex string `json:"regex,omitempty"`

	// Mode for hash extraction (e.g., "extract")
	Mode string `json:"mode,omitempty"`
}

// ScoopManifest represents a Scoop manifest configuration.
type ScoopManifest struct {
	// Version is the package version
	Version string `json:"version"`

	// Description is a short description
	Description string `json:"description"`

	// Homepage is the project homepage URL
	Homepage string `json:"homepage"`

	// License is the software license
	License string `json:"license"`

	// Architecture contains platform-specific binaries
	Architecture ScoopArchitectureManifest `json:"architecture"`

	// Bin is the binary to add to PATH
	Bin string `json:"bin"`

	// Checkver is the version checker configuration
	Checkver *ScoopCheckver `json:"checkver,omitempty"`

	// Autoupdate configuration
	Autoupdate *ScoopAutoupdate `json:"autoupdate,omitempty"`

	// Notes are displayed after installation
	Notes string `json:"notes,omitempty"`

	// Depends are dependencies
	Depends []string `json:"depends,omitempty"`

	// Suggest optional packages
	Suggest map[string][]string `json:"suggest,omitempty"`
}

// ScoopArchitectureManifest contains platform-specific binaries in Scoop format.
type ScoopArchitectureManifest struct {
	// AMD64 platform
	AMD64 *ScoopPlatform `json:"64bit,omitempty"`

	// ARM64 platform
	ARM64 *ScoopPlatform `json:"arm64,omitempty"`
}

// ScoopCheckver represents version checking configuration.
type ScoopCheckver struct {
	// GitHub repository to check for updates
	Github string `json:"github,omitempty"`

	// URL to check for version
	URL string `json:"url,omitempty"`

	// Regex to extract version
	Regex string `json:"regex,omitempty"`

	// JSON path for version extraction
	JSONPath string `json:"jsonpath,omitempty"`

	// Reverse check order
	Reverse bool `json:"reverse,omitempty"`
}

// DefaultScoopManifest returns a ScoopManifest with default values for Cline CLI.
func DefaultScoopManifest(version string) *ScoopManifest {
	return &ScoopManifest{
		Version:     version,
		Description: "AI-powered coding assistant CLI",
		Homepage:    "https://github.com/cline/cline",
		License:     "Apache-2.0",
		Bin:         "cline.exe",
		Architecture: ScoopArchitectureManifest{
			AMD64: nil,
			ARM64: nil,
		},
		Checkver: &ScoopCheckver{
			Github: "https://github.com/cline/cline",
		},
		Autoupdate: &ScoopAutoupdate{
			Architecture: ScoopArchitecture{
				AMD64: ScoopPlatformUpdate{
					URL: "https://github.com/cline/cline/releases/download/v$version/cline_$version_windows_amd64.zip",
					Hash: ScoopHashUpdate{
						URL: "$url.sha256",
					},
				},
				ARM64: ScoopPlatformUpdate{
					URL: "https://github.com/cline/cline/releases/download/v$version/cline_$version_windows_amd64.zip",
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
	sha256, err := CalculateScoopSHA256FromURL(url)
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
	sha256, err := CalculateSHA256FromFile(localPath)
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

	if m.Architecture.AMD64 != nil {
		if err := validateScoopPlatform(m.Architecture.AMD64, "amd64"); err != nil {
			return err
		}
	}
	if m.Architecture.ARM64 != nil {
		if err := validateScoopPlatform(m.Architecture.ARM64, "arm64"); err != nil {
			return err
		}
	}

	return nil
}

func validateScoopPlatform(p *ScoopPlatform, arch string) error {
	if p.URL == "" {
		return fmt.Errorf("%s: URL is required", arch)
	}
	if p.Hash == "" {
		return fmt.Errorf("%s: hash is required", arch)
	}
	if !isValidSHA256Hash(p.Hash) {
		return fmt.Errorf("%s: invalid hash format (expected sha256:<64-char-hex>)", arch)
	}
	return nil
}

func isValidSHA256Hash(hash string) bool {
	if len(hash) != 71 { // "sha256:" + 64 chars
		return false
	}
	if hash[:7] != "sha256:" {
		return false
	}
	hexPart := hash[7:]
	if len(hexPart) != 64 {
		return false
	}
	for _, c := range hexPart {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// Generate generates the Scoop manifest JSON.
func (m *ScoopManifest) Generate() (string, error) {
	if err := m.Validate(); err != nil {
		return "", fmt.Errorf("manifest validation failed: %w", err)
	}

	// Marshal with indentation for readability
	data, err := json.MarshalIndent(m, "", "    ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal manifest: %w", err)
	}

	return string(data), nil
}

// GenerateToFile generates the manifest and writes it to a file.
func (m *ScoopManifest) GenerateToFile(path string) error {
	content, err := m.Generate()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write manifest to %s: %w", path, err)
	}

	return nil
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

// CalculateScoopSHA256FromURL downloads a file and calculates its SHA256 checksum.
func CalculateScoopSHA256FromURL(url string) (string, error) {
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

// ScoopBucket represents a Scoop bucket repository structure.
type ScoopBucket struct {
	// Name is the bucket name
	Name string

	// ManifestDir is the directory containing manifest files
	ManifestDir string
}

// DefaultScoopBucket returns a ScoopBucket with default values for Cline.
func DefaultScoopBucket() *ScoopBucket {
	return &ScoopBucket{
		Name:        "cline-scoop",
		ManifestDir: "bucket",
	}
}

// CreateBucketStructure creates the directory structure for a Scoop bucket.
func (b *ScoopBucket) CreateBucketStructure(rootPath string) error {
	manifestPath := filepath.Join(rootPath, b.ManifestDir)
	if err := os.MkdirAll(manifestPath, 0755); err != nil {
		return fmt.Errorf("failed to create manifest directory: %w", err)
	}

	// Create a basic README
	readmePath := filepath.Join(rootPath, "README.md")
	if err := os.WriteFile(readmePath, []byte(scoopBucketReadmeTemplate), 0644); err != nil {
		return fmt.Errorf("failed to create README: %w", err)
	}

	return nil
}

// GetManifestPath returns the path where a manifest should be stored.
func (b *ScoopBucket) GetManifestPath(rootPath, manifestName string) string {
	return filepath.Join(rootPath, b.ManifestDir, manifestName+".json")
}

// GenerateScoopReleaseURL generates a download URL for a release asset.
func GenerateScoopReleaseURL(baseURL, version, binaryName, arch string) string {
	// Scoop uses zip files for Windows
	assetName := fmt.Sprintf("%s_%s_windows_%s.zip", binaryName, version, arch)
	return fmt.Sprintf("%s/v%s/%s", baseURL, version, assetName)
}

// ScoopSupportedArchitectures returns the list of supported architectures for Scoop.
func ScoopSupportedArchitectures() []string {
	return []string{"amd64", "arm64"}
}

// IsScoopSupportedArchitecture checks if an architecture is supported by Scoop.
func IsScoopSupportedArchitecture(arch string) bool {
	for _, a := range ScoopSupportedArchitectures() {
		if a == arch {
			return true
		}
	}
	return false
}

const scoopBucketReadmeTemplate = `# Scoop Bucket for Cline

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

## Available Manifests

- ` + "`cline`" + ` - AI-powered coding assistant CLI

## Documentation

For more information about Cline, visit the [official documentation](https://docs.cline.bot).

## License

Apache-2.0
`
