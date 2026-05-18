## Status
closed

## Assigned To
codex

## Created
2026-05-18T19:07:38Z

## Last Update
2026-05-18T19:22:52Z

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
(empty)

## Notes
Reproduced in .temp/repro-user-bugs/current. After team_id ABCDE12345 -> ZZZZZ99999 and generate project-config: Project.swift line 6 still let developmentTeam = ABCDE12345; host DEVELOPMENT_TEAM uses stale constant; Extensions/DemoAppWidget/Project.swift has no DEVELOPMENT_TEAM setting.
Fixed by adding generate team-id and including it in project-config. Verified scratch repro after team_id change: host and widget extension manifests plus generated xcodebuild settings show DEVELOPMENT_TEAM = ZZZZZ99999.
Final review: diff check clean; repro and fixed scratch runs archived under .temp/repro-user-bugs*. Team ID now syncs into host and extension Project.swift manifests and xcodebuild settings.

## Precondition Resources
(none)

## Outcome Resources
(none)
