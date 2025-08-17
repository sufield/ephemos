//go:build arch

package ports_test

import (
	"fmt"
	"go/ast"
	"runtime/debug"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

const maxMethods = 7

var forbiddenImportPrefixes = []string{
	"net/http", "database/sql", "google.golang.org/grpc",
	"github.com/redis", "github.com/go-redis", "github.com/segmentio/kafka-go",
	"github.com/spiffe", "go.opentelemetry.io/otel", "go.uber.org/zap",
	"github.com/rs/zerolog", "log/slog",
}

// modulePathTB gets the module path from build info.
func modulePathTB(t *testing.T) string {
	t.Helper()
	info, ok := debug.ReadBuildInfo()
	if !ok {
		t.Fatalf("read build info failed")
	}
	return info.Main.Path
}

// loadSyntax loads packages with AST and type information.
func loadSyntax(t *testing.T, patterns ...string) []*packages.Package {
	t.Helper()
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports,
	}
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		t.Fatalf("packages.Load: %v", err)
	}
	if packages.PrintErrors(pkgs) > 0 {
		t.Fatalf("failed to load some packages")
	}
	return pkgs
}

// isForbiddenImport checks if an import path is forbidden.
func isForbiddenImport(path string) bool {
	for _, pre := range forbiddenImportPrefixes {
		if path == pre || strings.HasPrefix(path, pre+"/") {
			return true
		}
	}
	return false
}

// checkInterfaceSize validates interface segregation principle.
func checkInterfaceSize(t *testing.T, _ *packages.Package, file *ast.File) []string {
	t.Helper()
	var out []string

	ast.Inspect(file, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}
		it, ok := ts.Type.(*ast.InterfaceType)
		if !ok {
			return true
		}

		// Count only actual methods (FuncType), not embedded interfaces
		count := 0
		for _, fld := range it.Methods.List {
			if _, ok := fld.Type.(*ast.FuncType); ok {
				count++
			}
		}

		if count > maxMethods {
			out = append(out, fmt.Sprintf("Interface %s has %d declared methods (max %d)", ts.Name.Name, count, maxMethods))
		}
		return true
	})
	return out
}

// Test_Interface_Segregation validates ISP compliance.
func Test_Interface_Segregation(t *testing.T) {
	t.Parallel()
	mp := modulePathTB(t)
	pkgs := loadSyntax(t, mp+"/internal/core/ports/...")

	var violations []string
	for _, p := range pkgs {
		for _, f := range p.Syntax {
			violations = append(violations, checkInterfaceSize(t, p, f)...)
		}
	}

	if len(violations) > 0 {
		t.Fatalf("Interface Segregation violations:\n%s", strings.Join(violations, "\n"))
	}
}

// checkAdapterPackageHasStruct validates that adapter packages have concrete implementations.
func checkAdapterPackageHasStruct(p *packages.Package) []string {
	hasStruct := false
	for _, f := range p.Syntax {
		ast.Inspect(f, func(n ast.Node) bool {
			ts, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}
			_, ok = ts.Type.(*ast.StructType)
			if ok {
				hasStruct = true
			}
			return true
		})
	}
	if !hasStruct {
		return []string{fmt.Sprintf("adapter package %q has no concrete struct types (implementations)", p.PkgPath)}
	}
	return nil
}

// Test_Adapter_Structure validates adapter implementation requirements.
func Test_Adapter_Structure(t *testing.T) {
	t.Parallel()
	mp := modulePathTB(t)
	pkgs := loadSyntax(t, mp+"/internal/adapters/...")

	var violations []string
	for _, p := range pkgs {
		violations = append(violations, checkAdapterPackageHasStruct(p)...)
	}

	if len(violations) > 0 {
		t.Fatalf("Adapter structure violations:\n%s", strings.Join(violations, "\n"))
	}
}

// checkDomainPurityAST inspects imports and types for infrastructure concerns.
func checkDomainPurityAST(t *testing.T, p *packages.Package, f *ast.File) []string {
	t.Helper()
	var out []string

	// 1) Check imports
	for _, imp := range f.Imports {
		path, _ := strconv.Unquote(imp.Path.Value)
		if isForbiddenImport(path) {
			out = append(out, fmt.Sprintf("%s imports %q", p.PkgPath, path))
		}
	}

	// 2) Check field types (ident, selector, pointers, containers)
	var checkType func(ast.Expr) string
	checkType = func(e ast.Expr) string {
		switch x := e.(type) {
		case *ast.Ident:
			return x.Name
		case *ast.SelectorExpr:
			// pkg.Type; we only have the name here—use types.Info for full path
			return x.Sel.Name
		case *ast.StarExpr:
			return checkType(x.X)
		case *ast.ArrayType:
			return checkType(x.Elt)
		case *ast.MapType:
			return checkType(x.Key) + "," + checkType(x.Value)
		default:
			return ""
		}
	}

	ast.Inspect(f, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}
		st, ok := ts.Type.(*ast.StructType)
		if !ok {
			return true
		}

		for _, fld := range st.Fields.List {
			nameHit := false
			for _, nm := range fld.Names {
				n := strings.ToLower(nm.Name)
				for _, s := range []string{"http", "grpc", "sql", "redis", "kafka", "db"} {
					if strings.Contains(n, s) {
						nameHit = true
						break
					}
				}
			}

			typ := strings.ToLower(checkType(fld.Type))
			typeHit := strings.Contains(typ, "http") || strings.Contains(typ, "grpc") ||
				strings.Contains(typ, "sql") || strings.Contains(typ, "redis") || strings.Contains(typ, "kafka")

			if nameHit || typeHit {
				out = append(out, fmt.Sprintf("domain struct %s has infrastructure concern in field %v", ts.Name.Name, fld.Names))
			}
		}
		return true
	})

	return out
}

// Test_Domain_Purity validates domain layer purity.
func Test_Domain_Purity(t *testing.T) {
	t.Parallel()
	mp := modulePathTB(t)
	pkgs := loadSyntax(t, mp+"/internal/core/domain/...")

	var violations []string
	for _, p := range pkgs {
		for _, f := range p.Syntax {
			violations = append(violations, checkDomainPurityAST(t, p, f)...)
		}
	}

	if len(violations) > 0 {
		t.Fatalf("Domain purity violations:\n%s", strings.Join(violations, "\n"))
	}
}

// checkPackageBoundaries validates import boundaries with chain tracking.
func checkPackageBoundaries(t *testing.T, p *packages.Package, chain []string) []string {
	t.Helper()
	var violations []string
	mp := modulePathTB(t)

	// Track current chain to avoid cycles
	currentChain := append(chain, p.PkgPath)
	if len(currentChain) > 5 { // Limit depth to prevent infinite recursion
		return violations
	}

	for _, imp := range p.Imports {
		// Check if core imports adapters (boundary violation)
		if strings.HasPrefix(p.PkgPath, mp+"/internal/core/") &&
			strings.HasPrefix(imp.PkgPath, mp+"/internal/adapters/") {
			chainStr := strings.Join(append(currentChain, imp.PkgPath), " -> ")
			violations = append(violations, fmt.Sprintf("Core imports adapters via: %s", chainStr))
		}

		// Check if domain imports forbidden packages
		if strings.HasPrefix(p.PkgPath, mp+"/internal/core/domain/") &&
			isForbiddenImport(imp.PkgPath) {
			chainStr := strings.Join(append(currentChain, imp.PkgPath), " -> ")
			violations = append(violations, fmt.Sprintf("Domain imports forbidden package via: %s", chainStr))
		}

		// Recurse into internal packages (limited depth)
		if strings.HasPrefix(imp.PkgPath, mp+"/internal/") && len(currentChain) < 4 {
			violations = append(violations, checkPackageBoundaries(t, imp, currentChain)...)
		}
	}

	return violations
}

// Test_Package_Boundaries validates import boundaries with violation paths.
func Test_Package_Boundaries(t *testing.T) {
	t.Parallel()
	mp := modulePathTB(t)
	pkgs := loadSyntax(t, mp+"/internal/...")

	var violations []string
	for _, p := range pkgs {
		violations = append(violations, checkPackageBoundaries(t, p, nil)...)
	}

	if len(violations) > 0 {
		t.Fatalf("Package boundary violations:\n%s", strings.Join(violations, "\n"))
	}
}
