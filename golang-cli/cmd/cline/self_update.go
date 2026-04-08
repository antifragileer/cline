package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// execCommandContext is an alias for exec.CommandContext for testing
var execCommandContext = exec.CommandContext

// SelfUpdateFlags holds flags for the self-update command
var SelfUpdateFlags struct {
	Force   bool
	DryRun  bool
	Token   string
	BaseURL string
}

func init() {
	// Add self-update subcommand to update command
	updateCmd.AddCommand(selfUpdateCmd)

	selfUpdateCmd.Flags().BoolVarP(&SelfUpdateFlags.Force, "force", "f", false, "Force update even if versions match")
	selfUpdateCmd.Flags().BoolVarP(&SelfUpdateFlags.DryRun, "dry-run", "d", false, "Show what would be updated without making changes")
	selfUpdateCmd.Flags().StringVarP(&SelfUpdateFlags.Token, "token", "t", "", "GitHub token for authentication (optional)")
	selfUpdateCmd.Flags().StringVar(&SelfUpdateFlags.BaseURL, "base-url", "https://github.com/cline/cline/releases/download", "Base URL for release downloads")
}

// selfUpdateCmd represents the self-update subcommand
var selfUpdateCmd = &cobra.Command{
	Use:   "self-update",
	Short: "Update the Cline CLI binary itself",
	Long: `Download and install the latest version of the Cline CLI binary directly.

This command downloads the appropriate binary for your platform from GitHub releases
and replaces the current binary. It requires write permissions to the CLI installation
directory.`,
	Example: `  # Check for updates and update if available
  cline update self-update

  # Force update to latest version
  cline update self-update --force

  # Preview what would be updated without making changes
  cline update self-update --dry-run`,
	RunE: runSelfUpdate,
}

// runSelfUpdate executes the self-update process
func runSelfUpdate(cmd *cobra.Command, args []string) error {
	fmt.Fprintln(cmd.OutOrStdout(), "=== Cline CLI Self-Update ===")
	fmt.Fprintln(cmd.OutOrStdout())

	// Get current binary path
	currentBinary, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to determine current binary path: %w", err)
	}

	// Resolve symlinks to get the actual binary path
	currentBinary, err = filepath.EvalSymlinks(currentBinary)
	if err != nil {
		return fmt.Errorf("failed to resolve binary path: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Current binary: %s\n", currentBinary)
	fmt.Fprintf(cmd.OutOrStdout(), "Current version: %s\n", Version)

	// Check for latest version
	latestVersion, _, checksumURL, err := getLatestReleaseInfo()
	if err != nil {
		return fmt.Errorf("failed to get latest release info: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Latest version: %s\n", latestVersion)

	// Compare versions
	isNewer := isNewerVersion(Version, latestVersion)
	if !isNewer && !SelfUpdateFlags.Force {
		fmt.Fprintln(cmd.OutOrStdout(), "\n✓ Already running the latest version.")
		return nil
	}

	if !isNewer && SelfUpdateFlags.Force {
		fmt.Fprintln(cmd.OutOrStdout(), "\n⚠ Forcing update to same version.")
	}

	// Determine download URL
	downloadURL := buildDownloadURL(latestVersion)
	fmt.Fprintf(cmd.OutOrStdout(), "Download URL: %s\n", downloadURL)

	// Dry run - don't actually update
	if SelfUpdateFlags.DryRun {
		fmt.Fprintln(cmd.OutOrStdout(), "\n[DRY RUN] Would download and install:")
		fmt.Fprintf(cmd.OutOrStdout(), "  From: %s\n", downloadURL)
		fmt.Fprintf(cmd.OutOrStdout(), "  To: %s\n", currentBinary)
		return nil
	}

	// Confirm with user
	if !SelfUpdateFlags.Force {
		fmt.Fprint(cmd.OutOrStdout(), "\nProceed with update? [y/N]: ")
		var response string
		fmt.Fscanln(cmd.InOrStdin(), &response)
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Fprintln(cmd.OutOrStdout(), "Update cancelled.")
			return nil
		}
	}

	// Download new binary
	fmt.Fprintln(cmd.OutOrStdout(), "\nDownloading latest version...")
	tempFile, err := downloadBinary(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}
	defer os.Remove(tempFile)

	// Verify checksum if available
	if checksumURL != "" {
		fmt.Fprintln(cmd.OutOrStdout(), "Verifying checksum...")
		if err := verifyChecksum(tempFile, checksumURL); err != nil {
			return fmt.Errorf("checksum verification failed: %w", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "✓ Checksum verified")
	}

	// Make new binary executable
	if err := os.Chmod(tempFile, 0755); err != nil {
		return fmt.Errorf("failed to make binary executable: %w", err)
	}

	// Test new binary
	fmt.Fprintln(cmd.OutOrStdout(), "Testing new binary...")
	testOutput, err := testBinary(tempFile)
	if err != nil {
		return fmt.Errorf("new binary test failed: %w", err)
	}

	// Extract version from test output
	newVersion := extractVersion(testOutput)
	if newVersion == "" {
		newVersion = latestVersion
	}
	fmt.Fprintf(cmd.OutOrStdout(), "✓ New binary version: %s\n", newVersion)

	// Backup current binary
	backupPath := currentBinary + ".backup"
	fmt.Fprintf(cmd.OutOrStdout(), "Creating backup: %s\n", backupPath)
	if err := backupBinary(currentBinary, backupPath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Replace current binary with new one
	fmt.Fprintln(cmd.OutOrStdout(), "Installing new binary...")
	if err := replaceBinary(currentBinary, tempFile); err != nil {
		// Try to restore backup
		restoreErr := restoreBackup(backupPath, currentBinary)
		if restoreErr != nil {
			return fmt.Errorf("failed to install new binary and restore backup: %v (restore error: %v)", err, restoreErr)
		}
		return fmt.Errorf("failed to install new binary, restored previous version: %w", err)
	}

	// Clean up backup on successful update
	os.Remove(backupPath)

	fmt.Fprintln(cmd.OutOrStdout(), "\n✓ Update successful!")
	fmt.Fprintf(cmd.OutOrStdout(), "Updated from %s to %s\n", Version, newVersion)
	fmt.Fprintln(cmd.OutOrStdout(), "\nPlease restart any running Cline CLI instances.")

	return nil
}

// getLatestReleaseInfo fetches the latest release information from GitHub
func getLatestReleaseInfo() (version, releaseURL, checksumURL string, err error) {
	client := &http.Client{Timeout: 30 * time.Second}

	// Use GitHub API to get latest release
	apiURL := "https://api.github.com/repos/cline/cline/releases/latest"
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", "", "", err
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if SelfUpdateFlags.Token != "" {
		req.Header.Set("Authorization", "token "+SelfUpdateFlags.Token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	// Parse response
	var release struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", "", "", fmt.Errorf("failed to parse release info: %w", err)
	}

	version = strings.TrimPrefix(release.TagName, "v")

	// Find download URLs
	platform := runtime.GOOS
	arch := runtime.GOARCH

	// Map Go arch to release arch
	if arch == "amd64" {
		arch = "amd64"
	}

	expectedAsset := fmt.Sprintf("cline_%s_%s_%s.tar.gz", version, platform, arch)
	expectedChecksum := expectedAsset + ".sha256"

	for _, asset := range release.Assets {
		if asset.Name == expectedAsset {
			releaseURL = asset.BrowserDownloadURL
		}
		if asset.Name == expectedChecksum {
			checksumURL = asset.BrowserDownloadURL
		}
	}

	if releaseURL == "" {
		return "", "", "", fmt.Errorf("no release asset found for %s/%s", platform, arch)
	}

	return version, releaseURL, checksumURL, nil
}

// buildDownloadURL builds the download URL for the current platform
func buildDownloadURL(version string) string {
	platform := runtime.GOOS
	arch := runtime.GOARCH

	// Normalize version
	version = strings.TrimPrefix(version, "v")

	return fmt.Sprintf("%s/v%s/cline_%s_%s_%s.tar.gz",
		SelfUpdateFlags.BaseURL, version, version, platform, arch)
}

// downloadBinary downloads the binary to a temporary file
func downloadBinary(url string) (string, error) {
	client := &http.Client{Timeout: 5 * time.Minute}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	// Add authorization if token provided
	if SelfUpdateFlags.Token != "" {
		req.Header.Set("Authorization", "token "+SelfUpdateFlags.Token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Create temp file
	tempFile, err := os.CreateTemp("", "cline-update-*.tar.gz")
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	// Copy and extract
	if err := extractBinary(resp.Body, tempFile.Name()); err != nil {
		return "", err
	}

	return tempFile.Name(), nil
}

// extractBinary extracts the binary from a tar.gz archive
func extractBinary(r io.Reader, destPath string) error {
	// Create temp directory for extraction
	tempDir, err := os.MkdirTemp("", "cline-extract-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	// Save archive to temp file
	archivePath := filepath.Join(tempDir, "archive.tar.gz")
	archiveFile, err := os.Create(archivePath)
	if err != nil {
		return err
	}

	if _, err := io.Copy(archiveFile, r); err != nil {
		archiveFile.Close()
		return err
	}
	archiveFile.Close()

	// Open archive
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	// Decompress gzip
	gzr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("failed to decompress: %w", err)
	}
	defer gzr.Close()

	// Extract tar
	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read archive: %w", err)
		}

		// Skip non-files and directories
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA {
			continue
		}

		// Look for the cline binary
		if header.Name == "cline" || header.Name == "./cline" || strings.HasSuffix(header.Name, "/cline") {
			// Create output file
			outFile, err := os.Create(destPath)
			if err != nil {
				return fmt.Errorf("failed to create output file: %w", err)
			}

			// Copy with progress
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to extract binary: %w", err)
			}
			outFile.Close()

			return nil
		}
	}

	return fmt.Errorf("binary not found in archive")
}

// verifyChecksum verifies the SHA256 checksum of the downloaded file
func verifyChecksum(filePath, checksumURL string) error {
	// Download checksum
	client := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequest("GET", checksumURL, nil)
	if err != nil {
		return err
	}

	if SelfUpdateFlags.Token != "" {
		req.Header.Set("Authorization", "token "+SelfUpdateFlags.Token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download checksum: status %d", resp.StatusCode)
	}

	// Read expected checksum
	checksumData, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Parse checksum (format: "<hash>  <filename>")
	checksumParts := strings.Fields(string(checksumData))
	if len(checksumParts) < 1 {
		return fmt.Errorf("invalid checksum format")
	}
	expectedChecksum := checksumParts[0]

	// Calculate actual checksum
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return err
	}

	actualChecksum := hex.EncodeToString(hasher.Sum(nil))

	if !strings.EqualFold(actualChecksum, expectedChecksum) {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}

	return nil
}

// testBinary tests that the binary works by running --version
func testBinary(binaryPath string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := execCommandContext(ctx, binaryPath, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("binary test failed: %w (output: %s)", err, string(output))
	}

	return string(output), nil
}

// extractVersion extracts version from version command output
func extractVersion(output string) string {
	// Look for version in output (e.g., "cline version 1.2.3")
	parts := strings.Fields(output)
	for i, part := range parts {
		if part == "version" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// backupBinary creates a backup of the current binary
func backupBinary(currentPath, backupPath string) error {
	// Remove old backup if exists
	os.Remove(backupPath)

	// Copy current binary to backup
	input, err := os.Open(currentPath)
	if err != nil {
		return err
	}
	defer input.Close()

	output, err := os.Create(backupPath)
	if err != nil {
		return err
	}
	defer output.Close()

	if _, err := io.Copy(output, input); err != nil {
		return err
	}

	// Copy permissions
	info, err := input.Stat()
	if err != nil {
		return err
	}

	return os.Chmod(backupPath, info.Mode())
}

// replaceBinary replaces the current binary with the new one
func replaceBinary(currentPath, newPath string) error {
	// On Windows, we can't overwrite a running binary directly
	// We need to use a different approach
	if runtime.GOOS == "windows" {
		return replaceBinaryWindows(currentPath, newPath)
	}

	// On Unix, we can rename the new binary over the old one
	return os.Rename(newPath, currentPath)
}

// replaceBinaryWindows handles the Windows-specific replacement
func replaceBinaryWindows(currentPath, newPath string) error {
	// On Windows, rename the current binary to .old, then move new binary
	oldPath := currentPath + ".old"

	// Remove any existing .old file
	os.Remove(oldPath)

	// Rename current to .old
	if err := os.Rename(currentPath, oldPath); err != nil {
		return fmt.Errorf("failed to rename current binary: %w", err)
	}

	// Move new binary to current location
	if err := os.Rename(newPath, currentPath); err != nil {
		// Try to restore
		os.Rename(oldPath, currentPath)
		return fmt.Errorf("failed to install new binary: %w", err)
	}

	// Remove .old file
	os.Remove(oldPath)

	return nil
}

// restoreBackup restores the backup binary
func restoreBackup(backupPath, currentPath string) error {
	// Remove failed new binary
	os.Remove(currentPath)

	// Restore backup
	return os.Rename(backupPath, currentPath)
}
