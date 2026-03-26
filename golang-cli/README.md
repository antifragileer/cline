# Cline CLI (Go Implementation)

The Go implementation of the Cline CLI - an autonomous coding agent that runs in your terminal.

## Overview

This is a complete reimplementation of the Cline CLI in Go, providing:
- **Single binary distribution** - No Node.js runtime required
- **Faster startup** - Native binary execution
- **Cross-platform** - macOS, Linux, Windows support
- **Full feature parity** - All features from the Node.js CLI

## Installation

### Homebrew (macOS/Linux)

```bash
brew tap cline/tap
brew install cline
```

### NPM (Cross-platform)

```bash
npm install -g @cline/golang-cli
```

### Scoop (Windows)

```powershell
scoop bucket add cline https://github.comine/cline
scoop install cline
```

### Direct Download

Download pre-built binaries from [GitHub Releases](https://github.com/cline/cline/releases).

### Build from Source

```bash
git clone https://github.com/cline/cline.git
cd cline/golang-cli
go build -o cline ./cmd/cline
```

## Quick Start

```bash
# Start a new task
cline "Create a React component"

# Plan mode
cline -p "Plan out this feature"

# Act mode with auto-approve
cline -a "Implement the feature"

# Resume a previous task
cline --continue

# View history
cline history

# Configure settings
cline config
```

## Commands

| Command | Description |
|---------|-------------|
| `cline [prompt]` | Start a new task with the given prompt |
| `cline task [prompt]` | Start a new task explicitly |
| `cline history` | View task history |
| `cline config` | Manage configuration |
| `cline auth` | Authenticate with providers |
| `cline mcp` | Manage MCP servers |
| `cline version` | Show version information |
| `cline update` | Check for updates |

## Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--act` | `-a` | Act mode (auto-execute) |
| `--plan` | `-p` | Plan mode (ask before acting) |
| `--yolo` | `-y` | YOLO mode (auto-approve all) |
| `--timeout` | `-t` | Set timeout (e.g., 30s, 5m) |
| `--model` | `-m` | Specify model to use |
| `--verbose` | `-v` | Verbose output |
| `--json` | | JSON output format |
| `--continue` | | Continue recent task |

## Configuration

Configuration is stored in `~/.cline/data/`:

- `globalState.json` - Global settings
- `secrets.json` - API keys (encrypted)
- `workspaceState.json` - Per-workspace settings

### Environment Variables

| Variable | Description |
|----------|-------------|
| `ANTHROPIC_API_KEY` | Anthropic API key |
| `OPENAI_API_KEY` | OpenAI API key |
| `OPENROUTER_API_KEY` | OpenRouter API key |
| `HTTPS_PROXY` | Proxy for API requests |

## Development

### Prerequisites

- Go 1.25+
- Node.js 18+ (for running the core extension)
- Protocol Buffers compiler (for proto generation)

### Building

```bash
# Build binary
make build

# Build all platforms
./scripts/build.sh --all

# Generate protobuf code
npm run protos
```

### Testing

```bash
# Run all tests
make test-all

# Run specific test suites
make test                # Unit tests
make test-integration    # Integration tests
make test-parity         # Parity tests vs Node.js CLI
make test-e2e           # End-to-end tests

# Run with coverage
make test-coverage
```

### Project Structure

```
golang-cli/
├── cmd/cline/          # CLI commands
├── internal/
│   ├── agent/          # AI agent logic
│   ├── api/            # API providers
│   ├── auth/           # Authentication
│   ├── config/         # Configuration management
│   ├── host/           # gRPC host communication
│   ├── mode/           # Mode detection (TTY, piped, etc.)
│   ├── services/       # Core services
│   ├── state/          # State management
│   ├── storage/        # File and secrets storage
│   ├── task/           # Task execution
│   └── tui/            # Terminal UI (Bubble Tea)
├── pkg/cline/          # Public API
├── proto/              # Protocol buffer definitions
├── scripts/            # Build and distribution scripts
└── tests/              # Test suites
```

## API Providers

Supported AI providers:

- Anthropic (Claude)
- OpenAI (GPT-4, GPT-5)
- OpenRouter
- Google Gemini
- AWS Bedrock
- Ollama (local)
- LM Studio (local)

## Migration from Node.js CLI

See [MIGRATION.md](MIGRATION.md) for detailed migration instructions.

Key differences:
- Single binary vs npm package
- Configuration location unchanged
- Full state compatibility

## Contributing

1. Fork the repository
2. Create a feature branch
3. Run tests: `make test-all`
4. Submit a pull request

See [CONTRIBUTING.md](../CONTRIBUTING.md) for details.

## License

Apache-2.0 - See [LICENSE](../LICENSE) for details.

## Support

- Documentation: https://docs.cline.bot
- Issues: https://github.com/cline/cline/issues
- Discord: https://discord.gg/cline