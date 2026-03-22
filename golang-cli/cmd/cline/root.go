package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	// Version is the CLI version
	Version = "0.1.0"

	// DefaultConfigName is the default configuration file name
	DefaultConfigName = "config"

	// DefaultConfigType is the default configuration file type
	DefaultConfigType = "yaml"

	// ConfigDirName is the name of the configuration directory
	ConfigDirName = "cline"
)

var (
	// Global flags
	cfgFile string
	verbose bool
	version bool

	// Logger instance
	logger *slog.Logger

	// rootCmd represents the base command when called without any subcommands
	rootCmd = &cobra.Command{
		Use:   "cline [prompt]",
		Short: "Cline CLI - AI-powered coding assistant",
		Long: `Cline is an AI-powered coding assistant that helps you write, edit, and understand code.

With Cline, you can:
  • Ask questions about your codebase
  • Generate new code files and functions
  • Refactor and improve existing code
  • Debug and fix issues

Usage:
  cline "your task or question here"    Execute a single task with the given prompt
  cline                                  Start interactive mode`,
		Example: `  # Execute a single task
  cline "refactor the auth module to use JWT"

  # Start interactive mode
  cline

  # Use a custom configuration file
  cline --config /path/to/config.yaml "explain this code"`,
		RunE: runRoot,
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig, initLogger)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", fmt.Sprintf("config file (default is $HOME/.%s/%s.%s)", ConfigDirName, DefaultConfigName, DefaultConfigType))
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.PersistentFlags().BoolVarP(&version, "version", "", false, "print version and exit")

	// Bind flags to viper
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))

	// Version flag handling
	rootCmd.SetVersionTemplate(fmt.Sprintf("Cline CLI version %s\n", Version))
	rootCmd.Version = Version
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
			os.Exit(1)
		}

		// Search config in home directory with name ".cline" (without extension).
		configDir := filepath.Join(home, "."+ConfigDirName)
		viper.AddConfigPath(configDir)
		viper.SetConfigName(DefaultConfigName)
		viper.SetConfigType(DefaultConfigType)
	}

	viper.SetEnvPrefix("CLINE")
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "Using config file: %s\n", viper.ConfigFileUsed())
		}
	}
}

// initLogger initializes the global logger
func initLogger() {
	var level slog.Level
	if verbose {
		level = slog.LevelDebug
	} else {
		level = slog.LevelInfo
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})
	logger = slog.New(handler)
	slog.SetDefault(logger)
}

// runRoot executes the root command logic
func runRoot(cmd *cobra.Command, args []string) error {
	// Handle version flag
	if version {
		fmt.Printf("Cline CLI version %s\n", Version)
		return nil
	}

	// Check if a prompt was provided as an argument
	if len(args) > 0 {
		prompt := args[0]
		logger.Debug("executing task mode", "prompt", prompt)
		return runTaskMode(prompt)
	}

	// No prompt provided, start interactive mode
	logger.Debug("starting interactive mode")
	return runInteractiveMode()
}

// runTaskMode executes a single task with the given prompt
func runTaskMode(prompt string) error {
	logger.Info("executing task", "prompt", prompt)

	// TODO: Implement task execution logic
	// This will integrate with the task package once implemented
	fmt.Printf("Task mode not yet implemented. Prompt: %s\n", prompt)

	return nil
}

// runInteractiveMode starts the interactive TUI
func runInteractiveMode() error {
	logger.Info("starting interactive mode")

	// TODO: Implement interactive mode logic
	// This will integrate with the TUI package once implemented
	fmt.Println("Interactive mode not yet implemented.")

	return nil
}