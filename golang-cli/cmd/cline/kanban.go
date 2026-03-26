package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

// kanbanCmd represents the kanban command
var kanbanCmd = &cobra.Command{
	Use:   "kanban",
	Short: "Run kanban board with Cline agent integration",
	Long: `Run the kanban CLI tool with Cline agent integration.

This command executes 'npx kanban@latest --agent cline' to launch
an interactive kanban board that integrates with Cline.

The kanban board allows you to:
- Create and manage tasks
- Track progress on projects
- Organize work visually
- Integrate with Cline for AI-powered task assistance`,
	Example: `  # Launch the kanban board
  cline kanban`,
	RunE: runKanban,
}

func init() {
	rootCmd.AddCommand(kanbanCmd)
}

// runKanban executes the kanban command
func runKanban(cmd *cobra.Command, args []string) error {
	// Check if npx is available
	if _, err := exec.LookPath("npx"); err != nil {
		return fmt.Errorf("npx is required but not found in PATH. Please install Node.js and npm")
	}

	// Execute: npx kanban@latest --agent cline
	npxCmd := exec.Command("npx", "kanban@latest", "--agent", "cline")
	
	// Pass through stdin/stdout/stderr for interactive use
	npxCmd.Stdin = os.Stdin
	npxCmd.Stdout = os.Stdout
	npxCmd.Stderr = os.Stderr
	
	// Run the command
	if err := npxCmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Return the exit code from npx
			return fmt.Errorf("kanban exited with code %d", exitErr.ExitCode())
		}
		return fmt.Errorf("failed to run kanban: %w", err)
	}

	return nil
}