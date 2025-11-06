# Contributing to kubectx-timeout

Thank you for your interest in contributing to kubectx-timeout! This document provides guidelines and standards for contributing to this project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [How to Contribute](#how-to-contribute)
- [Code Standards](#code-standards)
- [Testing Requirements](#testing-requirements)
- [Pull Request Process](#pull-request-process)
- [Review Process](#review-process)
- [Quick Reference](#quick-reference)

---

## Code of Conduct

We are committed to providing a welcoming and inclusive environment for all contributors. Please be respectful and constructive in all interactions.

---

## Getting Started

### Prerequisites

- Go 1.21 or later
- macOS (this project targets macOS specifically)
- kubectl installed and configured
- Git

### Fork and Clone

1. Fork the repository on GitHub
2. Clone your fork locally:
   ```bash
   git clone https://github.com/YOUR-USERNAME/kubectx-timeout.git
   cd kubectx-timeout
   ```
3. Add the upstream repository:
   ```bash
   git remote add upstream https://github.com/ORIGINAL-OWNER/kubectx-timeout.git
   ```

---

## Development Setup

### Install Development Tools

Install all required development tools:

```bash
make setup-tools
```

This installs:
- `gofmt` - Go code formatter (built-in)
- `goimports` - Import organizer
- `golangci-lint` - Comprehensive linter
- `gosec` - Security scanner
- `govulncheck` - Vulnerability scanner

Verify tools are installed:

```bash
make verify-tools
```

### Optional: Install Pre-commit Hook

We recommend installing the pre-commit hook to catch issues early:

```bash
make setup-hooks
```

This will run formatting, linting, and tests before each commit.

---

## How to Contribute

### Reporting Bugs

- Check if the bug has already been reported in Issues
- If not, create a new issue with:
  - Clear description of the problem
  - Steps to reproduce
  - Expected vs actual behavior
  - Your environment (macOS version, Go version, kubectl version)

### Suggesting Features

- Check if the feature has already been requested
- Create a new issue describing:
  - The problem you're trying to solve
  - Your proposed solution
  - Any alternatives considered

### Contributing Code

1. **Create a feature branch** from `main`:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes** following our [Code Standards](#code-standards)

3. **Write tests** for your changes (minimum 80% coverage)

4. **Run all checks** before committing:
   ```bash
   make pre-commit
   ```

5. **Commit your changes** with clear, descriptive commit messages

6. **Push to your fork**:
   ```bash
   git push origin feature/your-feature-name
   ```

7. **Open a Pull Request** using the PR template

---

## Code Standards

### Go Style Guidelines

We follow standard Go conventions and best practices:

- **Formatting**: Use `gofmt -s` (enforced by CI)
- **Imports**: Use `goimports` for organization
- **Naming**: Follow Go naming conventions
  - Packages: lowercase, single word (`config`, not `config_loader`)
  - Functions: camelCase, start with verb (`LoadConfig`)
  - Interfaces: noun or noun phrase (`ConfigLoader`)
- **Comments**: All exported types and functions must have godoc comments
- **Error Handling**: Always handle errors explicitly, never ignore with `_` without justification

### Code Quality

- **Function Length**: Max 50 lines (ideal 20-30)
- **Cyclomatic Complexity**: Max 10 per function
- **Line Length**: Max 120 characters
- **Nesting Depth**: Max 4 levels

### Key Principles

1. **Correctness First** - Code must work as intended
2. **Clarity Over Cleverness** - Readable code beats clever code
3. **Explicit Error Handling** - Never swallow errors
4. **Simple Design** - YAGNI (You Aren't Gonna Need It)

For detailed Go best practices and examples, see [DEVELOPMENT.md](DEVELOPMENT.md).

---

## Testing Requirements

### Coverage Requirements

- **Minimum**: 80% overall coverage
- **Core Logic**: 90% coverage
- **Error Paths**: 100% coverage

### Test Quality

- Use table-driven tests for multiple scenarios
- Test edge cases and error conditions
- Mock external dependencies (kubectl, filesystem)
- Tests must be deterministic (no flaky tests)
- Integration tests for cross-component features

### Running Tests

```bash
# Run all tests with race detector
make test

# Run tests with coverage
make coverage

# Run integration tests
go test -tags=integration ./...
```

### Test Structure

```go
func TestFeature(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {
            name:     "valid input",
            input:    "test",
            expected: "test-result",
            wantErr:  false,
        },
        // More test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := Feature(tt.input)
            if tt.wantErr {
                require.Error(t, err)
                return
            }
            require.NoError(t, err)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

---

## Pull Request Process

### Before Submitting

1. **Update your branch** with latest main:
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

2. **Run all checks**:
   ```bash
   make pre-commit
   ```

3. **Ensure tests pass**:
   ```bash
   make test
   ```

### PR Requirements

- [ ] PR title clearly describes the change
- [ ] Description explains what and why (not just how)
- [ ] Linked to related issue(s)
- [ ] All automated checks passing (CI must be green)
- [ ] Tests added/updated with adequate coverage
- [ ] Documentation updated if needed
- [ ] No merge conflicts with main

### PR Size Guidelines

- **Small** (< 200 lines): Ideal, ~30 min review
- **Medium** (200-500 lines): ~1 hour review
- **Large** (> 500 lines): Consider breaking into smaller PRs

### Commit Messages

Write clear, descriptive commit messages:

```
Short summary (50 chars or less)

More detailed explanation if needed. Wrap at 72 characters.
Explain what and why, not how.

- Bullet points are okay
- Use present tense ("Add feature" not "Added feature")
```

---

## Review Process

### What to Expect

1. **Automated Checks** run first (must pass before human review)
2. **Code Review** by maintainer (~45 min for medium PR)
3. **Feedback** may be provided at three levels:
   - **BLOCKER**: Must fix before merge
   - **HIGH PRIORITY**: Strongly recommend fixing
   - **SUGGESTION**: Optional improvement
4. **Revisions** may be requested
5. **Approval** and merge when all requirements met

### Review Timeline

- **Initial Review**: Within 24 hours
- **Follow-up Review**: Within 8 hours
- **Final Approval**: Within 48 hours of PR creation

### Automated Checks (Must Pass)

Every PR must pass:

- ✓ Code formatting (`gofmt`, `goimports`)
- ✓ Linting (`golangci-lint`)
- ✓ Tests with race detector (`go test -race`)
- ✓ Coverage threshold (80% minimum)
- ✓ Security scan (`gosec`)
- ✓ Vulnerability scan (`govulncheck`)

### Security Review

PRs are automatically checked for:

- No hardcoded credentials or secrets
- All external inputs validated
- Proper file permissions (0600 for sensitive files)
- No command injection vulnerabilities
- Safe error messages (no information leakage)
- Proper concurrency control (no race conditions)

---

## Quick Reference

### Common Commands

```bash
# Format code
make fmt

# Run linter
make lint

# Run tests
make test

# Check coverage
make coverage

# Security scan
make security

# Run ALL checks (do this before PR)
make pre-commit

# Run what CI will run
make ci

# Build binary
make build
```

### Development Workflow

```bash
# 1. Create feature branch
git checkout -b feature/my-feature

# 2. Make changes and write tests
# ... edit files ...

# 3. Run checks locally
make pre-commit

# 4. Commit changes
git add .
git commit -m "Add feature: description"

# 5. Push and create PR
git push origin feature/my-feature
# Then open PR on GitHub
```

### Project Structure

```
kubectx-timeout/
├── cmd/                    # Application entry points
│   └── kubectx-timeout/
│       └── main.go
├── internal/               # Private application code
│   ├── config.go          # Configuration loading & validation
│   ├── daemon.go          # Core daemon logic
│   ├── paths.go           # XDG path management
│   ├── state.go           # State file management
│   ├── switcher.go        # Context switching
│   └── tracker.go         # Activity tracking & shell integration
├── examples/              # Example configurations
│   ├── config.example.yaml
│   ├── config.minimal.yaml
│   └── com.kubectx-timeout.plist
├── scripts/               # Build and install scripts
├── .github/               # CI/CD and PR templates
├── Makefile               # Common development tasks
└── go.mod                 # Go module dependencies
```

### File Locations (XDG Compliant)

The project follows the [XDG Base Directory Specification](https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html):

**Configuration:**
- Default: `~/.config/kubectx-timeout/config.yaml`
- Override: `$XDG_CONFIG_HOME/kubectx-timeout/config.yaml`

**State & Logs:**
- Default: `~/.local/state/kubectx-timeout/`
- Override: `$XDG_STATE_HOME/kubectx-timeout/`
  - `state.json` - Activity tracking
  - `daemon.log` - Daemon logs
  - `daemon.stdout.log` - launchd stdout
  - `daemon.stderr.log` - launchd stderr

See [DEVELOPMENT.md](DEVELOPMENT.md#xdg-base-directory-compliance) for implementation details.

---

## Getting Help

- **Documentation**: See [DEVELOPMENT.md](DEVELOPMENT.md) for detailed technical guidelines
- **Issues**: Check existing issues or create a new one
- **Questions**: Open a discussion or issue tagged with `question`

---

## Additional Resources

- [DEVELOPMENT.md](DEVELOPMENT.md) - Detailed development guidelines, security practices, and Go best practices
- [PR_CHECKLIST.md](PR_CHECKLIST.md) - Quick reference for code reviewers
- [Makefile](Makefile) - All available make commands

---

**Thank you for contributing to kubectx-timeout!**
