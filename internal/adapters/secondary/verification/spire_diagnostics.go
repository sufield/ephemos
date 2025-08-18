// Package verification provides SPIRE diagnostics implementations using SPIRE's
// built-in CLI tools and APIs rather than implementing diagnostic functionality from scratch.
package verification

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/workloadapi"

	"github.com/sufield/ephemos/internal/adapters/common"
	"github.com/sufield/ephemos/internal/core/domain"
	"github.com/sufield/ephemos/internal/core/ports"
)

// SpireDiagnosticsProvider implements diagnostics using SPIRE's built-in CLI tools
// and APIs, leveraging existing diagnostic capabilities rather than reimplementing them
type SpireDiagnosticsProvider struct {
	config *ports.DiagnosticsConfig
}

// NewSpireDiagnosticsProvider creates a new SPIRE diagnostics provider
func NewSpireDiagnosticsProvider(config *ports.DiagnosticsConfig) *SpireDiagnosticsProvider {
	if config == nil {
		config = &ports.DiagnosticsConfig{
			ServerSocketPath: "unix:///tmp/spire-server/private/api.sock",
			AgentSocketPath:  "unix:///tmp/spire-agent/public/api.sock",
			Timeout:          30 * time.Second,
		}
	}

	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &SpireDiagnosticsProvider{
		config: config,
	}
}

// GetServerDiagnostics retrieves SPIRE server diagnostic information
func (d *SpireDiagnosticsProvider) GetServerDiagnostics(ctx context.Context) (*ports.DiagnosticInfo, error) {
	info := &ports.DiagnosticInfo{
		Component:   "spire-server",
		CollectedAt: time.Now(),
		Details:     make(map[string]interface{}),
	}

	// Get server version using CLI
	version, err := d.getComponentVersion(ctx, "spire-server")
	if err != nil {
		info.Details["version_error"] = err.Error()
	} else {
		info.Version = version
	}

	// Get server configuration and status
	if err := d.getServerStatus(ctx, info); err != nil {
		info.Details["status_error"] = err.Error()
		info.Status = "error"
	} else {
		info.Status = "running"
	}

	// Get registration entries information
	entries, err := d.getRegistrationEntriesInfo(ctx)
	if err != nil {
		info.Details["entries_error"] = err.Error()
	} else {
		info.Entries = entries
	}

	// Get trust bundle information
	bundles, err := d.getTrustBundleInfo(ctx)
	if err != nil {
		info.Details["bundles_error"] = err.Error()
	} else {
		info.Bundles = bundles
	}

	// Get agents information
	agents, err := d.getAgentsInfo(ctx)
	if err != nil {
		info.Details["agents_error"] = err.Error()
	} else {
		info.Agents = agents
	}

	return info, nil
}

// GetAgentDiagnostics retrieves SPIRE agent diagnostic information
func (d *SpireDiagnosticsProvider) GetAgentDiagnostics(ctx context.Context) (*ports.DiagnosticInfo, error) {
	info := &ports.DiagnosticInfo{
		Component:   "spire-agent",
		CollectedAt: time.Now(),
		Details:     make(map[string]interface{}),
	}

	// Get agent version
	version, err := d.getComponentVersion(ctx, "spire-agent")
	if err != nil {
		info.Details["version_error"] = err.Error()
	} else {
		info.Version = version
	}

	// Get agent health status
	if err := d.getAgentStatus(ctx, info); err != nil {
		info.Details["status_error"] = err.Error()
		info.Status = "error"
	} else {
		info.Status = "running"
	}

	// Get workload identity information via Workload API
	if err := d.getWorkloadInfo(ctx, info); err != nil {
		info.Details["workload_error"] = err.Error()
	}

	return info, nil
}

// ListRegistrationEntries lists all registration entries using SPIRE CLI
func (d *SpireDiagnosticsProvider) ListRegistrationEntries(ctx context.Context) ([]*ports.RegistrationEntry, error) {
	cmd := exec.CommandContext(ctx, "spire-server", "entry", "show", "-output", "json")
	if d.config.ServerSocketPath != "" {
		cmd.Args = append(cmd.Args, "-socketPath", strings.TrimPrefix(d.config.ServerSocketPath, "unix://"))
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list registration entries: %w", err)
	}

	var cliOutput struct {
		Entries []struct {
			ID            string   `json:"id"`
			SPIFFEID      string   `json:"spiffe_id"`
			ParentID      string   `json:"parent_id"`
			Selectors     []string `json:"selectors"`
			TTL           int32    `json:"ttl"`
			FederatesWith []string `json:"federates_with"`
			DNSNames      []string `json:"dns_names"`
			Admin         bool     `json:"admin"`
			Downstream    bool     `json:"downstream"`
			CreatedAt     int64    `json:"created_at"`
		} `json:"entries"`
	}

	if err := json.Unmarshal(output, &cliOutput); err != nil {
		return nil, fmt.Errorf("failed to parse registration entries: %w", err)
	}

	var entries []*ports.RegistrationEntry
	for _, e := range cliOutput.Entries {
		spiffeID, err := spiffeid.FromString(e.SPIFFEID)
		if err != nil {
			continue // Skip invalid SPIFFE IDs
		}

		parentID, err := spiffeid.FromString(e.ParentID)
		if err != nil {
			continue // Skip invalid parent IDs
		}

		var federatesWith []domain.TrustDomain
		for _, td := range e.FederatesWith {
			if spiffeTD, err := spiffeid.TrustDomainFromString(td); err == nil {
				federatesWith = append(federatesWith, common.ToCoreTrustDomain(spiffeTD))
			}
		}

		entries = append(entries, &ports.RegistrationEntry{
			ID:            e.ID,
			SPIFFEID:      spiffeID,
			ParentID:      parentID,
			Selectors:     e.Selectors,
			TTL:           e.TTL,
			FederatesWith: federatesWith,
			DNSNames:      e.DNSNames,
			Admin:         e.Admin,
			Downstream:    e.Downstream,
			CreatedAt:     time.Unix(e.CreatedAt, 0),
		})
	}

	return entries, nil
}

// ShowTrustBundle displays trust bundle information using SPIRE CLI
func (d *SpireDiagnosticsProvider) ShowTrustBundle(ctx context.Context, trustDomain domain.TrustDomain) (*ports.TrustBundleInfo, error) {
	// Convert to spiffeid.TrustDomain for SPIRE CLI
	spiffeTD, err := spiffeid.TrustDomainFromString(trustDomain.String())
	if err != nil {
		return nil, fmt.Errorf("invalid trust domain %s: %w", trustDomain, err)
	}
	cmd := exec.CommandContext(ctx, "spire-server", "bundle", "show", "-output", "json")
	if d.config.ServerSocketPath != "" {
		cmd.Args = append(cmd.Args, "-socketPath", strings.TrimPrefix(d.config.ServerSocketPath, "unix://"))
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to show trust bundle: %w", err)
	}

	var bundles map[string]interface{}
	if err := json.Unmarshal(output, &bundles); err != nil {
		return nil, fmt.Errorf("failed to parse trust bundle: %w", err)
	}

	info := &ports.TrustBundleInfo{
		Federated: make(map[string]*ports.BundleInfo),
	}

	// Parse local bundle
	if localData, ok := bundles[spiffeTD.String()]; ok {
		if local, err := d.parseBundleData(spiffeTD, localData); err == nil {
			info.Local = local
		}
	}

	// Parse federated bundles
	for td, data := range bundles {
		if td != spiffeTD.String() {
			if domain, err := spiffeid.TrustDomainFromString(td); err == nil {
				if bundle, err := d.parseBundleData(domain, data); err == nil {
					info.Federated[td] = bundle
				}
			}
		}
	}

	return info, nil
}

// ListAgents lists all connected agents using SPIRE CLI
func (d *SpireDiagnosticsProvider) ListAgents(ctx context.Context) ([]*ports.Agent, error) {
	cmd := exec.CommandContext(ctx, "spire-server", "agent", "list", "-output", "json")
	if d.config.ServerSocketPath != "" {
		cmd.Args = append(cmd.Args, "-socketPath", strings.TrimPrefix(d.config.ServerSocketPath, "unix://"))
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}

	var cliOutput struct {
		Agents []struct {
			ID              string   `json:"id"`
			AttestationType string   `json:"attestation_type"`
			SerialNumber    string   `json:"serial_number"`
			ExpiresAt       int64    `json:"expires_at"`
			Banned          bool     `json:"banned"`
			CanReattest     bool     `json:"can_reattest"`
			Selectors       []string `json:"selectors"`
		} `json:"agents"`
	}

	if err := json.Unmarshal(output, &cliOutput); err != nil {
		return nil, fmt.Errorf("failed to parse agents list: %w", err)
	}

	var agents []*ports.Agent
	for _, a := range cliOutput.Agents {
		agentID, err := spiffeid.FromString(a.ID)
		if err != nil {
			continue // Skip invalid agent IDs
		}

		agents = append(agents, &ports.Agent{
			ID:              agentID,
			AttestationType: a.AttestationType,
			SerialNumber:    a.SerialNumber,
			ExpiresAt:       time.Unix(a.ExpiresAt, 0),
			Banned:          a.Banned,
			CanReattest:     a.CanReattest,
			Selectors:       a.Selectors,
		})
	}

	return agents, nil
}

// GetComponentVersion gets the version of a SPIRE component using CLI
func (d *SpireDiagnosticsProvider) GetComponentVersion(ctx context.Context, component string) (string, error) {
	return d.getComponentVersion(ctx, component)
}

// Helper methods

func (d *SpireDiagnosticsProvider) getComponentVersion(ctx context.Context, component string) (string, error) {
	cmd := exec.CommandContext(ctx, component, "-version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get %s version: %w", component, err)
	}

	// Parse version from output (typically "spire-server version X.Y.Z")
	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		parts := strings.Fields(lines[0])
		if len(parts) >= 3 && parts[1] == "version" {
			return parts[2], nil
		}
	}

	return strings.TrimSpace(string(output)), nil
}

func (d *SpireDiagnosticsProvider) getServerStatus(ctx context.Context, info *ports.DiagnosticInfo) error {
	// Use healthcheck command if available
	cmd := exec.CommandContext(ctx, "spire-server", "healthcheck")
	if d.config.ServerSocketPath != "" {
		cmd.Args = append(cmd.Args, "-socketPath", strings.TrimPrefix(d.config.ServerSocketPath, "unix://"))
	}

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("server healthcheck failed: %w", err)
	}

	info.Details["healthcheck"] = strings.TrimSpace(string(output))
	return nil
}

func (d *SpireDiagnosticsProvider) getAgentStatus(ctx context.Context, info *ports.DiagnosticInfo) error {
	// Use agent healthcheck command
	cmd := exec.CommandContext(ctx, "spire-agent", "healthcheck")
	if d.config.AgentSocketPath != "" {
		cmd.Args = append(cmd.Args, "-socketPath", strings.TrimPrefix(d.config.AgentSocketPath, "unix://"))
	}

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("agent healthcheck failed: %w", err)
	}

	info.Details["healthcheck"] = strings.TrimSpace(string(output))
	return nil
}

func (d *SpireDiagnosticsProvider) getRegistrationEntriesInfo(ctx context.Context) (*ports.RegistrationEntryInfo, error) {
	entries, err := d.ListRegistrationEntries(ctx)
	if err != nil {
		return nil, err
	}

	info := &ports.RegistrationEntryInfo{
		Total:      len(entries),
		BySelector: make(map[string]int),
	}

	now := time.Now()
	for _, entry := range entries {
		// Count recent entries (last 24 hours)
		if now.Sub(entry.CreatedAt) < 24*time.Hour {
			info.Recent++
		}

		// Count by selector type
		for _, selector := range entry.Selectors {
			parts := strings.SplitN(selector, ":", 2)
			if len(parts) > 0 {
				info.BySelector[parts[0]]++
			}
		}
	}

	return info, nil
}

func (d *SpireDiagnosticsProvider) getTrustBundleInfo(ctx context.Context) (*ports.TrustBundleInfo, error) {
	// This would require parsing trust bundle data from SPIRE CLI
	// For now, return basic structure
	return &ports.TrustBundleInfo{
		Federated: make(map[string]*ports.BundleInfo),
	}, nil
}

func (d *SpireDiagnosticsProvider) getAgentsInfo(ctx context.Context) (*ports.AgentInfo, error) {
	agents, err := d.ListAgents(ctx)
	if err != nil {
		return nil, err
	}

	info := &ports.AgentInfo{
		Total: len(agents),
	}

	now := time.Now()
	for _, agent := range agents {
		if agent.Banned {
			info.Banned++
		} else if agent.ExpiresAt.After(now) {
			info.Active++
		} else {
			info.Inactive++
		}
	}

	return info, nil
}

func (d *SpireDiagnosticsProvider) getWorkloadInfo(ctx context.Context, info *ports.DiagnosticInfo) error {
	// Use Workload API to get current workload identity information
	clientOptions := workloadapi.WithClientOptions(
		workloadapi.WithAddr(d.config.AgentSocketPath),
	)

	source, err := workloadapi.NewX509Source(ctx, clientOptions)
	if err != nil {
		return fmt.Errorf("failed to create workload API source: %w", err)
	}
	defer source.Close()

	svid, err := source.GetX509SVID()
	if err != nil {
		return fmt.Errorf("failed to get SVID: %w", err)
	}

	info.TrustDomain = common.ToCoreTrustDomain(svid.ID.TrustDomain())
	info.Details["workload_spiffe_id"] = svid.ID.String()
	info.Details["certificate_expires_at"] = svid.Certificates[0].NotAfter
	info.Details["certificate_serial"] = svid.Certificates[0].SerialNumber.String()

	return nil
}

func (d *SpireDiagnosticsProvider) parseBundleData(trustDomain spiffeid.TrustDomain, data interface{}) (*ports.BundleInfo, error) {
	// Parse bundle data from SPIRE CLI output
	// This is a simplified implementation - actual parsing would depend on CLI output format
	bundle := &ports.BundleInfo{
		TrustDomain:      common.ToCoreTrustDomain(trustDomain),
		CertificateCount: 1, // Default assumption
		LastUpdated:      time.Now(),
		ExpiresAt:        time.Now().Add(24 * time.Hour), // Default assumption
	}

	return bundle, nil
}
