#!/bin/bash

# Benchmark comparison between goli (Go) and ratatui (Rust)
#
# Prerequisites:
#   - Go 1.21+
#   - Rust/Cargo (https://rustup.rs)
#
# Usage:
#   ./run.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Check for required tools
if ! command -v cargo &> /dev/null; then
    echo "Error: cargo not found. Please install Rust: https://rustup.rs"
    exit 1
fi

if ! command -v gox &> /dev/null; then
    echo "Error: gox not found. Please install gox."
    exit 1
fi

GO_VERSION=$(go version | cut -d' ' -f3)
RUST_VERSION=$(rustc --version | cut -d' ' -f2)

echo "Building and running benchmarks..."
echo

# Build goli benchmark
echo "Building goli benchmark..."
cd goli-bench
gox build -o ../goli-bench-bin . 2>/dev/null
cd ..

# Get goli binary size
GOLI_SIZE=$(ls -l goli-bench-bin | awk '{print $5}')
GOLI_SIZE_MB=$(echo "scale=2; $GOLI_SIZE / 1024 / 1024" | bc)

# Build ratatui benchmark (release mode for fair comparison)
echo "Building ratatui benchmark..."
cd ratatui-bench
cargo build --release --quiet 2>/dev/null
cd ..

# Get ratatui binary size
RATATUI_BIN="ratatui-bench/target/release/ratatui-bench"
RATATUI_SIZE=$(ls -l "$RATATUI_BIN" | awk '{print $5}')
RATATUI_SIZE_MB=$(echo "scale=2; $RATATUI_SIZE / 1024 / 1024" | bc)

# Run goli benchmark and capture output
echo "Running goli benchmark..."
GOLI_OUTPUT=$(./goli-bench-bin 2>/dev/null)

# Run ratatui benchmark and capture output
echo "Running ratatui benchmark..."
RATATUI_OUTPUT=$(./$RATATUI_BIN 2>/dev/null)

# Cleanup
rm -f goli-bench-bin

# Parse results
parse_value() {
    echo "$1" | grep -i "$2" | head -1 | sed 's/.*: *//' | sed 's/ .*//'
}

parse_updates() {
    echo "$1" | grep -i "updates/sec" | sed 's/.*(\([0-9]*\) updates.*/\1/'
}

parse_fps() {
    # For goli, prefer memo version if available
    local memo_fps=$(echo "$1" | grep -i "Max FPS (memo)" | sed 's/.*: *//' | sed 's/ .*//')
    if [ -n "$memo_fps" ]; then
        echo "$memo_fps"
    else
        echo "$1" | grep -i "Max FPS" | head -1 | sed 's/.*: *//' | sed 's/ .*//'
    fi
}

parse_large_fps() {
    # For goli, prefer memo version if available
    local memo_fps=$(echo "$1" | grep -i "Large screen FPS (memo)" | sed 's/.*: *//' | sed 's/ .*//')
    if [ -n "$memo_fps" ]; then
        echo "$memo_fps"
    else
        echo "$1" | grep -i "Large screen FPS" | head -1 | sed 's/.*: *//' | sed 's/ .*//'
    fi
}

GOLI_STARTUP=$(parse_value "$GOLI_OUTPUT" "Startup time")
GOLI_MEMORY=$(parse_value "$GOLI_OUTPUT" "Memory used")
GOLI_CPU=$(parse_value "$GOLI_OUTPUT" "Idle CPU")
GOLI_UPDATES=$(parse_updates "$GOLI_OUTPUT")
GOLI_FPS=$(parse_fps "$GOLI_OUTPUT")
GOLI_LARGE_FPS=$(parse_large_fps "$GOLI_OUTPUT")

RATATUI_STARTUP=$(parse_value "$RATATUI_OUTPUT" "Startup time")
RATATUI_MEMORY=$(parse_value "$RATATUI_OUTPUT" "Memory used")
RATATUI_CPU=$(parse_value "$RATATUI_OUTPUT" "Idle CPU")
RATATUI_UPDATES=$(parse_updates "$RATATUI_OUTPUT")
RATATUI_FPS=$(parse_fps "$RATATUI_OUTPUT")
RATATUI_LARGE_FPS=$(parse_large_fps "$RATATUI_OUTPUT")

# Calculate ratios (handle both directions)
calc_ratio() {
    local a=$1
    local b=$2
    if (( $(echo "$a > $b" | bc -l) )); then
        echo "scale=1; $a / $b" | bc
    else
        echo "scale=1; $b / $a" | bc
    fi
}

# Print summary
echo
echo "════════════════════════════════════════════════════════════"
echo "              goli vs ratatui Benchmark Results              "
echo "════════════════════════════════════════════════════════════"
echo
echo "Environment:"
echo "  Go:   $GO_VERSION"
echo "  Rust: $RUST_VERSION"
echo
printf "%-20s %15s %15s\n" "" "goli" "ratatui"
echo "  ─────────────────────────────────────────────────────────"
printf "%-20s %14s MB %14s MB\n" "  Binary size:" "$GOLI_SIZE_MB" "$RATATUI_SIZE_MB"
printf "%-20s %15s %15s\n" "  Startup:" "$GOLI_STARTUP" "$RATATUI_STARTUP"
printf "%-20s %13s MB %13s MB\n" "  Memory:" "$GOLI_MEMORY" "$RATATUI_MEMORY"
printf "%-20s %15s %15s\n" "  Idle CPU:" "$GOLI_CPU" "$RATATUI_CPU"
printf "%-20s %13s/s %13s/s\n" "  Updates:" "$GOLI_UPDATES" "$RATATUI_UPDATES"
printf "%-20s %15s %15s\n" "  Max FPS:" "$GOLI_FPS" "$RATATUI_FPS"
printf "%-20s %15s %15s\n" "  Large FPS:" "$GOLI_LARGE_FPS" "$RATATUI_LARGE_FPS"
echo
echo "Notes:"
echo "  - goli: retained-mode with fine-grained reactivity (signals)"
echo "  - ratatui: immediate-mode rendering"
echo "  - Both render to in-memory buffer (no actual terminal I/O)"
echo
