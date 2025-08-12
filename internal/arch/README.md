# Architectural Boundary Enforcement

This package contains comprehensive tests and runtime validation to enforce the hexagonal architecture boundaries in Ephemos. The goal is to prevent architectural drift and maintain clean separation of concerns.

## Architecture Overview

Ephemos follows a strict hexagonal (ports & adapters) architecture:

```
┌─────────────────────────────────────────┐
│              Adapters (3)               │  ← External world integration
│  ┌─────────────┐    ┌─────────────────┐ │
│  │ Primary     │    │ Secondary       │ │
│  │ (Driving)   │    │ (Driven)        │ │
│  └─────────────┘    └─────────────────┘ │
└─────────────────┬───────────────────────┘
                  │
┌─────────────────▼───────────────────────┐
│             Services (2)                │  ← Business logic orchestration
└─────────────────┬───────────────────────┘
                  │
┌─────────────────▼───────────────────────┐
│              Ports (1)                  │  ← Interfaces/contracts
└─────────────────┬───────────────────────┘
                  │
┌─────────────────▼───────────────────────┐
│            Domain (0)                   │  ← Pure business logic
│         ┌─────────┐  ┌─────────┐        │
│         │ Errors  │  │ Entities│        │
│         └─────────┘  └─────────┘        │
└─────────────────────────────────────────┘
```

**Dependency Rule**: Dependencies flow inward only. Lower-numbered layers cannot depend on higher-numbered layers.

## Boundary Tests

### 1. Import Boundary Tests (`dep_boundary_test.go`)

**Core Protection Tests:**
- `Test_Core_Has_No_Forbidden_Imports`: Ensures core domain has no external dependencies
- `Test_Core_Domain_Has_No_External_Dependencies`: Validates domain purity (stdlib only)
- `Test_Layer_Dependencies`: Enforces layered dependency rules

**Adapter Isolation Tests:**
- `Test_Adapters_Cannot_Import_Other_Adapters`: Prevents direct adapter-to-adapter communication
- `Test_Public_API_Boundary`: Ensures public API doesn't leak internal details

**Structural Integrity Tests:**
- `Test_Circular_Dependencies`: Detects circular import patterns
- `Test_Layer_Dependencies`: Validates proper layering

### 2. Architecture Tests (`architecture_test.go`)

**Protocol Independence Tests:**
- `TestDomainPortsHaveNoProtocolDependencies`: Ports remain protocol-agnostic
- `TestAdaptersCanImportProtocolPackages`: Adapters can use external libraries
- `TestPublicAPIHasNoDirectProtocolDependencies`: Public API stays clean

**API Design Tests:**
- `TestTransportServerIsProtocolAgnostic`: Validates server abstraction
- `TestMountAPIIsGeneric`: Ensures generic mounting interface

### 3. Interface Segregation Tests (`interface_segregation_test.go`)

**Interface Quality Tests:**
- `Test_Interface_Segregation`: Ensures ports follow ISP (max 7 methods)
- `Test_Port_Naming_Conventions`: Validates consistent naming
- `Test_Domain_Types_Are_Pure`: Prevents infrastructure leakage

**Adapter Structure Tests:**
- `Test_Adapter_Interface_Compliance`: Validates adapter implementations

### 4. Runtime Validation (`runtime_validation.go`)

**Runtime Boundary Checks:**
- `ValidateBoundary()`: Call at critical boundary points
- Detects adapter-to-adapter calls at runtime
- Prevents domain calling adapters directly

## Usage Guidelines

### For Developers

**1. Adding New Code:**
```bash
# Run boundary tests before committing
go test ./internal/arch/...

# Run all architectural tests
go test ./internal/core/ports/...
```

**2. Adding Runtime Validation:**
```go
import "github.com/sufield/ephemos/internal/arch"

func (a *MyAdapter) CriticalOperation() error {
    // Validate at boundary crossing points
    if err := arch.ValidateBoundary("critical-operation"); err != nil {
        return fmt.Errorf("boundary violation: %w", err)
    }
    
    // ... rest of implementation
}
```

**3. Checking Violations:**
```go
// In tests or debugging
violations := arch.GetGlobalViolations()
for _, v := range violations {
    log.Printf("Violation: %s", v)
}
```

### For Architects

**1. Adding New Layers:**
Update `layerHierarchy` in `Test_Layer_Dependencies` when adding new architectural layers.

**2. Adding Forbidden Imports:**
Update `getForbiddenPrefixes()` in `dep_boundary_test.go` to add new forbidden dependencies.

**3. Customizing Validation:**
- Modify interface size limits in `checkInterfaceSize()`
- Add new naming conventions in `checkPortNaming()`
- Extend runtime validation rules in `checkCallStackViolation()`

## Enforcement Levels

### Level 1: Compile-Time (Static Analysis)
- **Import boundary tests**: Prevent forbidden imports at build time
- **Circular dependency detection**: Catch cycles before they cause issues
- **Interface validation**: Ensure proper API design

### Level 2: Test-Time (Structural Analysis)
- **Architecture compliance tests**: Validate overall structure
- **Layer dependency tests**: Ensure proper layering
- **Interface segregation tests**: Validate ISP compliance

### Level 3: Runtime (Dynamic Analysis)
- **Boundary validation**: Catch violations during execution
- **Call stack analysis**: Detect architectural violations in real-time
- **Violation reporting**: Track and report boundary crossings

## Performance Considerations

**Runtime Validation Overhead:**
- Enabled by default in development/testing
- Can be disabled in production: `arch.SetGlobalValidationEnabled(false)`
- Minimal overhead when disabled (single boolean check)

**Test Performance:**
- Static tests run quickly (compile-time analysis)
- Structural tests may take longer for large codebases
- Use `go test -short` to skip expensive tests

## Violation Examples and Fixes

### ❌ Bad: Direct Adapter-to-Adapter Communication
```go
// In gRPC adapter
func (g *GrpcAdapter) Process() {
    // WRONG: Direct call to another adapter
    result := httpAdapter.Transform(data)
}
```

### ✅ Good: Communication Through Ports
```go
// In gRPC adapter
func (g *GrpcAdapter) Process() {
    // CORRECT: Use port interface
    result := g.transformPort.Transform(data)
}
```

### ❌ Bad: Domain Importing Adapters
```go
// In domain package
import "github.com/sufield/ephemos/internal/adapters/spiffe"

func (i *Identity) Validate() {
    // WRONG: Domain knows about specific adapter
    return spiffe.ValidateID(i.ID)
}
```

### ✅ Good: Domain Using Ports
```go
// In domain package - no adapter imports needed
func (i *Identity) Validate() error {
    // Domain validation logic only
    if i.ID == "" {
        return errors.New("identity ID cannot be empty")
    }
    return nil
}
```

### ❌ Bad: Large Interface (ISP Violation)
```go
// WRONG: Interface with too many methods
type HugePort interface {
    Method1()
    Method2()
    Method3()
    Method4()
    Method5()
    Method6()
    Method7()
    Method8() // Exceeds recommended limit
}
```

### ✅ Good: Segregated Interfaces
```go
// CORRECT: Small, focused interfaces
type ReaderPort interface {
    Read() ([]byte, error)
}

type WriterPort interface {
    Write([]byte) error
}
```

## Integration with CI/CD

Add these tests to your CI pipeline:

```yaml
# .github/workflows/architecture.yml
- name: Run Architecture Tests
  run: |
    go test -v ./internal/arch/...
    go test -v ./internal/core/ports/...
    
- name: Check Import Boundaries
  run: |
    go test -run TestCoreDependencies ./internal/arch/...
    
- name: Validate Layer Dependencies  
  run: |
    go test -run TestLayerDependencies ./internal/arch/...
```

## Maintenance

**Monthly Reviews:**
- Review `getForbiddenPrefixes()` for new dependencies to restrict
- Check interface sizes are still reasonable
- Validate that new adapters follow patterns

**When Adding Dependencies:**
1. Evaluate if it should be forbidden in core
2. Add to `getForbiddenPrefixes()` if needed
3. Update adapter tests to allow it where appropriate

**When Refactoring:**
1. Run full architecture test suite
2. Check for new circular dependencies
3. Validate layer boundaries are maintained

This system provides comprehensive protection against architectural drift while maintaining development velocity through clear guidelines and automated enforcement.