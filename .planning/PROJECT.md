# pure-simdjson

## What This Is

A Go library providing SIMD-accelerated JSON parsing at multi-GB/s throughput — implemented as a thin Rust shim over the C++ [simdjson](https://github.com/simdjson/simdjson) library, consumed from Go via [purego](https://github.com/ebitengine/purego) so downstream Go projects build without cgo. Pre-built shared libraries are distributed per platform through a CloudFlare R2 + Workers pipeline. Fourth entry in the amikos `pure-*` family, following the established handle-based FFI lifecycle pattern.

## Core Value

Replace `encoding/json` + `any` in parse-heavy Go workloads with a ≥3× faster, precision-preserving (distinct int64/uint64/float64) parser that does not force consumers to enable cgo. If every other feature is cut, the `[]byte → Doc → typed accessors` happy path on all five supported platforms must work.

## Requirements

### Validated

<!-- Shipped and confirmed valuable. -->

(None yet — ship to validate)

### Active

<!-- Current scope. Hypotheses until shipped and validated. -->

**v0.1 (MVP) — DOM API:**

- [ ] Parse JSON from `[]byte` into a reusable `Doc` handle (input copied into Rust-owned padded arena)
- [ ] Cursor/pull iteration (Go drives; no Go `map[string]any`/`[]any` materialization)
- [ ] Typed number access: distinct `int64`, `uint64`, `float64` getters with `ErrNumberOutOfRange` / `ErrPrecisionLoss`
- [ ] Typed accessors for string, bool, null, array, object
- [ ] Parser/Document reuse API (allocate once, reuse across parses) + `ParserPool` helper
- [ ] Explicit handle lifecycle: `Close()`/`Free()`, generation-stamped handles; finalizer as leak-warning only
- [ ] Five-platform support: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64
- [ ] musl/Alpine smoke-test job + `PURE_SIMDJSON_LIB_PATH` escape hatch (implementation strategy TBD in Phase 6)
- [ ] Rust FFI shim with C ABI, `ffi_fn!`-macro-enforced `catch_unwind` + error-code convention (mirrors pure-tokenizers)
- [ ] Binary bootstrap: download pre-built shared library from CloudFlare R2 (primary) + GitHub Releases (fallback); SHA-256 table in Go source
- [ ] CI release matrix with cosign keyless signing + ad-hoc macOS codesign
- [ ] Benchmarks vs `encoding/json`, `minio/simdjson-go`, `bytedance/sonic` on twitter.json, canada.json, citm_catalog.json

**v0.2 (next) — On-Demand:**

- [ ] On-Demand API: parse with pre-declared path set (skips unused keys) via `at_pointer()` / `at_path()`
- [ ] NDJSON streaming parse (parallel `iterate_many`, targeting ~3 GB/s)
- [ ] Zero-copy string views tied to `Doc` lifetime
- [ ] `ParsePinned` zero-copy-in via `runtime.Pinner`

### Out of Scope

<!-- Explicit boundaries with reasoning to prevent re-adding. -->

- **JSON marshalling/encoding** — parse-only library; encoding is a separate problem with separate perf characteristics
- **Struct binding / reflection-based Unmarshal** — reflection defeats simdjson's perf story; consumers can write explicit extractors
- **Full JSONPath query execution** — simple path extraction only; full JSONPath is a separate engine
- **JSON Schema validation** — orthogonal concern; belongs in a separate library
- **cgo build path** — the whole point of `pure-*` is no-cgo consumers; maintaining a cgo fallback doubles the surface area
- **Silent SIMD fallback feature flags** — simdjson has runtime kernel dispatch; if a target CPU genuinely lacks support we fail loudly rather than degrade quietly
- **linux/arm (32-bit)** — purego v0.10.0 requires `CGO_ENABLED=1` on this target; shipping it would break the no-cgo promise for every downstream consumer. Revisit if upstream purego adds support.
- **In-place document mutation / re-serialization** — `simdjson-go` offers this via its tape format; upstream C++ simdjson does not; our design (especially On-Demand in v0.2) makes it nonsensical. Pre-empts common simdjson-go migration ask.
- **Visitor/callback iteration from native into Go** — `purego.NewCallback` leaks callback trampolines (~2000 limit) and pays ~1μs per node across FFI; defeats the perf story. Cursor/pull iteration replaces it.

## Context

**Immediate consumer:** [`github.com/amikos-tech/ami-gin`](https://github.com/amikos-tech/ami-gin) Phase 07 build-time ingest. Its GIN index build currently bottlenecks on `json.Unmarshal(..., &any)` — needs selective-path extraction and faster parse.

**Other intended consumers:** log/event processors, schema validators, JSONL tooling, any Go project where JSON parse cost dominates.

**Reference projects (study before deviating — fourth `pure-*` library, pattern is established):**

- [`pure-tokenizers`](https://github.com/amikos-tech/pure-tokenizers) — canonical handle-based FFI lifecycle (New/Free), error-code convention, purego symbol binding
- [`pure-onnx`](https://github.com/amikos-tech/pure-onnx) — cross-platform build matrix, GitHub Actions release automation, R2 artifact layout
- [`fast-distance`](https://github.com/amikos-tech/fast-distance) — runtime CPU feature detection, NEON/AVX dispatch

**Why this exists:** Go's `encoding/json` into `any` is slow (byte-by-byte state machine, allocates boxed tree) and lossy (all numbers collapse to `float64`, losing precision above 2^53). simdjson parses at multi-GB/s, distinguishes int64/uint64/float64 natively, and supports lazy selective-path extraction.

**Research summary:** See `.planning/research/SUMMARY.md` for the consolidated findings from STACK, FEATURES, ARCHITECTURE, and PITFALLS research tracks, including the 8 scope decisions resolved 2026-04-14.

## Constraints

- **License**: MIT — consistent with the `pure-*` family and compatible with simdjson's Apache-2.0 upstream
- **Module path**: `github.com/amikos-tech/pure-simdjson` — fixed
- **Tech stack**: Go 1.24 (public API, purego consumer), Rust stable 1.85+ (FFI shim), C++17 (simdjson upstream); no other languages
- **Build**: no cgo at Go-consumer build time — purego runtime loading only
- **Platforms**: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64 — five first-class targets. musl/Alpine smoke-tested with documented escape hatch.
- **Distribution**: CloudFlare R2 + Workers (primary) + GitHub Releases (fallback); SHA-256 table embedded in Go source; cosign keyless signing
- **Failure mode**: explicit CPU-feature failures over silent SIMD-disabled fallback
- **FFI safety**: `ffi_fn!` macro enforces `catch_unwind` on every `extern "C"` boundary; error-code return only (no struct-by-value)

## Current State

Phase 1 is complete. The repository now has a generated public ABI header, a normative FFI contract, and static verification gates that lock the handle format, error-code space, parser lifecycle, ownership model, and ABI compatibility rules before shim implementation begins.

Next up: Phase 2 builds the real Rust shim and the first end-to-end parse path against this fixed contract.

## Key Decisions

<!-- Decisions that constrain future work. -->

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Rust shim over direct C++ FFI | Stable C ABI via `extern "C"`, no C++ name mangling/ABI instability; ~15-function hand-written surface | ABI-source crate + generated header established in Phase 1; shim implementation pending in Phase 2 |
| purego over cgo | Consumers build pure-Go; no cgo toolchain requirement; matches pure-* family | — Pending |
| CloudFlare R2 for binary distribution | Already-operational infra (pure-onnx, pure-tokenizers); per-platform artifacts on demand | — Pending |
| Generation-stamped opaque handles | Turns double-free / use-after-close into clean `ErrInvalidHandle` instead of segfaults | Locked in Phase 1 ABI (`pure_simdjson_handle_t` + `pure_simdjson_handle_parts_t`) |
| C-style error codes at FFI + typed Go errors at wrapper | pure-tokenizers convention; clean separation of FFI from idiomatic Go | Numeric FFI error-code space locked in Phase 1; Go mapping still pending |
| Parser/Doc reuse + `ParserPool` baked into v0.1 API | simdjson's primary performance lever; per-parser single-doc invariant requires explicit concurrency primitive | — Pending |
| DOM API for v0.1, On-Demand for v0.2 | DOM is re-readable, lifetime-simple; On-Demand's single-shot semantics + lazy UTF-8 validation need consumption-tracking that benefits from a stable DOM foundation | Locked in Phase 1 scope and contract |
| Cursor/pull iteration (no visitor callbacks) | Avoids `purego.NewCallback` leak (~2000-callback lifetime limit) + stack-switch cost per node | Locked in Phase 1 ABI (`*_iter_new`, `*_iter_next`, `object_get_field`) |
| Rust-owned padded input arena (Rust copies every `Parse` input) | purego has no pointer-pinning; Go GC may move/free the `[]byte` while Rust reads it. Copy-in also satisfies SIMDJSON_PADDING. | Locked in Phase 1 contract and header comments |
| `cc` crate over cmake for simdjson amalgamation | Simpler build.rs; no cmake toolchain dependency; simdjson single-file amalgamation is designed for this | — Pending |
| Raw `extern "C"` over `cxx`/`autocxx`/`bindgen` | ~15-function surface; matches pure-* family convention; avoids extra dep | — Pending |
| Drop linux/arm from v0.1 | purego requires `CGO_ENABLED=1` on arm7; keeping it breaks no-cgo promise | — Pending |
| Add musl/Alpine smoke-test to v0.1 | Dominant container base image; glibc-only `.so` won't load on Alpine | — Pending |
| Ad-hoc macOS codesign for v0.1 | Matches pure-tokenizers; avoids $99/yr Apple Developer ID prerequisite; users may need `xattr -d` once | — Pending |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-04-14 after Phase 1 completion*
