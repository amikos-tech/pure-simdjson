---
phase: 02
slug: rust-shim-minimal-parse-path
status: approved
nyquist_compliant: true
wave_0_complete: true
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
| **Full suite command** | `make verify-contract && cargo build --release && make phase2-smoke-linux` locally, then observe a successful remote `windows-smoke` run from `phase2-rust-shim-smoke.yml` |
| **Estimated runtime** | ~60 seconds local plus GitHub Actions runtime for the observed Windows smoke |

---

## Sampling Rate

- **After every task commit:** Run `cargo test`
- **After every plan wave:** Run `make verify-contract && cargo build --release`
- **Before `/gsd-verify-work`:** Full suite plus the native C smoke harness and an observed GitHub Actions `windows-smoke` success must be green
- **Max feedback latency:** 60 seconds for local loops; Task `02-03-02` is the explicit remote-CI exception because it must observe a GitHub Actions `windows-smoke` result

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 02-01-01 | 01 | 1 | SHIM-01 / SHIM-02 / SHIM-03 / SHIM-04 | T-02-01-01 / T-02-01-03 | `Cargo.toml` keeps `crate-type = ["cdylib", "staticlib"]`, vendored simdjson is pinned, and release builds produce both static and dynamic artifacts without `-march=native` | build | `cargo build --release && rg 'crate-type = \\["cdylib", "staticlib"\\]' Cargo.toml && test -f target/release/libpure_simdjson.a && find target/release -maxdepth 1 \\( -name 'libpure_simdjson.so' -o -name 'libpure_simdjson.dylib' -o -name 'pure_simdjson.dll' \\) | grep -q .` | `Cargo.toml`, `build.rs`, `.gitmodules` | ⬜ pending |
| 02-01-02 | 01 | 1 | SHIM-04 | T-02-01-02 | The bridge is narrow, `noexcept`, catches all C++ exceptions, and exposes the real simdjson padding constant | build/static | `cargo build --release && rg 'noexcept|catch \\(\\.\\.\\.\\)|SIMDJSON_PADDING' src/native/simdjson_bridge.h src/native/simdjson_bridge.cpp` | `src/native/simdjson_bridge.h`, `src/native/simdjson_bridge.cpp` | ⬜ pending |
| 02-02-01 | 02 | 2 | SHIM-06 | T-02-02-01 / T-02-02-02 | Parser/doc lifecycle rejects stale, zero, double-freed, or busy handles cleanly and routes every export through `ffi_wrap` | unit | `cargo test` | `src/lib.rs`, `src/runtime/mod.rs`, `src/runtime/registry.rs` | ⬜ pending |
| 02-02-02 | 02 | 2 | SHIM-05 / SHIM-06 / SHIM-07 | T-02-02-03 / T-02-02-04 | The minimal `42` path, implementation-name helpers, diagnostics spine, padding contract, and fallback CPU gate work without widening Phase 2 scope | unit | `cargo test --test rust_shim_minimal` | `tests/rust_shim_minimal.rs` | ⬜ pending |
| 02-03-01 | 03 | 3 | SHIM-06 | T-02-03-01 | Linux smoke harness compiles against the committed header, runs the full `42` path, and proves `doc_free`/`parser_free` cleanup | integration | `make verify-contract && cargo build --release && make phase2-smoke-linux` | `tests/smoke/minimal_parse.c`, `tests/smoke/README.md`, `Makefile` | ⬜ pending |
| 02-03-02 | 03 | 3 | SHIM-06 | T-02-03-02 / T-02-03-03 | Static lint confirms the workflow shape, and phase completion requires a dispatched run where the observed `windows-smoke` job concludes `success` after the Linux/Windows symbol and smoke checks run | CI/static+observed | `gh auth status && git remote get-url origin >/dev/null && test -f .github/workflows/phase2-rust-shim-smoke.yml && rg 'ilammy/msvc-dev-cmd|cl /nologo|nm -D|dumpbin /EXPORTS|phase2-smoke-linux' .github/workflows/phase2-rust-shim-smoke.yml && git push -u origin HEAD && gh workflow run phase2-rust-shim-smoke.yml --ref "$(git rev-parse --abbrev-ref HEAD)" && sleep 10 && RUN_ID="$(gh run list --workflow phase2-rust-shim-smoke.yml --branch "$(git rev-parse --abbrev-ref HEAD)" --limit 1 --json databaseId --jq '.[0].databaseId')" && gh run watch "$RUN_ID" --exit-status && gh run view "$RUN_ID" --json jobs --jq '.jobs[] | select(.name=="windows-smoke") | .conclusion' | grep -x success` | `.github/workflows/phase2-rust-shim-smoke.yml` | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements.

## Later-Wave Verification Artifacts

- [ ] `build.rs` + `.gitmodules` — vendored simdjson build pipeline created in Plan 01
- [ ] `src/native/simdjson_bridge.h` + `src/native/simdjson_bridge.cpp` — bridge exception containment and padding helper created in Plan 01
- [ ] `src/runtime/mod.rs` + `src/runtime/registry.rs` — lifecycle verification target created in Plan 02
- [ ] `tests/rust_shim_minimal.rs` — minimal-path and lifecycle unit coverage created in Plan 02
- [ ] `tests/smoke/minimal_parse.c` + `tests/smoke/README.md` + `Makefile` targets — Linux smoke proof created in Plan 03 Task 1
- [ ] `.github/workflows/phase2-rust-shim-smoke.yml` — Linux and Windows/MSVC smoke workflow created in Plan 03 Task 2
- [ ] Observed `phase2-rust-shim-smoke.yml` run where `windows-smoke` concludes `success` — remote proof captured during Plan 03 Task 2

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
- [ ] Local feedback latency stays < 60s, with Task `02-03-02` explicitly allowed to wait on remote GitHub Actions completion for the observed Windows proof
- [ ] Task `02-03-02` includes both static workflow lint and an observed green `windows-smoke` run; YAML inspection alone does not close the phase
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-04-15
