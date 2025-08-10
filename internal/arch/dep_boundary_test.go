package arch_test

import (
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
}

// newImportChecker creates a new import checker.
func newImportChecker(adaptersPrefix string, forbiddenPrefixes []string) *importChecker {
	return &importChecker{
		adaptersPrefix:    adaptersPrefix,
		forbiddenPrefixes: forbiddenPrefixes,
		violations:        make(map[string][]string),
		seen:              make(map[string]bool),
	}
}

// checkPackage checks all imports of a package.
func (ic *importChecker) checkPackage(owner string, p *packages.Package) {
	for path, imp := range p.Imports {
		ic.checkSingleImport(owner, path, imp)
	}
}

// checkSingleImport checks a single import.
func (ic *importChecker) checkSingleImport(owner, path string, imp *packages.Package) {
	// Skip if already seen this path from this owner
	if !ic.markSeen(owner, path) {
		return
	}

	// Check violation rules
	ic.checkAdaptersViolation(owner, path)
	ic.checkForbiddenPrefixViolation(owner, path)

	// Recurse if needed
	if imp != nil {
		ic.checkPackage(path, imp)
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
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedImports |
			packages.NeedDeps |
			packages.NeedModule |
			packages.NeedFiles,
		Dir: "../..", // Go up from internal/arch to repo root
	}

	// Load all core packages (both old structure and new structure).
	pkgs, err := packages.Load(cfg, "./internal/core/...", "./internal/domain/...", "./internal/app/...")
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

// appendRemediation adds remediation advice to the output.
func appendRemediation(b *strings.Builder) {
	b.WriteString("\nRemediation:\n")
	b.WriteString("  - Move framework usage behind ports in internal/adapters.\n")
	b.WriteString("  - If you need a capability in core, define an output port in internal/app and implement it in adapters.\n")
	b.WriteString("  - If a stdlib/framework type leaks into core APIs, introduce a small domain type and map in adapters.\n")
}

func Test_Core_Has_No_Forbidden_Imports(t *testing.T) {
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
		t.Fatalf("%s", formatViolations(checker.violations))
	}
}
