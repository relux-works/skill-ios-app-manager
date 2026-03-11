# TASK-260311-13qp41: define-shared-version-source

## Description
Define and implement the single authoritative source for MARKETING_VERSION and CURRENT_PROJECT_VERSION in scaffolded Tuist projects.

## Scope
Inspect the current init and extension scaffold flow, choose the source-of-truth shape that will be enforced by generation, and update generator inputs/helpers so version/build values are emitted once and referenced everywhere. Decide whether ios-app-manager.json remains generator input metadata or is explicitly non-authoritative at runtime.

## Acceptance Criteria
One shared source for marketing version and build number is introduced in scaffolded projects; generator code reads/writes version/build through that shared source only; target-specific templates no longer define independent raw version/build defaults; the chosen source preserves existing xflow-ios values.
