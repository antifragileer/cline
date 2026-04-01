# Cline Go CLI Troubleshooting Guide

This guide helps you diagnose and resolve common issues with the Cline Go CLI.

## Table of Contents

- [Installation Issues](#installation-issues)
- [Runtime Errors](#runtime-errors)
- [gRPC Connection Issues](#grpc-connection-issues)
- [Authentication Problems](#authentication-problems)
- [Performance Issues](#performance-issues)
- [TUI Rendering Issues](#tui-rendering-issues)
- [Task Execution Issues](#task-execution-issues)
- [State and Storage Issues](#state-and-storage-issues)
- [Debugging](#debugging)
- [Getting Help](#getting-help)

---

## Installation Issues

### Binary Not Found After Installation

**Symptoms:**
```
$ cline --help
command not found: cline
```

**Solutions:**

1. **Verify binary location:**
   ```bash
   which cline
   # If empty, binary is not in PATH
   
   # Find the binary
   find /usr/local -name "cline" 2>/dev/null
   find $HOME -name "cline" 2>/dev/null
   ```

2. **Add to PATH:**
   ```bash
   # For current session
   export PATH="$PATH:/usr/local/bin"
   
   # Permanently (add to ~/.bashrc, ~/.zshrc, or ~/.bash_profile)
   echo 'export PATH="$PATH:/usr/local/bin"' >> ~/.bashrc
   source ~/.bashrc
   ```

3. **Create symlink:**
   ```bash
   sudo ln -s /path/to/cline /usr/local/bin/cline
   ```

### Permission Denied

**Symptoms:**
```
$ cline --help
permission denied: cline
```

**Solutions:**

```bash
# Make binary executable
chmod +x /path/to/cline

# Or use sudo for system-wide installation
sudo chmod +x /usr/local/bin/cline
```

### Homebrew Installation Issues

**Symptoms:**
```
Error: cline/tap/cline is not a valid formula
```

**Solutions:**

```bash
# Update Homebrew
brew update

# Re-tap the repository
brew untap cline/tap
brew tap cline/tap

# Retry installation
brew install cline
```

### NPM Wrapper Issues

**Symptoms:**
```
$ cline-go
cline-go: command not found
```

**Solutions:**

```bash
# Verify npm global bin is in PATH
npm bin -g
echo $PATH | grep "$(npm bin -g)"

# Add to PATH if missing
export PATH="$PATH:$(npm bin -g)"

# Or use npx
npx @cline/golang-cli --help
```

---

## Runtime Errors

### Exit Code 1: General Error

**Symptoms:**
Command fails with exit code 1.

**Diagnosis:**
```bash
# Get detailed error
cline --verbose <command>
```

**Common Causes:**

1. **Invalid configuration:**
   ```bash
   # Reset to defaults
   cline config reset
   ```

2. **Corrupted state:**
   ```bash
   # Backup and clear state
   mv ~/.cline/data ~/.cline/data.backup
   # Restart cline - it will recreate state
   ```

3. **Missing required arguments:**
   ```bash
   # Check command usage
   cline <command> --help
   ```

### Exit Code 2: Misuse of Command

**Symptoms:**
```
Error: unknown flag: --invalid-flag
```

**Solution:**
```bash
# Check correct usage
cline --help
cline <command> --help
```

### Exit Code 126: Command Not Executable

**Symptoms:**
Binary exists but cannot execute.

**Solutions:**

```bash
# Check file type
file $(which cline)

# Check architecture compatibility
uname -m
# Compare with: cline version --json | grep arch

# Re-download correct binary
# See Installation section in README.md
```

### Exit Code 127: Command Not Found

**Symptoms:**
Binary missing or not in PATH.

**Solution:**
Follow steps in [Binary Not Found After Installation](#binary-not-found-after-installation).

---

## gRPC Connection Issues

### "Failed to connect to core extension"

**Symptoms:**
```
Error: gRPC connection failed: connection refused
```

**Diagnosis:**

1. **Check if VS Code extension is running:**
   - Open VS Code
   - Verify Cline extension is installed and enabled
   - Look for Cline icon in sidebar

2. **Check gRPC port:**
   ```bash
   # Find cline processes
   ps aux | grep cline
   
   # Check listening ports
   lsof -i | grep cline
   ```

3. **Check logs:**
   ```bash
   cline dev log
   ```

**Solutions:**

1. **Restart VS Code:**
   - Close VS Code completely
   - Reopen VS Code
   - Try CLI again

2. **Check extension version:**
   ```bash
   # In VS Code, check Cline extension version
   # Should be compatible with CLI version
   cline version
   ```

3. **Manual gRPC connection:**
   ```bash
   # Check if gRPC server is listening
   cline dev status
   ```

4. **Firewall/proxy issues:**
   ```bash
   # Check proxy settings
   echo $HTTP_PROXY
   echo $HTTPS_PROXY
   
   # Temporarily disable proxy for local connections
   unset HTTP_PROXY
   unset HTTPS_PROXY
   ```

### gRPC Timeout

**Symptoms:**
```
Error: gRPC call timeout
```

**Solutions:**

1. **Increase timeout:**
   ```bash
   cline -t 60s "your prompt"
   ```

2. **Check system load:**
   ```bash
   top
   # If system is overloaded, wait or reduce load
   ```

3. **Restart connection:**
   ```bash
   # Kill any hanging cline processes
   pkill -f cline
   
   # Retry
   cline "your prompt"
   ```

---

## Authentication Problems

### "API key not found"

**Symptoms:**
```
Error: ANTHROPIC_API_KEY not set
```

**Solutions:**

1. **Set via environment:**
   ```bash
   export ANTHROPIC_API_KEY="your-key-here"
   
   # Add to shell profile for persistence
   echo 'export ANTHROPIC_API_KEY="your-key-here"' >> ~/.bashrc
   ```

2. **Authenticate interactively:**
   ```bash
   cline auth
   ```

3. **Check auth status:**
   ```bash
   cline auth status
   ```

### "Invalid API key"

**Symptoms:**
```
Error: API request failed: 401 Unauthorized
```

**Solutions:**

1. **Verify key format:**
   ```bash
   # Anthropic keys start with 'sk-ant-'
   echo $ANTHROPIC_API_KEY | head -c 20
   ```

2. **Check key validity:**
   ```bash
   # Test with curl
   curl https://api.anthropic.com/v1/messages \
     -H "x-api-key: $ANTHROPIC_API_KEY" \
     -H "anthropic-version: 2023-06-01" \
     -H "content-type: application/json" \
     -d '{"model":"claude-3-sonnet-20240229","max_tokens":1024,"messages":[{"role":"user","content":"Hello"}]}'
   ```

3. **Re-authenticate:**
   ```bash
   cline auth logout
   cline auth
   ```

### OAuth Flow Issues

**Symptoms:**
OAuth callback fails or hangs.

**Solutions:**

1. **Check browser:**
   - Ensure default browser opens
   - Check if pop-up blocker is preventing OAuth window

2. **Manual callback:**
   ```bash
   # If automatic callback fails, copy the URL from browser
   # and run:
   cline auth callback <callback-url>
   ```

3. **Check firewall:**
   ```bash
   # Local callback server needs port access
   # Usually ports 8000-9000
   ```

---

## Performance Issues

### Slow Startup

**Symptoms:**
CLI takes more than 1 second to start.

**Diagnosis:**
```bash
# Time the startup
time cline version --short

# Check binary size
ls -lh $(which cline)

# Check if binary is stripped
file $(which cline)
```

**Solutions:**

1. **Use optimized binary:**
   ```bash
   # Ensure you're using release build, not dev build
   cline version --json
   # Check "buildType" - should be "release"
   ```

2. **Clear build cache:**
   ```bash
   make cache-clean
   ```

3. **Check antivirus:**
   - Some antivirus software scans binaries on each run
   - Add cline to exclusions list

### High Memory Usage

**Symptoms:**
CLI consumes >500MB RAM.

**Diagnosis:**
```bash
# Monitor memory usage
clinewatch -n 1 'ps aux | grep cline'

# Or use activity monitor
```

**Solutions:**

1. **Limit concurrent operations:**
   ```bash
   # Reduce parallelism
   cline --max-concurrent=2 "task"
   ```

2. **Clear task history:**
   ```bash
   # Large history can consume memory
   cline history --clear
   ```

3. **Restart:**
   ```bash
   # Kill and restart
   pkill -f cline
   ```

### Slow Task Execution

**Symptoms:**
Tasks take longer than expected.

**Solutions:**

1. **Check network:**
   ```bash
   ping api.anthropic.com
   # or
   curl -w "@curl-format.txt" -o /dev/null -s https://api.anthropic.com
   ```

2. **Use faster model:**
   ```bash
   # Use faster, cheaper model for simple tasks
   cline -m claude-3-haiku-20240307 "simple task"
   ```

3. **Enable streaming:**
   ```bash
   # Already enabled by default
   # Check if disabled in config
   cline config get streamingEnabled
   ```

---

## TUI Rendering Issues

### Garbled Output

**Symptoms:**
- Strange characters
- Misaligned text
- Missing UI elements

**Diagnosis:**
```bash
# Check terminal capabilities
echo $TERM
tput colors

# Check terminal size
stty size
```

**Solutions:**

1. **Use compatible terminal:**
   - iTerm2 (macOS)
   - Windows Terminal (Windows)
   - GNOME Terminal/Konsole (Linux)

2. **Set proper TERM:**
   ```bash
   export TERM=xterm-256color
   ```

3. **Increase terminal size:**
   - Minimum recommended: 80x24
   - Optimal: 120x40 or larger

4. **Disable complex UI:**
   ```bash
   # Use plain text mode
   cline --json "prompt" | jq -r '.message'
   ```

### Colors Not Displaying

**Symptoms:**
- No colors in output
- ANSI escape codes visible

**Solutions:**

1. **Force color:**
   ```bash
   FORCE_COLOR=1 cline "prompt"
   ```

2. **Check NO_COLOR:**
   ```bash
   # Ensure not set
   unset NO_COLOR
   ```

3. **Terminal settings:**
   - Enable "Draw bold text in bright colors" in terminal preferences
   - Ensure terminal supports ANSI colors

### TUI Freezes

**Symptoms:**
- UI becomes unresponsive
- Cannot exit with Ctrl+C

**Solutions:**

1. **Force quit:**
   ```bash
   # Press Ctrl+\ (SIGQUIT)
   # Or from another terminal:
   pkill -9 -f cline
   ```

2. **Disable TUI:**
   ```bash
   # Use non-interactive mode
   cline --json "prompt"
   ```

3. **Check terminal multiplexer:**
   - If using tmux/screen, ensure proper $TERM
   ```bash
   # In tmux
   export TERM=screen-256color
   ```

---

## Task Execution Issues

### Task Not Starting

**Symptoms:**
- CLI hangs after prompt
- No response from AI

**Diagnosis:**
```bash
# Check if task is queued
cline dev status

# Check logs
cline dev log
```

**Solutions:**

1. **Check API status:**
   ```bash
   # Check if API is available
   curl -I https://api.anthropic.com
   ```

2. **Verify authentication:**
   ```bash
   cline auth status
   ```

3. **Check model availability:**
   ```bash
   # Use default model
   cline -m claude-3-5-sonnet-20241022 "prompt"
   ```

### Task Interrupted

**Symptoms:**
- "Task was interrupted" message
- Partial results

**Solutions:**

1. **Resume task:**
   ```bash
   cline --continue
   ```

2. **Check task ID:**
   ```bash
   cline history
   cline -T <task-id> --continue
   ```

### Tool Approval Not Working

**Symptoms:**
- Cannot approve/reject tools
- Approval prompts don't respond

**Solutions:**

1. **Use yolo mode (temporary):**
   ```bash
   cline -y "task requiring tools"
   ```

2. **Check auto-approve settings:**
   ```bash
   cline config get autoApprove
   ```

3. **Keyboard navigation:**
   - Use Tab/Arrow keys to navigate
   - Enter to select
   - q to quit

---

## State and Storage Issues

### Task History Not Visible

**Symptoms:**
```
cline history
# Shows empty or incomplete history
```

**Diagnosis:**
```bash
# Check data directory
ls -la ~/.cline/data/

# Check permissions
ls -la ~/.cline/

# Verify disk space
df -h ~/.cline
```

**Solutions:**

1. **Check correct data directory:**
   ```bash
   # If using custom data dir
   export CLINE_DATA_DIR=/path/to/data
   
   # Or specify on command line
   cline --config /path/to/data history
   ```

2. **Fix permissions:**
   ```bash
   chmod -R 755 ~/.cline
   ```

3. **Repair corrupted history:**
   ```bash
   # Backup first
   cp -r ~/.cline/data ~/.cline/data.backup.$(date +%s)
   
   # Reset history
   rm -rf ~/.cline/data/tasks/*
   ```

### Configuration Not Persisting

**Symptoms:**
Settings reset after restart.

**Diagnosis:**
```bash
# Check config file
cat ~/.cline/data/globalState.json

# Check permissions
ls -la ~/.cline/data/globalState.json
```

**Solutions:**

1. **Verify write permissions:**
   ```bash
   # Test write
   touch ~/.cline/data/test-write && rm ~/.cline/data/test-write
   ```

2. **Check disk space:**
   ```bash
   df -h ~/.cline
   ```

3. **Reset configuration:**
   ```bash
   # Backup
   cp ~/.cline/data/globalState.json ~/.cline/data/globalState.json.backup
   
   # Reset
   cline config reset
   ```

### Migration from TypeScript CLI

**Symptoms:**
- State from TypeScript CLI not visible
- Different behavior

**Solutions:**

1. **Check migration status:**
   ```bash
   # Data should be in ~/.cline/data/
   ls ~/.cline/data/
   ```

2. **Manual migration:**
   ```bash
   # Old TypeScript CLI used ~/.cline/ directly
   # New Go CLI uses ~/.cline/data/
   
   # Migrate if needed
   mkdir -p ~/.cline/data
   mv ~/.cline/globalState.json ~/.cline/data/ 2>/dev/null
   mv ~/.cline/secrets.json ~/.cline/data/ 2>/dev/null
   ```

---

## Debugging

### Enable Debug Mode

```bash
# Set debug environment variable
export CLINE_DEBUG=1

# Or use verbose flag
cline -v "prompt"
```

### View Logs

```bash
# Stream logs
cline dev log

# View recent logs
cline dev log --tail 100

# Follow logs (like tail -f)
cline dev log --follow
```

### Check System Information

```bash
# CLI version and info
cline version --json

# System diagnostics
cline dev diagnostics
```

### Verbose Output

```bash
# Maximum verbosity
cline -v --json "prompt"

# With timing
time cline -v "prompt"
```

### Network Debugging

```bash
# Test API connectivity
cline dev test-api

# Check proxy settings
env | grep -i proxy

# Test without proxy
HTTP_PROXY="" HTTPS_PROXY="" cline "prompt"
```

---

## Getting Help

### Gather Information

Before seeking help, gather:

```bash
# System info
cline version --json > cline-info.json
uname -a >> cline-info.txt

# Recent logs
cline dev log --tail 100 > cline-logs.txt

# Config (sanitized)
cline config list --json | jq 'del(.apiKeys, .secrets)' > cline-config.json
```

### Support Channels

1. **GitHub Issues:**
   - https://github.com/cline/cline/issues
   - Include: version, OS, error message, reproduction steps

2. **Discord:**
   - https://discord.gg/cline
   - Real-time community support

3. **Documentation:**
   - https://docs.cline.bot

### Reporting Bugs

Include:

1. **CLI version:**
   ```bash
   cline version --json
   ```

2. **Operating system:**
   ```bash
   uname -a
   ```

3. **Terminal:**
   ```bash
   echo $TERM
   ```

4. **Error message:** (copy-paste exact error)

5. **Steps to reproduce**

6. **Expected vs actual behavior**

---

## Quick Fixes

### Nuclear Option: Complete Reset

```bash
# Backup everything
tar czf ~/cline-backup-$(date +%Y%m%d).tar.gz ~/.cline

# Remove all data
rm -rf ~/.cline/data

# Re-authenticate
cline auth

# Test
cline "hello"
```

### Common One-Liners

```bash
# Fix permissions
chmod -R 755 ~/.cline && chown -R $(whoami) ~/.cline

# Clear caches
make cache-clean-all  # If building from source

# Update to latest
cline update

# Reset config
cline config reset

# Clean reinstall
rm -rf ~/.cline/data
brew reinstall cline  # or your installation method
cline auth
```

---

## Error Code Reference

| Exit Code | Meaning | Action |
|-----------|---------|--------|
| 0 | Success | - |
| 1 | General error | Check logs, verify config |
| 2 | Misuse of command | Check `--help` |
| 126 | Command not executable | Check permissions, architecture |
| 127 | Command not found | Check PATH, reinstall |
| 130 | Interrupted (Ctrl+C) | Retry or resume |
| 137 | Killed (OOM) | Free memory, retry |
| 255 | Unknown error | Check logs, report bug |

---

*Last updated: March 2026*