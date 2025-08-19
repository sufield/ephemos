# Contributor Documentation

Welcome to the Ephemos contributor documentation! This directory contains guides and resources for developers working on the Ephemos project.

## Quick Start

1. **[Local CI/CD Testing](./local-ci-testing.md)** - Run GitHub Actions workflows locally with act
2. **[Development Setup](./development-setup.md)** - Set up your development environment
3. **[Code Quality](./code-quality.md)** - Linting, testing, and security practices
4. **[Build System](./build-system.md)** - Bazel build system and migration guide

## Available Guides

### Development Environment
- **[Local CI/CD Testing](./local-ci-testing.md)** - Test workflows locally before pushing
- **[Development Setup](./development-setup.md)** - IDE setup, dependencies, and tools
- **[Docker Development](./docker-development.md)** - Containerized development workflow

### Code Quality & Testing
- **[Code Quality](./code-quality.md)** - Linting, formatting, and static analysis
- **[Testing Guide](./testing-guide.md)** - Unit, integration, and security testing
- **[Security Practices](./security-practices.md)** - Security testing and best practices

### Build & Deployment
- **[Build System](./build-system.md)** - Bazel build system overview
- **[CI/CD Pipeline](./ci-cd-pipeline.md)** - GitHub Actions workflows and automation
- **[Release Process](./release-process.md)** - How to create and manage releases

### Architecture & Design
- **[Architecture Overview](./architecture.md)** - System architecture and components
- **[API Documentation](./api-docs.md)** - Internal API reference
- **[Protocol Documentation](./protocols.md)** - SPIFFE/SPIRE integration details

### Contributing
- **[Contributing Guide](./contributing.md)** - How to contribute to the project
- **[Code Review](./code-review.md)** - Code review process and guidelines
- **[Issue Triage](./issue-triage.md)** - Bug reports and feature requests

## Quick Reference

### Essential Commands

```bash
# Run local CI/CD tests
./act -j test --pull=false

# Build with Bazel
./bazel.sh build

# Run security scans
./bazel.sh security-all

# Run all tests
./bazel.sh test

# Format code
./bazel.sh format
```

### Workflow Testing

```bash
# List all available workflows
./act -l

# Test specific workflow
./act -W .github/workflows/secrets-scan.yml

# Dry run for validation
./act --dryrun -W .github/workflows/bazel-ci.yml
```

### Development Cycle

1. **Setup**: Follow [Development Setup](./development-setup.md)
2. **Code**: Make your changes following [Code Quality](./code-quality.md) guidelines
3. **Test**: Use [Local CI/CD Testing](./local-ci-testing.md) to validate changes
4. **Review**: Submit PR following [Contributing Guide](./contributing.md)

## Project Structure

```
ephemos/
├── pkg/ephemos/          # Core library
├── cmd/                  # Command-line tools
├── examples/             # Example applications
├── scripts/              # Build and utility scripts (now Bazel targets)
├── .github/workflows/    # CI/CD workflows
├── docs/
│   ├── contributors/     # This directory
│   └── api/             # API documentation
└── tools/               # Development tools
```

## Key Technologies

- **Language**: Go 1.24+
- **Build System**: Bazel 7.4.1+
- **CI/CD**: GitHub Actions + Act (local testing)
- **Security**: SPIFFE/SPIRE identity framework
- **Testing**: Go testing + Fuzzing + Property-based testing
- **Quality**: Multiple linters, security scanners, and formatters

## Getting Help

### Internal Resources
- Check existing [Issues](../../issues) and [Discussions](../../discussions)
- Review relevant documentation in this directory
- Ask questions in project discussions

### External Resources
- [Go Documentation](https://golang.org/doc/)
- [Bazel Documentation](https://bazel.build/docs)
- [SPIFFE Documentation](https://spiffe.io/docs/)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)

### Support Channels
- **Bug reports**: Open an issue with the bug template
- **Feature requests**: Start a discussion or open an enhancement issue
- **Questions**: Use GitHub Discussions
- **Security issues**: See [Security Policy](../../security/policy)

## Contributing

We welcome contributions! Please:

1. Read the [Contributing Guide](./contributing.md)
2. Set up your [Development Environment](./development-setup.md)
3. Test changes with [Local CI/CD](./local-ci-testing.md)
4. Follow [Code Quality](./code-quality.md) standards
5. Submit a well-documented pull request

## Documentation Standards

When adding to this documentation:

- **Clear titles**: Use descriptive headings and subheadings
- **Code examples**: Include working code snippets with explanations
- **Cross-references**: Link to related documentation
- **Keep current**: Update docs when making related code changes
- **User-focused**: Write for developers using the documentation

## Recent Additions

- ✅ **Local CI/CD Testing**: Complete guide for testing workflows locally with act
- ✅ **Bazel Migration**: Documentation for the new Bazel build system
- ✅ **Security Testing**: Enhanced security scanning and validation processes
- 🔄 **Performance Testing**: Benchmarking and profiling workflows (in progress)

---

**Note**: This documentation is actively maintained. If you find outdated information or have suggestions for improvement, please open an issue or submit a pull request.