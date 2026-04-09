// Package parity provides side-by-side testing of Node.js and GoLang CLIs
package parity

import (
	"fmt"
	"regexp"
	"time"
)

// Category represents a test scenario category
type Category string

const (
	CategoryHelp     Category = "help"
	CategoryVersion  Category = "version"
	CategoryTask     Category = "task"
	CategoryHistory  Category = "history"
	CategoryConfig   Category = "config"
	CategoryAuth     Category = "auth"
	CategoryTUI      Category = "tui"
	CategoryError    Category = "error"
	CategorySettings Category = "settings"
)

// ExecutionMode represents how the CLI should be executed
type ExecutionMode string

const (
	ExecutionModePlain       ExecutionMode = "plain"
	ExecutionModeInteractive ExecutionMode = "interactive"
	ExecutionModeJSON        ExecutionMode = "json"
)

// Validator is a custom validation function
type Validator func(expected, actual *Execution) error

// Scenario represents a single parity test case
type Scenario struct {
	Name            string            // Test name
	Description     string            // Human-readable description
	Category        Category          // Test category
	Args            []string          // CLI arguments
	Stdin           string            // Input to stdin (if any)
	Env             map[string]string // Environment variables
	Timeout         time.Duration     // Execution timeout
	Mode            ExecutionMode     // Interactive vs Plain vs JSON
	ExpectExitCode  int               // Expected exit code
	SkipNormalization []string        // Normalizers to skip by name
	CustomValidators  []Validator     // Additional validation
	ExpectBaseline  bool              // Whether to compare against baseline
}

// Registry contains all test scenarios
type Registry struct {
	scenarios []Scenario
}

// NewRegistry creates a new scenario registry with all defined scenarios
func NewRegistry() *Registry {
	r := &Registry{}
	r.registerAllScenarios()
	return r
}

// GetAll returns all registered scenarios
func (r *Registry) GetAll() []Scenario {
	return r.scenarios
}

// GetByCategory returns scenarios filtered by category
func (r *Registry) GetByCategory(cat Category) []Scenario {
	var filtered []Scenario
	for _, s := range r.scenarios {
		if s.Category == cat {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

// GetByName returns a scenario by name
func (r *Registry) GetByName(name string) (Scenario, bool) {
	for _, s := range r.scenarios {
		if s.Name == name {
			return s, true
		}
	}
	return Scenario{}, false
}

// Add adds a custom scenario to the registry
func (r *Registry) Add(s Scenario) {
	r.scenarios = append(r.scenarios, s)
}

// registerAllScenarios registers all built-in test scenarios
func (r *Registry) registerAllScenarios() {
	// Help category
	r.addHelpScenarios()
	
	// Version category
	r.addVersionScenarios()
	
	// Task category
	r.addTaskScenarios()
	
	// History category
	r.addHistoryScenarios()
	
	// Config category
	r.addConfigScenarios()
	
	// Auth category
	r.addAuthScenarios()
	
	// Error category
	r.addErrorScenarios()
}

func (r *Registry) addHelpScenarios() {
	r.scenarios = append(r.scenarios, []Scenario{
		{
			Name:           "help_flag",
			Description:    "Basic help output with --help flag",
			Category:       CategoryHelp,
			Args:           []string{"--help"},
			Timeout:        5 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: true,
		},
		{
			Name:           "help_command",
			Description:    "Help subcommand",
			Category:       CategoryHelp,
			Args:           []string{"help"},
			Timeout:        5 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: true,
		},
		{
			Name:           "help_short",
			Description:    "Help with -h short flag",
			Category:       CategoryHelp,
			Args:           []string{"-h"},
			Timeout:        5 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: true,
		},
		{
			Name:           "help_task",
			Description:    "Help for task subcommand",
			Category:       CategoryHelp,
			Args:           []string{"help", "task"},
			Timeout:        5 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: true,
		},
		{
			Name:           "help_auth",
			Description:    "Help for auth subcommand",
			Category:       CategoryHelp,
			Args:           []string{"help", "auth"},
			Timeout:        5 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: true,
		},
	}...)
}

func (r *Registry) addVersionScenarios() {
	r.scenarios = append(r.scenarios, []Scenario{
		{
			Name:           "version_flag",
			Description:    "Version with --version flag (not supported, returns error)",
			Category:       CategoryVersion,
			Args:           []string{"--version"},
			Timeout:        5 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 1, // Node.js CLI doesn't support --version flag
			ExpectBaseline: true,
		},
		{
			Name:           "version_command",
			Description:    "Version subcommand",
			Category:       CategoryVersion,
			Args:           []string{"version"},
			Timeout:        5 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: true,
		},
		{
			Name:           "version_short",
			Description:    "Version with -v short flag (not supported as version, returns error)",
			Category:       CategoryVersion,
			Args:           []string{"-v"},
			Timeout:        5 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0, // -v is verbose flag, not version
			ExpectBaseline: true,
			SkipNormalization: []string{"version"}, // Don't normalize version numbers
		},
	}...)
}

func (r *Registry) addTaskScenarios() {
	r.scenarios = append(r.scenarios, []Scenario{
		{
			Name:           "task_simple",
			Description:    "Simple task execution",
			Category:       CategoryTask,
			Args:           []string{"task", "hello"},
			Timeout:        60 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false, // AI output varies
		},
		{
			Name:           "task_act_mode",
			Description:    "Task with act mode (-a flag)",
			Category:       CategoryTask,
			Args:           []string{"task", "-a", "hello"},
			Timeout:        60 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "task_plan_mode",
			Description:    "Task with plan mode (-p flag)",
			Category:       CategoryTask,
			Args:           []string{"task", "-p", "hello"},
			Timeout:        60 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "task_yolo",
			Description:    "Task with yolo mode (-y flag)",
			Category:       CategoryTask,
			Args:           []string{"task", "-y", "hello"},
			Timeout:        60 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "task_json",
			Description:    "Task with JSON output",
			Category:       CategoryTask,
			Args:           []string{"task", "--json", "hello"},
			Timeout:        60 * time.Second,
			Mode:           ExecutionModeJSON,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "task_timeout",
			Description:    "Task with timeout flag",
			Category:       CategoryTask,
			Args:           []string{"task", "-t", "30", "hello"},
			Timeout:        60 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
		{
			Name:           "task_root",
			Description:    "Task as root command (no 'task' subcommand)",
			Category:       CategoryTask,
			Args:           []string{"hello"},
			Timeout:        60 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: false,
		},
	}...)
}

func (r *Registry) addHistoryScenarios() {
	r.scenarios = append(r.scenarios, []Scenario{
		{
			Name:           "history_list",
			Description:    "List task history",
			Category:       CategoryHistory,
			Args:           []string{"history"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: true,
		},
		{
			Name:           "history_paginated",
			Description:    "Paginated history list",
			Category:       CategoryHistory,
			Args:           []string{"history", "-n", "5", "-p", "1"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: true,
		},
		{
			Name:           "history_json",
			Description:    "History with JSON output",
			Category:       CategoryHistory,
			Args:           []string{"history", "--json"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModeJSON,
			ExpectExitCode: 0,
			ExpectBaseline: true,
		},
	}...)
}

func (r *Registry) addConfigScenarios() {
	r.scenarios = append(r.scenarios, []Scenario{
		{
			Name:           "config_show",
			Description:    "Show configuration",
			Category:       CategoryConfig,
			Args:           []string{"config"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: true,
		},
		{
			Name:           "config_json",
			Description:    "Config with JSON output",
			Category:       CategoryConfig,
			Args:           []string{"config", "--json"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModeJSON,
			ExpectExitCode: 0,
			ExpectBaseline: true,
		},
	}...)
}

func (r *Registry) addAuthScenarios() {
	r.scenarios = append(r.scenarios, []Scenario{
		{
			Name:           "auth_help",
			Description:    "Auth command help",
			Category:       CategoryAuth,
			Args:           []string{"auth", "--help"},
			Timeout:        5 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: true,
		},
		{
			Name:           "auth_list",
			Description:    "List auth providers",
			Category:       CategoryAuth,
			Args:           []string{"auth", "list"},
			Timeout:        10 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 0,
			ExpectBaseline: true,
		},
	}...)
}

func (r *Registry) addErrorScenarios() {
	r.scenarios = append(r.scenarios, []Scenario{
		{
			Name:           "invalid_flag",
			Description:    "Invalid flag handling",
			Category:       CategoryError,
			Args:           []string{"--invalid-flag"},
			Timeout:        5 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 1,
			ExpectBaseline: true,
		},
		{
			Name:           "invalid_subcommand",
			Description:    "Invalid subcommand handling",
			Category:       CategoryError,
			Args:           []string{"invalidcmd"},
			Timeout:        5 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 1,
			ExpectBaseline: true,
		},
		{
			Name:           "missing_arg",
			Description:    "Missing required argument",
			Category:       CategoryError,
			Args:           []string{"task"}, // Missing prompt argument
			Timeout:        5 * time.Second,
			Mode:           ExecutionModePlain,
			ExpectExitCode: 1,
			ExpectBaseline: true,
		},
	}...)
}

// NormalizeScenarioName converts a scenario name to a valid filename
func NormalizeScenarioName(name string) string {
	// Replace non-alphanumeric characters with underscores
	re := regexp.MustCompile(`[^a-zA-Z0-9_-]+`)
	return re.ReplaceAllString(name, "_")
}

// GetScenarioFilename returns the baseline filename for a scenario
func GetScenarioFilename(scenario Scenario, extension string) string {
	return fmt.Sprintf("%s_%s.%s", scenario.Category, NormalizeScenarioName(scenario.Name), extension)
}