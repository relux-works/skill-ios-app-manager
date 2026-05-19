# Runtime Error Diagnostics Research

## Goal

Extend runtime profiling so it can read and analyze runtime error signals, not only performance events:

- unified log `error` and `fault` messages;
- app-emitted structured error events;
- crash and hang hints visible in logs;
- future MetricKit crash/hang diagnostic exports.
- app startup latency from launch marker to first rendered SwiftUI subtree.

## Primary Sources

- Apple's unified logging docs identify Console, the `log` command-line tool, and Xcode debug console as viewers for unified logs.
- `log help predicates` on macOS 26 documents `logType` values: `default`, `release`, `info`, `debug`, `error`, and `fault`; it also documents fields such as `process`, `subsystem`, `category`, and `composedMessage`.
- Apple's crash report docs recommend collecting crash reports and diagnostic logs, and using console logs for non-crash issues.
- `OSLogStore` can read unified log entries programmatically, including from a log archive; that is useful for a future in-app/export flow.
- MetricKit can deliver on-device diagnostics such as `MXCrashDiagnostic` and `MXHangDiagnostic`; useful for a later structured diagnostics import path, but not required for local CLI MVP.

## Findings

- Local CLI MVP should analyze `log show --style ndjson` output because NDJSON is stable enough to parse and preserves process/subsystem/category/message fields.
- The default runtime-error collection predicate should be `(logType == "error" OR logType == "fault")`.
- Existing `PerformanceProbe` should grow an `error(...)` helper that emits `IAM_ERROR {json}` lines. This gives deterministic app-level error events even when OS unified log privacy redacts details.
- Existing `PerformanceProbe` should grow `markAppStart` and `markFirstRender` helpers that emit `IAM_PROFILE` lifecycle markers. This gives a cheap local measurement for "stuck before first render".
- Crash logs and MetricKit payloads are not the same shape as unified logs. The MVP should surface crash/hang hints from messages and leave deep crash report symbolication for a later dedicated command.

## MVP Architecture

Add:

```bash
ios-app-manager profile runtime errors [--input <log>] [--last 10m] [--process <name>] [--subsystem <id>]
```

Behavior:

- With `--input`, analyze an existing log file.
- Without `--input`, run host `log show --style ndjson --last <duration>` with a generated predicate.
- With `--simulator`, run `xcrun simctl spawn <device> log show --style ndjson ...`.
- Parse:
  - `IAM_ERROR` JSON lines;
  - unified-log NDJSON objects;
  - plain text lines containing crash/error/fault/fatal/exception/hang keywords.
- Group by severity, process, subsystem, category, and normalized message signature.
- Emit text/JSON report with top groups and crash/hang hints.
- Runtime performance analysis should report app startup intervals when it sees `app_start` and `first_render` markers.

## Follow-Up

- Add direct crash report parser and symbolication hints.
- Add MetricKit JSON import once generated apps expose an export format.
- Add `profile runtime errors --stream` for live capture.
