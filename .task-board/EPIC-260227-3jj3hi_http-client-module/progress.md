## Status
backlog

## Assigned To
(none)

## Created
2026-02-26T21:13:57Z

## Last Update
2026-02-27T10:54:14Z

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
(empty)

## Notes
Architecture update: HttpClient is now pure transport. No dependency on TokenProvider or ApiConfigurator. Feature relux-modules assemble their own API client from HttpClient + TokenProvider + ApiConfigurator, all injected via protocols. See diagrams/scaffolding-pipeline.puml for dependency graph.

## Precondition Resources
(none)

## Outcome Resources
(none)
