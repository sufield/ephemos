# Code Quality Tools

This document describes the code quality tools available for Ephemos development and how to use them effectively.

## Variable Shadowing Detection

### Overview
Variable shadowing occurs when a variable in an inner scope has the same name as a variable in an outer scope, potentially hiding the outer variable and leading to bugs. The Go `shadow` analyzer helps detect these cases.

### Installation
```bash
# Install the shadow analyzer (one-time setup)
go install golang.org/x/tools/go/analysis/passes/shadow/cmd/shadow@latest
```

### Usage

#### Basic Usage
```bash
# Check specific package
go vet -vettool=$(which shadow) ./internal/core/services/

# Check multiple packages with wildcard
go vet -vettool=$(which shadow) ./internal/adapters/...

# Check entire codebase
go vet -vettool=$(which shadow) ./...
```

#### Filtering Results
```bash
# Filter to show only shadowing issues (ignore type errors)
go vet -vettool=$(which shadow) ./... 2>&1 | grep "declaration of"

# Check specific packages and filter
go vet -vettool=$(which shadow) ./internal/core/... ./internal/adapters/... 2>&1 | grep "declaration of"
```

### Common Patterns and Fixes

#### Error Variable Shadowing
```go
// ❌ PROBLEM: Shadowing 'err' variable
func processData() error {
    data, err := fetchData()
    if err != nil {
        return err
    }
    
    // This shadows the outer 'err'
    if err := validateData(data); err != nil {
        return err  // Which 'err' is this?
    }
    
    return nil
}

// ✅ SOLUTION: Use specific variable names
func processData() error {
    data, err := fetchData()
    if err != nil {
        return err
    }
    
    // Use a descriptive name
    if validationErr := validateData(data); validationErr != nil {
        return validationErr
    }
    
    return nil
}
```

#### Loop Variable Shadowing
```go
// ❌ PROBLEM: Shadowing loop variable
func processItems(items []Item) error {
    for i, item := range items {
        if err := processItem(item); err != nil {
            // Shadowing loop variables in closure
            go func() {
                log.Printf("Error at index %d: %v", i, err)  // Wrong values!
            }()
        }
    }
    return nil
}

// ✅ SOLUTION: Capture variables explicitly
func processItems(items []Item) error {
    for i, item := range items {
        if processErr := processItem(item); processErr != nil {
            // Capture loop variables
            index, currentErr := i, processErr
            go func() {
                log.Printf("Error at index %d: %v", index, currentErr)
            }()
        }
    }
    return nil
}
```

#### Function Closure Shadowing
```go
// ❌ PROBLEM: Function closure shadows outer variables
func setupConnection() error {
    conn, err := dial()
    if err != nil {
        return err
    }
    
    // Closure that shadows 'err'
    cleanup := func() error {
        if err := conn.Close(); err != nil {  // Shadows outer 'err'
            return err
        }
        return nil
    }
    
    // ... use conn and cleanup
    return cleanup()
}

// ✅ SOLUTION: Use different variable names in closure
func setupConnection() error {
    conn, err := dial()
    if err != nil {
        return err
    }
    
    // Use specific variable name
    cleanup := func() error {
        if closeErr := conn.Close(); closeErr != nil {
            return closeErr
        }
        return nil
    }
    
    // ... use conn and cleanup
    return cleanup()
}
```

### Integration with CI/CD

Add shadowing checks to your development workflow:

#### Pre-commit Hook
```bash
#!/bin/bash
# .git/hooks/pre-commit

echo "Checking for variable shadowing..."
SHADOW_ISSUES=$(go vet -vettool=$(which shadow) ./... 2>&1 | grep "declaration of")

if [ -n "$SHADOW_ISSUES" ]; then
    echo "❌ Variable shadowing detected:"
    echo "$SHADOW_ISSUES"
    echo ""
    echo "Please fix shadowing issues before committing."
    exit 1
fi

echo "✅ No variable shadowing detected"
```

#### Makefile Target
Add to your `Makefile`:
```make
.PHONY: check-shadowing
check-shadowing:
	@echo "Checking for variable shadowing..."
	@if command -v shadow >/dev/null 2>&1; then \
		ISSUES=$$(go vet -vettool=$$(which shadow) ./... 2>&1 | grep "declaration of" || true); \
		if [ -n "$$ISSUES" ]; then \
			echo "❌ Variable shadowing detected:"; \
			echo "$$ISSUES"; \
			exit 1; \
		else \
			echo "✅ No variable shadowing detected"; \
		fi \
	else \
		echo "⚠️ Shadow analyzer not installed. Run: go install golang.org/x/tools/go/analysis/passes/shadow/cmd/shadow@latest"; \
		exit 1; \
	fi

.PHONY: install-shadow
install-shadow:
	go install golang.org/x/tools/go/analysis/passes/shadow/cmd/shadow@latest

# Add to existing quality checks
.PHONY: quality-checks
quality-checks: lint test check-shadowing
	@echo "✅ All quality checks passed"
```

## Architecture Testing

### Overview
Ephemos includes automated architecture tests that enforce design constraints and prevent common architectural violations.

### Running Architecture Tests
```bash
# Run all architecture tests
go test ./internal/arch/ -v

# Run specific test categories
go test ./internal/arch/ -run TestNoLongSelectorChains -v
go test ./internal/arch/ -run TestVendorTypeIsolation -v
go test ./internal/arch/ -run TestNoCrossPackageDeepAccess -v
```

### Architecture Constraints

#### Selector Chain Depth Limits
Prevents excessive chaining that violates the Law of Demeter:
- **Public API**: Maximum 2 levels (`client.connection()`)
- **Core packages**: Maximum 3 levels (`service.config.value()`)
- **Adapters**: Maximum 4 levels (`adapter.config.section.field()`)

#### Vendor Type Isolation
Ensures third-party types don't leak into public APIs:
- `spiffeid.ID` should not appear in public interfaces
- `x509svid.SVID` should be wrapped in domain types
- `tlsconfig.Authorizer` should be abstracted

#### Cross-Package Deep Access
Detects configuration coupling violations:
- Adapters should not directly access `config.Service.Domain`
- Use dependency injection instead of deep config access
- Prefer capability injection over configuration passing

### Example Violations and Fixes

#### Selector Chain Violation
```go
// ❌ VIOLATION: Exceeds depth limit (4 levels in core package)
if service.config.cache.settings.ttlMinutes > 0 {
    // ...
}

// ✅ FIX: Use intermediate variables
cacheConfig := service.config.cache
if cacheConfig != nil && cacheConfig.settings.ttlMinutes > 0 {
    // ...
}
```

#### Vendor Type Leakage
```go
// ❌ VIOLATION: go-spiffe type in public API
func GetIdentity() spiffeid.ID {
    // ...
}

// ✅ FIX: Use domain abstraction
func GetIdentity() *domain.ServiceIdentity {
    // ...
}
```

## Additional Quality Tools

### Go Vet
Standard Go static analysis:
```bash
go vet ./...
```

### golangci-lint
Comprehensive linter with multiple analyzers:
```bash
# Install
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# Run
golangci-lint run
```

### govulncheck
Security vulnerability scanner:
```bash
# Install
go install golang.org/x/vuln/cmd/govulncheck@latest

# Run
govulncheck ./...
```

## Automated Quality Pipeline

Combine all tools in a comprehensive quality check:

```bash
#!/bin/bash
# scripts/quality-check.sh

set -e

echo "🔍 Running comprehensive quality checks..."

echo "1. Running go vet..."
go vet ./...

echo "2. Checking for variable shadowing..."
if command -v shadow >/dev/null 2>&1; then
    SHADOW_ISSUES=$(go vet -vettool=$(which shadow) ./... 2>&1 | grep "declaration of" || true)
    if [ -n "$SHADOW_ISSUES" ]; then
        echo "❌ Variable shadowing detected:"
        echo "$SHADOW_ISSUES"
        exit 1
    fi
    echo "✅ No variable shadowing detected"
else
    echo "⚠️ Shadow analyzer not installed"
fi

echo "3. Running architecture tests..."
go test ./internal/arch/ -v

echo "4. Running unit tests..."
go test ./...

echo "5. Running linter..."
if command -v golangci-lint >/dev/null 2>&1; then
    golangci-lint run
else
    echo "⚠️ golangci-lint not installed"
fi

echo "6. Checking for vulnerabilities..."
if command -v govulncheck >/dev/null 2>&1; then
    govulncheck ./...
else
    echo "⚠️ govulncheck not installed"
fi

echo "✅ All quality checks passed!"
```

## Best Practices

1. **Run quality checks frequently** during development, not just before commits
2. **Install tools locally** for faster feedback loops
3. **Integrate with IDE** for real-time analysis
4. **Fix issues immediately** rather than accumulating technical debt
5. **Use descriptive variable names** to prevent shadowing issues naturally
6. **Follow architecture constraints** from the beginning rather than retrofitting

## IDE Integration

### VS Code
Add to `.vscode/settings.json`:
```json
{
    "go.lintTool": "golangci-lint",
    "go.lintFlags": [
        "--fast"
    ],
    "go.vetFlags": [
        "-vettool=/path/to/shadow"
    ]
}
```

### GoLand/IntelliJ
- Enable Go vet inspections
- Install golangci-lint plugin
- Configure external tools for shadow analyzer

This comprehensive approach to code quality helps maintain a clean, maintainable, and secure codebase.