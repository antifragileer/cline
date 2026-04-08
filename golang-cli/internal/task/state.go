// Package task provides task state management functionality for the Cline CLI.
// This file handles task state persistence, restoration, and interruption handling.
package task

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/cline/cline/golang-cli/internal/storage"
)

// TaskState represents the current state of a task execution
type TaskState struct {
	// TaskID is the unique identifier for the task
	TaskID string `json:"task_id"`

	// Status is the current task status
	Status TaskStatus `json:"status"`

	// StartedAt is when the task was started
	StartedAt int64 `json:"started_at"`

	// LastActivity is the timestamp of the last activity
	LastActivity int64 `json:"last_activity"`

	// CurrentPrompt is the current task prompt
	CurrentPrompt string `json:"current_prompt"`

	// Mode is the execution mode (act/plan)
	Mode string `json:"mode"`

	// CheckpointHash is the current git checkpoint hash
	CheckpointHash string `json:"checkpoint_hash,omitempty"`

	// Metadata contains additional task metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// TaskStatus represents the status of a task
type TaskStatus string

const (
	// TaskStatusPending indicates the task is pending
	TaskStatusPending TaskStatus = "pending"
	// TaskStatusRunning indicates the task is currently running
	TaskStatusRunning TaskStatus = "running"
	// TaskStatusPaused indicates the task was paused
	TaskStatusPaused TaskStatus = "paused"
	// TaskStatusCompleted indicates the task completed successfully
	TaskStatusCompleted TaskStatus = "completed"
	// TaskStatusFailed indicates the task failed
	TaskStatusFailed TaskStatus = "failed"
	// TaskStatusInterrupted indicates the task was interrupted
	TaskStatusInterrupted TaskStatus = "interrupted"
)

// StateManager handles task state persistence and restoration
type StateManager struct {
	// storage is the file storage backend
	storage storage.FileStorage

	// state holds the current task state
	state *TaskState

	// mu protects concurrent access to state
	mu sync.RWMutex

	// taskID identifies which task this state belongs to
	taskID string

	// dataDir is the directory for state data files
	dataDir string

	// autoSave enables automatic saving on state changes
	autoSave bool

	// dirty tracks if there are unsaved changes
	dirty bool
}

// StateStorageKey is the key used to store state data in file storage
const StateStorageKey = "task_state"

// NewStateManager creates a new state manager instance
func NewStateManager(taskID string, dataDir string) (*StateManager, error) {
	if taskID == "" {
		return nil, fmt.Errorf("taskID is required")
	}

	if dataDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		dataDir = filepath.Join(homeDir, ".cline", "data", "tasks", taskID)
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	storagePath := filepath.Join(dataDir, "state.json")
	fileStorage, err := storage.NewClineFileStorage(storagePath, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create state storage: %w", err)
	}

	sm := &StateManager{
		storage:  fileStorage,
		state:    nil,
		taskID:   taskID,
		dataDir:  dataDir,
		autoSave: true,
	}

	// Load existing state
	if err := sm.load(); err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	return sm, nil
}

// NewStateManagerWithStorage creates a state manager with existing storage
func NewStateManagerWithStorage(taskID string, fileStorage storage.FileStorage) (*StateManager, error) {
	if taskID == "" {
		return nil, fmt.Errorf("taskID is required")
	}

	sm := &StateManager{
		storage:  fileStorage,
		state:    nil,
		taskID:   taskID,
		dataDir:  "",
		autoSave: true,
	}

	// Load existing state
	if err := sm.load(); err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	return sm, nil
}

// Close closes the state manager and releases resources
func (sm *StateManager) Close() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Flush any pending changes
	if sm.dirty && sm.autoSave {
		if err := sm.save(); err != nil {
			return fmt.Errorf("failed to save state on close: %w", err)
		}
	}

	return sm.storage.Close()
}

// InitializeState initializes a new task state
func (sm *StateManager) InitializeState(prompt string, mode string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now().UnixMilli()

	sm.state = &TaskState{
		TaskID:        sm.taskID,
		Status:        TaskStatusRunning,
		StartedAt:     now,
		LastActivity:  now,
		CurrentPrompt: prompt,
		Mode:          mode,
		Metadata:      make(map[string]interface{}),
	}

	sm.dirty = true

	if sm.autoSave {
		return sm.save()
	}

	return nil
}

// GetState returns the current task state
func (sm *StateManager) GetState() (*TaskState, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if sm.state == nil {
		return nil, fmt.Errorf("no state initialized for task %s", sm.taskID)
	}

	// Return a copy to prevent external modification
	return copyState(sm.state), nil
}

// UpdateStatus updates the task status
func (sm *StateManager) UpdateStatus(status TaskStatus) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.state == nil {
		return fmt.Errorf("no state initialized")
	}

	sm.state.Status = status
	sm.state.LastActivity = time.Now().UnixMilli()
	sm.dirty = true

	if sm.autoSave {
		return sm.save()
	}

	return nil
}

// UpdateCheckpoint updates the checkpoint hash
func (sm *StateManager) UpdateCheckpoint(hash string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.state == nil {
		return fmt.Errorf("no state initialized")
	}

	sm.state.CheckpointHash = hash
	sm.state.LastActivity = time.Now().UnixMilli()
	sm.dirty = true

	if sm.autoSave {
		return sm.save()
	}

	return nil
}

// UpdatePrompt updates the current prompt
func (sm *StateManager) UpdatePrompt(prompt string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.state == nil {
		return fmt.Errorf("no state initialized")
	}

	sm.state.CurrentPrompt = prompt
	sm.state.LastActivity = time.Now().UnixMilli()
	sm.dirty = true

	if sm.autoSave {
		return sm.save()
	}

	return nil
}

// UpdateMetadata updates a metadata value
func (sm *StateManager) UpdateMetadata(key string, value interface{}) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.state == nil {
		return fmt.Errorf("no state initialized")
	}

	if sm.state.Metadata == nil {
		sm.state.Metadata = make(map[string]interface{})
	}

	sm.state.Metadata[key] = value
	sm.state.LastActivity = time.Now().UnixMilli()
	sm.dirty = true

	if sm.autoSave {
		return sm.save()
	}

	return nil
}

// MarkRunning marks the task as running
func (sm *StateManager) MarkRunning() error {
	return sm.UpdateStatus(TaskStatusRunning)
}

// MarkCompleted marks the task as completed
func (sm *StateManager) MarkCompleted() error {
	return sm.UpdateStatus(TaskStatusCompleted)
}

// MarkFailed marks the task as failed
func (sm *StateManager) MarkFailed() error {
	return sm.UpdateStatus(TaskStatusFailed)
}

// MarkInterrupted marks the task as interrupted
func (sm *StateManager) MarkInterrupted() error {
	return sm.UpdateStatus(TaskStatusInterrupted)
}

// IsRunning returns true if the task is currently running
func (sm *StateManager) IsRunning() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if sm.state == nil {
		return false
	}

	return sm.state.Status == TaskStatusRunning
}

// IsCompleted returns true if the task is completed
func (sm *StateManager) IsCompleted() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if sm.state == nil {
		return false
	}

	return sm.state.Status == TaskStatusCompleted
}

// IsInterrupted returns true if the task was interrupted
func (sm *StateManager) IsInterrupted() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if sm.state == nil {
		return false
	}

	return sm.state.Status == TaskStatusInterrupted
}

// SetAutoSave enables or disables automatic saving
func (sm *StateManager) SetAutoSave(enabled bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.autoSave = enabled

	// If enabling auto-save and there are pending changes, save now
	if enabled && sm.dirty {
		_ = sm.save()
	}
}

// Save manually saves the current state
func (sm *StateManager) Save() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	return sm.save()
}

// load loads state from storage
func (sm *StateManager) load() error {
	val, ok := sm.storage.Get(StateStorageKey)
	if !ok {
		// No existing state
		sm.state = nil
		return nil
	}

	// Convert to JSON and unmarshal
	data, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("failed to marshal stored state: %w", err)
	}

	var state TaskState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("failed to unmarshal state: %w", err)
	}

	sm.state = &state
	sm.dirty = false

	return nil
}

// save persists state to storage
func (sm *StateManager) save() error {
	if sm.state == nil {
		return nil
	}

	if err := sm.storage.Set(StateStorageKey, sm.state); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	sm.dirty = false
	return nil
}

// copyState creates a deep copy of a task state
func copyState(state *TaskState) *TaskState {
	if state == nil {
		return nil
	}

	stateCopy := &TaskState{
		TaskID:         state.TaskID,
		Status:         state.Status,
		StartedAt:      state.StartedAt,
		LastActivity:   state.LastActivity,
		CurrentPrompt:  state.CurrentPrompt,
		Mode:           state.Mode,
		CheckpointHash: state.CheckpointHash,
		Metadata:       make(map[string]interface{}),
	}

	// Deep copy metadata
	for k, v := range state.Metadata {
		stateCopy.Metadata[k] = v
	}

	return stateCopy
}

// GetDataDir returns the data directory for this state manager
func (sm *StateManager) GetDataDir() string {
	return sm.dataDir
}

// GetTaskID returns the task ID
func (sm *StateManager) GetTaskID() string {
	return sm.taskID
}

// StateStore manages state managers for multiple tasks
type StateStore struct {
	// states maps task IDs to state managers
	states map[string]*StateManager

	// mu protects concurrent access
	mu sync.RWMutex

	// dataDir is the base directory for all state data
	dataDir string
}

// NewStateStore creates a new state store
func NewStateStore(dataDir string) (*StateStore, error) {
	if dataDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		dataDir = filepath.Join(homeDir, ".cline", "data", "tasks")
	}

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create state store directory: %w", err)
	}

	return &StateStore{
		states:  make(map[string]*StateManager),
		dataDir: dataDir,
	}, nil
}

// GetOrCreateManager gets an existing state manager or creates a new one
func (ss *StateStore) GetOrCreateManager(taskID string) (*StateManager, error) {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if sm, ok := ss.states[taskID]; ok {
		return sm, nil
	}

	sm, err := NewStateManager(taskID, filepath.Join(ss.dataDir, taskID))
	if err != nil {
		return nil, fmt.Errorf("failed to create state manager for task %s: %w", taskID, err)
	}

	ss.states[taskID] = sm
	return sm, nil
}

// GetManager gets an existing state manager without creating
func (ss *StateStore) GetManager(taskID string) (*StateManager, bool) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	sm, ok := ss.states[taskID]
	return sm, ok
}

// RemoveManager removes a state manager from the store
func (ss *StateStore) RemoveManager(taskID string) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if sm, ok := ss.states[taskID]; ok {
		if err := sm.Close(); err != nil {
			return fmt.Errorf("failed to close state manager: %w", err)
		}
		delete(ss.states, taskID)
	}

	return nil
}

// Close closes all state managers in the store
func (ss *StateStore) Close() error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	var errs []string
	for taskID, sm := range ss.states {
		if err := sm.Close(); err != nil {
			errs = append(errs, fmt.Sprintf("task %s: %v", taskID, err))
		}
	}

	ss.states = make(map[string]*StateManager)

	if len(errs) > 0 {
		return fmt.Errorf("errors closing states: %s", joinErrors(errs))
	}

	return nil
}

func joinErrors(errs []string) string {
	if len(errs) == 0 {
		return ""
	}
	result := errs[0]
	for i := 1; i < len(errs); i++ {
		result += "; " + errs[i]
	}
	return result
}
