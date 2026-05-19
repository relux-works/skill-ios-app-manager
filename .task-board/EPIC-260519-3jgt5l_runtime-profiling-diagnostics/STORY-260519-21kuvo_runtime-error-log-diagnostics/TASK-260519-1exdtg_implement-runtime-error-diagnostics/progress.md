## Status
done

## Assigned To
codex

## Created
2026-05-19T12:20:45Z

## Last Update
2026-05-19T12:25:44Z

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
(empty)

## Notes
Scope: extend runtime profile diagnostics beyond performance events into runtime error/fault/crash/hang log analysis. Inputs: log show/stream NDJSON/plain output, app-emitted IAM_ERROR lines, future MetricKit export JSON.
Extended scope per user: include app startup timing until first render. Runtime probe should emit app_start and first_render markers; analyzer should report startup duration separately from generic events/errors.
Implemented profile runtime errors plus app startup timing. Analyzer parses unified-log NDJSON/plain lines and IAM_ERROR events, groups by severity/process/subsystem/category/signature, emits crash/exception/hang/thread-checker hints. Runtime profile analyzer now reports app_start to first_render duration. Verified with make test, make lint, smoke reports, and Swift helper typecheck.

## Precondition Resources
(none)

## Outcome Resources
(none)
