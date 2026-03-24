package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/cline/cline/golang-cli/internal/storage"
	"github.com/spf13/cobra"
)

// TaskHistoryEntry represents a single task history record
type TaskHistoryEntry struct {
	ID        string   `json:"id"`
	Task      string   `json:"task"`
	Timestamp int64    `json:"ts"`
	Metadata  Metadata `json:"metadata,omitempty"`
}

// Metadata contains additional task information
type Metadata struct {
	Model     string `json:"model,omitempty"`
	Mode      string `json:"mode,omitempty"`
	Completed bool   `json:"completed,omitempty"`
}

// HistoryConfig holds configuration options for the history command
type HistoryConfig struct {
	// Limit is the maximum number of entries to display per page
	Limit int

	// Page is the page number to display (1-based)
	Page int

	// ConfigPath is the path to a custom configuration file
	ConfigPath string

	// JSON enables JSON output format
	JSON bool

	// Output is the writer for output (used for testing)
	Output io.Writer
}

// historyFlags holds the parsed flag values for the history command
var historyFlags struct {
	limit  int
	page   int
	config string
	json   bool
}

// historyCmd represents the history command
var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Display task history with pagination",
	Long: `Display the history of tasks executed with Cline.

This command shows a paginated list of past tasks, including their
IDs, descriptions, and timestamps. You can navigate through pages
and control the number of entries displayed.`,
	Example: `  # Show last 10 tasks (default)
  cline history

  # Show last 20 tasks
  cline history -n 20

  # Show page 2 with 15 entries per page
  cline history -n 15 -p 2

  # Output as JSON
  cline history --json

  # Use custom config
  cline history --config /path/to/config.yaml`,
	RunE: runHistory,
}

func init() {
	rootCmd.AddCommand(historyCmd)

	// Add flags to history command
	historyCmd.Flags().IntVarP(&historyFlags.limit, "limit", "n", 10, "Number of entries to display per page")
	historyCmd.Flags().IntVarP(&historyFlags.page, "page", "p", 1, "Page number to display (1-based)")
	historyCmd.Flags().StringVar(&historyFlags.config, "config", "", "Path to configuration file")
	historyCmd.Flags().BoolVar(&historyFlags.json, "json", false, "Output in JSON format")
}

// runHistory executes the history command
func runHistory(cmd *cobra.Command, args []string) error {
	// Build configuration from flags
	config := HistoryConfig{
		Limit:      historyFlags.limit,
		Page:       historyFlags.page,
		ConfigPath: historyFlags.config,
		JSON:       historyFlags.json,
		Output:     cmd.OutOrStdout(),
	}

	// Validate configuration
	if err := validateHistoryConfig(config); err != nil {
		return err
	}

	// Load and display history
	return loadAndDisplayHistory(config)
}

// validateHistoryConfig validates the history configuration
func validateHistoryConfig(config HistoryConfig) error {
	var errs []string

	// Validate limit is positive
	if config.Limit <= 0 {
		errs = append(errs, "limit must be greater than 0")
	}

	// Validate page is positive
	if config.Page <= 0 {
		errs = append(errs, "page must be greater than 0")
	}

	// Validate config file exists if specified
	if config.ConfigPath != "" {
		if _, err := os.Stat(config.ConfigPath); os.IsNotExist(err) {
			errs = append(errs, fmt.Sprintf("config file not found: %s", config.ConfigPath))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}

	return nil
}

// loadAndDisplayHistory loads task history and displays it according to config
func loadAndDisplayHistory(config HistoryConfig) error {
	// Get the history file path
	historyPath := getTaskHistoryPath()

	// Check if file exists
	if _, err := os.Stat(historyPath); os.IsNotExist(err) {
		if config.JSON {
			return outputJSON(config.Output, []TaskHistoryEntry{}, config.Page, config.Limit, 0)
		}
		fmt.Fprintln(config.Output, "No task history found.")
		return nil
	}

	// Load entries from storage
	entries, err := loadTaskHistory(historyPath)
	if err != nil {
		return fmt.Errorf("failed to load task history: %w", err)
	}

	// Sort entries by timestamp (newest first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp > entries[j].Timestamp
	})

	// Calculate pagination
	totalEntries := len(entries)
	totalPages := (totalEntries + config.Limit - 1) / config.Limit
	if totalPages == 0 {
		totalPages = 1
	}

	// Validate page number
	if config.Page > totalPages {
		config.Page = totalPages
	}

	// Calculate offset
	offset := (config.Page - 1) * config.Limit
	end := offset + config.Limit
	if end > totalEntries {
		end = totalEntries
	}

	// Get page entries
	pageEntries := entries[offset:end]

	// Output results
	if config.JSON {
		return outputJSON(config.Output, pageEntries, config.Page, config.Limit, totalEntries)
	}

	return outputTable(config.Output, pageEntries, config.Page, config.Limit, totalEntries, totalPages)
}

// getTaskHistoryPathFunc is the function used to get the task history path
// This variable allows tests to override the path
var getTaskHistoryPathFunc = getTaskHistoryPathDefault

// getTaskHistoryPathDefault returns the default path to the task history file
func getTaskHistoryPathDefault() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fall back to current directory if home cannot be determined
		return "taskHistory.json"
	}
	return filepath.Join(homeDir, ".cline", "data", "taskHistory.json")
}

// getTaskHistoryPath returns the path to the task history file
func getTaskHistoryPath() string {
	return getTaskHistoryPathFunc()
}

// loadTaskHistory loads task entries from the history file
func loadTaskHistory(path string) ([]TaskHistoryEntry, error) {
	// Create storage instance
	fileStorage, err := storage.NewClineFileStorage(path, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}
	defer fileStorage.Close()

	// Try to get entries as a slice
	val, ok := fileStorage.Get("entries")
	if !ok {
		// No entries found, return empty slice
		return []TaskHistoryEntry{}, nil
	}

	// Handle different possible structures
	switch v := val.(type) {
	case []interface{}:
		// Convert []interface{} to []TaskHistoryEntry
		return convertToEntries(v)
	case []TaskHistoryEntry:
		// Already the correct type
		return v, nil
	default:
		// Try to marshal/unmarshal to handle map[string]interface{} from JSON
		data, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal history data: %w", err)
		}

		var entries []TaskHistoryEntry
		if err := json.Unmarshal(data, &entries); err != nil {
			return nil, fmt.Errorf("failed to unmarshal history entries: %w", err)
		}
		return entries, nil
	}
}

// convertToEntries converts []interface{} to []TaskHistoryEntry
func convertToEntries(data []interface{}) ([]TaskHistoryEntry, error) {
	entries := make([]TaskHistoryEntry, 0, len(data))

	for i, item := range data {
		entry, err := convertToEntry(item)
		if err != nil {
			return nil, fmt.Errorf("invalid entry at index %d: %w", i, err)
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// convertToEntry converts a single interface{} to TaskHistoryEntry
func convertToEntry(data interface{}) (TaskHistoryEntry, error) {
	// Marshal and unmarshal to handle conversion from map[string]interface{}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return TaskHistoryEntry{}, fmt.Errorf("failed to marshal entry: %w", err)
	}

	var entry TaskHistoryEntry
	if err := json.Unmarshal(jsonData, &entry); err != nil {
		return TaskHistoryEntry{}, fmt.Errorf("failed to unmarshal entry: %w", err)
	}

	return entry, nil
}

// outputJSON outputs history entries as JSON
func outputJSON(w io.Writer, entries []TaskHistoryEntry, page, limit, total int) error {
	response := struct {
		Entries []TaskHistoryEntry `json:"entries"`
		Page    int                `json:"page"`
		Limit   int                `json:"limit"`
		Total   int                `json:"total"`
		Pages   int                `json:"pages"`
	}{
		Entries: entries,
		Page:    page,
		Limit:   limit,
		Total:   total,
		Pages:   (total + limit - 1) / limit,
	}

	if response.Pages == 0 {
		response.Pages = 1
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(response)
}

// outputTable outputs history entries as a formatted table
func outputTable(w io.Writer, entries []TaskHistoryEntry, page, limit, total, pages int) error {
	if len(entries) == 0 {
		fmt.Fprintln(w, "No task history found.")
		return nil
	}

	// Create tabwriter for aligned columns
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

	// Print header
	fmt.Fprintln(tw, "ID\tTASK\tTIMESTAMP")
	fmt.Fprintln(tw, strings.Repeat("-", 20)+"\t"+strings.Repeat("-", 40)+"\t"+strings.Repeat("-", 20))

	// Print entries
	for _, entry := range entries {
		// Truncate task description if too long
		task := entry.Task
		if len(task) > 37 {
			task = task[:34] + "..."
		}

		// Format timestamp
		timestamp := formatTimestamp(entry.Timestamp)

		fmt.Fprintf(tw, "%s\t%s\t%s\n", entry.ID, task, timestamp)
	}

	tw.Flush()

	// Print pagination info
	fmt.Fprintf(w, "\nPage %d of %d | Showing %d-%d of %d entries\n",
		page, pages, (page-1)*limit+1, (page-1)*limit+len(entries), total)

	return nil
}

// formatTimestamp formats a Unix timestamp to a human-readable string
func formatTimestamp(ts int64) string {
	if ts == 0 {
		return "unknown"
	}

	t := time.Unix(ts/1000, 0).UTC() // Convert from milliseconds to seconds, use UTC
	return t.Format("2006-01-02 15:04")
}

// HistoryLoader defines the interface for loading task history
type HistoryLoader interface {
	Load(path string) ([]TaskHistoryEntry, error)
}

// DefaultHistoryLoader is the default implementation of HistoryLoader
type DefaultHistoryLoader struct{}

// Load implements HistoryLoader
func (d *DefaultHistoryLoader) Load(path string) ([]TaskHistoryEntry, error) {
	return loadTaskHistory(path)
}

// FormatEntries formats history entries for display
func FormatEntries(entries []TaskHistoryEntry, format string) (string, error) {
	switch format {
	case "json":
		var buf strings.Builder
		err := outputJSON(&buf, entries, 1, len(entries), len(entries))
		return buf.String(), err
	case "table":
		var buf strings.Builder
		err := outputTable(&buf, entries, 1, len(entries), len(entries), 1)
		return buf.String(), err
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}
