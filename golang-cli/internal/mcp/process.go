// Package mcp provides Model Context Protocol (MCP) server management.
// This file implements process lifecycle management for MCP servers.
package mcp

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

// ProcessState represents the current state of an MCP server process
type ProcessState string

const (
	ProcessStateStopped    ProcessState = "stopped"
	ProcessStateStarting   ProcessState = "starting"
	ProcessStateRunning    ProcessState = "running"
	ProcessStateError      ProcessState = "error"
	ProcessStateRestarting ProcessState = "restarting"
)

// ProcessConfig holds configuration for an MCP server process
type ProcessConfig struct {
	// Name is the unique identifier for the server
	Name string

	// Command is the executable to run
	Command string

	// Args are the arguments to pass to the command
	Args []string

	// Env is a map of environment variables
	Env map[string]string

	// WorkingDir is the working directory for the process
	WorkingDir string

	// Timeout is the startup timeout in seconds
	Timeout int

	// RestartPolicy controls automatic restart behavior
	RestartPolicy RestartPolicy

	// MaxRestarts is the maximum number of restart attempts
	MaxRestarts int
}

// RestartPolicy defines how the process should be restarted
type RestartPolicy string

const (
	RestartNever   RestartPolicy = "never"
	RestartOnError RestartPolicy = "on-error"
	RestartAlways  RestartPolicy = "always"
)

// Process represents a running MCP server process
type Process struct {
	config     *ProcessConfig
	cmd        *exec.Cmd
	state      ProcessState
	stateMu    sync.RWMutex
	startTime  time.Time
	stdin      io.WriteCloser
	stdout     io.ReadCloser
	stderr     io.ReadCloser
	outputMu   sync.RWMutex
	output     []OutputLine
	listeners  []chan OutputLine
	listenerMu sync.RWMutex
	restarts   int
	stopCh     chan struct{}
	wg         sync.WaitGroup
	lastError  error
}

// OutputLine represents a line of output from the process
type OutputLine struct {
	Timestamp time.Time `json:"timestamp"`
	Stream    string    `json:"stream"` // "stdout" or "stderr"
	Content   string    `json:"content"`
}

// ProcessManager manages multiple MCP server processes
type ProcessManager struct {
	processes map[string]*Process
	mu        sync.RWMutex
}

// NewProcessManager creates a new process manager
func NewProcessManager() *ProcessManager {
	return &ProcessManager{
		processes: make(map[string]*Process),
	}
}

// NewProcess creates a new process instance (not started)
func NewProcess(config *ProcessConfig) *Process {
	if config.Timeout == 0 {
		config.Timeout = 60
	}
	if config.MaxRestarts == 0 {
		config.MaxRestarts = 3
	}
	if config.RestartPolicy == "" {
		config.RestartPolicy = RestartOnError
	}
	if config.Env == nil {
		config.Env = make(map[string]string)
	}

	return &Process{
		config:    config,
		state:     ProcessStateStopped,
		output:    make([]OutputLine, 0),
		listeners: make([]chan OutputLine, 0),
		stopCh:    make(chan struct{}),
	}
}

// Start starts the MCP server process
func (p *Process) Start(ctx context.Context) error {
	p.stateMu.Lock()
	defer p.stateMu.Unlock()

	if p.state == ProcessStateRunning || p.state == ProcessStateStarting {
		return fmt.Errorf("process is already running")
	}

	p.state = ProcessStateStarting
	p.startTime = time.Now()
	p.lastError = nil

	// Create command
	p.cmd = exec.CommandContext(ctx, p.config.Command, p.config.Args...)

	// Set working directory
	if p.config.WorkingDir != "" {
		p.cmd.Dir = p.config.WorkingDir
	}

	// Set environment variables
	p.cmd.Env = os.Environ()
	for key, value := range p.config.Env {
		p.cmd.Env = append(p.cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Create pipes for stdin/stdout/stderr
	var err error
	p.stdin, err = p.cmd.StdinPipe()
	if err != nil {
		p.state = ProcessStateError
		p.lastError = fmt.Errorf("failed to create stdin pipe: %w", err)
		return p.lastError
	}

	p.stdout, err = p.cmd.StdoutPipe()
	if err != nil {
		p.state = ProcessStateError
		p.lastError = fmt.Errorf("failed to create stdout pipe: %w", err)
		return p.lastError
	}

	p.stderr, err = p.cmd.StderrPipe()
	if err != nil {
		p.state = ProcessStateError
		p.lastError = fmt.Errorf("failed to create stderr pipe: %w", err)
		return p.lastError
	}

	// Start the process
	if err := p.cmd.Start(); err != nil {
		p.state = ProcessStateError
		p.lastError = fmt.Errorf("failed to start process: %w", err)
		return p.lastError
	}

	// Start output readers
	p.wg.Add(2)
	go p.readOutput(p.stdout, "stdout")
	go p.readOutput(p.stderr, "stderr")

	// Monitor process in background
	go p.monitor()

	p.state = ProcessStateRunning
	return nil
}

// Stop stops the MCP server process
func (p *Process) Stop() error {
	p.stateMu.Lock()
	defer p.stateMu.Unlock()

	if p.state != ProcessStateRunning && p.state != ProcessStateStarting {
		return nil
	}

	close(p.stopCh)

	// Try graceful shutdown first
	if p.cmd != nil && p.cmd.Process != nil {
		// Send SIGTERM (or equivalent on Windows)
		p.cmd.Process.Signal(syscall.SIGTERM)

		// Wait for graceful shutdown with timeout
		done := make(chan struct{})
		go func() {
			p.wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			// Graceful shutdown successful
		case <-time.After(5 * time.Second):
			// Force kill after timeout
			p.cmd.Process.Kill()
		}
	}

	p.state = ProcessStateStopped
	return nil
}

// Restart restarts the MCP server process
func (p *Process) Restart(ctx context.Context) error {
	if err := p.Stop(); err != nil {
		return err
	}

	// Reset state
	p.stateMu.Lock()
	p.stopCh = make(chan struct{})
	p.output = make([]OutputLine, 0)
	p.stateMu.Unlock()

	// Wait a moment before restarting
	time.Sleep(100 * time.Millisecond)

	return p.Start(ctx)
}

// GetState returns the current process state
func (p *Process) GetState() ProcessState {
	p.stateMu.RLock()
	defer p.stateMu.RUnlock()
	return p.state
}

// IsRunning returns true if the process is running
func (p *Process) IsRunning() bool {
	state := p.GetState()
	return state == ProcessStateRunning || state == ProcessStateStarting
}

// GetPID returns the process ID (if running)
func (p *Process) GetPID() int {
	p.stateMu.RLock()
	defer p.stateMu.RUnlock()

	if p.cmd != nil && p.cmd.Process != nil {
		return p.cmd.Process.Pid
	}
	return -1
}

// GetStartTime returns when the process was started
func (p *Process) GetStartTime() time.Time {
	p.stateMu.RLock()
	defer p.stateMu.RUnlock()
	return p.startTime
}

// GetUptime returns how long the process has been running
func (p *Process) GetUptime() time.Duration {
	p.stateMu.RLock()
	defer p.stateMu.RUnlock()

	if p.startTime.IsZero() {
		return 0
	}
	return time.Since(p.startTime)
}

// GetLastError returns the last error encountered
func (p *Process) GetLastError() error {
	p.stateMu.RLock()
	defer p.stateMu.RUnlock()
	return p.lastError
}

// WriteStdin writes data to the process's stdin
func (p *Process) WriteStdin(data []byte) error {
	p.stateMu.RLock()
	defer p.stateMu.RUnlock()

	if p.state != ProcessStateRunning {
		return fmt.Errorf("process is not running")
	}

	if p.stdin == nil {
		return fmt.Errorf("stdin not available")
	}

	_, err := p.stdin.Write(data)
	return err
}

// GetOutput returns all captured output
func (p *Process) GetOutput() []OutputLine {
	p.outputMu.RLock()
	defer p.outputMu.RUnlock()

	// Return a copy
	result := make([]OutputLine, len(p.output))
	copy(result, p.output)
	return result
}

// GetRecentOutput returns the last n lines of output
func (p *Process) GetRecentOutput(n int) []OutputLine {
	p.outputMu.RLock()
	defer p.outputMu.RUnlock()

	if n >= len(p.output) {
		result := make([]OutputLine, len(p.output))
		copy(result, p.output)
		return result
	}

	result := make([]OutputLine, n)
	copy(result, p.output[len(p.output)-n:])
	return result
}

// SubscribeOutput returns a channel that receives new output lines
func (p *Process) SubscribeOutput() <-chan OutputLine {
	p.listenerMu.Lock()
	defer p.listenerMu.Unlock()

	ch := make(chan OutputLine, 100)
	p.listeners = append(p.listeners, ch)
	return ch
}

// UnsubscribeOutput removes a subscription
func (p *Process) UnsubscribeOutput(ch <-chan OutputLine) {
	p.listenerMu.Lock()
	defer p.listenerMu.Unlock()

	for i, listener := range p.listeners {
		if listener == ch {
			close(listener)
			p.listeners = append(p.listeners[:i], p.listeners[i+1:]...)
			return
		}
	}
}

// readOutput reads from a stream and captures output
func (p *Process) readOutput(reader io.ReadCloser, stream string) {
	defer p.wg.Done()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		select {
		case <-p.stopCh:
			return
		default:
		}

		line := OutputLine{
			Timestamp: time.Now(),
			Stream:    stream,
			Content:   scanner.Text(),
		}

		// Store output
		p.outputMu.Lock()
		p.output = append(p.output, line)
		// Limit stored output to prevent memory issues
		if len(p.output) > 10000 {
			p.output = p.output[len(p.output)-5000:]
		}
		p.outputMu.Unlock()

		// Notify listeners
		p.listenerMu.RLock()
		for _, ch := range p.listeners {
			select {
			case ch <- line:
			default:
				// Channel is full, skip
			}
		}
		p.listenerMu.RUnlock()
	}
}

// monitor monitors the process and handles restarts
func (p *Process) monitor() {
	if p.cmd == nil {
		return
	}

	err := p.cmd.Wait()

	p.stateMu.Lock()
	defer p.stateMu.Unlock()

	// Check if we were stopped intentionally
	select {
	case <-p.stopCh:
		p.state = ProcessStateStopped
		return
	default:
	}

	// Process exited unexpectedly
	if err != nil {
		p.lastError = err
	}

	// Handle restart policy
	if p.shouldRestart() {
		p.state = ProcessStateRestarting
		p.restarts++
		go p.attemptRestart()
	} else {
		p.state = ProcessStateError
	}
}

// shouldRestart determines if the process should be restarted
func (p *Process) shouldRestart() bool {
	if p.config.RestartPolicy == RestartNever {
		return false
	}

	if p.restarts >= p.config.MaxRestarts {
		return false
	}

	if p.config.RestartPolicy == RestartAlways {
		return true
	}

	// RestartOnError - check if there was an error
	return p.lastError != nil
}

// attemptRestart attempts to restart the process
func (p *Process) attemptRestart() {
	// Exponential backoff
	backoff := time.Duration(p.restarts) * time.Second
	if backoff > 30*time.Second {
		backoff = 30 * time.Second
	}

	time.Sleep(backoff)

	ctx := context.Background()
	if err := p.Restart(ctx); err != nil {
		p.stateMu.Lock()
		p.lastError = err
		p.stateMu.Unlock()
	}
}

// ProcessManager methods

// AddProcess adds a process to the manager
func (pm *ProcessManager) AddProcess(name string, process *Process) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.processes[name] = process
}

// RemoveProcess removes a process from the manager
func (pm *ProcessManager) RemoveProcess(name string) error {
	pm.mu.Lock()
	process, exists := pm.processes[name]
	pm.mu.Unlock()

	if !exists {
		return fmt.Errorf("process '%s' not found", name)
	}

	if err := process.Stop(); err != nil {
		return err
	}

	pm.mu.Lock()
	delete(pm.processes, name)
	pm.mu.Unlock()

	return nil
}

// GetProcess gets a process by name
func (pm *ProcessManager) GetProcess(name string) (*Process, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	process, exists := pm.processes[name]
	return process, exists
}

// ListProcesses lists all managed processes
func (pm *ProcessManager) ListProcesses() map[string]*Process {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	result := make(map[string]*Process)
	for name, process := range pm.processes {
		result[name] = process
	}
	return result
}

// StartAll starts all managed processes
func (pm *ProcessManager) StartAll(ctx context.Context) error {
	pm.mu.RLock()
	processes := make([]*Process, 0, len(pm.processes))
	for _, p := range pm.processes {
		processes = append(processes, p)
	}
	pm.mu.RUnlock()

	var errs []string
	for _, p := range processes {
		if err := p.Start(ctx); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", p.config.Name, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to start some processes: %s", strings.Join(errs, "; "))
	}

	return nil
}

// StopAll stops all managed processes
func (pm *ProcessManager) StopAll() error {
	pm.mu.RLock()
	processes := make([]*Process, 0, len(pm.processes))
	for _, p := range pm.processes {
		processes = append(processes, p)
	}
	pm.mu.RUnlock()

	var errs []string
	for _, p := range processes {
		if err := p.Stop(); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", p.config.Name, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to stop some processes: %s", strings.Join(errs, "; "))
	}

	return nil
}

// GetRunningProcesses returns all running processes
func (pm *ProcessManager) GetRunningProcesses() map[string]*Process {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	result := make(map[string]*Process)
	for name, process := range pm.processes {
		if process.IsRunning() {
			result[name] = process
		}
	}
	return result
}
