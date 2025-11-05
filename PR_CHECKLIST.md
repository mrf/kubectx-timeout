# Pull Request Review Checklist

Quick reference for code reviewers. See [CONTRIBUTING.md](CONTRIBUTING.md) and [DEVELOPMENT.md](DEVELOPMENT.md) for detailed guidelines.

---

## Pre-Review: Automated Checks

- [ ] All CI/CD checks passing (gofmt, golangci-lint, tests)
- [ ] Code coverage meets minimum (80% overall, 90% core logic)
- [ ] No security vulnerabilities (gosec scan clean)
- [ ] Dependencies up to date (no known CVEs)

---

## Level 1: Blockers (MUST Fix Before Merge)

### Security
- [ ] No hardcoded credentials, API keys, or secrets
- [ ] All external inputs validated (file paths, context names, config)
- [ ] File operations use restrictive permissions (0600 for state files)
- [ ] No command injection vulnerabilities (no shell commands with user input)
- [ ] Error messages don't leak sensitive system information
- [ ] Race conditions handled with proper locking

### Correctness
- [ ] Code fulfills requirements from linked Beads issue
- [ ] Edge cases handled (missing files, malformed input, timeouts)
- [ ] Error handling complete (no swallowed errors without justification)
- [ ] No unhandled promise rejections or goroutine leaks

### Testing
- [ ] Unit tests cover new/changed logic
- [ ] Tests cover error conditions and edge cases
- [ ] Integration tests for cross-component changes
- [ ] Tests are deterministic (no flaky tests)
- [ ] Tests use mocks for external dependencies (kubectl, filesystem)

### Breaking Changes
- [ ] No breaking API changes without migration plan
- [ ] Config schema changes documented and backward compatible

---

## Level 2: High Priority (Strongly Recommend Fixing)

### Architecture
- [ ] Functions have single, clear responsibility (SRP)
- [ ] No significant code duplication (DRY where appropriate)
- [ ] Abstractions don't leak implementation details
- [ ] Proper separation of concerns

### Performance
- [ ] No N+1 patterns or inefficient algorithms
- [ ] Resources properly cleaned up (defer for file handles)
- [ ] Goroutines properly managed (no leaks)
- [ ] Context cancellation properly propagated

### Error Handling
- [ ] Errors provide sufficient context for debugging
- [ ] Error wrapping uses `fmt.Errorf` with `%w`
- [ ] Errors not silently ignored

### Go Best Practices
- [ ] Idiomatic Go patterns used
- [ ] Interfaces small and focused
- [ ] Package organization follows project structure
- [ ] No global mutable state

---

## Level 3: Medium Priority (Consider for Follow-up)

### Readability
- [ ] Variable/function names clear and descriptive
- [ ] Complex logic broken into smaller functions
- [ ] Magic numbers replaced with named constants
- [ ] Cyclomatic complexity < 10 per function
- [ ] Function length reasonable (< 50 lines ideal)

### Documentation
- [ ] Public APIs have godoc comments
- [ ] Complex algorithms have explanatory comments
- [ ] README updated for user-facing changes
- [ ] Configuration changes documented

### Code Style
- [ ] Code formatted with gofmt/goimports
- [ ] Comments start with name of thing being documented
- [ ] No commented-out code blocks

---

## Project-Specific Checks

### Configuration (kubectx-timeout-6)
- [ ] YAML parsing errors handled gracefully
- [ ] Missing config file has sensible defaults
- [ ] Timeout durations validated (not negative/zero)
- [ ] Context names validated against kubectl
- [ ] Config reload is thread-safe

### State Tracking (kubectx-timeout-7)
- [ ] Concurrent state file access properly locked
- [ ] Timestamps use UTC consistently
- [ ] State file corruption handled
- [ ] File permissions restrictive (0600)

### Daemon Logic (kubectx-timeout-8)
- [ ] Graceful shutdown on signals (SIGTERM, SIGINT)
- [ ] SIGHUP triggers config reload
- [ ] Ticker properly stopped (no resource leak)
- [ ] Panics don't crash daemon
- [ ] Context used for cancellation

### Context Switching (kubectx-timeout-9)
- [ ] Target context existence validated
- [ ] No command injection via context name
- [ ] Not switching during active kubectl operations
- [ ] Kubectl errors properly handled
- [ ] Retry logic for transient failures

### Safety Checks (kubectx-timeout-13)
- [ ] Active kubectl processes detected before switch
- [ ] Network failures handled gracefully
- [ ] Dry-run mode available
- [ ] Comprehensive logging for debugging

---

## Review Process

### Time Allocation (Total ~45 min)
1. Architecture & Design (15 min)
2. Security & Safety (10 min)
3. Testing Adequacy (10 min)
4. Maintainability (10 min)

### Senior Review Required For
- [ ] Security-sensitive changes (auth, credentials, permissions)
- [ ] Architecture changes (new packages, major refactors)
- [ ] Performance-critical code (daemon loop, signal handling)
- [ ] High-risk areas (launchd, kubectl execution, state management)

### Approval Decision
- [ ] **APPROVED**: All blockers resolved, high-priority items addressed
- [ ] **APPROVED WITH SUGGESTIONS**: Minor improvements noted for future
- [ ] **NEEDS REVISION**: Blockers or critical issues remain

---

## Comment Template for Reviewers

### For Blockers
```
BLOCKER: [Issue description]

This creates a [security risk / critical bug / test gap] because [explanation].

Suggested fix:
[Concrete code example or approach]
```

### For High Priority
```
HIGH PRIORITY: [Issue description]

This violates [principle] and could lead to [consequence].

Consider refactoring to:
[Suggested approach]
```

### For Medium Priority
```
SUGGESTION: [Issue description]

This could be improved by [suggestion] for better [readability/maintainability].
```

### For Positive Feedback
```
GOOD: [What was done well]

[Why this is a good practice]
```

---

## Disagreement Resolution

1. **Discuss in PR comments** - Explain rationales and concerns
2. **Synchronous discussion** - Call/chat if comments aren't resolving
3. **Escalate** - Tag senior engineer if still unresolved

**Principles:**
- Assume good intent
- Focus on code, not person
- Use "we" language
- Provide specific suggestions
- Accept "good enough" over perfect

---

## Quick Commands

### Run all checks locally
```bash
make check
```

### Format code
```bash
gofmt -s -w .
goimports -w .
```

### Lint
```bash
golangci-lint run ./...
```

### Test with coverage
```bash
go test -race -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | grep total
```

### Security scan
```bash
gosec -quiet ./...
```

---

**See [CONTRIBUTING.md](CONTRIBUTING.md) and [DEVELOPMENT.md](DEVELOPMENT.md) for complete guidelines.**
