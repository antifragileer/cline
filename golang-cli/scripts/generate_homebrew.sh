#!/bin/bash
# Generate Homebrew formula for Cline CLI
# This script wraps the Go-based formula generator for convenience
#
# Usage:
#   ./generate_homebrew.sh -version 1.0.0 -output Formula/cline.rb
#   ./generate_homebrew.sh -version 1.0.0 -local -binary-dir ./dist
#   ./generate_homebrew.sh -version 1.0.0 -tap-dir ./homebrew-tap

set -e

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Build the generator if needed
build_generator() {
    local generator_path="$SCRIPT_DIR/generate_homebrew"

    if [[ ! -f "$generator_path" ]] || [[ "$SCRIPT_DIR/generate_homebrew.go" -nt "$generator_path" ]]; then
        echo "Building Homebrew formula generator..."
        (cd "$SCRIPT_DIR" && go build -o generate_homebrew generate_homebrew.go homebrew.go)
    fi

    echo "$generator_path"
}

# Show usage information
show_usage() {
    cat <<EOF
Usage: $(basename "$0") [options]

Generate Homebrew formula for Cline CLI distribution.

Options:
  -version string       Release version (required)
  -output string        Output path for formula file
  -base-url string      Base URL for release downloads (default: https://github.com/cline/cline/releases/download)
  -binary-dir string    Directory containing pre-built binaries for local SHA256 calculation
  -name string          Formula name (default: cline)
  -tap-dir string       Create tap structure in this directory
  -local                Use local files for SHA256 calculation
  -dry-run              Print formula without writing to file
  -h, -help             Show this help message

Examples:
  # Generate formula for a release
  $(basename "$0") -version 1.0.0 -output Formula/cline.rb

  # Generate formula using local binaries
  $(basename "$0") -version 1.0.0 -local -binary-dir ./dist -output Formula/cline.rb

  # Create complete tap structure
  $(basename "$0") -version 1.0.0 -tap-dir ./homebrew-tap

  # Dry run to preview formula
  $(basename "$0") -version 1.0.0 -dry-run
EOF
}

# Main function
main() {
    # Check if help is requested
    for arg in "$@"; do
        case "$arg" in
            -h|-help|--help)
                show_usage
                exit 0
                ;;
        esac
    done

    # Build and get generator path
    GENERATOR=$(build_generator)

    # Run the generator with all passed arguments
    exec "$GENERATOR" "$@"
}

# Run main function
main "$@"