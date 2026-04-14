---
phase: 1
slug: ffi-contract-design
status: draft
nyquist_compliant: true
wave_0_complete: false
created: 2026-04-14
---

# Phase 1 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | staged static contract checks via `cargo check`, `cbindgen`, `rg`, `cc`, and small helper scripts |
| **Config file** | staged: `Cargo.toml` + `cbindgen.toml` in Plan 01, `Makefile` in Plan 03 |
| **Quick run command** | Use the current task's `<automated>` command until Plan 03 Task 2 creates `make verify-contract` |
| **Full suite command** | After Plan 03 Task 2: `make verify-contract && make verify-docs` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After Plan 01 Task 1:** Run `command -v cbindgen >/dev/null && cargo check`
- **After Plan 01 Task 2 and Plan 02 Task 2:** Run the temp-header diff command from the plan task
- **After Plan 02 Task 1:** Run `cargo check`
- **After Plan 03 Task 1:** Run the document grep command from the plan task
- **After Plan 03 Task 2 and before `/gsd-verify-work`:** Run `make verify-contract && make verify-docs`
- **Max feedback latency:** 20 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 1-01-01 | 01 | 0 | FFI-01 | T-01-02 | The ABI-source crate and local generator exist so contract generation starts from committed source, not handwritten headers | bootstrap compile | `command -v cbindgen >/dev/null && cargo check` | `Cargo.toml`, `src/lib.rs` | ⬜ pending |
| 1-01-02 | 01 | 0 | FFI-01 | T-01-01 | The committed header baseline round-trips exactly through `cbindgen` | static diff | `tmp="$(mktemp)" && cbindgen --config cbindgen.toml --crate pure_simdjson --output "$tmp" && diff -u include/pure_simdjson.h "$tmp" && rm -f "$tmp"` | `cbindgen.toml`, `include/pure_simdjson.h` | ⬜ pending |
| 1-02-01 | 02 | 1 | FFI-02, FFI-03, FFI-04, FFI-07, FFI-08 | T-02-01, T-02-02, T-02-03 | The finalized ABI source compiles with exact error codes, handle/view layouts, ABI versioning, and parser-busy lifecycle comments | compile gate | `cargo check` | `src/lib.rs` | ⬜ pending |
| 1-02-02 | 02 | 1 | FFI-02, FFI-03, FFI-04, FFI-07, FFI-08 | T-02-01, T-02-02 | The generated header exposes the finalized symbols and stays byte-for-byte aligned with `src/lib.rs` | static diff | `tmp="$(mktemp)" && cbindgen --config cbindgen.toml --crate pure_simdjson --output "$tmp" && diff -u include/pure_simdjson.h "$tmp" && rm -f "$tmp"` | `include/pure_simdjson.h` | ⬜ pending |
| 1-03-01 | 03 | 2 | FFI-05, FFI-06, DOC-02 | T-03-01 | The normative doc states the unwind policy, parser lifecycle, and split-number accessor rules including `ERR_NUMBER_OUT_OF_RANGE` / `ERR_PRECISION_LOSS` | doc/static grep | `test -f docs/ffi-contract.md && rg '^# Scope|^# ABI invariants|^# Error code space|^# Value and iterator model|^# Parser lifecycle|ffi_fn!|catch_unwind|panic = \"abort\"|\\.get\\(err\\)|PURE_SIMDJSON_ERR_PARSER_BUSY|PURE_SIMDJSON_ERR_NUMBER_OUT_OF_RANGE|PURE_SIMDJSON_ERR_PRECISION_LOSS|pure_simdjson_element_get_int64|pure_simdjson_element_get_uint64|pure_simdjson_element_get_float64|SIMDJSON_PADDING|\\^0\\.1\\.x' docs/ffi-contract.md` | `docs/ffi-contract.md` | ⬜ pending |
| 1-03-02 | 03 | 2 | FFI-01, FFI-02, FFI-03, FFI-04, FFI-05, FFI-06, FFI-07, FFI-08, DOC-02 | T-03-02, T-03-03 | The repository can mechanically detect header drift, ABI-shape regressions, layout regressions, and missing contract prose | static suite | `make verify-contract && make verify-docs` | `Makefile`, `tests/abi/check_header.py`, `tests/abi/handle_layout.c`, `tests/abi/README.md` | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `Cargo.toml` — minimal ABI-source crate for `cbindgen`
- [ ] `src/lib.rs` — exported ABI signatures and repr(C) types only
- [ ] `cbindgen.toml` — stable header-generation config
- [ ] `include/pure_simdjson.h` — generated header target committed in-repo
- [ ] `cargo install --force cbindgen` — local generator install

## Later-Wave Verification Artifacts

- [ ] `docs/ffi-contract.md` — normative contract document created in Plan 03 Task 1
- [ ] `tests/abi/handle_layout.c` — static layout verification created in Plan 03 Task 2
- [ ] `tests/abi/check_header.py` — signature/lint verification created in Plan 03 Task 2
- [ ] `tests/abi/README.md` — requirement-to-rule traceability created in Plan 03 Task 2
- [ ] `Makefile` — `verify-contract` and `verify-docs` entrypoints created in Plan 03 Task 2

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Header and prose contract are reviewable as a single ABI source of truth | DOC-02 | Static checks prove alignment, but a human still needs to confirm the document is clear enough for later phases to implement without ambiguity | Read `docs/ffi-contract.md` side-by-side with `include/pure_simdjson.h`; verify every exported symbol, error-code rule, handle rule, and ownership invariant is explained in prose |
| Panic/exception policy wording does not overclaim release-mode behavior | FFI-05, FFI-06 | Tooling can grep for terms, but only a reviewer can confirm the wording distinguishes unwind-mode error returns from `panic=abort` release policy | Review the panic/exception section in `docs/ffi-contract.md`; confirm it does not promise recoverable release panics |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 20s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
