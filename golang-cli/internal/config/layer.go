// Package config provides configuration layering functionality for the Cline CLI.
// It implements a hierarchical configuration system with the following precedence:
// defaults < global state < workspace state < environment variables (CLINE_*) < CLI flags
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// LayeredConfig provides a unified interface for accessing configuration
// from multiple sources with proper precedence.
type LayeredConfig struct {
	v *viper.Viper

	// Sources track which source each config value came from
	sources map[string]Source

	// cliFlags holds values explicitly set via CLI flags
	cliFlags map[string]interface{}

	// workspaceHash is the current workspace identifier
	workspaceHash string
}

// Source indicates where a configuration value originated from
type Source int

const (
	// SourceDefault indicates the value came from hardcoded defaults
	SourceDefault Source = iota
	// SourceGlobal indicates the value came from global state file
	SourceGlobal
	// SourceWorkspace indicates the value came from workspace state file
	SourceWorkspace
	// SourceEnv indicates the value came from environment variables
	SourceEnv
	// SourceCLI indicates the value came from CLI flags
	SourceCLI
)

// String returns a human-readable name for the source
func (s Source) String() string {
	switch s {
	case SourceDefault:
		return "default"
	case SourceGlobal:
		return "global"
	case SourceWorkspace:
		return "workspace"
	case SourceEnv:
		return "environment"
	case SourceCLI:
		return "cli"
	default:
		return "unknown"
	}
}

// ConfigOptions provides options for creating a new LayeredConfig
type ConfigOptions struct {
	// WorkspaceHash is the workspace identifier for workspace-specific config
	WorkspaceHash string

	// BaseDir is the base directory for storage files (defaults to ~/.cline/data)
	BaseDir string

	// EnvPrefix is the prefix for environment variables (defaults to "CLINE")
	EnvPrefix string
}

// NewLayeredConfig creates a new LayeredConfig with the given options.
// It initializes viper with the proper precedence order and loads
// configuration from all sources.
func NewLayeredConfig(opts ConfigOptions) (*LayeredConfig, error) {
	if opts.EnvPrefix == "" {
		opts.EnvPrefix = "CLINE"
	}

	lc := &LayeredConfig{
		v:             viper.New(),
		sources:       make(map[string]Source),
		cliFlags:      make(map[string]interface{}),
		workspaceHash: opts.WorkspaceHash,
	}

	// Set up viper
	lc.v.SetEnvPrefix(opts.EnvPrefix)
	lc.v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	lc.v.AutomaticEnv()

	// Set defaults
	lc.setDefaults()

	// Load from files
	if err := lc.loadFromFiles(opts.BaseDir); err != nil {
		return nil, fmt.Errorf("failed to load configuration files: %w", err)
	}

	// Track environment variable sources
	lc.trackEnvSources(opts.EnvPrefix)

	return lc, nil
}

// setDefaults sets hardcoded default values for configuration options
func (lc *LayeredConfig) setDefaults() {
	// API defaults
	lc.v.SetDefault("api.timeout", 120)
	lc.v.SetDefault("api.retries", 3)
	lc.v.SetDefault("api.provider", "anthropic")

	// Model defaults
	lc.v.SetDefault("model", "claude-3-sonnet-20240229")

	// UI defaults
	lc.v.SetDefault("ui.theme", "system")
	lc.v.SetDefault("ui.auto_update", true)

	// Feature flags
	lc.v.SetDefault("features.auto_approve", false)
	lc.v.SetDefault("features.checkpoints", true)

	// Track defaults - use a predefined list since AllKeys() won't include defaults
	defaultKeys := []string{
		"api.timeout", "api.retries", "api.provider",
		"model",
		"ui.theme", "ui.auto_update",
		"features.auto_approve", "features.checkpoints",
	}
	for _, key := range defaultKeys {
		lc.sources[key] = SourceDefault
	}
}

// loadFromFiles loads configuration from global and workspace state files
// Respects environment variable precedence by not setting keys that have env vars
func (lc *LayeredConfig) loadFromFiles(baseDir string) error {
	if baseDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		baseDir = filepath.Join(homeDir, ".cline", "data")
	}

	// Get list of keys that have environment variable overrides
	envKeys := make(map[string]bool)
	envPrefix := lc.v.GetString("envPrefix")
	if envPrefix == "" {
		envPrefix = "CLINE"
	}
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, envPrefix+"_") {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				key := envKeyToConfigKey(parts[0], envPrefix)
				envKeys[key] = true
			}
		}
	}

	// Load global state first (lowest priority after defaults)
	globalPath := filepath.Join(baseDir, "globalState.json")
	if _, err := os.Stat(globalPath); err == nil {
		globalViper := viper.New()
		globalViper.SetConfigFile(globalPath)
		if err := globalViper.ReadInConfig(); err != nil {
			// Log but don't fail - global config might be corrupted
			fmt.Fprintf(os.Stderr, "Warning: failed to load global config: %v\n", err)
		} else {
			// Merge global config into main viper, but skip keys with env vars
			settings := globalViper.AllSettings()
			for key, value := range flattenMap(settings, "") {
				if !envKeys[key] {
					lc.v.Set(key, value)
					lc.sources[key] = SourceGlobal
				}
			}
		}
	}

	// Load workspace state (higher priority than global)
	if lc.workspaceHash != "" {
		workspacePath := filepath.Join(baseDir, "workspaces", lc.workspaceHash, "workspaceState.json")
		if _, err := os.Stat(workspacePath); err == nil {
			// Create a separate viper instance for workspace config
			workspaceViper := viper.New()
			workspaceViper.SetConfigFile(workspacePath)
			if err := workspaceViper.ReadInConfig(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to load workspace config: %v\n", err)
			} else {
				// Merge workspace config into main viper, but skip keys with env vars
				settings := workspaceViper.AllSettings()
				for key, value := range flattenMap(settings, "") {
					if !envKeys[key] {
						lc.v.Set(key, value)
						lc.sources[key] = SourceWorkspace
					}
				}
			}
		}
	}

	return nil
}

// trackEnvSources tracks which keys are set via environment variables
func (lc *LayeredConfig) trackEnvSources(prefix string) {
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, prefix+"_") {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				key := envKeyToConfigKey(parts[0], prefix)
				lc.sources[key] = SourceEnv
			}
		}
	}
}

// envKeyToConfigKey converts an environment variable name to a config key
func envKeyToConfigKey(envKey, prefix string) string {
	// Remove prefix
	key := strings.TrimPrefix(envKey, prefix+"_")
	// Convert to lowercase and replace underscores with dots
	key = strings.ToLower(key)
	key = strings.ReplaceAll(key, "_", ".")
	return key
}

// flattenMap flattens a nested map into dot-notation keys
func flattenMap(m map[string]interface{}, prefix string) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		switch val := v.(type) {
		case map[string]interface{}:
			for k2, v2 := range flattenMap(val, key) {
				result[k2] = v2
			}
		default:
			result[key] = v
		}
	}
	return result
}

// SetCLIFlag sets a configuration value from a CLI flag.
// CLI flags have the highest precedence.
func (lc *LayeredConfig) SetCLIFlag(key string, value interface{}) {
	lc.cliFlags[key] = value
	lc.v.Set(key, value)
	lc.sources[key] = SourceCLI
}

// SetCLIFlagsFromMap sets multiple CLI flags at once
func (lc *LayeredConfig) SetCLIFlagsFromMap(flags map[string]interface{}) {
	for key, value := range flags {
		lc.SetCLIFlag(key, value)
	}
}

// Get retrieves a configuration value by key
func (lc *LayeredConfig) Get(key string) interface{} {
	return lc.v.Get(key)
}

// GetString retrieves a string value by key
func (lc *LayeredConfig) GetString(key string) string {
	return lc.v.GetString(key)
}

// GetInt retrieves an integer value by key
func (lc *LayeredConfig) GetInt(key string) int {
	return lc.v.GetInt(key)
}

// GetBool retrieves a boolean value by key
func (lc *LayeredConfig) GetBool(key string) bool {
	return lc.v.GetBool(key)
}

// GetFloat64 retrieves a float64 value by key
func (lc *LayeredConfig) GetFloat64(key string) float64 {
	return lc.v.GetFloat64(key)
}

// GetStringSlice retrieves a string slice value by key
func (lc *LayeredConfig) GetStringSlice(key string) []string {
	return lc.v.GetStringSlice(key)
}

// IsSet checks if a key has been explicitly set (not just a default)
func (lc *LayeredConfig) IsSet(key string) bool {
	return lc.v.IsSet(key)
}

// GetSource returns the source of a configuration value
func (lc *LayeredConfig) GetSource(key string) Source {
	if source, exists := lc.sources[key]; exists {
		return source
	}
	return SourceDefault
}

// AllSettings returns all configuration settings as a map
func (lc *LayeredConfig) AllSettings() map[string]interface{} {
	return lc.v.AllSettings()
}

// AllKeys returns all configuration keys
func (lc *LayeredConfig) AllKeys() []string {
	return lc.v.AllKeys()
}

// Sub returns a sub-configuration by key
func (lc *LayeredConfig) Sub(key string) *viper.Viper {
	return lc.v.Sub(key)
}

// BindEnv binds an environment variable to a configuration key
func (lc *LayeredConfig) BindEnv(input ...string) error {
	return lc.v.BindEnv(input...)
}

// GetWorkspaceHash returns the workspace hash used for this config
func (lc *LayeredConfig) GetWorkspaceHash() string {
	return lc.workspaceHash
}

// GetSourceInfo returns detailed information about all configuration sources
func (lc *LayeredConfig) GetSourceInfo() map[string]SourceInfo {
	info := make(map[string]SourceInfo)
	for _, key := range lc.v.AllKeys() {
		info[key] = SourceInfo{
			Value:  lc.v.Get(key),
			Source: lc.GetSource(key),
		}
	}
	return info
}

// SourceInfo provides metadata about a configuration value
type SourceInfo struct {
	Value  interface{}
	Source Source
}

// MergeConfig merges configuration from another LayeredConfig
// Values from the other config take precedence
func (lc *LayeredConfig) MergeConfig(other *LayeredConfig) error {
	if other == nil {
		return nil
	}
	for _, key := range other.AllKeys() {
		lc.v.Set(key, other.Get(key))
		lc.sources[key] = other.GetSource(key)
	}
	return nil
}

// Unset removes a configuration value
func (lc *LayeredConfig) Unset(key string) {
	// Create a new viper instance without the key
	newV := viper.New()
	newV.SetEnvPrefix(lc.v.GetString("envPrefix"))
	newV.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	newV.AutomaticEnv()

	// Copy all settings except the one being unset
	for k, v := range lc.v.AllSettings() {
		if k != key && !strings.HasPrefix(k, key+".") {
			newV.Set(k, v)
		}
	}

	lc.v = newV
	delete(lc.sources, key)
	delete(lc.cliFlags, key)
}
