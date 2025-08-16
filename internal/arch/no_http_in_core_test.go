package arch

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNoHTTPInCore verifies that core packages (internal/ and pkg/) don't import HTTP framework dependencies.
// This ensures proper separation between core identity/gRPC functionality and HTTP framework integrations.
func TestNoHTTPInCore(t *testing.T) {
	// HTTP framework imports that should not appear in core
	prohibitedImports := []string{
		"github.com/gin-gonic/gin",
		"github.com/go-chi/chi",
		"github.com/gorilla/mux",
		"github.com/labstack/echo",
		"github.com/gofiber/fiber",
		"net/http",  // Direct HTTP usage should be minimal in core
	}

	// Directories that constitute "core" functionality (relative to repo root)
	coreDirs := []string{
		"../../internal/core",
		"../../internal/adapters/interceptors", // gRPC interceptors only
		"../../pkg/ephemos",
	}

	for _, coreDir := range coreDirs {
		t.Run(coreDir, func(t *testing.T) {
			err := filepath.Walk(coreDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				// Only check Go files
				if !strings.HasSuffix(path, ".go") {
					return nil
				}

				// Skip test files for now (they might import http for testing)
				if strings.HasSuffix(path, "_test.go") {
					return nil
				}

				// Parse the Go file
				fset := token.NewFileSet()
				node, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
				if err != nil {
					return fmt.Errorf("failed to parse Go file %s: %w", path, err)
				}

				// Check imports
				for _, imp := range node.Imports {
					importPath := strings.Trim(imp.Path.Value, "\"")
					
					for _, prohibited := range prohibitedImports {
						if importPath == prohibited {
							// Special allowance for specific cases
							if isAllowedHTTPUsage(path, importPath) {
								continue
							}
							
							t.Errorf("Core file %s imports prohibited HTTP package: %s", path, importPath)
						}
					}
				}

				return nil
			})
			require.NoError(t, err)
		})
	}
}

// isAllowedHTTPUsage checks if a specific file is allowed to import net/http for legitimate core reasons
func isAllowedHTTPUsage(filePath, importPath string) bool {
	// Only check net/http since frameworks are never allowed
	if importPath != "net/http" {
		return false
	}

	// Allowed net/http usage in core:
	allowedHTTPFiles := []string{
		// HTTP client primitives (for core HTTP client functionality)
		"pkg/ephemos/http.go",             // HTTP client helpers from PR2
		"pkg/ephemos/interfaces.go",       // Core interfaces that expose HTTP clients
		"pkg/ephemos/public_api.go",       // Public API that might expose HTTP clients
		"internal/core/ports/client.go",   // Core client ports that use HTTP primitives
		"internal/adapters/primary/api/",  // API adapters might need HTTP for client creation
	}

	for _, allowed := range allowedHTTPFiles {
		if strings.Contains(filePath, allowed) {
			return true
		}
	}

	return false
}

// TestContribHasHTTPFrameworks verifies that contrib examples use framework imports (positive test)
func TestContribHasHTTPFrameworks(t *testing.T) {
	testCases := []struct {
		dir      string
		expected string
	}{
		{"../../contrib/middleware/chi/examples/main.go", "github.com/go-chi/chi"},
		{"../../contrib/middleware/gin/examples/main.go", "github.com/gin-gonic/gin"},
	}

	for _, tc := range testCases {
		t.Run(tc.dir, func(t *testing.T) {
			// Parse the file
			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, tc.dir, nil, parser.ImportsOnly)
			require.NoError(t, err)

			// Check that it imports the expected framework
			found := false
			for _, imp := range node.Imports {
				importPath := strings.Trim(imp.Path.Value, "\"")
				if strings.Contains(importPath, tc.expected) {
					found = true
					break
				}
			}

			assert.True(t, found, "Expected contrib example %s to import %s", tc.dir, tc.expected)
		})
	}
}

// TestHTTPFrameworkIsolation verifies HTTP frameworks are only in contrib
func TestHTTPFrameworkIsolation(t *testing.T) {
	httpFrameworks := []string{
		"gin-gonic/gin",
		"go-chi/chi",
		"gorilla/mux",
		"labstack/echo",
		"gofiber/fiber",
	}

	// Check that these frameworks only appear in allowed locations
	allowedDirs := []string{
		"../../contrib/middleware/",
		"../../contrib/examples/",
		"../../scripts/",       // Test scripts might use frameworks
		"../../examples/",      // Example code
		"../../.gitallowed",    // Git-secrets allowlist
		"../../README.md",      // Documentation
		"../../docs/",          // Documentation
	}

	err := filepath.Walk("../..", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only check Go files
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Skip allowed directories
		allowed := false
		for _, allowedDir := range allowedDirs {
			if strings.HasPrefix(path, allowedDir) {
				allowed = true
				break
			}
		}
		if allowed {
			return nil
		}

		// Parse file and check imports
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			// Skip files that can't be parsed
			return nil
		}

		for _, imp := range node.Imports {
			importPath := strings.Trim(imp.Path.Value, "\"")
			
			for _, framework := range httpFrameworks {
				if strings.Contains(importPath, framework) {
					t.Errorf("HTTP framework %s found in non-contrib file: %s", framework, path)
				}
			}
		}

		return nil
	})
	require.NoError(t, err)
}

// TestCoreArchitectureBoundaries tests broader architectural boundaries
func TestCoreArchitectureBoundaries(t *testing.T) {
	t.Run("core_only_imports_core", func(t *testing.T) {
		// internal/core should only import other internal/core packages and stdlib
		err := filepath.Walk("../../internal/core", func(path string, info os.FileInfo, err error) error {
			if err != nil || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}

			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
			if err != nil {
				return nil // Skip unparseable files
			}

			for _, imp := range node.Imports {
				importPath := strings.Trim(imp.Path.Value, "\"")
				
				// Allow stdlib and internal/core imports
				if !strings.Contains(importPath, "/") || // stdlib packages
					strings.HasPrefix(importPath, "github.com/sufield/ephemos/internal/core") ||
					strings.HasPrefix(importPath, "github.com/spiffe/go-spiffe") ||
					strings.HasPrefix(importPath, "google.golang.org/grpc") ||
					strings.HasPrefix(importPath, "google.golang.org/protobuf") {
					continue
				}

				// Prohibited: adapters, external frameworks, etc.
				if strings.HasPrefix(importPath, "github.com/sufield/ephemos/internal/adapters") {
					t.Errorf("Core package %s imports adapter: %s (violates dependency inversion)", path, importPath)
				}
			}

			return nil
		})
		require.NoError(t, err)
	})
}