# STORY-260301-jpjkaa: e2e-validation

## Description
Final end-to-end validation after all modules migrated to registry + two-phase setup.

1. Clean slate: rm -rf demo, rebuild CLI
2. Run FULL pipeline with two-phase setup (every module shows plan, confirms, scaffolds):
   init → ioc setup → relux setup → secure-store setup → token-provider setup → utilities setup → http-client setup → app-config setup → defaults-store setup → module create Auth --type relux-feature
3. tuist install && tuist generate
4. xcodebuild — BUILD SUCCEEDED
5. Open in Xcode — iOS schemes visible, no red files
6. Verify every module printed usage guide during setup
7. Verify --dry-run shows plan without scaffolding
8. Verify --yes skips confirmation
9. Verify dependency validation catches missing prerequisites (e.g. app-config before ioc → error)
10. Update CLAUDE.md pipeline if needed

This is the final gate — nothing ships until demo app compiles clean with all modules.

## Scope
(define story scope)

## Acceptance Criteria
(define acceptance criteria)
