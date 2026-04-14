---
phase: 1
slug: ffi-contract-design
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-14
---

# Phase 1 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | static contract checks via `cc`, `cbindgen`, `rg`, and small helper scripts |
| **Config file** | none — Wave 0 installs and scaffolds it |
| **Quick run command** | `make verify-contract` |
| **Full suite command** | `make verify-contract && make verify-docs` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `make verify-contract`
- **After every plan wave:** Run `make verify-contract && make verify-docs`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 20 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 1-01-01 | 01 | 1 | FFI-01 | T-1-01 | Generated header remains aligned with the Rust ABI source | static diff | `cbindgen --config cbindgen.toml --crate pure_simdjson --output /tmp/pure_simdjson.h && diff -u include/pure_simdjson.h /tmp/pure_simdjson.h` | ❌ W0 | ⬜ pending |
| 1-01-02 | 01 | 1 | FFI-02, FFI-03 | T-1-02 | Every export uses `int32_t` + out-params and never mixes float/int args | static lint | `python3 tests/abi/check_header.py --rule int32-outparams --rule no-mixed-float-int include/pure_simdjson.h` | ❌ W0 | ⬜ pending |
| 1-01-03 | 01 | 1 | FFI-04 | T-1-01 | Handle layout stays packed `{slot:u32, gen:u32}` and stale handles are rejectable | compile/static_assert | `cc -Iinclude tests/abi/handle_layout.c -c -o /tmp/handle_layout.o` | ❌ W0 | ⬜ pending |
| 1-01-04 | 02 | 2 | FFI-05, FFI-06 | T-1-03 | Contract text and ABI source document the panic/exception boundary accurately | grep/doc check | `rg 'ffi_fn!|catch_unwind|panic *= *\"abort\"|\\.get\\(' docs/ffi-contract.md src include/pure_simdjson.h` | ❌ W0 | ⬜ pending |
| 1-01-05 | 02 | 2 | FFI-07, FFI-08, DOC-02 | T-1-04 | ABI version handshake, Rust-owned padded copy rule, and normative contract doc all exist together | file/grep check | `test -f docs/ffi-contract.md && rg 'get_abi_version|\\^0\\.1\\.x|Rust-owned|SIMDJSON_PADDING' docs/ffi-contract.md include/pure_simdjson.h` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `Cargo.toml` — minimal ABI-source crate for `cbindgen`
- [ ] `src/lib.rs` — exported ABI signatures and repr(C) types only
- [ ] `cbindgen.toml` — stable header-generation config
- [ ] `include/pure_simdjson.h` — generated header target committed in-repo
- [ ] `docs/ffi-contract.md` — normative contract document
- [ ] `tests/abi/handle_layout.c` — static layout verification
- [ ] `tests/abi/check_header.py` — signature/lint verification
- [ ] `Makefile` — `verify-contract` and `verify-docs` entrypoints
- [ ] `cargo install --force cbindgen` — local generator install

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
