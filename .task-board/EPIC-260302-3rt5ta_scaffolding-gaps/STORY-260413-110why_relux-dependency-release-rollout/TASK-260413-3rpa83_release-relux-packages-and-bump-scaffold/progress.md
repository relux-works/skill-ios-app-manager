## Status
done

## Assigned To
codex

## Created
2026-04-13T08:37:35Z

## Last Update
2026-04-13T08:44:57Z

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
(empty)

## Notes
Started release rollout. Next: restore library Package.swift files from temporary local path wiring to releasable remote form, commit/push/tag 4 package repos with 2026-04-12 evening Moscow timestamps, then bump scaffold pins in skill-ios-app-manager and finally switch x-platform-airdrop back to remote deps for rebuild validation.
Library releases pushed: darwin-relux 9.0.3, swift-ioc 1.0.3, swiftui-reluxrouter 11.0.2, swiftui-relux 8.0.3. Next: update ios-app-manager scaffold pins to these versions, commit/tag skill repo, then switch x-platform-airdrop back to remote deps and rebuild.
Library releases are already out: swift-relux 9.0.3, swift-ioc 1.0.3, swiftui-relux 8.0.3, swiftui-reluxrouter 11.0.2. Updated ios-app-manager scaffold pins and expectations; go test ./... in tuist-starter passed. Next: commit/tag skill-ios-app-manager, then switch x-platform-airdrop back to remote dependencies and rebuild.
Release rollout completed. skill-ios-app-manager scaffold pins bumped to swift-relux 9.0.3, swiftui-relux 8.0.3, swift-ioc 1.0.3; commit fd2df4e tagged v0.7.1 and pushed. x-platform-airdrop switched back to remote dependencies, tuist install resolved swift-relux 9.0.3 / swiftui-relux 8.0.3 / swift-ioc 1.0.3 / swiftui-reluxrouter 11.0.2, tuist generate succeeded, make build succeeded, and simulator launch succeeded for bundle ru.mws.ios.x-platform-airdrop.app.

## Precondition Resources
(none)

## Outcome Resources
(none)
