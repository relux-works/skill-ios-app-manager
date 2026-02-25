# STORY-260224-5pcbct: push-testing

## Description
End-to-end push notification testing toolchain via make targets.

Flow:
1. make push-token — launch app on simulator, parse simctl logs to extract device push token (printed at push subscription time)
2. make push-send ENV=dev PAYLOAD=path/to/payload.json — send real APNS push to device using p8 auth key
3. make push-send ENV=prod — same but to production APNS endpoint

Requires:
- .p8 auth key (path in project config)
- Auth Key ID, Team ID, Bundle ID (from project config)
- Device token (extracted from simctl logs or passed manually)

Adapt existing send-push-with-p8.sh script (see .research/ref_send-push-with-p8.sh):
- Parameterize: token, key path, key ID, team ID, bundle ID, env (dev/prod), payload
- Read all IDs from project config (no hardcoding)
- Support custom JSON payloads
- JWT signing via openssl (ES256, p8 key)
- curl to api.development.push.apple.com or api.push.apple.com

Simctl log parsing:
- xcrun simctl spawn booted log stream --predicate ... to catch device token registration
- Extract hex token from NSLog/os_log output

All exposed via Makefile targets.

## Scope
(define story scope)

## Acceptance Criteria
(define acceptance criteria)
