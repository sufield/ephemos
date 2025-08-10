package arch_test

import (
    "runtime/debug"
    "strings"
    "testing"

    "golang.org/x/tools/go/packages"
)

// Keep the list short, explicit, and reviewed.
// You can add/remove prefixes over time as needs evolve.
var forbiddenPrefixes = []string{
    // External frameworks/libs that must not enter core:
    "google.golang.org/grpc",
    "github.com/spiffe",               // spiffe/spire
    "go.opentelemetry.io/otel",        // observability
    "go.uber.org/zap",                 // logging impl
    "github.com/rs/zerolog",
    // Stdlib APIs we want to forbid in core (use ports instead):
    "log/slog",
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

func isStdlib(path string) bool {
    // Heuristic: stdlib imports have no dot in the first segment.
    // e.g., "context", "time", "crypto/x509".
    return !strings.Contains(strings.Split(path, "/")[0], ".")
}

func Test_Core_Has_No_Forbidden_Imports(t *testing.T) {
    mp := modulePath(t)
    adaptersPrefix := mp + "/internal/adapters"

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

    // Walk transitive imports of every core package.
    violations := make(map[string][]string) // offending import -> []who-imported
    seen := make(map[string]bool)

    var checkImport func(owner string, p *packages.Package)
    checkImport = func(owner string, p *packages.Package) {
        for path, imp := range p.Imports {
            key := owner + " -> " + path
            if seen[key] {
                continue
            }
            seen[key] = true

            // Rule 1: core must not import (transitively) internal/adapters.
            if strings.HasPrefix(path, adaptersPrefix) {
                violations[path] = append(violations[path], owner)
            }

            // Rule 2: core must not import forbidden external/framework packages.
            for _, pref := range forbiddenPrefixes {
                if path == pref || strings.HasPrefix(path, pref+"/") {
                    violations[path] = append(violations[path], owner)
                    break
                }
            }

            // Allow stdlib (except any explicitly blacklisted above, e.g., log/slog).
            if isStdlib(path) && path != "log/slog" {
                // still descend; stdlib may import nothing or other stdlib only
                // but it's cheap and safe.
            }

            // Recurse.
            if imp != nil {
                checkImport(path, imp)
            }
        }
    }

    for _, pkg := range pkgs {
        checkImport(pkg.PkgPath, pkg)
    }

    if len(violations) > 0 {
        var b strings.Builder
        b.WriteString("Import boundary violated:\n")
        for imp, owners := range violations {
            b.WriteString("  - ")
            b.WriteString(imp)
            b.WriteString("\n    introduced via:\n")
            // Show a few unique introducers for signal.
            seenOwner := map[string]bool{}
            count := 0
            for _, o := range owners {
                if seenOwner[o] {
                    continue
                }
                seenOwner[o] = true
                b.WriteString("      * ")
                b.WriteString(o)
                b.WriteString("\n")
                count++
                if count >= 5 {
                    break
                }
            }
        }
        b.WriteString("\nRemediation:\n")
        b.WriteString("  - Move framework usage behind ports in internal/adapters.\n")
        b.WriteString("  - If you need a capability in core, define an output port in internal/app and implement it in adapters.\n")
        b.WriteString("  - If a stdlib/framework type leaks into core APIs, introduce a small domain type and map in adapters.\n")
        t.Fatalf("%s", b.String())
    }
}