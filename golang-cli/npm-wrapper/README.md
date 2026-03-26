# @cline/golang-cli

NPM wrapper for the Cline CLI (Go implementation).

## Installation

```bash
npm install -g @cline/golang-cli
```

## Usage

After installation, the `cline-go` command will be available:

```bash
# Run Cline CLI
cline-go "Your prompt here"

# Get help
cline-go --help

# See all commands
cline-go --help-all
```

## Binary Name

The NPM wrapper installs the binary as `cline-go` to avoid conflicts with other installations. If you want to use `cline` as the command, you can create an alias:

```bash
# Bash/Zsh
alias cline='cline-go'

# Add to ~/.bashrc or ~/.zshrc for persistence
echo "alias cline='cline-go'" >> ~/.zshrc
```

## How It Works

This package downloads the appropriate binary for your platform during installation:

1. Detects your platform (macOS, Linux, Windows) and architecture (x64, arm64)
2. Downloads the correct binary from GitHub Releases
3. Stores it in `node_modules/@cline/golang-cli/bin/`
4. Provides the `cline-go` command

## Updating

To update to the latest version:

```bash
npm update -g @cline/golang-cli
```

## Manual Installation

If the automatic download fails, you can manually download the binary from:
https://github.com/cline/cline/releases

Place it in `node_modules/@cline/golang-cli/bin/` and name it:
- `cline-go` (macOS/Linux)
- `cline-go.exe` (Windows)

## Troubleshooting

### Binary not found after installation

```bash
# Reinstall the package
npm uninstall -g @cline/golang-cli
npm install -g @cline/golang-cli

# Or download manually from GitHub releases
```

### Permission denied (macOS/Linux)

```bash
# Make the binary executable
chmod +x $(npm root -g)/@cline/golang-cli/bin/cline-go
```

### Windows: Command not found

Make sure your npm global bin directory is in your PATH:

```powershell
# Find the global bin directory
npm config get prefix

# Add to PATH (PowerShell)
[Environment]::SetEnvironmentVariable("Path", $env:Path + ";$(npm config get prefix)", "User")
```

## Alternative Installations

If you prefer not to use NPM, you can install Cline CLI via:

- **Homebrew** (macOS/Linux): `brew install cline`
- **Scoop** (Windows): `scoop install cline`
- **Direct download**: https://github.com/cline/cline/releases

## Documentation

For full documentation, visit: https://docs.cline.bot

## License

Apache-2.0