// Package session provides session management for the Cline CLI
// Reference: src/shared/services/Session.ts, cli/src/index.ts line 25
package session

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ToolCallRecord represents a single tool call
// Reference: src/shared/services/Session.ts:3-8
type ToolCallRecord struct {
	Name           string    `json:"name"`
	Success        *bool     `json:"success,omitempty"`
	StartTime      time.Time `json:"start_time"`
	LastUpdateTime time.Time `json:"last_update_time"`
}

// ResourceUsage tracks memory and CPU usage
// Reference: src/shared/services/Session.ts:10-19
type ResourceUsage struct {
	// Memory (in bytes)
	HeapUsed  uint64 `json:"heap_used"`
	HeapTotal uint64 `json:"heap_total"`
	External  uint64 `json:"external"`
	RSS       uint64 `json:"rss"` // Resident Set Size

	// CPU time
	UserCPUMs   int64 `json:"user_cpu_ms"`
	SystemCPUMs int64 `json:"system_cpu_ms"`
}

// Stats contains all session statistics
// Reference: src/shared/services/Session.ts:21-34
type Stats struct {
	SessionID string `json:"session_id"`

	// Tool calls
	TotalToolCalls      int `json:"total_tool_calls"`
	SuccessfulToolCalls int `json:"successful_tool_calls"`
	FailedToolCalls     int `json:"failed_tool_calls"`

	// Timing
	SessionStartTime time.Time `json:"session_start_time"`
	APITimeMs        int64     `json:"api_time_ms"`
	ToolTimeMs       int64     `json:"tool_time_ms"`

	// Resources
	Resources       ResourceUsage `json:"resources"`
	PeakMemoryBytes uint64        `json:"peak_memory_bytes"`
}

// Manager defines the session manager interface
// Reference: cli/src/index.ts:25, src/shared/services/Session.ts:40-265
type Manager interface {
	// Reset resets session tracking for a new CLI run
	// Reference: src/shared/services/Session.ts:106-109
	Reset()

	// GetSessionID returns the current session ID
	// Reference: src/shared/services/Session.ts:114-116
	GetSessionID() string

	// StartAPICall records the start of an API call
	// Reference: src/shared/services/Session.ts:121-123
	StartAPICall()

	// EndAPICall records the end of an API call
	// Reference: src/shared/services/Session.ts:128-133
	EndAPICall()

	// UpdateToolCall updates a tool call record
	// Reference: src/shared/services/Session.ts:141-160
	UpdateToolCall(callID string, toolName string, success *bool)

	// AddAPITime adds API time directly
	// Reference: src/shared/services/Session.ts:165-167
	AddAPITime(ms int64)

	// FinalizeRequest finalizes all in-flight tool calls
	// Reference: src/shared/services/Session.ts:173-185
	FinalizeRequest()

	// GetStats returns all session statistics
	// Reference: src/shared/services/Session.ts:191-210
	GetStats() Stats

	// GetWallTimeMs returns the wall time since session started
	// Reference: src/shared/services/Session.ts:215-217
	GetWallTimeMs() int64

	// GetStartTime returns the session start time
	// Reference: src/shared/services/Session.ts:222-224
	GetStartTime() time.Time

	// GetEndTime returns the current time (session end time)
	// Reference: src/shared/services/Session.ts:229-231
	GetEndTime() time.Time

	// GetAgentActiveTimeMs returns the agent active time (API + tool time)
	// Reference: src/shared/services/Session.ts:249-253
	GetAgentActiveTimeMs() int64

	// GetSuccessRate returns the success rate as a percentage (0-100)
	// Reference: src/shared/services/Session.ts:258-264
	GetSuccessRate() float64
}

// DefaultManager implements the Manager interface
type DefaultManager struct {
	mu       sync.RWMutex
	sessionID string
	sessionStartTime time.Time
	toolCalls        []ToolCallRecord
	apiTimeMs        int64
	toolTimeMs       int64

	// In-flight tracking
	currentAPICallStart *time.Time
	inFlightToolCalls   map[string]ToolCallRecord

	// Resource tracking
	peakMemoryBytes uint64
}

// singleton instance
var (
	instance Manager
	once     sync.Once
)

// Get returns the singleton session manager instance
// Reference: src/shared/services/Session.ts:96-101
func Get() Manager {
	once.Do(func() {
		instance = NewManager()
	})
	return instance
}

// Reset resets the singleton instance for a new CLI run
// Reference: src/shared/services/Session.ts:106-109
func Reset() {
	once = sync.Once{}
	instance = nil
}

// NewManager creates a new session manager
// Reference: src/shared/services/Session.ts:57-62
func NewManager() Manager {
	return &DefaultManager{
		sessionID:         generateSessionID(),
		sessionStartTime:  time.Now(),
		toolCalls:         make([]ToolCallRecord, 0),
		inFlightToolCalls: make(map[string]ToolCallRecord),
		peakMemoryBytes:   getCurrentMemoryUsage(),
	}
}

// Reset resets session tracking for a new CLI run
// Reference: src/shared/services/Session.ts:106-109
func (m *DefaultManager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sessionID = generateSessionID()
	m.sessionStartTime = time.Now()
	m.toolCalls = make([]ToolCallRecord, 0)
	m.apiTimeMs = 0
	m.toolTimeMs = 0
	m.currentAPICallStart = nil
	m.inFlightToolCalls = make(map[string]ToolCallRecord)
	m.peakMemoryBytes = getCurrentMemoryUsage()
}

// GetSessionID returns the current session ID
// Reference: src/shared/services/Session.ts:114-116
func (m *DefaultManager) GetSessionID() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessionID
}

// StartAPICall records the start of an API call
// Reference: src/shared/services/Session.ts:121-123
func (m *DefaultManager) StartAPICall() {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	m.currentAPICallStart = &now
	m.updatePeakMemory()
}

// EndAPICall records the end of an API call
// Reference: src/shared/services/Session.ts:128-133
func (m *DefaultManager) EndAPICall() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.currentAPICallStart != nil {
		m.apiTimeMs += time.Since(*m.currentAPICallStart).Milliseconds()
		m.currentAPICallStart = nil
	}
	m.updatePeakMemory()
}

// UpdateToolCall updates a tool call record
// Reference: src/shared/services/Session.ts:141-160
func (m *DefaultManager) UpdateToolCall(callID string, toolName string, success *bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()

	if existing, ok := m.inFlightToolCalls[callID]; ok {
		// Update existing
		existing.LastUpdateTime = now
		if success != nil {
			existing.Success = success
		}
		m.inFlightToolCalls[callID] = existing
	} else {
		// Start new tool call
		m.inFlightToolCalls[callID] = ToolCallRecord{
			Name:           toolName,
			StartTime:      now,
			LastUpdateTime: now,
			Success:        success,
		}
	}

	m.updatePeakMemory()
}

// AddAPITime adds API time directly
// Reference: src/shared/services/Session.ts:165-167
func (m *DefaultManager) AddAPITime(ms int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.apiTimeMs += ms
}

// FinalizeRequest finalizes all in-flight tool calls
// Reference: src/shared/services/Session.ts:173-185
func (m *DefaultManager) FinalizeRequest() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()

	for id, record := range m.inFlightToolCalls {
		duration := now.Sub(record.StartTime).Milliseconds()
		m.toolTimeMs += duration

		// Add to completed tool calls
		m.toolCalls = append(m.toolCalls, ToolCallRecord{
			Name:           record.Name,
			Success:        record.Success,
			StartTime:      record.StartTime,
			LastUpdateTime: record.LastUpdateTime,
		})

		delete(m.inFlightToolCalls, id)
	}

	m.updatePeakMemory()
}

// GetStats returns all session statistics
// Reference: src/shared/services/Session.ts:191-210
func (m *DefaultManager) GetStats() Stats {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Finalize any in-flight calls first
	m.finalizeRequestLocked()

	// Calculate totals
	total := len(m.toolCalls)
	successful := 0
	failed := 0
	for _, call := range m.toolCalls {
		if call.Success != nil {
			if *call.Success {
				successful++
			} else {
				failed++
			}
		}
	}

	return Stats{
		SessionID:           m.sessionID,
		TotalToolCalls:      total,
		SuccessfulToolCalls: successful,
		FailedToolCalls:     failed,
		SessionStartTime:    m.sessionStartTime,
		APITimeMs:           m.apiTimeMs,
		ToolTimeMs:          m.toolTimeMs,
		Resources:           m.getResourceUsageLocked(),
		PeakMemoryBytes:     m.peakMemoryBytes,
	}
}

// GetWallTimeMs returns the wall time since session started
// Reference: src/shared/services/Session.ts:215-217
func (m *DefaultManager) GetWallTimeMs() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return time.Since(m.sessionStartTime).Milliseconds()
}

// GetStartTime returns the session start time
// Reference: src/shared/services/Session.ts:222-224
func (m *DefaultManager) GetStartTime() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessionStartTime
}

// GetEndTime returns the current time (session end time)
// Reference: src/shared/services/Session.ts:229-231
func (m *DefaultManager) GetEndTime() time.Time {
	return time.Now()
}

// GetAgentActiveTimeMs returns the agent active time (API + tool time)
// Reference: src/shared/services/Session.ts:249-253
func (m *DefaultManager) GetAgentActiveTimeMs() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Include in-flight tool time
	inFlightTime := int64(0)
	now := time.Now()
	for _, record := range m.inFlightToolCalls {
		inFlightTime += now.Sub(record.StartTime).Milliseconds()
	}

	return m.apiTimeMs + m.toolTimeMs + inFlightTime
}

// GetSuccessRate returns the success rate as a percentage (0-100)
// Reference: src/shared/services/Session.ts:258-264
func (m *DefaultManager) GetSuccessRate() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.toolCalls) == 0 {
		return 0
	}

	successful := 0
	for _, call := range m.toolCalls {
		if call.Success != nil && *call.Success {
			successful++
		}
	}

	return float64(successful) / float64(len(m.toolCalls)) * 100
}

// Internal helper methods

func (m *DefaultManager) finalizeRequestLocked() {
	now := time.Now()

	for id, record := range m.inFlightToolCalls {
		duration := now.Sub(record.StartTime).Milliseconds()
		m.toolTimeMs += duration

		m.toolCalls = append(m.toolCalls, ToolCallRecord{
			Name:           record.Name,
			Success:        record.Success,
			StartTime:      record.StartTime,
			LastUpdateTime: record.LastUpdateTime,
		})

		delete(m.inFlightToolCalls, id)
	}
}

func (m *DefaultManager) getResourceUsageLocked() ResourceUsage {
	var mstat runtime.MemStats
	runtime.ReadMemStats(&mstat)

	return ResourceUsage{
		HeapUsed:  mstat.HeapAlloc,
		HeapTotal: mstat.HeapSys,
		External:  mstat.OtherSys,
		RSS:       mstat.Sys,
		// Note: Go doesn't provide easy CPU time per process like Node.js
		// These would need platform-specific implementations
		UserCPUMs:   0,
		SystemCPUMs: 0,
	}
}

func (m *DefaultManager) updatePeakMemory() {
	current := getCurrentMemoryUsage()
	if current > m.peakMemoryBytes {
		m.peakMemoryBytes = current
	}
}

// Helper functions

func generateSessionID() string {
	// Generate a short UUID-like ID (10 characters like nanoid(10))
	u := uuid.New().String()
	// Remove dashes and take first 10 chars
	return fmt.Sprintf("%s", u[:8]+u[9:11])
}

func getCurrentMemoryUsage() uint64 {
	var mstat runtime.MemStats
	runtime.ReadMemStats(&mstat)
	return mstat.Sys
}

// FormatDuration formats a duration in milliseconds to a human-readable string
func FormatDuration(ms int64) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	seconds := ms / 1000
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	minutes := seconds / 60
	seconds = seconds % 60
	if minutes < 60 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	hours := minutes / 60
	minutes = minutes % 60
	return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
}

// FormatBytes formats bytes to a human-readable string
func FormatBytes(bytes uint64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}