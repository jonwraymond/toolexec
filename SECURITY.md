# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |

## Reporting a Vulnerability

If you discover a security vulnerability in toolexec, please report it responsibly.

### How to Report

1. **Do NOT open a public GitHub issue** for security vulnerabilities.
2. **Email the maintainer** with details of the vulnerability:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fixes (optional)
3. **Allow time for response** — we aim to respond within 48 hours and provide a fix timeline within 7 days.

### What to Expect

- **Acknowledgment**: We will acknowledge receipt within 48 hours.
- **Assessment**: We will assess severity and impact.
- **Fix Timeline**: For confirmed vulnerabilities, we will share an estimated fix timeline.
- **Disclosure**: We will coordinate public disclosure after a fix is available.
- **Credit**: With your permission, we will credit you in the advisory.

## Security Measures

### Input Validation

- Tool inputs are validated against JSON Schema
- Output validation is enforced when OutputSchema is present
- Backend resolution is deterministic and explicit

### Runtime Isolation

- Sandboxed runtimes (Docker, gVisor, WASM) are supported
- Security profiles determine runtime isolation levels
- Unsafe execution requires explicit opt-in

### Dependencies

- Dependencies are scanned with `govulncheck`
- Security scanning runs in CI via `gosec`

## Scope

This policy applies to:

- `run` (execution pipeline and schema validation)
- `backend` (backend registry and resolution)
- `runtime` (sandboxed execution)
- `code` (tool orchestration and limits)
- `exec` (unified facade)

## Out of Scope

- Vulnerabilities in upstream dependencies (report to maintainers)
- Example code intended for demonstration only
- Theoretical issues without demonstrated impact
