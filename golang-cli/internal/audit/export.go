package audit

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ReportFormat represents the format of the compliance report
type ReportFormat string

const (
	// ReportFormatJSON exports report as JSON
	ReportFormatJSON ReportFormat = "json"
	// ReportFormatCSV exports report as CSV
	ReportFormatCSV ReportFormat = "csv"
	// ReportFormatSummary exports a summary report
	ReportFormatSummary ReportFormat = "summary"
)

// ReportFilter defines filters for generating compliance reports
type ReportFilter struct {
	// StartTime is the inclusive start time for events
	StartTime time.Time
	// EndTime is the inclusive end time for events
	EndTime time.Time
	// EventTypes filters by specific event types (empty means all types)
	EventTypes []EventType
	// UserContext filters by specific user (empty means all users)
	UserContext string
	// TaskID filters by specific task (empty means all tasks)
	TaskID string
	// SessionID filters by specific session (empty means all sessions)
	SessionID string
}

// ComplianceReport represents a generated compliance report
type ComplianceReport struct {
	// GeneratedAt is the timestamp when the report was generated
	GeneratedAt time.Time `json:"generated_at"`
	// Filter contains the filters applied to generate this report
	Filter ReportFilter `json:"filter"`
	// TotalEvents is the total number of events in the report
	TotalEvents int `json:"total_events"`
	// Events contains the actual audit events (for JSON format)
	Events []Event `json:"events,omitempty"`
	// Summary contains aggregated statistics (for summary format)
	Summary *ReportSummary `json:"summary,omitempty"`
}

// ReportSummary contains aggregated statistics for the report
type ReportSummary struct {
	// EventTypeCounts maps event types to their occurrence count
	EventTypeCounts map[string]int `json:"event_type_counts"`
	// UserCounts maps user contexts to their event count
	UserCounts map[string]int `json:"user_counts"`
	// TaskCounts maps task IDs to their event count
	TaskCounts map[string]int `json:"task_counts"`
	// SessionCounts maps session IDs to their event count
	SessionCounts map[string]int `json:"session_counts"`
	// TimeRangeStart is the earliest event timestamp
	TimeRangeStart time.Time `json:"time_range_start"`
	// TimeRangeEnd is the latest event timestamp
	TimeRangeEnd time.Time `json:"time_range_end"`
	// UniqueUsers is the count of unique users
	UniqueUsers int `json:"unique_users"`
	// UniqueTasks is the count of unique tasks
	UniqueTasks int `json:"unique_tasks"`
	// UniqueSessions is the count of unique sessions
	UniqueSessions int `json:"unique_sessions"`
}

// ReportGenerator handles the generation of compliance reports
type ReportGenerator struct {
	// logDir is the directory containing audit log files
	logDir string
	// bufferSize is the size of the read buffer for streaming
	bufferSize int
}

// NewReportGenerator creates a new report generator for the given log directory
func NewReportGenerator(logDir string) *ReportGenerator {
	return &ReportGenerator{
		logDir:     logDir,
		bufferSize: 64 * 1024, // 64KB buffer for reading
	}
}

// NewReportGeneratorWithLogger creates a report generator using the logger's log directory
func NewReportGeneratorWithLogger(logger *Logger) *ReportGenerator {
	return NewReportGenerator(logger.config.LogDir)
}

// Generate generates a compliance report with the given filter and format
func (rg *ReportGenerator) Generate(filter ReportFilter, format ReportFormat) (*ComplianceReport, error) {
	// Validate filter times
	if !filter.StartTime.IsZero() && !filter.EndTime.IsZero() && filter.StartTime.After(filter.EndTime) {
		return nil, fmt.Errorf("start time cannot be after end time")
	}

	// Normalize times to UTC
	if !filter.StartTime.IsZero() {
		filter.StartTime = filter.StartTime.UTC()
	}
	if !filter.EndTime.IsZero() {
		filter.EndTime = filter.EndTime.UTC()
	}

	report := &ComplianceReport{
		GeneratedAt: time.Now().UTC(),
		Filter:      filter,
		Events:      make([]Event, 0),
	}

	// Collect events from log files
	if err := rg.collectEvents(filter, func(event Event) error {
		report.Events = append(report.Events, event)
		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to collect events: %w", err)
	}

	report.TotalEvents = len(report.Events)

	// Generate summary if requested
	if format == ReportFormatSummary {
		report.Summary = rg.generateSummary(report.Events)
	}

	return report, nil
}

// GenerateStream generates a compliance report and writes it directly to a writer
// This is useful for large reports that shouldn't be held entirely in memory
func (rg *ReportGenerator) GenerateStream(w io.Writer, filter ReportFilter, format ReportFormat) error {
	// Validate filter times
	if !filter.StartTime.IsZero() && !filter.EndTime.IsZero() && filter.StartTime.After(filter.EndTime) {
		return fmt.Errorf("start time cannot be after end time")
	}

	// Normalize times to UTC
	if !filter.StartTime.IsZero() {
		filter.StartTime = filter.StartTime.UTC()
	}
	if !filter.EndTime.IsZero() {
		filter.EndTime = filter.EndTime.UTC()
	}

	switch format {
	case ReportFormatJSON:
		return rg.generateJSONStream(w, filter)
	case ReportFormatCSV:
		return rg.generateCSVStream(w, filter)
	case ReportFormatSummary:
		return rg.generateSummaryStream(w, filter)
	default:
		return fmt.Errorf("unsupported report format: %s", format)
	}
}

// ExportToFile generates a report and writes it to a file
func (rg *ReportGenerator) ExportToFile(outputPath string, filter ReportFilter, format ReportFormat) error {
	// Create output directory if it doesn't exist
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	if err := rg.GenerateStream(file, filter, format); err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	return nil
}

// collectEvents iterates through log files and applies filters
func (rg *ReportGenerator) collectEvents(filter ReportFilter, callback func(Event) error) error {
	// Get all log files (including rotated backups)
	logFiles, err := rg.getLogFiles()
	if err != nil {
		return fmt.Errorf("failed to get log files: %w", err)
	}

	// Sort files by name to process in chronological order (oldest first)
	sort.Strings(logFiles)

	for _, logFile := range logFiles {
		if err := rg.processLogFile(logFile, filter, callback); err != nil {
			return fmt.Errorf("failed to process log file %s: %w", logFile, err)
		}
	}

	return nil
}

// getLogFiles returns all audit log files in the log directory
func (rg *ReportGenerator) getLogFiles() ([]string, error) {
	entries, err := os.ReadDir(rg.logDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var logFiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, "audit.log") {
			logFiles = append(logFiles, filepath.Join(rg.logDir, name))
		}
	}

	return logFiles, nil
}

// processLogFile processes a single log file and applies filters
func (rg *ReportGenerator) processLogFile(logPath string, filter ReportFilter, callback func(Event) error) error {
	file, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	reader := bufio.NewReaderSize(file, rg.bufferSize)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				// Process last line if not empty
				if len(line) > 0 {
					if event, ok := rg.parseLogLine(line); ok && rg.matchesFilter(event, filter) {
						if err := callback(event); err != nil {
							return err
						}
					}
				}
				break
			}
			return err
		}

		// Remove trailing newline
		line = strings.TrimRight(line, "\n\r")

		if event, ok := rg.parseLogLine(line); ok && rg.matchesFilter(event, filter) {
			if err := callback(event); err != nil {
				return err
			}
		}
	}

	return nil
}

// parseLogLine parses a single log line into an Event
func (rg *ReportGenerator) parseLogLine(line string) (Event, bool) {
	var event Event

	// slog outputs JSON format, parse the event field
	var logEntry struct {
		Event struct {
			Timestamp   string                 `json:"timestamp"`
			EventType   string                 `json:"event_type"`
			UserContext string                 `json:"user_context"`
			TaskID      string                 `json:"task_id"`
			SessionID   string                 `json:"session_id"`
			Message     string                 `json:"message"`
			Details     map[string]interface{} `json:"details"`
		} `json:"event"`
	}

	if err := json.Unmarshal([]byte(line), &logEntry); err != nil {
		return event, false
	}

	// Parse timestamp
	timestamp, err := time.Parse(time.RFC3339Nano, logEntry.Event.Timestamp)
	if err != nil {
		// Try alternative formats
		timestamp, err = time.Parse(time.RFC3339, logEntry.Event.Timestamp)
		if err != nil {
			return event, false
		}
	}

	event = Event{
		Timestamp:   timestamp.UTC(),
		EventType:   EventType(logEntry.Event.EventType),
		UserContext: logEntry.Event.UserContext,
		TaskID:      logEntry.Event.TaskID,
		SessionID:   logEntry.Event.SessionID,
		Message:     logEntry.Event.Message,
		Details:     logEntry.Event.Details,
	}

	return event, true
}

// matchesFilter checks if an event matches the given filter
func (rg *ReportGenerator) matchesFilter(event Event, filter ReportFilter) bool {
	// Check time range
	if !filter.StartTime.IsZero() && event.Timestamp.Before(filter.StartTime) {
		return false
	}
	if !filter.EndTime.IsZero() && event.Timestamp.After(filter.EndTime) {
		return false
	}

	// Check event types
	if len(filter.EventTypes) > 0 {
		found := false
		for _, et := range filter.EventTypes {
			if event.EventType == et {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check user context
	if filter.UserContext != "" && event.UserContext != filter.UserContext {
		return false
	}

	// Check task ID
	if filter.TaskID != "" && event.TaskID != filter.TaskID {
		return false
	}

	// Check session ID
	if filter.SessionID != "" && event.SessionID != filter.SessionID {
		return false
	}

	return true
}

// generateSummary creates a summary from a slice of events
func (rg *ReportGenerator) generateSummary(events []Event) *ReportSummary {
	summary := &ReportSummary{
		EventTypeCounts: make(map[string]int),
		UserCounts:      make(map[string]int),
		TaskCounts:      make(map[string]int),
		SessionCounts:   make(map[string]int),
	}

	if len(events) == 0 {
		return summary
	}

	summary.TimeRangeStart = events[0].Timestamp
	summary.TimeRangeEnd = events[0].Timestamp

	uniqueUsers := make(map[string]struct{})
	uniqueTasks := make(map[string]struct{})
	uniqueSessions := make(map[string]struct{})

	for _, event := range events {
		// Update counts
		summary.EventTypeCounts[string(event.EventType)]++
		summary.UserCounts[event.UserContext]++
		summary.TaskCounts[event.TaskID]++
		summary.SessionCounts[event.SessionID]++

		// Track unique values
		uniqueUsers[event.UserContext] = struct{}{}
		uniqueTasks[event.TaskID] = struct{}{}
		uniqueSessions[event.SessionID] = struct{}{}

		// Update time range
		if event.Timestamp.Before(summary.TimeRangeStart) {
			summary.TimeRangeStart = event.Timestamp
		}
		if event.Timestamp.After(summary.TimeRangeEnd) {
			summary.TimeRangeEnd = event.Timestamp
		}
	}

	summary.UniqueUsers = len(uniqueUsers)
	summary.UniqueTasks = len(uniqueTasks)
	summary.UniqueSessions = len(uniqueSessions)

	return summary
}

// generateJSONStream streams report as JSON
func (rg *ReportGenerator) generateJSONStream(w io.Writer, filter ReportFilter) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")

	// Write report header
	header := struct {
		GeneratedAt time.Time    `json:"generated_at"`
		Filter      ReportFilter `json:"filter"`
	}{
		GeneratedAt: time.Now().UTC(),
		Filter:      filter,
	}

	if _, err := w.Write([]byte("{\n")); err != nil {
		return err
	}

	// Encode header
	headerJSON, err := json.MarshalIndent(header, "  ", "  ")
	if err != nil {
		return err
	}
	if _, err := w.Write(append(headerJSON, ',', '\n')); err != nil {
		return err
	}

	// Write events array
	if _, err := w.Write([]byte("  \"events\": [\n")); err != nil {
		return err
	}

	first := true
	eventCount := 0

	if err := rg.collectEvents(filter, func(event Event) error {
		eventCount++

		if !first {
			if _, err := w.Write([]byte(",\n")); err != nil {
				return err
			}
		}
		first = false

		eventJSON, err := json.MarshalIndent(event, "    ", "  ")
		if err != nil {
			return err
		}

		if _, err := w.Write(eventJSON); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return err
	}

	// Close events array and write total
	if _, err := w.Write([]byte("\n  ],\n")); err != nil {
		return err
	}

	totalJSON, err := json.MarshalIndent(struct {
		TotalEvents int `json:"total_events"`
	}{
		TotalEvents: eventCount,
	}, "  ", "  ")
	if err != nil {
		return err
	}

	if _, err := w.Write(append(totalJSON, '\n', '}')); err != nil {
		return err
	}

	return nil
}

// generateCSVStream streams report as CSV
func (rg *ReportGenerator) generateCSVStream(w io.Writer, filter ReportFilter) error {
	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	// Write header
	headers := []string{"timestamp", "event_type", "user_context", "task_id", "session_id", "message", "details"}
	if err := csvWriter.Write(headers); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	if err := rg.collectEvents(filter, func(event Event) error {
		detailsJSON, err := json.Marshal(event.Details)
		if err != nil {
			detailsJSON = []byte("{}")
		}

		record := []string{
			event.Timestamp.UTC().Format(time.RFC3339Nano),
			string(event.EventType),
			event.UserContext,
			event.TaskID,
			event.SessionID,
			event.Message,
			string(detailsJSON),
		}

		if err := csvWriter.Write(record); err != nil {
			return fmt.Errorf("failed to write CSV record: %w", err)
		}

		// Flush periodically to avoid buffering too much
		if csvWriter.Error() != nil {
			return csvWriter.Error()
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

// generateSummaryStream streams a summary report
func (rg *ReportGenerator) generateSummaryStream(w io.Writer, filter ReportFilter) error {
	// Collect all events to build summary
	var events []Event
	if err := rg.collectEvents(filter, func(event Event) error {
		events = append(events, event)
		return nil
	}); err != nil {
		return err
	}

	summary := rg.generateSummary(events)

	report := struct {
		GeneratedAt time.Time      `json:"generated_at"`
		Filter      ReportFilter   `json:"filter"`
		TotalEvents int            `json:"total_events"`
		Summary     *ReportSummary `json:"summary"`
	}{
		GeneratedAt: time.Now().UTC(),
		Filter:      filter,
		TotalEvents: len(events),
		Summary:     summary,
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

// ValidateReportFormat validates a report format string
func ValidateReportFormat(format string) (ReportFormat, error) {
	switch ReportFormat(format) {
	case ReportFormatJSON, ReportFormatCSV, ReportFormatSummary:
		return ReportFormat(format), nil
	default:
		return "", fmt.Errorf("invalid report format: %s (valid formats: json, csv, summary)", format)
	}
}

// DefaultReportPath returns a default report file path based on format and timestamp
func DefaultReportPath(logDir string, format ReportFormat) string {
	timestamp := time.Now().UTC().Format("20060102-150405")
	ext := string(format)
	if ext == "summary" {
		ext = "json"
	}
	return filepath.Join(logDir, fmt.Sprintf("compliance-report-%s.%s", timestamp, ext))
}

// BufferSize returns the current buffer size for streaming operations
func (rg *ReportGenerator) BufferSize() int {
	return rg.bufferSize
}

// SetBufferSize sets the buffer size for streaming operations
func (rg *ReportGenerator) SetBufferSize(size int) {
	if size > 0 {
		rg.bufferSize = size
	}
}

// ReportToJSON converts a ComplianceReport to JSON bytes
func ReportToJSON(report *ComplianceReport) ([]byte, error) {
	return json.MarshalIndent(report, "", "  ")
}

// ReportToCSV converts a ComplianceReport to CSV bytes
func ReportToCSV(report *ComplianceReport) ([]byte, error) {
	var buf bytes.Buffer
	csvWriter := csv.NewWriter(&buf)

	// Write header
	headers := []string{"timestamp", "event_type", "user_context", "task_id", "session_id", "message", "details"}
	if err := csvWriter.Write(headers); err != nil {
		return nil, fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write events
	for _, event := range report.Events {
		detailsJSON, err := json.Marshal(event.Details)
		if err != nil {
			detailsJSON = []byte("{}")
		}

		record := []string{
			event.Timestamp.UTC().Format(time.RFC3339Nano),
			string(event.EventType),
			event.UserContext,
			event.TaskID,
			event.SessionID,
			event.Message,
			string(detailsJSON),
		}

		if err := csvWriter.Write(record); err != nil {
			return nil, fmt.Errorf("failed to write CSV record: %w", err)
		}
	}

	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
