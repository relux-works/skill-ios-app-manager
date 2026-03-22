## Status
done

## Assigned To
codex

## Created
2026-03-22T10:16:15Z

## Last Update
2026-03-22T10:43:37Z

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
(empty)

## Notes
Findings: generate already has a reusable plugin registry, but generate versions is a field-specific text patcher. marketing_version and project_version are synced post-init; min_target exists in config and root Project.swift.tmpl but is not propagated into extension Project.swift templates. tuistproj.ApplyManifestEditsToFile is reusable for list edits, but it does not support scalar build setting or deployment target rewrites. Current recommendation is to build a generalized project-config sync engine and keep versions as its first slice or compatibility alias.

## Precondition Resources
(none)

## Outcome Resources
(none)
