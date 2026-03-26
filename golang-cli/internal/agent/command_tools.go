// Package agent provides the core agent functionality for the Cline CLI.
package agent

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cline/cline/golang-cli/internal/security"
)

// ExecuteCommandTool executes terminal commands
type ExecuteCommandTool struct {
	BaseTool
	// runningCommands tracks running commands by ID
	runningCommands map[string]*exec.Cmd
	mu              sync.RWMutex
}

// NewExecuteCommandTool creates a new execute command tool
func NewExecuteCommandTool() *ExecuteCommandTool {
	return &ExecuteCommandTool{
		BaseTool: BaseTool{
			Name:        "execute_command",
			Description: "Execute a CLI command on the system. Use this when you need to perform system operations or run specific commands to accomplish any step in the user's task. You must tailor your command to the user's system and provide a clear explanation of what the command does.",
			Usage: `<execute_command>
<command>Your command here</command>
<requires_approval>true or false</requires_approval>
</execute_command>`,
			Parameters: []ToolParameter{
				{
					Name:        "command",
					Type:        "string",
					Description: "The CLI command to execute",
					Required:    true,
				},
				{
					Name:        "requires_approval",
					Type:        "boolean",
					Description: "Whether this command requires explicit user approval",
					Required:    false,
					Default:     true,
				},
				{
					Name:        "timeout_seconds",
					Type:        "integer",
					Description: "Timeout in seconds (0 for no timeout)",
					Required:    false,
					Default:     300, // 5 minutes default
				},
				{
					Name:        "working_directory",
					Type:        "string",
					Description: "Working directory for command execution",
					Required:    false,
					Default:     "",
				},
			},
			Dangerous: true,
		},
		runningCommands: make(map[string]*exec.Cmd),
	}
}

// Execute runs a command
func (t *ExecuteCommandTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	if err := t.Validate(params); err != nil {
		return "", err
	}

	command := GetStringParam(params, "command", "")
	timeout := GetIntParam(params, "timeout_seconds", 300)
	workingDir := GetStringParam(params, "working_directory", "")

	// Validate command for security
	if err := t.validateCommand(command); err != nil {
		return "", fmt.Errorf("command validation failed: %w", err)
	}

	// Parse command
	cmdParts := t.parseCommand(command)
	if len(cmdParts) == 0 {
		return "", fmt.Errorf("empty command")
	}

	// Create command
	cmd := exec.CommandContext(ctx, cmdParts[0], cmdParts[1:]...)
	if workingDir != "" {
		cmd.Dir = workingDir
	}

	// Set up pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start command: %w", err)
	}

	// Generate command ID and track it
	cmdID := t.generateCommandID()
	t.mu.Lock()
	t.runningCommands[cmdID] = cmd
	t.mu.Unlock()

	// Clean up when done
	defer func() {
		t.mu.Lock()
		delete(t.runningCommands, cmdID)
		t.mu.Unlock()
	}()

	// Set up timeout if specified
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
		defer cancel()
	}

	// Read output
	var outputBuilder strings.Builder
	var wg sync.WaitGroup

	// Read stdout
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			outputBuilder.WriteString(line)
			outputBuilder.WriteString("\n")
		}
	}()

	// Read stderr
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			outputBuilder.WriteString("STDERR: ")
			outputBuilder.WriteString(line)
			outputBuilder.WriteString("\n")
		}
	}()

	// Wait for command to complete
	done := make(chan error, 1)
	go func() {
		wg.Wait()
		done <- cmd.Wait()
	}()

	// Wait for completion or timeout
	select {
	case err := <-done:
		output := outputBuilder.String()
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				return output, fmt.Errorf("command exited with code %d: %s", exitErr.ExitCode(), output)
			}
			return output, fmt.Errorf("command failed: %w", err)
		}
		return output, nil

	case <-ctx.Done():
		// Kill the process
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return outputBuilder.String(), fmt.Errorf("command timed out or was cancelled")
	}
}

// Validate validates the parameters
func (t *ExecuteCommandTool) Validate(params map[string]interface{}) error {
	return ValidateRequiredParams(params, []string{"command"})
}

// validateCommand checks if a command is safe to execute
func (t *ExecuteCommandTool) validateCommand(command string) error {
	// Check for dangerous patterns
	dangerousPatterns := []string{
		"rm -rf /",
		"rm -rf /*",
		"mkfs",
		"dd if=/dev/zero",
		"> /dev/sda",
		"curl.*|.*sh",
		"wget.*|.*sh",
	}

	cmdLower := strings.ToLower(command)
	for _, pattern := range dangerousPatterns {
		if matched, _ := filepath.Match(pattern, cmdLower); matched {
			return fmt.Errorf("potentially dangerous command detected")
		}
	}

	return nil
}

// parseCommand splits a command string into parts
func (t *ExecuteCommandTool) parseCommand(command string) []string {
	var parts []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for _, char := range command {
		switch {
		case char == '"' || char == '\'':
			if !inQuote {
				inQuote = true
				quoteChar = char
			} else if char == quoteChar {
				inQuote = false
				quoteChar = 0
			} else {
				current.WriteRune(char)
			}

		case char == ' ' && !inQuote:
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}

		default:
			current.WriteRune(char)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

// generateCommandID generates a unique command ID
func (t *ExecuteCommandTool) generateCommandID() string {
	return fmt.Sprintf("cmd_%d", time.Now().UnixNano())
}

// KillCommand kills a running command by ID
func (t *ExecuteCommandTool) KillCommand(cmdID string) error {
	t.mu.RLock()
	cmd, ok := t.runningCommands[cmdID]
	t.mu.RUnlock()

	if !ok {
		return fmt.Errorf("command not found: %s", cmdID)
	}

	if cmd.Process != nil {
		return cmd.Process.Kill()
	}

	return nil
}

// BrowserTool handles browser automation
type BrowserTool struct {
	BaseTool
	// browserSession tracks the browser state
	session *BrowserSession
	mu      sync.RWMutex
}

// BrowserSession represents a browser session
type BrowserSession struct {
	URL     string
	Open    bool
	LaunchedAt time.Time
}

// NewBrowserTool creates a new browser tool
func NewBrowserTool() *BrowserTool {
	return &BrowserTool{
		BaseTool: BaseTool{
			Name:        "browser_action",
			Description: "Launch a browser and perform actions like clicking, typing, or taking screenshots. Use this when you need to interact with web pages.",
			Usage: `<browser_action>
<action>launch</action>
<url>https://example.com</url>
</browser_action>`,
			Parameters: []ToolParameter{
				{
					Name:        "action",
					Type:        "string",
					Description: "Action to perform: launch, click, type, close",
					Required:    true,
				},
				{
					Name:        "url",
					Type:        "string",
					Description: "URL to navigate to (for launch action)",
					Required:    false,
				},
				{
					Name:        "coordinate",
					Type:        "string",
					Description: "Coordinate for click action (format: x,y)",
					Required:    false,
				},
				{
					Name:        "text",
					Type:        "string",
					Description: "Text to type (for type action)",
					Required:    false,
				},
			},
			Dangerous: false,
		},
	}
}

// Execute performs browser actions
func (t *BrowserTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	if err := t.Validate(params); err != nil {
		return "", err
	}

	action := GetStringParam(params, "action", "")

	switch action {
	case "launch":
		return t.launchBrowser(ctx, params)
	case "click":
		return t.click(ctx, params)
	case "type":
		return t.typeText(ctx, params)
	case "close":
		return t.closeBrowser(ctx)
	default:
		return "", fmt.Errorf("unknown browser action: %s", action)
	}
}

// Validate validates the parameters
func (t *BrowserTool) Validate(params map[string]interface{}) error {
	return ValidateRequiredParams(params, []string{"action"})
}

func (t *BrowserTool) launchBrowser(ctx context.Context, params map[string]interface{}) (string, error) {
	url := GetStringParam(params, "url", "")
	if url == "" {
		return "", fmt.Errorf("URL is required for launch action")
	}

	// Open browser using system command
	var cmd *exec.Cmd
	switch {
	case commandExists("open"):
		cmd = exec.Command("open", url)
	case commandExists("xdg-open"):
		cmd = exec.Command("xdg-open", url)
	case commandExists("start"):
		cmd = exec.Command("start", url)
	default:
		return "", fmt.Errorf("no browser launcher found")
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to launch browser: %w", err)
	}

	t.mu.Lock()
	t.session = &BrowserSession{
		URL:        url,
		Open:       true,
		LaunchedAt: time.Now(),
	}
	t.mu.Unlock()

	return fmt.Sprintf("Browser launched with URL: %s", url), nil
}

func (t *BrowserTool) click(ctx context.Context, params map[string]interface{}) (string, error) {
	// Browser automation would require a proper browser driver
	// For now, return a message indicating this would need additional setup
	return "Browser click action - requires browser automation driver (playwright/puppeteer)", nil
}

func (t *BrowserTool) typeText(ctx context.Context, params map[string]interface{}) (string, error) {
	// Browser automation would require a proper browser driver
	return "Browser type action - requires browser automation driver (playwright/puppeteer)", nil
}

func (t *BrowserTool) closeBrowser(ctx context.Context) (string, error) {
	t.mu.Lock()
	t.session = nil
	t.mu.Unlock()

	return "Browser session closed", nil
}

// UseMcpServerTool handles MCP server interactions
type UseMcpServerTool struct {
	BaseTool
}

// NewUseMcpServerTool creates a new MCP server tool
func NewUseMcpServerTool() *UseMcpServerTool {
	return &UseMcpServerTool{
		BaseTool: BaseTool{
			Name:        "use_mcp_server",
			Description: "Use an MCP (Model Context Protocol) server to access external tools and resources.",
			Usage: `<use_mcp_server>
<server_name>server-name</server_name>
<tool_name>tool name</tool_name>
<arguments>
{
  "param1": "value1",
  "param2": "value2"
}
</arguments>
</use_mcp_server>`,
			Parameters: []ToolParameter{
				{
					Name:        "server_name",
					Type:        "string",
					Description: "Name of the MCP server to use",
					Required:    true,
				},
				{
					Name:        "tool_name",
					Type:        "string",
					Description: "Name of the tool to call on the server",
					Required:    true,
				},
				{
					Name:        "arguments",
					Type:        "object",
					Description: "Arguments to pass to the tool",
					Required:    false,
				},
			},
			Dangerous: false,
		},
	}
}

// Execute calls an MCP server tool
func (t *UseMcpServerTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	if err := t.Validate(params); err != nil {
		return "", err
	}

	serverName := GetStringParam(params, "server_name", "")
	toolName := GetStringParam(params, "tool_name", "")

	// MCP server integration would connect to the McpHub
	// For now, return a placeholder
	return fmt.Sprintf("MCP server call: %s/%s - MCP integration requires McpHub connection", serverName, toolName), nil
}

// Validate validates the parameters
func (t *UseMcpServerTool) Validate(params map[string]interface{}) error {
	return ValidateRequiredParams(params, []string{"server_name", "tool_name"})
}

// commandExists checks if a command exists in PATH
func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// ReplaceInFileTool replaces content in a file using SEARCH/REPLACE blocks
type ReplaceInFileTool struct {
	BaseTool
	workingDir string
}

// NewReplaceInFileTool creates a new replace in file tool
func NewReplaceInFileTool(workingDir string) *ReplaceInFileTool {
	return &ReplaceInFileTool{
		BaseTool: BaseTool{
			Name:        "replace_in_file",
			Description: "Replace sections of content in an existing file using SEARCH/REPLACE blocks that define exact changes to specific parts of the file.",
			Usage: `<replace_in_file>
<path>path/to/file</path>
<diff>
------- SEARCH
[exact content to find]
=======
[new content to replace with]
+++++++ REPLACE
</diff>
</replace_in_file>`,
			Parameters: []ToolParameter{
				{
					Name:        "path",
					Type:        "string",
					Description: "The path of the file to modify",
					Required:    true,
				},
				{
					Name:        "diff",
					Type:        "string",
					Description: "One or more SEARCH/REPLACE blocks",
					Required:    true,
				},
			},
			Dangerous: true,
		},
		workingDir: workingDir,
	}
}

// Execute replaces content in a file
func (t *ReplaceInFileTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	if err := t.Validate(params); err != nil {
		return "", err
	}

	path := GetStringParam(params, "path", "")
	diff := GetStringParam(params, "diff", "")
	fullPath := t.resolvePath(path)

	// Validate the path
	if err := security.ValidateFilePath(fullPath, t.workingDir); err != nil {
		return "", fmt.Errorf("invalid file path: %w", err)
	}

	// Read the original file
	originalContent, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Apply the diff
	newContent, err := applyDiff(string(originalContent), diff)
	if err != nil {
		return "", fmt.Errorf("failed to apply diff: %w", err)
	}

	// Write the modified content
	if err := os.WriteFile(fullPath, []byte(newContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return fmt.Sprintf("File updated: %s", path), nil
}

// Validate validates the parameters
func (t *ReplaceInFileTool) Validate(params map[string]interface{}) error {
	return ValidateRequiredParams(params, []string{"path", "diff"})
}

func (t *ReplaceInFileTool) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(t.workingDir, path)
}