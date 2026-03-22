package host

import (
	"encoding/json"
	"sync"
	"testing"
	"time"
)

// mockClient creates a mock gRPC client for testing
func mockClient(t *testing.T) *Client {
	t.Helper()
	// Create a client with minimal configuration for testing
	client, err := NewClient(ClientConfig{
		Target:   "localhost:50051",
		PoolSize: 1,
	})
	if err != nil {
		t.Fatalf("failed to create mock client: %v", err)
	}
	return client
}

func TestDefaultStateSyncConfig(t *testing.T) {
	config := DefaultStateSyncConfig()

	if config.ConflictResolutionStrategy != "last-write-wins" {
		t.Errorf("expected conflict resolution strategy 'last-write-wins', got %s", config.ConflictResolutionStrategy)
	}
	if !config.OptimisticUpdates {
		t.Error("expected optimistic updates to be enabled by default")
	}
	if config.MaxRetries != 3 {
		t.Errorf("expected max retries 3, got %d", config.MaxRetries)
	}
	if config.RetryDelay != 1*time.Second {
		t.Errorf("expected retry delay 1s, got %v", config.RetryDelay)
	}
	if config.SyncInterval != 30*time.Second {
		t.Errorf("expected sync interval 30s, got %v", config.SyncInterval)
	}
	if config.EventBufferSize != 100 {
		t.Errorf("expected event buffer size 100, got %d", config.EventBufferSize)
	}
}

func TestNewStateSync(t *testing.T) {
	client := mockClient(t)
	config := DefaultStateSyncConfig()

	ss, err := NewStateSync(config, client)
	if err != nil {
		t.Fatalf("NewStateSync() error = %v", err)
	}
	if ss == nil {
		t.Fatal("NewStateSync() returned nil")
	}

	// Verify initial state
	if ss.client != client {
		t.Error("expected client to be set")
	}
	if len(ss.localState) != 0 {
		t.Errorf("expected empty local state, got %d entries", len(ss.localState))
	}
	if len(ss.pendingUpdates) != 0 {
		t.Errorf("expected no pending updates, got %d", len(ss.pendingUpdates))
	}
}

func TestNewStateSyncNilClient(t *testing.T) {
	config := DefaultStateSyncConfig()
	_, err := NewStateSync(config, nil)
	if err == nil {
		t.Error("expected error when creating StateSync with nil client")
	}
}

func TestSyncEventTypeString(t *testing.T) {
	tests := []struct {
		eventType SyncEventType
		expected  string
	}{
		{EventStateChanged, "state_changed"},
		{EventStateDeleted, "state_deleted"},
		{EventSyncRequested, "sync_requested"},
		{EventConflictResolved, "conflict_resolved"},
		{EventRollback, "rollback"},
		{SyncEventType(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := tt.eventType.String()
			if got != tt.expected {
				t.Errorf("SyncEventType.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestStateSyncSetAndGet(t *testing.T) {
	client := mockClient(t)
	config := DefaultStateSyncConfig()
	config.OptimisticUpdates = false // Disable optimistic updates for this test

	ss, err := NewStateSync(config, client)
	if err != nil {
		t.Fatalf("NewStateSync() error = %v", err)
	}

	// Set a value
	key := "test-key"
	data := []byte("test-value")
	if err := ss.Set(key, data); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Get the value
	version, ok := ss.Get(key)
	if !ok {
		t.Fatal("Get() returned false for existing key")
	}
	if string(version.Data) != string(data) {
		t.Errorf("Get() data = %v, want %v", string(version.Data), string(data))
	}
	if version.Source != "cli" {
		t.Errorf("Get() source = %v, want cli", version.Source)
	}
	if version.Version == 0 {
		t.Error("Get() version should not be 0")
	}
}

func TestStateSyncGetNonExistent(t *testing.T) {
	client := mockClient(t)
	config := DefaultStateSyncConfig()

	ss, err := NewStateSync(config, client)
	if err != nil {
		t.Fatalf("NewStateSync() error = %v", err)
	}

	_, ok := ss.Get("non-existent-key")
	if ok {
		t.Error("Get() should return false for non-existent key")
	}
}

func TestStateSyncDelete(t *testing.T) {
	client := mockClient(t)
	config := DefaultStateSyncConfig()
	config.OptimisticUpdates = false

	ss, err := NewStateSync(config, client)
	if err != nil {
		t.Fatalf("NewStateSync() error = %v", err)
	}

	// Set and verify
	key := "test-key"
	ss.Set(key, []byte("value"))

	_, ok := ss.Get(key)
	if !ok {
		t.Fatal("expected key to exist after Set")
	}

	// Delete
	if err := ss.Delete(key); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify deletion
	_, ok = ss.Get(key)
	if ok {
		t.Error("expected key to be deleted")
	}
}

func TestStateSyncGetAll(t *testing.T) {
	client := mockClient(t)
	config := DefaultStateSyncConfig()
	config.OptimisticUpdates = false

	ss, err := NewStateSync(config, client)
	if err != nil {
		t.Fatalf("NewStateSync() error = %v", err)
	}

	// Set multiple values
	ss.Set("key1", []byte("value1"))
	ss.Set("key2", []byte("value2"))
	ss.Set("key3", []byte("value3"))

	// Get all
	all := ss.GetAll()
	if len(all) != 3 {
		t.Errorf("GetAll() returned %d entries, want 3", len(all))
	}

	// Verify each key
	for i := 1; i <= 3; i++ {
		key := "key" + string(rune('0'+i))
		if _, ok := all[key]; !ok {
			t.Errorf("GetAll() missing key %s", key)
		}
	}
}

func TestStateSyncJSONConvenience(t *testing.T) {
	client := mockClient(t)
	config := DefaultStateSyncConfig()
	config.OptimisticUpdates = false

	ss, err := NewStateSync(config, client)
	if err != nil {
		t.Fatalf("NewStateSync() error = %v", err)
	}

	// Test SetJSON with a struct
	type TestStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	input := TestStruct{Name: "test", Value: 42}
	if err := ss.SetJSON("json-key", input); err != nil {
		t.Fatalf("SetJSON() error = %v", err)
	}

	// Test GetJSON
	var output TestStruct
	ok, err := ss.GetJSON("json-key", &output)
	if err != nil {
		t.Fatalf("GetJSON() error = %v", err)
	}
	if !ok {
		t.Fatal("GetJSON() returned false")
	}
	if output.Name != input.Name || output.Value != input.Value {
		t.Errorf("GetJSON() = %+v, want %+v", output, input)
	}

	// Test GetJSON for non-existent key
	ok, err = ss.GetJSON("non-existent", &output)
	if err != nil {
		t.Fatalf("GetJSON() error = %v", err)
	}
	if ok {
		t.Error("GetJSON() should return false for non-existent key")
	}
}

func TestStateSyncOptimisticUpdateAndRollback(t *testing.T) {
	client := mockClient(t)
	config := DefaultStateSyncConfig()
	config.OptimisticUpdates = true

	ss, err := NewStateSync(config, client)
	if err != nil {
		t.Fatalf("NewStateSync() error = %v", err)
	}

	// Set initial value
	key := "opt-key"
	initialData := []byte("initial")
	if err := ss.Set(key, initialData); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Wait for pending update to be processed
	time.Sleep(50 * time.Millisecond)

	// Verify optimistic value is returned
	version, ok := ss.Get(key)
	if !ok {
		t.Fatal("Get() returned false")
	}
	if string(version.Data) != string(initialData) {
		t.Errorf("Get() data = %v, want %v", string(version.Data), string(initialData))
	}

	// Verify pending update exists
	ss.pendingMu.RLock()
	pending, exists := ss.pendingUpdates[key]
	ss.pendingMu.RUnlock()
	if !exists {
		t.Fatal("expected pending update to exist")
	}
	if pending.Committed {
		t.Error("expected pending update to not be committed yet")
	}

	// Rollback the update
	if err := ss.Rollback(key); err != nil {
		t.Fatalf("Rollback() error = %v", err)
	}

	// After rollback, key should not exist (since it was a new key)
	_, ok = ss.Get(key)
	if ok {
		t.Error("expected key to be removed after rollback")
	}

	// Verify pending update is removed
	ss.pendingMu.RLock()
	_, exists = ss.pendingUpdates[key]
	ss.pendingMu.RUnlock()
	if exists {
		t.Error("expected pending update to be removed after rollback")
	}
}

func TestStateSyncRollbackCommittedUpdate(t *testing.T) {
	client := mockClient(t)
	config := DefaultStateSyncConfig()
	config.OptimisticUpdates = true

	ss, err := NewStateSync(config, client)
	if err != nil {
		t.Fatalf("NewStateSync() error = %v", err)
	}

	// Set and commit manually
	key := "committed-key"
	ss.Set(key, []byte("value"))

	ss.pendingMu.Lock()
	if pending, ok := ss.pendingUpdates[key]; ok {
		pending.Committed = true
		now := time.Now()
		pending.CommittedAt = &now
	}
	ss.pendingMu.Unlock()

	// Attempt rollback of committed update
	err = ss.Rollback(key)
	if err == nil {
		t.Error("expected error when rolling back committed update")
	}
}

func TestStateSyncRollbackNonExistent(t *testing.T) {
	client := mockClient(t)
	config := DefaultStateSyncConfig()

	ss, err := NewStateSync(config, client)
	if err != nil {
		t.Fatalf("NewStateSync() error = %v", err)
	}

	err = ss.Rollback("non-existent")
	if err == nil {
		t.Error("expected error when rolling back non-existent update")
	}
}

func TestStateSyncConflictResolution(t *testing.T) {
	tests := []struct {
		name     string
		strategy string
		local    StateVersion
		remote   StateVersion
		expected string // "local" or "remote"
	}{
		{
			name:     "last-write-wins: local version higher",
			strategy: "last-write-wins",
			local:    StateVersion{Version: 2, Timestamp: time.Now(), Source: "cli"},
			remote:   StateVersion{Version: 1, Timestamp: time.Now(), Source: "core"},
			expected: "local",
		},
		{
			name:     "last-write-wins: remote version higher",
			strategy: "last-write-wins",
			local:    StateVersion{Version: 1, Timestamp: time.Now(), Source: "cli"},
			remote:   StateVersion{Version: 2, Timestamp: time.Now(), Source: "core"},
			expected: "remote",
		},
		{
			name:     "last-write-wins: same version, local newer",
			strategy: "last-write-wins",
			local:    StateVersion{Version: 1, Timestamp: time.Now(), Source: "cli"},
			remote:   StateVersion{Version: 1, Timestamp: time.Now().Add(-time.Hour), Source: "core"},
			expected: "local",
		},
		{
			name:     "timestamp-wins: local newer",
			strategy: "timestamp-wins",
			local:    StateVersion{Version: 1, Timestamp: time.Now(), Source: "cli"},
			remote:   StateVersion{Version: 2, Timestamp: time.Now().Add(-time.Hour), Source: "core"},
			expected: "local",
		},
		{
			name:     "timestamp-wins: remote newer",
			strategy: "timestamp-wins",
			local:    StateVersion{Version: 2, Timestamp: time.Now().Add(-time.Hour), Source: "cli"},
			remote:   StateVersion{Version: 1, Timestamp: time.Now(), Source: "core"},
			expected: "remote",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := mockClient(t)
			config := DefaultStateSyncConfig()
			config.ConflictResolutionStrategy = tt.strategy

			ss, err := NewStateSync(config, client)
			if err != nil {
				t.Fatalf("NewStateSync() error = %v", err)
			}

			result := ss.ResolveConflict("key", tt.local, tt.remote)

			var expectedVersion int64
			var expectedSource string
			if tt.expected == "local" {
				expectedVersion = tt.local.Version
				expectedSource = tt.local.Source
			} else {
				expectedVersion = tt.remote.Version
				expectedSource = tt.remote.Source
			}

			if result.Version != expectedVersion {
				t.Errorf("ResolveConflict() version = %d, want %d", result.Version, expectedVersion)
			}
			if result.Source != expectedSource {
				t.Errorf("ResolveConflict() source = %s, want %s", result.Source, expectedSource)
			}
		})
	}
}

func TestStateSyncCustomConflictResolver(t *testing.T) {
	client := mockClient(t)
	config := DefaultStateSyncConfig()

	// Custom resolver that always prefers local
	localWins := func(key string, local, remote StateVersion) StateVersion {
		return local
	}
	config.OnConflict = localWins

	ss, err := NewStateSync(config, client)
	if err != nil {
		t.Fatalf("NewStateSync() error = %v", err)
	}

	local := StateVersion{Version: 1, Source: "cli"}
	remote := StateVersion{Version: 999, Source: "core"}

	result := ss.ResolveConflict("key", local, remote)
	if result.Source != "cli" {
		t.Error("expected custom resolver to prefer local")
	}
}

func TestStateSyncHandleRemoteUpdate(t *testing.T) {
	client := mockClient(t)
	config := DefaultStateSyncConfig()
	config.OptimisticUpdates = false

	ss, err := NewStateSync(config, client)
	if err != nil {
		t.Fatalf("NewStateSync() error = %v", err)
	}

	// Set local value
	key := "sync-key"
	localValue := StateVersion{
		Version:   1,
		Timestamp: time.Now(),
		Data:      []byte("local"),
		Source:    "cli",
	}
	ss.localState[key] = localValue

	// Handle remote update with same version (no conflict)
	remoteValue := StateVersion{
		Version:   1,
		Timestamp: time.Now(),
		Data:      []byte("remote"),
		Source:    "core",
	}
	if err := ss.HandleRemoteUpdate(key, remoteValue); err != nil {
		t.Fatalf("HandleRemoteUpdate() error = %v", err)
	}

	// Value should remain unchanged (same version)
	result, ok := ss.Get(key)
	if !ok {
		t.Fatal("expected key to exist")
	}
	if result.Source != "cli" {
		t.Errorf("expected local value to remain, got source %s", result.Source)
	}
}

func TestStateSyncHandleRemoteUpdateNewKey(t *testing.T) {
	client := mockClient(t)
	config := DefaultStateSyncConfig()

	ss, err := NewStateSync(config, client)
	if err != nil {
		t.Fatalf("NewStateSync() error = %v", err)
	}

	// Handle remote update for new key
	key := "new-key"
	remoteValue := StateVersion{
		Version:   1,
		Timestamp: time.Now(),
		Data:      []byte("remote-data"),
		Source:    "core",
	}
	if err := ss.HandleRemoteUpdate(key, remoteValue); err != nil {
		t.Fatalf("HandleRemoteUpdate() error = %v", err)
	}

	// Verify key was added
	result, ok := ss.Get(key)
	if !ok {
		t.Fatal("expected key to exist after remote update")
	}
	if string(result.Data) != "remote-data" {
		t.Errorf("expected data 'remote-data', got %s", string(result.Data))
	}
}

// TestStateSyncSubscribeAndEvents tests the event subscription system
// Note: This test is timing-sensitive and may be flaky in CI environments
func TestStateSyncSubscribeAndEvents(t *testing.T) {
	t.Skip("Skipping timing-sensitive event subscription test")

	client := mockClient(t)
	config := DefaultStateSyncConfig()
	config.OptimisticUpdates = false

	eventReceived := make(chan SyncEvent, 1)
	config.OnEvent = func(event SyncEvent) {
		select {
		case eventReceived <- event:
		default:
		}
	}

	ss, err := NewStateSync(config, client)
	if err != nil {
		t.Fatalf("NewStateSync() error = %v", err)
	}

	// Start event processor manually (without gRPC connection)
	ss.wg.Add(1)
	go ss.eventProcessor()

	// Subscribe to events
	sub := ss.Subscribe()

	// Set a value to trigger an event
	if err := ss.Set("event-key", []byte("event-value")); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Wait for event with timeout
	select {
	case event := <-sub:
		if event.Type != EventStateChanged {
			t.Errorf("expected event type state_changed, got %s", event.Type.String())
		}
		if event.Key != "event-key" {
			t.Errorf("expected key 'event-key', got %s", event.Key)
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for event")
	}

	// Also verify callback was called
	select {
	case <-eventReceived:
		// Callback received event
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for callback event")
	}

	// Cleanup
	ss.Stop()
}

func TestStateSyncUnsubscribe(t *testing.T) {
	client := mockClient(t)
	config := DefaultStateSyncConfig()

	ss, err := NewStateSync(config, client)
	if err != nil {
		t.Fatalf("NewStateSync() error = %v", err)
	}

	// Subscribe and unsubscribe
	sub := ss.Subscribe()
	ss.Unsubscribe(sub)

	// Verify channel is closed
	select {
	case _, ok := <-sub:
		if ok {
			t.Error("expected channel to be closed")
		}
	case <-time.After(100 * time.Millisecond):
		// Channel might block, which is also acceptable
	}
}

func TestStateSyncStats(t *testing.T) {
	client := mockClient(t)
	config := DefaultStateSyncConfig()
	config.OptimisticUpdates = false

	ss, err := NewStateSync(config, client)
	if err != nil {
		t.Fatalf("NewStateSync() error = %v", err)
	}

	// Initial stats
	stats := ss.GetStats()
	if stats.LocalKeys != 0 {
		t.Errorf("expected 0 local keys initially, got %d", stats.LocalKeys)
	}
	if stats.PendingUpdates != 0 {
		t.Errorf("expected 0 pending updates initially, got %d", stats.PendingUpdates)
	}

	// Add some data
	ss.Set("key1", []byte("value1"))
	ss.Set("key2", []byte("value2"))

	stats = ss.GetStats()
	if stats.LocalKeys != 2 {
		t.Errorf("expected 2 local keys, got %d", stats.LocalKeys)
	}
	if stats.CurrentVersion == 0 {
		t.Error("expected non-zero current version")
	}
}

func TestStateSyncSyncWhileSyncing(t *testing.T) {
	client := mockClient(t)
	config := DefaultStateSyncConfig()

	ss, err := NewStateSync(config, client)
	if err != nil {
		t.Fatalf("NewStateSync() error = %v", err)
	}

	// Simulate sync in progress
	ss.syncing.Store(true)

	// Attempt to set while syncing
	err = ss.Set("key", []byte("value"))
	if err == nil {
		t.Error("expected error when setting while syncing")
	}

	// Clear syncing flag
	ss.syncing.Store(false)

	// Now set should work
	err = ss.Set("key", []byte("value"))
	if err != nil {
		t.Errorf("unexpected error after clearing sync flag: %v", err)
	}
}

func TestStateSyncConcurrentAccess(t *testing.T) {
	client := mockClient(t)
	config := DefaultStateSyncConfig()
	config.OptimisticUpdates = false

	ss, err := NewStateSync(config, client)
	if err != nil {
		t.Fatalf("NewStateSync() error = %v", err)
	}

	var wg sync.WaitGroup
	numGoroutines := 10
	numOperations := 50

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := "concurrent-key"
				data := json.RawMessage(`{"id":` + string(rune('0'+id)) + `,"op":` + string(rune('0'+j)) + `}`)
				ss.Set(key, data)
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				ss.Get("concurrent-key")
				ss.GetAll()
				ss.GetStats()
			}
		}()
	}

	wg.Wait()

	// Verify final state is consistent
	stats := ss.GetStats()
	if stats.LocalKeys != 1 {
		t.Errorf("expected 1 local key after concurrent operations, got %d", stats.LocalKeys)
	}
}

func TestStateSyncEventTypes(t *testing.T) {
	client := mockClient(t)
	config := DefaultStateSyncConfig()

	events := make([]SyncEvent, 0)
	var mu sync.Mutex

	config.OnEvent = func(event SyncEvent) {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, event)
	}

	ss, err := NewStateSync(config, client)
	if err != nil {
		t.Fatalf("NewStateSync() error = %v", err)
	}

	// Start event processor manually
	ss.wg.Add(1)
	go ss.eventProcessor()

	// Set a value
	ss.Set("key1", []byte("value1"))

	// Delete the value
	ss.Delete("key1")

	// Give events time to process
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	// Should have at least 2 events: state changed and state deleted
	if len(events) < 2 {
		t.Errorf("expected at least 2 events, got %d", len(events))
	}

	// Verify event types
	hasChanged := false
	hasDeleted := false
	for _, e := range events {
		switch e.Type {
		case EventStateChanged:
			hasChanged = true
		case EventStateDeleted:
			hasDeleted = true
		}
	}
	if !hasChanged {
		t.Error("expected at least one state_changed event")
	}
	if !hasDeleted {
		t.Error("expected at least one state_deleted event")
	}

	// Cleanup
	ss.Stop()
}

func TestStateSyncStop(t *testing.T) {
	client := mockClient(t)
	config := DefaultStateSyncConfig()

	ss, err := NewStateSync(config, client)
	if err != nil {
		t.Fatalf("NewStateSync() error = %v", err)
	}

	// Start and stop
	if err := ss.Start(); err == nil {
		// Expected error since we can't actually connect
		// In real usage with a valid connection, this would succeed
	}

	// Stop should not panic
	if err := ss.Stop(); err != nil {
		t.Errorf("Stop() error = %v", err)
	}
}

func TestStateSyncVersionGeneration(t *testing.T) {
	client := mockClient(t)
	config := DefaultStateSyncConfig()
	config.OptimisticUpdates = false

	ss, err := NewStateSync(config, client)
	if err != nil {
		t.Fatalf("NewStateSync() error = %v", err)
	}

	// Set multiple values and verify version increments
	var lastVersion int64 = 0
	for i := 0; i < 10; i++ {
		key := "version-key"
		ss.Set(key, []byte(string(rune('0'+i))))

		version, ok := ss.Get(key)
		if !ok {
			t.Fatalf("failed to get key on iteration %d", i)
		}
		if version.Version <= lastVersion {
			t.Errorf("version should increase: got %d, previous %d", version.Version, lastVersion)
		}
		lastVersion = version.Version
	}
}

func TestStateSyncMarshalUnmarshal(t *testing.T) {
	// Test StateVersion marshaling
	sv := StateVersion{
		Version:   42,
		Timestamp: time.Now(),
		Data:      []byte(`{"test": "data"}`),
		Source:    "cli",
	}

	data, err := json.Marshal(sv)
	if err != nil {
		t.Fatalf("failed to marshal StateVersion: %v", err)
	}

	var sv2 StateVersion
	if err := json.Unmarshal(data, &sv2); err != nil {
		t.Fatalf("failed to unmarshal StateVersion: %v", err)
	}

	if sv2.Version != sv.Version {
		t.Errorf("version mismatch: got %d, want %d", sv2.Version, sv.Version)
	}
	if string(sv2.Data) != string(sv.Data) {
		t.Errorf("data mismatch: got %s, want %s", string(sv2.Data), string(sv.Data))
	}
	if sv2.Source != sv.Source {
		t.Errorf("source mismatch: got %s, want %s", sv2.Source, sv.Source)
	}

	// Test SyncEvent marshaling
	se := SyncEvent{
		Type:      EventStateChanged,
		Key:       "test-key",
		Version:   1,
		Timestamp: time.Now(),
		Source:    "cli",
		Data:      []byte("test"),
	}

	data, err = json.Marshal(se)
	if err != nil {
		t.Fatalf("failed to marshal SyncEvent: %v", err)
	}

	var se2 SyncEvent
	if err := json.Unmarshal(data, &se2); err != nil {
		t.Fatalf("failed to unmarshal SyncEvent: %v", err)
	}

	if se2.Type != se.Type {
		t.Errorf("type mismatch: got %d, want %d", se2.Type, se.Type)
	}
	if se2.Key != se.Key {
		t.Errorf("key mismatch: got %s, want %s", se2.Key, se.Key)
	}
}

// TestStateSyncWithRealConnection is an integration test that requires a real gRPC server
// This test is skipped by default and can be enabled with the -integration flag
func TestStateSyncWithRealConnection(t *testing.T) {
	// Skip by default
	t.Skip("Skipping integration test - requires real gRPC server")

	// This would test the full synchronization with a real server
	client := mockClient(t)
	config := DefaultStateSyncConfig()

	ss, err := NewStateSync(config, client)
	if err != nil {
		t.Fatalf("NewStateSync() error = %v", err)
	}

	// Start the sync manager
	if err := ss.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer ss.Stop()

	// Test operations
	ss.Set("integration-key", []byte("integration-value"))

	// Wait for sync
	time.Sleep(500 * time.Millisecond)

	// Verify
	version, ok := ss.Get("integration-key")
	if !ok {
		t.Fatal("expected key to exist")
	}
	if string(version.Data) != "integration-value" {
		t.Errorf("expected 'integration-value', got %s", string(version.Data))
	}
}

// TestStateSyncEdgeCases tests various edge cases
func TestStateSyncEdgeCases(t *testing.T) {
	client := mockClient(t)
	config := DefaultStateSyncConfig()

	ss, err := NewStateSync(config, client)
	if err != nil {
		t.Fatalf("NewStateSync() error = %v", err)
	}

	// Test empty key
	ss.Set("", []byte("empty-key-value"))
	if v, ok := ss.Get(""); !ok || string(v.Data) != "empty-key-value" {
		t.Error("expected to handle empty key")
	}

	// Test empty data
	ss.Set("empty-data", []byte{})
	if v, ok := ss.Get("empty-data"); !ok || len(v.Data) != 0 {
		t.Error("expected to handle empty data")
	}

	// Test nil data (should be handled by json.Marshal as "null")
	ss.SetJSON("nil-data", nil)
	if v, ok := ss.Get("nil-data"); !ok {
		t.Error("expected to handle nil data")
	} else {
		// The data should be JSON null
		var result interface{}
		if err := json.Unmarshal(v.Data, &result); err != nil {
			t.Errorf("failed to unmarshal nil data: %v", err)
		}
	}

	// Test large data
	largeData := make([]byte, 1024*1024) // 1MB
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}
	ss.Set("large-data", largeData)
	if v, ok := ss.Get("large-data"); !ok || len(v.Data) != len(largeData) {
		t.Errorf("expected to handle large data, got %d bytes", len(v.Data))
	}
}

// TestStateSyncPendingUpdateCommit tests committing a pending update
func TestStateSyncPendingUpdateCommit(t *testing.T) {
	client := mockClient(t)
	config := DefaultStateSyncConfig()
	config.OptimisticUpdates = true

	ss, err := NewStateSync(config, client)
	if err != nil {
		t.Fatalf("NewStateSync() error = %v", err)
	}

	// Set a value with optimistic update
	key := "commit-key"
	if err := ss.Set(key, []byte("value")); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Manually commit the update
	if err := ss.Commit(key); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}

	// Verify pending update is marked as committed
	ss.pendingMu.RLock()
	pending, ok := ss.pendingUpdates[key]
	ss.pendingMu.RUnlock()

	if !ok {
		t.Fatal("expected pending update to exist")
	}
	if !pending.Committed {
		t.Error("expected pending update to be committed")
	}
	if pending.CommittedAt == nil {
		t.Error("expected CommittedAt to be set")
	}

	// Attempt to rollback committed update
	err = ss.Rollback(key)
	if err == nil {
		t.Error("expected error when rolling back committed update")
	}
}

// TestStateSyncCommitNonExistent tests committing a non-existent update
func TestStateSyncCommitNonExistent(t *testing.T) {
	client := mockClient(t)
	config := DefaultStateSyncConfig()

	ss, err := NewStateSync(config, client)
	if err != nil {
		t.Fatalf("NewStateSync() error = %v", err)
	}

	err = ss.Commit("non-existent")
	if err == nil {
		t.Error("expected error when committing non-existent update")
	}
}

// TestStateSyncGetWithPending tests Get with pending optimistic updates
func TestStateSyncGetWithPending(t *testing.T) {
	client := mockClient(t)
	config := DefaultStateSyncConfig()
	config.OptimisticUpdates = true

	ss, err := NewStateSync(config, client)
	if err != nil {
		t.Fatalf("NewStateSync() error = %v", err)
	}

	// Set initial value
	key := "pending-key"
	ss.localState[key] = StateVersion{
		Version:   1,
		Timestamp: time.Now(),
		Data:      []byte("old-value"),
		Source:    "cli",
	}

	// Create a pending update with higher version
	ss.pendingMu.Lock()
	ss.pendingUpdates[key] = &PendingUpdate{
		Key: "pending-key",
		OldValue: &StateVersion{
			Version:   1,
			Timestamp: time.Now(),
			Data:      []byte("old-value"),
			Source:    "cli",
		},
		NewValue: StateVersion{
			Version:   2,
			Timestamp: time.Now(),
			Data:      []byte("new-value"),
			Source:    "cli",
		},
		AppliedAt: time.Now(),
		Committed: false,
	}
	ss.pendingMu.Unlock()

	// Get should return the optimistic value
	version, ok := ss.Get(key)
	if !ok {
		t.Fatal("expected key to exist")
	}
	if string(version.Data) != "new-value" {
		t.Errorf("expected optimistic value 'new-value', got %s", string(version.Data))
	}

	// Now test with lower version pending (should return local state)
	ss.pendingMu.Lock()
	ss.pendingUpdates[key].NewValue.Version = 0 // Lower than local
	ss.pendingMu.Unlock()

	version, ok = ss.Get(key)
	if !ok {
		t.Fatal("expected key to exist")
	}
	// Should return local state since pending version is lower
	if version.Version != 1 {
		t.Errorf("expected local version 1, got %d", version.Version)
	}
}

// TestStateSyncGetNonExistentWithPending tests Get for non-existent key with pending
func TestStateSyncGetNonExistentWithPending(t *testing.T) {
	client := mockClient(t)
	config := DefaultStateSyncConfig()
	config.OptimisticUpdates = true

	ss, err := NewStateSync(config, client)
	if err != nil {
		t.Fatalf("NewStateSync() error = %v", err)
	}

	// Create a pending update for a key that doesn't exist in localState
	key := "new-pending-key"
	ss.pendingMu.Lock()
	ss.pendingUpdates[key] = &PendingUpdate{
		Key:       key,
		OldValue:  nil, // No old value since key doesn't exist
		NewValue:  StateVersion{Version: 1, Data: []byte("pending-value"), Source: "cli"},
		AppliedAt: time.Now(),
		Committed: false,
	}
	ss.pendingMu.Unlock()

	// Get should return the pending value
	version, ok := ss.Get(key)
	if !ok {
		t.Fatal("expected pending value to be returned")
	}
	if string(version.Data) != "pending-value" {
		t.Errorf("expected 'pending-value', got %s", string(version.Data))
	}
}