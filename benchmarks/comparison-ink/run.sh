#!/bin/bash

# Benchmark comparison between goli (Go) and Ink (Node.js/Bun)
#
# Prerequisites:
#   - Go 1.21+
#   - Bun (https://bun.sh) or Node.js 22+
#
# Usage:
#   ./run.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Check for bun or node
if command -v bun &> /dev/null; then
    JS_RUNTIME="bun"
    JS_VERSION=$(bun --version)
elif command -v node &> /dev/null; then
    JS_RUNTIME="node"
    JS_VERSION=$(node --version)
else
    echo "Error: Neither bun nor node found. Please install one."
    exit 1
fi

GO_VERSION=$(go version | cut -d' ' -f3)

echo "Building and running benchmarks..."
echo

# Build goli benchmark
cd goli-bench
gox build -o ../goli-bench-bin . 2>/dev/null
cd ..

# Get binary size
GOLI_SIZE=$(ls -l goli-bench-bin | awk '{print $5}')
GOLI_SIZE_MB=$(echo "scale=2; $GOLI_SIZE / 1024 / 1024" | bc)

# Install Ink dependencies and get size
cd ink-bench
if [ "$JS_RUNTIME" = "bun" ]; then
    bun install --silent 2>/dev/null || bun install >/dev/null 2>&1
else
    npm install --silent 2>/dev/null || npm install >/dev/null 2>&1
fi
INK_SIZE=$(du -s node_modules 2>/dev/null | cut -f1)
INK_SIZE_MB=$(echo "scale=2; $INK_SIZE / 1024" | bc)
cd ..

# Run goli benchmark and capture output
echo "Running goli benchmark..."
GOLI_OUTPUT=$(./goli-bench-bin 2>/dev/null)

# Run Ink benchmark and capture output (redirect stderr to /dev/null to hide UI)
echo "Running Ink benchmark..."
cd ink-bench
if [ "$JS_RUNTIME" = "bun" ]; then
    INK_OUTPUT=$(bun run benchmark.tsx 2>/dev/null)
else
    INK_OUTPUT=$(node --experimental-strip-types benchmark.tsx 2>/dev/null)
fi
cd ..

# Cleanup
rm -f goli-bench-bin

# Parse results
parse_value() {
    echo "$1" | grep -i "$2" | head -1 | sed 's/.*: *//' | sed 's/ .*//'
}

GOLI_STARTUP=$(parse_value "$GOLI_OUTPUT" "Startup time")
GOLI_MEMORY=$(parse_value "$GOLI_OUTPUT" "Memory used")
GOLI_CPU=$(parse_value "$GOLI_OUTPUT" "Idle CPU")
GOLI_UPDATES=$(echo "$GOLI_OUTPUT" | grep -i "updates/sec" | sed 's/.*(\([0-9]*\) updates.*/\1/')

INK_STARTUP=$(parse_value "$INK_OUTPUT" "Startup time")
INK_MEMORY=$(parse_value "$INK_OUTPUT" "Memory used")
INK_CPU=$(parse_value "$INK_OUTPUT" "Idle CPU")
INK_UPDATES=$(echo "$INK_OUTPUT" | grep -i "updates/sec" | sed 's/.*(\([0-9]*\) updates.*/\1/')

# Print summary
echo
echo "========================================"
echo "  goli vs Ink Benchmark Results"
echo "========================================"
echo
echo "Environment:"
echo "  Go: $GO_VERSION"
echo "  JS: $JS_RUNTIME $JS_VERSION"
echo
echo "                        goli          Ink"
echo "  ─────────────────────────────────────────"
echo "  Binary size:       ${GOLI_SIZE_MB} MB       ${INK_SIZE_MB} MB"
echo "  Startup:           ${GOLI_STARTUP}       ${INK_STARTUP}"
echo "  Memory:            ${GOLI_MEMORY} MB       ${INK_MEMORY} MB"
echo "  Idle CPU:          ${GOLI_CPU}         ${INK_CPU}"
echo "  Updates:           ${GOLI_UPDATES}/s        ${INK_UPDATES}/s"
echo
