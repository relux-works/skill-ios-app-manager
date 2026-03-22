# STORY-260322-3p8k21: project-config-sync

## Description
Generalize the current generate versions scaffold plugin into a reusable project configuration sync layer for Tuist manifests. Introduce one config-driven sync engine for root app and extension manifests so build settings and target metadata do not drift. First scope: marketing_version, project_version, min_target / deployment target, and any shared target-level config that should stay aligned across host app and extensions. Decide whether versions becomes a sub-plugin/alias under a broader generate project-config command, and define the extension model for future config syncs.

## Scope
Refactor the generate scaffold layer around reusable project-config sync primitives instead of field-specific ad hoc patchers. Cover root app Project.swift, generated extension Project.swift manifests, and the init/extension template pipeline. First synced fields are marketing_version, project_version, min_target / deployment target, and other shared target-level settings that must not drift across app and extensions. Define whether versions stays as an alias or becomes a sub-plugin under a broader project-config command.

## Acceptance Criteria
There is one reusable project-config sync path for scaffolded manifests; current version sync behavior is preserved or aliased without regression; min_target can be synced after init for both app and extension manifests; extension scaffolding inherits the same config model by default; tests cover root plus extension sync and idempotent rewrites; documentation tells maintainers which command and source of truth to use for future config changes.
