# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- GoReleaser configuration for automated multi-platform builds
- Homebrew tap and formula for easy macOS installation
- GitHub Actions workflow for automated releases
- Release automation script (scripts/release.sh) for version management
- Docker image support for containerized environments
- Linux package support (deb, rpm, apk)
- CHANGELOG.md for tracking version history
- Versioning documentation
- **File system monitoring with fswatch** for detecting context switches from any tool
  - Monitors `~/.kube/config` for modifications using FSEvents API on macOS
  - Detects context switches from IDE plugins, GUI tools, kubectx, and manual edits
  - Runs in separate goroutine alongside periodic timeout checking
  - Graceful degradation when fswatch is not installed
  - Automatic activity recording when context changes are detected
- Comprehensive documentation for fswatch monitoring feature

### Changed
-

### Fixed
-

## [1.0.0] - TBD

### Added
- Initial release
- Core daemon functionality for monitoring kubectl context activity
- Automatic context switching after configurable timeout periods
- XDG Base Directory specification compliance
- Shell integration for bash, zsh, and fish
- Configuration management with YAML files
- Activity tracking and state management
- Safe context switching with validation
- Security hardening and comprehensive testing
- CI/CD pipeline with GitHub Actions
- launchd integration for macOS daemon management
- Comprehensive documentation and examples

### Security
- Command injection prevention
- Path traversal protection
- Configuration validation
- Secure file permissions

[unreleased]: https://github.com/mrf/kubectx-timeout/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/mrf/kubectx-timeout/releases/tag/v1.0.0
