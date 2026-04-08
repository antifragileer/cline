package host

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
)

// StateVersion represents a versioned state entry for conflict resolution
type StateVersion struct {
	Version   int64     `json:"version"`
	Timestamp time.Time `json:"timestamp"`
	Data      []byte    `json:"data"`
	Source    string    `json:"source"` // "cli" or "core"
}

// SyncEvent represents a state synchronization event
type SyncEvent struct {
	Type      SyncEventType `json:"type"`
	Key       string        `json:"key"`
	Version   int64         `json:"version"`
	Timestamp time.Time     `json:"timestamp"`
	Source    string        `json:"source"`
	Data      []byte        `json:"data"`
}

// SyncEventType represents the type of synchronization event
type SyncEventType int

const (
	// EventStateChanged indicates a state value has changed
	EventStateChanged SyncEventType = iota
	// EventStateDeleted indicates a state value has been deleted
	EventStateDeleted
	// EventSyncRequested indicates a sync has been requested
	EventSyncRequested
	// EventConflictResolved indicates a conflict has been resolved
	EventConflictResolved
	// EventRollback indicates an optimistic update has been rolled back
	EventRollback
)

func (t SyncEventType) String() string {
	switch t {
	case EventStateChanged:
		return "state_changed"
	case EventStateDeleted:
		return "state_deleted"
	case EventSyncRequested:
		return "sync_requested"
	case EventConflictResolved:
		return "conflict_resolved"
	case EventRollback:
		return "rollback"
	default:
		return "unknown"
	}
}

// StateSyncConfig holds configuration for state synchronization
type StateSyncConfig struct {
	// ConflictResolutionStrategy determines how conflicts are resolved
	// "last-write-wins" (default) or "timestamp-wins"
	ConflictResolutionStrategy string

	// OptimisticUpdates enables optimistic updates with rollback capability
	OptimisticUpdates bool

	// MaxRetries for sync operations
	MaxRetries int

	// RetryDelay between sync retries
	RetryDelay time.Duration

	// SyncInterval for periodic full sync
	SyncInterval time.Duration

	// EventBufferSize for the event channel
	EventBufferSize int

	// OnEvent callback for sync events
	OnEvent func(event SyncEvent)

	// OnConflict callback when conflicts are detected
	OnConflict func(key string, local, remote StateVersion) StateVersion

	// OnError callback for errors
	OnError func(error)
}

// DefaultStateSyncConfig returns a default configuration
func DefaultStateSyncConfig() StateSyncConfig {
	return StateSyncConfig{
		ConflictResolutionStrategy: "last-write-wins",
		OptimisticUpdates:          true,
		MaxRetries:                 3,
		RetryDelay:                 1 * time.Second,
		SyncInterval:               30 * time.Second,
		EventBufferSize:            100,
	}
}

// PendingUpdate represents an optimistic update that can be rolled back
type PendingUpdate struct {
	Key         string
	OldValue    *StateVersion
	NewValue    StateVersion
	AppliedAt   time.Time
	Committed   bool
	CommittedAt *time.Time
}

// StateSync manages bidirectional state synchronization between CLI and core
type StateSync struct {
	config StateSyncConfig

	// Local state storage
	mu         sync.RWMutex
	localState map[string]StateVersion
	versionGen atomic.Int64

	// Optimistic updates tracking
	pendingMu      sync.RWMutex
	pendingUpdates map[string]*PendingUpdate

	// Event handling
	eventChan chan SyncEvent
	eventSubs []chan SyncEvent

	// gRPC client for core communication
	client   *Client
	grpcConn *grpc.ClientConn
	stream   grpc.ClientStream

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Sync status
	syncing    atomic.Bool
	lastSyncAt atomic.Value
}

// NewStateSync creates a new state synchronization manager
func NewStateSync(config StateSyncConfig, client *Client) (*StateSync, error) {
	if client == nil {
		return nil, fmt.Errorf("gRPC client is required")
	}

	ctx, cancel := context.WithCancel(context.Background())

	ss := &StateSync{
		config:         config,
		localState:     make(map[string]StateVersion),
		pendingUpdates: make(map[string]*PendingUpdate),
		eventChan:      make(chan SyncEvent, config.EventBufferSize),
		client:         client,
		ctx:            ctx,
		cancel:         cancel,
	}

	// Initialize version generator with current timestamp
	ss.versionGen.Store(time.Now().UnixNano())

	return ss, nil
}

// Start initializes the state synchronization
func (ss *StateSync) Start() error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	// Get gRPC connection from client pool
	conn, err := ss.client.GetPool().GetConnection()
	if err != nil {
		return fmt.Errorf("failed to get gRPC connection: %w", err)
	}
	ss.grpcConn = conn

	// Start event processor
	ss.wg.Add(1)
	go ss.eventProcessor()

	// Start periodic sync if configured
	if ss.config.SyncInterval > 0 {
		ss.wg.Add(1)
		go ss.periodicSync()
	}

	return nil
}

// Stop shuts down the state synchronization
func (ss *StateSync) Stop() error {
	ss.cancel()
	ss.wg.Wait()

	ss.mu.Lock()
	defer ss.mu.Unlock()

	if ss.grpcConn != nil {
		// Don't close the connection as it's managed by the pool
		ss.grpcConn = nil
	}

	// Close event channel
	close(ss.eventChan)

	// Close subscriber channels
	for _, sub := range ss.eventSubs {
		close(sub)
	}
	ss.eventSubs = nil

	return nil
}

// Get retrieves a value from local state
func (ss *StateSync) Get(key string) (*StateVersion, bool) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	version, ok := ss.localState[key]
	if !ok {
		// Check pending updates for optimistic value
		ss.pendingMu.RLock()
		if pending, ok := ss.pendingUpdates[key]; ok && !pending.Committed {
			ss.pendingMu.RUnlock()
			return &pending.NewValue, true
		}
		ss.pendingMu.RUnlock()
		return nil, false
	}

	// Check if there's a pending update with higher version
	ss.pendingMu.RLock()
	if pending, ok := ss.pendingUpdates[key]; ok && !pending.Committed {
		if pending.NewValue.Version > version.Version {
			ss.pendingMu.RUnlock()
			return &pending.NewValue, true
		}
	}
	ss.pendingMu.RUnlock()

	return &version, true
}

// Set updates a value with optimistic update support
func (ss *StateSync) Set(key string, data []byte) error {
	if ss.syncing.Load() {
		return fmt.Errorf("cannot update while syncing")
	}

	// Generate new version
	newVersion := ss.versionGen.Add(1)

	// Get current value for potential rollback (with proper locking)
	ss.mu.RLock()
	var oldValue *StateVersion
	if current, ok := ss.localState[key]; ok {
		oldCopy := current
		oldValue = &oldCopy
	}
	ss.mu.RUnlock()

	// Create new state version
	stateVersion := StateVersion{
		Version:   newVersion,
		Timestamp: time.Now(),
		Data:      data,
		Source:    "cli",
	}

	// Apply optimistic update if enabled
	if ss.config.OptimisticUpdates {
		ss.pendingMu.Lock()
		ss.pendingUpdates[key] = &PendingUpdate{
			Key:       key,
			OldValue:  oldValue,
			NewValue:  stateVersion,
			AppliedAt: time.Now(),
			Committed: false,
		}
		ss.pendingMu.Unlock()

		// Emit optimistic update event
		ss.emitEvent(SyncEvent{
			Type:      EventStateChanged,
			Key:       key,
			Version:   newVersion,
			Timestamp: time.Now(),
			Source:    "cli",
			Data:      data,
		})
	}

	// Apply to local state
	ss.mu.Lock()
	ss.localState[key] = stateVersion
	ss.mu.Unlock()

	// Sync to core
	go ss.syncToCore(key, stateVersion)

	return nil
}

// Delete removes a value from state
func (ss *StateSync) Delete(key string) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	delete(ss.localState, key)

	// Emit delete event
	ss.emitEvent(SyncEvent{
		Type:      EventStateDeleted,
		Key:       key,
		Timestamp: time.Now(),
		Source:    "cli",
	})

	return nil
}

// SyncFromCore requests a full sync from the core
func (ss *StateSync) SyncFromCore(ctx context.Context) error {
	ss.syncing.Store(true)
	defer ss.syncing.Store(false)

	// This would typically make a gRPC call to get the full state from core
	// For now, we emit a sync request event
	ss.emitEvent(SyncEvent{
		Type:      EventSyncRequested,
		Timestamp: time.Now(),
		Source:    "cli",
	})

	return nil
}

// GetAll returns all local state
func (ss *StateSync) GetAll() map[string]StateVersion {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	result := make(map[string]StateVersion, len(ss.localState))
	for k, v := range ss.localState {
		result[k] = v
	}

	return result
}

// Subscribe returns a channel for receiving sync events
func (ss *StateSync) Subscribe() <-chan SyncEvent {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	ch := make(chan SyncEvent, ss.config.EventBufferSize)
	ss.eventSubs = append(ss.eventSubs, ch)
	return ch
}

// Unsubscribe removes a subscription
func (ss *StateSync) Unsubscribe(ch <-chan SyncEvent) {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	for i, sub := range ss.eventSubs {
		if sub == ch {
			// Close and remove the channel
			close(sub)
			ss.eventSubs = append(ss.eventSubs[:i], ss.eventSubs[i+1:]...)
			return
		}
	}
}

// Rollback reverts an optimistic update
func (ss *StateSync) Rollback(key string) error {
	ss.pendingMu.Lock()
	defer ss.pendingMu.Unlock()

	pending, ok := ss.pendingUpdates[key]
	if !ok {
		return fmt.Errorf("no pending update for key: %s", key)
	}

	if pending.Committed {
		return fmt.Errorf("update already committed for key: %s", key)
	}

	// Revert to old value
	ss.mu.Lock()
	if pending.OldValue != nil {
		ss.localState[key] = *pending.OldValue
	} else {
		delete(ss.localState, key)
	}
	ss.mu.Unlock()

	// Remove from pending
	delete(ss.pendingUpdates, key)

	// Emit rollback event
	ss.emitEvent(SyncEvent{
		Type:      EventRollback,
		Key:       key,
		Timestamp: time.Now(),
		Source:    "cli",
	})

	return nil
}

// Commit marks a pending update as committed
func (ss *StateSync) Commit(key string) error {
	ss.pendingMu.Lock()
	defer ss.pendingMu.Unlock()

	pending, ok := ss.pendingUpdates[key]
	if !ok {
		return fmt.Errorf("no pending update for key: %s", key)
	}

	now := time.Now()
	pending.Committed = true
	pending.CommittedAt = &now

	return nil
}

// ResolveConflict resolves a conflict between local and remote versions
func (ss *StateSync) ResolveConflict(key string, local, remote StateVersion) StateVersion {
	// Use custom conflict resolver if provided
	if ss.config.OnConflict != nil {
		return ss.config.OnConflict(key, local, remote)
	}

	// Default: last-write-wins based on timestamp
	switch ss.config.ConflictResolutionStrategy {
	case "timestamp-wins":
		if local.Timestamp.After(remote.Timestamp) {
			return local
		}
		return remote
	case "last-write-wins":
		fallthrough
	default:
		// Higher version wins
		if local.Version > remote.Version {
			return local
		}
		if remote.Version > local.Version {
			return remote
		}
		// Same version, use timestamp
		if local.Timestamp.After(remote.Timestamp) {
			return local
		}
		return remote
	}
}

// HandleRemoteUpdate processes an update from the core
func (ss *StateSync) HandleRemoteUpdate(key string, remoteVersion StateVersion) error {
	ss.mu.Lock()
	localVersion, exists := ss.localState[key]
	ss.mu.Unlock()

	if !exists {
		// New key from core, accept it
		ss.mu.Lock()
		ss.localState[key] = remoteVersion
		ss.mu.Unlock()

		ss.emitEvent(SyncEvent{
			Type:      EventStateChanged,
			Key:       key,
			Version:   remoteVersion.Version,
			Timestamp: time.Now(),
			Source:    "core",
			Data:      remoteVersion.Data,
		})
		return nil
	}

	// Check for conflict
	if localVersion.Version != remoteVersion.Version {
		// Conflict detected, resolve it
		resolved := ss.ResolveConflict(key, localVersion, remoteVersion)

		ss.mu.Lock()
		ss.localState[key] = resolved
		ss.mu.Unlock()

		ss.emitEvent(SyncEvent{
			Type:      EventConflictResolved,
			Key:       key,
			Version:   resolved.Version,
			Timestamp: time.Now(),
			Source:    "core",
			Data:      resolved.Data,
		})

		// If local won, sync back to core
		if resolved.Version == localVersion.Version {
			go ss.syncToCore(key, resolved)
		}
	}

	return nil
}

// GetStats returns synchronization statistics
func (ss *StateSync) GetStats() SyncStats {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	ss.pendingMu.RLock()
	pendingCount := len(ss.pendingUpdates)
	ss.pendingMu.RUnlock()

	lastSync, _ := ss.lastSyncAt.Load().(time.Time)

	return SyncStats{
		LocalKeys:      len(ss.localState),
		PendingUpdates: pendingCount,
		LastSyncAt:     lastSync,
		CurrentVersion: ss.versionGen.Load(),
	}
}

// SyncStats contains synchronization statistics
type SyncStats struct {
	LocalKeys      int
	PendingUpdates int
	LastSyncAt     time.Time
	CurrentVersion int64
}

// Internal methods

func (ss *StateSync) syncToCore(key string, version StateVersion) {
	// Retry logic for sync operations
	var lastErr error
	for attempt := 0; attempt <= ss.config.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ss.ctx.Done():
				return
			case <-time.After(ss.config.RetryDelay):
			}
		}

		// Attempt to sync to core
		// This would typically make a gRPC call to update the core state
		// For now, we simulate success
		if err := ss.sendUpdateToCore(key, version); err != nil {
			lastErr = err
			if ss.config.OnError != nil {
				ss.config.OnError(fmt.Errorf("sync attempt %d failed: %w", attempt+1, err))
			}
			continue
		}

		// Success - commit the update
		if err := ss.Commit(key); err != nil {
			if ss.config.OnError != nil {
				ss.config.OnError(err)
			}
		}
		return
	}

	// All retries failed, rollback
	if ss.config.OnError != nil {
		ss.config.OnError(fmt.Errorf("max retries exceeded for key %s: %w", key, lastErr))
	}

	// Rollback the optimistic update
	if err := ss.Rollback(key); err != nil {
		if ss.config.OnError != nil {
			ss.config.OnError(err)
		}
	}
}

func (ss *StateSync) sendUpdateToCore(key string, version StateVersion) error {
	// This would make the actual gRPC call to update core state
	// Placeholder implementation
	if ss.grpcConn == nil {
		return fmt.Errorf("no gRPC connection available")
	}

	// Serialize the update
	data, err := json.Marshal(version)
	if err != nil {
		return fmt.Errorf("failed to marshal update: %w", err)
	}

	// Simulate network delay and potential failures
	// In production, this would be a real gRPC call
	select {
	case <-ss.ctx.Done():
		return ss.ctx.Err()
	case <-time.After(10 * time.Millisecond):
		_ = data
		return nil
	}
}

func (ss *StateSync) eventProcessor() {
	defer ss.wg.Done()

	for {
		select {
		case <-ss.ctx.Done():
			return
		case event, ok := <-ss.eventChan:
			if !ok {
				return
			}

			// Process event
			ss.processEvent(event)

			// Notify subscribers
			ss.notifySubscribers(event)

			// Call user callback
			if ss.config.OnEvent != nil {
				ss.config.OnEvent(event)
			}
		}
	}
}

func (ss *StateSync) processEvent(event SyncEvent) {
	// Event processing logic
	switch event.Type {
	case EventStateChanged:
		// State changed, update tracking
	case EventConflictResolved:
		// Conflict resolved, log or handle specially
	case EventRollback:
		// Rollback occurred, clean up
	}
}

func (ss *StateSync) notifySubscribers(event SyncEvent) {
	ss.mu.RLock()
	subs := make([]chan SyncEvent, len(ss.eventSubs))
	copy(subs, ss.eventSubs)
	ss.mu.RUnlock()

	for _, sub := range subs {
		select {
		case sub <- event:
		default:
			// Channel full, skip
		}
	}
}

func (ss *StateSync) emitEvent(event SyncEvent) {
	select {
	case ss.eventChan <- event:
	default:
		// Channel full, drop event
		if ss.config.OnError != nil {
			ss.config.OnError(fmt.Errorf("event channel full, dropping event"))
		}
	}
}

func (ss *StateSync) periodicSync() {
	defer ss.wg.Done()

	ticker := time.NewTicker(ss.config.SyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ss.ctx.Done():
			return
		case <-ticker.C:
			if err := ss.SyncFromCore(ss.ctx); err != nil {
				if ss.config.OnError != nil {
					ss.config.OnError(fmt.Errorf("periodic sync failed: %w", err))
				}
			}
			ss.lastSyncAt.Store(time.Now())
		}
	}
}

// SetJSON is a convenience method for setting JSON-serializable values
func (ss *StateSync) SetJSON(key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}
	return ss.Set(key, data)
}

// GetJSON is a convenience method for getting JSON-deserializable values
func (ss *StateSync) GetJSON(key string, dest interface{}) (bool, error) {
	version, ok := ss.Get(key)
	if !ok {
		return false, nil
	}

	if err := json.Unmarshal(version.Data, dest); err != nil {
		return false, fmt.Errorf("failed to unmarshal value: %w", err)
	}

	return true, nil
}
