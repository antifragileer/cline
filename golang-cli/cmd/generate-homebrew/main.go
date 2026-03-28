// Command generate-homebrew generates Homebrew formulas for the Cline CLI.
//
// This tool creates Homebrew formula files (.rb) for distributing the CLI
// via Homebrew on macOS and Linux.
//
// Usage:
//
//	# Generate formula for a release (downloads binaries to calculate SHA256)
//	generate-homebrew -version 1.0.0 -output Formula/cline.rb
//
//	# Generate formula using local binaries
//	generate-homebrew -version 1.0.0 -local -binary-dir ./dist -output Formula/cline.rb
//
//	# Create complete tap structure
//	generate-homebrew -version 1.0.0 -tap-dir ./homebrew-tap
//
//	# Dry run to preview formula
//	generate-homebrew -version 1.0.0 -dry-run
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cline/cline/golang-cli/scripts"
)

type config struct {
	version    string
	outputPath string
	baseURL    string
	binaryDir  string
	formulaName string
	tapDir     string
	localMode  bool
	dryRun     bool
}

func parseFlags() *config {
	cfg := &config{}

	flag.StringVar(&cfg.version, "version", "", "Release version (required)")
	flag.StringVar(&cfg.outputPath, "output", "", "Output path for formula file")
	flag.StringVar(&cfg.baseURL, "base-url", "https://github.com/cline/cline/releases/download", "Base URL for release downloads")
	flag.StringVar(&cfg.binaryDir, "binary-dir", "", "Directory containing pre-built binaries for local SHA256 calculation")
	flag.StringVar(&cfg.formulaName, "name", "cline", "Formula name")
	flag.StringVar(&cfg.tapDir, "tap-dir", "", "Create tap structure in this directory")
	flag.BoolVar(&cfg.localMode, "local", false, "Use local files for SHA256 calculation")
	flag.BoolVar(&cfg.dryRun, "dry-run", false, "Print formula without writing to file")

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

	return cfg
}

func (c *config) validate() error {
	if c.version == "" {
		return fmt.Errorf("version is required (use -version flag)")
	}

	if !c.dryRun && c.outputPath == "" && c.tapDir == "" {
		return fmt.Errorf("either -output, -tap-dir, or -dry-run must be specified")
	}

	if c.localMode && c.binaryDir == "" {
		return fmt.Errorf("binary-dir is required when using local mode")
	}

	return nil
}

func run(cfg *config) error {
	if err := cfg.validate(); err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	// Create tap structure if requested
	if cfg.tapDir != "" {
		tap := scripts.DefaultHomebrewTap()
		fmt.Fprintf(os.Stderr, "Creating tap structure in %s...\n", cfg.tapDir)
		if err := tap.CreateTapStructure(cfg.tapDir); err != nil {
			return fmt.Errorf("failed to create tap structure: %w", err)
		}
	}

	// Generate the formula
	fmt.Fprintf(os.Stderr, "Generating formula for version %s...\n", cfg.version)
	formula := scripts.DefaultHomebrewFormula(cfg.version)
	if cfg.formulaName != "" {
		formula.Name = cfg.formulaName
	}

	// Generate URLs and SHA256 for each platform
	platforms := scripts.HomebrewSupportedPlatforms()

	for _, p := range platforms {
		url := scripts.GenerateHomebrewPlatformURL(cfg.baseURL, cfg.version, p.OS, p.Arch)

		if cfg.localMode {
			// Use local file for SHA256 calculation
			binaryName := fmt.Sprintf("cline-%s-%s-%s.tar.gz", cfg.version, p.OS, p.Arch)
			localPath := filepath.Join(cfg.binaryDir, binaryName)

			if _, err := os.Stat(localPath); os.IsNotExist(err) {
				return fmt.Errorf("local binary not found: %s", localPath)
			}

			if err := formula.AddPlatformWithLocalFile(p.OS, p.Arch, url, localPath); err != nil {
				return fmt.Errorf("failed to add platform %s/%s: %w", p.OS, p.Arch, err)
			}
		} else {
			// Download and calculate SHA256
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
	if cfg.dryRun {
		fmt.Println(content)
		return nil
	}

	// Determine output path
	outputPath := cfg.outputPath
	if outputPath == "" && cfg.tapDir != "" {
		tap := scripts.DefaultHomebrewTap()
		outputPath = tap.GetFormulaPath(cfg.tapDir, formula.Name)
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
	cfg := parseFlags()

	if err := run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}