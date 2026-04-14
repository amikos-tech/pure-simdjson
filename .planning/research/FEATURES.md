# Feature Research

**Domain:** High-performance Go JSON parser (SIMD-accelerated, native-lib backed, purego-consumed)
**Researched:** 2026-04-14
**Confidence:** HIGH (Context-adjacent sources: simdjson docs, simdjson-go README, sonic DeepWiki, goccy/go-json README, Go stdlib issue tracker)

## Competitive Matrix

Feature comparison across the five relevant points in the Go JSON space plus the upstream C++ source.

| Capability | `encoding/json` (stdlib) | `minio/simdjson-go` | `bytedance/sonic` | `goccy/go-json` | `sugawarayuuta/sonnet` | `simdjson` C++ (upstream) |
|---|---|---|---|---|---|---|
| **Parse: DOM tree** | via `any`/`map`/`struct` | yes (tape + `Iter`) | via `ast.Node` / struct | via struct/`any` | via struct/`any` | yes (`dom::parser`) |
| **Parse: On-Demand / lazy** | no | no | partial (`ast.Searcher`) | no | no | yes (`ondemand::parser` — flagship) |
| **Parse: visitor/callback** | `Decoder.Token()` (slow) | `ForEach` on `Iter` | no first-class | no | no | iterator model (`for value : array`) |
| **Parse: NDJSON streaming** | `Decoder.Decode` loop | `ParseND` (x86 only) | no | `Decoder` | `Decoder` | `iterate_many` / `parse_many` (multi-threaded, 3.5 GB/s) |
| **Value: reflection Unmarshal** | yes (primary) | no | yes (JIT-compiled) | yes | yes | n/a (C++) |
| **Value: typed accessors** | no (go through `any`) | yes via `Iter` getters | yes via `ast.Node` | no | no | yes — `double()`, `int64_t()`, `uint64_t()`, `bool()`, `string_view()` |
| **Value: path extraction** | no | no (must traverse) | `Searcher.GetByPath` (dot/index) | no | no | `at_pointer()` (RFC 6901) + `at_path()` (RFC 9535 subset) |
| **Numbers: `int64`/`uint64`/`float64` distinct** | only via `json.Number` + explicit call | yes (tape encodes type) | yes (`ast.Node.Int64`/`Float64`) | matches stdlib | matches stdlib | yes — separate cast operators |
| **Numbers: precision preservation to `any`** | NO (all → float64, lossy >2^53) | yes (tape holds original) | yes (json.Number path) | partial | partial | yes |
| **Numbers: big int / arbitrary precision** | `json.Number` (string) | no native, can read raw | `json.Number` | no | no | no native (string slice available) |
| **Strings: zero-copy view** | no (always copies) | opt-in `WithCopyStrings(false)` (tied to input buffer) | partial (`ast.Node` slice) | no | no | `string_view` / `get_raw_json_string()` |
| **Strings: UTF-8 validation** | yes (mandatory) | yes | yes | yes | yes | yes (13 GB/s UTF-8 validator) |
| **Errors: position info** | partial (offset on syntax err) | limited | offset + message | stdlib-compat | stdlib-compat | yes (`error_code` enum + context) |
| **Errors: structured error codes** | no (string-matching) | error values | error values | wrapped stdlib | wrapped stdlib | yes (enum `error_code`) |
| **Concurrency: parser thread-safety** | `Decoder` not safe; `Unmarshal` reentrant | `ParsedJson` not safe, reusable serially | thread-safe encoders, pooled decoders | safe per-call | safe per-call | parser NOT shared; one per thread |
| **Concurrency: parallel NDJSON** | no | manual | no | no | no | built-in (main + worker thread) |
| **Lifecycle: explicit close/free** | none needed (GC) | `ParsedJson.Reset()` | pool pretouch | none | none | parser lifetime scoped; memory owned by parser |
| **Lifecycle: parser/doc reuse** | n/a | yes (`ParsedJson` reusable) | JIT cache per type | no | no | yes — primary perf lever ("terabytes with zero new allocation") |
| **Validation: JSON Schema** | no | no | no | no | no | no |
| **SIMD kernels** | none | AVX2 + CLMUL, **x86-64 only** | AVX2 / AVX-512 (x86) + ARM64 asm | none (VM) | SWAR | runtime dispatch: ICX/AVX-512, Haswell/AVX2, Westmere/SSE4.2, ARM NEON, fallback |
| **Platforms** | all | linux/darwin/windows amd64 only | amd64 + arm64 (limited) | all | all | all major |

Confidence on matrix rows: HIGH where cited to library docs; MEDIUM where inferred from README feature lists without code inspection.

## Feature Landscape

### Table Stakes (Users Will Not Adopt Without)

| Feature | Why Expected | Complexity | Notes |
|---|---|---|---|
| Parse `[]byte` → document handle | Baseline; everything else builds on it | LOW | Core happy path. Matches `simdjson::ondemand::parser::iterate`. |
| Typed accessors for string / bool / null / array / object | Without these, library is unusable | LOW | Mirror simdjson's `get_string()`, `get_bool()`, `is_null()`, `get_array()`, `get_object()`. |
| Distinct `int64` / `uint64` / `float64` getters | The explicit stdlib gap — this is why users come | LOW | Upstream has these natively; exposing them is the biggest stdlib differentiator after raw speed. |
| Iteration over arrays and object fields | Every non-trivial JSON has these | LOW | Visitor / range pattern. |
| Explicit `Close()`/`Free()` on handles | FFI resources cannot rely on GC finalizers alone | LOW | Established pure-tokenizers convention. |
| Structured Go errors with message + offset | Debugging is impossible otherwise | MEDIUM | Map C-style error codes → typed Go errors at wrapper boundary. |
| UTF-8 validation on parse | simdjson validates by default; disabling is surprising | LOW | Free — upstream already does this. |
| Works on all 6 target platforms | Stated constraint; partial support blocks adoption | HIGH | Biggest lift is build/distribution, not feature code. |
| Benchmarks vs `encoding/json` and `simdjson-go` | Users will not take perf claims on faith | LOW | Twitter.json, canada.json, citm_catalog.json are the standard corpus. |

### Differentiators (Why Choose `pure-simdjson` over `encoding/json` or `simdjson-go`)

| Feature | Value Proposition | Complexity | Notes |
|---|---|---|---|
| **Parser/Doc reuse API** | simdjson's *primary* perf lever — "parse terabytes with zero new allocation." Retrofitting later would be an API break. Backs the v0.1 requirement. | MEDIUM | Directly exposes `ondemand::parser` + iteration. Without this, we leave ~2× perf on the floor. |
| **On-Demand API with selective path extraction** | The single biggest win over `simdjson-go`. Ami-gin's GIN ingest reads ~3 known fields from fat JSON — On-Demand skips the rest. | HIGH | Wraps `at_pointer()` / field-chain access. Must decide: caller-supplied paths vs builder DSL. |
| **ARM64 + Apple Silicon support** | `simdjson-go` is x86-only — `pure-simdjson` runs on darwin/arm64 and linux/arm64 via upstream NEON kernel. Huge for M-series dev machines and Graviton. | MEDIUM | Free from upstream kernel dispatch; cost is in the build/distribution matrix. |
| **No-cgo consumer build** | The whole point of `pure-*`. Consumers get native speed without cgo toolchain. | HIGH | Lives in the Rust shim + purego binding, not in feature code. |
| **NDJSON parallel streaming** | Upstream sustains 3.5 GB/s multithreaded. Log/event/JSONL pipelines are the target workload. v0.2 scope. | HIGH | Wrap `iterate_many`. Must decide whether Go callback runs on the worker thread or via channel. |
| **Zero-copy string views tied to Doc lifetime** | Kills allocation overhead in field-heavy workloads. v0.2 scope. | MEDIUM | Return `[]byte` aliasing native buffer; document the aliasing contract loudly. |
| **Precision-preserving number tape** | `encoding/json` silently loses precision above 2^53 when target is `any` — an actual data-loss bug for log/event IDs, financial amounts, Twitter IDs. | LOW | Follows directly from exposing the 3 typed getters. |
| **Loud failure on unsupported CPU** | Users want to know now, not mysteriously slow later. Matches stated constraint. | LOW | Check kernel dispatch result at init; return error if only fallback available. |

### Anti-Features (Deliberately NOT Built)

Validating the PROJECT.md out-of-scope list and flagging any that warrant reconsideration:

| Feature | Why Requested | Why Problematic | Verdict |
|---|---|---|---|
| **JSON marshalling/encoding** | Symmetric API is "nice" | Different perf shape, different code path, doubles surface. simdjson itself is parse-only. | **KEEP OUT** — validated. |
| **Struct binding / reflection `Unmarshal`** | Feels like the drop-in story | Reflection defeats simdjson's perf story; sonic's JIT is the only workable version and it is an enormous project on its own. | **KEEP OUT** — validated. Caller writes explicit extractors. Revisit if real consumer feedback says extraction ergonomics are unbearable. |
| **Full JSONPath (RFC 9535) query engine** | "Extract anything with one call" | Full JSONPath is a query engine; upstream simdjson only implements a subset via `at_path()`. A full engine is its own library. | **KEEP OUT** — but **expose the upstream `at_pointer()` (RFC 6901) + `at_path()` subset** as part of the On-Demand API. They are already there, free, and cover the common cases. |
| **JSON Schema validation** | Common CI/ingress concern | Orthogonal to parsing; plenty of existing libraries consume a parse tree. | **KEEP OUT** — validated. |
| **cgo build path** | "Just in case" | The whole point of `pure-*`. Doubles CI + release surface permanently. | **KEEP OUT** — validated. |
| **Silent SIMD-disabled fallback** | "Best-effort on exotic CPUs" | Masks perf cliffs; user thinks they have multi-GB/s and get 50 MB/s. | **KEEP OUT** — validated. Fail loudly. |
| **Arena allocator exposed to Go** | Advanced users want manual memory control | Parser/Doc reuse already gives you ~all the benefit; native arena is not safely crossable through purego. | **KEEP OUT — add implicitly.** Reuse API is the Go-idiomatic shape of this. |
| **In-place mutation / re-serialization** | simdjson-go has it | Upstream simdjson does *not*; comes from tape-format design that we are deliberately skipping in favour of On-Demand. Mutation on On-Demand is nonsensical. | **KEEP OUT** — explicit addition; should be documented so expectations do not leak from simdjson-go. |
| **Multi-document streaming from `io.Reader`** | `encoding/json` has `Decoder.Decode` | Upstream `iterate_many` works on padded `[]byte`, not a streaming reader. Adapting needs an internal buffer + resync. | **DEFER to v0.2+** — `[]byte` NDJSON only for v0.2; `io.Reader` adapter is a later wrapper. |

## Feature Dependencies

```
[Rust FFI shim + C ABI]
    └── [Handle lifecycle: New/Close/Free]
         ├── [Parse []byte → Doc]
         │    ├── [Typed accessors: string/bool/null/array/object]
         │    ├── [Typed numbers: int64/uint64/float64]
         │    ├── [Iteration: arrays + object fields]
         │    └── [Parser/Doc reuse API]           ◄─── v0.1 perf-critical
         │
         ├── [On-Demand API]                       ◄─── v0.2
         │    ├── requires── [Parse + typed accessors]
         │    ├── enables──> [at_pointer / JSON Pointer extraction]
         │    └── enables──> [Zero-copy string views]
         │
         ├── [NDJSON streaming parse]              ◄─── v0.2
         │    ├── requires── [Parser/Doc reuse API]
         │    └── depends-on── [iterate_many threading model decision]
         │
         └── [Structured Go errors]
              └── requires── [C error code convention]

[Binary bootstrap: CloudFlare R2 download]
    └── required-by── everything runtime

[CPU feature detection + loud failure]
    └── blocks── [Parse []byte → Doc] on unsupported CPUs
```

### Dependency Notes

- **Parser/Doc reuse is a v0.1 gate, not a v0.2 optimization.** simdjson's own performance notes state reuse is what turns "fast parse" into "terabytes with zero new allocation." Retrofitting a reuse API onto a non-reuse API is an API break, not an addition. This is why PROJECT.md correctly puts it in v0.1.
- **On-Demand enables zero-copy strings cleanly.** DOM-style APIs tend to materialize; On-Demand's iterator model naturally returns views into the input buffer. Combining them in v0.2 is coherent.
- **NDJSON parallel parse conflicts with goroutine-pinned handles.** The v0.2 NDJSON design has to answer: does the Go callback fire on upstream's internal worker thread? If yes, the callback must be C-callable and re-entrancy-safe. If no, we marshal results through a channel, losing some of the upstream speedup. This is the hardest design decision in v0.2.
- **Selective path extraction depends on On-Demand semantics.** With DOM, `at_pointer()` is just a walk; with On-Demand, it is a one-pass skip. The perf story for ami-gin depends on the On-Demand variant.

## MVP Definition

### Launch With (v0.1)

Ruthlessly minimum — ship to validate the core thesis that a no-cgo SIMD parser with typed numbers is worth adopting.

- [ ] `Parse([]byte) (*Doc, error)` — **P1** — entry point
- [ ] `Doc.Close() / Free()` — **P1** — no-finalizer lifecycle
- [ ] Typed accessors: `GetString`, `GetBool`, `IsNull`, iteration over array / object — **P1** — table stakes
- [ ] Distinct number getters: `GetInt64`, `GetUint64`, `GetFloat64` — **P1** — the stdlib-gap differentiator
- [ ] Parser/Doc reuse API (allocate once, reuse across parses) — **P1** — perf lever, API-break risk if deferred
- [ ] Explicit CPU feature check with loud failure — **P1** — stated constraint
- [ ] Structured Go errors (code + message + offset if available) — **P1**
- [ ] Six-platform shared library distribution via R2 — **P1** — stated constraint
- [ ] Benchmarks vs `encoding/json` + `minio/simdjson-go` on canonical corpus — **P1** — claims need receipts

### Add After Validation (v0.2)

- [ ] On-Demand API with field-chain access — **P2** — trigger: ami-gin ingest pilot
- [ ] `at_pointer()` / `at_path()` extraction — **P2** — trigger: on-demand landed
- [ ] Zero-copy string views tied to `Doc` lifetime — **P2** — trigger: on-demand landed
- [ ] NDJSON streaming parse on `[]byte` — **P2** — trigger: log/JSONL consumer signal
- [ ] Parser pool / goroutine-per-parser guidance + helpers — **P2** — trigger: concurrency patterns emerge from real usage

### Future Consideration (v0.3+)

- [ ] `io.Reader` NDJSON adapter (internal buffer + resync on newline) — **P3** — trigger: streaming use cases
- [ ] Raw JSON slice accessor (return unparsed sub-document as `[]byte`) — **P3** — trigger: pass-through scenarios (store-and-forward)
- [ ] Minify helper (upstream has 6 GB/s minifier) — **P3** — trigger: explicit ask; cheap to wrap
- [ ] Standalone UTF-8 validator export (13 GB/s) — **P3** — trigger: explicit ask

### Explicitly Out of Scope (do not add without a requirements review)

- JSON encoding / marshalling
- Reflection-based `Unmarshal` into Go structs
- Full JSONPath query engine
- JSON Schema validation
- cgo build path
- Silent SIMD fallback
- In-place document mutation / re-serialization (simdjson-go has this; we do not)

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---|---|---|---|
| Parse + Doc + Close | HIGH | LOW | P1 |
| Typed number getters (int64/uint64/float64) | HIGH | LOW | P1 |
| Typed string/bool/array/object accessors + iteration | HIGH | LOW | P1 |
| Parser/Doc reuse API | HIGH | MEDIUM | P1 |
| Six-platform R2 distribution | HIGH | HIGH | P1 |
| Structured errors | MEDIUM | MEDIUM | P1 |
| CPU feature detection + loud failure | MEDIUM | LOW | P1 |
| Benchmarks vs stdlib + simdjson-go | HIGH | LOW | P1 |
| On-Demand + path extraction | HIGH | HIGH | P2 |
| Zero-copy string views | MEDIUM | MEDIUM | P2 |
| NDJSON parallel streaming | HIGH | HIGH | P2 |
| `io.Reader` NDJSON adapter | MEDIUM | MEDIUM | P3 |
| Raw JSON slice accessor | LOW | LOW | P3 |
| Minify helper | LOW | LOW | P3 |

## Consumer Pain Points (Real-World Signal)

**ami-gin Phase 07 build-time ingest** (immediate consumer, from PROJECT.md):
- Bottleneck today: `json.Unmarshal(..., &any)` — byte-at-a-time state machine + boxed `map[string]any` tree.
- What it actually needs: selective extraction of a small, known field set from each document. **This is literally the On-Demand API's textbook use case.**
- Inferred from PROJECT.md: a DOM-materialize-then-walk approach (simdjson-go's shape) still leaves perf on the table for this workload. On-Demand is what justifies the project vs. just vendoring simdjson-go.

**Log/event processors** (secondary consumer class):
- Pattern: read a file or socket of NDJSON; extract 2–5 known fields per line; discard rest.
- Pain on `encoding/json`: per-line `Unmarshal` allocates heavily; GC pressure dominates above ~50 MB/s.
- Pain on `simdjson-go`: x86-only, so Apple Silicon / Graviton deployments can't adopt. No parallel NDJSON.
- What they need from us: v0.1 reuse + v0.2 NDJSON streaming. ARM64 support alone is a reason to switch from `simdjson-go`.

**JSONL tooling** (tertiary):
- Today's ceiling on Go side: ~200–400 MB/s with `goccy/go-json` or `sonic`, reflection-based.
- Upstream simdjson does 3.5 GB/s multi-threaded on the same workload. The gap is an order of magnitude.

**`encoding/json` number precision** (cross-cutting):
- Documented bug-class: integers above 2^53 (e.g. Twitter IDs, Snowflake IDs, log sequence numbers) silently round when decoded into `any`. Fix requires `json.Decoder` + `UseNumber()` + explicit `Int64()` conversion — easy to forget, easy to miss in tests.
- Our typed getters eliminate this entire class of bug.

## The v0.1 "Parser/Doc Reuse" Requirement — Backing Data

PROJECT.md makes Parser/Doc reuse a v0.1 requirement. The simdjson upstream performance docs support this decisively:

> "If you're parsing multiple documents in a loop, you should make a parser once and reuse it. The simdjson library will allocate and retain internal buffers between parses, keeping buffers hot in cache and keeping memory allocation and initialization to a minimum. In this manner, you can parse terabytes of JSON data without doing any new allocation."
> — [simdjson performance.md](https://github.com/simdjson/simdjson/blob/master/doc/performance.md)

And from the `iterate_many` docs:

> "simdjson creates only one parser object and therefore allocates its memory once, then recycles it for every document in a given file."
> — [simdjson iterate_many](https://simdjson.github.io/simdjson/md_doc_iterate_many.html)

Concretely: for ami-gin's GIN ingest (many documents processed in a loop at build time), not exposing reuse in v0.1 would mean every `Parse()` re-allocates parser-side scratch buffers. That is a 1.5–2× perf regression vs. the achievable ceiling, and the API shape (`Parser.Parse(buf) → Doc` vs. `Parse(buf) → Doc`) is not retrofittable without a breaking change. PROJECT.md's decision to bake it in at v0.1 is correct and well-supported.

## Competitor Feature Cross-Reference (Summary)

| Feature | stdlib | simdjson-go | sonic | goccy | sonnet | Our Approach |
|---|---|---|---|---|---|---|
| SIMD | no | AVX2 x86 only | AVX2/512 + ARM | no | SWAR | Upstream C++ (full kernel dispatch) — **our win** |
| Typed int64/uint64/float64 | via `json.Number` | yes | yes | stdlib-compat | stdlib-compat | yes — **table stakes** |
| On-Demand / selective | no | no | `ast.Searcher` | no | no | yes (`at_pointer` + field chain) — **our flagship diff** |
| Zero-copy strings | no | opt-in | partial | no | no | yes, lifetime-tied — **v0.2 diff** |
| Parallel NDJSON | no | no | no | no | no | yes (`iterate_many`) — **v0.2 diff** |
| ARM64 | yes | **no** | yes | yes | yes | yes — **our win over simdjson-go** |
| No-cgo consumer build | yes | yes (pure Go) | yes | yes | yes | yes (purego) — **parity not differentiator; table stakes for Go ecosystem** |
| Reflection Unmarshal | yes (primary) | no | yes (JIT) | yes | yes | **NO — explicit anti-feature** |

**Strategic read:** The unique position is "SIMD upstream + On-Demand + ARM64 + no-cgo." No existing library hits all four. `sonic` has SIMD + ARM64 + no-cgo but no On-Demand selective parse. `simdjson-go` has On-Demand-adjacent access but is x86-only. `pure-simdjson` wins by composing upstream simdjson's actual flagship feature (On-Demand) with purego distribution and upstream's full CPU dispatch.

## Sources

- [simdjson C++ — basics.md](https://github.com/simdjson/simdjson/blob/master/doc/basics.md) — HIGH: typed accessors, JSON Pointer
- [simdjson C++ — performance.md](https://github.com/simdjson/simdjson/blob/master/doc/performance.md) — HIGH: parser reuse rationale
- [simdjson C++ — ondemand_design.md](https://github.com/simdjson/simdjson/blob/master/doc/ondemand_design.md) — HIGH: On-Demand semantics and limits
- [simdjson C++ — iterate_many.md](https://simdjson.github.io/simdjson/md_doc_iterate_many.html) — HIGH: 3.5 GB/s NDJSON, two-thread model
- [simdjson — main README](https://github.com/simdjson/simdjson) — HIGH: kernel dispatch, perf claims, consumer list
- [minio/simdjson-go — README](https://github.com/minio/simdjson-go) — HIGH: AVX2+CLMUL x86-only requirement, `SupportedCPU()`, ForEach, WithCopyStrings, in-place mutation
- [minio/simdjson-go — blog post](https://blog.min.io/simdjson-go-parsing-gigabyes-of-json-per-second-in-go/) — HIGH: ~40–60% of upstream, ~10× stdlib
- [bytedance/sonic — DeepWiki](https://deepwiki.com/bytedance/sonic) — HIGH: JIT decode, Pretouch, ast.Searcher, AMD64 + ARM64 optimization
- [bytedance/sonic — INTRODUCTION.md](https://github.com/bytedance/sonic/blob/main/docs/INTRODUCTION.md) — HIGH: feature list
- [goccy/go-json — README](https://github.com/goccy/go-json) — HIGH: encoding/json-compat, VM-based
- [sugawarayuuta/sonnet — README](https://github.com/sugawarayuuta/sonnet) — MEDIUM: ~5× stdlib; SWAR; stdlib-compat
- [Go stdlib — json package docs](https://pkg.go.dev/encoding/json) — HIGH: `UseNumber`, `json.Number.Int64/Float64`
- [Go issue #12409 — large-number precision loss](https://github.com/golang/go/issues/12409) — HIGH: confirmed stdlib behavior
- [Go issue #6384 — encode precision](https://github.com/golang/go/issues/6384) — HIGH: longstanding stdlib precision issues
- [On-Demand JSON paper — Keiser & Lemire 2024](https://onlinelibrary.wiley.com/doi/10.1002/spe.3313) — HIGH: academic rationale for On-Demand vs DOM

---
*Feature research for: high-performance Go JSON parser (SIMD, purego, handle-based)*
*Researched: 2026-04-14*
