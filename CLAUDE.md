# Claude Instructions

## Project Context
Ephemos is a Go-based identity authentication system using SPIFFE/SPIRE for service identity management. The project follows hexagonal architecture principles with a clear separation between domain, ports, and adapters.

## Development Rules

Claude must **never** add any backward compatible code.  
All code written must be **forward-only** and must not include deprecated, legacy, or transitional support for older versions.

### Core Rules:
1. **No feature flags for backward compatibility**.
2. **No conditional code paths for legacy behavior**.
3. **No deprecated APIs or functions should be referenced**.
4. **All new code must assume only the latest version** of the system/environment.
5. If backward compatibility is requested, respond with:  
   `"Backward compatibility is not allowed as per project rules."`

### Dependency Management:
6. Always check for the **latest stable version** of all dependencies and install only stable versions.  
   - Do not use alpha, beta, or release candidate versions.  
   - Automatically reject unstable or experimental releases.
7. Minimize writing code from scratch by using **open source Go libraries** that are **active and well maintained**.  
   - Prefer libraries with frequent updates and strong community support.  
   - Avoid unmaintained or abandoned projects.

### Code Quality:
8. Follow **Go coding best practices**.  
   - Use idiomatic Go style (naming, error handling, simplicity).  
   - Avoid code smells such as primitive obsession, deep nesting, and excessive coupling.  
   - Write clear, maintainable, and testable code.
   - Follow the Law of Demeter - avoid long selector chains (e.g., `a.b.c.d()`).
   - Extract intermediate variables to improve readability.

### Scope:
9. **Focus only on identity-based authentication**. **Authorization is out of scope**.  
   - Do not implement RBAC/ABAC, policy engines, or permission checks.  
   - If authorization is requested, respond with:  
     `"Authorization is out of scope as per project rules."`
   - Identity authentication includes: SPIFFE/SPIRE integration, mTLS, certificate management, identity validation.

## Architecture Guidelines

### Hexagonal Architecture
- **Domain Core**: Pure business logic with zero external dependencies
- **Ports**: Interface definitions (contracts)
- **Adapters**: Implementation of external integrations
- Dependencies flow: `adapters ’ ports ’ domain`

### Selector Chain Depth Limits
Per architectural constraints (see `internal/arch/selector_depth_test.go`):
- **Public API** (`pkg/`): Maximum 2 levels
- **Core packages** (`internal/core/`): Maximum 3 levels  
- **Adapter packages** (`internal/adapters/`): Maximum 4 levels

### Testing Requirements
- Run `make test` for unit tests
- Run `go test ./internal/arch/ -v` for architecture tests
- Run `make check-shadowing` to detect variable shadowing
- Run `make quality-checks` for comprehensive quality validation

## Code Quality Tools

### Before Committing:
1. **Format code**: `make fmt`
2. **Run tests**: `make test`
3. **Check architecture**: `go test ./internal/arch/ -v`
4. **Check shadowing**: `make check-shadowing`
5. **Lint code**: `make lint`

### Variable Shadowing
Avoid variable shadowing by using specific names:
```go
// L Bad
cert, err := getCertificate()
if err := validateCert(cert); err != nil {  // shadows outer 'err'

//  Good  
cert, err := getCertificate()
if validationErr := validateCert(cert); validationErr != nil {
```

## Commands and Workflows

### Building
```bash
make build          # Build with reproducible build flags
make clean          # Clean artifacts
make examples       # Build examples
```

### Testing
```bash
make test           # Run all tests
make arch-test      # Run architecture tests
make check-shadowing # Check for variable shadowing
make quality-checks # Run all quality checks
```

### Demo
```bash
make demo           # Run 5-minute demo
make demo-force     # Force reinstall SPIRE and run demo
```

## Important Files

### Configuration
- `config/ephemos.yaml` - Main configuration file
- `internal/core/ports/configuration.go` - Configuration structure

### Architecture Tests
- `internal/arch/selector_depth_test.go` - Selector chain constraints
- `docs/architecture/adr-004-selector-chain-constraints.md` - ADR for constraints
- `docs/architecture/architecture-testing.md` - Architecture testing guide

### Contributing
- `docs/contributing/CONTRIBUTING.md` - Contributing guide
- `docs/contributing/CODE_QUALITY_TOOLS.md` - Quality tools documentation
- `scripts/check-shadowing.sh` - Variable shadowing detection script

## Environment
- **Go Version**: 1.23+ (1.24.5+ recommended)
- **OS**: Linux (Ubuntu 24 optimized)
- **Build System**: Make with modular makefiles
- **Architecture**: Hexagonal/Clean Architecture
- **Identity Provider**: SPIFFE/SPIRE

---
This document enforces a **strict forward-only development policy**, **stable dependency usage**, **efficient reuse of maintained libraries**, **adherence to Go best practices**, and a **narrow scope on identity-based authentication only**.