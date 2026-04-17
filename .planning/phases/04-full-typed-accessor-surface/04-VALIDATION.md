---
phase: 04
slug: full-typed-accessor-surface
status: approved
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-17
---

# Phase 04 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | `cargo test` + `go test` with repo-local native builds |
| **Config file** | `Cargo.toml` + `go.mod` |
| **Quick run command** | `cargo test --lib && go test ./...` |
| **Full suite command** | `cargo test && cargo build --release && go test ./... -race` |
| **Estimated runtime** | ~120 seconds local |

---

## Sampling Rate

- **After every task commit:** Run the task's `<automated>` command.
- **After every plan wave:** Run `cargo test --lib && go test ./...`.
- **Before `$gsd-verify-work`:** `cargo test && cargo build --release && go test ./... -race` must be green.
- **Max feedback latency:** 120 seconds local.

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 04-01-01 | 01 | 1 | API-04 / API-05 / API-06 | T-04-01-01 / T-04-01-02 / T-04-01-03 | Descendant views, scalar accessors, string copy-out, and `bytes_free` reject stale transport state and preserve locked error-code mapping | unit/native | `cargo test --lib` | `src/native/simdjson_bridge.h`, `src/native/simdjson_bridge.cpp`, `src/runtime/mod.rs`, `src/runtime/registry.rs`, `src/lib.rs` | ⬜ pending |
| 04-01-02 | 01 | 1 | API-04 / API-05 / API-06 | T-04-01-02 | Hidden purego wrappers expose uint64/float64/string/bool/null safely and free native string buffers after copy-out | build/integration | `cargo build --release && go test ./...` | `internal/ffi/bindings.go`, `tests/rust_shim_accessors.rs` | ⬜ pending |
| 04-02-01 | 02 | 2 | API-04 / API-05 / API-06 | T-04-02-01 / T-04-02-02 / T-04-02-03 | Public `Element` methods expose exact numeric kinds, explicit closed-state semantics, and no lossy coercion | unit | `cargo build --release && go test ./... -run 'Test(ElementTypeClassification|GetUint64|GetFloat64|GetString|GetBool|IsNull)'` | `element.go` | ⬜ pending |
| 04-02-02 | 02 | 2 | API-04 / API-05 / API-06 | T-04-02-02 | Semantic tests prove overflow, precision-loss, string copy-out, bool/null, and closed-element classification | unit | `cargo build --release && go test ./... -run 'Test(ElementTypeClassification|GetUint64|GetFloat64|GetString|GetBool|IsNull)'` | `element_scalar_test.go` | ⬜ pending |
| 04-03-01 | 03 | 2 | API-07 / API-08 | T-04-03-01 / T-04-03-02 | Iterator transport and object lookup validate tags/reserved bits/doc ownership and stay callback-free | unit/native | `cargo test --lib` | `src/native/simdjson_bridge.h`, `src/native/simdjson_bridge.cpp`, `src/runtime/mod.rs`, `src/runtime/registry.rs`, `src/lib.rs` | ⬜ pending |
| 04-03-02 | 03 | 2 | API-07 / API-08 | T-04-03-03 | Hidden Go iterator mirrors preserve ABI layout exactly and expose typed wrappers for iteration and direct lookup | build/integration | `cargo build --release && go test ./...` | `internal/ffi/types.go`, `internal/ffi/bindings.go`, `tests/rust_shim_iterators.rs` | ⬜ pending |
| 04-04-01 | 04 | 3 | API-05 / API-07 / API-08 | T-04-04-01 / T-04-04-02 / T-04-04-03 | Public iterators are scanner-style, field lookup preserves missing-vs-null, and `GetStringField(name)` composes over the primitive helpers | unit | `cargo build --release && go test ./... -run 'Test(ArrayIterOrder|ObjectIterOrder|ObjectGetFieldMissingVsNull|GetStringField)'` | `element.go`, `iterator.go` | ⬜ pending |
| 04-04-02 | 04 | 3 | API-05 / API-07 / API-08 | T-04-04-01 / T-04-04-02 | Semantic tests prove iteration order, copied object keys, missing-vs-null behavior, helper semantics, and iterator failure after `Doc.Close()` | unit | `cargo build --release && go test ./... -run 'Test(ArrayIterOrder|ObjectIterOrder|ObjectGetFieldMissingVsNull|GetStringField|IteratorAfterDocClose)'` | `iterator_test.go` | ⬜ pending |
| 04-05-01 | 05 | 4 | DOC-03 | T-04-05-01 | Godoc and examples describe only the shipped Phase 4 API and compile under `go test` | doc/example | `go test ./... -run 'Example'` | `purejson.go`, `element.go`, `iterator.go`, `example_test.go` | ⬜ pending |
| 04-05-02 | 05 | 4 | API-04 / API-05 / API-06 / API-07 / API-08 / DOC-03 | T-04-05-02 / T-04-05-03 | Phase-close verification proves numeric edge cases, malformed UTF-8 handling, traversal order, and race safety across the finished native + Go surface | full suite | `cargo test && cargo build --release && go test ./... -race` | `element_scalar_test.go`, `iterator_test.go` | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements.

- `cargo test` already exists for native verification.
- `go test ./...` already exists for public wrapper verification.
- No new test framework installation is required before execution.

---

## Manual-Only Verifications

All planned Phase 4 behaviors have automated verification targets.

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verification commands
- [x] Sampling continuity preserved across waves
- [x] Wave 0 gaps resolved by existing native + Go test infrastructure
- [x] No watch-mode flags
- [x] Local feedback latency target stays under 120 seconds
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-04-17
