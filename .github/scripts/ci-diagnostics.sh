#!/bin/bash
# Comprehensive CI/CD Diagnostic Library with Fail-Fast Verbose Reporting
# Usage: source ./.github/scripts/ci-diagnostics.sh

# Guard against multiple sourcing
if [ "${CI_DIAGNOSTICS_LOADED:-}" = "1" ]; then
    return 0 2>/dev/null || exit 0
fi
export CI_DIAGNOSTICS_LOADED=1

set -euo pipefail

# Global diagnostic configuration
DIAGNOSTIC_MODE=${EPHEMOS_DIAGNOSTIC_VERBOSE:-1}
SCRIPT_NAME="${0##*/}"
DIAGNOSTIC_LOG_FILE="${GITHUB_WORKSPACE:-/tmp}/ci-diagnostic.log"
JOB_START_TIME=$(date +%s)

# Color codes for enhanced output (use different names to avoid conflicts)
readonly DIAG_RED='\033[0;31m'
readonly DIAG_GREEN='\033[0;32m'
readonly DIAG_YELLOW='\033[1;33m'
readonly DIAG_BLUE='\033[0;34m'
readonly DIAG_PURPLE='\033[0;35m'
readonly DIAG_CYAN='\033[0;36m'
readonly DIAG_WHITE='\033[1;37m'
readonly DIAG_NC='\033[0m' # No Color

# Initialize diagnostic logging
init_diagnostics() {
    local job_name="${1:-unknown-job}"
    local step_name="${2:-unknown-step}"
    
    echo "🚀 INITIALIZING CI DIAGNOSTICS" | tee -a "$DIAGNOSTIC_LOG_FILE"
    echo "=============================================" | tee -a "$DIAGNOSTIC_LOG_FILE"
    echo "Job: $job_name" | tee -a "$DIAGNOSTIC_LOG_FILE"
    echo "Step: $step_name" | tee -a "$DIAGNOSTIC_LOG_FILE"
    echo "Time: $(date -u '+%Y-%m-%d %H:%M:%S UTC')" | tee -a "$DIAGNOSTIC_LOG_FILE"
    echo "Workflow: ${GITHUB_WORKFLOW:-unknown}" | tee -a "$DIAGNOSTIC_LOG_FILE"
    echo "Actor: ${GITHUB_ACTOR:-unknown}" | tee -a "$DIAGNOSTIC_LOG_FILE"
    echo "Ref: ${GITHUB_REF:-unknown}" | tee -a "$DIAGNOSTIC_LOG_FILE"
    echo "SHA: ${GITHUB_SHA:-unknown}" | tee -a "$DIAGNOSTIC_LOG_FILE"
    echo "Run ID: ${GITHUB_RUN_ID:-unknown}" | tee -a "$DIAGNOSTIC_LOG_FILE"
    echo "Run Number: ${GITHUB_RUN_NUMBER:-unknown}" | tee -a "$DIAGNOSTIC_LOG_FILE"
    echo "=============================================" | tee -a "$DIAGNOSTIC_LOG_FILE"
    echo ""
}

# Enhanced logging with levels and context
log_diagnostic() {
    local level="$1"
    local message="$2"
    local source_file="${BASH_SOURCE[2]:-unknown}"
    local location="${source_file##*/}:${BASH_LINENO[1]:-0}"
    local timestamp="$(date -u '+%Y-%m-%d %H:%M:%S UTC')"
    local function_name="${FUNCNAME[2]:-main}"
    local elapsed_time=$(($(date +%s) - JOB_START_TIME))
    
    local color_code=""
    local icon=""
    
    case "$level" in
        "TRACE")   color_code="$DIAG_CYAN";    icon="🔍"; [[ ${DIAGNOSTIC_MODE:-1} -lt 2 ]] && return 0 ;;
        "DEBUG")   color_code="$DIAG_BLUE";    icon="🐛"; [[ ${DIAGNOSTIC_MODE:-1} -lt 1 ]] && return 0 ;;
        "INFO")    color_code="$DIAG_GREEN";   icon="ℹ️" ;;
        "WARN")    color_code="$DIAG_YELLOW";  icon="⚠️" ;;
        "ERROR")   color_code="$DIAG_RED";     icon="❌" ;;
        "FATAL")   color_code="$DIAG_PURPLE";  icon="💀" ;;
        "SUCCESS") color_code="$DIAG_GREEN";   icon="✅" ;;
        *) color_code="$DIAG_WHITE"; icon="📝" ;;
    esac
    
    local formatted_message
    formatted_message=$(printf "[%s] [+%03ds] [%s] [%s] %s %s: %s" \
        "$timestamp" "$elapsed_time" "$location" "$function_name" "$icon" "$level" "$message")
    
    # Output to console with color
    echo -e "${color_code}${formatted_message}${DIAG_NC}"
    
    # Output to log file without color codes
    echo "$formatted_message" >> "$DIAGNOSTIC_LOG_FILE"
    
    # For GitHub Actions, add to step summary for ERROR/FATAL
    if [[ "$level" == "ERROR" || "$level" == "FATAL" ]] && [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
        echo "## 🚨 ${level}: ${message}" >> "$GITHUB_STEP_SUMMARY"
        echo "" >> "$GITHUB_STEP_SUMMARY"
        echo "- **Location**: \`${location}\`" >> "$GITHUB_STEP_SUMMARY"
        echo "- **Function**: \`${function_name}\`" >> "$GITHUB_STEP_SUMMARY"
        echo "- **Time**: ${timestamp}" >> "$GITHUB_STEP_SUMMARY"
        echo "- **Elapsed**: ${elapsed_time}s" >> "$GITHUB_STEP_SUMMARY"
        echo "" >> "$GITHUB_STEP_SUMMARY"
    fi
}

# Critical failure handler with comprehensive diagnostics
fail_with_comprehensive_diagnostics() {
    local exit_code="$1"
    local error_message="$2"
    local root_cause="${3:-Unknown root cause}"
    local fix_suggestion="${4:-No specific fix available}"
    local context="${5:-No additional context}"
    local line_number="${BASH_LINENO[1]:-unknown}"
    local failing_function="${FUNCNAME[1]:-unknown}"
    local failing_file="${BASH_SOURCE[1]##*/}"
    
    log_diagnostic "FATAL" "CRITICAL FAILURE DETECTED"
    log_diagnostic "FATAL" "════════════════════════════════════════════════════════════════"
    log_diagnostic "ERROR" "Exit Code: $exit_code"
    log_diagnostic "ERROR" "Error Message: $error_message"
    log_diagnostic "ERROR" "Root Cause: $root_cause"
    log_diagnostic "ERROR" "How to Fix: $fix_suggestion"
    log_diagnostic "ERROR" "Context: $context"
    log_diagnostic "ERROR" "Location: $failing_file:$line_number in $failing_function()"
    log_diagnostic "ERROR" "════════════════════════════════════════════════════════════════"
    
    # Capture comprehensive environment state
    capture_failure_environment "$exit_code"
    
    # Generate failure report
    generate_failure_report "$exit_code" "$error_message" "$root_cause" "$fix_suggestion"
    
    # Output diagnostic log content to GitHub Actions
    if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
        echo "## 📋 Complete Diagnostic Log" >> "$GITHUB_STEP_SUMMARY"
        echo "" >> "$GITHUB_STEP_SUMMARY"
        echo '<details><summary>Click to expand full diagnostic log</summary>' >> "$GITHUB_STEP_SUMMARY"
        echo "" >> "$GITHUB_STEP_SUMMARY"
        echo '```' >> "$GITHUB_STEP_SUMMARY"
        tail -n 100 "$DIAGNOSTIC_LOG_FILE" >> "$GITHUB_STEP_SUMMARY" 2>/dev/null || echo "Diagnostic log unavailable" >> "$GITHUB_STEP_SUMMARY"
        echo '```' >> "$GITHUB_STEP_SUMMARY"
        echo "</details>" >> "$GITHUB_STEP_SUMMARY"
        echo "" >> "$GITHUB_STEP_SUMMARY"
    fi
    
    log_diagnostic "FATAL" "Job terminating with exit code $exit_code"
    exit "$exit_code"
}

# Capture comprehensive environment state on failure
capture_failure_environment() {
    local exit_code="$1"
    
    log_diagnostic "DEBUG" "Capturing failure environment state..."
    
    {
        echo "=== FAILURE ENVIRONMENT CAPTURE ==="
        echo "Exit Code: $exit_code"
        echo "Timestamp: $(date -u '+%Y-%m-%d %H:%M:%S UTC')"
        echo "Working Directory: $(pwd)"
        echo ""
        
        echo "=== SYSTEM INFORMATION ==="
        echo "OS: $(uname -a)"
        echo "CPU Info:"
        if command -v nproc >/dev/null 2>&1; then
            echo "  CPUs: $(nproc)"
        fi
        echo "Memory Info:"
        if command -v free >/dev/null 2>&1; then
            free -h | head -3
        fi
        echo "Disk Usage:"
        df -h | head -5
        echo ""
        
        echo "=== GITHUB ACTIONS ENVIRONMENT ==="
        echo "Runner OS: ${RUNNER_OS:-unknown}"
        echo "Runner Arch: ${RUNNER_ARCH:-unknown}"
        echo "Runner Name: ${RUNNER_NAME:-unknown}"
        echo "Runner Tool Cache: ${RUNNER_TOOL_CACHE:-unknown}"
        echo "GitHub Workspace: ${GITHUB_WORKSPACE:-unknown}"
        echo "GitHub Event Name: ${GITHUB_EVENT_NAME:-unknown}"
        echo "GitHub Head Ref: ${GITHUB_HEAD_REF:-unknown}"
        echo "GitHub Base Ref: ${GITHUB_BASE_REF:-unknown}"
        echo ""
        
        echo "=== GO ENVIRONMENT ==="
        if command -v go >/dev/null 2>&1; then
            echo "Go Version: $(go version)"
            echo "GOROOT: $(go env GOROOT)"
            echo "GOPATH: $(go env GOPATH)"
            echo "GOMODCACHE: $(go env GOMODCACHE)"
            echo "GOPROXY: $(go env GOPROXY)"
            echo "GONOPROXY: $(go env GONOPROXY)"
            echo "GONOSUMDB: $(go env GONOSUMDB)"
            echo "GO111MODULE: $(go env GO111MODULE)"
        else
            echo "Go: NOT INSTALLED"
        fi
        echo ""
        
        echo "=== PATH INFORMATION ==="
        echo "PATH: $PATH"
        echo ""
        echo "Available commands:"
        for cmd in protoc protoc-gen-go protoc-gen-go-grpc golangci-lint staticcheck; do
            if command -v "$cmd" >/dev/null 2>&1; then
                echo "  $cmd: $(which "$cmd")"
            else
                echo "  $cmd: NOT FOUND"
            fi
        done
        echo ""
        
        echo "=== PROJECT STRUCTURE ==="
        echo "Repository root contents:"
        ls -la . 2>/dev/null | head -20 || echo "Cannot list directory contents"
        echo ""
        
        if [[ -d "pkg/ephemos" ]]; then
            echo "pkg/ephemos contents:"
            ls -la pkg/ephemos/ | head -10
            echo ""
        fi
        
        if [[ -d "examples/proto" ]]; then
            echo "examples/proto contents:"
            ls -la examples/proto/
            echo ""
        fi
        
        echo "=== PROCESS INFORMATION ==="
        echo "Running processes (sample):"
        ps aux | head -10 2>/dev/null || ps | head -10 2>/dev/null || echo "Cannot list processes"
        echo ""
        
        if [[ -n "${GITHUB_WORKSPACE:-}" ]]; then
            echo "=== WORKSPACE DISK USAGE ==="
            du -sh "${GITHUB_WORKSPACE}"/* 2>/dev/null | sort -hr | head -10 || echo "Cannot calculate disk usage"
            echo ""
        fi
        
        echo "=== RECENT LOG ENTRIES ==="
        echo "Last 20 lines of diagnostic log:"
        tail -n 20 "$DIAGNOSTIC_LOG_FILE" 2>/dev/null || echo "No diagnostic log available"
        echo ""
        
        echo "=== END ENVIRONMENT CAPTURE ==="
    } >> "$DIAGNOSTIC_LOG_FILE"
    
    log_diagnostic "DEBUG" "Environment state captured to diagnostic log"
}

# Generate comprehensive failure report
generate_failure_report() {
    local exit_code="$1"
    local error_message="$2"
    local root_cause="$3"
    local fix_suggestion="$4"
    
    local report_file="${GITHUB_WORKSPACE:-/tmp}/failure-report.md"
    
    {
        echo "# 🚨 CI/CD Failure Report"
        echo ""
        echo "**Job**: \`${GITHUB_JOB:-unknown}\`  "
        echo "**Workflow**: \`${GITHUB_WORKFLOW:-unknown}\`  "
        echo "**Event**: \`${GITHUB_EVENT_NAME:-unknown}\`  "
        echo "**Run**: [\#${GITHUB_RUN_NUMBER:-unknown}](${GITHUB_SERVER_URL:-https://github.com}/${GITHUB_REPOSITORY:-unknown}/actions/runs/${GITHUB_RUN_ID:-unknown})  "
        echo "**Actor**: \`${GITHUB_ACTOR:-unknown}\`  "
        echo "**Ref**: \`${GITHUB_REF:-unknown}\`  "
        echo "**SHA**: \`${GITHUB_SHA:0:8}...\`  "
        echo ""
        echo "## 📊 Failure Summary"
        echo ""
        echo "| Field | Value |"
        echo "|-------|-------|"
        echo "| **Exit Code** | \`$exit_code\` |"
        echo "| **Error Message** | \`$error_message\` |"
        echo "| **Root Cause** | $root_cause |"
        echo "| **Timestamp** | $(date -u '+%Y-%m-%d %H:%M:%S UTC') |"
        echo "| **Job Duration** | $(($(date +%s) - JOB_START_TIME))s |"
        echo ""
        echo "## 🔧 How to Fix"
        echo ""
        echo "$fix_suggestion"
        echo ""
        echo "## 🔍 Environment Information"
        echo ""
        echo "- **Runner OS**: \`${RUNNER_OS:-unknown}\`"
        echo "- **Go Version**: \`$(command -v go >/dev/null 2>&1 && go version || echo 'Not installed')\`"
        echo "- **Working Directory**: \`$(pwd)\`"
        echo "- **Available Tools**: $(for tool in protoc protoc-gen-go protoc-gen-go-grpc; do command -v "$tool" >/dev/null 2>&1 && echo -n "$tool " || true; done)"
        echo ""
        echo "## 📋 Next Steps"
        echo ""
        echo "1. **Investigate the root cause** listed above"
        echo "2. **Apply the suggested fix** or similar resolution"
        echo "3. **Re-run the workflow** to verify the fix"
        echo "4. **Review the diagnostic log** below for additional context"
        echo ""
        echo "## 🗂️ Diagnostic Log"
        echo ""
        echo '<details><summary>Full diagnostic log (click to expand)</summary>'
        echo ""
        echo '```'
        tail -n 200 "$DIAGNOSTIC_LOG_FILE" 2>/dev/null || echo "Diagnostic log unavailable"
        echo '```'
        echo ""
        echo "</details>"
        echo ""
        echo "---"
        echo "*Report generated by Ephemos CI/CD Diagnostic System*"
    } > "$report_file"
    
    log_diagnostic "INFO" "Failure report generated: $report_file"
}

# Step validation with comprehensive checks
validate_step_prerequisites() {
    local step_name="$1"
    shift
    local -a requirements=("$@")
    
    log_diagnostic "INFO" "🔍 Validating prerequisites for step: $step_name"
    
    local validation_failed=0
    
    for requirement in "${requirements[@]}"; do
        case "$requirement" in
            "go")
                if ! command -v go >/dev/null 2>&1; then
                    log_diagnostic "ERROR" "Go toolchain not found in PATH"
                    validation_failed=1
                else
                    local go_version
                    go_version=$(go version | awk '{print $3}' | sed 's/go//')
                    log_diagnostic "SUCCESS" "Go $go_version found at $(which go)"
                fi
                ;;
            "protoc")
                if ! command -v protoc >/dev/null 2>&1; then
                    log_diagnostic "ERROR" "protoc compiler not found in PATH"
                    validation_failed=1
                else
                    local protoc_version
                    protoc_version=$(protoc --version | awk '{print $2}')
                    log_diagnostic "SUCCESS" "protoc $protoc_version found at $(which protoc)"
                fi
                ;;
            "protoc-gen-go")
                if ! command -v protoc-gen-go >/dev/null 2>&1; then
                    log_diagnostic "ERROR" "protoc-gen-go plugin not found in PATH"
                    validation_failed=1
                else
                    log_diagnostic "SUCCESS" "protoc-gen-go found at $(which protoc-gen-go)"
                fi
                ;;
            "protoc-gen-go-grpc")
                if ! command -v protoc-gen-go-grpc >/dev/null 2>&1; then
                    log_diagnostic "ERROR" "protoc-gen-go-grpc plugin not found in PATH"
                    validation_failed=1
                else
                    log_diagnostic "SUCCESS" "protoc-gen-go-grpc found at $(which protoc-gen-go-grpc)"
                fi
                ;;
            "workspace")
                if [[ -z "${GITHUB_WORKSPACE:-}" ]]; then
                    log_diagnostic "ERROR" "GITHUB_WORKSPACE not set"
                    validation_failed=1
                elif [[ ! -d "${GITHUB_WORKSPACE}" ]]; then
                    log_diagnostic "ERROR" "GITHUB_WORKSPACE directory does not exist: ${GITHUB_WORKSPACE}"
                    validation_failed=1
                else
                    log_diagnostic "SUCCESS" "GitHub workspace verified: ${GITHUB_WORKSPACE}"
                fi
                ;;
            "proto-files")
                local proto_files_missing=0
                for proto_file in "examples/proto/echo.pb.go" "examples/proto/echo_grpc.pb.go"; do
                    if [[ ! -f "$proto_file" ]]; then
                        log_diagnostic "ERROR" "Required protobuf file missing: $proto_file"
                        proto_files_missing=1
                    fi
                done
                if [[ $proto_files_missing -eq 0 ]]; then
                    log_diagnostic "SUCCESS" "All required protobuf files present"
                else
                    validation_failed=1
                fi
                ;;
            "go-mod")
                if [[ ! -f "go.mod" ]]; then
                    log_diagnostic "ERROR" "go.mod file not found in repository root"
                    validation_failed=1
                else
                    log_diagnostic "SUCCESS" "go.mod file present"
                fi
                ;;
            *)
                log_diagnostic "WARN" "Unknown prerequisite: $requirement"
                ;;
        esac
    done
    
    if [[ $validation_failed -eq 1 ]]; then
        fail_with_comprehensive_diagnostics 1 \
            "Step prerequisite validation failed for: $step_name" \
            "One or more required tools, files, or environment conditions are missing" \
            "Install missing tools, generate missing files, or fix environment configuration" \
            "Prerequisites: ${requirements[*]}"
    fi
    
    log_diagnostic "SUCCESS" "✅ All prerequisites validated for step: $step_name"
}

# Command execution with comprehensive error handling
execute_with_diagnostics() {
    local command_name="$1"
    local command_description="$2"
    shift 2
    local -a command=("$@")
    
    log_diagnostic "INFO" "🚀 Executing: $command_description"
    log_diagnostic "DEBUG" "Command: ${command[*]}"
    
    local start_time
    start_time=$(date +%s)
    
    local exit_code=0
    local command_output
    
    # Execute command and capture output
    if command_output=$(command "${command[@]}" 2>&1); then
        local end_time
        end_time=$(date +%s)
        local duration=$((end_time - start_time))
        
        log_diagnostic "SUCCESS" "✅ $command_name completed successfully (${duration}s)"
        
        # Log command output in debug mode
        if [[ $DIAGNOSTIC_MODE -ge 1 && -n "$command_output" ]]; then
            log_diagnostic "DEBUG" "Command output:"
            echo "$command_output" | while IFS= read -r line; do
                log_diagnostic "TRACE" "  $line"
            done
        fi
        
        return 0
    else
        exit_code=$?
        local end_time
        end_time=$(date +%s)
        local duration=$((end_time - start_time))
        
        log_diagnostic "ERROR" "❌ $command_name failed (${duration}s, exit code: $exit_code)"
        
        # Always log command output on failure
        if [[ -n "$command_output" ]]; then
            log_diagnostic "ERROR" "Command output:"
            echo "$command_output" | while IFS= read -r line; do
                log_diagnostic "ERROR" "  $line"
            done
        fi
        
        # Analyze common failure patterns and provide specific guidance
        local root_cause="Command execution failed"
        local fix_suggestion="Check command syntax and dependencies"
        
        if echo "$command_output" | grep -q "command not found"; then
            root_cause="Required command not found in PATH"
            fix_suggestion="Install missing command or add it to PATH"
        elif echo "$command_output" | grep -q "permission denied"; then
            root_cause="Permission denied accessing file or directory"
            fix_suggestion="Check file permissions or run with appropriate privileges"
        elif echo "$command_output" | grep -q "no such file or directory"; then
            root_cause="Required file or directory does not exist"
            fix_suggestion="Verify file paths and ensure required files are present"
        elif echo "$command_output" | grep -q "connection refused\|network\|timeout"; then
            root_cause="Network connectivity issue"
            fix_suggestion="Check network connectivity and service availability"
        elif echo "$command_output" | grep -q "out of memory\|killed"; then
            root_cause="Insufficient system resources"
            fix_suggestion="Increase available memory or optimize resource usage"
        fi
        
        fail_with_comprehensive_diagnostics "$exit_code" \
            "$command_name failed: $command_description" \
            "$root_cause" \
            "$fix_suggestion" \
            "Command: ${command[*]}"
    fi
}

# Timing and performance monitoring
monitor_step_performance() {
    local step_name="$1"
    local max_duration_seconds="${2:-300}" # Default 5 minute timeout
    local warning_threshold_seconds="${3:-180}" # Default 3 minute warning
    
    log_diagnostic "INFO" "⏱️ Starting performance monitoring for: $step_name"
    log_diagnostic "DEBUG" "Max duration: ${max_duration_seconds}s, Warning threshold: ${warning_threshold_seconds}s"
    
    local step_start_time
    step_start_time=$(date +%s)
    
    # Return a function that can be called to check/complete the step
    echo "$step_start_time"
}

complete_step_performance_monitoring() {
    local step_name="$1"
    local step_start_time="$2"
    local max_duration_seconds="${3:-300}"
    local warning_threshold_seconds="${4:-180}"
    
    local step_end_time
    step_end_time=$(date +%s)
    local step_duration=$((step_end_time - step_start_time))
    
    if [[ $step_duration -gt $max_duration_seconds ]]; then
        log_diagnostic "ERROR" "⏰ Step '$step_name' exceeded maximum duration: ${step_duration}s > ${max_duration_seconds}s"
        fail_with_comprehensive_diagnostics 124 \
            "Step timeout: $step_name" \
            "Step took longer than maximum allowed duration" \
            "Optimize the step or increase timeout limit" \
            "Duration: ${step_duration}s, Max: ${max_duration_seconds}s"
    elif [[ $step_duration -gt $warning_threshold_seconds ]]; then
        log_diagnostic "WARN" "⚠️ Step '$step_name' took longer than expected: ${step_duration}s > ${warning_threshold_seconds}s"
    else
        log_diagnostic "SUCCESS" "⏱️ Step '$step_name' completed within expected time: ${step_duration}s"
    fi
}

# Artifact validation
validate_build_artifacts() {
    local artifact_type="$1"
    shift
    local -a expected_artifacts=("$@")
    
    log_diagnostic "INFO" "🔍 Validating $artifact_type artifacts"
    
    local validation_failed=0
    
    for artifact in "${expected_artifacts[@]}"; do
        if [[ ! -f "$artifact" ]]; then
            log_diagnostic "ERROR" "Missing artifact: $artifact"
            validation_failed=1
        elif [[ ! -s "$artifact" ]]; then
            log_diagnostic "ERROR" "Empty artifact: $artifact"
            validation_failed=1
        else
            local artifact_size
            artifact_size=$(stat -c%s "$artifact" 2>/dev/null || stat -f%z "$artifact" 2>/dev/null || echo "unknown")
            log_diagnostic "SUCCESS" "Valid artifact: $artifact ($artifact_size bytes)"
        fi
    done
    
    if [[ $validation_failed -eq 1 ]]; then
        fail_with_comprehensive_diagnostics 1 \
            "Build artifact validation failed" \
            "One or more expected build artifacts are missing or empty" \
            "Check build process and ensure all artifacts are generated correctly" \
            "Artifact type: $artifact_type, Expected artifacts: ${expected_artifacts[*]}"
    fi
    
    log_diagnostic "SUCCESS" "✅ All $artifact_type artifacts validated successfully"
}

# Resource monitoring
monitor_system_resources() {
    local context="$1"
    
    log_diagnostic "DEBUG" "🖥️ System resources check: $context"
    
    # Memory usage
    if command -v free >/dev/null 2>&1; then
        local mem_info
        mem_info=$(free -m | awk 'NR==2{printf "Memory: %s/%sMB (%.1f%%)\n", $3,$2,$3*100/$2}')
        log_diagnostic "DEBUG" "$mem_info"
        
        # Check for low memory
        local mem_usage_percent
        mem_usage_percent=$(free | awk 'NR==2{printf "%.1f", $3*100/$2}')
        if (( $(echo "$mem_usage_percent > 90" | bc -l) )); then
            log_diagnostic "WARN" "High memory usage detected: ${mem_usage_percent}%"
        fi
    fi
    
    # Disk usage
    local disk_usage
    disk_usage=$(df -h . | awk 'NR==2{printf "Disk: %s/%s (%s)", $3,$2,$5}')
    log_diagnostic "DEBUG" "$disk_usage"
    
    # Check for low disk space
    local disk_usage_percent
    disk_usage_percent=$(df . | awk 'NR==2{print $5}' | sed 's/%//')
    if [[ ${disk_usage_percent:-0} -gt 80 ]]; then
        log_diagnostic "WARN" "High disk usage detected: ${disk_usage_percent}%"
    fi
    
    # Load average (Linux/macOS)
    if command -v uptime >/dev/null 2>&1; then
        local load_avg
        load_avg=$(uptime | awk -F'load average:' '{print $2}')
        log_diagnostic "DEBUG" "Load average:$load_avg"
    fi
}

# Cleanup function for exit traps
cleanup_diagnostics() {
    local exit_code=$?
    
    if [[ $exit_code -eq 0 ]]; then
        log_diagnostic "SUCCESS" "🎉 Job completed successfully"
    else
        log_diagnostic "ERROR" "💥 Job failed with exit code: $exit_code"
    fi
    
    local total_duration=$(($(date +%s) - JOB_START_TIME))
    log_diagnostic "INFO" "📊 Total job duration: ${total_duration}s"
    
    # Final system resource check
    monitor_system_resources "job-completion"
    
    # If diagnostic log exists, show summary
    if [[ -f "$DIAGNOSTIC_LOG_FILE" ]]; then
        local log_lines
        log_lines=$(wc -l < "$DIAGNOSTIC_LOG_FILE")
        log_diagnostic "INFO" "📋 Diagnostic log contains $log_lines entries"
        
        # Add final summary to GitHub Actions
        if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
            echo "## 📊 Job Summary" >> "$GITHUB_STEP_SUMMARY"
            echo "" >> "$GITHUB_STEP_SUMMARY"
            echo "- **Duration**: ${total_duration}s" >> "$GITHUB_STEP_SUMMARY"
            echo "- **Exit Code**: $exit_code" >> "$GITHUB_STEP_SUMMARY"
            echo "- **Log Entries**: $log_lines" >> "$GITHUB_STEP_SUMMARY"
            echo "- **Status**: $([ $exit_code -eq 0 ] && echo "✅ Success" || echo "❌ Failed")" >> "$GITHUB_STEP_SUMMARY"
            echo "" >> "$GITHUB_STEP_SUMMARY"
        fi
    fi
}

# Set up exit trap for cleanup
trap cleanup_diagnostics EXIT

# Export functions for use in CI scripts
export -f init_diagnostics
export -f log_diagnostic
export -f fail_with_comprehensive_diagnostics
export -f validate_step_prerequisites
export -f execute_with_diagnostics
export -f monitor_step_performance
export -f complete_step_performance_monitoring
export -f validate_build_artifacts
export -f monitor_system_resources

log_diagnostic "INFO" "🔧 CI/CD Diagnostic Library loaded successfully"
log_diagnostic "DEBUG" "Diagnostic mode: $DIAGNOSTIC_MODE, Log file: $DIAGNOSTIC_LOG_FILE"