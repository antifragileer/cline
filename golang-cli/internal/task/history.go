// Package task provides task execution functionality for the Cline CLI.
package task

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/cline/cline/golang-cli/internal/storage"
)

// HistoryItem represents a task history entry.
type HistoryItem struct {
	TaskID    string    `json:"task_id"`
	Prompt    string    `json:"prompt"`
	Mode      string    `json:"mode"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// GetTaskHistory retrieves task history from storage.
//
// Parameters:
//   - storage: The storage context
//   - limit: Maximum number of items to return (0 for all)
//
// Returns a slice of history items, sorted by most recent first.
func GetTaskHistory(storage *storage.StorageContext, limit int) ([]HistoryItem, error) {
	if storage == nil {
		return nil, fmt.Errorf("storage is nil")
	}

	// Get task history from global state
	data, ok := storage.GlobalState.Get("taskHistory")
	if !ok {
		// If not found, return empty list
		return []HistoryItem{}, nil
	}

	// Convert to string
	dataStr, ok := data.(string)
	if !ok {
		return nil, fmt.Errorf("task history is not a string")
	}

	// Parse JSON
	var history []HistoryItem
	if err := json.Unmarshal([]byte(dataStr), &history); err != nil {
		return nil, fmt.Errorf("failed to parse task history: %w", err)
	}

	// Sort by most recent first
	sort.Slice(history, func(i, j int) bool {
		return history[i].UpdatedAt.After(history[j].UpdatedAt)
	})

	// Apply limit
	if limit > 0 && len(history) > limit {
		history = history[:limit]
	}

	return history, nil
}

// SaveTaskHistory saves task history to storage.
func SaveTaskHistory(storage *storage.StorageContext, item HistoryItem) error {
	if storage == nil {
		return fmt.Errorf("storage is nil")
	}

	// Get existing history
	history, err := GetTaskHistory(storage, 0)
	if err != nil {
		return err
	}

	// Check if item already exists
	found := false
	for i, existing := range history {
		if existing.TaskID == item.TaskID {
			// Update existing
			history[i] = item
			found = true
			break
		}
	}

	// Add new item if not found
	if !found {
		history = append([]HistoryItem{item}, history...)
	}

	// Keep only last 100 items
	if len(history) > 100 {
		history = history[:100]
	}

	// Serialize and save
	data, err := json.Marshal(history)
	if err != nil {
		return fmt.Errorf("failed to marshal task history: %w", err)
	}

	if err := storage.GlobalState.Set("taskHistory", string(data)); err != nil {
		return fmt.Errorf("failed to save task history: %w", err)
	}

	return nil
}

// FindMostRecentTask finds the most recent task.
//
// Parameters:
//   - storage: The storage context
//   - statusFilter: Optional status filter (empty for any status)
//
// Returns the most recent history item matching the filter.
func FindMostRecentTask(storage *storage.StorageContext, statusFilter string) (*HistoryItem, error) {
	history, err := GetTaskHistory(storage, 0)
	if err != nil {
		return nil, err
	}

	for _, item := range history {
		if statusFilter == "" || item.Status == statusFilter {
			return &item, nil
		}
	}

	return nil, fmt.Errorf("no matching task found")
}

// FindTaskByID finds a task by its ID.
func FindTaskByID(storage *storage.StorageContext, taskID string) (*HistoryItem, error) {
	history, err := GetTaskHistory(storage, 0)
	if err != nil {
		return nil, err
	}

	for _, item := range history {
		if item.TaskID == taskID {
			return &item, nil
		}
	}

	return nil, fmt.Errorf("task not found: %s", taskID)
}

// DeleteTaskFromHistory removes a task from history.
func DeleteTaskFromHistory(storage *storage.StorageContext, taskID string) error {
	if storage == nil {
		return fmt.Errorf("storage is nil")
	}

	history, err := GetTaskHistory(storage, 0)
	if err != nil {
		return err
	}

	// Filter out the task
	newHistory := make([]HistoryItem, 0, len(history))
	for _, item := range history {
		if item.TaskID != taskID {
			newHistory = append(newHistory, item)
		}
	}

	// Serialize and save
	data, err := json.Marshal(newHistory)
	if err != nil {
		return fmt.Errorf("failed to marshal task history: %w", err)
	}

	if err := storage.GlobalState.Set("taskHistory", string(data)); err != nil {
		return fmt.Errorf("failed to save task history: %w", err)
	}

	return nil
}

// ClearTaskHistory removes all tasks from history.
func ClearTaskHistory(storage *storage.StorageContext) error {
	if storage == nil {
		return fmt.Errorf("storage is nil")
	}

	if err := storage.GlobalState.Delete("taskHistory"); err != nil {
		return fmt.Errorf("failed to clear task history: %w", err)
	}

	return nil
}