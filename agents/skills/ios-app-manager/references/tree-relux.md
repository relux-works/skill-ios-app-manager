# Relux Scaffolding Decision Tree

This tree documents how Relux module scaffolding currently works in `ios-app-manager`, including template inventory, wiring points, dependency rules, and extension workflows.

## Start Here

1. Need a new Relux module scaffold?
   Run [A. Create Module by Type](#a-create-module-by-type).
2. Need to understand interface/impl split and file layout?
   Run [B. Interface + Impl Split](#b-interface--impl-package-split-explanation).
3. Need to know exactly what each type scaffolds?
   Run [C. Module Type Matrix](#c-module-types-and-what-they-scaffold).
4. Need wiring guidance (registration/resolution/composition root)?
   Run [D. Wiring Tree](#d-wiring-tree-module-registration--ioc-registrationresolution--composition-root).
5. Need dependency guidance?
   Run [E. Dependency Patterns](#e-dependency-patterns-interface-only-deps--impl-isolation).
6. Need to add actions/middleware to an existing module?
   Run [F. Extend Existing Modules](#f-extend-existing-modules-actions--middleware).
7. Stuck on generation/layout issues?
   Run [G. Troubleshooting](#g-troubleshooting).

## Working Variables

```bash
CONFIG=ios-app-manager.json
MODULE=Auth
```

## A. Create Module by Type

When to use:
- Create a new module with Relux scaffolding and/or UI scaffolding.

Commands:

```bash
ios-app-manager module create Auth --type feature --config "$CONFIG"
ios-app-manager module create CoreKit --type kit --config "$CONFIG"
ios-app-manager module create SharedState --type shared --config "$CONFIG"
ios-app-manager module create DesignUI --type ui --config "$CONFIG"
ios-app-manager module create Logging --type utility --config "$CONFIG"
```

Validate result:

```bash
ios-app-manager module list --config "$CONFIG"
```

## B. Interface + Impl Package Split Explanation

Conceptual model:
- Interface package exposes protocols/DTO/public actions.
- Implementation package contains store/reducer/middleware/view/wiring internals.

Current generated package directories for split types (`feature|kit|shared|ui`):

```text
Packages/<Module>/Package.swift
Packages/<Module>Impl/Package.swift
```

Current Relux source file layout used by generator:

```text
Packages/<Module>/<Module>Interface/Sources/*.swift
Packages/<Module>/<Module>Impl/Sources/*.swift
```

Example (`Auth`):

```text
Packages/Auth/Package.swift
Packages/AuthImpl/Package.swift
Packages/Auth/AuthInterface/Sources/protocol.swift
Packages/Auth/AuthInterface/Sources/dto.swift
Packages/Auth/AuthImpl/Sources/store.swift
Packages/Auth/AuthImpl/Sources/reducer.swift
...
```

For `utility`, only single package exists and no Relux templates are generated.

## C. Module Types and What They Scaffold

### Type summary

- `feature`
  - Split: yes
  - Relux: yes
  - UI: yes
  - Typical files: store/reducer/actions/state/middleware/view/protocol/dto + wiring files
- `kit`
  - Split: yes
  - Relux: yes
  - UI: no
  - Typical files: feature minus `view.swift`
- `shared`
  - Split: yes
  - Relux: partial/shared patterns
  - UI: no
  - Typical files: protocol/dto + IoC resolver/registration
- `ui`
  - Split: yes
  - Relux: no
  - UI: yes
  - Typical files: view/protocol
- `utility`
  - Split: no
  - Relux: no
  - UI: no
  - Typical files: package placeholder only

### Template inventory (core set)

- `store.swift` -> Observable store + dispatch pipeline
- `reducer.swift` -> pure state transitions
- `actions.swift` -> internal action enum
- `state.swift` -> module state model
- `middleware.swift` -> async side-effects bridge
- `view.swift` -> SwiftUI view for state rendering
- `protocol.swift` -> service protocol contract
- `dto.swift` -> DTO/public models

Additional templates used by scaffolding:
- `actions_public.swift`
- `module_registration.swift`
- `ioc_registration.swift`
- `ioc_resolver.swift`
- `composition_root.swift` (template available; include manually/explicitly when needed)

## D. Wiring Tree (Module Registration + IoC Registration/Resolution + Composition Root)

When to use:
- Wire module service/middleware/store in container.
- Provide app-level composition root with module/dependency registration.

### D1. Module registration (`module_registration.swift`)

Purpose:
- Registers service factory, middleware, and store into `IoC`.

Use when:
- You need container-level module bootstrap in a feature boundary.

### D2. IoC registration (`ioc_registration.swift`)

Purpose:
- Registers concrete service implementation for protocol.
- Supports constructor injection of dependencies.

Use when:
- Module has service implementation and optional dependencies.

### D3. IoC resolver (`ioc_resolver.swift`)

Purpose:
- Central helper for `resolve`, `optionalResolve`, async resolve, and dependency preflight.

Use when:
- You want consistent dependency resolution logic inside module boundary.

### D4. Composition root (`composition_root.swift`)

Purpose:
- Top-level registration orchestration across dependencies + current module.

Use when:
- Building app assembly layer for module graph bootstrapping.

Note:
- `composition_root.swift` template exists in inventory but is not part of every default type template set.

## E. Dependency Patterns (Interface-Only Deps + Impl Isolation)

Rule 1: Depend on interface packages only.
- Internal dependency operations enforce interface module names.
- Do not depend on `*Impl` package names.

Command pattern:

```bash
ios-app-manager dep add Auth --depends-on CoreKit --config "$CONFIG"
ios-app-manager dep list Auth --config "$CONFIG"
```

Rule 2: Keep implementation isolated.
- Place concrete services/middleware/reducers in impl side.
- Expose only protocol/DTO/public action surface from interface side.

Rule 3: Resolve impl details via IoC.
- Use registration/resolver helpers rather than direct concrete references from callers.

## F. Extend Existing Modules (Actions + Middleware)

When to use:
- Feature evolves and needs new state transitions or side-effects.

Important:
- There is no direct top-level CLI command yet for `add action`/`add middleware`.
- Current extension path is source update (or internal Go API usage in automation/code paths).

### F1. Add an action to existing module

Target files:
- `<module>/.../actions.swift`
- `<module>/.../reducer.swift`

Manual pattern:

1. Add action enum case in `actions.swift` (before `.forward(...)` case if present).
2. Add matching reducer case in `reducer.swift` `switch action`.
3. Implement state update or TODO stub.
4. Build/test.

### F2. Add middleware to existing module

Target file:
- `<module>/.../<new_name>_middleware.swift`

Manual pattern:

1. Create middleware file from existing middleware conventions.
2. Implement `handle(action:state:)`.
3. Register middleware usage in module registration if needed.
4. Build/test.

### F3. Validation commands after extension

```bash
go test ./internal/relux/...
go test ./...
```

## G. Troubleshooting

1. `initialize relux manager` or template engine errors
   Fix:
   - Ensure templates exist under `internal/relux/templates`.

2. Generated files land in unexpected directories
   Fix:
   - Verify current layout behavior:
     - `<Module>/<Module>Interface/Sources`
     - `<Module>/<Module>Impl/Sources`
   - If custom layout exists (`Interface/Sources`, `Impl/Sources`), generator can reuse it.

3. `template "... " is not supported`
   Fix:
   - Use supported template names only (`store`, `reducer`, `actions`, `state`, `middleware`, `view`, `protocol`, `dto`, `actions_public`, `module_registration`, `ioc_registration`, `ioc_resolver`, `composition_root`).

4. `action case "... " already exists` / reducer duplicate errors
   Fix:
   - Remove duplicate action/reducer case and rerun.

5. `middleware file "... " already exists`
   Fix:
   - Rename middleware, or delete/replace existing file intentionally.

6. `invalid module/action/middleware name`
   Fix:
   - Use valid Swift identifiers.
   - Module names should be PascalCase.

7. Dependency cycle after wiring
   Fix:
   - Remove one edge:
   ```bash
   ios-app-manager dep remove A --depends-on B --config "$CONFIG"
   ```

