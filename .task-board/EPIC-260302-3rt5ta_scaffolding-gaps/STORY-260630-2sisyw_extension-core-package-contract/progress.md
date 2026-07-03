## Status
done

## Assigned To
codex-inline

## Created
2026-06-30T16:48:15Z

## Last Update
2026-07-03T19:34:40Z

## Blocked By
- (none)

## Blocks
- STORY-260630-ig7pof
- STORY-260630-29qaph

## Checklist
(empty)

## Notes
Architecture principle from user: continue pluginized scaffolding. The extension base defines shared contracts/registry only; every concrete extension type is its own plugin, and extension internals move to that extension-specific Core package. Do not pile all extension types into one generator.
Implemented shared extension Core package contract in app-extensions helper. Concrete extension scaffolds now produce thin appex wrapper plus <ExtensionName>Core package with Swift Testing target and root Package.swift path dependency.

## Precondition Resources
(none)

## Outcome Resources
(none)
