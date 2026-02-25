# STORY-260224-2wlfpb: deep-clean

## Description
Deep clean functionality — thorough cleanup of all cached/derived artifacts that accumulate and cause stale builds.

What to clean:
- Local DerivedData (./DerivedData)
- Global DerivedData (~/Library/Developer/Xcode/DerivedData)
- SPM caches (~/Library/org.swift.swiftpm, ~/Library/Caches/org.swift.swiftpm)
- Tuist cache (if applicable)
- Stale .build directories
- Package.resolved (force fresh resolution)
- Xcode index/build caches
- SourceKit caches

Exposed via make clean (quick) and make deep-clean (nuclear option).

Reference: membrana Makefile clean target (see .research/ref_membrana-makefile).

Should also kill Xcode before cleaning to avoid file locks (like membrana kill_xcode target).

## Scope
(define story scope)

## Acceptance Criteria
(define acceptance criteria)
