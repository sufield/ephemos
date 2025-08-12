package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

// TestEvent represents a test event from go test -json
type TestEvent struct {
	Time    time.Time `json:"Time"`
	Action  string    `json:"Action"`
	Package string    `json:"Package"`
	Test    string    `json:"Test"`
	Output  string    `json:"Output"`
	Elapsed float64   `json:"Elapsed"`
}

// PackageResult tracks results for a package
type PackageResult struct {
	Package      string
	Passed       int
	Failed       int
	Skipped      int
	Errors       []string
	FailedTests  []string
	TotalElapsed float64
	BuildFailed  bool
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run test-summary.go <test-results.json>")
		os.Exit(1)
	}

	file, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	results := make(map[string]*PackageResult)
	scanner := bufio.NewScanner(file)
	
	var totalTests, passedTests, failedTests, skippedTests int
	var compilationErrors []string

	for scanner.Scan() {
		var event TestEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			// Check if it's a compilation error line
			line := scanner.Text()
			if strings.Contains(line, "build failed") || strings.Contains(line, "undefined") {
				compilationErrors = append(compilationErrors, line)
			}
			continue
		}

		// Initialize package result if needed
		if _, exists := results[event.Package]; !exists {
			results[event.Package] = &PackageResult{
				Package:     event.Package,
				Errors:      []string{},
				FailedTests: []string{},
			}
		}

		pkg := results[event.Package]

		// Track build failures
		if event.Action == "fail" && event.Test == "" && strings.Contains(event.Output, "build failed") {
			pkg.BuildFailed = true
		}

		// Track test results
		if event.Test != "" {
			switch event.Action {
			case "pass":
				pkg.Passed++
				passedTests++
				totalTests++
			case "fail":
				pkg.Failed++
				failedTests++
				totalTests++
				pkg.FailedTests = append(pkg.FailedTests, event.Test)
				// Capture failure output
				if strings.Contains(event.Output, "Error") || strings.Contains(event.Output, "FAIL") {
					pkg.Errors = append(pkg.Errors, strings.TrimSpace(event.Output))
				}
			case "skip":
				pkg.Skipped++
				skippedTests++
				totalTests++
			}
		}

		// Track elapsed time for package
		if event.Elapsed > 0 {
			pkg.TotalElapsed = event.Elapsed
		}
	}

	// Sort packages for consistent output
	var packages []string
	for pkg := range results {
		packages = append(packages, pkg)
	}
	sort.Strings(packages)

	// Print compilation errors first if any
	if len(compilationErrors) > 0 {
		fmt.Println("❌ COMPILATION ERRORS DETECTED:")
		fmt.Println("--------------------------------")
		for _, err := range compilationErrors {
			fmt.Printf("  %s\n", err)
		}
		fmt.Println()
	}

	// Print per-package results
	fmt.Println("📦 PACKAGE RESULTS:")
	fmt.Println("-------------------")
	
	for _, pkgName := range packages {
		pkg := results[pkgName]
		shortName := getShortPackageName(pkgName)
		
		if pkg.BuildFailed {
			fmt.Printf("❌ %-40s [BUILD FAILED]\n", shortName)
			continue
		}

		status := "✅"
		if pkg.Failed > 0 {
			status = "❌"
		} else if pkg.Skipped > 0 && pkg.Passed == 0 {
			status = "⏭️"
		}

		fmt.Printf("%s %-40s Pass:%3d  Fail:%3d  Skip:%3d  (%.2fs)\n",
			status, shortName, pkg.Passed, pkg.Failed, pkg.Skipped, pkg.TotalElapsed)
		
		// Show failed test names
		if len(pkg.FailedTests) > 0 {
			for _, test := range pkg.FailedTests {
				fmt.Printf("     ↳ FAILED: %s\n", test)
			}
		}
	}

	// Print overall summary
	fmt.Println()
	fmt.Println("📈 OVERALL SUMMARY:")
	fmt.Println("-------------------")
	fmt.Printf("Total Packages:     %d\n", len(results))
	fmt.Printf("Total Tests Run:    %d\n", totalTests)
	fmt.Printf("✅ Tests Passed:    %d\n", passedTests)
	fmt.Printf("❌ Tests Failed:    %d\n", failedTests)
	fmt.Printf("⏭️  Tests Skipped:   %d\n", skippedTests)
	
	if failedTests > 0 {
		fmt.Printf("\n🔥 %d TEST(S) FAILED - See details above\n", failedTests)
	}

	// Look for CI verification tests
	ciVerificationCount := 0
	for _, pkg := range results {
		for _, test := range pkg.FailedTests {
			if strings.Contains(test, "CIVerification") {
				ciVerificationCount++
			}
		}
	}

	if ciVerificationCount > 0 {
		fmt.Printf("\n⚠️  CI VERIFICATION: Found %d intentional CI verification test failures\n", ciVerificationCount)
		fmt.Println("   These are expected to fail - see CI_VERIFICATION_TESTS.md")
	}

	// Exit with appropriate code
	if failedTests > 0 || len(compilationErrors) > 0 {
		os.Exit(1)
	}
}

func getShortPackageName(fullName string) string {
	parts := strings.Split(fullName, "/")
	if len(parts) > 3 {
		// Show last 3 parts for clarity
		return ".../" + strings.Join(parts[len(parts)-3:], "/")
	}
	return fullName
}