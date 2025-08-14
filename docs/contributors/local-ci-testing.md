# Local CI/CD Testing with Act

This guide explains how to run GitHub Actions workflows locally using `act`, enabling faster development cycles and debugging without consuming GitHub Actions minutes.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Configuration](#configuration)
- [Usage](#usage)
- [Available Workflows](#available-workflows)
- [Troubleshooting](#troubleshooting)
- [Best Practices](#best-practices)

## Overview

[Act](https://github.com/nektos/act) is a tool that allows you to run GitHub Actions locally using Docker containers. This enables:

- **Faster feedback loops**: Test changes without pushing to GitHub
- **Cost savings**: Reduce GitHub Actions compute usage
- **Enhanced debugging**: Full access to workflow execution environment
- **Offline development**: Test CI/CD pipelines without internet connectivity

## Prerequisites

- Docker installed and running
- Git repository with GitHub Actions workflows
- Sufficient disk space for Docker images (~2-4GB)

## Installation

### Option 1: Download Pre-built Binary (Recommended)

```bash
# Download act v0.2.80 (Linux x86_64)
curl -s https://api.github.com/repos/nektos/act/releases/latest \
  | grep "browser_download_url.*linux_amd64" \
  | cut -d '"' -f 4 \
  | xargs curl -L -o act

# Make executable
chmod +x act

# Move to PATH (optional)
sudo mv act /usr/local/bin/
```

### Option 2: Package Manager

```bash
# macOS (Homebrew)
brew install act

# Linux (AUR)
yay -S act

# Windows (Chocolatey)
choco install act-cli
```

## Configuration

The repository includes pre-configured files for act:

### `.actrc` - Act Configuration

```yaml
# Use medium-sized runner by default
-P ubuntu-latest=catthehacker/ubuntu:act-latest

# Enable verbose output for debugging
--verbose

# Use local secrets file
--secret-file .secrets

# Set default event
--eventpath .github/workflows/events/push.json
```

### `.secrets` - Environment Variables

```bash
# GitHub Actions secrets for act local testing
# Add your secrets here in KEY=VALUE format
# Example:
# GITHUB_TOKEN=your_token_here
# BAZEL_CACHE_KEY=your_cache_key_here
```

### `.github/workflows/events/push.json` - Event Simulation

Simulates a push event to the current branch for testing workflows.

## Usage

### List Available Workflows

```bash
# Show all workflows and jobs
./act -l

# Show jobs for specific workflow
./act -W .github/workflows/bazel-ci.yml -l
```

### Run Workflows

```bash
# Run all push-triggered workflows
./act push

# Run specific workflow
./act -W .github/workflows/secrets-scan.yml

# Run specific job
./act -W .github/workflows/bazel-ci.yml -j bazel-build

# Run with different event
./act pull_request
```

### Dry Run (Validation Only)

```bash
# Validate workflow without execution
./act --dryrun

# Validate specific workflow
./act --dryrun -W .github/workflows/bazel-ci.yml
```

### Debug Options

```bash
# Verbose output
./act --verbose

# Reuse containers for debugging
./act --reuse

# Keep containers after failure
./act --rm=false
```

## Available Workflows

The repository contains the following workflows that can be tested locally:

### Core CI/CD Pipeline (`ci.yml`)
- **deps**: Dependency checks
- **security**: Security scanning
- **build**: Build and compilation
- **test**: Unit and integration tests
- **lint**: Code quality checks
- **integration**: Integration tests
- **benchmark**: Performance benchmarks

### Bazel Build System (`bazel-ci.yml`)
- **bazel-build**: Build all Bazel targets
- **bazel-integration**: Integration tests with Bazel

### Security Scanning (`secrets-scan.yml`)
- **git-secrets-scan**: AWS secrets detection
- **gitleaks-scan**: Generic secrets detection
- **trufflehog-scan**: Advanced secrets detection
- **github-secret-scan**: GitHub-specific scanning

### Documentation (`docs-and-release.yml`)
- **docs-lint**: Documentation linting
- **api-docs**: API documentation generation
- **validate-examples**: Example validation

### Security & Dependencies (`security.yml`)
- **vulnerability-scan**: Dependency vulnerabilities
- **container-security**: Container security scanning
- **license-check**: License compliance
- **sast-scan**: Static analysis security testing

### Performance Testing (`performance.yml`)
- **benchmark**: Performance benchmarks
- **memory-profile**: Memory usage analysis
- **load-test**: Load testing

### Code Analysis (`codeql.yml`)
- **analyze**: CodeQL security analysis

### Fuzzing (`fuzzing.yml`)
- **go-fuzzing**: Go native fuzzing
- **clusterfuzz-lite**: ClusterFuzz testing
- **property-testing**: Property-based testing

## Troubleshooting

### Common Issues

#### 1. Docker Permission Errors
```bash
# Add user to docker group (Linux)
sudo usermod -aG docker $USER
# Log out and back in, or run:
newgrp docker
```

#### 2. Container Image Pull Failures
```bash
# Pre-pull required images
docker pull catthehacker/ubuntu:act-latest
docker pull node:16-bullseye-slim
```

#### 3. Workflow Fails with "command not found"
Some workflows expect tools that aren't in the base container. Either:
- Use a different runner image with more tools pre-installed
- Skip tool-specific steps when running locally
- Install tools as part of the workflow

#### 4. Secrets Not Available
```bash
# Check secrets file exists and is readable
cat .secrets

# Use explicit secrets flag
./act --secret-file .secrets
```

#### 5. Out of Disk Space
```bash
# Clean up old containers and images
docker system prune -f

# Remove act-specific volumes
docker volume prune -f
```

### Debugging Failed Workflows

```bash
# Run with maximum verbosity
./act --verbose --pull=false

# Keep container for inspection
./act --reuse

# Enter container for debugging
docker exec -it <container_id> /bin/bash
```

## Best Practices

### Development Workflow

1. **Test locally first**: Run workflows with act before pushing
2. **Use specific jobs**: Test only relevant jobs during development
3. **Cache images**: Use `--pull=false` to avoid re-downloading images
4. **Iterate quickly**: Use `--reuse` for faster iteration cycles

### Performance Optimization

```bash
# Skip image pulls for faster runs
./act --pull=false

# Reuse containers
./act --reuse

# Run specific jobs only
./act -j specific-job-name

# Use local event files to avoid network calls
./act --eventpath .github/workflows/events/push.json
```

### Security Considerations

- **Secrets management**: Never commit real secrets to `.secrets` file
- **Network isolation**: Act containers run on host network by default
- **Resource limits**: Be mindful of CPU/memory usage for intensive workflows

### CI/CD Integration

```bash
# Pre-commit hook example
#!/bin/bash
# .git/hooks/pre-commit
echo "Running CI checks locally..."
./act -j lint -j test --pull=false
if [ $? -ne 0 ]; then
    echo "❌ Local CI checks failed. Fix issues before committing."
    exit 1
fi
echo "✅ Local CI checks passed"
```

## Advanced Configuration

### Custom Runner Images

```bash
# Use different images for different platforms
./act -P ubuntu-latest=catthehacker/ubuntu:full-latest
./act -P ubuntu-20.04=catthehacker/ubuntu:act-20.04
```

### Environment Variables

```bash
# Set variables for specific runs
./act --env CUSTOM_VAR=value

# Use environment file
./act --env-file .env.local
```

### Workflow-Specific Configs

```bash
# Run with specific matrix configurations
./act --matrix os:ubuntu-latest --matrix go-version:1.21

# Override job conditions
./act --job build --env FORCE_BUILD=true
```

## Integration with Development Tools

### VS Code Integration

Create `.vscode/tasks.json`:

```json
{
    "version": "2.0.0",
    "tasks": [
        {
            "label": "Run Local CI",
            "type": "shell",
            "command": "./act",
            "args": ["-j", "test", "--pull=false"],
            "group": "test",
            "presentation": {
                "echo": true,
                "reveal": "always",
                "focus": false,
                "panel": "shared"
            }
        }
    ]
}
```

### Make Integration

```makefile
# Makefile targets
.PHONY: ci-local ci-lint ci-test

ci-local:
	./act --pull=false

ci-lint:
	./act -j lint --pull=false

ci-test:
	./act -j test --pull=false

ci-security:
	./act -W .github/workflows/secrets-scan.yml --pull=false
```

## Further Reading

- [Act Documentation](https://github.com/nektos/act)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Docker Documentation](https://docs.docker.com/)
- [Repository CI/CD Documentation](./ci-cd-overview.md)

## Support

For issues with local CI testing:

1. Check this documentation first
2. Review act's [GitHub Issues](https://github.com/nektos/act/issues)
3. Test the same workflow on GitHub Actions to isolate act-specific issues
4. Open an issue in this repository with:
   - Act version (`./act --version`)
   - Docker version (`docker --version`)
   - Full command and error output
   - Workflow file content