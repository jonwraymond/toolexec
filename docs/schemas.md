# Schemas and Contracts

toolexec relies on **toolfoundation/model.Tool** as the canonical tool schema.
It does **not** introduce new input/output JSON Schema formats; instead it
executes tools and validates against the schemas defined on each `model.Tool`.

This page documents:
- The canonical tool schema fields and constraints (summary)
- Input/Output schema requirements as enforced by the runner
- JSON Schema dialect support and limitations
- Execution payload contracts (RunResult, StreamEvent, ChainStep)
- Recommended schema patterns

For full schema details, see **toolfoundation**’s schema docs.

## Tool schema fields/constraints (summary)

`model.Tool` embeds the MCP SDK `mcp.Tool` and adds stack extensions.

### Core MCP fields

| Field | Required | Notes |
|-------|----------|-------|
| `name` | Yes | 1–128 chars, `[A-Za-z0-9_.-]` only |
| `description` | No | Human-readable description |
| `inputSchema` | Yes | JSON Schema object for tool parameters |
| `outputSchema` | No | JSON Schema object for structured output |
| `title` | No | Display label |
| `annotations` | No | Hints (readOnly, idempotent, destructive, openWorld) |
| `_meta` | No | Arbitrary metadata |
| `icons` | No | Optional icon assets |

### Extensions

| Field | Required | Notes |
|-------|----------|-------|
| `namespace` | No | Tool ID is `namespace:name` when set |
| `version` | No | SemVer (`v1.2.3` or `1.2.3`) |
| `tags` | No | Normalized tags for discovery |

## InputSchema / OutputSchema requirements

- **InputSchema is required.** A tool without `inputSchema` is invalid.
- **OutputSchema is optional.** If omitted, output validation is skipped.
- Validation in `run` uses `model.SchemaValidator`:
  - Input validation runs before execution.
  - Output validation runs after execution when `OutputSchema` is present.

## Supported dialects and limitations

Inherited from toolfoundation:

- Default dialect: **JSON Schema 2020-12** (when `$schema` is absent)
- Supported: **2020-12** and **draft-07**
- External `$ref` resolution is **disabled** (no network I/O)
- `format` is treated as annotation (not enforced)

## Execution payload contracts

### RunResult (`run.RunResult`)

Normalized result of executing a tool:

| Field | Type | Notes |
|-------|------|-------|
| `tool` | `model.Tool` | Resolved tool definition |
| `backend` | `model.ToolBackend` | Backend used for execution |
| `structured` | `any` | Normalized result value |
| `mcpResult` | `*mcp.CallToolResult` | Raw MCP result when backend is MCP |

### StreamEvent (`run.StreamEvent`)

Transport-agnostic streaming envelope:

| Field | Type | Notes |
|-------|------|-------|
| `kind` | string | `progress`, `chunk`, `done`, `error` |
| `toolId` | string | Canonical tool ID |
| `data` | any | Event payload (progress/chunk details) |

### ChainStep (`run.ChainStep`)

| Field | Type | Notes |
|-------|------|-------|
| `toolId` | string | Canonical tool ID |
| `args` | map | Tool arguments |
| `usePrevious` | bool | Inject prior result into `args["previous"]` |

## Recommended “no parameters” schema

Strict MCP-recommended schema:

```json
{
  "type": "object",
  "additionalProperties": false
}
```

Less strict variant:

```json
{
  "type": "object"
}
```

## Example schema patterns

### Required string property

```json
{
  "type": "object",
  "properties": {
    "path": {"type": "string", "description": "File path"}
  },
  "required": ["path"],
  "additionalProperties": false
}
```

### Optional enum with default

```json
{
  "type": "object",
  "properties": {
    "encoding": {"type": "string", "enum": ["utf8", "ascii"], "default": "utf8"}
  },
  "additionalProperties": false
}
```

### Array of objects

```json
{
  "type": "object",
  "properties": {
    "items": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "id": {"type": "string"},
          "value": {"type": "number"}
        },
        "required": ["id"],
        "additionalProperties": false
      }
    }
  },
  "additionalProperties": false
}
```

### One-of variants

```json
{
  "type": "object",
  "properties": {
    "mode": {
      "oneOf": [
        {"type": "string", "enum": ["fast", "safe"]},
        {"type": "number", "minimum": 1, "maximum": 10}
      ]
    }
  }
}
```

## Links

- [Architecture](architecture.md)
- [Design notes](design-notes.md)
- [User journey](user-journey.md)
- [toolfoundation schemas](https://github.com/jonwraymond/toolfoundation/blob/main/docs/schemas.md)
