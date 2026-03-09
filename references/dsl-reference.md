# ios-app-manager DSL reference

## CLI entrypoints

Query:
```bash
ios-app-manager q 'operation(params) { fields }'
```

Mutation:
```bash
ios-app-manager m 'operation(params)'
```

Output format for both:
- `--format pretty` (default)
- `--format compact`

## Expression grammar

Canonical form:
```text
operation(key=value, key2="quoted value") { field1 field2 }
```

Rules:
- Exactly one DSL expression per command invocation.
- `operation` and parameter keys are identifiers.
- Identifier charset:
  - first char: letter or `_`
  - subsequent chars: letters, digits, `_`, `-`, `.`
- Parameters are `key=value`, comma-separated.
- Duplicate keys are invalid.
- Values:
  - unquoted: accepted as raw non-empty string
  - quoted: `'...'` or `"..."`, with escapes (`\n`, `\r`, `\t`, `\\`, `\'`, `\"`)
- Field projection block `{ ... }` is optional.
- If `{}` is present, it cannot be empty.

## Available query operations

These operations are registered in the current query executor:

| Operation | Intent | Closest CLI command |
|---|---|---|
| `summary` | Project summary/status query | `status` |
| `modules` | Module listing/filtering | `module list` |
| `get` | Entity lookup | n/a (DSL-only placeholder) |
| `deps` | Dependency inspection | `dep list` |
| `config` | Config inspection | `--config` workflows |
| `entitlements` | Entitlements inspection | `entitlements list` |

Examples:
```bash
ios-app-manager q 'summary() { message kind operation }'
ios-app-manager q 'modules(type=feature) { operation params }'
ios-app-manager q 'deps(module=Auth) { operation params fields }'
ios-app-manager q 'entitlements(path="App/App.entitlements") { * }'
```

## Available mutation operations

These operations are registered in the current mutation executor:

| Operation | Intent | Closest CLI command |
|---|---|---|
| `init` | Project initialization | `init` |
| `create_module` | Module creation | `module create` |
| `delete_module` | Module deletion | `module delete` |
| `add_dep` | Add internal dependency | `dep add` |
| `remove_dep` | Remove internal dependency | `dep remove` |
| `add_entitlement` | Add entitlement value | _(no CLI equivalent yet)_ |

Examples:
```bash
ios-app-manager m 'init(config=ios-app-manager.json,output=.)'
ios-app-manager m 'create_module(name=Auth,type=feature)'
ios-app-manager m 'add_dep(module=Auth,depends_on=CoreKit)'
ios-app-manager m 'add_entitlement(key=aps-environment,value=development)'
```

## Field projection syntax

Projection is optional for both query and mutation expressions. When used:

- Space-separated fields:
  - `{ message kind operation }`
- Comma-separated fields:
  - `{ message, kind, operation }`
- Mixed spacing/commas is valid.
- `*` is valid as wildcard.

Current scaffold response shape:
- `message`
- `kind`
- `operation`
- `params`
- `fields`

Suggested projection presets:

| Preset | Expansion | Use |
|---|---|---|
| `overview` | `message kind operation` | quick sanity checks |
| `inputs` | `operation params fields` | verify parsing and argument passing |
| `full` | `message kind operation params fields` | debug traces |

Examples:
```bash
ios-app-manager q 'modules(type=feature) { message kind operation }'
ios-app-manager q --format compact 'get(module=Auth) { operation, params, fields }'
ios-app-manager m --format compact 'create_module(name=Auth,type=feature) { full }'
```

## Common workflow examples

### 1) Project overview (query flow)

```bash
ios-app-manager q 'summary() { message kind operation }'
ios-app-manager q 'config() { operation params }'
```

### 2) Module planning via DSL placeholders

```bash
ios-app-manager m 'create_module(name=Feed,type=feature)'
ios-app-manager q 'modules(type=feature) { operation params }'
```

### 3) Dependency planning and verification

```bash
ios-app-manager m 'add_dep(module=Feed,depends_on=CoreKit)'
ios-app-manager q 'deps(module=Feed) { operation params fields }'
```

### 4) Entitlements planning

```bash
ios-app-manager m 'add_entitlement(key=aps-environment,value=development)'
ios-app-manager q 'entitlements() { operation params }'
```

### 5) Parser/format debugging

```bash
ios-app-manager q --format compact 'deps(module="resolve(name=Auth, level=2)") { * }'
ios-app-manager m --format compact 'init(config=ios-app-manager.json,output=.)'
```

## Current scaffold behavior

All registered DSL operations currently return placeholder data (`"message": "not implemented"`).

Example:

```json
{
  "message": "not implemented",
  "kind": "query",
  "operation": "modules",
  "params": {
    "type": "feature"
  },
  "fields": [
    "operation",
    "params"
  ]
}
```

Use imperative CLI commands from [`cli-reference.md`](cli-reference.md) for real project changes until DSL handlers are implemented.
