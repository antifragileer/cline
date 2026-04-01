# Migration Guide: TypeScript CLI to Go CLI

This guide helps you migrate from the TypeScript/Node.js CLI to the new Go implementation.

## Overview

The Go CLI is a drop-in replacement for the TypeScript CLI with:
- **10x faster startup** (~200ms vs ~2s)
- **3x lower memory usage** (~50MB vs ~150MB)
- **Single binary** distribution (no Node.js required)
- **Full feature parity** with the TypeScript CLI

## Table of Contents

- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Configuration Migration](#configuration-migration)
- [Breaking Changes](#breaking-changes)
- [Command Mapping](#command-mapping)
- [Environment Variables](#environment-variables)
- [Scripts and Automation](#scripts-and-automation)
- [Troubleshooting Migration](#troubleshooting-migration)
- [Rollback](#rollback)

---

## Prerequisites

Before migrating:

1. **Backup your data:**
   ```bash
   # Backup existing TypeScript CLI data
   tar czf ~/cline-backup-$(date +%Y%m%d).tar.gz ~/.cline
   
   # Note your current configuration
   cline config list > ~/cline-config-backup.txt
   ```

2. **Check compatibility:**
   - Go CLI requires VS Code extension v3.0+
   - Some older task history may not be fully compatible

3. **Verify Node.js CLI version:**
   ```bash
   # Check current version
   cline version
   
   # The Go CLI will use the same version scheme
   ```

---

## Installation

### Option 1: Install Alongside TypeScript CLI

You can install both CLIs simultaneously:

```bash
# Install Go CLI with different name
brew install cline/tap/cline-go

# Or download directly
curl -L -o cline-go "https://github.com/cline/cline/releases/latest/download/cline_$(uname -s)_$(uname -m)"
chmod +x cline-go
sudo mv cline-go /usr/local/bin/
```

**Test the Go CLI:**
```bash
cline-go version
cline-go --help
```

### Option 2: Replace TypeScript CLI

```bash
# Uninstall TypeScript CLI
npm uninstall -g @cline/cli

# Install Go CLI
brew install cline/tap/cline

# Or use npm wrapper
npm install -g @cline/golang-cli
```

### Option 3: Automated Migration Script

```bash
# Download and run migration script
curl -fsSL https://raw.githubusercontent.com/cline/cline/main/golang-cli/scripts/migrate.sh | bash
```

---

## Configuration Migration

### Automatic Migration

The Go CLI automatically migrates configuration on first run:

```bash
# Just run any command
cline version

# The CLI will:
# 1. Detect existing TypeScript CLI config
# 2. Migrate to new location (~/.cline/data/)
# 3. Preserve all settings
```

### Manual Migration

If automatic migration fails:

```bash
# 1. Stop any running cline processes
pkill -f cline

# 2. Create new data directory
mkdir -p ~/.cline/data

# 3. Migrate config files
mv ~/.cline/globalState.json ~/.cline/data/ 2>/dev/null
mv ~/.cline/workspaceState.json ~/.cline/data/ 2>/dev/null
mv ~/.cline/secrets.json ~/.cline/data/ 2>/dev/null

# 4. Migrate task history
mkdir -p ~/.cline/data/tasks
mv ~/.cline/tasks/* ~/.cline/data/tasks/ 2>/dev/null

# 5. Verify migration
cline config list
cline history
```

### Configuration Locations

| Data Type | TypeScript CLI | Go CLI |
|-----------|---------------|--------|
| Global config | `~/.cline/globalState.json` | `~/.cline/data/globalState.json` |
| Workspace config | `~/.cline/workspaceState.json` | `~/.cline/data/workspaceState.json` |
| Secrets | `~/.cline/secrets.json` | `~/.cline/data/secrets.json` |
| Task history | `~/.cline/tasks/` | `~/.cline/data/tasks/` |
| Logs | `~/.cline/logs/` | `~/.cline/data/logs/` |

### Environment Variable Changes

The Go CLI respects the same environment variables:

```bash
# API Keys (unchanged)
export ANTHROPIC_API_KEY="..."
export OPENAI_API_KEY="..."
export OPENROUTER_API_KEY="..."

# Data directory (unchanged)
export CLINE_DATA_DIR="/custom/path"

# New in Go CLI
export CLINE_CONFIG_DIR="/custom/path"  # Same as DATA_DIR
export CLINE_DEBUG=1                     # Enable debug logging
```

---

## Breaking Changes

### 1. State Storage Format

**Change:** YAML → JSON  
**Impact:** Configuration files now use JSON instead of YAML.

**Migration:** Automatic - no action needed.

### 2. Binary Name

**Change:** `cline` → `cline` (same name, different implementation)  
**Impact:** Scripts using `cline` will automatically use the new version.

**Migration:** Update scripts if you need specific behavior from old CLI.

### 3. Task History Format

**Change:** Slightly different internal format  
**Impact:** Old task history may not be fully browsable.

**Migration:** Tasks will appear in history but some metadata may be missing.

### 4. Log Format

**Change:** Different log format and location  
**Impact:** Log aggregation scripts may need updates.

**Migration:** Update log parsing scripts to handle new format.

### 5. Exit Codes

**Change:** More precise exit codes  
**Impact:** Scripts checking exit codes may see different values.

**Migration:** Review exit code handling in scripts.

| Exit Code | Meaning |
|-----------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Misuse of command |
| 126 | Command not executable |
| 127 | Command not found |
| 130 | Interrupted (Ctrl+C) |

---

## Command Mapping

All commands work the same way:

| Command | TypeScript CLI | Go CLI | Notes |
|---------|---------------|--------|-------|
| Version | `cline version` | `cline version` | Identical |
| Help | `cline --help` | `cline --help` | Identical |
| Task | `cline "prompt"` | `cline "prompt"` | Identical |
| Plan mode | `cline -p "prompt"` | `cline -p "prompt"` | Identical |
| Act mode | `cline -a "prompt"` | `cline -a "prompt"` | Identical |
| YOLO mode | `cline -y "prompt"` | `cline -y "prompt"` | Identical |
| Config | `cline config` | `cline config` | Identical |
| History | `cline history` | `cline history` | Identical |
| Auth | `cline auth` | `cline auth` | Identical |
| MCP | `cline mcp` | `cline mcp` | Identical |
| Continue | `cline --continue` | `cline --continue` | Identical |

### Flag Mapping

| Flag | TypeScript CLI | Go CLI |
|------|---------------|--------|
| `--act`, `-a` | ✅ | ✅ |
| `--plan`, `-p` | ✅ | ✅ |
| `--yolo`, `-y` | ✅ | ✅ |
| `--json` | ✅ | ✅ |
| `--verbose`, `-v` | ✅ | ✅ |
| `--timeout`, `-t` | ✅ | ✅ |
| `--model`, `-m` | ✅ | ✅ |
| `--image`, `-i` | ✅ | ✅ |
| `--cwd`, `-c` | ✅ | ✅ |
| `--config` | ✅ | ✅ |
| `--continue` | ✅ | ✅ |
| `--taskId`, `-T` | ✅ | ✅ |
| `--thinking` | ✅ | ✅ |
| `--reasoning-effort` | ✅ | ✅ |
| `--hooks-dir` | ✅ | ✅ |
| `--kanban` | ✅ | ✅ |
| `--acp` | ✅ | ✅ |

---

## Environment Variables

All environment variables work identically:

```bash
# API Keys
ANTHROPIC_API_KEY
OPENAI_API_KEY
OPENROUTER_API_KEY
AWS_ACCESS_KEY_ID
AWS_SECRET_ACCESS_KEY
GOOGLE_API_KEY

# Configuration
CLINE_DATA_DIR          # Custom data directory
CLINE_CONFIG_DIR        # Same as above (Go CLI alias)
HTTPS_PROXY             # Proxy for API requests
NO_COLOR                # Disable colored output
FORCE_COLOR             # Force colored output

# Debug
CLINE_DEBUG             # Enable debug logging
CLINE_LOG_LEVEL         # Log level (debug, info, warn, error)
```

---

## Scripts and Automation

### Shell Scripts

Update shebang lines if using full paths:

```bash
#!/usr/bin/env cline
# Change to:
#!/usr/bin/env -S cline -a
```

Or use standard shell with cline calls:

```bash
#!/bin/bash
# Before
cline -a "implement feature"

# After (same command)
cline -a "implement feature"
```

### CI/CD Pipelines

**GitHub Actions:**

```yaml
# Before
- name: Setup Cline
  run: npm install -g @cline/cli

# After
- name: Setup Cline
  run: |
    curl -L -o cline "https://github.com/cline/cline/releases/latest/download/cline_linux_amd64"
    chmod +x cline
    sudo mv cline /usr/local/bin/
```

**Docker:**

```dockerfile
# Before
FROM node:18
RUN npm install -g @cline/cli

# After
FROM alpine:latest
RUN wget -O /usr/local/bin/cline https://github.com/cline/cline/releases/latest/download/cline_linux_amd64 \
    && chmod +x /usr/local/bin/cline
```

### Makefiles

```makefile
# Before
CLINE := npx @cline/cli

# After
CLINE := cline

# Or for projects without global install
CLINE := ./bin/cline
```

---

## Troubleshooting Migration

### "Config not found" After Migration

**Cause:** Config file location changed.

**Fix:**
```bash
# Check if old config exists
ls -la ~/.cline/globalState.json

# If yes, migrate manually
mkdir -p ~/.cline/data
mv ~/.cline/globalState.json ~/.cline/data/
mv ~/.cline/secrets.json ~/.cline/data/
```

### "Task history empty" After Migration

**Cause:** Task history path changed.

**Fix:**
```bash
# Migrate task history
mkdir -p ~/.cline/data/tasks
mv ~/.cline/tasks/* ~/.cline/data/tasks/ 2>/dev/null

# Or just start fresh
# Old tasks will remain in old location
```

### "Authentication lost" After Migration

**Cause:** Secrets file location changed.

**Fix:**
```bash
# Re-authenticate
cline auth

# Or migrate secrets
mv ~/.cline/secrets.json ~/.cline/data/
```

### Performance Regression

**Cause:** Using debug build instead of release.

**Fix:**
```bash
# Check build type
cline version --json | grep buildType

# Should be "release"
# If "debug", reinstall:
brew reinstall cline/tap/cline
```

### Binary Compatibility Issues

**Cause:** Wrong architecture binary.

**Fix:**
```bash
# Check architecture
uname -m

# Download correct binary
# For Apple Silicon (M1/M2/M3):
curl -L -o cline "https://.../cline_darwin_arm64"

# For Intel Mac:
curl -L -o cline "https://.../cline_darwin_amd64"

# For Linux x86_64:
curl -L -o cline "https://.../cline_linux_amd64"
```

---

## Rollback

If you need to rollback to the TypeScript CLI:

### Immediate Rollback

```bash
# Remove Go CLI
rm /usr/local/bin/cline  # or brew uninstall cline/tap/cline

# Reinstall TypeScript CLI
npm install -g @cline/cli

# Verify
cline version
```

### Restore Data

```bash
# If you backed up before migration
tar xzf ~/cline-backup-YYYYMMDD.tar.gz -C ~

# Or manually move files back
mv ~/.cline/data/globalState.json ~/.cline/
mv ~/.cline/data/secrets.json ~/.cline/
mv ~/.cline/data/tasks/* ~/.cline/tasks/
```

### Version Pinning

To stay on TypeScript CLI temporarily:

```bash
# Install specific version
npm install -g @cline/cli@2.x.x

# Pin in package.json
{
  "dependencies": {
    "@cline/cli": "2.x.x"
  }
}
```

---

## Verification Checklist

After migration, verify:

- [ ] `cline version` shows correct version
- [ ] `cline config list` shows your configuration
- [ ] `cline history` shows recent tasks (if migrated)
- [ ] `cline auth status` shows authentication status
- [ ] `cline "hello"` starts a task successfully
- [ ] `--json` flag produces valid JSON output
- [ ] Exit codes match expectations in scripts

---

## Getting Help

### Migration Issues

- **GitHub Issues:** https://github.com/cline/cline/issues
- **Discord:** https://discord.gg/cline
- **Documentation:** https://docs.cline.bot

### Report Migration Bug

Include:
1. TypeScript CLI version you were using
2. Go CLI version you migrated to
3. Installation method (brew, npm, direct download)
4. Error message or unexpected behavior
5. Steps to reproduce

---

## Summary

The Go CLI migration should be seamless for most users:

1. **Backup** your data
2. **Install** the Go CLI
3. **Run** any command - migration is automatic
4. **Verify** everything works
5. **Enjoy** 10x faster startup!

The Go CLI maintains full compatibility with the TypeScript CLI while providing significant performance improvements.

---

*Last updated: March 2026*