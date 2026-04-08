// internal/mcp/transport.go
// Transport implementations for MCP communication
package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// JSONRPCMessage represents a JSON-RPC message
type JSONRPCMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// StdioTransportLegacy implements Transport using stdio (legacy implementation)
// Note: For new code, use the Process/Executor pattern from process.go and executor.go
type StdioTransportLegacy struct {
	command string
	args    []string
	env     map[string]string
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  io.ReadCloser
	scanner *bufio.Scanner
	mu      sync.Mutex
}

// NewStdioTransport creates a new stdio transport
func NewStdioTransport(command string, args []string, env map[string]string) *StdioTransportLegacy {
	return &StdioTransportLegacy{
		command: command,
		args:    args,
		env:     env,
	}
}

// Connect starts the command and sets up pipes
func (t *StdioTransportLegacy) Connect() error {
	t.cmd = exec.Command(t.command, t.args...)

	// Set environment variables
	for key, value := range t.env {
		t.cmd.Env = append(t.cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	var err error
	t.stdin, err = t.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	t.stdout, err = t.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	t.scanner = bufio.NewScanner(t.stdout)

	if err := t.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	return nil
}

// Disconnect stops the command
func (t *StdioTransportLegacy) Disconnect() error {
	if t.stdin != nil {
		t.stdin.Close()
	}

	if t.cmd != nil && t.cmd.Process != nil {
		// Try to terminate gracefully first
		t.cmd.Process.Signal(os.Interrupt)
		// Give it a moment to terminate gracefully
		done := make(chan error, 1)
		go func() {
			done <- t.cmd.Wait()
		}()
		select {
		case <-done:
			// Process terminated gracefully
		case <-time.After(2 * time.Second):
			// Force kill after timeout
			t.cmd.Process.Kill()
		}
	}

	return nil
}

// Send writes data to the command's stdin
func (t *StdioTransportLegacy) Send(data []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.stdin == nil {
		return fmt.Errorf("not connected")
	}

	// Write message with newline delimiter
	data = append(data, '\n')
	_, err := t.stdin.Write(data)
	return err
}

// Receive reads data from the command's stdout
func (t *StdioTransportLegacy) Receive() ([]byte, error) {
	if t.scanner == nil {
		return nil, fmt.Errorf("not connected")
	}

	if !t.scanner.Scan() {
		if err := t.scanner.Err(); err != nil {
			return nil, err
		}
		return nil, io.EOF
	}

	return t.scanner.Bytes(), nil
}

// SSETransport implements Transport using Server-Sent Events
type SSETransport struct {
	url       string
	client    *http.Client
	eventChan chan string
	mu        sync.Mutex
	connected bool
}

// NewSSETransport creates a new SSE transport
func NewSSETransport(url string) *SSETransport {
	return &SSETransport{
		url:       url,
		client:    &http.Client{Timeout: 30 * time.Second},
		eventChan: make(chan string, 100),
	}
}

// Connect establishes the SSE connection
func (t *SSETransport) Connect() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.connected {
		return nil
	}

	// Start SSE connection in background
	go t.sseLoop()

	t.connected = true
	return nil
}

// sseLoop maintains the SSE connection
func (t *SSETransport) sseLoop() {
	for {
		req, err := http.NewRequest("GET", t.url+"/sse", nil)
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}

		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("Cache-Control", "no-cache")

		resp, err := t.client.Do(req)
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}

		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				resp.Body.Close()
				break
			}

			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "data:") {
				data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				select {
				case t.eventChan <- data:
				default:
					// Channel full, drop message
				}
			}
		}
	}
}

// Disconnect closes the SSE connection
func (t *SSETransport) Disconnect() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.connected = false
	return nil
}

// Send posts data via HTTP POST
func (t *SSETransport) Send(data []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", t.url+"/messages", bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// Receive gets data from SSE channel
func (t *SSETransport) Receive() ([]byte, error) {
	select {
	case data := <-t.eventChan:
		return []byte(data), nil
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("timeout waiting for message")
	}
}
