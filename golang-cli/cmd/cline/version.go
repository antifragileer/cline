package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"time"

	"github.com/spf13/cobra"
)

// Build information variables - set by ldflags during build
var (
	// BuildDate is the date when the binary was built
	BuildDate = "unknown"

	// GitCommit is the git commit hash
	GitCommit = "unknown"

	// GitBranch is the git branch name
	GitBranch = "unknown"

	// BuildHost is the hostname where the binary was built
	BuildHost = "unknown"

	// GoVersion is the Go version used to build
	GoVersion = runtime.Version()
)

// VersionInfo holds all version and build information
type VersionInfo struct {
	// Version is the semantic version
	Version string `json:"version"`

	// BuildDate is the build timestamp
	BuildDate string `json:"buildDate"`

	// GitCommit is the git commit hash
	GitCommit string `json:"gitCommit"`

	// GitBranch is the git branch name
	GitBranch string `json:"gitBranch"`

	// BuildHost is the build hostname
	BuildHost string `json:"buildHost"`

	// GoVersion is the Go version
	GoVersion string `json:"goVersion"`

	// OS is the operating system
	OS string `json:"os"`

	// Arch is the architecture
	Arch string `json:"arch"`

	// Compiler is the compiler used
	Compiler string `json:"compiler"`
}

// versionFlags holds the parsed flag values for version command
var versionFlags struct {
	json    bool
	short   bool
	verbose bool
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long: `Show detailed version information about Cline CLI.

This command displays the current version, build date, git commit,
and other build-related information.`,
	Example: `  # Show version
  cline version

  # Show short version
  cline version --short

  # Show version as JSON
  cline version --json

  # Show verbose version information
  cline version --verbose`,
	RunE: runVersion,
}

func init() {
	rootCmd.AddCommand(versionCmd)

	// Add flags to version command
	versionCmd.Flags().BoolVarP(&versionFlags.json, "json", "j", false, "Output in JSON format")
	versionCmd.Flags().BoolVarP(&versionFlags.short, "short", "s", false, "Show only version number")
	versionCmd.Flags().BoolVarP(&versionFlags.verbose, "verbose", "v", false, "Show verbose information")
}

// runVersion executes the version command
func runVersion(cmd *cobra.Command, args []string) error {
	info := GetVersionInfo()

	// Short format - just version number
	if versionFlags.short {
		fmt.Fprintln(cmd.OutOrStdout(), info.Version)
		return nil
	}

	// JSON format
	if versionFlags.json {
		return outputVersionJSON(cmd.OutOrStdout(), info)
	}

	// Verbose format
	if versionFlags.verbose {
		return outputVersionVerbose(cmd.OutOrStdout(), info)
	}

	// Standard format
	return outputVersionStandard(cmd.OutOrStdout(), info)
}

// GetVersionInfo returns the complete version information
func GetVersionInfo() *VersionInfo {
	return &VersionInfo{
		Version:   Version,
		BuildDate: BuildDate,
		GitCommit: GitCommit,
		GitBranch: GitBranch,
		BuildHost: BuildHost,
		GoVersion: GoVersion,
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		Compiler:  runtime.Compiler,
	}
}

// outputVersionJSON outputs version info as JSON
func outputVersionJSON(output io.Writer, info *VersionInfo) error {
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	return encoder.Encode(info)
}

// outputVersionStandard outputs version info in standard format
func outputVersionStandard(output io.Writer, info *VersionInfo) error {
	// Use cyan color for the version (matches TypeScript chalk.cyan)
	cyan := "\033[36m"
	reset := "\033[0m"
	
	fmt.Fprintf(output, "Cline CLI version: %s%s%s\n", cyan, info.Version, reset)

	if info.GitCommit != "unknown" {
		fmt.Fprintf(output, "  Git commit: %s\n", formatCommit(info.GitCommit))
	}

	if info.BuildDate != "unknown" {
		fmt.Fprintf(output, "  Built: %s\n", formatBuildDate(info.BuildDate))
	}

	fmt.Fprintf(output, "  Go version: %s\n", info.GoVersion)
	fmt.Fprintf(output, "  OS/Arch: %s/%s\n", info.OS, info.Arch)

	return nil
}

// outputVersionVerbose outputs verbose version information
func outputVersionVerbose(output io.Writer, info *VersionInfo) error {
	fmt.Fprintln(output, "=== Cline CLI Version Information ===")
	fmt.Fprintln(output)

	fmt.Fprintf(output, "Version:        %s\n", info.Version)
	fmt.Fprintf(output, "Go Version:     %s\n", info.GoVersion)
	fmt.Fprintf(output, "Compiler:       %s\n", info.Compiler)
	fmt.Fprintf(output, "OS/Arch:        %s/%s\n", info.OS, info.Arch)
	fmt.Fprintln(output)

	fmt.Fprintln(output, "=== Build Information ===")
	fmt.Fprintf(output, "Build Date:     %s\n", info.BuildDate)
	fmt.Fprintf(output, "Git Commit:     %s\n", info.GitCommit)
	fmt.Fprintf(output, "Git Branch:     %s\n", info.GitBranch)
	fmt.Fprintf(output, "Build Host:     %s\n", info.BuildHost)
	fmt.Fprintln(output)

	fmt.Fprintln(output, "=== Runtime Information ===")
	fmt.Fprintf(output, "NumCPU:         %d\n", runtime.NumCPU())
	fmt.Fprintf(output, "GOROOT:         %s\n", runtime.GOROOT())

	return nil
}

// formatCommit formats a git commit hash for display
func formatCommit(commit string) string {
	if len(commit) > 7 {
		return commit[:7]
	}
	return commit
}

// formatBuildDate formats the build date for display
func formatBuildDate(date string) string {
	// Try to parse and reformat
	t, err := time.Parse(time.RFC3339, date)
	if err != nil {
		// Try other common formats
		formats := []string{
			"2006-01-02 15:04:05",
			"2006-01-02T15:04:05",
			time.RFC1123,
			time.UnixDate,
		}

		for _, format := range formats {
			t, err = time.Parse(format, date)
			if err == nil {
				break
			}
		}
	}

	if err == nil {
		return t.Format("Jan 2, 2006 15:04:05 MST")
	}

	return date
}

// PrintVersion prints the version to the given writer
func PrintVersion(output io.Writer) {
	fmt.Fprintf(output, "Cline CLI version %s\n", Version)
}

// GetShortVersion returns just the version string
func GetShortVersion() string {
	return Version
}

// GetBuildInfo returns build information as a map
func GetBuildInfo() map[string]string {
	return map[string]string{
		"version":   Version,
		"buildDate": BuildDate,
		"gitCommit": GitCommit,
		"gitBranch": GitBranch,
		"goVersion": GoVersion,
		"os":        runtime.GOOS,
		"arch":      runtime.GOARCH,
	}
}

// IsDevelopmentBuild returns true if this is a development build
func IsDevelopmentBuild() bool {
	return GitCommit == "unknown" || BuildDate == "unknown"
}

// CheckVersionCompatibility checks if the current version meets minimum requirements
func CheckVersionCompatibility(minVersion string) bool {
	return CheckVersionCompatibilityWith(Version, minVersion)
}

// CheckVersionCompatibilityWith checks if a specific version meets minimum requirements
func CheckVersionCompatibilityWith(currentVersion, minVersion string) bool {
	// Simple version comparison - assumes semantic versioning
	current := parseVersionString(currentVersion)
	minimum := parseVersionString(minVersion)

	for i := 0; i < 3; i++ {
		if current[i] > minimum[i] {
			return true
		}
		if current[i] < minimum[i] {
			return false
		}
	}

	return true // Equal versions
}

// parseVersionString parses a version string into components
func parseVersionString(v string) [3]int {
	// Remove 'v' prefix if present
	if len(v) > 0 && v[0] == 'v' {
		v = v[1:]
	}

	var result [3]int
	parts := make([]string, 0, 3)

	// Split by dot
	current := ""
	for _, c := range v {
		if c == '.' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else if c >= '0' && c <= '9' {
			current += string(c)
		} else {
			// Stop at non-numeric (e.g., pre-release tag)
			break
		}
	}
	if current != "" {
		parts = append(parts, current)
	}

	// Parse components
	for i := 0; i < 3 && i < len(parts); i++ {
		fmt.Sscanf(parts[i], "%d", &result[i])
	}

	return result
}

// WriteVersionInfo writes version information to a file
func WriteVersionInfo(path string) error {
	info := GetVersionInfo()

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create version file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(info)
}

// ReadVersionInfo reads version information from a file
func ReadVersionInfo(path string) (*VersionInfo, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open version file: %w", err)
	}
	defer file.Close()

	var info VersionInfo
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to parse version file: %w", err)
	}

	return &info, nil
}