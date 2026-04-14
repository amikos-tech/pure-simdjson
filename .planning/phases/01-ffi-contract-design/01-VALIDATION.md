---
phase: 1
slug: ffi-contract-design
status: approved
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-14
---

# Phase 1 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | static contract checks via `cargo check`, `cbindgen`, `rg`, `cc`, and `python3 tests/abi/check_header.py` |
| **Config file** | `Cargo.toml`, `cbindgen.toml`, `Makefile` |
| **Quick run command** | `make verify-contract` |
| **Full suite command** | `make verify-contract && make verify-docs` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After ABI or header changes:** Run `make verify-contract`
- **After contract-doc changes:** Run `make verify-docs`
- **Before `/gsd-verify-work` or milestone audit:** Run `make verify-contract && make verify-docs`
- **Max feedback latency:** 20 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 1-01-01 | 01 | 0 | FFI-01 | T-01-02 | The ABI-source crate and local generator exist so contract generation starts from committed source, not handwritten headers | bootstrap compile | `command -v cbindgen >/dev/null && cargo check` | `Cargo.toml`, `src/lib.rs` | ✅ green |
| 1-01-02 | 01 | 0 | FFI-01 | T-01-01 | The committed header baseline round-trips exactly through `cbindgen` | static diff | `tmp="$(mktemp)" && cbindgen --config cbindgen.toml --crate pure_simdjson --output "$tmp" && diff -u include/pure_simdjson.h "$tmp" && rm -f "$tmp"` | `cbindgen.toml`, `include/pure_simdjson.h` | ✅ green |
| 1-02-01 | 02 | 1 | FFI-02, FFI-03, FFI-04, FFI-07, FFI-08 | T-02-01, T-02-02, T-02-03 | The finalized ABI source compiles with exact error codes, handle/view layouts, ABI versioning, and parser-busy lifecycle comments | compile gate | `cargo check` | `src/lib.rs` | ✅ green |
| 1-02-02 | 02 | 1 | FFI-02, FFI-03, FFI-04, FFI-07, FFI-08 | T-02-01, T-02-02 | The generated header exposes the finalized symbols and stays byte-for-byte aligned with `src/lib.rs` | static diff | `tmp="$(mktemp)" && cbindgen --config cbindgen.toml --crate pure_simdjson --output "$tmp" && diff -u include/pure_simdjson.h "$tmp" && rm -f "$tmp"` | `include/pure_simdjson.h` | ✅ green |
| 1-03-01 | 03 | 2 | FFI-05, FFI-06, DOC-02 | T-03-01 | The normative doc states the unwind policy, parser lifecycle, and split-number accessor rules including `ERR_NUMBER_OUT_OF_RANGE` / `ERR_PRECISION_LOSS` | doc/static grep | `test -f docs/ffi-contract.md && rg '^# Scope|^# ABI invariants|^# Error code space|^# Value and iterator model|^# Parser lifecycle|ffi_fn!|catch_unwind|panic = \"abort\"|\\.get\\(err\\)|PURE_SIMDJSON_ERR_PARSER_BUSY|PURE_SIMDJSON_ERR_NUMBER_OUT_OF_RANGE|PURE_SIMDJSON_ERR_PRECISION_LOSS|pure_simdjson_element_get_int64|pure_simdjson_element_get_uint64|pure_simdjson_element_get_float64|SIMDJSON_PADDING|\\^0\\.1\\.x' docs/ffi-contract.md` | `docs/ffi-contract.md` | ✅ green |
| 1-03-02 | 03 | 2 | FFI-01, FFI-02, FFI-03, FFI-04, FFI-05, FFI-06, FFI-07, FFI-08, DOC-02 | T-03-02, T-03-03 | The repository can mechanically detect header drift, ABI-shape regressions, layout regressions, and missing contract prose | static suite | `make verify-contract && make verify-docs` | `Makefile`, `tests/abi/check_header.py`, `tests/abi/handle_layout.c`, `tests/abi/README.md` | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] `Cargo.toml` — minimal ABI-source crate for `cbindgen`
- [x] `src/lib.rs` — exported ABI signatures and repr(C) types only
- [x] `cbindgen.toml` — stable header-generation config
- [x] `include/pure_simdjson.h` — generated header target committed in-repo
- [x] `cargo install --force cbindgen` — local generator install

## Later-Wave Verification Artifacts

- [x] `docs/ffi-contract.md` — normative contract document created in Plan 03 Task 1
- [x] `tests/abi/handle_layout.c` — static layout verification created in Plan 03 Task 2
- [x] `tests/abi/check_header.py` — signature/lint verification created in Plan 03 Task 2
- [x] `tests/abi/README.md` — requirement-to-rule traceability created in Plan 03 Task 2
- [x] `Makefile` — `verify-contract` and `verify-docs` entrypoints created in Plan 03 Task 2

---

## Manual-Only Verifications

All phase behaviors have automated verification.

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 20s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-04-14

---

## Validation Audit 2026-04-14

| Metric | Count |
|--------|-------|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |

Fresh audit evidence:

- `command -v cbindgen >/dev/null && cargo check` passed.
- `tmp="$(mktemp)" && cbindgen --config cbindgen.toml --crate pure_simdjson --output "$tmp" && diff -u include/pure_simdjson.h "$tmp" && rm -f "$tmp"` passed.
- `make verify-contract` passed, including header-lint and layout-assertion checks.
- `make verify-docs` passed against the required contract clauses in `docs/ffi-contract.md`.
