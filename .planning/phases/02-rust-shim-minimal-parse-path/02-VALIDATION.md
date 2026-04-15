---
phase: 02
slug: rust-shim-minimal-parse-path
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-15
---

# Phase 02 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | `cargo test` + native C smoke harness |
| **Config file** | none |
| **Quick run command** | `cargo test` |
| **Full suite command** | `make verify-contract && cargo build --release` |
| **Estimated runtime** | ~30 seconds before Windows smoke |

---

## Sampling Rate

- **After every task commit:** Run `cargo test`
- **After every plan wave:** Run `make verify-contract && cargo build --release`
- **Before `/gsd-verify-work`:** Full suite plus the native C smoke harnesses must be green
- **Max feedback latency:** 60 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 02-01-01 | 01 | 1 | SHIM-02 / SHIM-03 | — | Build scripts use pinned vendored simdjson inputs only | build | `cargo build --release` | ❌ W0 | ⬜ pending |
| 02-02-01 | 02 | 1 | SHIM-01 / SHIM-04 / SHIM-05 / SHIM-07 | T-02-01 / — | Runtime selects simdjson implementation safely and rejects fallback without UB | unit | `cargo test` | ✅ | ⬜ pending |
| 02-02-02 | 02 | 1 | SHIM-06 | T-02-02 / — | Parser/doc lifecycle rejects stale or busy handles cleanly | unit | `cargo test` | ✅ | ⬜ pending |
| 02-03-01 | 03 | 2 | SHIM-06 | — | C smoke harness proves `parser_new -> parser_parse -> doc_root -> element_get_int64` on produced artifacts | integration | `make verify-contract && cargo build --release` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `build.rs` — vendored simdjson build pipeline exists and is exercised in CI/local build
- [ ] `tests` coverage for parser busy, stale handles, and minimal int64 happy path
- [ ] `tests/smoke/` or equivalent native C smoke harness directory
- [ ] Windows smoke invocation path documented in repo commands or CI workflow

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Forced fallback-kernel path | SHIM-07 | Requires a hidden test-only bypass or controlled environment not guaranteed in normal CI | Run the dedicated fallback-gate test path and verify `parser_new` returns `PURE_SIMDJSON_ERR_CPU_UNSUPPORTED` without the bypass |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
