# Agent Instructions for kubectx-timeout

This file contains mandatory instructions for all AI agents working on this repository. These instructions ensure code quality, consistency, and adherence to best practices.

## Critical Rules

### 0. Never Merge with Failing Tests

**NEVER merge with failing tests.** All tests must pass before merging to main.

- If tests fail, investigate and fix them before merging
- If a test is flaky or unrelated to your changes, fix it first or ensure it's addressed separately
- No exceptions to this rule

### 1. Test-Driven Development (TDD) is MANDATORY

All code changes MUST follow TDD practices:

1. **Write the test first** - Before implementing any feature or fix:
   - Write a failing test that describes the expected behavior
   - Run the test to verify it fails for the right reason
   - Only then implement the code to make the test pass

2. **Refactor with confidence** - After tests pass:
   - Refactor code while keeping tests green
   - Ensure all existing tests continue to pass

3. **Test coverage requirements**:
   - Internal package coverage MUST be >= 60%
   - All new code MUST have corresponding tests
   - Edge cases and error conditions MUST be tested

### 2. Definition of "Done"

A change is NOT complete until ALL of the following checks pass locally:

#### Required Local Checks (in order):

1. **Format Check**
   ```bash
   # Check formatting
   gofmt -s -l .
   # Should output nothing. If files are listed, run:
   gofmt -s -w .
   ```

2. **Import Organization**
   ```bash
   # Install goimports if needed
   go install golang.org/x/tools/cmd/goimports@latest

   # Check imports
   goimports -l .
   # Should output nothing. If files are listed, run:
   goimports -w .
   ```

3. **Dependency Verification**
   ```bash
   go mod download
   go mod verify
   go mod tidy
   ```

4. **Linting**
   ```bash
   # Run go vet
   go vet ./...

   # Install and run staticcheck
   go install honnef.co/go/tools/cmd/staticcheck@latest
   staticcheck ./...
   ```

5. **Testing**
   ```bash
   # Run tests with race detector and coverage
   go test -race -v -coverprofile=coverage.out ./...

   # Verify coverage threshold (>= 60% for internal package)
   go tool cover -func=coverage.out | grep -E "github.com/mrf/kubectx-timeout/internal"
   ```

6. **Security Scanning**
   ```bash
   # Install gosec if needed
   go install github.com/securego/gosec/v2/cmd/gosec@latest

   # Run security scan - MUST have zero issues
   gosec -fmt=json -out=gosec-report.json ./...

   # Check results
   cat gosec-report.json | jq '.Issues | length'
   # Must output: 0

   # Install govulncheck if needed
   go install golang.org/x/vuln/cmd/govulncheck@latest

   # Check for vulnerabilities - MUST have zero vulnerabilities
   govulncheck ./...
   ```

7. **Build Verification**
   ```bash
   # Build the binary
   make build

   # Verify binary exists and is valid
   file bin/kubectx-timeout
   ls -lh bin/kubectx-timeout
   ```

### 3. Automated Pre-Commit Check

**MANDATORY**: Before declaring any work complete, run this comprehensive check:

```bash
# Complete pre-commit validation
./scripts/pre-commit-check.sh
```

If the script doesn't exist, create and run this sequence:

```bash
#!/bin/bash
set -e  # Exit on first error

echo "=== Running Pre-Commit Checks ==="
echo ""

echo "1. Formatting..."
if [ "$(gofmt -s -l . | wc -l)" -gt 0 ]; then
    echo "❌ Code not formatted. Running gofmt..."
    gofmt -s -w .
fi

echo "2. Organizing imports..."
if ! command -v goimports &> /dev/null; then
    go install golang.org/x/tools/cmd/goimports@latest
fi
if [ "$(goimports -l . | wc -l)" -gt 0 ]; then
    echo "❌ Imports not organized. Running goimports..."
    goimports -w .
fi

echo "3. Tidying dependencies..."
go mod tidy

echo "4. Running go vet..."
go vet ./...

echo "5. Running staticcheck..."
if ! command -v staticcheck &> /dev/null; then
    go install honnef.co/go/tools/cmd/staticcheck@latest
fi
staticcheck ./...

echo "6. Running tests with race detector..."
go test -race -v -coverprofile=coverage.out ./...

echo "7. Checking coverage threshold..."
coverage=$(go tool cover -func=coverage.out | \
  grep -E "github.com/mrf/kubectx-timeout/internal" | \
  awk '{sum+=$NF; count++} END {if (count>0) printf "%.1f", sum/count; else print "0"}')
echo "Internal package coverage: $coverage%"
if (( $(echo "$coverage < 60" | bc -l) )); then
    echo "❌ Coverage $coverage% is below 60% threshold"
    exit 1
fi

echo "8. Running security scan (gosec)..."
if ! command -v gosec &> /dev/null; then
    go install github.com/securego/gosec/v2/cmd/gosec@latest
fi
gosec -fmt=json -out=gosec-report.json ./...
issues=$(cat gosec-report.json | jq '.Issues | length' 2>/dev/null || echo "0")
if [ "$issues" -gt 0 ]; then
    echo "❌ Security issues found: $issues"
    cat gosec-report.json | jq '.Issues'
    exit 1
fi

echo "9. Running vulnerability check (govulncheck)..."
if ! command -v govulncheck &> /dev/null; then
    go install golang.org/x/vuln/cmd/govulncheck@latest
fi
govulncheck ./...

echo "10. Building binary..."
make build

echo "11. Verifying binary..."
if [ ! -f bin/kubectx-timeout ]; then
    echo "❌ Binary build failed"
    exit 1
fi

echo ""
echo "✅ All pre-commit checks passed!"
```

## Development Workflow

### When Implementing a New Feature

1. **Understand the requirement**
   - Read the issue/request carefully
   - Ask clarifying questions if needed

2. **Plan the implementation**
   - Identify which components need changes
   - Consider edge cases and error conditions

3. **Write tests FIRST (TDD)**
   ```bash
   # Example workflow:
   # 1. Create test file or add to existing
   # 2. Write failing test
   go test -v ./internal -run TestNewFeature
   # 3. Implement the feature
   # 4. Run tests until they pass
   go test -v ./internal -run TestNewFeature
   # 5. Run full test suite
   go test -v ./...
   ```

4. **Implement the feature**
   - Write minimal code to make tests pass
   - Follow Go best practices
   - Add documentation comments

5. **Refactor**
   - Improve code quality
   - Keep tests passing

6. **Run ALL pre-commit checks**
   ```bash
   ./scripts/pre-commit-check.sh
   ```

7. **ONLY THEN declare work complete**

### When Fixing a Bug

1. **Write a failing test** that reproduces the bug
   - This ensures the bug is fixed and won't regress

2. **Fix the bug** - make the test pass

3. **Run ALL pre-commit checks**

4. **Verify the fix** with the original reproduction steps

### When Refactoring

1. **Ensure tests exist** for the code being refactored

2. **Keep tests green** throughout the refactoring

3. **Run ALL pre-commit checks**

## Code Quality Standards

### Testing

- Use table-driven tests for multiple test cases
- Use `t.Run()` for subtests
- Mock external dependencies (kubectl, file system when needed)
- Test both success and error paths
- Test edge cases and boundary conditions
- Use `-race` flag to detect race conditions

### Security

- Never execute shell commands with user input without validation
- Use `exec.Command()` with separate arguments (not shell strings)
- Validate all file paths
- Check for path traversal vulnerabilities
- Follow principle of least privilege

### Documentation

- Add godoc comments for all exported functions and types
- Update README.md if behavior changes
- Add examples for complex functions

### Error Handling

- Always check and handle errors
- Return meaningful error messages
- Wrap errors with context using `fmt.Errorf("context: %w", err)`
- Don't panic unless absolutely necessary

## Integration Test Requirements

For code that interacts with kubectl or shell:

1. Create integration tests in `*_integration_test.go`
2. Use build tags: `// +build integration`
3. Set up test fixtures in `testdata/`
4. Clean up resources after tests
5. Make tests runnable without actual kubernetes cluster when possible

## Platform-Specific Code

This project targets macOS but should be testable on Linux:

- Use build tags for platform-specific code
- Provide test mocks for platform-specific functionality
- Document platform requirements clearly

## CI/CD Alignment

The CI/CD pipeline runs these jobs (all must pass):

1. **Lint Job** - gofmt, go vet, staticcheck
2. **Test Job** - tests with race detector, coverage check (>= 60%)
3. **Security Job** - gosec, govulncheck
4. **Build Job** - binary compilation
5. **Quality Metrics** - cyclomatic complexity, code metrics

Your local checks MUST match these CI jobs.

## Common Pitfalls to Avoid

1. ❌ Implementing code before writing tests
2. ❌ Declaring work "done" without running all checks locally
3. ❌ Skipping security scans
4. ❌ Ignoring test coverage requirements
5. ❌ Not testing error conditions
6. ❌ Committing unformatted code
7. ❌ Not verifying the build succeeds
8. ❌ Introducing race conditions (always use `-race` flag)

## Quick Reference Commands

```bash
# Format code
gofmt -s -w .
goimports -w .

# Tidy dependencies
go mod tidy

# Run linters
go vet ./...
staticcheck ./...

# Run tests with coverage
go test -race -v -coverprofile=coverage.out ./...

# Check coverage
go tool cover -html=coverage.out  # View in browser
go tool cover -func=coverage.out   # View in terminal

# Security scans
gosec ./...
govulncheck ./...

# Build
make build

# Run everything (recommended)
./scripts/pre-commit-check.sh
```

## When in Doubt

1. Run the tests first: `go test -v ./...`
2. Check the CI/CD workflows: `.github/workflows/`
3. Look at existing tests for examples
4. Follow the patterns already established in the codebase

## Remember

**Your work is not done until:**
- ✅ Tests are written (TDD)
- ✅ All tests pass
- ✅ Coverage >= 60%
- ✅ Code is formatted
- ✅ Linters pass
- ✅ Security scans pass
- ✅ Build succeeds

**No shortcuts. No exceptions.**

## Issue Management

This project uses GitHub Issues for tracking work. Use the `gh` CLI to find and manage issues:

```bash
# List open issues
gh issue list

# List issues with specific label
gh issue list --label "priority:p1"

# View issue details
gh issue view 123

# Create a new issue
gh issue create --title "Title" --body "Description"

# Add labels to an issue
gh issue edit 123 --add-label "bug"

# Close an issue
gh issue close 123

# Search issues
gh issue list --search "keyword"
```

Always reference issue numbers in commits and PRs when applicable.
