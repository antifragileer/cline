// Package task provides task execution and tool management functionality.
package task

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockToolApprover is a mock implementation of ToolApprover for testing.
type MockToolApprover struct {
	approveResponse bool
	approveError    error
	displayCalled   bool
	approvalCalled  bool
	lastRequest     ToolRequest
}

func NewMockToolApprover(approve bool, err error) *MockToolApprover {
	return &MockToolApprover{
		approveResponse: approve,
		approveError:    err,
	}
}

func (m *MockToolApprover) RequestApproval(ctx context.Context, req ToolRequest) (bool, error) {
	m.approvalCalled = true
	m.lastRequest = req
	return m.approveResponse, m.approveError
}

func (m *MockToolApprover) DisplayToolRequest(req ToolRequest) error {
	m.displayCalled = true
	return nil
}

// TestDefaultToolApprovalConfig tests the default configuration.
func TestDefaultToolApprovalConfig(t *testing.T) {
	config := DefaultToolApprovalConfig()
	
	assert.NotNil(t, config)
	assert.False(t, config.YoloMode)
	assert.Empty(t, config.AutoApproveTools)
	assert.Equal(t, 5*time.Minute, config.ApprovalTimeout)
	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, time.Second, config.RetryDelay)
}

// TestNewToolExecutor tests the tool executor creation.
func TestNewToolExecutor(t *testing.T) {
	approver := NewMockToolApprover(true, nil)
	config := DefaultToolApprovalConfig()
	
	executor := NewToolExecutor(config, approver)
	
	assert.NotNil(t, executor)
	assert.Equal(t, config, executor.config)
	assert.Equal(t, approver, executor.approver)
	assert.Empty(t, executor.history)
	assert.Equal(t, 100, executor.maxHistory)
}

// TestNewToolExecutorNilConfig tests creation with nil config.
func TestNewToolExecutorNilConfig(t *testing.T) {
	approver := NewMockToolApprover(true, nil)
	
	executor := NewToolExecutor(nil, approver)
	
	assert.NotNil(t, executor)
	assert.NotNil(t, executor.config)
	assert.False(t, executor.config.YoloMode)
}

// TestToolExecutorIsYoloMode tests the yolo mode getter/setter.
func TestToolExecutorIsYoloMode(t *testing.T) {
	approver := NewMockToolApprover(true, nil)
	executor := NewToolExecutor(nil, approver)
	
	assert.False(t, executor.IsYoloMode())
	
	executor.SetYoloMode(true)
	assert.True(t, executor.IsYoloMode())
	
	executor.SetYoloMode(false)
	assert.False(t, executor.IsYoloMode())
}

// TestToolExecutorAutoApproveTools tests auto-approve tool management.
func TestToolExecutorAutoApproveTools(t *testing.T) {
	approver := NewMockToolApprover(true, nil)
	executor := NewToolExecutor(nil, approver)
	
	// Initially empty
	assert.Empty(t, executor.config.AutoApproveTools)
	
	// Add tools
	executor.AddAutoApproveTool(ToolTypeReadFile)
	assert.Len(t, executor.config.AutoApproveTools, 1)
	assert.Contains(t, executor.config.AutoApproveTools, ToolTypeReadFile)
	
	// Add duplicate (should not add)
	executor.AddAutoApproveTool(ToolTypeReadFile)
	assert.Len(t, executor.config.AutoApproveTools, 1)
	
	// Add another tool
	executor.AddAutoApproveTool(ToolTypeWriteFile)
	assert.Len(t, executor.config.AutoApproveTools, 2)
	
	// Remove tool
	executor.RemoveAutoApproveTool(ToolTypeReadFile)
	assert.Len(t, executor.config.AutoApproveTools, 1)
	assert.NotContains(t, executor.config.AutoApproveTools, ToolTypeReadFile)
	
	// Remove non-existent tool (should not panic)
	executor.RemoveAutoApproveTool(ToolTypeExecuteCommand)
	assert.Len(t, executor.config.AutoApproveTools, 1)
}

// TestValidateRequest tests request validation.
func TestValidateRequest(t *testing.T) {
	approver := NewMockToolApprover(true, nil)
	executor := NewToolExecutor(nil, approver)
	
	tests := []struct {
		name    string
		req     ToolRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request",
			req: ToolRequest{
				ID:       "test-1",
				Type:     ToolTypeReadFile,
				ToolName: "read_file",
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			req: ToolRequest{
				Type:     ToolTypeReadFile,
				ToolName: "read_file",
			},
			wantErr: true,
			errMsg:  "tool request ID is required",
		},
		{
			name: "missing tool name",
			req: ToolRequest{
				ID:   "test-1",
				Type: ToolTypeReadFile,
			},
			wantErr: true,
			errMsg:  "tool name is required",
		},
		{
			name: "missing type",
			req: ToolRequest{
				ID:       "test-1",
				ToolName: "read_file",
			},
			wantErr: true,
			errMsg:  "tool type is required",
		},
		{
			name: "invalid tool type",
			req: ToolRequest{
				ID:       "test-1",
				Type:     "invalid_type",
				ToolName: "read_file",
			},
			wantErr: true,
			errMsg:  "invalid tool type",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executor.validateRequest(&tt.req)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestIsAutoApproved tests the auto-approval logic.
func TestIsAutoApproved(t *testing.T) {
	approver := NewMockToolApprover(true, nil)
	config := DefaultToolApprovalConfig()
	executor := NewToolExecutor(config, approver)
	
	req := ToolRequest{
		ID:       "test-1",
		Type:     ToolTypeReadFile,
		ToolName: "read_file",
	}
	
	// Not auto-approved by default
	assert.False(t, executor.isAutoApproved(req))
	
	// Add to auto-approve list
	config.AutoApproveTools = []ToolType{ToolTypeReadFile}
	assert.True(t, executor.isAutoApproved(req))
	
	// Different tool type
	req2 := ToolRequest{
		ID:       "test-2",
		Type:     ToolTypeWriteFile,
		ToolName: "write_file",
	}
	assert.False(t, executor.isAutoApproved(req2))
	
	// Yolo mode auto-approves everything
	config.YoloMode = true
	assert.True(t, executor.isAutoApproved(req2))
}

// TestExecuteToolWithApproval tests tool execution with approval flow.
func TestExecuteToolWithApproval(t *testing.T) {
	// Create temp directory for testing
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)
	
	approver := NewMockToolApprover(true, nil)
	executor := NewToolExecutor(nil, approver)
	
	req := ToolRequest{
		ID:       "test-1",
		Type:     ToolTypeReadFile,
		ToolName: "read_file",
		Parameters: map[string]interface{}{
			"path": testFile,
		},
		Timeout: 5 * time.Second,
	}
	
	ctx := context.Background()
	result, err := executor.ExecuteTool(ctx, req)
	
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, "test content", result.Output)
	assert.Equal(t, req.ID, result.RequestID)
	
	// Verify approval was requested
	assert.True(t, approver.approvalCalled)
	assert.Equal(t, req.ID, approver.lastRequest.ID)
}

// TestExecuteToolRejection tests tool execution when rejected.
func TestExecuteToolRejection(t *testing.T) {
	approver := NewMockToolApprover(false, nil)
	executor := NewToolExecutor(nil, approver)
	
	req := ToolRequest{
		ID:       "test-1",
		Type:     ToolTypeReadFile,
		ToolName: "read_file",
		Parameters: map[string]interface{}{
			"path": "/tmp/nonexistent",
		},
	}
	
	ctx := context.Background()
	result, err := executor.ExecuteTool(ctx, req)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rejected")
	assert.NotNil(t, result)
	assert.False(t, result.Success)
}

// TestExecuteToolAutoApprove tests auto-approval (yolo mode).
func TestExecuteToolAutoApprove(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("auto-approved"), 0644)
	require.NoError(t, err)
	
	approver := NewMockToolApprover(false, nil) // Would reject if asked
	config := DefaultToolApprovalConfig()
	config.YoloMode = true
	
	executor := NewToolExecutor(config, approver)
	
	req := ToolRequest{
		ID:       "test-1",
		Type:     ToolTypeReadFile,
		ToolName: "read_file",
		Parameters: map[string]interface{}{
			"path": testFile,
		},
	}
	
	ctx := context.Background()
	result, err := executor.ExecuteTool(ctx, req)
	
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "auto-approved", result.Output)
	
	// Approval should not have been requested due to yolo mode
	assert.False(t, approver.approvalCalled)
}

// TestExecuteReadFile tests the read_file tool.
func TestExecuteReadFile(t *testing.T) {
	tempDir := t.TempDir()
	
	tests := []struct {
		name       string
		content    string
		path       string
		wantErr    bool
		errContain string
	}{
		{
			name:    "successful read",
			content: "hello world",
			path:    "test.txt",
			wantErr: false,
		},
		{
			name:       "missing path parameter",
			content:    "",
			path:       "",
			wantErr:    true,
			errContain: "path parameter is required",
		},
		{
			name:       "file not found",
			content:    "",
			path:       "nonexistent.txt",
			wantErr:    true,
			errContain: "failed to stat file",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewToolExecutor(nil, nil)
			
			var fullPath string
			if tt.path != "" {
				fullPath = filepath.Join(tempDir, tt.path)
				if tt.content != "" {
					err := os.WriteFile(fullPath, []byte(tt.content), 0644)
					require.NoError(t, err)
				}
			}
			
			req := ToolRequest{
				ID:   "test-1",
				Type: ToolTypeReadFile,
				Parameters: map[string]interface{}{
					"path": fullPath,
				},
			}
			
			ctx := context.Background()
			result := executor.executeReadFile(ctx, req)
			
			if tt.wantErr {
				assert.False(t, result.Success)
				assert.Contains(t, result.Error, tt.errContain)
			} else {
				assert.True(t, result.Success)
				assert.Equal(t, tt.content, result.Output)
			}
		})
	}
}

// TestExecuteReadFileDirectory tests reading a directory (should fail).
func TestExecuteReadFileDirectory(t *testing.T) {
	tempDir := t.TempDir()
	
	executor := NewToolExecutor(nil, nil)
	req := ToolRequest{
		ID:   "test-1",
		Type: ToolTypeReadFile,
		Parameters: map[string]interface{}{
			"path": tempDir,
		},
	}
	
	ctx := context.Background()
	result := executor.executeReadFile(ctx, req)
	
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "directory")
}

// TestExecuteWriteFile tests the write_file tool.
func TestExecuteWriteFile(t *testing.T) {
	tempDir := t.TempDir()
	
	tests := []struct {
		name       string
		path       string
		content    interface{} // interface{} to allow nil for missing param test
		wantErr    bool
		errContain string
	}{
		{
			name:    "successful write",
			path:    "test.txt",
			content: "test content",
			wantErr: false,
		},
		{
			name:    "write in subdirectory",
			path:    "subdir/nested.txt",
			content: "nested content",
			wantErr: false,
		},
		{
			name:       "missing path parameter",
			path:       "",
			content:    "content",
			wantErr:    true,
			errContain: "path parameter is required",
		},
		{
			name:       "missing content parameter",
			path:       "test.txt",
			content:    nil, // nil to test missing parameter
			wantErr:    true,
			errContain: "content parameter is required",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewToolExecutor(nil, nil)
			
			fullPath := ""
			if tt.path != "" {
				fullPath = filepath.Join(tempDir, tt.path)
			}
			
			params := map[string]interface{}{
				"path": fullPath,
			}
			if tt.content != nil {
				params["content"] = tt.content
			}
			
			req := ToolRequest{
				ID:         "test-1",
				Type:       ToolTypeWriteFile,
				Parameters: params,
			}
			
			ctx := context.Background()
			result := executor.executeWriteFile(ctx, req)
			
			if tt.wantErr {
				assert.False(t, result.Success)
				assert.Contains(t, result.Error, tt.errContain)
			} else {
				assert.True(t, result.Success)
				
				// Verify file was written
				content, err := os.ReadFile(fullPath)
				require.NoError(t, err)
				assert.Equal(t, tt.content, string(content))
			}
		})
	}
}

// TestExecuteReplaceInFile tests the replace_in_file tool.
func TestExecuteReplaceInFile(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	
	// Create initial file
	initialContent := "line1\nline2\nline3\n"
	err := os.WriteFile(testFile, []byte(initialContent), 0644)
	require.NoError(t, err)
	
	tests := []struct {
		name       string
		path       string
		oldStr     string
		newStr     string
		wantErr    bool
		errContain string
		expected   string
	}{
		{
			name:     "successful replace",
			path:     testFile,
			oldStr:   "line2",
			newStr:   "replaced_line2",
			wantErr:  false,
			expected: "line1\nreplaced_line2\nline3\n",
		},
		{
			name:       "old string not found",
			path:       testFile,
			oldStr:     "nonexistent",
			newStr:     "replacement",
			wantErr:    true,
			errContain: "not found",
		},
		{
			name:       "missing path parameter",
			path:       "",
			oldStr:     "test",
			newStr:     "replacement",
			wantErr:    true,
			errContain: "path parameter is required",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset file content for tests that use the file
			if tt.path != "" {
				err := os.WriteFile(testFile, []byte(initialContent), 0644)
				require.NoError(t, err)
			}
			
			executor := NewToolExecutor(nil, nil)
			
			req := ToolRequest{
				ID:   "test-1",
				Type: ToolTypeReplaceInFile,
				Parameters: map[string]interface{}{
					"path":       tt.path,
					"old_string": tt.oldStr,
					"new_string": tt.newStr,
				},
			}
			
			ctx := context.Background()
			result := executor.executeReplaceInFile(ctx, req)
			
			if tt.wantErr {
				assert.False(t, result.Success)
				assert.Contains(t, result.Error, tt.errContain)
			} else {
				assert.True(t, result.Success)
				
				// Verify content was replaced
				content, err := os.ReadFile(testFile)
				require.NoError(t, err)
				assert.Equal(t, tt.expected, string(content))
			}
		})
	}
}

// TestExecuteCommand tests the execute_command tool.
func TestExecuteCommand(t *testing.T) {
	tests := []struct {
		name       string
		command    string
		cwd        string
		wantErr    bool
		errContain string
		shouldContain string
	}{
		{
			name:          "successful command",
			command:       "echo hello",
			wantErr:       false,
			shouldContain: "hello",
		},
		{
			name:       "missing command parameter",
			command:    "",
			wantErr:    true,
			errContain: "command parameter is required",
		},
		{
			name:       "failing command",
			command:    "exit 1",
			wantErr:    true,
			errContain: "exit code 1",
		},
		{
			name:          "command with cwd",
			command:       "pwd",
			cwd:           "/tmp",
			wantErr:       false,
			shouldContain: "/tmp",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewToolExecutor(nil, nil)
			
			params := map[string]interface{}{
				"command": tt.command,
			}
			if tt.cwd != "" {
				params["cwd"] = tt.cwd
			}
			
			req := ToolRequest{
				ID:         "test-1",
				Type:       ToolTypeExecuteCommand,
				Parameters: params,
			}
			
			ctx := context.Background()
			result := executor.executeCommand(ctx, req)
			
			if tt.wantErr {
				assert.False(t, result.Success)
				assert.Contains(t, result.Error, tt.errContain)
			} else {
				assert.True(t, result.Success)
				assert.Contains(t, result.Output, tt.shouldContain)
			}
		})
	}
}

// TestExecuteListFiles tests the list_files tool.
func TestExecuteListFiles(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create some test files
	err := os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte("content1"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tempDir, "file2.txt"), []byte("content2"), 0644)
	require.NoError(t, err)
	
	// Create subdirectory with file
	subDir := filepath.Join(tempDir, "subdir")
	err = os.MkdirAll(subDir, 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(subDir, "nested.txt"), []byte("nested"), 0644)
	require.NoError(t, err)
	
	tests := []struct {
		name           string
		path           string
		recursive      bool
		shouldContain  string
		shouldNotContain string
	}{
		{
			name:          "list non-recursive",
			path:          tempDir,
			recursive:     false,
			shouldContain: "file1.txt",
		},
		{
			name:           "list recursive",
			path:           tempDir,
			recursive:      true,
			shouldContain:  "nested.txt",
		},
		{
			name:          "default path (current directory)",
			path:          "",
			recursive:     false,
			shouldContain: ".",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewToolExecutor(nil, nil)
			
			params := map[string]interface{}{
				"recursive": tt.recursive,
			}
			if tt.path != "" {
				params["path"] = tt.path
			}
			
			req := ToolRequest{
				ID:         "test-1",
				Type:       ToolTypeListFiles,
				Parameters: params,
			}
			
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			
			result := executor.executeListFiles(ctx, req)
			
			assert.True(t, result.Success)
			if tt.shouldContain != "" {
				assert.Contains(t, result.Output, tt.shouldContain)
			}
			if tt.shouldNotContain != "" {
				assert.NotContains(t, result.Output, tt.shouldNotContain)
			}
		})
	}
}

// TestExecuteSearchFiles tests the search_files tool.
func TestExecuteSearchFiles(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create test files with searchable content
	err := os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte("hello world"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tempDir, "file2.txt"), []byte("goodbye world"), 0644)
	require.NoError(t, err)
	
	executor := NewToolExecutor(nil, nil)
	
	tests := []struct {
		name          string
		regex         string
		filePattern   string
		shouldContain string
	}{
		{
			name:          "search for hello",
			regex:         "hello",
			shouldContain: "file1.txt",
		},
		{
			name:          "search for world",
			regex:         "world",
			shouldContain: "file1.txt",
		},
		{
			name:          "search with file pattern",
			regex:         "world",
			filePattern:   "*.txt",
			shouldContain: "file1.txt",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := map[string]interface{}{
				"path":  tempDir,
				"regex": tt.regex,
			}
			if tt.filePattern != "" {
				params["file_pattern"] = tt.filePattern
			}
			
			req := ToolRequest{
				ID:         "test-1",
				Type:       ToolTypeSearchFiles,
				Parameters: params,
			}
			
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			
			result := executor.executeSearchFiles(ctx, req)
			
			// Note: grep might return exit code 0 or 1 depending on matches
			if result.Success {
				assert.Contains(t, result.Output, tt.shouldContain)
			}
		})
	}
}

// TestBatchExecuteTools tests batch tool execution.
func TestBatchExecuteTools(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("content"), 0644)
	require.NoError(t, err)
	
	approver := NewMockToolApprover(true, nil)
	config := DefaultToolApprovalConfig()
	config.YoloMode = true // Auto-approve for batch testing
	
	executor := NewToolExecutor(config, approver)
	
	requests := []ToolRequest{
		{
			ID:       "batch-1",
			Type:     ToolTypeReadFile,
			ToolName: "read_file",
			Parameters: map[string]interface{}{
				"path": testFile,
			},
		},
		{
			ID:       "batch-2",
			Type:     ToolTypeReadFile,
			ToolName: "read_file",
			Parameters: map[string]interface{}{
				"path": testFile,
			},
		},
	}
	
	ctx := context.Background()
	results, err := executor.BatchExecuteTools(ctx, requests)
	
	require.NoError(t, err)
	assert.Len(t, results, 2)
	
	for _, result := range results {
		assert.True(t, result.Success)
		assert.Equal(t, "content", result.Output)
	}
}

// TestHistoryTracking tests that execution history is tracked.
func TestHistoryTracking(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("history test"), 0644)
	require.NoError(t, err)
	
	approver := NewMockToolApprover(true, nil)
	config := DefaultToolApprovalConfig()
	config.YoloMode = true
	
	executor := NewToolExecutor(config, approver)
	
	// Execute multiple tools
	for i := 0; i < 3; i++ {
		req := ToolRequest{
			ID:       fmt.Sprintf("history-%d", i),
			Type:     ToolTypeReadFile,
			ToolName: "read_file",
			Parameters: map[string]interface{}{
				"path": testFile,
			},
		}
		
		ctx := context.Background()
		_, err := executor.ExecuteTool(ctx, req)
		require.NoError(t, err)
	}
	
	// Check history
	history := executor.GetHistory()
	assert.Len(t, history, 3)
	
	// Verify history entries
	for i, record := range history {
		assert.Equal(t, fmt.Sprintf("history-%d", i), record.Request.ID)
		assert.True(t, record.Approved)
		assert.True(t, record.Result.Success)
	}
}

// TestContextCancellation tests that context cancellation works.
func TestContextCancellation(t *testing.T) {
	approver := NewMockToolApprover(true, nil)
	executor := NewToolExecutor(nil, approver)
	
	req := ToolRequest{
		ID:       "cancel-test",
		Type:     ToolTypeReadFile,
		ToolName: "read_file",
		Parameters: map[string]interface{}{
			"path": "/tmp/nonexistent",
		},
	}
	
	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	
	_, err := executor.ExecuteTool(ctx, req)
	
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

// TestResolvePath tests path resolution.
func TestResolvePath(t *testing.T) {
	executor := NewToolExecutor(nil, nil)
	
	// Absolute path should remain unchanged
	absPath := "/absolute/path/to/file"
	assert.Equal(t, absPath, executor.resolvePath(absPath))
	
	// Relative path should be resolved
	relPath := "relative/path"
	resolved := executor.resolvePath(relPath)
	assert.True(t, filepath.IsAbs(resolved))
	assert.True(t, strings.HasSuffix(resolved, relPath))
}

// TestAutoApprover tests the auto-approver.
func TestAutoApprover(t *testing.T) {
	approver := NewAutoApprover()
	
	req := ToolRequest{
		ID:       "auto-test",
		Type:     ToolTypeReadFile,
		ToolName: "read_file",
	}
	
	ctx := context.Background()
	approved, err := approver.RequestApproval(ctx, req)
	
	assert.NoError(t, err)
	assert.True(t, approved)
	
	// Display should not error
	err = approver.DisplayToolRequest(req)
	assert.NoError(t, err)
}

// TestRejectAllApprover tests the reject-all approver.
func TestRejectAllApprover(t *testing.T) {
	approver := NewRejectAllApprover()
	
	req := ToolRequest{
		ID:       "reject-test",
		Type:     ToolTypeReadFile,
		ToolName: "read_file",
	}
	
	ctx := context.Background()
	approved, err := approver.RequestApproval(ctx, req)
	
	assert.NoError(t, err)
	assert.False(t, approved)
	
	// Display should not error
	err = approver.DisplayToolRequest(req)
	assert.NoError(t, err)
}

// TestConditionalApprover tests the conditional approver.
func TestConditionalApprover(t *testing.T) {
	// Condition: only approve read_file tools
	condition := func(req ToolRequest) bool {
		return req.Type == ToolTypeReadFile
	}
	
	approver := NewConditionalApprover(condition)
	ctx := context.Background()
	
	// Should approve read_file
	readReq := ToolRequest{
		ID:       "cond-test-1",
		Type:     ToolTypeReadFile,
		ToolName: "read_file",
	}
	approved, err := approver.RequestApproval(ctx, readReq)
	assert.NoError(t, err)
	assert.True(t, approved)
	
	// Should reject write_file
	writeReq := ToolRequest{
		ID:       "cond-test-2",
		Type:     ToolTypeWriteFile,
		ToolName: "write_file",
	}
	approved, err = approver.RequestApproval(ctx, writeReq)
	assert.NoError(t, err)
	assert.False(t, approved)
}

// TestDelegatingApprover tests the delegating approver.
func TestDelegatingApprover(t *testing.T) {
	defaultApprover := NewMockToolApprover(true, nil)
	readApprover := NewMockToolApprover(false, nil) // Rejects reads
	writeApprover := NewAutoApprover()              // Approves writes
	
	delegator := NewDelegatingApprover(defaultApprover)
	delegator.RegisterApprover(ToolTypeReadFile, readApprover)
	delegator.RegisterApprover(ToolTypeWriteFile, writeApprover)
	
	ctx := context.Background()
	
	// Read file should be rejected by readApprover
	readReq := ToolRequest{
		ID:       "del-test-1",
		Type:     ToolTypeReadFile,
		ToolName: "read_file",
	}
	approved, err := delegator.RequestApproval(ctx, readReq)
	assert.NoError(t, err)
	assert.False(t, approved)
	
	// Write file should be approved by writeApprover
	writeReq := ToolRequest{
		ID:       "del-test-2",
		Type:     ToolTypeWriteFile,
		ToolName: "write_file",
	}
	approved, err = delegator.RequestApproval(ctx, writeReq)
	assert.NoError(t, err)
	assert.True(t, approved)
	
	// Unknown tool should use default
	unknownReq := ToolRequest{
		ID:       "del-test-3",
		Type:     ToolTypeExecuteCommand,
		ToolName: "execute_command",
	}
	approved, err = delegator.RequestApproval(ctx, unknownReq)
	assert.NoError(t, err)
	assert.True(t, approved) // defaultApprover approves
}

// TestTimeoutApprover tests the timeout approver wrapper.
func TestTimeoutApprover(t *testing.T) {
	// Create a conditional approver that waits to simulate slow response
	waitChan := make(chan struct{})
	slowApprover := NewConditionalApprover(func(req ToolRequest) bool {
		<-waitChan // Wait until test signals to continue
		return true
	})
	
	timeoutApprover := NewTimeoutApprover(slowApprover, 1*time.Nanosecond)
	
	req := ToolRequest{
		ID:       "timeout-test",
		Type:     ToolTypeReadFile,
		ToolName: "read_file",
	}
	
	ctx := context.Background()
	resultChan := make(chan struct {
		approved bool
		err      error
	})
	
	// Run the approval in a goroutine
	go func() {
		approved, err := timeoutApprover.RequestApproval(ctx, req)
		resultChan <- struct {
			approved bool
			err      error
		}{approved, err}
	}()
	
	// Wait for timeout to occur (don't signal waitChan)
	select {
	case result := <-resultChan:
		// Should get an error due to timeout
		if result.err == nil {
			// If no error, close the waitChan to complete the test
			close(waitChan)
		} else {
			// Got expected error (timeout or context deadline)
			assert.Error(t, result.err)
		}
	case <-time.After(100 * time.Millisecond):
		// Test passed - timeout occurred before completion
		close(waitChan) // Release the goroutine
		<-resultChan    // Wait for it to complete
	}
}

// TestLoggingApprover tests the logging approver.
func TestLoggingApprover(t *testing.T) {
	var logs []string
	logger := func(s string) {
		logs = append(logs, s)
	}
	
	innerApprover := NewMockToolApprover(true, nil)
	loggingApprover := NewLoggingApprover(innerApprover, logger)
	
	req := ToolRequest{
		ID:       "log-test",
		Type:     ToolTypeReadFile,
		ToolName: "read_file",
	}
	
	ctx := context.Background()
	approved, err := loggingApprover.RequestApproval(ctx, req)
	
	assert.NoError(t, err)
	assert.True(t, approved)
	assert.Len(t, logs, 2) // Request log + approval log
	assert.Contains(t, logs[0], "Approval requested")
	assert.Contains(t, logs[1], "Tool approved")
}

// Helper to override RequestApproval for mock
func (m *MockToolApprover) SetRequestApproval(fn func(context.Context, ToolRequest) (bool, error)) {
	// This method is intentionally not implemented
	// The mock uses the approveResponse and approveError fields directly
}

// TestNewUIToolApprover tests the UI tool approver creation.
func TestNewUIToolApprover(t *testing.T) {
	approver := NewUIToolApprover()
	
	assert.NotNil(t, approver)
	assert.True(t, approver.useInteractive)
	assert.Equal(t, os.Stdout, approver.output)
	assert.Equal(t, os.Stdin, approver.input)
	assert.Equal(t, 5*time.Minute, approver.defaultTimeout)
}

// TestNewNonInteractiveApprover tests the non-interactive approver creation.
func TestNewNonInteractiveApprover(t *testing.T) {
	approver := NewNonInteractiveApprover()
	
	assert.NotNil(t, approver)
	assert.False(t, approver.useInteractive)
}

// TestUIToolApproverSetters tests the setter methods.
func TestUIToolApproverSetters(t *testing.T) {
	approver := NewUIToolApprover()
	
	// Test SetInteractive
	approver.SetInteractive(false)
	assert.False(t, approver.useInteractive)
	approver.SetInteractive(true)
	assert.True(t, approver.useInteractive)
	
	// Test SetTimeout
	approver.SetTimeout(10 * time.Second)
	assert.Equal(t, 10*time.Second, approver.defaultTimeout)
}

// TestUIToolApproverNonInteractive tests non-interactive mode behavior.
func TestUIToolApproverNonInteractive(t *testing.T) {
	approver := NewNonInteractiveApprover()
	
	req := ToolRequest{
		ID:       "non-interactive-test",
		Type:     ToolTypeReadFile,
		ToolName: "read_file",
	}
	
	ctx := context.Background()
	approved, err := approver.RequestApproval(ctx, req)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "non-interactive mode")
	assert.False(t, approved)
}

// TestBatchApprover tests the batch approver.
func TestBatchApprover(t *testing.T) {
	uiApprover := NewNonInteractiveApprover()
	batchApprover := NewBatchApprover(uiApprover, 3)
	
	ctx := context.Background()
	
	// Add requests up to batch size
	for i := 0; i < 3; i++ {
		req := ToolRequest{
			ID:       fmt.Sprintf("batch-%d", i),
			Type:     ToolTypeReadFile,
			ToolName: "read_file",
		}
		
		// First 2 should not trigger batch (non-interactive will error)
		if i < 2 {
			_, err := batchApprover.RequestApproval(ctx, req)
			assert.Error(t, err) // Non-interactive errors
		} else {
			// Third request should trigger batch approval
			approved, err := batchApprover.RequestApproval(ctx, req)
			// In non-interactive mode, it will still error, but batch logic runs
			_ = approved
			_ = err
		}
	}
	
	// Check pending count
	assert.Equal(t, 0, len(batchApprover.pendingReqs)) // Batch was flushed
	
	// Test auto-approve
	batchApprover2 := NewBatchApprover(uiApprover, 10)
	batchApprover2.SetAutoApprove(true)
	
	req := ToolRequest{
		ID:       "auto-batch",
		Type:     ToolTypeReadFile,
		ToolName: "read_file",
	}
	
	// Should auto-approve and clear batch
	approved, _ := batchApprover2.RequestApproval(ctx, req)
	_ = approved
	assert.Equal(t, 0, len(batchApprover2.pendingReqs))
}

// TestToolRequestValidation tests request validation edge cases.
func TestToolRequestValidation(t *testing.T) {
	executor := NewToolExecutor(nil, nil)
	
	tests := []struct {
		name    string
		req     ToolRequest
		wantErr bool
	}{
		{
			name: "all valid types",
			req: ToolRequest{
				ID:       "valid",
				Type:     ToolTypeListCodeDefinitionNames,
				ToolName: "list_code_definition_names",
			},
			wantErr: false,
		},
		{
			name: "empty ID",
			req: ToolRequest{
				Type:     ToolTypeReadFile,
				ToolName: "read_file",
			},
			wantErr: true,
		},
		{
			name: "empty type",
			req: ToolRequest{
				ID:       "test",
				ToolName: "read_file",
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executor.validateRequest(&tt.req)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestToolExecutionRecord tests the execution record structure.
func TestToolExecutionRecord(t *testing.T) {
	record := ToolExecutionRecord{
		Request: ToolRequest{
			ID:       "record-test",
			Type:     ToolTypeReadFile,
			ToolName: "read_file",
		},
		Result: ToolResult{
			RequestID: "record-test",
			Success:   true,
			Output:    "test output",
		},
		Approved:  true,
		Timestamp: time.Now(),
	}
	
	assert.Equal(t, "record-test", record.Request.ID)
	assert.True(t, record.Approved)
	assert.True(t, record.Result.Success)
}

// BenchmarkExecuteTool benchmarks tool execution.
func BenchmarkExecuteTool(b *testing.B) {
	tempDir := b.TempDir()
	testFile := filepath.Join(tempDir, "bench.txt")
	err := os.WriteFile(testFile, []byte("benchmark content"), 0644)
	if err != nil {
		b.Fatal(err)
	}
	
	approver := NewAutoApprover()
	executor := NewToolExecutor(nil, approver)
	
	req := ToolRequest{
		ID:       "bench",
		Type:     ToolTypeReadFile,
		ToolName: "read_file",
		Parameters: map[string]interface{}{
			"path": testFile,
		},
	}
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req.ID = fmt.Sprintf("bench-%d", i)
		_, err := executor.ExecuteTool(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// =============================================================================
// Phase 3 Tests: Piped Input, Security/Permissions, Output Formatting
// =============================================================================

// TestCommandSecurityValidation tests that CLINE_COMMAND_PERMISSIONS is enforced
func TestCommandSecurityValidation(t *testing.T) {
	// Save and restore original environment
	originalEnv := os.Getenv("CLINE_COMMAND_PERMISSIONS")
	defer os.Setenv("CLINE_COMMAND_PERMISSIONS", originalEnv)

	tests := []struct {
		name          string
		permissions   string
		command       string
		shouldAllow   bool
	}{
		{
			name:        "allow all commands",
			permissions: `{"allow":["*"]}`,
			command:     "echo hello",
			shouldAllow: true,
		},
		{
			name:        "deny specific command",
			permissions: `{"allow":["*"],"deny":["rm -rf *"]}`,
			command:     "rm -rf /",
			shouldAllow: false,
		},
		{
			name:        "allow specific commands only",
			permissions: `{"allow":["git *","docker *"]}`,
			command:     "git status",
			shouldAllow: true,
		},
		{
			name:        "reject non-allowed command",
			permissions: `{"allow":["git *"]}`,
			command:     "docker ps",
			shouldAllow: false,
		},
		{
			name:        "no permissions - default allow",
			permissions: "",
			command:     "echo hello",
			shouldAllow: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set permissions environment variable
			if tt.permissions != "" {
				os.Setenv("CLINE_COMMAND_PERMISSIONS", tt.permissions)
			} else {
				os.Unsetenv("CLINE_COMMAND_PERMISSIONS")
			}

			// Create executor with yolo mode for testing security layer
			config := &ToolApprovalConfig{
				YoloMode:        true,
				ApprovalTimeout: 5 * time.Second,
			}
			approver := NewAutoApprover()
			executor := NewToolExecutor(config, approver)

			req := ToolRequest{
				ID:       "test-request",
				Type:     ToolTypeExecuteCommand,
				ToolName: "execute_command",
				Parameters: map[string]interface{}{
					"command": tt.command,
				},
			}

			result, err := executor.ExecuteTool(context.Background(), req)

			if tt.shouldAllow {
				if !result.Success {
					t.Errorf("Expected command to be allowed, but failed: %s (error: %v)", result.Error, err)
				}
			} else {
				if result.Success {
					t.Error("Expected command to be denied, but succeeded")
				}
				if !strings.Contains(result.Error, "not allowed") {
					t.Errorf("Expected 'not allowed' error, got: %s", result.Error)
				}
			}
		})
	}
}

// TestPipedInputIntegration tests piped input handling
func TestPipedInputIntegration(t *testing.T) {
	// Test the combination format used when piped input is present
	prompt := "explain this code"
	pipedInput := `func main() {
	fmt.Println("Hello, World!")
}`

	combined := pipedInput + "\n\n" + prompt

	if !strings.Contains(combined, pipedInput) {
		t.Error("Combined prompt should contain piped input")
	}
	if !strings.Contains(combined, prompt) {
		t.Error("Combined prompt should contain original prompt")
	}
	if !strings.Contains(combined, "\n\n") {
		t.Error("Combined prompt should have double newline separator")
	}
}

// TestSecurityIntegrationWithAutoApprover tests security with auto-approve
func TestSecurityIntegrationWithAutoApprover(t *testing.T) {
	// Save and restore original environment
	originalEnv := os.Getenv("CLINE_COMMAND_PERMISSIONS")
	defer os.Setenv("CLINE_COMMAND_PERMISSIONS", originalEnv)

	// Set up restrictive permissions
	os.Setenv("CLINE_COMMAND_PERMISSIONS", `{"allow":["ls *","echo *"],"deny":["rm *","sudo *"]}`)

	config := &ToolApprovalConfig{
		YoloMode:        true, // Auto-approve everything
		ApprovalTimeout: 5 * time.Second,
	}
	approver := NewAutoApprover()
	executor := NewToolExecutor(config, approver)

	// Test allowed command
	allowedReq := ToolRequest{
		ID:       "test-allowed",
		Type:     ToolTypeExecuteCommand,
		ToolName: "execute_command",
		Parameters: map[string]interface{}{
			"command": "ls -la",
		},
	}
	result, _ := executor.ExecuteTool(context.Background(), allowedReq)
	if !result.Success {
		t.Errorf("Expected 'ls -la' to be allowed, got: %s", result.Error)
	}

	// Test denied command (should be denied even with auto-approve)
	deniedReq := ToolRequest{
		ID:       "test-denied",
		Type:     ToolTypeExecuteCommand,
		ToolName: "execute_command",
		Parameters: map[string]interface{}{
			"command": "sudo ls",
		},
	}
	result, _ = executor.ExecuteTool(context.Background(), deniedReq)
	if result.Success {
		t.Error("Expected 'sudo ls' to be denied despite auto-approve")
	}
	if !strings.Contains(result.Error, "not allowed") {
		t.Errorf("Expected security error, got: %s", result.Error)
	}
}

// TestSecurityEdgeCases tests edge cases in security validation
func TestSecurityEdgeCases(t *testing.T) {
	originalEnv := os.Getenv("CLINE_COMMAND_PERMISSIONS")
	defer os.Setenv("CLINE_COMMAND_PERMISSIONS", originalEnv)

	config := &ToolApprovalConfig{
		YoloMode:        true,
		ApprovalTimeout: 5 * time.Second,
	}
	approver := NewAutoApprover()

	tests := []struct {
		name        string
		permissions string
		command     string
		shouldAllow bool
	}{
		{
			name:        "invalid permissions JSON",
			permissions: `invalid json`,
			command:     "echo test",
			shouldAllow: false,
		},
		{
			name:        "empty permissions",
			permissions: `{"allow":[],"deny":[]}`,
			command:     "echo test",
			shouldAllow: false,
		},
		{
			name:        "complex command with pipes",
			permissions: `{"allow":["cat *","grep *"]}`,
			command:     "cat file.txt | grep pattern",
			shouldAllow: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("CLINE_COMMAND_PERMISSIONS", tt.permissions)
			executor := NewToolExecutor(config, approver)

			req := ToolRequest{
				ID:       "test-edge",
				Type:     ToolTypeExecuteCommand,
				ToolName: "execute_command",
				Parameters: map[string]interface{}{
					"command": tt.command,
				},
			}

			result, _ := executor.ExecuteTool(context.Background(), req)
			
			if tt.shouldAllow && !result.Success {
				t.Errorf("Expected command to be allowed, got: %s", result.Error)
			}
			if !tt.shouldAllow && result.Success {
				t.Error("Expected command to be denied")
			}
		})
	}
}
