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

echo "========================================"
echo "  goli vs Ink Benchmark Comparison"
echo "========================================"
echo

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

echo "Go version: $(go version | cut -d' ' -f3)"
echo "JS runtime: $JS_RUNTIME $JS_VERSION"
echo

# Build goli benchmark
echo "Building goli benchmark..."
cd goli-bench
go build -o ../goli-bench-bin .
cd ..

# Get binary size
GOLI_SIZE=$(ls -l goli-bench-bin | awk '{print $5}')
GOLI_SIZE_MB=$(echo "scale=2; $GOLI_SIZE / 1024 / 1024" | bc)

# Install Ink dependencies and get size
echo "Installing Ink dependencies..."
cd ink-bench
if [ "$JS_RUNTIME" = "bun" ]; then
    bun install --silent 2>/dev/null || bun install
else
    npm install --silent 2>/dev/null || npm install
fi
INK_SIZE=$(du -s node_modules 2>/dev/null | cut -f1)
INK_SIZE_MB=$(echo "scale=2; $INK_SIZE / 1024" | bc)
cd ..

echo
echo "========================================"
echo "  Binary/Bundle Size"
echo "========================================"
echo "goli binary:     ${GOLI_SIZE_MB} MB"
echo "Ink node_modules: ${INK_SIZE_MB} MB"
echo

echo "========================================"
echo "  Running Benchmarks"
echo "========================================"
echo

# Run goli benchmark
echo "--- goli (Go) ---"
./goli-bench-bin
echo

# Run Ink benchmark
echo "--- Ink (React/Node.js) ---"
cd ink-bench
if [ "$JS_RUNTIME" = "bun" ]; then
    bun run benchmark.tsx
else
    node --experimental-strip-types benchmark.tsx
fi
cd ..

# Cleanup
rm -f goli-bench-bin

echo
echo "========================================"
echo "  Benchmark Complete"
echo "========================================"
