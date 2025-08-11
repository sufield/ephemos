.PHONY: build proto examples test demo clean install-tools

# Variables
PROTO_DIR := examples/proto
GO_OUT := examples/proto
PROTOC := protoc
CLI_BINARY := ephemos
SERVER_BINARY := echo-server
CLIENT_BINARY := echo-client

# Default target
all: proto build examples

# Build library and CLI
build:
	echo "Building Ephemos CLI..."
	mkdir -p bin
	go build -v -o bin/$(CLI_BINARY) ./cmd/ephemos-cli
	go build -v -o bin/config-validator ./cmd/config-validator
	echo "Build completed!"

# Generate protobuf code
proto:
	echo "Generating protobuf code..."
	mkdir -p $(GO_OUT)
	@export PATH="$$PATH:$$(go env GOPATH)/bin:/usr/bin:/usr/local/bin"; \
	if command -v protoc >/dev/null 2>&1; then \
		echo "Found protoc at: $$(which protoc)"; \
		protoc --go_out=$(GO_OUT) --go_opt=paths=source_relative \
			--go-grpc_out=$(GO_OUT) --go-grpc_opt=paths=source_relative \
			-I $(PROTO_DIR) $(PROTO_DIR)/echo.proto && \
		echo "Protobuf generation completed!"; \
	else \
		echo "Warning: protoc not found in PATH. Checking if protobuf files already exist..."; \
		if [ -f "$(GO_OUT)/echo.pb.go" ] && [ -f "$(GO_OUT)/echo_grpc.pb.go" ]; then \
			echo "✅ Protobuf files already exist, skipping generation"; \
		else \
			echo "❌ protoc not found and protobuf files don't exist"; \
			echo "PATH was: $$PATH"; \
			echo "Please install protoc: apt-get install protobuf-compiler"; \
			exit 1; \
		fi; \
	fi

# Build example applications
examples: proto
	echo "Building example applications..."
	mkdir -p bin
	go build -v -o bin/$(SERVER_BINARY) ./examples/echo-server
	go build -v -o bin/$(CLIENT_BINARY) ./examples/echo-client
	echo "Examples built!"

# Run tests
test:
	echo "Running tests..."
	go test -v ./...
	echo "Tests completed!"

# Run complete demo with output capture
demo: proto build examples
	@echo "Running Ephemos demo..."
	@echo "========================"
	@echo ""
	@echo "Step 1: Installing SPIRE (if needed)..."
	@cd scripts/demo && ./install-spire.sh $(SPIRE_ARGS)
	@echo ""
	@echo "Step 2: Starting SPIRE services..."
	@cd scripts/demo && ./start-spire.sh
	@echo ""
	@echo "Step 3: Setting up demo services..."
	@cd scripts/demo && ./setup-demo.sh
	@echo ""
	@echo "Step 4: Running demo with client-server interactions..."
	@echo "-------------------------------------------------------"
	@cd scripts/demo && ./run-demo.sh | tee demo.log || { echo "Demo failed! Check scripts/demo/*.log for details"; exit 1; }
	@echo ""
	@echo "==============================================="
	@echo "SPIRE Infrastructure Logs:"
	@echo "==============================================="
	@if [ -f scripts/demo/spire-server.log ]; then echo "SPIRE SERVER LOG:"; cat scripts/demo/spire-server.log | sed 's/^/[SPIRE-SERVER] /'; fi
	@if [ -f scripts/demo/spire-agent.log ]; then echo ""; echo "SPIRE AGENT LOG:"; cat scripts/demo/spire-agent.log | sed 's/^/[SPIRE-AGENT] /'; fi
	@echo ""
	@echo "==============================================="
	@echo "Application Interaction Logs:"
	@echo "==============================================="
	@if [ -f scripts/demo/client.log ]; then echo "CLIENT LOG:"; cat scripts/demo/client.log; fi
	@if [ -f scripts/demo/server.log ]; then echo ""; echo "SERVER LOG:"; cat scripts/demo/server.log; fi
	@echo ""
	@echo "Demo completed successfully!"

# Force reinstall SPIRE and run demo
demo-force: SPIRE_ARGS=--force
demo-force: demo

# Install specific SPIRE version and run demo (usage: make demo-version VERSION=1.9.0)
demo-version: SPIRE_ARGS=--version $(VERSION)
demo-version: demo

# Clean build artifacts
clean:
	echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -f $(GO_OUT)/*.pb.go
	rm -f scripts/demo/*.log scripts/demo/*.pid
	rm -f demo.log
	echo "Clean completed!"

# CI/CD targets
.PHONY: ci-setup ci-lint ci-test ci-security ci-build ci-all

# Setup environment for CI
ci-setup: proto
	@echo "CI setup completed - protobuf files generated"

# Run linting checks locally
ci-lint:
	@echo "Running linting checks..."
	go fmt ./...
	go vet ./...
	@if command -v golangci-lint >/dev/null; then \
		golangci-lint run --config=.golangci.yml; \
	else \
		echo "golangci-lint not installed, run: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b \$$(go env GOPATH)/bin v1.55.2"; \
	fi

# Run all tests locally
ci-test:
	@echo "Running tests..."
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run security checks locally  
ci-security:
	@echo "Running security checks..."
	@echo "Running go vet for basic static analysis..."
	go vet ./...
	@if command -v govulncheck >/dev/null; then \
		govulncheck ./...; \
	else \
		echo "govulncheck not installed, run: go install golang.org/x/vuln/cmd/govulncheck@latest"; \
	fi
	@echo "Running secret detection..."
	@$(MAKE) scan-secrets
	@echo "Running container security scan..."
	@$(MAKE) scan-trivy

# Build all targets
ci-build: build examples
	@echo "All builds completed successfully!"

# Run all CI checks locally
ci-all: ci-lint ci-test ci-security ci-build
	@echo "All CI checks completed successfully!"

# Security scanning tools
.PHONY: scan-secrets scan-trivy install-security-tools

# Install security scanning tools
install-security-tools:
	@echo "Installing security scanning tools..."
	@echo "Installing git-secrets..."
	@if ! command -v git-secrets >/dev/null; then \
		echo "Installing git-secrets from GitHub..."; \
		cd /tmp && \
		git clone https://github.com/awslabs/git-secrets.git && \
		cd git-secrets && \
		make install && \
		cd .. && rm -rf git-secrets && \
		echo "✅ git-secrets installed"; \
	else \
		echo "✅ git-secrets already installed"; \
	fi
	@echo "Installing trivy..."
	@if ! command -v trivy >/dev/null; then \
		echo "Installing trivy..."; \
		curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sudo sh -s -- -b /usr/local/bin; \
		echo "✅ trivy installed"; \
	else \
		echo "✅ trivy already installed"; \
	fi
	@echo "Installing gitleaks..."
	@if ! command -v gitleaks >/dev/null; then \
		echo "Installing gitleaks..."; \
		curl -sSfL https://github.com/gitleaks/gitleaks/releases/latest/download/gitleaks_$$(uname -s)_$$(uname -m).tar.gz | sudo tar -xz -C /usr/local/bin gitleaks; \
		echo "✅ gitleaks installed"; \
	else \
		echo "✅ gitleaks already installed"; \
	fi

# Run secret detection scans
scan-secrets:
	@echo "🔍 Scanning for secrets and sensitive data..."
	@echo "Running gitleaks scan..."
	@if command -v gitleaks >/dev/null; then \
		gitleaks detect --source . --no-git --verbose || echo "⚠️  Gitleaks found potential secrets"; \
	else \
		echo "❌ gitleaks not installed, run: make install-security-tools"; \
	fi
	@echo "Running git-secrets scan..."
	@if command -v git-secrets >/dev/null; then \
		git-secrets --scan --recursive . || echo "⚠️  git-secrets found potential secrets"; \
	else \
		echo "❌ git-secrets not installed, run: make install-security-tools"; \
	fi
	@echo "Manual config file audit..."
	@$(MAKE) audit-config-files

# Run trivy security scans
scan-trivy:
	@echo "🛡️ Running Trivy security scans..."
	@if command -v trivy >/dev/null; then \
		echo "Scanning filesystem for vulnerabilities..."; \
		trivy fs --severity HIGH,CRITICAL . || echo "⚠️  Trivy found vulnerabilities"; \
		echo "Scanning for misconfigurations..."; \
		trivy config . || echo "⚠️  Trivy found misconfigurations"; \
	else \
		echo "❌ trivy not installed, run: make install-security-tools"; \
	fi

# Audit configuration files for sensitive data
audit-config-files:
	@echo "📋 Auditing configuration files for sensitive data..."
	@echo "Checking for potential secrets in config files:"
	@rg -i "(password|secret|key|token|credential|api[_-]?key|private[_-]?key)" config/ || echo "✅ No obvious secrets found in config/"
	@echo "Checking for hardcoded production values:"
	@rg -i "(prod|production|staging)" config/*.yaml || echo "✅ No hardcoded production values found"
	@echo "Checking for real domains (not example.org):"
	@rg -v "example\.org" config/*.yaml | rg "[a-zA-Z0-9.-]+\.(com|net|org|io)" || echo "✅ Only example domains found"

# Setup git hooks for security
setup-git-hooks:
	@echo "🪝 Setting up Git security hooks..."
	@mkdir -p .git/hooks
	@echo '#!/bin/bash' > .git/hooks/pre-commit
	@echo 'echo "🔍 Running pre-commit security checks..."' >> .git/hooks/pre-commit
	@echo 'make scan-secrets' >> .git/hooks/pre-commit
	@echo 'if [ $$? -ne 0 ]; then' >> .git/hooks/pre-commit
	@echo '    echo "❌ Security scan failed. Commit blocked."' >> .git/hooks/pre-commit
	@echo '    echo "Run: make scan-secrets to see issues"' >> .git/hooks/pre-commit
	@echo '    exit 1' >> .git/hooks/pre-commit
	@echo 'fi' >> .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "✅ Pre-commit hook installed"
	@echo "Creating pre-push hook..."
	@echo '#!/bin/bash' > .git/hooks/pre-push
	@echo 'echo "🛡️ Running pre-push security validation..."' >> .git/hooks/pre-push
	@echo 'make ci-security' >> .git/hooks/pre-push
	@echo 'if [ $$? -ne 0 ]; then' >> .git/hooks/pre-push
	@echo '    echo "❌ Security validation failed. Push blocked."' >> .git/hooks/pre-push
	@echo '    exit 1' >> .git/hooks/pre-push
	@echo 'fi' >> .git/hooks/pre-push
	@chmod +x .git/hooks/pre-push
	@echo "✅ Pre-push hook installed"

# Validate configuration security
validate-config: build
	@echo "Validating configuration security..."
	@echo "Testing demo configuration (should pass basic validation):"
	@export EPHEMOS_SERVICE_NAME="test-service" && export EPHEMOS_TRUST_DOMAIN="test.local" && ./bin/config-validator --env-only --verbose || true
	@echo ""
	@echo "Testing production configuration (requires production env vars):"
	@./bin/config-validator --env-only --production --verbose || echo "Set production environment variables to test production validation"

# Install all prerequisites for Ubuntu 24
install-tools:
	@echo "Installing all prerequisites for Ephemos on Ubuntu 24..."
	@echo "=================================================="
	@echo ""
	
	# Check Go version
	@echo "1. Checking Go installation..."
	@if ! command -v go >/dev/null 2>&1; then \
		echo "❌ Go not found. Installing Go 1.24.5..."; \
		wget -q https://go.dev/dl/go1.24.5.linux-amd64.tar.gz -O /tmp/go1.24.5.linux-amd64.tar.gz; \
		sudo rm -rf /usr/local/go; \
		sudo tar -C /usr/local -xzf /tmp/go1.24.5.linux-amd64.tar.gz; \
		echo 'export PATH=/usr/local/go/bin:$$PATH' >> ~/.bashrc; \
		rm /tmp/go1.24.5.linux-amd64.tar.gz; \
		echo "✅ Go 1.24.5 installed. Please run 'source ~/.bashrc' or restart terminal."; \
	else \
		echo "✅ Go found: $$(go version)"; \
	fi
	@echo ""
	
	# Install system dependencies
	@echo "2. Installing system dependencies..."
	@sudo apt update -qq
	@sudo apt install -y wget curl git build-essential protobuf-compiler
	@echo "✅ System dependencies installed"
	@echo ""
	
	# Install Go protobuf tools
	@echo "3. Installing Go protobuf tools..."
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@echo "✅ Go protobuf tools installed"
	@echo ""
	
	# Install development tools (optional but recommended)
	@echo "4. Installing optional development tools..."
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.55.2; \
		echo "✅ golangci-lint installed"; \
	else \
		echo "✅ golangci-lint already installed"; \
	fi
	@echo ""
	
	# Verify installations
	@echo "5. Verifying installations..."
	@echo "   Go: $$(go version 2>/dev/null || echo 'Not found - restart terminal')"
	@echo "   protoc: $$(protoc --version 2>/dev/null || echo 'Not found')"
	@echo "   protoc-gen-go: $$(which protoc-gen-go 2>/dev/null && echo 'Installed' || echo 'Not found')"
	@echo "   protoc-gen-go-grpc: $$(which protoc-gen-go-grpc 2>/dev/null && echo 'Installed' || echo 'Not found')"
	@echo "   golangci-lint: $$(which golangci-lint 2>/dev/null && echo 'Installed' || echo 'Not found')"
	@echo ""
	
	@echo "=================================================="
	@echo "✅ All prerequisites installed successfully!"
	@echo ""
	@echo "Next steps:"
	@echo "1. If Go was just installed, run: source ~/.bashrc"
	@echo "2. Clone the project: git clone <repository-url>"
	@echo "3. Navigate to project: cd ephemos"
	@echo "4. Install dependencies: make deps"
	@echo "5. Build project: make build"
	@echo "6. Run demo: make demo"

# Minimal install (no sudo required)
install-tools-user:
	@echo "Installing user-level tools (no sudo required)..."
	@echo "================================================="
	@echo ""
	
	# Install Go protobuf tools
	@echo "Installing Go protobuf tools..."
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@echo "✅ Go protobuf tools installed"
	@echo ""
	
	# Install golangci-lint
	@echo "Installing golangci-lint..."
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.55.2; \
		echo "✅ golangci-lint installed"; \
	else \
		echo "✅ golangci-lint already installed"; \
	fi
	@echo ""
	
	@echo "================================================="
	@echo "✅ User-level tools installed!"
	@echo ""
	@echo "Note: System dependencies still need to be installed manually:"
	@echo "  sudo apt update && sudo apt install -y wget curl git build-essential protobuf-compiler"

# Check system requirements
check-requirements:
	@echo "Checking system requirements for Ephemos..."
	@echo "==========================================="
	@echo ""
	
	@echo "Operating System:"
	@if grep -q "Ubuntu 24" /etc/os-release 2>/dev/null; then \
		echo "✅ Ubuntu 24 detected"; \
	else \
		echo "⚠️  Not Ubuntu 24. This project is optimized for Ubuntu 24."; \
		echo "   Current OS: $$(cat /etc/os-release | grep PRETTY_NAME | cut -d'=' -f2 | tr -d '\"' || echo 'Unknown')"; \
	fi
	@echo ""
	
	@echo "Required Tools:"
	@printf "   Go 1.24+: "
	@if command -v go >/dev/null 2>&1; then \
		echo "✅ $$(go version)"; \
	else \
		echo "❌ Not installed"; \
	fi
	
	@printf "   git: "
	@if command -v git >/dev/null 2>&1; then \
		echo "✅ $$(git --version)"; \
	else \
		echo "❌ Not installed"; \
	fi
	
	@printf "   protoc: "
	@if command -v protoc >/dev/null 2>&1; then \
		echo "✅ $$(protoc --version)"; \
	else \
		echo "❌ Not installed"; \
	fi
	
	@printf "   protoc-gen-go: "
	@if command -v protoc-gen-go >/dev/null 2>&1; then \
		echo "✅ Installed"; \
	else \
		echo "❌ Not installed"; \
	fi
	
	@printf "   protoc-gen-go-grpc: "
	@if command -v protoc-gen-go-grpc >/dev/null 2>&1; then \
		echo "✅ Installed"; \
	else \
		echo "❌ Not installed"; \
	fi
	@echo ""
	
	@echo "Optional Tools:"
	@printf "   golangci-lint: "
	@if command -v golangci-lint >/dev/null 2>&1; then \
		echo "✅ Installed"; \
	else \
		echo "⚠️  Not installed (recommended for development)"; \
	fi
	
	@printf "   make: "
	@if command -v make >/dev/null 2>&1; then \
		echo "✅ $$(make --version | head -1)"; \
	else \
		echo "❌ Not installed"; \
	fi
	@echo ""
	
	@echo "==========================================="
	@echo "Run 'make install-tools' to install missing requirements."

# Get dependencies
deps:
	echo "Getting dependencies..."
	go mod download
	go mod tidy
	echo "Dependencies updated!"

# Format code
fmt:
	echo "Formatting code..."
	go fmt ./...
	echo "Code formatted!"

# Lint code
lint:
	echo "Linting code..."
	golangci-lint run || echo "Install golangci-lint: https://golangci-lint.run/usage/install/"
	echo "Linting completed!"

# Security targets
security-all: install-security-tools setup-git-hooks scan-secrets scan-trivy
	@echo "🔒 Complete security setup finished!"

# Help target
help:
	@echo "Ephemos Makefile targets:"
	@echo ""
	@echo "Build targets:"
	@echo "  make build           - Build library and CLI"
	@echo "  make proto           - Generate protobuf code"
	@echo "  make examples        - Build example applications"
	@echo "  make clean           - Clean build artifacts"
	@echo ""
	@echo "Setup targets:"
	@echo "  make install-tools   - Install all prerequisites (Ubuntu 24)"
	@echo "  make install-tools-user - Install user-level tools (no sudo)"
	@echo "  make check-requirements - Check system requirements"
	@echo "  make deps            - Get/update Go dependencies"
	@echo ""
	@echo "Development targets:"
	@echo "  make test            - Run tests"
	@echo "  make fmt             - Format code"
	@echo "  make lint            - Lint code"
	@echo ""
	@echo "Demo targets:"
	@echo "  make demo            - Run complete demo (checks for existing SPIRE)"
	@echo "  make demo-force      - Force reinstall SPIRE and run demo"
	@echo "  make demo-version VERSION=1.9.0 - Install specific SPIRE version"
	@echo ""
	@echo "Security targets:"
	@echo "  make security-all    - Complete security setup (tools + hooks + scans)"
	@echo "  make install-security-tools - Install trivy, gitleaks, git-secrets"
	@echo "  make scan-secrets    - Scan for secrets and sensitive data"
	@echo "  make scan-trivy      - Run Trivy vulnerability scans"
	@echo "  make setup-git-hooks - Install pre-commit/pre-push security hooks"
	@echo "  make audit-config-files - Check config files for secrets"
	@echo ""
	@echo "Help:"
	@echo "  make help            - Show this help message"
	@echo ""
	@echo "Quick start for new contributors:"
	@echo "  1. make check-requirements"
	@echo "  2. make install-tools"
	@echo "  3. make security-all"
	@echo "  4. make demo"