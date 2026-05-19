# Build Profiling Diagnostics Research

## Goal

Provide a local `ios-app-manager` workflow that explains why a Tuist/Xcode build is slow:

- what commands and targets consume wall/work time;
- which targets can or cannot build in parallel;
- which dependencies block downstream work;
- which build settings or scheme knobs should be checked next.

## Primary Sources

- `xcodebuild -help` on Xcode 26.4 exposes:
  - `-showBuildTimingSummary` for command timing summaries;
  - `-parallelizeTargets` for independent target parallelism;
  - `-jobs NUMBER` for maximum concurrent build operations;
  - `-resultBundlePath PATH` for `.xcresult` generation;
  - `-derivedDataPath PATH` for reproducible artifact location.
- `xcresulttool get build-results --path <bundle>` returns high-level build result JSON.
- `xcresulttool get log --path <bundle> --type build` returns the build log from a result bundle.
- Tuist Build Insights require `tuist xcodebuild` and `-resultBundlePath`; without a result bundle, Tuist cannot analyze build activity logs.
- `tuist graph --format legacyJSON --no-open --path <root> --output-path <dir>` provides a local target dependency graph suitable for critical-path analysis.

## Findings

- Xcode timing summary is the lowest-friction local source for "what took long" because it is emitted by `xcodebuild` without extra dependencies.
- `.xcresult` is useful as an artifact handle and for future richer parsing, but Apple's modern `xcresulttool get build-results` currently exposes build issues and metadata more directly than detailed command timing.
- `.xcactivitylog` contains detailed build operation events, but it is compressed/private-ish and best treated as an optional later parser, not the MVP foundation.
- `tuist inspect build` is useful for Tuist dashboard analytics, but it is not enough for this tool's local terminal-first workflow.
- Parallelism diagnosis needs both time data and dependency graph data. Timing alone identifies expensive work; graph alone identifies possible blockers. Combining both gives critical-path estimates.

## MVP Architecture

Add `ios-app-manager profile build`.

Inputs:

- project config for defaults (`app_name`, `product_name`, project root);
- optional existing raw build log via `--log`;
- optional graph JSON via `--graph-json`;
- otherwise run `tuist generate --no-open`, `tuist graph`, and `xcodebuild`.

Build command defaults:

```bash
xcodebuild \
  -workspace "<AppName>.xcworkspace" \
  -scheme "<ProductName or AppName>" \
  -configuration Debug \
  -destination "generic/platform=iOS Simulator" \
  -derivedDataPath ".temp/build-profile/<run>/DerivedData" \
  -resultBundlePath ".temp/build-profile/<run>/Build.xcresult" \
  -parallelizeTargets \
  -showBuildTimingSummary \
  build
```

Analysis model:

- `TimingEntry`: command, target, project, duration.
- `TargetWork`: sum of timing durations by target.
- `TargetGraph`: target dependencies from Tuist graph.
- `CriticalPath`: longest weighted dependency path using target work as node weight.
- `Parallelism`: total target work / critical-path time as ideal parallelism ceiling.

Output:

- top slow commands;
- target work table;
- critical path and dependency blockers;
- raw artifact paths.

## Non-Goals For MVP

- exact Xcode scheduler reconstruction;
- direct `.xcactivitylog` binary parsing;
- cloud upload;
- profiler UI.

## Follow-Up

- Parse result bundle build log directly when `--result-bundle-path` is supplied.
- Add `.xcactivitylog` parser if timing summary proves insufficient.
- Add JSON schema stability tests for machine-readable report output.
