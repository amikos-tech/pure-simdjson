---
phase: 02-rust-shim-minimal-parse-path
plan: "03"
subsystem: testing
tags: [ffi, c-smoke, github-actions, msvc, linux, darwin]
requires:
  - phase: 02-01
    provides: "simdjson bridge build plumbing and native library artifacts"
  - phase: 02-02
    provides: "real parser/doc runtime, diagnostics, and minimal int64 parse path"
provides:
  - "Public-header C smoke harness for the minimal literal-42 ABI path plus one invalid-JSON diagnostic case"
  - "Exact Linux and MSVC smoke commands with optional export verification guidance"
  - "Dedicated Linux smoke, Windows/MSVC smoke, and Darwin build-only GitHub Actions workflow"
affects: [phase-04-accessors, phase-06-release-matrix]
tech-stack:
  added: [github-actions]
  patterns: [public-header smoke harness, observed windows-smoke gate, push-scoped CI smoke trigger]
key-files:
  created: [tests/smoke/minimal_parse.c, tests/smoke/README.md, .github/workflows/phase2-rust-shim-smoke.yml]
  modified: [Makefile, cbindgen.toml]
key-decisions:
  - "Kept the smoke proof anchored to include/pure_simdjson.h so the public ABI is validated externally rather than through Rust-only helpers."
  - "Used Windows/MSVC as the second real smoke target and limited Darwin to artifact existence checks to match the Phase 2 proof budget."
  - "Added a smoke-scoped push trigger because GitHub would not workflow_dispatch a brand-new workflow file that existed only on the working branch."
patterns-established:
  - "ABI proof pattern: build the release library, compile an external C harness against the committed header, then require runtime success before claiming compatibility."
  - "CI proof pattern: treat an observed windows-smoke success as the gate, not workflow YAML inspection."
requirements-completed: [SHIM-01, SHIM-02, SHIM-06]
duration: 12min
completed: 2026-04-15
---

# Phase 2 Plan 3: Smoke Proof Summary

**Public-header C smoke proof with a real Linux run, observed Windows/MSVC success, and a Darwin build-only verification workflow**

## Performance

- **Duration:** 12 min
- **Started:** 2026-04-15T13:31:00Z
- **Completed:** 2026-04-15T13:42:50Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments
- Added `tests/smoke/minimal_parse.c`, which proves ABI version negotiation, parser/doc lifecycle, `element_type`, `element_get_int64`, re-parse after `doc_free`, and one invalid-JSON diagnostic path through the committed public header.
- Documented the exact Linux and MSVC compile/run commands and wired Make targets for the Linux smoke path plus optional export checks.
- Added `.github/workflows/phase2-rust-shim-smoke.yml` and observed GitHub Actions run `24457845948` succeed on `linux-smoke`, `windows-smoke`, and `darwin-build-only`, with `windows-smoke=success`.

## Task Commits

Each task was committed as work completed, with one extra fix commit required to unblock remote verification:

1. **Task 1: Add a public-header smoke harness with exact Linux and MSVC commands** - `9448af4` (`feat`)
2. **Task 2: Add the dedicated linux/windows smoke workflow plus Darwin build-only verification** - `44490f7` (`feat`)
3. **Task 2 blocking fix: auto-trigger the smoke workflow on branch pushes** - `25b3822` (`fix`)

No separate metadata commit was created here because the orchestrator owns `STATE.md` and `ROADMAP.md` writes after execution.

## Files Created/Modified
- `tests/smoke/minimal_parse.c` - Native C harness for the Phase 2 happy path and invalid-JSON diagnostic assertion.
- `tests/smoke/README.md` - Exact Linux/MSVC smoke commands and optional export verification commands.
- `.github/workflows/phase2-rust-shim-smoke.yml` - Dedicated Linux, Windows/MSVC, and Darwin smoke/build verification workflow.
- `Makefile` - Added `phase2-smoke-linux`, `phase2-smoke-windows`, and `phase2-verify-exports`.
- `cbindgen.toml` - Excluded private `psimdjson_*` bridge items so `make verify-contract` keeps the public header clean.

## Decisions Made

- Kept the smoke harness intentionally narrow and public-surface-only, using `include/pure_simdjson.h` rather than any internal Rust bridge API.
- Treated the GitHub Actions proof itself as part of the deliverable: the plan is not complete on a passing local Linux run alone.
- Preserved `workflow_dispatch` for future manual runs while adding a narrow `push` trigger solely to overcome GitHub's branch-only workflow dispatch limitation during plan execution.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Excluded private bridge symbols from generated headers**
- **Found during:** Task 1 verification
- **Issue:** `make verify-contract` failed because `cbindgen` was exporting internal `psimdjson_*` types and bridge functions into the public header diff.
- **Fix:** Added an explicit `export.exclude` list in `cbindgen.toml` so only the committed `pure_simdjson_*` ABI remains public.
- **Files modified:** `cbindgen.toml`
- **Verification:** `make verify-contract && cargo build --release && make phase2-smoke-linux`
- **Committed in:** `9448af4`

**2. [Rule 3 - Blocking] Added a push-scoped trigger for branch-only workflow execution**
- **Found during:** Task 2 remote verification
- **Issue:** `gh workflow run phase2-rust-shim-smoke.yml --ref gsd/phase-02-rust-shim-minimal-parse-path` returned HTTP 404 because GitHub would not manually dispatch a newly added workflow that was not yet present on the default branch.
- **Fix:** Added a narrow `push` trigger for smoke-related files, pushed the branch again, and used the observed run as the required Windows proof while keeping `workflow_dispatch` enabled.
- **Files modified:** `.github/workflows/phase2-rust-shim-smoke.yml`
- **Verification:** Observed GitHub Actions run `24457845948` at `https://github.com/amikos-tech/pure-simdjson/actions/runs/24457845948` complete successfully with `windows-smoke=success`.
- **Committed in:** `25b3822`

---

**Total deviations:** 2 auto-fixed (2 blocking)
**Impact on plan:** Both fixes were required to complete the planned verification honestly. Scope stayed within the smoke-proof objective.

## Issues Encountered

- `gh auth status` returned non-zero because of an unrelated timeout on `redacted-github-host.invalid`; repository-host verification was instead done with `gh auth status -h github.com`, which succeeded.
- GitHub Actions emitted Node 20 deprecation annotations for `actions/checkout@v4` and `ilammy/msvc-dev-cmd@v1`, but the smoke workflow still completed successfully.

## Threat Flags

| Flag | File | Description |
|------|------|-------------|
| `threat_flag: ci_trigger` | `.github/workflows/phase2-rust-shim-smoke.yml` | Added a push-triggered CI execution path to work around branch-only `workflow_dispatch` limitations; future CI hardening should keep the path scope narrow. |

## User Setup Required

None - GitHub CLI authentication and write access were already present, and no additional external configuration was needed.

## Next Phase Readiness

- The Phase 2 native surface now has an external proof path: local Linux smoke, documented MSVC commands, and an observed green Windows job.
- Later phases can extend accessors and Go bindings against a smoke-tested public ABI instead of relying on Rust integration tests alone.

## Self-Check: PASSED

- Verified `.planning/phases/02-rust-shim-minimal-parse-path/02-03-SUMMARY.md` exists on disk.
- Verified commits `9448af4`, `44490f7`, and `25b3822` exist in git history.
- Verified GitHub Actions run `24457845948` completed successfully and reported `windows-smoke=success`.

---
*Phase: 02-rust-shim-minimal-parse-path*
*Completed: 2026-04-15*
