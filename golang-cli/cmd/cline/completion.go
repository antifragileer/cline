package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/cline/cline/golang-cli/internal/storage"
	"github.com/spf13/cobra"
)

// completionCmd represents the completion command
var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion script",
	Long: `Generate shell completion script for Cline CLI.

To load completions:

Bash:
  $ source <(cline completion bash)
  # To load completions for each session, execute once:
  # Linux:
  $ cline completion bash > /etc/bash_completion.d/cline
  # macOS:
  $ cline completion bash > $(brew --prefix)/etc/bash_completion.d/cline

Zsh:
  $ source <(cline completion zsh)
  # To load completions for each session, execute once:
  $ cline completion zsh > "${fpath[1]}/_cline"

Fish:
  $ cline completion fish | source
  # To load completions for each session, execute once:
  $ cline completion fish > ~/.config/fish/completions/cline.fish

PowerShell:
  PS> cline completion powershell | Out-String | Invoke-Expression
  # To load completions for every new session, run:
  PS> cline completion powershell > cline.ps1
  # and source this file from your PowerShell profile.`,
	Example: `  cline completion bash
  cline completion zsh
  cline completion fish
  cline completion powershell`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			return rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			return rootCmd.GenFishCompletion(os.Stdout, true)
		case "powershell":
			return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
		default:
			return fmt.Errorf("unsupported shell type: %s", args[0])
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}

// registerCompletionFunctions registers dynamic completion functions
func registerCompletionFunctions() {
	// Task ID completion
	_ = rootCmd.RegisterFlagCompletionFunc("taskId", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// Try to get task history for completion
		ctx, err := storage.NewStorageContext("", getWorkspaceHash())
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		defer ctx.Close()

		tasks, err := loadTasks(ctx)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		var completions []string
		for _, task := range tasks {
			if strings.HasPrefix(task.ID, toComplete) {
				desc := task.Task
				if len(desc) > 40 {
					desc = desc[:37] + "..."
				}
				completions = append(completions, fmt.Sprintf("%s\t%s", task.ID, desc))
			}
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	})

	// Provider completion for auth command
	_ = rootCmd.RegisterFlagCompletionFunc("provider", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		providers := []string{
			"anthropic\tAnthropic Claude API",
			"openai\tOpenAI API",
			"openai-native\tOpenAI Native API",
			"moonshot\tMoonshot AI",
			"deepseek\tDeepSeek AI",
			"ollama\tLocal models via Ollama",
		}

		var completions []string
		for _, p := range providers {
			if strings.HasPrefix(strings.ToLower(p), strings.ToLower(toComplete)) {
				completions = append(completions, p)
			}
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	})

	// Mode completion
	_ = rootCmd.RegisterFlagCompletionFunc("mode", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		modes := []string{
			"act\tExecute mode (default)",
			"plan\tPlanning mode",
		}

		var completions []string
		for _, m := range modes {
			if strings.HasPrefix(strings.ToLower(m), strings.ToLower(toComplete)) {
				completions = append(completions, m)
			}
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	})
}
