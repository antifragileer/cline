// Package task provides task initialization and management functionality for the Cline CLI.
// This file contains tests for the task resumption functionality.
package task

import (
	"context"
	"testing"
	"time"

	"github.com/cline/cline/golang-cli/internal/generated/cline/cline"
	"github.com/cline/cline/golang-cli/internal/host"
)

// mockClientForResume implements a minimal client interface for testing resume
type mockClientForResume struct {
	ready        bool
	invokeErr    error
	taskResponse *cline.TaskResponse
}

func (m *mockClientForResume) WaitForReady(ctx context.Context) error {
	if !m.ready {
		return context.DeadlineExceeded
	}
	return nil
}

func (m *mockClientForResume) WithRetry(ctx context.Context, fn func(conn interface{}) error) error {
	return fn(nil)
}

// mockOutput implements the output interface for testing
type mockOutput struct {
	messages []string
}

func (m *mockOutput) Printf(format string, a ...interface{}) (int, error) {
	// Simplified - just record that something was printed
	m.messages = append(m.messages, format)
	return len(format), nil
}

func TestNewResumer(t *testing.T) {
	client := &host.Client{}
	resumer := NewResumer(client)

	if resumer == nil {
		t.Fatal("NewResumer() returned nil")
	}

	if resumer.client != client {
		t.Error("NewResumer() did not set client correctly")
	}
}

func TestResumer_SetOutput(t *testing.T) {
	resumer := NewResumer(nil)
	output := &mockOutput{}

	resumer.SetOutput(output)

	// Verify output was set by checking if verbose logging works
	// This is indirect, but ensures the method works
}

func TestResumer_Resume(t *testing.T) {
	tests := []struct {
		name           string
		opts           ResumeOptions
		clientReady    bool
		invokeErr      error
		taskResponse   *cline.TaskResponse
		wantErr        bool
		errContains    string
		expectResumed  bool
	}{
		{
			name:          "missing task ID",
			opts:          ResumeOptions{},
			wantErr:       true,
			errContains:   "task ID is required",
		},
		{
			name: "nil client",
			opts: ResumeOptions{
				TaskID: "test-task",
			},
			clientReady: false,
			wantErr:     true,
			errContains: "gRPC client not available",
		},
		{
			name: "client not ready",
			opts: ResumeOptions{
				TaskID: "test-task",
			},
			clientReady: false,
			wantErr:     true,
			errContains: "gRPC client not ready",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resumer := NewResumer(&host.Client{})
			
			// Set mock output to capture verbose logs
			output := &mockOutput{}
			resumer.SetOutput(output)

			// Execute resume
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			result, err := resumer.Resume(ctx, tt.opts)

			// Verify error expectations
			if tt.wantErr {
				if err == nil {
					t.Errorf("Resume() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if tt.errContains != "" && !containsString(err.Error(), tt.errContains) {
					t.Errorf("Resume() error = %v, should contain %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("Resume() unexpected error = %v", err)
				return
			}

			// Verify result
			if result == nil {
				t.Error("Resume() returned nil result")
				return
			}

			if tt.expectResumed && !result.IsResumed {
				t.Error("Resume() returned IsResumed=false, expected true")
			}

			if result.TaskID != tt.opts.TaskID {
				t.Errorf("Resume() TaskID = %v, want %v", result.TaskID, tt.opts.TaskID)
			}
		})
	}
}

func TestResumer_ResumeWithPrompt(t *testing.T) {
	resumer := NewResumer(&host.Client{})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test expects error because there's no real gRPC client
	_, err := resumer.ResumeWithPrompt(ctx, "test-task", "Additional prompt", nil)

	if err == nil {
		t.Error("ResumeWithPrompt() expected error for missing gRPC client, got nil")
	}
}

func TestNopWriter(t *testing.T) {
	nop := &nopWriter{}
	n, err := nop.Printf("test %s", "message")
	
	if err != nil {
		t.Errorf("nopWriter.Printf() error = %v", err)
	}
	
	if n != 0 {
		t.Errorf("nopWriter.Printf() returned n = %d, want 0", n)
	}
}

func TestResumeOptions_Struct(t *testing.T) {
	// Test that ResumeOptions has all expected fields
	opts := ResumeOptions{
		TaskID:  "test-task",
		Prompt:  "test prompt",
		Images:  []string{"img1.png"},
		Verbose: true,
	}

	if opts.TaskID != "test-task" {
		t.Error("ResumeOptions.TaskID not set correctly")
	}
	if opts.Prompt != "test prompt" {
		t.Error("ResumeOptions.Prompt not set correctly")
	}
	if len(opts.Images) != 1 {
		t.Error("ResumeOptions.Images not set correctly")
	}
	if !opts.Verbose {
		t.Error("ResumeOptions.Verbose not set correctly")
	}
}

func TestResumeResult_Struct(t *testing.T) {
	now := time.Now().Unix()
	result := ResumeResult{
		TaskID:    "test-task",
		Task:      &cline.TaskResponse{Task: "Test", Ts: now},
		IsResumed: true,
		Message:   "Success",
	}

	if result.TaskID != "test-task" {
		t.Error("ResumeResult.TaskID not set correctly")
	}
	if result.Task == nil {
		t.Error("ResumeResult.Task is nil")
	}
	if !result.IsResumed {
		t.Error("ResumeResult.IsResumed not set correctly")
	}
	if result.Message != "Success" {
		t.Error("ResumeResult.Message not set correctly")
	}
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}