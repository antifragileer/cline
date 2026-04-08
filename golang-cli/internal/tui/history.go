// Package tui provides terminal UI components for user input handling.
package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// History manages a circular buffer of input history entries.
type History struct {
	entries  []string
	maxSize  int
	position int // -1 means not navigating history
	mu       sync.RWMutex
	filePath string
}

// NewHistory creates a new history with the specified maximum size.
func NewHistory(maxSize int) *History {
	return &History{
		entries:  make([]string, 0, maxSize),
		maxSize:  maxSize,
		position: -1,
	}
}

// NewHistoryWithFile creates a new history that persists to a file.
func NewHistoryWithFile(maxSize int, filePath string) (*History, error) {
	h := NewHistory(maxSize)
	h.filePath = filePath

	// Load existing history if file exists
	if _, err := os.Stat(filePath); err == nil {
		if err := h.Load(); err != nil {
			return nil, fmt.Errorf("failed to load history: %w", err)
		}
	}

	return h, nil
}

// Add adds a new entry to the history.
// If the entry is empty or equals the last entry, it is not added.
func (h *History) Add(entry string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Don't add empty entries
	if entry == "" {
		return
	}

	// Don't add duplicate of the last entry
	if len(h.entries) > 0 && h.entries[len(h.entries)-1] == entry {
		h.ResetPosition()
		return
	}

	// Add the entry
	h.entries = append(h.entries, entry)

	// Remove oldest entry if we exceed max size
	if len(h.entries) > h.maxSize {
		h.entries = h.entries[1:]
	}

	h.ResetPosition()

	// Persist to file if configured
	if h.filePath != "" {
		h.save()
	}
}

// Previous returns the previous history entry.
// Returns empty string if there is no previous entry.
func (h *History) Previous() string {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.entries) == 0 {
		return ""
	}

	// Move position back in history (towards older entries)
	if h.position < len(h.entries)-1 {
		h.position++
	}

	// Return entry at current position (from end)
	idx := len(h.entries) - 1 - h.position
	if idx < 0 || idx >= len(h.entries) {
		return ""
	}

	return h.entries[idx]
}

// Next returns the next history entry (newer).
// Returns empty string if at the end of history.
func (h *History) Next() string {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.entries) == 0 || h.position <= 0 {
		h.position = -1
		return ""
	}

	// Move position forward in history (towards newer entries)
	h.position--

	// Return entry at current position
	idx := len(h.entries) - 1 - h.position
	if idx < 0 || idx >= len(h.entries) {
		return ""
	}

	return h.entries[idx]
}

// ResetPosition resets the navigation position to the end of history.
func (h *History) ResetPosition() {
	h.position = -1
}

// Get returns the entry at the specified index.
// Index 0 is the oldest entry.
func (h *History) Get(index int) (string, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if index < 0 || index >= len(h.entries) {
		return "", false
	}

	return h.entries[index], true
}

// GetAll returns all history entries as a slice.
// The slice is ordered from oldest to newest.
func (h *History) GetAll() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.getAllLocked()
}

// getAllLocked returns all entries (must hold read lock).
func (h *History) getAllLocked() []string {
	// Return a copy to prevent external modification
	result := make([]string, len(h.entries))
	copy(result, h.entries)
	return result
}

// Len returns the number of entries in the history.
func (h *History) Len() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.entries)
}

// MaxSize returns the maximum number of entries allowed.
func (h *History) MaxSize() int {
	return h.maxSize
}

// SetMaxSize changes the maximum size of the history.
// If the new size is smaller than the current number of entries,
// the oldest entries are removed.
func (h *History) SetMaxSize(maxSize int) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.maxSize = maxSize

	// Remove oldest entries if we exceed the new max size
	if len(h.entries) > h.maxSize {
		h.entries = h.entries[len(h.entries)-h.maxSize:]
		h.ResetPosition()
	}
}

// Clear removes all entries from the history.
func (h *History) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.entries = h.entries[:0]
	h.ResetPosition()

	// Remove file if configured
	if h.filePath != "" {
		os.Remove(h.filePath)
	}
}

// Search searches the history for entries containing the given query.
// Returns a slice of matching entries (oldest first).
func (h *History) Search(query string) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if query == "" {
		return h.getAllLocked()
	}

	var matches []string
	for _, entry := range h.entries {
		if contains(entry, query) {
			matches = append(matches, entry)
		}
	}

	return matches
}

// Contains checks if the history contains the given entry.
func (h *History) Contains(entry string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, e := range h.entries {
		if e == entry {
			return true
		}
	}

	return false
}

// Remove removes the entry at the specified index.
func (h *History) Remove(index int) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	if index < 0 || index >= len(h.entries) {
		return false
	}

	h.entries = append(h.entries[:index], h.entries[index+1:]...)
	h.ResetPosition()

	// Persist to file if configured
	if h.filePath != "" {
		h.save()
	}

	return true
}

// Load loads history entries from the configured file.
func (h *History) Load() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.filePath == "" {
		return fmt.Errorf("no file path configured")
	}

	data, err := os.ReadFile(h.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist yet, not an error
			return nil
		}
		return fmt.Errorf("failed to read history file: %w", err)
	}

	if len(data) == 0 {
		return nil
	}

	var entries []string
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("failed to unmarshal history: %w", err)
	}

	// Validate and limit entries
	if len(entries) > h.maxSize {
		entries = entries[len(entries)-h.maxSize:]
	}

	h.entries = entries
	h.ResetPosition()

	return nil
}

// Save saves history entries to the configured file.
func (h *History) Save() error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.save()
}

// save saves history entries to the configured file (must hold lock).
func (h *History) save() error {
	if h.filePath == "" {
		return nil
	}

	// Ensure directory exists
	dir := filepath.Dir(h.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal entries to JSON
	data, err := json.MarshalIndent(h.entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal history: %w", err)
	}

	// Write to file
	if err := os.WriteFile(h.filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write history file: %w", err)
	}

	return nil
}

// GetFilePath returns the configured file path for persistence.
func (h *History) GetFilePath() string {
	return h.filePath
}

// SetFilePath sets the file path for persistence.
func (h *History) SetFilePath(filePath string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.filePath = filePath
}

// Position returns the current navigation position.
// -1 means at the end (default), 0 means at the newest entry.
func (h *History) Position() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.position
}

// contains performs a case-insensitive substring search.
func contains(s, substr string) bool {
	if substr == "" {
		return true
	}
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(findSubstring(s, substr) >= 0)))
}

// findSubstring finds the first occurrence of substr in s.
func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
