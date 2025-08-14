# Ephemos Makefile - Secure, Modular Build System
# Split from monolithic Makefile for better security and maintainability

.PHONY: all help

# Include modular makefiles
include Makefile.core
include Makefile.ci
include Makefile.security

# Demo targets (kept minimal for security)
.PHONY: demo demo-force demo-version

# Run complete demo with output capture
demo: proto build
	@echo "Running Ephemos demo..."
	@./scripts/demo/run-demo.sh

# Force reinstall SPIRE and run demo
demo-force: SPIRE_ARGS=--force
demo-force: demo

# Install specific SPIRE version and run demo
demo-version: SPIRE_ARGS=--version $(VERSION)
demo-version: demo

# GoReleaser targets
.PHONY: release-snapshot release-check release

# Test release build
release-snapshot:
	@echo "Building snapshot release..."
	goreleaser release --snapshot --clean --skip=publish

# Check release configuration
release-check:
	@echo "Checking release configuration..."
	goreleaser check

# Create release (requires proper setup)
release:
	@echo "Creating release..."
	goreleaser release --clean

# System setup targets
.PHONY: install-goreleaser check-requirements

# Install GoReleaser
install-goreleaser:
	@echo "Installing GoReleaser..."
	@if ! command -v goreleaser >/dev/null 2>&1; then \
		echo "Installing GoReleaser..."; \
		curl -sSL https://goreleaser.com/static/run | bash -s -- --version; \
	else \
		echo "✅ GoReleaser already installed"; \
		goreleaser --version; \
	fi

# Check system requirements
check-requirements:
	@echo "Checking system requirements for Ephemos..."
	@echo "==========================================="
	@./scripts/system/check-requirements.sh

# Global help
help:
	@echo "Ephemos Build System"
	@echo "===================="
	@echo ""
	@echo "Core targets (Makefile.core):"
	@echo "  make build            - Build library and CLI"
	@echo "  make proto            - Generate protobuf code"
	@echo "  make test             - Run tests"
	@echo "  make clean            - Clean build artifacts"
	@echo ""
	@echo "Security targets (Makefile.security):"
	@echo "  make security-all     - Complete security setup"
	@echo "  make security-scan    - Run security scans"
	@echo "  make audit-config     - Audit configuration files"
	@echo "  make sbom-generate    - Generate SBOM files (SPDX + CycloneDX)"
	@echo "  make sbom-validate    - Validate generated SBOM files"
	@echo "  make sbom-all         - Complete SBOM generation and validation"
	@echo ""
	@echo "CI/CD targets (Makefile.ci):"
	@echo "  make ci-all           - Run all CI checks"
	@echo "  make ci-test          - Run tests with coverage"
	@echo "  make ci-security      - Run security checks"
	@echo ""
	@echo "Demo targets:"
	@echo "  make demo             - Run complete demo"
	@echo "  make demo-force       - Force reinstall SPIRE and run demo"
	@echo ""
	@echo "Release targets:"
	@echo "  make release-snapshot - Build snapshot release"
	@echo "  make release-check    - Check release configuration"
	@echo "  make release          - Create production release"
	@echo ""
	@echo "Setup targets:"
	@echo "  make check-requirements - Check system requirements"
	@echo "  make install-goreleaser - Install GoReleaser"
	@echo ""
	@echo "Security first! Run 'make security-all' before development."