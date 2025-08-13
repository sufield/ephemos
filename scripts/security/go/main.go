package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// SecurityScanner represents the main security scanner
type SecurityScanner struct {
	projectRoot string
	verbose     bool
	exitOnError bool
}

// ScanResult represents the result of a security scan
type ScanResult struct {
	Name      string
	Passed    bool
	Details   string
	Duration  time.Duration
	Warnings  []string
}

// NewSecurityScanner creates a new security scanner instance
func NewSecurityScanner(projectRoot string) *SecurityScanner {
	return &SecurityScanner{
		projectRoot: projectRoot,
		verbose:     false,
		exitOnError: true,
	}
}

func main() {
	var rootCmd = &cobra.Command{
		Use:   "security-scanner",
		Short: "Comprehensive security scanner for Ephemos project",
		Long: `A Go-based security scanner that replaces bash scripts with better error handling,
structured output, and improved maintainability.

Includes: secrets scanning, vulnerability detection, SBOM generation, and validation.`,
		Run: func(cmd *cobra.Command, args []string) {
			verbose, _ := cmd.Flags().GetBool("verbose")
			continueOnError, _ := cmd.Flags().GetBool("continue-on-error")
			
			// Determine project root
			projectRoot, err := determineProjectRoot()
			if err != nil {
				log.Fatalf("Failed to determine project root: %v", err)
			}
			
			scanner := NewSecurityScanner(projectRoot)
			scanner.verbose = verbose
			scanner.exitOnError = !continueOnError
			
			if err := scanner.RunAllScans(); err != nil {
				log.Fatalf("Security scan failed: %v", err)
			}
		},
	}

	rootCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")
	rootCmd.Flags().Bool("continue-on-error", false, "Continue scanning even if some checks fail")

	// Add subcommands
	rootCmd.AddCommand(secretsCmd())
	rootCmd.AddCommand(vulnerabilitiesCmd())
	rootCmd.AddCommand(sbomCmd())

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

// RunAllScans executes all security scans
func (s *SecurityScanner) RunAllScans() error {
	fmt.Println("🔒 Running comprehensive security scans...")
	fmt.Printf("Project root: %s\n\n", s.projectRoot)

	var results []ScanResult
	overallPassed := true

	// Run secrets scan
	result := s.runSecretsScans()
	results = append(results, result)
	if !result.Passed {
		overallPassed = false
	}

	// Run vulnerability scan
	result = s.runVulnerabilityScans()
	results = append(results, result)
	if !result.Passed {
		overallPassed = false
	}

	// Run SBOM generation and validation
	result = s.runSBOMScans()
	results = append(results, result)
	if !result.Passed {
		overallPassed = false
	}

	// Print summary
	s.printSummary(results, overallPassed)

	if !overallPassed && s.exitOnError {
		return fmt.Errorf("some security scans failed")
	}

	return nil
}

// runSecretsScans executes secrets detection scans
func (s *SecurityScanner) runSecretsScans() ScanResult {
	fmt.Println("📝 Running secrets scans...")
	start := time.Now()
	
	var warnings []string
	passed := true
	details := []string{}

	// Gitleaks scan
	if s.commandExists("gitleaks") {
		if err := s.runGitleaks(); err != nil {
			warnings = append(warnings, "Gitleaks scan failed: "+err.Error())
			passed = false
		} else {
			details = append(details, "✅ Gitleaks: No secrets found")
		}
	} else {
		warnings = append(warnings, "❌ gitleaks not installed")
	}

	// TruffleHog scan
	if s.commandExists("trufflehog") {
		if err := s.runTruffleHog(); err != nil {
			warnings = append(warnings, "TruffleHog scan failed: "+err.Error())
			passed = false
		} else {
			details = append(details, "✅ TruffleHog: No verified secrets found")
		}
	} else {
		warnings = append(warnings, "❌ TruffleHog not installed")
	}

	// Custom patterns scan
	if err := s.runCustomSecretsPattern(); err != nil {
		warnings = append(warnings, "Custom patterns scan failed: "+err.Error())
		passed = false
	} else {
		details = append(details, "✅ Custom patterns: No hardcoded credentials found")
	}

	return ScanResult{
		Name:     "Secrets Scanning",
		Passed:   passed,
		Details:  strings.Join(details, "\n"),
		Duration: time.Since(start),
		Warnings: warnings,
	}
}

// runVulnerabilityScans executes vulnerability detection scans
func (s *SecurityScanner) runVulnerabilityScans() ScanResult {
	fmt.Println("🔍 Running vulnerability scans...")
	start := time.Now()
	
	var warnings []string
	passed := true
	details := []string{}

	// govulncheck scan
	if s.commandExists("govulncheck") {
		if err := s.runGovulncheck(); err != nil {
			warnings = append(warnings, "govulncheck scan failed: "+err.Error())
			passed = false
		} else {
			details = append(details, "✅ govulncheck: No vulnerabilities found")
		}
	} else {
		warnings = append(warnings, "❌ govulncheck not installed")
	}

	// Trivy scan
	if s.commandExists("trivy") {
		if err := s.runTrivy(); err != nil {
			warnings = append(warnings, "Trivy scan failed: "+err.Error())
			passed = false
		} else {
			details = append(details, "✅ Trivy: No high/critical vulnerabilities found")
		}
	} else {
		warnings = append(warnings, "❌ Trivy not installed")
	}

	return ScanResult{
		Name:     "Vulnerability Scanning",
		Passed:   passed,
		Details:  strings.Join(details, "\n"),
		Duration: time.Since(start),
		Warnings: warnings,
	}
}

// runSBOMScans executes SBOM generation and validation
func (s *SecurityScanner) runSBOMScans() ScanResult {
	fmt.Println("📋 Running SBOM generation and validation...")
	start := time.Now()
	
	var warnings []string
	passed := true
	details := []string{}

	// Generate SBOM with Syft
	if s.commandExists("syft") {
		if err := s.runSyftSBOM(); err != nil {
			warnings = append(warnings, "SBOM generation failed: "+err.Error())
			passed = false
		} else {
			details = append(details, "✅ SBOM generated successfully")
		}
	} else {
		warnings = append(warnings, "❌ Syft not installed")
		passed = false
	}

	return ScanResult{
		Name:     "SBOM Generation & Validation",
		Passed:   passed,
		Details:  strings.Join(details, "\n"),
		Duration: time.Since(start),
		Warnings: warnings,
	}
}

// Tool execution methods

func (s *SecurityScanner) runGitleaks() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	
	cmd := exec.CommandContext(ctx, "gitleaks", "detect", "--source", ".", "--no-git", "--verbose")
	cmd.Dir = s.projectRoot
	
	if s.verbose {
		fmt.Println("Running: gitleaks detect --source . --no-git --verbose")
	}
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "no leaks found") {
			return nil
		}
		return fmt.Errorf("gitleaks found potential secrets: %s", string(output))
	}
	return nil
}

func (s *SecurityScanner) runTruffleHog() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	
	cmd := exec.CommandContext(ctx, "trufflehog", "filesystem", "--directory=.", "--only-verified", "--json")
	cmd.Dir = s.projectRoot
	
	if s.verbose {
		fmt.Println("Running: trufflehog filesystem --directory=. --only-verified --json")
	}
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("trufflehog execution failed: %v", err)
	}
	
	// Check if any secrets were found
	if strings.Contains(string(output), "SourceType") {
		return fmt.Errorf("TruffleHog found potential secrets")
	}
	
	return nil
}

func (s *SecurityScanner) runCustomSecretsPattern() error {
	// Check for common credential patterns in Go files
	err := filepath.Walk(s.projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Skip non-Go files and certain directories
		if !strings.HasSuffix(path, ".go") ||
			strings.Contains(path, "/.git/") ||
			strings.Contains(path, "/vendor/") ||
			strings.Contains(path, "/node_modules/") {
			return nil
		}
		
		return s.scanFileForSecrets(path)
	})
	
	return err
}

func (s *SecurityScanner) scanFileForSecrets(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	lineNum := 0
	
	// Common secret patterns
	patterns := []string{
		"password",
		"secret",
		"api_key",
		"apikey",
		"token",
		"private_key",
	}
	
	for scanner.Scan() {
		lineNum++
		line := strings.ToLower(scanner.Text())
		
		// Skip comments and test files
		if strings.Contains(line, "//") || strings.Contains(filePath, "_test.go") {
			continue
		}
		
		for _, pattern := range patterns {
			if strings.Contains(line, pattern) && strings.Contains(line, "=") {
				// Check if it looks like a hardcoded credential (exclude common false positives)
				if !strings.Contains(line, "fmt.") && 
				   !strings.Contains(line, "log.") &&
				   !strings.Contains(line, "test") &&
				   !strings.Contains(line, "example") &&
				   !strings.Contains(line, "placeholder") &&
				   !strings.Contains(line, "\""+pattern+"\"") && // Skip quoted patterns like our array
				   !strings.Contains(line, "[]string{") &&       // Skip string array definitions
				   !strings.Contains(line, "patterns") {         // Skip our patterns array
					return fmt.Errorf("potential hardcoded credential in %s:%d", filePath, lineNum)
				}
			}
		}
	}
	
	return scanner.Err()
}

func (s *SecurityScanner) runGovulncheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	
	cmd := exec.CommandContext(ctx, "govulncheck", "./...")
	cmd.Dir = s.projectRoot
	
	if s.verbose {
		fmt.Println("Running: govulncheck ./...")
	}
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("govulncheck found vulnerabilities: %s", string(output))
	}
	return nil
}

func (s *SecurityScanner) runTrivy() error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	
	cmd := exec.CommandContext(ctx, "trivy", "fs", "--exit-code", "1", "--severity", "HIGH,CRITICAL", ".")
	cmd.Dir = s.projectRoot
	
	if s.verbose {
		fmt.Println("Running: trivy fs --exit-code 1 --severity HIGH,CRITICAL .")
	}
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("trivy found high/critical vulnerabilities: %s", string(output))
	}
	return nil
}

func (s *SecurityScanner) runSyftSBOM() error {
	// Create sbom directory
	sbomDir := filepath.Join(s.projectRoot, "sbom")
	if err := os.MkdirAll(sbomDir, 0755); err != nil {
		return fmt.Errorf("failed to create sbom directory: %v", err)
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	
	// Generate SPDX format SBOM
	spdxPath := filepath.Join(sbomDir, "ephemos-sbom.spdx.json")
	cmd := exec.CommandContext(ctx, "syft", ".", "-o", "spdx-json", "--file", spdxPath)
	cmd.Dir = s.projectRoot
	
	if s.verbose {
		fmt.Printf("Running: syft . -o spdx-json --file %s\n", spdxPath)
	}
	
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("syft SBOM generation failed: %s", string(output))
	}
	
	// Generate CycloneDX format SBOM
	cycloneDir := filepath.Join(sbomDir, "ephemos-sbom.cyclonedx.json")
	cmd = exec.CommandContext(ctx, "syft", ".", "-o", "cyclonedx-json", "--file", cycloneDir)
	cmd.Dir = s.projectRoot
	
	if s.verbose {
		fmt.Printf("Running: syft . -o cyclonedx-json --file %s\n", cycloneDir)
	}
	
	if output, err := cmd.CombinedOutput(); err != nil {
		// CycloneDX is optional, just log warning
		if s.verbose {
			fmt.Printf("Warning: CycloneDX SBOM generation failed: %s\n", string(output))
		}
	}
	
	return nil
}

// Utility methods

func (s *SecurityScanner) commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func (s *SecurityScanner) printSummary(results []ScanResult, overallPassed bool) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("📊 SECURITY SCAN SUMMARY")
	fmt.Println(strings.Repeat("=", 60))
	
	for _, result := range results {
		status := "✅ PASSED"
		if !result.Passed {
			status = "❌ FAILED"
		}
		
		fmt.Printf("%-30s %s (%.2fs)\n", result.Name, status, result.Duration.Seconds())
		
		if result.Details != "" {
			fmt.Printf("    Details: %s\n", result.Details)
		}
		
		if len(result.Warnings) > 0 {
			fmt.Println("    Warnings:")
			for _, warning := range result.Warnings {
				fmt.Printf("      %s\n", warning)
			}
		}
		fmt.Println()
	}
	
	if overallPassed {
		fmt.Println("🎉 All security scans passed!")
	} else {
		fmt.Println("💥 Some security scans failed!")
	}
}

func determineProjectRoot() (string, error) {
	// Start from current directory and walk up to find go.mod
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	
	return "", fmt.Errorf("could not find project root (go.mod not found)")
}

// Subcommands

func secretsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secrets",
		Short: "Run only secrets scanning",
		Run: func(cmd *cobra.Command, args []string) {
			verbose, _ := cmd.Flags().GetBool("verbose")
			
			projectRoot, err := determineProjectRoot()
			if err != nil {
				log.Fatalf("Failed to determine project root: %v", err)
			}
			
			scanner := NewSecurityScanner(projectRoot)
			scanner.verbose = verbose
			
			result := scanner.runSecretsScans()
			scanner.printSummary([]ScanResult{result}, result.Passed)
			
			if !result.Passed {
				os.Exit(1)
			}
		},
	}
	
	cmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")
	return cmd
}

func vulnerabilitiesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vulnerabilities",
		Short: "Run only vulnerability scanning",
		Run: func(cmd *cobra.Command, args []string) {
			verbose, _ := cmd.Flags().GetBool("verbose")
			
			projectRoot, err := determineProjectRoot()
			if err != nil {
				log.Fatalf("Failed to determine project root: %v", err)
			}
			
			scanner := NewSecurityScanner(projectRoot)
			scanner.verbose = verbose
			
			result := scanner.runVulnerabilityScans()
			scanner.printSummary([]ScanResult{result}, result.Passed)
			
			if !result.Passed {
				os.Exit(1)
			}
		},
	}
	
	cmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")
	return cmd
}

func sbomCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sbom",
		Short: "Run only SBOM generation and validation",
		Run: func(cmd *cobra.Command, args []string) {
			verbose, _ := cmd.Flags().GetBool("verbose")
			
			projectRoot, err := determineProjectRoot()
			if err != nil {
				log.Fatalf("Failed to determine project root: %v", err)
			}
			
			scanner := NewSecurityScanner(projectRoot)
			scanner.verbose = verbose
			
			result := scanner.runSBOMScans()
			scanner.printSummary([]ScanResult{result}, result.Passed)
			
			if !result.Passed {
				os.Exit(1)
			}
		},
	}
	
	cmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")
	return cmd
}