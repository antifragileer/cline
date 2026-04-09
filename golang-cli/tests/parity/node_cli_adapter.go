// Package parity provides side-by-side testing of Node.js and GoLang CLIs
package parity

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// NodeCLIAdapter adapts the Node.js CLI for testing
type NodeCLIAdapter struct {
	cliPath    string
	workingDir string
	env        map[string]string
}

// NewNodeCLIAdapter creates a new Node.js CLI adapter
func NewNodeCLIAdapter() (*NodeCLIAdapter, error) {
	path, err := discoverNodeCLI()
	if err != nil {
		return nil, fmt.Errorf("failed to discover Node.js CLI: %w", err)
	}

	return &NodeCLIAdapter{
		cliPath:    path,
		workingDir: "",
		env:        make(map[string]string),
	}, nil
}

// NewNodeCLIAdapterWithPath creates a new Node.js CLI adapter with a specific path
func NewNodeCLIAdapterWithPath(path string) (*NodeCLIAdapter, error) {
	if !fileExists(path) {
		// Try to find it in PATH
		found, err := exec.LookPath(path)
		if err != nil {
			return nil, fmt.Errorf("Node.js CLI not found at %s: %w", path, err)
		}
		path = found
	}

	return &NodeCLIAdapter{
		cliPath:    path,
		workingDir: "",
		env:        make(map[string]string),
	}, nil
}

// discoverNodeCLI attempts to find the Node.js CLI in various locations
func discoverNodeCLI() (string, error) {
	// 1. Check for npm-installed 'cline' command
	if path, err := exec.LookPath("cline"); err == nil {
		// Verify it's the Node.js version by checking if it's a script
		if isNodeCLI(path) {
			return path, nil
		}
	}

	// 2. Check for local CLI build in repository
	if path := findLocalNodeCLI(); path != "" {
		return path, nil
	}

	// 3. Check for npm exec cline
	if _, err := exec.LookPath("npm"); err == nil {
		// Test if we can run via npm exec (use --silent to avoid prompts)
		testCmd := exec.Command("npm", "exec", "--silent", "--", "cline", "--version")
		testCmd.Stdin = strings.NewReader("")
		if err := testCmd.Run(); err == nil {
			// Return npm exec command as a special marker
			return "npm:cline", nil
		}
	}

	// 4. Check for npx cline
	if _, err := exec.LookPath("npx"); err == nil {
		testCmd := exec.Command("npx", "--yes", "cline", "--version")
		if err := testCmd.Run(); err == nil {
			return "npx:cline", nil
		}
	}

	return "", fmt.Errorf("Node.js CLI not found. Install with: npm install -g cline")
}

// findLocalNodeCLI looks for the CLI in the local repository
func findLocalNodeCLI() string {
	// Try to find from current working directory up to root
	cwd, _ := os.Getwd()
	
	// Check common locations
	paths := []string{
		filepath.Join(cwd, "cli", "dist", "index.js"),
		filepath.Join(cwd, "cli", "dist", "cli.js"),
		filepath.Join(cwd, "..", "cli", "dist", "index.js"),
		filepath.Join(cwd, "..", "cli", "dist", "cli.js"),
		filepath.Join(cwd, "..", "..", "cli", "dist", "index.js"),
	}
	
	// Also check CLINE_CLI_PATH environment variable
	if envPath := os.Getenv("CLINE_CLI_PATH"); envPath != "" {
		paths = append([]string{envPath}, paths...)
	}
	
	for _, path := range paths {
		if fileExists(path) {
			return path
		}
	}
	
	return ""
}

// isNodeCLI checks if the given path is a Node.js CLI
func isNodeCLI(path string) bool {
	// Check if it's in a node_modules directory (npm global install)
	if strings.Contains(path, "node_modules") {
		return true
	}
	
	// Check if it's the 'cline' command (special case for npm-installed cline)
	if filepath.Base(path) == "cline" {
		// Additional check: see if it's linked to node_modules
		if realPath, err := filepath.EvalSymlinks(path); err == nil {
			if strings.Contains(realPath, "node_modules") {
				return true
			}
		}
		// If it's executable and named 'cline', assume it's valid
		// (we already know it passed --version test in discoverNodeCLI)
		return true
	}
	
	// Read first line to check if it's a node script
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil {
		return false
	}
	
	content := string(buf[:n])
	// Check for shebang with node
	if strings.Contains(content, "#!/usr/bin/env node") ||
		strings.Contains(content, "#!/usr/bin/node") ||
		strings.Contains(content, "node") {
		return true
	}
	
	// Check if it's a JavaScript file
	if strings.HasSuffix(path, ".js") {
		return true
	}
	
	return false
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// Execute runs the Node.js CLI with given arguments and input
func (a *NodeCLIAdapter) Execute(ctx context.Context, args []string, stdin string) (*Execution, error) {
	var cmd *exec.Cmd
	
	// Handle special npm/npx markers
	switch a.cliPath {
	case "npm:cline":
		// Use --silent to suppress npm output and avoid interactive prompts
		cmd = exec.CommandContext(ctx, "npm", append([]string{"exec", "--silent", "--", "cline"}, args...)...)
	case "npx:cline":
		cmd = exec.CommandContext(ctx, "npx", append([]string{"--yes", "cline"}, args...)...)
	default:
		// Direct execution
		if strings.HasSuffix(a.cliPath, ".js") {
			// It's a JS file, run with node
			cmd = exec.CommandContext(ctx, "node", append([]string{a.cliPath}, args...)...)
		} else {
			// Assume it's a binary or script
			cmd = exec.CommandContext(ctx, a.cliPath, args...)
		}
	}
	
	// Set working directory
	if a.workingDir != "" {
		cmd.Dir = a.workingDir
	}
	
	// Set environment
	cmd.Env = os.Environ()
	for k, v := range a.env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	
	// Set stdin - always provide a Reader to avoid hanging on stdin reads
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	} else {
		// Provide empty stdin to prevent commands from waiting for input
		cmd.Stdin = strings.NewReader("")
	}
	
	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	// Track execution time
	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)
	
	// Get exit code
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return nil, fmt.Errorf("failed to execute Node.js CLI: %w", err)
		}
	}
	
	return &Execution{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
		Duration: duration,
	}, nil
}

// GetVersion returns CLI version
func (a *NodeCLIAdapter) GetVersion() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	result, err := a.Execute(ctx, []string{"--version"}, "")
	if err != nil {
		return "", err
	}
	
	return strings.TrimSpace(result.Stdout), nil
}

// IsAvailable checks if CLI is installed/available
func (a *NodeCLIAdapter) IsAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	_, err := a.Execute(ctx, []string{"--version"}, "")
	return err == nil
}

// SetWorkingDir sets the working directory for executions
func (a *NodeCLIAdapter) SetWorkingDir(dir string) {
	a.workingDir = dir
}

// SetEnv sets an environment variable
func (a *NodeCLIAdapter) SetEnv(key, value string) {
	a.env[key] = value
}

// GetPath returns the detected CLI path
func (a *NodeCLIAdapter) GetPath() string {
	return a.cliPath
}

// EnsureNodeCLI ensures the Node.js CLI is available, returns error if not
func EnsureNodeCLI() error {
	adapter, err := NewNodeCLIAdapter()
	if err != nil {
		return err
	}
	
	if !adapter.IsAvailable() {
		return fmt.Errorf("Node.js CLI is not available at %s", adapter.GetPath())
	}
	
	return nil
}

// GetNodeCLIInfo returns information about the Node.js CLI installation
func GetNodeCLIInfo() (map[string]string, error) {
	info := make(map[string]string)
	
	adapter, err := NewNodeCLIAdapter()
	if err != nil {
		return info, err
	}
	
	info["path"] = adapter.GetPath()
	
	version, err := adapter.GetVersion()
	if err != nil {
		info["version"] = "unknown"
		info["error"] = err.Error()
	} else {
		info["version"] = version
	}
	
	// Get Node.js version
	nodeVersion, err := exec.Command("node", "--version").Output()
	if err == nil {
		info["node_version"] = strings.TrimSpace(string(nodeVersion))
	}
	
	// Get npm version
	npmVersion, err := exec.Command("npm", "--version").Output()
	if err == nil {
		info["npm_version"] = strings.TrimSpace(string(npmVersion))
	}
	
	info["platform"] = runtime.GOOS
	info["arch"] = runtime.GOARCH
	
	return info, nil
}