## Status
done

## Assigned To
[implementer] developer (claude)

## Created
2026-03-02T10:57:43Z

## Last Update
2026-03-02T11:17:51Z

## Blocked By
- (none)

## Blocks
- TASK-260302-35m022

## Checklist
- [ ] Code written per task description and AC
- [ ] Tests written and passing
- [ ] Lint clean
- [ ] Build not broken
- [ ] Result noted on the board
- [ ] All 3 Swift files ported and adapted from tuist-akme
- [ ] embed.go exposes CapabilityFiles as embed.FS
- [ ] All Namespacing/ConfigurationHelper/sharedRoot references removed
- [ ] Identifier enum simplified (default + custom only)
- [ ] Go tests pass verifying embedded content
- [ ] make test passes (no regressions)

## Notes
agent spawned: claude (pid=90085, exit=0)
Implemented in internal/scaffold/capability_files/. 4 Swift files ported from tuist-akme + embed.go + embed_test.go. Removed: Namespacing enum, .shared cases, ConfigurationHelper, sharedRoot, environmentSuffix. Simplified Identifier to .default + .custom(id:). All 118 portal capabilities kept. All tests pass. Pre-existing test failure in internal/cli (unrelated to this task).

## Precondition Resources
(none)

## Outcome Resources
(none)
