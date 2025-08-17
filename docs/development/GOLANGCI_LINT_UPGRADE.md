# golangci-lint Upgrade Guide

## Current Status

**Current Version:** v1.64.8 (latest stable v1.x)
**Configuration:** `.golangci.yml` (v1 format)

## Future v2 Migration

golangci-lint v2 introduces breaking changes but provides automatic migration tools.

### Key Changes in v2
- `disable-all` and `enable` options replaced with `linters.default`
- New configuration structure
- Enhanced performance and new linters

### Migration Command
```bash
# Automatic configuration migration
golangci-lint migrate
```

### v2 Installation
```bash
# Install v2 (when ready to migrate)
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
```

### Current v1 Installation
```bash
# Current stable v1 installation
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.64.8
```

## Upgrade Steps (When Ready)

1. **Backup current config**: `cp .golangci.yml .golangci.yml.backup`
2. **Install v2**: Use installation command above
3. **Migrate config**: Run `golangci-lint migrate`
4. **Test thoroughly**: Run linting on entire codebase
5. **Update CI/CD**: Update any CI workflows to use v2

## Notes

- v1.64.8 provides excellent stability and comprehensive linting
- v2 migration can be deferred until project requirements demand it
- Current configuration works well with Go 1.24+ projects