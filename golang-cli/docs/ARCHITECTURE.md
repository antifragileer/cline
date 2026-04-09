# Cline CLI Architecture

This document describes the architecture of the Cline CLI Go implementation.

## Overview

The Cline CLI is a command-line interface for the Cline AI coding assistant. It provides a terminal-based interface for interacting with AI models, managing tasks, and automating coding workflows.

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Cline CLI                                       │
├─────────────────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │
│  │   Commands  │  │  Settings   │  │  Auth       │  │  Task Management    │ │
│  │   (cmd/)    │  │  (config/)  │  │  (auth/)    │  │  (task/, history/)  │ │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └──────────┬──────────┘ │
│         │                │                │                    │            │
│         └────────────────┴────────────────┴────────────────────┘            │
│                                    │                                        │
│                         ┌──────────┴──────────┐                             │
│                         │   Core Controller   │                             │
│                         │    (internal/)      │                             │
│                         └──────────┬──────────┘                             │
│                                    │                                        │
│         ┌──────────────────────────┼──────────────────────────┐             │
│         │                          │                          │             │
│  ┌──────┴──────┐          ┌────────┴────────┐        ┌───────┴──────┐      │
│  │    TUI      │          │  gRPC Client    │        │   Storage    │      │
│  │ (internal/  │          │  (proto/,       │        │  (storage/)  │      │
│  │  tui/)      │          │   transport/)   │        │              │      │
│  └─────────────┘          └─────────────────┘        └──────────────┘      │
│                                    │                                        │
│                         ┌──────────┴──────────┐                             │
│                         │   VS Code Core      │                             │
│                         │   Extension         │                             │
│                         │   (via gRPC)        │                             │
│                         └─────────────────────┘                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Package Structure

### `/cmd/cline`
The main entry point for the CLI application.
- **Purpose**: Application bootstrap and command execution
- **Key Files**:
  - `main.go` - Entry point, handles exit codes
  - `root.go` - Root command definition
  - `version.go` - Version command

### `/internal`

#### `/internal/auth`
Authentication management for API providers.
- **Purpose**: OAuth flows, API key management, credential storage
- **Key Components**:
  - `oauth.go` - OAuth implementation
  - `credentials.go` - Secure credential storage
  - `providers.go` - Provider-specific auth logic

#### `/internal/config`
Configuration management.
- **Purpose**: Settings storage, validation, and retrieval
- **Key Components**:
  - `config.go` - Configuration structures
  - `settings.go` - Settings management
  - `validation.go` - Config validation

#### `/internal/exit`
Exit code management.
- **Purpose**: Standardized exit codes across the CLI
- **Key Components**:
  - `codes.go` - Exit code definitions
  - `mapping.go` - Error to exit code mapping

#### `/internal/history`
Task history management.
- **Purpose**: Store and retrieve task history
- **Key Components**:
  - `store.go` - History storage backend
  - `query.go` - History querying
  - `format.go` - Output formatting

#### `/internal/mcp`
MCP (Model Context Protocol) server management.
- **Purpose**: Manage MCP server connections and operations
- **Key Components**:
  - `server.go` - Server management
  - `client.go` - MCP client implementation

#### `/internal/task`
Task execution and management.
- **Purpose**: Execute AI tasks, handle streaming responses
- **Key Components**:
  - `executor.go` - Task execution logic
  - `stream.go` - Response streaming
  - `state.go` - Task state management

#### `/internal/tui`
Terminal User Interface components.
- **Purpose**: Rich interactive terminal UI
- **Key Components**:
  - `app.go` - Main TUI application
  - `components/` - Reusable UI components
  - `styles.go` - UI styling and themes
  - `keymap.go` - Keyboard shortcuts

### `/proto`
Protocol Buffer definitions.
- **Purpose**: gRPC communication with VS Code extension
- **Key Files**:
  - `.proto` files - Service definitions
  - Generated Go code for gRPC clients

### `/scripts`
Build and distribution utilities.
- **Purpose**: Release automation, package generation
- **Key Files**:
  - `build.sh` - Cross-platform build script
  - `release.sh` - Release automation
  - `generate_*.go` - Package manifest generators

### `/pkg`
Public API packages (for library use).
- **Purpose**: Expose stable APIs for external use
- **Note**: Currently minimal as CLI is primarily a binary

## Data Flow

### Starting a Task

1. **Command Parsing**: Cobra parses command-line arguments
2. **Configuration Loading**: Settings loaded from storage
3. **Authentication Check**: Verify API credentials
4. **gRPC Connection**: Connect to VS Code extension
5. **Task Creation**: Create task via gRPC
6. **TUI Rendering**: Display interactive interface
7. **Streaming**: Stream responses from AI
8. **Tool Execution**: Handle tool approval/execution
9. **State Persistence**: Save task state

### Configuration Flow

```
User Input → Validation → Storage → gRPC Sync
     ↑                                  ↓
   Display ←─── UI Update ←── State Change
```

## Key Design Decisions

### 1. gRPC Communication
The CLI communicates with the VS Code extension via gRPC rather than direct API calls. This allows:
- Shared state with the extension
- Access to extension capabilities (file watching, etc.)
- Consistent behavior between CLI and GUI

### 2. Bubble Tea TUI Framework
We use [Bubble Tea](https://github.com/charmbracelet/bubbletea) for the TUI because:
- Elm architecture provides clean state management
- Excellent performance
- Rich component ecosystem (bubbles)
- Easy to test

### 3. Single Binary Distribution
The CLI is distributed as a single static binary:
- No runtime dependencies
- Easy installation
- Consistent behavior across platforms

### 4. Platform Abstraction
Platform-specific code is isolated:
- Build tags for OS-specific implementations
- Platform detection at runtime
- Abstracted file system operations

## Security Architecture

### Credential Storage
- API keys stored in OS-specific secure storage
- Keyring/keychain integration where available
- Fallback to encrypted file storage

### gRPC Security
- Local-only connections (localhost)
- No external network exposure
- TLS optional for remote connections

### Tool Approval
- All destructive operations require approval by default
- Configurable auto-approval policies
- Audit logging of all actions

## Error Handling Strategy

### Exit Codes
Standard Unix exit codes are used:
- `0` - Success
- `1` - General error
- `2` - Misuse of command
- `126` - Command not executable
- `127` - Command not found
- `130` - Interrupted (Ctrl+C)

### Error Propagation
- Errors bubble up to main.go
- Context is preserved for debugging
- User-friendly messages at UI layer

## Testing Architecture

### Unit Tests
- Test individual packages in isolation
- Mock gRPC connections
- Table-driven tests for command parsing

### Integration Tests
- Test full command execution
- Mock VS Code extension
- Temporary filesystem for state

### E2E Tests
- Test against real extension
- Full workflow validation
- Performance benchmarks

## Performance Considerations

### Startup Time
- Minimal initialization before first output
- Lazy loading of heavy components
- Connection pooling for gRPC

### Memory Usage
- Streaming responses to minimize memory
- Cleanup of completed tasks
- Efficient data structures

### Binary Size
- Stripped release builds (`-s -w` ldflags)
- No CGO for simpler cross-compilation
- UPX compression optional

## Extension Points

### Adding New Commands
1. Create command file in `cmd/`
2. Add to root command in `root.go`
3. Add tests
4. Update documentation

### Adding New API Providers
1. Add provider config to `internal/auth/providers.go`
2. Implement OAuth flow if needed
3. Add UI components for auth
4. Update validation

### Custom TUI Components
1. Create component in `internal/tui/components/`
2. Implement Bubble Tea Model interface
3. Add to main app
4. Define styles in `styles.go`

## Build System

### Cross-Compilation
Supported via `scripts/build.sh`:
- macOS (Intel & Apple Silicon)
- Linux (x64 & ARM64)
- Windows (x64)

### Release Pipeline
Automated via `scripts/release.sh`:
1. Run tests
2. Build all platforms
3. Sign macOS binaries
4. Create distribution packages
5. Generate release notes
6. Create GitHub release

## Monitoring and Observability

### Logging
- Structured logging with levels
- Debug mode for troubleshooting
- Log rotation

### Metrics
- Task execution time
- API call latency
- Error rates
- Memory usage

### Health Checks
- gRPC connection status
- Extension availability
- Configuration validity

## Future Considerations

### Planned Improvements
- Standalone mode (no VS Code required)
- Plugin system for custom tools
- Remote workspace support
- Collaboration features

### Scalability
- Horizontal scaling for task processing
- Distributed state management
- Caching layer for common operations

## Contributing

See [CONTRIBUTING.md](../CONTRIBUTING.md) for guidelines on contributing to the architecture.

## References

- [Bubble Tea Documentation](https://github.com/charmbracelet/bubbletea)
- [Cobra Documentation](https://github.com/spf13/cobra)
- [gRPC Go Documentation](https://grpc.io/docs/languages/go/)