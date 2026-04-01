#!/bin/bash
# Performance profiling script for Cline Go CLI
# Usage: ./scripts/profile.sh [cpu|mem|all]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BINARY="$PROJECT_DIR/cline"
PROFILE_DIR="$PROJECT_DIR/profiles"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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

# Ensure binary exists
ensure_binary() {
    if [[ ! -f "$BINARY" ]]; then
        log_info "Building binary..."
        cd "$PROJECT_DIR"
        go build -o cline ./cmd/cline
        log_success "Binary built successfully"
    fi
}

# Create profile directory
setup_profile_dir() {
    mkdir -p "$PROFILE_DIR"
    log_info "Profile directory: $PROFILE_DIR"
}

# Profile CPU usage
profile_cpu() {
    log_info "Profiling CPU usage..."
    
    local profile_file="$PROFILE_DIR/cpu_${TIMESTAMP}.prof"
    local trace_file="$PROFILE_DIR/trace_${TIMESTAMP}.out"
    
    # Run with CPU profiling
    log_info "Running CPU profile (30 seconds)..."
    "$BINARY" version --json > /dev/null 2>&1 &
    local pid=$!
    
    # Wait a moment for startup
    sleep 0.5
    
    # Capture CPU profile
    go tool pprof -raw -output="$profile_file" "http://localhost:6060/debug/pprof/profile?seconds=10" 2>/dev/null || {
        # Fallback: use time command for basic timing
        log_warn "Could not capture detailed CPU profile, using basic timing"
    }
    
    wait $pid 2>/dev/null || true
    
    # Use hyperfine for accurate timing if available
    if command -v hyperfine &> /dev/null; then
        log_info "Running hyperfine benchmark..."
        hyperfine --warmup 3 --runs 10 \
            --export-json "$PROFILE_DIR/hyperfine_${TIMESTAMP}.json" \
            "$BINARY version --short" \
            2>&1 | tee "$PROFILE_DIR/hyperfine_${TIMESTAMP}.txt"
    else
        log_warn "hyperfine not installed. Install with: brew install hyperfine"
        
        # Fallback to basic timing
        log_info "Running basic timing (10 iterations)..."
        for i in {1..10}; do
            { time "$BINARY" version --short; } 2>&1 | grep real
        done | tee "$PROFILE_DIR/timing_${TIMESTAMP}.txt"
    fi
    
    log_success "CPU profiling complete"
    log_info "Results saved to: $PROFILE_DIR/"
}

# Profile memory usage
profile_memory() {
    log_info "Profiling memory usage..."
    
    local mem_file="$PROFILE_DIR/mem_${TIMESTAMP}.prof"
    
    # Check if we can use /usr/bin/time
    if [[ -f /usr/bin/time ]]; then
        log_info "Capturing memory profile..."
        /usr/bin/time -v "$BINARY" version --json > /dev/null 2> "$PROFILE_DIR/mem_stats_${TIMESTAMP}.txt"
        log_success "Memory statistics captured"
    else
        log_warn "/usr/bin/time not available, using basic memory check"
    fi
    
    # Check if we can use ps for memory monitoring
    log_info "Monitoring memory during execution..."
    
    # Start process in background
    "$BINARY" version --json > /dev/null 2>&1 &
    local pid=$!
    
    # Monitor memory
    local max_mem=0
    while kill -0 $pid 2>/dev/null; do
        local mem=$(ps -o rss= -p $pid 2>/dev/null || echo 0)
        if [[ "$mem" -gt "$max_mem" ]]; then
            max_mem=$mem
        fi
        sleep 0.01
    done
    
    wait $pid 2>/dev/null || true
    
    # Convert KB to MB
    local max_mem_mb=$((max_mem / 1024))
    
    echo "Peak Memory Usage: ${max_mem_mb} MB" > "$PROFILE_DIR/mem_peak_${TIMESTAMP}.txt"
    log_success "Peak memory: ${max_mem_mb} MB"
    
    # Memory benchmark
    log_info "Running memory benchmark..."
    go test -bench=BenchmarkMemory -benchmem -run=^$ ./tests/... 2>/dev/null || \
        log_warn "Memory benchmarks not available"
    
    log_success "Memory profiling complete"
}

# Profile binary size
profile_binary() {
    log_info "Profiling binary size..."
    
    local size_file="$PROFILE_DIR/binary_size_${TIMESTAMP}.txt"
    
    # Get binary size
    local size_bytes=$(stat -f%z "$BINARY" 2>/dev/null || stat -c%s "$BINARY" 2>/dev/null || echo 0)
    local size_mb=$(echo "scale=2; $size_bytes / 1024 / 1024" | bc 2>/dev/null || echo "N/A")
    
    echo "Binary: $BINARY" > "$size_file"
    echo "Size: $size_bytes bytes (${size_mb} MB)" >> "$size_file"
    
    # Check sections
    if command -v size &> /dev/null; then
        echo "" >> "$size_file"
        echo "Section sizes:" >> "$size_file"
        size "$BINARY" >> "$size_file" 2>/dev/null || true
    fi
    
    # Check with go tool
    if command -v go &> /dev/null; then
        echo "" >> "$size_file"
        echo "Build info:" >> "$size_file"
        go version -m "$BINARY" >> "$size_file" 2>/dev/null || true
    fi
    
    log_success "Binary size: ${size_mb} MB"
    cat "$size_file"
}

# Profile startup time
profile_startup() {
    log_info "Profiling startup time..."
    
    local startup_file="$PROFILE_DIR/startup_${TIMESTAMP}.txt"
    
    # Cold start (first run)
    log_info "Measuring cold start..."
    sync && echo 3 | sudo tee /proc/sys/vm/drop_caches > /dev/null 2>&1 || true
    
    local cold_start=$( (time "$BINARY" version --short) 2>&1 | grep real | awk '{print $2}' )
    echo "Cold start: $cold_start" > "$startup_file"
    
    # Warm start (cached)
    log_info "Measuring warm start..."
    local warm_times=()
    for i in {1..10}; do
        local t=$( (time "$BINARY" version --short) 2>&1 | grep real | awk '{print $2}' | sed 's/0m//' | sed 's/s//' )
        warm_times+=($t)
    done
    
    # Calculate average
    local sum=0
    for t in "${warm_times[@]}"; do
        sum=$(echo "$sum + $t" | bc 2>/dev/null || echo "$sum")
    done
    local avg=$(echo "scale=3; $sum / 10" | bc 2>/dev/null || echo "N/A")
    
    echo "Warm start (average of 10): ${avg}s" >> "$startup_file"
    echo "Individual times: ${warm_times[*]}" >> "$startup_file"
    
    log_success "Startup profiling complete"
    cat "$startup_file"
}

# Run comparison with TypeScript CLI
compare_with_ts() {
    log_info "Comparing with TypeScript CLI..."
    
    local ts_binary=$(which cline-ts 2>/dev/null || which cline_node 2>/dev/null || echo "")
    
    if [[ -z "$ts_binary" ]]; then
        log_warn "TypeScript CLI not found for comparison"
        return
    fi
    
    local compare_file="$PROFILE_DIR/comparison_${TIMESTAMP}.txt"
    
    echo "Comparison: Go CLI vs TypeScript CLI" > "$compare_file"
    echo "" >> "$compare_file"
    
    # Startup comparison
    echo "=== Startup Time ===" >> "$compare_file"
    echo "Go CLI:" >> "$compare_file"
    { time "$BINARY" version --short; } 2>&1 | grep real >> "$compare_file"
    
    echo "TypeScript CLI:" >> "$compare_file"
    { time "$ts_binary" version --short; } 2>&1 | grep real >> "$compare_file" || true
    
    log_success "Comparison complete"
    cat "$compare_file"
}

# Generate report
generate_report() {
    log_info "Generating performance report..."
    
    local report_file="$PROFILE_DIR/report_${TIMESTAMP}.md"
    
    cat > "$report_file" << EOF
# Cline Go CLI Performance Report

Generated: $(date)

## System Information

- OS: $(uname -s)
- Arch: $(uname -m)
- Go Version: $(go version 2>/dev/null || echo "N/A")

## Binary Information

- Path: $BINARY
- Size: $(ls -lh "$BINARY" | awk '{print $5}')

## Performance Metrics

### Startup Time

$(cat "$PROFILE_DIR/startup_${TIMESTAMP}.txt" 2>/dev/null || echo "N/A")

### Memory Usage

$(cat "$PROFILE_DIR/mem_peak_${TIMESTAMP}.txt" 2>/dev/null || echo "N/A")

### Binary Size

$(cat "$PROFILE_DIR/binary_size_${TIMESTAMP}.txt" 2>/dev/null || echo "N/A")

## Recommendations

$(if command -v bc &> /dev/null; then
    size_mb=$(stat -f%z "$BINARY" 2>/dev/null | awk '{print $1/1024/1024}' || stat -c%s "$BINARY" 2>/dev/null | awk '{print $1/1024/1024}')
    if (( $(echo "$size_mb > 50" | bc -l) )); then
        echo "- Binary size is large (>50MB). Consider stripping debug symbols."
    else
        echo "- Binary size is within acceptable range."
    fi
fi)

## Profile Files

$(ls -la "$PROFILE_DIR"/*_"${TIMESTAMP}".* 2>/dev/null | awk '{print "- " $9 " (" $5 " bytes)"}')

EOF
    
    log_success "Report generated: $report_file"
}

# Main function
main() {
    local profile_type="${1:-all}"
    
    log_info "Cline Go CLI Performance Profiler"
    log_info "Profile type: $profile_type"
    
    ensure_binary
    setup_profile_dir
    
    case "$profile_type" in
        cpu)
            profile_cpu
            ;;
        mem|memory)
            profile_memory
            ;;
        startup)
            profile_startup
            ;;
        binary|size)
            profile_binary
            ;;
        compare)
            compare_with_ts
            ;;
        all)
            profile_binary
            profile_startup
            profile_memory
            profile_cpu
            compare_with_ts
            generate_report
            ;;
        *)
            echo "Usage: $0 [cpu|mem|startup|binary|compare|all]"
            echo ""
            echo "Commands:"
            echo "  cpu      - Profile CPU usage"
            echo "  mem      - Profile memory usage"
            echo "  startup  - Profile startup time"
            echo "  binary   - Profile binary size"
            echo "  compare  - Compare with TypeScript CLI"
            echo "  all      - Run all profiles (default)"
            exit 1
            ;;
    esac
    
    log_success "Profiling complete!"
    log_info "Results saved to: $PROFILE_DIR/"
}

main "$@"