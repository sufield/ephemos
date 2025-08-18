// Package arch_test provides architectural constraint tests to prevent design violations.
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

var fset = token.NewFileSet()

// TestNoLongSelectorChains ensures that selector chains don't exceed reasonable depth.
// Long chains (w.x.y.z) typically indicate Law of Demeter violations.
func TestNoLongSelectorChains(t *testing.T) {
	// Check public API packages - strict limits
	checkSelectorDepth(t, "../../pkg", 2, "Public API should use facade methods, not deep chains")
	
	// Check core packages - slightly more lenient  
	checkSelectorDepth(t, "../../internal/core", 3, "Core packages should minimize deep access")
	
	// Check adapters - most lenient but still constrained
	checkSelectorDepth(t, "../../internal/adapters", 4, "Adapters should use dependency injection over deep config access")
}

// TestNoCrossPackageDeepAccess specifically checks for configuration deep access patterns
// that indicate tight coupling between components and configuration structure.
func TestNoCrossPackageDeepAccess(t *testing.T) {
	violations := findConfigDeepAccess("../../internal/adapters")
	if len(violations) > 0 {
		t.Errorf("Found %d config deep access violations that should use capability injection:\n%s", 
			len(violations), strings.Join(violations, "\n"))
	}
	
	// Check for direct certificate field access
	certViolations := findCertificateFieldAccess("../../examples")
	if len(certViolations) > 0 {
		t.Errorf("Found %d certificate field access violations that should use domain methods:\n%s",
			len(certViolations), strings.Join(certViolations, "\n"))
	}
}

// TestVendorTypeIsolation ensures that vendor-specific types from go-spiffe 
// don't leak into the public API surface.
func TestVendorTypeIsolation(t *testing.T) {
	vendorTypes := []string{
		"spiffeid.ID",
		"spiffeid.TrustDomain", 
		"x509svid.SVID",
		"tlsconfig.Authorizer",
		"x509bundle.Bundle",
	}
	
	checkVendorLeakage(t, "../../pkg", vendorTypes)
	checkVendorLeakage(t, "../../examples", vendorTypes)
}

// checkSelectorDepth walks the AST and reports selector chains exceeding maxDepth.
func checkSelectorDepth(t *testing.T, rootPath string, maxDepth int, context string) {
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		
		file, err := parser.ParseFile(fset, path, content, 0)
		if err != nil {
			return err
		}
		
		ast.Inspect(file, func(n ast.Node) bool {
			if sel, ok := n.(*ast.SelectorExpr); ok {
				depth := getSelectorDepth(sel)
				if depth > maxDepth {
					pos := fset.Position(sel.Pos())
					t.Errorf("%s:%d:%d: Long selector chain (depth=%d, max=%d) - %s\n%s", 
						pos.Filename, pos.Line, pos.Column, depth, maxDepth, context, 
						extractLineContext(content, pos.Line))
				}
			}
			return true
		})
		
		return nil
	})
	
	if err != nil {
		t.Fatalf("Failed to walk directory %s: %v", rootPath, err)
	}
}

// getSelectorDepth calculates the depth of a selector expression.
// Example: a.b.c.d has depth 3 (3 dots)
func getSelectorDepth(expr ast.Expr) int {
	depth := 0
	for {
		sel, ok := expr.(*ast.SelectorExpr)
		if !ok {
			break
		}
		depth++
		expr = sel.X
	}
	return depth
}

// findConfigDeepAccess looks for patterns like "config.Service.Domain" that
// indicate tight coupling to configuration structure.
func findConfigDeepAccess(rootPath string) []string {
	var violations []string
	
	// Pattern to detect: config.Something.SomethingElse
	patterns := []string{
		".config.Service.",
		".config.Agent.", 
		".config.Health.",
	}
	
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		
		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			for _, pattern := range patterns {
				if strings.Contains(line, pattern) {
					violations = append(violations, fmt.Sprintf("%s:%d: %s", 
						path, i+1, strings.TrimSpace(line)))
				}
			}
		}
		
		return nil
	})
	
	if err != nil {
		// Log error but don't fail the test
		return []string{fmt.Sprintf("Error scanning %s: %v", rootPath, err)}
	}
	
	return violations
}

// findCertificateFieldAccess looks for patterns like "cert.Cert.NotAfter" that
// expose X.509 certificate internals.
func findCertificateFieldAccess(rootPath string) []string {
	var violations []string
	
	// Patterns that indicate direct certificate field access
	patterns := []string{
		".Cert.NotAfter",
		".Cert.NotBefore", 
		".Cert.Subject",
		".Cert.Issuer",
	}
	
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		
		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			for _, pattern := range patterns {
				if strings.Contains(line, pattern) {
					violations = append(violations, fmt.Sprintf("%s:%d: %s", 
						path, i+1, strings.TrimSpace(line)))
				}
			}
		}
		
		return nil
	})
	
	if err != nil {
		return []string{fmt.Sprintf("Error scanning %s: %v", rootPath, err)}
	}
	
	return violations
}

// checkVendorLeakage ensures vendor types don't appear in public interfaces.
func checkVendorLeakage(t *testing.T, rootPath string, vendorTypes []string) {
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		
		contentStr := string(content)
		lines := strings.Split(contentStr, "\n")
		
		for i, line := range lines {
			// Skip import statements and comments
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "import") || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
				continue
			}
			
			for _, vendorType := range vendorTypes {
				if strings.Contains(line, vendorType) {
					t.Errorf("%s:%d: Vendor type %s leaked into public interface\n%s", 
						path, i+1, vendorType, trimmed)
				}
			}
		}
		
		return nil
	})
	
	if err != nil {
		t.Fatalf("Failed to check vendor leakage in %s: %v", rootPath, err)
	}
}

// extractLineContext extracts the source line for better error messages.
func extractLineContext(content []byte, lineNum int) string {
	lines := strings.Split(string(content), "\n")
	if lineNum > 0 && lineNum <= len(lines) {
		return strings.TrimSpace(lines[lineNum-1])
	}
	return ""
}