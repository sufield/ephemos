# Composite Actions

This directory contains reusable GitHub Actions composite actions for the Ephemos project.

## Available Actions

### setup-go

Sets up the Go environment with caching and version verification.

**Usage:**
```yaml
- name: Setup Go Environment
  uses: ./.github/actions/setup-go
  with:
    go-version: '1.24'        # Optional, defaults to '1.24'
    verify-version: 'true'    # Optional, defaults to 'true'
```

**Features:**
- Installs specified Go version
- Caches Go modules and build artifacts
- Verifies Go version matches expected versions (1.23 or 1.24)
- Downloads project dependencies


## Benefits

1. **DRY Principle**: Eliminates repetitive setup code across workflow jobs
2. **Consistency**: Ensures identical setup across all CI/CD jobs
3. **Maintainability**: Single place to update setup logic
4. **Reusability**: Can be used across different workflows

## Alternative Approaches

For more complex setup requirements, you can also use:

1. **Makefile targets**: `make ci-setup`, `make ci-build`, etc.
2. **Shell scripts**: `scripts/ci-setup.sh` for complex multi-step setup
3. **Direct commands**: For simple one-off setup steps

## Examples

### Simple job with Go setup:
```yaml
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    - uses: ./.github/actions/setup-go
    - run: make ci-test
```

### Matrix job with different Go versions:
```yaml
jobs:
  build:
    strategy:
      matrix:
        go-version: ['1.23', '1.24']
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    - uses: ./.github/actions/setup-go
      with:
        go-version: ${{ matrix.go-version }}
    - run: make ci-build
```