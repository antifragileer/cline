# @cline/golang-cli

[![NPM Version](https://img.shields.io/npm/v/@cline/golang-cli.svg)](https://www.npmjs.com/package/@cline/golang-cli)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

NPM wrapper for the Cline CLI Go implementation. This package downloads and installs the appropriate Cline CLI binary for your platform.

## Features

- **Automatic platform detection** - Downloads the correct binary for your OS and architecture
- **Checksum verification** - Verifies downloaded binaries against SHA256 checksums
- **Cross-platform** - Supports macOS, Linux, and Windows
- **Zero configuration** - Works out of the box after installation
- **Lightweight** - No Node.js runtime dependencies for the CLI itself

## Installation

### Global Installation (Recommended)

```bash
npm install -g @cline/golang-cli
```

After installation, the `cline` command will be available globally.

## Binary Name

The NPM wrapper installs the binary as `cline`, which is the same name as the native binary. This provides a seamless experience whether you install via NPM, Homebrew, or download the binary directly.

### Binary Location

By default, the binary is downloaded to:
- **Global install**: `node_modules/@cline/golang-cli/bin/cline`
- **Local install**: `./node_modules/@cline/golang-cli/bin/cline`

The wrapper creates a symlink in your npm bin directory so the `cline` command is available in your PATH.

### Using npx

```bash
npx @cline/golang-cli "your prompt here"
```

### Local Installation

```bash
npm install --save-dev @cline/golang-cli
```

Then use in your `package.json` scripts:

```json
{
  "scripts": {
    "cline": "cline"
  }
}
```

## Platform Support

| Platform | Architecture | Status |
|----------|--------------|--------|
| macOS    | Intel (x64)  | ✅ Supported |
| macOS    | Apple Silicon (arm64) | ✅ Supported |
| Linux    | x64          | ✅ Supported |
| Linux    | ARM64        | ✅ Supported |
| Windows  | x64          | ✅ Supported |

## Usage

Once installed, use the `cline` command exactly like the native binary:

```bash
# Start a new task
cline "Create a React component"

# Plan mode
cline -p "Plan this feature"

# Act mode (auto-execute)
cline -a "Implement this feature"

# YOLO mode (auto-approve all)
cline -y "Make all the changes"

# View history
cline history

# Configure settings
cline config

# Check version
cline version
```

## Post-Install Behavior

When you install this package, it automatically:

1. Detects your operating system and architecture
2. Downloads the appropriate Cline CLI binary
3. Verifies the binary against its SHA256 checksum
4. Makes the binary available as the `cline` command

If the download fails, you'll see an error message with troubleshooting steps.

## Environment Variables

| Variable | Description |
|----------|-------------|
| `CLINE_INSTALL_DIR` | Custom directory for the binary (default: `node_modules/@cline/golang-cli/bin`) |
| `CLINE_SKIP_DOWNLOAD` | Skip binary download during install (set to `1` or `true`) |
| `CLINE_FORCE_DOWNLOAD` | Force re-download even if binary exists |
| `CLINE_MIRROR` | Use a custom mirror URL for downloads |

## Troubleshooting

### Binary not found after installation

```bash
# Check if npm global bin is in your PATH
npm bin -g

# Add to your shell profile if needed
export PATH="$PATH:$(npm bin -g)"
```

### Download fails during installation

1. Check your internet connection
2. Verify you can access GitHub: `curl -I https://github.com`
3. If behind a proxy, set `HTTPS_PROXY` environment variable
4. Try manual download: See [Cline CLI releases](https://github.com/cline/cline/releases)

### Wrong binary downloaded

```bash
# Check detected platform
node -e "console.log(require('@cline/golang-cli/platform').getPlatformInfo())"

# Force specific platform (not recommended)
CLINE_PLATFORM=linux CLINE_ARCH=amd64 npm install -g @cline/golang-cli
```

### Permission denied

```bash
# On Linux/macOS, ensure proper permissions
chmod +x $(npm bin -g)/cline

# Or use sudo for global install
sudo npm install -g @cline/golang-cli
```

## Wrapper Commands

The wrapper provides additional commands:

```bash
# Check wrapper installation
cline --wrapper-check

# Show wrapper version
cline --wrapper-version

# Get installation diagnostics
cline --wrapper-check --verbose
```

## Differences from Native Binary

This NPM wrapper is a thin layer around the native Cline CLI binary. The actual CLI functionality comes from the Go binary, not Node.js. The wrapper only handles:

- Platform detection
- Binary download
- Process spawning
- Exit code propagation

All CLI features are identical to the native binary.

## Updating

```bash
# Update to latest version
npm update -g @cline/golang-cli

# Or reinstall
npm uninstall -g @cline/golang-cli
npm install -g @cline/golang-cli
```

## Uninstallation

```bash
npm uninstall -g @cline/golang-cli
```

This removes the wrapper and the downloaded binary.

## License

Apache-2.0 - See [LICENSE](../LICENSE) for details.

## Related Packages

- [`@cline/cli`](https://www.npmjs.com/package/@cline/cli) - Original TypeScript CLI (deprecated in favor of this package)
- [`cline`](https://www.npmjs.com/package/cline) - Different package (not affiliated)

## Support

- **Issues**: https://github.com/cline/cline/issues
- **Documentation**: https://docs.cline.bot
- **Discord**: https://discord.gg/cline