# Cline CLI API Documentation

This document describes the public APIs available in the Cline CLI Go implementation.

## Overview

The Cline CLI is primarily distributed as a command-line binary. However, it also exposes Go packages that can be imported for programmatic use or extension.

## Package Index

| Package | Path | Description |
|---------|------|-------------|
| `cmd/cline` | `github.com/cline/cline/golang-cli/cmd/cline` | CLI entry point |
| `internal/auth` | `github.com/cline/cline/golang-cli/internal/auth` | Authentication |
| `internal/config` | `github.com/cline/cline/golang-cli/internal/config` | Configuration |
| `internal/exit` | `github.com/cline/cline/golang-cli/internal/exit` | Exit codes |
| `internal/history` | `github.com/cline/cline/golang-cli/internal/history` | Task history |
| `internal/task` | `github.com/cline/cline/golang-cli/internal/task` | Task execution |
| `scripts` | `github.com/cline/cline/golang-cli/scripts` | Build utilities |

**Note**: Internal packages (`internal/`) can only be imported by code within the same module.

## Authentication API (`internal/auth`)

The authentication package provides secure credential management for API providers.

### Types

#### `Provider`

```go
// Provider represents an OAuth or API key authentication provider.
type Provider struct {
    ID          string
    Name        string
    Type        AuthType  // AuthTypeOAuth or AuthTypeAPIKey
    AuthURL     string    // For OAuth
    TokenURL    string    // For OAuth
    Scopes      []string  // For OAuth
}
```

#### `Credentials`

```go
// Credentials represents stored authentication credentials.
type Credentials struct {
    ProviderID   string
    AccessToken  string
    RefreshToken string
    ExpiresAt    time.Time
    APIKey       string  // For API key auth
}
```

### Functions

#### `Authenticate`

```go
// Authenticate performs interactive authentication with a provider.
// For OAuth providers, this opens a browser and handles the callback.
// For API key providers, this prompts for the key.
func Authenticate(provider *Provider) (*Credentials, error)
```

**Example:**
```go
provider := &auth.Provider{
    ID:      "anthropic",
    Name:    "Anthropic",
    Type:    auth.AuthTypeAPIKey,
}

creds, err := auth.Authenticate(provider)
if err != nil {
    log.Fatal(err)
}
fmt.Println("Authenticated successfully")
```

#### `StoreCredentials`

```go
// StoreCredentials securely stores credentials in the OS keyring.
// Falls back to encrypted file storage if keyring is unavailable.
func StoreCredentials(creds *Credentials) error
```

#### `LoadCredentials`

```go
// LoadCredentials retrieves credentials for a provider.
// Returns ErrNotFound if no credentials exist.
func LoadCredentials(providerID string) (*Credentials, error)
```

#### `DeleteCredentials`

```go
// DeleteCredentials removes stored credentials for a provider.
func DeleteCredentials(providerID string) error
```

### Errors

```go
var (
    ErrNotFound     = errors.New("credentials not found")
    ErrInvalidAuth  = errors.New("invalid authentication")
    ErrExpired      = errors.New("credentials expired")
)
```

## Configuration API (`internal/config`)

The configuration package manages CLI settings.

### Types

#### `Config`

```go
// Config represents the complete CLI configuration.
type Config struct {
    APIProvider           string            `json:"apiProvider"`
    Model                 string            `json:"model"`
    PlanActSeparateModels bool              `json:"planActSeparateModels"`
    AutoApprove           AutoApproveConfig `json:"autoApprove"`
    Theme                 string            `json:"theme"`
    // ... additional fields
}
```

#### `AutoApproveConfig`

```go
// AutoApproveConfig controls which operations can be auto-approved.
type AutoApproveConfig struct {
    ReadFiles       bool `json:"readFiles"`
    EditFiles       bool `json:"editFiles"`
    ExecuteCommands bool `json:"executeCommands"`
    UseBrowser      bool `json:"useBrowser"`
    UseMcp          bool `json:"useMcp"`
}
```

### Functions

#### `Load`

```go
// Load loads configuration from disk.
// Returns default config if no config exists.
func Load() (*Config, error)
```

**Example:**
```go
cfg, err := config.Load()
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Using provider: %s\n", cfg.APIProvider)
```

#### `Save`

```go
// Save persists configuration to disk.
func Save(cfg *Config) error
```

#### `Get`

```go
// Get retrieves a configuration value by key.
// Supports dot notation for nested values (e.g., "autoApprove.readFiles").
func Get(key string) (interface{}, error)
```

#### `Set`

```go
// Set updates a configuration value by key.
// Supports dot notation for nested values.
func Set(key string, value interface{}) error
```

#### `Reset`

```go
// Reset restores configuration to default values.
func Reset() error
```

### Default Values

```go
var DefaultConfig = &Config{
    APIProvider: "anthropic",
    Model:       "claude-3-5-sonnet-20241022",
    AutoApprove: AutoApproveConfig{
        ReadFiles: false,
        EditFiles: false,
        // ...
    },
    Theme: "default",
}
```

## Exit Codes (`internal/exit`)

Standardized exit codes for the CLI.

### Constants

```go
const (
    Success          Code = 0
    GeneralError     Code = 1
    MisuseOfCommand  Code = 2
    CommandNotExecutable Code = 126
    CommandNotFound  Code = 127
    Interrupted      Code = 130
)
```

### Functions

#### `MapErrorToCode`

```go
// MapErrorToCode maps an error to an appropriate exit code.
func MapErrorToCode(err error) Code
```

**Example:**
```go
if err != nil {
    code := exit.MapErrorToCode(err)
    os.Exit(int(code))
}
```

## Task History API (`internal/history`)

Manage task history and retrieval.

### Types

#### `TaskRecord`

```go
// TaskRecord represents a saved task in history.
type TaskRecord struct {
    ID          string    `json:"id"`
    Prompt      string    `json:"prompt"`
    CreatedAt   time.Time `json:"createdAt"`
    CompletedAt time.Time `json:"completedAt,omitempty"`
    Status      Status    `json:"status"`
    MessageCount int      `json:"messageCount"`
}
```

#### `Status`

```go
// Status represents the execution status of a task.
type Status string

const (
    StatusPending   Status = "pending"
    StatusRunning   Status = "running"
    StatusCompleted Status = "completed"
    StatusFailed    Status = "failed"
    StatusCancelled Status = "cancelled"
)
```

### Functions

#### `List`

```go
// List retrieves task history with optional filtering.
func List(opts ListOptions) ([]TaskRecord, error)
```

**Example:**
```go
records, err := history.List(history.ListOptions{
    Limit:  10,
    Status: history.StatusCompleted,
})
if err != nil {
    log.Fatal(err)
}
for _, r := range records {
    fmt.Printf("%s: %s\n", r.ID, r.Prompt)
}
```

#### `Get`

```go
// Get retrieves a specific task by ID.
func Get(taskID string) (*TaskRecord, error)
```

#### `Delete`

```go
// Delete removes a task from history.
func Delete(taskID string) error
```

#### `Clear`

```go
// Clear removes all task history.
func Clear() error
```

### ListOptions

```go
type ListOptions struct {
    Limit   int      // Maximum number of results (0 = unlimited)
    Offset  int      // Number of results to skip
    Status  Status   // Filter by status (empty = all)
    Since   time.Time // Filter by creation date
    Until   time.Time
}
```

## Task Execution API (`internal/task`)

Execute AI tasks programmatically.

### Types

#### `Executor`

```go
// Executor handles task execution.
type Executor struct {
    // contains filtered or unexported fields
}
```

#### `Options`

```go
// Options configures task execution.
type Options struct {
    Prompt          string
    Mode            Mode           // ModePlan, ModeAct, ModeYolo
    Model           string
    Timeout         time.Duration
    WorkingDir      string
    Images          []string       // Image paths for vision
    ContinueTask    bool           // Continue previous task
    TaskID          string         // Resume specific task
}
```

#### `Mode`

```go
// Mode controls AI execution behavior.
type Mode string

const (
    ModePlan Mode = "plan"   // Plan only, don't execute
    ModeAct  Mode = "act"    // Execute with approval
    ModeYolo Mode = "yolo"   // Auto-approve all
)
```

### Functions

#### `NewExecutor`

```go
// NewExecutor creates a new task executor.
func NewExecutor(grpcConn *grpc.ClientConn) (*Executor, error)
```

#### `Execute`

```go
// Execute runs a task with the given options.
// Returns a stream of events that must be consumed.
func (e *Executor) Execute(ctx context.Context, opts Options) (<-chan Event, error)
```

**Example:**
```go
executor, err := task.NewExecutor(conn)
if err != nil {
    log.Fatal(err)
}

events, err := executor.Execute(ctx, task.Options{
    Prompt: "Create a React component",
    Mode:   task.ModeAct,
    Model:  "claude-3-5-sonnet-20241022",
})
if err != nil {
    log.Fatal(err)
}

for event := range events {
    switch event.Type {
    case task.EventTypeMessage:
        fmt.Print(event.Content)
    case task.EventTypeToolUse:
        fmt.Printf("Tool: %s\n", event.ToolName)
    case task.EventTypeComplete:
        fmt.Println("\nTask complete!")
    }
}
```

### Event Types

```go
type EventType string

const (
    EventTypeMessage   EventType = "message"
    EventTypeToolUse   EventType = "tool_use"
    EventTypeToolResult EventType = "tool_result"
    EventTypeError     EventType = "error"
    EventTypeComplete  EventType = "complete"
)

type Event struct {
    Type       EventType
    Content    string
    ToolName   string
    ToolInput  map[string]interface{}
    Error      error
}
```

## Build Scripts API (`scripts`)

Utilities for building and distributing the CLI.

### Homebrew Generation

#### `HomebrewFormula`

```go
// HomebrewFormula represents a Homebrew formula configuration.
type HomebrewFormula struct {
    Name         string
    Desc         string
    Homepage     string
    Version      string
    License      string
    Platforms    []HomebrewPlatform
    BinaryName   string
    Dependencies []string
}
```

#### `GenerateHomebrewFormulaFromLocalBinaries`

```go
// GenerateHomebrewFormulaFromLocalBinaries generates a Homebrew formula
// using locally built binaries.
func GenerateHomebrewFormulaFromLocalBinaries(version, binaryDir, outputPath string) error
```

**Example:**
```go
err := scripts.GenerateHomebrewFormulaFromLocalBinaries(
    "1.0.0",
    "./dist",
    "./Formula/cline.rb",
)
if err != nil {
    log.Fatal(err)
}
```

### Scoop Generation

#### `GenerateScoop`

```go
// GenerateScoop generates a Scoop manifest for Windows distribution.
func GenerateScoop(config *ScoopConfig) (*ScoopManifest, error)
```

### Linux Packages

#### `GenerateLinuxPackages`

```go
// GenerateLinuxPackages generates DEB and RPM packages.
func GenerateLinuxPackages(version, binaryDir, outputDir string, pkgTypes []string) error
```

## gRPC API

The CLI communicates with the VS Code extension via gRPC.

### Connection

```go
// Connect establishes a gRPC connection to the VS Code extension.
func Connect(ctx context.Context, port int) (*grpc.ClientConn, error)
```

### Services

See the `proto/` directory for complete service definitions. Key services:

- `TaskService` - Task management
- `AuthService` - Authentication operations
- `ConfigService` - Configuration management
- `McpService` - MCP server operations

## Error Handling

### Common Error Types

```go
var (
    ErrConnectionFailed = errors.New("failed to connect to extension")
    ErrTaskNotFound     = errors.New("task not found")
    ErrInvalidConfig    = errors.New("invalid configuration")
    ErrAuthRequired     = errors.New("authentication required")
)
```

### Error Wrapping

All errors should be wrapped with context:

```go
result, err := someOperation()
if err != nil {
    return fmt.Errorf("failed to perform operation: %w", err)
}
```

## Best Practices

### 1. Context Cancellation

Always respect context cancellation:

```go
select {
case <-ctx.Done():
    return ctx.Err()
case result := <-ch:
    // process result
}
```

### 2. Resource Cleanup

Use defer for cleanup:

```go
file, err := os.Open(path)
if err != nil {
    return err
}
defer file.Close()
```

### 3. Structured Logging

Use structured logging with context:

```go
log.Printf("[task:%s] Starting execution", taskID)
```

### 4. Configuration Validation

Validate configuration early:

```go
if err := cfg.Validate(); err != nil {
    return fmt.Errorf("invalid configuration: %w", err)
}
```

## Examples

### Complete Task Execution

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    
    "github.com/cline/cline/golang-cli/internal/config"
    "github.com/cline/cline/golang-cli/internal/task"
    "github.com/cline/cline/golang-cli/proto"
)

func main() {
    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        log.Fatal(err)
    }
    
    // Connect to extension
    conn, err := proto.Connect(context.Background(), 0) // Auto-detect port
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()
    
    // Create executor
    executor, err := task.NewExecutor(conn)
    if err != nil {
        log.Fatal(err)
    }
    
    // Execute task
    events, err := executor.Execute(context.Background(), task.Options{
        Prompt: "Create a hello world program",
        Mode:   task.ModeAct,
        Model:  cfg.Model,
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Process events
    for event := range events {
        switch event.Type {
        case task.EventTypeMessage:
            fmt.Print(event.Content)
        case task.EventTypeError:
            log.Printf("Error: %v", event.Error)
            os.Exit(1)
        }
    }
}
```

### Custom Authentication Provider

```go
package main

import (
    "github.com/cline/cline/golang-cli/internal/auth"
)

func main() {
    // Define custom provider
    provider := &auth.Provider{
        ID:       "custom",
        Name:     "Custom Provider",
        Type:     auth.AuthTypeAPIKey,
    }
    
    // Authenticate
    creds, err := auth.Authenticate(provider)
    if err != nil {
        log.Fatal(err)
    }
    
    // Store credentials
    if err := auth.StoreCredentials(creds); err != nil {
        log.Fatal(err)
    }
}
```

## Version Compatibility

| CLI Version | Go Version | VS Code Extension |
|-------------|------------|-------------------|
| 1.0.x       | 1.25+      | 3.0+              |
| 0.9.x       | 1.23+      | 2.5+              |

## Additional Resources

- [Architecture Documentation](ARCHITECTURE.md)
- [Contributing Guidelines](../CONTRIBUTING.md)
- [gRPC Protocol Definitions](../proto/)

## Changelog

See [CHANGELOG.md](../CHANGELOG.md) for API changes between versions.