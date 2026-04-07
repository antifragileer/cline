#!/bin/bash
#
# Cline Process Monitor - Detect and prevent process leaks
#
# This script monitors for runaway cline processes and provides
# cleanup utilities to prevent system resource exhaustion.
#
# Usage: ./cline-process-monitor.sh [command]
#
# Commands:
#   status      Show current cline process count and details
#   check       Check if process counts exceed thresholds (exit 1 if critical)
#   kill-zombies  Kill zombie/defunct cline processes
#   kill-all    Kill all cline processes (USE WITH CAUTION)
#   watch       Continuously monitor (useful in CI/CD)
#

set -euo pipefail

# Thresholds
readonly WARNING_THRESHOLD=5
readonly CRITICAL_THRESHOLD=10
readonly MAX_RUNTIME_SECONDS=1800  # 30 minutes max for any single process

# Colors
readonly RED='\033[0;31m'
readonly YELLOW='\033[1;33m'
readonly GREEN='\033[0;32m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m'

# Logging
log_info() {
    printf "${BLUE}[INFO]${NC} %s\n" "$1"
}

log_warn() {
    printf "${YELLOW}[WARN]${NC} %s\n" "$1" >&2
}

log_error() {
    printf "${RED}[ERROR]${NC} %s\n" "$1" >&2
}

log_success() {
    printf "${GREEN}[SUCCESS]${NC} %s\n" "$1"
}

# Get all cline-related processes
get_cline_processes() {
    # Get processes matching cline (excluding this script and grep)
    ps aux | grep -E "(cline|parity\.test)" | grep -v grep | grep -v "cline-process-monitor" || true
}

# Count cline processes
count_cline_processes() {
    get_cline_processes | wc -l | tr -d ' '
}

# Get process details in JSON format
get_process_details() {
    local count=0
    local old_ifs="$IFS"
    
    get_cline_processes | while IFS= read -r line; do
        if [[ -n "$line" ]]; then
            count=$((count + 1))
            # Extract PID, runtime, command
            local pid=$(echo "$line" | awk '{print $2}')
            local cpu=$(echo "$line" | awk '{print $3}')
            local mem=$(echo "$line" | awk '{print $4}')
            local start_time=$(echo "$line" | awk '{print $9}')
            local cmd=$(echo "$line" | awk '{print $11}' | xargs basename 2>/dev/null || echo "unknown")
            local full_cmd=$(echo "$line" | cut -d' ' -f11-)
            
            # Calculate runtime if possible
            local runtime="unknown"
            if [[ -f "/proc/$pid/stat" ]]; then
                # Get process start time in jiffies
                local start_jiffies=$(cat /proc/$pid/stat 2>/dev/null | awk '{print $22}' || echo "0")
                if [[ "$start_jiffies" != "0" && -f /proc/uptime ]]; then
                    local uptime_jiffies=$(cat /proc/uptime | awk '{print $1 * 100}' | cut -d. -f1)
                    local runtime_jiffies=$((uptime_jiffies - start_jiffies))
                    local runtime_secs=$((runtime_jiffies / 100))
                    if [[ $runtime_secs -gt 3600 ]]; then
                        runtime="$((runtime_secs / 3600))h $(((runtime_secs % 3600) / 60))m"
                    elif [[ $runtime_secs -gt 60 ]]; then
                        runtime="$((runtime_secs / 60))m $((runtime_secs % 60))s"
                    else
                        runtime="${runtime_secs}s"
                    fi
                fi
            fi
            
            echo "  Process $count:"
            echo "    PID: $pid"
            echo "    CPU: ${cpu}% | MEM: ${mem}%"
            echo "    Runtime: $runtime"
            echo "    Command: $cmd"
            echo "    Full: ${full_cmd:0:80}..."
            echo ""
        fi
    done
    
    IFS="$old_ifs"
}

# Get zombie processes
get_zombie_processes() {
    ps aux | grep -E "cline.*<defunct>|defunct.*cline|parity\.test.*<defunct>" | grep -v grep || true
}

# Kill zombie processes
kill_zombies() {
    log_info "Looking for zombie/defunct cline processes..."
    
    local zombies=$(get_zombie_processes)
    local count=$(echo "$zombies" | grep -c . || echo "0")
    
    if [[ "$count" -eq 0 ]]; then
        log_success "No zombie processes found"
        return 0
    fi
    
    log_warn "Found $count zombie process(es)"
    echo "$zombies"
    
    # Try to kill parent processes of zombies
    echo "$zombies" | while IFS= read -r line; do
        if [[ -n "$line" ]]; then
            local ppid=$(echo "$line" | awk '{print $3}')
            if [[ -n "$ppid" && "$ppid" != "1" ]]; then
                log_info "Killing parent process $ppid of zombie"
                kill -TERM "$ppid" 2>/dev/null || true
                sleep 1
                kill -KILL "$ppid" 2>/dev/null || true
            fi
        fi
    done
    
    log_success "Zombie cleanup attempted"
}

# Kill long-running processes
kill_long_running() {
    log_info "Checking for long-running cline processes (> ${MAX_RUNTIME_SECONDS}s)..."
    
    local killed=0
    get_cline_processes | while IFS= read -r line; do
        if [[ -n "$line" ]]; then
            local pid=$(echo "$line" | awk '{print $2}')
            local elapsed=$(ps -o etimes= -p "$pid" 2>/dev/null || echo "0")
            
            if [[ "$elapsed" -gt "$MAX_RUNTIME_SECONDS" ]]; then
                log_warn "Killing long-running process $pid (${elapsed}s)"
                kill -TERM "$pid" 2>/dev/null || true
                sleep 2
                kill -KILL "$pid" 2>/dev/null || true
                killed=$((killed + 1))
            fi
        fi
    done
    
    if [[ $killed -gt 0 ]]; then
        log_success "Killed $killed long-running process(es)"
    else
        log_success "No long-running processes found"
    fi
}

# Kill all cline processes
kill_all_cline() {
    log_warn "WARNING: This will kill ALL cline processes!"
    read -p "Are you sure? (type 'yes' to confirm): " confirm
    
    if [[ "$confirm" != "yes" ]]; then
        log_info "Aborted"
        return 1
    fi
    
    log_info "Killing all cline processes..."
    
    # Try graceful termination first
    pkill -TERM -f "cline" 2>/dev/null || true
    sleep 2
    
    # Force kill remaining
    pkill -KILL -f "cline" 2>/dev/null || true
    pkill -KILL -f "parity\.test" 2>/dev/null || true
    
    sleep 1
    local remaining=$(count_cline_processes)
    
    if [[ "$remaining" -eq 0 ]]; then
        log_success "All cline processes terminated"
    else
        log_error "$remaining process(es) could not be killed"
        get_cline_processes
        return 1
    fi
}

# Show status
show_status() {
    local count=$(count_cline_processes)
    local zombies
    zombies=$(get_zombie_processes | grep -c . 2>/dev/null || echo "0")
    zombies=$(echo "$zombies" | tr -d '\n' | tr -d ' ')
    
    echo "========================================"
    echo "Cline Process Status"
    echo "========================================"
    echo "Active processes: $count"
    echo "Zombie processes: $zombies"
    echo ""
    
    if [[ "$count" -gt "$CRITICAL_THRESHOLD" ]]; then
        log_error "CRITICAL: Process count ($count) exceeds threshold ($CRITICAL_THRESHOLD)"
    elif [[ "$count" -gt "$WARNING_THRESHOLD" ]]; then
        log_warn "WARNING: Process count ($count) exceeds threshold ($WARNING_THRESHOLD)"
    else
        log_success "Process count is normal ($count/$WARNING_THRESHOLD)"
    fi
    
    if [[ "$count" -gt 0 ]]; then
        echo ""
        echo "Process Details:"
        echo "----------------------------------------"
        get_process_details
    fi
    
    if [[ "$zombies" -gt 0 ]]; then
        echo ""
        log_warn "Zombie processes detected:"
        get_zombie_processes
    fi
}

# Check thresholds (for CI/CD)
check_thresholds() {
    local count=$(count_cline_processes)
    local zombies
    zombies=$(get_zombie_processes | grep -c . 2>/dev/null || echo "0")
    zombies=$(echo "$zombies" | tr -d '\n' | tr -d ' ')
    
    local exit_code=0
    
    if [[ "$count" -gt "$CRITICAL_THRESHOLD" ]]; then
        log_error "CRITICAL: $count cline processes running (threshold: $CRITICAL_THRESHOLD)"
        exit_code=1
    elif [[ "$count" -gt "$WARNING_THRESHOLD" ]]; then
        log_warn "WARNING: $count cline processes running (threshold: $WARNING_THRESHOLD)"
    fi
    
    if [[ "$zombies" -gt 0 ]]; then
        log_error "CRITICAL: $zombies zombie process(es) detected"
        exit_code=1
    fi
    
    # Check for long-running processes
    get_cline_processes | while IFS= read -r line; do
        if [[ -n "$line" ]]; then
            local pid=$(echo "$line" | awk '{print $2}')
            local elapsed=$(ps -o etimes= -p "$pid" 2>/dev/null || echo "0")
            
            if [[ "$elapsed" -gt "$MAX_RUNTIME_SECONDS" ]]; then
                log_warn "Long-running process detected: PID $pid (${elapsed}s)"
            fi
        fi
    done
    
    return $exit_code
}

# Watch mode
watch_mode() {
    local interval=${1:-30}
    
    log_info "Starting watch mode (interval: ${interval}s)"
    log_info "Press Ctrl+C to stop"
    
    while true; do
        clear
        show_status
        echo ""
        echo "Last updated: $(date)"
        echo "Press Ctrl+C to stop watching"
        sleep "$interval"
    done
}

# Main command handler
main() {
    local command=${1:-status}
    
    case "$command" in
        status)
            show_status
            ;;
        check)
            check_thresholds
            exit $?
            ;;
        kill-zombies|kill-zombie|zombies)
            kill_zombies
            ;;
        kill-long-running|long-running)
            kill_long_running
            ;;
        kill-all|killall)
            kill_all_cline
            ;;
        watch)
            watch_mode "${2:-30}"
            ;;
        help|--help|-h)
            cat << EOF
Cline Process Monitor

Usage: $(basename "$0") [command]

Commands:
  status              Show current process count and details (default)
  check               Check thresholds (exit 1 if critical - for CI/CD)
  kill-zombies        Kill zombie/defunct processes
  kill-long-running   Kill processes running > ${MAX_RUNTIME_SECONDS}s
  kill-all            Kill ALL cline processes (USE WITH CAUTION)
  watch [interval]    Continuously monitor (default 30s interval)
  help                Show this help message

Thresholds:
  Warning:  > $WARNING_THRESHOLD processes
  Critical: > $CRITICAL_THRESHOLD processes
  Max Runtime: ${MAX_RUNTIME_SECONDS}s per process

Examples:
  # Check status
  $(basename "$0") status

  # Monitor in watch mode (10s intervals)
  $(basename "$0") watch 10

  # Clean up zombies and long-running processes
  $(basename "$0") kill-zombies
  $(basename "$0") kill-long-running

  # Use in CI/CD pipeline
  $(basename "$0") check || exit 1
EOF
            ;;
        *)
            log_error "Unknown command: $command"
            log_info "Run '$(basename "$0") help' for usage"
            exit 1
            ;;
    esac
}

main "$@"