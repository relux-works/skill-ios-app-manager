# TASK-260630-8ckxrc: sync-extension-bundle-and-versions

## Description
Propagate bundle identifiers and marketing/build versions into every registered extension target and verify deterministic idempotent updates.

## Scope
Implement deterministic bundle identifier and version convergence for extension manifests. Marketing/build versions must sync to extension Info.plist, and bundle IDs must derive from configured host bundle id plus existing extension suffix.

## Acceptance Criteria
Changing bundle_id updates every extension manifest hostBundleId/derived bundle id without changing suffix; changing marketing_version/project_version updates every extension manifest; reruns are idempotent; tests cover stale extension values.
