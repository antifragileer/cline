// Package cli provides CLI initialization functionality for the Cline CLI.
// It handles setting up storage, state management, configuration, and gRPC connections.
package cli

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/cline/cline/golang-cli/internal/config"
	"github.com/cline/cline/golang-cli/internal/host"
	"github.com/cline/cline/golang-cli/internal/state"
	"github.com/cline/cline/golang-cli/internal/storage"
)

// InitOptions provides options for CLI initialization
type InitOptions struct {
	// Working directory for the CLI
	Cwd string

	// Config file path
	ConfigPath string

	// Verbose output
	Verbose bool

	// Workspace hash (empty for auto-detect)
	WorkspaceHash string

	// gRPC target address (empty for default)
	GRPCTarget string

	// Session overrides from CLI flags
	SessionOverrides map[string]interface{}

	// Output writer for logging
	Output io.Writer
}

// Context holds all initialized CLI components
type Context struct {
	// StateManager provides state management
	StateManager *state.StateManager

	// Storage provides file storage
	Storage *storage.StorageContext

	// Config provides layered configuration
	Config *config.LayeredConfig

	// gRPC client for communicating with core
	ProtoClient *host.ProtoClient

	// Client is the underlying host client
	Client *host.Client

	// Output writer
	Output io.Writer

	// Verbose mode
	Verbose bool

	// Cleanup functions
	cleanupFuncs []func() error
}

// Close cleans up all resources
func (c *Context) Close() error {
	var errs []error

	// Execute cleanup functions in reverse order
	for i := len(c.cleanupFuncs) - 1; i >= 0; i-- {
		if err := c.cleanupFuncs[i](); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("cleanup errors: %v", errs)
	}

	return nil
}

// Initialize initializes the CLI with all required components
func Initialize(opts InitOptions) (*Context, error) {
	ctx := &Context{
		Output:       opts.Output,
		Verbose:      opts.Verbose,
		cleanupFuncs: make([]func() error, 0),
	}

	if ctx.Output == nil {
		ctx.Output = os.Stdout
	}

	// Step 1: Set up working directory
	cwd, err := setupWorkingDirectory(opts.Cwd)
	if err != nil {
		return nil, fmt.Errorf("failed to set up working directory: %w", err)
	}

	if opts.Verbose {
		fmt.Fprintf(ctx.Output, "Working directory: %s\n", cwd)
	}

	// Step 2: Determine workspace hash
	workspaceHash := opts.WorkspaceHash
	if workspaceHash == "" {
		hash, err := storage.GetWorkspaceHash(cwd)
		if err != nil {
			return nil, fmt.Errorf("failed to get workspace hash: %w", err)
		}
		workspaceHash = hash
	}

	if opts.Verbose {
		fmt.Fprintf(ctx.Output, "Workspace hash: %s\n", workspaceHash)
	}

	// Step 3: Initialize storage
	storageCtx, err := initializeStorage(workspaceHash)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}
	ctx.Storage = storageCtx
	ctx.cleanupFuncs = append(ctx.cleanupFuncs, func() error {
		return storageCtx.Close()
	})

	if opts.Verbose {
		fmt.Fprintln(ctx.Output, "Storage initialized")
	}

	// Step 4: Initialize StateManager
	stateManager, err := initializeStateManager(storageCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize state manager: %w", err)
	}
	ctx.StateManager = stateManager
	ctx.cleanupFuncs = append(ctx.cleanupFuncs, func() error {
		return stateManager.Close()
	})

	if opts.Verbose {
		fmt.Fprintln(ctx.Output, "StateManager initialized")
	}

	// Step 5: Apply session overrides from CLI flags
	applySessionOverrides(stateManager, opts.SessionOverrides)

	// Step 6: Initialize configuration
	cfg, err := initializeConfig(workspaceHash, opts.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize configuration: %w", err)
	}
	ctx.Config = cfg

	if opts.Verbose {
		fmt.Fprintln(ctx.Output, "Configuration initialized")
	}

	// Step 7: Initialize gRPC client
	protoClient, client, err := initializeGRPCClient(opts.GRPCTarget, opts.Verbose, ctx.Output)
	if err != nil {
		if opts.Verbose {
			fmt.Fprintf(ctx.Output, "Warning: gRPC client initialization failed: %v\n", err)
		}
		// Don't fail if gRPC is not available - CLI can work in offline mode
	} else {
		ctx.ProtoClient = protoClient
		ctx.Client = client
		ctx.cleanupFuncs = append(ctx.cleanupFuncs, func() error {
			return protoClient.Stop()
		})

		if opts.Verbose {
			fmt.Fprintln(ctx.Output, "gRPC client initialized")
		}
	}

	// Step 8: Setup signal handling for graceful shutdown
	setupSignalHandling(ctx)

	return ctx, nil
}

// setupWorkingDirectory sets up and validates the working directory
func setupWorkingDirectory(cwd string) (string, error) {
	if cwd == "" {
		// Use current directory
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(cwd)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Validate directory exists and is accessible
	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("working directory not accessible: %w", err)
	}

	if !info.IsDir() {
		return "", fmt.Errorf("path is not a directory: %s", absPath)
	}

	return absPath, nil
}

// initializeStorage initializes the storage context
func initializeStorage(workspaceHash string) (*storage.StorageContext, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	baseDir := filepath.Join(homeDir, ".cline", "data")

	storageCtx, err := storage.NewStorageContext(baseDir, workspaceHash)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage context: %w", err)
	}

	return storageCtx, nil
}

// initializeStateManager initializes the StateManager
func initializeStateManager(storageCtx *storage.StorageContext) (*state.StateManager, error) {
	opts := state.ManagerOptions{
		Storage:       storageCtx,
		FlushInterval: 100 * time.Millisecond,
	}

	stateManager := state.NewStateManager(opts)

	if err := stateManager.Load(); err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	return stateManager, nil
}

// applySessionOverrides applies CLI flag overrides to the state manager
func applySessionOverrides(sm *state.StateManager, overrides map[string]interface{}) {
	for key, value := range overrides {
		sm.SetSessionOverride(key, value)
	}
}

// initializeConfig initializes the layered configuration
func initializeConfig(workspaceHash, configPath string) (*config.LayeredConfig, error) {
	opts := config.ConfigOptions{
		WorkspaceHash: workspaceHash,
	}

	cfg, err := config.NewLayeredConfig(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create layered config: %w", err)
	}

	// Load from custom config file if specified
	if configPath != "" {
		// This would load additional config from the specified file
		// and merge it into the layered config
		// For now, we'll just note it
		_ = configPath
	}

	return cfg, nil
}

// initializeGRPCClient initializes the gRPC client connection
func initializeGRPCClient(target string, verbose bool, output io.Writer) (*host.ProtoClient, *host.Client, error) {
	if target == "" {
		// Use default target - typically a local socket or port
		// This could be read from environment or config
		target = os.Getenv("CLINE_GRPC_TARGET")
		if target == "" {
			target = "localhost:50051" // Default
		}
	}

	clientConfig := host.ClientConfig{
		Target:              target,
		PoolSize:            3,
		ConnTimeout:         10 * time.Second,
		ReconnectDelay:      5 * time.Second,
		MaxRetries:          3,
		HealthCheckInterval: 30 * time.Second,
		OnStateChange: func(state host.ConnectionState) {
			if verbose {
				fmt.Fprintf(output, "gRPC connection state: %s\n", state)
			}
		},
		OnError: func(err error) {
			if verbose {
				fmt.Fprintf(output, "gRPC error: %v\n", err)
			}
		},
	}

	client, err := host.NewClient(clientConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create gRPC client: %w", err)
	}

	if err := client.Start(); err != nil {
		return nil, nil, fmt.Errorf("failed to start gRPC client: %w", err)
	}

	protoClient := host.NewProtoClient(client)
	if err := protoClient.Start(); err != nil {
		client.Stop()
		return nil, nil, fmt.Errorf("failed to start proto client: %w", err)
	}

	return protoClient, client, nil
}

// setupSignalHandling sets up graceful shutdown on SIGINT/SIGTERM
func setupSignalHandling(ctx *Context) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		if ctx.Verbose {
			fmt.Fprintln(ctx.Output, "\nReceived shutdown signal, cleaning up...")
		}
		ctx.Close()
		os.Exit(0)
	}()
}

// MustInitialize initializes the CLI or exits on error
func MustInitialize(opts InitOptions) *Context {
	ctx, err := Initialize(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize CLI: %v\n", err)
		os.Exit(1)
	}
	return ctx
}

// QuickInit provides a quick initialization with defaults
func QuickInit() (*Context, error) {
	return Initialize(InitOptions{})
}

// QuickInitVerbose provides a quick initialization with verbose output
func QuickInitVerbose() (*Context, error) {
	return Initialize(InitOptions{
		Verbose: true,
		Output:  os.Stdout,
	})
}