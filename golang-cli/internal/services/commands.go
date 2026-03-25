// Package services provides gRPC service implementations for the Cline CLI.
package services

import (
	"context"
	"fmt"

	"github.com/cline/cline/golang-cli/internal/generated/cline/cline"
)

// CommandsService implements the CommandsService gRPC interface
// Note: This service is primarily used by the IDE (VSCode) to trigger actions.
// In CLI mode, these commands are handled differently.
type CommandsService struct {
	cline.UnimplementedCommandsServiceServer
}

// NewCommandsService creates a new CommandsService instance
func NewCommandsService() *CommandsService {
	return &CommandsService{}
}

// AddToCline adds the selected text/file to Cline as a new task
func (s *CommandsService) AddToCline(ctx context.Context, req *cline.CommandContext) (*cline.Empty, error) {
	// In CLI mode, this would create a new task with the provided context
	// For now, return an indication that this is handled via task creation
	return &cline.Empty{}, fmt.Errorf("use 'cline task' command to add files in CLI mode")
}

// FixWithCline creates a task to fix issues in the selected code
func (s *CommandsService) FixWithCline(ctx context.Context, req *cline.CommandContext) (*cline.Empty, error) {
	return &cline.Empty{}, fmt.Errorf("use 'cline task' command with fix prompt in CLI mode")
}

// ExplainWithCline creates a task to explain the selected code
func (s *CommandsService) ExplainWithCline(ctx context.Context, req *cline.CommandContext) (*cline.Empty, error) {
	return &cline.Empty{}, fmt.Errorf("use 'cline task' command with explain prompt in CLI mode")
}

// ImproveWithCline creates a task to improve the selected code
func (s *CommandsService) ImproveWithCline(ctx context.Context, req *cline.CommandContext) (*cline.Empty, error) {
	return &cline.Empty{}, fmt.Errorf("use 'cline task' command with improve prompt in CLI mode")
}
