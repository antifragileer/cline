package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// updateFlags holds the parsed flag values for update command
var updateFlags struct {
	checkOnly bool
	force     bool
	verbose   bool
}

// NPMRegistryResponse represents the response from npm registry
type NPMRegistryResponse struct {
	Version string `json:"version"`
	Dist    struct {
		TagLine string `json:"tagline"`
	} `json:"dist"`
	Time struct {
		Modified string `json:"modified"`
	} `json:"time"`
	Homepage string `json:"homepage"`
}

// UpdateInfo holds information about available updates
type UpdateInfo struct {
	CurrentVersion  string
	LatestVersion   string
	UpdateAvailable bool
	ReleaseDate     string
	Homepage        string
}

// npmRegistryURL is the URL for checking the latest version
const npmRegistryURL = "https://registry.npmjs.org/cline/latest"

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Check for and install updates",
	Long: `Check for and install updates for Cline CLI.

This command checks the npm registry for the latest version of Cline CLI
and can optionally install the update.`,
	Example: `  # Check for updates
  cline update

  # Check for updates without installing
  cline update --check-only

  # Force update check
  cline update --force`,
	RunE: runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)

	// Add flags to update command
	updateCmd.Flags().BoolVarP(&updateFlags.checkOnly, "check-only", "c", false, "Only check for updates, don't install")
	updateCmd.Flags().BoolVarP(&updateFlags.force, "force", "f", false, "Force update check")
	updateCmd.Flags().BoolVarP(&updateFlags.verbose, "verbose", "v", false, "Show verbose output")
}

// runUpdate executes the update command
func runUpdate(cmd *cobra.Command, args []string) error {
	// Check for updates
	updateInfo, err := checkForUpdate()
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	// Display update information
	if err := displayUpdateInfo(cmd.OutOrStdout(), updateInfo); err != nil {
		return err
	}

	// Show verbose information if requested
	if updateFlags.verbose {
		if err := displayVerboseInfo(cmd.OutOrStdout(), updateInfo); err != nil {
			return err
		}
	}

	// If check-only or no update available, we're done
	if updateFlags.checkOnly || !updateInfo.UpdateAvailable {
		return nil
	}

	// Prompt user for update
	fmt.Fprintln(cmd.OutOrStdout(), "\nRun the following command to update:")
	fmt.Fprintf(cmd.OutOrStdout(), "  npm install -g cline@%s\n", updateInfo.LatestVersion)

	return nil
}

// checkForUpdate checks the npm registry for the latest version
func checkForUpdate() (*UpdateInfo, error) {
	info := &UpdateInfo{
		CurrentVersion: Version,
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Fetch latest version from npm registry
	resp, err := client.Get(npmRegistryURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch version info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("npm registry returned status %d", resp.StatusCode)
	}

	// Parse response
	var registryResp NPMRegistryResponse
	if err := json.NewDecoder(resp.Body).Decode(&registryResp); err != nil {
		return nil, fmt.Errorf("failed to parse registry response: %w", err)
	}

	info.LatestVersion = registryResp.Version
	info.ReleaseDate = registryResp.Time.Modified
	info.Homepage = registryResp.Homepage

	// Compare versions
	info.UpdateAvailable = isNewerVersion(Version, registryResp.Version)

	return info, nil
}

// displayUpdateInfo displays update information
func displayUpdateInfo(output io.Writer, info *UpdateInfo) error {
	fmt.Fprintln(output, "=== Cline CLI Update ===")
	fmt.Fprintf(output, "Current version: %s\n", info.CurrentVersion)
	fmt.Fprintf(output, "Latest version:  %s\n", info.LatestVersion)

	if info.UpdateAvailable {
		fmt.Fprintln(output, "\n✓ Update available!")
		if info.ReleaseDate != "" {
			fmt.Fprintf(output, "Released: %s\n", formatReleaseDate(info.ReleaseDate))
		}
		if info.Homepage != "" {
			fmt.Fprintf(output, "Release notes: %s\n", info.Homepage)
		}
	} else {
		fmt.Fprintln(output, "\n✓ You're running the latest version.")
	}

	return nil
}

// displayVerboseInfo displays verbose update information
func displayVerboseInfo(output io.Writer, info *UpdateInfo) error {
	fmt.Fprintln(output, "\n=== Verbose Information ===")
	fmt.Fprintf(output, "Registry URL: %s\n", npmRegistryURL)
	fmt.Fprintf(output, "Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Fprintf(output, "Go Version: %s\n", runtime.Version())

	if info.ReleaseDate != "" {
		fmt.Fprintf(output, "Release Date (raw): %s\n", info.ReleaseDate)
	}

	return nil
}

// isNewerVersion compares two semantic versions
// Returns true if latest is newer than current
func isNewerVersion(current, latest string) bool {
	// Normalize versions by removing 'v' prefix
	current = strings.TrimPrefix(current, "v")
	latest = strings.TrimPrefix(latest, "v")

	// Parse version components
	currentParts := parseVersion(current)
	latestParts := parseVersion(latest)

	// Compare major, minor, patch
	for i := 0; i < 3; i++ {
		if latestParts[i] > currentParts[i] {
			return true
		}
		if latestParts[i] < currentParts[i] {
			return false
		}
	}

	return false
}

// parseVersion parses a semantic version string into components
func parseVersion(version string) [3]int {
	// Trim 'v' prefix if present
	version = strings.TrimPrefix(version, "v")

	parts := strings.Split(version, ".")
	result := [3]int{0, 0, 0}

	for i := 0; i < 3 && i < len(parts); i++ {
		// Extract numeric part (handle pre-release tags like "1.0.0-beta")
		numericPart := parts[i]
		for j, c := range numericPart {
			if c < '0' || c > '9' {
				numericPart = numericPart[:j]
				break
			}
		}

		if numericPart != "" {
			fmt.Sscanf(numericPart, "%d", &result[i])
		}
	}

	return result
}

// formatReleaseDate formats the release date for display
func formatReleaseDate(date string) string {
	// Try to parse ISO 8601 format
	t, err := time.Parse(time.RFC3339, date)
	if err != nil {
		// Try alternative format
		t, err = time.Parse("2006-01-02T15:04:05.000Z", date)
		if err != nil {
			return date
		}
	}

	// Calculate time ago
	duration := time.Since(t)
	if duration < time.Hour {
		return "just now"
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else if duration < 30*24*time.Hour {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	} else {
		return t.Format("Jan 2, 2006")
	}
}

// GetLatestVersion returns the latest version available from npm
func GetLatestVersion() (string, error) {
	info, err := checkForUpdate()
	if err != nil {
		return "", err
	}
	return info.LatestVersion, nil
}

// UpdateChecker provides methods for checking updates
type UpdateChecker struct {
	client      *http.Client
	registryURL string
}

// NewUpdateChecker creates a new UpdateChecker
func NewUpdateChecker() *UpdateChecker {
	return &UpdateChecker{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		registryURL: npmRegistryURL,
	}
}

// Check checks for available updates
func (u *UpdateChecker) Check(currentVersion string) (*UpdateInfo, error) {
	info := &UpdateInfo{
		CurrentVersion: currentVersion,
	}

	resp, err := u.client.Get(u.registryURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned status %d", resp.StatusCode)
	}

	var registryResp NPMRegistryResponse
	if err := json.NewDecoder(resp.Body).Decode(&registryResp); err != nil {
		return nil, err
	}

	info.LatestVersion = registryResp.Version
	info.ReleaseDate = registryResp.Time.Modified
	info.Homepage = registryResp.Homepage
	info.UpdateAvailable = isNewerVersion(currentVersion, registryResp.Version)

	return info, nil
}

// AutoUpdateCheck performs an automatic update check
// This can be called during startup to notify users of updates
func AutoUpdateCheck() {
	// Skip if CLINE_NO_UPDATE_CHECK is set
	if os.Getenv("CLINE_NO_UPDATE_CHECK") != "" {
		return
	}

	checker := NewUpdateChecker()
	info, err := checker.Check(Version)
	if err != nil {
		// Silently fail - don't interrupt user workflow
		return
	}

	if info.UpdateAvailable {
		fmt.Fprintf(os.Stderr, "\n📦 A new version of Cline CLI is available: %s → %s\n",
			info.CurrentVersion, info.LatestVersion)
		fmt.Fprintf(os.Stderr, "   Run 'cline update' for more information.\n\n")
	}
}

// GetPlatformInfo returns platform information for update purposes
func GetPlatformInfo() map[string]string {
	return map[string]string{
		"os":      runtime.GOOS,
		"arch":    runtime.GOARCH,
		"version": Version,
	}
}
