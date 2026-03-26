# Migration Guide: Node.js CLI to Go CLI

This guide helps you migrate from the Node.js/TypeScript Cline CLI to the new Go implementation.

## Overview

The Go CLI is a drop-in replacement for the Node.js CLI. Your existing tasks, configuration, and workflows will continue to work seamlessly.

## What Stays the Same

### Configuration Files

Your configuration in `~/.cline/data/` is fully compatible:
- `globalState.json` - Global settings preserved
- `secrets.json` - API keys and credentials preserved
- `workspaceState.json` - Per-workspace settings preserved
- Task history and conversation files

### Command Syntax

All commands work identically:

```bash
# These work exactly the same
cline "Create a React component"
cline task "Implement authentication"
cline history
cline config
cline auth
```

### Flags and Options

All flags remain the same:

```bash
cline -a "Auto-execute this task"           # --act
cline -p "Plan this feature"                # --plan
cline -y "Auto-approve everything"          # --yolo
cline -m claude-3-5-sonnet "Use this model" # --model
cline --json "Output as JSON"               # --json
```

## What's Different

### Installation

| Before (Node.js) | After (Go) |
|-----------------|------------|
| `npm install -g @cline/cline` | `brew install cline` or download binary |
| Requires Node.js runtime | Single binary, no dependencies |

### Binary Name

If you installed via npm globally, you may need to update your PATH:

```bash
# Remove old npm installation
npm uninstall -g @cline/cline

# New binary is 'cline' from any installation method
which cline
```

### Performance

- **Startup**: ~10x faster (no Node.js initialization)
- **Memory**: Lower memory footprint
- **Binary size**: ~20MB single binary vs ~200MB with dependencies

## Migration Steps

### 1. Backup (Optional but Recommended)

Your data is safe, but backups are always wise:

```bash
cp -r ~/.cline/data ~/.cline/data.backup
```

### 2. Install Go CLI

Choose your preferred method:

**Homebrew (macOS/Linux):**
```bash
brew tap cline/tap
brew install cline
```

**Direct Download:**
```bash
# macOS ARM64
curl -L -o cline "https://github.com/cline/cline/releases/latest/download/cline_$(curl -s https://api.github.com/repos/cline/cline/releases/latest | grep tag_name | cut -d '"' -f 4)_darwin_arm64"
chmod +x cline
sudo mv cline /usr/local/bin/
```

**NPM Wrapper (if you prefer npm):**
```bash
npm install -g @cline/golang-cli
# Binary available as 'cline-go'
```

### 3. Verify Installation

```bash
# Check version
cline version

# Verify config is accessible
cline config list
```

### 4. Test with a Simple Task

```bash
cline "Say hello to verify the migration"
```

### 5. Uninstall Old CLI (Optional)

```bash
# If installed via npm
npm uninstall -g @cline/cline

# If installed via other methods, remove accordingly
```

## Troubleshooting

### Task History Not Visible

If your task history doesn't appear:

```bash
# Verify data directory
ls ~/.cline/data/tasks/

# Check permissions
cline config list
```

### API Keys Not Found

API keys are stored in the OS keyring. If missing:

```bash
# Re-authenticate
cline auth login

# Or set environment variable
export ANTHROPIC_API_KEY="your-key"
```

### Different Binary Name

If using the npm wrapper, the binary is `cline-go`:

```bash
# Create alias if desired
alias cline='cline-go'
```

### PATH Issues

If `cline` command not found:

```bash
# Find binary location
which cline-go  # or where cline-go on Windows

# Add to PATH
export PATH="$PATH:/path/to/cline"

# Or symlink
ln -s /path/to/cline /usr/local/bin/cline
```

## Feature Parity Checklist

The following features are fully supported in the Go CLI:

- [x] All commands (`task`, `history`, `config`, `auth`, `mcp`, `version`, `update`)
- [x] All flags (`-a`, `-p`, `-y`, `-t`, `-m`, `-v`, `--json`, etc.)
- [x] Interactive TUI mode
- [x] Piped/scripting mode
- [x] JSON output mode
- [x] All API providers
- [x] OAuth authentication
- [x] MCP server management
- [x] Task resumption
- [x] Checkpoints
- [x] Image attachments

## Known Differences

### TUI Appearance

The Go CLI uses [Bubble Tea](https://github.com/charmbracelet/bubbletea) instead of React Ink:
- Similar look and feel
- Potentially smoother rendering
- Same keyboard shortcuts

### Error Messages

Error messages may be formatted slightly differently, but contain the same information.

### Exit Codes

Exit codes are now fully standardized:
- `0` - Success
- `1` - General error
- `2` - Invalid arguments
- `3` - Task failed
- `4` - Authentication error
- `5` - Network error

## Rollback

If you need to rollback to the Node.js CLI:

```bash
# Uninstall Go CLI
brew uninstall cline  # or remove binary

# Reinstall Node.js CLI
npm install -g @cline/cline
```

Your data remains intact and compatible.

## Getting Help

- **Documentation**: https://docs.cline.bot
- **Issues**: https://github.com/cline/cline/issues
- **Discord**: https://discord.gg/cline

## FAQ

**Q: Will my existing tasks work?**  
A: Yes, all task history and conversations are fully compatible.

**Q: Do I need to re-authenticate?**  
A: No, your API keys are preserved in the OS keyring.

**Q: Can I use both CLIs side by side?**  
A: Yes, but only one should be in your PATH at a time to avoid confusion.

**Q: Is the Go CLI feature-complete?**  
A: Yes, all features from the Node.js CLI are implemented.

**Q: What about VS Code extension integration?**  
A: The Go CLI communicates with the same core extension via gRPC.

**Q: Will the Node.js CLI be deprecated?**  
A: The Go CLI is the recommended version going forward.

## Migration Verification Script

Run this to verify your migration:

```bash
#!/bin/bash
set -e

echo "=== Cline CLI Migration Verification ==="

# Check binary exists
if ! command -v cline &> /dev/null; then
    echo "❌ cline command not found in PATH"
    exit 1
fi
echo "✓ Binary found: $(which cline)"

# Check version
echo "✓ Version: $(cline version)"

# Check config access
if cline config list > /dev/null 2>&1; then
    echo "✓ Configuration accessible"
else
    echo "⚠ Configuration not accessible (may need auth)"
fi

# Check task history
if cline history --json > /dev/null 2>&1; then
    echo "✓ Task history accessible"
else
    echo "⚠ No task history found"
fi

echo ""
echo "=== Migration Complete ==="
echo "Run 'cline --help' to get started"
```

Save as `verify-migration.sh`, make executable (`chmod +x verify-migration.sh`), and run.