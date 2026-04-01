// Package state provides centralized state management with caching
// and session-scoped overrides for the Cline CLI.
// This replicates the TypeScript StateManager functionality.
package state

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/cline/cline/golang-cli/internal/storage"
)

// StateManager provides centralized state management with caching
// and session-scoped overrides (non-persistent)
type StateManager struct {
	storage          *storage.StorageContext
	globalCache      map[string]interface{}
	workspaceCache   map[string]interface{}
	sessionOverrides map[string]interface{}
	pendingWrites    map[string]storage.StorageType // key -> storage type
	mu               sync.RWMutex
	flushTimer       *time.Timer
	flushInterval    time.Duration
	isDirty          bool
}

// ManagerOptions provides options for creating a StateManager
type ManagerOptions struct {
	Storage       *storage.StorageContext
	FlushInterval time.Duration
}

// NewStateManager creates a new StateManager with the given storage context
func NewStateManager(opts ManagerOptions) *StateManager {
	flushInterval := opts.FlushInterval
	if flushInterval <= 0 {
		flushInterval = 100 * time.Millisecond // Default debounce interval
	}

	return &StateManager{
		storage:          opts.Storage,
		globalCache:      make(map[string]interface{}),
		workspaceCache:   make(map[string]interface{}),
		sessionOverrides: make(map[string]interface{}),
		pendingWrites:    make(map[string]storage.StorageType),
		flushInterval:    flushInterval,
	}
}

// Load initializes caches from storage
func (sm *StateManager) Load() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Load global state into cache
	if sm.storage.GlobalState != nil {
		allGlobal := sm.storage.GlobalState.GetAll()
		for key, value := range allGlobal {
			sm.globalCache[key] = value
		}
	}

	// Load workspace state into cache
	if sm.storage.WorkspaceState != nil {
		allWorkspace := sm.storage.WorkspaceState.GetAll()
		for key, value := range allWorkspace {
			sm.workspaceCache[key] = value
		}
	}

	return nil
}

// GetGlobalStateKey retrieves value from global state
// Checks session overrides first, then cache, then storage
func (sm *StateManager) GetGlobalStateKey(key string) (interface{}, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Check session override first (highest priority)
	if val, ok := sm.sessionOverrides[key]; ok {
		return val, true
	}

	// Check cache
	if val, ok := sm.globalCache[key]; ok {
		return val, true
	}

	// Fall back to storage
	if sm.storage != nil && sm.storage.GlobalState != nil {
		return sm.storage.GlobalState.Get(key)
	}

	return nil, false
}

// GetWorkspaceStateKey retrieves value from workspace state
// Checks session overrides first, then cache, then storage
func (sm *StateManager) GetWorkspaceStateKey(key string) (interface{}, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Check session override first (highest priority)
	if val, ok := sm.sessionOverrides[key]; ok {
		return val, true
	}

	// Check cache
	if val, ok := sm.workspaceCache[key]; ok {
		return val, true
	}

	// Fall back to storage
	if sm.storage != nil && sm.storage.WorkspaceState != nil {
		return sm.storage.WorkspaceState.Get(key)
	}

	return nil, false
}

// GetSecretKey retrieves a secret value
// Secrets are not cached for security reasons
func (sm *StateManager) GetSecretKey(key string) (interface{}, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if sm.storage != nil && sm.storage.Secrets != nil {
		return sm.storage.Secrets.Get(key)
	}

	return nil, false
}

// SetGlobalState updates global state (marks for persistence)
func (sm *StateManager) SetGlobalState(key string, value interface{}) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.globalCache[key] = value
	sm.pendingWrites[key] = storage.GlobalStateType
	sm.isDirty = true

	// Schedule flush
	sm.scheduleFlush()
}

// SetWorkspaceState updates workspace state (marks for persistence)
func (sm *StateManager) SetWorkspaceState(key string, value interface{}) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.workspaceCache[key] = value
	sm.pendingWrites[key] = storage.WorkspaceStateType
	sm.isDirty = true

	// Schedule flush
	sm.scheduleFlush()
}

// SetSecret sets a secret value
func (sm *StateManager) SetSecret(key string, value interface{}) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.storage.Secrets == nil {
		return fmt.Errorf("secrets storage not initialized")
	}

	// Secrets are written immediately (not batched) for security
	return sm.storage.Secrets.Set(key, value)
}

// SetSessionOverride sets a session-scoped value (not persisted)
// Session overrides take precedence over all other state
func (sm *StateManager) SetSessionOverride(key string, value interface{}) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.sessionOverrides[key] = value
}

// GetSessionOverride gets a session-scoped value
func (sm *StateManager) GetSessionOverride(key string) (interface{}, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	val, ok := sm.sessionOverrides[key]
	return val, ok
}

// ClearSessionOverride removes a session override
func (sm *StateManager) ClearSessionOverride(key string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.sessionOverrides, key)
}

// ClearAllSessionOverrides removes all session overrides
func (sm *StateManager) ClearAllSessionOverrides() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.sessionOverrides = make(map[string]interface{})
}

// DeleteGlobalState deletes a key from global state
func (sm *StateManager) DeleteGlobalState(key string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.globalCache, key)
	delete(sm.pendingWrites, key)

	if sm.storage.GlobalState != nil {
		return sm.storage.GlobalState.Delete(key)
	}

	return nil
}

// DeleteWorkspaceState deletes a key from workspace state
func (sm *StateManager) DeleteWorkspaceState(key string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.workspaceCache, key)
	delete(sm.pendingWrites, key)

	if sm.storage.WorkspaceState != nil {
		return sm.storage.WorkspaceState.Delete(key)
	}

	return nil
}

// DeleteSecret deletes a secret
func (sm *StateManager) DeleteSecret(key string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.storage.Secrets != nil {
		return sm.storage.Secrets.Delete(key)
	}

	return nil
}

// FlushPendingState persists pending writes to storage
func (sm *StateManager) FlushPendingState() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	return sm.flushUnlocked()
}

// flushUnlocked performs the actual flush (must hold lock)
func (sm *StateManager) flushUnlocked() error {
	if !sm.isDirty || len(sm.pendingWrites) == 0 {
		return nil
	}

	// Group writes by storage type
	globalWrites := make(map[string]interface{})
	workspaceWrites := make(map[string]interface{})

	for key, storageType := range sm.pendingWrites {
		switch storageType {
		case storage.GlobalStateType:
			if val, ok := sm.globalCache[key]; ok {
				globalWrites[key] = val
			}
		case storage.WorkspaceStateType:
			if val, ok := sm.workspaceCache[key]; ok {
				workspaceWrites[key] = val
			}
		}
	}

	// Perform batch writes
	var errs []error

	if len(globalWrites) > 0 && sm.storage.GlobalState != nil {
		if err := sm.storage.GlobalState.SetBatch(globalWrites); err != nil {
			errs = append(errs, fmt.Errorf("failed to write global state: %w", err))
		}
	}

	if len(workspaceWrites) > 0 && sm.storage.WorkspaceState != nil {
		if err := sm.storage.WorkspaceState.SetBatch(workspaceWrites); err != nil {
			errs = append(errs, fmt.Errorf("failed to write workspace state: %w", err))
		}
	}

	// Clear pending writes on success
	if len(errs) == 0 {
		sm.pendingWrites = make(map[string]storage.StorageType)
		sm.isDirty = false
	}

	if len(errs) > 0 {
		return fmt.Errorf("flush errors: %v", errs)
	}

	return nil
}

// scheduleFlush schedules a debounced flush
func (sm *StateManager) scheduleFlush() {
	// Cancel existing timer if any
	if sm.flushTimer != nil {
		sm.flushTimer.Stop()
	}

	// Schedule new flush
	sm.flushTimer = time.AfterFunc(sm.flushInterval, func() {
		_ = sm.FlushPendingState()
	})
}

// ForceFlush immediately flushes all pending state
func (sm *StateManager) ForceFlush() error {
	// Cancel any pending timer
	sm.mu.Lock()
	if sm.flushTimer != nil {
		sm.flushTimer.Stop()
		sm.flushTimer = nil
	}
	sm.mu.Unlock()

	return sm.FlushPendingState()
}

// GetAllGlobalState returns all global state as a map
func (sm *StateManager) GetAllGlobalState() map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	result := make(map[string]interface{}, len(sm.globalCache))
	for k, v := range sm.globalCache {
		result[k] = v
	}
	return result
}

// GetAllWorkspaceState returns all workspace state as a map
func (sm *StateManager) GetAllWorkspaceState() map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	result := make(map[string]interface{}, len(sm.workspaceCache))
	for k, v := range sm.workspaceCache {
		result[k] = v
	}
	return result
}

// GetStorage returns the underlying storage context
func (sm *StateManager) GetStorage() *storage.StorageContext {
	return sm.storage
}

// GetTyped retrieves a value and unmarshals it into the provided type from global state
func (sm *StateManager) GetTyped(key string, dest interface{}) (bool, error) {
	val, ok := sm.GetGlobalStateKey(key)
	if !ok {
		return false, nil
	}

	// Marshal and unmarshal to convert types
	data, err := json.Marshal(val)
	if err != nil {
		return false, fmt.Errorf("failed to marshal value: %w", err)
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return false, fmt.Errorf("failed to unmarshal value: %w", err)
	}

	return true, nil
}

// GetTypedFromWorkspace retrieves a value and unmarshals it into the provided type from workspace state
func (sm *StateManager) GetTypedFromWorkspace(key string, dest interface{}) (bool, error) {
	val, ok := sm.GetWorkspaceStateKey(key)
	if !ok {
		return false, nil
	}

	// Marshal and unmarshal to convert types
	data, err := json.Marshal(val)
	if err != nil {
		return false, fmt.Errorf("failed to marshal value: %w", err)
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return false, fmt.Errorf("failed to unmarshal value: %w", err)
	}

	return true, nil
}

// SetTyped sets a JSON-serializable value in global state
func (sm *StateManager) SetTyped(key string, value interface{}) error {
	// Validate that value can be serialized
	if _, err := json.Marshal(value); err != nil {
		return fmt.Errorf("value is not JSON-serializable: %w", err)
	}

	sm.SetGlobalState(key, value)
	return nil
}

// Close flushes pending state and cleans up resources
func (sm *StateManager) Close() error {
	// Flush any pending writes
	if err := sm.ForceFlush(); err != nil {
		return fmt.Errorf("failed to flush state on close: %w", err)
	}

	return nil
}

// IsDirty returns true if there are pending writes
func (sm *StateManager) IsDirty() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.isDirty
}

// PendingWriteCount returns the number of pending writes
func (sm *StateManager) PendingWriteCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.pendingWrites)
}