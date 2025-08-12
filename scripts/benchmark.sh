#!/bin/bash

# Benchmark runner script for Ephemos
# Configures environment for clean benchmark execution

set -e

echo "🚀 Running Ephemos benchmarks..."

# Set up environment for benchmarks
export EPHEMOS_LOG_LEVEL=error  # Reduce log noise
export SPIFFE_ENDPOINT_SOCKET=""  # Disable SPIRE connection attempts
export EPHEMOS_SPIFFE_ENABLED=false  # Disable SPIFFE features completely
export EPHEMOS_BENCHMARK_MODE=true  # Signal benchmark mode to skip SPIRE setup

# Run benchmarks with clean output
echo "📊 Executing benchmark suite..."
if go test -bench=. -benchmem -run=^$ ./... > benchmark-results.txt 2>&1; then
    echo "✅ Benchmarks completed successfully"
    echo ""
    echo "📈 Benchmark Results Summary:"
    echo "==============================="
    
    # Show summary of benchmark results
    grep -E "^Benchmark.*-[0-9]+" benchmark-results.txt | head -20 || true
    
    echo ""
    echo "💾 Full results saved to: benchmark-results.txt"
    echo "📊 Lines of output: $(wc -l < benchmark-results.txt)"
    
    exit 0
else
    echo "❌ Benchmarks failed"
    echo ""
    echo "🔍 Error details:"
    tail -50 benchmark-results.txt || true
    exit 1
fi