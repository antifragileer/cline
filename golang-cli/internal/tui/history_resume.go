// Package tui provides terminal UI components for the Cline CLI.
package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// HistoryResume handles task resumption from history
type HistoryResume struct {
	historyDir   string
	grpcClient   *MockGRPCClient
	tasks        []TaskInfo
	selectedIdx  int
	loading      bool
	error        error
	styles       HistoryResumeStyles
	width        int
	height       int
}

// TaskInfo represents a task from history that can be resumed
type TaskInfo struct {
	ID          string
	Task        string
	Timestamp   time.Time
	Status      TaskStatus
	MessageCount int
	Dir         string
	Provider    string
	Model       string
	LastModified time.Time
}

// TaskStatus represents the status of a task
type TaskStatus string

const (
	TaskStatusActive      TaskStatus = "active"
	TaskStatusCompleted   TaskStatus = "completed"
	TaskStatusFailed      TaskStatus = "failed"
	TaskStatusPaused      TaskStatus = "paused"
	TaskStatusInterrupted TaskStatus = "interrupted"
)

// HistoryResumeStyles holds styles for history resume
type HistoryResumeStyles struct {
	containerStyle lipgloss.Style
	titleStyle     lipgloss.Style
	taskStyle      lipgloss.Style
	selectedStyle  lipgloss.Style
	statusActive   lipgloss.Style
	statusCompleted lipgloss.Style
	statusFailed   lipgloss.Style
	statusPaused   lipgloss.Style
	timestampStyle lipgloss.Style
	infoStyle      lipgloss.Style
	helpStyle      lipgloss.Style
	loadingStyle   lipgloss.Style
	errorStyle     lipgloss.Style
}

// DefaultHistoryResumeStyles returns default styles
func DefaultHistoryResumeStyles() HistoryResumeStyles {
	return HistoryResumeStyles{
		containerStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(1),

		titleStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(PrimaryBlue)).
			Bold(true).
			MarginBottom(1),

		taskStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(White)).
			PaddingLeft(2),

		selectedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(SelectionBlue)).
			Bold(true).
			Background(lipgloss.Color(DarkBackground)).
			PaddingLeft(2),

		statusActive: lipgloss.NewStyle().
			Foreground(lipgloss.Color(SuccessGreen)).
			Bold(true),

		statusCompleted: lipgloss.NewStyle().
			Foreground(lipgloss.Color(PrimaryBlue)),

		statusFailed: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ErrorRed)),

		statusPaused: lipgloss.NewStyle().
			Foreground(lipgloss.Color(PlanYellow)),

		timestampStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(Gray)).
			Italic(true),

		infoStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(Gray)),

		helpStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(Gray)).
			MarginTop(1),

		loadingStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(SelectionBlue)),

		errorStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ErrorRed)),
	}
}

// NewHistoryResume creates a new history resume handler
func NewHistoryResume(historyDir string) *HistoryResume {
	return &HistoryResume{
		historyDir: historyDir,
		tasks:      make([]TaskInfo, 0),
		styles:     DefaultHistoryResumeStyles(),
		loading:    true,
	}
}

// Init initializes the history resume
func (hr *HistoryResume) Init() tea.Cmd {
	return hr.loadTasks()
}

// loadTasks loads tasks from the history directory
func (hr *HistoryResume) loadTasks() tea.Cmd {
	return func() tea.Msg {
		tasks, err := hr.loadTasksFromDisk()
		if err != nil {
			return historyResumeErrorMsg{err: err}
		}
		return historyResumeLoadedMsg{tasks: tasks}
	}
}

// loadTasksFromDisk reads task history from disk
func (hr *HistoryResume) loadTasksFromDisk() ([]TaskInfo, error) {
	// Check if history directory exists
	if _, err := os.Stat(hr.historyDir); os.IsNotExist(err) {
		// Return empty list if directory doesn't exist
		return []TaskInfo{}, nil
	}

	entries, err := os.ReadDir(hr.historyDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read history directory: %w", err)
	}

	var tasks []TaskInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		taskInfo, err := hr.loadTaskInfo(entry.Name())
		if err != nil {
			// Skip invalid tasks but continue loading others
			continue
		}
		tasks = append(tasks, taskInfo)
	}

	// Sort by timestamp (newest first)
	hr.sortTasksByTimestamp(tasks)
	return tasks, nil
}

// loadTaskInfo loads info for a specific task
func (hr *HistoryResume) loadTaskInfo(taskID string) (TaskInfo, error) {
	taskDir := filepath.Join(hr.historyDir, taskID)

	// Read task metadata
	metaPath := filepath.Join(taskDir, "task_metadata.json")
	metaData, err := os.ReadFile(metaPath)
	if err != nil {
		// Try alternative paths for legacy tasks
		metaPath = filepath.Join(taskDir, "metadata.json")
		metaData, err = os.ReadFile(metaPath)
		if err != nil {
			return TaskInfo{}, fmt.Errorf("task metadata not found: %w", err)
		}
	}

	var taskInfo TaskInfo
	if err := json.Unmarshal(metaData, &taskInfo); err != nil {
		return TaskInfo{}, fmt.Errorf("failed to parse task metadata: %w", err)
	}

	taskInfo.ID = taskID
	taskInfo.Dir = taskDir

	// Count messages
	taskInfo.MessageCount = hr.countMessages(taskDir)

	// Get last modified time
	info, err := os.Stat(metaPath)
	if err == nil {
		taskInfo.LastModified = info.ModTime()
	}

	return taskInfo, nil
}

// countMessages counts the number of messages in a task
func (hr *HistoryResume) countMessages(taskDir string) int {
	messagesPath := filepath.Join(taskDir, "messages.jsonl")
	if _, err := os.Stat(messagesPath); err != nil {
		return 0
	}

	data, err := os.ReadFile(messagesPath)
	if err != nil {
		return 0
	}

	// Count newline characters as approximate message count
	return strings.Count(string(data), "\n")
}

// sortTasksByTimestamp sorts tasks by timestamp (newest first)
func (hr *HistoryResume) sortTasksByTimestamp(tasks []TaskInfo) {
	// Simple bubble sort for now
	for i := 0; i < len(tasks); i++ {
		for j := i + 1; j < len(tasks); j++ {
			if tasks[j].Timestamp.After(tasks[i].Timestamp) {
				tasks[i], tasks[j] = tasks[j], tasks[i]
			}
		}
	}
}

// Update handles messages
func (hr *HistoryResume) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case historyResumeLoadedMsg:
		hr.tasks = msg.tasks
		hr.loading = false

	case historyResumeErrorMsg:
		hr.error = msg.err
		hr.loading = false

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if hr.selectedIdx > 0 {
				hr.selectedIdx--
			}
		case "down", "j":
			if hr.selectedIdx < len(hr.tasks)-1 {
				hr.selectedIdx++
			}
		case "enter":
			return hr, hr.resumeSelectedTask()
		case "r":
			return hr, hr.resumeSelectedTask()
		case "d":
			if hr.selectedIdx < len(hr.tasks) {
				return hr, hr.deleteTask(hr.tasks[hr.selectedIdx].ID)
			}
		case "q", "esc":
			return hr, tea.Quit
		}
	}

	return hr, nil
}

// View renders the history resume view
func (hr *HistoryResume) View() string {
	if hr.loading {
		return hr.styles.containerStyle.Render(
			hr.styles.loadingStyle.Render("Loading task history..."),
		)
	}

	if hr.error != nil {
		return hr.styles.containerStyle.Render(
			hr.styles.titleStyle.Render("❌ Error") + "\n" +
				hr.styles.errorStyle.Render(hr.error.Error()) + "\n\n" +
				hr.styles.helpStyle.Render("Press q to quit"),
		)
	}

	if len(hr.tasks) == 0 {
		return hr.styles.containerStyle.Render(
			hr.styles.titleStyle.Render("Task History") + "\n\n" +
				lipgloss.NewStyle().Foreground(lipgloss.Color(Gray)).Render("No tasks found") + "\n\n" +
				hr.styles.helpStyle.Render("Press q to quit"),
		)
	}

	var content strings.Builder

	// Title
	content.WriteString(hr.styles.titleStyle.Render("📜 Task History"))
	content.WriteString("\n\n")

	// Tasks list
	maxVisible := hr.getMaxVisibleItems()
	startIdx := 0
	if hr.selectedIdx >= maxVisible {
		startIdx = hr.selectedIdx - maxVisible + 1
	}

	endIdx := startIdx + maxVisible
	if endIdx > len(hr.tasks) {
		endIdx = len(hr.tasks)
	}

	for i := startIdx; i < endIdx; i++ {
		task := hr.tasks[i]
		line := hr.renderTask(task, i == hr.selectedIdx)
		content.WriteString(line)
		content.WriteString("\n")
	}

	// Scroll indicator
	if len(hr.tasks) > maxVisible {
		content.WriteString("\n")
		scrollInfo := fmt.Sprintf("Showing %d-%d of %d",
			startIdx+1, endIdx, len(hr.tasks))
		content.WriteString(hr.styles.infoStyle.Render(scrollInfo))
		content.WriteString("\n")
	}

	// Help
	content.WriteString("\n")
	content.WriteString(hr.styles.helpStyle.Render(
		"↑↓/j/k: navigate • Enter/r: resume • d: delete • q: quit"))

	return hr.styles.containerStyle.Render(content.String())
}

// renderTask renders a single task
func (hr *HistoryResume) renderTask(task TaskInfo, selected bool) string {
	var parts []string

	// Status indicator
	status := hr.getStatusIndicator(task.Status)
	parts = append(parts, status)

	// Task summary (truncated)
	summary := hr.truncateTaskSummary(task.Task, 40)
	parts = append(parts, summary)

	// Message count
	if task.MessageCount > 0 {
		parts = append(parts, hr.styles.infoStyle.Render(
			fmt.Sprintf("(%d msgs)", task.MessageCount)))
	}

	// Timestamp
	timeStr := hr.formatTimestamp(task.Timestamp)
	parts = append(parts, hr.styles.timestampStyle.Render(timeStr))

	line := strings.Join(parts, " ")

	if selected {
		return hr.styles.selectedStyle.Render(line)
	}
	return hr.styles.taskStyle.Render(line)
}

// getStatusIndicator returns the styled status indicator
func (hr *HistoryResume) getStatusIndicator(status TaskStatus) string {
	switch status {
	case TaskStatusActive:
		return hr.styles.statusActive.Render("●")
	case TaskStatusCompleted:
		return hr.styles.statusCompleted.Render("✓")
	case TaskStatusFailed:
		return hr.styles.statusFailed.Render("✗")
	case TaskStatusPaused:
		return hr.styles.statusPaused.Render("⏸")
	case TaskStatusInterrupted:
		return hr.styles.statusPaused.Render("⏹")
	default:
		return "?"
	}
}

// truncateTaskSummary truncates a task summary
func (hr *HistoryResume) truncateTaskSummary(task string, maxLen int) string {
	if len(task) <= maxLen {
		return task
	}
	return task[:maxLen-3] + "..."
}

// formatTimestamp formats a timestamp
func (hr *HistoryResume) formatTimestamp(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		return fmt.Sprintf("%dm ago", int(diff.Minutes()))
	case diff < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(diff.Hours()))
	case diff < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(diff.Hours()/24))
	default:
		return t.Format("Jan 2")
	}
}

// getMaxVisibleItems returns max visible items
func (hr *HistoryResume) getMaxVisibleItems() int {
	available := hr.height - 8
	if available < 3 {
		return 3
	}
	return available
}

// resumeSelectedTask resumes the currently selected task
func (hr *HistoryResume) resumeSelectedTask() tea.Cmd {
	if hr.selectedIdx >= len(hr.tasks) {
		return nil
	}

	task := hr.tasks[hr.selectedIdx]
	return func() tea.Msg {
		return historyResumeTaskMsg{
			taskID: task.ID,
			task:   task.Task,
		}
	}
}

// deleteTask deletes a task from history
func (hr *HistoryResume) deleteTask(taskID string) tea.Cmd {
	return func() tea.Msg {
		taskDir := filepath.Join(hr.historyDir, taskID)
		if err := os.RemoveAll(taskDir); err != nil {
			return historyResumeErrorMsg{err: fmt.Errorf("failed to delete task: %w", err)}
		}
		// Reload tasks
		return hr.loadTasks()()
	}
}

// GetSelectedTask returns the currently selected task
func (hr *HistoryResume) GetSelectedTask() *TaskInfo {
	if hr.selectedIdx >= len(hr.tasks) {
		return nil
	}
	task := hr.tasks[hr.selectedIdx]
	return &task
}

// SetDimensions sets the display dimensions
func (hr *HistoryResume) SetDimensions(width, height int) {
	hr.width = width
	hr.height = height
}

// SetGRPCClient sets the gRPC client for task operations
func (hr *HistoryResume) SetGRPCClient(client *MockGRPCClient) {
	hr.grpcClient = client
}

// Message types
type historyResumeLoadedMsg struct {
	tasks []TaskInfo
}

type historyResumeErrorMsg struct {
	err error
}

type historyResumeTaskMsg struct {
	taskID string
	task   string
}

// ResumeTask resumes a task by ID
func (hr *HistoryResume) ResumeTask(ctx context.Context, taskID string) error {
	if hr.grpcClient == nil {
		return fmt.Errorf("gRPC client not initialized")
	}

	// Load task context
	taskDir := filepath.Join(hr.historyDir, taskID)
	messagesPath := filepath.Join(taskDir, "messages.jsonl")

	// Check if task exists
	if _, err := os.Stat(messagesPath); err != nil {
		return fmt.Errorf("task not found: %s", taskID)
	}

	// Resume task via gRPC
	// This would normally call the actual gRPC resume method
	fmt.Printf("Resuming task %s...\n", taskID)

	return nil
}

// GetTaskHistoryDir returns the default task history directory
func GetTaskHistoryDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ".cline/tasks"
	}
	return filepath.Join(homeDir, ".cline", "tasks")
}

// MockGRPCClient is a mock client for gRPC operations
type MockGRPCClient struct {
	// Add real gRPC client fields here
}

// NewMockGRPCClient creates a new mock gRPC client
func NewMockGRPCClient() *MockGRPCClient {
	return &MockGRPCClient{}
}