# Generated Makefile Hook Contract

Status: proposed

This specification defines the next generated `Makefile` contract for
`ios-app-manager generate makefile` and the `init` scaffold output.

## Goals

- Make bare `make` a useful project entrypoint after scaffold bootstrap.
- Expose generic pre/post hook points without coupling the scaffold to any
  project-specific tool.
- Preserve the existing custom section contract.
- Keep `build` and `test` cleanup behavior intact.
- Keep the existing `generate makefile` scaffold generator plugin; only the
  rendered Makefile contract changes.

## Non-Goals

- Do not add a new scaffold generator plugin.
- Do not add tool-specific hook targets or tool-specific commands.
- Do not change how `generator_makefile.go` reads an existing Makefile,
  preserves the custom section, and writes the regenerated content.
- Do not change the custom section marker or preservation algorithm.

## Entrypoints

The generated Makefile has three distinct project lifecycle entrypoints:

- `project`: daily project regeneration entrypoint. This is the first ordinary
  target, so bare `make` runs it.
- `setup`: bootstrap entrypoint. It installs Tuist dependencies, then delegates
  generation to `make generate`.
- `generate`: raw Tuist project generation entrypoint.

`help` remains available as an explicit command through `make help`, but the
generated Makefile must not set `.DEFAULT_GOAL := help`.

Special targets, variable declarations, and `.PHONY` may appear before
`project`; `project` only needs to be the first ordinary target.

Expected shape:

```make
.SECONDEXPANSION:

PRE_PROJECT_HOOKS ?=
POST_PROJECT_HOOKS ?=
PRE_SETUP_HOOKS ?=
POST_SETUP_HOOKS ?=
PRE_GENERATE_HOOKS ?=
POST_GENERATE_HOOKS ?=
PRE_BUILD_HOOKS ?=
POST_BUILD_HOOKS ?=
PRE_TEST_HOOKS ?=
POST_TEST_HOOKS ?=
PRE_CLEAN_HOOKS ?=
POST_CLEAN_HOOKS ?=

.PHONY: project pre-project post-project setup pre-setup post-setup generate pre-generate post-generate

project: ## Generate the Xcode project
	@$(MAKE) pre-project
	@$(MAKE) generate
	@$(MAKE) post-project

setup: ## Install Tuist dependencies and generate project files
	@$(MAKE) pre-setup
	@tuist install
	@$(MAKE) generate
	@$(MAKE) post-setup

generate: ## Generate the Xcode project with Tuist
	@$(MAKE) pre-generate
	@tuist generate $(TUIST_GENERATE_FLAGS)
	@$(MAKE) post-generate
```

## Hook Variables

Each hook target depends on a corresponding hook variable:

```make
pre-generate: $$(PRE_GENERATE_HOOKS)
	@:

post-generate: $$(POST_GENERATE_HOOKS)
	@:
```

The generated Makefile must use `.SECONDEXPANSION:` and escaped prerequisites
such as `$$(PRE_GENERATE_HOOKS)`. The custom section is below the generated
section, so a first-pass prerequisite expansion such as
`pre-generate: $(PRE_GENERATE_HOOKS)` would capture the variable before custom
section appends are parsed.

Project-specific integrations belong in the preserved custom section:

```make
# --- Custom targets below --- #
# Add custom targets, variables, and hook variable appends below.
# This section is preserved on regeneration.

PRE_GENERATE_HOOKS += install-local-dependencies

.PHONY: install-local-dependencies
install-local-dependencies:
	@some-project-command
```

The generated default custom section should mention hook variables, but it must
not include examples tied to a specific external tool.

## Hook Set

The minimum generated hook set is:

- `pre-project`, `post-project`
- `pre-setup`, `post-setup`
- `pre-generate`, `post-generate`
- `pre-build`, `post-build`
- `pre-test`, `post-test`
- `pre-clean`, `post-clean`

Each hook target should be listed in `.PHONY` and should default to a no-op
recipe:

```make
pre-build: $$(PRE_BUILD_HOOKS)
	@:
```

## Build And Test Semantics

`build` and `test` must delegate Tuist generation through `$(MAKE) generate`
instead of calling `tuist generate` inline. This makes generate hooks behave the
same way for direct generation and for build/test workflows.

Expected shape:

```make
build: ## Build the app with xcodebuild
	@$(MAKE) pre-build
	@set -e; \
	trap '$(MAKE) clean-package-artifacts >/dev/null' EXIT; \
	$(MAKE) generate; \
	xcodebuild -workspace "$(WORKSPACE)" -scheme "$(SCHEME)" -destination "$(BUILD_DESTINATION)" -derivedDataPath "$(DERIVED_DATA_PATH)" build
	@$(MAKE) post-build
```

The cleanup trap intentionally wraps the subshell containing
`$(MAKE) generate` and `xcodebuild`. It does not wrap `pre-build` or
`post-build`, which run as separate recipe lines.

The effective `make build` hook order is:

```text
pre-build -> pre-generate -> tuist generate -> post-generate -> xcodebuild -> post-build
```

`make test` follows the same pattern with `pre-test` and `post-test`.

`clean-package-artifacts` remains a generated target and continues to use:

```make
PACKAGE_ARTIFACT_CLEANUP_CMD ?= :
```

## Rejected Alternative: Double-Colon Rules

Double-colon hook targets were considered:

```make
pre-generate::
	@:

# custom section
pre-generate:: install-local-dependencies
```

This avoids `.SECONDEXPANSION`, but it makes the extension point less explicit
and requires users to understand the difference between single-colon and
double-colon rules. Hook variables are preferred because they provide one
discoverable knob per hook and are easier to document.

## Test Requirements

Unit and scaffold tests must cover:

- `project:` exists and appears before `setup:`.
- `.DEFAULT_GOAL := help` is absent.
- `.SECONDEXPANSION:` is present.
- Hook variables such as `PRE_GENERATE_HOOKS ?=` and
  `POST_GENERATE_HOOKS ?=` are present.
- Hook targets use second-expansion prerequisites, for example
  `pre-generate: $$(PRE_GENERATE_HOOKS)`.
- `generate` invokes `pre-generate` and `post-generate`.
- `setup` invokes `generate` instead of calling `tuist generate` directly.
- `build` and `test` invoke `$(MAKE) generate`.
- `build` and `test` keep the `clean-package-artifacts` cleanup trap.
- The generated Makefile does not include tool-specific hook commands.
- The golden Makefile fixture matches the new contract.

At least one behavioral test must execute real `make`, not only grep generated
text. The test should render or write a Makefile whose preserved custom section
contains a hook append such as:

```make
PRE_GENERATE_HOOKS += test-hook

.PHONY: test-hook
test-hook:
	@printf "hook-ran\n" > hook.out
```

Then it should run:

```bash
make pre-generate
```

and assert that `hook.out` was created. This specifically protects against
regressing from `$$(PRE_GENERATE_HOOKS)` back to `$(PRE_GENERATE_HOOKS)`.

## Documentation Requirements

Update the user-facing docs that describe generated Makefile behavior:

- `README.md`
- `SKILL.md`
- `references/cli-reference.md`
- `references/tree-tuist.md`

Required wording:

> Generated Makefiles expose generic pre/post hook targets and hook variables;
> project-specific integrations belong in the preserved custom section.

Docs must also explain:

- bare `make` runs `project`;
- `make help` is the explicit discovery command;
- `setup` is bootstrap and includes `tuist install`;
- `project` is the daily generation entrypoint;
- `build` and `test` recursively run `generate`, so generate hooks run inside
  build/test workflows;
- the cleanup trap wraps only the `generate + xcodebuild` recipe line.

## Verification Requirements

From `tuist-starter/`:

```bash
go test ./...
go test ./internal/scaffold ./internal/cli ./internal/e2e
```

Because this changes scaffold-generated output, the demo app must also be
rebuilt and compiled according to `CLAUDE.md`:

```bash
cd tuist-starter && make build && cd ..
rm -rf .temp/demo-project/
mkdir -p .temp/demo-project
cp .temp/demo-config.json .temp/demo-project/ios-app-manager.json
cd .temp/demo-project
../../tuist-starter/ios-app-manager init
../../tuist-starter/ios-app-manager ioc setup
../../tuist-starter/ios-app-manager relux setup
../../tuist-starter/ios-app-manager secure-store setup --access-group group.org.xflow.app
../../tuist-starter/ios-app-manager token-provider setup
../../tuist-starter/ios-app-manager utilities setup
../../tuist-starter/ios-app-manager foundation-plus setup
../../tuist-starter/ios-app-manager swiftui-plus setup
../../tuist-starter/ios-app-manager app-extensions setup
../../tuist-starter/ios-app-manager widget-base setup
../../tuist-starter/ios-app-manager app-intents setup
../../tuist-starter/ios-app-manager static-widget setup
../../tuist-starter/ios-app-manager live-activity setup
../../tuist-starter/ios-app-manager module create --from auth.blueprint.json
../../tuist-starter/ios-app-manager app-config setup
../../tuist-starter/ios-app-manager http-client setup
tuist install && tuist generate
xcodebuild -workspace *.xcworkspace -scheme XFlow -sdk iphonesimulator -destination 'platform=iOS Simulator,name=iPhone 17 Pro' build
```
