---
phase: 11
slug: upstream-simdjson-refresh-bigint-and-diagnostics
status: verified
threats_total: 21
threats_closed: 21
threats_open: 0
asvs_level: 1
block_on: high
created: 2026-07-30
---

# Phase 11 — Security

> Threat-model verification against implemented code, executable tests, repository provenance, and immutable hosted-release records.

## Audit Scope and Method

- The register is the union of every `<threat_model>` row in Plans 11-01 through 11-18. Repeated IDs are one threat with cumulative mitigation requirements.
- All 21 unique threats have disposition `mitigate`; there are no accepted or transferred risks.
- Each mitigation started as open. A threat closed only after the declared control was found at the relevant entry points and its executable gate passed.
- Implementation and test files were read-only during this audit. Only this security report was created.
- Plan 11-14's hosted evidence applies to the immutable `v0.1.7` publication. Later Phase 11 source-only gap closures are not attributed to that tag, and this audit does not claim that current HEAD was republished.

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| Caller → Go API | Untrusted JSON, capacity/depth options, and kernel names enter the public API. | Caller-controlled bytes and configuration |
| Go → Rust ABI | Handles, copied input, status codes, diagnostics, and owned byte buffers cross purego bindings. | Pointers, lengths, handles, status/offset fields |
| Rust → C++/simdjson | The Rust registry calls a `noexcept` bridge over an audited, patched simdjson source. | Padded input, native parser/doc pointers, borrowed spans |
| Source → generated ABI | Rust declarations produce the public C header and mandatory Go binding surface. | ABI version, layouts, symbols, enum values |
| Tag → hosted artifact | A main-anchored tag drives CI builds, smoke gates, checksums, signatures, and publication. | Five platform libraries and release metadata |

## Threat Register

| Threat ID | Category | Component | Plan variants | Disposition | Verification evidence | Status |
|-----------|----------|-----------|---------------|-------------|-----------------------|--------|
| T-11-01 | Denial of Service | Parser capacity/depth options | 04, 11, 13 | mitigate | Rust resets diagnostics and rejects over-capacity input before `mem::take`, padding, resize, or copy (`src/runtime/registry.rs:488-507`); Go and native validate options before library/native allocation (`parser_options.go:51-103`, `src/native/simdjson_bridge.cpp:716-751`); arena and N/N+1 tests are at `src/runtime/registry.rs:1366-1382` and `tests/rust_shim_limits.rs:102-165`; packaged smoke constructs bounded options at `tests/smoke/go_bootstrap_smoke.go:18-32`. | closed |
| T-11-02 | Denial of Service | Native/Go implementation selection | 06, 12, 13 | mitigate | Native lookup requires an exact compiled name and runtime CPU support before assignment (`src/native/simdjson_bridge.cpp:1017-1054`); Go maps invalid and unsupported results without changing cached state (`kernel.go:39-68`); isolated invalid/unsupported and fallback tests are in `kernel_test.go:50-116`; packaged smoke reads the real implementation at `tests/smoke/go_bootstrap_smoke.go:26-27`. | closed |
| T-11-03 | Tampering | ABI, tag, loader, and artifact identity | 01, 07, 08, 09, 13, 14 | mitigate | The explicit version-decision commit `b59ac666da279dae62a0fd8fe2228f3bbd19cce7` and ABI-policy commit `7113c8f280e0377828ea53c506143ca5629fa45b` precede the first annotated ABI 1.2 tag object in Git history. ABI 1.2 is pinned in Rust/Go and by a bidirectional compile canary (`src/lib.rs:17`, `internal/ffi/types.go:7`, `internal/bootstrap/abi_assertion.go:5-11`); the source-state checker enforces exact version/ABI policy in both directions (`scripts/release/check_bootstrap_abi_state.py:73-119`); ABI is probed before mandatory bind and cache install (`internal/ffi/bindings.go:60-146`, `library_loading.go:91-126`) with fail-closed fixtures (`library_loading_test.go:350-461`); deterministic header and layout gates are wired at `Makefile:6-18`; release and public-validation controls are at `.github/workflows/release.yml:3-64,579-634,678-683,796-810` and `.github/workflows/public-bootstrap-validation.yml:53-185`. The remote annotated `v0.1.7` ref and peeled target match the local immutable objects, the target is on `origin/main`, release run `30030435051` succeeded from that target, and public-validation run `30448288140` passed all five R2 plus three fallback jobs. | closed |
| T-11-04 | Information Disclosure / Tampering | BigInt ownership and lifetime | 02, 03, 08, 09, 10, 13 | mitigate | Native frame/view spans remain doc-owned (`src/native/simdjson_bridge.cpp:998-1008,1388-1408,1543-1577`), with exact root/nested and replacement-lifetime tests at `tests/rust_shim_fast_materializer.rs:211-268`; Rust copies immediately, records the exact allocation, and rejects mismatched/double free (`src/runtime/registry.rs:837-926`); the public export returns only that owned copy (`src/lib.rs:789-833`); Go copies then frees with `KeepAlive` (`internal/ffi/bindings.go:381-408`) and copies materializer spans while the doc is live (`materializer_fastpath.go:29-36,165-170,210-217`). Lifetime, parity, and adversarial-span tests are at `bigint_test.go:140-166` and `materializer_fastpath_test.go:143-246`; header ownership is enforced at `tests/abi/check_header.py:217-256`; C and packaged smokes copy exact text at `tests/smoke/ffi_export_surface.c:468-481` and `tests/smoke/go_bootstrap_smoke.go:40-50`. | closed |
| T-11-05 | Repudiation | Diagnostic replay and explicit offsets | 04, 05, 09, 12, 13, 15 | mitigate | Diagnostic state clears before the Rust capacity gate (`src/runtime/registry.rs:488-495`) and native reset clears message, offset, and known bit (`src/native/simdjson_bridge.cpp:226-235`); pointer subtraction occurs only after overflow and in-range proof (`src/native/simdjson_bridge.cpp:352-382`); replay is limited to ordinary syntax errors, uses the exact configured capacity/depth in both fresh parsers, and performs at most two passes (`src/native/simdjson_bridge.cpp:385-438,539-710`); Go transports and formats the explicit known bit, including known zero (`errors.go:69-130,150-219`), while the contract forbids numeric inference (`docs/ffi-contract.md:220-227`). Boundary, resource-stop, stale-state, deep-input, known-zero, and unknown tests are at `tests/rust_shim_diagnostics.rs:180-234,315-523`, `tests/rust_shim_limits.rs:199-214`, `errors_test.go:11-63`, and `tests/smoke/ffi_export_surface.c:515-543`. | closed |
| T-11-06 | Tampering / Denial of Service | Process-global kernel lifecycle and pools | 06, 11, 12, 13 | mitigate | Native setter, constructor, and explicit lock share one mutex and lock permanently at first construction (`src/native/simdjson_bridge.cpp:110-118,716-758,1017-1063`); Go setter/parser/pool transitions share `kernelMu` (`kernel.go:10-74`, `parser.go:40-61`, `pool.go:18-37`); pool configuration is comparable and mismatched `Put` is rejected while holding the parser lock (`pool.go:40-62`); status 10 maps centrally (`errors.go:225-239`). Real native post-lock, setter/parser, setter/pool, heterogeneous pool, and concurrent reuse tests are at `kernel_test.go:160-344` and `pool_test.go:93-130,256-285`; the full Go suite passed under `-race`. | closed |
| T-11-07 | Denial of Service | Recursive replay/materialization depth | 15 | mitigate | Go rejects depth above 1024 (`parser_options.go:5-9,76-94`); native construction and replay use the same ceiling (`src/native/simdjson_bridge.cpp:93-95,447-515,546-658,716-751`), and materialization repeats the ceiling defensively (`src/native/simdjson_bridge.cpp:870-880`). Native rejection and N-1/N tests are at `tests/rust_shim_limits.rs:102-165`; the largest accepted malformed case runs in a child and requires invalid JSON plus a proven in-range offset or exact unknown at `tests/rust_shim_diagnostics.rs:179-234`. | closed |
| T-11-16-01 | Tampering | BigInt token boundary | 16 | mitigate | Every patched BigInt early return first checks `is_not_structural_or_whitespace(*p)` (`patches/simdjson-v4.6.4-positive-bigint.patch:21-229`); real-ABI tests cover both signs, root/array/object contexts, `x`, `_`, `+`, `/`, and NUL suffixes while preserving document output sentinels (`tests/rust_shim_bigint.rs:114-154`) and exact valid controls (`tests/rust_shim_bigint.rs:156-180`). | closed |
| T-11-16-02 | Tampering | Multi-architecture patch parity | 16 | mitigate | Build-time constants define the three guarded and three forbidden branch forms (`build.rs:10-34`); each guarded form must occur exactly nine times and each legacy form zero times after the one output-copy patch (`build.rs:43-53,113-132`). Fresh debug and release builds passed these fail-closed assertions. | closed |
| T-11-16-03 | Repudiation | JSONTestSuite oracle | 16 | mitigate | Manifest-complete positive/negative/array/object reject fixtures are recorded at `testdata/jsontestsuite/expectations.tsv:39,97-98,122`; valid oversized controls remain accepted at lines 9-11; `benchmark_oracle_test.go:13-75` enforces bidirectional manifest/file completeness and the full Go gate passed. | closed |
| T-11-17-01 | Tampering | C++ exception/status translation | 17 | mitigate | `std::bad_alloc`, `std::exception`, and unknown catches all route through the common status-97 mapper (`src/native/simdjson_bridge.cpp:249-309`); bridge functions remain `noexcept`; fixed-selector tests assert exact 97/97/invalid-argument at `tests/rust_shim_minimal.rs:349-371`. | closed |
| T-11-17-02 | Repudiation | Exception versus returned-engine classification | 17 | mitigate | Thrown C++ exceptions map to 97 (`src/native/simdjson_bridge.cpp:253-271`), while returned `simdjson::MEMALLOC` remains internal status 127 (`src/native/simdjson_bridge.cpp:170-197`); public enum/header/document values remain 97 and 127 (`src/lib.rs:52-53`, `include/pure_simdjson.h:55-56`, `docs/ffi-contract.md:52-53,281-294`). Generated contract and Rust assertion suites passed. | closed |
| T-11-17-03 | Elevation of Privilege | Deterministic exception seam | 17 | mitigate | The seam accepts only fixed selectors and rejects the default (`src/native/simdjson_bridge.cpp:1784-1805`); it is declared only in the private bridge header/runtime (`src/native/simdjson_bridge.h:240`, `src/runtime/mod.rs:208,774-775`) and explicitly excluded by cbindgen (`cbindgen.toml:55-64`). Searches found no seam symbol in the public header or Go bindings. | closed |
| T-11-18-01 | Denial of Service | Parser-aware C++ exception boundary | 18 | mitigate | Private selector-3 helper throws through the production parser-aware catch macro (`src/native/simdjson_bridge.cpp:297-323,1784-1805`), the same macro used by `psimdjson_parser_parse` (`src/native/simdjson_bridge.cpp:1150-1187`); the exact child requires status 97 (`tests/rust_shim_minimal.rs:373-411`). | closed |
| T-11-18-02 | Denial of Service | `std::bad_alloc` diagnostic capture | 18 | mitigate | The bad-allocation overload passes the fixed literal directly as a non-allocating `string_view` (`src/native/simdjson_bridge.cpp:274-275`); all diagnostic-buffer assignment occurs inside `try_set_last_error_message`'s catch-all guard (`src/native/simdjson_bridge.cpp:232-246`). | closed |
| T-11-18-03 | Denial of Service | `noexcept` capture helper | 18 | mitigate | Both best-effort capture and the parser catch are `noexcept`/catch-all (`src/native/simdjson_bridge.cpp:242-246,274-309`); the isolated child must exit normally and pass exactly one test (`tests/rust_shim_minimal.rs:373-411`). A signal, abort, or escaping second exception fails the parent. | closed |
| T-11-18-04 | Tampering | Exception-seam output sentinel | 18 | mitigate | Selector 3 owns sentinel `0xA5A5A5A5A5A5A5A5`, returns 127 if it changes, and otherwise propagates the helper status (`src/native/simdjson_bridge.cpp:1793-1800`); the child accepts only 97 before printing its success marker (`tests/rust_shim_minimal.rs:374-410`). | closed |
| T-11-18-05 | Elevation of Privilege | Hidden fault selector | 18 | mitigate | Selector 3 reuses the existing hidden symbol and private helper (`src/native/simdjson_bridge.cpp:311-323,1784-1805`); Rust wrappers and private declaration remain unchanged (`src/runtime/mod.rs:208,774-775`, `src/lib.rs:267-271`, `src/native/simdjson_bridge.h:240`). The Plan 17→18 implementation diff changes only the private C++ source and its Rust test, not public header, bindings, wrappers, cbindgen, dependencies, or production controls. | closed |
| T-11-18-06 | Repudiation | Process-survival evidence | 18 | mitigate | The parent launches the current test executable with one exact filter, captures stdout/stderr, requires successful termination, exactly one passing test, and a marker emitted only after the status assertion (`tests/rust_shim_minimal.rs:373-411`). The fresh audit run passed. | closed |
| T-11-18-07 | Tampering | Returned engine errors and ABI statuses | 18 | mitigate | Returned `simdjson::MEMALLOC` remains status 127 (`src/native/simdjson_bridge.cpp:170-197`), while thrown exceptions remain 97; Rust, generated header, and normative docs retain exact values (`src/lib.rs:52-53`, `include/pure_simdjson.h:55-56`, `docs/ffi-contract.md:52-53`). Full contract and diagnostic regressions passed. | closed |
| T-11-SC | Tampering | Vendored source, dependencies, generated ABI, and release supply chain | 01–18 | mitigate | `third_party/simdjson` is clean at exact official `v4.6.4` commit `1bcf71bd85059ab6574ea1159de9298dcc1212c5`; `build.rs:7-10,56-134` verifies the base, clean tree, one patch, hunk applicability, and 9/9/9 guarded shapes; only `patches/simdjson-v4.6.4-positive-bigint.patch` exists; Cargo/Go manifests and locks have no Phase 11 diff; no second vendored/package tree exists. The header is generated only from the Rust crate and diffed (`Makefile:3-18`). Workflow search finds the publishing script invoked only from tag-triggered `.github/workflows/release.yml`, which gates checksums, packaged smoke, keyless signing/verification, immutable upload, and release creation (`.github/workflows/release.yml:3-64,561-634,678-683,796-810`). Immutable recovery used new annotated tags rather than moving existing refs; the remote `v0.1.7` object/target and hosted run/asset records match. | closed |

## Executed Verification

| Check | Result |
|-------|--------|
| `cargo test --locked --test rust_shim_bigint --test rust_shim_minimal --test rust_shim_limits --test rust_shim_diagnostics --test rust_shim_kernel -- --test-threads=1` | PASS — 48/48 tests, including both subprocess survival checks |
| `make verify-contract && make verify-docs` | PASS — full Rust suite, deterministic cbindgen diff, 25 header audits, C layout compile, and documentation gate |
| Bootstrap/release/public-validation Python contract suites | PASS — 10 + 12 + 11 tests |
| `cargo build --release --locked` | PASS — exact source, clean-tree, patch, and architecture-parity assertions |
| Explicit release library + `go test ./... -race -count=1 -timeout=180s` | PASS — all four Go packages |
| Submodule/tag/status/manifests provenance checks | PASS — exact clean v4.6.4, one patch, no dependency manifest/lock drift |
| Hosted release run `30030435051` | PASS — tag target `ab86c2e1e666c6c313d1dd951c37a8c43538c407`; source verification, five platform builds, Alpine smoke, and publish all succeeded |
| Hosted release `v0.1.7` | PASS — five raw libraries, five `.sig`, five `.pem`, `SHA256SUMS`, and its `.sig`/`.pem` are uploaded with digests |
| Hosted public-validation run `30448288140` | PASS — five R2 platform jobs and Linux/macOS/Windows fallback jobs succeeded |

## Unregistered Flags

None. No Phase 11 `*-SUMMARY.md` contains a `Threat Flags` entry.

## Accepted Risks Log

No accepted risks. Every registered Phase 11 threat has disposition `mitigate`.

## Transfer Log

No transferred risks.

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-07-30 | 21 | 21 | 0 | Codex security auditor |

## Sign-Off

- [x] All threats have a disposition.
- [x] Every repeated threat ID's cumulative mitigation variants were verified.
- [x] Accepted and transferred risk logs were checked.
- [x] Executor threat flags were checked.
- [x] `threats_open: 0` confirmed.
- [x] `status: verified` set in frontmatter.

**Approval:** verified 2026-07-30
