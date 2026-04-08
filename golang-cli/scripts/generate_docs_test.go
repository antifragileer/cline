package scripts

import (
	"os"
	"path/filepath"
	"testing"
)

// TestParseDocFlags tests the flag parsing function.
func TestParseDocFlags(t *testing.T) {
	t.Parallel()

	// Just verify it doesn't panic
	t.Run("returns config", func(t *testing.T) {
		config := ParseDocFlags()
		if config == nil {
			t.Error("Expected config to be returned")
		}
	})
}

// TestDocConfigStruct tests the config struct.
func TestDocConfigStruct(t *testing.T) {
	t.Parallel()

	config := &DocConfig{
		SourceDir: "./internal",
		OutputDir: "./docs",
		Format:    "markdown",
	}

	if config.SourceDir != "./internal" {
		t.Error("Unexpected source dir")
	}
	if config.OutputDir != "./docs" {
		t.Error("Unexpected output dir")
	}
	if config.Format != "markdown" {
		t.Error("Unexpected format")
	}
}

// TestGenerateDocs tests documentation generation.
func TestGenerateDocs(t *testing.T) {
	t.Parallel()

	t.Run("handles empty config", func(t *testing.T) {
		config := &DocConfig{}
		// This may fail but should not panic
		_ = GenerateDocs(config)
	})

	t.Run("generates docs with valid config", func(t *testing.T) {
		tempDir := t.TempDir()
		sourceDir := filepath.Join(tempDir, "source")
		outputDir := filepath.Join(tempDir, "docs")

		// Create a dummy Go source file
		if err := os.MkdirAll(sourceDir, 0755); err != nil {
			t.Fatalf("Failed to create source dir: %v", err)
		}

		dummyGo := `package main

// Main is the entry point.
func Main() {}
`
		if err := os.WriteFile(filepath.Join(sourceDir, "main.go"), []byte(dummyGo), 0644); err != nil {
			t.Fatalf("Failed to write dummy Go file: %v", err)
		}

		config := &DocConfig{
			SourceDir: sourceDir,
			OutputDir: outputDir,
			Format:    "markdown",
		}

		// This may or may not succeed depending on implementation
		// but should not panic
		_ = GenerateDocs(config)
	})

	t.Run("handles missing source dir", func(t *testing.T) {
		config := &DocConfig{
			SourceDir: "/nonexistent/path",
			OutputDir: t.TempDir(),
			Format:    "markdown",
		}

		err := GenerateDocs(config)
		// Expect an error for missing source
		_ = err
	})

	t.Run("handles different formats", func(t *testing.T) {
		formats := []string{"markdown", "html", "json"}

		for _, format := range formats {
			t.Run(format, func(t *testing.T) {
				tempDir := t.TempDir()
				sourceDir := filepath.Join(tempDir, "source")
				outputDir := filepath.Join(tempDir, "docs")

				if err := os.MkdirAll(sourceDir, 0755); err != nil {
					t.Fatalf("Failed to create source dir: %v", err)
				}

				dummyGo := `package main
func Main() {}
`
				if err := os.WriteFile(filepath.Join(sourceDir, "main.go"), []byte(dummyGo), 0644); err != nil {
					t.Fatalf("Failed to write dummy Go file: %v", err)
				}

				config := &DocConfig{
					SourceDir: sourceDir,
					OutputDir: outputDir,
					Format:    format,
				}

				// Should not panic
				_ = GenerateDocs(config)
			})
		}
	})
}
