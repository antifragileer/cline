#!/bin/bash
#
# install.sh - Cline CLI Installation Script
#
# This script installs the Cline CLI on macOS and Linux systems.
# It automatically detects the platform architecture and downloads
# the appropriate binary.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/cline/cline/main/golang-cli/scripts/install.sh | sh
#   ./install.sh [--version VERSION] [--install-dir DIR] [--no-sudo]
#
# Options:
#   --version VERSION    Specific version to install (default: latest)
#   --install-dir DIR    Directory to install to (default: /usr/local/bin)
#   --no-sudo            Don't use sudo (install to ~/.local/bin instead)
#   -h, --help           Show this help message
#

set -euo pipefail

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

# Configuration
GITHUB_REPO="cline/cline"
BINARY_NAME="cline"
DEFAULT_INSTALL_DIR="/usr/local/bin"
USER_INSTALL_DIR="$HOME/.local/bin"
VERSION="latest"
USE_SUDO=true
INSTALL_DIR="$DEFAULT_INSTALL_DIR"

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1" >&2
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" >&2
}

log_warn() {
    echo -e "${YELLOW}[WARNING]${NC} $1" >&2
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

# Show help
show_help() {
    head -n 20 "$0" | tail -n 18
}

# Parse arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --version)
                VERSION="$2"
                shift 2
                ;;
            --install-dir)
                INSTALL_DIR="$2"
                shift 2
                ;;
            --no-sudo)
                USE_SUDO=false
                INSTALL_DIR="$USER_INSTALL_DIR"
                shift
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

# Detect OS
detect_os() {
    local os
    os="$(uname -s)"
    
    case "$os" in
        Linux*)     echo "linux" ;;
        Darwin*)    echo "darwin" ;;
        CYGWIN*|MINGW*|MSYS*) echo "windows" ;;
        *)
            log_error "Unsupported operating system: $os"
            exit 1
            ;;
    esac
}

# Detect architecture
detect_arch() {
    local arch
    arch="$(uname -m)"
    
    case "$arch" in
        x86_64|amd64)   echo "amd64" ;;
        arm64|aarch64)  echo "arm64" ;;
        armv7l)         echo "arm" ;;
        i386|i686)      
            log_error "32-bit systems are not supported"
            exit 1
            ;;
        *)
            log_error "Unsupported architecture: $arch"
            exit 1
            ;;
    esac
}

# Get latest version from GitHub API
get_latest_version() {
    local api_url="https://api.github.com/repos/${GITHUB_REPO}/releases/latest"
    
    if command -v curl &>/dev/null; then
        curl -fsSL "$api_url" 2>/dev/null | grep -o '"tag_name": "[^"]*' | grep -o '[^"]*$' | sed 's/^v//'
    elif command -v wget &>/dev/null; then
        wget -qO- "$api_url" 2>/dev/null | grep -o '"tag_name": "[^"]*' | grep -o '[^"]*$' | sed 's/^v//'
    else
        log_error "curl or wget is required"
        exit 1
    fi
}

# Download file
download_file() {
    local url="$1"
    local dest="$2"
    
    if command -v curl &>/dev/null; then
        curl -fsSL --progress-bar "$url" -o "$dest"
    elif command -v wget &>/dev/null; then
        wget --progress=bar:force -O "$dest" "$url" 2>&1
    else
        log_error "curl or wget is required"
        exit 1
    fi
}

# Verify checksum
verify_checksum() {
    local file="$1"
    local expected="$2"
    
    if [[ -z "$expected" ]]; then
        log_warn "No checksum provided, skipping verification"
        return 0
    fi
    
    local actual
    if command -v sha256sum &>/dev/null; then
        actual="$(sha256sum "$file" | awk '{print $1}')"
    elif command -v shasum &>/dev/null; then
        actual="$(shasum -a 256 "$file" | awk '{print $1}')"
    else
        log_warn "sha256sum or shasum not found, skipping verification"
        return 0
    fi
    
    if [[ "$actual" != "$expected" ]]; then
        log_error "Checksum verification failed!"
        log_error "  Expected: $expected"
        log_error "  Actual:   $actual"
        return 1
    fi
    
    return 0
}

# Create directory if needed
ensure_dir() {
    local dir="$1"
    
    if [[ ! -d "$dir" ]]; then
        log_info "Creating directory: $dir"
        mkdir -p "$dir"
    fi
}

# Check if directory is writable
is_writable() {
    local dir="$1"
    [[ -w "$dir" ]] 2>/dev/null || [[ -w "$(dirname "$dir")" ]] 2>/dev/null
}

# Main installation
main() {
    parse_args "$@"
    
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}  Cline CLI Installation${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""
    
    # Detect platform
    local os arch
    os="$(detect_os)"
    arch="$(detect_arch)"
    
    log_info "Detected platform: $os/$arch"
    
    # Get version
    if [[ "$VERSION" == "latest" ]]; then
        log_info "Detecting latest version..."
        VERSION="$(get_latest_version)"
        if [[ -z "$VERSION" ]]; then
            log_error "Failed to detect latest version"
            exit 1
        fi
    fi
    
    log_info "Installing Cline CLI v$VERSION"
    
    # Check if we need sudo
    if [[ "$USE_SUDO" == true ]] && ! is_writable "$INSTALL_DIR"; then
        if ! command -v sudo &>/dev/null; then
            log_error "sudo is required to install to $INSTALL_DIR"
            log_info "Use --no-sudo to install to $USER_INSTALL_DIR instead"
            exit 1
        fi
        log_info "Using sudo for installation to $INSTALL_DIR"
    fi
    
    # Set up paths
    local asset_name="cline-v${VERSION}-${os}-${arch}.tar.gz"
    local download_url="https://github.com/${GITHUB_REPO}/releases/download/v${VERSION}/${asset_name}"
    local temp_dir
    temp_dir="$(mktemp -d)"
    local archive_path="${temp_dir}/${asset_name}"
    local checksum_url="${download_url}.sha256"
    local checksum_path="${temp_dir}/${asset_name}.sha256"
    
    # Create install directory
    ensure_dir "$INSTALL_DIR"
    
    # Download archive
    log_info "Downloading $asset_name..."
    if ! download_file "$download_url" "$archive_path"; then
        log_error "Failed to download from $download_url"
        rm -rf "$temp_dir"
        exit 1
    fi
    log_success "Download complete"
    
    # Download checksum
    log_info "Downloading checksum..."
    local expected_checksum=""
    if download_file "$checksum_url" "$checksum_path" 2>/dev/null; then
        expected_checksum="$(cat "$checksum_path" | awk '{print $1}')"
        log_info "Expected checksum: $expected_checksum"
    else
        log_warn "Could not download checksum file"
    fi
    
    # Verify checksum
    if [[ -n "$expected_checksum" ]]; then
        log_info "Verifying checksum..."
        if ! verify_checksum "$archive_path" "$expected_checksum"; then
            rm -rf "$temp_dir"
            exit 1
        fi
        log_success "Checksum verified"
    fi
    
    # Extract archive
    log_info "Extracting archive..."
    if ! tar -xzf "$archive_path" -C "$temp_dir"; then
        log_error "Failed to extract archive"
        rm -rf "$temp_dir"
        exit 1
    fi
    
    # Find extracted binary
    local extracted_binary
    extracted_binary="$(find "$temp_dir" -name "$BINARY_NAME" -type f | head -1)"
    
    if [[ -z "$extracted_binary" ]]; then
        log_error "Binary not found in extracted archive"
        rm -rf "$temp_dir"
        exit 1
    fi
    
    # Install binary
    local dest_path="${INSTALL_DIR}/${BINARY_NAME}"
    
    log_info "Installing to $dest_path..."
    
    if [[ -f "$dest_path" ]]; then
        log_info "Removing existing installation..."
        if [[ "$USE_SUDO" == true ]] && ! is_writable "$INSTALL_DIR"; then
            sudo rm -f "$dest_path"
        else
            rm -f "$dest_path"
        fi
    fi
    
    if [[ "$USE_SUDO" == true ]] && ! is_writable "$INSTALL_DIR"; then
        sudo mv "$extracted_binary" "$dest_path"
        sudo chmod +x "$dest_path"
    else
        mv "$extracted_binary" "$dest_path"
        chmod +x "$dest_path"
    fi
    
    # Verify installation
    log_info "Verifying installation..."
    local installed_version
    installed_version="$("$dest_path" --version 2>&1)" || true
    
    if [[ -n "$installed_version" ]]; then
        log_success "Installation successful!"
        echo ""
        echo -e "${GREEN}Installed:${NC} $dest_path"
        echo -e "${GREEN}Version:${NC} $installed_version"
        echo ""
    else
        log_error "Installation verification failed"
        rm -rf "$temp_dir"
        exit 1
    fi
    
    # Check if in PATH
    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        log_warn "$INSTALL_DIR is not in your PATH"
        echo ""
        echo "Add the following to your shell configuration:"
        echo ""
        if [[ "$SHELL" == *"zsh"* ]]; then
            echo "  echo 'export PATH=\"\$PATH:$INSTALL_DIR\"' >> ~/.zshrc"
            echo "  source ~/.zshrc"
        elif [[ "$SHELL" == *"bash"* ]]; then
            echo "  echo 'export PATH=\"\$PATH:$INSTALL_DIR\"' >> ~/.bashrc"
            echo "  source ~/.bashrc"
        else
            echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
        fi
        echo ""
    fi
    
    # Cleanup
    rm -rf "$temp_dir"
    
    # Success message
    echo -e "${BLUE}========================================${NC}"
    echo -e "${GREEN}  Installation Complete!${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""
    echo "Get started with:"
    echo "  cline --help          Show help"
    echo "  cline --version       Show version"
    echo "  cline 'Hello world'   Run your first task"
    echo ""
    echo "Documentation: https://docs.cline.bot"
    echo ""
}

# Run main function
main "$@"