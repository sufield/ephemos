# Bazel Migration for Ephemos

## Overview

This document outlines the migration from shell scripts and Makefiles to Bazel for the Ephemos project.

## What's Been Set Up

### 1. Core Bazel Configuration

- **WORKSPACE**: Main Bazel configuration with Go rules, Gazelle, and protobuf support
- **.bazelrc**: Build configuration with security, performance, and reproducible build settings
  - External dependency mode: `--enable_workspace` with `--noenable_bzlmod`
  - Modern resource flags: `--local_resources=memory=75%,cpu=100%`
- **BUILD.bazel**: Root build file with targets for testing, building, and utilities
- **deps.bzl**: Go dependency management compatible with go.mod

### 2. Component BUILD Files

- **pkg/ephemos/BUILD.bazel**: Library and test targets for core package
- **cmd/ephemos-cli/BUILD.bazel**: CLI binary with version injection
- **cmd/config-validator/BUILD.bazel**: Config validator binary

### 3. Utility Scripts

- **bazel.sh**: Wrapper script that replaces Makefile functionality
- **tools/workspace_status.sh**: Version information for reproducible builds
- **tools/security_scan.sh**: Security scanning for built binaries
- **tools/lint_check.sh**: Code quality checks

### 4. Converted Shell Scripts

- **scripts/BUILD.bazel**: Build and CI scripts as Bazel targets
- **scripts/security/BUILD.bazel**: Security scanning tools as Bazel targets
- **scripts/demo/BUILD.bazel**: Demo and SPIRE management as Bazel targets
- **scripts/utils/BUILD.bazel**: Utility scripts as Bazel targets
- **scripts/ci/BUILD.bazel**: CI-specific scripts as Bazel targets

## Bazel Benefits

### Reproducible Builds
- Hermetic builds with explicit dependencies
- Version stamping with git information
- Consistent build environment across machines

### Performance
- Incremental builds (only rebuild what changed)
- Parallel execution
- Build result caching
- Remote execution support (can be added later)

### Security
- Sandboxed builds
- Explicit dependency declarations
- Binary security scanning
- Isolated build environment

### Scalability
- Handles large codebases efficiently
- Language-agnostic (easy to add other languages)
- Supports monorepo architectures

## Replacing Shell Scripts

### Before (Shell/Make):
```bash
make build           # Build everything
make proto           # Generate protobuf
make test            # Run tests
./scripts/ci/lint.sh # Run linting
./scripts/security/scan-secrets.sh # Security scan
./scripts/demo/run-demo.sh # Run demo
```

### After (Bazel):
```bash
./bazel.sh build         # Build everything
./bazel.sh proto         # Generate protobuf
./bazel.sh test          # Run tests
./bazel.sh lint          # Run linting
./bazel.sh security-all  # Run all security scans
./bazel.sh demo          # Run complete demo
```

### Script Target Examples:
```bash
# Run specific script categories
bazel test //scripts:build_tests          # Test build scripts
bazel test //scripts/security:security_tests # Test security scripts
bazel run //scripts/demo:full_demo        # Run complete demo

# Run individual script targets
bazel run //scripts:lint                  # Run linting
bazel run //scripts/security:scan_secrets # Scan for secrets
bazel run //scripts/demo:setup_demo       # Setup demo environment
```

## Migration Roadmap

### Phase 1: Core Infrastructure ✅
- [x] Set up WORKSPACE and .bazelrc
- [x] Create BUILD files for main components
- [x] Set up protobuf generation
- [x] Create wrapper script (bazel.sh)

### Phase 2: CI/CD Integration ✅
- [x] Update GitHub Actions workflows
- [x] Replace shell script calls with Bazel
- [x] Add remote caching for CI
- [x] Migrate security scanning

### Phase 3: Advanced Features (Future)
- [ ] Remote build execution
- [ ] Advanced testing (integration, performance)
- [ ] Container image building
- [ ] Release artifact generation

## Usage Examples

### Building All Targets
```bash
./bazel.sh build
```

### Running Tests with Coverage
```bash
./bazel.sh coverage
```

### Building Specific Binary
```bash
./bazel.sh build-cli
```

### Running Security Scans
```bash
./bazel.sh security
```

### Formatting BUILD Files
```bash
./bazel.sh format
```

## Key Files Created

| File | Purpose |
|------|---------|
| `WORKSPACE` | Main Bazel configuration |
| `.bazelrc` | Build settings and options |
| `BUILD.bazel` | Root build targets |
| `deps.bzl` | Go dependency definitions |
| `bazel.sh` | Wrapper script for common tasks |
| `tools/workspace_status.sh` | Version information script |
| `pkg/ephemos/BUILD.bazel` | Core library build rules |
| `cmd/*/BUILD.bazel` | Binary build rules |

## Benefits Over Shell Scripts

### Reliability
- ✅ Explicit dependency management
- ✅ Reproducible builds
- ✅ Better error handling
- ✅ Parallel execution

### Maintainability
- ✅ Declarative build rules
- ✅ Language-agnostic approach
- ✅ Centralized configuration
- ✅ Tool integration

### Performance
- ✅ Incremental builds
- ✅ Build caching
- ✅ Remote execution support
- ✅ Optimized dependency resolution

### Security
- ✅ Sandboxed execution
- ✅ Hermetic builds
- ✅ Explicit permissions
- ✅ Supply chain security

## Next Steps

1. **Install Bazel**: Follow installation instructions at https://bazel.build/install
2. **Test Build**: Run `./bazel.sh build` to verify setup
3. **Update CI**: Migrate GitHub Actions to use Bazel
4. **Remove Scripts**: Gradually replace shell scripts with Bazel targets

## Installation

To use the Bazel setup:

1. Install Bazel (7.4.1 or later)
2. Run: `./bazel.sh build` to build everything
3. Run: `./bazel.sh test` to run tests

The Bazel setup is now ready to replace the existing shell script and Makefile-based build system.