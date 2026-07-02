# STORY-260702-1bgmds: background-modes

## Description
Support `UIBackgroundModes` in project config so generated and existing apps get scaffold-owned background modes (audio, voip, push-to-talk, etc.) instead of hand-edited `Project.swift` Info.plist entries. Requested by relux-works/skill-ios-app-manager#6 (Memori needs `audio` today and `push-to-talk` + `voip` next for background voice sessions and system Push to Talk integration).

## Scope
1. `background_modes` string-array field in `ProjectConfig` with whitelist validation against Apple's documented UIBackgroundModes values.
2. `Project.swift.tmpl` renders the `UIBackgroundModes` Info.plist array for new projects.
3. `generate background-modes` leaf generator syncs the key into an existing scaffolded `Project.swift` (insert/replace/remove, anchored like the other manifest sync generators).
4. Tests: validator cases, sync unit tests, golden coverage via the sample config fixture.
5. `references/cli-reference.md` documents the new generator and config field.

## Acceptance Criteria
- `ios-app-manager generate background-modes` is idempotent: no-op on an up-to-date manifest, inserts before `UILaunchScreen` when missing, replaces in place when drifted, removes when the config list is empty.
- Invalid mode strings fail config validation with the allowed list in the message.
- `make test` green; demo app regenerates and compiles per repo workflow.
