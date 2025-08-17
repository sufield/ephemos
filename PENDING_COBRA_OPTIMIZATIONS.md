# Pending Cobra Optimization Tasks

**Created:** August 17, 2025  
**Status:** Future optimization opportunities identified during CLI refactoring

## Background

During the Cobra optimization work to reduce custom code, several additional opportunities were identified for future improvement. These represent the next phase of optimization to further leverage Cobra's built-in features.

## Future Optimization Opportunities

### 1. **JSON Output Consolidation**
**Priority:** Medium  
**Effort:** 2-3 hours  
**Impact:** Reduce code duplication across 12+ files

**Current State:**
- Multiple files have similar `json.NewEncoder(os.Stdout).Encode()` patterns
- Each command implements JSON output independently
- Inconsistent formatting and error handling

**Files Affected:**
- `/home/zepho/work/ephemos/internal/cli/inspect.go` (5 occurrences)
- `/home/zepho/work/ephemos/internal/cli/diagnose.go` (5 occurrences)
- `/home/zepho/work/ephemos/internal/cli/verify.go` (2 occurrences)
- `/home/zepho/work/ephemos/internal/cli/health.go` (1 occurrence)

**Proposed Solution:**
```go
// Create shared JSON output utilities
func outputStructuredData(cmd *cobra.Command, data interface{}) error {
    format, _ := cmd.Flags().GetString("format")
    quiet, _ := cmd.Flags().GetBool("quiet")
    
    switch format {
    case "json":
        return outputJSON(data)
    case "yaml":
        return outputYAML(data)
    default:
        return outputTemplate(cmd, data)
    }
}
```

**Benefits:**
- Consistent JSON formatting across all commands
- Centralized error handling for output operations
- Easier to add new output formats (YAML, CSV, etc.)
- Reduced maintenance overhead

### 2. **Template System Enhancement**
**Priority:** Low  
**Effort:** 3-4 hours  
**Impact:** Create reusable template management system

**Current State:**
- Each command defines its own templates
- Manual template parsing and execution
- Repeated emoji replacement logic

**Proposed Solution:**
```go
type TemplateManager struct {
    templates map[string]*template.Template
    noEmoji   bool
}

func NewTemplateManager(noEmoji bool) *TemplateManager {
    // Pre-parse and cache common templates
    // Handle emoji replacement centrally
}

func (tm *TemplateManager) Render(name string, data interface{}) error {
    // Centralized template rendering with error handling
}
```

**Benefits:**
- Template caching and reuse
- Centralized emoji replacement
- Consistent template error handling
- Easier template management and updates

### 3. **Configuration Integration with Viper**
**Priority:** Low  
**Effort:** 4-5 hours  
**Impact:** Leverage Cobra's Viper integration for configuration

**Current State:**
- Manual configuration file loading
- Custom precedence handling
- Repeated config file validation

**Files Affected:**
- `/home/zepho/work/ephemos/internal/cli/register.go`
- `/home/zepho/work/ephemos/internal/cli/health.go`
- `/home/zepho/work/ephemos/cmd/config-validator/main.go`

**Proposed Solution:**
```go
// Use Cobra's Viper integration
func init() {
    cobra.OnInitialize(initConfig)
    
    rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file")
    viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))
}

func initConfig() {
    if cfgFile != "" {
        viper.SetConfigFile(cfgFile)
    } else {
        viper.SetConfigName("ephemos")
        viper.AddConfigPath(".")
        viper.AddConfigPath("$HOME/.ephemos")
    }
    
    viper.AutomaticEnv()
    viper.ReadInConfig()
}
```

**Benefits:**
- Automatic environment variable binding
- Standardized configuration precedence (flags > env > config file > defaults)
- Built-in configuration file discovery
- Reduced custom configuration code

## Implementation Priority

### Phase 1 (Next Sprint)
1. **JSON Output Consolidation** - High impact, medium effort

### Phase 2 (Future)
2. **Template System Enhancement** - Medium impact, medium effort  
3. **Configuration Integration** - Low impact, high effort

## Estimated Benefits

**Code Reduction:**
- JSON Consolidation: ~15% reduction in output-related code
- Template System: ~10% reduction in template-related code  
- Viper Integration: ~20% reduction in configuration code

**Total Additional Reduction:** ~15% more code reduction beyond current 49%

## Notes

- These optimizations should be implemented incrementally
- Each change should be thoroughly tested to ensure no regression
- Consider backward compatibility when implementing changes
- Document any breaking changes in configuration handling

## Related Files

- `COBRA_OPTIMIZATION_RECOMMENDATIONS.md` - Original optimization analysis
- `PRODUCTION_READINESS_ASSESSMENT.md` - Production readiness evaluation
- Current CLI implementation in `/home/zepho/work/ephemos/internal/cli/`