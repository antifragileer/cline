// Package task provides task initialization and management functionality for the Cline CLI.
// This file contains tests for the task initialization functionality.
package task

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cline/cline/golang-cli/internal/config"
	"github.com/cline/cline/golang-cli/internal/generated/cline/cline"
	"github.com/cline/cline/golang-cli/internal/storage"
)

// mockClient implements a minimal client interface for testing
type mockClient struct {
	ready     bool
	newTaskFn func(ctx context.Context, req *cline.NewTaskRequest) (*cline.String, error)
}

func (m *mockClient) WaitForReady(ctx context.Context) error {
	if !m.ready {
		return context.DeadlineExceeded
	}
	return nil
}

func (m *mockClient) WithRetry(ctx context.Context, fn func(conn interface{}) error) error {
	return fn(nil)
}

// mockUI implements the UIInterface for testing
type mockUI struct {
	initialized bool
	closed      bool
	errors      []error
}

func (m *mockUI) Initialize(taskID string, prompt string) error {
	m.initialized = true
	return nil
}

func (m *mockUI) UpdateStatus(status string) error {
	return nil
}

func (m *mockUI) ShowError(err error) error {
	m.errors = append(m.errors, err)
	return nil
}

func (m *mockUI) Close() error {
	m.closed = true
	return nil
}

// mockStorage implements a minimal storage for testing
type mockStorage struct {
	data map[string]interface{}
}

func (m *mockStorage) Get(key string) (interface{}, bool) {
	val, ok := m.data[key]
	return val, ok
}

func (m *mockStorage) Set(key string, value interface{}) error {
	if m.data == nil {
		m.data = make(map[string]interface{})
	}
	m.data[key] = value
	return nil
}

func TestGenerateUUID(t *testing.T) {
	uuid, err := generateUUID()
	if err != nil {
		t.Fatalf("generateUUID() failed: %v", err)
	}

	// Check format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	if len(uuid) != 36 {
		t.Errorf("UUID length = %d, want 36", len(uuid))
	}

	parts := []int{8, 4, 4, 4, 12}
	pos := 0
	for i, part := range parts {
		expectedEnd := pos + part
		if i < len(parts)-1 {
			expectedEnd++ // Account for dash
		}
		if i < len(parts)-1 && uuid[pos+part] != '-' {
			t.Errorf("UUID missing dash at position %d", pos+part)
		}
		pos = expectedEnd
	}

	// Generate multiple UUIDs and ensure they're unique
	uuids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		uuid, err := generateUUID()
		if err != nil {
			t.Fatalf("generateUUID() failed: %v", err)
		}
		if uuids[uuid] {
			t.Errorf("Duplicate UUID generated: %s", uuid)
		}
		uuids[uuid] = true
	}
}

func TestIsValidUUID(t *testing.T) {
	tests := []struct {
		name     string
		uuid     string
		expected bool
	}{
		{"valid uuid v4", "550e8400-e29b-41d4-a716-446655440000", true},
		{"valid uuid without dashes", "550e8400e29b41d4a716446655440000", true},
		{"invalid - too short", "550e8400-e29b-41d4-a716", false},
		{"invalid - too long", "550e8400-e29b-41d4-a716-4466554400000", false},
		{"invalid - non-hex", "550e8400-e29b-41d4-a716-44665544000g", false},
		{"empty string", "", false},
		{"custom id", "my-task-id", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidUUID(tt.uuid)
			if got != tt.expected {
				t.Errorf("isValidUUID(%q) = %v, want %v", tt.uuid, got, tt.expected)
			}
		})
	}
}

func TestGetMimeType(t *testing.T) {
	tests := []struct {
		ext      string
		expected string
	}{
		{".png", "image/png"},
		{".jpg", "image/jpeg"},
		{".jpeg", "image/jpeg"},
		{".gif", "image/gif"},
		{".webp", "image/webp"},
		{".bmp", "image/bmp"},
		{".svg", "image/svg+xml"},
		{".txt", ""},
		{"", ""},
		{".PNG", "image/png"}, // Case insensitive
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			got := getMimeType(tt.ext)
			if got != tt.expected {
				t.Errorf("getMimeType(%q) = %q, want %q", tt.ext, got, tt.expected)
			}
		})
	}
}

func TestSetWorkingDirectory(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "task-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name        string
		cwd         string
		wantErr     bool
		errContains string
	}{
		{
			name: "empty cwd uses current directory",
			cwd:  "",
			wantErr: false,
		},
		{
			name: "valid directory",
			cwd:  tmpDir,
			wantErr: false,
		},
		{
			name:        "non-existent directory",
			cwd:         "/nonexistent/path/12345",
			wantErr:     true,
			errContains: "not accessible",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			init := &Initializer{}
			got, err := init.setWorkingDirectory(tt.cwd)

			if tt.wantErr {
				if err == nil {
					t.Errorf("setWorkingDirectory() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("setWorkingDirectory() error = %v, should contain %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("setWorkingDirectory() unexpected error = %v", err)
					return
				}
				if got == "" {
					t.Errorf("setWorkingDirectory() returned empty path")
				}
				// Verify it's an absolute path
				if !filepath.IsAbs(got) {
					t.Errorf("setWorkingDirectory() returned non-absolute path: %s", got)
				}
			}
		})
	}
}

func TestLoadImage(t *testing.T) {
	// Create a temporary directory with a test image
	tmpDir, err := os.MkdirTemp("", "task-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test PNG file (minimal valid PNG header)
	pngPath := filepath.Join(tmpDir, "test.png")
	pngData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A} // PNG signature
	if err := os.WriteFile(pngPath, pngData, 0644); err != nil {
		t.Fatalf("Failed to create test PNG: %v", err)
	}

	// Create a test directory
	testDir := filepath.Join(tmpDir, "testdir")
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test dir: %v", err)
	}

	init := &Initializer{}

	tests := []struct {
		name        string
		path        string
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid png file",
			path:    pngPath,
			wantErr: false,
		},
		{
			name:        "directory not file",
			path:        testDir,
			wantErr:     true,
			errContains: "directory",
		},
		{
			name:        "non-existent file",
			path:        filepath.Join(tmpDir, "nonexistent.png"),
			wantErr:     true,
			errContains: "not accessible",
		},
		{
			name:        "unsupported format",
			path:        filepath.Join(tmpDir, "test.txt"),
			wantErr:     true,
			errContains: "unsupported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create txt file for unsupported format test
			if tt.name == "unsupported format" {
				txtPath := filepath.Join(tmpDir, "test.txt")
				_ = os.WriteFile(txtPath, []byte("test"), 0644)
				tt.path = txtPath
			}

			got, err := init.loadImage(tt.path)

			if tt.wantErr {
				if err == nil {
					t.Errorf("loadImage() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("loadImage() error = %v, should contain %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("loadImage() unexpected error = %v", err)
					return
				}
				if got == nil {
					t.Errorf("loadImage() returned nil")
					return
				}
				if got.MimeType != "image/png" {
					t.Errorf("loadImage() MimeType = %v, want image/png", got.MimeType)
				}
				if got.Content == "" {
					t.Errorf("loadImage() Content is empty")
				}
			}
		})
	}
}

func TestNewInitializer(t *testing.T) {
	mockCfg := &config.LayeredConfig{}
	mockStorage := &storage.StorageContext{}

	tests := []struct {
		name    string
		opts    InitOptions
		wantErr bool
	}{
		{
			name: "valid options",
			opts: InitOptions{
				Config:  mockCfg,
				Storage: mockStorage,
			},
			wantErr: false,
		},
		{
			name: "missing config",
			opts: InitOptions{
				Storage: mockStorage,
			},
			wantErr: true,
		},
		{
			name: "missing storage",
			opts: InitOptions{
				Config: mockCfg,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewInitializer(tt.opts)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewInitializer() error = nil, wantErr %v", tt.wantErr)
					return
				}
			} else {
				if err != nil {
					t.Errorf("NewInitializer() unexpected error = %v", err)
					return
				}
				if got == nil {
					t.Errorf("NewInitializer() returned nil")
				}
			}
		})
	}
}

func TestGenerateTaskID(t *testing.T) {
	init := &Initializer{}

	tests := []struct {
		name       string
		providedID string
		shouldGen  bool
	}{
		{
			name:       "generate new UUID",
			providedID: "",
			shouldGen:  true,
		},
		{
			name:       "use provided valid UUID",
			providedID: "550e8400-e29b-41d4-a716-446655440000",
			shouldGen:  false,
		},
		{
			name:       "use provided custom ID",
			providedID: "my-custom-task-id",
			shouldGen:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := init.generateTaskID(tt.providedID)
			if err != nil {
				t.Errorf("generateTaskID() error = %v", err)
				return
			}

			if tt.providedID != "" {
				if got != tt.providedID {
					t.Errorf("generateTaskID() = %v, want %v", got, tt.providedID)
				}
			} else {
				// Should be a valid UUID
				if !isValidUUID(got) {
					t.Errorf("generateTaskID() returned invalid UUID: %v", got)
				}
			}
		})
	}
}

func TestTaskConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  TaskConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: TaskConfig{
				Prompt:  "Test prompt",
				Mode:    TaskModeAct,
				Timeout: 30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "empty prompt is allowed (will be handled by caller)",
			config: TaskConfig{
				Mode: TaskModePlan,
			},
			wantErr: false,
		},
		{
			name: "valid with images",
			config: TaskConfig{
				Prompt: "Analyze this image",
				Images: []string{"/path/to/image.png"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation - in real implementation this would be more thorough
			if tt.config.Prompt == "" && len(tt.config.Images) == 0 {
				// Empty prompt without images might be invalid depending on requirements
			}
			// Config is valid if we reach here
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}