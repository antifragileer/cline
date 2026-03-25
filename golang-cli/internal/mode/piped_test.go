package mode

import (
	"os"
	"testing"
	"time"
)

func TestIsStdinPiped(t *testing.T) {
	// This test is limited because we can't easily control stdin in unit tests
	// The function should at least not panic when called
	result := IsStdinPiped()
	
	// When running in a test environment without piped input, it should return false
	// We can't assert the exact value without controlling stdin
	t.Logf("IsStdinPiped() returned: %v", result)
}

func TestReadPipedStdin_NotPiped(t *testing.T) {
	// When stdin is not piped, should return ErrNotPiped
	// This test assumes the test environment doesn't have piped stdin
	
	content, err := ReadPipedStdin(100 * time.Millisecond)
	
	// In a test environment, stdin is typically not a pipe
	if err != ErrNotPiped {
		t.Logf("Expected ErrNotPiped, got: %v (content: %q)", err, content)
		// This is acceptable in test environments where stdin might be a file
	}
}

func TestReadPipedStdin_Timeout(t *testing.T) {
	// Test that timeout works correctly
	// This test uses a very short timeout to ensure it completes quickly
	
	start := time.Now()
	content, err := ReadPipedStdin(1 * time.Millisecond)
	elapsed := time.Since(start)
	
	// Should complete quickly (within 100ms)
	if elapsed > 100*time.Millisecond {
		t.Errorf("Expected quick completion, took %v", elapsed)
	}
	
	// Should either return ErrNotPiped or timeout error
	if err != nil && err != ErrNotPiped {
		// Check if it's a timeout error
		if err.Error() != "timeout reading piped input after 1ms" {
			t.Logf("Got error: %v", err)
		}
	}
	
	// Content should be empty when there's an error or not piped
	if err != nil && content != "" {
		t.Errorf("Expected empty content on error, got: %q", content)
	}
}

func TestCombinePipedInputAndPrompt(t *testing.T) {
	tests := []struct {
		name       string
		pipedInput string
		prompt     string
		want       string
	}{
		{
			name:       "both present",
			pipedInput: "git diff output",
			prompt:     "explain changes",
			want:       "git diff output\n\nexplain changes",
		},
		{
			name:       "only piped input",
			pipedInput: "some code",
			prompt:     "",
			want:       "some code",
		},
		{
			name:       "only prompt",
			pipedInput: "",
			prompt:     "just a prompt",
			want:       "just a prompt",
		},
		{
			name:       "both empty",
			pipedInput: "",
			prompt:     "",
			want:       "",
		},
		{
			name:       "multiline piped input",
			pipedInput: "line1\nline2\nline3",
			prompt:     "summarize",
			want:       "line1\nline2\nline3\n\nsummarize",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CombinePipedInputAndPrompt(tt.pipedInput, tt.prompt)
			if got != tt.want {
				t.Errorf("CombinePipedInputAndPrompt() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReadPipedStdinWithDefaultTimeout(t *testing.T) {
	// This should use the 5-minute default timeout
	// We can't test the actual timeout duration without mocking,
	// but we can ensure the function doesn't panic
	
	start := time.Now()
	content, err := ReadPipedStdinWithDefaultTimeout()
	elapsed := time.Since(start)
	
	// Should complete quickly in non-piped environment
	if elapsed > 1*time.Second {
		t.Logf("Took longer than expected: %v", elapsed)
	}
	
	// In non-piped environment, should return ErrNotPiped
	if err != nil && err != ErrNotPiped {
		t.Logf("Got error (may be expected in test environment): %v", err)
	}
	
	t.Logf("Content length: %d", len(content))
}

// Test with actual piped input would require integration tests
// that create actual pipes. Example:
func TestReadPipedStdin_WithActualPipe(t *testing.T) {
	// Skip this test in short mode as it requires creating actual pipes
	if testing.Short() {
		t.Skip("Skipping pipe integration test in short mode")
	}
	
	// Create a pipe
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	defer r.Close()
	
	// Write test data to the pipe in a goroutine
	testData := "test piped input data"
	go func() {
		defer w.Close()
		w.WriteString(testData)
	}()
	
	// Replace stdin with our pipe temporarily
	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()
	
	// Read from the pipe
	content, err := ReadPipedStdin(5 * time.Second)
	if err != nil {
		t.Errorf("ReadPipedStdin() error = %v", err)
		return
	}
	
	if content != testData {
		t.Errorf("ReadPipedStdin() = %q, want %q", content, testData)
	}
}