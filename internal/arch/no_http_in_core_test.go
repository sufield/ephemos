package arch

import (
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ArchConfig holds configuration for architecture tests.
type ArchConfig struct {
	ProhibitedImports   []string `json:"prohibited_imports"`
	AllowedHTTPFiles    []string `json:"allowed_http_files"`
	AllowedHTTPDirs     []string `json:"allowed_http_dirs"`
	AllowedPaths        []string `json:"allowed_paths"`
	HTTPFrameworks      []string `json:"http_frameworks"`
	CoreDirs            []string `json:"core_dirs"`
	AllowedExternalDeps []string `json:"allowed_external_deps"`
	SkipPatterns        []string `json:"skip_patterns"`
	IncludeTestFiles    bool     `json:"include_test_files"`

	// Compiled regexes for performance (populated during config load)
	compiledFrameworkRegexes map[string]*regexp.Regexp `json:"-"`
}

// getDefaultConfig returns the default configuration for architecture tests
func getDefaultConfig() *ArchConfig {
	return &ArchConfig{
		ProhibitedImports: []string{
			"github.com/gin-gonic/gin",
			"github.com/go-chi/chi",
			"github.com/gorilla/mux",
			"github.com/labstack/echo",
			"github.com/gofiber/fiber",
			"net/http", // Restricted in core with specific allowances
		},
		AllowedHTTPFiles: []string{
			"pkg/ephemos/http.go",
			"pkg/ephemos/interfaces.go",
			"pkg/ephemos/public_api.go",
			"internal/core/ports/client.go",
		},
		AllowedHTTPDirs: []string{
			"internal/adapters/primary/api/",
		},
		AllowedPaths: []string{
			"contrib/middleware/",
			"contrib/examples/",
			"scripts/",
			"examples/",
			".gitallowed",
			"README.md",
			"docs/",
		},
		HTTPFrameworks: []string{
			"github.com/gin-gonic/gin",
			"github.com/go-chi/chi",
			"github.com/gorilla/mux",
			"github.com/labstack/echo",
			"github.com/gofiber/fiber",
		},
		CoreDirs: []string{
			"internal/core",
			"internal/adapters/interceptors",
			"pkg/ephemos",
		},
		AllowedExternalDeps: []string{
			"github.com/spiffe/go-spiffe",
			"google.golang.org/grpc",
			"google.golang.org/protobuf",
			"github.com/sufield/ephemos/internal/core",
		},
		SkipPatterns: []string{
			"_generated.go",
			".pb.go",
			"mock_",
		},
		IncludeTestFiles: false,
	}
}

// loadConfig loads configuration from file or returns default.
func loadConfig(t *testing.T, repoRoot string) *ArchConfig {
	t.Helper()

	configPath := filepath.Join(repoRoot, ".arch-test-config.json")
	var config *ArchConfig

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		config = getDefaultConfig()
	} else {
		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Logf("Failed to read config file %s, using defaults: %v", configPath, err)
			config = getDefaultConfig()
		} else {
			var loadedConfig ArchConfig
			if err := json.Unmarshal(data, &loadedConfig); err != nil {
				t.Logf("Failed to parse config file %s, using defaults: %v", configPath, err)
				config = getDefaultConfig()
			} else {
				config = &loadedConfig
			}
		}
	}

	// Precompile framework regexes for performance
	config.compiledFrameworkRegexes = make(map[string]*regexp.Regexp)
	for _, framework := range config.HTTPFrameworks {
		if regex, err := compileFrameworkRegex(framework); err == nil {
			config.compiledFrameworkRegexes[framework] = regex
		} else {
			t.Logf("Failed to compile regex for framework %s: %v", framework, err)
		}
	}

	return config
}

// getRepoRoot returns the absolute path to the repository root
func getRepoRoot(t *testing.T) string {
	t.Helper()

	// First try git rev-parse for most reliable method
	if cmd := exec.Command("git", "rev-parse", "--show-toplevel"); cmd.Dir == "" {
		if output, err := cmd.Output(); err == nil {
			root := strings.TrimSpace(string(output))
			if abs, err := filepath.Abs(root); err == nil {
				return abs
			}
		}
	}

	// Fallback: start from current test file and find go.mod
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok, "Failed to get current file path")

	dir := filepath.Dir(filename)
	for {
		absDir, err := filepath.Abs(dir)
		require.NoError(t, err, "Failed to get absolute path")

		if _, err := os.Stat(filepath.Join(absDir, "go.mod")); err == nil {
			return absDir
		}

		parent := filepath.Dir(absDir)
		if parent == absDir {
			t.Fatal("Could not find repository root (no go.mod or git repo)")
		}
		dir = parent
	}
}

// parseImports extracts import paths from a Go file with consistent error handling
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

// shouldSkipFile determines if a file should be skipped based on patterns and test file inclusion
func shouldSkipFile(filePath string, config *ArchConfig) bool {
	fileName := filepath.Base(filePath)

	// Skip non-Go files
	if !strings.HasSuffix(fileName, ".go") {
		return true
	}

	// Skip test files unless explicitly included
	if !config.IncludeTestFiles && strings.HasSuffix(fileName, "_test.go") {
		return true
	}

	// Skip files matching skip patterns
	for _, pattern := range config.SkipPatterns {
		if strings.Contains(fileName, pattern) {
			return true
		}
	}

	return false
}

// normalizeRelPath converts a file path to a normalized relative path from repo root
func normalizeRelPath(t *testing.T, filePath, repoRoot string) (string, error) {
	t.Helper()

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", fmt.Errorf("getting absolute path for %s: %w", filePath, err)
	}

	relPath, err := filepath.Rel(repoRoot, absPath)
	if err != nil {
		return "", fmt.Errorf("getting relative path for %s: %w", filePath, err)
	}

	// Normalize path separators for cross-platform compatibility
	return filepath.ToSlash(relPath), nil
}

// compileFrameworkRegex creates a regex for matching framework imports with version support
func compileFrameworkRegex(framework string) (*regexp.Regexp, error) {
	// Escape the framework name and add version suffix support
	escaped := regexp.QuoteMeta(framework)
	pattern := fmt.Sprintf(`^%s($|/v\d+)?`, escaped)
	return regexp.Compile(pattern)
}

// FileError represents an error from processing a specific file.
type FileError struct {
	Path string
	Err  error
}

// Error implements the error interface.
func (fe *FileError) Error() string {
	return fmt.Sprintf("%s: %v", fe.Path, fe.Err)
}

// walkDirectoryParallel walks a directory in parallel for improved performance.
// It collects all errors and is safe for concurrent use with testing.T.
func walkDirectoryParallel(t *testing.T, rootDir string, processFn func(path string, info os.FileInfo) error) []FileError {
	t.Helper()

	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []FileError

	// Function to safely add errors
	addError := func(path string, err error) {
		mu.Lock()
		errors = append(errors, FileError{Path: path, Err: err})
		mu.Unlock()
	}

	err := filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			addError(path, fmt.Errorf("walking directory: %w", err))
			return nil // Continue walking
		}

		if d.IsDir() {
			return nil
		}

		wg.Add(1)
		go func(p string, dirEntry os.DirEntry) {
			defer wg.Done()

			info, err := dirEntry.Info()
			if err != nil {
				addError(p, fmt.Errorf("getting file info: %w", err))
				return
			}

			if err := processFn(p, info); err != nil {
				addError(p, err)
			}
		}(path, d)

		return nil
	})

	// Wait for all goroutines to complete
	wg.Wait()

	// Add walk error if any
	if err != nil {
		mu.Lock()
		errors = append(errors, FileError{Path: rootDir, Err: fmt.Errorf("directory walk failed: %w", err)})
		mu.Unlock()
	}

	return errors
}

// TestNoHTTPInCore verifies that core packages (internal/ and pkg/) don't import HTTP framework dependencies.
// This ensures proper separation between core identity/gRPC functionality and HTTP framework integrations.
func TestNoHTTPInCore(t *testing.T) {
	repoRoot := getRepoRoot(t)
	config := loadConfig(t, repoRoot)

	for _, coreDir := range config.CoreDirs {
		t.Run(coreDir, func(t *testing.T) {
			fullPath := filepath.Join(repoRoot, coreDir)

			errorList := walkDirectoryParallel(t, fullPath, func(path string, _ os.FileInfo) error {
				// Skip files based on configuration
				if shouldSkipFile(path, config) {
					return nil
				}

				// Parse imports from the file
				imports, err := parseImports(t, path)
				if err != nil {
					return fmt.Errorf("parsing imports from %s: %w", path, err)
				}

				// Get normalized relative path for checking allowances
				relPath, err := normalizeRelPath(t, path, repoRoot)
				if err != nil {
					return fmt.Errorf("normalizing path %s: %w", path, err)
				}

				// Check imports against prohibited list
				for _, importPath := range imports {
					if err := checkProhibitedImport(importPath, relPath, config); err != nil {
						return fmt.Errorf("core file %s: %v", relPath, err)
					}
				}

				return nil
			})

			// Report all errors found during parallel processing
			for _, fileErr := range errorList {
				t.Error(fileErr.Error())
			}
		})
	}
}

// checkProhibitedImport validates that an import is not prohibited for core packages.
func checkProhibitedImport(importPath, relPath string, config *ArchConfig) error {
	// Check against prohibited imports with regex support for frameworks
	for _, prohibited := range config.ProhibitedImports {
		// Handle net/http specially (exact match)
		if prohibited == "net/http" && importPath == "net/http" {
			if isAllowedHTTPUsage(relPath, config) {
				continue
			}
			return fmt.Errorf("imports prohibited HTTP package: %s", importPath)
		}

		// Handle framework imports with cached regex for versioned imports
		if frameworkRegex, exists := config.compiledFrameworkRegexes[prohibited]; exists {
			if frameworkRegex.MatchString(importPath) {
				return fmt.Errorf("imports prohibited framework: %s (matches %s)", importPath, prohibited)
			}
		}
	}

	return nil
}

// isAllowedHTTPUsage checks if a specific file is allowed to import net/http for legitimate core reasons
func isAllowedHTTPUsage(relPath string, config *ArchConfig) bool {
	// Check exact file matches
	for _, allowed := range config.AllowedHTTPFiles {
		if relPath == allowed {
			return true
		}
	}

	// Check directory prefixes
	for _, allowedDir := range config.AllowedHTTPDirs {
		if strings.HasPrefix(relPath, allowedDir) {
			return true
		}
	}

	return false
}

// TestContribHasHTTPFrameworks verifies that contrib examples use framework imports (positive test).
func TestContribHasHTTPFrameworks(t *testing.T) {
	repoRoot := getRepoRoot(t)
	config := loadConfig(t, repoRoot)

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

			// Check that it imports the expected framework using cached regex
			frameworkRegex, exists := config.compiledFrameworkRegexes[tc.expected]
			require.True(t, exists, "Framework %s not found in compiled regexes", tc.expected)

			found := false
			for _, importPath := range imports {
				if frameworkRegex.MatchString(importPath) {
					found = true
					break
				}
			}

			assert.True(t, found, "Expected contrib example %s to import framework matching %s", tc.dir, tc.expected)
		})
	}
}

// TestHTTPFrameworkIsolation verifies HTTP frameworks are only in contrib
func TestHTTPFrameworkIsolation(t *testing.T) {
	repoRoot := getRepoRoot(t)
	config := loadConfig(t, repoRoot)

	errorList := walkDirectoryParallel(t, repoRoot, func(path string, _ os.FileInfo) error {
		// Skip files based on configuration
		if shouldSkipFile(path, config) {
			return nil
		}

		// Get normalized relative path for checking allowances
		relPath, err := normalizeRelPath(t, path, repoRoot)
		if err != nil {
			return fmt.Errorf("normalizing path %s: %w", path, err)
		}

		// Skip allowed directories/files
		allowed := false
		for _, allowedPath := range config.AllowedPaths {
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
			// Don't fail for unparseable files (might be generated code)
			return nil
		}

		for _, importPath := range imports {
			for _, framework := range config.HTTPFrameworks {
				// Use cached regex matching for precise version support
				if frameworkRegex, exists := config.compiledFrameworkRegexes[framework]; exists {
					if frameworkRegex.MatchString(importPath) {
						return fmt.Errorf("HTTP framework %s found in non-contrib file: %s", importPath, relPath)
					}
				}
			}
		}

		return nil
	})

	// Report all errors found during parallel processing
	for _, fileErr := range errorList {
		t.Error(fileErr.Error())
	}
}

// TestCoreArchitectureBoundaries tests broader architectural boundaries
func TestCoreArchitectureBoundaries(t *testing.T) {
	repoRoot := getRepoRoot(t)
	config := loadConfig(t, repoRoot)

	t.Run("core_only_imports_core", func(t *testing.T) {
		// internal/core should only import other internal/core packages and stdlib
		coreDir := filepath.Join(repoRoot, "internal/core")

		errorList := walkDirectoryParallel(t, coreDir, func(path string, _ os.FileInfo) error {
			// Skip files based on configuration (includes test files if configured)
			if shouldSkipFile(path, config) {
				return nil
			}

			// Parse imports
			imports, err := parseImports(t, path)
			if err != nil {
				// Don't fail for unparseable files
				return nil
			}

			// Get relative path for error reporting
			relPath, err := normalizeRelPath(t, path, repoRoot)
			if err != nil {
				return fmt.Errorf("normalizing path %s: %w", path, err)
			}

			for _, importPath := range imports {
				// Allow stdlib packages (no domain in import path)
				if !strings.Contains(importPath, "/") {
					continue
				}

				// Allow specific external dependencies that core legitimately needs
				allowed := false
				for _, allowedDep := range config.AllowedExternalDeps {
					if strings.HasPrefix(importPath, allowedDep) {
						allowed = true
						break
					}
				}

				if allowed {
					continue
				}

				// Prohibited: adapters, contrib, other internal packages
				switch {
				case strings.HasPrefix(importPath, "github.com/sufield/ephemos/internal/adapters"):
					return fmt.Errorf("core package %s imports adapter: %s (violates dependency inversion)", relPath, importPath)
				case strings.HasPrefix(importPath, "github.com/sufield/ephemos/contrib"):
					return fmt.Errorf("core package %s imports contrib: %s (core should not depend on contrib)", relPath, importPath)
				case strings.HasPrefix(importPath, "github.com/sufield/ephemos/internal/"):
					return fmt.Errorf("core package %s imports non-core internal package: %s", relPath, importPath)
				}
			}

			return nil
		})

		// Report all errors found during parallel processing
		for _, fileErr := range errorList {
			t.Error(fileErr.Error())
		}
	})
}
