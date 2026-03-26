// Package services provides service implementations for the Cline CLI.
package services

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// CheckpointManager manages git-based checkpoints for the agent
type CheckpointManager struct {
	// workingDir is the base directory for checkpoint operations
	workingDir string

	// gitDir is the root of the git repository
	gitDir string

	// checkpoints tracks created checkpoints
	checkpoints []Checkpoint
}

// Checkpoint represents a single checkpoint
type Checkpoint struct {
	Hash      string
	Message   string
	Timestamp time.Time
	Index     int
}

// NewCheckpointManager creates a new checkpoint manager
func NewCheckpointManager(workingDir string) (*CheckpointManager, error) {
	if workingDir == "" {
		var err error
		workingDir, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}
	}

	// Find git repository
	gitDir, err := findGitRepo(workingDir)
	if err != nil {
		// Initialize git if not found
		gitDir = workingDir
		if err := initGitRepo(gitDir); err != nil {
			return nil, fmt.Errorf("failed to initialize git repository: %w", err)
		}
	}

	return &CheckpointManager{
		workingDir:  workingDir,
		gitDir:      gitDir,
		checkpoints: make([]Checkpoint, 0),
	}, nil
}

// CreateCheckpoint creates a new git checkpoint
func (cm *CheckpointManager) CreateCheckpoint(message string) (string, error) {
	if message == "" {
		message = fmt.Sprintf("Checkpoint at %s", time.Now().Format("2006-01-02 15:04:05"))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stage all changes
	cmd := exec.CommandContext(ctx, "git", "-C", cm.gitDir, "add", "-A")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if there are any changes to commit
		if strings.Contains(string(output), "nothing to commit") {
			// Get current HEAD
			return cm.getCurrentHash()
		}
		// Ignore errors from add (e.g., no changes)
		_ = output
	}

	// Check if there are changes to commit
	if !cm.hasChanges() {
		return cm.getCurrentHash()
	}

	// Create commit
	cmd = exec.CommandContext(ctx, "git", "-C", cm.gitDir, "commit", "-m", message, "--allow-empty")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create checkpoint: %w\nOutput: %s", err, string(output))
	}

	// Get the commit hash
	hash, err := cm.getCurrentHash()
	if err != nil {
		return "", err
	}

	// Track the checkpoint
	checkpoint := Checkpoint{
		Hash:      hash,
		Message:   message,
		Timestamp: time.Now(),
		Index:     len(cm.checkpoints),
	}
	cm.checkpoints = append(cm.checkpoints, checkpoint)

	return hash, nil
}

// RestoreCheckpoint restores to a specific checkpoint
func (cm *CheckpointManager) RestoreCheckpoint(hash string, hard bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	if hard {
		cmd = exec.CommandContext(ctx, "git", "-C", cm.gitDir, "reset", "--hard", hash)
	} else {
		// Soft reset - keep changes in working directory
		cmd = exec.CommandContext(ctx, "git", "-C", cm.gitDir, "reset", "--soft", hash)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to restore checkpoint: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// GetDiff returns the diff between the current state and a checkpoint
func (cm *CheckpointManager) GetDiff(hash string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "-C", cm.gitDir, "diff", hash)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get diff: %w\nOutput: %s", err, string(output))
	}

	return string(output), nil
}

// GetCheckpointHistory returns the list of checkpoints
func (cm *CheckpointManager) GetCheckpointHistory() []Checkpoint {
	return cm.checkpoints
}

// GetCurrentCheckpoint returns the current checkpoint hash
func (cm *CheckpointManager) GetCurrentCheckpoint() (string, error) {
	return cm.getCurrentHash()
}

// HasCheckpoints returns true if any checkpoints exist
func (cm *CheckpointManager) HasCheckpoints() bool {
	return len(cm.checkpoints) > 0
}

// getCurrentHash gets the current git HEAD hash
func (cm *CheckpointManager) getCurrentHash() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "-C", cm.gitDir, "rev-parse", "HEAD")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get current hash: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// hasChanges checks if there are uncommitted changes
func (cm *CheckpointManager) hasChanges() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "-C", cm.gitDir, "status", "--porcelain")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}

	return len(strings.TrimSpace(string(output))) > 0
}

// findGitRepo finds the git repository root starting from a directory
func findGitRepo(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}

	for {
		gitDir := filepath.Join(dir, ".git")
		if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("no git repository found")
}

// initGitRepo initializes a new git repository
func initGitRepo(dir string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Initialize git
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "init")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to init git: %w\nOutput: %s", err, string(output))
	}

	// Configure git user (required for commits)
	cmd = exec.CommandContext(ctx, "git", "-C", dir, "config", "user.email", "cline@localhost")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to configure git email: %w\nOutput: %s", err, string(output))
	}

	cmd = exec.CommandContext(ctx, "git", "-C", dir, "config", "user.name", "Cline")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to configure git name: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// GetWorkingDir returns the working directory
func (cm *CheckpointManager) GetWorkingDir() string {
	return cm.workingDir
}

// GetGitDir returns the git directory
func (cm *CheckpointManager) GetGitDir() string {
	return cm.gitDir
}

// CreateCheckpointWithTracking creates a checkpoint and returns the checkpoint info
func (cm *CheckpointManager) CreateCheckpointWithTracking(message string) (*Checkpoint, error) {
	hash, err := cm.CreateCheckpoint(message)
	if err != nil {
		return nil, err
	}

	checkpoint := Checkpoint{
		Hash:      hash,
		Message:   message,
		Timestamp: time.Now(),
		Index:     len(cm.checkpoints),
	}
	cm.checkpoints = append(cm.checkpoints, checkpoint)

	return &checkpoint, nil
}

// GetPreviousCheckpoint returns the checkpoint before the current one
func (cm *CheckpointManager) GetPreviousCheckpoint() (*Checkpoint, error) {
	if len(cm.checkpoints) < 2 {
		return nil, fmt.Errorf("no previous checkpoint available")
	}

	return &cm.checkpoints[len(cm.checkpoints)-2], nil
}

// CompareCheckpoints returns the diff between two checkpoints
func (cm *CheckpointManager) CompareCheckpoints(hash1, hash2 string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "-C", cm.gitDir, "diff", hash1, hash2)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to compare checkpoints: %w\nOutput: %s", err, string(output))
	}

	return string(output), nil
}

// GetCheckpointAt returns the checkpoint at a specific index
func (cm *CheckpointManager) GetCheckpointAt(index int) (*Checkpoint, error) {
	if index < 0 || index >= len(cm.checkpoints) {
		return nil, fmt.Errorf("checkpoint index out of range")
	}

	return &cm.checkpoints[index], nil
}

// DeleteCheckpoint removes a checkpoint (not typically used, but available)
func (cm *CheckpointManager) DeleteCheckpoint(hash string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use git reset to remove the commit
	cmd := exec.CommandContext(ctx, "git", "-C", cm.gitDir, "reset", "--hard", hash+"^")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete checkpoint: %w\nOutput: %s", err, string(output))
	}

	// Update internal tracking
	for i, cp := range cm.checkpoints {
		if cp.Hash == hash {
			cm.checkpoints = append(cm.checkpoints[:i], cm.checkpoints[i+1:]...)
			break
		}
	}

	return nil
}

// CleanCheckpoints removes old checkpoints, keeping only the most recent N
func (cm *CheckpointManager) CleanCheckpoints(keep int) error {
	if keep <= 0 {
		return fmt.Errorf("keep must be positive")
	}

	if len(cm.checkpoints) <= keep {
		return nil // Nothing to clean
	}

	// Remove old checkpoints from tracking
	cm.checkpoints = cm.checkpoints[len(cm.checkpoints)-keep:]

	return nil
}