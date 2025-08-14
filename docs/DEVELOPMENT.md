# Development Guide

This guide helps you set up a local development environment for Ephemos.

## Quick Setup

For the fastest setup experience, run our automated installer:

```bash
# Install all development dependencies
make setup
```

This will install:
- Go protobuf generation tools
- Development and security tools
- Verify everything works

## Manual Setup

If you prefer to install dependencies manually:

### 1. Prerequisites

**Go 1.21 or later**
```bash
# Check your Go version
go version
```

Install from: https://golang.org/dl/

### 2. Protocol Buffers

**Ubuntu/Debian:**
```bash
sudo apt-get update
sudo apt-get install protobuf-compiler
```

**CentOS/RHEL:**
```bash
sudo yum install protobuf-compiler
# or on newer systems:
sudo dnf install protobuf-compiler
```

**macOS:**
```bash
brew install protobuf
```

**Windows:**
```bash
```

### 3. Go Protobuf Tools

```bash
```

### 4. Optional Development Tools

```bash
# Linting
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# Security tools
go install github.com/securego/gosec/v2/cmd/gosec@latest
go install golang.org/x/vuln/cmd/govulncheck@latest
```

## Verify Setup

Check if all dependencies are installed:

```bash
make check-deps
```

## Building

Once dependencies are installed:

```bash
# Build main CLI tools
make build

# Build example applications  
make examples

# Run tests
make test

# Run linting
make lint
```

## Common Issues


**Problem:** The build fails with protobuf generation errors.

**Solutions:** 
1. **Automatic (Recommended)**: Just run any build command - dependencies install automatically
   ```bash
   make build    # Auto-installs missing deps
   ```

2. **Manual Setup**: Run the comprehensive installer
   ```bash
   make setup    # Installs all development tools
   ```

   ```bash
   sudo apt-get install protobuf-compiler  # Ubuntu/Debian
   brew install protobuf                    # macOS
   ```

4. **CI Environments**: Use CI-friendly build targets
   ```bash
   ```


**Problem:** Protobuf generation fails with missing Go tools.

**Solution:**
```bash

# Add Go bin to PATH (add to ~/.bashrc or ~/.zshrc)
export PATH="$PATH:$(go env GOPATH)/bin"
```

### Build Cache Issues

**Problem:** Builds fail with module or cache errors.

**Solution:**
```bash
go clean -cache
go clean -testcache
go clean -modcache
make build
```

## Development Workflow

1. **First time setup:**
   ```bash
   git clone <repository>
   cd ephemos
   make setup
   ```

2. **Regular development:**
   ```bash
   # Make your changes
   make build        # Build and test
   make test         # Run tests  
   make lint         # Check code quality
   ```

3. **Before committing:**
   ```bash
   ./scripts/security-scan.sh    # Security checks
   make test                     # Full test suite
   make lint                     # Code quality
   ```

## Project Structure

```
ephemos/
├── cmd/                    # CLI applications
├── pkg/ephemos/           # Main library (public API)
├── internal/              # Internal packages
├── examples/              # Example applications
├── scripts/               # Build and utility scripts
├── .github/               # CI/CD workflows
└── docs/                  # Documentation
```

## Make Targets

| Target | Description |
|--------|-------------|
| `make setup` | Install all development dependencies |
| `make check-deps` | Check if dependencies are installed |
| `make build` | Build CLI tools |
| `make examples` | Build example applications |
| `make test` | Run test suite |
| `make lint` | Run code linting |
| `make clean` | Clean build artifacts |
| `make security` | Run security checks |

## Security Development

Ephemos includes comprehensive security tooling:

- **CodeQL**: Static analysis (runs in CI)
- **gosec**: Go security analyzer
- **govulncheck**: Vulnerability scanner
- **gitleaks**: Secret detection

Run all security checks:
```bash
./scripts/security-scan.sh
```

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Run `make setup` to install dependencies
4. Make your changes
5. Run tests and security checks: `make test && ./scripts/security-scan.sh`
6. Commit your changes: `git commit -m 'Add amazing feature'`
7. Push to the branch: `git push origin feature/amazing-feature`
8. Open a Pull Request

## Getting Help

- 📖 **Documentation**: Check the `docs/` directory
- 🐛 **Issues**: Open an issue on GitHub
- 💬 **Discussions**: Use GitHub Discussions for questions
- 🔒 **Security**: See [SECURITY.md](../SECURITY.md) for security issues

Happy coding! 🚀