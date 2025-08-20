//go:build integration

package arch_test

import (
	"fmt"
	"go/ast"
	"go/token"
	"runtime/debug"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

// getForbiddenPrefixes returns the list of forbidden import prefixes.
// Keep the list short, explicit, and reviewed.
// You can add/remove prefixes over time as needs evolve.
func getForbiddenPrefixes() []string {
	return []string{
		// External frameworks/libs that must not enter core:
		"google.golang.org/grpc",
		"github.com/spiffe",        // spiffe/spire
		"go.opentelemetry.io/otel", // observability
		"go.uber.org/zap",          // logging impl
		"github.com/rs/zerolog",
		// Stdlib APIs we want to forbid in core (use ports instead):
		"log/slog",
	}
}

// Get the module path, e.g., "github.com/sufield/ephemos".
func modulePath(t *testing.T) string {
	t.Helper()
	info, ok := debug.ReadBuildInfo()
	if !ok {
		t.Fatalf("failed to read build info")
	}
	// The main module is the first one in info.Main
	return info.Main.Path
}

// importChecker encapsulates the state and logic for checking imports.
type importChecker struct {
	adaptersPrefix    string
	forbiddenPrefixes []string
	violations        map[string][]string
	seen              map[string]bool
	importChains      map[string][]string // tracks import chains for violation remediation
}

// newImportChecker creates a new import checker.
func newImportChecker(adaptersPrefix string, forbiddenPrefixes []string) *importChecker {
	return &importChecker{
		adaptersPrefix:    adaptersPrefix,
		forbiddenPrefixes: forbiddenPrefixes,
		violations:        make(map[string][]string),
		seen:              make(map[string]bool),
		importChains:      make(map[string][]string),
	}
}

// checkPackage checks all imports of a package.
func (ic *importChecker) checkPackage(owner string, p *packages.Package) {
	ic.checkPackageWithChain(owner, p, []string{owner})
}

// checkPackageWithChain checks all imports of a package with import chain tracking.
func (ic *importChecker) checkPackageWithChain(owner string, p *packages.Package, chain []string) {
	for path, imp := range p.Imports {
		ic.checkSingleImportWithChain(owner, path, imp, chain)
	}
}

// checkSingleImport checks a single import.
func (ic *importChecker) checkSingleImport(owner, path string, imp *packages.Package) {
	ic.checkSingleImportWithChain(owner, path, imp, []string{owner})
}

// checkSingleImportWithChain checks a single import with chain tracking.
func (ic *importChecker) checkSingleImportWithChain(owner, path string, imp *packages.Package, chain []string) {
	// Skip if already seen this path from this owner
	if !ic.markSeen(owner, path) {
		return
	}

	// Store the import chain for this violation
	newChain := append(chain, path)
	ic.importChains[path] = newChain

	// Check violation rules
	ic.checkAdaptersViolation(owner, path)
	ic.checkForbiddenPrefixViolation(owner, path)

	// Recurse if needed (with depth limit to prevent infinite loops)
	if imp != nil && len(newChain) < 10 {
		ic.checkPackageWithChain(path, imp, newChain)
	}
}

// markSeen marks an import as seen and returns true if it's new.
func (ic *importChecker) markSeen(owner, path string) bool {
	key := owner + " -> " + path
	if ic.seen[key] {
		return false
	}
	ic.seen[key] = true
	return true
}

// checkAdaptersViolation checks if import violates adapters boundary.
func (ic *importChecker) checkAdaptersViolation(owner, path string) {
	if strings.HasPrefix(path, ic.adaptersPrefix) {
		ic.violations[path] = append(ic.violations[path], owner)
	}
}

// checkForbiddenPrefixViolation checks if import uses forbidden packages.
func (ic *importChecker) checkForbiddenPrefixViolation(owner, path string) {
	for _, prefix := range ic.forbiddenPrefixes {
		if ic.matchesForbiddenPrefix(path, prefix) {
			ic.violations[path] = append(ic.violations[path], owner)
			break
		}
	}
}

// matchesForbiddenPrefix checks if a path matches a forbidden prefix.
func (ic *importChecker) matchesForbiddenPrefix(path, prefix string) bool {
	return path == prefix || strings.HasPrefix(path, prefix+"/")
}

// loadCorePackages loads all core packages for testing.
func loadCorePackages(t *testing.T) []*packages.Package {
	t.Helper()
	mp := modulePath(t)
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedImports |
			packages.NeedDeps |
			packages.NeedModule |
			packages.NeedFiles,
	}

	// Load all core packages for hexagonal architecture.
	pkgs, err := packages.Load(cfg, mp+"/internal/core/...")
	if err != nil {
		t.Fatalf("packages.Load: %v", err)
	}

	// Fail early if there were loader errors.
	if packages.PrintErrors(pkgs) > 0 {
		t.Fatalf("failed to load some core packages")
	}

	return pkgs
}

// formatViolations formats violation messages for output.
func formatViolations(violations map[string][]string) string {
	var b strings.Builder
	b.WriteString("Import boundary violated:\n")

	for imp, owners := range violations {
		formatSingleViolation(&b, imp, owners)
	}

	appendRemediation(&b)
	return b.String()
}

// formatViolationsWithChains formats violation messages with import chain tracking.
func (ic *importChecker) formatViolationsWithChains() string {
	var b strings.Builder
	b.WriteString("Import boundary violated:\n")

	for imp, owners := range ic.violations {
		ic.formatSingleViolationWithChain(&b, imp, owners)
	}

	appendRemediation(&b)
	return b.String()
}

// formatSingleViolation formats a single violation entry.
func formatSingleViolation(b *strings.Builder, imp string, owners []string) {
	b.WriteString("  - ")
	b.WriteString(imp)
	b.WriteString("\n    introduced via:\n")

	// Show unique introducers
	seenOwner := map[string]bool{}
	count := 0

	for _, owner := range owners {
		if seenOwner[owner] {
			continue
		}
		seenOwner[owner] = true

		b.WriteString("      * ")
		b.WriteString(owner)
		b.WriteString("\n")

		count++
		if count >= 5 {
			break
		}
	}
}

// formatSingleViolationWithChain formats a single violation entry with import chain.
func (ic *importChecker) formatSingleViolationWithChain(b *strings.Builder, imp string, owners []string) {
	b.WriteString("  - ")
	b.WriteString(imp)
	b.WriteString("\n    introduced via:\n")

	// Show unique introducers
	seenOwner := map[string]bool{}
	count := 0

	for _, owner := range owners {
		if seenOwner[owner] {
			continue
		}
		seenOwner[owner] = true

		b.WriteString("      * ")
		b.WriteString(owner)

		// Add import chain if available
		if chain, exists := ic.importChains[imp]; exists && len(chain) > 1 {
			b.WriteString(" (via: ")
			b.WriteString(strings.Join(chain[:len(chain)-1], " -> "))
			b.WriteString(")")
		}

		b.WriteString("\n")

		count++
		if count >= 5 {
			break
		}
	}
}

// appendRemediation adds remediation advice to the output.
func appendRemediation(b *strings.Builder) {
	b.WriteString("\nRemediation:\n")
	b.WriteString("  - Move framework usage behind ports in internal/adapters.\n")
	b.WriteString("  - If you need a capability in core, define an output port in internal/core/ports and implement it in adapters.\n")
	b.WriteString("  - If a stdlib/framework type leaks into core APIs, introduce a small domain type and map in adapters.\n")
	b.WriteString("  - Follow hexagonal architecture: Core -> Ports -> Adapters (dependencies flow inward).\n")
}

func Test_Core_Has_No_Forbidden_Imports(t *testing.T) {
	t.Parallel()
	mp := modulePath(t)
	adaptersPrefix := mp + "/internal/adapters"
	forbiddenPrefixes := getForbiddenPrefixes()

	// Load packages
	pkgs := loadCorePackages(t)

	// Check imports
	checker := newImportChecker(adaptersPrefix, forbiddenPrefixes)
	for _, pkg := range pkgs {
		checker.checkPackage(pkg.PkgPath, pkg)
	}

	// Report violations
	if len(checker.violations) > 0 {
		t.Fatalf("%s", checker.formatViolationsWithChains())
	}
}

// Test_Adapters_Cannot_Import_Other_Adapters ensures adapters are isolated.
//
//nolint:cyclop // Test function complexity acceptable for readability
func Test_Adapters_Cannot_Import_Other_Adapters(t *testing.T) {
	t.Parallel()
	mp := modulePath(t)
	adaptersPrefix := mp + "/internal/adapters"

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedImports | packages.NeedDeps | packages.NeedModule | packages.NeedFiles,
	}

	pkgs, err := packages.Load(cfg, mp+"/internal/adapters/...")
	if err != nil {
		t.Fatalf("packages.Load: %v", err)
	}

	if packages.PrintErrors(pkgs) > 0 {
		t.Fatalf("failed to load some adapter packages")
	}

	violations := make(map[string][]string)

	for _, pkg := range pkgs {
		for importPath := range pkg.Imports {
			// Check if this adapter imports another adapter
			if strings.HasPrefix(importPath, adaptersPrefix) && importPath != pkg.PkgPath {
				// Allow shared adapter subtrees
				if isSharedAdapterPath(importPath, adaptersPrefix) {
					continue
				}

				// Extract adapter types
				ownerAdapter := extractAdapterType(pkg.PkgPath, adaptersPrefix)
				importedAdapter := extractAdapterType(importPath, adaptersPrefix)

				// Only flag if they're different adapter types
				if ownerAdapter != importedAdapter && ownerAdapter != "" && importedAdapter != "" {
					violations[importPath] = append(violations[importPath], pkg.PkgPath)
				}
			}
		}
	}

	if len(violations) > 0 {
		var b strings.Builder
		b.WriteString("Adapter isolation violated - adapters should not import other adapters:\n")
		for imp, owners := range violations {
			b.WriteString("  - ")
			b.WriteString(imp)
			b.WriteString("\n    imported by:\n")
			for _, owner := range owners {
				b.WriteString("      * ")
				b.WriteString(owner)
				b.WriteString("\n")
			}
		}
		b.WriteString("\nAdapters should communicate through ports, not direct imports.\n")
		t.Fatalf("%s", b.String())
	}
}

// Test_Core_Domain_Has_No_External_Dependencies ensures domain is pure.
//
//nolint:cyclop // Test function complexity acceptable for readability
func Test_Core_Domain_Has_No_External_Dependencies(t *testing.T) {
	t.Parallel()
	mp := modulePath(t)

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedImports | packages.NeedDeps | packages.NeedModule | packages.NeedFiles,
	}

	pkgs, err := packages.Load(cfg, mp+"/internal/core/domain/...")
	if err != nil {
		t.Fatalf("packages.Load: %v", err)
	}

	if packages.PrintErrors(pkgs) > 0 {
		t.Fatalf("failed to load domain packages")
	}

	violations := make(map[string][]string)

	allowedStdlib := map[string]bool{
		"errors":  true,
		"fmt":     true,
		"strings": true,
		"time":    true,
		"context": true,
	}

	for _, pkg := range pkgs {
		for importPath := range pkg.Imports {
			allowed := false

			// Check if it's an allowed stdlib package
			if !strings.Contains(importPath, ".") {
				if allowedStdlib[importPath] {
					allowed = true
				}
			} else {
				// Check self-imports within domain
				if importPath == mp+"/internal/core/domain" || strings.HasPrefix(importPath, mp+"/internal/core/domain/") {
					allowed = true
				}
			}

			if !allowed {
				violations[importPath] = append(violations[importPath], pkg.PkgPath)
			}
		}
	}

	if len(violations) > 0 {
		var b strings.Builder
		b.WriteString("Domain purity violated - domain should only use stdlib and self-imports:\n")
		for imp, owners := range violations {
			b.WriteString("  - ")
			b.WriteString(imp)
			b.WriteString("\n    imported by:\n")
			for _, owner := range owners {
				b.WriteString("      * ")
				b.WriteString(owner)
				b.WriteString("\n")
			}
		}
		t.Fatalf("%s", b.String())
	}
}

// Test_Public_API_Boundary ensures pkg/ephemos doesn't leak internal details.
func Test_Public_API_Boundary(t *testing.T) {
	t.Parallel()
	mp := modulePath(t)
	internalPrefix := mp + "/internal/"

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedImports | packages.NeedDeps | packages.NeedModule | packages.NeedFiles,
	}

	pkgs, err := packages.Load(cfg, mp+"/pkg/ephemos/...")
	if err != nil {
		t.Fatalf("packages.Load: %v", err)
	}

	if packages.PrintErrors(pkgs) > 0 {
		t.Fatalf("failed to load public API packages")
	}

	violations := make(map[string][]string)

	for _, pkg := range pkgs {
		for importPath := range pkg.Imports {
			// Check if public API imports internal packages
			if strings.HasPrefix(importPath, internalPrefix) {
				violations[importPath] = append(violations[importPath], pkg.PkgPath)
			}
		}
	}

	if len(violations) > 0 {
		var b strings.Builder
		b.WriteString("Public API boundary violated - pkg/ephemos should not import internal packages:\n")
		for imp, owners := range violations {
			b.WriteString("  - ")
			b.WriteString(imp)
			b.WriteString("\n    imported by:\n")
			for _, owner := range owners {
				b.WriteString("      * ")
				b.WriteString(owner)
				b.WriteString("\n")
			}
		}
		b.WriteString("\nPublic API should only expose abstractions, not internal implementations.\n")
		t.Fatalf("%s", b.String())
	}
}

// Test_Public_API_No_SPIFFE_Types ensures pkg/ephemos doesn't leak SPIFFE types.
func Test_Public_API_No_SPIFFE_Types(t *testing.T) {
	t.Parallel()
	mp := modulePath(t)

	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedImports |
			packages.NeedDeps |
			packages.NeedFiles |
			packages.NeedTypes |
			packages.NeedSyntax |
			packages.NeedTypesInfo,
	}

	pkgs, err := packages.Load(cfg, mp+"/pkg/ephemos/...")
	if err != nil {
		t.Fatalf("packages.Load: %v", err)
	}

	if packages.PrintErrors(pkgs) > 0 {
		t.Fatalf("failed to load public API packages")
	}

	spiffeTypePrefixes := []string{
		"github.com/spiffe/go-spiffe/v2/spiffeid",
		"github.com/spiffe/go-spiffe/v2/svid",
		"github.com/spiffe/go-spiffe/v2/bundle",
		"github.com/spiffe/go-spiffe/v2/workloadapi",
	}

	violations := make(map[string][]string)

	for _, pkg := range pkgs {
		// Check exported functions and methods for SPIFFE types in signatures
		for _, file := range pkg.Syntax {
			for _, decl := range file.Decls {
				switch d := decl.(type) {
				case *ast.FuncDecl:
					if d.Name.IsExported() {
						violations = checkFunctionForSPIFFETypes(violations, pkg.PkgPath, d, spiffeTypePrefixes)
					}
				case *ast.GenDecl:
					if d.Tok == token.TYPE {
						for _, spec := range d.Specs {
							if ts, ok := spec.(*ast.TypeSpec); ok && ts.Name.IsExported() {
								violations = checkTypeForSPIFFETypes(violations, pkg.PkgPath, ts, spiffeTypePrefixes)
							}
						}
					}
				}
			}
		}
	}

	if len(violations) > 0 {
		var b strings.Builder
		b.WriteString("Public API SPIFFE type leak detected - pkg/ephemos should not expose SPIFFE types:\n")
		for location, types := range violations {
			b.WriteString("  - ")
			b.WriteString(location)
			b.WriteString("\n    SPIFFE types found:\n")
			for _, spiffeType := range types {
				b.WriteString("      * ")
				b.WriteString(spiffeType)
				b.WriteString("\n")
			}
		}
		b.WriteString("\nRemediation:\n")
		b.WriteString("  - Replace SPIFFE types with domain types (e.g., domain.Identity instead of spiffeid.ID)\n")
		b.WriteString("  - Use adapter pattern to convert between domain and SPIFFE types\n")
		b.WriteString("  - Keep SPIFFE integration isolated to internal/adapters\n")
		t.Fatalf("%s", b.String())
	}
}

// Test_Circular_Dependencies detects circular import patterns.
func Test_Circular_Dependencies(t *testing.T) {
	t.Parallel()
	mp := modulePath(t)
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedImports | packages.NeedDeps | packages.NeedModule | packages.NeedFiles,
	}

	pkgs, err := packages.Load(cfg, mp+"/internal/...")
	if err != nil {
		t.Fatalf("packages.Load: %v", err)
	}

	if packages.PrintErrors(pkgs) > 0 {
		t.Fatalf("failed to load internal packages")
	}

	graph := make(map[string][]string)
	for _, pkg := range pkgs {
		for importPath := range pkg.Imports {
			if strings.HasPrefix(importPath, mp+"/internal/") {
				graph[pkg.PkgPath] = append(graph[pkg.PkgPath], importPath)
			}
		}
	}

	cycles := findCycles(graph)
	if len(cycles) > 0 {
		var b strings.Builder
		b.WriteString("Circular dependencies detected:\n")
		for i, cycle := range cycles {
			b.WriteString("  Cycle ")
			b.WriteString(fmt.Sprintf("%d", i+1))
			b.WriteString(": ")
			b.WriteString(strings.Join(cycle, " -> "))
			b.WriteString("\n")
		}
		t.Fatalf("%s", b.String())
	}
}

// Test_Layer_Dependencies ensures proper layering (domain <- ports <- services <- adapters).
func Test_Layer_Dependencies(t *testing.T) {
	t.Parallel()
	mp := modulePath(t)

	layerHierarchy := map[string]int{
		mp + "/internal/core/domain":   0, // Bottom layer
		mp + "/internal/core/errors":   0,
		mp + "/internal/core/ports":    1,
		mp + "/internal/core/services": 2,
		mp + "/internal/adapters":      3, // Top layer
		mp + "/pkg/ephemos":            3, // Same level as adapters
	}

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedImports | packages.NeedDeps | packages.NeedModule | packages.NeedFiles,
	}

	pkgs, err := packages.Load(cfg, mp+"/internal/...", mp+"/pkg/ephemos/...")
	if err != nil {
		t.Fatalf("packages.Load: %v", err)
	}

	violations := make(map[string][]string)

	for _, pkg := range pkgs {
		pkgLayer := getLayerLevel(pkg.PkgPath, layerHierarchy)

		for importPath := range pkg.Imports {
			importLayer := getLayerLevel(importPath, layerHierarchy)

			// Check if import violates layer hierarchy (lower layer importing higher layer is violation)
			if importLayer != -1 && pkgLayer != -1 && pkgLayer < importLayer {
				violations[pkg.PkgPath] = append(violations[pkg.PkgPath], importPath)
			}
		}
	}

	if len(violations) > 0 {
		var b strings.Builder
		b.WriteString("Layer dependency violations detected:\n")
		b.WriteString("Layers should follow: Domain(0) <- Ports(1) <- Services(2) <- Adapters(3)\n")
		for owner, imports := range violations {
			b.WriteString("  Package: ")
			b.WriteString(owner)
			b.WriteString("\n    Illegally imports:\n")
			for _, imp := range imports {
				b.WriteString("      * ")
				b.WriteString(imp)
				b.WriteString("\n")
			}
		}
		t.Fatalf("%s", b.String())
	}
}

// Helper functions

func extractAdapterType(path, adaptersPrefix string) string {
	if !strings.HasPrefix(path, adaptersPrefix) {
		return ""
	}

	remainder := strings.TrimPrefix(path, adaptersPrefix+"/")
	parts := strings.Split(remainder, "/")
	if len(parts) > 0 {
		return parts[0] // primary, secondary, grpc, http, etc.
	}
	return ""
}

// isSharedAdapterPath checks if a path points to a shared adapter subtree
// that should be allowed to be imported by multiple adapter types.
func isSharedAdapterPath(path, adaptersPrefix string) bool {
	if !strings.HasPrefix(path, adaptersPrefix) {
		return false
	}

	remainder := strings.TrimPrefix(path, adaptersPrefix+"/")

	// Allow shared utilities, common types, etc.
	sharedPrefixes := []string{
		"shared/",
		"common/",
		"utils/",
		"types/",
	}

	for _, prefix := range sharedPrefixes {
		if strings.HasPrefix(remainder, prefix) {
			return true
		}
	}

	return false
}

func findCycles(graph map[string][]string) [][]string {
	var cycles [][]string
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	path := make(map[string]string)

	var dfs func(string) bool
	dfs = func(node string) bool {
		visited[node] = true
		recStack[node] = true

		foundCycle := false
		for _, neighbor := range graph[node] {
			if !visited[neighbor] {
				path[neighbor] = node
				if dfs(neighbor) {
					foundCycle = true
				}
			} else if recStack[neighbor] {
				// Found cycle, reconstruct it
				cycle := []string{neighbor}
				current := node
				for current != neighbor {
					cycle = append(cycle, current)
					current = path[current]
				}
				cycle = append(cycle, neighbor) // Complete the cycle

				// Reverse to get correct order
				for i, j := 0, len(cycle)-1; i < j; i, j = i+1, j-1 {
					cycle[i], cycle[j] = cycle[j], cycle[i]
				}

				cycles = append(cycles, cycle)
				foundCycle = true
			}
		}

		recStack[node] = false
		return foundCycle
	}

	for node := range graph {
		if !visited[node] {
			dfs(node)
		}
	}

	return cycles
}

func getLayerLevel(pkgPath string, hierarchy map[string]int) int {
	// Find the most specific match
	bestMatch := ""
	bestLevel := -1

	for prefix, level := range hierarchy {
		if strings.HasPrefix(pkgPath, prefix) {
			if len(prefix) > len(bestMatch) {
				bestMatch = prefix
				bestLevel = level
			}
		}
	}

	return bestLevel
}

// checkFunctionForSPIFFETypes checks if a function signature contains SPIFFE types
func checkFunctionForSPIFFETypes(violations map[string][]string, pkgPath string, fn *ast.FuncDecl, spiffePrefixes []string) map[string][]string {
	location := pkgPath + "." + fn.Name.Name

	// Check parameters
	if fn.Type.Params != nil {
		for _, param := range fn.Type.Params.List {
			if spiffeType := extractSPIFFEType(param.Type, spiffePrefixes); spiffeType != "" {
				violations[location] = append(violations[location], spiffeType+" (parameter)")
			}
		}
	}

	// Check return types
	if fn.Type.Results != nil {
		for _, result := range fn.Type.Results.List {
			if spiffeType := extractSPIFFEType(result.Type, spiffePrefixes); spiffeType != "" {
				violations[location] = append(violations[location], spiffeType+" (return)")
			}
		}
	}

	return violations
}

// checkTypeForSPIFFETypes checks if a type definition contains SPIFFE types
func checkTypeForSPIFFETypes(violations map[string][]string, pkgPath string, ts *ast.TypeSpec, spiffePrefixes []string) map[string][]string {
	location := pkgPath + "." + ts.Name.Name

	if spiffeType := extractSPIFFEType(ts.Type, spiffePrefixes); spiffeType != "" {
		violations[location] = append(violations[location], spiffeType+" (type)")
	}

	// Check struct fields
	if structType, ok := ts.Type.(*ast.StructType); ok {
		for _, field := range structType.Fields.List {
			if spiffeType := extractSPIFFEType(field.Type, spiffePrefixes); spiffeType != "" {
				fieldName := "anonymous"
				if len(field.Names) > 0 {
					fieldName = field.Names[0].Name
				}
				violations[location] = append(violations[location], spiffeType+" (field "+fieldName+")")
			}
		}
	}

	return violations
}

// extractSPIFFEType extracts SPIFFE type names from AST expressions
func extractSPIFFEType(expr ast.Expr, spiffePrefixes []string) string {
	switch e := expr.(type) {
	case *ast.SelectorExpr:
		// Handle qualified identifiers like spiffeid.ID
		if ident, ok := e.X.(*ast.Ident); ok {
			qualifiedName := ident.Name + "." + e.Sel.Name
			for _, prefix := range spiffePrefixes {
				if strings.HasSuffix(prefix, "/"+ident.Name) {
					return qualifiedName
				}
			}
		}
	case *ast.StarExpr:
		// Handle pointer types
		if baseType := extractSPIFFEType(e.X, spiffePrefixes); baseType != "" {
			return "*" + baseType
		}
	case *ast.ArrayType:
		// Handle slice/array types
		if elemType := extractSPIFFEType(e.Elt, spiffePrefixes); elemType != "" {
			return "[]" + elemType
		}
	case *ast.Ident:
		// Handle simple identifiers
		for _, prefix := range spiffePrefixes {
			if strings.HasSuffix(prefix, "/"+e.Name) || strings.HasSuffix(prefix, "."+e.Name) {
				return e.Name
			}
		}
	}
	return ""
}
