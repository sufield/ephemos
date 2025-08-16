package arch

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getRepoRoot returns the absolute path to the repository root by finding go.mod
func getRepoRoot(t *testing.T) string {
	t.Helper()
	
	// Start from the current test file's directory
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok, "Failed to get current file path")
	
	dir := filepath.Dir(filename)
	
	// Walk up the directory tree to find go.mod
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("Could not find go.mod file - not in a Go module")
		}
		dir = parent
	}
}

// parseImports extracts import paths from a Go file
func parseImports(t *testing.T, filePath string) ([]string, error) {
	t.Helper()
	
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ImportsOnly)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Go file %s: %w", filePath, err)
	}
	
	var imports []string
	for _, imp := range node.Imports {
		importPath := strings.Trim(imp.Path.Value, "\"")
		imports = append(imports, importPath)
	}
	
	return imports, nil
}

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

	repoRoot := getRepoRoot(t)
	
	// Directories that constitute "core" functionality
	coreDirs := []string{
		"internal/core",
		"internal/adapters/interceptors", // gRPC interceptors only
		"pkg/ephemos",
	}

	for _, coreDir := range coreDirs {
		t.Run(coreDir, func(t *testing.T) {
			fullPath := filepath.Join(repoRoot, coreDir)
			err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return fmt.Errorf("walking directory %s: %w", path, err)
				}

				// Only check Go files
				if !strings.HasSuffix(path, ".go") {
					return nil
				}

				// Skip test files for now (they might import http for testing)
				if strings.HasSuffix(path, "_test.go") {
					return nil
				}

				// Parse imports from the file
				imports, err := parseImports(t, path)
				if err != nil {
					return err
				}

				// Check imports against prohibited list
				for _, importPath := range imports {
					for _, prohibited := range prohibitedImports {
						if importPath == prohibited { // Exact match to avoid false positives
							// Special allowance for specific cases
							if isAllowedHTTPUsage(path, importPath, repoRoot) {
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
func isAllowedHTTPUsage(filePath, importPath, repoRoot string) bool {
	// Only check net/http since frameworks are never allowed
	if importPath != "net/http" {
		return false
	}

	// Convert to relative path from repo root for consistent checking
	relPath, err := filepath.Rel(repoRoot, filePath)
	if err != nil {
		return false // If we can't determine relative path, deny
	}
	
	// Normalize path separators for cross-platform compatibility
	relPath = filepath.ToSlash(relPath)

	// Allowed net/http usage in core (exact file matches or directory prefixes)
	allowedHTTPFiles := []string{
		// HTTP client primitives (for core HTTP client functionality)
		"pkg/ephemos/http.go",             // HTTP client helpers from PR2
		"pkg/ephemos/interfaces.go",       // Core interfaces that expose HTTP clients
		"pkg/ephemos/public_api.go",       // Public API that might expose HTTP clients
		"internal/core/ports/client.go",   // Core client ports that use HTTP primitives
	}
	
	allowedHTTPDirs := []string{
		"internal/adapters/primary/api/",  // API adapters might need HTTP for client creation
	}

	// Check exact file matches
	for _, allowed := range allowedHTTPFiles {
		if relPath == allowed {
			return true
		}
	}
	
	// Check directory prefixes
	for _, allowedDir := range allowedHTTPDirs {
		if strings.HasPrefix(relPath, allowedDir) {
			return true
		}
	}

	return false
}

// TestContribHasHTTPFrameworks verifies that contrib examples use framework imports (positive test)
func TestContribHasHTTPFrameworks(t *testing.T) {
	repoRoot := getRepoRoot(t)
	
	testCases := []struct {
		dir      string
		expected string
	}{
		{"contrib/middleware/chi/examples/main.go", "github.com/go-chi/chi"},
		{"contrib/middleware/gin/examples/main.go", "github.com/gin-gonic/gin"},
	}

	for _, tc := range testCases {
		t.Run(tc.dir, func(t *testing.T) {
			filePath := filepath.Join(repoRoot, tc.dir)
			
			// Parse imports from the file
			imports, err := parseImports(t, filePath)
			require.NoError(t, err)

			// Check that it imports the expected framework (with prefix matching for versioned imports)
			found := false
			for _, importPath := range imports {
				if strings.HasPrefix(importPath, tc.expected) {
					found = true
					break
				}
			}

			assert.True(t, found, "Expected contrib example %s to import framework starting with %s", tc.dir, tc.expected)
		})
	}
}

// TestHTTPFrameworkIsolation verifies HTTP frameworks are only in contrib
func TestHTTPFrameworkIsolation(t *testing.T) {
	repoRoot := getRepoRoot(t)
	
	// Framework import prefixes that should only appear in allowed locations
	httpFrameworks := []string{
		"github.com/gin-gonic/gin",
		"github.com/go-chi/chi", 
		"github.com/gorilla/mux",
		"github.com/labstack/echo",
		"github.com/gofiber/fiber",
	}

	// Directories/files where frameworks are allowed (relative to repo root)
	allowedPaths := []string{
		"contrib/middleware/",
		"contrib/examples/", 
		"scripts/",       // Test scripts might use frameworks
		"examples/",      // Example code
		".gitallowed",    // Git-secrets allowlist
		"README.md",      // Documentation
		"docs/",          // Documentation
	}

	err := filepath.Walk(repoRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("walking directory %s: %w", path, err)
		}

		// Only check Go files
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Convert to relative path for consistent checking
		relPath, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return fmt.Errorf("getting relative path for %s: %w", path, err)
		}
		relPath = filepath.ToSlash(relPath) // Normalize separators

		// Skip allowed directories/files
		allowed := false
		for _, allowedPath := range allowedPaths {
			if relPath == allowedPath || strings.HasPrefix(relPath, allowedPath) {
				allowed = true
				break
			}
		}
		if allowed {
			return nil
		}

		// Parse imports and check for framework usage
		imports, err := parseImports(t, path)
		if err != nil {
			// Log but don't fail for unparseable files (might be generated code)
			t.Logf("Skipping unparseable file %s: %v", relPath, err)
			return nil
		}

		for _, importPath := range imports {
			for _, framework := range httpFrameworks {
				// Use prefix matching to catch versioned imports like /v2, /v5, etc.
				if strings.HasPrefix(importPath, framework) {
					t.Errorf("HTTP framework %s found in non-contrib file: %s", framework, relPath)
				}
			}
		}

		return nil
	})
	require.NoError(t, err)
}

// TestCoreArchitectureBoundaries tests broader architectural boundaries
func TestCoreArchitectureBoundaries(t *testing.T) {
	repoRoot := getRepoRoot(t)
	
	t.Run("core_only_imports_core", func(t *testing.T) {
		// internal/core should only import other internal/core packages and stdlib
		coreDir := filepath.Join(repoRoot, "internal/core")
		err := filepath.Walk(coreDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return fmt.Errorf("walking core directory %s: %w", path, err)
			}
			
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}

			// Parse imports
			imports, err := parseImports(t, path)
			if err != nil {
				t.Logf("Skipping unparseable core file %s: %v", path, err)
				return nil
			}

			// Get relative path for error reporting
			relPath, _ := filepath.Rel(repoRoot, path)
			relPath = filepath.ToSlash(relPath)

			for _, importPath := range imports {
				// Allow stdlib packages (no domain in import path)
				if !strings.Contains(importPath, "/") {
					continue
				}
				
				// Allow specific external dependencies that core legitimately needs
				allowedExternalDeps := []string{
					"github.com/spiffe/go-spiffe",
					"google.golang.org/grpc",
					"google.golang.org/protobuf",
					"github.com/sufield/ephemos/internal/core", // Self-references are OK
				}
				
				allowed := false
				for _, allowedDep := range allowedExternalDeps {
					if strings.HasPrefix(importPath, allowedDep) {
						allowed = true
						break
					}
				}
				
				if allowed {
					continue
				}

				// Prohibited: adapters, contrib, other internal packages
				if strings.HasPrefix(importPath, "github.com/sufield/ephemos/internal/adapters") {
					t.Errorf("Core package %s imports adapter: %s (violates dependency inversion)", relPath, importPath)
				} else if strings.HasPrefix(importPath, "github.com/sufield/ephemos/contrib") {
					t.Errorf("Core package %s imports contrib: %s (core should not depend on contrib)", relPath, importPath)
				} else if strings.HasPrefix(importPath, "github.com/sufield/ephemos/internal/") {
					t.Errorf("Core package %s imports non-core internal package: %s", relPath, importPath)
				}
			}

			return nil
		})
		require.NoError(t, err)
	})
}