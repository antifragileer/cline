package main

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

func TestTaskNewCmd(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "simple prompt",
			args:    []string{"hello world"},
			wantErr: false,
		},
		{
			name:    "with act flag",
			args:    []string{"--act", "do something"},
			wantErr: false,
		},
		{
			name:    "with plan flag",
			args:    []string{"--plan", "plan something"},
			wantErr: false,
		},
		{
			name:    "with yolo flag",
			args:    []string{"--yolo", "auto approve task"},
			wantErr: false,
		},
		{
			name:    "with short flags",
			args:    []string{"-y", "-t", "300", "-m", "gpt-4", "task"},
			wantErr: false,
		},
		{
			name:    "with long flags",
			args:    []string{"--yolo", "--timeout", "600", "--model", "claude", "task"},
			wantErr: false,
		},
		{
			name:    "with taskId",
			args:    []string{"-T", "task-123", "resume task"},
			wantErr: false,
		},
		{
			name:    "with json flag",
			args:    []string{"--json", "json task"},
			wantErr: false,
		},
		{
			name:    "empty args",
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "mutually exclusive act and plan",
			args:    []string{"--act", "--plan", "task"},
			wantErr: true,
			errMsg:  "cannot use both --act and --plan flags",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh command for testing
			cmd := &cobra.Command{
				Use:   "new [prompt]",
				Short: "Create a new task",
				Args:  cobra.MinimumNArgs(0),
				RunE:  runTaskNew,
			}

			// Add flags
			cmd.Flags().BoolP("act", "a", false, "Run in act mode")
			cmd.Flags().BoolP("plan", "p", false, "Run in plan mode")
			cmd.Flags().BoolP("yolo", "y", false, "Enable yolo mode")
			cmd.Flags().Bool("auto-approve-all", false, "Auto-approve all actions")
			cmd.Flags().StringP("timeout", "t", "", "Timeout in seconds")
			cmd.Flags().StringP("model", "m", "", "Model to use")
			cmd.Flags().StringP("taskId", "T", "", "Resume existing task")
			cmd.Flags().Bool("json", false, "Output as JSON")
			cmd.Flags().String("thinking", "", "Enable thinking")
			cmd.Flags().String("reasoning-effort", "", "Reasoning effort")
			cmd.Flags().String("max-consecutive-mistakes", "", "Max mistakes")
			cmd.Flags().Bool("double-check-completion", false, "Double check")
			cmd.Flags().Bool("auto-condense", false, "Auto condense")
			cmd.Flags().String("hooks-dir", "", "Hooks directory")
			cmd.Flags().StringP("cwd", "c", "", "Working directory")
			cmd.Flags().StringArrayP("image", "i", nil, "Image attachment")
			cmd.Flags().BoolP("verbose", "v", false, "Verbose output")

			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && tt.errMsg != "" && err != nil {
				if err.Error() != tt.errMsg {
					t.Errorf("expected error '%s', got '%s'", tt.errMsg, err.Error())
				}
			}
		})
	}
}

func TestTaskListCmd(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List recent task history",
		RunE:  runTaskList,
	}

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}

	output := buf.String()
	if output != "Listing recent tasks...\n" {
		t.Errorf("unexpected output: %s", output)
	}
}

func TestTaskChatCmd(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "chat",
		Short: "Chat with the current task",
		RunE:  runTaskChat,
	}

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}

	output := buf.String()
	if output != "Starting chat mode...\n" {
		t.Errorf("unexpected output: %s", output)
	}
}

func TestTaskOpenCmd(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "valid task id",
			args:    []string{"task-123"},
			wantErr: false,
		},
		{
			name:    "no args",
			args:    []string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{
				Use:   "open <id>",
				Short: "Open a task by ID",
				Args:  cobra.ExactArgs(1),
				RunE:  runTaskOpen,
			}

			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				output := buf.String()
				expected := "Opening task: task-123\n"
				if output != expected {
					t.Errorf("unexpected output: got %q, want %q", output, expected)
				}
			}
		})
	}
}

func TestTaskSendCmd(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "with message",
			args:    []string{"hello"},
			wantErr: false,
		},
		{
			name:    "no args",
			args:    []string{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{
				Use:   "send [message]",
				Short: "Send a followup message",
				Args:  cobra.ArbitraryArgs,
				RunE:  runTaskSend,
			}

			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTaskViewCmd(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "valid task id",
			args:    []string{"task-456"},
			wantErr: false,
		},
		{
			name:    "no args",
			args:    []string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{
				Use:   "view <id>",
				Short: "View task conversation",
				Args:  cobra.ExactArgs(1),
				RunE:  runTaskView,
			}

			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				output := buf.String()
				expected := "Viewing task: task-456\n"
				if output != expected {
					t.Errorf("unexpected output: got %q, want %q", output, expected)
				}
			}
		})
	}
}

func TestTaskPauseCmd(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "pause",
		Short: "Pause the current task",
		RunE:  runTaskPause,
	}

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}

	output := buf.String()
	if output != "Pausing current task...\n" {
		t.Errorf("unexpected output: %s", output)
	}
}

func TestTaskRestoreCmd(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "valid task id",
			args:    []string{"task-789"},
			wantErr: false,
		},
		{
			name:    "no args",
			args:    []string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{
				Use:   "restore <id>",
				Short: "Restore task to checkpoint",
				Args:  cobra.ExactArgs(1),
				RunE:  runTaskRestore,
			}

			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				output := buf.String()
				expected := "Restoring task: task-789\n"
				if output != expected {
					t.Errorf("unexpected output: got %q, want %q", output, expected)
				}
			}
		})
	}
}

func TestTaskCmdInit(t *testing.T) {
	// Verify taskCmd is properly initialized
	if taskCmd.Use != "task" {
		t.Errorf("taskCmd.Use = %v, want 'task'", taskCmd.Use)
	}

	if len(taskCmd.Aliases) != 1 || taskCmd.Aliases[0] != "t" {
		t.Errorf("taskCmd.Aliases = %v, want ['t']", taskCmd.Aliases)
	}

	// Verify subcommands are added
	subcommands := taskCmd.Commands()
	expectedSubcommands := []string{"new", "list", "chat", "open", "send", "view", "pause", "restore"}

	if len(subcommands) != len(expectedSubcommands) {
		t.Errorf("expected %d subcommands, got %d", len(expectedSubcommands), len(subcommands))
	}

	subcommandNames := make(map[string]bool)
	for _, cmd := range subcommands {
		subcommandNames[cmd.Name()] = true
	}

	for _, expected := range expectedSubcommands {
		if !subcommandNames[expected] {
			t.Errorf("missing subcommand: %s", expected)
		}
	}
}
