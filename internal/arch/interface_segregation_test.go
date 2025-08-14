// Package arch_test provides comprehensive architectural boundary tests.
// This file focuses on Interface Segregation Principle validation.
package arch_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test_Interface_Segregation ensures that ports follow ISP.
func Test_Interface_Segregation(t *testing.T) {
	portsDir := "../../internal/core/ports"

	err := filepath.Walk(portsDir, func(path string, _ os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		violations := checkInterfaceSize(t, path)
		if len(violations) > 0 {
			t.Errorf("Interface segregation violations in %s:\n%s", path, strings.Join(violations, "\n"))
		}

		return nil
	})
	if err != nil {
		t.Fatalf("Failed to walk ports directory: %v", err)
	}
}

// Test_Port_Naming_Conventions ensures consistent naming.
func Test_Port_Naming_Conventions(t *testing.T) {
	portsDir := "../../internal/core/ports"

	err := filepath.Walk(portsDir, func(path string, _ os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		violations := checkPortNaming(t, path)
		if len(violations) > 0 {
			t.Errorf("Port naming violations in %s:\n%s", path, strings.Join(violations, "\n"))
		}

		return nil
	})
	if err != nil {
		t.Fatalf("Failed to walk ports directory: %v", err)
	}
}

// Test_Adapter_Interface_Compliance ensures adapters properly implement ports.
func Test_Adapter_Interface_Compliance(t *testing.T) {
	// This test would require more sophisticated analysis to check that
	// adapters implement the ports they claim to implement.
	// For now, we'll do basic structural validation.

	adaptersDir := "../../internal/adapters"

	err := filepath.Walk(adaptersDir, func(path string, _ os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		violations := checkAdapterStructure(t, path)
		if len(violations) > 0 {
			t.Errorf("Adapter structure violations in %s:\n%s", path, strings.Join(violations, "\n"))
		}

		return nil
	})
	if err != nil {
		t.Fatalf("Failed to walk adapters directory: %v", err)
	}
}

// Test_Domain_Types_Are_Pure ensures domain types don't leak infrastructure concerns.
func Test_Domain_Types_Are_Pure(t *testing.T) {
	domainDir := "../../internal/core/domain"

	err := filepath.Walk(domainDir, func(path string, _ os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		violations := checkDomainPurity(t, path)
		if len(violations) > 0 {
			t.Errorf("Domain purity violations in %s:\n%s", path, strings.Join(violations, "\n"))
		}

		return nil
	})
	if err != nil {
		t.Fatalf("Failed to walk domain directory: %v", err)
	}
}

// Helper functions

func checkInterfaceSize(t *testing.T, filePath string) []string {
	t.Helper()

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse %s: %v", filePath, err)
	}

	var violations []string

	ast.Inspect(node, func(n ast.Node) bool {
		if ts, ok := n.(*ast.TypeSpec); ok {
			if iface, ok := ts.Type.(*ast.InterfaceType); ok {
				methodCount := len(iface.Methods.List)

				// Interfaces should be small and focused (ISP)
				// Allow some flexibility, but flag very large interfaces
				maxMethods := 7 // Reasonable upper bound

				if methodCount > maxMethods {
					violations = append(violations,
						fmt.Sprintf("Interface %s has %d methods (max recommended: %d)",
							ts.Name.Name, methodCount, maxMethods))
				}
			}
		}
		return true
	})

	return violations
}

func checkPortNaming(t *testing.T, filePath string) []string {
	t.Helper()

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse %s: %v", filePath, err)
	}

	var violations []string

	ast.Inspect(node, func(n ast.Node) bool {
		if ts, ok := n.(*ast.TypeSpec); ok { //nolint:nestif // Necessary for AST traversal
			if _, ok := ts.Type.(*ast.InterfaceType); ok {
				name := ts.Name.Name

				// Port interfaces should follow naming conventions
				if !strings.HasSuffix(name, "Port") &&
					!strings.HasSuffix(name, "Provider") &&
					!strings.HasSuffix(name, "Service") &&
					!strings.HasSuffix(name, "Repository") {
					violations = append(violations,
						fmt.Sprintf("Interface %s doesn't follow port naming conventions (should end with Port, Provider, Service, or Repository)", name))
				}

				// Should be exported (start with capital letter)
				if !ast.IsExported(name) {
					violations = append(violations,
						fmt.Sprintf("Port interface %s should be exported", name))
				}
			}
		}
		return true
	})

	return violations
}

func checkAdapterStructure(t *testing.T, filePath string) []string {
	t.Helper()

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse %s: %v", filePath, err)
	}

	var violations []string

	// Check that adapter files have proper structure
	hasStruct := false
	_ = false // hasInterface used for future extensibility

	ast.Inspect(node, func(n ast.Node) bool {
		if ts, ok := n.(*ast.TypeSpec); ok {
			if _, ok := ts.Type.(*ast.StructType); ok {
				hasStruct = true
			}
			if _, ok := ts.Type.(*ast.InterfaceType); ok {
				_ = true // hasInterface - reserved for future structural checks
			}
		}
		return true
	})

	// Adapters typically should have structs (implementations)
	// Primary adapters might have interfaces for external contracts
	if !hasStruct {
		violations = append(violations, "Adapter file should contain at least one struct type")
	}

	return violations
}

//nolint:cyclop,nestif // Test helper complexity acceptable for thorough validation
func checkDomainPurity(t *testing.T, filePath string) []string {
	t.Helper()

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse %s: %v", filePath, err)
	}

	var violations []string

	// Check for infrastructure-specific types in domain
	infrastructureTypes := []string{
		"grpc", "http", "sql", "db", "database", "redis", "kafka",
		"json", "xml", "serialization", "pb", "rest", "graphql",
	}

	ast.Inspect(node, func(n ast.Node) bool {
		// Check struct fields
		if ts, ok := n.(*ast.TypeSpec); ok {
			if st, ok := ts.Type.(*ast.StructType); ok {
				for _, field := range st.Fields.List {
					for _, name := range field.Names {
						fieldName := strings.ToLower(name.Name)
						for _, infraType := range infrastructureTypes {
							if strings.Contains(fieldName, infraType) {
								violations = append(violations,
									fmt.Sprintf("Domain type %s has infrastructure-specific field %s",
										ts.Name.Name, name.Name))
							}
						}
					}

					// Check field types
					if ident, ok := field.Type.(*ast.Ident); ok {
						typeName := strings.ToLower(ident.Name)
						for _, infraType := range infrastructureTypes {
							if strings.Contains(typeName, infraType) {
								violations = append(violations,
									fmt.Sprintf("Domain uses infrastructure-specific type %s", ident.Name))
							}
						}
					}
				}
			}
		}

		// Check function names
		if fd, ok := n.(*ast.FuncDecl); ok {
			if fd.Name != nil {
				funcName := strings.ToLower(fd.Name.Name)
				for _, infraType := range infrastructureTypes {
					if strings.Contains(funcName, infraType) {
						violations = append(violations,
							fmt.Sprintf("Domain function %s has infrastructure-specific name", fd.Name.Name))
					}
				}
			}
		}

		return true
	})

	return violations
}
