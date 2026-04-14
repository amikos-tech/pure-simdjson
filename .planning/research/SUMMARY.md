# Project Research Summary

**Project:** pure-simdjson
**Domain:** Go library wrapping C++ simdjson via a Rust FFI shim + purego, distributed as prebuilt shared libraries over CloudFlare R2
**Researched:** 2026-04-14
**Confidence:** HIGH on the pattern and FFI invariants; MEDIUM on simdjson-specific API style, musl/Alpine decision, and `linux/arm` viability.

---

## Decisions — User Resolved 2026-04-14

These findings from the four research tracks contradict or materially extend the original PROJECT.md brief. All resolved:

### 1. `linux/arm` (32-bit) vs the no-cgo promise — **DROPPED from v0.1**

purego v0.10.0's README explicitly requires `CGO_ENABLED=1` for `linux/arm`. Keeping it silently breaks the `pure-*` family's defining promise. Matrix reduced from 6 targets to 5 (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64). Revisit if purego upstream gains native support.

### 2. DOM for v0.1, On-Demand for v0.2 — **CONFIRMED**

DOM is re-readable with simpler lifetime semantics; maps cleanly onto the handle/Element pattern. On-Demand's single-shot semantics + lazy UTF-8 validation need consumption-tracking plumbing that is better designed after the DOM surface stabilizes.

### 3. ParserPool promoted from v0.2 → **v0.1**

simdjson's per-parser single-doc invariant means naive shared-Parser use either corrupts state or silently serializes. `ParserPool` is ~100 lines on `sync.Pool` and is the canonical Go-idiomatic shape of simdjson's "reuse parser" advice. Must ship with v0.1.

### 4. Input-buffer ownership — **Rust copies into padded Rust-owned buffer on every Parse**

purego does not apply cgo's pointer-pinning rules; Go GC is free to move/free a `[]byte` the Rust shim holds. Copying into Rust-owned padded memory solves ownership + SIMDJSON_PADDING in one mechanism. v0.1 has no zero-copy-in path; `ParsePinned` using `runtime.Pinner` becomes a v0.2 opt-in.

### 5. musl/Alpine — **ADDED to v0.1 scope**

Alpine is dominant in containerized Go deployments (likely ami-gin target). Implementation approach deferred to Phase 6 research (static-link-into-glibc-so vs ship .a with documented relink, vs smoke-test-only with PURE_SIMDJSON_LIB_PATH escape hatch).

### 6. macOS codesigning — **Ad-hoc sign for v0.1** (no $99 Apple Dev ID requirement)

Matches pure-tokenizers today. Users may need `xattr -d com.apple.quarantine` once after download. Documented workaround; proper notarization is a v0.2+ polish task if user friction warrants.

### 7. Iteration API — **Cursor/pull, not visitor/callback**

purego.NewCallback has a ~2000-callback lifetime leak (allocated memory never freed), stack-pinning issues, and panic-across-FFI UB risk. Per-element callbacks cost ~1μs each (~5ms on twitter.json) — defeats simdjson's perf story. Cursor-pull (Go drives iteration, calls native for each step) avoids both. The no-tree-materialization property is preserved.

### 8. Anti-feature added — **In-place mutation / re-serialization**

Added to Out-of-Scope. `minio/simdjson-go` ships it via its tape format; upstream C++ simdjson does not; our design (especially On-Demand in v0.2) makes it nonsensical. Pre-empts a common simdjson-go migration ask.

---

## Executive Summary

pure-simdjson is the fourth entry in the amikos `pure-*` family — a Go library delivering multi-GB/s SIMD JSON parsing by wrapping upstream simdjson (C++) in a thin Rust FFI shim and loading it from Go via purego, with prebuilt shared libraries distributed through CloudFlare R2. The pattern is well-validated across pure-tokenizers (Rust cdylib + cbindgen + purego + R2), pure-onnx (bootstrap/download + finalizer safety-net), and fast-distance (cbindgen + runtime-pinner for slice pointers). Three new wrinkles specific to this project: compiling C++ inside a Rust `build.rs` (solved via `cc` crate over simdjson's single-file amalgamation), the per-parser single-doc invariant (API surfaces `ErrParserBusy` and guides users to `ParserPool`), and opaque tape-index "Element" views (deliberate deviation from pure-tokenizers' copy-out model, justified by the "parse terabytes with zero new allocation" goal).

The recommended approach: v0.1 as **DOM-based** (re-readable, lifetime-simple, maps to familiar handle patterns), with **Rust copying every input buffer into a padded Rust-owned arena** (resolves purego's Go-GC ownership ambiguity + SIMDJSON_PADDING in one move), **cursor/pull iteration driven from Go** (sidesteps purego.NewCallback's hard limits), **distinct `GetInt64`/`GetUint64`/`GetFloat64` accessors with overflow errors** (the core stdlib-gap differentiator), **ParserPool in v0.1**, and **generation-stamped opaque handles** (double-free and use-after-close become clean `ErrInvalidHandle`, not segfaults). v0.2 layers On-Demand + `at_pointer` selective extraction + zero-copy string views + NDJSON parallel streaming on top.

Key risks all sit in the FFI contract and the 5-target (+ musl) build matrix: macOS codesigning, musl/Alpine coverage strategy, and the dozen P0 FFI pitfalls (parser-reuse invalidation, Rust panic containment, C++ exception containment, struct-by-value ABI differences across Windows vs Linux vs Darwin, generation-stamped handles for safe double-close). Mitigation is mechanical once decisions are locked: an `ffi_fn!` wrapper macro enforcing `catch_unwind` + error-code return on every exported function, a grep-based CI rule blocking `operator[]` without `.get(err)` on simdjson calls, a per-platform FFI smoke test, and a SHA-256-table-in-Go-source integrity check on binary downloads.

## Key Findings

### Recommended Stack

See `STACK.md`. Near-verbatim lift of pure-tokenizers with three deltas: purego upgraded to **v0.10.0** (struct passing on Linux — pure-tokenizers still at v0.8.4), simdjson **v4.6.1** vendored at `third_party/simdjson` as a pinned submodule and compiled via the `cc` crate over the single-file amalgamation (not cmake), and `cxx`/`autocxx`/`bindgen` explicitly **not** used (hand-written ~15-function C ABI is simpler).

**Core:** Go 1.24 + purego v0.10.0; Rust stable (1.85+) with `crate-type=["cdylib","staticlib"]`; simdjson v4.6.1 + `cc` crate + C++17 with `-static-libstdc++ -static-libgcc`; cbindgen 0.29.2; cosign keyless OIDC + CloudFlare R2 at `releases.amikos.tech/pure-simdjson/<version>/`.

**Rejected:** `cxx`/`autocxx`/`bindgen` (wrong tool for narrow hand-written C ABI), `-march=native` (breaks runtime kernel dispatch), musl cdylib (Rust musl targets only emit staticlib — drives the Alpine strategy question).

### Expected Features

See `FEATURES.md`. Strategic position: **SIMD upstream + On-Demand + ARM64 + no-cgo** — no existing Go JSON library occupies all four quadrants. `simdjson-go` is x86-only; `sonic` has no On-Demand; `goccy/go-json` has no SIMD.

**v0.1 must-haves:**
- `Parse([]byte) → Doc` on all 5 platforms (+ musl smoke)
- Distinct `GetInt64` / `GetUint64` / `GetFloat64` — the stdlib-gap differentiator
- `GetString` / `GetBool` / `IsNull` + cursor-pull iteration over arrays and objects
- Explicit `Close()` on `Parser` and `Doc`; finalizer as leak-warning safety net only
- Loud failure on unsupported CPU (`fallback` kernel → `ErrCPUUnsupported`) unless user opts in
- Parser/Doc reuse API + `ParserPool`
- Three-tier benchmark suite vs `encoding/json`, `simdjson-go`, `sonic`

**v0.2:**
- On-Demand API with `at_pointer()` / `at_path()` (selective extraction — ami-gin pilot)
- Zero-copy string views tied to `Doc` lifetime
- NDJSON parallel streaming on `[]byte` (`iterate_many` at ~3.5 GB/s)
- `ParsePinned` zero-copy-in opt-in via `runtime.Pinner`

**Out of Scope (validated + extended):**
- JSON encoding, struct-reflection Unmarshal, full JSONPath, JSON Schema, cgo fallback, silent SIMD fallback, **in-place document mutation / re-serialization**.

### Architecture Approach

See `ARCHITECTURE.md`. Three stacked layers with a strict C ABI at the Go↔Rust seam and opaque-pointer handles at every lifetime boundary. Element is deliberately not a heap handle — `{doc, tape_node_index}`, copyable, register-sized, lifetime-tied to `Doc`. Every FFI op re-locks parent's RWMutex and checks `closed`; use-after-close produces clean error, not UB.

**Components:**
1. **Go public API** (`package purejson`) — `Parser`, `Doc`, `Element`, `Array`, `Object`, `ParserPool`; thin purego bindings; `sync.RWMutex`-guarded lifecycles; cursor-pull iteration
2. **Rust cdylib shim** — `#[no_mangle] extern "C"` with mandatory `ffi_fn!(catch_unwind, error_code_return)` wrapper; generation-stamped handle registry; Rust-owned padded input arena
3. **C++ simdjson (vendored)** — compiled by `cc` crate from `build.rs`; runtime kernel dispatch (icelake/haswell/westmere/arm64/fallback); statically linked libstdc++/libgcc
4. **Bootstrap/download** — R2 primary + GitHub Releases fallback; SHA-256 table in Go source; OS-user-cache-dir storage; `PURE_SIMDJSON_LIB_PATH` override
5. **ABI-version handshake** — compile-time constraint (`^0.1.x`); runtime `get_abi_version()` check; mismatch → `ErrABIVersionMismatch`

### Critical Pitfalls

See `PITFALLS.md`. 12 P0, 14 P1, 6 P2. Top 5 to design against before a single function is written:

1. **Parser-reuse invalidates prior Doc views** — Fix: generation-stamped handles; stale ops return `ERR_DOC_INVALIDATED`
2. **Input `[]byte` freed/moved while Doc alive** — Fix: Rust copies into padded Rust-owned buffer
3. **Rust panic or C++ exception crossing FFI** — Fix: mandatory `ffi_fn!` macro with `catch_unwind`; `.get(err)` form only; grep CI check; `panic = "abort"`
4. **Handle double-free across goroutines** — Fix: generation-stamped `{slot, gen}`; atomic CAS on Go handle; idempotent `Close()`
5. **Struct-by-value + mixed float/int args break on Windows** — Fix: every `extern "C"` returns an error-code integer; multi-value via out-pointers; no float/int mixing; per-platform FFI smoke test

Plus P1 distribution gotchas: macOS codesigning (ad-hoc for v0.1), Windows DLL hijacking (full-path `LoadLibrary`), binary integrity (SHA-256 table).

## Implications for Roadmap

Research converges on **7 phases** for v0.1:

### Phase 1: FFI Contract Design (design-only)
Locks error-code space, handle format (generation-stamped), ownership rules (Rust-owned input arena), no-struct-returns rule, no-float/int-mixing rule, cursor-pull iteration signatures, UTF-8 validation contract, number-accessor split. Gates every subsequent phase. Addresses 7/12 P0 pitfalls at the contract level.

### Phase 2: Rust Shim Skeleton + Minimal Parse Path
`Cargo.toml` cdylib+staticlib; `build.rs` driving `cc` crate over simdjson amalgamation; `ffi_fn!` macro; `parser_new`/`parser_free`/`parser_parse`/`doc_free`/`doc_root`/`element_type`/`element_get_int64`; cbindgen generating `pure_simdjson.h`; ABI version exports.

### Phase 3: Go Public API + purego Binding Happy Path
`Parser` + `Doc` + single `GetInt64` smoke test; platform-specific `library_*.go`; env → cache → fail path; `ParserPool`; typed Go errors; finalizer warning in test builds.

### Phase 4: Full Typed Accessor Surface
`GetUint64`, `GetFloat64`, `GetString`, `GetBool`, `IsNull`; `Array` + cursor iteration; `Object` + `object_iter_next` + `GetField`; `GetNumberType`; `ErrNumberOutOfRange` / `ErrPrecisionLoss`.

### Phase 5: Bootstrap + Distribution (parallel with Phase 4)
R2 primary + GH fallback + retry + context timeout; SHA-256 table; OS-user-cache storage; `PURE_SIMDJSON_LIB_PATH` + `PURE_SIMDJSON_BINARY_MIRROR`; `BootstrapSync()`; `cmd/pure-simdjson-bootstrap`.

### Phase 6: CI Release Matrix
`rust-release.yml` + composite actions (from pure-tokenizers); per-target builds; cosign signing; ad-hoc codesign on macOS; manylinux2014-baseline glibc; Alpine smoke-test job; artifact naming includes toolchain.

### Phase 7: Benchmark Harness + v0.1 Release
Three-tier suite on twitter.json, canada.json, citm_catalog.json, mesh.json, numbers.json vs `encoding/json`, `simdjson-go`, `sonic`, `go-json`; benchstat reports; native vs Go alloc stats; cold-start bench; CPU floor docs; `Kernel()`/`ImplementationName()` diagnostic API.

### Ordering rationale
- Phase 1 alone first — contract is expensive to walk back after code
- Phase 2 before Phase 3 — Go can't call nonexistent symbols
- Phase 3 before Phase 4 — one-FFI-call debug loop before surface-area volume
- Phase 4 ∥ Phase 5 — independent
- Phase 6 after 2–5 — needs a complete codebase
- Phase 7 last — credible benchmarks need the final `.so` on the final channel

### Research flags

Phases likely needing `/gsd-research-phase` during planning:
- **Phase 1** — bespoke FFI contract; no prior `pure-*` lib compiled C++ inside Rust build
- **Phase 6** — musl strategy + manylinux vs zig-cc + final target matrix

Phases with standard patterns (skip research-phase): 2 (pure-tokenizers lift), 3 (pure-tokenizers `tokenizers.go`), 4 (mechanical expansion), 5 (pure-onnx bootstrap.go lift), 7 (pure-tokenizers benchmark.yml lift).

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | 3 reference repos validate every choice; MEDIUM only on `cc`-crate compile path |
| Features | HIGH | Every matrix row sourced from library docs or upstream simdjson docs |
| Architecture | HIGH | Layering/FFI verbatim from pure-tokenizers; Element-as-view maps to simdjson's borrowed-view model |
| Pitfalls | HIGH | Every P0 sourced to simdjson upstream + purego docs; MEDIUM only on distribution-layer items |

Overall: **HIGH**, all 8 scope decisions resolved. Requirements can proceed.

## Sources

### Primary (HIGH)
- `github.com/amikos-tech/pure-tokenizers`, `pure-onnx`, `fast-distance`
- `github.com/simdjson/simdjson` docs (basics, performance, ondemand_design, iterate_many, dom, padding, UTF-8, runtime dispatch); issues #1246, #938, #906, Discussion #2195
- `github.com/ebitengine/purego` v0.10.0 README + `RegisterFunc`/`NewCallback` docs
- Rust Nomicon (FFI unwinding, `catch_unwind`, C-unwind ABI)
- Microsoft Learn (DLL search order, `LoadLibraryEx` safe flags)

### Secondary (MEDIUM)
- `bytedance/sonic`, `minio/simdjson-go`, `goccy/go-json`, `sugawarayuuta/sonnet` — feature comparison
- cxx.rs (C++ exception bridging — rejected choice)
- Keiser & Lemire 2024, On-Demand JSON paper

---
*Research completed: 2026-04-14*
