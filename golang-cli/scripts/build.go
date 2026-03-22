// Build utilities and cross-platform build support for Cline CLI.
// This file implements build configuration and cross-compilation support.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// BuildConfig holds configuration for building the CLI binary.
type BuildConfig struct {
	// BinaryName is the name of the output binary
	BinaryName string
	// Version is the build version
	Version string
	// GitCommit is the git commit hash
	GitCommit string
	// BuildTime is the build timestamp
	BuildTime string
	// OutputDir is the directory for build artifacts
	OutputDir string
	// SourcePath is the path to the main package
	SourcePath string
	// LDFlags contains additional linker flags
	LDFlags string
	// CGOEnabled controls CGO (default: 0 for cross-compilation)
	CGOEnabled string
	// GOOS is the target operating system
	GOOS string
	// GOARCH is the target architecture
	GOARCH string
}

// DefaultBuildConfig returns a BuildConfig with sensible defaults.
func DefaultBuildConfig() *BuildConfig {
	return &BuildConfig{
		BinaryName: "cline",
		Version:    "dev",
		GitCommit:  "unknown",
		BuildTime:  time.Now().UTC().Format(time.RFC3339),
		OutputDir:  "build",
		SourcePath: "./cmd/cline",
		CGOEnabled: "0",
		GOOS:       runtime.GOOS,
		GOARCH:     runtime.GOARCH,
	}
}

// BuildPlatform represents a target platform for cross-compilation.
type BuildPlatform struct {
	OS   string
	Arch string
}

// String returns the platform in GOOS/GOARCH format.
func (p BuildPlatform) String() string {
	return fmt.Sprintf("%s/%s", p.OS, p.Arch)
}

// BinaryName returns the binary name with appropriate extension for the platform.
func (p BuildPlatform) BinaryName(baseName string) string {
	if p.OS == "windows" {
		return baseName + ".exe"
	}
	return baseName
}

// ArchiveExtension returns the appropriate archive extension for the platform.
func (p BuildPlatform) ArchiveExtension() string {
	if p.OS == "windows" {
		return ".zip"
	}
	return ".tar.gz"
}

// BuildSupportedPlatforms returns the list of platforms supported for cross-compilation.
func BuildSupportedPlatforms() []BuildPlatform {
	return []BuildPlatform{
		{OS: "linux", Arch: "amd64"},
		{OS: "linux", Arch: "arm64"},
		{OS: "darwin", Arch: "amd64"},
		{OS: "darwin", Arch: "arm64"},
		{OS: "windows", Arch: "amd64"},
	}
}

// ValidateBuildPlatform checks if a platform is supported.
func ValidateBuildPlatform(os, arch string) error {
	for _, p := range BuildSupportedPlatforms() {
		if p.OS == os && p.Arch == arch {
			return nil
		}
	}
	return fmt.Errorf("unsupported platform: %s/%s", os, arch)
}

// ParseBuildPlatform parses a platform string in the format "os/arch".
func ParseBuildPlatform(s string) (BuildPlatform, error) {
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return BuildPlatform{}, fmt.Errorf("invalid platform format: %s (expected os/arch)", s)
	}
	p := BuildPlatform{OS: parts[0], Arch: parts[1]}
	if err := ValidateBuildPlatform(p.OS, p.Arch); err != nil {
		return BuildPlatform{}, err
	}
	return p, nil
}

// Build executes the Go build command with the configured settings.
func (c *BuildConfig) Build() error {
	if c.BinaryName == "" {
		return fmt.Errorf("binary name cannot be empty")
	}

	outputName := c.BinaryName
	if c.GOOS == "windows" {
		outputName += ".exe"
	}

	outputPath := filepath.Join(c.OutputDir, outputName)
	if err := os.MkdirAll(c.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	ldflags := c.LDFlags
	if ldflags == "" {
		ldflags = c.DefaultLDFlags()
	}

	args := []string{
		"build",
		"-ldflags", ldflags,
		"-trimpath",
		"-o", outputPath,
		c.SourcePath,
	}

	cmd := exec.Command("go", args...)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env,
		"GOOS="+c.GOOS,
		"GOARCH="+c.GOARCH,
		"CGO_ENABLED="+c.CGOEnabled,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("build failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// DefaultLDFlags returns the default linker flags with version info.
func (c *BuildConfig) DefaultLDFlags() string {
	return fmt.Sprintf(
		"-s -w -X github.com/cline/cline/golang-cli/cmd/cline.Version=%s -X github.com/cline/cline/golang-cli/cmd/cline.BuildTime=%s -X github.com/cline/cline/golang-cli/cmd/cline.GitCommit=%s",
		c.Version, c.BuildTime, c.GitCommit,
	)
}

// GenerateChecksum calculates the SHA256 checksum of a file.
func GenerateChecksum(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// WriteChecksum writes the checksum to a file.
func WriteChecksum(filePath, checksum string) error {
	checksumFile := filePath + ".sha256"
	return os.WriteFile(checksumFile, []byte(checksum+"\n"), 0644)
}

// VerifyChecksum verifies a file against its checksum.
func VerifyChecksum(filePath, expectedChecksum string) error {
	actualChecksum, err := GenerateChecksum(filePath)
	if err != nil {
		return err
	}

	if actualChecksum != strings.TrimSpace(expectedChecksum) {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}

	return nil
}

// VersionInfo holds version information for the build.
type VersionInfo struct {
	Version   string   `json:"version"`
	GitCommit string   `json:"git_commit"`
	BuildTime string   `json:"build_time"`
	Platforms []string `json:"platforms"`
}

// GetVersionInfo returns version information for the current build.
func GetVersionInfo() *VersionInfo {
	version := os.Getenv("VERSION")
	if version == "" {
		version = "dev"
	}

	gitCommit := os.Getenv("GIT_COMMIT")
	if gitCommit == "" {
		gitCommit = "unknown"
	}

	buildTime := os.Getenv("BUILD_TIME")
	if buildTime == "" {
		buildTime = time.Now().UTC().Format(time.RFC3339)
	}

	platforms := make([]string, len(BuildSupportedPlatforms()))
	for i, p := range BuildSupportedPlatforms() {
		platforms[i] = p.String()
	}

	return &VersionInfo{
		Version:   version,
		GitCommit: gitCommit,
		BuildTime: buildTime,
		Platforms: platforms,
	}
}

// CrossCompile builds for all supported platforms.
func CrossCompile(config *BuildConfig) ([]string, error) {
	var built []string
	var errors []error

	for _, platform := range BuildSupportedPlatforms() {
		cfg := *config
		cfg.GOOS = platform.OS
		cfg.GOARCH = platform.Arch

		outputName := fmt.Sprintf("%s-%s-%s-%s", config.BinaryName, config.Version, platform.OS, platform.Arch)
		if platform.OS == "windows" {
			outputName += ".exe"
		}

		if err := cfg.Build(); err != nil {
			errors = append(errors, fmt.Errorf("failed to build for %s: %w", platform.String(), err))
			continue
		}

		built = append(built, outputName)
	}

	if len(errors) > 0 {
		return built, fmt.Errorf("build errors: %v", errors)
	}

	return built, nil
}

// Clean removes build artifacts.
func Clean(outputDir string) error {
	return os.RemoveAll(outputDir)
}

// ArchiveName generates the archive name for a platform.
func ArchiveName(binaryName, version string, platform BuildPlatform) string {
	return fmt.Sprintf("%s-%s-%s-%s%s", binaryName, version, platform.OS, platform.Arch, platform.ArchiveExtension())
}

// IsSupportedOS checks if an OS is supported.
func IsSupportedOS(os string) bool {
	for _, p := range BuildSupportedPlatforms() {
		if p.OS == os {
			return true
		}
	}
	return false
}

// IsSupportedArch checks if an architecture is supported for an OS.
func IsSupportedArch(os, arch string) bool {
	for _, p := range BuildSupportedPlatforms() {
		if p.OS == os && p.Arch == arch {
			return true
		}
	}
	return false
}
