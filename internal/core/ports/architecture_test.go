// Package ports contains architecture tests to enforce import boundaries
// and ensure the hexagonal architecture is maintained.
package ports_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDomainPortsHaveNoProtocolDependencies ensures that domain ports
// don't import any protocol-specific packages (gRPC, HTTP, etc.)
func TestDomainPortsHaveNoProtocolDependencies(t *testing.T) {
	prohibited := []string{
		"google.golang.org/grpc",
		"google.golang.org/protobuf",
		"net/http",
		"github.com/gin-gonic",
		"github.com/gorilla/mux",
		"github.com/labstack/echo",
		"github.com/valyala/fasthttp",
	}

	err := filepath.Walk("../../../internal/core/ports", func(path string, _ os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		violations := checkFileImports(t, path, prohibited)
		if len(violations) > 0 {
			t.Errorf("Domain ports file %s imports prohibited packages: %v", path, violations)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("Failed to walk domain ports directory: %v", err)
	}
}

// TestAdaptersCanImportProtocolPackages ensures that adapters CAN import
// protocol packages (this is expected and required).
func TestAdaptersCanImportProtocolPackages(t *testing.T) {
	// Test that gRPC adapter imports gRPC packages
	grpcAdapterPath := "../../../internal/adapters/grpc/adapter.go"
	requiredGRPCImports := []string{
		"google.golang.org/grpc",
	}

	if !fileExistsAndImports(t, grpcAdapterPath, requiredGRPCImports) {
		t.Error("gRPC adapter should import gRPC packages")
	}

	// Test that HTTP adapter imports HTTP packages
	httpAdapterPath := "../../../internal/adapters/http/adapter.go"
	requiredHTTPImports := []string{
		"net/http",
	}

	if !fileExistsAndImports(t, httpAdapterPath, requiredHTTPImports) {
		t.Error("HTTP adapter should import HTTP packages")
	}
}

// TestPublicAPIHasNoDirectProtocolDependencies ensures that the public API
// (pkg/ephemos) doesn't directly depend on protocols, only on domain interfaces.
func TestPublicAPIHasNoDirectProtocolDependencies(t *testing.T) {
	// The public API should not directly import protocol packages
	// It should only depend on domain interfaces and adapters
	prohibited := []string{
		"google.golang.org/protobuf", // protobuf should not leak to public API
		// Note: google.golang.org/grpc is allowed for the legacy Server interface
		// but the new TransportServer should be protocol-agnostic
	}

	publicAPIPath := "../../../pkg/ephemos"
	err := filepath.Walk(publicAPIPath, func(path string, _ os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		violations := checkFileImports(t, path, prohibited)
		if len(violations) > 0 {
			t.Errorf("Public API file %s imports prohibited packages: %v", path, violations)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("Failed to walk public API directory: %v", err)
	}
}

// TestTransportServerIsProtocolAgnostic specifically tests that the new
// TransportServer implementation doesn't leak protocol details.
func TestTransportServerIsProtocolAgnostic(t *testing.T) {
	serverFile := "../../../pkg/ephemos/server.go"

	// Parse the file to check its structure
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, serverFile, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse server.go: %v", err)
	}

	// Check that TransportServer struct doesn't expose protocol-specific fields
	ast.Inspect(node, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok || ts.Name.Name != "TransportServer" {
			return true
		}

		if st, ok := ts.Type.(*ast.StructType); ok {
			checkStructFields(t, st)
		}
		return true
	})
}

// TestMountAPIIsGeneric ensures that the Mount function is truly generic
// and doesn't have protocol-specific constraints.
func TestMountAPIIsGeneric(t *testing.T) {
	serverFile := "../../../pkg/ephemos/server.go"

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, serverFile, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse server.go: %v", err)
	}

	foundMount := false
	ast.Inspect(node, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok && fn.Name.Name == "Mount" {
			foundMount = true
			validateMountFunction(t, fn)
		}
		return true
	})

	if !foundMount {
		t.Error("Mount function not found in server.go")
	}
}

// Helper functions

func checkStructFields(t *testing.T, st *ast.StructType) {
	t.Helper()

	for _, field := range st.Fields.List {
		for _, name := range field.Names {
			// These fields should be private (internal to the server)
			if !name.IsExported() {
				continue // Private fields are okay
			}

			// No exported fields should be protocol-specific
			if strings.Contains(name.Name, "grpc") ||
				strings.Contains(name.Name, "http") ||
				strings.Contains(name.Name, "HTTP") ||
				strings.Contains(name.Name, "GRPC") {
				t.Errorf("TransportServer has exported protocol-specific field: %s", name.Name)
			}
		}
	}
}

func checkFileImports(t *testing.T, filePath string, prohibited []string) []string {
	t.Helper()
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse %s: %v", filePath, err)
	}

	var violations []string
	for _, imp := range node.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		for _, forbidden := range prohibited {
			if importPath == forbidden || strings.HasPrefix(importPath, forbidden+"/") {
				violations = append(violations, importPath)
			}
		}
	}

	return violations
}

func fileExistsAndImports(t *testing.T, filePath string, required []string) bool {
	t.Helper()
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		t.Logf("Could not parse %s: %v", filePath, err)
		return false
	}

	importMap := make(map[string]bool)
	for _, imp := range node.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		importMap[importPath] = true
	}

	for _, req := range required {
		if !importMap[req] {
			t.Logf("File %s missing required import: %s", filePath, req)
			return false
		}
	}

	return true
}

func validateMountFunction(t *testing.T, fn *ast.FuncDecl) {
	t.Helper()

	checkGenericTypeParameter(t, fn)
	checkParameterTypes(t, fn)
}

func checkGenericTypeParameter(t *testing.T, fn *ast.FuncDecl) {
	t.Helper()

	if fn.Type.TypeParams == nil || len(fn.Type.TypeParams.List) == 0 {
		t.Error("Mount function should have generic type parameter [T any]")
	}
}

func checkParameterTypes(t *testing.T, fn *ast.FuncDecl) {
	t.Helper()

	for _, param := range fn.Type.Params.List {
		if ident, ok := param.Type.(*ast.Ident); ok {
			if isProtocolSpecific(ident.Name) {
				t.Errorf("Mount function has protocol-specific parameter type: %s", ident.Name)
			}
		}
	}
}

func isProtocolSpecific(name string) bool {
	return strings.Contains(name, "grpc") ||
		strings.Contains(name, "http") ||
		strings.Contains(name, "pb")
}
