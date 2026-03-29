package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/cline/cline/golang-cli/internal/storage"
	internalMCP "github.com/cline/cline/golang-cli/internal/mcp"
)

// mcpFlags holds the parsed flag values for mcp command
var mcpFlags struct {
	name        string
	description string
	command     string
	args        []string
	env         []string
	timeout     int
	autoApprove bool
	json        bool
}

// MCPServer represents an MCP server configuration
type MCPServer struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Command     string            `json:"command"`
	Args        []string          `json:"args,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	Timeout     int               `json:"timeout,omitempty"`
	AutoApprove bool              `json:"autoApprove,omitempty"`
	Disabled    bool              `json:"disabled,omitempty"`
}

// MCPRegistryEntry represents an entry in the MCP marketplace
type MCPRegistryEntry struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Author      string `json:"author"`
	Repository  string `json:"repository"`
	License     string `json:"license"`
	Stars       int    `json:"stars"`
	Downloads   int    `json:"downloads"`
}

// mcpCmd represents the mcp command
var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Manage MCP (Model Context Protocol) servers",
	Long: `Manage MCP servers for extending Cline's capabilities.

MCP servers provide additional tools and resources that Cline can use.
This command allows you to add, remove, list, and configure MCP servers.`,
}

// mcpAddCmd represents the mcp add subcommand
var mcpAddCmd = &cobra.Command{
	Use:   "add [name]",
	Short: "Add an MCP server",
	Long: `Add an MCP server to Cline configuration.

You can add a server by specifying its name from the marketplace,
or provide a custom command to run the server.`,
	Example: `  # Add from marketplace
  cline mcp add filesystem

  # Add with custom command
  cline mcp add my-server --command npx --args "@modelcontextprotocol/server-filesystem /path"

  # Add with environment variables
  cline mcp add github --command npx --args "@modelcontextprotocol/server-github" --env "GITHUB_TOKEN=xxx"`,
	Args: cobra.ExactArgs(1),
	RunE: runMCPAdd,
}

// mcpRemoveCmd represents the mcp remove subcommand
var mcpRemoveCmd = &cobra.Command{
	Use:   "remove [name]",
	Short: "Remove an MCP server",
	Long:  `Remove an MCP server from Cline configuration.`,
	Example: `  # Remove a server
  cline mcp remove filesystem`,
	Aliases: []string{"rm"},
	Args:    cobra.ExactArgs(1),
	RunE:    runMCPRemove,
}

// mcpListCmd represents the mcp list subcommand
var mcpListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured MCP servers",
	Long:  `List all configured MCP servers with their status.`,
	Example: `  # List all servers
  cline mcp list

  # List with details
  cline mcp list --verbose`,
	Aliases: []string{"ls"},
	RunE:    runMCPList,
}

// mcpEnableCmd represents the mcp enable subcommand
var mcpEnableCmd = &cobra.Command{
	Use:   "enable [name]",
	Short: "Enable an MCP server",
	Long:  `Enable a previously disabled MCP server.`,
	Example: `  cline mcp enable filesystem`,
	Args: cobra.ExactArgs(1),
	RunE: runMCPEnable,
}

// mcpDisableCmd represents the mcp disable subcommand
var mcpDisableCmd = &cobra.Command{
	Use:   "disable [name]",
	Short: "Disable an MCP server",
	Long:  `Disable an MCP server without removing it from configuration.`,
	Example: `  cline mcp disable filesystem`,
	Args: cobra.ExactArgs(1),
	RunE: runMCPDisable,
}

// mcpMarketplaceCmd represents the mcp marketplace subcommand
var mcpMarketplaceCmd = &cobra.Command{
	Use:   "marketplace",
	Short: "Browse available MCP servers",
	Long:  `Browse and search the MCP marketplace for available servers.`,
	Example: `  # List available servers
  cline mcp marketplace

  # Search for servers
  cline mcp marketplace --search filesystem`,
	Aliases: []string{"mp", "market"},
	RunE:    runCPMarketplace,
}

// mcpRunCmd represents the mcp run subcommand
var mcpRunCmd = &cobra.Command{
	Use:   "run [name]",
	Short: "Start and run an MCP server",
	Long: `Start and run an MCP server in the foreground.

This command starts an MCP server and keeps it running until interrupted.
Useful for testing and debugging MCP servers.`,
	Example: `  # Run a configured MCP server
  cline mcp run filesystem

  # Run with verbose output
  cline mcp run filesystem --verbose`,
	Args: cobra.ExactArgs(1),
	RunE: runMCPRun,
}

// mcpToolsCmd represents the mcp tools subcommand
var mcpToolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "List tools from an MCP server",
	Long:  `List all available tools from a running MCP server.`,
	Example: `  # List tools from a server
  cline mcp tools filesystem`,
	Args: cobra.ExactArgs(1),
	RunE: runMCPTools,
}

func init() {
	rootCmd.AddCommand(mcpCmd)
	
	// Add subcommands
	mcpCmd.AddCommand(mcpAddCmd)
	mcpCmd.AddCommand(mcpRemoveCmd)
	mcpCmd.AddCommand(mcpListCmd)
	mcpCmd.AddCommand(mcpEnableCmd)
	mcpCmd.AddCommand(mcpDisableCmd)
	mcpCmd.AddCommand(mcpMarketplaceCmd)
	mcpCmd.AddCommand(mcpRunCmd)
	mcpCmd.AddCommand(mcpToolsCmd)

	// Add flags for mcp add
	mcpAddCmd.Flags().StringVar(&mcpFlags.command, "command", "", "Command to run the MCP server (required for custom servers)")
	mcpAddCmd.Flags().StringArrayVar(&mcpFlags.args, "args", nil, "Arguments for the command")
	mcpAddCmd.Flags().StringArrayVar(&mcpFlags.env, "env", nil, "Environment variables (KEY=value format)")
	mcpAddCmd.Flags().IntVar(&mcpFlags.timeout, "timeout", 60, "Timeout in seconds")
	mcpAddCmd.Flags().BoolVar(&mcpFlags.autoApprove, "auto-approve", false, "Auto-approve all tool requests from this server")
	
	// Add flags for mcp list
	mcpListCmd.Flags().BoolVarP(&mcpFlags.json, "json", "j", false, "Output in JSON format")
	
	// Add flags for marketplace
	mcpMarketplaceCmd.Flags().StringVar(&mcpFlags.name, "search", "", "Search term")

	// Add flags for mcp run
	mcpRunCmd.Flags().BoolP("verbose", "v", false, "Show verbose output")
}

// runMCPRun executes the mcp run command
func runMCPRun(cmd *cobra.Command, args []string) error {
	serverName := args[0]
	
	// Initialize storage
	ctx, err := storage.NewStorageContext("", "")
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer ctx.Close()

	// Load servers
	servers, err := loadMCPServers(ctx)
	if err != nil {
		return fmt.Errorf("failed to load MCP servers: %w", err)
	}

	// Check if server exists
	server, exists := servers[serverName]
	if !exists {
		return fmt.Errorf("MCP server '%s' not found", serverName)
	}

	if server.Disabled {
		return fmt.Errorf("MCP server '%s' is disabled. Enable it with: cline mcp enable %s", serverName, serverName)
	}

	verbose, _ := cmd.Flags().GetBool("verbose")
	
	// Create process config
	config := &internalMCP.ProcessConfig{
		Name:        serverName,
		Command:     server.Command,
		Args:        server.Args,
		Env:         server.Env,
		Timeout:     server.Timeout,
		RestartPolicy: internalMCP.RestartNever,
		MaxRestarts: 0,
	}

	// Create and start process
	process := internalMCP.NewProcess(config)
	
	fmt.Fprintf(cmd.OutOrStdout(), "Starting MCP server '%s'...\n", serverName)
	fmt.Fprintf(cmd.OutOrStdout(), "Command: %s %s\n\n", server.Command, strings.Join(server.Args, " "))

	if err := process.Start(cmd.Context()); err != nil {
		return fmt.Errorf("failed to start MCP server: %w", err)
	}

	// Create executor
	executor := internalMCP.NewExecutor(process)
	if err := executor.Connect(cmd.Context()); err != nil {
		process.Stop()
		return fmt.Errorf("failed to connect to MCP server: %w", err)
	}
	defer executor.Close()

	fmt.Fprintf(cmd.OutOrStdout(), "✓ MCP server '%s' is running (PID: %d)\n", serverName, process.GetPID())
	fmt.Fprintln(cmd.OutOrStdout(), "Press Ctrl+C to stop")
	fmt.Fprintln(cmd.OutOrStdout())

	// Subscribe to output if verbose
	if verbose {
		outputCh := process.SubscribeOutput()
		defer process.UnsubscribeOutput(outputCh)
		
		go func() {
			for line := range outputCh {
				prefix := "[stdout]"
				if line.Stream == "stderr" {
					prefix = "[stderr]"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", prefix, line.Content)
			}
		}()
	}

	// Wait for process to exit
	for process.IsRunning() {
		time.Sleep(100 * time.Millisecond)
	}

	// Check if there was an error
	if lastErr := process.GetLastError(); lastErr != nil {
		return fmt.Errorf("MCP server exited with error: %w", lastErr)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\nMCP server '%s' stopped\n", serverName)
	return nil
}

// runMCPTools executes the mcp tools command
func runMCPTools(cmd *cobra.Command, args []string) error {
	serverName := args[0]
	
	// Initialize storage
	ctx, err := storage.NewStorageContext("", "")
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer ctx.Close()

	// Load servers
	servers, err := loadMCPServers(ctx)
	if err != nil {
		return fmt.Errorf("failed to load MCP servers: %w", err)
	}

	// Check if server exists
	server, exists := servers[serverName]
	if !exists {
		return fmt.Errorf("MCP server '%s' not found", serverName)
	}

	if server.Disabled {
		return fmt.Errorf("MCP server '%s' is disabled", serverName)
	}

	// Create process config
	config := &internalMCP.ProcessConfig{
		Name:        serverName,
		Command:     server.Command,
		Args:        server.Args,
		Env:         server.Env,
		Timeout:     server.Timeout,
		RestartPolicy: internalMCP.RestartNever,
		MaxRestarts: 0,
	}

	// Create and start process
	process := internalMCP.NewProcess(config)
	
	fmt.Fprintf(cmd.OutOrStdout(), "Starting MCP server '%s' to list tools...\n", serverName)

	if err := process.Start(cmd.Context()); err != nil {
		return fmt.Errorf("failed to start MCP server: %w", err)
	}
	defer process.Stop()

	// Create executor
	executor := internalMCP.NewExecutor(process)
	if err := executor.Connect(cmd.Context()); err != nil {
		return fmt.Errorf("failed to connect to MCP server: %w", err)
	}
	defer executor.Close()

	// List tools
	tools, err := executor.ListTools(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}

	if len(tools) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No tools available from this server.")
		return nil
	}

	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "Available Tools:")
	fmt.Fprintln(cmd.OutOrStdout())

	for _, tool := range tools {
		fmt.Fprintf(cmd.OutOrStdout(), "  🔧 %s\n", tool.Name)
		if tool.Description != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "     %s\n", tool.Description)
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}

	return nil
}

// runMCPAdd executes the mcp add command
func runMCPAdd(cmd *cobra.Command, args []string) error {
	serverName := args[0]
	
	// Initialize storage
	ctx, err := storage.NewStorageContext("", "")
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer ctx.Close()

	// Check if server already exists
	servers, err := loadMCPServers(ctx)
	if err != nil {
		return fmt.Errorf("failed to load existing servers: %w", err)
	}

	if _, exists := servers[serverName]; exists {
		return fmt.Errorf("MCP server '%s' already exists. Use 'cline mcp remove %s' first to replace it", serverName, serverName)
	}

	// Create server configuration
	server := &MCPServer{
		Name:        serverName,
		Description: mcpFlags.description,
		Timeout:     mcpFlags.timeout,
		AutoApprove: mcpFlags.autoApprove,
		Env:         make(map[string]string),
	}

	// If command is specified, use it directly
	if mcpFlags.command != "" {
		server.Command = mcpFlags.command
		server.Args = mcpFlags.args
		
		// Parse environment variables
		for _, env := range mcpFlags.env {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				server.Env[parts[0]] = parts[1]
			}
		}
	} else {
		// Try to get from marketplace
		marketServer, err := fetchMarketplaceServer(serverName)
		if err != nil {
			return fmt.Errorf("failed to fetch server from marketplace: %w\n\nYou can specify a custom server with --command", err)
		}
		
		server.Description = marketServer.Description
		// For marketplace servers, we'll use npx as the default runner
		server.Command = "npx"
		server.Args = []string{fmt.Sprintf("@modelcontextprotocol/server-%s", serverName)}
	}

	// Save server
	servers[serverName] = server
	if err := saveMCPServers(ctx, servers); err != nil {
		return fmt.Errorf("failed to save MCP server: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ MCP server '%s' added successfully\n", serverName)
	if server.Description != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "  Description: %s\n", server.Description)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "  Command: %s %s\n", server.Command, strings.Join(server.Args, " "))
	
	return nil
}

// runMCPRemove executes the mcp remove command
func runMCPRemove(cmd *cobra.Command, args []string) error {
	serverName := args[0]
	
	// Initialize storage
	ctx, err := storage.NewStorageContext("", "")
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer ctx.Close()

	// Load existing servers
	servers, err := loadMCPServers(ctx)
	if err != nil {
		return fmt.Errorf("failed to load MCP servers: %w", err)
	}

	// Check if server exists
	if _, exists := servers[serverName]; !exists {
		return fmt.Errorf("MCP server '%s' not found", serverName)
	}

	// Remove server
	delete(servers, serverName)
	if err := saveMCPServers(ctx, servers); err != nil {
		return fmt.Errorf("failed to save MCP servers: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ MCP server '%s' removed successfully\n", serverName)
	return nil
}

// runMCPList executes the mcp list command
func runMCPList(cmd *cobra.Command, args []string) error {
	// Initialize storage
	ctx, err := storage.NewStorageContext("", "")
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer ctx.Close()

	// Load servers
	servers, err := loadMCPServers(ctx)
	if err != nil {
		return fmt.Errorf("failed to load MCP servers: %w", err)
	}

	if len(servers) == 0 {
		if mcpFlags.json {
			emptyResult := map[string]interface{}{
				"servers": []interface{}{},
			}
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(emptyResult)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "No MCP servers configured.")
		fmt.Fprintln(cmd.OutOrStdout(), "\nRun 'cline mcp marketplace' to browse available servers.")
		return nil
	}

	// Output as JSON if requested
	if mcpFlags.json {
		serverList := make([]map[string]interface{}, 0, len(servers))
		for name, server := range servers {
			serverInfo := map[string]interface{}{
				"name":     name,
				"command":  server.Command,
				"args":     server.Args,
				"disabled": server.Disabled,
			}
			if server.Description != "" {
				serverInfo["description"] = server.Description
			}
			if server.AutoApprove {
				serverInfo["autoApprove"] = true
			}
			if len(server.Env) > 0 {
				serverInfo["env"] = server.Env
			}
			serverList = append(serverList, serverInfo)
		}
		result := map[string]interface{}{
			"servers": serverList,
		}
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)
	}

	// Output servers in human-readable format
	fmt.Fprintln(cmd.OutOrStdout(), "Configured MCP Servers:")
	fmt.Fprintln(cmd.OutOrStdout())

	for name, server := range servers {
		status := "✓ enabled"
		if server.Disabled {
			status = "✗ disabled"
		}
		
		fmt.Fprintf(cmd.OutOrStdout(), "  %s %s\n", status, name)
		
		if server.Description != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "    Description: %s\n", server.Description)
		}
		
		fmt.Fprintf(cmd.OutOrStdout(), "    Command: %s %s\n", server.Command, strings.Join(server.Args, " "))
		
		if server.AutoApprove {
			fmt.Fprintln(cmd.OutOrStdout(), "    Auto-approve: yes")
		}
		
		if len(server.Env) > 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "    Environment:")
			for key := range server.Env {
				fmt.Fprintf(cmd.OutOrStdout(), "      %s=***\n", key)
			}
		}
		
		fmt.Fprintln(cmd.OutOrStdout())
	}

	return nil
}

// runMCPEnable executes the mcp enable command
func runMCPEnable(cmd *cobra.Command, args []string) error {
	serverName := args[0]
	
	// Initialize storage
	ctx, err := storage.NewStorageContext("", "")
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer ctx.Close()

	// Load servers
	servers, err := loadMCPServers(ctx)
	if err != nil {
		return fmt.Errorf("failed to load MCP servers: %w", err)
	}

	// Check if server exists
	server, exists := servers[serverName]
	if !exists {
		return fmt.Errorf("MCP server '%s' not found", serverName)
	}

	// Enable server
	if !server.Disabled {
		fmt.Fprintf(cmd.OutOrStdout(), "MCP server '%s' is already enabled\n", serverName)
		return nil
	}

	server.Disabled = false
	servers[serverName] = server
	
	if err := saveMCPServers(ctx, servers); err != nil {
		return fmt.Errorf("failed to save MCP servers: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ MCP server '%s' enabled\n", serverName)
	return nil
}

// runMCPDisable executes the mcp disable command
func runMCPDisable(cmd *cobra.Command, args []string) error {
	serverName := args[0]
	
	// Initialize storage
	ctx, err := storage.NewStorageContext("", "")
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer ctx.Close()

	// Load servers
	servers, err := loadMCPServers(ctx)
	if err != nil {
		return fmt.Errorf("failed to load MCP servers: %w", err)
	}

	// Check if server exists
	server, exists := servers[serverName]
	if !exists {
		return fmt.Errorf("MCP server '%s' not found", serverName)
	}

	// Disable server
	if server.Disabled {
		fmt.Fprintf(cmd.OutOrStdout(), "MCP server '%s' is already disabled\n", serverName)
		return nil
	}

	server.Disabled = true
	servers[serverName] = server
	
	if err := saveMCPServers(ctx, servers); err != nil {
		return fmt.Errorf("failed to save MCP servers: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ MCP server '%s' disabled\n", serverName)
	return nil
}

// runCPMarketplace executes the mcp marketplace command
func runCPMarketplace(cmd *cobra.Command, args []string) error {
	// For now, show a curated list of popular MCP servers
	// In the future, this could fetch from an actual marketplace API
	
	servers := []MCPRegistryEntry{
		{
			ID:          "filesystem",
			Name:        "Filesystem",
			Description: "Access and manage files on the local filesystem",
			Author:      "Anthropic",
			Repository:  "https://github.com/modelcontextprotocol/servers",
			License:     "MIT",
			Stars:       1500,
			Downloads:   50000,
		},
		{
			ID:          "github",
			Name:        "GitHub",
			Description: "Interact with GitHub repositories, issues, and pull requests",
			Author:      "Anthropic",
			Repository:  "https://github.com/modelcontextprotocol/servers",
			License:     "MIT",
			Stars:       1200,
			Downloads:   35000,
		},
		{
			ID:          "postgres",
			Name:        "PostgreSQL",
			Description: "Query and manage PostgreSQL databases",
			Author:      "Anthropic",
			Repository:  "https://github.com/modelcontextprotocol/servers",
			License:     "MIT",
			Stars:       800,
			Downloads:   25000,
		},
		{
			ID:          "sqlite",
			Name:        "SQLite",
			Description: "Query and manage SQLite databases",
			Author:      "Anthropic",
			Repository:  "https://github.com/modelcontextprotocol/servers",
			License:     "MIT",
			Stars:       600,
			Downloads:   20000,
		},
		{
			ID:          "fetch",
			Name:        "Fetch",
			Description: "Fetch web content and convert to markdown",
			Author:      "Anthropic",
			Repository:  "https://github.com/modelcontextprotocol/servers",
			License:     "MIT",
			Stars:       500,
			Downloads:   18000,
		},
		{
			ID:          "brave-search",
			Name:        "Brave Search",
			Description: "Search the web using Brave Search API",
			Author:      "Anthropic",
			Repository:  "https://github.com/modelcontextprotocol/servers",
			License:     "MIT",
			Stars:       400,
			Downloads:   15000,
		},
		{
			ID:          "slack",
			Name:        "Slack",
			Description: "Interact with Slack workspaces",
			Author:      "Anthropic",
			Repository:  "https://github.com/modelcontextprotocol/servers",
			License:     "MIT",
			Stars:       350,
			Downloads:   12000,
		},
		{
			ID:          "google-maps",
			Name:        "Google Maps",
			Description: "Access Google Maps geocoding and directions",
			Author:      "Anthropic",
			Repository:  "https://github.com/modelcontextprotocol/servers",
			License:     "MIT",
			Stars:       300,
			Downloads:   10000,
		},
	}

	// Filter by search term if provided
	searchTerm := strings.ToLower(mcpFlags.name)
	if searchTerm != "" {
		var filtered []MCPRegistryEntry
		for _, server := range servers {
			if strings.Contains(strings.ToLower(server.Name), searchTerm) ||
			   strings.Contains(strings.ToLower(server.Description), searchTerm) {
				filtered = append(filtered, server)
			}
		}
		servers = filtered
	}

	if len(servers) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No MCP servers found matching your search.")
		return nil
	}

	// Output servers
	fmt.Fprintln(cmd.OutOrStdout(), "Available MCP Servers:")
	fmt.Fprintln(cmd.OutOrStdout())
	
	for _, server := range servers {
		fmt.Fprintf(cmd.OutOrStdout(), "  📦 %s\n", server.Name)
		fmt.Fprintf(cmd.OutOrStdout(), "     %s\n", server.Description)
		fmt.Fprintf(cmd.OutOrStdout(), "     Author: %s | ⭐ %d | 📥 %d\n", 
			server.Author, server.Stars, server.Downloads)
		fmt.Fprintf(cmd.OutOrStdout(), "     Install: cline mcp add %s\n", server.ID)
		fmt.Fprintln(cmd.OutOrStdout())
	}

	fmt.Fprintln(cmd.OutOrStdout(), "To install a server, run: cline mcp add <name>")

	return nil
}

// loadMCPServers loads MCP servers from storage
func loadMCPServers(ctx *storage.StorageContext) (map[string]*MCPServer, error) {
	servers := make(map[string]*MCPServer)

	// Get from global state
	data, ok := ctx.GlobalState.Get("mcpServers")
	if !ok || data == nil {
		return servers, nil
	}

	// Convert to JSON and parse
	jsonData, err := json.Marshal(data)
	if err != nil {
		return servers, nil
	}

	if err := json.Unmarshal(jsonData, &servers); err != nil {
		// Try loading as array format
		var serverList []*MCPServer
		if err := json.Unmarshal(jsonData, &serverList); err != nil {
			return servers, nil
		}
		
		for _, server := range serverList {
			if server.Name != "" {
				servers[server.Name] = server
			}
		}
	}

	return servers, nil
}

// saveMCPServers saves MCP servers to storage
func saveMCPServers(ctx *storage.StorageContext, servers map[string]*MCPServer) error {
	return ctx.GlobalState.Set("mcpServers", servers)
}

// fetchMarketplaceServer fetches server info from marketplace
func fetchMarketplaceServer(name string) (*MCPRegistryEntry, error) {
	// For now, return a placeholder
	// In the future, this would query an actual marketplace API
	return &MCPRegistryEntry{
		ID:          name,
		Name:        name,
		Description: fmt.Sprintf("MCP server: %s", name),
	}, nil
}

// GetMCPServersPath returns the path to MCP servers configuration
func GetMCPServersPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".cline", "mcp-servers.json")
}

// ValidateMCPServer validates an MCP server configuration
func ValidateMCPServer(server *MCPServer) error {
	if server.Name == "" {
		return fmt.Errorf("server name is required")
	}
	if server.Command == "" {
		return fmt.Errorf("server command is required")
	}
	return nil
}

// FormatMCPCommand formats an MCP server command for display
func FormatMCPCommand(server *MCPServer) string {
	if len(server.Args) == 0 {
		return server.Command
	}
	return fmt.Sprintf("%s %s", server.Command, strings.Join(server.Args, " "))
}

// IsMCPServerRunning checks if an MCP server process is running
// This is a placeholder - actual implementation would check process state
func IsMCPServerRunning(name string) bool {
	// TODO: Implement actual process checking
	return false
}

// StartMCPServer starts an MCP server process
// This is a placeholder - actual implementation would start the process
func StartMCPServer(server *MCPServer) error {
	if err := ValidateMCPServer(server); err != nil {
		return err
	}
	// TODO: Implement actual server startup
	return fmt.Errorf("MCP server startup not yet implemented")
}

// StopMCPServer stops an MCP server process
// This is a placeholder - actual implementation would stop the process
func StopMCPServer(name string) error {
	// TODO: Implement actual server shutdown
	return fmt.Errorf("MCP server shutdown not yet implemented")
}

// RestartMCPServer restarts an MCP server
func RestartMCPServer(server *MCPServer) error {
	if err := StopMCPServer(server.Name); err != nil {
		// Server might not be running, that's OK
	}
	return StartMCPServer(server)
}

// GetMCPServerStatus returns the status of an MCP server
func GetMCPServerStatus(name string) string {
	if IsMCPServerRunning(name) {
		return "running"
	}
	return "stopped"
}

// ListMCPResources lists resources available from an MCP server
// This is a placeholder - actual implementation would query the server
func ListMCPResources(serverName string) ([]string, error) {
	return nil, fmt.Errorf("MCP resource listing not yet implemented")
}

// ListMCPTools lists tools available from an MCP server
// This is a placeholder - actual implementation would query the server
func ListMCPTools(serverName string) ([]string, error) {
	return nil, fmt.Errorf("MCP tool listing not yet implemented")
}