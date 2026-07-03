# Extension target metadata sync

## Description
Extend project-config sync so cross-cutting target metadata applies to generated app extensions as first-class targets. Extension plugins must register their targets in a discoverable registry/manifest shape. Metadata plugins then read that registry and converge all app and extension targets. This direction keeps concrete extension plugins independent from every metadata plugin while letting version/min-target/team/build-flag changes update extensions consistently.

## Scope
Cover bundle identifier derivation, marketing/build version, minimum deployment target, development team id, Swift/build flags, ApplicationConfiguration where applicable, and generated Info.plist/build settings for extension targets. Keep existing host app behavior intact. Update docs and tests for project-config orchestration order.

## Acceptance Criteria
Changing bundle id inputs converges extension bundle identifiers according to deterministic suffix rules; changing marketing_version/project_version updates every extension target; changing min_target updates extension deploymentTargets and IPHONEOS_DEPLOYMENT_TARGET; changing team_id updates extension signing settings; changing build flags updates extension build settings; rerunning project-config is idempotent and does not duplicate generated declarations; extension plugins do not duplicate metadata logic; docs explain that extension plugins register targets and project-config plugins propagate cross-cutting metadata.
