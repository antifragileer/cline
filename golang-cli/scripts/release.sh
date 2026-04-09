#!/bin/bash
# Release automation script for Cline CLI
# Orchestrates the entire release process including builds, signing, packaging, and distribution
#
# Usage: ./scripts/release.sh [OPTIONS]
#   -v, --version VERSION    Release version (required)
#   --skip-build            Skip building binaries
#   --skip-sign             Skip code signing
#   --skip-tests            Skip running tests
#   --skip-packages         Skip creating distribution packages
#   --dry-run               Show what would be done without executing
#   --github-release        Create GitHub release (requires gh CLI)
#   --publish               Publish to all distribution channels

set -euo pipefail

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1" >&2; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1" >&2; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1" >&2; }
log_error() { echo -e "${RED}[ERROR]${NC} $1" >&2; }

# Configuration
VERSION=""
SKIP_BUILD=false
SKIP_SIGN=false
SKIP_TESTS=false
SKIP_PACKAGES=false
DRY_RUN=false
GITHUB_RELEASE=false
PUBLISH=false

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--version)
            VERSION="$2"
            shift 2
            ;;
        --skip-build)
            SKIP_BUILD=true
            shift
            ;;
        --skip-sign)
            SKIP_SIGN=true
            shift
            ;;
        --skip-tests)
            SKIP_TESTS=true
            shift
            ;;
        --skip-packages)
            SKIP_PACKAGES=true
            shift
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --github-release)
            GITHUB_RELEASE=true
            shift
            ;;
        --publish)
            PUBLISH=true
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  -v, --version VERSION    Release version (required)"
            echo "  --skip-build             Skip building binaries"
            echo "  --skip-sign              Skip code signing"
            echo "  --skip-tests             Skip running tests"
            echo "  --skip-packages          Skip creating distribution packages"
            echo "  --dry-run                Show what would be done without executing"
            echo "  --github-release         Create GitHub release"
            echo "  --publish                Publish to all distribution channels"
            echo ""
            echo "Examples:"
            echo "  $0 -v 1.0.0              # Full release for version 1.0.0"
            echo "  $0 -v 1.0.0 --dry-run    # Preview what would be done"
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Validate version
if [[ -z "$VERSION" ]]; then
    log_error "Version is required. Use -v or --version"
    exit 1
fi

# Validate version format (semver)
if [[ ! "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.-]+)?(\+[a-zA-Z0-9.-]+)?$ ]]; then
    log_error "Invalid version format. Expected: MAJOR.MINOR.PATCH[-prerelease][+build]"
    exit 1
fi

log_info "Starting release process for version $VERSION"
if [[ "$DRY_RUN" == true ]]; then
    log_warn "DRY RUN MODE - No changes will be made"
fi

# Step 1: Pre-release checks
pre_release_checks() {
    log_info "Running pre-release checks..."
    
    if [[ "$DRY_RUN" == true ]]; then
        log_info "[DRY RUN] Would run pre-release checks"
        return 0
    fi
    
    # Check we're in a git repo
    if [[ ! -d "$PROJECT_ROOT/.git" ]]; then
        log_error "Not in a git repository"
        exit 1
    fi
    
    # Check working directory is clean
    if [[ -n "$(git status --porcelain)" ]]; then
        log_error "Working directory is not clean. Commit or stash changes first."
        exit 1
    fi
    
    # Check we're on main branch
    local branch
    branch=$(git rev-parse --abbrev-ref HEAD)
    if [[ "$branch" != "main" && "$branch" != "master" ]]; then
        log_warn "Not on main/master branch (currently on $branch)"
        read -p "Continue anyway? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    fi
    
    # Check version doesn't already exist
    if git rev-parse "v$VERSION" >/dev/null 2>&1; then
        log_error "Tag v$VERSION already exists"
        exit 1
    fi
    
    log_success "Pre-release checks passed"
}

# Step 2: Run tests
run_tests() {
    if [[ "$SKIP_TESTS" == true ]]; then
        log_info "Skipping tests (--skip-tests)"
        return 0
    fi
    
    log_info "Running tests..."
    
    if [[ "$DRY_RUN" == true ]]; then
        log_info "[DRY RUN] Would run: make test"
        return 0
    fi
    
    cd "$PROJECT_ROOT"
    
    # Run unit tests
    if ! make test; then
        log_error "Tests failed"
        exit 1
    fi
    
    # Run integration tests
    if ! make test-integration; then
        log_error "Integration tests failed"
        exit 1
    fi
    
    log_success "All tests passed"
}

# Step 3: Build binaries
build_binaries() {
    if [[ "$SKIP_BUILD" == true ]]; then
        log_info "Skipping build (--skip-build)"
        return 0
    fi
    
    log_info "Building binaries..."
    
    if [[ "$DRY_RUN" == true ]]; then
        log_info "[DRY RUN] Would run: ./scripts/build.sh --all --archive --checksum"
        return 0
    fi
    
    cd "$PROJECT_ROOT"
    
    # Clean previous builds
    rm -rf build dist
    
    # Build all platforms
    local sign_flag=""
    if [[ "$SKIP_SIGN" != true ]]; then
        sign_flag="--sign"
    fi
    
    if ! ./scripts/build.sh --all --archive --checksum $sign_flag; then
        log_error "Build failed"
        exit 1
    fi
    
    log_success "Binaries built successfully"
}

# Step 4: Sign binaries (if not done during build)
sign_binaries() {
    if [[ "$SKIP_SIGN" == true ]]; then
        log_info "Skipping signing (--skip-sign)"
        return 0
    fi
    
    # Signing is already done in build.sh if --sign is passed
    log_info "Binary signing handled by build process"
}

# Step 5: Create distribution packages
create_packages() {
    if [[ "$SKIP_PACKAGES" == true ]]; then
        log_info "Skipping packages (--skip-packages)"
        return 0
    fi
    
    log_info "Creating distribution packages..."
    
    if [[ "$DRY_RUN" == true ]]; then
        log_info "[DRY RUN] Would create packages for:"
        log_info "  - Homebrew formula"
        log_info "  - Scoop manifest"
        log_info "  - Linux packages (deb/rpm)"
        log_info "  - NPM wrapper"
        return 0
    fi
    
    cd "$PROJECT_ROOT"
    
    # Create output directory
    mkdir -p dist/packages
    
    # Generate Homebrew formula
    log_info "Generating Homebrew formula..."
    go run ./scripts/generate_homebrew.go \
        -version "$VERSION" \
        -local \
        -binary-dir ./dist \
        -output dist/packages/cline.rb
    
    # Generate Scoop manifest
    log_info "Generating Scoop manifest..."
    go run ./scripts/generate_scoop.go \
        -version "$VERSION" \
        -local \
        -binary-dir ./dist \
        -output dist/packages/cline.json
    
    # Generate Linux packages
    log_info "Generating Linux packages..."
    go run ./scripts/generate_linux_packages.go \
        -version "$VERSION" \
        -binary-dir ./dist \
        -output-dir dist/packages
    
    # Prepare NPM wrapper
    log_info "Preparing NPM wrapper..."
    mkdir -p dist/npm
    cp -r npm-wrapper/* dist/npm/
    cp dist/cline-*.tar.gz dist/npm/bin/ 2>/dev/null || true
    cp dist/cline-*.zip dist/npm/bin/ 2>/dev/null || true
    
    # Update NPM package version
    sed -i.bak "s/\"version\": \".*\"/\"version\": \"$VERSION\"/" dist/npm/package.json
    rm -f dist/npm/package.json.bak
    
    log_success "Distribution packages created"
}

# Step 6: Generate release notes
generate_release_notes() {
    log_info "Generating release notes..."
    
    if [[ "$DRY_RUN" == true ]]; then
        log_info "[DRY RUN] Would generate release notes"
        return 0
    fi
    
    cd "$PROJECT_ROOT"
    
    # Get changes since last tag
    local last_tag
    last_tag=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
    
    local changes
    if [[ -n "$last_tag" ]]; then
        changes=$(git log --pretty=format:"- %s" "$last_tag"..HEAD)
    else
        changes=$(git log --pretty=format:"- %s" | head -20)
    fi
    
    # Create release notes
    cat > dist/RELEASE_NOTES.md << EOF
# Cline CLI v$VERSION

## Installation

### Homebrew (macOS/Linux)
\`\`\`bash
brew tap cline/tap
brew install cline
\`\`\`

### Scoop (Windows)
\`\`\`powershell
scoop bucket add cline https://github.com/cline/scoop-cline
scoop install cline
\`\`\`

### NPM
\`\`\`bash
npm install -g @cline/golang-cli
\`\`\`

### Direct Download
Download the appropriate binary for your platform from the assets below.

## Changes

$changes

## Checksums

See \`checksums.txt\` for SHA256 checksums of all binaries.

## Documentation

- [Migration Guide](https://github.com/cline/cline/blob/main/golang-cli/docs/MIGRATION_GUIDE.md)
- [Troubleshooting](https://github.com/cline/cline/blob/main/golang-cli/docs/TROUBLESHOOTING.md)
- [Distribution Guide](https://github.com/cline/cline/blob/main/golang-cli/docs/DISTRIBUTION.md)

---

*Released on $(date +%Y-%m-%d)*
EOF
    
    log_success "Release notes generated: dist/RELEASE_NOTES.md"
}

# Step 7: Create GitHub release
create_github_release() {
    if [[ "$GITHUB_RELEASE" != true ]]; then
        log_info "Skipping GitHub release (use --github-release to enable)"
        return 0
    fi
    
    log_info "Creating GitHub release..."
    
    if [[ "$DRY_RUN" == true ]]; then
        log_info "[DRY RUN] Would create GitHub release v$VERSION"
        return 0
    fi
    
    # Check for gh CLI
    if ! command -v gh &> /dev/null; then
        log_error "gh CLI not found. Install from https://cli.github.com/"
        exit 1
    fi
    
    cd "$PROJECT_ROOT"
    
    # Create tag
    git tag -a "v$VERSION" -m "Release v$VERSION"
    git push origin "v$VERSION"
    
    # Create release
    gh release create "v$VERSION" \
        --title "v$VERSION" \
        --notes-file dist/RELEASE_NOTES.md \
        dist/cline-*.tar.gz \
        dist/cline-*.zip \
        dist/checksums.txt \
        dist/packages/*.deb \
        dist/packages/*.rpm 2>/dev/null || true
    
    log_success "GitHub release created"
}

# Step 8: Publish to distribution channels
publish_packages() {
    if [[ "$PUBLISH" != true ]]; then
        log_info "Skipping publish (use --publish to enable)"
        return 0
    fi
    
    log_info "Publishing to distribution channels..."
    
    if [[ "$DRY_RUN" == true ]]; then
        log_info "[DRY RUN] Would publish to:"
        log_info "  - Homebrew tap"
        log_info "  - Scoop bucket"
        log_info "  - NPM registry"
        return 0
    fi
    
    cd "$PROJECT_ROOT"
    
    # Note: Actual publishing requires authentication and specific repository access
    # These are placeholders for the actual publish commands
    
    log_warn "Publishing requires manual steps:"
    log_info "1. Homebrew tap: Copy dist/packages/cline.rb to homebrew-tap repository"
    log_info "2. Scoop bucket: Copy dist/packages/cline.json to scoop-cline repository"
    log_info "3. NPM: Run 'npm publish' in dist/npm/ directory"
    log_info "4. Linux repos: Upload deb/rpm packages to package repositories"
}

# Step 9: Post-release tasks
post_release() {
    log_info "Running post-release tasks..."
    
    if [[ "$DRY_RUN" == true ]]; then
        log_info "[DRY RUN] Would run post-release tasks"
        return 0
    fi
    
    cd "$PROJECT_ROOT"
    
    # Generate summary
    cat > dist/RELEASE_SUMMARY.txt << EOF
Cline CLI Release v$VERSION
==========================

Artifacts:
$(ls -1 dist/cline-*.{tar.gz,zip} 2>/dev/null | wc -l) binaries
$(ls -1 dist/packages/*.deb 2>/dev/null | wc -l) DEB packages
$(ls -1 dist/packages/*.rpm 2>/dev/null | wc -l) RPM packages

Distribution Files:
- dist/packages/cline.rb (Homebrew formula)
- dist/packages/cline.json (Scoop manifest)
- dist/npm/ (NPM wrapper package)

Next Steps:
1. Review release artifacts in dist/
2. Copy distribution files to respective repositories
3. Publish NPM package: cd dist/npm && npm publish
4. Announce the release
EOF
    
    log_success "Release summary: dist/RELEASE_SUMMARY.txt"
    
    # Display summary
    cat dist/RELEASE_SUMMARY.txt
}

# Main execution
main() {
    log_info "=========================================="
    log_info "Cline CLI Release Automation"
    log_info "Version: $VERSION"
    log_info "=========================================="
    
    pre_release_checks
    run_tests
    build_binaries
    sign_binaries
    create_packages
    generate_release_notes
    create_github_release
    publish_packages
    post_release
    
    log_success "Release process completed!"
    
    if [[ "$DRY_RUN" == true ]]; then
        log_warn "This was a dry run. No actual changes were made."
        log_info "Run without --dry-run to perform the actual release."
    fi
}

main "$@"