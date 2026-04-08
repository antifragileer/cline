// Package testutil provides utilities for test execution.
// This file implements serialization for AI connection tests to prevent
// parallel execution of tests that invoke real LLM commands.
package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

const (
	// LockFileName is the name of the lock file used for AI test serialization
	LockFileName = ".ai_test_lock"
	// LockTimeout is the maximum time to wait for the lock
	LockTimeout = 10 * time.Minute
	// LockRetryInterval is how often to retry acquiring the lock
	LockRetryInterval = 100 * time.Millisecond
)

var (
	// globalLockPath caches the lock file path
	globalLockPath string
	// initLock ensures thread-safe initialization of the lock path
	initLock sync.Once
)

// getLockPath returns the path to the lock file.
// The lock is stored in the system's temp directory to be accessible across test processes.
func getLockPath() string {
	initLock.Do(func() {
		// Use a consistent location in temp directory
		tmpDir := os.TempDir()
		globalLockPath = filepath.Join(tmpDir, "cline_ai_test.lock")
	})
	return globalLockPath
}

// AcquireAILock attempts to acquire a lock for AI connection tests.
// This ensures only one AI test runs at a time across all test processes.
// Returns a release function that must be called when the test completes.
func AcquireAILock(t *testing.T) func() {
	t.Helper()

	lockPath := getLockPath()
	startTime := time.Now()

	// Try to acquire the lock with timeout
	for {
		// Check if we've exceeded the timeout
		if time.Since(startTime) > LockTimeout {
			t.Fatalf("Timeout waiting for AI test lock after %v. Another test may be stuck.", LockTimeout)
		}

		// Try to create the lock file exclusively
		file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err == nil {
			// Successfully acquired the lock
			// Write the test name and PID for debugging
			testInfo := fmt.Sprintf("Test: %s\nPID: %d\nStarted: %s\n",
				t.Name(), os.Getpid(), time.Now().Format(time.RFC3339))
			file.WriteString(testInfo)
			file.Close()

			t.Logf("Acquired AI test lock: %s", lockPath)

			// Return the release function
			return func() {
				releaseLock(t, lockPath)
			}
		}

		// Lock is held by another test, wait and retry
		time.Sleep(LockRetryInterval)
	}
}

// releaseLock releases the AI test lock.
func releaseLock(t *testing.T, lockPath string) {
	t.Helper()

	err := os.Remove(lockPath)
	if err != nil && !os.IsNotExist(err) {
		t.Logf("Warning: failed to remove AI test lock %s: %v", lockPath, err)
	} else {
		t.Logf("Released AI test lock: %s", lockPath)
	}
}

// SkipIfParallelAI tests should call this to skip if they can't acquire the lock immediately.
// This is useful for tests that should fail fast rather than wait.
func SkipIfParallelAI(t *testing.T) {
	t.Helper()

	lockPath := getLockPath()

	// Try to acquire the lock without waiting
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Skip("Skipping AI test: another AI test is currently running")
	}

	// Acquired the lock, close and remove it immediately since we're skipping
	file.Close()
	os.Remove(lockPath)
}

// CleanupStaleLocks removes any stale lock files that may have been left behind
// by crashed tests. This should be called in TestMain or setup functions.
func CleanupStaleLocks() {
	lockPath := getLockPath()

	// Check if lock file exists
	info, err := os.Stat(lockPath)
	if err != nil {
		return // No lock file to clean up
	}

	// If lock is older than the timeout, it's stale
	if time.Since(info.ModTime()) > LockTimeout {
		os.Remove(lockPath)
	}
}
