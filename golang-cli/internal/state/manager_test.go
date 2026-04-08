package state

import (
	"os"
	"testing"
	"time"

	"github.com/cline/cline/golang-cli/internal/storage"
)

func setupTestStorage(t *testing.T) (*storage.StorageContext, func()) {
	t.Helper()

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "state-manager-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	// Create storage context
	storageCtx, err := storage.NewStorageContext(tempDir, "test-workspace")
	if err != nil {
		cleanup()
		t.Fatalf("failed to create storage context: %v", err)
	}

	return storageCtx, cleanup
}

func TestNewStateManager(t *testing.T) {
	storageCtx, cleanup := setupTestStorage(t)
	defer cleanup()

	opts := ManagerOptions{
		Storage:       storageCtx,
		FlushInterval: 50 * time.Millisecond,
	}

	sm := NewStateManager(opts)
	if sm == nil {
		t.Fatal("NewStateManager returned nil")
	}

	if sm.storage != storageCtx {
		t.Error("storage not set correctly")
	}

	if sm.flushInterval != 50*time.Millisecond {
		t.Errorf("expected flush interval 50ms, got %v", sm.flushInterval)
	}
}

func TestStateManager_Load(t *testing.T) {
	storageCtx, cleanup := setupTestStorage(t)
	defer cleanup()

	// Pre-populate storage
	storageCtx.GlobalState.Set("test-key", "test-value")
	storageCtx.WorkspaceState.Set("workspace-key", "workspace-value")

	opts := ManagerOptions{
		Storage:       storageCtx,
		FlushInterval: 50 * time.Millisecond,
	}

	sm := NewStateManager(opts)
	if err := sm.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify global state loaded
	if val, ok := sm.GetGlobalStateKey("test-key"); !ok || val != "test-value" {
		t.Errorf("expected global test-key=test-value, got %v, ok=%v", val, ok)
	}

	// Verify workspace state loaded
	if val, ok := sm.GetWorkspaceStateKey("workspace-key"); !ok || val != "workspace-value" {
		t.Errorf("expected workspace workspace-key=workspace-value, got %v, ok=%v", val, ok)
	}
}

func TestStateManager_GetGlobalStateKey(t *testing.T) {
	storageCtx, cleanup := setupTestStorage(t)
	defer cleanup()

	opts := ManagerOptions{
		Storage:       storageCtx,
		FlushInterval: 50 * time.Millisecond,
	}

	sm := NewStateManager(opts)
	sm.Load()

	// Set a value
	sm.SetGlobalState("key1", "value1")

	// Should be in cache immediately
	if val, ok := sm.GetGlobalStateKey("key1"); !ok || val != "value1" {
		t.Errorf("expected key1=value1, got %v, ok=%v", val, ok)
	}

	// Non-existent key
	if _, ok := sm.GetGlobalStateKey("nonexistent"); ok {
		t.Error("expected nonexistent key to not be found")
	}
}

func TestStateManager_GetWorkspaceStateKey(t *testing.T) {
	storageCtx, cleanup := setupTestStorage(t)
	defer cleanup()

	opts := ManagerOptions{
		Storage:       storageCtx,
		FlushInterval: 50 * time.Millisecond,
	}

	sm := NewStateManager(opts)
	sm.Load()

	// Set a value
	sm.SetWorkspaceState("ws-key", "ws-value")

	// Should be in cache immediately
	if val, ok := sm.GetWorkspaceStateKey("ws-key"); !ok || val != "ws-value" {
		t.Errorf("expected ws-key=ws-value, got %v, ok=%v", val, ok)
	}
}

func TestStateManager_SessionOverrides(t *testing.T) {
	storageCtx, cleanup := setupTestStorage(t)
	defer cleanup()

	opts := ManagerOptions{
		Storage:       storageCtx,
		FlushInterval: 50 * time.Millisecond,
	}

	sm := NewStateManager(opts)
	sm.Load()

	// Set a global value
	sm.SetGlobalState("override-test", "global-value")

	// Set session override (should take precedence)
	sm.SetSessionOverride("override-test", "session-value")

	// Should get session value
	if val, ok := sm.GetGlobalStateKey("override-test"); !ok || val != "session-value" {
		t.Errorf("expected session override session-value, got %v", val)
	}

	// Clear override
	sm.ClearSessionOverride("override-test")

	// Should now get global value
	if val, ok := sm.GetGlobalStateKey("override-test"); !ok || val != "global-value" {
		t.Errorf("expected global value after clearing override, got %v", val)
	}
}

func TestStateManager_SessionOverrides_All(t *testing.T) {
	storageCtx, cleanup := setupTestStorage(t)
	defer cleanup()

	opts := ManagerOptions{
		Storage:       storageCtx,
		FlushInterval: 50 * time.Millisecond,
	}

	sm := NewStateManager(opts)
	sm.Load()

	// Set multiple overrides
	sm.SetSessionOverride("key1", "value1")
	sm.SetSessionOverride("key2", "value2")

	// Clear all
	sm.ClearAllSessionOverrides()

	// Should not find any
	if _, ok := sm.GetSessionOverride("key1"); ok {
		t.Error("expected key1 to be cleared")
	}
	if _, ok := sm.GetSessionOverride("key2"); ok {
		t.Error("expected key2 to be cleared")
	}
}

func TestStateManager_FlushPendingState(t *testing.T) {
	storageCtx, cleanup := setupTestStorage(t)
	defer cleanup()

	opts := ManagerOptions{
		Storage:       storageCtx,
		FlushInterval: 50 * time.Millisecond,
	}

	sm := NewStateManager(opts)
	sm.Load()

	// Set values
	sm.SetGlobalState("flush-test", "flush-value")
	sm.SetWorkspaceState("flush-ws", "flush-ws-value")

	// Force flush
	if err := sm.ForceFlush(); err != nil {
		t.Fatalf("ForceFlush failed: %v", err)
	}

	// Verify values persisted to storage
	if val, ok := storageCtx.GlobalState.Get("flush-test"); !ok || val != "flush-value" {
		t.Errorf("expected global state persisted, got %v, ok=%v", val, ok)
	}

	if val, ok := storageCtx.WorkspaceState.Get("flush-ws"); !ok || val != "flush-ws-value" {
		t.Errorf("expected workspace state persisted, got %v, ok=%v", val, ok)
	}

	// Should not be dirty anymore
	if sm.IsDirty() {
		t.Error("expected IsDirty to be false after flush")
	}

	if sm.PendingWriteCount() != 0 {
		t.Errorf("expected 0 pending writes, got %d", sm.PendingWriteCount())
	}
}

func TestStateManager_AutoFlush(t *testing.T) {
	storageCtx, cleanup := setupTestStorage(t)
	defer cleanup()

	opts := ManagerOptions{
		Storage:       storageCtx,
		FlushInterval: 50 * time.Millisecond,
	}

	sm := NewStateManager(opts)
	sm.Load()

	// Set value
	sm.SetGlobalState("auto-flush", "auto-value")

	// Should be dirty
	if !sm.IsDirty() {
		t.Error("expected IsDirty to be true")
	}

	// Wait for auto-flush
	time.Sleep(150 * time.Millisecond)

	// Should be flushed
	if sm.IsDirty() {
		t.Error("expected IsDirty to be false after auto-flush")
	}

	// Verify persisted
	if val, ok := storageCtx.GlobalState.Get("auto-flush"); !ok || val != "auto-value" {
		t.Errorf("expected value to be persisted after auto-flush, got %v", val)
	}
}

func TestStateManager_Secrets(t *testing.T) {
	storageCtx, cleanup := setupTestStorage(t)
	defer cleanup()

	opts := ManagerOptions{
		Storage:       storageCtx,
		FlushInterval: 50 * time.Millisecond,
	}

	sm := NewStateManager(opts)
	sm.Load()

	// Set secret
	if err := sm.SetSecret("api-key", "secret-value"); err != nil {
		t.Fatalf("SetSecret failed: %v", err)
	}

	// Get secret
	if val, ok := sm.GetSecretKey("api-key"); !ok || val != "secret-value" {
		t.Errorf("expected api-key=secret-value, got %v, ok=%v", val, ok)
	}

	// Delete secret
	if err := sm.DeleteSecret("api-key"); err != nil {
		t.Fatalf("DeleteSecret failed: %v", err)
	}

	// Should not exist
	if _, ok := sm.GetSecretKey("api-key"); ok {
		t.Error("expected secret to be deleted")
	}
}

func TestStateManager_Delete(t *testing.T) {
	storageCtx, cleanup := setupTestStorage(t)
	defer cleanup()

	opts := ManagerOptions{
		Storage:       storageCtx,
		FlushInterval: 50 * time.Millisecond,
	}

	sm := NewStateManager(opts)
	sm.Load()

	// Set and flush
	sm.SetGlobalState("delete-me", "value")
	sm.ForceFlush()

	// Delete
	if err := sm.DeleteGlobalState("delete-me"); err != nil {
		t.Fatalf("DeleteGlobalState failed: %v", err)
	}

	// Should not be in cache
	if _, ok := sm.GetGlobalStateKey("delete-me"); ok {
		t.Error("expected key to be deleted from cache")
	}

	// Should not be in storage
	if _, ok := storageCtx.GlobalState.Get("delete-me"); ok {
		t.Error("expected key to be deleted from storage")
	}
}

func TestStateManager_GetTyped(t *testing.T) {
	storageCtx, cleanup := setupTestStorage(t)
	defer cleanup()

	opts := ManagerOptions{
		Storage:       storageCtx,
		FlushInterval: 50 * time.Millisecond,
	}

	sm := NewStateManager(opts)
	sm.Load()

	// Set a complex value
	type TestStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	sm.SetGlobalState("typed", map[string]interface{}{
		"name":  "test",
		"value": 42,
	})

	// Get typed
	var result TestStruct
	found, err := sm.GetTyped("typed", &result)
	if err != nil {
		t.Fatalf("GetTyped failed: %v", err)
	}
	if !found {
		t.Error("expected to find the key")
	}
	if result.Name != "test" || result.Value != 42 {
		t.Errorf("expected {test, 42}, got %+v", result)
	}

	// Non-existent key
	var empty TestStruct
	found, err = sm.GetTyped("nonexistent", &empty)
	if err != nil {
		t.Fatalf("GetTyped failed for nonexistent: %v", err)
	}
	if found {
		t.Error("expected not to find nonexistent key")
	}
}

func TestStateManager_SetTyped(t *testing.T) {
	storageCtx, cleanup := setupTestStorage(t)
	defer cleanup()

	opts := ManagerOptions{
		Storage:       storageCtx,
		FlushInterval: 50 * time.Millisecond,
	}

	sm := NewStateManager(opts)
	sm.Load()

	// Set typed value
	type TestStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	testStruct := TestStruct{Name: "test", Value: 42}
	if err := sm.SetTyped("set-typed", testStruct); err != nil {
		t.Fatalf("SetTyped failed: %v", err)
	}

	// Flush and verify
	sm.ForceFlush()

	if val, ok := storageCtx.GlobalState.Get("set-typed"); !ok {
		t.Error("expected to find set-typed in storage")
	} else {
		// Verify it's a map (JSON unmarshaled)
		if m, ok := val.(map[string]interface{}); !ok {
			t.Errorf("expected map, got %T", val)
		} else {
			if m["name"] != "test" || m["value"] != float64(42) {
				t.Errorf("unexpected values: %+v", m)
			}
		}
	}
}

func TestStateManager_Close(t *testing.T) {
	storageCtx, cleanup := setupTestStorage(t)
	defer cleanup()

	opts := ManagerOptions{
		Storage:       storageCtx,
		FlushInterval: 50 * time.Millisecond,
	}

	sm := NewStateManager(opts)
	sm.Load()

	// Set value without flushing
	sm.SetGlobalState("close-test", "close-value")

	// Close should flush
	if err := sm.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Verify persisted
	if val, ok := storageCtx.GlobalState.Get("close-test"); !ok || val != "close-value" {
		t.Errorf("expected value to be flushed on close, got %v", val)
	}
}

func TestStateManager_Close_Idempotent(t *testing.T) {
	storageCtx, cleanup := setupTestStorage(t)
	defer cleanup()

	opts := ManagerOptions{
		Storage:       storageCtx,
		FlushInterval: 50 * time.Millisecond,
	}

	sm := NewStateManager(opts)
	sm.Load()

	// Close multiple times should not error
	if err := sm.Close(); err != nil {
		t.Fatalf("First Close failed: %v", err)
	}

	// Second close should succeed (idempotent)
	if err := sm.Close(); err != nil {
		t.Fatalf("Second Close failed: %v", err)
	}
}

func TestStateManager_NewStateManager_DefaultFlushInterval(t *testing.T) {
	storageCtx, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create manager without specifying flush interval
	opts := ManagerOptions{
		Storage: storageCtx,
		// FlushInterval not set, should use default
	}

	sm := NewStateManager(opts)
	if sm == nil {
		t.Fatal("NewStateManager returned nil")
	}

	// Default should be 100ms
	if sm.flushInterval != 100*time.Millisecond {
		t.Errorf("expected default flush interval 100ms, got %v", sm.flushInterval)
	}
}

func TestStateManager_GetGlobalStateKey_WithStorageFallback(t *testing.T) {
	storageCtx, cleanup := setupTestStorage(t)
	defer cleanup()

	// Pre-populate storage directly
	storageCtx.GlobalState.Set("storage-key", "storage-value")

	opts := ManagerOptions{
		Storage:       storageCtx,
		FlushInterval: 50 * time.Millisecond,
	}

	sm := NewStateManager(opts)
	// Don't call Load() - test direct storage fallback

	// Should get value from storage even without Load
	if val, ok := sm.GetGlobalStateKey("storage-key"); !ok || val != "storage-value" {
		t.Errorf("expected storage-key=storage-value from storage fallback, got %v, ok=%v", val, ok)
	}
}

func TestStateManager_GetWorkspaceStateKey_WithStorageFallback(t *testing.T) {
	storageCtx, cleanup := setupTestStorage(t)
	defer cleanup()

	// Pre-populate storage directly
	storageCtx.WorkspaceState.Set("ws-storage-key", "ws-storage-value")

	opts := ManagerOptions{
		Storage:       storageCtx,
		FlushInterval: 50 * time.Millisecond,
	}

	sm := NewStateManager(opts)
	// Don't call Load() - test direct storage fallback

	// Should get value from storage even without Load
	if val, ok := sm.GetWorkspaceStateKey("ws-storage-key"); !ok || val != "ws-storage-value" {
		t.Errorf("expected ws-storage-key=ws-storage-value from storage fallback, got %v, ok=%v", val, ok)
	}
}

func TestStateManager_GetSecretKey_NoStorage(t *testing.T) {
	// Test when storage is nil
	sm := &StateManager{
		storage: nil,
	}

	// Should return false, not panic
	if _, ok := sm.GetSecretKey("any-key"); ok {
		t.Error("expected false when storage is nil")
	}
}

func TestStateManager_SetSecret_NoStorage(t *testing.T) {
	// Test when storage is nil
	sm := &StateManager{
		storage: &storage.StorageContext{
			Secrets: nil,
		},
	}

	// Should return error, not panic
	err := sm.SetSecret("key", "value")
	if err == nil {
		t.Error("expected error when secrets storage is nil")
	}
}

func TestStateManager_DeleteSecret_NoStorage(t *testing.T) {
	// Test when storage is nil
	sm := &StateManager{
		storage: &storage.StorageContext{
			Secrets: nil,
		},
	}

	// Should not panic
	err := sm.DeleteSecret("key")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestStateManager_GetTyped_UnmarshalError(t *testing.T) {
	storageCtx, cleanup := setupTestStorage(t)
	defer cleanup()

	opts := ManagerOptions{
		Storage:       storageCtx,
		FlushInterval: 50 * time.Millisecond,
	}

	sm := NewStateManager(opts)
	sm.Load()

	// Set a value that can't be unmarshaled into the target type
	sm.SetGlobalState("invalid-typed", map[string]interface{}{
		"complex": make(chan int), // channels can't be marshaled
	})

	// Try to get as typed struct
	type TestStruct struct {
		Name string `json:"name"`
	}
	var result TestStruct
	_, err := sm.GetTyped("invalid-typed", &result)
	if err == nil {
		t.Error("expected error when unmarshaling incompatible type")
	}
}

func TestStateManager_GetTypedFromWorkspace_UnmarshalError(t *testing.T) {
	storageCtx, cleanup := setupTestStorage(t)
	defer cleanup()

	opts := ManagerOptions{
		Storage:       storageCtx,
		FlushInterval: 50 * time.Millisecond,
	}

	sm := NewStateManager(opts)
	sm.Load()

	// Set a value that can't be unmarshaled into the target type
	sm.SetWorkspaceState("invalid-typed-ws", map[string]interface{}{
		"complex": make(chan int), // channels can't be marshaled
	})

	// Try to get as typed struct
	type TestStruct struct {
		Name string `json:"name"`
	}
	var result TestStruct
	_, err := sm.GetTypedFromWorkspace("invalid-typed-ws", &result)
	if err == nil {
		t.Error("expected error when unmarshaling incompatible type")
	}
}

func TestStateManager_GetAll(t *testing.T) {
	storageCtx, cleanup := setupTestStorage(t)
	defer cleanup()

	opts := ManagerOptions{
		Storage:       storageCtx,
		FlushInterval: 50 * time.Millisecond,
	}

	sm := NewStateManager(opts)
	sm.Load()

	sm.SetGlobalState("key1", "value1")
	sm.SetGlobalState("key2", "value2")
	sm.SetWorkspaceState("ws1", "wsvalue1")

	// GetAllGlobalState
	globalAll := sm.GetAllGlobalState()
	if len(globalAll) != 2 {
		t.Errorf("expected 2 global keys, got %d", len(globalAll))
	}
	if globalAll["key1"] != "value1" || globalAll["key2"] != "value2" {
		t.Errorf("unexpected global values: %+v", globalAll)
	}

	// GetAllWorkspaceState
	wsAll := sm.GetAllWorkspaceState()
	if len(wsAll) != 1 {
		t.Errorf("expected 1 workspace key, got %d", len(wsAll))
	}
	if wsAll["ws1"] != "wsvalue1" {
		t.Errorf("unexpected workspace values: %+v", wsAll)
	}
}

func TestStateManager_DeleteWorkspaceState(t *testing.T) {
	storageCtx, cleanup := setupTestStorage(t)
	defer cleanup()

	opts := ManagerOptions{
		Storage:       storageCtx,
		FlushInterval: 50 * time.Millisecond,
	}

	sm := NewStateManager(opts)
	sm.Load()

	// Set workspace state
	sm.SetWorkspaceState("delete-ws-key", "delete-ws-value")

	// Verify it exists
	if val, ok := sm.GetWorkspaceStateKey("delete-ws-key"); !ok || val != "delete-ws-value" {
		t.Fatal("workspace key should exist before deletion")
	}

	// Delete the workspace state
	if err := sm.DeleteWorkspaceState("delete-ws-key"); err != nil {
		t.Fatalf("DeleteWorkspaceState failed: %v", err)
	}

	// Verify removed from cache
	if _, ok := sm.GetWorkspaceStateKey("delete-ws-key"); ok {
		t.Error("expected workspace key to be deleted from cache")
	}

	// Verify removed from storage after flush
	sm.ForceFlush()
	if _, ok := storageCtx.WorkspaceState.Get("delete-ws-key"); ok {
		t.Error("expected workspace key to be deleted from storage")
	}
}

func TestStateManager_GetStorage(t *testing.T) {
	storageCtx, cleanup := setupTestStorage(t)
	defer cleanup()

	opts := ManagerOptions{
		Storage:       storageCtx,
		FlushInterval: 50 * time.Millisecond,
	}

	sm := NewStateManager(opts)
	sm.Load()

	// Get storage
	retrievedStorage := sm.GetStorage()
	if retrievedStorage == nil {
		t.Fatal("GetStorage returned nil")
	}

	// Verify it's the same instance
	if retrievedStorage != storageCtx {
		t.Error("GetStorage did not return the same storage instance")
	}
}

func TestStateManager_GetTypedFromWorkspace(t *testing.T) {
	storageCtx, cleanup := setupTestStorage(t)
	defer cleanup()

	opts := ManagerOptions{
		Storage:       storageCtx,
		FlushInterval: 50 * time.Millisecond,
	}

	sm := NewStateManager(opts)
	sm.Load()

	// Set a typed workspace value
	type TestStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	sm.SetWorkspaceState("typed-ws", map[string]interface{}{
		"name":  "workspace-test",
		"value": 100,
	})

	// Get typed from workspace
	var result TestStruct
	found, err := sm.GetTypedFromWorkspace("typed-ws", &result)
	if err != nil {
		t.Fatalf("GetTypedFromWorkspace failed: %v", err)
	}
	if !found {
		t.Error("expected to find the workspace key")
	}
	if result.Name != "workspace-test" || result.Value != 100 {
		t.Errorf("expected {workspace-test, 100}, got %+v", result)
	}

	// Non-existent key
	var empty TestStruct
	found, err = sm.GetTypedFromWorkspace("nonexistent", &empty)
	if err != nil {
		t.Fatalf("GetTypedFromWorkspace failed for nonexistent: %v", err)
	}
	if found {
		t.Error("expected not to find nonexistent workspace key")
	}
}

func TestStateManager_GetTypedFromWorkspace_SessionOverride(t *testing.T) {
	storageCtx, cleanup := setupTestStorage(t)
	defer cleanup()

	opts := ManagerOptions{
		Storage:       storageCtx,
		FlushInterval: 50 * time.Millisecond,
	}

	sm := NewStateManager(opts)
	sm.Load()

	// Set workspace value
	sm.SetWorkspaceState("override-ws-key", map[string]interface{}{
		"name":  "original",
		"value": 1,
	})

	// Set session override (simulated by setting in sessionOverrides)
	type TestStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	// The session override takes precedence over workspace state
	sm.SetSessionOverride("override-ws-key", map[string]interface{}{
		"name":  "overridden",
		"value": 999,
	})

	var result TestStruct
	found, err := sm.GetTypedFromWorkspace("override-ws-key", &result)
	if err != nil {
		t.Fatalf("GetTypedFromWorkspace failed: %v", err)
	}
	if !found {
		t.Error("expected to find the key")
	}

	// Note: GetTypedFromWorkspace checks sessionOverrides first
	// So we should get the overridden value
	if result.Name != "overridden" || result.Value != 999 {
		t.Errorf("expected overridden value {overridden, 999}, got %+v", result)
	}
}

func BenchmarkStateManager_GetGlobalStateKey(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "bench-*")
	if err != nil {
		b.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storageCtx, err := storage.NewStorageContext(tempDir, "bench-workspace")
	if err != nil {
		b.Fatalf("failed to create storage context: %v", err)
	}

	opts := ManagerOptions{
		Storage:       storageCtx,
		FlushInterval: 50 * time.Millisecond,
	}

	sm := NewStateManager(opts)
	sm.Load()
	sm.SetGlobalState("bench-key", "bench-value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = sm.GetGlobalStateKey("bench-key")
	}
}

func BenchmarkStateManager_SetGlobalState(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "bench-*")
	if err != nil {
		b.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storageCtx, err := storage.NewStorageContext(tempDir, "bench-workspace")
	if err != nil {
		b.Fatalf("failed to create storage context: %v", err)
	}

	opts := ManagerOptions{
		Storage:       storageCtx,
		FlushInterval: 1 * time.Second, // Long interval to not trigger flush
	}

	sm := NewStateManager(opts)
	sm.Load()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sm.SetGlobalState("bench-key", i)
	}
}

func TestStateManager_SetTyped_NonSerializable(t *testing.T) {
	storageCtx, cleanup := setupTestStorage(t)
	defer cleanup()

	opts := ManagerOptions{
		Storage:       storageCtx,
		FlushInterval: 50 * time.Millisecond,
	}

	sm := NewStateManager(opts)
	sm.Load()

	// Try to set a non-JSON-serializable value (channel)
	nonSerializable := make(chan int)
	err := sm.SetTyped("invalid", nonSerializable)
	if err == nil {
		t.Error("expected error for non-serializable value")
	}
}
