// Package main demonstrates how to create an HTTP client using Ephemos core primitives.
// This example shows the minimal approach for building HTTP clients with SPIFFE mTLS
// authentication without using framework middleware.
package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/spiffe/go-spiffe/v2/bundle/x509bundle"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
)

func main() {
	if err := runHTTPClient(); err != nil {
		log.Fatal(err)
	}
}

func runHTTPClient() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Step 1: Connect to SPIRE Workload API to get certificates and trust bundle
	fmt.Println("🔗 Connecting to SPIRE Workload API...")

	// Create X509Source to fetch certificates and trust bundles from SPIRE
	source, err := workloadapi.NewX509Source(ctx)
	if err != nil {
		return fmt.Errorf("failed to create X509Source: %w", err)
	}
	defer source.Close()

	// Step 2: Get our service's SVID (certificate) and trust bundle
	fmt.Println("📜 Fetching SPIFFE certificate and trust bundle...")

	svid, err := source.GetX509SVID()
	if err != nil {
		return fmt.Errorf("failed to get X509 SVID: %w", err)
	}

	// Parse the trust domain from our SPIFFE ID
	trustDomain := svid.ID.TrustDomain()
	fmt.Printf("🏛️  Service identity: %s (trust domain: %s)\n", svid.ID, trustDomain)

	// Step 3: Create bundle source for trust bundle management
	bundleSource := x509bundle.NewSet()
	bundle, err := source.GetX509BundleForTrustDomain(trustDomain)
	if err != nil {
		return fmt.Errorf("failed to get trust bundle: %w", err)
	}
	bundleSource.Add(bundle)

	// Step 4: Create authorizer for peer validation
	// In this example, we accept any service in our trust domain
	authorizer := tlsconfig.AuthorizeMemberOf(trustDomain)

	// For stricter authorization, you could use:
	// authorizer := tlsconfig.AuthorizeID(spiffeid.Must("spiffe://prod.company.com/specific-service"))
	// or
	// authorizer := tlsconfig.AuthorizeOneOf(
	//     spiffeid.Must("spiffe://prod.company.com/service-a"),
	//     spiffeid.Must("spiffe://prod.company.com/service-b"),
	// )

	// Step 5: Create TLS configuration using go-spiffe primitives
	fmt.Println("🔐 Creating mTLS configuration...")

	tlsConfig := tlsconfig.MTLSClientConfig(source, bundleSource, authorizer)
	tlsConfig.MinVersion = tls.VersionTLS13 // Enforce TLS 1.3

	// Step 6: Create HTTP client with SPIFFE mTLS transport
	fmt.Println("🌐 Creating HTTP client with SPIFFE authentication...")

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig:       tlsConfig,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
		Timeout: 30 * time.Second,
	}

	// Step 7: Make authenticated HTTP requests
	fmt.Println("📡 Making authenticated HTTP request...")

	// Example request - replace with your target service URL
	targetURL := "https://localhost:8080/api/health"

	// For demo purposes, we'll show how to make a request
	// In real usage, this would be your target service's HTTPS endpoint
	resp, err := httpClient.Get(targetURL)
	if err != nil {
		// This is expected to fail in the demo since we don't have a target service
		fmt.Printf("⚠️  Request failed (expected for demo): %v\n", err)
		fmt.Println("✅ HTTP client successfully created with SPIFFE mTLS!")
		return nil
	}
	defer resp.Body.Close()

	// If the request succeeds, read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	fmt.Printf("✅ Authenticated request successful!\n")
	fmt.Printf("Response status: %s\n", resp.Status)
	fmt.Printf("Response body: %s\n", string(body))

	return nil
}