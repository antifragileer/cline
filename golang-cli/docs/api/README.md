# Cline CLI Documentation

**Version:** 1.0.0

**Generated:** 2026-04-08

## Overview

This documentation covers the Cline Go CLI internal packages and APIs. The Cline CLI is a powerful AI coding assistant that runs in your terminal, providing a rich interactive experience for AI-assisted development.

## Documentation Index

### User Documentation

| Document | Description |
|----------|-------------|
| [API.md](../API.md) | Public API reference for programmatic use |
| [ARCHITECTURE.md](../ARCHITECTURE.md) | System architecture and design decisions |
| [MIGRATION_GUIDE.md](../MIGRATION_GUIDE.md) | Guide for migrating from TypeScript CLI |
| [TROUBLESHOOTING.md](../TROUBLESHOOTING.md) | Common issues and solutions |
| [DISTRIBUTION.md](../DISTRIBUTION.md) | Build and distribution guide |

### Development Documentation

| Document | Description |
|----------|-------------|
| [CONTRIBUTING.md](../../CONTRIBUTING.md) | Contribution guidelines |
| [DEVELOPMENT.md](../../DEVELOPMENT.md) | Development setup and workflow |
| [README.md](../../README.md) | Project overview and quick start |

## Package Structure

### Command Layer (`cmd/cline/`)
Main application entry point and CLI command definitions.

### Internal Packages (`internal/`)

| Package | Purpose |
|---------|---------|
| `auth` | Authentication and credential management |
| `config` | Configuration storage and validation |
| `exit` | Exit code definitions and error mapping |
| `history` | Task history storage and retrieval |
| `mcp` | MCP server management |
| `task` | Task execution and streaming |
| `tui` | Terminal User Interface components |

### Build Scripts (`scripts/`)
Utilities for building, packaging, and distributing the CLI.

### Protocol Buffers (`proto/`)
gRPC service definitions for communication with VS Code extension.

## Quick Links

- [GitHub Repository](https://github.com/cline/cline)
- [Issue Tracker](https://github.com/cline/cline/issues)
- [Discussions](https://github.com/cline/cline/discussions)
- [Discord Community](https://discord.gg/cline)

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0.0 | 2026-04-08 | Initial stable release |
| 0.9.0 | 2026-03-15 | Beta release |

## License

Apache-2.0 - See [LICENSE](../../LICENSE) for details.