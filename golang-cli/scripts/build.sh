#!/bin/bash
# Cross-platform build script for Cline CLI
# Supports cross-compilation for multiple platforms, code signing, and release packaging

set -euo pipefail

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Build configuration
BINARY_NAME="cline"
BUILD_DIR="$PROJECT_ROOT/build"
DIST_DIR="$PROJECT_ROOT/dist"
VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')}"
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT="${GIT_COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')}"

# Go build flags
LDFLAGS="-s -w \
    -X github.com/cline/cline/golang-cli/cmd/cline.Version=$VERSION \
    -X github.com/cline/cline/golang-cli/cmd/cline.BuildTime=$BUILD_TIME \
    -X github.com/cline/cline/golang-cli/cmd/cline.GitCommit=$GIT_COMMIT"

# Supported platforms
PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Print usage information
usage() {
    cat << EOF
Usage: $(basename "$0") [OPTIONS]

Cross-platform build script for Cline CLI

Options:
    -h, --help              Show this help message
    -v, --version VERSION   Set build version (default: git tag or 'dev')
    -p, --platform PLATFORM Build for specific platform only (e.g., linux/amd64)
    -s, --sign              Sign macOS binaries (requires codesign certificate)
    -c, --checksum          Generate checksums for built binaries
    -a, --archive           Create release archives
    -A, --all               Enable signing, checksums, and archives
    -C, --clean             Clean build directory before building
    --codesign-identity ID  Specify codesign identity (default: from environment)

Environment Variables:
    VERSION                 Build version
    GIT_COMMIT              Git commit hash
    CODESIGN_IDENTITY       macOS codesign identity
    APPLE_DEVELOPER_ID      Apple Developer ID for notarization
    NOTARIZE_PASSWORD       App-specific password for notarization

Examples:
    $(basename "$0")                    # Build all platforms
    $(basename "$0") -p linux/amd64     # Build for Linux AMD64 only
    $(basename "$0") --all              # Full release build with signing
    $(basename "$0") -C -a              # Clean build with archives
EOF
}

# Parse command line arguments
SIGN=false
CHECKSUM=false
ARCHIVE=false
CLEAN=false
SPECIFIC_PLATFORM=""
CODESIGN_IDENTITY="${CODESIGN_IDENTITY:-}"

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            usage
            exit 0
            ;;
        -v|--version)
            VERSION="$2"
            shift 2
            ;;
        -p|--platform)
            SPECIFIC_PLATFORM="$2"
            shift 2
            ;;
        -s|--sign)
            SIGN=true
            shift
            ;;
        -c|--checksum)
            CHECKSUM=true
            shift
            ;;
        -a|--archive)
            ARCHIVE=true
            shift
            ;;
        -A|--all)
            SIGN=true
            CHECKSUM=true
            ARCHIVE=true
            shift
            ;;
        -C|--clean)
            CLEAN=true
            shift
            ;;
        --codesign-identity)
            CODESIGN_IDENTITY="$2"
            shift 2
            ;;
        *)
            log_error "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Validate environment
validate_environment() {
    log_info "Validating build environment..."

    if ! command -v go &> /dev/null; then
        log_error "Go is not installed or not in PATH"
        exit 1
    fi

    local go_version
    go_version=$(go version | awk '{print $3}' | sed 's/go//')
    log_info "Go version: $go_version"

    if [[ "$SIGN" == true ]]; then
        if ! command -v codesign &> /dev/null; then
            log_warn "codesign not found. macOS signing will be skipped."
            SIGN=false
        elif [[ -z "$CODESIGN_IDENTITY" ]]; then
            log_warn "CODESIGN_IDENTITY not set. Attempting to find default..."
            CODESIGN_IDENTITY=$(security find-identity -v -p codesigning 2>/dev/null | \
                grep "Developer ID Application" | \
                head -1 | \
                sed -n 's/.*"\(.*\)".*/\1/p')
            if [[ -n "$CODESIGN_IDENTITY" ]]; then
                log_info "Found codesign identity: $CODESIGN_IDENTITY"
            else
                log_warn "No codesign identity found. Signing will be skipped."
                SIGN=false
            fi
        fi
    fi

    if [[ "$CHECKSUM" == true ]]; then
        if ! command -v shasum &> /dev/null && ! command -v sha256sum &> /dev/null; then
            log_error "Neither shasum nor sha256sum found. Cannot generate checksums."
            exit 1
        fi
    fi

    log_success "Environment validation complete"
}

# Clean build directories
clean_build() {
    log_info "Cleaning build directories..."
    rm -rf "$BUILD_DIR" "$DIST_DIR"
    log_success "Build directories cleaned"
}

# Create build directories
create_directories() {
    mkdir -p "$BUILD_DIR" "$DIST_DIR"
}

# Get binary extension for platform
get_binary_extension() {
    local os=$1
    if [[ "$os" == "windows" ]]; then
        echo ".exe"
    else
        echo ""
    fi
}

# Build for a specific platform
build_platform() {
    local platform=$1
    local os=${platform%/*}
    local arch=${platform#*/}
    local extension
    extension=$(get_binary_extension "$os")
    local output_name="${BINARY_NAME}-${VERSION}-${os}-${arch}${extension}"
    local output_path="$BUILD_DIR/$output_name"

    log_info "Building for $os/$arch..."

    GOOS="$os" GOARCH="$arch" CGO_ENABLED=0 \
        go build \
        -ldflags "$LDFLAGS" \
        -trimpath \
        -o "$output_path" \
        "$PROJECT_ROOT/cmd/cline"

    if [[ ! -f "$output_path" ]]; then
        log_error "Build failed for $platform"
        return 1
    fi

    # Get binary size
    local size
    size=$(du -h "$output_path" | cut -f1)
    log_success "Built: $output_name ($size)"

    echo "$output_name"
}

# Sign macOS binary
sign_binary() {
    local binary_path=$1
    local platform=$2

    if [[ "$SIGN" != true ]]; then
        return 0
    fi

    local os=${platform%/*}
    if [[ "$os" != "darwin" ]]; then
        return 0
    fi

    if [[ -z "$CODESIGN_IDENTITY" ]]; then
        log_warn "Skipping signing for $binary_path (no identity)"
        return 0
    fi

    log_info "Signing $binary_path..."

    if codesign --sign "$CODESIGN_IDENTITY" \
        --force \
        --options runtime \
        --timestamp \
        --verbose \
        "$binary_path" 2>/dev/null; then
        log_success "Signed: $(basename "$binary_path")"

        # Verify signature
        if codesign --verify --verbose "$binary_path" 2>/dev/null; then
            log_success "Signature verified: $(basename "$binary_path")"
        else
            log_warn "Signature verification failed: $(basename "$binary_path")"
        fi
    else
        log_warn "Failed to sign: $(basename "$binary_path")"
    fi
}

# Generate checksum for a file
generate_checksum() {
    local file=$1
    local checksum_file="$file.sha256"

    if [[ "$CHECKSUM" != true ]]; then
        return 0
    fi

    log_info "Generating checksum for $(basename "$file")..."

    if command -v sha256sum &> /dev/null; then
        sha256sum "$file" | cut -d' ' -f1 > "$checksum_file"
    else
        shasum -a 256 "$file" | cut -d' ' -f1 > "$checksum_file"
    fi

    log_success "Checksum: $(basename "$checksum_file")"
}

# Create release archive
create_archive() {
    local binary_name=$1
    local platform=$2
    local binary_path="$BUILD_DIR/$binary_name"
    local os=${platform%/*}
    local arch=${platform#*/}

    if [[ "$ARCHIVE" != true ]]; then
        return 0
    fi

    log_info "Creating archive for $binary_name..."

    local archive_name="${BINARY_NAME}-${VERSION}-${os}-${arch}"
    local archive_path

    if [[ "$os" == "windows" ]]; then
        archive_path="$DIST_DIR/${archive_name}.zip"
        (cd "$BUILD_DIR" && zip -q "$archive_path" "$binary_name" "$binary_name.sha256")
    else
        archive_path="$DIST_DIR/${archive_name}.tar.gz"
        (cd "$BUILD_DIR" && tar -czf "$archive_path" "$binary_name" "$binary_name.sha256")
    fi

    local size
    size=$(du -h "$archive_path" | cut -f1)
    log_success "Archive: $(basename "$archive_path") ($size)"

    # Generate checksum for archive
    generate_checksum "$archive_path"
}

# Build all platforms
build_all() {
    log_info "Starting cross-platform build..."
    log_info "Version: $VERSION"
    log_info "Commit: $GIT_COMMIT"
    log_info "Build Time: $BUILD_TIME"

    local platforms_to_build=()
    if [[ -n "$SPECIFIC_PLATFORM" ]]; then
        platforms_to_build=("$SPECIFIC_PLATFORM")
    else
        platforms_to_build=("${PLATFORMS[@]}")
    fi

    local built_binaries=()

    for platform in "${platforms_to_build[@]}"; do
        local binary_name
        if binary_name=$(build_platform "$platform"); then
            built_binaries+=("$binary_name")
            local binary_path="$BUILD_DIR/$binary_name"

            # Sign if applicable
            sign_binary "$binary_path" "$platform"

            # Generate checksum
            generate_checksum "$binary_path"

            # Create archive
            create_archive "$binary_name" "$platform"
        fi
    done

    log_success "Build complete! Built ${#built_binaries[@]} binaries."

    if [[ "$CHECKSUM" == true ]]; then
        log_info "Generating checksums file..."
        local checksums_file="$DIST_DIR/${BINARY_NAME}-${VERSION}-checksums.txt"
        {
            echo "Checksums for Cline CLI $VERSION"
            echo "Generated: $BUILD_TIME"
            echo ""
            for binary in "${built_binaries[@]}"; do
                if [[ -f "$BUILD_DIR/$binary.sha256" ]]; then
                    local checksum
                    checksum=$(cat "$BUILD_DIR/$binary.sha256")
                    echo "$checksum  $binary"
                fi
            done
        } > "$checksums_file"
        log_success "Checksums file: $checksums_file"
    fi
}

# Generate version info file
generate_version_info() {
    local version_file="$DIST_DIR/version.json"
    log_info "Generating version info..."

    cat > "$version_file" << EOF
{
    "version": "$VERSION",
    "git_commit": "$GIT_COMMIT",
    "build_time": "$BUILD_TIME",
    "platforms": [
$(for p in "${PLATFORMS[@]}"; do echo "        \"$p\","; done | sed '$ s/,$//')
    ]
}
EOF
    log_success "Version info: $version_file"
}

# Main execution
main() {
    validate_environment

    if [[ "$CLEAN" == true ]]; then
        clean_build
    fi

    create_directories
    build_all

    if [[ "$ARCHIVE" == true ]]; then
        generate_version_info
    fi

    log_success "All tasks completed successfully!"
    log_info "Build artifacts available in: $BUILD_DIR"
    if [[ "$ARCHIVE" == true ]]; then
        log_info "Release archives available in: $DIST_DIR"
    fi
}

main "$@"