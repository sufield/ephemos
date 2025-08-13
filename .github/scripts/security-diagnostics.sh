#!/bin/bash
# Enhanced Security Diagnostics for CI/CD
# Provides comprehensive security scanning with detailed failure analysis and remediation guidance

set -euo pipefail

# Source the common diagnostic library
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/ci-diagnostics.sh"

# Security-specific diagnostic functions
validate_security_environment() {
    local job_name="$1"
    local scan_type="${2:-comprehensive}"
    
    init_diagnostics "$job_name" "security-environment-validation"
    
    log_diagnostic "INFO" "🔒 Validating security scanning environment: $scan_type"
    
    # Start performance monitoring
    local perf_start
    perf_start=$(monitor_step_performance "security-environment-validation" 300 120)
    
    # Validate basic security prerequisites
    validate_step_prerequisites "security-environment" \
        "go" "workspace" "go-mod" "proto-files"
    
    # Validate security tool availability
    validate_security_tools "$scan_type"
    
    # Check repository security configuration
    validate_repository_security_config
    
    # Monitor system resources for security scanning
    monitor_system_resources "security-environment-validation"
    
    complete_step_performance_monitoring "security-environment-validation" "$perf_start" 300 120
    
    log_diagnostic "SUCCESS" "✅ Security environment validation completed successfully"
}

validate_security_tools() {
    local scan_type="$1"
    
    log_diagnostic "INFO" "🔍 Validating security tools for scan type: $scan_type"
    
    # Map of security tools and their installation methods
    declare -A security_tools=(
        ["govulncheck"]="golang.org/x/vuln/cmd/govulncheck@latest"
        ["golangci-lint"]="github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
        ["gosec"]="github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"
        ["staticcheck"]="honnef.co/go/tools/cmd/staticcheck@latest"
    )
    
    local missing_tools=()
    local tools_to_check=()
    
    # Determine which tools are needed based on scan type
    case "$scan_type" in
        "vulnerability")
            tools_to_check=("govulncheck")
            ;;
        "static-analysis")
            tools_to_check=("golangci-lint" "gosec" "staticcheck")
            ;;
        "comprehensive")
            tools_to_check=("govulncheck" "golangci-lint" "gosec" "staticcheck")
            ;;
        *)
            tools_to_check=("govulncheck" "golangci-lint")
            ;;
    esac
    
    # Check for each required tool
    for tool in "${tools_to_check[@]}"; do
        if ! command -v "$tool" >/dev/null 2>&1; then
            log_diagnostic "WARN" "Security tool not found: $tool"
            missing_tools+=("$tool")
        else
            local tool_version
            case "$tool" in
                "govulncheck")
                    tool_version=$(govulncheck -h 2>&1 | head -1 || echo "unknown")
                    ;;
                "golangci-lint")
                    tool_version=$(golangci-lint version 2>/dev/null | head -1 || echo "unknown")
                    ;;
                "gosec")
                    tool_version=$(gosec -version 2>/dev/null || echo "unknown")
                    ;;
                "staticcheck")
                    tool_version=$(staticcheck -version 2>/dev/null || echo "unknown")
                    ;;
            esac
            log_diagnostic "SUCCESS" "✅ Security tool found: $tool ($tool_version)"
        fi
    done
    
    # Install missing tools
    if [[ ${#missing_tools[@]} -gt 0 ]]; then
        log_diagnostic "INFO" "Installing missing security tools: ${missing_tools[*]}"
        install_security_tools "${missing_tools[@]}"
    fi
    
    log_diagnostic "SUCCESS" "✅ All required security tools validated"
}

install_security_tools() {
    local -a tools=("$@")
    
    log_diagnostic "INFO" "📦 Installing security tools: ${tools[*]}"
    
    # Security tool installation mappings
    declare -A tool_packages=(
        ["govulncheck"]="golang.org/x/vuln/cmd/govulncheck@latest"
        ["golangci-lint"]="github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
        ["gosec"]="github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"
        ["staticcheck"]="honnef.co/go/tools/cmd/staticcheck@latest"
    )
    
    local installation_failed=0
    
    for tool in "${tools[@]}"; do
        local package="${tool_packages[$tool]:-}"
        
        if [[ -z "$package" ]]; then
            log_diagnostic "ERROR" "Unknown security tool: $tool"
            installation_failed=1
            continue
        fi
        
        log_diagnostic "INFO" "Installing $tool from $package"
        
        # Install with retries
        local max_retries=3
        local installed=false
        
        for attempt in $(seq 1 $max_retries); do
            if execute_with_diagnostics "install-$tool-attempt-$attempt" \
                "Installing $tool (attempt $attempt/$max_retries)" \
                go install "$package"; then
                
                # Verify installation
                if command -v "$tool" >/dev/null 2>&1; then
                    log_diagnostic "SUCCESS" "✅ $tool installed successfully"
                    installed=true
                    break
                else
                    log_diagnostic "WARN" "$tool installed but not found in PATH"
                fi
            fi
            
            if [[ $attempt -lt $max_retries ]]; then
                log_diagnostic "WARN" "$tool installation attempt $attempt failed, retrying..."
                sleep 2
            fi
        done
        
        if [[ "$installed" != "true" ]]; then
            log_diagnostic "ERROR" "Failed to install $tool after $max_retries attempts"
            installation_failed=1
        fi
    done
    
    if [[ $installation_failed -eq 1 ]]; then
        fail_with_comprehensive_diagnostics 1 \
            "Security tool installation failed" \
            "One or more security tools could not be installed" \
            "Check network connectivity, Go proxy settings, and tool availability" \
            "Failed tools may need manual installation or alternative approaches"
    fi
    
    log_diagnostic "SUCCESS" "✅ Security tools installation completed"
}

validate_repository_security_config() {
    log_diagnostic "INFO" "🔍 Validating repository security configuration"
    
    # Check for security-related configuration files
    local security_configs=()
    
    # Check for .golangci.yml or .golangci.yaml
    if [[ -f ".golangci.yml" ]] || [[ -f ".golangci.yaml" ]]; then
        security_configs+=("golangci-lint configuration found")
        log_diagnostic "SUCCESS" "✅ golangci-lint configuration present"
    else
        log_diagnostic "INFO" "No golangci-lint configuration found (will use defaults)"
    fi
    
    # Check for .goreleaser.yml or .goreleaser.yaml  
    if [[ -f ".goreleaser.yml" ]] || [[ -f ".goreleaser.yaml" ]]; then
        security_configs+=("goreleaser configuration found")
        log_diagnostic "SUCCESS" "✅ goreleaser configuration present"
    fi
    
    # Check for security.md or SECURITY.md
    if [[ -f "SECURITY.md" ]] || [[ -f "security.md" ]]; then
        security_configs+=("security policy found")
        log_diagnostic "SUCCESS" "✅ Security policy document present"
    else
        log_diagnostic "INFO" "No security policy document found"
    fi
    
    # Check for GitHub security features
    if [[ -d ".github/workflows" ]]; then
        local security_workflows=()
        
        # Look for security-related workflows
        while IFS= read -r -d '' workflow; do
            local workflow_name
            workflow_name=$(basename "$workflow")
            if grep -q -i -E "(security|codeql|dependabot|vulnerability|gosec)" "$workflow"; then
                security_workflows+=("$workflow_name")
            fi
        done < <(find .github/workflows -name "*.yml" -o -name "*.yaml" -print0)
        
        if [[ ${#security_workflows[@]} -gt 0 ]]; then
            log_diagnostic "SUCCESS" "✅ Security workflows found: ${security_workflows[*]}"
        else
            log_diagnostic "WARN" "No security-focused workflows detected"
        fi
    fi
    
    log_diagnostic "SUCCESS" "✅ Repository security configuration validated"
}

execute_security_scan() {
    local scan_type="${1:-comprehensive}"
    local fail_on_issues="${2:-true}"
    local severity_threshold="${3:-medium}"
    
    log_diagnostic "INFO" "🔒 Executing security scan: $scan_type"
    
    # Start performance monitoring
    local perf_start
    perf_start=$(monitor_step_performance "security-scan-$scan_type" 1800 900)
    
    # Pre-scan validation
    pre_security_scan_validation "$scan_type"
    
    # Initialize scan results tracking
    declare -A scan_results=()
    local overall_scan_status=0
    
    # Execute scans based on type
    case "$scan_type" in
        "vulnerability")
            execute_vulnerability_scan scan_results
            ;;
        "static-analysis")
            execute_static_analysis_scan scan_results
            ;;
        "dependency")
            execute_dependency_scan scan_results
            ;;
        "comprehensive")
            execute_vulnerability_scan scan_results
            execute_static_analysis_scan scan_results
            execute_dependency_scan scan_results
            execute_secrets_scan scan_results
            ;;
        *)
            fail_with_comprehensive_diagnostics 1 \
                "Unknown security scan type: $scan_type" \
                "Invalid scan type specified" \
                "Use one of: vulnerability, static-analysis, dependency, comprehensive" \
                "Available scan types defined in security-diagnostics.sh"
            ;;
    esac
    
    # Generate comprehensive security report
    generate_security_report scan_results "$scan_type" "$severity_threshold"
    
    # Determine overall scan status
    evaluate_scan_results scan_results "$fail_on_issues" "$severity_threshold" overall_scan_status
    
    complete_step_performance_monitoring "security-scan-$scan_type" "$perf_start" 1800 900
    
    if [[ $overall_scan_status -eq 0 ]]; then
        log_diagnostic "SUCCESS" "✅ Security scan completed successfully: $scan_type"
    else
        log_diagnostic "ERROR" "❌ Security scan found issues: $scan_type"
        
        if [[ "$fail_on_issues" == "true" ]]; then
            fail_with_comprehensive_diagnostics $overall_scan_status \
                "Security scan found critical issues" \
                "Security vulnerabilities or issues detected that exceed threshold" \
                "Review security report and fix identified issues before proceeding" \
                "Scan type: $scan_type, Threshold: $severity_threshold"
        fi
    fi
}

pre_security_scan_validation() {
    local scan_type="$1"
    
    log_diagnostic "INFO" "🔍 Pre-security scan validation for: $scan_type"
    
    # Ensure clean build state for accurate security scanning
    log_diagnostic "DEBUG" "Cleaning build artifacts for security scan"
    go clean -cache -testcache -modcache >/dev/null 2>&1 || true
    
    # Download dependencies for security scanning
    if ! execute_with_diagnostics "pre-scan-mod-download" \
        "Downloading dependencies for security scanning" \
        go mod download; then
        fail_with_comprehensive_diagnostics 1 \
            "Failed to download dependencies for security scanning" \
            "Network connectivity or module proxy issues" \
            "Check network connectivity and GOPROXY settings" \
            "Security scanning requires all dependencies to be available"
    fi
    
    # Verify module integrity before scanning
    if ! execute_with_diagnostics "pre-scan-mod-verify" \
        "Verifying module integrity" \
        go mod verify; then
        fail_with_comprehensive_diagnostics 1 \
            "Module integrity verification failed" \
            "Dependencies have been tampered with or corrupted" \
            "Check dependency sources and consider running 'go mod tidy'" \
            "This is a critical security issue - do not proceed with compromised dependencies"
    fi
    
    log_diagnostic "SUCCESS" "✅ Pre-security scan validation completed"
}

execute_vulnerability_scan() {
    local -n results_ref=$1
    
    log_diagnostic "INFO" "🔍 Executing vulnerability scan with govulncheck"
    
    local vuln_output
    local vuln_exit_code=0
    
    if vuln_output=$(govulncheck -json ./... 2>&1); then
        log_diagnostic "SUCCESS" "✅ Vulnerability scan completed - no vulnerabilities found"
        results_ref["vulnerability"]="clean"
        
        # Save clean scan result
        echo "$vuln_output" > "vulnerability-scan-results.json"
        
    else
        vuln_exit_code=$?
        log_diagnostic "WARN" "⚠️ Vulnerability scan found issues"
        
        # Save vulnerability scan results
        echo "$vuln_output" > "vulnerability-scan-results.json"
        
        # Parse vulnerability results
        parse_vulnerability_results "$vuln_output"
        
        results_ref["vulnerability"]="issues_found:$vuln_exit_code"
    fi
}

parse_vulnerability_results() {
    local vuln_output="$1"
    
    log_diagnostic "INFO" "📋 Analyzing vulnerability scan results"
    
    # Count different types of vulnerabilities
    local critical_count=0
    local high_count=0
    local medium_count=0
    local low_count=0
    local info_count=0
    
    # Parse JSON output if it's valid JSON
    if echo "$vuln_output" | jq empty 2>/dev/null; then
        log_diagnostic "DEBUG" "Parsing structured vulnerability data"
        
        # Extract vulnerability information from JSON
        while IFS= read -r vuln_info; do
            if [[ -n "$vuln_info" ]]; then
                log_diagnostic "ERROR" "🚨 VULNERABILITY: $vuln_info"
                
                # Categorize by severity (basic heuristic)
                if echo "$vuln_info" | grep -qi "critical"; then
                    critical_count=$((critical_count + 1))
                elif echo "$vuln_info" | grep -qi "high"; then
                    high_count=$((high_count + 1))
                elif echo "$vuln_info" | grep -qi "medium"; then
                    medium_count=$((medium_count + 1))
                elif echo "$vuln_info" | grep -qi "low"; then
                    low_count=$((low_count + 1))
                else
                    info_count=$((info_count + 1))
                fi
            fi
        done < <(echo "$vuln_output" | jq -r '.vulns[]?.details // empty' 2>/dev/null || echo "")
    else
        # Parse text output
        log_diagnostic "DEBUG" "Parsing text vulnerability data"
        
        while IFS= read -r line; do
            if echo "$line" | grep -q "Vulnerability"; then
                log_diagnostic "ERROR" "🚨 $line"
                medium_count=$((medium_count + 1))  # Default to medium for text parsing
            fi
        done <<< "$vuln_output"
    fi
    
    # Log vulnerability summary
    local total_vulns=$((critical_count + high_count + medium_count + low_count + info_count))
    
    log_diagnostic "ERROR" "📊 Vulnerability Summary:"
    log_diagnostic "ERROR" "  🔴 Critical: $critical_count"
    log_diagnostic "ERROR" "  🟠 High: $high_count"
    log_diagnostic "ERROR" "  🟡 Medium: $medium_count"
    log_diagnostic "ERROR" "  🟢 Low: $low_count"
    log_diagnostic "ERROR" "  ℹ️ Info: $info_count"
    log_diagnostic "ERROR" "  📈 Total: $total_vulns"
    
    # Add to GitHub Actions summary
    if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
        {
            echo "## 🚨 Vulnerability Scan Results"
            echo ""
            echo "| Severity | Count |"
            echo "|----------|-------|"
            echo "| 🔴 Critical | $critical_count |"
            echo "| 🟠 High | $high_count |"
            echo "| 🟡 Medium | $medium_count |"
            echo "| 🟢 Low | $low_count |"
            echo "| ℹ️ Info | $info_count |"
            echo "| **Total** | **$total_vulns** |"
            echo ""
        } >> "$GITHUB_STEP_SUMMARY"
    fi
}

execute_static_analysis_scan() {
    local -n results_ref=$1
    
    log_diagnostic "INFO" "🔍 Executing static analysis scan"
    
    # Run golangci-lint
    execute_golangci_lint_scan results_ref
    
    # Run gosec if available
    if command -v gosec >/dev/null 2>&1; then
        execute_gosec_scan results_ref
    fi
    
    # Run staticcheck if available
    if command -v staticcheck >/dev/null 2>&1; then
        execute_staticcheck_scan results_ref
    fi
}

execute_golangci_lint_scan() {
    local -n results_ref=$1
    
    log_diagnostic "INFO" "🔍 Running golangci-lint static analysis"
    
    local lint_output
    local lint_exit_code=0
    
    # Configure golangci-lint for security-focused scanning
    local lint_args=(
        "run"
        "--timeout=10m"
        "--issues-exit-code=1"
        "--enable=gosec,gocritic,gocyclo,goconst,goimports,misspell"
        "--enable=ineffassign,staticcheck,unused,errcheck"
        "--exclude-use-default=false"
        "--max-issues-per-linter=50"
        "--max-same-issues=10"
    )
    
    if lint_output=$(golangci-lint "${lint_args[@]}" 2>&1); then
        log_diagnostic "SUCCESS" "✅ golangci-lint scan completed - no issues found"
        results_ref["golangci-lint"]="clean"
    else
        lint_exit_code=$?
        log_diagnostic "WARN" "⚠️ golangci-lint found issues"
        
        # Parse and categorize issues
        parse_golangci_lint_results "$lint_output"
        
        results_ref["golangci-lint"]="issues_found:$lint_exit_code"
        
        # Save results
        echo "$lint_output" > "golangci-lint-results.txt"
    fi
}

parse_golangci_lint_results() {
    local lint_output="$1"
    
    log_diagnostic "INFO" "📋 Analyzing golangci-lint results"
    
    local issue_count=0
    local security_issues=0
    local performance_issues=0
    local style_issues=0
    
    while IFS= read -r line; do
        if [[ "$line" =~ ^[^:]+:[0-9]+:[0-9]+: ]]; then
            issue_count=$((issue_count + 1))
            
            # Categorize by linter type
            if echo "$line" | grep -q "gosec\|security"; then
                security_issues=$((security_issues + 1))
                log_diagnostic "ERROR" "🔒 SECURITY: $line"
            elif echo "$line" | grep -q "performance\|ineffassign\|gocyclo"; then
                performance_issues=$((performance_issues + 1))
                log_diagnostic "WARN" "⚡ PERFORMANCE: $line"
            else
                style_issues=$((style_issues + 1))
                log_diagnostic "INFO" "📝 STYLE: $line"
            fi
        fi
    done <<< "$lint_output"
    
    log_diagnostic "INFO" "📊 golangci-lint Summary:"
    log_diagnostic "INFO" "  🔒 Security Issues: $security_issues"
    log_diagnostic "INFO" "  ⚡ Performance Issues: $performance_issues"
    log_diagnostic "INFO" "  📝 Style Issues: $style_issues"
    log_diagnostic "INFO" "  📈 Total Issues: $issue_count"
}

execute_gosec_scan() {
    local -n results_ref=$1
    
    log_diagnostic "INFO" "🔍 Running gosec security scan"
    
    local gosec_output
    local gosec_exit_code=0
    
    if gosec_output=$(gosec -fmt json -out gosec-results.json ./... 2>&1); then
        log_diagnostic "SUCCESS" "✅ gosec scan completed"
        
        # Parse gosec JSON results
        if [[ -f "gosec-results.json" ]]; then
            parse_gosec_results "gosec-results.json"
        fi
        
        results_ref["gosec"]="completed"
    else
        gosec_exit_code=$?
        log_diagnostic "WARN" "⚠️ gosec scan issues or findings"
        
        results_ref["gosec"]="issues_found:$gosec_exit_code"
        
        # Parse results even on failure
        if [[ -f "gosec-results.json" ]]; then
            parse_gosec_results "gosec-results.json"
        fi
    fi
}

parse_gosec_results() {
    local results_file="$1"
    
    if [[ ! -f "$results_file" ]]; then
        log_diagnostic "WARN" "gosec results file not found: $results_file"
        return
    fi
    
    log_diagnostic "INFO" "📋 Analyzing gosec security results"
    
    # Extract issue count from JSON
    if command -v jq >/dev/null 2>&1; then
        local issue_count
        issue_count=$(jq '.Issues | length' "$results_file" 2>/dev/null || echo "0")
        
        if [[ $issue_count -gt 0 ]]; then
            log_diagnostic "WARN" "🔒 gosec found $issue_count security issues"
            
            # Extract and log individual issues
            jq -r '.Issues[] | "🚨 \(.severity) (\(.rule_id)): \(.details) at \(.file):\(.line)"' "$results_file" 2>/dev/null | while IFS= read -r issue; do
                log_diagnostic "ERROR" "$issue"
            done
        else
            log_diagnostic "SUCCESS" "✅ gosec found no security issues"
        fi
    else
        log_diagnostic "WARN" "jq not available - cannot parse gosec JSON results"
    fi
}

execute_staticcheck_scan() {
    local -n results_ref=$1
    
    log_diagnostic "INFO" "🔍 Running staticcheck analysis"
    
    local staticcheck_output
    local staticcheck_exit_code=0
    
    if staticcheck_output=$(staticcheck ./... 2>&1); then
        log_diagnostic "SUCCESS" "✅ staticcheck scan completed - no issues found"
        results_ref["staticcheck"]="clean"
    else
        staticcheck_exit_code=$?
        log_diagnostic "WARN" "⚠️ staticcheck found issues"
        
        # Count and categorize staticcheck issues
        local issue_count=0
        while IFS= read -r line; do
            if [[ -n "$line" && "$line" != *"staticcheck"* ]]; then
                issue_count=$((issue_count + 1))
                log_diagnostic "WARN" "📊 STATIC: $line"
            fi
        done <<< "$staticcheck_output"
        
        log_diagnostic "INFO" "📊 staticcheck found $issue_count issues"
        results_ref["staticcheck"]="issues_found:$staticcheck_exit_code:$issue_count"
        
        # Save results
        echo "$staticcheck_output" > "staticcheck-results.txt"
    fi
}

execute_dependency_scan() {
    local -n results_ref=$1
    
    log_diagnostic "INFO" "🔍 Executing dependency security scan"
    
    # Check for known vulnerable dependencies
    execute_mod_vulnerability_check results_ref
    
    # Check for license compliance
    execute_license_scan results_ref
}

execute_mod_vulnerability_check() {
    local -n results_ref=$1
    
    log_diagnostic "INFO" "🔍 Checking dependencies for known vulnerabilities"
    
    # This is covered by govulncheck, but we can add additional checks here
    log_diagnostic "DEBUG" "Dependency vulnerability check integrated with govulncheck"
    results_ref["dependency-vulns"]="integrated"
}

execute_license_scan() {
    local -n results_ref=$1
    
    log_diagnostic "INFO" "🔍 Scanning dependency licenses"
    
    # Check if go-licenses is available
    if command -v go-licenses >/dev/null 2>&1; then
        local license_output
        local license_exit_code=0
        
        if license_output=$(go-licenses check ./... 2>&1); then
            log_diagnostic "SUCCESS" "✅ License scan completed - no issues found"
            results_ref["licenses"]="clean"
        else
            license_exit_code=$?
            log_diagnostic "WARN" "⚠️ License scan found potential issues"
            
            # Parse license issues
            parse_license_results "$license_output"
            
            results_ref["licenses"]="issues_found:$license_exit_code"
        fi
    else
        log_diagnostic "INFO" "go-licenses not available - skipping license scan"
        results_ref["licenses"]="skipped"
    fi
}

parse_license_results() {
    local license_output="$1"
    
    log_diagnostic "INFO" "📋 Analyzing license scan results"
    
    local license_issues=0
    
    while IFS= read -r line; do
        if [[ "$line" =~ [Ee]rror ]] || [[ "$line" =~ [Ww]arn ]]; then
            license_issues=$((license_issues + 1))
            log_diagnostic "WARN" "📄 LICENSE: $line"
        fi
    done <<< "$license_output"
    
    if [[ $license_issues -gt 0 ]]; then
        log_diagnostic "WARN" "📊 Found $license_issues license-related issues"
    fi
}

execute_secrets_scan() {
    local -n results_ref=$1
    
    log_diagnostic "INFO" "🔍 Scanning for exposed secrets"
    
    # Basic secrets scanning using git and grep
    local secrets_found=0
    
    # Common secret patterns
    local secret_patterns=(
        "password\s*=\s*['\"][^'\"]*['\"]"
        "api[_-]?key\s*=\s*['\"][^'\"]*['\"]"
        "secret\s*=\s*['\"][^'\"]*['\"]"
        "token\s*=\s*['\"][^'\"]*['\"]"
        "-----BEGIN.*PRIVATE KEY-----"
        "ssh-rsa\s+[A-Za-z0-9+/]+"
    )
    
    for pattern in "${secret_patterns[@]}"; do
        local matches
        matches=$(grep -r -i -E "$pattern" --exclude-dir=.git --exclude-dir=bin --exclude-dir=dist . || true)
        
        if [[ -n "$matches" ]]; then
            secrets_found=$((secrets_found + 1))
            log_diagnostic "ERROR" "🔐 POTENTIAL SECRET DETECTED:"
            echo "$matches" | while IFS= read -r match; do
                log_diagnostic "ERROR" "  $match"
            done
        fi
    done
    
    if [[ $secrets_found -eq 0 ]]; then
        log_diagnostic "SUCCESS" "✅ No exposed secrets detected"
        results_ref["secrets"]="clean"
    else
        log_diagnostic "ERROR" "❌ Found $secrets_found potential secret exposures"
        results_ref["secrets"]="issues_found:$secrets_found"
    fi
}

generate_security_report() {
    local -n results_ref=$1
    local scan_type="$2"
    local severity_threshold="$3"
    
    log_diagnostic "INFO" "📊 Generating comprehensive security report"
    
    local report_file="security-report.md"
    
    {
        echo "# 🔒 Security Scan Report"
        echo ""
        echo "**Scan Type**: \`$scan_type\`  "
        echo "**Severity Threshold**: \`$severity_threshold\`  "
        echo "**Timestamp**: $(date -u '+%Y-%m-%d %H:%M:%S UTC')  "
        echo "**Repository**: \`${GITHUB_REPOSITORY:-unknown}\`  "
        echo "**Workflow Run**: [\#${GITHUB_RUN_NUMBER:-unknown}](${GITHUB_SERVER_URL:-https://github.com}/${GITHUB_REPOSITORY:-unknown}/actions/runs/${GITHUB_RUN_ID:-unknown})  "
        echo ""
        
        echo "## 📊 Scan Results Summary"
        echo ""
        echo "| Component | Status | Details |"
        echo "|-----------|---------|---------|"
        
        for scan_component in "${!results_ref[@]}"; do
            local result="${results_ref[$scan_component]}"
            local status_icon="❓"
            local status_text="Unknown"
            local details=""
            
            if [[ "$result" == "clean" ]]; then
                status_icon="✅"
                status_text="Clean"
                details="No issues found"
            elif [[ "$result" =~ ^issues_found: ]]; then
                status_icon="⚠️"
                status_text="Issues Found"
                details="${result#issues_found:}"
            elif [[ "$result" == "skipped" ]]; then
                status_icon="⏭️"
                status_text="Skipped"
                details="Not applicable"
            elif [[ "$result" == "completed" ]]; then
                status_icon="✅"
                status_text="Completed"
                details="Scan finished"
            fi
            
            echo "| **$scan_component** | $status_icon $status_text | \`$details\` |"
        done
        
        echo ""
        
        # Add remediation section if issues found
        local has_issues=false
        for result in "${results_ref[@]}"; do
            if [[ "$result" =~ ^issues_found: ]]; then
                has_issues=true
                break
            fi
        done
        
        if [[ "$has_issues" == "true" ]]; then
            echo "## 🛠️ Remediation Guidance"
            echo ""
            echo "### Immediate Actions Required"
            echo ""
            
            for scan_component in "${!results_ref[@]}"; do
                local result="${results_ref[$scan_component]}"
                
                if [[ "$result" =~ ^issues_found: ]]; then
                    case "$scan_component" in
                        "vulnerability")
                            echo "- **Vulnerability Issues**: Update dependencies to patched versions"
                            echo "  - Run \`go get -u ./...\` to update to latest versions"
                            echo "  - Check vulnerability database at https://vuln.go.dev/"
                            ;;
                        "golangci-lint")
                            echo "- **Static Analysis Issues**: Fix code quality and security issues"
                            echo "  - Run \`golangci-lint run\` locally to see specific issues"
                            echo "  - Consider using \`golangci-lint run --fix\` for auto-fixes"
                            ;;
                        "gosec")
                            echo "- **Security Issues**: Address security vulnerabilities in code"
                            echo "  - Review gosec-results.json for detailed security findings"
                            echo "  - Follow security best practices for Go development"
                            ;;
                        "secrets")
                            echo "- **Secret Exposure**: Remove exposed secrets from code"
                            echo "  - Use environment variables or secure secret management"
                            echo "  - Consider git history cleanup if secrets were committed"
                            ;;
                    esac
                    echo ""
                fi
            done
            
            echo "### Security Best Practices"
            echo ""
            echo "1. **Regular Updates**: Keep dependencies updated to latest secure versions"
            echo "2. **Code Review**: Implement security-focused code review processes"
            echo "3. **Secret Management**: Use proper secret management solutions"
            echo "4. **Monitoring**: Implement continuous security monitoring"
            echo "5. **Training**: Ensure team is trained on secure coding practices"
            echo ""
        fi
        
        echo "## 📋 Artifacts Generated"
        echo ""
        echo "The following security scan artifacts are available:"
        echo ""
        
        local artifacts=("vulnerability-scan-results.json" "golangci-lint-results.txt" "gosec-results.json" "staticcheck-results.txt")
        for artifact in "${artifacts[@]}"; do
            if [[ -f "$artifact" ]]; then
                local artifact_size
                artifact_size=$(stat -c%s "$artifact" 2>/dev/null || stat -f%z "$artifact" 2>/dev/null || echo "unknown")
                echo "- \`$artifact\` ($artifact_size bytes)"
            fi
        done
        echo ""
        
        echo "---"
        echo "*Report generated by Ephemos Security Diagnostics*"
        
    } > "$report_file"
    
    log_diagnostic "INFO" "📊 Security report generated: $report_file"
    
    # Add report summary to GitHub Actions summary
    if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
        cat "$report_file" >> "$GITHUB_STEP_SUMMARY"
    fi
}

evaluate_scan_results() {
    local -n results_ref=$1
    local fail_on_issues="$2"
    local severity_threshold="$3"
    local -n status_ref=$4
    
    log_diagnostic "INFO" "📊 Evaluating security scan results"
    
    local critical_issues=0
    local high_issues=0
    local medium_issues=0
    local total_issues=0
    
    # Analyze results for severity
    for scan_component in "${!results_ref[@]}"; do
        local result="${results_ref[$scan_component]}"
        
        if [[ "$result" =~ ^issues_found: ]]; then
            case "$scan_component" in
                "vulnerability")
                    critical_issues=$((critical_issues + 1))
                    ;;
                "gosec"|"secrets")
                    high_issues=$((high_issues + 1))
                    ;;
                *)
                    medium_issues=$((medium_issues + 1))
                    ;;
            esac
            total_issues=$((total_issues + 1))
        fi
    done
    
    log_diagnostic "INFO" "📊 Issue severity breakdown:"
    log_diagnostic "INFO" "  🔴 Critical: $critical_issues"
    log_diagnostic "INFO" "  🟠 High: $high_issues"
    log_diagnostic "INFO" "  🟡 Medium: $medium_issues"
    log_diagnostic "INFO" "  📈 Total: $total_issues"
    
    # Determine if scan should fail based on threshold
    local should_fail=false
    
    case "$severity_threshold" in
        "critical")
            [[ $critical_issues -gt 0 ]] && should_fail=true
            ;;
        "high")
            [[ $((critical_issues + high_issues)) -gt 0 ]] && should_fail=true
            ;;
        "medium")
            [[ $total_issues -gt 0 ]] && should_fail=true
            ;;
        "low"|*)
            [[ $total_issues -gt 0 ]] && should_fail=true
            ;;
    esac
    
    if [[ "$should_fail" == "true" && "$fail_on_issues" == "true" ]]; then
        status_ref=1
        log_diagnostic "ERROR" "❌ Security scan failed - issues exceed severity threshold: $severity_threshold"
    else
        status_ref=0
        if [[ "$should_fail" == "true" ]]; then
            log_diagnostic "WARN" "⚠️ Security issues found but not failing due to configuration"
        else
            log_diagnostic "SUCCESS" "✅ Security scan passed - no issues exceed severity threshold"
        fi
    fi
}

# Export security-specific functions
export -f validate_security_environment
export -f execute_security_scan
export -f validate_security_tools
export -f install_security_tools
export -f parse_vulnerability_results
export -f generate_security_report

log_diagnostic "INFO" "🔒 Security Diagnostics Library loaded successfully"