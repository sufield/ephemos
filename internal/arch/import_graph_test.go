// Package arch provides architectural constraint tests.
// These tests enforce boundary rules and prevent unwanted dependencies.
package arch

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestImportGraphConstraints verifies that the core domain doesn't leak external dependencies.
// This test fails if internal/core imports forbidden packages like go-spiffe, grpc, net/http, etc.
func TestImportGraphConstraints(t *testing.T) {
	// Define forbidden imports for core domain
	forbiddenImports := map[string][]string{
		"internal/core": {
			"github.com/spiffe/go-spiffe",
			"google.golang.org/grpc",
			"net/http",
			"net",
			"crypto/tls",
			"github.com/golang/protobuf",
			"google.golang.org/protobuf",
		},
		"internal/core/domain": {
			"github.com/spiffe/go-spiffe",
			"google.golang.org/grpc",
			"net/http",
			"net",
			"crypto/tls",
		},
		"internal/core/services": {
			"github.com/spiffe/go-spiffe",
			"google.golang.org/grpc",
			"net/http",
			"net",
			"crypto/tls",
		},
		"internal/core/ports": {
			// Ports can only use standard library abstractions
			"github.com/spiffe/go-spiffe",
			"google.golang.org/grpc",
			"net/http", // Should use our HTTPClient abstraction instead
			"net",      // Should use our NetworkListener abstraction instead
			"crypto/tls",
		},
	}

	// Allowed standard library packages for core
	allowedStdLib := map[string]bool{
		"context":      true,
		"crypto":       true,
		"crypto/x509":  true,
		"encoding/json": true,
		"errors":       true,
		"fmt":          true,
		"io":           true,
		"strings":      true,
		"time":         true,
		"sync":         true,
		"testing":      true,
	}

	for packagePath, forbidden := range forbiddenImports {
		t.Run(packagePath, func(t *testing.T) {
			violations := checkPackageImports(t, packagePath, forbidden, allowedStdLib)
			if len(violations) > 0 {
				t.Errorf("Package %s has forbidden imports:", packagePath)
				for file, imports := range violations {
					t.Errorf("  %s imports:", file)
					for _, imp := range imports {
						t.Errorf("    - %s (FORBIDDEN)", imp)
					}
				}
				t.Error("\nCore domain must not depend on infrastructure packages.")
				t.Error("Use port abstractions instead of direct dependencies.")
			}
		})
	}
}

// TestPortAbstractionIntegrity verifies that port abstractions don't leak infrastructure types.
// This ensures our abstractions remain clean and don't expose underlying implementations.
func TestPortAbstractionIntegrity(t *testing.T) {
	// Check that port files only use allowed types
	portFiles := []string{
		"internal/core/ports/http_abstractions.go",
		"internal/core/ports/network_abstractions.go",
	}

	forbiddenTypes := []string{
		"*http.Client",
		"*http.Server",
		"http.Handler",
		"net.Listener",
		"net.Conn",
		"*grpc.ClientConn",
		"*grpc.Server",
		"*tls.Config",
		"spiffeid.ID",
		"*x509svid.SVID",
	}

	for _, portFile := range portFiles {
		t.Run(portFile, func(t *testing.T) {
			violations := checkFileForForbiddenTypes(t, portFile, forbiddenTypes)
			if len(violations) > 0 {
				t.Errorf("Port abstraction file %s contains forbidden types:", portFile)
				for _, violation := range violations {
					t.Errorf("  - %s", violation)
				}
				t.Error("\nPort abstractions must not expose infrastructure types.")
				t.Error("Use primitive types and interfaces instead.")
			}
		})
	}
}

// TestAdapterIsolation verifies that adapters properly encapsulate external dependencies.
// Adapters should be thin wrappers that convert between our abstractions and external APIs.
func TestAdapterIsolation(t *testing.T) {
	// Check that only adapters import external packages
	allowedExternalImporters := []string{
		"internal/adapters/",
		"internal/factory/",
		"pkg/ephemos/", // Public API can use external packages
		"cmd/",
		"examples/",
		"contrib/",
	}

	// Find all Go files in the project
	goFiles := findGoFiles(t, ".")
	
	for _, file := range goFiles {
		// Skip files that are allowed to import external packages
		allowed := false
		for _, allowedPrefix := range allowedExternalImporters {
			if strings.HasPrefix(file, allowedPrefix) || strings.Contains(file, "_test.go") {
				allowed = true
				break
			}
		}
		
		if allowed {
			continue
		}

		t.Run(file, func(t *testing.T) {
			externalImports := findExternalImports(t, file)
			if len(externalImports) > 0 {
				t.Errorf("Non-adapter file %s imports external packages:", file)
				for _, imp := range externalImports {
					t.Errorf("  - %s", imp)
				}
				t.Error("\nOnly adapters should import external packages.")
				t.Error("Move external dependencies to adapters and use ports for abstractions.")
			}
		})
	}
}

// Helper functions

func checkPackageImports(t *testing.T, packagePath string, forbidden []string, allowedStdLib map[string]bool) map[string][]string {
	violations := make(map[string][]string)
	
	// Find all Go files in the package
	pattern := filepath.Join(packagePath, "*.go")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("failed to find Go files in %s: %v", packagePath, err)
	}

	for _, file := range matches {
		// Skip test files for now (they may need external dependencies)
		if strings.HasSuffix(file, "_test.go") {
			continue
		}

		fileViolations := checkFileImports(t, file, forbidden, allowedStdLib)
		if len(fileViolations) > 0 {
			violations[file] = fileViolations
		}
	}

	return violations
}

func checkFileImports(t *testing.T, filename string, forbidden []string, allowedStdLib map[string]bool) []string {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("failed to parse %s: %v", filename, err)
		return nil
	}

	var violations []string
	
	for _, imp := range node.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		
		// Check against forbidden imports
		for _, forbidden := range forbidden {
			if strings.HasPrefix(importPath, forbidden) {
				violations = append(violations, importPath)
				continue
			}
		}

		// Check if it's an external package (not standard library or internal)
		if !isStandardLibrary(importPath) && !strings.HasPrefix(importPath, "github.com/sufield/ephemos") {
			if !allowedStdLib[importPath] {
				violations = append(violations, importPath)
			}
		}
	}

	return violations
}

func checkFileForForbiddenTypes(t *testing.T, filename string, forbiddenTypes []string) []string {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		// File might not exist, that's ok
		return nil
	}

	var violations []string
	
	ast.Inspect(node, func(n ast.Node) bool {
		// Check type specifications
		if typeSpec, ok := n.(*ast.TypeSpec); ok {
			typeStr := nodeToString(typeSpec.Type)
			for _, forbidden := range forbiddenTypes {
				if strings.Contains(typeStr, forbidden) {
					violations = append(violations, typeStr)
				}
			}
		}
		
		// Check function signatures
		if funcType, ok := n.(*ast.FuncType); ok {
			funcStr := nodeToString(funcType)
			for _, forbidden := range forbiddenTypes {
				if strings.Contains(funcStr, forbidden) {
					violations = append(violations, funcStr)
				}
			}
		}
		
		return true
	})

	return violations
}

func findGoFiles(t *testing.T, root string) []string {
	var files []string
	
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if strings.HasSuffix(path, ".go") && !strings.Contains(path, "vendor/") {
			files = append(files, path)
		}
		
		return nil
	})
	
	if err != nil {
		t.Fatalf("failed to walk directory tree: %v", err)
	}
	
	return files
}

func findExternalImports(t *testing.T, filename string) []string {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ImportsOnly)
	if err != nil {
		return nil // File might not exist
	}

	var external []string
	
	for _, imp := range node.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		
		// Skip standard library and internal imports
		if isStandardLibrary(importPath) || strings.HasPrefix(importPath, "github.com/sufield/ephemos") {
			continue
		}
		
		external = append(external, importPath)
	}

	return external
}

func isStandardLibrary(importPath string) bool {
	// Standard library packages don't contain dots or are well-known
	stdLibPrefixes := []string{
		"bufio", "bytes", "context", "crypto", "database", "encoding", 
		"errors", "fmt", "go", "hash", "html", "image", "io", "log", 
		"math", "mime", "net", "os", "path", "reflect", "regexp", 
		"runtime", "sort", "strconv", "strings", "sync", "syscall", 
		"testing", "time", "unicode", "unsafe",
	}
	
	for _, prefix := range stdLibPrefixes {
		if importPath == prefix || strings.HasPrefix(importPath, prefix+"/") {
			return true
		}
	}
	
	return false
}

func nodeToString(node ast.Node) string {
	// Simple node to string conversion for type checking
	// This is a simplified version - in practice you might use go/format
	switch n := node.(type) {
	case *ast.Ident:
		return n.Name
	case *ast.StarExpr:
		return "*" + nodeToString(n.X)
	case *ast.SelectorExpr:
		return nodeToString(n.X) + "." + nodeToString(n.Sel)
	default:
		return "unknown"
	}
}