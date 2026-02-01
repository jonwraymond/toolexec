# Contributing to toolexec

Thank you for your interest in contributing to toolexec.

## Development Setup

### Prerequisites

- Go 1.25 or later
- golangci-lint (for linting)
- gosec (for security scanning)

### Clone and Build

```bash
git clone https://github.com/jonwraymond/toolexec.git
cd toolexec
go mod download
go build ./...
```

### Run Tests

```bash
go test -race ./...
```

### Run Linting

```bash
golangci-lint run
```

### Run Security Scans

```bash
gosec ./...
govulncheck ./...
```

### Run Examples

```bash
go run ./examples/basic
go run ./examples/chain
go run ./examples/discovery
go run ./examples/streaming
go run ./examples/runtime
go run ./examples/full
```

## Code Quality

We follow these quality standards:

- All exported types and functions must have GoDoc comments
- Tests are required for new functionality
- Maintain or improve test coverage
- Avoid breaking API changes without a major version bump

## Commit Conventions

Use conventional commits:

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

## Pull Request Process

1. **Fork the repository** and create your branch from `main`.
2. **Write tests** for any new functionality. Maintain or improve test coverage.
3. **Run the full test suite** to ensure your changes don't break existing functionality:
   ```bash
   go test -race ./...
   golangci-lint run
   ```
4. **Update documentation** if you're changing public APIs or behavior.
5. **Use conventional commit messages** for your commits.
6. **Submit your PR** with a clear description of the changes.

## Code Style

- Follow standard Go conventions and `gofmt` formatting
- Add doc comments to all exported types and functions
- Keep interfaces small and focused
- Ensure execution ordering is deterministic where required

## Package Guidelines

### run

- Keep the execution pipeline deterministic and observable
- Validate input and output schemas consistently
- Preserve structured results for chaining

### backend

- Backends must be side-effect free during resolution
- Prefer explicit configuration over hidden defaults
- Ensure backend selection is deterministic

### runtime

- Security profiles must be explicit and documented
- Sandboxed runtimes should fail closed on configuration errors
- Avoid hidden network or filesystem access

### code

- Enforce execution limits (time, memory, tools) by default
- Surface tool call errors with clear context
- Keep orchestration results deterministic

### exec

- Treat the facade as a thin composition layer
- Avoid leaking internal types into the public API
- Keep defaults safe and minimal

## Questions?

Open a GitHub issue if you need clarification.
