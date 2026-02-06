# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.3](https://github.com/jonwraymond/toolexec/compare/v0.2.2...v0.2.3) (2026-02-06)


### Documentation

* update version matrix ([f79b3df](https://github.com/jonwraymond/toolexec/commit/f79b3df566bb8420403aa033239595976804d894))

## [0.2.2](https://github.com/jonwraymond/toolexec/compare/v0.2.1...v0.2.2) (2026-02-03)


### Bug Fixes

* **runtime:** harden builder and __out extraction ([#16](https://github.com/jonwraymond/toolexec/issues/16)) ([92d8bbf](https://github.com/jonwraymond/toolexec/commit/92d8bbf9956ccf965b3a218aadd776bed7d178c7))


### Documentation

* update version matrix ([#14](https://github.com/jonwraymond/toolexec/issues/14)) ([deb4291](https://github.com/jonwraymond/toolexec/commit/deb429146e667d3fbae9248b88cdf208f780ab7e))

## [Unreleased]

### Added
- **exec package**: Unified facade combining discovery + execution into single API
  - `Exec.RunTool()` - Execute single tools
  - `Exec.RunChain()` - Execute tool sequences with result passing
  - `Exec.SearchTools()` - Search tool index
  - `Exec.GetToolDoc()` - Get tool documentation
  - Simple `Handler` function type for local tool registration
- **Examples**: 6 runnable examples demonstrating different use cases
  - `examples/basic/` - Simple tool execution
  - `examples/chain/` - Sequential tool chaining
  - `examples/discovery/` - Search and execute workflow
  - `examples/streaming/` - Streaming execution events
  - `examples/runtime/` - Security profile configuration
  - `examples/full/` - Complete integration example
- **Example tests**: pkg.go.dev documentation examples for backend/, code/, runtime/
- **Test coverage improvements**:
  - backend/: 67% → 99%
  - backend/local/: 63% → 100%
  - code/: 73% → 96%
  - runtime/gateway/proxy/: 68% → 90%

### Changed
- Updated README with quick start guide and package overview
- Updated docs/index.md with unified facade documentation
- Updated docs/design-notes.md with architecture diagram

## [0.1.0] - Initial Release

### Added
- Initial repository structure
