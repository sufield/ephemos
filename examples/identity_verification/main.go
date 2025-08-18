// Example demonstrating SPIRE identity verification and diagnostics
// using Ephemos' built-in capabilities that leverage SPIRE's native mechanisms.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/spiffe/go-spiffe/v2/spiffeid"

	"github.com/sufield/ephemos/internal/adapters/secondary/verification"
	"github.com/sufield/ephemos/internal/core/domain"
	"github.com/sufield/ephemos/internal/core/ports"
)

func main() {
	fmt.Println("🔍 SPIRE Identity Verification and Diagnostics Example")
	fmt.Println("=====================================================")
	fmt.Println()

	ctx := context.Background()

	// Example 1: Identity Verification using go-spiffe/v2
	fmt.Println("1. Identity Verification using SPIRE's Workload API")
	fmt.Println("--------------------------------------------------")

	if err := demonstrateIdentityVerification(ctx); err != nil {
		log.Printf("Identity verification example failed: %v", err)
	}
	fmt.Println()

	// Example 2: Diagnostics using SPIRE CLI tools
	fmt.Println("2. SPIRE Diagnostics using built-in CLI tools")
	fmt.Println("--------------------------------------------")

	if err := demonstrateDiagnostics(ctx); err != nil {
		log.Printf("Diagnostics example failed: %v", err)
	}
	fmt.Println()

	// Example 3: Comprehensive monitoring setup
	fmt.Println("3. Comprehensive SPIRE Monitoring Setup")
	fmt.Println("--------------------------------------")

	if err := demonstrateComprehensiveMonitoring(ctx); err != nil {
		log.Printf("Comprehensive monitoring example failed: %v", err)
	}

	fmt.Println("\n✅ Examples completed successfully!")
}

func demonstrateIdentityVerification(ctx context.Context) error {
	// Configure identity verification to use SPIRE's Workload API
	trustDomain, err := domain.NewTrustDomain("example.org")
	if err != nil {
		return fmt.Errorf("invalid trust domain: %w", err)
	}
	
	config := &ports.VerificationConfig{
		WorkloadAPISocket: "unix:///tmp/spire-agent/public/api.sock",
		Timeout:           30 * time.Second,
		// Optionally configure allowed trust domains and SPIFFE IDs
		TrustDomain: trustDomain,
	}

	// Create identity verifier using go-spiffe/v2 library
	verifier, err := verification.NewSpireIdentityVerifier(config)
	if err != nil {
		return fmt.Errorf("failed to create identity verifier: %w", err)
	}
	defer verifier.Close()

	fmt.Println("   📋 Verifier created successfully using go-spiffe/v2")

	// Example: Get current workload identity
	fmt.Println("   🔍 Getting current workload identity...")
	identity, err := verifier.GetCurrentIdentity(ctx)
	if err != nil {
		fmt.Printf("   ⚠️  Failed to get current identity: %v\n", err)
		fmt.Println("   💡 This is expected if not running in a SPIRE workload environment")
	} else {
		fmt.Printf("   ✅ Current SPIFFE ID: %s\n", identity.SPIFFEID)
		fmt.Printf("   ✅ Trust Domain: %s\n", identity.SPIFFEID.TrustDomain())
		fmt.Printf("   ✅ Source: %s\n", identity.Source)
	}

	// Example: Verify a specific identity
	expectedID := spiffeid.RequireFromString("spiffe://example.org/my-service")
	fmt.Printf("   🔍 Verifying identity against: %s\n", expectedID)

	result, err := verifier.VerifyIdentity(ctx, expectedID)
	if err != nil {
		fmt.Printf("   ⚠️  Identity verification failed: %v\n", err)
		fmt.Println("   💡 This is expected if not running with the expected SPIFFE ID")
	} else {
		fmt.Printf("   %s Identity verification result: %t\n",
			getStatusEmoji(result.Valid), result.Valid)
		fmt.Printf("   📝 Message: %s\n", result.Message)
		if result.Valid {
			fmt.Printf("   🎫 Certificate expires: %s\n", result.NotAfter.Format(time.RFC3339))
		}
	}

	return nil
}

func demonstrateDiagnostics(ctx context.Context) error {
	// Configure diagnostics to use SPIRE's built-in CLI tools
	config := &ports.DiagnosticsConfig{
		ServerSocketPath: "unix:///tmp/spire-server/private/api.sock",
		AgentSocketPath:  "unix:///tmp/spire-agent/public/api.sock",
		Timeout:          30 * time.Second,
		UseServerAPI:     false, // Use CLI commands instead of API
	}

	// Create diagnostics provider using SPIRE CLI integration
	provider := verification.NewSpireDiagnosticsProvider(config)
	fmt.Println("   📋 Diagnostics provider created using SPIRE CLI integration")

	// Example: Get SPIRE server diagnostics
	fmt.Println("   🖥️  Getting SPIRE server diagnostics...")
	serverDiag, err := provider.GetServerDiagnostics(ctx)
	if err != nil {
		fmt.Printf("   ⚠️  Failed to get server diagnostics: %v\n", err)
		fmt.Println("   💡 This is expected if SPIRE server is not running or accessible")
	} else {
		fmt.Printf("   ✅ Server Status: %s\n", serverDiag.Status)
		fmt.Printf("   ✅ Server Version: %s\n", serverDiag.Version)
		if serverDiag.Entries != nil {
			fmt.Printf("   📊 Registration Entries: %d total, %d recent\n",
				serverDiag.Entries.Total, serverDiag.Entries.Recent)
		}
		if serverDiag.Agents != nil {
			fmt.Printf("   🤖 Agents: %d total, %d active, %d inactive\n",
				serverDiag.Agents.Total, serverDiag.Agents.Active, serverDiag.Agents.Inactive)
		}
	}

	// Example: Get SPIRE agent diagnostics
	fmt.Println("   🔧 Getting SPIRE agent diagnostics...")
	agentDiag, err := provider.GetAgentDiagnostics(ctx)
	if err != nil {
		fmt.Printf("   ⚠️  Failed to get agent diagnostics: %v\n", err)
		fmt.Println("   💡 This is expected if SPIRE agent is not running or accessible")
	} else {
		fmt.Printf("   ✅ Agent Status: %s\n", agentDiag.Status)
		fmt.Printf("   ✅ Agent Version: %s\n", agentDiag.Version)
		fmt.Printf("   ✅ Trust Domain: %s\n", agentDiag.TrustDomain)
	}

	// Example: List registration entries
	fmt.Println("   📋 Listing registration entries...")
	entries, err := provider.ListRegistrationEntries(ctx)
	if err != nil {
		fmt.Printf("   ⚠️  Failed to list registration entries: %v\n", err)
		fmt.Println("   💡 This is expected if SPIRE server is not accessible")
	} else {
		fmt.Printf("   ✅ Found %d registration entries\n", len(entries))
		for i, entry := range entries {
			if i >= 3 { // Show first 3 entries only
				fmt.Printf("   ... and %d more entries\n", len(entries)-3)
				break
			}
			fmt.Printf("   📝 Entry %d: %s -> %s\n", i+1, entry.ParentID, entry.SPIFFEID)
		}
	}

	return nil
}

func demonstrateComprehensiveMonitoring(ctx context.Context) error {
	fmt.Println("   🎯 Setting up comprehensive SPIRE monitoring...")

	// Create both verification and diagnostics components
	verificationConfig := &ports.VerificationConfig{
		WorkloadAPISocket: "unix:///tmp/spire-agent/public/api.sock",
		Timeout:           30 * time.Second,
	}

	diagnosticsConfig := &ports.DiagnosticsConfig{
		ServerSocketPath: "unix:///tmp/spire-server/private/api.sock",
		AgentSocketPath:  "unix:///tmp/spire-agent/public/api.sock",
		Timeout:          30 * time.Second,
	}

	verifier, err := verification.NewSpireIdentityVerifier(verificationConfig)
	if err != nil {
		return fmt.Errorf("failed to create identity verifier: %w", err)
	}
	defer verifier.Close()

	provider := verification.NewSpireDiagnosticsProvider(diagnosticsConfig)

	fmt.Println("   ✅ Both identity verification and diagnostics components ready")

	// Simulate a monitoring workflow
	fmt.Println("   🔄 Running monitoring workflow...")

	// Step 1: Check our own identity
	fmt.Println("      1️⃣  Checking workload identity...")
	identity, err := verifier.GetCurrentIdentity(ctx)
	if err != nil {
		fmt.Printf("      ⚠️  Workload identity unavailable: %v\n", err)
	} else {
		fmt.Printf("      ✅ Workload SPIFFE ID: %s\n", identity.SPIFFEID)
	}

	// Step 2: Check SPIRE infrastructure health
	fmt.Println("      2️⃣  Checking SPIRE infrastructure health...")
	serverDiag, err := provider.GetServerDiagnostics(ctx)
	if err != nil {
		fmt.Printf("      ⚠️  Server health check failed: %v\n", err)
	} else {
		fmt.Printf("      ✅ Server is %s (version %s)\n", serverDiag.Status, serverDiag.Version)
	}

	agentDiag, err := provider.GetAgentDiagnostics(ctx)
	if err != nil {
		fmt.Printf("      ⚠️  Agent health check failed: %v\n", err)
	} else {
		fmt.Printf("      ✅ Agent is %s (version %s)\n", agentDiag.Status, agentDiag.Version)
	}

	// Step 3: Validate trust relationships
	fmt.Println("      3️⃣  Validating trust relationships...")

	trustDomainForBundle, err := domain.NewTrustDomain("example.org")
	if err != nil {
		fmt.Printf("      ⚠️  Invalid trust domain: %v\n", err)
		return nil
	}
	bundleInfo, err := provider.ShowTrustBundle(ctx, trustDomainForBundle)
	if err != nil {
		fmt.Printf("      ⚠️  Trust bundle check failed: %v\n", err)
	} else {
		fmt.Printf("      ✅ Trust bundle available for domain: %s\n", trustDomainForBundle)
		if bundleInfo.Local != nil {
			fmt.Printf("      📋 Local bundle has %d certificates\n", bundleInfo.Local.CertificateCount)
		}
		if len(bundleInfo.Federated) > 0 {
			fmt.Printf("      🔗 %d federated trust domains configured\n", len(bundleInfo.Federated))
		}
	}

	fmt.Println("   ✅ Monitoring workflow completed")

	return nil
}

func getStatusEmoji(valid bool) string {
	if valid {
		return "✅"
	}
	return "❌"
}
