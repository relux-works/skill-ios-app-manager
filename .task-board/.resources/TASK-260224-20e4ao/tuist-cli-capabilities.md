# Tuist CLI capabilities (tuist@4.146.2) — 2026-02-24

## Scope

This report documents **Tuist CLI capabilities available in this environment** (Homebrew `tuist@4.146.2`), with a **command-by-command reference** and the main areas we can leverage from our own tooling:

- project generation, target focusing, and graphing
- dependencies install
- build/test (including `xcodebuild` wrapper)
- caching (binary cache + Xcode compilation cache proxy)
- signing (code signing considerations + cryptographic signing of cache responses)
- plugins, templates/scaffolding
- inspections (bundle/build/test/deps) + migration utilities
- “cloud/server” features: auth, accounts, orgs, projects, registry, remote build/test/bundle metadata
- configuration files + env vars used on CI / debugging

Because the sandboxed execution environment prevents running `tuist --help` directly (Tuist attempts to create user-scoped support directories), this report **fact-checks claims against the official shipped CLI binary’s embedded help/strings** and Homebrew installation metadata. See **Sources** and **Repro**.

## Highlights / key takeaways

- `tuist build` and `tuist test` are **deprecated** in favor of `tuist xcodebuild build` / `tuist xcodebuild test`. Tuist keeps `tuist build list/show` and `tuist test case …` for server-backed workflows. [^build-deprecated] [^test-deprecated] [^xcodebuild-extends]
- `Tuist/Config.swift` is **deprecated**; rename it to `Tuist.swift` at repo root. [^config-swift-deprecated]
- Caching has two major axes:
  - **Binary caching** (warm local/remote caches as XCFrameworks). [^cache-warm] [^cache-warm-xcframeworks]
  - **Xcode Compilation Cache** via a local proxy + LaunchAgent/Daemon setup; requires enabling `enableCaching` in `Tuist.swift` and adding Xcode build settings (`COMPILATION_CACHE_*`). [^setup-cache] [^enable-caching-tuist-swift]
- Remote cache responses are expected to be **cryptographically signed**; unsigned/invalid responses are blocked (`x-tuist-signature`). [^cache-signature]
- Registry support exists (`tuist registry setup/login/logout`) and is intended for CI via `TUIST_TOKEN`. [^registry-setup] [^registry-token-ci]
- Tuist now includes **MCP (Model Context Protocol)** integration: start a local MCP server and configure editors (Cursor, VS Code, Zed, Claude, Claude Code) to talk to it. [^mcp-start] [^mcp-setup-ides]

## Command reference (purpose + usage)

Notes on notation:
- “Usage” is a **practical skeleton**, not a full flag list.
- Options shown are only those we can **verify from the shipped binary’s embedded strings**.

### Global / help

- `tuist version`
  - Purpose: prints the current Tuist version. [^version-purpose]
- `tuist --help` / `tuist help`
  - Purpose: shows help (subcommands, options). `HelpCommand` is present in the binary. [^help-command-type]
- `tuist --generate-completion-script=<shell>`
  - Purpose: emits shell completion scripts (ArgumentParser built-in). [^completion-flag]

### Project initialization & configuration

- `tuist init`
  - Purpose: initialize Tuist server features / project configuration; also used as a prerequisite for server-backed features like bundle analysis and sharing. [^init-required-remote] [^inspect-bundle-init]
  - Inputs: interactive answers / templates / platform are supported (environment keys exist). [^init-envvars]
  - Config expectations:
    - `Tuist.swift` can contain `fullHandle` for server features; missing it blocks some flows (e.g., registry login/share/preview). [^missing-fullhandle] [^registry-missing-fullhandle]
    - `tuist.toml` can also provide a project identifier (`project`), used when CLI args aren’t provided. [^tuist-toml-project]

### Generation (workspace/project)

- `tuist generate`
  - Purpose: “Generate a project or inspect generation runs.” [^generate-abstract]
  - Generates: “an Xcode workspace to start working on the project.” [^generate-workspace]
  - Key behaviors/options:
    - Focus targets by name or tag query like `tag:feature` (“Other targets will be linked as binaries if possible”). [^generate-focus-targets]
    - Do not open after generation (`--no-open`). [^generate-no-open]
    - Cache profile support (defaults to config in `Tuist.swift`, or `only-external`). [^cache-profile-default]
  - Server-backed inspection:
    - `tuist generate list` — list generation runs. [^generate-list]
    - `tuist generate show <id>` — show one generation run (requires project full handle/path). [^generate-show] [^generate-full-handle-missing]
  - Deprecations:
    - `--no-binary-cache` is deprecated. [^no-binary-cache-deprecated]

### Dependencies (external content)

- `tuist install`
  - Purpose: “Installs any remote content (e.g. dependencies) necessary to interact with the project.” [^install-abstract]
  - Key behaviors/options:
    - `--update`: update external content when available. [^install-update]
    - Passthrough args to underlying `swift package …` invocation. [^install-passthrough]
  - Operational notes:
    - If dependencies aren’t fetched, Tuist prompts to run `tuist install`. [^install-required]

### Graph visualization

- `tuist graph`
  - Purpose: “Generates a graph from the workspace or project in the current directory.” [^graph-abstract]
  - Output formats: `dot`, `json`, `png`, `svg`. [^graph-formats]
  - Layout algorithms: `dot`, `neato`, `twopi`, `circo`, `fdp`, `sfdp`, `patchwork`. [^graph-layouts]
  - Key behaviors/options:
    - Skip test targets. [^graph-skip-tests]
    - Skip external dependencies. [^graph-skip-external]
    - Filter by targets (includes dependent targets). [^graph-target-filter]
    - Don’t open output file after generating it. [^graph-no-open]

### Build

- `tuist xcodebuild …` (preferred)
  - Purpose: extends `xcodebuild` with Tuist server capabilities (selective testing, analytics). [^xcodebuild-extends]
  - Supports:
    - `tuist xcodebuild build` [^xcodebuild-build]
    - `tuist xcodebuild test` [^xcodebuild-test]
    - `tuist xcodebuild archive` [^xcodebuild-archive]
    - `tuist xcodebuild build-for-testing` [^xcodebuild-build-for-testing]
    - `tuist xcodebuild test-without-building` [^xcodebuild-test-without-building]
  - Passthrough: forwards all args to `xcodebuild` (examples are embedded in CLI strings). [^xcodebuild-test] [^xcodebuild-archive]

#### Signing (code signing)

- Tuist’s build automation may perform code signing when signing is enabled, invoking `/usr/bin/codesign` based on standard Xcode env vars (e.g., `EXPANDED_CODE_SIGN_IDENTITY*`, `CODE_SIGNING_REQUIRED`, `CODE_SIGNING_ALLOWED`). [^code-signing-script]

- `tuist build` (deprecated local runner)
  - Purpose: “Builds a project (deprecated: Use 'tuist xcodebuild build' instead).” [^build-deprecated]
  - Passthrough: supports `--` passthrough to `xcodebuild` (and warns about terminator-handled args). [^build-passthrough]

- `tuist build list/show` (server-backed build records)
  - Purpose: “A set of commands to manage your project builds.” [^build-manage]
  - `tuist build list` supports filtering by git branch, status, scheme, configuration, tags, and custom values (`key=value`). [^build-filters]
  - `tuist build show <id>` shows details for a build (requires project full handle/path). [^build-show]

### Test

- `tuist xcodebuild test` (preferred)
  - See `tuist xcodebuild …` above. [^xcodebuild-test]

- `tuist test` (deprecated local runner)
  - Purpose: intended for generated projects / Swift packages, but now deprecated in favor of `tuist xcodebuild test`. [^test-deprecated]
  - Passthrough: forwards `-- …` args to `xcodebuild`. [^test-passthrough]
  - Selective testing:
    - Has a “no selective testing” mode (“runs all tests without using selective testing”). [^test-no-selective]
    - Can persist (or not) selection results to server (“not persisted”). [^test-no-upload-selection]

- `tuist test case …` (server-backed test case catalog)
  - Purpose: manage test cases and their runs. [^test-case-groups]
  - `tuist test case list` supports filtering for quarantined and flaky cases; can output as `xcodebuild -skip-testing` arguments. [^test-case-list]
  - `tuist test case show <identifier>` shows one test case. [^test-case-show]
  - `tuist test case run list/show …` manages runs for a test case (pagination + flaky filter). [^test-case-run]

### Caching

- `tuist cache warm`
  - Purpose: “Warms the local and remote cache.” [^cache-warm]
  - Inputs: path, configuration, targets; can cache dependencies-only and/or external-only. [^cache-warm-options]
  - Debugging:
    - “generates the project and skips warming the cache” (generate-only mode). [^cache-generate-only]
  - Output: stores cached targets as XCFrameworks. [^cache-warm-xcframeworks]
  - Deprecation: `tuist cache --print-hashes` deprecated; use `tuist hash cache`. [^cache-print-hashes-deprecated]

- `tuist cache start`
  - Purpose: “Start a proxy server to listen for Xcode Compilation Cache requests.” [^cache-start]
  - Upload control exists (“Whether to upload cache artifacts to the remote. Defaults to true.”). [^cache-start-upload]

- `tuist cache config`
  - Purpose: fetch “remote cache configuration” (endpoint URL + authentication token) for configuring remote build caching (e.g., Gradle). [^cache-config]
  - Auth: requires project full handle via arg or `tuist.toml`. [^cache-config-full-handle]

- `tuist setup cache`
  - Purpose: “Set up the Tuist Xcode cache.” [^setup-cache]
  - Produces: LaunchAgent/Daemon configuration and setup guidance; requires adding `COMPILATION_CACHE_*` build settings to Xcode projects. [^setup-cache-settings]
  - Requires `Tuist.swift` config:
    - enable caching with `enableCaching: true` under `generationOptions`. [^enable-caching-tuist-swift]

#### Cache security / signing

- Remote cache responses must be signed; Tuist checks `x-tuist-signature` and blocks invalid/unsigned responses. [^cache-signature]

### Clean

- `tuist clean`
  - Purpose: clean cache/artifact categories (cleans everything when no category is specified). [^clean-categories]
  - Modes: clean the remote cache and/or clean all artifacts stored locally. [^clean-modes]
  - Output includes success messages such as “Successfully cleaned the remote storage.” [^clean-success]

### Scaffolding / templates

- `tuist scaffold`
  - Purpose: “Generates new project based on a template.” [^scaffold-generate]
  - Template discovery: “Lists available scaffold templates.” [^scaffold-list]
  - Safety: refuses to generate into a non-empty directory. [^scaffold-non-empty]
  - Default templates installed with Homebrew:
    - `/opt/homebrew/Cellar/tuist@4.146.2/4.146.2/share/Templates/default` (platform variants under `ios/`, `macos/`, `tvos/`, `watchos/`). [^templates-path]

### Plugins

- `tuist plugin archive`
  - Purpose: “Archives a plugin into a NameOfPlugin.tuist-plugin.zip.” [^plugin-archive]
  - Output: `.tuist-plugin.zip`. [^plugin-zip]
- `tuist plugin build`
  - Purpose: builds a plugin; supports building targets/products and printing the binary output path. [^plugin-build] [^plugin-build-opts]
- `tuist plugin run`
  - Purpose: runs a plugin task; supports passing arguments and skipping build. [^plugin-run-opts]
- `tuist plugin test`
  - Purpose: test plugin products (strings indicate test-products/build-tests controls). [^plugin-test-envvars]
  - Gotcha: tasks must have executable products prefixed with `tuist-`. [^plugin-tuist-prefix]

### Inspect / analyze

- `tuist inspect bundle`
  - Purpose: inspect an app bundle (`.app`, `.xcarchive`, `.ipa`). [^inspect-bundle-abstract]
  - Can upload for analysis on server (“Pushing bundle to the server…”); requires `tuist init` to connect to server. [^inspect-bundle-upload]
- `tuist inspect build`
  - Purpose: inspect the latest build for a project path; can run in background and can wait for result (env keys exist). [^inspect-build]
- `tuist inspect test`
  - Purpose: inspect a test run; supports derived data + `.xcresult` resolution and can run in background/wait. [^inspect-test]
- `tuist inspect dependencies`
  - Purpose: checks implicit/redundant dependency issues and fails when found. [^inspect-deps-abstract]
  - Replacement for deprecated commands:
    - `tuist inspect redundant-imports` → `tuist inspect dependencies --only redundant`. [^inspect-redundant-deprecated]
    - `tuist inspect implicit-imports` → `tuist inspect dependencies --only implicit`. [^inspect-implicit-deprecated]

### Edit

- `tuist edit`
  - Purpose: “Generates a temporary project to edit the project in the current directory.” [^edit-abstract]

### Dump (manifest as JSON)

- `tuist dump`
  - Purpose: dump a manifest as JSON. [^dump-abstract]
  - Takes: “The manifest to be dumped”. [^dump-manifest]

### Run

- `tuist run <scheme-or-target> [-- <app-args…>]`
  - Purpose: build & run a runnable scheme/target; forwards all args after scheme/target to the app. [^run-abstract]
  - Example embedded: `tuist run MyApp --verbose --config debug` (args forwarded to the app). [^run-example]

### Share

- `tuist share <App> …`
  - Purpose: “Generate a link to share your app.” [^share-abstract]
  - Constraints:
    - Can’t share multiple apps in one invocation. [^share-multiple-apps]
    - If not using Tuist projects, you must specify platforms and app name. [^share-platforms] [^share-appname]
  - Server requires: `fullHandle` for remote flows. [^share-missing-fullhandle]

### Migration (from `.xcodeproj` → Tuist)

- `tuist migration …`
  - Purpose: “A set of utilities to assist in the migration of Xcode projects to Tuist.” [^migration-abstract]
  - Subcommands (names + behavior are embedded in the CLI):
    - `tuist migration check-empty-settings [--target <name>]` — checks if build settings are empty; exits unsuccessfully otherwise. [^migration-check-empty-settings]
    - `tuist migration settings-to-xcconfig --xcconfig-path <file> [--target <name>]` — extracts build settings into an `.xcconfig`. [^migration-settings-to-xcconfig]
    - Target listing sorted by number of dependencies is supported (description embedded). [^migration-targets-by-deps]

### MCP (Model Context Protocol)

- `tuist mcp start`
  - Purpose: “Start an MCP server to interface LLMs with your local dev environment.” [^mcp-start]
- `tuist mcp setup …`
  - Purpose: configure tools/editors to point at Tuist’s MCP server. Supported setup targets (descriptions embedded):
    - Cursor, VS Code, Zed, Claude, Claude Code. [^mcp-setup-ides]

### Server / cloud features

These commands operate on Tuist server accounts/projects/orgs and require authentication.

- `tuist auth …`
  - Purpose: “Manage authentication”. [^auth-abstract]
  - Subcommands:
    - `tuist auth whoami` — show the currently-authenticated identity (email). [^auth-whoami]
    - `tuist auth login` — authenticate (email/password; CI may use OIDC). [^auth-login]
    - `tuist auth refresh-token --url <server>` — refreshes the token for a particular server URL. [^auth-refresh]
    - `tuist auth logout` — removes the existing session / logs out. [^auth-logout]
- `tuist account …`
  - Purpose: “A set of commands to manage your Tuist account”. [^account-abstract]
  - Subcommands:
    - `tuist account tokens list/create/revoke` — manage account tokens with scopes + optional expiry (`30d`, `6m`, `1y`). [^account-tokens]
    - `tuist account update …` — update account settings. [^account-update]
- `tuist organization …`
  - Purpose: manage organizations (membership, SSO, billing). [^org-subcommands]
  - Common subcommands:
    - `tuist organization list` / `show <org>` / `create <org>` / `delete <org>` / `billing <org>` [^org-subcommands]
    - `tuist organization invite <org> <email>` [^org-subcommands]
    - `tuist organization remove member|invite|sso …` [^org-subcommands]
    - `tuist organization update member|sso …` [^org-subcommands]
- `tuist project …`
  - Purpose: manage Tuist projects and project tokens. [^project-commands]
  - Common subcommands:
    - `tuist project list` / `show <full-handle> [--web]` / `create <full-handle>` / `delete <full-handle>` / `update <full-handle>` [^project-commands]
    - `tuist project tokens list/create/revoke …` (project tokens are deprecated in favor of account tokens). [^project-tokens-deprecated]
- `tuist bundle list/show`
  - Purpose: manage server-side bundle records for a project (list/show + git branch filtering). [^bundle-commands]
- `tuist registry setup/login/logout`
  - Purpose: configure and authenticate to the Tuist Registry. [^registry-setup] [^registry-login] [^registry-logout]

## Configuration & environment variables (practical)

### Repo files / directories

- `Tuist.swift`
  - Primary Tuist project configuration; expected to contain `fullHandle` for server-backed features (and `enableCaching` for Xcode cache). [^missing-fullhandle] [^enable-caching-tuist-swift]
- `tuist.toml`
  - Can provide `project` identifier when CLI args aren’t provided. [^tuist-toml-project]
- `.tuist-generated`
  - Generated artifacts directory name appears in the CLI. [^tuist-generated-dir]
- `.tuist-version`, `.tuist-bin`
  - Present in CLI strings (Tuist version/bin management). [^tuist-version-bin]

### CI authentication

- Prefer `TUIST_TOKEN` for CI auth (“Use `TUIST_TOKEN` environment variable instead of `TUIST_CONFIG_TOKEN` …”). [^tuist-token-preferred]
- Registry on CI requires project token in `TUIST_TOKEN`. [^registry-token-ci]

### Debugging / logging

- Completion scripts: `--generate-completion-script=<shell>`. [^completion-flag]
- Thread dumps:
  - `TUIST_THREAD_DUMP_SIGNAL` / `TUIST_THREAD_DUMP_SAMPLE` enable SIGUSR1 diagnostics (thread dump or `/usr/bin/sample`). [^thread-dump-env]

## Repro (how to regenerate evidence)

All evidence in this report can be reproduced offline:

```bash
# Identify installed Tuist version (Homebrew)
cat /opt/homebrew/Cellar/tuist@4.146.2/4.146.2/INSTALL_RECEIPT.json

# Extract embedded CLI strings (used for option/command verification)
strings /opt/homebrew/Cellar/tuist@4.146.2/4.146.2/bin/tuist > .temp/tuist_strings.txt

# Extract Tuist-related env keys and command-type names (supporting completeness)
rg -n '^TUIST_' .temp/tuist_strings.txt | cut -d: -f2- | sort -u
rg -n '^[A-Za-z0-9]+Command$' .temp/tuist_strings.txt | cut -d: -f2- | sort -u
```

## Sources (offline, official)

All citations point to local, official artifacts shipped in the installed Tuist build:

- Homebrew receipt: `/opt/homebrew/Cellar/tuist@4.146.2/4.146.2/INSTALL_RECEIPT.json`
- Tuist CLI binary: `/opt/homebrew/Cellar/tuist@4.146.2/4.146.2/bin/tuist`
- Extracted embedded strings: `.temp/tuist_strings.txt` (generated via `strings …/bin/tuist`)
- Built-in templates: `/opt/homebrew/Cellar/tuist@4.146.2/4.146.2/share/Templates/`

---

## Citations

[^build-deprecated]: `.temp/tuist_strings.txt` lines 20723 and 18148.
[^test-deprecated]: `.temp/tuist_strings.txt` line 20827.
[^xcodebuild-extends]: `.temp/tuist_strings.txt` line 20910.
[^config-swift-deprecated]: `.temp/tuist_strings.txt` line 18471.
[^setup-cache]: `.temp/tuist_strings.txt` line 20786.
[^enable-caching-tuist-swift]: `.temp/tuist_strings.txt` line 20675 (includes `enableCaching: true` guidance).
[^cache-signature]: `.temp/tuist_strings.txt` lines 18462 and 18463 (header + blocking behavior).
[^registry-setup]: `.temp/tuist_strings.txt` line 21251.
[^registry-token-ci]: `.temp/tuist_strings.txt` line 21259.
[^mcp-start]: `.temp/tuist_strings.txt` line 20766.
[^mcp-setup-ides]: `.temp/tuist_strings.txt` lines 20482, 20638, 20900, 20494.
[^version-purpose]: `.temp/tuist_strings.txt` line 21947.
[^help-command-type]: `.temp/tuist_strings.txt` lines 25889 and 63361.
[^completion-flag]: `.temp/tuist_strings.txt` line 7883.
[^init-required-remote]: `.temp/tuist_strings.txt` line 20733.
[^inspect-bundle-init]: `.temp/tuist_strings.txt` line 20899.
[^init-envvars]: `.temp/tuist_envvars.txt` entries `TUIST_INIT_*` (derived from `.temp/tuist_strings.txt`).
[^missing-fullhandle]: `.temp/tuist_strings.txt` line 20488.
[^registry-missing-fullhandle]: `.temp/tuist_strings.txt` line 21260.
[^tuist-toml-project]: `.temp/tuist_strings.txt` line 18365.
[^generate-abstract]: `.temp/tuist_strings.txt` line 18938.
[^generate-workspace]: `.temp/tuist_strings.txt` line 18941.
[^generate-focus-targets]: `.temp/tuist_strings.txt` line 18939.
[^generate-no-open]: `.temp/tuist_strings.txt` line 18940.
[^cache-profile-default]: `.temp/tuist_strings.txt` line 18930.
[^generate-list]: `.temp/tuist_strings.txt` line 18947.
[^generate-show]: `.temp/tuist_strings.txt` line 18935.
[^generate-full-handle-missing]: `.temp/tuist_strings.txt` line 18937.
[^no-binary-cache-deprecated]: `.temp/tuist_strings.txt` line 18944.
[^install-abstract]: `.temp/tuist_strings.txt` line 20515.
[^install-update]: `.temp/tuist_strings.txt` line 20512.
[^install-passthrough]: `.temp/tuist_strings.txt` line 20513.
[^install-required]: `.temp/tuist_strings.txt` line 20981.
[^graph-abstract]: `.temp/tuist_strings.txt` line 20746.
[^graph-formats]: `.temp/tuist_strings.txt` line 20742.
[^graph-layouts]: `.temp/tuist_strings.txt` line 20744.
[^graph-skip-tests]: `.temp/tuist_strings.txt` line 20740.
[^graph-skip-external]: `.temp/tuist_strings.txt` line 20741.
[^graph-target-filter]: `.temp/tuist_strings.txt` line 20745.
[^graph-no-open]: `.temp/tuist_strings.txt` line 20743.
[^xcodebuild-build]: `.temp/tuist_strings.txt` line 20815.
[^xcodebuild-test]: `.temp/tuist_strings.txt` line 20489.
[^xcodebuild-archive]: `.temp/tuist_strings.txt` line 20508.
[^xcodebuild-build-for-testing]: `.temp/tuist_strings.txt` line 20558.
[^xcodebuild-test-without-building]: `.temp/tuist_strings.txt` line 20797.
[^build-passthrough]: `.temp/tuist_strings.txt` line 18137 and 18146.
[^code-signing-script]: `.temp/tuist_strings.txt` lines 20099 to 20104.
[^build-manage]: `.temp/tuist_strings.txt` line 18172.
[^build-filters]: `.temp/tuist_strings.txt` lines 18173 to 18178.
[^build-show]: `.temp/tuist_strings.txt` line 18157 and 18165.
[^test-passthrough]: `.temp/tuist_strings.txt` line 21917.
[^test-no-selective]: `.temp/tuist_strings.txt` line 21914.
[^test-no-upload-selection]: `.temp/tuist_strings.txt` line 21908.
[^test-case-groups]: `.temp/tuist_strings.txt` lines 21925 and 21926.
[^test-case-list]: `.temp/tuist_strings.txt` lines 21934 and 21935.
[^test-case-show]: `.temp/tuist_strings.txt` line 21937.
[^test-case-run]: `.temp/tuist_strings.txt` lines 21888 and 21892.
[^cache-warm]: `.temp/tuist_strings.txt` line 18346.
[^cache-warm-options]: `.temp/tuist_strings.txt` lines 18340 to 18343.
[^cache-generate-only]: `.temp/tuist_strings.txt` line 18344.
[^cache-warm-xcframeworks]: `.temp/tuist_strings.txt` line 20849.
[^cache-print-hashes-deprecated]: `.temp/tuist_strings.txt` lines 18347 to 18349.
[^cache-start]: `.temp/tuist_strings.txt` line 20749.
[^cache-start-upload]: `.temp/tuist_strings.txt` line 20748.
[^cache-config]: `.temp/tuist_strings.txt` lines 18355 to 18360.
[^cache-config-full-handle]: `.temp/tuist_strings.txt` line 18365.
[^setup-cache-settings]: `.temp/tuist_strings.txt` lines 20666 to 20673.
[^scaffold-generate]: `.temp/tuist_strings.txt` line 20499.
[^scaffold-list]: `.temp/tuist_strings.txt` line 20498.
[^scaffold-non-empty]: `.temp/tuist_strings.txt` line 20471.
[^templates-path]: `find /opt/homebrew/Cellar/tuist@4.146.2/4.146.2/share/Templates -maxdepth 2 -type d` output.
[^plugin-archive]: `.temp/tuist_strings.txt` line 20501.
[^plugin-zip]: `.temp/tuist_strings.txt` line 20770.
[^plugin-build]: `.temp/tuist_strings.txt` line 20812.
[^plugin-build-opts]: `.temp/tuist_strings.txt` lines 20809 to 20811.
[^plugin-run-opts]: `.temp/tuist_envvars.txt` entries `TUIST_PLUGIN_RUN_*` (derived from `.temp/tuist_strings.txt`).
[^plugin-test-envvars]: `.temp/tuist_envvars.txt` entries `TUIST_PLUGIN_TEST_*` (derived from `.temp/tuist_strings.txt`).
[^plugin-tuist-prefix]: `.temp/tuist_strings.txt` line 20768.
[^inspect-bundle-abstract]: `.temp/tuist_strings.txt` line 20750.
[^inspect-bundle-upload]: `.temp/tuist_strings.txt` lines 20898 and 20899.
[^inspect-build]: `.temp/tuist_strings.txt` line 20478.
[^inspect-test]: `.temp/tuist_strings.txt` lines 20460, 20753, and 20765.
[^inspect-deps-abstract]: `.temp/tuist_strings.txt` line 20511.
[^inspect-redundant-deprecated]: `.temp/tuist_strings.txt` line 20584.
[^inspect-implicit-deprecated]: `.temp/tuist_strings.txt` line 20877.
[^edit-abstract]: `.temp/tuist_strings.txt` line 20474.
[^dump-abstract]: `.temp/tuist_strings.txt` line 20628.
[^dump-manifest]: `.temp/tuist_strings.txt` line 20627.
[^run-abstract]: `.temp/tuist_strings.txt` lines 20662 to 20664.
[^run-example]: `.temp/tuist_strings.txt` line 20661.
[^share-abstract]: `.temp/tuist_strings.txt` line 20544.
[^share-multiple-apps]: `.temp/tuist_strings.txt` line 20735.
[^share-platforms]: `.temp/tuist_strings.txt` line 20737.
[^share-appname]: `.temp/tuist_strings.txt` line 20736.
[^share-missing-fullhandle]: `.temp/tuist_strings.txt` line 20732.
[^migration-abstract]: `.temp/tuist_strings.txt` line 20466.
[^migration-targets-by-deps]: `.temp/tuist_strings.txt` line 20467.
[^migration-check-empty-settings]: `.temp/tuist_strings.txt` lines 20882 to 20884.
[^migration-settings-to-xcconfig]: `.temp/tuist_strings.txt` lines 20589 to 20592.
[^auth-abstract]: `.temp/tuist_strings.txt` line 18035.
[^auth-whoami]: `.temp/tuist_strings.txt` line 18031.
[^auth-login]: `.temp/tuist_strings.txt` lines 18032, 18033, 18043, and 18044.
[^auth-refresh]: `.temp/tuist_strings.txt` line 18047.
[^auth-logout]: `.temp/tuist_strings.txt` lines 18042 and 18050.
[^account-abstract]: `.temp/tuist_strings.txt` line 18024.
[^account-tokens]: `.temp/tuist_strings.txt` lines 18004 to 18006, 18013, 18016, and 18023.
[^account-update]: `.temp/tuist_strings.txt` line 18019.
[^org-subcommands]: `.temp/tuist_strings.txt` lines 21137, 21145, 21154, 21159, 21161, 21162, 21167, 21169, and 21174.
[^project-commands]: `.temp/tuist_strings.txt` lines 21210, 21212, 21213, 21216, 21226, 21231, and 21232.
[^project-tokens-deprecated]: `.temp/tuist_strings.txt` line 21217.
[^bundle-commands]: `.temp/tuist_strings.txt` lines 18180 to 18186.
[^registry-login]: `.temp/tuist_strings.txt` lines 21253, 21254, and 21261.
[^registry-logout]: `.temp/tuist_strings.txt` lines 21243, 21263, and 21264.
[^clean-categories]: `.temp/tuist_strings.txt` line 20889.
[^clean-modes]: `.temp/tuist_strings.txt` lines 20890 and 20891.
[^clean-success]: `.temp/tuist_strings.txt` line 20506.
[^tuist-generated-dir]: `.temp/tuist_strings.txt` line 18479.
[^tuist-version-bin]: `.temp/tuist_strings.txt` lines 38075 and 38076.
[^tuist-token-preferred]: `.temp/tuist_strings.txt` line 20688.
[^thread-dump-env]: `.temp/tuist_strings.txt` lines 21856 to 21866.
