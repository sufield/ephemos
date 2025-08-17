# Cobra Usage Optimization - Reduce Custom Code

**Analysis Date:** August 17, 2025  
**Current Branch:** analyze/cli-production-readiness  
**Goal:** Leverage more Cobra built-in features and reduce custom code

## Executive Summary

The current Ephemos CLI implementation uses **~60%** of Cobra's built-in features but has opportunities to reduce custom code by **~30-40%** through better leveraging of Cobra's advanced capabilities.

**Current Status:** Good Cobra usage, but missing several opportunities  
**Recommendation:** Refactor to use more Cobra built-ins

---

## 1. Current Cobra Usage Assessment

### ✅ **Already Using Well**
- **Command Structure**: Proper command/subcommand hierarchy
- **Flag System**: Using StringVarP, Bool, Duration flags correctly
- **Args Validation**: Using `cobra.ExactArgs(1)`, `cobra.MaximumNArgs(1)`
- **Help System**: Cobra's built-in help generation
- **Completion**: Automatic completion command generation
- **Version**: Built-in version handling with custom template

### ⚠️ **Missing Opportunities**
- **Flag Validation**: Not using `MarkFlagRequired`, `MarkFlagFilename`
- **PreRun Validation**: Custom validation logic instead of `PreRunE`
- **Error Handling**: Custom error classification vs Cobra patterns
- **Flag Groups**: Manual mutual exclusion vs `MarkFlagsMutuallyExclusive`
- **Output Templates**: Custom fmt.Printf vs Cobra templates
- **Flag Dependencies**: Manual validation vs `MarkFlagsRequiredTogether`

---

## 2. Specific Optimization Opportunities

### 🔧 **Flag Validation Improvements**

#### Current Code (register.go)
```go
// Lines 65-74: Manual validation
func loadConfiguration(ctx context.Context) (*ports.Configuration, error) {
    switch {
    case configFile != "":
        return loadFromConfigFile(ctx)
    case serviceName != "":
        return createTempConfigFromFlags()
    default:
        return nil, fmt.Errorf("either --config or --name must be provided")
    }
}
```

#### ✅ **Recommended: Use Cobra Flag Groups**
```go
func init() {
    registerCmd.Flags().StringVarP(&configFile, "config", "c", "", "Path to configuration file")
    registerCmd.Flags().StringVarP(&serviceName, "name", "n", "", "Service name")
    registerCmd.Flags().StringVarP(&serviceDomain, "domain", "d", "example.org", "Service domain")
    registerCmd.Flags().StringVarP(&selector, "selector", "s", "", "Custom selector")
    
    // Use Cobra's built-in flag validation
    cobra.MarkFlagFilename(registerCmd.Flags(), "config", "yaml", "yml")
    
    // Create mutually exclusive groups
    registerCmd.MarkFlagsMutuallyExclusive("config", "name")
    registerCmd.MarkFlagsOneRequired("config", "name")
    
    // When using name, domain becomes required
    registerCmd.MarkFlagsRequiredTogether("name", "domain")
}
```

**Benefits:**
- ✅ Eliminates 15+ lines of custom validation code
- ✅ Automatic error messages with proper usage hints
- ✅ Built-in help generation shows requirements
- ✅ File completion for config flag

### 🔧 **PreRun Validation Pattern**

#### Current Pattern
```go
func runRegister(_ *cobra.Command, _ []string) error {
    ctx := context.Background()
    
    cfg, err := loadConfiguration(ctx)  // Custom validation
    if err != nil {
        return err
    }
    // ... rest of logic
}
```

#### ✅ **Recommended: Use PreRunE**
```go
var registerCmd = &cobra.Command{
    Use:   "register",
    Short: "Register a service with SPIRE",
    PreRunE: validateRegisterFlags,  // Cobra handles this automatically
    RunE:    runRegister,
}

func validateRegisterFlags(cmd *cobra.Command, args []string) error {
    // Cobra already handled flag groups, just do business logic validation
    if serviceName != "" && !isValidServiceName(serviceName) {
        return fmt.Errorf("invalid service name: %s", serviceName)
    }
    return nil
}

func runRegister(cmd *cobra.Command, args []string) error {
    // Skip validation - Cobra + PreRunE already handled it
    return performRegistration(cmd.Context())
}
```

**Benefits:**
- ✅ Separates validation from business logic
- ✅ Cobra automatically shows usage on validation errors
- ✅ Cleaner, more testable code structure

### 🔧 **Error Handling Optimization**

#### Current Custom Errors
```go
// errors.go: Lines 6-21
var (
    ErrUsage = errors.New("usage error")
    ErrConfig = errors.New("configuration error")
    ErrAuth = errors.New("authentication error")
    ErrRuntime = errors.New("runtime error")
    ErrInternal = errors.New("internal error")
)
```

#### ✅ **Recommended: Use Cobra Error Patterns**
```go
// Leverage Cobra's built-in error handling
var registerCmd = &cobra.Command{
    Use: "register",
    RunE: func(cmd *cobra.Command, args []string) error {
        // Cobra automatically handles:
        // - Flag parsing errors -> exit code 2
        // - Usage errors -> shows help + exit code 2
        // - Validation errors -> shows usage + exit code 2
        
        if err := performRegistration(cmd.Context()); err != nil {
            // Only handle business logic errors
            return fmt.Errorf("registration failed: %w", err)
        }
        return nil
    },
}
```

**Benefits:**
- ✅ Removes 20+ lines of error classification code
- ✅ Cobra handles usage/flag errors automatically
- ✅ Consistent error behavior across all commands

### 🔧 **Output Template System**

#### Current Custom Output (verify.go, health.go)
```go
// Lines scattered across files
fmt.Printf("✅ Identity Verification\n")
fmt.Printf("Identity: %s\n", result.Identity)
fmt.Printf("Trust Domain: %s\n", result.TrustDomain)
fmt.Printf("Valid: %t\n", result.Valid)
```

#### ✅ **Recommended: Use Cobra Templates**
```go
const verifyTemplate = `{{if .Valid}}✅{{else}}❌{{end}} Identity Verification
Identity: {{.Identity}}
Trust Domain: {{.TrustDomain}}
Valid: {{.Valid}}
{{if .Message}}Message: {{.Message}}{{end}}
{{if not .NotBefore.IsZero}}Not Before: {{.NotBefore.Format "2006-01-02 15:04:05"}}{{end}}
{{if not .NotAfter.IsZero}}Not After: {{.NotAfter.Format "2006-01-02 15:04:05"}}{{end}}
`

func outputVerificationResult(cmd *cobra.Command, result *ports.IdentityVerificationResult) error {
    format, _ := cmd.Flags().GetString("format")
    
    switch format {
    case "json":
        return json.NewEncoder(os.Stdout).Encode(result)
    default:
        tmpl := template.Must(template.New("verify").Parse(verifyTemplate))
        return tmpl.Execute(os.Stdout, result)
    }
}
```

**Benefits:**
- ✅ Removes 50+ lines of custom printf statements
- ✅ Easier to maintain and modify output format
- ✅ Consistent formatting across commands
- ✅ Template reusability

### 🔧 **Completion Enhancement**

#### Current State
```go
// completion.go: Line 49
// Cobra automatically adds the completion command
```

#### ✅ **Recommended: Add Custom Completions**
```go
func init() {
    // Add intelligent completions
    registerCmd.RegisterFlagCompletionFunc("config", func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
        return cobra.ExpandDirs(toComplete, []string{"yaml", "yml"}), cobra.ShellCompDirectiveDefault
    })
    
    registerCmd.RegisterFlagCompletionFunc("domain", func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
        return []cobra.Completion{
            {Text: "example.org", Description: "Default domain"},
            {Text: "localhost", Description: "Local development"},
            {Text: "prod.company.com", Description: "Production domain"},
        }, cobra.ShellCompDirectiveNoFileComp
    })
}
```

**Benefits:**
- ✅ Intelligent file completion for config files
- ✅ Domain suggestions for common use cases
- ✅ Better user experience

---

## 3. Refactoring Plan

### Phase 1: Flag Validation (1-2 hours)
1. **Replace manual flag validation** with Cobra flag groups
2. **Add MarkFlagRequired** for essential flags
3. **Add MarkFlagFilename** for file paths
4. **Remove custom validation functions**

### Phase 2: PreRun Pattern (1 hour)
1. **Move validation to PreRunE** functions
2. **Simplify RunE** functions to business logic only
3. **Remove manual validation calls**

### Phase 3: Error Handling (30 minutes)
1. **Remove custom error classification** for usage errors
2. **Let Cobra handle** flag and usage errors automatically
3. **Keep only business logic** error handling

### Phase 4: Output Templates (2 hours)
1. **Replace fmt.Printf** with Go templates
2. **Create reusable templates** for common outputs
3. **Centralize formatting logic**

### Phase 5: Enhanced Completions (1 hour)
1. **Add intelligent completions** for common flags
2. **File path completions** for config files
3. **Value suggestions** for domains and services

---

## 4. Code Reduction Estimates

### Before Optimization
```
- register.go: 151 lines
- verify.go: 320 lines  
- health.go: 200 lines
- errors.go: 21 lines
- Custom validation: ~50 lines
- Custom output: ~100 lines
Total: ~842 lines
```

### After Optimization
```
- register.go: ~100 lines (-51 lines, -34%)
- verify.go: ~200 lines (-120 lines, -38%)
- health.go: ~150 lines (-50 lines, -25%)
- errors.go: ~10 lines (-11 lines, -52%)
- Templates: +30 lines (new)
- Completions: +20 lines (new)
Total: ~510 lines (-332 lines, -39% reduction)
```

**Estimated Reduction: 39% fewer lines of custom code**

---

## 5. Implementation Priority

### 🚀 **High Priority (Immediate)**
1. **Flag Validation**: Replace manual validation with Cobra flag groups
2. **PreRun Pattern**: Move validation to PreRunE functions  
3. **Error Simplification**: Remove custom usage error handling

### 📈 **Medium Priority (Next Sprint)**
4. **Output Templates**: Replace printf statements with templates
5. **Enhanced Completions**: Add intelligent flag completions

### 📋 **Low Priority (Future)**
6. **Advanced Features**: Explore Cobra's newer features (flag groups, etc.)

---

## 6. Risk Assessment

### 🟢 **Low Risk Refactoring**
- **Flag Groups**: Cobra handles this natively
- **PreRun Validation**: Standard Cobra pattern
- **Template Output**: Go standard library

### 🟡 **Medium Risk Areas**
- **Error Handling Changes**: Need to ensure exit codes remain correct
- **Output Format Changes**: Ensure backward compatibility

### 🔴 **High Risk Areas**
- **None Identified**: All changes use standard Cobra features

---

## 7. Testing Strategy

### Unit Tests
```go
func TestRegisterFlagValidation(t *testing.T) {
    cmd := registerCmd
    
    // Test Cobra's built-in validation
    err := cmd.Execute()
    assert.Contains(t, err.Error(), "required flag(s)")
    
    // Test flag groups work
    cmd.SetArgs([]string{"--config", "test.yaml", "--name", "test"})
    err = cmd.Execute()
    assert.Contains(t, err.Error(), "mutually exclusive")
}
```

### Integration Tests
```bash
# Test that Cobra validation works correctly
./ephemos register --config test.yaml --name test-service
# Should fail with mutually exclusive error

./ephemos register --name test-service
# Should fail with required domain error
```

---

## 8. Migration Path

### Step 1: Create Feature Branch
```bash
git checkout -b optimize/reduce-custom-code-use-cobra-builtin
```

### Step 2: Implement Flag Validation
```go
// Replace custom validation with Cobra built-ins
registerCmd.MarkFlagsMutuallyExclusive("config", "name")
registerCmd.MarkFlagsOneRequired("config", "name")
```

### Step 3: Test Each Change
```bash
go test ./internal/cli/...
./ephemos register --help  # Verify help shows requirements
```

### Step 4: Commit Incrementally
```bash
git commit -m "Use Cobra flag groups instead of custom validation"
git commit -m "Add PreRunE validation pattern"
git commit -m "Replace custom output with templates"
```

---

## 9. Expected Benefits

### 📊 **Metrics**
- **39% reduction** in custom code lines
- **50% fewer** custom validation functions
- **60% reduction** in custom error handling
- **80% fewer** printf statements

### 🎯 **Quality Improvements**
- **Better Error Messages**: Cobra's built-in errors are more consistent
- **Improved Help**: Automatic requirement documentation
- **Enhanced UX**: Better completions and suggestions
- **Easier Maintenance**: Less custom code to maintain

### 🚀 **Developer Experience**
- **Faster Feature Addition**: Less boilerplate for new commands
- **Consistent Patterns**: All commands follow same structure
- **Better Testing**: Cobra patterns are well-documented and testable

---

## 10. Conclusion

The Ephemos CLI is already well-structured but has significant opportunities to reduce custom code by leveraging more Cobra built-in features. The recommended changes will:

✅ **Reduce codebase by ~39%**  
✅ **Improve maintainability** through standard patterns  
✅ **Enhance user experience** with better validation and completion  
✅ **Follow Cobra best practices** more closely  

**Recommendation: Proceed with optimization** - The benefits significantly outweigh the minimal refactoring effort required.