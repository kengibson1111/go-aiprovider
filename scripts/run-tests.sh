#!/bin/bash

# Bash script for running tests on Linux/macOS
# Usage: ./scripts/run-tests.sh [unit|integration|all|coverage]

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
GRAY='\033[0;37m'
NC='\033[0m' # No Color

# Default test type
TEST_TYPE=${1:-unit}

# Validate test type
case "$TEST_TYPE" in
    unit|integration|all|coverage)
        ;;
    *)
        echo -e "${RED}Error: Invalid test type '$TEST_TYPE'${NC}"
        echo "Usage: $0 [unit|integration|all|coverage]"
        exit 1
        ;;
esac

# Set up custom temp directory (optional, mainly for consistency)
CUSTOM_TEMP_DIR="/tmp/go-build-$$"
export GOTMPDIR="$CUSTOM_TEMP_DIR"

# Create temp directory
mkdir -p "$CUSTOM_TEMP_DIR"
echo -e "${GREEN}Using custom temp directory: $CUSTOM_TEMP_DIR${NC}"

# Cleanup function
cleanup() {
    if [ -d "$CUSTOM_TEMP_DIR" ]; then
        rm -rf "$CUSTOM_TEMP_DIR"
        echo -e "${GREEN}Cleaned up temp directory${NC}"
    fi
}

# Set trap to cleanup on exit
trap cleanup EXIT

case "$TEST_TYPE" in
    "unit")
        echo -e "${YELLOW}Running unit tests...${NC}"
        go test -short ./...
        ;;
    "integration")
        echo -e "${YELLOW}Running integration tests...${NC}"
        if [ ! -f ".env" ]; then
            echo -e "${YELLOW}Warning: .env file not found. Integration tests may be skipped.${NC}"
            echo "Copy .env.sample to .env and add your API keys to run integration tests."
        fi
        go test -run Integration ./...
        ;;
    "all")
        echo -e "${YELLOW}Running all tests...${NC}"
        if [ ! -f ".env" ]; then
            echo -e "${YELLOW}Warning: .env file not found. Integration tests may be skipped.${NC}"
        fi
        go test ./...
        ;;
    "coverage")
        echo -e "${YELLOW}Running tests with coverage...${NC}"
        COVERAGE_FILE="coverage.out"
        go test -short -coverprofile="$COVERAGE_FILE" ./...
        if [ $? -eq 0 ]; then
            echo -e "${GREEN}Coverage report generated: $COVERAGE_FILE${NC}"
            echo -e "${CYAN}View coverage in terminal:${NC}"
            echo -e "${GRAY}  go tool cover -func=$COVERAGE_FILE${NC}"
            echo -e "${CYAN}View coverage in browser:${NC}"
            echo -e "${GRAY}  go tool cover -html=$COVERAGE_FILE${NC}"
        fi
        ;;
esac

echo -e "${GREEN}Tests completed successfully!${NC}"