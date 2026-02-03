# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## 1.0.0 (2026-02-03)


### Features

* align docs, exec facade, and examples ([a4cafd3](https://github.com/jonwraymond/toolexec/commit/a4cafd3da8d49b820402b7b7df09683304eb1380))
* **backend:** extract backend management to toolexec/backend (PRD-143) ([c98adf3](https://github.com/jonwraymond/toolexec/commit/c98adf392fa5635093f2fa51ab0b95c63cbdb7f1))
* **code:** migrate toolcode to toolexec/code package (PRD-142) ([d3045b8](https://github.com/jonwraymond/toolexec/commit/d3045b809bbbe1985cef92553b8c78e6e085441b))
* initial repository structure ([d521d97](https://github.com/jonwraymond/toolexec/commit/d521d97dffb82e77074497ba1a607d33c577dff3))
* **run:** migrate toolrun to toolexec/run package (PRD-140) ([f6aa10d](https://github.com/jonwraymond/toolexec/commit/f6aa10d7fe9ae7a42edb001feed4507f2ca6aa0a))
* **runtime:** migrate toolruntime to toolexec/runtime package (PRD-141) ([d580a6e](https://github.com/jonwraymond/toolexec/commit/d580a6e3f2687006255b37e27117d22431813e39))


### Bug Fixes

* **deps:** remove local replace directives for module resolution ([4f2d597](https://github.com/jonwraymond/toolexec/commit/4f2d5978906622ad25da5ebd938cab0b602435cc))


### Code Refactoring

* **runtime:** interface-only core ([#12](https://github.com/jonwraymond/toolexec/issues/12)) ([d28d3ca](https://github.com/jonwraymond/toolexec/commit/d28d3cab71e0c1ca3011759765437e40e5330a2f))


### Documentation

* add architecture improvement plan ([45dffa6](https://github.com/jonwraymond/toolexec/commit/45dffa67e9807838fd68317950a2c04af039f169))
* add mkdocs config ([bc50353](https://github.com/jonwraymond/toolexec/commit/bc5035368a52ff30cfd43fbbcafe8c0cfa6dd4f0))
* **toolexec:** align runtime docs and deps ([bc14ad2](https://github.com/jonwraymond/toolexec/commit/bc14ad21fff4dda16ed4befbb97ed978fa2a0a21))
* **toolexec:** fix user journey examples link ([1d6b708](https://github.com/jonwraymond/toolexec/commit/1d6b7081b8895c1c0ae3d4cb610f699e0f33dfda))

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
