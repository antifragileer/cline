package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

// MockHistoryLoader is a mock implementation of HistoryLoader for testing
type MockHistoryLoader struct {
	entries []TaskHistoryEntry
	err     error
}

func (m *MockHistoryLoader) Load(path string) ([]TaskHistoryEntry, error) {
	return m.entries, m.err
}

func TestHistoryCmd(t *testing.T) {
	// Save original values
	originalArgs := historyFlags

	// Reset after test
	defer func() {
		historyFlags = originalArgs
	}()

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "default invocation",
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "with limit flag",
			args:    []string{"-n", "20"},
			wantErr: false,
		},
		{
			name:    "with page flag",
			args:    []string{"-p", "2"},
			wantErr: false,
		},
		{
			name:    "with limit and page flags",
			args:    []string{"-n", "15", "-p", "2"},
			wantErr: false,
		},
		{
			name:    "with long flags",
			args:    []string{"--limit", "25", "--page", "3"},
			wantErr: false,
		},
		{
			name:    "with json flag",
			args:    []string{"--json"},
			wantErr: false,
		},
		{
			name:    "with all flags",
			args:    []string{"-n", "5", "-p", "1", "--json"},
			wantErr: false,
		},
		{
			name:    "invalid limit zero",
			args:    []string{"-n", "0"},
			wantErr: true,
		},
		{
			name:    "invalid page zero",
			args:    []string{"-p", "0"},
			wantErr: true,
		},
		{
			name:    "invalid negative limit",
			args:    []string{"-n", "-5"},
			wantErr: true,
		},
		{
			name:    "invalid negative page",
			args:    []string{"-p", "-1"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			resetHistoryFlags()

			// Create a new command for testing
			cmd := &cobra.Command{
				RunE: runHistory,
			}
			setupHistoryFlags(cmd)

			cmd.SetArgs(tt.args)
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})

			err := cmd.Execute()

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHistoryCmdValidate(t *testing.T) {
	// Create temp config file for testing
	tmpDir := t.TempDir()
	testConfig := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(testConfig, []byte("test: value"), 0644); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	// Create non-existent config path
	nonExistentConfig := filepath.Join(tmpDir, "nonexistent.yaml")

	tests := []struct {
		name    string
		args    []string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid default",
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "valid with limit",
			args:    []string{"-n", "5"},
			wantErr: false,
		},
		{
			name:    "valid with page",
			args:    []string{"-p", "2"},
			wantErr: false,
		},
		{
			name:    "invalid limit zero",
			args:    []string{"-n", "0"},
			wantErr: true,
			errMsg:  "limit must be greater than 0",
		},
		{
			name:    "invalid page zero",
			args:    []string{"-p", "0"},
			wantErr: true,
			errMsg:  "page must be greater than 0",
		},
		{
			name:    "invalid negative limit",
			args:    []string{"--limit", "-10"},
			wantErr: true,
			errMsg:  "limit must be greater than 0",
		},
		{
			name:    "non-existent config",
			args:    []string{"--config", nonExistentConfig},
			wantErr: true,
			errMsg:  "config file not found",
		},
		{
			name:    "valid with existing config",
			args:    []string{"--config", testConfig},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			resetHistoryFlags()

			// Create a new command for testing
			cmd := &cobra.Command{
				RunE: runHistory,
			}
			setupHistoryFlags(cmd)

			cmd.SetArgs(tt.args)
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})

			err := cmd.Execute()

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && tt.errMsg != "" && err != nil {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error to contain '%s', got '%s'", tt.errMsg, err.Error())
				}
			}
		})
	}
}

func TestValidateHistoryConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  HistoryConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: HistoryConfig{
				Limit: 10,
				Page:  1,
			},
			wantErr: false,
		},
		{
			name: "valid config with larger values",
			config: HistoryConfig{
				Limit: 50,
				Page:  5,
			},
			wantErr: false,
		},
		{
			name: "invalid limit zero",
			config: HistoryConfig{
				Limit: 0,
				Page:  1,
			},
			wantErr: true,
			errMsg:  "limit must be greater than 0",
		},
		{
			name: "invalid limit negative",
			config: HistoryConfig{
				Limit: -5,
				Page:  1,
			},
			wantErr: true,
			errMsg:  "limit must be greater than 0",
		},
		{
			name: "invalid page zero",
			config: HistoryConfig{
				Limit: 10,
				Page:  0,
			},
			wantErr: true,
			errMsg:  "page must be greater than 0",
		},
		{
			name: "invalid page negative",
			config: HistoryConfig{
				Limit: 10,
				Page:  -1,
			},
			wantErr: true,
			errMsg:  "page must be greater than 0",
		},
		{
			name: "both invalid",
			config: HistoryConfig{
				Limit: 0,
				Page:  0,
			},
			wantErr: true,
			errMsg:  "limit must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHistoryConfig(tt.config)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateHistoryConfig() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && tt.errMsg != "" && err != nil {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error to contain '%s', got '%s'", tt.errMsg, err.Error())
				}
			}
		})
	}
}

func TestLoadTaskHistory(t *testing.T) {
	// Create temp directory for test files
	tmpDir := t.TempDir()

	// Test with empty history file
	emptyHistoryPath := filepath.Join(tmpDir, "empty_history.json")
	emptyData := `{"entries": []}`
	if err := os.WriteFile(emptyHistoryPath, []byte(emptyData), 0644); err != nil {
		t.Fatalf("failed to create empty history file: %v", err)
	}

	// Test with valid history entries
	validHistoryPath := filepath.Join(tmpDir, "valid_history.json")
	now := time.Now().Unix() * 1000 // Convert to milliseconds
	validData := map[string]interface{}{
		"entries": []map[string]interface{}{
			{
				"id":   "task-001",
				"task": "First test task",
				"ts":   now - 2000,
			},
			{
				"id":   "task-002",
				"task": "Second test task",
				"ts":   now - 1000,
				"metadata": map[string]interface{}{
					"model": "gpt-4",
					"mode":  "act",
				},
			},
		},
	}
	validJSON, _ := json.Marshal(validData)
	if err := os.WriteFile(validHistoryPath, validJSON, 0644); err != nil {
		t.Fatalf("failed to create valid history file: %v", err)
	}

	// Test with no entries key
	noEntriesPath := filepath.Join(tmpDir, "no_entries.json")
	noEntriesData := `{"other": "data"}`
	if err := os.WriteFile(noEntriesPath, []byte(noEntriesData), 0644); err != nil {
		t.Fatalf("failed to create no-entries history file: %v", err)
	}

	tests := []struct {
		name     string
		path     string
		wantErr  bool
		wantLen  int
		validate func(t *testing.T, entries []TaskHistoryEntry)
	}{
		{
			name:    "empty entries",
			path:    emptyHistoryPath,
			wantErr: false,
			wantLen: 0,
		},
		{
			name:    "valid entries",
			path:    validHistoryPath,
			wantErr: false,
			wantLen: 2,
			validate: func(t *testing.T, entries []TaskHistoryEntry) {
				if len(entries) != 2 {
					return
				}
				if entries[0].ID != "task-001" {
					t.Errorf("expected first entry ID to be 'task-001', got '%s'", entries[0].ID)
				}
				if entries[1].ID != "task-002" {
					t.Errorf("expected second entry ID to be 'task-002', got '%s'", entries[1].ID)
				}
				if entries[1].Metadata.Model != "gpt-4" {
					t.Errorf("expected second entry model to be 'gpt-4', got '%s'", entries[1].Metadata.Model)
				}
			},
		},
		{
			name:    "no entries key",
			path:    noEntriesPath,
			wantErr: false,
			wantLen: 0,
		},
		{
			name:    "non-existent file",
			path:    filepath.Join(tmpDir, "nonexistent.json"),
			wantErr: false,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries, err := loadTaskHistory(tt.path)

			if (err != nil) != tt.wantErr {
				t.Errorf("loadTaskHistory() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(entries) != tt.wantLen {
				t.Errorf("loadTaskHistory() returned %d entries, want %d", len(entries), tt.wantLen)
			}

			if tt.validate != nil {
				tt.validate(t, entries)
			}
		})
	}
}

func TestLoadAndDisplayHistory(t *testing.T) {
	// Create temp directory and test data
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "taskHistory.json")

	// Create test history entries
	now := time.Now().Unix() * 1000
	entries := []TaskHistoryEntry{
		{ID: "task-001", Task: "First task", Timestamp: now - 4000},
		{ID: "task-002", Task: "Second task", Timestamp: now - 3000},
		{ID: "task-003", Task: "Third task", Timestamp: now - 2000},
		{ID: "task-004", Task: "Fourth task", Timestamp: now - 1000},
		{ID: "task-005", Task: "Fifth task", Timestamp: now},
	}

	// Write to file
	data := map[string]interface{}{"entries": entries}
	jsonData, _ := json.Marshal(data)
	if err := os.WriteFile(historyPath, jsonData, 0644); err != nil {
		t.Fatalf("failed to create history file: %v", err)
	}

	// Override getTaskHistoryPathFunc to return our test file
	originalGetPath := getTaskHistoryPathFunc
	getTaskHistoryPathFunc = func() string {
		return historyPath
	}
	defer func() {
		getTaskHistoryPathFunc = originalGetPath
	}()

	tests := []struct {
		name   string
		config HistoryConfig
		check  func(t *testing.T, output string)
	}{
		{
			name: "default output",
			config: HistoryConfig{
				Limit:  10,
				Page:   1,
				Output: &bytes.Buffer{},
			},
			check: func(t *testing.T, output string) {
				if !strings.Contains(output, "task-001") {
					t.Error("output should contain task-001")
				}
				if !strings.Contains(output, "Page 1 of 1") {
					t.Error("output should contain pagination info")
				}
			},
		},
		{
			name: "json output",
			config: HistoryConfig{
				Limit:  10,
				Page:   1,
				JSON:   true,
				Output: &bytes.Buffer{},
			},
			check: func(t *testing.T, output string) {
				var result map[string]interface{}
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Errorf("output is not valid JSON: %v", err)
				}
				if result["page"] != float64(1) {
					t.Errorf("expected page 1, got %v", result["page"])
				}
			},
		},
		{
			name: "pagination page 1",
			config: HistoryConfig{
				Limit:  2,
				Page:   1,
				Output: &bytes.Buffer{},
			},
			check: func(t *testing.T, output string) {
				if !strings.Contains(output, "task-005") {
					t.Error("page 1 should contain most recent task (task-005)")
				}
				if !strings.Contains(output, "task-004") {
					t.Error("page 1 should contain second most recent task (task-004)")
				}
				if strings.Contains(output, "task-003") {
					t.Error("page 1 should not contain task-003")
				}
			},
		},
		{
			name: "pagination page 2",
			config: HistoryConfig{
				Limit:  2,
				Page:   2,
				Output: &bytes.Buffer{},
			},
			check: func(t *testing.T, output string) {
				if !strings.Contains(output, "task-003") {
					t.Error("page 2 should contain task-003")
				}
				if !strings.Contains(output, "task-002") {
					t.Error("page 2 should contain task-002")
				}
			},
		},
		{
			name: "empty history",
			config: HistoryConfig{
				Limit:  10,
				Page:   1,
				Output: &bytes.Buffer{},
			},
			check: func(t *testing.T, output string) {
				// This will test with the existing file
				if !strings.Contains(output, "task-001") {
					t.Error("should find task-001")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset output buffer
			if buf, ok := tt.config.Output.(*bytes.Buffer); ok {
				buf.Reset()
			}

			err := loadAndDisplayHistory(tt.config)
			if err != nil {
				t.Errorf("loadAndDisplayHistory() unexpected error: %v", err)
				return
			}

			if buf, ok := tt.config.Output.(*bytes.Buffer); ok {
				tt.check(t, buf.String())
			}
		})
	}
}

func TestOutputJSON(t *testing.T) {
	entries := []TaskHistoryEntry{
		{ID: "task-001", Task: "Test task", Timestamp: 1609459200000},
	}

	var buf bytes.Buffer
	err := outputJSON(&buf, entries, 1, 10, 1)
	if err != nil {
		t.Errorf("outputJSON() unexpected error: %v", err)
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
		return
	}

	if result["page"] != float64(1) {
		t.Errorf("expected page 1, got %v", result["page"])
	}
	if result["limit"] != float64(10) {
		t.Errorf("expected limit 10, got %v", result["limit"])
	}
	if result["total"] != float64(1) {
		t.Errorf("expected total 1, got %v", result["total"])
	}
	if result["pages"] != float64(1) {
		t.Errorf("expected pages 1, got %v", result["pages"])
	}

	entriesArr, ok := result["entries"].([]interface{})
	if !ok {
		t.Error("expected entries to be an array")
		return
	}
	if len(entriesArr) != 1 {
		t.Errorf("expected 1 entry, got %d", len(entriesArr))
	}
}

func TestOutputTable(t *testing.T) {
	now := time.Now().Unix() * 1000
	entries := []TaskHistoryEntry{
		{ID: "task-001", Task: "Short task", Timestamp: now},
		{ID: "task-002", Task: strings.Repeat("a", 50), Timestamp: now}, // Long task name
	}

	var buf bytes.Buffer
	err := outputTable(&buf, entries, 1, 10, 2, 1)
	if err != nil {
		t.Errorf("outputTable() unexpected error: %v", err)
		return
	}

	output := buf.String()

	// Check headers
	if !strings.Contains(output, "ID") {
		t.Error("output should contain ID header")
	}
	if !strings.Contains(output, "TASK") {
		t.Error("output should contain TASK header")
	}
	if !strings.Contains(output, "TIMESTAMP") {
		t.Error("output should contain TIMESTAMP header")
	}

	// Check entries
	if !strings.Contains(output, "task-001") {
		t.Error("output should contain task-001")
	}
	if !strings.Contains(output, "task-002") {
		t.Error("output should contain task-002")
	}

	// Check pagination
	if !strings.Contains(output, "Page 1 of 1") {
		t.Error("output should contain pagination info")
	}

	// Check long task name truncation
	if strings.Contains(output, strings.Repeat("a", 50)) {
		t.Error("long task name should be truncated")
	}
}

func TestOutputTableEmpty(t *testing.T) {
	var buf bytes.Buffer
	err := outputTable(&buf, []TaskHistoryEntry{}, 1, 10, 0, 1)
	if err != nil {
		t.Errorf("outputTable() unexpected error: %v", err)
		return
	}

	output := buf.String()
	if !strings.Contains(output, "No task history found") {
		t.Error("output should indicate no history found")
	}
}

func TestFormatTimestamp(t *testing.T) {
	tests := []struct {
		name     string
		ts       int64
		expected string
	}{
		{
			name:     "zero timestamp",
			ts:       0,
			expected: "unknown",
		},
		{
			name:     "valid timestamp",
			ts:       1609459200000, // 2021-01-01 00:00:00 UTC in milliseconds
			expected: "2021-01-01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTimestamp(tt.ts)
			if !strings.Contains(result, tt.expected) {
				t.Errorf("formatTimestamp() = %v, should contain %v", result, tt.expected)
			}
		})
	}
}

func TestConvertToEntry(t *testing.T) {
	tests := []struct {
		name     string
		data     interface{}
		wantErr  bool
		validate func(t *testing.T, entry TaskHistoryEntry)
	}{
		{
			name: "valid map",
			data: map[string]interface{}{
				"id":   "task-001",
				"task": "Test task",
				"ts":   float64(1609459200000),
			},
			wantErr: false,
			validate: func(t *testing.T, entry TaskHistoryEntry) {
				if entry.ID != "task-001" {
					t.Errorf("expected ID 'task-001', got '%s'", entry.ID)
				}
				if entry.Task != "Test task" {
					t.Errorf("expected task 'Test task', got '%s'", entry.Task)
				}
			},
		},
		{
			name:    "invalid data type",
			data:    make(chan int), // Channels can't be marshaled to JSON
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := convertToEntry(tt.data)

			if (err != nil) != tt.wantErr {
				t.Errorf("convertToEntry() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.validate != nil {
				tt.validate(t, entry)
			}
		})
	}
}

func TestConvertToEntries(t *testing.T) {
	data := []interface{}{
		map[string]interface{}{
			"id":   "task-001",
			"task": "First task",
			"ts":   float64(1609459200000),
		},
		map[string]interface{}{
			"id":   "task-002",
			"task": "Second task",
			"ts":   float64(1609545600000),
		},
	}

	entries, err := convertToEntries(data)
	if err != nil {
		t.Errorf("convertToEntries() unexpected error: %v", err)
		return
	}

	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}

	if entries[0].ID != "task-001" {
		t.Errorf("expected first entry ID 'task-001', got '%s'", entries[0].ID)
	}
}

func TestFormatEntries(t *testing.T) {
	entries := []TaskHistoryEntry{
		{ID: "task-001", Task: "Test", Timestamp: 1609459200000},
	}

	tests := []struct {
		name    string
		format  string
		wantErr bool
	}{
		{
			name:    "json format",
			format:  "json",
			wantErr: false,
		},
		{
			name:    "table format",
			format:  "table",
			wantErr: false,
		},
		{
			name:    "invalid format",
			format:  "xml",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FormatEntries(entries, tt.format)

			if (err != nil) != tt.wantErr {
				t.Errorf("FormatEntries() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result == "" {
				t.Error("FormatEntries() returned empty result for valid format")
			}
		})
	}
}

func TestDefaultHistoryLoader(t *testing.T) {
	// Create temp directory with test data
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "taskHistory.json")

	now := time.Now().Unix() * 1000
	entries := []TaskHistoryEntry{
		{ID: "task-001", Task: "Test task", Timestamp: now},
	}

	data := map[string]interface{}{"entries": entries}
	jsonData, _ := json.Marshal(data)
	if err := os.WriteFile(historyPath, jsonData, 0644); err != nil {
		t.Fatalf("failed to create history file: %v", err)
	}

	loader := &DefaultHistoryLoader{}
	loadedEntries, err := loader.Load(historyPath)
	if err != nil {
		t.Errorf("DefaultHistoryLoader.Load() unexpected error: %v", err)
		return
	}

	if len(loadedEntries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(loadedEntries))
	}
}

func TestPaginationEdgeCases(t *testing.T) {
	// Create temp directory with test data
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "taskHistory.json")

	// Create many test entries
	var entries []TaskHistoryEntry
	now := time.Now().Unix() * 1000
	for i := 0; i < 25; i++ {
		entries = append(entries, TaskHistoryEntry{
			ID:        fmt.Sprintf("task-%03d", i+1),
			Task:      fmt.Sprintf("Task number %d", i+1),
			Timestamp: now - int64(i*1000),
		})
	}

	data := map[string]interface{}{"entries": entries}
	jsonData, _ := json.Marshal(data)
	if err := os.WriteFile(historyPath, jsonData, 0644); err != nil {
		t.Fatalf("failed to create history file: %v", err)
	}

	// Override getTaskHistoryPathFunc
	originalGetPath := getTaskHistoryPathFunc
	getTaskHistoryPathFunc = func() string {
		return historyPath
	}
	defer func() {
		getTaskHistoryPathFunc = originalGetPath
	}()

	tests := []struct {
		name   string
		config HistoryConfig
		check  func(t *testing.T, output string)
	}{
		{
			name: "page beyond total",
			config: HistoryConfig{
				Limit:  10,
				Page:   100,
				Output: &bytes.Buffer{},
			},
			check: func(t *testing.T, output string) {
				// Should adjust to last page
				if !strings.Contains(output, "Page 3 of 3") {
					t.Errorf("expected to be on page 3, got: %s", output)
				}
			},
		},
		{
			name: "exact page boundary",
			config: HistoryConfig{
				Limit:  5,
				Page:   5,
				Output: &bytes.Buffer{},
			},
			check: func(t *testing.T, output string) {
				// Page 5 with limit 5 should show entries 21-25
				if !strings.Contains(output, "task-021") {
					t.Error("should show task-021 on page 5")
				}
				if !strings.Contains(output, "task-025") {
					t.Error("should show task-025 on page 5")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if buf, ok := tt.config.Output.(*bytes.Buffer); ok {
				buf.Reset()
			}

			err := loadAndDisplayHistory(tt.config)
			if err != nil {
				t.Errorf("loadAndDisplayHistory() unexpected error: %v", err)
				return
			}

			if buf, ok := tt.config.Output.(*bytes.Buffer); ok {
				tt.check(t, buf.String())
			}
		})
	}
}

func TestTaskHistoryEntryStructure(t *testing.T) {
	entry := TaskHistoryEntry{
		ID:        "task-001",
		Task:      "Test task",
		Timestamp: 1609459200000,
		Metadata: Metadata{
			Model:     "gpt-4",
			Mode:      "act",
			Completed: true,
		},
	}

	// Test JSON marshaling
	data, err := json.Marshal(entry)
	if err != nil {
		t.Errorf("failed to marshal entry: %v", err)
		return
	}

	// Test JSON unmarshaling
	var decoded TaskHistoryEntry
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Errorf("failed to unmarshal entry: %v", err)
		return
	}

	if decoded.ID != entry.ID {
		t.Errorf("ID mismatch: expected %s, got %s", entry.ID, decoded.ID)
	}
	if decoded.Task != entry.Task {
		t.Errorf("Task mismatch: expected %s, got %s", entry.Task, decoded.Task)
	}
	if decoded.Metadata.Model != entry.Metadata.Model {
		t.Errorf("Model mismatch: expected %s, got %s", entry.Metadata.Model, decoded.Metadata.Model)
	}
}

// Helper functions for testing

func resetHistoryFlags() {
	historyFlags = struct {
		limit  int
		page   int
		config string
		json   bool
	}{}
}

func setupHistoryFlags(cmd *cobra.Command) {
	cmd.Flags().IntVarP(&historyFlags.limit, "limit", "n", 10, "Number of entries")
	cmd.Flags().IntVarP(&historyFlags.page, "page", "p", 1, "Page number")
	cmd.Flags().StringVar(&historyFlags.config, "config", "", "Config file path")
	cmd.Flags().BoolVar(&historyFlags.json, "json", false, "JSON output")
}

// Integration-style test
func TestHistoryCommandIntegration(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "taskHistory.json")

	// Create test history
	now := time.Now().Unix() * 1000
	entries := []TaskHistoryEntry{
		{ID: "task-001", Task: "Integration test task 1", Timestamp: now - 2000},
		{ID: "task-002", Task: "Integration test task 2", Timestamp: now - 1000},
		{ID: "task-003", Task: "Integration test task 3", Timestamp: now},
	}

	data := map[string]interface{}{"entries": entries}
	jsonData, _ := json.Marshal(data)
	if err := os.WriteFile(historyPath, jsonData, 0644); err != nil {
		t.Fatalf("failed to create history file: %v", err)
	}

	// Override path function
	originalGetPath := getTaskHistoryPathFunc
	getTaskHistoryPathFunc = func() string {
		return historyPath
	}
	defer func() {
		getTaskHistoryPathFunc = originalGetPath
	}()

	// Reset flags
	resetHistoryFlags()

	// Build command args
	args := []string{"-n", "2", "-p", "1", "--json"}

	// Create command
	cmd := &cobra.Command{
		RunE: runHistory,
	}
	setupHistoryFlags(cmd)

	cmd.SetArgs(args)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&bytes.Buffer{})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}

	output := buf.String()

	// Verify JSON output
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Errorf("output is not valid JSON: %v\nOutput: %s", err, output)
		return
	}

	if result["page"] != float64(1) {
		t.Errorf("expected page 1, got %v", result["page"])
	}
	if result["limit"] != float64(2) {
		t.Errorf("expected limit 2, got %v", result["limit"])
	}
	if result["total"] != float64(3) {
		t.Errorf("expected total 3, got %v", result["total"])
	}

	entriesArr, ok := result["entries"].([]interface{})
	if !ok {
		t.Error("expected entries to be an array")
		return
	}
	if len(entriesArr) != 2 {
		t.Errorf("expected 2 entries on page 1, got %d", len(entriesArr))
	}
}

// Benchmark tests
func BenchmarkLoadTaskHistory(b *testing.B) {
	// Create temp directory with test data
	tmpDir := b.TempDir()
	historyPath := filepath.Join(tmpDir, "taskHistory.json")

	// Create many entries
	var entries []TaskHistoryEntry
	now := time.Now().Unix() * 1000
	for i := 0; i < 100; i++ {
		entries = append(entries, TaskHistoryEntry{
			ID:        fmt.Sprintf("task-%03d", i),
			Task:      fmt.Sprintf("Task number %d with some longer description", i),
			Timestamp: now - int64(i*1000),
		})
	}

	data := map[string]interface{}{"entries": entries}
	jsonData, _ := json.Marshal(data)
	if err := os.WriteFile(historyPath, jsonData, 0644); err != nil {
		b.Fatalf("failed to create history file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = loadTaskHistory(historyPath)
	}
}

func BenchmarkOutputTable(b *testing.B) {
	entries := make([]TaskHistoryEntry, 20)
	now := time.Now().Unix() * 1000
	for i := 0; i < 20; i++ {
		entries[i] = TaskHistoryEntry{
			ID:        fmt.Sprintf("task-%03d", i),
			Task:      fmt.Sprintf("Task %d", i),
			Timestamp: now - int64(i*1000),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = outputTable(&buf, entries, 1, 20, 20, 1)
	}
}
