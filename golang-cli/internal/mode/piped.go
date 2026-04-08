// Package mode provides terminal mode detection capabilities for the Cline CLI.
package mode

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"time"
)

// ErrNotPiped is returned when stdin is not actually piped
var ErrNotPiped = fmt.Errorf("stdin is not piped")

// IsStdinPiped checks if stdin is actually piped (not just non-TTY).
// It verifies that stdin is a FIFO (pipe) or regular file by checking file stats.
func IsStdinPiped() bool {
	// Get file stats for stdin (fd 0)
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}

	// Check if it's a FIFO (named pipe) or regular file
	// A real pipe will have ModeNamedPipe, a file will have ModeRegular
	return (stat.Mode()&os.ModeNamedPipe != 0) || stat.Mode().IsRegular()
}

// ReadPipedStdin reads from stdin if it's piped.
//
// Returns:
//   - (content, nil) if piped with content
//   - ("", nil) if piped but empty
//   - ("", ErrNotPiped) if not piped
//   - ("", error) for other errors
//
// The timeout parameter specifies the maximum time to wait for input.
// A 5-minute timeout is recommended for most use cases.
func ReadPipedStdin(timeout time.Duration) (string, error) {
	// First check if stdin is actually piped
	if !IsStdinPiped() {
		return "", ErrNotPiped
	}

	// Use a channel to receive the result
	resultChan := make(chan pipedResult, 1)

	// Read stdin in a goroutine
	go func() {
		var buf bytes.Buffer
		reader := bufio.NewReader(os.Stdin)

		// Read all data from stdin
		for {
			line, err := reader.ReadString('\n')
			if len(line) > 0 {
				buf.WriteString(line)
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				resultChan <- pipedResult{"", fmt.Errorf("failed to read stdin: %w", err)}
				return
			}
		}

		resultChan <- pipedResult{buf.String(), nil}
	}()

	// Set up timeout
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	// Wait for either the read to complete or timeout
	select {
	case result := <-resultChan:
		if result.err != nil {
			return "", result.err
		}
		// Return content (may be empty string if stdin was piped but empty)
		return result.content, nil

	case <-timer.C:
		// Timeout - return what we have so far
		return "", fmt.Errorf("timeout reading piped input after %v", timeout)
	}
}

// ReadPipedStdinWithDefaultTimeout reads from stdin with a 5-minute default timeout.
// This is the recommended timeout for most piped input scenarios.
func ReadPipedStdinWithDefaultTimeout() (string, error) {
	return ReadPipedStdin(5 * time.Minute)
}

// pipedResult holds the result of reading from stdin
type pipedResult struct {
	content string
	err     error
}

// CombinePipedInputAndPrompt combines piped input with a prompt argument.
// If piped input is present, it's prepended to the prompt with a separator.
// Returns the combined prompt string.
func CombinePipedInputAndPrompt(pipedInput, prompt string) string {
	if pipedInput == "" {
		return prompt
	}
	if prompt == "" {
		return pipedInput
	}
	// Combine with double newline separator as specified in requirements
	return pipedInput + "\n\n" + prompt
}
