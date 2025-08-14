package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// YAMLFile represents a YAML file with its path and content
type YAMLFile struct {
	Path    string
	Content []byte
}

// ValidationResult holds the result of validating a single YAML file
type ValidationResult struct {
	Path  string
	Valid bool
	Error error
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run validate-yaml.go <directory>")
		fmt.Println("Example: go run validate-yaml.go .")
		os.Exit(1)
	}

	rootDir := os.Args[1]

	// Find all YAML files
	yamlFiles, err := findYAMLFiles(rootDir)
	if err != nil {
		fmt.Printf("Error finding YAML files: %v\n", err)
		os.Exit(1)
	}

	if len(yamlFiles) == 0 {
		fmt.Println("No YAML files found")
		return
	}

	fmt.Printf("Found %d YAML files to validate:\n", len(yamlFiles))

	// Validate each file
	var results []ValidationResult
	var hasErrors bool

	for _, yamlFile := range yamlFiles {
		result := validateYAMLFile(yamlFile)
		results = append(results, result)

		if result.Valid {
			fmt.Printf("✅ %s - Valid\n", result.Path)
		} else {
			fmt.Printf("❌ %s - Error: %v\n", result.Path, result.Error)
			hasErrors = true
		}
	}

	// Summary
	fmt.Printf("\n--- Validation Summary ---\n")
	validCount := 0
	for _, result := range results {
		if result.Valid {
			validCount++
		}
	}

	fmt.Printf("Valid files: %d/%d\n", validCount, len(results))

	if hasErrors {
		fmt.Printf("❌ Validation failed - %d files have errors\n", len(results)-validCount)
		os.Exit(1)
	} else {
		fmt.Println("✅ All YAML files are valid")
	}
}

// findYAMLFiles recursively finds all .yml and .yaml files in the directory
func findYAMLFiles(rootDir string) ([]YAMLFile, error) {
	var yamlFiles []YAMLFile

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories and files, except .github (but don't skip root ".")
		if strings.HasPrefix(d.Name(), ".") && d.IsDir() && d.Name() != ".github" && path != rootDir {
			return filepath.SkipDir
		}

		// Skip vendor, node_modules, and other common directories
		if d.IsDir() && (d.Name() == "vendor" || d.Name() == "node_modules" || d.Name() == ".git") {
			return filepath.SkipDir
		}

		// Check for YAML files
		if !d.IsDir() && (strings.HasSuffix(path, ".yml") || strings.HasSuffix(path, ".yaml")) {
			content, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failed to read %s: %w", path, err)
			}

			yamlFiles = append(yamlFiles, YAMLFile{
				Path:    path,
				Content: content,
			})
		}

		return nil
	})

	return yamlFiles, err
}

// validateYAMLFile validates a single YAML file
func validateYAMLFile(yamlFile YAMLFile) ValidationResult {
	var data interface{}

	// Try to parse the YAML
	err := yaml.Unmarshal(yamlFile.Content, &data)
	if err != nil {
		return ValidationResult{
			Path:  yamlFile.Path,
			Valid: false,
			Error: fmt.Errorf("YAML parse error: %w", err),
		}
	}

	// Additional validation for empty files
	if data == nil {
		return ValidationResult{
			Path:  yamlFile.Path,
			Valid: false,
			Error: fmt.Errorf("empty YAML file"),
		}
	}

	return ValidationResult{
		Path:  yamlFile.Path,
		Valid: true,
		Error: nil,
	}
}
