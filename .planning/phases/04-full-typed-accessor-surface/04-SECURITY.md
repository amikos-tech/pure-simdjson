---
phase: 04
slug: full-typed-accessor-surface
status: verified
threats_open: 0
threats_total: 15
threats_closed: 15
asvs_level: 1
block_on: open
created: 2026-04-17
---

# Phase 04 - Security

> Per-phase security contract: threat register, accepted risks, threat flags, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| public value views -> Rust runtime validation | Malformed or stale transport structs must fail as `ERR_INVALID_HANDLE`, not become dangling native pointers. | `pure_simdjson_value_view_t`, doc handles, descendant tape indexes |
| borrowed native string bytes -> Rust-owned ABI buffers | String copy-out must never depend on C++ allocator symmetry across DLL boundaries. | simdjson string views, Rust-owned copied byte buffers |
| native string allocations -> Go consumers | Successful string reads now include explicit native memory release requirements. | Rust-allocated `ptr + len` buffers, Go string copies |
| C++ bridge errors -> public ABI status codes | Wrong numeric or string error mapping would break the locked Go error semantics. | native errors, ABI status codes, Go sentinels |
| public `Element` methods -> hidden FFI layer | Wrong translation here would hide overflow or precision-loss, or return incorrect sentinel behavior. | public accessors, `ffi.ValueKind`, typed FFI results |
| closed or invalid descendant views -> error-free inspectors | `Type()` and `IsNull()` must remain total without pretending a closed or tampered element is still valid. | element views, close state, sentinel public values |
| public iterator transport structs -> Rust runtime | Iterator tags, reserved bits, and progress state must be validated before native traversal continues. | `pure_simdjson_array_iter_t`, `pure_simdjson_object_iter_t` |
| native iteration -> Go hidden FFI layer | The Go layer must preserve exact iterator state and done flags without exposing raw pointer semantics publicly. | iterator state, descendant views, `done` flags |
| public iterator API -> hidden iterator transport | Public methods must not leak raw iterator assumptions or continue after doc closure. | iterator objects, doc lifecycle state, terminal errors |
| direct field helpers -> scalar string access | The helper must preserve the same missing, null, and wrong-type semantics as the underlying primitives. | field names, descendant views, copied strings |
| user-facing docs/examples -> shipped API | Wrong docs would freeze incorrect usage patterns into the public v0.1 surface. | package docs, examples, exported symbol behavior |
| close-out verification -> release confidence | Missing numeric, UTF-8, race, or close-out coverage would leave the hardest Phase 4 regressions unproven. | local test/build/race proof, release readiness |

---

## Threat Register

| Threat ID | Category | Component | Disposition | Mitigation | Status |
|-----------|----------|-----------|-------------|------------|--------|
| T-04-01-01 | T | descendant-view validation | mitigate | Lock `PSDJROOT` / `PSDJDESC`, reject unknown tags and non-zero reserved state, and route descendants through `doc + json_index` helpers. | closed |
| T-04-01-02 | T | string allocation/free | mitigate | Keep string copy-out allocator ownership in Rust and accept only buffers issued by that path in `bytes_free`. | closed |
| T-04-01-03 | I | error-code mapping | mitigate | Keep bridge-side mappings aligned with the Phase 1 contract for wrong-type, overflow, precision-loss, and invalid JSON cases. | closed |
| T-04-02-01 | T | public type classification | mitigate | Map public `ElementType` directly from hidden `ffi.ValueKind` values and collapse invalid-handle paths to `TypeInvalid`. | closed |
| T-04-02-02 | T | numeric/string accessors | mitigate | Reuse `wrapStatus(...)` for overflow, precision-loss, invalid-json, and wrong-type results without widening or coercing silently. | closed |
| T-04-02-03 | I | closed-state inspectors | mitigate | Keep `Type()` and `IsNull()` total with sentinel or false outputs after `Doc.Close()` or a tampered view. | closed |
| T-04-03-01 | T | iterator-state validation | mitigate | Lock one inline iterator state model, validate tags, reserved bits, doc ownership, and issued iterator positions, and reject malformed states as `ERR_INVALID_HANDLE`. | closed |
| T-04-03-02 | D | iterator allocation | mitigate | Forbid native heap allocation in iterator creation and advance because the ABI has no iterator destructor. | closed |
| T-04-03-03 | I | hidden Go iterator wrappers | mitigate | Mirror the ABI structs exactly in `internal/ffi/types.go`, keep `runtime.KeepAlive(...)` around iterator purego calls, and verify under `-race`. | closed |
| T-04-04-01 | T | iterator closed-state | mitigate | Tie iterators to the owning doc, stop iteration after closure, and surface terminal failure via `Err()`. | closed |
| T-04-04-02 | T | copied object keys | mitigate | Decode key views through the existing hidden string helper so native bytes are copied and freed before `Key()` exposes the result. | closed |
| T-04-04-03 | I | `GetStringField(name)` helper | mitigate | Implement the helper as explicit `GetField(name)` plus `GetString()` composition so behavior cannot drift from the primitives. | closed |
| T-04-05-01 | I | Godoc/examples | mitigate | Document only shipped behavior and back it with executable examples that must compile under `go test`. | closed |
| T-04-05-02 | T | numeric/UTF-8 verification | mitigate | Add explicit table-driven overflow, wrong-type, precision-loss, and malformed UTF-8 cases before phase completion. | closed |
| T-04-05-03 | T | phase-close stability | mitigate | Require the full `cargo test && cargo build --release && go test ./... -race` sweep before the phase can be marked done. | closed |

*Status: open · closed*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Threat Flags

No explicit `## Threat Flags` sections were present in the Phase 04 summary files. The Phase 04 plan threat models remained the source of truth for this audit.

---

## Verification Notes

- `T-04-01-01` is closed by `src/runtime/mod.rs:15-21` and `src/runtime/registry.rs:503-559`, which lock the view tags, reject unknown tags and non-zero reserved bits, validate descendant indices, and reconstruct descendants from `doc + json_index` instead of caller-owned raw pointers.
- `T-04-01-02` is closed by `src/runtime/registry.rs:68-72,698-768`, which now registers every Rust-issued string buffer in runtime-owned allocation state and rejects non-issued or mismatched `ptr,len` pairs before release. `tests/rust_shim_accessors.rs:197-211` proves `pure_simdjson_bytes_free` now refuses a non-issued pointer with `ERR_INVALID_HANDLE`, while the existing round-trip tests still pass for issued buffers.
- `T-04-01-03` is closed by `src/native/simdjson_bridge.cpp:57-103,491-580`, which keep the native wrong-type, out-of-range, precision-loss, and invalid-JSON mappings aligned with the locked ABI contract.
- `T-04-02-01`, `T-04-02-02`, and `T-04-02-03` are closed by `internal/ffi/types.go`, `element.go`, `errors.go`, and `element_scalar_test.go`, which map public type classification directly from hidden `ValueKind`, centralize status translation in `wrapStatus(...)`, and keep `Type()` and `IsNull()` total on closed or tampered views.
- `T-04-03-01` is closed by `src/runtime/registry.rs:51-59,286-345,814-920`, which now issues per-document iterator leases, validates `state0/state1/index/tag` against the currently issued lease, and updates that lease on every successful advance so stale or forged iterator copies fail as `ERR_INVALID_HANDLE`. `tests/rust_shim_iterators.rs:258-314` proves stale copied array/object iterators are rejected after the original iterator advances.
- `T-04-03-02` and `T-04-03-03` are closed by `src/runtime/registry.rs`, `src/native/simdjson_bridge.cpp`, `internal/ffi/types.go`, and `internal/ffi/bindings.go`, which keep iterator state inline, avoid native heap iterator ownership, mirror the ABI structs exactly, and wrap iterator calls with `runtime.KeepAlive(...)`.
- `T-04-04-01`, `T-04-04-02`, and `T-04-04-03` are closed by `iterator.go`, `element.go`, `internal/ffi/bindings.go`, and `iterator_test.go`, which tie iterators to the owning doc, copy and free object keys through the hidden string helper, and implement `GetStringField(name)` as pure composition over the existing primitives.
- `T-04-05-01` and `T-04-05-02` are closed by package docs and examples in `purejson.go`, `parser.go`, `doc.go`, `element.go`, `iterator.go`, `pool.go`, `errors.go`, plus executable and semantic coverage in `example_test.go`, `element_scalar_test.go`, `iterator_test.go`, and `element_fuzz_test.go`.
- `T-04-05-03` is closed by the local close-out sweep rerun after these mitigations on 2026-04-17: `cargo test --test rust_shim_accessors --quiet`, `cargo test --test rust_shim_iterators --quiet`, `cargo test --quiet`, `cargo build --release --quiet`, and `go test ./... -race` all passed. The release build still emitted one non-blocking dead-code warning for `err_internal` in `src/lib.rs`.
- This rerun verified the targeted mitigation changes in `src/runtime/registry.rs`, the ABI contract comments in `include/pure_simdjson.h`, `src/lib.rs`, and `docs/ffi-contract.md`, plus the new regression coverage in `tests/rust_shim_accessors.rs` and `tests/rust_shim_iterators.rs`.

---

## Accepted Risks Log

No accepted risks.

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-04-17 | 15 | 13 | 2 | Codex + gsd-security-auditor |
| 2026-04-17 | 15 | 15 | 0 | Codex (post-mitigation rerun) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-04-17
