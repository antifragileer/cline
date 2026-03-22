# @cline/golang-cli

NPM wrapper package for the Cline Go CLI - downloads and installs the platform-specific binary during npm install.

## Installation

```bash
npm install -g @cline/golang-cli
```

Or as a project dependency:

```bash
npm install --save-dev @cline/golang-cli
```

## Usage

### Command Line

After installation, the `cline-go` binary will be available:

```bash
# Run the CLI
cline-go --help

# Start a new task
cline-go task "Create a React component"

# Check version
cline-go --version
```

### Programmatic API

```javascript
const cline = require('@cline/golang-cli');

// Check if binary is installed
if (cline.isInstalled()) {
    console.log('Cline is installed!');
}

// Get binary path
const binaryPath = cline.getBinaryPath();
console.log(`Binary at: ${binaryPath}`);

// Run a command
const child = cline.run(['--version']);
child.on('exit', (code) => {
    console.log(`Process exited with code ${code}`);
});

// Run synchronously
const result = cline.runSync(['--help']);
console.log(result.stdout);
```

## Supported Platforms

| Platform | Architecture |
|----------|-------------|
| macOS    | x64, arm64  |
| Linux    | x64, arm64  |
| Windows  | x64, arm64  |

## Proxy Configuration

The installer respects standard proxy environment variables:

```bash
# HTTP proxy
export HTTP_PROXY=http://proxy.example.com:8080

# HTTPS proxy (preferred)
export HTTPS_PROXY=https://proxy.example.com:8080

# With authentication
export HTTPS_PROXY=https://user:pass@proxy.example.com:8080
```

## Installation Process

During `npm install`, the postinstall script:

1. Detects your platform and architecture
2. Downloads the appropriate binary from GitHub releases
3. Verifies the SHA256 checksum
4. Sets executable permissions (Unix/Linux/macOS)
5. Places the binary in `bin/`

## Troubleshooting

### Binary not found after installation

If the binary is not found:

```bash
# Re-run the installer
npm run postinstall

# Or manually trigger
node node_modules/@cline/golang-cli/install.js
```

### Download failures

If downloads fail behind a proxy, ensure `HTTPS_PROXY` is set correctly:

```bash
export HTTPS_PROXY=http://your-proxy:8080
npm install @cline/golang-cli
```

### Checksum verification failed

If checksum verification fails, the download may be corrupted:

```bash
# Remove and reinstall
rm -rf node_modules/@cline/golang-cli
npm install @cline/golang-cli
```

### Platform not supported

If you see "Unsupported platform" or "Unsupported architecture", your system is not in the supported list. Check your platform with:

```bash
node -e "console.log(process.platform, process.arch)"
```

## Development

### Running Tests

```bash
# Run all tests
node --test install.test.js index.test.js

# Run with debug output
DEBUG=1 node --test install.test.js
```

### Building the Package

```bash
# Create npm package
npm pack

# Publish (requires auth)
npm publish --access public
```

## Configuration

The installer can be configured via environment variables:

| Variable | Description |
|----------|-------------|
| `DEBUG` | Enable debug output during installation |
| `HTTPS_PROXY` | HTTPS proxy URL |
| `HTTP_PROXY` | HTTP proxy URL |

## Exit Codes

The installer uses the following exit codes:

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Unsupported platform |
| 2 | Unsupported architecture |
| 3 | Download failed |
| 4 | Checksum verification failed |
| 5 | Installation failed |
| 6 | Network error |

## License

Apache-2.0 - See [LICENSE](../LICENSE) for details.

## Repository

https://github.com/cline/cline