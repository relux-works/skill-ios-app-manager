# push-token implementation

## What was implemented
- Added APNs token extraction in internal/push/token.go using xcrun simctl spawn booted log show --last 1h --style compact.
- Parser filters APNs registration-related log lines and extracts 64-char hex tokens; returns the most recent token.
- Added fallback support for reading token from a known file path when simulator logs are unavailable/unreliable.
- Wired CLI command ios-app-manager push token in internal/cli/push.go; output is token-only (pipe-friendly).
- Added --token-file flag and config fallback via new push_token_path in internal/config/schema.go.

## Tests
- Added parser and fallback unit tests in internal/push/token_test.go (no real simctl execution).
- Extended CLI tests in internal/cli/push_test.go for push token behavior and fallback wiring.

## Verification
- GOCACHE=/tmp/go-build go test ./internal/push/...
- GOCACHE=/tmp/go-build go test ./...
- GOCACHE=/tmp/go-build go vet ./...
- GOCACHE=/tmp/go-build go build ./...
All commands pass.