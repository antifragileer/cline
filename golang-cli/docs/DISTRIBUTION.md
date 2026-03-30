# Cline CLI Distribution Guide

This document describes how to build, package, and distribute the Cline CLI across all supported platforms.

## Table of Contents

- [Distribution Overview](#distribution-overview)
- [Prerequisites](#prerequisites)
- [Build Process](#build-process)
- [Package Managers](#package-managers)
- [Release Process](#release-process)
- [Verification](#verification)
- [Troubleshooting](#troubleshooting)

## Distribution Overview

The Cline CLI is distributed through multiple channels:

| Method | Platforms | Users |
|--------|-----------|-------|
| Homebrew | macOS, Linux | Developers |
| Scoop | Windows | Developers |
| NPM | All (wrapper) | Node.js users |
| Direct Download | All | All users |
| Linux Packages | Linux | Enterprise |

### Supported Platforms

- **macOS**: Intel (amd64), Apple Silicon (arm64)
- **Linux**: x86_64 (amd64), ARM64
- **Windows**: x86_64 (amd64), ARM64

## Prerequisites

### Build Requirements

- **Go 1.25+**: [Download](https://go.dev/dl/)
- **Git**: For version information
- **Make**: For build automation
- **tar/zip**: For archive creation

### macOS Signing (Optional but Recommended)

For signed macOS binaries:

- Apple Developer ID certificate
- `codesign` and `notarytool` tools

Install certificates and configure environment:

```bash
export CODESIGN_IDENTITY="Developer ID Application: Your Name (TEAM_ID)"
export APPLE_ID="your.email@example.com"
export APPLE_TEAM_ID="TEAM_ID"
export APPLE_APP_PASSWORD="app-specific-password"
```

### Windows Signing (Optional)

For signed Windows binaries:

- Windows code signing certificate
- `signtool.exe` from Windows SDK

## Build Process

### Quick Build (Development)

```bash
# Build for current platform
make build

# Build with debug info
make dev
```

### Cross-Platform Build

```bash
# Build all platforms
make build-all

# Build signed binaries (macOS only)
make build-signed
```

### Build Outputs

Build artifacts are placed in `dist/`:

```
dist/
├── cline-v1.0.0-darwin-amd64         # macOS Intel binary
├── cline-v1.0.0-darwin-amd64.tar.gz  # macOS Intel archive
├── cline-v1.0.0-darwin-arm64         # macOS Apple Silicon binary
├── cline-v1.0.0-darwin-arm64.tar.gz  # macOS Apple Silicon archive
├── cline-v1.0.0-linux-amd64          # Linux x86_64 binary
├── cline-v1.0.0-linux-amd64.tar.gz   # Linux x86_64 archive
├── cline-v1.0.0-linux-arm64          # Linux ARM64 binary
├── cline-v1.0.0-linux-arm64.tar.gz   # Linux ARM64 archive
├── cline-v1.0.0-windows-amd64.exe    # Windows x86_64 binary
├── cline-v1.0.0-windows-amd64.zip    # Windows x86_64 archive
├── checksums.txt                     # SHA256 checksums
└── checksums.txt.sig                 # GPG signature (optional)
```

## Package Managers

### Homebrew

#### Formula Generation

Generate the Homebrew formula:

```bash
# From release binaries
make release-homebrew VERSION=1.0.0

# Or manually
go run ./cmd/generate-homebrew/main.go \
  -version 1.0.0 \
  -local \
  -binary-dir ./dist \
  -output dist/cline.rb
```

#### Tap Structure

The Homebrew tap repository structure:

```
homebrew-tap/
├── Formula/
│   └── cline.rb          # Formula file
├── README.md             # Tap documentation
└── .github/
    └── workflows/        # CI workflows
```

#### Publishing to Tap

```bash
# Clone the tap repository
git clone https://github.com/cline/homebrew-tap.git
cd homebrew-tap

# Copy the generated formula
cp ../cline/golang-cli/dist/cline.rb Formula/

# Commit and push
git add Formula/cline.rb
git commit -m "Update cline to v1.0.0"
git push
```

#### Testing the Formula

```bash
# Install from local formula
brew install --formula ./Formula/cline.rb

# Test the installation
cline version

# Verify binary location
which cline
```

### Scoop (Windows)

#### Manifest Generation

Generate the Scoop manifest:

```bash
# From release binaries
make release-scoop VERSION=1.0.0

# Or manually
go run ./cmd/scoop-generate/main.go \
  -version 1.0.0 \
  -local \
  -binary-dir ./dist \
  -output dist/cline.json
```

#### Bucket Structure

The Scoop bucket repository structure:

```
scoop-cline/
├── bucket/
│   ├── cline.json        # Main manifest
│   └── cline-beta.json   # Beta manifest (optional)
├── README.md
└── .github/
    └── workflows/        # CI workflows
```

#### Publishing to Bucket

```bash
# Clone the bucket repository
git clone https://github.com/cline/scoop-cline.git
cd scoop-cline

# Copy the generated manifest
cp ../cline/golang-cli/dist/cline.json bucket/

# Commit and push
git add bucket/cline.json
git commit -m "Update cline to v1.0.0"
git push
```

#### Testing the Manifest

```bash
# Add the bucket locally
scoop bucket add cline-local ./scoop-cline

# Install from local bucket
scoop install cline-local/cline

# Test the installation
cline version
```

### NPM Wrapper

#### Package Structure

The NPM wrapper package structure:

```
npm-wrapper/
├── package.json          # Package configuration
├── index.js              # Main entry point
├── install.js            # Post-install script
├── platform.js           # Platform detection
├── README.md
└── bin/                  # Pre-built binaries (optional)
    ├── cline-v1.0.0-darwin-amd64.tar.gz
    ├── cline-v1.0.0-darwin-arm64.tar.gz
    ├── cline-v1.0.0-linux-amd64.tar.gz
    ├── cline-v1.0.0-linux-arm64.tar.gz
    └── cline-v1.0.0-windows-amd64.zip
```

#### Building NPM Package

```bash
# Create NPM package structure
make release-npm VERSION=1.0.0

# Or manually
mkdir -p dist/npm/bin
cp npm-wrapper/package.json npm-wrapper/index.js npm-wrapper/install.js npm-wrapper/platform.js dist/npm/
cp dist/cline-*.tar.gz dist/npm/bin/ 2>/dev/null || true
cp dist/cline-*.zip dist/npm/bin/ 2>/dev/null || true
```

#### Publishing to NPM

```bash
cd dist/npm

# Test installation locally (optional)
npm pack
npm install -g ./cline-golang-cli-1.0.0.tgz

# Publish to NPM
npm publish --access public
```

#### NPM Package Features

- **Automatic binary download**: Downloads appropriate binary during install
- **Checksum verification**: Verifies SHA256 checksums
- **Platform detection**: Automatically detects OS and architecture
- **Fallback mechanism**: Can bundle binaries for offline installation

### Linux Packages

#### DEB Package (Debian/Ubuntu)

Create a DEB package:

```bash
# Create package structure
mkdir -p dist/deb/cline_1.0.0_amd64/DEBIAN
mkdir -p dist/deb/cline_1.0.0_amd64/usr/bin
mkdir -p dist/deb/cline_1.0.0_amd64/usr/share/doc/cline

# Copy binary
cp dist/cline-v1.0.0-linux-amd64 dist/deb/cline_1.0.0_amd64/usr/bin/cline
chmod 755 dist/deb/cline_1.0.0_amd64/usr/bin/cline

# Create control file
cat > dist/deb/cline_1.0.0_amd64/DEBIAN/control << 'EOF'
Package: cline
Version: 1.0.0
Section: devel
Priority: optional
Architecture: amd64
Maintainer: Cline <support@cline.bot>
Description: AI-powered coding assistant CLI
 Cline is an AI-powered coding assistant that runs in your terminal.
Homepage: https://github.com/cline/cline
EOF

# Create copyright file
cp LICENSE dist/deb/cline_1.0.0_amd64/usr/share/doc/cline/copyright

# Build package
dpkg-deb --build dist/deb/cline_1.0.0_amd64
```

#### RPM Package (RedHat/CentOS/Fedora)

Create an RPM package using `rpmbuild`:

```bash
# Create RPM structure
mkdir -p ~/rpmbuild/{BUILD,RPMS,SOURCES,SPECS,SRPMS}

# Copy binary
cp dist/cline-v1.0.0-linux-amd64 ~/rpmbuild/SOURCES/cline

# Create spec file
cat > ~/rpmbuild/SPECS/cline.spec << 'EOF'
Name:           cline
Version:        1.0.0
Release:        1%{?dist}
Summary:        AI-powered coding assistant CLI

License:        Apache-2.0
URL:            https://github.com/cline/cline
Source0:        cline

%description
Cline is an AI-powered coding assistant that runs in your terminal.

%install
mkdir -p %{buildroot}/usr/bin
cp %{SOURCE0} %{buildroot}/usr/bin/cline
chmod 755 %{buildroot}/usr/bin/cline

%files
/usr/bin/cline

%changelog
* Mon Jan 01 2024 Cline <support@cline.bot> - 1.0.0-1
- Initial release
EOF

# Build RPM
rpmbuild -bb ~/rpmbuild/SPECS/cline.spec
```

### Direct Download

Users can download binaries directly from GitHub Releases:

**macOS (Apple Silicon):**
```bash
curl -L -o cline "https://github.com/cline/cline/releases/latest/download/cline_$(curl -s https://api.github.com/repos/cline/cline/releases/latest | grep tag_name | cut -d '"' -f 4)_darwin_arm64"
chmod +x cline
sudo mv cline /usr/local/bin/
```

**macOS (Intel):**
```bash
curl -L -o cline "https://github.com/cline/cline/releases/latest/download/cline_$(curl -s https://api.github.com/repos/cline/cline/releases/latest | grep tag_name | cut -d '"' -f 4)_darwin_amd64"
chmod +x cline
sudo mv cline /usr/local/bin/
```

**Linux:**
```bash
curl -L -o cline "https://github.com/cline/cline/releases/latest/download/cline_$(curl -s https://api.github.com/repos/cline/cline/releases/latest | grep tag_name | cut -d '"' -f 4)_linux_amd64"
chmod +x cline
sudo mv cline /usr/local/bin/
```

**Windows (PowerShell):**
```powershell
Invoke-WebRequest -Uri "https://github.com/cline/cline/releases/latest/download/cline_VERSION_windows_amd64.exe" -OutFile "cline.exe"
```

## Release Process

### Automated Release (GitHub Actions)

The release process is automated via GitHub Actions:

1. **Tag the release:**
   ```bash
   git tag -a v1.0.0 -m "Release version 1.0.0"
   git push origin v1.0.0
   ```

2. **GitHub Actions will:**
   - Build all platform binaries
   - Run tests
   - Sign macOS binaries (if configured)
   - Create archives and checksums
   - Create GitHub release with artifacts
   - Update Homebrew tap
   - Update Scoop bucket
   - Publish NPM package (if configured)

### Manual Release

If you need to create a release manually:

```bash
# 1. Set version
export VERSION=1.0.0

# 2. Clean and build
make clean
make build-all

# 3. Create archives
./scripts/build.sh --all --archive --checksum

# 4. Generate package manager files
make release-homebrew VERSION=$VERSION
make release-scoop VERSION=$VERSION
make release-npm VERSION=$VERSION

# 5. Create GitHub release
gh release create v$VERSION \
  --title "v$VERSION" \
  --notes "Release notes here" \
  dist/cline-v$VERSION-*.{tar.gz,zip} \
  dist/checksums.txt

# 6. Update package managers
# - Copy dist/cline.rb to homebrew-tap repository
# - Copy dist/cline.json to scoop-cline repository
# - Publish dist/npm to NPM registry
```

### Version Numbering

We follow [Semantic Versioning](https://semver.org/):

- **MAJOR**: Incompatible API changes
- **MINOR**: New functionality (backward compatible)
- **PATCH**: Bug fixes (backward compatible)

Pre-release versions:

```bash
# Alpha
v1.0.0-alpha.1

# Beta
v1.0.0-beta.1

# Release Candidate
v1.0.0-rc.1
```

## Verification

### Binary Verification

#### Checksum Verification

```bash
# Download checksums
curl -L -O https://github.com/cline/cline/releases/download/v1.0.0/checksums.txt

# Verify a binary
sha256sum -c checksums.txt --ignore-missing
```

#### GPG Signature (Optional)

```bash
# Import public key
gpg --keyserver keyserver.ubuntu.com --recv-keys KEY_ID

# Verify signature
gpg --verify checksums.txt.sig checksums.txt
```

### Installation Verification

```bash
# Verify binary works
cline version

# Verify all commands
cline --help
cline task --help
cline history --help

# Verify no Node.js dependencies (Go CLI only)
./scripts/verify_independence.sh
```

### Package Manager Verification

#### Homebrew

```bash
# Verify formula syntax
brew style cline/tap/cline

# Audit formula
brew audit --strict cline/tap/cline

# Test installation
brew test cline/tap/cline
```

#### Scoop

```bash
# Validate manifest
scoop validate bucket/cline.json

# Check for updates
scoop update
scoop status
```

#### NPM

```bash
# Verify package
npm pack --dry-run

# Check for vulnerabilities
npm audit

# Test installation
npm install -g @cline/golang-cli
```

## Troubleshooting

### Build Issues

#### Go Build Fails

```bash
# Clean caches
make cache-clean-all

# Re-download dependencies
go mod download

# Try building with verbose output
go build -v ./cmd/cline
```

#### Cross-Compilation Fails

Ensure you have the required toolchains:

```bash
# macOS: Install Xcode command line tools
xcode-select --install

# Linux: Install build-essential
sudo apt-get install build-essential

# Windows: Install mingw-w64 (for cross-compiling from Linux/macOS)
brew install mingw-w64  # macOS
sudo apt-get install mingw-w64  # Ubuntu/Debian
```

### Signing Issues

#### macOS Signing Fails

```bash
# Check certificate
security find-identity -v -p codesigning

# Reset certificates
security delete-identity -c "Developer ID Application"

# Import certificate again
security import certificate.p12 -k ~/Library/Keychains/login.keychain
```

#### Notarization Fails

```bash
# Check notarization status
xcrun notarytool history --apple-id "$APPLE_ID" --team-id "$APPLE_TEAM_ID"

# Staple ticket manually
xcrun stapler staple dist/cline-v1.0.0-darwin-amd64.tar.gz
```

### Distribution Issues

#### Homebrew Formula Fails

```bash
# Debug formula generation
go run ./cmd/generate-homebrew/main.go -version 1.0.0 -dry-run

# Test formula locally
brew install --build-from-source ./Formula/cline.rb
```

#### NPM Package Fails

```bash
# Check package contents
npm pack --dry-run

# Debug install script
node -e "require('./npm-wrapper/install.js')"

# Check platform detection
node -e "console.log(require('./npm-wrapper/platform').getPlatformInfo())"
```

## Maintenance

### Regular Tasks

- **Update Go version**: Update `go.mod` and CI workflows
- **Update dependencies**: `make update-deps`
- **Security updates**: Monitor CVEs for dependencies
- **Documentation**: Keep README and docs up to date

### Monitoring

Monitor distribution channels:

- GitHub Releases download counts
- Homebrew install metrics
- NPM download statistics
- Scoop install counts
- Issue reports and feedback

## Support

For distribution-related issues:

- **Homebrew**: https://github.com/cline/homebrew-tap/issues
- **Scoop**: https://github.com/cline/scoop-cline/issues
- **NPM**: https://github.com/cline/cline/issues
- **General**: https://github.com/cline/cline/issues