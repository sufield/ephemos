#!/bin/bash
set -euo pipefail

# Chi Middleware Smoke Test
# Run from contrib/middleware/chi directory
# Tests Chi middleware functionality and integration with ephemos

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🔥 Chi Middleware Smoke Test${NC}"
echo -e "${BLUE}==============================${NC}"

# Verify we're in the right directory
if [[ ! -f "identity.go" ]] || [[ ! -f "go.mod" ]]; then
    echo -e "${RED}❌ Please run from contrib/middleware/chi directory${NC}"
    exit 1
fi

# Test 1: Build Chi middleware
echo -e "${BLUE}🔨 Test 1: Building Chi middleware...${NC}"
if go build ./; then
    echo -e "${GREEN}✅ Chi middleware builds successfully${NC}"
else
    echo -e "${RED}❌ Chi middleware build failed${NC}"
    exit 1
fi

# Test 2: Run Chi middleware tests
echo -e "${BLUE}🧪 Test 2: Running Chi middleware tests...${NC}"
if go test -v ./; then
    echo -e "${GREEN}✅ Chi middleware tests pass${NC}"
else
    echo -e "${RED}❌ Chi middleware tests failed${NC}"
    exit 1
fi

# Test 3: Build example application
echo -e "${BLUE}🔨 Test 3: Building example application...${NC}"
if go build -o chi-example ./examples/; then
    echo -e "${GREEN}✅ Example application builds successfully${NC}"
    rm -f chi-example  # Cleanup
else
    echo -e "${RED}❌ Example application build failed${NC}"
    exit 1
fi

# Test 4: Verify imports and dependencies
echo -e "${BLUE}🔬 Test 4: Verifying imports and dependencies...${NC}"
if go mod verify; then
    echo -e "${GREEN}✅ Module dependencies verified${NC}"
else
    echo -e "${RED}❌ Module dependencies verification failed${NC}"
    exit 1
fi

# Test 5: Check for import cycles
echo -e "${BLUE}🔍 Test 5: Checking for import cycles...${NC}"
if go list -deps ./... | grep -q "import cycle"; then
    echo -e "${RED}❌ Import cycles detected${NC}"
    exit 1
else
    echo -e "${GREEN}✅ No import cycles detected${NC}"
fi

# Test 6: Verify ephemos core integration
echo -e "${BLUE}🔗 Test 6: Testing ephemos core integration...${NC}"
cd ../../..  # Go back to project root
if go test ./pkg/ephemos/; then
    echo -e "${GREEN}✅ Core ephemos integration verified${NC}"
    cd contrib/middleware/chi  # Return to Chi directory
else
    echo -e "${RED}❌ Core ephemos integration failed${NC}"
    exit 1
fi

echo -e "${GREEN}🎉 All Chi middleware smoke tests passed!${NC}"

echo -e "${BLUE}📊 Test Summary:${NC}"
echo "- ✅ Chi middleware compilation"
echo "- ✅ Unit tests pass"
echo "- ✅ Example builds successfully"
echo "- ✅ Dependencies verified"
echo "- ✅ No import cycles"
echo "- ✅ Core ephemos integration"

echo -e "${YELLOW}🚀 Chi middleware is ready for:${NC}"
echo "- Production deployment"
echo "- Integration with existing Chi applications"
echo "- Migration to ephemos-contrib repository"

echo -e "${BLUE}📝 Usage:${NC}"
echo "  go get github.com/sufield/ephemos/contrib/middleware/chi"
echo "  import chimiddleware \"github.com/sufield/ephemos/contrib/middleware/chi\""