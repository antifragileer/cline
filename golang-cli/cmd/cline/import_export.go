package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/cline/cline/golang-cli/internal/storage"
)

// importExportFlags holds flags for import/export commands
var importExportFlags struct {
	file string
	force bool
}

// ConfigExport represents the exported configuration structure
type ConfigExport struct {
	Version    string                 `json:"version"`
	ExportedAt string                 `json:"exported_at"`
	Global     map[string]interface{} `json:"global,omitempty"`
	Secrets    map[string]string      `json:"secrets,omitempty"`
}

// initImportExportCommands initializes import and export commands
func initImportExportCommands() {
	// Import command
	importCmd := &cobra.Command{
		Use:   "import <file>",
		Short: "Import Cline settings from a file",
		Long: `Import Cline configuration and settings from a previously exported file.

This command imports global settings and optionally secrets from a JSON file
created by the 'export' command. Use with caution as it will overwrite
existing settings.`,
		Example: `  # Import settings from a file
  cline import cline-config.json

  # Import and overwrite existing settings without prompting
  cline import cline-config.json --force`,
		Args: cobra.ExactArgs(1),
		RunE: runImport,
	}
	importCmd.Flags().BoolVarP(&importExportFlags.force, "force", "f", false, "Overwrite existing settings without confirmation")
	rootCmd.AddCommand(importCmd)

	// Export command
	exportCmd := &cobra.Command{
		Use:   "export [file]",
		Short: "Export Cline settings to a file",
		Long: `Export Cline configuration and settings to a JSON file.

This command exports global settings and optionally secrets to a file
that can be imported later or shared across different installations.`,
		Example: `  # Export settings to default location
  cline export

  # Export to a specific file
  cline export cline-config.json

  # Export with secrets included (warning: stores secrets in plaintext)
  cline export cline-config.json --with-secrets`,
		Args: cobra.MaximumNArgs(1),
		RunE: runExport,
	}
	exportCmd.Flags().BoolVar(&importExportFlags.force, "with-secrets", false, "Include secrets in export (warning: plaintext storage)")
	rootCmd.AddCommand(exportCmd)
}

// runImport executes the import command
func runImport(cmd *cobra.Command, args []string) error {
	importFile := args[0]

	// Check if file exists
	if _, err := os.Stat(importFile); os.IsNotExist(err) {
		return fmt.Errorf("import file not found: %s", importFile)
	}

	// Read import file
	data, err := os.ReadFile(importFile)
	if err != nil {
		return fmt.Errorf("failed to read import file: %w", err)
	}

	// Parse import data
	var export ConfigExport
	if err := json.Unmarshal(data, &export); err != nil {
		return fmt.Errorf("failed to parse import file: %w", err)
	}

	// Initialize storage
	ctx, err := storage.NewStorageContext("", getWorkspaceHash())
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer ctx.Close()

	// Confirm overwrite if not forced
	if !importExportFlags.force {
		globalKeys := len(export.Global)
		fmt.Printf("This will import %d global settings.\n", globalKeys)
		fmt.Print("Continue? [y/N]: ")
		
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Println("Import cancelled.")
			return nil
		}
	}

	// Import global settings
	imported := 0
	for key, value := range export.Global {
		if err := ctx.GlobalState.Set(key, value); err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "Warning: failed to import %s: %v\n", key, err)
			continue
		}
		imported++
	}

	// Import secrets if present
	if len(export.Secrets) > 0 {
		for key, value := range export.Secrets {
			if err := ctx.Secrets.Set(key, value); err != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "Warning: failed to import secret %s: %v\n", key, err)
				continue
			}
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Imported %d secrets\n", len(export.Secrets))
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ Successfully imported %d settings\n", imported)
	return nil
}

// runExport executes the export command
func runExport(cmd *cobra.Command, args []string) error {
	// Initialize storage
	ctx, err := storage.NewStorageContext("", getWorkspaceHash())
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer ctx.Close()

	// Build export data
	export := ConfigExport{
		Version:    Version,
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Global:     ctx.GlobalState.GetAll(),
	}

	// Include secrets if requested
	withSecrets, _ := cmd.Flags().GetBool("with-secrets")
	if withSecrets {
		export.Secrets = make(map[string]string)
		allSecrets := ctx.Secrets.GetAll()
		for key, value := range allSecrets {
			if strVal, ok := value.(string); ok {
				export.Secrets[key] = strVal
			}
		}
		fmt.Fprintln(cmd.OutOrStderr(), "Warning: Secrets are exported in plaintext. Handle with care!")
	}

	// Determine output file
	var exportFile string
	if len(args) > 0 {
		exportFile = args[0]
	} else {
		// Default filename with timestamp
		timestamp := time.Now().Format("2006-01-02-150405")
		exportFile = fmt.Sprintf("cline-config-%s.json", timestamp)
	}

	// Ensure directory exists
	dir := filepath.Dir(exportFile)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal export data: %w", err)
	}

	// Write to file
	if err := os.WriteFile(exportFile, jsonData, 0600); err != nil {
		return fmt.Errorf("failed to write export file: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ Settings exported to %s\n", exportFile)
	if withSecrets {
		fmt.Fprintln(cmd.OutOrStdout(), "  (Secrets included in export)")
	}
	
	return nil
}
