# Go CLI Development Guide

This guide provides detailed information for developers working on the Cline Go CLI.

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Getting Started](#getting-started)
- [Project Structure](#project-structure)
- [Development Workflow](#development-workflow)
- [Testing](#testing)
- [Building and Distribution](#building-and-distribution)
- [Adding New Features](#adding-new-features)
- [Debugging](#debugging)
- [Contributing](#contributing)

## Architecture Overview

The Go CLI follows a layered architecture:

```
┌─────────────────────────────────────────┐
│           CLI Commands (cmd/cline)      │
├─────────────────────────────────────────┤
│  TUI (Bubble Tea)  │  Task Execution    │
│  internal/tui      │  internal/task     │
├─────────────────────────────────────────┤
│       gRPC Client (internal/host)       │
├─────────────────────────────────────────┤
│    Core Services (internal/services)    │
├─────────────────────────────────────────┤
│  Storage (internal/storage)  │  Auth    │
└─────────────────────────────────────────┘
```

### Key Components

1. **CLI Commands** (`cmd/cline`): Cobra-based command definitions
2. **TUI** (`internal/tui`): Bubble Tea-based interactive interface
3. **Task Runner** (`internal/task`): Task execution and message handling
4. **gRPC Client** (`internal/host`): Communication with core extension
5. **Services** (`internal/services`): Checkpoint, browser, and other services
6. **Storage** (`internal/storage`): File-based state and secrets management

## Getting Started

### Prerequisites

- **Go 1.25+**: [Download](https://go.dev/dl/)
- **Node.js 18+**: For running the core extension during development
- **Protocol Buffers compiler**: For regenerating proto files
- **Make**: For using Makefile targets

### Installation

```bash
# Clone the repository
git clone https://github.com/cline/cline.git
cd cline/golang-cli

# Install dependencies
go mod download

# Verify installation
go build -o cline ./cmd/cline
./cline version
```

### Development Setup

```bash
# Build the binary
make build

# Run tests
make test

# Run the CLI
./cline --help
```

## Project Structure

```
golang-cli/
├── cmd/cline/              # CLI entry point and commands
│   ├── root.go            # Root command and flag definitions
│   ├── task.go            # Task command
│   ├── history.go         # History command
│   ├── config.go          # Config command
│   ├── auth.go            # Auth command
│   ├── mcp.go             # MCP command
│   ├── version.go         # Version command
│   └── update.go          # Update command
├── internal/
│   ├── agent/             # AI agent logic
│   ├── api/               # API provider implementations
│   ├── auth/              # Authentication handling
│   ├── config/            # Configuration management
│   ├── host/              # gRPC host communication
│   │   ├── client.go      # gRPC client
│   │   └── stream.go      # Streaming support
│   ├── mode/              # Mode detection (TTY, piped)
│   ├── services/          # Core services
│   │   ├── checkpoint.go  # Checkpoint management
│   │   └── browser.go     # Browser automation
│   ├── state/             # State management
│   ├── storage/           # Storage layer
│   │   ├── file.go        # File storage
│   │   └── keyring.go     # Secrets storage
│   ├── task/              # Task execution
│   │   ├── runner.go      # Task runner
│   │   ├── handler.go     # Message handlers
│   │   └── json_handler.go # JSON output handler
│   └── tui/               # Terminal UI
│       ├── app_model.go   # Main TUI application
│       ├── chat_model.go  # Chat interface
│       ├── welcome_model.go # Welcome screen
│       ├── message.go     # Message rendering
│       └── approval.go    # Tool approval UI
├── pkg/cline/             # Public API
├── proto/                 # Protocol buffer definitions
├── scripts/               # Build and distribution scripts
│   ├── build.go           # Build utilities
│   ├── build.sh           # Cross-platform build script
│   ├── homebrew.go        # Homebrew formula generation
│   ├── scoop.go           # Scoop manifest generation
│   ├── generate_homebrew.go # Homebrew CLI tool
│   └── generate_scoop.go  # Scoop CLI tool
├── tests/                 # Test suites
│   ├── unit/              # Unit tests
│   ├── integration/       # Integration tests
│   ├── e2e/               # End-to-end tests
│   └── parity/            # Parity tests vs TypeScript CLI
├── Makefile               # Build automation
├── go.mod                 # Go module definition
└── README.md              # User documentation
```

## Development Workflow

### Building

```bash
# Build for current platform
make build

# Build all platforms
make build-all

# Build specific platform
./scripts/build.sh -p linux/amd64

# Clean build artifacts
make clean
```

### Running Tests

```bash
# Run all tests
make test-all

# Run unit tests only
make test

# Run with coverage
make test-coverage

# Run specific test
go test ./internal/tui -v -run TestChatModel
```

### Code Generation

```bash
# Generate protobuf code (requires Node.js)
npm run protos

# Generate mocks (if using mockgen)
go generate ./...
```

### Linting

```bash
# Run linter
make lint

# Fix linting issues
golangci-lint run --fix
```

## Testing

### Test Structure

Tests are organized by type:

- **Unit tests**: `*_test.go` files alongside source code
- **Integration tests**: `tests/integration/` 
- **E2E tests**: `tests/e2e/`
- **Parity tests**: `tests/parity/` (compare with TypeScript CLI)

### Writing Tests

```go
// Example unit test
func TestSomething(t *testing.T) {
    // Arrange
    input := "test"
    
    // Act
    result := DoSomething(input)
    
    // Assert
    if result != expected {
        t.Errorf("DoSomething(%q) = %q, want %q", input, result, expected)
    }
}

// Example table-driven test
func TestSomethingTableDriven(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"empty", "", "default"},
        {"valid", "test", "test"},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := DoSomething(tt.input)
            if result != tt.expected {
                t.Errorf("got %q, want %q", result, tt.expected)
            }
        })
    }
}
```

### Integration Testing

Integration tests require a running core extension:

```bash
# Start core extension in one terminal
cd /path/to/cline
npm run dev

# Run integration tests in another terminal
cd golang-cli
make test-integration
```

### E2E Testing

E2E tests run the full CLI:

```bash
# Run E2E tests
make test-e2e

# Run specific E2E test
go test ./tests/e2e -v -run TestTaskExecution
```

## Building and Distribution

### Local Build

```bash
# Build for current platform
go build -o cline ./cmd/cline

# Build with version info
go build -ldflags "-X github.com/cline/cline/golang-cli/cmd/cline.Version=1.0.0" -o cline ./cmd/cline
```

### Cross-Compilation

```bash
# Build all platforms
./scripts/build.sh --all

# Build with archives and checksums
./scripts/build.sh --all --archive --checksum

# Build with signing (macOS only, requires certificates)
./scripts/build.sh --all --sign
```

### Package Managers

#### Homebrew

```bash
# Generate formula from release
go run ./scripts/generate_homebrew.go -version 1.0.0 -output Formula/cline.rb

# Generate from local binaries
go run ./scripts/generate_homebrew.go -version 1.0.0 -local -binary-dir ./dist -output Formula/cline.rb
```

#### Scoop

```bash
# Generate manifest from release
go run ./scripts/generate_scoop.go -version 1.0.0 -output bucket/cline.json

# Generate from local binaries
go run ./scripts/generate_scoop.go -version 1.0.0 -local -binary-dir ./dist -output bucket/cline.json
```

### CI/CD

The GitHub Actions workflow (`.github/workflows/golang-cli-build.yml`) handles:

1. **Build**: Cross-compilation for all platforms
2. **Test**: Unit, integration, and lint tests
3. **Sign**: macOS binary signing and notarization
4. **Package**: Archive creation and checksum generation
5. **Release**: GitHub release creation with artifacts
6. **Distribute**: Homebrew tap and Scoop bucket updates

## Adding New Features

### Adding a New Command

1. Create command file in `cmd/cline/`:

```go
// cmd/cline/newcommand.go
package main

import (
    "github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
    Use:   "newcommand [args]",
    Short: "Brief description",
    Long:  `Long description of the command.`,
    RunE:  runNewCommand,
}

func runNewCommand(cmd *cobra.Command, args []string) error {
    // Implementation
    return nil
}

func init() {
    rootCmd.AddCommand(newCmd)
    newCmd.Flags().String("flag", "", "Flag description")
}
```

2. Add tests in `cmd/cline/newcommand_test.go`

### Adding a New TUI Component

1. Create model file in `internal/tui/`:

```go
// internal/tui/new_model.go
package tui

import tea "github.com/charmbracelet/bubbletea"

type NewModel struct {
    // Fields
}

func NewNewModel() *NewModel {
    return &NewModel{}
}

func (m *NewModel) Init() tea.Cmd {
    return nil
}

func (m *NewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        // Handle key presses
    }
    return m, nil
}

func (m *NewModel) View() string {
    return "View content"
}
```

2. Integrate into `app_model.go`

### Adding gRPC Message Handlers

1. Add handler in `internal/task/handler.go`:

```go
func (h *InteractiveHandler) handleNewMessageType(ctx context.Context, msg *NewMessageType) error {
    // Handle the message
    return nil
}
```

2. Register in the handler map

## Debugging

### Local Debugging

```bash
# Build with debug symbols
go build -gcflags="all=-N -l" -o cline-debug ./cmd/cline

# Run with delve
dlv exec ./cline-debug -- task "test prompt"
```

### Verbose Logging

```bash
# Enable verbose output
./cline -v "test prompt"

# Enable debug logging
CLINE_DEBUG=1 ./cline "test prompt"
```

### gRPC Debugging

```bash
# Enable gRPC tracing
GRPC_TRACE=all ./cline "test prompt"

# Use grpcurl for testing
grpcurl -plaintext localhost:50051 list
```

### Common Issues

**Import cycles**: Avoid importing `internal/tui` from `internal/task`. Use interfaces or move shared types to `pkg/`.

**gRPC connection failures**: Ensure the core extension is running and the port is correct.

**TUI rendering issues**: Check terminal compatibility with Bubble Tea.

## Contributing

### Before Submitting

1. **Run tests**: `make test-all`
2. **Check linting**: `make lint`
3. **Verify build**: `make build-all`
4. **Update documentation**: If adding features

### Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Add comments for exported functions
- Keep functions focused and small

### Commit Messages

Use conventional commits:

```
feat: add new command
fix: resolve TUI rendering issue
docs: update README
test: add unit tests
refactor: simplify handler logic
```

### Pull Request Checklist

- [ ] Tests pass
- [ ] Linting passes
- [ ] Documentation updated
- [ ] CHANGELOG.md updated (if applicable)
- [ ] Breaking changes documented

## Resources

- [Go Documentation](https://go.dev/doc/)
- [Bubble Tea Documentation](https://github.com/charmbracelet/bubbletea)
- [Cobra Documentation](https://github.com/spf13/cobra)
- [gRPC Go Documentation](https://grpc.io/docs/languages/go/)
- [Project README](README.md)

## Getting Help

- GitHub Issues: https://github.com/cline/cline/issues
- Discord: https://discord.gg/cline
- Documentation: https://docs.cline.bot