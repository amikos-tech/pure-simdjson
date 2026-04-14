# pure-simdjson

## What This Is

A Go library providing SIMD-accelerated JSON parsing at multi-GB/s throughput — implemented as a thin Rust shim over the C++ [simdjson](https://github.com/simdjson/simdjson) library, consumed from Go via [purego](https://github.com/ebitengine/purego) so downstream Go projects build without cgo. Pre-built shared libraries are distributed per platform through a CloudFlare R2 + Workers pipeline. Fourth entry in the amikos `pure-*` family, following the established handle-based FFI lifecycle pattern.

## Core Value

Replace `encoding/json` + `any` in parse-heavy Go workloads with a ≥3× faster, precision-preserving (distinct int64/uint64/float64) parser that does not force consumers to enable cgo. If every other feature is cut, the `[]byte → Doc → typed accessors` happy path on all six platforms must work.

## Requirements

### Validated

<!-- Shipped and confirmed valuable. -->

(None yet — ship to validate)

### Active

<!-- Current scope. Hypotheses until shipped and validated. -->

**v0.1 (MVP):**

- [ ] Parse JSON from `[]byte` into a reusable `Doc` handle
- [ ] Visitor/callback iteration (no Go `map[string]any`/`[]any` materialization)
- [ ] Typed number access: distinct `int64`, `uint64`, `float64` getters
- [ ] Typed accessors for string, bool, null, array, object
- [ ] Parser/Document reuse API (allocate once, reuse across parses)
- [ ] Explicit handle lifecycle: `Close()`/`Free()`, no finalizer-only reliance
- [ ] Six-platform support: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64, linux/arm
- [ ] Rust FFI shim with C ABI + error-code convention (mirrors pure-tokenizers)
- [ ] Binary bootstrap: download pre-built shared library from CloudFlare R2 on first use
- [ ] CI release matrix covering all six targets
- [ ] Benchmarks vs `encoding/json` and `minio/simdjson-go` on twitter.json, canada.json, citm_catalog.json

**v0.2 (next):**

- [ ] On-Demand API: parse with pre-declared path set (skips unused keys)
- [ ] NDJSON streaming parse (parallel, targeting ~3 GB/s)
- [ ] Zero-copy string views tied to `Doc` lifetime

### Out of Scope

<!-- Explicit boundaries with reasoning to prevent re-adding. -->

- **JSON marshalling/encoding** — parse-only library; encoding is a separate problem with separate perf characteristics
- **Struct binding / reflection-based Unmarshal** — reflection defeats simdjson's perf story; consumers can write explicit extractors
- **Full JSONPath query execution** — simple path extraction only; full JSONPath is a separate engine
- **JSON Schema validation** — orthogonal concern; belongs in a separate library
- **cgo build path** — the whole point of `pure-*` is no-cgo consumers; maintaining a cgo fallback doubles the surface area
- **Silent SIMD fallback feature flags** — simdjson has runtime kernel dispatch; if a target CPU genuinely lacks support we fail loudly rather than degrade quietly

## Context

**Immediate consumer:** [`github.com/amikos-tech/ami-gin`](https://github.com/amikos-tech/ami-gin) Phase 07 build-time ingest. Its GIN index build currently bottlenecks on `json.Unmarshal(..., &any)` — needs selective-path extraction and faster parse.

**Other intended consumers:** log/event processors, schema validators, JSONL tooling, any Go project where JSON parse cost dominates.

**Reference projects (study before deviating — fourth `pure-*` library, pattern is established):**

- [`pure-tokenizers`](https://github.com/amikos-tech/pure-tokenizers) — canonical handle-based FFI lifecycle (New/Free), error-code convention, purego symbol binding
- [`pure-onnx`](https://github.com/amikos-tech/pure-onnx) — cross-platform build matrix, GitHub Actions release automation, R2 artifact layout
- [`fast-distance`](https://github.com/amikos-tech/fast-distance) — runtime CPU feature detection, NEON/AVX dispatch

**Why this exists:** Go's `encoding/json` into `any` is slow (byte-by-byte state machine, allocates boxed tree) and lossy (all numbers collapse to `float64`, losing precision above 2^53). simdjson parses at multi-GB/s, distinguishes int64/uint64/float64 natively, and supports lazy selective-path extraction.

**Key design questions deferred to phase discussions:**
- Concurrency model: one parser per goroutine, or a pool? How do handles cross goroutines safely?
- Number semantics: expose simdjson's raw number type (caller picks getter), or probe-and-return based on representation?
- String ownership: return `string` (copy) or `[]byte` view into native buffer (zero-copy, tied to Doc lifetime)?
- Error model: C-style codes at FFI boundary → typed Go errors at wrapper (default, per pure-tokenizers)

## Constraints

- **License**: MIT — consistent with the `pure-*` family and compatible with simdjson's Apache-2.0 upstream
- **Module path**: `github.com/amikos-tech/pure-simdjson` — fixed
- **Tech stack**: Go (public API, purego consumer), Rust (FFI shim), C++ (simdjson upstream); no other languages
- **Build**: no cgo at Go-consumer build time — purego runtime loading only
- **Platforms**: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64, linux/arm — all first-class, no exceptions
- **Distribution**: CloudFlare R2 + Workers, same URL scheme as pure-onnx
- **Failure mode**: explicit CPU-feature failures over silent SIMD-disabled fallback

## Key Decisions

<!-- Decisions that constrain future work. -->

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Rust shim over direct C++ FFI | Stable C ABI via `extern "C"`, no C++ name mangling/ABI instability; ~100-line thin layer | — Pending |
| purego over cgo | Consumers build pure-Go; no cgo toolchain requirement; matches pure-* family | — Pending |
| CloudFlare R2 for binary distribution | Already-operational infra (pure-onnx, pure-tokenizers); per-platform artifacts on demand | — Pending |
| Handle-based lifecycle (New/Free) | Mirrors pure-tokenizers convention; explicit `Close()` avoids finalizer races | — Pending |
| C-style error codes at FFI + typed Go errors at wrapper | pure-tokenizers convention; clean separation of FFI from idiomatic Go | — Pending |
| Parser/Doc reuse baked into v0.1 API | simdjson's primary performance lever — retrofitting later would be an API break | — Pending |
| Defer simdjson API style choice (DOM vs On-Demand vs tape) | Binding style affects entire surface; phase discussion should choose after reading simdjson's own API guidance | — Pending |

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
*Last updated: 2026-04-14 after initialization*
