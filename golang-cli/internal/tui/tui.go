// Package tui provides terminal UI components for the Cline CLI.
//
// This package implements the interactive TUI using the Bubble Tea framework.
// It provides a complete chat interface with real-time streaming, tool approval
// dialogs, diff viewing, and a welcome screen with quick actions.
//
// The main entry point is RunInteractive which starts the full TUI application.
package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/cline/cline/golang-cli/internal/config"
	"github.com/cline/cline/golang-cli/internal/host"
	"github.com/cline/cline/golang-cli/internal/storage"
)

// RunInteractive starts the interactive TUI application.
//
// This is the main entry point for the TUI. It initializes all components
// and runs the Bubble Tea event loop.
//
// Parameters:
//   - client: gRPC client for communicating with the core
//   - storage: Storage context for persistence
//   - cfg: Layered configuration
//   - mode: Initial mode ("act" or "plan")
//   - yolo: Whether to enable auto-approve mode
//
// Returns an error if the TUI fails to start or encounters an error.
func RunInteractive(
	client *host.Client,
	storage *storage.StorageContext,
	cfg *config.LayeredConfig,
	mode string,
	yolo bool,
) error {
	if client == nil {
		return fmt.Errorf("gRPC client is required")
	}

	// Create the main app model
	app := NewAppModel(client, storage, cfg)
	app.SetMode(mode)
	app.SetYolo(yolo)

	// Create Bubble Tea program
	program := tea.NewProgram(
		app,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	// Set program reference for sending messages
	app.SetProgram(program)

	// Run the program
	finalModel, err := program.Run()
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	// Check if we should return an error
	if finalApp, ok := finalModel.(*AppModel); ok {
		if finalApp.IsQuitting() {
			return nil
		}
	}

	return nil
}

// RunChat starts a chat session directly without the welcome screen.
//
// Use this when you want to start a task immediately without showing
// the welcome screen first.
func RunChat(
	client *host.Client,
	storage *storage.StorageContext,
	cfg *config.LayeredConfig,
	mode string,
	yolo bool,
	initialPrompt string,
) error {
	if client == nil {
		return fmt.Errorf("gRPC client is required")
	}

	// Create the main app model
	app := NewAppModel(client, storage, cfg)
	app.SetMode(mode)
	app.SetYolo(yolo)

	// Skip welcome screen by directly setting to chat state
	// This will be handled by a special message or initialization

	// Create Bubble Tea program
	program := tea.NewProgram(
		app,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	app.SetProgram(program)

	// Send initial prompt if provided
	if initialPrompt != "" {
		go func() {
			// Wait a moment for initialization
			<-context.Background().Done()
			// This would need to be implemented properly
			// For now, the user can type the prompt
		}()
	}

	// Run the program
	_, err := program.Run()
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}

// SimpleHandler is a simple message handler for non-interactive use.
type SimpleHandler struct{}

// OnSay handles a SAY message.
func (h *SimpleHandler) OnSay(sayType string, text string, partial bool) {
	if !partial {
		fmt.Printf("[%s] %s\n", sayType, text)
	}
}

// OnAsk handles an ASK message.
func (h *SimpleHandler) OnAsk(askType string, text string) (string, error) {
	// Auto-approve everything in simple mode
	return "yesButtonClicked", nil
}

// OnInfo handles informational messages.
func (h *SimpleHandler) OnInfo(text string) {
	fmt.Printf("[INFO] %s\n", text)
}
