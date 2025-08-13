#!/bin/bash
# Enhanced Test Diagnostics for CI/CD
# Provides comprehensive test execution monitoring with detailed failure analysis

set -euo pipefail

# Source the common diagnostic library
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/ci-diagnostics.sh"

# Test-specific diagnostic functions
validate_test_environment() {
    local job_name="$1"
    local matrix_os="${2:-ubuntu-latest}"
    
    init_diagnostics "$job_name" "test-environment-validation"
    
    log_diagnostic "INFO" "🧪 Validating test environment for $matrix_os"
    
    # Start performance monitoring
    local perf_start
    perf_start=$(monitor_step_performance "test-environment-validation" 300 120)
    
    # Validate basic test prerequisites
    validate_step_prerequisites "test-environment" \
        "go" "workspace" "go-mod" "proto-files"
    
    # Validate test-specific requirements
    validate_test_infrastructure
    
    # Check for race condition detection capabilities
    validate_race_detection_support
    
    # Monitor system resources for test stability
    monitor_system_resources "test-environment-validation"
    
    complete_step_performance_monitoring "test-environment-validation" "$perf_start" 300 120
    
    log_diagnostic "SUCCESS" "✅ Test environment validation completed successfully"
}

validate_test_infrastructure() {
    log_diagnostic "INFO" "🔍 Validating test infrastructure"
    
    # Check test compilation without execution
    log_diagnostic "DEBUG" "Pre-compiling all test files"
    local packages
    packages=$(go list ./...)
    local compile_failures=0
    
    for pkg in $packages; do
        log_diagnostic "TRACE" "Compiling tests for $pkg"
        if ! go test -c -o /dev/null "$pkg" 2>/dev/null; then
            log_diagnostic "ERROR" "Test compilation failed for package: $pkg"
            compile_failures=1
            
            # Get detailed error information
            local compile_error
            compile_error=$(go test -c -o /dev/null "$pkg" 2>&1 || true)
            log_diagnostic "ERROR" "Compilation error details:"
            echo "$compile_error" | while IFS= read -r line; do
                log_diagnostic "ERROR" "  $line"
            done
        fi
    done
    
    if [[ $compile_failures -eq 1 ]]; then
        fail_with_comprehensive_diagnostics 1 \
            "Test compilation failed for one or more packages" \
            "Tests contain compilation errors or missing dependencies" \
            "Fix compilation errors in test files before running tests" \
            "This prevents running tests that are guaranteed to fail compilation"
    fi
    
    # Check for test files existence
    local test_files_found=0
    while IFS= read -r -d '' test_file; do
        test_files_found=1
        log_diagnostic "TRACE" "Found test file: $test_file"
    done < <(find . -name "*_test.go" -print0)
    
    if [[ $test_files_found -eq 0 ]]; then
        log_diagnostic "WARN" "No test files found in the repository"
        log_diagnostic "WARN" "This may indicate missing tests or incorrect file naming"
    else
        log_diagnostic "SUCCESS" "✅ Test files found and compiled successfully"
    fi
    
    log_diagnostic "SUCCESS" "✅ Test infrastructure validated"
}

validate_race_detection_support() {
    log_diagnostic "DEBUG" "Validating race detection support"
    
    # Check if race detection is supported on current platform
    if go test -race -c -o /dev/null ./pkg/ephemos 2>/dev/null; then
        log_diagnostic "SUCCESS" "✅ Race detection supported on this platform"
        echo "RACE_DETECTION_SUPPORTED=true" >> "$GITHUB_ENV" 2>/dev/null || true
    else
        log_diagnostic "WARN" "Race detection not supported on this platform"
        echo "RACE_DETECTION_SUPPORTED=false" >> "$GITHUB_ENV" 2>/dev/null || true
    fi
}

execute_test_suite() {
    local test_type="${1:-unit}"
    local coverage_enabled="${2:-true}"
    local race_detection="${3:-auto}"
    
    log_diagnostic "INFO" "🧪 Executing test suite: $test_type"
    
    # Start performance monitoring
    local perf_start
    perf_start=$(monitor_step_performance "test-execution-$test_type" 900 600)
    
    # Pre-test validation
    pre_test_validation "$test_type"
    
    # Execute tests based on type
    case "$test_type" in
        "unit")
            execute_unit_tests "$coverage_enabled" "$race_detection"
            ;;
        "integration")
            execute_integration_tests "$coverage_enabled"
            ;;
        "benchmark")
            execute_benchmark_tests
            ;;
        "fuzz")
            execute_fuzz_tests
            ;;
        "all")
            execute_unit_tests "$coverage_enabled" "$race_detection"
            execute_integration_tests "$coverage_enabled"
            ;;
        *)
            fail_with_comprehensive_diagnostics 1 \
                "Unknown test type: $test_type" \
                "Invalid test type specified" \
                "Use one of: unit, integration, benchmark, fuzz, all" \
                "Available test types defined in test-diagnostics.sh"
            ;;
    esac
    
    # Post-test validation
    post_test_validation "$test_type" "$coverage_enabled"
    
    complete_step_performance_monitoring "test-execution-$test_type" "$perf_start" 900 600
    
    log_diagnostic "SUCCESS" "✅ Test suite execution completed: $test_type"
}

pre_test_validation() {
    local test_type="$1"
    
    log_diagnostic "INFO" "🔍 Pre-test validation for: $test_type"
    
    # Ensure clean test environment
    log_diagnostic "DEBUG" "Cleaning test cache"
    execute_with_diagnostics "clean-test-cache" "Cleaning Go test cache" \
        go clean -testcache
    
    # Verify test dependencies are available
    log_diagnostic "DEBUG" "Verifying test dependencies"
    if ! execute_with_diagnostics "go-mod-download" "Downloading test dependencies" \
        go mod download; then
        fail_with_comprehensive_diagnostics 1 \
            "Failed to download test dependencies" \
            "Network connectivity issues or invalid module references" \
            "Check internet connection and module proxy settings" \
            "Test execution requires all dependencies to be available"
    fi
    
    # Check for conflicting processes (integration tests)
    if [[ "$test_type" == "integration" ]]; then
        check_port_conflicts
    fi
    
    log_diagnostic "SUCCESS" "✅ Pre-test validation completed"
}

check_port_conflicts() {
    log_diagnostic "DEBUG" "Checking for port conflicts"
    
    # Check common ports used by the application
    local test_ports=("50051" "50099" "8080" "9090")
    local port_conflicts=0
    
    for port in "${test_ports[@]}"; do
        if command -v netstat >/dev/null 2>&1; then
            if netstat -an | grep -q ":$port.*LISTEN"; then
                log_diagnostic "WARN" "Port $port is already in use"
                port_conflicts=1
            fi
        elif command -v ss >/dev/null 2>&1; then
            if ss -an | grep -q ":$port.*LISTEN"; then
                log_diagnostic "WARN" "Port $port is already in use"
                port_conflicts=1
            fi
        elif command -v lsof >/dev/null 2>&1; then
            if lsof -i ":$port" >/dev/null 2>&1; then
                log_diagnostic "WARN" "Port $port is already in use"
                port_conflicts=1
            fi
        fi
    done
    
    if [[ $port_conflicts -eq 1 ]]; then
        log_diagnostic "WARN" "Port conflicts detected - integration tests may fail"
        log_diagnostic "WARN" "Consider using random ports or cleaning up existing processes"
    fi
}

execute_unit_tests() {
    local coverage_enabled="$1"
    local race_detection="$2"
    
    log_diagnostic "INFO" "🏃 Executing unit tests"
    
    # Determine race detection setting
    local race_flag=""
    if [[ "$race_detection" == "true" ]] || \
       [[ "$race_detection" == "auto" && "${RACE_DETECTION_SUPPORTED:-false}" == "true" ]]; then
        race_flag="-race"
        log_diagnostic "INFO" "Race detection enabled"
    fi
    
    # Determine coverage setting
    local coverage_flags=()
    local coverage_file=""
    if [[ "$coverage_enabled" == "true" ]]; then
        coverage_file="coverage.out"
        coverage_flags=("-coverprofile=$coverage_file" "-covermode=atomic")
        log_diagnostic "INFO" "Coverage collection enabled: $coverage_file"
    fi
    
    # Build test command
    local test_cmd=("go" "test" "-v" "-timeout=10m")
    
    if [[ -n "$race_flag" ]]; then
        test_cmd+=("$race_flag")
    fi
    
    if [[ ${#coverage_flags[@]} -gt 0 ]]; then
        test_cmd+=("${coverage_flags[@]}")
    fi
    
    test_cmd+=("./...")
    
    # Execute unit tests with comprehensive error handling
    local test_output
    local test_exit_code=0
    
    log_diagnostic "DEBUG" "Test command: ${test_cmd[*]}"
    
    if test_output=$(command "${test_cmd[@]}" 2>&1); then
        log_diagnostic "SUCCESS" "✅ Unit tests passed"
        
        # Parse test results
        parse_test_results "$test_output" "unit"
        
        # Validate coverage if enabled
        if [[ "$coverage_enabled" == "true" && -f "$coverage_file" ]]; then
            validate_test_coverage "$coverage_file"
        fi
    else
        test_exit_code=$?
        log_diagnostic "ERROR" "❌ Unit tests failed"
        
        # Parse test failures for detailed diagnostics
        parse_test_failures "$test_output" "unit"
        
        fail_with_comprehensive_diagnostics "$test_exit_code" \
            "Unit tests failed" \
            "One or more unit tests did not pass" \
            "Review test failure details above and fix failing tests" \
            "Test command: ${test_cmd[*]}"
    fi
}

execute_integration_tests() {
    local coverage_enabled="$1"
    
    log_diagnostic "INFO" "🔗 Executing integration tests"
    
    # Integration tests typically run with different tags or in specific packages
    local integration_packages=()
    
    # Find packages with integration tests
    while IFS= read -r package; do
        if [[ -n "$package" ]]; then
            integration_packages+=("$package")
        fi
    done < <(find . -name "*integration*test.go" -exec dirname {} \; | sort -u | sed 's|^\./||')
    
    if [[ ${#integration_packages[@]} -eq 0 ]]; then
        log_diagnostic "INFO" "No integration test packages found"
        return 0
    fi
    
    log_diagnostic "INFO" "Found integration test packages: ${integration_packages[*]}"
    
    # Execute integration tests with longer timeout
    local test_cmd=("go" "test" "-v" "-timeout=30m" "-tags=integration")
    
    if [[ "$coverage_enabled" == "true" ]]; then
        test_cmd+=("-coverprofile=integration-coverage.out" "-covermode=atomic")
    fi
    
    for package in "${integration_packages[@]}"; do
        test_cmd+=("./$package")
    done
    
    if execute_with_diagnostics "integration-tests" "Running integration tests" \
        "${test_cmd[@]}"; then
        log_diagnostic "SUCCESS" "✅ Integration tests passed"
    else
        fail_with_comprehensive_diagnostics $? \
            "Integration tests failed" \
            "One or more integration tests did not pass" \
            "Check service dependencies, network connectivity, and test environment setup" \
            "Integration tests require external dependencies to be properly configured"
    fi
}

execute_benchmark_tests() {
    log_diagnostic "INFO" "⚡ Executing benchmark tests"
    
    # Run benchmarks with memory allocation stats
    local benchmark_cmd=("go" "test" "-v" "-bench=." "-benchmem" "-timeout=10m" "-run=^$")
    
    # Find packages with benchmark tests
    if ! find . -name "*_test.go" -exec grep -l "func Benchmark" {} \; | head -1 | grep -q .; then
        log_diagnostic "INFO" "No benchmark tests found"
        return 0
    fi
    
    benchmark_cmd+=("./...")
    
    local benchmark_output
    if benchmark_output=$(command "${benchmark_cmd[@]}" 2>&1); then
        log_diagnostic "SUCCESS" "✅ Benchmark tests completed"
        
        # Save benchmark results
        echo "$benchmark_output" > "benchmark-results.txt"
        
        # Parse benchmark results for analysis
        parse_benchmark_results "$benchmark_output"
        
        log_diagnostic "INFO" "Benchmark results saved to benchmark-results.txt"
    else
        log_diagnostic "ERROR" "❌ Benchmark tests failed"
        echo "$benchmark_output" | while IFS= read -r line; do
            log_diagnostic "ERROR" "  $line"
        done
        
        fail_with_comprehensive_diagnostics $? \
            "Benchmark tests failed" \
            "Benchmark execution encountered errors" \
            "Check benchmark test implementation and system resources" \
            "Benchmark command: ${benchmark_cmd[*]}"
    fi
}

execute_fuzz_tests() {
    log_diagnostic "INFO" "🎯 Executing fuzz tests"
    
    # Find fuzz tests
    local fuzz_tests=()
    while IFS= read -r fuzz_test; do
        fuzz_tests+=("$fuzz_test")
    done < <(grep -r "func Fuzz" --include="*_test.go" . | sed 's/.*func \(Fuzz[^(]*\).*/\1/' | sort -u)
    
    if [[ ${#fuzz_tests[@]} -eq 0 ]]; then
        log_diagnostic "INFO" "No fuzz tests found"
        return 0
    fi
    
    log_diagnostic "INFO" "Found fuzz tests: ${fuzz_tests[*]}"
    
    # Run each fuzz test for a short duration
    local fuzz_duration="30s"
    if [[ "${GITHUB_EVENT_NAME:-}" == "schedule" ]]; then
        fuzz_duration="5m"
    fi
    
    for fuzz_test in "${fuzz_tests[@]}"; do
        log_diagnostic "INFO" "Running fuzz test: $fuzz_test"
        
        if execute_with_diagnostics "fuzz-$fuzz_test" "Fuzzing $fuzz_test" \
            go test -fuzz="$fuzz_test" -fuzztime="$fuzz_duration" ./pkg/ephemos/; then
            log_diagnostic "SUCCESS" "✅ Fuzz test $fuzz_test completed"
        else
            log_diagnostic "WARN" "⚠️ Fuzz test $fuzz_test found issues or timed out"
        fi
    done
}

parse_test_results() {
    local test_output="$1"
    local test_type="$2"
    
    log_diagnostic "DEBUG" "Parsing $test_type test results"
    
    # Extract test statistics
    local total_tests=0
    local passed_tests=0
    local failed_tests=0
    local skipped_tests=0
    
    # Parse Go test output format
    while IFS= read -r line; do
        if [[ "$line" =~ ^--- ]]; then
            total_tests=$((total_tests + 1))
            if [[ "$line" =~ PASS ]]; then
                passed_tests=$((passed_tests + 1))
            elif [[ "$line" =~ FAIL ]]; then
                failed_tests=$((failed_tests + 1))
            elif [[ "$line" =~ SKIP ]]; then
                skipped_tests=$((skipped_tests + 1))
            fi
        fi
    done <<< "$test_output"
    
    log_diagnostic "INFO" "📊 $test_type Test Summary:"
    log_diagnostic "INFO" "  Total: $total_tests"
    log_diagnostic "INFO" "  Passed: $passed_tests"
    log_diagnostic "INFO" "  Failed: $failed_tests"
    log_diagnostic "INFO" "  Skipped: $skipped_tests"
    
    # Add to GitHub Actions summary if available
    if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
        {
            echo "## 📊 $test_type Test Results"
            echo ""
            echo "| Status | Count |"
            echo "|--------|-------|"
            echo "| ✅ Passed | $passed_tests |"
            echo "| ❌ Failed | $failed_tests |" 
            echo "| ⏭️ Skipped | $skipped_tests |"
            echo "| 📈 **Total** | **$total_tests** |"
            echo ""
        } >> "$GITHUB_STEP_SUMMARY"
    fi
}

parse_test_failures() {
    local test_output="$1"
    local test_type="$2"
    
    log_diagnostic "ERROR" "📋 Analyzing $test_type test failures"
    
    # Extract failed test details
    local current_test=""
    local in_failure=false
    
    while IFS= read -r line; do
        # Detect start of test failure
        if [[ "$line" =~ ^---[[:space:]]FAIL:[[:space:]] ]]; then
            current_test=$(echo "$line" | sed 's/^--- FAIL: \([^(]*\).*/\1/')
            in_failure=true
            log_diagnostic "ERROR" "❌ FAILED TEST: $current_test"
        # Detect test output lines
        elif [[ $in_failure == true && "$line" =~ ^[[:space:]]*[^=] ]]; then
            log_diagnostic "ERROR" "  $line"
        # Detect end of current test
        elif [[ "$line" =~ ^=== ]]; then
            in_failure=false
            current_test=""
        fi
        
        # Look for common failure patterns
        if [[ "$line" =~ panic: ]] || [[ "$line" =~ runtime\ error: ]]; then
            log_diagnostic "ERROR" "🚨 PANIC/RUNTIME ERROR: $line"
        elif [[ "$line" =~ timeout ]]; then
            log_diagnostic "ERROR" "⏰ TIMEOUT: $line"
        elif [[ "$line" =~ race\ detected ]] || [[ "$line" =~ DATA\ RACE ]]; then
            log_diagnostic "ERROR" "🏁 RACE CONDITION: $line"
        fi
    done <<< "$test_output"
}

parse_benchmark_results() {
    local benchmark_output="$1"
    
    log_diagnostic "DEBUG" "Parsing benchmark results"
    
    # Extract benchmark statistics
    local benchmark_count=0
    
    while IFS= read -r line; do
        if [[ "$line" =~ ^Benchmark ]]; then
            benchmark_count=$((benchmark_count + 1))
            log_diagnostic "INFO" "📊 $line"
            
            # Check for performance regressions (basic analysis)
            if [[ "$line" =~ [0-9]+\ ns/op ]]; then
                local ns_per_op=$(echo "$line" | grep -o '[0-9]*\ ns/op' | grep -o '[0-9]*')
                if [[ $ns_per_op -gt 1000000 ]]; then  # > 1ms per operation
                    log_diagnostic "WARN" "⚠️ Slow benchmark detected: $line"
                fi
            fi
        fi
    done <<< "$benchmark_output"
    
    log_diagnostic "INFO" "📊 Total benchmarks executed: $benchmark_count"
}

validate_test_coverage() {
    local coverage_file="$1"
    
    log_diagnostic "INFO" "📊 Validating test coverage"
    
    if [[ ! -f "$coverage_file" ]]; then
        log_diagnostic "WARN" "Coverage file not found: $coverage_file"
        return 0
    fi
    
    # Generate coverage report
    local coverage_report
    if coverage_report=$(go tool cover -func="$coverage_file" 2>&1); then
        log_diagnostic "INFO" "Coverage report generated successfully"
        
        # Extract total coverage percentage
        local total_coverage
        total_coverage=$(echo "$coverage_report" | tail -1 | grep -o '[0-9]*\.[0-9]*%' | head -1)
        
        if [[ -n "$total_coverage" ]]; then
            log_diagnostic "INFO" "📊 Total test coverage: $total_coverage"
            
            # Parse coverage percentage for validation
            local coverage_value
            coverage_value=$(echo "$total_coverage" | sed 's/%//')
            
            if (( $(echo "$coverage_value < 50.0" | bc -l) )); then
                log_diagnostic "WARN" "⚠️ Low test coverage detected: $total_coverage"
                log_diagnostic "WARN" "Consider adding more tests to improve coverage"
            elif (( $(echo "$coverage_value >= 80.0" | bc -l) )); then
                log_diagnostic "SUCCESS" "✅ Good test coverage: $total_coverage"
            else
                log_diagnostic "INFO" "📊 Moderate test coverage: $total_coverage"
            fi
        else
            log_diagnostic "WARN" "Could not extract coverage percentage"
        fi
        
        # Save coverage report for CI artifacts
        echo "$coverage_report" > "coverage-report.txt"
        
    else
        log_diagnostic "ERROR" "Failed to generate coverage report"
        log_diagnostic "ERROR" "$coverage_report"
    fi
}

post_test_validation() {
    local test_type="$1"
    local coverage_enabled="$2"
    
    log_diagnostic "INFO" "🔍 Post-test validation for: $test_type"
    
    # Validate test artifacts
    local expected_artifacts=()
    
    if [[ "$coverage_enabled" == "true" ]]; then
        expected_artifacts+=("coverage.out")
    fi
    
    if [[ "$test_type" == "benchmark" ]] || [[ "$test_type" == "all" ]]; then
        if [[ -f "benchmark-results.txt" ]]; then
            expected_artifacts+=("benchmark-results.txt")
        fi
    fi
    
    # Validate artifacts exist and are not empty
    for artifact in "${expected_artifacts[@]}"; do
        if [[ ! -f "$artifact" ]]; then
            log_diagnostic "WARN" "Expected test artifact missing: $artifact"
        elif [[ ! -s "$artifact" ]]; then
            log_diagnostic "WARN" "Test artifact is empty: $artifact"
        else
            local artifact_size
            artifact_size=$(stat -c%s "$artifact" 2>/dev/null || stat -f%z "$artifact" 2>/dev/null)
            log_diagnostic "SUCCESS" "✅ Test artifact validated: $artifact ($artifact_size bytes)"
        fi
    done
    
    # Check for test cache issues
    validate_test_cache_state
    
    log_diagnostic "SUCCESS" "✅ Post-test validation completed"
}

validate_test_cache_state() {
    log_diagnostic "DEBUG" "Validating test cache state"
    
    # Check if test cache is getting too large
    local test_cache_dir
    test_cache_dir=$(go env GOCACHE)
    
    if [[ -d "$test_cache_dir" ]]; then
        local cache_size
        if command -v du >/dev/null 2>&1; then
            cache_size=$(du -sh "$test_cache_dir" 2>/dev/null | cut -f1 || echo "unknown")
            log_diagnostic "DEBUG" "Test cache size: $cache_size"
        fi
    fi
}

# Export test-specific functions
export -f validate_test_environment
export -f execute_test_suite
export -f validate_test_infrastructure
export -f parse_test_results
export -f parse_test_failures
export -f validate_test_coverage

log_diagnostic "INFO" "🧪 Test Diagnostics Library loaded successfully"