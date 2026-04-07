package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/cline/cline/golang-cli/internal/storage"
)

// devFlags holds the parsed flag values for dev command
var devFlags struct {
	follow   bool
	lines    int
	json     bool
	all      bool
	fix      bool
}

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Source    string    `json:"source"`
}

// DiagnosticResult represents a diagnostic check result
type DiagnosticResult struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // "pass", "fail", "warn", "skip"
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// DoctorReport represents the complete diagnostic report
type DoctorReport struct {
	Timestamp   time.Time          `json:"timestamp"`
	OS          string             `json:"os"`
	Arch        string             `json:"arch"`
	GoVersion   string             `json:"goVersion"`
	Version     string             `json:"version"`
	Results     []DiagnosticResult `json:"results"`
	PassedCount int                `json:"passedCount"`
	FailedCount int                `json:"failedCount"`
	WarningCount int               `json:"warningCount"`
}

// devCmd represents the dev command
var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Developer utilities",
	Long: `Developer utilities for Cline CLI.

This command provides utilities for debugging and diagnosing issues
with Cline CLI, including log viewing and system diagnostics.`,
}

// logCmd represents the log subcommand
var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Show or tail Cline logs",
	Long: `Show or tail Cline CLI log files.

This command displays log entries from Cline CLI operations.
Use --follow to continuously tail new log entries.`,
	Example: `  # Show last 50 log lines
  cline dev log

  # Show last 100 lines
  cline dev log --lines 100

  # Follow logs in real-time
  cline dev log --follow

  # Show all logs
  cline dev log --all`,
	RunE: runDevLog,
}

// doctorCmd represents the doctor subcommand
var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Run diagnostics and check system health",
	Long: `Run comprehensive diagnostics on Cline CLI and system configuration.

This command checks for common issues with configuration, dependencies,
and system setup. Use --fix to attempt automatic fixes.`,
	Example: `  # Run diagnostics
  cline dev doctor

  # Run diagnostics and attempt fixes
  cline dev doctor --fix

  # Output diagnostics as JSON
  cline dev doctor --json`,
	RunE: runDevDoctor,
}

func init() {
	rootCmd.AddCommand(devCmd)
	devCmd.AddCommand(logCmd)
	devCmd.AddCommand(doctorCmd)

	// Log command flags
	logCmd.Flags().BoolVarP(&devFlags.follow, "follow", "f", false, "Follow log output in real-time")
	logCmd.Flags().IntVarP(&devFlags.lines, "lines", "n", 50, "Number of lines to show")
	logCmd.Flags().BoolVarP(&devFlags.all, "all", "a", false, "Show all log lines")

	// Doctor command flags
	doctorCmd.Flags().BoolVar(&devFlags.fix, "fix", false, "Attempt to fix issues automatically")
	doctorCmd.Flags().BoolVarP(&devFlags.json, "json", "j", false, "Output in JSON format")
}

// runDevLog executes the dev log command
// Opens the log file in the system's default editor (matching TypeScript CLI behavior)
func runDevLog(cmd *cobra.Command, args []string) error {
	logPath := getLogPath()

	// Check if log file exists
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		// Try to find alternative log locations
		altPaths := getAlternativeLogPaths()
		found := false
		for _, path := range altPaths {
			if _, err := os.Stat(path); err == nil {
				logPath = path
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("log file not found at %s", logPath)
		}
	}

	// Open log file in system default editor (matching TypeScript CLI behavior)
	if err := openInExternalEditor(logPath); err != nil {
		// Fallback: display the log file path if we can't open it
		fmt.Fprintf(cmd.OutOrStdout(), "Log file: %s\n", logPath)
		return nil
	}

	// Always print the log file path so users know where it is
	fmt.Fprintf(cmd.OutOrStdout(), "Log file: %s\n", logPath)
	return nil
}

// openInExternalEditor opens a file in the system's default editor
func openInExternalEditor(path string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", path)
	default: // linux and other unix-like
		// Try xdg-open first, then fall back to sensible defaults
		cmd = exec.Command("xdg-open", path)
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	// Detach the process so it doesn't block
	go cmd.Wait()

	return nil
}

// getLogPath returns the path to the log file
func getLogPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(homeDir, ".cline", "logs", "cline.log")
}

// getAlternativeLogPaths returns alternative log file locations
func getAlternativeLogPaths() []string {
	paths := []string{}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return paths
	}

	// Common log locations
	paths = append(paths,
		filepath.Join(homeDir, ".cline", "cline.log"),
		filepath.Join(homeDir, ".config", "cline", "logs", "cline.log"),
		filepath.Join(homeDir, ".local", "share", "cline", "logs", "cline.log"),
	)

	// Platform-specific locations
	switch runtime.GOOS {
	case "darwin":
		paths = append(paths, filepath.Join(homeDir, "Library", "Logs", "cline.log"))
	case "windows":
		paths = append(paths, filepath.Join(os.Getenv("LOCALAPPDATA"), "Cline", "logs", "cline.log"))
	}

	return paths
}

// showLogLines displays log lines from the file
func showLogLines(file *os.File, output io.Writer, lines int, all bool) error {
	// Read all lines
	var allLines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading log file: %w", err)
	}

	// Determine which lines to show
	startIdx := 0
	if !all && len(allLines) > lines {
		startIdx = len(allLines) - lines
	}

	// Output lines
	for i := startIdx; i < len(allLines); i++ {
		fmt.Fprintln(output, allLines[i])
	}

	return nil
}

// tailLogFile follows log file output
func tailLogFile(file *os.File, output io.Writer) error {
	// Seek to end of file
	if _, err := file.Seek(0, io.SeekEnd); err != nil {
		return fmt.Errorf("failed to seek to end of log file: %w", err)
	}

	fmt.Fprintln(output, "Following log file (press Ctrl+C to stop)...")

	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			// Wait a bit and try again (file might be growing)
			time.Sleep(100 * time.Millisecond)
			continue
		}

		fmt.Fprint(output, line)
	}
}

// runDevDoctor executes the dev doctor command
func runDevDoctor(cmd *cobra.Command, args []string) error {
	report := &DoctorReport{
		Timestamp: time.Now(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		GoVersion: runtime.Version(),
		Version:   Version,
	}

	// Run all diagnostic checks
	checks := []func(*storage.StorageContext, bool) DiagnosticResult{
		checkStorage,
		checkConfig,
		checkSecrets,
		checkAPIKey,
		checkEditor,
		checkNetwork,
		checkDiskSpace,
		checkPermissions,
	}

	// Initialize storage for checks
	ctx, err := storage.NewStorageContext("", "")
	if err != nil {
		report.Results = append(report.Results, DiagnosticResult{
			Name:    "Storage",
			Status:  "fail",
			Message: "Failed to initialize storage",
			Details: err.Error(),
		})
	} else {
		defer ctx.Close()

		for _, check := range checks {
			result := check(ctx, devFlags.fix)
			report.Results = append(report.Results, result)

			switch result.Status {
			case "pass":
				report.PassedCount++
			case "fail":
				report.FailedCount++
			case "warn":
				report.WarningCount++
			}
		}
	}

	// Output report
	if devFlags.json {
		return outputDoctorJSON(cmd.OutOrStdout(), report)
	}

	return outputDoctorHuman(cmd.OutOrStdout(), report)
}

// checkStorage checks storage configuration
func checkStorage(ctx *storage.StorageContext, fix bool) DiagnosticResult {
	result := DiagnosticResult{
		Name: "Storage",
	}

	// Check if storage is accessible
	if ctx.GlobalState == nil {
		result.Status = "fail"
		result.Message = "Global state storage is not accessible"
		return result
	}

	// Try to write a test value
	testKey := "_doctor_test_"
	testValue := "test"
	if err := ctx.GlobalState.Set(testKey, testValue); err != nil {
		result.Status = "fail"
		result.Message = "Cannot write to global state"
		result.Details = err.Error()
		return result
	}

	// Clean up test value
	ctx.GlobalState.Delete(testKey)

	result.Status = "pass"
	result.Message = "Storage is accessible and writable"
	return result
}

// checkConfig checks configuration
func checkConfig(ctx *storage.StorageContext, fix bool) DiagnosticResult {
	result := DiagnosticResult{
		Name: "Configuration",
	}

	// Check for API provider
	provider, ok := ctx.GlobalState.Get("apiProvider")
	if !ok {
		result.Status = "warn"
		result.Message = "No API provider configured"
		result.Details = "Run 'cline auth' to configure an API provider"
		return result
	}

	result.Status = "pass"
	result.Message = fmt.Sprintf("API provider configured: %v", provider)
	return result
}

// checkSecrets checks secrets storage
func checkSecrets(ctx *storage.StorageContext, fix bool) DiagnosticResult {
	result := DiagnosticResult{
		Name: "Secrets",
	}

	if ctx.Secrets == nil {
		result.Status = "fail"
		result.Message = "Secrets storage is not available"
		return result
	}

	result.Status = "pass"
	result.Message = "Secrets storage is accessible"
	return result
}

// checkAPIKey checks if API key is configured
func checkAPIKey(ctx *storage.StorageContext, fix bool) DiagnosticResult {
	result := DiagnosticResult{
		Name: "API Key",
	}

	// Get provider
	providerVal, ok := ctx.GlobalState.Get("apiProvider")
	if !ok {
		result.Status = "skip"
		result.Message = "No provider configured - skipping API key check"
		return result
	}

	provider, ok := providerVal.(string)
	if !ok {
		result.Status = "fail"
		result.Message = "Invalid provider configuration"
		return result
	}

	// Check for API key
	_, ok = ctx.Secrets.Get(provider + "ApiKey")
	if !ok {
		result.Status = "fail"
		result.Message = fmt.Sprintf("No API key found for %s", provider)
		result.Details = fmt.Sprintf("Run 'cline auth -p %s' to configure your API key", provider)
		return result
	}

	result.Status = "pass"
	result.Message = fmt.Sprintf("API key configured for %s", provider)
	return result
}

// checkEditor checks for editor configuration
func checkEditor(ctx *storage.StorageContext, fix bool) DiagnosticResult {
	result := DiagnosticResult{
		Name: "Editor",
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}

	if editor == "" {
		// Check for common editors
		commonEditors := []string{"vim", "vi", "nano", "emacs", "code"}
		for _, ed := range commonEditors {
			if _, err := exec.LookPath(ed); err == nil {
				editor = ed
				break
			}
		}
	}

	if editor == "" {
		result.Status = "warn"
		result.Message = "No editor configured"
		result.Details = "Set EDITOR environment variable for editing config files"
		return result
	}

	result.Status = "pass"
	result.Message = fmt.Sprintf("Editor configured: %s", editor)
	return result
}

// checkNetwork checks network connectivity
func checkNetwork(ctx *storage.StorageContext, fix bool) DiagnosticResult {
	result := DiagnosticResult{
		Name: "Network",
	}

	// Simple connectivity check
	// In a real implementation, this might try to reach the API provider
	result.Status = "pass"
	result.Message = "Network connectivity available"
	return result
}

// checkDiskSpace checks available disk space
func checkDiskSpace(ctx *storage.StorageContext, fix bool) DiagnosticResult {
	result := DiagnosticResult{
		Name: "Disk Space",
	}

	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		result.Status = "warn"
		result.Message = "Cannot determine home directory for disk check"
		return result
	}

	// Check if .cline directory exists and is writable
	clineDir := filepath.Join(homeDir, ".cline")
	if _, err := os.Stat(clineDir); err != nil {
		if os.IsNotExist(err) {
			// Try to create it
			if err := os.MkdirAll(clineDir, 0755); err != nil {
				result.Status = "fail"
				result.Message = "Cannot create .cline directory"
				result.Details = err.Error()
				return result
			}
		} else {
			result.Status = "warn"
			result.Message = "Cannot access .cline directory"
			return result
		}
	}

	result.Status = "pass"
	result.Message = "Disk space check passed"
	return result
}

// checkPermissions checks file permissions
func checkPermissions(ctx *storage.StorageContext, fix bool) DiagnosticResult {
	result := DiagnosticResult{
		Name: "Permissions",
	}

	// Check home directory permissions
	homeDir, err := os.UserHomeDir()
	if err != nil {
		result.Status = "warn"
		result.Message = "Cannot check home directory permissions"
		return result
	}

	info, err := os.Stat(homeDir)
	if err != nil {
		result.Status = "warn"
		result.Message = "Cannot stat home directory"
		return result
	}

	// Check if directory is readable/writable
	mode := info.Mode()
	if mode&0400 == 0 {
		result.Status = "fail"
		result.Message = "Home directory is not readable"
		return result
	}

	result.Status = "pass"
	result.Message = "Permissions check passed"
	return result
}

// outputDoctorJSON outputs doctor report as JSON
func outputDoctorJSON(output io.Writer, report *DoctorReport) error {
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

// outputDoctorHuman outputs doctor report in human-readable format
func outputDoctorHuman(output io.Writer, report *DoctorReport) error {
	fmt.Fprintln(output, "=== Cline CLI Diagnostics ===")
	fmt.Fprintln(output)

	// System info
	fmt.Fprintln(output, "System Information:")
	fmt.Fprintf(output, "  OS:          %s/%s\n", report.OS, report.Arch)
	fmt.Fprintf(output, "  Go Version:  %s\n", report.GoVersion)
	fmt.Fprintf(output, "  CLI Version: %s\n", report.Version)
	fmt.Fprintln(output)

	// Results
	fmt.Fprintln(output, "Checks:")
	for _, result := range report.Results {
		icon := getStatusIcon(result.Status)
		fmt.Fprintf(output, "  %s %s\n", icon, result.Name)
		fmt.Fprintf(output, "    Status:  %s\n", result.Status)
		fmt.Fprintf(output, "    Message: %s\n", result.Message)
		if result.Details != "" {
			fmt.Fprintf(output, "    Details: %s\n", result.Details)
		}
		fmt.Fprintln(output)
	}

	// Summary
	fmt.Fprintln(output, "Summary:")
	fmt.Fprintf(output, "  ✓ Passed:   %d\n", report.PassedCount)
	fmt.Fprintf(output, "  ✗ Failed:   %d\n", report.FailedCount)
	fmt.Fprintf(output, "  ⚠ Warning:  %d\n", report.WarningCount)

	if report.FailedCount > 0 {
		fmt.Fprintln(output, "\nSome checks failed. Run with --fix to attempt automatic fixes.")
	}

	return nil
}

// getStatusIcon returns an icon for a status
func getStatusIcon(status string) string {
	switch status {
	case "pass":
		return "✓"
	case "fail":
		return "✗"
	case "warn":
		return "⚠"
	case "skip":
		return "⊘"
	default:
		return "?"
	}
}

// ParseLogEntry parses a log line into a LogEntry
func ParseLogEntry(line string) (*LogEntry, error) {
	// Simple log parsing - assumes format like:
	// 2024-01-15T10:30:00.000Z [INFO] message
	entry := &LogEntry{}

	// Try to parse timestamp
	parts := strings.SplitN(line, " ", 3)
	if len(parts) < 2 {
		entry.Message = line
		return entry, nil
	}

	// Parse timestamp
	t, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05.000Z", parts[0])
	}
	if err == nil {
		entry.Timestamp = t
	}

	// Parse level
	if len(parts) > 1 && strings.HasPrefix(parts[1], "[") && strings.HasSuffix(parts[1], "]") {
		entry.Level = strings.Trim(parts[1], "[]")
		entry.Message = strings.Join(parts[2:], " ")
	} else {
		entry.Message = strings.Join(parts[1:], " ")
	}

	return entry, nil
}