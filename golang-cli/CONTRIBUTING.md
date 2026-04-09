# Contributing to Cline CLI (Go)

Thank you for your interest in contributing to the Cline CLI! This document provides guidelines and instructions for contributing to the Go implementation.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Project Structure](#project-structure)
- [Coding Standards](#coding-standards)
- [Testing](#testing)
- [Documentation](#documentation)
- [Submitting Changes](#submitting-changes)
- [Release Process](#release-process)

## Code of Conduct

This project adheres to the [Contributor Covenant Code of Conduct](../CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## Getting Started

### Prerequisites

- **Go 1.25+**: [Download](https://go.dev/dl/)
- **Git**: For version control
- **Make**: For build automation
- **Protocol Buffers** (optional): For regenerating protobuf files

### Fork and Clone

```bash
# Fork the repository on GitHub, then:
git clone https://github.com/YOUR_USERNAME/cline.git
cd cline/golang-cli
```

### Install Dependencies

```bash
# Download Go modules
go mod download

# Verify installation
go version
make --version
```

## Development Setup

### Build the Project

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Run in development mode
make dev
```

### Run Tests

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run integration tests
make test-integration

# Run specific test
go test -v ./internal/config -run TestLoadConfig
```

### Code Quality

```bash
# Run linter
make lint

# Format code
make fmt

# Run all checks
make check
```

## Project Structure

```
golang-cli/
├── cmd/cline/          # Main application entry point
├── internal/           # Internal packages
│   ├── auth/          # Authentication
│   ├── config/        # Configuration management
│   ├── exit/          # Exit codes
│   ├── history/       # Task history
│   ├── mcp/           # MCP server management
│   ├── task/          # Task execution
│   └── tui/           # Terminal UI
├── proto/             # Protocol buffer definitions
├── scripts/           # Build and release scripts
├── pkg/               # Public API (minimal)
└── docs/              # Documentation
```

## Coding Standards

### Go Code Style

We follow standard Go conventions:

- **Formatting**: Use `gofmt` or `go fmt`
- **Linting**: We use `golangci-lint`
- **Naming**: Follow [Effective Go](https://go.dev/doc/effective_go) guidelines
- **Comments**: All exported items must have GoDoc comments

### Example Code Style

```go
// Package auth provides authentication mechanisms for API providers.
package auth

import "fmt"

// Provider represents an authentication provider.
type Provider struct {
    Name        string
    ClientID    string
    AuthURL     string
    TokenURL    string
}

// Authenticate performs OAuth authentication with the provider.
// Returns an access token or an error if authentication fails.
func (p *Provider) Authenticate() (string, error) {
    // Implementation
    return "", nil
}
```

### Error Handling

- Use wrapped errors with context: `fmt.Errorf("context: %w", err)`
- Define sentinel errors for common cases
- Map errors to appropriate exit codes

### Logging

- Use structured logging
- Include relevant context (task ID, operation)
- Respect log level settings

## Testing

### Test Coverage

Aim for high test coverage, especially for:

- Command parsing
- Configuration handling
- Error paths
- Public APIs

### Writing Tests

```go
func TestLoadConfig(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected *Config
        wantErr  bool
    }{
        {
            name:     "valid config",
            input:    `{"apiProvider": "anthropic"}`,
            expected: &Config{APIProvider: "anthropic"},
            wantErr:  false,
        },
        {
            name:     "invalid json",
            input:    `{invalid}`,
            expected: nil,
            wantErr:  true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := LoadConfig(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !reflect.DeepEqual(got, tt.expected) {
                t.Errorf("LoadConfig() = %v, want %v", got, tt.expected)
            }
        })
    }
}
```

### Integration Tests

Integration tests require a running VS Code extension. Use mocks when possible:

```go
func TestTaskExecution(t *testing.T) {
    // Create mock gRPC connection
    mockConn := newMockGRPCConnection()
    
    // Execute task
    err := ExecuteTask(mockConn, "test prompt")
    
    // Assert results
    if err != nil {
        t.Errorf("ExecuteTask() failed: %v", err)
    }
}
```

### Benchmarks

Add benchmarks for performance-critical code:

```go
func BenchmarkParseCommand(b *testing.B) {
    for i := 0; i < b.N; i++ {
        ParseCommand("task --model claude-3 'hello world'")
    }
}
```

Run benchmarks:
```bash
go test -bench=. ./internal/task
```

## Documentation

### GoDoc Comments

All exported items must have GoDoc comments:

```go
// Task represents a single AI-assisted coding task.
// It contains the prompt, context, and execution state.
type Task struct {
    ID      string
    Prompt  string
    Context TaskContext
    State   TaskState
}

// Execute runs the task and returns the result.
// The context controls cancellation and timeouts.
func (t *Task) Execute(ctx context.Context) (*Result, error) {
    // ...
}
```

### Markdown Documentation

- Use clear, concise language
- Include code examples
- Update table of contents
- Check for broken links

### Architecture Decision Records (ADRs)

For significant architectural decisions, create an ADR in `docs/adr/`:

```markdown
# ADR-001: Use Bubble Tea for TUI

## Status
Accepted

## Context
We need a TUI framework that is...

## Decision
Use [Bubble Tea](https://github.com/charmbracelet/bubbletea)...

## Consequences
- Positive: Clean architecture, easy to test
- Negative: Learning curve for contributors
```

## Submitting Changes

### Branch Naming

- `feature/description` - New features
- `bugfix/description` - Bug fixes
- `docs/description` - Documentation updates
- `refactor/description` - Code refactoring

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation only
- `style`: Code style (formatting, no logic changes)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Build process, dependencies, etc.

Examples:
```
feat(auth): add OAuth support for OpenAI

Implement OAuth flow for OpenAI provider with PKCE.
Includes token refresh and secure storage.

Closes #123
```

```
fix(tui): handle terminal resize correctly

Previously, resizing the terminal would cause UI glitches.
Now we properly handle SIGWINCH and redraw the interface.

Fixes #456
```

### Pull Request Process

1. **Create a branch**: `git checkout -b feature/my-feature`
2. **Make changes**: Follow coding standards
3. **Add tests**: Ensure coverage
4. **Update docs**: Add/update documentation
5. **Run checks**: `make check`
6. **Commit**: Follow commit message guidelines
7. **Push**: `git push origin feature/my-feature`
8. **Create PR**: Include description and link issues

### PR Checklist

- [ ] Tests added/updated
- [ ] Documentation updated
- [ ] Code follows style guidelines
- [ ] Commit messages follow conventions
- [ ] No breaking changes (or clearly documented)
- [ ] CHANGELOG.md updated (if applicable)

### Code Review

All PRs require review before merging:

- At least one approval from a maintainer
- All CI checks must pass
- No unresolved conversations

## Release Process

### Version Numbering

We follow [Semantic Versioning](https://semver.org/):

- **MAJOR**: Incompatible API changes
- **MINOR**: New functionality (backward compatible)
- **PATCH**: Bug fixes (backward compatible)

### Creating a Release

1. Update version in relevant files
2. Update CHANGELOG.md
3. Create a PR with these changes
4. After merge, tag the release:
   ```bash
   git tag -a v1.0.0 -m "Release version 1.0.0"
   git push origin v1.0.0
   ```
5. GitHub Actions will build and publish

### Hotfixes

For critical fixes:

1. Create branch from latest tag: `git checkout -b hotfix/description v1.0.0`
2. Make fix with minimal changes
3. Tag new version: `v1.0.1`
4. Merge back to main

## Getting Help

### Resources

- [Development Guide](DEVELOPMENT.md)
- [Architecture Documentation](docs/ARCHITECTURE.md)
- [Troubleshooting Guide](docs/TROUBLESHOOTING.md)

### Communication

- **Issues**: https://github.com/cline/cline/issues
- **Discussions**: https://github.com/cline/cline/discussions
- **Discord**: https://discord.gg/cline

### Questions?

- Check existing issues and discussions
- Ask in Discord #development channel
- Open a discussion for design questions

## Recognition

Contributors will be:

- Listed in CONTRIBUTORS.md
- Mentioned in release notes
- Added to the project's authors list

Thank you for contributing to Cline CLI!