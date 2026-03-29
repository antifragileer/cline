package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/cline/cline/golang-cli/internal/storage"
)

// Workflow represents a Cline workflow configuration
type Workflow struct {
	Path        string `json:"path"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Enabled     bool   `json:"enabled"`
	Source      string `json:"source,omitempty"` // "global" or "local"
}

// workflowsFlags holds flags for workflows commands
var workflowsFlags struct {
	json   bool
	global bool
}

// workflowsCmd represents the workflows command
var workflowsCmd = &cobra.Command{
	Use:   "workflows",
	Short: "Manage Cline workflows",
	Long: `Manage Cline workflows for automated task execution.

Workflows are predefined sequences of actions that can be triggered
to automate common tasks and processes.`,
	Example: `  # List all workflows
  cline workflows list

  # List workflows in JSON format
  cline workflows list --json

  # List only global workflows
  cline workflows list --global`,
}

// workflowsListCmd represents the workflows list subcommand
var workflowsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured workflows",
	Long:  `List all configured workflows with their status and description.`,
	Example: `  cline workflows list
  cline workflows list --json
  cline workflows list --global`,
	RunE: runWorkflowsList,
}

// workflowsEnableCmd represents the workflows enable subcommand
var workflowsEnableCmd = &cobra.Command{
	Use:   "enable <path>",
	Short: "Enable a workflow",
	Long:  `Enable a previously disabled workflow.`,
	Example: `  cline workflows enable my-workflow`,
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkflowsEnable,
}

// workflowsDisableCmd represents the workflows disable subcommand
var workflowsDisableCmd = &cobra.Command{
	Use:   "disable <path>",
	Short: "Disable a workflow",
	Long:  `Disable a workflow without removing it from configuration.`,
	Example: `  cline workflows disable my-workflow`,
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkflowsDisable,
}

func init() {
	rootCmd.AddCommand(workflowsCmd)

	// Add subcommands
	workflowsCmd.AddCommand(workflowsListCmd)
	workflowsCmd.AddCommand(workflowsEnableCmd)
	workflowsCmd.AddCommand(workflowsDisableCmd)

	// Add flags
	workflowsListCmd.Flags().BoolVarP(&workflowsFlags.json, "json", "j", false, "Output in JSON format")
	workflowsListCmd.Flags().BoolVarP(&workflowsFlags.global, "global", "g", false, "Show only global workflows")
}

// runWorkflowsList executes the workflows list command
func runWorkflowsList(cmd *cobra.Command, args []string) error {
	// Initialize storage
	ctx, err := storage.NewStorageContext("", getWorkspaceHash())
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer ctx.Close()

	// Load global workflows from toggles
	var globalWorkflows []Workflow
	if val, ok := ctx.GlobalState.Get("globalWorkflowToggles"); ok {
		if toggles, err := parseToggles(val); err == nil {
			for path, enabled := range toggles {
				globalWorkflows = append(globalWorkflows, Workflow{
					Path:    path,
					Name:    filepath.Base(path),
					Enabled: enabled,
					Source:  "global",
				})
			}
		}
	}

	// Load local workflows from toggles
	var localWorkflows []Workflow
	if !workflowsFlags.global && ctx.WorkspaceState != nil {
		if val, ok := ctx.WorkspaceState.Get("localWorkflowToggles"); ok {
			if toggles, err := parseToggles(val); err == nil {
				for path, enabled := range toggles {
					localWorkflows = append(localWorkflows, Workflow{
						Path:    path,
						Name:    filepath.Base(path),
						Enabled: enabled,
						Source:  "local",
					})
				}
			}
		}
	}

	// Output as JSON if requested
	if workflowsFlags.json {
		output := map[string]interface{}{
			"global": globalWorkflows,
		}
		if !workflowsFlags.global {
			output["local"] = localWorkflows
		}
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	// Human-readable output
	if len(globalWorkflows) == 0 && len(localWorkflows) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No workflows configured.")
		fmt.Fprintln(cmd.OutOrStdout(), "\nWorkflows allow you to automate common tasks.")
		return nil
	}

	// Display global workflows
	if len(globalWorkflows) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "=== Global Workflows ===")
		for _, workflow := range globalWorkflows {
			printWorkflow(cmd.OutOrStdout(), workflow)
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}

	// Display local workflows
	if !workflowsFlags.global && len(localWorkflows) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "=== Workspace Workflows ===")
		for _, workflow := range localWorkflows {
			printWorkflow(cmd.OutOrStdout(), workflow)
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}

	return nil
}

// printWorkflow prints a single workflow in human-readable format
func printWorkflow(w io.Writer, workflow Workflow) {
	status := "✓ enabled"
	if !workflow.Enabled {
		status = "✗ disabled"
	}

	name := workflow.Name
	if name == "" {
		name = filepath.Base(workflow.Path)
	}

	fmt.Fprintf(w, "  %s %s\n", status, name)
	if workflow.Description != "" {
		fmt.Fprintf(w, "    Description: %s\n", workflow.Description)
	}
	fmt.Fprintf(w, "    Path: %s\n", workflow.Path)
}

// runWorkflowsEnable executes the workflows enable command
func runWorkflowsEnable(cmd *cobra.Command, args []string) error {
	workflowPath := args[0]

	// Initialize storage
	ctx, err := storage.NewStorageContext("", getWorkspaceHash())
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer ctx.Close()

	// Try global workflows first
	if val, ok := ctx.GlobalState.Get("globalWorkflowToggles"); ok {
		if toggles, err := parseToggles(val); err == nil {
			if _, exists := toggles[workflowPath]; exists {
				toggles[workflowPath] = true
				if err := ctx.GlobalState.Set("globalWorkflowToggles", toggles); err != nil {
					return fmt.Errorf("failed to enable workflow: %w", err)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "✓ Workflow '%s' enabled\n", workflowPath)
				return nil
			}
		}
	}

	// Try local workflows
	if ctx.WorkspaceState != nil {
		if val, ok := ctx.WorkspaceState.Get("localWorkflowToggles"); ok {
			if toggles, err := parseToggles(val); err == nil {
				if _, exists := toggles[workflowPath]; exists {
					toggles[workflowPath] = true
					if err := ctx.WorkspaceState.Set("localWorkflowToggles", toggles); err != nil {
						return fmt.Errorf("failed to enable workflow: %w", err)
					}
					fmt.Fprintf(cmd.OutOrStdout(), "✓ Workflow '%s' enabled\n", workflowPath)
					return nil
				}
			}
		}
	}

	return fmt.Errorf("workflow '%s' not found", workflowPath)
}

// runWorkflowsDisable executes the workflows disable command
func runWorkflowsDisable(cmd *cobra.Command, args []string) error {
	workflowPath := args[0]

	// Initialize storage
	ctx, err := storage.NewStorageContext("", getWorkspaceHash())
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer ctx.Close()

	// Try global workflows first
	if val, ok := ctx.GlobalState.Get("globalWorkflowToggles"); ok {
		if toggles, err := parseToggles(val); err == nil {
			if _, exists := toggles[workflowPath]; exists {
				toggles[workflowPath] = false
				if err := ctx.GlobalState.Set("globalWorkflowToggles", toggles); err != nil {
					return fmt.Errorf("failed to disable workflow: %w", err)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "✓ Workflow '%s' disabled\n", workflowPath)
				return nil
			}
		}
	}

	// Try local workflows
	if ctx.WorkspaceState != nil {
		if val, ok := ctx.WorkspaceState.Get("localWorkflowToggles"); ok {
			if toggles, err := parseToggles(val); err == nil {
				if _, exists := toggles[workflowPath]; exists {
					toggles[workflowPath] = false
					if err := ctx.WorkspaceState.Set("localWorkflowToggles", toggles); err != nil {
						return fmt.Errorf("failed to disable workflow: %w", err)
					}
					fmt.Fprintf(cmd.OutOrStdout(), "✓ Workflow '%s' disabled\n", workflowPath)
					return nil
				}
			}
		}
	}

	return fmt.Errorf("workflow '%s' not found", workflowPath)
}

// parseToggles parses toggle data from storage
func parseToggles(data interface{}) (map[string]bool, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var toggles map[string]bool
	if err := json.Unmarshal(jsonData, &toggles); err != nil {
		// Try as map[string]interface{} and convert
		var rawToggles map[string]interface{}
		if err := json.Unmarshal(jsonData, &rawToggles); err != nil {
			return nil, err
		}
		toggles = make(map[string]bool)
		for k, v := range rawToggles {
			if b, ok := v.(bool); ok {
				toggles[k] = b
			}
		}
	}

	return toggles, nil
}

// GetWorkflowsDir returns the workflows directory path
func GetWorkflowsDir(global bool) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	if global {
		return filepath.Join(homeDir, ".cline", "workflows"), nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	return filepath.Join(cwd, ".cline", "workflows"), nil
}

// ValidateWorkflowPath validates a workflow path
func ValidateWorkflowPath(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("workflow path cannot be empty")
	}

	// Check for invalid characters
	if strings.ContainsAny(path, "<>:\"|?*") {
		return fmt.Errorf("workflow path contains invalid characters")
	}

	return nil
}