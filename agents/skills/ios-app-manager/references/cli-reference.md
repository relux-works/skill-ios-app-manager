# ios-app-manager CLI reference

## Binary

```bash
ios-app-manager [command] [flags]
```

## Global flags

| Flag | Description | Default |
|---|---|---|
| `-c, --config <path>` | Project config JSON path | `ios-app-manager.json` |
| `-v, --verbose` | Verbose output toggle | `false` |
| `--version` | Print version and exit | - |
| `-h, --help` | Help output | - |

## Command index

| Command | Description | Example |
|---|---|---|
| `init --config <path> --output <dir> [--force]` | Scaffold a new/updated project tree | `ios-app-manager init --config ios-app-manager.json --output .` |
| `module create <name> --type <type>` | Create a module package layout | `ios-app-manager module create Auth --type feature` |
| `module list` | List modules and metadata | `ios-app-manager module list` |
| `module delete <name> [--force]` | Delete module packages and references | `ios-app-manager module delete Auth --force` |
| `dep add <module> --depends-on <other>` | Add internal module dependency | `ios-app-manager dep add Auth --depends-on CoreKit` |
| `dep add-external --url <git_url> --version <ver> [--module <target>]` | Add external Swift package dependency | `ios-app-manager dep add-external --url https://github.com/apple/swift-collections.git --version 'branch: "main"' --module Auth` |
| `dep remove <module> --depends-on <other>` | Remove internal module dependency | `ios-app-manager dep remove Auth --depends-on CoreKit` |
| `dep remove-external --package <name>` | Remove external Swift package dependency | `ios-app-manager dep remove-external --package swift-collections` |
| `dep list [<module>]` | List internal and external dependencies | `ios-app-manager dep list Auth` |
| `entitlements add <key> [--value <val>]` | Add/update entitlement key | `ios-app-manager entitlements add aps-environment --value development` |
| `entitlements remove <key>` | Remove entitlement key | `ios-app-manager entitlements remove aps-environment` |
| `entitlements list` | Print effective entitlement values | `ios-app-manager entitlements list` |
| `generate makefile` | Generate or update `Makefile` | `ios-app-manager generate makefile` |
| `clean [--deep] [--kill-xcode]` | Clean local/global build artifacts | `ios-app-manager clean --deep` |
| `push send --token <token> [--env dev\|prod] [--payload <file>]` | Send APNs push using project credentials | `ios-app-manager push send --token "$TOKEN" --env dev` |
| `push token` | Print latest device token from logs/fallback file | `ios-app-manager push token` |
| `status` | Project status placeholder command | `ios-app-manager status` |
| `q '<query>'` | Run DSL query expression | `ios-app-manager q 'modules(type=feature) { operation params }'` |
| `m '<mutation>'` | Run DSL mutation expression | `ios-app-manager m 'create_module(name=Auth,type=feature)'` |

## Command reference

### `init`

Syntax:
```bash
ios-app-manager init --config <path> --output <dir> [--force]
```

Description:
- Loads config and scaffolds project files to the output directory.
- Without `--force`, existing scaffold files are protected from overwrite.

Flags:
- `--config <path>`: Config file path.
- `--output <dir>`: Target directory (default `.`).
- `--force`: Overwrite scaffold files if they exist.

Examples:
```bash
ios-app-manager init --config ios-app-manager.json --output .
ios-app-manager init --config ios-app-manager.json --output ./DemoApp --force
```

### `module create`

Syntax:
```bash
ios-app-manager module create <name> --type <feature|kit|shared|ui|utility>
```

Description:
- Creates module packages/files according to Relux conventions.
- Module name is validated (PascalCase).

Flags:
- `--type <type>`: Required module kind.
- `--config <path>`: Optional command-level config override.

Example:
```bash
ios-app-manager module create Auth --type feature
```

### `module list`

Syntax:
```bash
ios-app-manager module list
```

Description:
- Prints module table with name, type, package names, and dependency count.

Flags:
- `--config <path>`: Optional command-level config override.

Example:
```bash
ios-app-manager module list
```

### `module delete`

Syntax:
```bash
ios-app-manager module delete <name> [--force]
```

Description:
- Deletes module packages and cleans manifest references.
- Interactive confirmation is required unless `--force` is set.

Flags:
- `--force`: Skip confirmation prompt.
- `--config <path>`: Optional command-level config override.

Examples:
```bash
ios-app-manager module delete Auth
ios-app-manager module delete Auth --force
```

### `dep add`

Syntax:
```bash
ios-app-manager dep add <module> --depends-on <other>
```

Description:
- Adds an internal dependency from `<module>` to `<other>`.
- Detects invalid dependency changes (for example, circular dependencies).

Flags:
- `--depends-on <other>`: Required dependency module.
- `--config <path>`: Optional command-level config override.

Example:
```bash
ios-app-manager dep add Auth --depends-on CoreKit
```

### `dep add-external`

Syntax:
```bash
ios-app-manager dep add-external --url <git_url> --version <ver> [--module <target>]
```

Description:
- Adds an external Swift package dependency.
- Optional `--module` links the package to a specific module package manifest.

Flags:
- `--url <git_url>`: Required package repository URL.
- `--version <ver>`: Required version requirement.
- `--module <target>`: Optional target module scope.
- `--config <path>`: Optional command-level config override.

Example:
```bash
ios-app-manager dep add-external \
  --url https://github.com/apple/swift-collections.git \
  --version 'branch: "main"' \
  --module Auth
```

### `dep remove`

Syntax:
```bash
ios-app-manager dep remove <module> --depends-on <other>
```

Description:
- Removes an internal dependency relation.

Flags:
- `--depends-on <other>`: Required dependency module.
- `--config <path>`: Optional command-level config override.

Example:
```bash
ios-app-manager dep remove Auth --depends-on CoreKit
```

### `dep remove-external`

Syntax:
```bash
ios-app-manager dep remove-external --package <name>
```

Description:
- Removes an external package from project/module manifests.

Flags:
- `--package <name>`: Required package name.
- `--config <path>`: Optional command-level config override.

Example:
```bash
ios-app-manager dep remove-external --package swift-collections
```

### `dep list`

Syntax:
```bash
ios-app-manager dep list [<module>]
```

Description:
- Prints internal dependency table and external package table.
- With `<module>`, output is scoped to one module and root/module-level external deps.

Flags:
- `--config <path>`: Optional command-level config override.

Examples:
```bash
ios-app-manager dep list
ios-app-manager dep list Auth
```

### `entitlements add`

Syntax:
```bash
ios-app-manager entitlements add <key> [--value <val>]
```

Description:
- Adds or updates entitlement key.
- Value can be omitted for boolean-like keys; arrays can be passed as comma-separated values.

Flags:
- `--value <val>`: Optional value.
- `-p, --path <plist>`: Optional explicit entitlements file path.
- `--config <path>`: Used when `--path` is not provided.

Examples:
```bash
ios-app-manager entitlements add aps-environment --value development
ios-app-manager entitlements add healthkit
```

### `entitlements remove`

Syntax:
```bash
ios-app-manager entitlements remove <key>
```

Description:
- Removes an entitlement key.

Flags:
- `-p, --path <plist>`: Optional explicit entitlements file path.
- `--config <path>`: Used when `--path` is not provided.

Example:
```bash
ios-app-manager entitlements remove aps-environment
```

### `entitlements list`

Syntax:
```bash
ios-app-manager entitlements list
```

Description:
- Lists entitlement keys and resolved values.

Flags:
- `-p, --path <plist>`: Optional explicit entitlements file path.
- `--config <path>`: Used when `--path` is not provided.

Examples:
```bash
ios-app-manager entitlements list
ios-app-manager entitlements --path App/App.entitlements list
```

### `generate makefile`

Syntax:
```bash
ios-app-manager generate makefile
```

Description:
- Creates `Makefile` if missing, or regenerates the managed section while preserving custom section.

Flags:
- `--config <path>`: Optional command-level config override.

Example:
```bash
ios-app-manager generate makefile
```

Note:
- `ios-app-manager generate swiftlint` is also available, but not part of this task checklist.

### `clean`

Syntax:
```bash
ios-app-manager clean [--deep] [--kill-xcode]
```

Description:
- Default mode performs quick project clean.
- `--deep` includes global cache cleanup.
- `--kill-xcode` kills Xcode first and implies deep clean.

Flags:
- `--deep`: Deep clean mode.
- `--kill-xcode`: Kill Xcode before clean; implies deep clean.

Examples:
```bash
ios-app-manager clean
ios-app-manager clean --deep
ios-app-manager clean --kill-xcode
```

### `push send`

Syntax:
```bash
ios-app-manager push send --token <token> [--env dev|prod] [--payload <file>]
```

Description:
- Sends APNs push request using credentials from config.
- Fails when APNs returns a non-2xx status.

Flags:
- `--token <token>`: Required device token.
- `--env <dev|prod>`: APNs environment (default `dev`).
- `--payload <file>`: Optional JSON payload file.
- `--config <path>`: Optional command-level config override.

Example:
```bash
ios-app-manager push send --token "$DEVICE_TOKEN" --env dev --payload ./push.json
```

### `push token`

Syntax:
```bash
ios-app-manager push token
```

Description:
- Prints latest APNs token detected from simulator logs.
- Can use explicit fallback token file.

Flags:
- `--token-file <path>`: Optional fallback token file.
- `--config <path>`: Optional config; used to read fallback token path if set.

Examples:
```bash
ios-app-manager push token
ios-app-manager push token --token-file ./tmp/device-token.txt
```

### `status`

Syntax:
```bash
ios-app-manager status
```

Description:
- Placeholder command in current scaffold.
- Current output: `not implemented`.

Example:
```bash
ios-app-manager status
```

### `q`

Syntax:
```bash
ios-app-manager q '<query>'
```

Description:
- Parses and executes DSL query expression (`operation(params) { fields }`).

Flags:
- `--format pretty|compact`: Output format (default `pretty`).

Examples:
```bash
ios-app-manager q 'summary() { message kind operation }'
ios-app-manager q --format compact 'deps(module=Auth) { operation params }'
```

### `m`

Syntax:
```bash
ios-app-manager m '<mutation>'
```

Description:
- Parses and executes DSL mutation expression (`operation(params)`).

Flags:
- `--format pretty|compact`: Output format (default `pretty`).

Examples:
```bash
ios-app-manager m 'create_module(name=Auth,type=feature)'
ios-app-manager m --format compact 'add_dep(module=Auth,depends_on=CoreKit)'
```

## Exit codes

| Code | Meaning |
|---|---|
| `0` | Command succeeded |
| `1` | Command failed (argument/flag validation, parse/runtime errors, APNs failure, unsupported DSL operation) |
