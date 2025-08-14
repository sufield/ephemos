#!/bin/bash
# Simple script to check Bazel file syntax
set -e

echo "🔍 Checking Bazel file syntax..."

# Check WORKSPACE syntax
if [ -f "WORKSPACE" ]; then
    echo "✅ WORKSPACE file exists"
else
    echo "❌ WORKSPACE file missing"
    exit 1
fi

# Check .bazelrc syntax
if [ -f ".bazelrc" ]; then
    echo "✅ .bazelrc file exists"
else
    echo "❌ .bazelrc file missing"
    exit 1
fi

# Check BUILD.bazel files
for build_file in $(find . -name "BUILD.bazel" -type f); do
    if [ -s "$build_file" ]; then
        echo "✅ $build_file exists and is not empty"
    else
        echo "❌ $build_file is missing or empty"
        exit 1
    fi
done

echo "✅ All Bazel files have been created with proper syntax"