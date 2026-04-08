// Package scripts provides build and distribution utilities for the Cline CLI.
// This file implements the CLI tool for generating Linux packages (DEB/RPM).
package scripts

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

// LinuxPackageConfig holds the configuration for generating Linux packages.
type LinuxPackageCLIConfig struct {
	// Version is the release version
	Version string

	// BinaryDir is the directory containing pre-built binaries
	BinaryDir string

	// OutputDir is the directory for output packages
	OutputDir string

	// PackageTypes specifies which packages to build (deb, rpm, or both)
	PackageTypes []string

	// Architectures specifies which architectures to build for
	Architectures []string

	// DryRun prints what would be done without executing
	DryRun bool
}

// ParseFlags parses command-line flags and returns a LinuxPackageCLIConfig.
func ParseLinuxPackageFlags() *LinuxPackageCLIConfig {
	config := &LinuxPackageCLIConfig{
		PackageTypes:  []string{"deb", "rpm"},
		Architectures: []string{"amd64", "arm64"},
	}

	var pkgTypes string
	var arches string

	flag.StringVar(&config.Version, "version", "", "Release version (required)")
	flag.StringVar(&config.BinaryDir, "binary-dir", "", "Directory containing pre-built binaries")
	flag.StringVar(&config.OutputDir, "output-dir", "dist", "Directory for output packages")
	flag.StringVar(&pkgTypes, "types", "deb,rpm", "Package types to build (comma-separated: deb,rpm)")
	flag.StringVar(&arches, "arch", "amd64,arm64", "Architectures to build (comma-separated)")
	flag.BoolVar(&config.DryRun, "dry-run", false, "Print what would be done without executing")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Generate Linux packages (DEB/RPM) for Cline CLI.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Generate all packages\n")
		fmt.Fprintf(os.Stderr, "  %s -version 1.0.0 -binary-dir ./dist\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Generate only DEB packages\n")
		fmt.Fprintf(os.Stderr, "  %s -version 1.0.0 -binary-dir ./dist -types deb\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Generate for specific architecture\n")
		fmt.Fprintf(os.Stderr, "  %s -version 1.0.0 -binary-dir ./dist -arch amd64\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Dry run\n")
		fmt.Fprintf(os.Stderr, "  %s -version 1.0.0 -binary-dir ./dist -dry-run\n", os.Args[0])
	}

	flag.Parse()

	// Parse comma-separated values
	if pkgTypes != "" {
		config.PackageTypes = splitAndTrim(pkgTypes, ",")
	}
	if arches != "" {
		config.Architectures = splitAndTrim(arches, ",")
	}

	return config
}

// splitAndTrim splits a string and trims whitespace from each part
func splitAndTrim(s, sep string) []string {
	parts := []string{}
	start := 0
	for i := 0; i < len(s); i++ {
		if i == len(s)-1 || string(s[i]) == sep {
			end := i + 1
			if i == len(s)-1 && string(s[i]) != sep {
				end = i + 1
			}
			part := s[start:end]
			if string(s[i]) == sep {
				part = s[start:i]
			}
			if trimmed := trimSpace(part); trimmed != "" {
				parts = append(parts, trimmed)
			}
			start = i + 1
		}
	}
	return parts
}

// trimSpace removes leading and trailing whitespace
func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

// Validate checks if the configuration is valid.
func (c *LinuxPackageCLIConfig) Validate() error {
	if c.Version == "" {
		return fmt.Errorf("version is required (use -version flag)")
	}

	if c.BinaryDir == "" {
		return fmt.Errorf("binary-dir is required (use -binary-dir flag)")
	}

	if _, err := os.Stat(c.BinaryDir); os.IsNotExist(err) {
		return fmt.Errorf("binary directory does not exist: %s", c.BinaryDir)
	}

	if len(c.PackageTypes) == 0 {
		return fmt.Errorf("at least one package type must be specified")
	}

	for _, pkgType := range c.PackageTypes {
		if pkgType != "deb" && pkgType != "rpm" {
			return fmt.Errorf("invalid package type: %s (must be 'deb' or 'rpm')", pkgType)
		}
	}

	if len(c.Architectures) == 0 {
		return fmt.Errorf("at least one architecture must be specified")
	}

	return nil
}

// RunLinuxPackages executes the Linux package generation.
func RunLinuxPackages(config *LinuxPackageCLIConfig) error {
	if err := config.Validate(); err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	if config.DryRun {
		fmt.Println("Dry run mode - would generate the following packages:")
	}

	// Create output directory
	if !config.DryRun {
		if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	generatedPackages := []string{}

	// Generate packages for each type and architecture
	for _, pkgType := range config.PackageTypes {
		for _, arch := range config.Architectures {
			var goArch string
			switch arch {
			case "amd64":
				goArch = "amd64"
			case "arm64":
				goArch = "arm64"
			case "i386", "386":
				goArch = "386"
			case "armhf", "arm":
				goArch = "arm"
			default:
				goArch = arch
			}

			// Find source binary
			sourceFile := filepath.Join(config.BinaryDir, fmt.Sprintf("cline-%s-linux-%s", config.Version, goArch))
			if _, err := os.Stat(sourceFile); os.IsNotExist(err) {
				// Try with .tar.gz extension
				sourceFile = sourceFile + ".tar.gz"
				if _, err := os.Stat(sourceFile); os.IsNotExist(err) {
					fmt.Fprintf(os.Stderr, "Warning: source binary not found for %s/%s: %s\n", pkgType, arch, sourceFile)
					continue
				}
			}

			if config.DryRun {
				fmt.Printf("  - %s package for %s: %s -> %s/\n", pkgType, arch, sourceFile, config.OutputDir)
				continue
			}

			// Create package configuration
			var pkgConfig *LinuxPackageConfig
			if pkgType == "deb" {
				pkgConfig = DefaultLinuxPackageConfig(PackageTypeDEB, config.Version, arch)
			} else {
				pkgConfig = DefaultLinuxPackageConfig(PackageTypeRPM, config.Version, arch)
			}

			pkgConfig.SourcePath = sourceFile
			pkgConfig.OutputDir = config.OutputDir

			// Build package
			fmt.Fprintf(os.Stderr, "Building %s package for %s...\n", pkgType, arch)
			builder := NewLinuxPackageBuilder(pkgConfig)
			pkgPath, err := builder.Build()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to build %s package for %s: %v\n", pkgType, arch, err)
				continue
			}

			generatedPackages = append(generatedPackages, pkgPath)
			fmt.Fprintf(os.Stderr, "  Created: %s\n", pkgPath)
		}
	}

	if config.DryRun {
		return nil
	}

	fmt.Fprintf(os.Stderr, "\nSuccessfully generated %d packages:\n", len(generatedPackages))
	for _, pkg := range generatedPackages {
		fmt.Fprintf(os.Stderr, "  - %s\n", filepath.Base(pkg))
	}

	return nil
}

// GenerateLinuxPackages generates packages for a specific release version.
func GenerateLinuxPackages(version, binaryDir, outputDir string, pkgTypes []string) error {
	config := &LinuxPackageCLIConfig{
		Version:       version,
		BinaryDir:     binaryDir,
		OutputDir:     outputDir,
		PackageTypes:  pkgTypes,
		Architectures: []string{"amd64", "arm64"},
	}

	return RunLinuxPackages(config)
}

// GenerateDEBPackages generates only DEB packages.
func GenerateDEBPackages(version, binaryDir, outputDir string) error {
	return GenerateLinuxPackages(version, binaryDir, outputDir, []string{"deb"})
}

// GenerateRPMPackages generates only RPM packages.
func GenerateRPMPackages(version, binaryDir, outputDir string) error {
	return GenerateLinuxPackages(version, binaryDir, outputDir, []string{"rpm"})
}

// CreateAPTRepository creates an APT repository structure.
func CreateAPTRepository(rootPath, version, binaryDir string) error {
	config := &LinuxPackageCLIConfig{
		Version:       version,
		BinaryDir:     binaryDir,
		OutputDir:     filepath.Join(rootPath, "pool", "main"),
		PackageTypes:  []string{"deb"},
		Architectures: []string{"amd64", "arm64"},
	}

	if err := RunLinuxPackages(config); err != nil {
		return err
	}

	// Generate repository metadata
	metadata := &LinuxPackageMetadata{
		Distribution: "stable",
		Component:    "main",
		Architecture: "amd64",
	}

	// In a real implementation, we would scan the pool directory
	// and add all packages to the metadata
	fmt.Fprintf(os.Stderr, "APT repository structure created at: %s\n", rootPath)
	fmt.Fprintf(os.Stderr, "To complete the repository, run: apt-ftparchive packages . > Packages\n")

	return GenerateAPTRepository(rootPath, metadata)
}

// CheckLinuxBuildTools checks if required build tools are installed.
func CheckLinuxBuildTools(pkgType string) error {
	tools := []string{}
	if pkgType == "deb" || pkgType == "" {
		tools = append(tools, "dpkg-deb")
	}
	if pkgType == "rpm" || pkgType == "" {
		tools = append(tools, "rpmbuild")
	}

	var missing []string
	for _, tool := range tools {
		if !commandExists(tool) {
			missing = append(missing, tool)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required tools: %v", missing)
	}

	return nil
}

// commandExists checks if a command exists in PATH
func commandExists(cmd string) bool {
	_, err := os.Stat("/usr/bin/" + cmd)
	if err == nil {
		return true
	}
	_, err = os.Stat("/usr/local/bin/" + cmd)
	return err == nil
}

// GetLinuxPackageFilename returns the expected filename for a package.
func GetLinuxPackageFilename(name, version, arch string, pkgType LinuxPackageType) string {
	if pkgType == PackageTypeDEB {
		return fmt.Sprintf("%s_%s_%s.deb", name, version, arch)
	}
	return fmt.Sprintf("%s-%s-1.%s.rpm", name, version, arch)
}

// ListLinuxPackageRequirements returns the build requirements.
func ListLinuxPackageRequirements() map[string][]string {
	return map[string][]string{
		"deb": {"dpkg-deb", "fakeroot"},
		"rpm": {"rpmbuild", "rpm-build"},
	}
}
