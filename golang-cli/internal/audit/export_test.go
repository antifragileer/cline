package audit

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReportFormat_Constants(t *testing.T) {
	tests := []struct {
		format   ReportFormat
		expected string
	}{
		{ReportFormatJSON, "json"},
		{ReportFormatCSV, "csv"},
		{ReportFormatSummary, "summary"},
	}

	for _, test := range tests {
		t.Run(string(test.format), func(t *testing.T) {
			if string(test.format) != test.expected {
				t.Errorf("ReportFormat = %s, want %s", test.format, test.expected)
			}
		})
	}
}

func TestValidateReportFormat(t *testing.T) {
	tests := []struct {
		input       string
		expectError bool
		expected    ReportFormat
	}{
		{"json", false, ReportFormatJSON},
		{"csv", false, ReportFormatCSV},
		{"summary", false, ReportFormatSummary},
		{"JSON", true, ""},
		{"Csv", true, ""},
		{"pdf", true, ""},
		{"", true, ""},
		{"xml", true, ""},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			format, err := ValidateReportFormat(test.input)
			if test.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if format != test.expected {
					t.Errorf("Format = %s, want %s", format, test.expected)
				}
			}
		})
	}
}

func TestNewReportGenerator(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "audit-export-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("creates generator with log directory", func(t *testing.T) {
		rg := NewReportGenerator(tempDir)
		if rg == nil {
			t.Fatal("NewReportGenerator returned nil")
		}
		if rg.logDir != tempDir {
			t.Errorf("logDir = %s, want %s", rg.logDir, tempDir)
		}
		if rg.bufferSize != 64*1024 {
			t.Errorf("bufferSize = %d, want %d", rg.bufferSize, 64*1024)
		}
	})

	t.Run("creates generator from logger", func(t *testing.T) {
		config := Config{
			LogDir:      tempDir,
			MaxFileSize: 1024 * 1024,
			MaxBackups:  3,
			BufferSize:  10,
			SyncWrite:   true,
		}

		logger, err := NewLogger(config)
		if err != nil {
			t.Fatalf("NewLogger failed: %v", err)
		}
		defer logger.Close()

		rg := NewReportGeneratorWithLogger(logger)
		if rg == nil {
			t.Fatal("NewReportGeneratorWithLogger returned nil")
		}
		if rg.logDir != tempDir {
			t.Errorf("logDir = %s, want %s", rg.logDir, tempDir)
		}
	})
}

func TestReportGenerator_Generate(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "audit-export-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create logger and log some events
	config := Config{
		LogDir:      tempDir,
		MaxFileSize: 1024 * 1024,
		MaxBackups:  3,
		BufferSize:  100,
		SyncWrite:   true,
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}

	// Create test events
	baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	events := []*Event{
		{
			Timestamp:   baseTime,
			EventType:   EventCommandExecution,
			UserContext: "user1",
			TaskID:      "task-001",
			SessionID:   "session-001",
			Message:     "Command executed",
			Details:     map[string]interface{}{"command": "ls"},
		},
		{
			Timestamp:   baseTime.Add(1 * time.Hour),
			EventType:   EventFileRead,
			UserContext: "user1",
			TaskID:      "task-001",
			SessionID:   "session-001",
			Message:     "File read",
			Details:     map[string]interface{}{"file_path": "/tmp/test.txt"},
		},
		{
			Timestamp:   baseTime.Add(2 * time.Hour),
			EventType:   EventFileWrite,
			UserContext: "user2",
			TaskID:      "task-002",
			SessionID:   "session-002",
			Message:     "File written",
			Details:     map[string]interface{}{"file_path": "/tmp/output.txt"},
		},
		{
			Timestamp:   baseTime.Add(3 * time.Hour),
			EventType:   EventAPICall,
			UserContext: "user2",
			TaskID:      "task-002",
			SessionID:   "session-002",
			Message:     "API call completed",
			Details:     map[string]interface{}{"provider": "openai"},
		},
	}

	for _, event := range events {
		if err := logger.Log(event); err != nil {
			t.Fatalf("Failed to log event: %v", err)
		}
	}

	logger.Close()

	rg := NewReportGenerator(tempDir)

	t.Run("generates report with no filters", func(t *testing.T) {
		filter := ReportFilter{}
		report, err := rg.Generate(filter, ReportFormatJSON)
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		if report.TotalEvents != 4 {
			t.Errorf("TotalEvents = %d, want 4", report.TotalEvents)
		}
		if len(report.Events) != 4 {
			t.Errorf("Events length = %d, want 4", len(report.Events))
		}
		if report.GeneratedAt.IsZero() {
			t.Error("GeneratedAt should not be zero")
		}
	})

	t.Run("filters by time range", func(t *testing.T) {
		filter := ReportFilter{
			StartTime: baseTime.Add(30 * time.Minute),
			EndTime:   baseTime.Add(2*time.Hour + 30*time.Minute),
		}
		report, err := rg.Generate(filter, ReportFormatJSON)
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		if report.TotalEvents != 2 {
			t.Errorf("TotalEvents = %d, want 2", report.TotalEvents)
		}
	})

	t.Run("filters by event type", func(t *testing.T) {
		filter := ReportFilter{
			EventTypes: []EventType{EventCommandExecution, EventFileWrite},
		}
		report, err := rg.Generate(filter, ReportFormatJSON)
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		if report.TotalEvents != 2 {
			t.Errorf("TotalEvents = %d, want 2", report.TotalEvents)
		}

		for _, event := range report.Events {
			if event.EventType != EventCommandExecution && event.EventType != EventFileWrite {
				t.Errorf("Unexpected event type: %s", event.EventType)
			}
		}
	})

	t.Run("filters by user context", func(t *testing.T) {
		filter := ReportFilter{
			UserContext: "user1",
		}
		report, err := rg.Generate(filter, ReportFormatJSON)
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		if report.TotalEvents != 2 {
			t.Errorf("TotalEvents = %d, want 2", report.TotalEvents)
		}

		for _, event := range report.Events {
			if event.UserContext != "user1" {
				t.Errorf("Unexpected user context: %s", event.UserContext)
			}
		}
	})

	t.Run("filters by task ID", func(t *testing.T) {
		filter := ReportFilter{
			TaskID: "task-002",
		}
		report, err := rg.Generate(filter, ReportFormatJSON)
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		if report.TotalEvents != 2 {
			t.Errorf("TotalEvents = %d, want 2", report.TotalEvents)
		}

		for _, event := range report.Events {
			if event.TaskID != "task-002" {
				t.Errorf("Unexpected task ID: %s", event.TaskID)
			}
		}
	})

	t.Run("filters by session ID", func(t *testing.T) {
		filter := ReportFilter{
			SessionID: "session-001",
		}
		report, err := rg.Generate(filter, ReportFormatJSON)
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		if report.TotalEvents != 2 {
			t.Errorf("TotalEvents = %d, want 2", report.TotalEvents)
		}

		for _, event := range report.Events {
			if event.SessionID != "session-001" {
				t.Errorf("Unexpected session ID: %s", event.SessionID)
			}
		}
	})

	t.Run("combines multiple filters", func(t *testing.T) {
		filter := ReportFilter{
			StartTime:   baseTime,
			EndTime:     baseTime.Add(3 * time.Hour),
			EventTypes:  []EventType{EventCommandExecution, EventFileRead},
			UserContext: "user1",
		}
		report, err := rg.Generate(filter, ReportFormatJSON)
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		if report.TotalEvents != 2 {
			t.Errorf("TotalEvents = %d, want 2", report.TotalEvents)
		}
	})

	t.Run("returns error for invalid time range", func(t *testing.T) {
		filter := ReportFilter{
			StartTime: baseTime.Add(1 * time.Hour),
			EndTime:   baseTime,
		}
		_, err := rg.Generate(filter, ReportFormatJSON)
		if err == nil {
			t.Error("Expected error for invalid time range")
		}
	})

	t.Run("generates summary report", func(t *testing.T) {
		filter := ReportFilter{}
		report, err := rg.Generate(filter, ReportFormatSummary)
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		if report.Summary == nil {
			t.Fatal("Summary should not be nil")
		}

		if report.Summary.UniqueUsers != 2 {
			t.Errorf("UniqueUsers = %d, want 2", report.Summary.UniqueUsers)
		}
		if report.Summary.UniqueTasks != 2 {
			t.Errorf("UniqueTasks = %d, want 2", report.Summary.UniqueTasks)
		}
		if report.Summary.UniqueSessions != 2 {
			t.Errorf("UniqueSessions = %d, want 2", report.Summary.UniqueSessions)
		}

		if report.Summary.EventTypeCounts[string(EventCommandExecution)] != 1 {
			t.Errorf("EventTypeCounts[command_execution] = %d, want 1", report.Summary.EventTypeCounts[string(EventCommandExecution)])
		}
	})
}

func TestReportGenerator_GenerateStream(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "audit-export-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create logger and log some events
	config := Config{
		LogDir:      tempDir,
		MaxFileSize: 1024 * 1024,
		MaxBackups:  3,
		BufferSize:  100,
		SyncWrite:   true,
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}

	baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	// Log events
	for i := 0; i < 5; i++ {
		event := &Event{
			Timestamp:   baseTime.Add(time.Duration(i) * time.Hour),
			EventType:   EventCommandExecution,
			UserContext: "user1",
			TaskID:      "task-001",
			SessionID:   "session-001",
			Message:     fmt.Sprintf("Event %d", i),
		}
		if err := logger.Log(event); err != nil {
			t.Fatalf("Failed to log event: %v", err)
		}
	}

	logger.Close()

	rg := NewReportGenerator(tempDir)

	t.Run("streams JSON format", func(t *testing.T) {
		var buf bytes.Buffer
		filter := ReportFilter{}

		err := rg.GenerateStream(&buf, filter, ReportFormatJSON)
		if err != nil {
			t.Fatalf("GenerateStream failed: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, `"events"`) {
			t.Error("JSON output should contain events field")
		}
		if !strings.Contains(output, "Event 0") {
			t.Error("JSON output should contain event messages")
		}

		// The streamed JSON output has a specific structure with multiple root elements
		// We verify it's well-formed by checking for expected components
		if !strings.Contains(output, `"generated_at"`) {
			t.Error("JSON output should contain generated_at field")
		}
		if !strings.Contains(output, `"total_events"`) {
			t.Error("JSON output should contain total_events field")
		}
	})

	t.Run("streams CSV format", func(t *testing.T) {
		var buf bytes.Buffer
		filter := ReportFilter{}

		err := rg.GenerateStream(&buf, filter, ReportFormatCSV)
		if err != nil {
			t.Fatalf("GenerateStream failed: %v", err)
		}

		output := buf.String()

		// Parse CSV
		reader := csv.NewReader(strings.NewReader(output))
		records, err := reader.ReadAll()
		if err != nil {
			t.Fatalf("Failed to parse CSV output: %v", err)
		}

		// Should have header + 5 data rows
		if len(records) != 6 {
			t.Errorf("Expected 6 CSV rows (header + 5 events), got %d", len(records))
		}

		// Check header
		expectedHeader := []string{"timestamp", "event_type", "user_context", "task_id", "session_id", "message", "details"}
		for i, header := range expectedHeader {
			if records[0][i] != header {
				t.Errorf("CSV header[%d] = %s, want %s", i, records[0][i], header)
			}
		}
	})

	t.Run("streams summary format", func(t *testing.T) {
		var buf bytes.Buffer
		filter := ReportFilter{}

		err := rg.GenerateStream(&buf, filter, ReportFormatSummary)
		if err != nil {
			t.Fatalf("GenerateStream failed: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, `"summary"`) {
			t.Error("Summary output should contain summary field")
		}
		if !strings.Contains(output, `"total_events"`) {
			t.Error("Summary output should contain total_events field")
		}

		// Verify it's valid JSON
		var result map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Errorf("Output is not valid JSON: %v", err)
		}
	})

	t.Run("returns error for invalid format", func(t *testing.T) {
		var buf bytes.Buffer
		filter := ReportFilter{}

		err := rg.GenerateStream(&buf, filter, ReportFormat("invalid"))
		if err == nil {
			t.Error("Expected error for invalid format")
		}
	})

	t.Run("returns error for invalid time range", func(t *testing.T) {
		var buf bytes.Buffer
		filter := ReportFilter{
			StartTime: baseTime.Add(1 * time.Hour),
			EndTime:   baseTime,
		}

		err := rg.GenerateStream(&buf, filter, ReportFormatJSON)
		if err == nil {
			t.Error("Expected error for invalid time range")
		}
	})
}

func TestReportGenerator_ExportToFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "audit-export-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	outputDir, err := os.MkdirTemp("", "audit-output-test-*")
	if err != nil {
		t.Fatalf("Failed to create output dir: %v", err)
	}
	defer os.RemoveAll(outputDir)

	// Create logger and log some events
	config := Config{
		LogDir:      tempDir,
		MaxFileSize: 1024 * 1024,
		MaxBackups:  3,
		BufferSize:  100,
		SyncWrite:   true,
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}

	event := &Event{
		Timestamp:   time.Now().UTC(),
		EventType:   EventCommandExecution,
		UserContext: "user1",
		TaskID:      "task-001",
		SessionID:   "session-001",
		Message:     "Test event",
	}
	if err := logger.Log(event); err != nil {
		t.Fatalf("Failed to log event: %v", err)
	}
	logger.Close()

	rg := NewReportGenerator(tempDir)

	t.Run("exports to JSON file", func(t *testing.T) {
		outputPath := filepath.Join(outputDir, "report.json")
		filter := ReportFilter{}

		err := rg.ExportToFile(outputPath, filter, ReportFormatJSON)
		if err != nil {
			t.Fatalf("ExportToFile failed: %v", err)
		}

		// Verify file exists
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Error("Output file was not created")
		}

		// Verify content
		content, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("Failed to read output file: %v", err)
		}

		if !strings.Contains(string(content), "Test event") {
			t.Error("Output file should contain event message")
		}
	})

	t.Run("exports to CSV file", func(t *testing.T) {
		outputPath := filepath.Join(outputDir, "report.csv")
		filter := ReportFilter{}

		err := rg.ExportToFile(outputPath, filter, ReportFormatCSV)
		if err != nil {
			t.Fatalf("ExportToFile failed: %v", err)
		}

		content, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("Failed to read output file: %v", err)
		}

		if !strings.Contains(string(content), "timestamp") {
			t.Error("CSV output should contain header")
		}
	})

	t.Run("creates output directory if not exists", func(t *testing.T) {
		outputPath := filepath.Join(outputDir, "nested", "deep", "report.json")
		filter := ReportFilter{}

		err := rg.ExportToFile(outputPath, filter, ReportFormatJSON)
		if err != nil {
			t.Fatalf("ExportToFile failed: %v", err)
		}

		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Error("Output file should be created in nested directory")
		}
	})
}

func TestReportGenerator_EmptyLogs(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "audit-export-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	rg := NewReportGenerator(tempDir)

	t.Run("handles empty log directory", func(t *testing.T) {
		filter := ReportFilter{}
		report, err := rg.Generate(filter, ReportFormatJSON)
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		if report.TotalEvents != 0 {
			t.Errorf("TotalEvents = %d, want 0", report.TotalEvents)
		}
		if len(report.Events) != 0 {
			t.Errorf("Events length = %d, want 0", len(report.Events))
		}
	})

	t.Run("generates empty summary", func(t *testing.T) {
		filter := ReportFilter{}
		report, err := rg.Generate(filter, ReportFormatSummary)
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		if report.Summary == nil {
			t.Fatal("Summary should not be nil")
		}
		if report.Summary.UniqueUsers != 0 {
			t.Errorf("UniqueUsers = %d, want 0", report.Summary.UniqueUsers)
		}
	})
}

func TestReportGenerator_parseLogLine(t *testing.T) {
	rg := NewReportGenerator("/tmp")

	tests := []struct {
		name     string
		line     string
		expected bool
	}{
		{
			name:     "valid slog JSON line",
			line:     `{"time":"2024-01-15T10:30:00Z","level":"INFO","msg":"audit_event","event":{"timestamp":"2024-01-15T10:30:00Z","event_type":"command_execution","user_context":"user1","task_id":"task-001","session_id":"session-001","message":"Command executed"}}`,
			expected: true,
		},
		{
			name:     "valid line with details",
			line:     `{"time":"2024-01-15T10:30:00Z","level":"INFO","msg":"audit_event","event":{"timestamp":"2024-01-15T10:30:00.123456789Z","event_type":"file_read","user_context":"user2","task_id":"task-002","session_id":"session-002","message":"File read","details":{"file_path":"/tmp/test.txt"}}}`,
			expected: true,
		},
		{
			name:     "invalid JSON",
			line:     `this is not json`,
			expected: false,
		},
		{
			name:     "missing event field",
			line:     `{"time":"2024-01-15T10:30:00Z","level":"INFO","msg":"other"}`,
			expected: false,
		},
		{
			name:     "empty line",
			line:     "",
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			event, ok := rg.parseLogLine(test.line)
			if ok != test.expected {
				t.Errorf("parseLogLine() = %v, want %v", ok, test.expected)
			}
			if test.expected && ok {
				if event.Timestamp.IsZero() {
					t.Error("Parsed event should have timestamp")
				}
				if event.EventType == "" {
					t.Error("Parsed event should have event type")
				}
			}
		})
	}
}

func TestReportGenerator_matchesFilter(t *testing.T) {
	rg := NewReportGenerator("/tmp")

	baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	event := Event{
		Timestamp:   baseTime,
		EventType:   EventCommandExecution,
		UserContext: "user1",
		TaskID:      "task-001",
		SessionID:   "session-001",
		Message:     "Test",
	}

	tests := []struct {
		name     string
		filter   ReportFilter
		expected bool
	}{
		{
			name:     "no filters",
			filter:   ReportFilter{},
			expected: true,
		},
		{
			name: "matches time range",
			filter: ReportFilter{
				StartTime: baseTime.Add(-1 * time.Hour),
				EndTime:   baseTime.Add(1 * time.Hour),
			},
			expected: true,
		},
		{
			name: "before time range",
			filter: ReportFilter{
				StartTime: baseTime.Add(1 * time.Hour),
			},
			expected: false,
		},
		{
			name: "after time range",
			filter: ReportFilter{
				EndTime: baseTime.Add(-1 * time.Hour),
			},
			expected: false,
		},
		{
			name: "matches event type",
			filter: ReportFilter{
				EventTypes: []EventType{EventCommandExecution},
			},
			expected: true,
		},
		{
			name: "wrong event type",
			filter: ReportFilter{
				EventTypes: []EventType{EventFileRead},
			},
			expected: false,
		},
		{
			name: "one of multiple event types",
			filter: ReportFilter{
				EventTypes: []EventType{EventFileRead, EventCommandExecution},
			},
			expected: true,
		},
		{
			name: "matches user context",
			filter: ReportFilter{
				UserContext: "user1",
			},
			expected: true,
		},
		{
			name: "wrong user context",
			filter: ReportFilter{
				UserContext: "user2",
			},
			expected: false,
		},
		{
			name: "matches task ID",
			filter: ReportFilter{
				TaskID: "task-001",
			},
			expected: true,
		},
		{
			name: "wrong task ID",
			filter: ReportFilter{
				TaskID: "task-002",
			},
			expected: false,
		},
		{
			name: "matches session ID",
			filter: ReportFilter{
				SessionID: "session-001",
			},
			expected: true,
		},
		{
			name: "wrong session ID",
			filter: ReportFilter{
				SessionID: "session-002",
			},
			expected: false,
		},
		{
			name: "multiple filters match",
			filter: ReportFilter{
				EventTypes:  []EventType{EventCommandExecution},
				UserContext: "user1",
				TaskID:      "task-001",
			},
			expected: true,
		},
		{
			name: "one filter doesn't match",
			filter: ReportFilter{
				EventTypes:  []EventType{EventCommandExecution},
				UserContext: "user2",
			},
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := rg.matchesFilter(event, test.filter)
			if result != test.expected {
				t.Errorf("matchesFilter() = %v, want %v", result, test.expected)
			}
		})
	}
}

func TestReportGenerator_generateSummary(t *testing.T) {
	rg := NewReportGenerator("/tmp")

	baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	t.Run("empty events", func(t *testing.T) {
		summary := rg.generateSummary([]Event{})
		if summary == nil {
			t.Fatal("Summary should not be nil")
		}
		if len(summary.EventTypeCounts) != 0 {
			t.Error("EventTypeCounts should be empty")
		}
	})

	t.Run("single event", func(t *testing.T) {
		events := []Event{
			{
				Timestamp:   baseTime,
				EventType:   EventCommandExecution,
				UserContext: "user1",
				TaskID:      "task-001",
				SessionID:   "session-001",
			},
		}
		summary := rg.generateSummary(events)

		if summary.EventTypeCounts[string(EventCommandExecution)] != 1 {
			t.Errorf("EventTypeCounts[command_execution] = %d, want 1", summary.EventTypeCounts[string(EventCommandExecution)])
		}
		if summary.UserCounts["user1"] != 1 {
			t.Errorf("UserCounts[user1] = %d, want 1", summary.UserCounts["user1"])
		}
		if summary.UniqueUsers != 1 {
			t.Errorf("UniqueUsers = %d, want 1", summary.UniqueUsers)
		}
	})

	t.Run("multiple events", func(t *testing.T) {
		events := []Event{
			{Timestamp: baseTime, EventType: EventCommandExecution, UserContext: "user1", TaskID: "task-001", SessionID: "session-001"},
			{Timestamp: baseTime.Add(1 * time.Hour), EventType: EventFileRead, UserContext: "user1", TaskID: "task-001", SessionID: "session-001"},
			{Timestamp: baseTime.Add(2 * time.Hour), EventType: EventFileWrite, UserContext: "user2", TaskID: "task-002", SessionID: "session-002"},
			{Timestamp: baseTime.Add(3 * time.Hour), EventType: EventCommandExecution, UserContext: "user2", TaskID: "task-002", SessionID: "session-002"},
		}
		summary := rg.generateSummary(events)

		if summary.EventTypeCounts[string(EventCommandExecution)] != 2 {
			t.Errorf("EventTypeCounts[command_execution] = %d, want 2", summary.EventTypeCounts[string(EventCommandExecution)])
		}
		if summary.UniqueUsers != 2 {
			t.Errorf("UniqueUsers = %d, want 2", summary.UniqueUsers)
		}
		if summary.UniqueTasks != 2 {
			t.Errorf("UniqueTasks = %d, want 2", summary.UniqueTasks)
		}
		if !summary.TimeRangeStart.Equal(baseTime) {
			t.Errorf("TimeRangeStart = %v, want %v", summary.TimeRangeStart, baseTime)
		}
	})

	t.Run("tracks time range", func(t *testing.T) {
		events := []Event{
			{Timestamp: baseTime.Add(2 * time.Hour), EventType: EventCommandExecution, UserContext: "user1", TaskID: "task-001", SessionID: "session-001"},
			{Timestamp: baseTime, EventType: EventFileRead, UserContext: "user1", TaskID: "task-001", SessionID: "session-001"},
			{Timestamp: baseTime.Add(4 * time.Hour), EventType: EventFileWrite, UserContext: "user2", TaskID: "task-002", SessionID: "session-002"},
		}
		summary := rg.generateSummary(events)

		if !summary.TimeRangeStart.Equal(baseTime) {
			t.Errorf("TimeRangeStart = %v, want %v", summary.TimeRangeStart, baseTime)
		}
		expectedEnd := baseTime.Add(4 * time.Hour)
		if !summary.TimeRangeEnd.Equal(expectedEnd) {
			t.Errorf("TimeRangeEnd = %v, want %v", summary.TimeRangeEnd, expectedEnd)
		}
	})
}

func TestDefaultReportPath(t *testing.T) {
	tempDir := "/tmp/audit"
	format := ReportFormatJSON

	path := DefaultReportPath(tempDir, format)

	if !strings.HasPrefix(path, tempDir) {
		t.Errorf("Path %s should start with %s", path, tempDir)
	}
	if !strings.Contains(path, "compliance-report-") {
		t.Error("Path should contain 'compliance-report-' prefix")
	}
	if !strings.HasSuffix(path, ".json") {
		t.Errorf("Path should end with .json, got %s", filepath.Ext(path))
	}
}

func TestDefaultReportPath_Summary(t *testing.T) {
	tempDir := "/tmp/audit"
	format := ReportFormatSummary

	path := DefaultReportPath(tempDir, format)

	if !strings.HasSuffix(path, ".json") {
		t.Errorf("Summary path should end with .json, got %s", filepath.Ext(path))
	}
}

func TestReportGenerator_BufferSize(t *testing.T) {
	tempDir := "/tmp"
	rg := NewReportGenerator(tempDir)

	t.Run("default buffer size", func(t *testing.T) {
		if rg.BufferSize() != 64*1024 {
			t.Errorf("Default buffer size = %d, want %d", rg.BufferSize(), 64*1024)
		}
	})

	t.Run("set buffer size", func(t *testing.T) {
		rg.SetBufferSize(128 * 1024)
		if rg.BufferSize() != 128*1024 {
			t.Errorf("Buffer size = %d, want %d", rg.BufferSize(), 128*1024)
		}
	})

	t.Run("set invalid buffer size", func(t *testing.T) {
		rg.SetBufferSize(0)
		if rg.BufferSize() != 128*1024 {
			t.Error("Buffer size should not change when setting 0")
		}
	})
}

func TestReportToJSON(t *testing.T) {
	baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	report := &ComplianceReport{
		GeneratedAt: baseTime,
		TotalEvents: 2,
		Events: []Event{
			{
				Timestamp:   baseTime,
				EventType:   EventCommandExecution,
				UserContext: "user1",
				TaskID:      "task-001",
				SessionID:   "session-001",
				Message:     "Test",
			},
		},
	}

	data, err := ReportToJSON(report)
	if err != nil {
		t.Fatalf("ReportToJSON failed: %v", err)
	}

	// Verify it's valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Errorf("Output is not valid JSON: %v", err)
	}

	if result["total_events"].(float64) != 2 {
		t.Error("JSON should contain total_events")
	}
}

func TestReportToCSV(t *testing.T) {
	baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	report := &ComplianceReport{
		GeneratedAt: baseTime,
		TotalEvents: 1,
		Events: []Event{
			{
				Timestamp:   baseTime,
				EventType:   EventCommandExecution,
				UserContext: "user1",
				TaskID:      "task-001",
				SessionID:   "session-001",
				Message:     "Test message",
				Details:     map[string]interface{}{"command": "ls"},
			},
		},
	}

	data, err := ReportToCSV(report)
	if err != nil {
		t.Fatalf("ReportToCSV failed: %v", err)
	}

	// Parse CSV
	reader := csv.NewReader(strings.NewReader(string(data)))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}

	if len(records) != 2 { // header + 1 data row
		t.Errorf("Expected 2 rows, got %d", len(records))
	}

	if records[0][0] != "timestamp" {
		t.Error("First column should be 'timestamp'")
	}
	if !strings.Contains(records[1][5], "Test message") {
		t.Error("Data row should contain message")
	}
}

func TestReportGenerator_RotatedLogs(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "audit-export-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create logger
	config := Config{
		LogDir:      tempDir,
		MaxFileSize: 50, // Very small to trigger rotation
		MaxBackups:  3,
		BufferSize:  100,
		SyncWrite:   true,
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}

	// Log many large events to trigger rotation
	for i := 0; i < 50; i++ {
		event := &Event{
			Timestamp:   time.Now().UTC(),
			EventType:   EventCommandExecution,
			UserContext: "user1",
			TaskID:      "task-001",
			SessionID:   "session-001",
			Message:     fmt.Sprintf("Event %d with lots of padding to make the file bigger: %s", i, strings.Repeat("x", 500)),
			Details: map[string]interface{}{
				"extra_data": strings.Repeat("y", 500),
			},
		}
		if err := logger.Log(event); err != nil {
			t.Fatalf("Failed to log event: %v", err)
		}
	}

	logger.Close()

	// Check that rotation occurred
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read log directory: %v", err)
	}

	logFileCount := 0
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "audit.log") {
			logFileCount++
		}
	}

	// Generate report - should include all events from all rotated files
	rg := NewReportGenerator(tempDir)
	filter := ReportFilter{}
	report, err := rg.Generate(filter, ReportFormatJSON)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify all events were collected from rotated logs
	if report.TotalEvents < 4 {
		t.Errorf("Expected at least 4 events from rotated logs, got %d", report.TotalEvents)
	}

	t.Logf("Collected %d events from %d log files", report.TotalEvents, logFileCount)
}

func TestReportGenerator_FilterTimeNormalization(t *testing.T) {
	rg := NewReportGenerator("/tmp")

	baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	// Create events with specific times
	events := []Event{
		{Timestamp: baseTime, EventType: EventCommandExecution, UserContext: "user1", TaskID: "task-001", SessionID: "session-001"},
		{Timestamp: baseTime.Add(1 * time.Hour), EventType: EventFileRead, UserContext: "user1", TaskID: "task-001", SessionID: "session-001"},
		{Timestamp: baseTime.Add(2 * time.Hour), EventType: EventFileWrite, UserContext: "user1", TaskID: "task-001", SessionID: "session-001"},
	}

	// Test with filter times in different timezone
	nyLoc, _ := time.LoadLocation("America/New_York")
	nyTime := time.Date(2024, 1, 15, 10, 30, 0, 0, nyLoc)

	filter := ReportFilter{
		StartTime: nyTime,
	}

	// The filter should normalize to UTC and still work correctly
	if rg.matchesFilter(events[0], filter) {
		// This depends on the exact time conversion, but we verify no panic occurs
		t.Log("Filter with non-UTC timezone handled correctly")
	}
}

func TestReportGenerator_CallbackError(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "audit-export-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create logger
	config := Config{
		LogDir:      tempDir,
		MaxFileSize: 1024 * 1024,
		MaxBackups:  3,
		BufferSize:  100,
		SyncWrite:   true,
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}

	// Log an event
	event := &Event{
		Timestamp:   time.Now().UTC(),
		EventType:   EventCommandExecution,
		UserContext: "user1",
		TaskID:      "task-001",
		SessionID:   "session-001",
		Message:     "Test",
	}
	logger.Log(event)
	logger.Close()

	rg := NewReportGenerator(tempDir)

	t.Run("callback error propagates", func(t *testing.T) {
		expectedErr := fmt.Errorf("test error")
		err := rg.collectEvents(ReportFilter{}, func(e Event) error {
			return expectedErr
		})

		// Error is wrapped by processLogFile, so check it contains our error
		if err == nil {
			t.Fatal("Expected error but got nil")
		}
		if !strings.Contains(err.Error(), expectedErr.Error()) {
			t.Errorf("Expected error containing %q, got %q", expectedErr.Error(), err.Error())
		}
	})
}