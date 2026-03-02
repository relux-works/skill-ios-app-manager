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
- (none)

## Checklist
- [ ] Code written per task description and AC
- [ ] Tests written and passing
- [ ] Lint clean
- [ ] Build not broken
- [ ] Result noted on the board
- [ ] entitlements add command removed
- [ ] entitlements remove command removed
- [ ] entitlements list command preserved and working
- [ ] keys.go removed or emptied
- [ ] manager.go Add/Remove functions removed
- [ ] scaffold/entitlements.go GenerateEntitlements removed
- [ ] scaffold.go no longer generates .entitlements plist file
- [ ] All tests updated and passing
- [ ] make test passes
- [ ] make lint clean

## Notes
agent spawned: claude (pid=90093, exit=0)
Removed entitlements add/remove CLI commands, kept list as read-only. Deleted keys.go (alias system). Removed Add/Remove from manager.go, kept List. Deleted scaffold/entitlements.go (GenerateEntitlements). Removed entitlements plist from scaffold planFiles. Updated tests in cli/entitlements_test.go, cli/init_test.go, entitlements/manager_test.go, scaffold/scaffold_test.go, e2e/pipeline_test.go. All tests pass, lint clean.

## Precondition Resources
(none)

## Outcome Resources
(none)
