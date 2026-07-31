---
phase: 260731-q1j-address-the-release-workflow-contract-sa
plan: quick-full
subsystem: release-ci-and-ffi-contracts
tags: [github-actions, semver, simdjson, ffi, concurrency, go, rust]
requires:
  - phase: 12
    provides: ABI 1.3 navigation, utility, and materialization surfaces
provides:
  - PR-to-main smoke workflow contracts and prerelease-aware release validation
  - Native wildcard syntax classification with safe carrier ownership
  - Reserved utility kernel-state transitions and value/frame ABI guards
affects: [release workflow, bootstrap ABI checks, wildcard navigation, kernel selection, materialization]
tech-stack:
  added: []
  patterns: [constrained workflow mapping parser, native wildcard classification, utility reservation protocol]
key-files:
  modified:
    - .github/workflows/phase2-rust-shim-smoke.yml
    - scripts/release/check_bootstrap_abi_state.py
    - src/native/simdjson_bridge.cpp
    - kernel.go
    - materializer_fastpath.go
key-decisions:
  - "Validate wildcard structure in the C++ bridge before traversal can suppress syntax errors."
  - "Reserve kernel selection while utility library/native work runs without kernelMu."
requirements-completed: [C1, C2, C3, I1, I2, I3, I4, I5, I6, I7, I8, I9, I10]
duration: 32min
completed: 2026-07-31
---

# Quick Task 260731-q1j: Release Workflow and ABI Contract Safety Summary

**PR smoke coverage, semver release floors, wildcard carrier safety, and utility kernel reservations are enforced with behavioral regression tests.**

## Accomplishments

- Added pull-request-to-main workflow coverage without temporary branch allowlists, removed the unmeasured sanitizer flag, and order SemVer prereleases below the corresponding final release.
- Classified wildcard expressions in the C++ bridge before traversal, matching vendored simdjson literal behavior and preserving quoted-bracket key semantics.
- Normalized native carrier/free sentinels, zeroed wildcard error outputs, and added a deterministic SetKernel-versus-utility reservation test.
- Replaced delayed uintptr frame conversions with pointer-typed FFI fields, pinned all value-view field offsets, and tested bounded copied string spans.

## Task Commits

1. **Plan 01: Release workflow contracts** — `e07a9c9`
2. **Plan 02: Wildcard FFI contracts** — `3dd3697`
3. **Plan 03: Utility kernel reservation and ABI guards** — `fb61fb7`

## Verification Evidence

- `python3 -m unittest scripts/release/test_release_workflow_contracts.py scripts/release/test_check_bootstrap_abi_state.py` — 37 tests passed.
- `bash scripts/ci/verify_minify_buffer_safety.sh` — three ASan/UBSan probe runs passed (`kernels=arm64,fallback`, `total=24`).
- `cargo test --test rust_shim_navigation --test rust_shim_accessors -- --test-threads=1` — 29 tests passed.
- `cargo test -- --test-threads=1` and `make verify-contract` — passed, including generated-header and C ABI checks.
- `go test ./internal/ffi ./... -race`, `go test ./... -race`, and `go vet ./...` — passed.
- `cc -Iinclude tests/abi/handle_layout.c -c` — passed.

## Decisions Made

- The workflow checker uses a deliberately narrow, indentation-aware extractor for the asserted event/job subset; unsupported YAML constructs in that subset fail closed.
- A valid wildcard with no matches is the native null/zero sentinel; malformed or literal-star expressions return `ErrInvalidPath` on every receiver type.
- Utility calls reserve selection before unlocked library/native work and publish their final status before `SetKernel` can bind a new implementation.

## Deviations from Plan

None - plans executed as specified. Existing frame tests already covered document-close lifetime; direct bounded-copy coverage was added alongside the pointer representation fix.

## Known Stubs

None.

## Next Readiness

All C1-C3 and I1-I10 contracts covered by this quick task are implemented and verified. No external setup, release action, push, or merge was performed.

## Self-Check: PASSED

- Summary exists at the planned quick-task path.
- Commits `e07a9c9`, `3dd3697`, and `fb61fb7` are present in git history.
