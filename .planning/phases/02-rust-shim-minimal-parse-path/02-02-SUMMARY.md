---
phase: 02-rust-shim-minimal-parse-path
plan: "02"
subsystem: runtime
tags: [rust, simdjson, ffi, registry, diagnostics, testing]
requires:
  - phase: 02-01
    provides: "simdjson bridge build, native exception containment, and bridge test hook"
provides:
  - "Generation-checked parser/doc registry with explicit Idle/Busy lifecycle"
  - "Real minimal parse path with padded Rust-owned input, implementation-name helpers, and parser diagnostics"
  - "Deterministic lifecycle, C++ exception, and fallback CPU-gate integration tests"
affects: [02-03-smoke-proof, phase-04-accessors]
tech-stack:
  added: [none]
  patterns: [packed-handle registry, ffi_wrap on every export, hidden env-gated fallback testing]
key-files:
  created: [src/runtime/mod.rs, src/runtime/registry.rs, tests/rust_shim_minimal.rs, tests/rust_shim_fallback_gate.rs, .gitignore]
  modified: [src/lib.rs, Cargo.toml]
key-decisions:
  - "Used a single global registry mutex so handle validation and native pointer use stay in one critical section."
  - "Stored the zero-padded input arena inside each live doc entry to preserve simdjson's padded-buffer lifetime contract."
  - "Kept fallback override and forced-exception coverage hidden to Rust tests without changing the public C ABI."
patterns-established:
  - "Parser/doc safety: validate slot+generation before every native pointer access and clear Busy only from doc_free."
  - "Phase 2 scope guard: only parser/doc lifecycle, implementation-name helpers, diagnostics, element_type, and element_get_int64 are real; later accessors remain explicit stubs."
requirements-completed: [SHIM-05, SHIM-06, SHIM-07]
duration: 21min
completed: 2026-04-15
---

# Phase 2 Plan 2: Runtime Core Summary

**Generation-checked parser/doc runtime with a real padded-input simdjson parse path, thin diagnostics, root-kind mapping, and hidden fallback gating**

## Performance

- **Duration:** 21 min
- **Started:** 2026-04-15T13:06:00Z
- **Completed:** 2026-04-15T13:26:51Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments
- Replaced the parser/doc handle stubs with a real registry-backed lifecycle that enforces `Idle`/`Busy`, stale-handle rejection, and doc-owned parser release semantics.
- Wired the minimal real Phase 2 data path: implementation-name helpers, parser create/free/parse, parser last-error helpers, doc root, `element_type`, and `element_get_int64`.
- Added integration coverage for lifecycle edges, invalid JSON diagnostics, C++ exception containment, and deterministic fallback CPU rejection/bypass behavior.

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement the precise parser/doc state machine and mechanical Rust panic containment** - `3d7ddbc` (`feat`)
2. **Task 2: Wire the minimal parse path, real diagnostics spine, and padding contract** - `c929001` (`feat`)

No separate metadata commit was created here because the orchestrator owns `STATE.md` and `ROADMAP.md` writes after execution.

## Files Created/Modified
- `src/runtime/mod.rs` - Internal bridge bindings, implementation-name helpers, padding lookup, native parse helpers, and hidden test hooks.
- `src/runtime/registry.rs` - Global packed-handle registry with `ParserState::{Idle, Busy}` and validated root-view access.
- `src/lib.rs` - Real Phase 2 exports routed through `ffi_wrap`, fallback CPU gate, and retained explicit stubs for deferred accessors.
- `tests/rust_shim_minimal.rs` - Integration tests for ABI version, implementation name, minimal parse path, diagnostics, lifecycle errors, and the C++ exception hook.
- `tests/rust_shim_fallback_gate.rs` - Serialized env-driven fallback gate tests.
- `Cargo.toml` - Added `rlib` output so integration tests can link the crate.
- `.gitignore` - Ignores generated build/test artifacts created during verification.

## Decisions Made

- Used the packed handle itself plus a root-view tag/pointer check to reject stale or forged value views before calling native element accessors.
- Evaluated `PURE_SIMDJSON_TEST_FORCE_IMPLEMENTATION` and `PURE_SIMDJSON_ALLOW_FALLBACK_FOR_TESTS` at `parser_new` call time only, keeping them undocumented and test-only in practice.
- Left the broader accessor and iterator surface as explicit internal-error stubs so Phase 2 stays confined to the minimal parse path.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Passed the logical JSON length instead of the padded buffer length into simdjson**
- **Found during:** Task 2 (minimal parse-path verification)
- **Issue:** The first Task 2 test run passed `input_len + padding` into the bridge, so simdjson parsed trailing zero bytes and returned `PURE_SIMDJSON_ERR_INVALID_JSON` for valid literals.
- **Fix:** Kept the zero-padded `Vec<u8>` alive in the doc entry but passed only the logical JSON prefix length to `psimdjson_parser_parse`.
- **Files modified:** `src/runtime/registry.rs`
- **Verification:** `cargo test --test rust_shim_minimal`
- **Committed in:** `c929001`

**2. [Rule 3 - Blocking] Added `rlib` output so Cargo integration tests can link the crate**
- **Found during:** Task 2 (first `cargo test --test rust_shim_minimal` run)
- **Issue:** With only `cdylib` and `staticlib`, Cargo could not resolve the crate from `tests/*.rs`.
- **Fix:** Added `rlib` to `crate-type`.
- **Files modified:** `Cargo.toml`
- **Verification:** `cargo test --test rust_shim_minimal`
- **Committed in:** `c929001`

**3. [Rule 3 - Blocking] Added ignores for generated verification artifacts**
- **Found during:** Task 2 close-out
- **Issue:** `target/`, `Cargo.lock`, `.playwright-mcp/`, and `tests/abi/__pycache__/` were left untracked by the verification loop, which would have polluted the task commit boundary.
- **Fix:** Added a minimal `.gitignore` covering the generated artifacts.
- **Files modified:** `.gitignore`
- **Verification:** `git status --short`
- **Committed in:** `c929001`

---

**Total deviations:** 3 auto-fixed (1 bug, 2 blocking)
**Impact on plan:** All deviations were required for correctness or clean execution. No scope creep beyond the planned Phase 2 runtime surface.

## Issues Encountered

- simdjson accepted the padded buffer contract once the bridge received the logical JSON length instead of the padded allocation length.
- Cargo integration tests required an `rlib` artifact even though the shipped library formats remain `cdylib` and `staticlib`.

## Known Stubs

- `src/lib.rs:557` - `pure_simdjson_element_get_uint64` remains an explicit Phase 4 stub by plan.
- `src/lib.rs:576` - `pure_simdjson_element_get_float64` remains an explicit Phase 4 stub by plan.
- `src/lib.rs:599` - `pure_simdjson_element_get_string` remains an explicit Phase 4 stub by plan.
- `src/lib.rs:618` - `pure_simdjson_bytes_free` remains an explicit Phase 4 stub by plan.
- `src/lib.rs:637` - `pure_simdjson_element_get_bool` remains an explicit Phase 4 stub by plan.
- `src/lib.rs:656` - `pure_simdjson_element_is_null` remains an explicit Phase 4 stub by plan.
- `src/lib.rs:675` - `pure_simdjson_array_iter_new` remains an explicit Phase 4 stub by plan.
- `src/lib.rs:695` - `pure_simdjson_array_iter_next` remains an explicit Phase 4 stub by plan.
- `src/lib.rs:714` - `pure_simdjson_object_iter_new` remains an explicit Phase 4 stub by plan.
- `src/lib.rs:735` - `pure_simdjson_object_iter_next` remains an explicit Phase 4 stub by plan.
- `src/lib.rs:757` - `pure_simdjson_object_get_field` remains an explicit Phase 4 stub by plan.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The native library now exposes a real minimal parse path that a C smoke harness can drive directly in Plan 03.
- The remaining risk is build/smoke proof on the planned Linux and Windows targets, not the parser/doc runtime core.

## Self-Check: PASSED

- Verified summary file exists on disk.
- Verified task commits `3d7ddbc` and `c929001` exist in git history.

---
*Phase: 02-rust-shim-minimal-parse-path*
*Completed: 2026-04-15*
