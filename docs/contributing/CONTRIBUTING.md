# Contributing to Ephemos

Welcome to Ephemos! This guide will help you set up your development environment and start contributing.

## Quick Setup for Ubuntu 24

### 1. Check System Requirements
```bash
make check-requirements
```

This will verify if you have all necessary tools installed.

### 2. Install Prerequisites (Automated)
```bash
# Install everything (requires sudo)
make install-tools

# Or install only user-level tools (no sudo required)
make install-tools-user
```

### 3. Build and Test
```bash
# Get dependencies
make deps

# Build the project
make build

# Run the demo
make demo
```

## Manual Installation (Alternative)

If you prefer to install dependencies manually:

### System Dependencies
```bash
sudo apt update
sudo apt install -y wget curl git build-essential protobuf-compiler
```

### Go 1.24.5 (if not already installed)
```bash
wget https://go.dev/dl/go1.24.5.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.24.5.linux-amd64.tar.gz
echo 'export PATH=/usr/local/go/bin:$PATH' >> ~/.bashrc
source ~/.bashrc
```

### Go Protobuf Tools
```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### Optional Development Tools
```bash
# golangci-lint for linting
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.55.2
```

## Development Workflow

### Building
```bash
# Build CLI and library
make build

# Build example applications
make examples

# Generate protobuf code (if needed)
make proto
```

### Testing
```bash
# Run tests
make test

# Format code
make fmt

# Lint code
make lint
```

### Running the Demo
```bash
# Complete 5-minute demonstration
make demo
```

This will:
1. Install SPIRE server and agent
2. Start SPIRE services  
3. Register demo services
4. Run echo server and client
5. Demonstrate authentication success and failure

## Project Structure

```
ephemos/
├── cmd/
│   └── ephemos-cli/        # CLI tool for service registration
├── internal/
│   ├── core/
│   │   ├── domain/         # Business entities (no external deps)
│   │   ├── ports/          # Interface definitions  
│   │   └── services/       # Domain services
│   ├── adapters/
│   │   ├── primary/        # Inbound adapters (API, CLI)
│   │   └── secondary/      # Outbound adapters (SPIFFE, gRPC)
│   └── proto/              # Generated protobuf code
├── pkg/ephemos/            # Public API
├── examples/               # Example applications
├── configs/                # Configuration files
├── scripts/demo/           # Demo and setup scripts
└── docs/                   # Documentation
```

## Development Guidelines

### Architecture
- Follow **hexagonal architecture** principles
- Dependencies flow: adapters → ports → domain
- Domain core must have zero external dependencies
- Use dependency injection through interfaces

### Code Style
- Run `make fmt` before committing
- Follow Go conventions and best practices
- Add tests for new functionality
- Use meaningful variable and function names

### Commits
- Use conventional commit format
- Keep commits focused and atomic
- Include tests with new features
- Update documentation as needed

### Pull Requests
1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run `make test lint` to verify
5. Submit a pull request with clear description

## Troubleshooting

### Common Issues

**Go not found after installation:**
```bash
source ~/.bashrc
# or restart your terminal
```

**Permission errors with SPIRE:**
```bash
# Make sure you're running demo scripts with proper permissions
sudo systemctl status spire-server
sudo systemctl status spire-agent
```

**Build errors:**
```bash
# Clean and rebuild
make clean
go mod tidy
make build
```

**protoc not found:**
```bash
# The project works with pre-generated protobuf code
# But for development, install protoc:
sudo apt install protobuf-compiler
```

### Getting Help

- Check the [FAQ](FAQ.md) for common questions
- Review the [README](README.md) for project overview
- Look at existing issues in the repository
- Run `make help` for available commands

## Testing Your Changes

Before submitting changes:

1. **Build everything:**
   ```bash
   make clean
   make build examples
   ```

2. **Run tests:**
   ```bash
   make test
   ```

3. **Lint code:**
   ```bash
   make lint
   ```

4. **Test demo:**
   ```bash
   make demo
   ```

5. **Verify requirements:**
   ```bash
   make check-requirements
   ```

## Requirements Summary

- **OS**: Ubuntu 24 (optimized, but works on other Linux distros)
- **Go**: 1.24.5 or later
- **System tools**: git, make, curl, wget, build-essential
- **Protobuf**: protoc compiler + Go plugins
- **Optional**: golangci-lint for development

The `make install-tools` target handles all of this automatically!

## Questions?

Feel free to open an issue or reach out to the maintainers. We're here to help new contributors get started!