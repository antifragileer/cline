// Package agent provides the core agent functionality for the Cline CLI.
package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestToolRegistry(t *testing.T) {
	registry := NewToolRegistry()

	// Test that registry is empty initially
	if len(registry.GetAllTools()) != 0 {
		t.Error("New registry should be empty")
	}

	// Register a tool using the proper constructor
	tool := NewReadFileTool("/tmp")
	registry.Register(tool)

	// Test that tool is registered
	if !registry.IsToolAvailable("read_file") {
		t.Error("Tool should be available after registration")
	}

	// Test GetTool
	retrieved, err := registry.GetTool("read_file")
	if err != nil {
		t.Errorf("Failed to get tool: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Retrieved tool should not be nil")
	}
	if retrieved.GetName() != "read_file" {
		t.Errorf("Expected tool name 'read_file', got '%s'", retrieved.GetName())
	}

	// Test GetToolNames
	names := registry.GetToolNames()
	if len(names) != 1 || names[0] != "read_file" {
		t.Errorf("Expected ['read_file'], got %v", names)
	}

	// Test Unregister
	registry.Unregister("read_file")
	if registry.IsToolAvailable("read_file") {
		t.Error("Tool should not be available after unregister")
	}

	// Test GetTool for non-existent tool
	_, err = registry.GetTool("non_existent")
	if err == nil {
		t.Error("Expected error for non-existent tool")
	}
}

func TestValidateRequiredParams(t *testing.T) {
	tests := []struct {
		name     string
		params   map[string]interface{}
		required []string
		wantErr  bool
	}{
		{
			name:     "All required present",
			params:   map[string]interface{}{"path": "/test", "content": "data"},
			required: []string{"path", "content"},
			wantErr:  false,
		},
		{
			name:     "Missing required",
			params:   map[string]interface{}{"path": "/test"},
			required: []string{"path", "content"},
			wantErr:  true,
		},
		{
			name:     "Empty required list",
			params:   map[string]interface{}{"path": "/test"},
			required: []string{},
			wantErr:  false,
		},
		{
			name:     "Empty params",
			params:   map[string]interface{}{},
			required: []string{"path"},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRequiredParams(tt.params, tt.required)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRequiredParams() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetStringParam(t *testing.T) {
	params := map[string]interface{}{
		"existing": "value",
		"number":   123,
	}

	tests := []struct {
		name         string
		paramName    string
		defaultValue string
		expected     string
	}{
		{"Existing param", "existing", "default", "value"},
		{"Non-existing param", "missing", "default", "default"},
		{"Non-string param", "number", "default", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetStringParam(params, tt.paramName, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("GetStringParam() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestGetBoolParam(t *testing.T) {
	params := map[string]interface{}{
		"true_value":  true,
		"false_value": false,
		"string":      "not a bool",
	}

	tests := []struct {
		name         string
		paramName    string
		defaultValue bool
		expected     bool
	}{
		{"True value", "true_value", false, true},
		{"False value", "false_value", true, false},
		{"Non-existing", "missing", true, true},
		{"Non-bool", "string", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetBoolParam(params, tt.paramName, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("GetBoolParam() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestGetIntParam(t *testing.T) {
	params := map[string]interface{}{
		"int":     42,
		"int64":   int64(100),
		"float":   3.14,
		"string":  "not a number",
	}

	tests := []struct {
		name         string
		paramName    string
		defaultValue int
		expected     int
	}{
		{"Int value", "int", 0, 42},
		{"Int64 value", "int64", 0, 100},
		{"Float value", "float", 0, 3},
		{"Non-existing", "missing", 99, 99},
		{"Non-number", "string", 99, 99},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetIntParam(params, tt.paramName, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("GetIntParam() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestReadFileTool(t *testing.T) {
	// Create temp directory and file
	tempDir, err := os.MkdirTemp("", "readfile-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testContent := "Hello, World!"
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tool := NewReadFileTool(tempDir)

	t.Run("Read existing file", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"path": "test.txt",
		})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if result != testContent {
			t.Errorf("Expected '%s', got '%s'", testContent, result)
		}
	})

	t.Run("Read non-existent file", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), map[string]interface{}{
			"path": "nonexistent.txt",
		})
		if err == nil {
			t.Error("Expected error for non-existent file")
		}
	})

	t.Run("Read directory", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), map[string]interface{}{
			"path": ".",
		})
		if err == nil {
			t.Error("Expected error when reading directory")
		}
	})

	t.Run("Missing path parameter", func(t *testing.T) {
		err := tool.Validate(map[string]interface{}{})
		if err == nil {
			t.Error("Expected validation error for missing path")
		}
	})
}

func TestWriteFileTool(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "writefile-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tool := NewWriteFileTool(tempDir)

	t.Run("Create new file", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"path":    "newfile.txt",
			"content": "New content",
		})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		expected := "File created: newfile.txt"
		if result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}

		// Verify file was created
		content, err := os.ReadFile(filepath.Join(tempDir, "newfile.txt"))
		if err != nil {
			t.Errorf("Failed to read created file: %v", err)
		}
		if string(content) != "New content" {
			t.Errorf("Expected 'New content', got '%s'", string(content))
		}
	})

	t.Run("Update existing file", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "existing.txt")
		os.WriteFile(testFile, []byte("Old content"), 0644)

		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"path":    "existing.txt",
			"content": "Updated content",
		})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		expected := "File updated: existing.txt"
		if result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("Create nested file", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"path":    "nested/dir/file.txt",
			"content": "Nested content",
		})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if !strings.Contains(result, "nested/dir/file.txt") {
			t.Errorf("Expected path in result, got: %s", result)
		}

		// Verify directory was created
		_, err = os.Stat(filepath.Join(tempDir, "nested", "dir"))
		if err != nil {
			t.Errorf("Expected nested directory to be created: %v", err)
		}
	})
}

func TestApplyDiffTool(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "diff-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tool := NewApplyDiffTool(tempDir)

	t.Run("Apply simple diff", func(t *testing.T) {
		// Create original file
		original := "line1\nline2\nline3\n"
		testFile := filepath.Join(tempDir, "test.txt")
		os.WriteFile(testFile, []byte(original), 0644)

		diff := `------- SEARCH
line2
=======
replaced line
+++++++ REPLACE`

		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"path": "test.txt",
			"diff": diff,
		})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		expected := "Diff applied to: test.txt"
		if result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}

		// Verify content was updated
		content, _ := os.ReadFile(testFile)
		expectedContent := "line1\nreplaced line\nline3\n"
		if string(content) != expectedContent {
			t.Errorf("Expected '%s', got '%s'", expectedContent, string(content))
		}
	})

	t.Run("Diff with no matches", func(t *testing.T) {
		original := "line1\nline2\n"
		testFile := filepath.Join(tempDir, "nomatch.txt")
		os.WriteFile(testFile, []byte(original), 0644)

		diff := `------- SEARCH
nonexistent content
=======
replacement
+++++++ REPLACE`

		_, err := tool.Execute(context.Background(), map[string]interface{}{
			"path": "nomatch.txt",
			"diff": diff,
		})
		if err == nil {
			t.Error("Expected error when search content not found")
		}
	})
}

func TestListFilesTool(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "listfiles-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files and directories
	os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte{}, 0644)
	os.WriteFile(filepath.Join(tempDir, "file2.txt"), []byte{}, 0644)
	os.Mkdir(filepath.Join(tempDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(tempDir, "subdir", "file3.txt"), []byte{}, 0644)

	tool := NewListFilesTool(tempDir)

	t.Run("List files non-recursive", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"path":      ".",
			"recursive": false,
		})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if !strings.Contains(result, "file1.txt") {
			t.Error("Expected file1.txt in result")
		}
		if !strings.Contains(result, "file2.txt") {
			t.Error("Expected file2.txt in result")
		}
		if !strings.Contains(result, "subdir/") {
			t.Error("Expected subdir/ in result")
		}
	})

	t.Run("List files recursive", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"path":      ".",
			"recursive": true,
		})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if !strings.Contains(result, "file3.txt") {
			t.Error("Expected file3.txt in recursive result")
		}
	})

	t.Run("List non-existent directory", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), map[string]interface{}{
			"path": "nonexistent",
		})
		if err == nil {
			t.Error("Expected error for non-existent directory")
		}
	})
}

func TestSearchFilesTool(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "searchfiles-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files with content
	os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte("hello world"), 0644)
	os.WriteFile(filepath.Join(tempDir, "file2.txt"), []byte("goodbye world"), 0644)
	os.Mkdir(filepath.Join(tempDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(tempDir, "subdir", "file3.txt"), []byte("hello again"), 0644)

	tool := NewSearchFilesTool(tempDir)

	t.Run("Search for pattern", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"path":  ".",
			"regex": "hello",
		})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if !strings.Contains(result, "file1.txt") {
			t.Error("Expected file1.txt in results")
		}
		if strings.Contains(result, "file2.txt") {
			t.Error("Did not expect file2.txt in results")
		}
		if !strings.Contains(result, "file3.txt") {
			t.Error("Expected file3.txt in results")
		}
	})

	t.Run("Search with no matches", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"path":  ".",
			"regex": "xyz123",
		})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if !strings.Contains(result, "No matches found") {
			t.Errorf("Expected 'No matches found', got: %s", result)
		}
	})

	t.Run("Search with file pattern", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"path":         ".",
			"regex":        "hello",
			"file_pattern": "*.txt",
		})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Should find matches in .txt files
		if !strings.Contains(result, "file1.txt") && !strings.Contains(result, "file3.txt") {
			t.Error("Expected to find matches in .txt files")
		}
	})

	t.Run("Invalid regex", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), map[string]interface{}{
			"path":  ".",
			"regex": "[invalid",
		})
		if err == nil {
			t.Error("Expected error for invalid regex")
		}
	})
}

func TestExecuteCommandTool(t *testing.T) {
	tool := NewExecuteCommandTool()

	t.Run("Execute simple command", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"command": "echo hello",
		})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if !strings.Contains(result, "hello") {
			t.Errorf("Expected 'hello' in output, got: %s", result)
		}
	})

	t.Run("Execute command with timeout", func(t *testing.T) {
		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"command":         "sleep 5",
			"timeout_seconds": 1,
		})
		if err == nil {
			t.Error("Expected timeout error")
		}
		if !strings.Contains(result, "") && err == nil {
			t.Error("Expected timeout")
		}
	})

	t.Run("Missing command parameter", func(t *testing.T) {
		err := tool.Validate(map[string]interface{}{})
		if err == nil {
			t.Error("Expected validation error for missing command")
		}
	})

	t.Run("Empty command", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), map[string]interface{}{
			"command": "",
		})
		if err == nil {
			t.Error("Expected error for empty command")
		}
	})
}

func TestBrowserTool(t *testing.T) {
	tool := NewBrowserTool()

	t.Run("Missing action parameter", func(t *testing.T) {
		err := tool.Validate(map[string]interface{}{})
		if err == nil {
			t.Error("Expected validation error for missing action")
		}
	})

	t.Run("Unknown action", func(t *testing.T) {
		_, err := tool.Execute(context.Background(), map[string]interface{}{
			"action": "unknown",
		})
		if err == nil {
			t.Error("Expected error for unknown action")
		}
	})
}

func TestBaseTool(t *testing.T) {
	tool := &BaseTool{
		Name:        "test_tool",
		Description: "A test tool",
		Usage:       "<test></test>",
		Parameters: []ToolParameter{
			{Name: "param1", Type: "string", Required: true},
		},
		Dangerous: false,
	}

	if tool.GetName() != "test_tool" {
		t.Errorf("Expected name 'test_tool', got '%s'", tool.GetName())
	}

	if tool.GetDescription() != "A test tool" {
		t.Errorf("Expected description 'A test tool', got '%s'", tool.GetDescription())
	}

	if tool.IsDangerous() != false {
		t.Error("Expected tool to not be dangerous")
	}

	params := tool.GetParameters()
	if len(params) != 1 || params[0].Name != "param1" {
		t.Errorf("Expected 1 parameter 'param1', got %v", params)
	}
}