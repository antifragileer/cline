package state

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAtomicFileWriter(t *testing.T) {
	writer := NewAtomicFileWriter()
	assert.NotNil(t, writer)
}

func TestAtomicFileWriter_WriteFile(t *testing.T) {
	writer := NewAtomicFileWriter()
	require.NotNil(t, writer)

	t.Run("successful write", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "test.txt")

		data := []byte("hello world")
		err := writer.WriteFile(testFile, data, 0644)
		require.NoError(t, err)

		// Verify file exists
		_, err = os.Stat(testFile)
		assert.NoError(t, err)

		// Verify content
		content, err := os.ReadFile(testFile)
		require.NoError(t, err)
		assert.Equal(t, data, content)

		// Verify permissions
		info, err := os.Stat(testFile)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0644), info.Mode().Perm())
	})

	t.Run("fails when directory cannot be created", func(t *testing.T) {
		// Try to write to a path where we can't create the directory
		invalidPath := "/nonexistent_dir_that_cannot_be_created/test.txt"
		err := writer.WriteFile(invalidPath, []byte("data"), 0644)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create directory")
	})

	t.Run("creates nested directories", func(t *testing.T) {
		tempDir := t.TempDir()
		nestedFile := filepath.Join(tempDir, "a", "b", "c", "test.txt")

		data := []byte("nested content")
		err := writer.WriteFile(nestedFile, data, 0755)
		require.NoError(t, err)

		content, err := os.ReadFile(nestedFile)
		require.NoError(t, err)
		assert.Equal(t, data, content)
	})

	t.Run("overwrites existing file", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "test.txt")

		// Write initial content
		err := os.WriteFile(testFile, []byte("old content"), 0644)
		require.NoError(t, err)

		// Overwrite with new content
		newData := []byte("new content")
		err = writer.WriteFile(testFile, newData, 0644)
		require.NoError(t, err)

		content, err := os.ReadFile(testFile)
		require.NoError(t, err)
		assert.Equal(t, newData, content)
	})

	t.Run("empty data", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "empty.txt")

		err := writer.WriteFile(testFile, []byte{}, 0644)
		require.NoError(t, err)

		content, err := os.ReadFile(testFile)
		require.NoError(t, err)
		assert.Empty(t, content)
	})

	t.Run("large data", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "large.txt")

		// Create 1MB of data
		data := make([]byte, 1024*1024)
		for i := range data {
			data[i] = byte(i % 256)
		}

		err := writer.WriteFile(testFile, data, 0644)
		require.NoError(t, err)

		content, err := os.ReadFile(testFile)
		require.NoError(t, err)
		assert.Equal(t, data, content)
	})
}

func TestAtomicFileWriter_CopyFile(t *testing.T) {
	writer := NewAtomicFileWriter()
	require.NotNil(t, writer)

	t.Run("successful copy", func(t *testing.T) {
		tempDir := t.TempDir()
		srcFile := filepath.Join(tempDir, "source.txt")
		dstFile := filepath.Join(tempDir, "dest.txt")

		// Create source file
		data := []byte("source content")
		err := os.WriteFile(srcFile, data, 0644)
		require.NoError(t, err)

		// Copy file
		err = writer.CopyFile(srcFile, dstFile, 0755)
		require.NoError(t, err)

		// Verify destination exists
		_, err = os.Stat(dstFile)
		assert.NoError(t, err)

		// Verify content
		content, err := os.ReadFile(dstFile)
		require.NoError(t, err)
		assert.Equal(t, data, content)

		// Verify permissions
		info, err := os.Stat(dstFile)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0755), info.Mode().Perm())
	})

	t.Run("copy to nested directory", func(t *testing.T) {
		tempDir := t.TempDir()
		srcFile := filepath.Join(tempDir, "source.txt")
		dstFile := filepath.Join(tempDir, "nested", "dest.txt")

		// Create source file
		data := []byte("nested copy content")
		err := os.WriteFile(srcFile, data, 0644)
		require.NoError(t, err)

		// Copy file
		err = writer.CopyFile(srcFile, dstFile, 0644)
		require.NoError(t, err)

		content, err := os.ReadFile(dstFile)
		require.NoError(t, err)
		assert.Equal(t, data, content)
	})

	t.Run("copy non-existent source", func(t *testing.T) {
		tempDir := t.TempDir()
		srcFile := filepath.Join(tempDir, "nonexistent.txt")
		dstFile := filepath.Join(tempDir, "dest.txt")

		err := writer.CopyFile(srcFile, dstFile, 0644)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to open source file")
	})

	t.Run("copy overwrites existing", func(t *testing.T) {
		tempDir := t.TempDir()
		srcFile := filepath.Join(tempDir, "source.txt")
		dstFile := filepath.Join(tempDir, "dest.txt")

		// Create source file
		err := os.WriteFile(srcFile, []byte("new content"), 0644)
		require.NoError(t, err)

		// Create existing destination
		err = os.WriteFile(dstFile, []byte("old content"), 0644)
		require.NoError(t, err)

		// Copy should overwrite
		err = writer.CopyFile(srcFile, dstFile, 0644)
		require.NoError(t, err)

		content, err := os.ReadFile(dstFile)
		require.NoError(t, err)
		assert.Equal(t, "new content", string(content))
	})

	t.Run("fails when destination directory cannot be created", func(t *testing.T) {
		tempDir := t.TempDir()
		srcFile := filepath.Join(tempDir, "source.txt")
		_ = os.WriteFile(srcFile, []byte("data"), 0644)

		// Try to copy to an invalid path
		invalidDst := "/nonexistent_dir_that_cannot_be_created/dest.txt"
		err := writer.CopyFile(srcFile, invalidDst, 0644)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create destination directory")
	})
}

func TestAtomicFileWriter_SafeAppend(t *testing.T) {
	writer := NewAtomicFileWriter()
	require.NotNil(t, writer)

	t.Run("append to new file", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "append.txt")

		data := []byte("first line\n")
		err := writer.SafeAppend(testFile, data, 0644)
		require.NoError(t, err)

		content, err := os.ReadFile(testFile)
		require.NoError(t, err)
		assert.Equal(t, data, content)
	})

	t.Run("append to existing file", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "append.txt")

		// Create initial file
		err := os.WriteFile(testFile, []byte("existing content\n"), 0644)
		require.NoError(t, err)

		// Append more data
		data := []byte("appended content\n")
		err = writer.SafeAppend(testFile, data, 0644)
		require.NoError(t, err)

		content, err := os.ReadFile(testFile)
		require.NoError(t, err)
		assert.Equal(t, "existing content\nappended content\n", string(content))
	})

	t.Run("append creates directories", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "a", "b", "append.txt")

		data := []byte("nested append\n")
		err := writer.SafeAppend(testFile, data, 0644)
		require.NoError(t, err)

		content, err := os.ReadFile(testFile)
		require.NoError(t, err)
		assert.Equal(t, data, content)
	})

	t.Run("multiple appends", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "multi.txt")

		for i := 0; i < 10; i++ {
			data := []byte(string(rune('0' + i)))
			err := writer.SafeAppend(testFile, data, 0644)
			require.NoError(t, err)
		}

		content, err := os.ReadFile(testFile)
		require.NoError(t, err)
		assert.Equal(t, "0123456789", string(content))
	})

	t.Run("concurrent appends", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "concurrent.txt")

		var wg sync.WaitGroup
		numGoroutines := 10
		appendsPerGoroutine := 10

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < appendsPerGoroutine; j++ {
					data := []byte("goroutine\n")
					err := writer.SafeAppend(testFile, data, 0644)
					require.NoError(t, err)
				}
			}(i)
		}

		wg.Wait()

		// Verify file has all appends (100 lines)
		content, err := os.ReadFile(testFile)
		require.NoError(t, err)
		lines := len(content) / 10 // Each "goroutine\n" is 10 bytes
		assert.Equal(t, numGoroutines*appendsPerGoroutine, lines)
	})

	t.Run("fails when directory cannot be created", func(t *testing.T) {
		// Try to append to a path where we can't create the directory
		invalidPath := "/nonexistent_dir_that_cannot_be_created/append.txt"
		err := writer.SafeAppend(invalidPath, []byte("data"), 0644)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create directory")
	})
}

func TestNewFileLock(t *testing.T) {
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "test.lock")

	lock := NewFileLock(lockPath)
	assert.NotNil(t, lock)
	assert.Equal(t, lockPath, lock.path)
	assert.Nil(t, lock.file)
}

func TestFileLock_Lock(t *testing.T) {
	t.Run("lock and unlock", func(t *testing.T) {
		tempDir := t.TempDir()
		lockPath := filepath.Join(tempDir, "test.lock")

		lock := NewFileLock(lockPath)
		require.NotNil(t, lock)

		// Lock should succeed
		err := lock.Lock()
		require.NoError(t, err)
		assert.True(t, lock.IsLocked())

		// Unlock should succeed
		err = lock.Unlock()
		require.NoError(t, err)
		assert.False(t, lock.IsLocked())

		// Lock file should exist
		_, err = os.Stat(lockPath)
		assert.NoError(t, err)
	})

	t.Run("double lock fails", func(t *testing.T) {
		tempDir := t.TempDir()
		lockPath := filepath.Join(tempDir, "test.lock")

		lock := NewFileLock(lockPath)
		require.NotNil(t, lock)

		// First lock
		err := lock.Lock()
		require.NoError(t, err)

		// Second lock should fail
		err = lock.Lock()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already locked")

		// Cleanup
		_ = lock.Unlock()
	})

	t.Run("creates directories", func(t *testing.T) {
		tempDir := t.TempDir()
		lockPath := filepath.Join(tempDir, "a", "b", "test.lock")

		lock := NewFileLock(lockPath)
		require.NotNil(t, lock)

		err := lock.Lock()
		require.NoError(t, err)
		assert.True(t, lock.IsLocked())

		err = lock.Unlock()
		require.NoError(t, err)
	})

	t.Run("unlock without lock fails", func(t *testing.T) {
		tempDir := t.TempDir()
		lockPath := filepath.Join(tempDir, "test.lock")

		lock := NewFileLock(lockPath)
		require.NotNil(t, lock)

		err := lock.Unlock()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not locked")
	})

	t.Run("lock fails when directory cannot be created", func(t *testing.T) {
		// Try to lock a file in an invalid path
		invalidPath := "/nonexistent_dir_that_cannot_be_created/test.lock"
		lock := NewFileLock(invalidPath)

		err := lock.Lock()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create directory")
	})

	t.Run("try lock fails when directory cannot be created", func(t *testing.T) {
		// Try to lock a file in an invalid path
		invalidPath := "/nonexistent_dir_that_cannot_be_created/test.lock"
		lock := NewFileLock(invalidPath)

		err := lock.TryLock()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create directory")
	})
}

func TestFileLock_TryLock(t *testing.T) {
	t.Run("successful try lock", func(t *testing.T) {
		tempDir := t.TempDir()
		lockPath := filepath.Join(tempDir, "test.lock")

		lock := NewFileLock(lockPath)
		require.NotNil(t, lock)

		err := lock.TryLock()
		require.NoError(t, err)
		assert.True(t, lock.IsLocked())

		err = lock.Unlock()
		require.NoError(t, err)
	})

	t.Run("try lock when already locked fails", func(t *testing.T) {
		tempDir := t.TempDir()
		lockPath := filepath.Join(tempDir, "test.lock")

		lock := NewFileLock(lockPath)
		require.NotNil(t, lock)

		// First lock
		err := lock.TryLock()
		require.NoError(t, err)

		// Second try lock should fail
		err = lock.TryLock()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already locked")

		// Cleanup
		_ = lock.Unlock()
	})
}

func TestFileLock_ConcurrentAccess(t *testing.T) {
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "concurrent.lock")

	// Use a channel to track successful locks
	lockAcquired := make(chan bool, 100)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			lock := NewFileLock(lockPath)
			err := lock.TryLock()
			if err == nil {
				lockAcquired <- true
				time.Sleep(10 * time.Millisecond) // Hold lock briefly
				_ = lock.Unlock()
			} else {
				lockAcquired <- false
			}
		}()
	}

	wg.Wait()
	close(lockAcquired)

	// Count successful locks
	successCount := 0
	for acquired := range lockAcquired {
		if acquired {
			successCount++
		}
	}

	// Only one goroutine should have acquired the lock
	assert.Equal(t, 1, successCount)
}

func TestFileLock_IsLocked(t *testing.T) {
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "test.lock")

	lock := NewFileLock(lockPath)
	require.NotNil(t, lock)

	// Initially not locked
	assert.False(t, lock.IsLocked())

	// Lock it
	err := lock.Lock()
	require.NoError(t, err)
	assert.True(t, lock.IsLocked())

	// Unlock it
	err = lock.Unlock()
	require.NoError(t, err)
	assert.False(t, lock.IsLocked())
}

func TestAtomicFileWriter_ConcurrentWrites(t *testing.T) {
	writer := NewAtomicFileWriter()
	require.NotNil(t, writer)

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "concurrent.txt")

	var wg sync.WaitGroup
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			data := []byte(string(rune('A' + id)))
			err := writer.WriteFile(testFile, data, 0644)
			require.NoError(t, err)
		}(i)
	}

	wg.Wait()

	// File should exist and contain one of the written values
	content, err := os.ReadFile(testFile)
	require.NoError(t, err)
	assert.Len(t, content, 1)
	assert.True(t, content[0] >= 'A' && content[0] < 'A'+byte(numGoroutines))
}

func TestAtomicFileWriter_ThreadSafety(t *testing.T) {
	writer := NewAtomicFileWriter()
	require.NotNil(t, writer)

	tempDir := t.TempDir()

	var wg sync.WaitGroup
	numGoroutines := 20

	// Mix of operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			switch id % 3 {
			case 0:
				// Write file
				testFile := filepath.Join(tempDir, "write.txt")
				data := []byte("write content")
				err := writer.WriteFile(testFile, data, 0644)
				require.NoError(t, err)
			case 1:
				// Copy file
				srcFile := filepath.Join(tempDir, "source.txt")
				dstFile := filepath.Join(tempDir, "dest.txt")
				_ = os.WriteFile(srcFile, []byte("source"), 0644)
				err := writer.CopyFile(srcFile, dstFile, 0644)
				require.NoError(t, err)
			case 2:
				// Append file
				testFile := filepath.Join(tempDir, "append.txt")
				data := []byte("append\n")
				err := writer.SafeAppend(testFile, data, 0644)
				require.NoError(t, err)
			}
		}(i)
	}

	wg.Wait()

	// All files should exist
	_, err := os.Stat(filepath.Join(tempDir, "write.txt"))
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(tempDir, "dest.txt"))
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(tempDir, "append.txt"))
	assert.NoError(t, err)
}

func TestFileLock_Unlock_FileRemoved(t *testing.T) {
	t.Run("unlock after lock file removed", func(t *testing.T) {
		tempDir := t.TempDir()
		lockPath := filepath.Join(tempDir, "test.lock")

		lock := NewFileLock(lockPath)
		require.NotNil(t, lock)

		// Lock
		err := lock.Lock()
		require.NoError(t, err)

		// Remove the lock file while still locked
		err = os.Remove(lockPath)
		require.NoError(t, err)

		// Unlock should handle this gracefully
		// The releaseLock may fail since file is gone
		err = lock.Unlock()
		// This may or may not error depending on platform
		// but should not panic
		_ = err
	})
}
