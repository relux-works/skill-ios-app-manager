# STORY-260311-ydil65: shared-app-extension-versioning

## Description
Centralize MARKETING_VERSION and CURRENT_PROJECT_VERSION for scaffolded iOS app and app extension targets. Today the generated project layout duplicates version/build literals across multiple Tuist manifests, while ios-app-manager.json may contain similar metadata without being enforced as the runtime or project source of truth. This creates release drift risk between the app target and .appex targets and makes version bumps multi-file edits.

## Scope
Choose and enforce one authoritative source for version/build values in the scaffold system; update the main app target and every generated app extension target to reference that shared source; remove duplicated per-target literals where possible; ensure newly scaffolded extensions inherit the shared version/build configuration by default; keep ios-app-manager.json aligned only if it is intentionally the generator input; preserve existing shipped values when migrating xflow-ios.

## Acceptance Criteria
Exactly one authoritative source exists for MARKETING_VERSION; exactly one authoritative source exists for CURRENT_PROJECT_VERSION; the app target consumes the shared source; all generated .appex targets consume the same source; scaffolded projects no longer duplicate raw version/build literals across target manifests; a version bump requires changing one file or one generator input only; xflow-ios can migrate without changing shipped values; documentation explains where version/build should be updated going forward; validation fails if a target hardcodes divergent version/build values; regenerated app and widget resolve identical CFBundleShortVersionString and CFBundleVersion.
