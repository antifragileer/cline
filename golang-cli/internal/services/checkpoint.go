// Package services provides gRPC service implementations for the Cline CLI.
package services

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/cline/cline/golang-cli/internal/generated/cline/cline"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// CheckpointService implements the CheckpointsService gRPC interface
type CheckpointService struct {
	cline.UnimplementedCheckpointsServiceServer
}

// NewCheckpointService creates a new CheckpointService instance
func NewCheckpointService() *CheckpointService {
	return &CheckpointService{}
}

// CheckpointDiff shows the diff for a checkpoint
func (s *CheckpointService) CheckpointDiff(ctx context.Context, req *cline.Int64Request) (*cline.Empty, error) {
	// Get the checkpoint number
	checkpointNum := req.Value

	// Find the git repository
	gitDir, err := s.findGitRepo()
	if err != nil {
		return nil, fmt.Errorf("not in a git repository: %w", err)
	}

	// Show diff for the checkpoint
	cmd := exec.CommandContext(ctx, "git", "-C", gitDir, "diff", fmt.Sprintf("HEAD~%d", checkpointNum))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to show diff: %w\nOutput: %s", err, string(output))
	}

	// In a real implementation, this would send the diff to the UI
	// For now, just print it
	fmt.Println(string(output))

	return &cline.Empty{}, nil
}

// CheckpointRestore restores to a checkpoint
func (s *CheckpointService) CheckpointRestore(ctx context.Context, req *cline.CheckpointRestoreRequest) (*cline.Empty, error) {
	// Find the git repository
	gitDir, err := s.findGitRepo()
	if err != nil {
		return nil, fmt.Errorf("not in a git repository: %w", err)
	}

	var cmd *exec.Cmd

	switch req.RestoreType {
	case "soft":
		// Soft reset - keep changes in working directory
		cmd = exec.CommandContext(ctx, "git", "-C", gitDir, "reset", "--soft", fmt.Sprintf("HEAD~%d", req.Number))
	case "hard":
		// Hard reset - discard all changes
		cmd = exec.CommandContext(ctx, "git", "-C", gitDir, "reset", "--hard", fmt.Sprintf("HEAD~%d", req.Number))
	default:
		// Default to mixed reset
		cmd = exec.CommandContext(ctx, "git", "-C", gitDir, "reset", fmt.Sprintf("HEAD~%d", req.Number))
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to restore checkpoint: %w\nOutput: %s", err, string(output))
	}

	return &cline.Empty{}, nil
}

// SubscribeToCheckpoints subscribes to checkpoint events
func (s *CheckpointService) SubscribeToCheckpoints(req *cline.CheckpointSubscriptionRequest, stream cline.CheckpointsService_SubscribeToCheckpointsServer) error {
	// Get the cwd hash for this subscription
	cwdHash := req.CwdHash

	// Send initial checkpoint state
	event := &cline.CheckpointEvent{
		Operation: cline.CheckpointEvent_CHECKPOINT_INIT,
		CwdHash:   cwdHash,
		IsActive:  true,
		Timestamp: timestamppb.Now(),
	}

	if err := stream.Send(event); err != nil {
		return fmt.Errorf("failed to send checkpoint event: %w", err)
	}

	// Keep stream open to listen for checkpoint events
	// In a real implementation, this would watch for git commits
	<-stream.Context().Done()
	return nil
}

// GetCwdHash returns a hash for the current working directory
func (s *CheckpointService) GetCwdHash(ctx context.Context, req *cline.StringArrayRequest) (*cline.PathHashMap, error) {
	result := &cline.PathHashMap{
		PathHash: make(map[string]string),
	}

	for _, path := range req.Value {
		absPath, err := filepath.Abs(path)
		if err != nil {
			continue
		}

		// Simple hash: use the directory name
		hash := filepath.Base(absPath)
		result.PathHash[path] = hash
	}

	return result, nil
}

// Helper methods

func (s *CheckpointService) findGitRepo() (string, error) {
	// Start from current directory
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Walk up the directory tree
	for {
		gitDir := filepath.Join(dir, ".git")
		if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
			return dir, nil
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// We've reached the root
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("no .git directory found")
}

// CreateCheckpoint creates a new checkpoint (git commit)
func (s *CheckpointService) CreateCheckpoint(ctx context.Context, message string) (string, error) {
	gitDir, err := s.findGitRepo()
	if err != nil {
		return "", err
	}

	// Stage all changes
	cmd := exec.CommandContext(ctx, "git", "-C", gitDir, "add", "-A")
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to stage changes: %w\nOutput: %s", err, string(output))
	}

	// Create commit
	cmd = exec.CommandContext(ctx, "git", "-C", gitDir, "commit", "-m", message)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to create checkpoint: %w\nOutput: %s", err, string(output))
	}

	// Get the commit hash
	cmd = exec.CommandContext(ctx, "git", "-C", gitDir, "rev-parse", "HEAD")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get commit hash: %w", err)
	}

	return string(output), nil
}