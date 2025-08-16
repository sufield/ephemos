package ephemos_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	ephemos "github.com/sufield/ephemos/pkg/ephemos"
)

// Test_NoInternalImports_InPublicPkgs ensures the public package doesn't import internal packages
func Test_NoInternalImports_InPublicPkgs(t *testing.T) {
	t.Parallel()
	var violations []string

	// Limit scope to public package dir
	root := filepath.FromSlash(".")

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			t.Fatalf("walk err: %v", err)
		}
		if d.IsDir() {
			base := d.Name()
			// Skip directories we shouldn't scan
			if base == "internal" || base == "vendor" || base == ".git" || base == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		
		// Only check Go files, skip test files
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		// Skip generated files
		if isGeneratedFile(path) {
			return nil
		}

		src, rerr := os.ReadFile(path)
		if rerr != nil {
			t.Fatalf("read %s: %v", path, rerr)
		}

		fset := token.NewFileSet()
		f, perr := parser.ParseFile(fset, path, src, parser.ImportsOnly)
		if perr != nil {
			t.Fatalf("parse %s: %v", path, perr)
		}

		for _, imp := range f.Imports {
			p := strings.Trim(imp.Path.Value, `"`)
			if strings.HasPrefix(p, "github.com/sufield/ephemos/internal/") {
				violations = append(violations, fmt.Sprintf("❌ %s: imports %q", path, p))
			}
		}
		return nil
	})

	if err != nil {
		t.Fatalf("walk failed: %v", err)
	}

	if len(violations) > 0 {
		t.Fatalf("Found %d internal imports in public package:\n%s", len(violations), strings.Join(violations, "\n"))
	}
}

// Test_PublicAPI_Sufficiency verifies that the public API is sufficient for external users
func Test_PublicAPI_Sufficiency(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		code string
		desc string
	}{
		{
			name: "Client_Creation_With_Options",
			desc: "External users can create clients using the options pattern",
			code: `
package main

import (
	"context"
	ephemos "github.com/sufield/ephemos/pkg/ephemos"
	"github.com/sufield/ephemos/internal/core/ports"
)

func main() {
	ctx := context.Background()
	
	// Configuration is passed via options, not directly exposed
	config := &ports.Configuration{
		Service: ports.ServiceConfig{
			Name: "test-service",
			Domain: "example.com",
		},
	}
	
	// Client creation with options
	client, err := ephemos.IdentityClient(ctx, ephemos.WithConfig(config))
	if err != nil {
		panic(err)
	}
	defer client.Close()
}
`,
		},
		{
			name: "Server_Creation_With_Options",
			desc: "External users can create servers using the options pattern",
			code: `
package main

import (
	"context"
	ephemos "github.com/sufield/ephemos/pkg/ephemos"
	"github.com/sufield/ephemos/internal/core/ports"
)

func main() {
	ctx := context.Background()
	
	config := &ports.Configuration{
		Service: ports.ServiceConfig{
			Name: "test-service",
			Domain: "example.com",
		},
	}
	
	// Server creation with options
	server, err := ephemos.IdentityServer(ctx, 
		ephemos.WithServerConfig(config),
		ephemos.WithAddress(":8080"))
	if err != nil {
		panic(err)
	}
	defer server.Close()
}
`,
		},
		{
			name: "Client_Connection_Usage",
			desc: "External users can connect to services and get HTTP clients",
			code: `
package main

import (
	"context"
	ephemos "github.com/sufield/ephemos/pkg/ephemos"
	"github.com/sufield/ephemos/internal/core/ports"
)

func main() {
	ctx := context.Background()
	
	config := &ports.Configuration{
		Service: ports.ServiceConfig{
			Name: "test-service",
			Domain: "example.com",
		},
	}
	
	client, err := ephemos.IdentityClient(ctx, ephemos.WithConfig(config))
	if err != nil {
		panic(err)
	}
	defer client.Close()
	
	// Connect to a service
	conn, err := client.Connect(ctx, "payment-service:8080")
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	
	// Get HTTP client for authenticated requests
	httpClient, err := conn.HTTPClient()
	if err != nil {
		panic(err)
	}
	_ = httpClient
}
`,
		},
		{
			name: "Error_Handling",
			desc: "External users can handle errors using sentinel values",
			code: `
package main

import (
	"context"
	"errors"
	ephemos "github.com/sufield/ephemos/pkg/ephemos"
	"github.com/sufield/ephemos/internal/core/ports"
)

func main() {
	ctx := context.Background()
	
	config := &ports.Configuration{
		Service: ports.ServiceConfig{
			Name: "test-service",
			Domain: "example.com",
		},
	}
	
	client, err := ephemos.IdentityClient(ctx, ephemos.WithConfig(config))
	if err != nil {
		// Check for specific errors
		if errors.Is(err, ephemos.ErrConfigInvalid) {
			println("Invalid configuration")
		}
		return
	}
	defer client.Close()
	
	conn, err := client.Connect(ctx, "service:8080")
	if err != nil {
		if errors.Is(err, ephemos.ErrConnectionFailed) {
			println("Connection failed")
		}
		return
	}
	defer conn.Close()
}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			
			// Compile test to verify the public API is sufficient
			if err := compileTestCode(t, tt.code); err != nil {
				t.Errorf("%s failed: %v\nDescription: %s", tt.name, err, tt.desc)
			}
		})
	}
}

// Test_PublicAPI_Types verifies that expected types are accessible
func Test_PublicAPI_Types(t *testing.T) {
	t.Parallel()

	// These should compile successfully as they use the public API
	var _ ephemos.Client
	var _ ephemos.Server
	var _ ephemos.ClientOption
	var _ ephemos.ServerOption
	var _ ephemos.DialOption
	var _ error = ephemos.ErrNoAuth
	var _ error = ephemos.ErrConfigInvalid
	var _ error = ephemos.ErrConnectionFailed

	// Verify functions exist
	_ = ephemos.IdentityClient
	_ = ephemos.IdentityServer
	_ = ephemos.IdentityClientFromFile
	_ = ephemos.IdentityServerFromFile
	
	// Verify option functions exist
	_ = ephemos.WithConfig
	_ = ephemos.WithServerConfig
	_ = ephemos.WithAddress
	_ = ephemos.WithListener
	_ = ephemos.WithClientTimeout
	_ = ephemos.WithServerTimeout
	_ = ephemos.WithDialTimeout
}

// Test_PackageDocumentation ensures the public package has proper documentation
func Test_PackageDocumentation(t *testing.T) {
	t.Parallel()

	// Check that the main public_api.go has package documentation
	hasDoc, err := hasPackageDoc("public_api.go")
	if err != nil {
		t.Fatalf("failed to check package doc: %v", err)
	}
	if !hasDoc {
		t.Error("public_api.go should have package documentation starting with '// Package ephemos'")
	}

	// Check other files that should contribute to the API
	files := []string{
		"errors.go",
		"options.go",
	}

	for _, file := range files {
		path := file
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue // File may not exist, that's ok
		}
		
		// These files don't need package docs if public_api.go has it
		// But they should have type/function docs
		if err := checkFileHasTypeOrFunctionDocs(path); err != nil {
			t.Logf("INFO: %s might benefit from more documentation: %v", file, err)
		}
	}
}

// compileTestCode actually compiles the code to verify it works
func compileTestCode(t *testing.T, code string) error {
	t.Helper()
	
	dir := t.TempDir()
	
	// Get the absolute path to the ephemos module root
	ephemosRoot, err := filepath.Abs("../..")
	if err != nil {
		return fmt.Errorf("get ephemos root: %w", err)
	}
	
	// Create a temporary module
	modContent := fmt.Sprintf(`module testmod

go 1.22

require github.com/sufield/ephemos v0.0.0

replace github.com/sufield/ephemos => %s
`, ephemosRoot)
	modPath := filepath.Join(dir, "go.mod")
	if err := os.WriteFile(modPath, []byte(modContent), 0o644); err != nil {
		return fmt.Errorf("write go.mod: %w", err)
	}

	// Write the test code
	srcPath := filepath.Join(dir, "main.go")
	if err := os.WriteFile(srcPath, []byte(code), 0o644); err != nil {
		return fmt.Errorf("write main.go: %w", err)
	}

	// Run go mod tidy to resolve dependencies
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = dir
	tidyCmd.Env = append(os.Environ(), "GO111MODULE=on")
	if out, err := tidyCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("go mod tidy failed: %w\nOutput:\n%s", err, out)
	}

	// Try to build it
	cmd := exec.Command("go", "build", "-o", filepath.Join(dir, "bin"), srcPath)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go build failed: %w\nOutput:\n%s", err, out)
	}
	
	return nil
}

// isGeneratedFile checks if a file is generated (and should be skipped)
func isGeneratedFile(path string) bool {
	// Check common generated file patterns
	base := filepath.Base(path)
	if strings.HasSuffix(base, ".pb.go") ||
		strings.HasSuffix(base, "_grpc.pb.go") ||
		strings.HasPrefix(base, "zz_generated.") {
		return true
	}

	// Check for generated file header
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	// Read first few lines to check for generation marker
	buf := make([]byte, 512)
	n, _ := file.Read(buf)
	firstLines := string(buf[:n])
	
	return strings.Contains(firstLines, "Code generated") && 
		strings.Contains(firstLines, "DO NOT EDIT")
}

// hasPackageDoc checks if a file has package documentation
func hasPackageDoc(path string) (bool, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return false, err
	}
	
	if f.Doc == nil {
		return false, nil
	}
	
	for _, c := range f.Doc.List {
		if strings.HasPrefix(c.Text, "// Package ephemos") {
			return true, nil
		}
	}
	return false, nil
}

// checkFileHasTypeOrFunctionDocs verifies a file has some documentation
func checkFileHasTypeOrFunctionDocs(path string) error {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	hasAnyDocs := false
	
	// Check for any exported type or function with documentation
	ast.Inspect(f, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			if ast.IsExported(node.Name.Name) && node.Doc != nil {
				hasAnyDocs = true
				return false
			}
		case *ast.GenDecl:
			if node.Doc != nil {
				hasAnyDocs = true
				return false
			}
		}
		return true
	})

	if !hasAnyDocs {
		return fmt.Errorf("no documentation found for exported types or functions")
	}
	
	return nil
}