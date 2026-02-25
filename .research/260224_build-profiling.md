# Build profiling (Xcode / `xcodebuild`) — research

- Task: `TASK-260224-kboaed`
- Date: **2026-02-24**
- Goal: profile Xcode builds from CLI to identify **slow targets/steps**, **dependency stalls**, and **parallelism issues**; propose an automatable `make build-profile`.

## Highlights / key takeaways

1. **Fast signal:** `xcodebuild -showBuildTimingSummary` prints timings for the *commands invoked during the build* (text output). Useful for “what’s slow right now?”.
2. **Best timeline from CLI:** parse Xcode build logs (`.xcactivitylog` in DerivedData) and emit **HTML + Chrome Trace** via `xclogparser` (XCLogParser). This is the most practical way to visualize stalls/parallel work without opening Xcode.
3. **Important gotcha:** XCLogParser notes that (since Xcode 11) `xcodebuild` only generates `.xcactivitylog` logs when `-resultBundlePath` is present — so always set `-resultBundlePath` for automated profiling.
4. **`.xcresult` is still useful:** use `xcrun xcresulttool` (and optionally `xcparse`) to extract build/test logs, warnings, issues, and attachments from the result bundle. Timing detail is typically better sourced from `.xcactivitylog`.
5. **`make build-profile` is feasible:** run `xcodebuild` with fixed `-derivedDataPath` + `-resultBundlePath`, save stdout/stderr, then run `xclogparser parse` to produce `report.html` + `trace.json`. Optionally dump `xcresulttool get build-results` JSON for machine-readable metadata.

---

## Scope

### In scope
- Build profiling from CLI (local & CI): timing summaries, command/step breakdown, evidence of (lack of) parallelism.
- Data sources: `xcodebuild` outputs, DerivedData build logs, `.xcresult` bundles.
- Tools: `xcrun xcresulttool`, `xclogparser` (XCLogParser), `xcparse` (optional), `tuist graph` (dependency visualization).

### Out of scope
- Runtime profiling (Instruments).
- Simulator runtime control (`simctl`) beyond selecting a destination for tests/builds.

---

## Environment used for fact-checking

These commands were executed locally on **2026-02-24** to confirm flags and tool behavior:

- `xcodebuild -version` → **Xcode 26.2 (Build 17C52)**
- `xcrun xcresulttool help` → “Xcode Result Bundle Tool (version **24514**)”
- `xcodebuild -help` → used to verify profiling-related flags and their descriptions

---

## Questions from task description (answered)

### Can we profile the build plan via `xcodebuild` CLI?
Yes, to varying depths:

- **Build “plan” without execution:** `xcodebuild -dry-run` (“do everything except actually running the commands”).
- **Command-level timings:** `xcodebuild -showBuildTimingSummary`.
- **Structured artifacts for deeper analysis:** `-resultBundlePath` (produces a result bundle) + `-derivedDataPath` (stable log location).

### Can we profile this via `simctl`?
Not directly. `simctl` manages simulators. Build planning/timing is owned by `xcodebuild`/Xcode’s build system. Use `simctl` only as an *execution destination* concern (e.g., for tests), not for build graph/timing.

### Can we extract an Xcode-like build timeline from CLI?
Indirectly, yes:
- parse `.xcactivitylog` into a trace (e.g., `xclogparser --reporter chromeTracer`) and inspect the timeline in `chrome://tracing`.

### Is there a hidden `-buildTimingJSON` (or similar) flag?
Not found in `xcodebuild -help` for Xcode 26.2. The timing summary flag present is `-showBuildTimingSummary` and it does not advertise a JSON output mode.

---

## 1) `xcodebuild` flags for profiling (timings + parallelism)

### Core flags (verified in `xcodebuild -help`, Xcode 26.2)

```text
-parallelizeTargets                 build independent targets in parallel
-jobs NUMBER                        specify the maximum number of concurrent build operations
-dry-run                            do everything except actually running the commands
-resultBundlePath PATH              specifies the directory where a result bundle describing what occurred will be placed
-derivedDataPath PATH               specifies the directory where build products and other derived data will go
-showBuildTimingSummary             display a report of the timings of all the commands invoked during the build
```

Also present (newer / less commonly documented):

```text
-resultStreamPath PATH              specifies the file where a result stream will be written to (the file must already exist)
```

**What you get:**
- `-showBuildTimingSummary` gives a quick per-build textual summary; great for a first-pass “what got slower?”
- `-jobs` and `-parallelizeTargets` let you test if the build is bottlenecked on parallelism vs. dependency chain.
- `-dry-run` helps confirm what would run, without paying compile/link time.
- `-derivedDataPath` makes log/DerivedData locations deterministic for automation.
- `-resultBundlePath` is a key enabler for later steps (`.xcresult`, and also `.xcactivitylog` generation per XCLogParser docs).

**Limitation:** the timing summary is still **text** and **command-oriented**; it’s not a dependency graph visualization. For timeline/parallelism diagnosis, parse `.xcactivitylog` to a trace.

**Sources (this section):**
- Local: `xcodebuild -help` output (Xcode 26.2 / 17C52), captured 2026-02-24.

---

## 2) `.xcresult` bundles (`-resultBundlePath`) and `xcresulttool` (+ optional `xcparse`)

### What `.xcresult` gives you
- A structured “result bundle” directory containing metadata, issues, and logs for the build/test action.
- Queryable via `xcrun xcresulttool` (built into the Xcode toolchain).

### Practical `xcresulttool` queries (verified)

```bash
# High-level build metadata + warnings/issues (JSON)
xcrun xcresulttool get build-results --path path/to/Build.xcresult --compact > build-results.json

# Build/action/console log references (JSON)
xcrun xcresulttool get log --path path/to/Build.xcresult --type build --compact > build-log.json

# JSON schema for build-results output (useful for tooling)
xcrun xcresulttool get build-results --schema --path path/to/Build.xcresult > build-results.schema.json
```

Notes:
- `xcresulttool get object` and `xcresulttool graph` are marked deprecated in `xcresulttool help` output; prefer newer subcommands like `get build-results` and `get log`.

### Optional: `xcparse` (3rd party)
`xcparse` focuses on extracting *artifacts* from `.xcresult` (screenshots, coverage, logs, etc.), which can be handy if you want a simple “export everything” step in CI.

**Sources (this section):**
- Local: `xcrun xcresulttool help`, `xcrun xcresulttool help get`, `xcrun xcresulttool help get build-results`, `xcrun xcresulttool help get log` (Xcode Result Bundle Tool v24514), captured 2026-02-24.
- `xcparse` README: https://github.com/ChargePoint/xcparse

---

## 3) DerivedData build logs (`.xcactivitylog`) + `xclogparser` (XCLogParser)

### Why this matters
The `.xcactivitylog` contains the richest “what happened when” build detail and is the best input for:
- visualizing parallelism and stalls (timeline trace),
- breaking down durations by target/file/step (depending on parser and reporter),
- building a repeatable build-profiling artifact pipeline.

### Key behavior (from XCLogParser docs)
- “The `.xcactivitylog` files are created by Xcode/xcodebuild a few seconds after a build completes … in … `~/Library/Developer/Xcode/DerivedData` … (or the directory specified by the `-derivedDataPath` option) … in `.../Logs/Build/`.” (XCLogParser)
- “Since Xcode 11, `xcodebuild` only generates `.xcactivitylog` build logs when option `-resultBundlePath` is present.” (XCLogParser)

### `xclogparser` usage for profiling reports

Generate an HTML report:

```bash
xclogparser parse --reporter html --workspace YourApp.xcworkspace
```

Generate a Chrome Trace timeline (open in `chrome://tracing`):

```bash
xclogparser parse --reporter chromeTracer --workspace YourApp.xcworkspace
```

For deterministic automation, always provide explicit locations:
- pass `-derivedDataPath` to `xcodebuild`
- pass `--derivedData` (or equivalent) to `xclogparser` so it reads the correct logs

### Bonus: compiler-level hotspots (optional, still in `xclogparser`)
XCLogParser also documents reporters like:
- `compilationTime` / `longCompilationTime` (Swift file compile durations),
- `longFunctionCompilationTime` (long functions),
- `warning`, `error`, `buildStep`, `swiftFunction`, etc.

These are useful if the build bottleneck is mostly “Swift compile time” rather than target dependency structure.

**Sources (this section):**
- `XCLogParser` / `xclogparser` README: https://github.com/MobileNativeFoundation/XCLogParser

---

## 4) Tuist dependency graph (`tuist graph`) as a complement

Build timing tools tell you *what is slow*. A dependency graph helps explain *why parallelism might be limited* (long chains / critical path / oversized modules).

Tuist provides `tuist graph` to “generate and open a graph that represents the targets dependencies.” The docs list output formats like `dot` and `json`, and flags to skip various dependency categories (tests, external deps, SPM deps, static/dynamic frameworks, etc.).

**Sources (this section):**
- Tuist docs (`graph` command): https://docs.tuist.dev/commands/graph

---

## 5) Feasibility: `make build-profile` target (recommended design)

### Goal
One command that:
1) runs a build with profiling artifacts enabled, and
2) emits a **human-readable report** + **timeline trace** into a deterministic artifacts folder.

### Proposed pipeline (minimal, high value)

1) Choose a stable output dir:
- `ARTIFACTS_DIR=artifacts/build-profile/$(date +%Y%m%d_%H%M%S)`
- `DERIVED_DATA_DIR=$(ARTIFACTS_DIR)/DerivedData`
- `RESULT_BUNDLE=$(ARTIFACTS_DIR)/Build.xcresult`

2) Run `xcodebuild` with profiling flags:
- always set `-resultBundlePath` (enables `.xcresult` and per XCLogParser also enables `.xcactivitylog` generation)
- always set `-derivedDataPath` (deterministic logs)
- include `-showBuildTimingSummary` (fast summary)
- optionally toggle `-parallelizeTargets` and `-jobs` for experimentation

3) Parse build logs with `xclogparser`:
- HTML report for quick browsing
- Chrome trace for timeline/parallelism

4) (Optional) Dump machine-readable `.xcresult` slices:
- `xcresulttool get build-results ... > build-results.json`
- `xcresulttool get log --type build ... > build-log.json`

### Example `make` target (sketch)

```make
build-profile:
	@set -euo pipefail; \
	OUT="artifacts/build-profile/$$(date +%Y%m%d_%H%M%S)"; \
	DD="$$OUT/DerivedData"; \
	RB="$$OUT/Build.xcresult"; \
	mkdir -p "$$OUT"; \
	xcodebuild \
	  -workspace YourApp.xcworkspace \
	  -scheme YourScheme \
	  -configuration Debug \
	  -destination 'platform=iOS Simulator,name=iPhone 16' \
	  -derivedDataPath "$$DD" \
	  -resultBundlePath "$$RB" \
	  -showBuildTimingSummary \
	  build | tee "$$OUT/xcodebuild.log"; \
	xclogparser parse --reporter html --derivedData "$$DD" --output "$$OUT/xclogparser-html"; \
	xclogparser parse --reporter chromeTracer --derivedData "$$DD" --output "$$OUT/xclogparser-trace"; \
	xcrun xcresulttool get build-results --path "$$RB" --compact > "$$OUT/build-results.json"
```

### Feasibility assessment

**Feasible now** (no private APIs required):
- All key inputs/outputs are CLI-accessible: `xcodebuild`, DerivedData logs, `.xcresult`, `xcresulttool`, `xclogparser`.
- Produces artifacts suitable for CI retention and comparison over time.

**Caveats / operational details:**
- **Race:** `.xcactivitylog` may be written “a few seconds after a build completes” (per XCLogParser). Parsing immediately after build may require a short retry/backoff or “find newest log file once it appears”.
- **Tooling dependency:** `xclogparser` isn’t built-in; you’ll need to install/pin it (e.g., Homebrew, Mint, or checked-in build toolchain).
- **Interpretation:** `-showBuildTimingSummary` is command-level; use trace/HTML for “who blocked whom?”.
- **Stability:** `-resultStreamPath` exists in help output but is less documented; treat as experimental unless you confirm the stream format and schema for your Xcode version.

**Sources (this section):**
- Local: `xcodebuild -help`, `xcresulttool help` outputs (captured 2026-02-24).
- `XCLogParser` README (log generation + timing + reporters): https://github.com/MobileNativeFoundation/XCLogParser

---

## Recommendation

If the goal is “see what freezes/bottlenecks the build” with minimal effort:

1) Start by adding a `build-profile` automation that always captures:
   - `xcodebuild -showBuildTimingSummary` output (saved to a log file)
   - `.xcresult` bundle (`-resultBundlePath`)
   - DerivedData (`-derivedDataPath`)
2) Make `xclogparser` the default report generator:
   - HTML for quick browsing
   - Chrome trace for parallelism/stalls diagnosis
3) Use `tuist graph` occasionally to correlate slow builds with dependency structure (critical path / large leaf modules), not as a timing source.

