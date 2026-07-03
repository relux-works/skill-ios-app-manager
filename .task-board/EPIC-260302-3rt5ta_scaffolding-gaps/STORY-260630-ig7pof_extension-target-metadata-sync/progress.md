## Status
done

## Assigned To
codex-inline

## Created
2026-06-30T16:48:42Z

## Last Update
2026-06-30T18:04:56Z

## Blocked By
- STORY-260630-2sisyw

## Blocks
- (none)

## Checklist
(empty)

## Notes
User clarified concurrency restrictions must propagate to extension targets too. Treat project_settings.swift concurrency/strictness as part of extension metadata sync, owned by generate build-flags/project-config, not by concrete extension plugins.
Implemented extension metadata sync as pluggable generators. Added generate bundle-id for host and extension manifests, wired it into project-config, documented the contract, and covered extension propagation for bundle id, versions, min target, team id, build flags, and configured concurrency restrictions. Verification: go test ./... from tuist-starter passed.

## Precondition Resources
(none)

## Outcome Resources
(none)
