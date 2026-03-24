#!/bin/bash

# Generate Go gRPC code from proto definitions
# This script generates Go code for all proto files in proto/cline/

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
PROTO_DIR="$PROJECT_ROOT/proto/cline"
OUT_DIR="$PROJECT_ROOT/internal/generated/cline"

echo "Generating Go gRPC code from proto definitions..."
echo "Proto directory: $PROTO_DIR"
echo "Output directory: $OUT_DIR"

# Create output directory if it doesn't exist
mkdir -p "$OUT_DIR"

# Find all proto files
PROTO_FILES=$(find "$PROTO_DIR" -name "*.proto" | sort)

if [ -z "$PROTO_FILES" ]; then
    echo "Error: No proto files found in $PROTO_DIR"
    exit 1
fi

echo "Found proto files:"
echo "$PROTO_FILES"

# Generate Go code using protoc
# --go_out: Generate Go messages
# --go-grpc_out: Generate Go gRPC service interfaces
# paths=source_relative: Keep directory structure relative to proto files
protoc \
    --go_out=paths=source_relative:"$OUT_DIR" \
    --go-grpc_out=paths=source_relative:"$OUT_DIR" \
    -I "$PROJECT_ROOT/proto" \
    $PROTO_FILES

echo "Go gRPC code generation complete!"
echo "Generated files in: $OUT_DIR"