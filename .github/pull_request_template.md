## Description

<!-- Provide a brief description of the changes in this PR -->

## Beads Issue

Closes: `kubectx-timeout-XXX`

<!-- Link to the Beads issue this PR addresses -->

## Type of Change

- [ ] Bug fix (non-breaking change that fixes an issue)
- [ ] New feature (non-breaking change that adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Documentation update
- [ ] Refactoring (no functional changes)
- [ ] Performance improvement
- [ ] Test coverage improvement

## Testing

### Unit Tests
- [ ] Unit tests added for new/changed code
- [ ] All unit tests pass locally
- [ ] Code coverage meets minimum threshold (80% overall, 90% core logic)

### Integration Tests
- [ ] Integration tests added if cross-component changes
- [ ] All integration tests pass locally

### Manual Testing
- [ ] Manual testing performed (describe below)

**Manual Testing Details:**
<!-- Describe what you tested manually and the results -->

## Code Quality Checklist

### Required (Must Pass)
- [ ] Code formatted with `gofmt` and `goimports`
- [ ] All linter checks pass (`golangci-lint run ./...`)
- [ ] No security issues (`gosec ./...`)
- [ ] Self-review performed (read through your own code)
- [ ] No commented-out code blocks
- [ ] Git commit messages are clear and descriptive

### Code Review Standards
- [ ] Functions have single, clear responsibility
- [ ] All errors properly handled (no swallowed errors)
- [ ] Public APIs have godoc comments
- [ ] Complex logic has explanatory comments
- [ ] No hardcoded credentials or secrets
- [ ] File operations use appropriate permissions
- [ ] Input validation for external inputs
- [ ] Race conditions handled with proper locking

## Documentation

- [ ] README.md updated (if user-facing changes)
- [ ] Configuration documentation updated (if config changes)
- [ ] API documentation updated (if API changes)
- [ ] Comments added for complex/non-obvious code

## Security Considerations

<!-- Describe any security implications of this change -->
<!-- Check all that apply: -->
- [ ] No security implications
- [ ] Input validation added/modified
- [ ] Authentication/authorization changes
- [ ] Credential handling changes
- [ ] File permission changes
- [ ] Command execution changes

**Security Review Details:**
<!-- If security implications exist, describe them here -->

## Performance Considerations

<!-- Describe any performance implications of this change -->
- [ ] No performance implications
- [ ] Performance improved
- [ ] Performance impact acceptable (explain below)

**Performance Details:**
<!-- If performance implications exist, describe them here -->

## Breaking Changes

<!-- If this is a breaking change, describe: -->
<!-- 1. What breaks and why -->
<!-- 2. Migration path for users -->
<!-- 3. Backward compatibility plan -->

- [ ] No breaking changes
- [ ] Breaking changes documented above

## Screenshots / Logs

<!-- Add screenshots, logs, or output if helpful for review -->

```
[Paste relevant terminal output or logs here]
```

## Reviewer Notes

<!-- Any specific areas you want reviewers to focus on? -->
<!-- Any concerns or questions you have about this implementation? -->

---

## Pre-Review Checklist (for submitter)

Before requesting review, verify:
- [ ] All CI checks pass
- [ ] Branch is up to date with main
- [ ] No merge conflicts
- [ ] PR description is clear and complete
- [ ] All checklist items above are addressed

---

**Review Priority:**
- [ ] Standard review (48h turnaround)
- [ ] Urgent review needed (explain why below)

**Senior Review Required:** (auto-detected by changes)
- [ ] Security-sensitive changes
- [ ] Architecture changes
- [ ] Performance-critical code
- [ ] High-risk areas (launchd, kubectl execution, state management)
