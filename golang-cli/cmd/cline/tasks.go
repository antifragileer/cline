package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cline/cline/golang-cli/internal/storage"
	"github.com/spf13/cobra"
)

// Task represents a task entry
type Task struct {
	ID        string `json:"id"`
	Task      string `json:"task"`
	Timestamp int64  `json:"ts"`
	Metadata  struct {
		Model     string `json:"model,omitempty"`
		Mode      string `json:"mode,omitempty"`
		Completed bool   `json:"completed,omitempty"`
	} `json:"metadata,omitempty"`
}

// tasksFlags holds flags for tasks commands
var tasksFlags struct {
	json    bool
	limit   int
	page    int
	format  string
	output  string
	taskID  string
	confirm bool
}

// tasksCmd represents the tasks command
var tasksCmd = &cobra.Command{
	Use:    "tasks",
	Short:  "Manage Cline tasks",
	Hidden: true, // Hidden - redundant with 'task' command in TypeScript CLI
	Long: `Manage Cline tasks - list, show, delete, and export task history.

This command provides comprehensive task management capabilities
for viewing and manipulating your Cline task history.`,
	Example: `  # List tasks
  cline tasks list

  # Show task details
  cline tasks show <task-id>

  # Delete a task
  cline tasks delete <task-id>

  # Export tasks to JSON
  cline tasks export --output tasks.json`,
}

// tasksListCmd represents the tasks list subcommand
var tasksListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List tasks with pagination",
	Long:    `List all tasks with pagination support and filtering options.`,
	Example: `  cline tasks list
  cline tasks list --limit 20
  cline tasks list --json`,
	RunE: runTasksList,
}

// tasksShowCmd represents the tasks show subcommand
var tasksShowCmd = &cobra.Command{
	Use:     "show <task-id>",
	Aliases: []string{"view", "get"},
	Short:   "Show detailed information about a task",
	Long:    `Display detailed information about a specific task including its full history.`,
	Example: `  cline tasks show abc123
  cline tasks show abc123 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runTasksShow,
}

// tasksDeleteCmd represents the tasks delete subcommand
var tasksDeleteCmd = &cobra.Command{
	Use:     "delete <task-id>",
	Aliases: []string{"rm", "remove"},
	Short:   "Delete a task from history",
	Long:    `Remove a task and its associated data from the task history.`,
	Example: `  cline tasks delete abc123
  cline tasks delete abc123 --force`,
	Args: cobra.ExactArgs(1),
	RunE: runTasksDelete,
}

// tasksExportCmd represents the tasks export subcommand
var tasksExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export tasks to a file",
	Long: `Export task history to a JSON file for backup or analysis.

This command exports task metadata and optionally full conversation history.`,
	Example: `  cline tasks export --output tasks.json
  cline tasks export --limit 100`,
	RunE: runTasksExport,
}

func init() {
	rootCmd.AddCommand(tasksCmd)

	// List subcommand
	tasksCmd.AddCommand(tasksListCmd)
	tasksListCmd.Flags().IntVarP(&tasksFlags.limit, "limit", "n", 10, "Number of tasks to display")
	tasksListCmd.Flags().IntVarP(&tasksFlags.page, "page", "p", 1, "Page number (1-based)")
	tasksListCmd.Flags().BoolVarP(&tasksFlags.json, "json", "j", false, "Output in JSON format")

	// Show subcommand
	tasksCmd.AddCommand(tasksShowCmd)
	tasksShowCmd.Flags().BoolVarP(&tasksFlags.json, "json", "j", false, "Output in JSON format")

	// Delete subcommand
	tasksCmd.AddCommand(tasksDeleteCmd)
	tasksDeleteCmd.Flags().BoolVarP(&tasksFlags.confirm, "force", "f", false, "Delete without confirmation")

	// Export subcommand
	tasksCmd.AddCommand(tasksExportCmd)
	tasksExportCmd.Flags().StringVarP(&tasksFlags.output, "output", "o", "", "Output file path")
	tasksExportCmd.Flags().IntVarP(&tasksFlags.limit, "limit", "n", 0, "Limit number of tasks (0 = all)")
}

// runTasksList executes the tasks list command
func runTasksList(cmd *cobra.Command, args []string) error {
	// Initialize storage
	ctx, err := storage.NewStorageContext("", getWorkspaceHash())
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer ctx.Close()

	// Load task history
	tasks, err := loadTasks(ctx)
	if err != nil {
		return fmt.Errorf("failed to load tasks: %w", err)
	}

	// Sort by timestamp (newest first)
	sortTasksByTimestamp(tasks, false)

	// Calculate pagination
	totalTasks := len(tasks)
	totalPages := 1
	if tasksFlags.limit > 0 {
		totalPages = (totalTasks + tasksFlags.limit - 1) / tasksFlags.limit
	}
	if totalPages == 0 {
		totalPages = 1
	}

	// Validate page number
	if tasksFlags.page < 1 {
		tasksFlags.page = 1
	}
	if tasksFlags.page > totalPages {
		tasksFlags.page = totalPages
	}

	// Calculate slice indices
	start := (tasksFlags.page - 1) * tasksFlags.limit
	end := start + tasksFlags.limit
	if end > totalTasks {
		end = totalTasks
	}

	// Get page of tasks
	pageTasks := tasks[start:end]

	// Output as JSON if requested
	if tasksFlags.json {
		response := struct {
			Tasks      []Task `json:"tasks"`
			Page       int    `json:"page"`
			Limit      int    `json:"limit"`
			Total      int    `json:"total"`
			TotalPages int    `json:"totalPages"`
		}{
			Tasks:      pageTasks,
			Page:       tasksFlags.page,
			Limit:      tasksFlags.limit,
			Total:      totalTasks,
			TotalPages: totalPages,
		}
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(response)
	}

	// Human-readable output
	if len(pageTasks) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No tasks found.")
		return nil
	}

	// Create tabwriter for aligned columns
	tw := newTabWriter(cmd.OutOrStdout())

	// Print header
	fmt.Fprintln(tw, "ID\tTASK\tTIMESTAMP\tMODE")
	fmt.Fprintln(tw, strings.Repeat("-", 20)+"\t"+strings.Repeat("-", 40)+"\t"+strings.Repeat("-", 20)+"\t"+strings.Repeat("-", 10))

	// Print entries
	for _, task := range pageTasks {
		// Truncate task description if too long
		taskDesc := task.Task
		if len(taskDesc) > 37 {
			taskDesc = taskDesc[:34] + "..."
		}

		// Format timestamp
		timestamp := formatTaskTimestamp(task.Timestamp)

		// Get mode
		mode := task.Metadata.Mode
		if mode == "" {
			mode = "act"
		}

		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", task.ID, taskDesc, timestamp, mode)
	}

	tw.Flush()

	// Print pagination info
	fmt.Fprintf(cmd.OutOrStdout(), "\nPage %d of %d | Showing %d-%d of %d tasks\n",
		tasksFlags.page, totalPages, start+1, end, totalTasks)

	return nil
}

// runTasksShow executes the tasks show command
func runTasksShow(cmd *cobra.Command, args []string) error {
	taskID := args[0]

	// Initialize storage
	ctx, err := storage.NewStorageContext("", getWorkspaceHash())
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer ctx.Close()

	// Find the task
	task, err := findTaskByID(ctx, taskID)
	if err != nil {
		return err
	}

	if task == nil {
		return fmt.Errorf("task not found: %s", taskID)
	}

	// Load conversation history if available
	conversation, err := loadTaskConversation(ctx, taskID)
	if err != nil {
		// Non-fatal - just don't include conversation
		conversation = nil
	}

	// Output as JSON if requested
	if tasksFlags.json {
		response := struct {
			Task         *Task                 `json:"task"`
			Conversation []ConversationMessage `json:"conversation,omitempty"`
		}{
			Task:         task,
			Conversation: conversation,
		}
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(response)
	}

	// Human-readable output
	fmt.Fprintf(cmd.OutOrStdout(), "Task ID: %s\n", task.ID)
	fmt.Fprintf(cmd.OutOrStdout(), "Description: %s\n", task.Task)
	fmt.Fprintf(cmd.OutOrStdout(), "Created: %s\n", formatTaskTimestamp(task.Timestamp))

	if task.Metadata.Mode != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Mode: %s\n", task.Metadata.Mode)
	}
	if task.Metadata.Model != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Model: %s\n", task.Metadata.Model)
	}

	completed := "no"
	if task.Metadata.Completed {
		completed = "yes"
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Completed: %s\n", completed)

	if len(conversation) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "\nConversation (%d messages):\n", len(conversation))
		for i, msg := range conversation {
			if i >= 10 {
				fmt.Fprintf(cmd.OutOrStdout(), "\n... and %d more messages\n", len(conversation)-10)
				break
			}
			preview := msg.Content
			if len(preview) > 60 {
				preview = preview[:57] + "..."
			}
			fmt.Fprintf(cmd.OutOrStdout(), "  [%s] %s\n", msg.Role, preview)
		}
	}

	return nil
}

// runTasksDelete executes the tasks delete command
func runTasksDelete(cmd *cobra.Command, args []string) error {
	taskID := args[0]

	// Initialize storage
	ctx, err := storage.NewStorageContext("", getWorkspaceHash())
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer ctx.Close()

	// Find the task first
	task, err := findTaskByID(ctx, taskID)
	if err != nil {
		return err
	}

	if task == nil {
		return fmt.Errorf("task not found: %s", taskID)
	}

	// Confirm deletion unless forced
	if !tasksFlags.confirm {
		fmt.Fprintf(cmd.OutOrStdout(), "Delete task '%s': %s? [y/N]: ", taskID, task.Task)
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Fprintln(cmd.OutOrStdout(), "Deletion cancelled.")
			return nil
		}
	}

	// Delete from task history
	if err := deleteTaskFromHistory(ctx, taskID); err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	// Delete associated conversation files
	if err := deleteTaskConversationFiles(taskID); err != nil {
		// Non-fatal - log but don't fail
		fmt.Fprintf(cmd.OutOrStderr(), "Warning: failed to delete conversation files: %v\n", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ Task '%s' deleted successfully\n", taskID)
	return nil
}

// runTasksExport executes the tasks export command
func runTasksExport(cmd *cobra.Command, args []string) error {
	// Initialize storage
	ctx, err := storage.NewStorageContext("", getWorkspaceHash())
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer ctx.Close()

	// Load task history
	tasks, err := loadTasks(ctx)
	if err != nil {
		return fmt.Errorf("failed to load tasks: %w", err)
	}

	// Sort by timestamp
	sortTasksByTimestamp(tasks, false)

	// Apply limit if specified
	if tasksFlags.limit > 0 && tasksFlags.limit < len(tasks) {
		tasks = tasks[:tasksFlags.limit]
	}

	// Determine output destination
	var output io.Writer = cmd.OutOrStdout()
	var file *os.File

	if tasksFlags.output != "" {
		file, err = os.Create(tasksFlags.output)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer file.Close()
		output = file
	}

	// Create export data
	export := struct {
		ExportedAt string `json:"exported_at"`
		Count      int    `json:"count"`
		Tasks      []Task `json:"tasks"`
	}{
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Count:      len(tasks),
		Tasks:      tasks,
	}

	// Write JSON
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(export); err != nil {
		return fmt.Errorf("failed to encode export: %w", err)
	}

	if tasksFlags.output != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "✓ Exported %d tasks to %s\n", len(tasks), tasksFlags.output)
	}

	return nil
}

// ConversationMessage represents a message in a conversation
type ConversationMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// loadTasks loads all tasks from storage
func loadTasks(ctx *storage.StorageContext) ([]Task, error) {
	var tasks []Task

	// Try to get from global state
	if val, ok := ctx.GlobalState.Get("taskHistory"); ok {
		jsonData, err := json.Marshal(val)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(jsonData, &tasks); err != nil {
			// Try as slice of interface{}
			var rawTasks []interface{}
			if err := json.Unmarshal(jsonData, &rawTasks); err != nil {
				return nil, err
			}

			// Convert each task
			for _, raw := range rawTasks {
				taskJSON, _ := json.Marshal(raw)
				var task Task
				if err := json.Unmarshal(taskJSON, &task); err == nil {
					tasks = append(tasks, task)
				}
			}
		}
	}

	return tasks, nil
}

// findTaskByID finds a task by its ID
func findTaskByID(ctx *storage.StorageContext, taskID string) (*Task, error) {
	tasks, err := loadTasks(ctx)
	if err != nil {
		return nil, err
	}

	for _, task := range tasks {
		if task.ID == taskID {
			return &task, nil
		}
	}

	return nil, nil
}

// deleteTaskFromHistory removes a task from the task history
func deleteTaskFromHistory(ctx *storage.StorageContext, taskID string) error {
	tasks, err := loadTasks(ctx)
	if err != nil {
		return err
	}

	// Filter out the task to delete
	var filtered []Task
	for _, task := range tasks {
		if task.ID != taskID {
			filtered = append(filtered, task)
		}
	}

	// Save back to storage
	return ctx.GlobalState.Set("taskHistory", filtered)
}

// deleteTaskConversationFiles deletes conversation files for a task
func deleteTaskConversationFiles(taskID string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// Try to delete various conversation file patterns
	patterns := []string{
		filepath.Join(homeDir, ".cline", "data", "tasks", taskID, "*.json"),
		filepath.Join(homeDir, ".cline", "tasks", taskID, "*.json"),
	}

	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		for _, match := range matches {
			os.Remove(match)
		}
	}

	// Remove task directory if empty
	taskDirs := []string{
		filepath.Join(homeDir, ".cline", "data", "tasks", taskID),
		filepath.Join(homeDir, ".cline", "tasks", taskID),
	}

	for _, dir := range taskDirs {
		// Try to remove (will fail if not empty, which is fine)
		os.Remove(dir)
	}

	return nil
}

// loadTaskConversation loads conversation history for a task
func loadTaskConversation(ctx *storage.StorageContext, taskID string) ([]ConversationMessage, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	// Look for conversation files
	possiblePaths := []string{
		filepath.Join(homeDir, ".cline", "data", "tasks", taskID, "conversation.json"),
		filepath.Join(homeDir, ".cline", "tasks", taskID, "conversation.json"),
		filepath.Join(homeDir, ".cline", "data", "tasks", taskID, "messages.json"),
	}

	for _, path := range possiblePaths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var messages []ConversationMessage
		if err := json.Unmarshal(data, &messages); err == nil {
			return messages, nil
		}
	}

	return nil, fmt.Errorf("conversation not found")
}

// sortTasksByTimestamp sorts tasks by timestamp
func sortTasksByTimestamp(tasks []Task, ascending bool) {
	for i := 0; i < len(tasks)-1; i++ {
		for j := i + 1; j < len(tasks); j++ {
			shouldSwap := false
			if ascending {
				shouldSwap = tasks[i].Timestamp > tasks[j].Timestamp
			} else {
				shouldSwap = tasks[i].Timestamp < tasks[j].Timestamp
			}
			if shouldSwap {
				tasks[i], tasks[j] = tasks[j], tasks[i]
			}
		}
	}
}

// formatTaskTimestamp formats a timestamp for display
func formatTaskTimestamp(ts int64) string {
	if ts == 0 {
		return "unknown"
	}

	// Handle both seconds and milliseconds
	if ts > 1e12 {
		// Milliseconds
		ts = ts / 1000
	}

	t := time.Unix(ts, 0).UTC()
	return t.Format("2006-01-02 15:04")
}

// newTabWriter creates a new tabwriter for aligned output
func newTabWriter(w io.Writer) *tabWriter {
	return &tabWriter{writer: w}
}

// tabWriter wraps io.Writer for tabular output
type tabWriter struct {
	writer io.Writer
}

func (tw *tabWriter) Write(p []byte) (n int, err error) {
	return tw.writer.Write(p)
}

func (tw *tabWriter) Flush() error {
	// Simple implementation - in production, use text/tabwriter
	return nil
}
