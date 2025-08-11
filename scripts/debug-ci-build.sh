#!/bin/bash
# Comprehensive CI diagnostic script
set -e

echo "🔍 === COMPREHENSIVE CI BUILD DIAGNOSTIC ==="
echo "Working directory: $(pwd)"
echo "User: $(whoami)"
echo "Date: $(date)"
echo "Platform: $(uname -s 2>/dev/null || echo 'Windows')"
echo

echo "🗂️ === DIRECTORY STRUCTURE ==="
if command -v ls >/dev/null 2>&1; then
    ls -la
else
    # Windows/PowerShell fallback
    dir
fi
echo

echo "📁 === PKG/EPHEMOS DIRECTORY ==="
if [ -d "pkg/ephemos" ]; then
    ls -la pkg/ephemos/
    echo
    echo "📄 === EPHEMOS.GO EXISTS ==="
    if [ -f "pkg/ephemos/ephemos.go" ]; then
        echo "✅ ephemos.go exists"
        wc -l pkg/ephemos/ephemos.go
    else
        echo "❌ ephemos.go NOT found"
    fi
    echo
    echo "📄 === SERVER.GO EXISTS ==="
    if [ -f "pkg/ephemos/server.go" ]; then
        echo "✅ server.go exists" 
        wc -l pkg/ephemos/server.go
    else
        echo "❌ server.go NOT found"
    fi
else
    echo "❌ pkg/ephemos directory NOT found"
fi
echo

echo "🔍 === FUNCTION DEFINITIONS SEARCH ==="
echo "Looking for TransportServer type definition:"
grep -rn "type TransportServer" . || echo "❌ TransportServer type not found"
echo
echo "Looking for newTransportServer function:"
grep -rn "func newTransportServer" . || echo "❌ newTransportServer function not found"
echo
echo "Looking for mount function:"  
grep -rn "func mount\[" . || echo "❌ mount function not found"
echo

echo "🏗️ === GO BUILD TESTS ==="
echo "Testing go build on pkg/ephemos:"
if cd pkg/ephemos && go build -v . && cd ../..; then
    echo "✅ pkg/ephemos builds successfully in isolation"
else 
    echo "❌ pkg/ephemos failed to build in isolation"
fi
echo

echo "📦 === GO MODULE STATUS ==="
go mod verify || echo "❌ go mod verify failed"
echo
echo "Go version: $(go version)"
echo "GOPATH: $(go env GOPATH)" 
echo "GOCACHE: $(go env GOCACHE)"
echo "GO111MODULE: $(go env GO111MODULE)"
echo

echo "🧹 === CACHE CLEARING ==="
go clean -cache
go clean -testcache  
go clean -modcache || true
echo "✅ All caches cleared"
echo

echo "📥 === FRESH MODULE DOWNLOAD ==="
go mod download
echo "✅ Modules downloaded"
echo

echo "🏗️ === BUILDING ALL TARGETS ==="
echo "Building main CLI binary..."
make build || echo "❌ Main build failed"

echo "Building examples..."
make examples || {
    echo "❌ Examples build failed"
    echo "Let's try building each example individually:"
    
    for example in examples/*/; do
        if [ -d "$example" ] && [ -f "$example/main.go" ]; then
            echo "Building $example..."
            go build -v "$example" || echo "❌ Failed to build $example"
        fi
    done
}
echo

echo "🔍 === FINAL FILE VERIFICATION ==="
echo "Final check of critical files:"
ls -la pkg/ephemos/ephemos.go pkg/ephemos/server.go || echo "❌ Critical files missing"

echo "✅ === DIAGNOSTIC COMPLETE ==="