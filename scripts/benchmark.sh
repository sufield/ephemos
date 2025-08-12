#!/bin/bash
# Benchmark runner script for Ephemos
# Configures environment for clean benchmark execution
set -e
command -v go >/dev/null || { echo "❌ Go not installed"; exit 1; }
test -d ./pkg/ephemos || { echo "❌ ./pkg/ephemos not found"; exit 1; }
echo "🚀 Running Ephemos benchmarks..."
# Set up environment for benchmarks
export EPHEMOS_LOG_LEVEL=error # Reduce log noise
export SPIFFE_ENDPOINT_SOCKET="" # Disable SPIRE connection attempts
export EPHEMOS_SPIFFE_ENABLED=false # Disable SPIFFE features completely
export EPHEMOS_BENCHMARK_MODE=true # Signal benchmark mode to skip SPIRE setup
# Run benchmarks with clean output - target only pkg/ephemos to avoid problematic tests
echo "📊 Executing benchmark suite..."
echo "Go version: $(go version)"
env | grep EPHEMOS
# Store exit code but continue to process results
set +e
go mod download
go test -bench=. -benchmem -run=^$ -timeout=5m ./pkg/ephemos > benchmark-results.txt 2>&1
TEST_EXIT_CODE=$?
set -e
# Check if benchmarks actually ran and produced results
if [ $TEST_EXIT_CODE -eq 0 ] && grep -q "^Benchmark.*-[0-9]+" benchmark-results.txt && grep -q "PASS" benchmark-results.txt; then
    echo "✅ Benchmarks completed successfully"
    echo ""
    echo "📈 Benchmark Results Summary:"
    echo "==============================="
   
    # Show summary of benchmark results
    grep -E "^Benchmark.*-[0-9]+" benchmark-results.txt | head -20 || true
   
    echo ""
    echo "💾 Full results saved to: benchmark-results.txt"
    echo "📊 Lines of output: $(wc -l < benchmark-results.txt)"
   
    # Exit successfully even if there were SPIFFE logging issues
    exit 0
else
    echo "❌ Benchmarks failed"
    echo ""
    echo "🔍 Error details:"
    cat benchmark-results.txt || true
    echo ""
    echo "Test exit code: $TEST_EXIT_CODE"
    exit 1
fi