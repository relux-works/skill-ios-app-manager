# TASK-260630-3rj9u6: sync-extension-min-target-team-build-flags

## Description
Propagate min target, development team id, and Swift/build flags into every registered extension target and generated package where applicable.

## Scope
Implement propagation of min_target, team_id, Swift language mode, strict memory safety, strict concurrency, approachable concurrency, actor-isolation defaults, and upcoming Swift feature flags into every generated extension target Project.swift. Keep package strictness behavior separate for SwiftPM package manifests.

## Acceptance Criteria
generate min-target updates extension deploymentTargets and IPHONEOS_DEPLOYMENT_TARGET; generate team-id updates extension developmentTeam and DEVELOPMENT_TEAM; generate build-flags updates every extension target with the same Swift strictness/concurrency build settings as host app; reruns are idempotent and replace stale values instead of appending duplicates; tests cover at least one generated extension manifest.
