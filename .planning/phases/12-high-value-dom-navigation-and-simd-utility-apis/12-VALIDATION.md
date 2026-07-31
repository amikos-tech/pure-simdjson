---
phase: 12
slug: high-value-dom-navigation-and-simd-utility-apis
status: approved
nyquist_compliant: true
wave_0_complete: false
created: 2026-07-30
revised: 2026-07-31
plan_count: 10
wave_count: 8
---

# Phase 12 — Validation Strategy

> Regenerated from the revision-1 plans after checker feedback. This is the authoritative
> task/wave map; the planning-local Spike 004 script is not an execution dependency.

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Frameworks** | Go `testing` (stdlib), Rust integration tests, C smoke, Python `unittest`, ASan/UBSan native probe |
| **Task feedback target** | Focused commands below, normally under 30 seconds on an incremental checkout |
| **Native wave gate** | `make verify-contract` after Waves 1–5 |
| **Go integration gate** | One `cargo build --release && go test ./... -race` after both Wave 6 plans |
| **Phase gate** | `make verify-contract`, `make verify-docs`, release build, Go race suite, native smoke, workflow contracts, and hosted five-platform/D-14 jobs |
| **Full-gate budget** | 60–120 seconds locally; intentionally reserved for wave/phase boundaries |

## Sampling Rate

- **After every task:** run that task's focused `<verify><automated>` command. No task command pipes
  a compiler through `tail`; compiler exit status is authoritative.
- **After Waves 1–5:** run `make verify-contract` once per completed native wave.
- **After Wave 6:** run `make verify-contract && make verify-docs`, then one fresh
  `cargo build --release && go test ./... -race`. Also run
  `python3 scripts/release/test_release_workflow_contracts.py`.
- **After Wave 7:** run the focused public-navigation suite against the Wave 6 release library.
- **After Wave 8 / before verification:** run the complete phase gate and require the current
  Phase 12 branch's five-platform Go workflow plus Linux D-14 job to be green.

## Execution Waves

| Wave | Plans | Dependency result |
|------|-------|-------------------|
| 1 | 12-09 | ABI 1.3/status-code foundation |
| 2 | 12-01 | Native AtPointer/AtPath |
| 3 | 12-02 | Native Array.At/Len/Object.Size |
| 4 | 12-03 | Native wildcard transport |
| 5 | 12-04 | Native Minify/ValidateUTF8 |
| 6 | 12-05, 12-10 | Go bindings/bootstrap docs and CI/smoke/docs closeout; zero file overlap |
| 7 | 12-06 | Public navigation API |
| 8 | 12-07, 12-08 | Public utilities and indexed/container API; zero file overlap |

## Per-Task Verification Map

| Task ID | Wave | Requirement | Secure behavior | Focused automated command | Pre-execution artifact |
|---------|------|-------------|-----------------|---------------------------|------------------------|
| 12-09-T1 | 1 | all | ABI/status values cannot drift across Rust/generated C/C layout | Rust numeric tests + header generation + handle-layout compile | existing |
| 12-09-T2 | 1 | all | Python/header/docs pins require ABI 1.3 and statuses 11/12 | Python header tests + `diag-surface` + `make verify-docs` | existing |
| 12-01-T1 | 2 | DOM-01/02 | Native pointer/path bridge preserves typed status mapping | `cargo build` | existing sources |
| 12-01-T2 | 2 | DOM-01/02 | Pointer/path behavior and exact signatures | Rust navigation test + focused header rules | `tests/rust_shim_navigation.rs` W0 |
| 12-02-T1 | 3 | DOM-04 | Array bounds/kind and zero-size paths are checked natively | `cargo build` | existing sources |
| 12-02-T2 | 3 | DOM-04 | Empty/wrong-kind size behavior plus exact signatures | Rust navigation test + focused header rules | navigation test W0 |
| 12-03-T1 | 4 | DOM-03 | Scratch indices are synchronously copied into tracked owned views | `cargo build` | existing sources |
| 12-03-T2 | 4 | DOM-03 | Ordered wildcard, free discipline, lifetime, and concurrency | Rust navigation test + focused header rules | navigation test W0 |
| 12-04-T1 | 5 | UTIL-01/02 | Capacity/overlap/CPU gate precedes scanning | `cargo build` | existing sources |
| 12-04-T2 | 5 | UTIL-01/02 | Raw boundaries, exact alias, Rust pre-FFI fallback, successful native lock | Rust minify + utility-lock tests + focused header rules | two Rust tests W0 |
| 12-05-T1 | 6 | all | Go ABI/bootstrap identity and current source docs agree | focused internal Go tests + `make verify-docs` | existing |
| 12-05-T2 | 6 | all | All nine bindings marshal pointer/count/capacity safely | `go build ./... && go vet ./...` | existing |
| 12-05-T3 | 6 | all | ABI 1.2 rejects; complete 1.3 binds; 1.4 remains additive | release-policy unittest + focused binding/loader tests | existing |
| 12-10-T1 | 6 | UTIL-01 D-14 | Durable probe is dynamic and its own paths trigger Linux smoke | shell syntax + focused workflow-contract test | probe/script W0 |
| 12-10-T2 | 6 | all | C smoke invokes all nine exports; Phase 12 runs five-platform Go gate | C compile + workflow contracts + `make verify-docs` | existing C/workflows |
| 12-06-T1 | 7 | DOM-01/02/03, UTIL-01 | Public sentinels map 1:1 | `go build . && go vet .` | existing |
| 12-06-T2 | 7 | DOM-01/02/03 | Public methods enforce documented grammar/branch behavior | `go build . && go vet .` | existing |
| 12-06-T3 | 7 | DOM-01/02/03 | RFC edges, proven ErrWrongType traversal, wildcard/lifetime cases | focused root navigation tests | `element_pointer_test.go` W0 |
| 12-07-T1 | 8 | UTIL-01/02 | Go prechecks and native lock state remain aligned | `go build . && go vet .` | three Go sources W0 |
| 12-07-T2 | 8 | UTIL-01/02 | Alias/capacity/UTF-8/CPU states are public-test pinned | focused root utility tests | two Go tests W0 |
| 12-08-T1 | 8 | DOM-04 | Negative/out-of-range/type/lifecycle behavior is explicit | `go build . && go vet .` | existing |
| 12-08-T2 | 8 | DOM-04 | Direct forged wrong-kind At/LenErr/SizeErr guards are exercised | focused root indexed/container tests | `element_indexed_test.go` W0 |

## Wave 0 Requirements

- [ ] `tests/rust_shim_navigation.rs` — pointer/path, indexed/size, wildcard ownership/lifetime,
  concurrency, and exact status tests.
- [ ] `tests/rust_shim_minify.rs` — raw pointer, capacity, alias/disjoint/partial-overlap, malformed,
  UTF-8, and forced-fallback tests.
- [ ] `tests/rust_shim_utility_lock.rs` — narrowly proves Rust pre-FFI fallback rejection leaves
  native state untouched, then proves a successful native utility gate locks selection.
- [ ] `tests/native/minify_buffer_safety_probe.cpp` and
  `scripts/ci/verify_minify_buffer_safety.sh` — durable dynamic D-14 gate.
- [ ] `element_pointer_test.go`, `minify_test.go`, `utf8_test.go`,
  `element_indexed_test.go` — public Go behavior tests.
- [ ] `tests/abi/check_header.py` / `tests/abi/test_check_header.py` — all nine required names and
  exact signatures.
- [ ] `scripts/release/test_release_workflow_contracts.py` — durable path triggers, Phase 12 branch
  trigger, five job identities, and nine smoke calls.

## Hosted Automated Gates

| Gate | Trigger | Required evidence |
|------|---------|-------------------|
| D-14 Linux x86-64 | Push changing native/source/script/probe paths | `phase2-rust-shim-smoke` linux-smoke runs `scripts/ci/verify_minify_buffer_safety.sh`; fallback plus haswell/westmere; three sanitizer-clean runs |
| Five-platform Go wrapper | Every push to `gsd/phase-12-high-value-dom-navigation-and-simd-utility-apis` | linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64 race jobs all green |
| Release native smoke contract | Existing `run_native_smoke.sh` on staged artifacts | `ffi_export_surface.c` resolves and calls every ABI 1.3 export before any release claim |

## Manual-Only Verification

| Behavior | Requirement | Why manual | Instructions |
|----------|-------------|------------|--------------|
| `icelake` AVX-512 aliasing safety | D-14 | Hosted x86 runners do not reliably expose AVX-512 | On an AVX-512 x86-64 host run `bash scripts/ci/verify_minify_buffer_safety.sh`; confirm `icelake` appears in the stable supported-kernel list and all three ASan/UBSan runs are clean |

## Multi-Source Coverage Audit

| Source | ID | Feature / constraint | Plans | Status |
|--------|----|----------------------|-------|--------|
| GOAL | — | Thin high-value DOM navigation, indexed/container helpers, wildcard, minify, UTF-8 validation | 01–10 | COVERED |
| REQ | DOM-01 | RFC 6901 AtPointer | 01, 05, 06, 09, 10 | COVERED |
| REQ | DOM-02 | simdjson dot/index AtPath subset | 01, 05, 06, 09, 10 | COVERED |
| REQ | DOM-03 | ordered document-tied wildcard results | 03, 05, 06, 09, 10 | COVERED |
| REQ | DOM-04 | indexed array access plus array/object size | 02, 05, 08, 09, 10 | COVERED |
| REQ | UTIL-01 | allocation-conscious minify with explicit alias contract | 04, 05, 06, 07, 09, 10 | COVERED |
| REQ | UTIL-02 | standalone ValidateUTF8 with parser validation unchanged | 04, 05, 07, 09, 10 | COVERED |
| CONTEXT | D-01 | Two new navigation sentinels; reuse missing/wrong-type | 09, 01, 05, 06 | COVERED |
| CONTEXT | D-02 | AtPathAll requires `*`; valid branch misses become non-nil empty | 03, 05, 06 | COVERED |
| CONTEXT | D-03 | Shared out-of-range sentinel and document-tied views | 01, 02, 03, 06, 08 | COVERED |
| CONTEXT | D-04 | Array.At returns `(Element,error)` only | 08 | COVERED |
| CONTEXT | D-05 | Go int index; negative rejected | 08 | COVERED |
| CONTEXT | D-06 | Len/LenErr and Size/SizeErr dual methods | 08 | COVERED |
| CONTEXT | D-07 | Array.At out-of-range uses ErrIndexOutOfRange | 02, 08 | COVERED |
| CONTEXT | D-08 | Allocating Minify plus MinifyInto | 04, 05, 07 | COVERED |
| CONTEXT | D-09 | dst==src tested; short destination errors | 04, 07, 10 | COVERED |
| CONTEXT | D-10 | Utility buffers are not Phase 14 borrowed views | 07 | COVERED |
| CONTEXT | D-11 | ValidateUTF8 returns `(bool,error)` | 07 | COVERED |
| CONTEXT | D-12 | No ValidateUTF8 dual method | 07 | COVERED |
| CONTEXT | D-13 | Exact same-start alias validated; partial overlap rejected | 04, 07, 10 | COVERED |
| CONTEXT | D-14 | Durable x86-64 aliasing proof | 10 | COVERED |
| CONTEXT | D-15 | CPU rejection and selection-lock timing | 04, 07, 10 | COVERED |
| RESEARCH | R-01 | Delegate pointer/path grammar to vendored simdjson | 01, 06 | COVERED |
| RESEARCH | R-02 | Wildcard scratch indices -> Rust owned tracked array | 03, 05, 10 | COVERED |
| RESEARCH | R-03 | Array.At O(n); size helpers saturate at 0xFFFFFF | 02, 08 | COVERED |
| RESEARCH | R-04 | Native minify must enforce dst_cap and overlap before write | 04, 07 | COVERED |
| RESEARCH | R-05 | Minify is non-validating except UNCLOSED_STRING | 04, 07, 10 | COVERED |
| RESEARCH | R-06 | Standalone utilities share CPU/selection policy | 04, 07 | COVERED |
| RESEARCH | R-07 | ABI growth is additive and all public symbols are mandatory | 09, 01–05, 10 | COVERED |
| RESEARCH | R-08 | Bracket quoting, leading separator, trailing empty key are documented/tested | 06 | COVERED |
| RESEARCH | R-09 | No new dependency/package install | all | COVERED |
| RESEARCH | R-10 | D-14 proof must be durable and host-count-independent | 10 | COVERED |

Deferred ideas: none. Explicitly excluded encoder, reflection Unmarshal, full RFC 9535 engine,
mutable DOM, file wrapper, and Phase 14 borrowed views do not appear in any plan.

## Validation Sign-Off

- [x] Every task has a focused automated command.
- [x] Compiler commands are not hidden behind non-pipefail pipelines.
- [x] Full contract/release/race suites are wave or phase gates, not repeated task feedback.
- [x] Actual waves and dependencies match all ten plan frontmatters.
- [x] D-14 uses `scripts/ci/verify_minify_buffer_safety.sh`; no execution gate depends on
  `.planning/spikes`.
- [x] Phase 12 branch automatically schedules the five-platform Go workflow.
- [x] Source audit covers GOAL, all six requirements, D-01..D-15, and required research constraints.
- [x] `nyquist_compliant: true`.

**Approval:** revised and re-approved 2026-07-31 after checker iteration 1.
