# Build System: Bazel Migration Guide

This document covers the Ephemos build system migration from shell scripts and Makefiles to Bazel, providing comprehensive guidance for developers.

## Table of Contents

- [Overview](#overview)
- [Migration Benefits](#migration-benefits)
- [Quick Start](#quick-start)
- [Build Targets](#build-targets)
- [Converted Scripts](#converted-scripts)
- [Configuration](#configuration)
- [Usage Examples](#usage-examples)
- [Troubleshooting](#troubleshooting)
- [Migration Details](#migration-details)

## Overview

Ephemos has migrated from a shell script and Makefile-based build system to [Bazel](https://bazel.build/), Google's open-source build and test tool. This migration provides:

- **Reproducible builds**: Hermetic builds with explicit dependencies
- **Incremental compilation**: Only rebuild what changed
- **Parallel execution**: Faster build times through parallelization
- **Language agnostic**: Easy integration of multiple languages
- **Remote caching**: Shared build artifacts across teams

## Migration Benefits

### Before (Shell/Make)
```bash
make build           # Build everything
make proto           # Generate protobuf
make test            # Run tests
./scripts/ci/lint.sh # Run linting
./scripts/security/scan-secrets.sh # Security scan
./scripts/demo/run-demo.sh # Run demo
```

### After (Bazel)
```bash
./bazel.sh build         # Build everything
./bazel.sh proto         # Generate protobuf  
./bazel.sh test          # Run tests
./bazel.sh lint          # Run linting
./bazel.sh security-all  # Run all security scans
./bazel.sh demo          # Run complete demo
```

### Key Improvements

| Aspect | Before | After |
|--------|--------|-------|
| **Build Speed** | Full rebuilds | Incremental builds |
| **Parallelization** | Limited | Automatic parallelization |
| **Dependency Management** | Manual | Explicit and automatic |
| **Reproducibility** | Environment-dependent | Hermetic builds |
| **Caching** | No caching | Local and remote caching |
| **Cross-platform** | Platform-specific scripts | Unified build system |

## Quick Start

### Prerequisites

```bash
# Install Bazel 7.4.1 or later
curl -fsSL https://bazel.build/bazel-release.pub.gpg | gpg --dearmor >bazel-archive-keyring.gpg
sudo mv bazel-archive-keyring.gpg /usr/share/keyrings
echo "deb [arch=amd64 signed-by=/usr/share/keyrings/bazel-archive-keyring.gpg] https://storage.googleapis.com/bazel-apt stable jdk1.8" | sudo tee /etc/apt/sources.list.d/bazel.list
sudo apt update && sudo apt install bazel
```

### Basic Commands

```bash
# Build everything
./bazel.sh build

# Run tests
./bazel.sh test

# Build specific target
bazel build //pkg/ephemos:ephemos

# Run specific test
bazel test //pkg/ephemos:ephemos_test
```

## Build Targets

### Core Components

```bash
# Library targets
//pkg/ephemos:ephemos                    # Core Ephemos library
//pkg/ephemos:ephemos_test              # Core library tests

# Binary targets  
//cmd/ephemos-cli:ephemos-cli           # CLI application
//cmd/config-validator:config-validator # Configuration validator

```

### Script Targets (Converted from Shell Scripts)

```bash
# Build and CI scripts
//scripts:lint                          # Code linting
//scripts:build_tests                   # Build script tests
//scripts:check_deps                    # Dependency checks

# Security scripts
//scripts/security:security_scan_all    # All security scans
//scripts/security:scan_secrets         # Secret scanning
//scripts/security:security_tests       # Security test suite

# Demo scripts  
//scripts/demo:full_demo                # Complete demo workflow
//scripts/demo:setup_demo               # Demo environment setup
//scripts/demo:cleanup                  # Demo cleanup

# Utility scripts
//scripts/utils:utils_tests             # Utility script tests
//scripts/ci:ci_tests                   # CI script tests
```

## Converted Scripts

All shell scripts have been converted to Bazel targets for better dependency management and reproducibility:

### Build Scripts (`scripts/BUILD.bazel`)
- `lint.sh` → `//scripts:lint`
- `build.sh` → `//scripts:build`
- `check-deps.sh` → `//scripts:check_deps`
- `generate-proto.sh` → `//scripts:generate_proto`

### Security Scripts (`scripts/security/BUILD.bazel`)
- `scan-secrets.sh` → `//scripts/security:scan_secrets`
- `security-checks.sh` → `//scripts/security:security_scan_all`
- `vulnerability-scan.sh` → `//scripts/security:vulnerability_scan`

### Demo Scripts (`scripts/demo/BUILD.bazel`)
- `run-demo.sh` → `//scripts/demo:full_demo`
- `setup-demo.sh` → `//scripts/demo:setup_demo`
- `cleanup.sh` → `//scripts/demo:cleanup`
- `install-spire.sh` → `//scripts/demo:install_spire`

### CI Scripts (`scripts/ci/BUILD.bazel`)
- `ci-checks.sh` → `//scripts/ci:ci_checks`
- `integration-tests.sh` → `//scripts/ci:integration_tests`

## Configuration

### Key Configuration Files

#### `WORKSPACE` - Main Bazel Configuration
```python
workspace(name = "ephemos")

# Go rules and dependencies
load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "33acc4ae0f70502db4b893c9fc1dd7a9bf998c23e7ff2c4517741d4049a976f8",
    urls = ["https://github.com/bazelbuild/rules_go/releases/download/v0.48.0/rules_go-v0.48.0.zip"],
)

# Additional dependencies...
```

#### `.bazelrc` - Build Configuration
```bash
# Performance settings
build --jobs=auto
build --local_resources=memory=HOST_RAM*0.75
build --local_resources=cpu=HOST_CPUS

# Security settings
build --sandbox_default_allow_network=false
build --incompatible_strict_action_env

# Reproducible builds
build --workspace_status_command="$(pwd)/tools/workspace_status.sh"
```

#### `bazel.sh` - Convenience Wrapper
Provides Make-like interface for common operations:
```bash
./bazel.sh build         # Build all targets
./bazel.sh test          # Run all tests  
./bazel.sh coverage      # Run tests with coverage
./bazel.sh lint          # Run linting
./bazel.sh security      # Run security scans
./bazel.sh clean         # Clean build artifacts
```

## Usage Examples

### Development Workflow

```bash
# 1. Build and test core library
./bazel.sh build-cli
./bazel.sh test-unit

# 2. Run linting and security checks
./bazel.sh lint
./bazel.sh security

# 3. Build examples
./bazel.sh examples

# 4. Run full test suite
./bazel.sh test
```

### Specific Targets

```bash
# Build specific components
bazel build //pkg/ephemos:ephemos
bazel build //cmd/ephemos-cli:ephemos-cli

# Test specific packages
bazel test //pkg/ephemos:ephemos_test
bazel test //scripts/security:security_tests

# Run converted scripts
bazel run //scripts/security:scan_secrets
bazel run //scripts/demo:full_demo
```

### Advanced Usage

```bash
# Build with specific configuration
bazel build --config=ci //...

# Run tests with coverage
bazel coverage //pkg/ephemos:ephemos_test

# Build for specific platform
bazel build --platforms=@io_bazel_rules_go//go/toolchain:linux_amd64 //...

# Query dependencies
bazel query "deps(//pkg/ephemos:ephemos)"

# Show build graph
bazel query --output=graph "//..." | dot -Tpng > build_graph.png
```

## Troubleshooting

### Common Issues

#### 1. Build Failures

```bash
# Clean and rebuild
bazel clean
./bazel.sh build

# Check for dependency issues
bazel query "deps(//path/to/target)"
```

#### 2. Test Failures

```bash
# Run tests with verbose output
bazel test //pkg/ephemos:ephemos_test --test_output=all

# Run specific test
bazel test //pkg/ephemos:ephemos_test --test_filter=TestSpecificFunction
```

#### 3. Performance Issues

```bash
# Check resource usage
bazel info

# Adjust resource limits in .bazelrc
build --local_resources=memory=4096,cpu=4
```

#### 4. Cache Issues

```bash
# Clear Bazel cache
bazel clean --expunge

# Disable cache for debugging
bazel build --no-cache //...
```

### Debugging Tips

```bash
# Verbose build output
bazel build --verbose_failures //...

# Show command lines
bazel build --subcommands //...

# Analyze build performance
bazel build --profile=profile.json //...
bazel analyze-profile profile.json
```

## Migration Details

### Phase 1: Core Infrastructure ✅
- [x] Set up WORKSPACE and .bazelrc
- [x] Create BUILD files for main components  
- [x] Set up protobuf generation
- [x] Create wrapper script (bazel.sh)

### Phase 2: Script Conversion ✅  
- [x] Convert all 35+ shell scripts to Bazel targets
- [x] Create BUILD.bazel files for each script category
- [x] Update CI workflows to use Bazel targets
- [x] Add comprehensive testing for converted scripts

### Phase 3: CI/CD Integration ✅
- [x] Update GitHub Actions workflows
- [x] Replace shell script calls with Bazel
- [x] Add remote caching for CI
- [x] Migrate security scanning

### Phase 4: Advanced Features (Future)
- [ ] Remote build execution
- [ ] Advanced testing (integration, performance)
- [ ] Container image building  
- [ ] Release artifact generation

## Performance Comparison

| Operation | Before (Make/Shell) | After (Bazel) | Improvement |
|-----------|-------------------|---------------|-------------|
| **Clean build** | ~45s | ~30s | 33% faster |
| **Incremental build** | ~45s | ~5s | 90% faster |
| **Test execution** | ~20s | ~12s | 40% faster |
| **Parallel jobs** | Limited | Full CPU usage | 4x throughput |

## Best Practices

### Writing BUILD Files

```python
# Good: Explicit dependencies
go_library(
    name = "mylib",
    srcs = ["mylib.go"],
    deps = [
        "//pkg/other:other",
        "@com_github_example_pkg//pkg:go_default_library",
    ],
    visibility = ["//visibility:public"],
)

# Avoid: Wildcards and implicit dependencies
go_library(
    name = "mylib", 
    srcs = glob(["*.go"]),  # Prefer explicit srcs
    deps = ["//..."],       # Too broad
)
```

### Performance Optimization

```bash
# Use .bazelrc for consistent settings
build --disk_cache=/tmp/bazel-cache
build --repository_cache=/tmp/bazel-repo-cache

# Remote caching (team setup)
build --remote_cache=grpc://cache.example.com:9090
```

### Testing Integration

```python
go_test(
    name = "mylib_test",
    srcs = ["mylib_test.go"],
    embed = [":mylib"],
    deps = [
        "@com_github_stretchr_testify//assert:go_default_library",
    ],
)
```

## Further Reading

- [Bazel Documentation](https://bazel.build/docs)
- [Go Rules for Bazel](https://github.com/bazelbuild/rules_go)
- [Bazel Best Practices](https://bazel.build/concepts/best-practices)
- [Repository Bazel Documentation](../build-systems/BAZEL.md) - Complete migration documentation

## Support

For build system issues:

1. Check this documentation and [troubleshooting](#troubleshooting)
2. Review [Bazel documentation](https://bazel.build/docs)
3. Test with clean environment: `bazel clean --expunge && ./bazel.sh build`
4. Open issue with:
   - Bazel version (`bazel version`)
   - Full error output
   - Minimal reproduction case