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

// GoLangCLIAdapter adapts the GoLang CLI for testing
type GoLangCLIAdapter struct {
	cliPath    string
	workingDir string
	env        map[string]string
	useGoRun   bool // Whether to use 'go run' or binary
	sourceDir  string // Directory containing main.go (for go run)
}

// NewGoLangCLIAdapter creates a new GoLang CLI adapter
func NewGoLangCLIAdapter() (*GoLangCLIAdapter, error) {
	path, sourceDir, useGoRun, err := discoverGoLangCLI()
	if err != nil {
		return nil, fmt.Errorf("failed to discover GoLang CLI: %w", err)
	}

	return &GoLangCLIAdapter{
		cliPath:    path,
		workingDir: "",
		env:        make(map[string]string),
		useGoRun:   useGoRun,
		sourceDir:  sourceDir,
	}, nil
}

// NewGoLangCLIAdapterWithPath creates a new GoLang CLI adapter with a specific binary path
func NewGoLangCLIAdapterWithPath(path string) (*GoLangCLIAdapter, error) {
	if !fileExists(path) {
		return nil, fmt.Errorf("GoLang CLI binary not found at %s", path)
	}

	return &GoLangCLIAdapter{
		cliPath:    path,
		workingDir: "",
		env:        make(map[string]string),
		useGoRun:   false,
	}, nil
}

// discoverGoLangCLI attempts to find the GoLang CLI in various locations
func discoverGoLangCLI() (string, string, bool, error) {
	// 1. Check for pre-built binary in common locations
	cwd, _ := os.Getwd()
	paths := []string{
		filepath.Join(cwd, "golang-cli", "cline"),
		filepath.Join(cwd, "cline"),
		filepath.Join(cwd, "bin", "cline"),
		filepath.Join(cwd, "dist", "cline"),
		filepath.Join(cwd, "..", "golang-cli", "cline"),
		filepath.Join(cwd, "..", "..", "golang-cli", "cline"),
	}
	
	// Add OS-specific binary names
	if runtime.GOOS == "windows" {
		for i, p := range paths {
			paths[i] = p + ".exe"
		}
	}
	
	// Check CLINE_GOLANG_CLI_PATH environment variable
	if envPath := os.Getenv("CLINE_GOLANG_CLI_PATH"); envPath != "" {
		paths = append([]string{envPath}, paths...)
	}
	
	for _, path := range paths {
		if fileExists(path) {
			return path, "", false, nil
		}
	}
	
	// 2. Check for source code to use with 'go run'
	sourcePaths := []string{
		filepath.Join(cwd, "cmd", "cline", "main.go"),
		filepath.Join(cwd, "golang-cli", "cmd", "cline", "main.go"),
		filepath.Join(cwd, "..", "golang-cli", "cmd", "cline", "main.go"),
		filepath.Join(cwd, "..", "..", "golang-cli", "cmd", "cline", "main.go"),
	}
	
	for _, path := range sourcePaths {
		if fileExists(path) {
			// Found source code, use 'go run'
			sourceDir := filepath.Dir(path)
			return path, sourceDir, true, nil
		}
	}
	
	// 3. Try to build it
	if sourceDir := findGoLangSource(); sourceDir != "" {
		binaryPath, err := buildGoLangCLI(sourceDir)
		if err == nil {
			return binaryPath, "", false, nil
		}
	}
	
	return "", "", false, fmt.Errorf("GoLang CLI not found. Build with: cd golang-cli && go build -o cline ./cmd/cline")
}

// findGoLangSource finds the GoLang CLI source directory
func findGoLangSource() string {
	cwd, _ := os.Getwd()
	
	checks := []string{
		filepath.Join(cwd, "cmd", "cline"),
		filepath.Join(cwd, "golang-cli", "cmd", "cline"),
		filepath.Join(cwd, "..", "golang-cli", "cmd", "cline"),
		filepath.Join(cwd, "..", "..", "golang-cli", "cmd", "cline"),
	}
	
	for _, dir := range checks {
		if fileExists(filepath.Join(dir, "main.go")) {
			return dir
		}
	}
	
	return ""
}

// buildGoLangCLI builds the GoLang CLI from source
func buildGoLangCLI(sourceDir string) (string, error) {
	// Create temp directory for binary
	tempDir, err := os.MkdirTemp("", "cline-parity-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	
	binaryName := "cline"
	if runtime.GOOS == "windows" {
		binaryName = "cline.exe"
	}
	
	outputPath := filepath.Join(tempDir, binaryName)
	
	// Run go build
	buildCmd := exec.Command("go", "build", "-o", outputPath, ".")
	buildCmd.Dir = sourceDir
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	
	if err := buildCmd.Run(); err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to build GoLang CLI: %w", err)
	}
	
	return outputPath, nil
}

// Execute runs the GoLang CLI with given arguments and input
func (a *GoLangCLIAdapter) Execute(ctx context.Context, args []string, stdin string) (*Execution, error) {
	var cmd *exec.Cmd
	
	if a.useGoRun {
		// Use 'go run' for development
		cmd = exec.CommandContext(ctx, "go", append([]string{"run", "."}, args...)...)
		cmd.Dir = a.sourceDir
	} else {
		// Use pre-built binary
		cmd = exec.CommandContext(ctx, a.cliPath, args...)
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
	
	// Set stdin
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
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
			return nil, fmt.Errorf("failed to execute GoLang CLI: %w", err)
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
func (a *GoLangCLIAdapter) GetVersion() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	result, err := a.Execute(ctx, []string{"--version"}, "")
	if err != nil {
		return "", err
	}
	
	return strings.TrimSpace(result.Stdout), nil
}

// IsAvailable checks if CLI is installed/available
func (a *GoLangCLIAdapter) IsAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// Try --version first
	_, err := a.Execute(ctx, []string{"--version"}, "")
	if err == nil {
		return true
	}
	
	// If that fails, try --help
	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()
	
	_, err = a.Execute(ctx2, []string{"--help"}, "")
	return err == nil
}

// SetWorkingDir sets the working directory for executions
func (a *GoLangCLIAdapter) SetWorkingDir(dir string) {
	a.workingDir = dir
}

// SetEnv sets an environment variable
func (a *GoLangCLIAdapter) SetEnv(key, value string) {
	a.env[key] = value
}

// GetPath returns the detected CLI path
func (a *GoLangCLIAdapter) GetPath() string {
	if a.useGoRun {
		return fmt.Sprintf("go run %s", a.sourceDir)
	}
	return a.cliPath
}

// IsUsingGoRun returns true if using 'go run' instead of binary
func (a *GoLangCLIAdapter) IsUsingGoRun() bool {
	return a.useGoRun
}

// Build builds the GoLang CLI from source if needed
func (a *GoLangCLIAdapter) Build() error {
	if a.useGoRun {
		// Already using source, no need to build
		return nil
	}
	
	if a.cliPath != "" && fileExists(a.cliPath) {
		// Binary already exists
		return nil
	}
	
	sourceDir := findGoLangSource()
	if sourceDir == "" {
		return fmt.Errorf("GoLang CLI source not found")
	}
	
	binaryPath, err := buildGoLangCLI(sourceDir)
	if err != nil {
		return err
	}
	
	a.cliPath = binaryPath
	a.useGoRun = false
	return nil
}

// EnsureGoLangCLI ensures the GoLang CLI is available, building if necessary
func EnsureGoLangCLI() (*GoLangCLIAdapter, error) {
	adapter, err := NewGoLangCLIAdapter()
	if err != nil {
		return nil, err
	}
	
	if !adapter.IsAvailable() {
		// Try to build it
		if err := adapter.Build(); err != nil {
			return nil, fmt.Errorf("GoLang CLI not available and could not be built: %w", err)
		}
	}
	
	if !adapter.IsAvailable() {
		return nil, fmt.Errorf("GoLang CLI is not available at %s", adapter.GetPath())
	}
	
	return adapter, nil
}

// GetGoLangCLIInfo returns information about the GoLang CLI installation
func GetGoLangCLIInfo() (map[string]string, error) {
	info := make(map[string]string)
	
	adapter, err := NewGoLangCLIAdapter()
	if err != nil {
		return info, err
	}
	
	info["path"] = adapter.GetPath()
	info["using_go_run"] = fmt.Sprintf("%v", adapter.IsUsingGoRun())
	
	version, err := adapter.GetVersion()
	if err != nil {
		info["version"] = "unknown"
		info["error"] = err.Error()
	} else {
		info["version"] = version
	}
	
	// Get Go version
	goVersion, err := exec.Command("go", "version").Output()
	if err == nil {
		info["go_version"] = strings.TrimSpace(string(goVersion))
	}
	
	info["platform"] = runtime.GOOS
	info["arch"] = runtime.GOARCH
	
	return info, nil
}