#!/bin/bash
# Workspace status script for reproducible builds

set -eu

# Version information
if git rev-parse HEAD >/dev/null 2>&1; then
    echo "STABLE_VERSION $(git describe --tags --always --dirty 2>/dev/null || echo "dev")"
    echo "STABLE_COMMIT $(git rev-parse --short HEAD 2>/dev/null || echo "unknown")"
    echo "STABLE_COMMIT_HASH $(git rev-parse --short HEAD 2>/dev/null || echo "unknown")"
else
    echo "STABLE_VERSION dev"
    echo "STABLE_COMMIT unknown"
    echo "STABLE_COMMIT_HASH unknown"
fi

# Build information
echo "STABLE_BUILD_DATE $(date -u +"%Y-%m-%dT%H:%M:%SZ")"
echo "STABLE_BUILD_TIME $(date -u +"%Y-%m-%dT%H:%M:%SZ")"
echo "STABLE_BUILD_USER $(whoami)"
echo "STABLE_BUILD_HOST $(hostname)"

# Git status
if git rev-parse HEAD >/dev/null 2>&1; then
    if git diff-index --quiet HEAD --; then
        echo "STABLE_GIT_STATUS clean"
    else
        echo "STABLE_GIT_STATUS dirty"
    fi
else
    echo "STABLE_GIT_STATUS unknown"
fi