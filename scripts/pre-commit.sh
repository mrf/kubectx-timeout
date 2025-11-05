#!/bin/bash
# Pre-commit hook for kubectx-timeout
# Install with: make setup-hooks
# Or manually: cp scripts/pre-commit.sh .git/hooks/pre-commit && chmod +x .git/hooks/pre-commit

set -e

echo "🔍 Running pre-commit checks..."
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# Track if any checks fail
FAILED=0

# Function to print step
step() {
    echo -e "${BLUE}▶ $1${NC}"
}

# Function to print success
success() {
    echo -e "${GREEN}✓ $1${NC}"
}

# Function to print error
error() {
    echo -e "${RED}✗ $1${NC}"
    FAILED=1
}

# Function to print warning
warning() {
    echo -e "${YELLOW}! $1${NC}"
}

# Check if we're in the right directory
if [ ! -f "go.mod" ]; then
    error "go.mod not found. Run this from the project root."
    exit 1
fi

# 1. Format check
step "Checking code formatting..."
UNFORMATTED=$(gofmt -s -l . 2>&1)
if [ -n "$UNFORMATTED" ]; then
    echo "$UNFORMATTED"
    error "Code not formatted. Run: gofmt -s -w ."
else
    success "Code formatting OK"
fi

# 2. Import organization
step "Checking import organization..."
if command -v goimports >/dev/null 2>&1; then
    UNIMPORTED=$(goimports -l . 2>&1)
    if [ -n "$UNIMPORTED" ]; then
        echo "$UNIMPORTED"
        error "Imports not organized. Run: goimports -w ."
    else
        success "Import organization OK"
    fi
else
    warning "goimports not found. Install: go install golang.org/x/tools/cmd/goimports@latest"
fi

# 3. Linting
step "Running linter..."
if command -v golangci-lint >/dev/null 2>&1; then
    if ! golangci-lint run --timeout=5m ./...; then
        error "Linting failed"
    else
        success "Linting passed"
    fi
else
    warning "golangci-lint not found. Install: https://golangci-lint.run/usage/install/"
fi

# 4. Tests
step "Running tests..."
if ! go test -race ./...; then
    error "Tests failed"
else
    success "Tests passed"
fi

# 5. Coverage check
step "Checking test coverage..."
go test -race -coverprofile=coverage.out ./... > /dev/null 2>&1
COVERAGE=$(go tool cover -func=coverage.out 2>/dev/null | grep total | awk '{print $3}' | sed 's/%//')

if [ -n "$COVERAGE" ]; then
    echo "Coverage: ${COVERAGE}%"
    if (( $(echo "$COVERAGE < 80" | bc -l) )); then
        error "Coverage ${COVERAGE}% is below 80% threshold"
    else
        success "Coverage ${COVERAGE}% meets threshold"
    fi
else
    warning "Could not calculate coverage"
fi

# 6. Security scan
step "Running security scan..."
if command -v gosec >/dev/null 2>&1; then
    if ! gosec -quiet ./...; then
        error "Security issues found"
    else
        success "Security scan passed"
    fi
else
    warning "gosec not found. Install: go install github.com/securego/gosec/v2/cmd/gosec@latest"
fi

# 7. Vulnerability check
step "Checking for known vulnerabilities..."
if command -v govulncheck >/dev/null 2>&1; then
    if ! govulncheck ./... > /dev/null 2>&1; then
        error "Vulnerabilities found"
    else
        success "No known vulnerabilities"
    fi
else
    warning "govulncheck not found. Install: go install golang.org/x/vuln/cmd/govulncheck@latest"
fi

# Cleanup
rm -f coverage.out

echo ""
if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}═══════════════════════════════════════${NC}"
    echo -e "${GREEN}✓ All pre-commit checks passed!${NC}"
    echo -e "${GREEN}═══════════════════════════════════════${NC}"
    exit 0
else
    echo -e "${RED}═══════════════════════════════════════${NC}"
    echo -e "${RED}✗ Pre-commit checks failed${NC}"
    echo -e "${RED}═══════════════════════════════════════${NC}"
    echo ""
    echo "Fix the issues above before committing."
    echo "To bypass this check (not recommended): git commit --no-verify"
    exit 1
fi
