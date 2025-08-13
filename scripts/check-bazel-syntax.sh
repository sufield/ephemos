#!/bin/bash
#
# Simple Bazel Syntax Check Script
# Validates BUILD.bazel files using buildifier and bazel query
#

set -euo pipefail

# Check if buildifier is available
if ! command -v buildifier >/dev/null 2>&1; then
    echo "❌ buildifier not found. Please install it:"
    echo "curl -LO https://github.com/bazelbuild/buildtools/releases/latest/download/buildifier-linux-amd64"
    echo "chmod +x buildifier-linux-amd64 && sudo mv buildifier-linux-amd64 /usr/local/bin/buildifier"
    exit 1
fi

# Check if bazel is available
if ! command -v bazel >/dev/null 2>&1; then
    echo "❌ bazel not found. Please install Bazelisk:"
    echo "curl -LO https://github.com/bazelbuild/bazelisk/releases/latest/download/bazelisk-linux-amd64"
    echo "chmod +x bazelisk-linux-amd64 && sudo mv bazelisk-linux-amd64 /usr/local/bin/bazel"
    exit 1
fi

echo "🔍 Checking Bazel file syntax..."

# Find and check all BUILD.bazel and .bzl files
BAZEL_FILES=$(find . -name "BUILD.bazel" -o -name "*.bzl" | wc -l)
echo "Found $BAZEL_FILES Bazel files"

# Check syntax with buildifier
echo "Running buildifier checks..."
if find . -name "BUILD.bazel" -o -name "*.bzl" | xargs buildifier --mode=check --lint=warn; then
    echo "✅ Buildifier checks passed"
else
    echo "❌ Buildifier found issues. Run: buildifier --lint=fix \$(find . -name 'BUILD.bazel' -o -name '*.bzl')"
    exit 1
fi

# Check workspace parsing
echo "Checking workspace parsing..."
if bazel query --ui_event_filters=-info --noshow_progress '//...' >/dev/null 2>&1; then
    echo "✅ Bazel workspace parsing successful"
else
    echo "❌ Bazel workspace parsing failed"
    bazel query '//...' 2>&1 | head -10
    exit 1
fi

echo "🎉 All Bazel syntax checks passed!"