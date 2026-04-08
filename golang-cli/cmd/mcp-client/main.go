// cmd/mcp-client/main.go
// MCP client command for managing external tool servers
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/cline/cline/golang-cli/internal/mcp"
	"github.com/spf13/cobra"
)

var (
	// Server flags
	serverURL     string
	serverCommand string
	serverArgs    []string
	serverEnv     map[string]string
	autoApprove   bool
	serverTimeout int

	// Operation flags
	listTools     bool
	listResources bool
	callTool      string
	toolArgs      string
	readResource  string
	resourceURI   string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "mcp-client",
		Short: "MCP client for managing external tool servers",
		Long: `The MCP client provides tools for managing Model Context Protocol servers.
		
This command allows you to:
  - Connect to MCP servers via stdio or SSE
  - List available tools and resources
  - Execute tools with arguments
  - Read resources from servers
  - Manage server lifecycle`,
	}

	// Add subcommands
	rootCmd.AddCommand(createConnectCommand())
	rootCmd.AddCommand(createListCommand())
	rootCmd.AddCommand(createCallCommand())
	rootCmd.AddCommand(createReadCommand())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func createConnectCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "connect",
		Short: "Connect to an MCP server",
		Long:  "Establish a connection to an MCP server via stdio or SSE transport.",
		RunE:  runConnect,
	}

	cmd.Flags().StringVarP(&serverURL, "url", "u", "", "SSE server URL (for HTTP transport)")
	cmd.Flags().StringVarP(&serverCommand, "command", "c", "", "Command to start server (for stdio transport)")
	cmd.Flags().StringArrayVarP(&serverArgs, "arg", "a", nil, "Arguments for server command")
	cmd.Flags().StringToStringVarP(&serverEnv, "env", "e", nil, "Environment variables (KEY=value)")
	cmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "Auto-approve all tool calls")
	cmd.Flags().IntVar(&serverTimeout, "timeout", 30, "Server connection timeout in seconds")

	return cmd
}

func createListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List MCP resources",
		Long:  "List available tools or resources from connected MCP servers.",
		RunE:  runList,
	}

	cmd.Flags().BoolVarP(&listTools, "tools", "t", true, "List available tools")
	cmd.Flags().BoolVarP(&listResources, "resources", "r", false, "List available resources")

	return cmd
}

func createCallCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "call [tool-name]",
		Short: "Call an MCP tool",
		Long:  "Execute a tool on an MCP server with the specified arguments.",
		Args:  cobra.ExactArgs(1),
		RunE:  runCall,
	}

	cmd.Flags().StringVarP(&toolArgs, "args", "a", "{}", "Tool arguments as JSON string")

	return cmd
}

func createReadCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "read [uri]",
		Short: "Read an MCP resource",
		Long:  "Read a resource from an MCP server by URI.",
		Args:  cobra.ExactArgs(1),
		RunE:  runRead,
	}

	return cmd
}

func runConnect(cmd *cobra.Command, args []string) error {
	// Validate inputs
	if serverURL == "" && serverCommand == "" {
		return fmt.Errorf("either --url or --command must be specified")
	}

	if serverURL != "" && serverCommand != "" {
		return fmt.Errorf("cannot specify both --url and --command")
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupts
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt, shutting down...")
		cancel()
	}()

	// Create client configuration
	config := &mcp.ClientConfig{
		AutoApprove: autoApprove,
		Timeout:     serverTimeout,
	}

	// Create transport based on flags
	var transport mcp.Transport
	if serverURL != "" {
		transport = mcp.NewSSETransport(serverURL)
	} else {
		transport = mcp.NewStdioTransport(serverCommand, serverArgs, serverEnv)
	}

	// Create and start client
	client := mcp.NewClient(transport, config)

	fmt.Println("Connecting to MCP server...")
	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	fmt.Println("Connected successfully!")

	// List available capabilities
	capabilities, err := client.GetCapabilities(ctx)
	if err != nil {
		return fmt.Errorf("failed to get capabilities: %w", err)
	}

	fmt.Println("\nServer capabilities:")
	if capabilities.Tools {
		fmt.Println("  ✓ Tools")
	}
	if capabilities.Resources {
		fmt.Println("  ✓ Resources")
	}
	if capabilities.Prompts {
		fmt.Println("  ✓ Prompts")
	}
	if capabilities.Logging {
		fmt.Println("  ✓ Logging")
	}

	// Keep connection alive until interrupted
	fmt.Println("\nConnected. Press Ctrl+C to disconnect.")
	<-ctx.Done()

	fmt.Println("Disconnecting...")
	if err := client.Disconnect(); err != nil {
		return fmt.Errorf("failed to disconnect: %w", err)
	}

	return nil
}

func runList(cmd *cobra.Command, args []string) error {
	if !listTools && !listResources {
		return fmt.Errorf("either --tools or --resources must be specified")
	}

	_ = context.Background()

	// This would typically connect to existing servers and list from them
	// For now, we show a placeholder
	fmt.Println("Listing MCP resources...")

	if listTools {
		fmt.Println("\nAvailable tools:")
		fmt.Println("  (connect to a server first to see available tools)")
	}

	if listResources {
		fmt.Println("\nAvailable resources:")
		fmt.Println("  (connect to a server first to see available resources)")
	}

	return nil
}

func runCall(cmd *cobra.Command, args []string) error {
	toolName := args[0]

	_ = context.Background()

	fmt.Printf("Calling tool: %s\n", toolName)
	fmt.Printf("Arguments: %s\n", toolArgs)

	// This would connect to server and call the tool
	// For now, show a placeholder
	fmt.Println("\nTool execution not implemented in this version.")
	fmt.Println("Use 'cline mcp call' instead.")

	return nil
}

func runRead(cmd *cobra.Command, args []string) error {
	uri := args[0]

	fmt.Printf("Reading resource: %s\n", uri)

	// This would connect to server and read the resource
	// For now, show a placeholder
	fmt.Println("\nResource reading not implemented in this version.")
	fmt.Println("Use 'cline mcp read' instead.")

	return nil
}
