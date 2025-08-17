# golangci-lint Upgrade Guide

## Current Status

**Current Version:** v2.4.0 (latest stable v2.x)
**Configuration:** `.golangci.yml` (v2 format)

## Recent v2 Migration

golangci-lint v2 introduced breaking changes that have been migrated in this project.

### Key Changes in v2 (Now Applied)
- `disable-all` and `enable` options replaced with `linters.default` ✅
- New configuration structure ✅
- Enhanced performance and new linters ✅
- Go 1.25 support ✅

### v2.4.0 Installation
```bash
# Current v2.4.0 installation
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.4.0
```

### v2.4.0 Features
- Added support for Go 1.25
- Updated dependencies and linters
- Improved godox linter (trim filepath from report messages)
- Enhanced staticcheck with empty options support

## Migration Completed

1. **Updated installation script** to use v2.4.0 ✅
2. **Migrated configuration** from v1 to v2 format ✅
3. **Updated documentation** to reflect v2 usage ✅

## Notes

- v2.4.0 provides enhanced performance and Go 1.25 support
- Configuration has been updated to use the new v2 format
- All linters remain functionally equivalent with improved performance