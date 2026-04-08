// Package piped provides utilities for handling piped/redirected input for the Cline CLI.
package piped

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// InputSource represents the source of input
type InputSource int

const (
	// SourceInteractive indicates interactive terminal input
	SourceInteractive InputSource = iota
	// SourcePipe indicates piped input from another command
	SourcePipe
	// SourceFile indicates input redirected from a file
	SourceFile
	// SourceEnvironment indicates input from environment variable
	SourceEnvironment
)

// InputReader handles reading input from various sources
type InputReader struct {
	source    InputSource
	reader    io.Reader
	content   string
	mu        sync.RWMutex
	lines     []string
	lineIndex int
}

// InputOption configures an InputReader
type InputOption func(*InputReader)

// WithReader sets a custom reader
func WithReader(r io.Reader) InputOption {
	return func(ir *InputReader) {
		ir.reader = r
	}
}

// WithSource sets the input source
func WithSource(source InputSource) InputOption {
	return func(ir *InputReader) {
		ir.source = source
	}
}

// NewInputReader creates a new input reader
func NewInputReader(opts ...InputOption) *InputReader {
	ir := &InputReader{
		source: SourceInteractive,
		reader: os.Stdin,
		lines:  make([]string, 0),
	}

	for _, opt := range opts {
		opt(ir)
	}

	return ir
}

// DetectSource detects the input source based on stdin
func DetectSource() InputSource {
	// Check if stdin is a terminal
	stat, err := os.Stdin.Stat()
	if err != nil {
		return SourceInteractive
	}

	// Check if stdin is a pipe or redirected file
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		// Check if it's a named pipe
		if stat.Mode()&os.ModeNamedPipe != 0 {
			return SourcePipe
		}
		// Otherwise it's likely a file redirect
		return SourceFile
	}

	return SourceInteractive
}

// ReadAll reads all input and stores it
func (ir *InputReader) ReadAll() error {
	ir.mu.Lock()
	defer ir.mu.Unlock()

	if ir.source == SourceInteractive {
		return nil
	}

	data, err := io.ReadAll(ir.reader)
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	ir.content = string(data)
	ir.lines = splitLines(ir.content)

	return nil
}

// ReadLines reads input line by line with a callback
func (ir *InputReader) ReadLines(ctx context.Context, callback func(string) bool) error {
	scanner := bufio.NewScanner(ir.reader)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Text()
		if !callback(line) {
			break
		}
	}

	return scanner.Err()
}

// ReadWithTimeout reads input with a timeout
func (ir *InputReader) ReadWithTimeout(timeout time.Duration) ([]byte, error) {
	if ir.source == SourceInteractive {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	dataChan := make(chan []byte, 1)
	errChan := make(chan error, 1)

	go func() {
		data, err := io.ReadAll(ir.reader)
		if err != nil {
			errChan <- err
			return
		}
		dataChan <- data
	}()

	select {
	case data := <-dataChan:
		return data, nil
	case err := <-errChan:
		return nil, err
	case <-ctx.Done():
		return nil, fmt.Errorf("timeout reading input after %v", timeout)
	}
}

// GetContent returns the full content
func (ir *InputReader) GetContent() string {
	ir.mu.RLock()
	defer ir.mu.RUnlock()
	return ir.content
}

// GetLines returns the input as lines
func (ir *InputReader) GetLines() []string {
	ir.mu.RLock()
	defer ir.mu.RUnlock()
	result := make([]string, len(ir.lines))
	copy(result, ir.lines)
	return result
}

// NextLine returns the next line
func (ir *InputReader) NextLine() (string, bool) {
	ir.mu.Lock()
	defer ir.mu.Unlock()

	if ir.lineIndex >= len(ir.lines) {
		return "", false
	}

	line := ir.lines[ir.lineIndex]
	ir.lineIndex++
	return line, true
}

// HasMore returns true if there are more lines
func (ir *InputReader) HasMore() bool {
	ir.mu.RLock()
	defer ir.mu.RUnlock()
	return ir.lineIndex < len(ir.lines)
}

// Reset resets the line pointer to the beginning
func (ir *InputReader) Reset() {
	ir.mu.Lock()
	defer ir.mu.Unlock()
	ir.lineIndex = 0
}

// IsPiped returns true if input is piped or redirected
func (ir *InputReader) IsPiped() bool {
	return ir.source != SourceInteractive
}

// GetSource returns the input source
func (ir *InputReader) GetSource() InputSource {
	return ir.source
}

// String returns a string representation of the source
func (s InputSource) String() string {
	switch s {
	case SourceInteractive:
		return "interactive"
	case SourcePipe:
		return "pipe"
	case SourceFile:
		return "file"
	case SourceEnvironment:
		return "environment"
	default:
		return "unknown"
	}
}

// splitLines splits content into lines
func splitLines(content string) []string {
	lines := strings.Split(content, "\n")
	// Remove empty trailing line if present
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// PromptResponse represents a response to a prompt
type PromptResponse struct {
	Response string
	Approved bool
	Always   bool
}

// PromptReader handles reading responses from prompts
type PromptReader struct {
	reader  io.Reader
	timeout time.Duration
}

// NewPromptReader creates a new prompt reader
func NewPromptReader(reader io.Reader) *PromptReader {
	return &PromptReader{
		reader:  reader,
		timeout: 30 * time.Second,
	}
}

// ReadResponse reads a response with a timeout
func (pr *PromptReader) ReadResponse() (*PromptResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), pr.timeout)
	defer cancel()

	responseChan := make(chan string, 1)
	errChan := make(chan error, 1)

	go func() {
		scanner := bufio.NewScanner(pr.reader)
		if scanner.Scan() {
			responseChan <- scanner.Text()
		} else if err := scanner.Err(); err != nil {
			errChan <- err
		} else {
			responseChan <- ""
		}
	}()

	select {
	case response := <-responseChan:
		return parseResponse(response), nil
	case err := <-errChan:
		return nil, err
	case <-ctx.Done():
		return nil, fmt.Errorf("timeout waiting for response")
	}
}

// parseResponse parses a response string
func parseResponse(response string) *PromptResponse {
	response = strings.ToLower(strings.TrimSpace(response))

	switch response {
	case "y", "yes":
		return &PromptResponse{Response: "yesButtonClicked", Approved: true, Always: false}
	case "n", "no":
		return &PromptResponse{Response: "noButtonClicked", Approved: false, Always: false}
	case "a", "always":
		return &PromptResponse{Response: "yesButtonClicked", Approved: true, Always: true}
	default:
		return &PromptResponse{Response: "messageResponse", Approved: false, Always: false}
	}
}

// SetTimeout sets the timeout for reading responses
func (pr *PromptReader) SetTimeout(timeout time.Duration) {
	pr.timeout = timeout
}
