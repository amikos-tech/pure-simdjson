# Phase 8: Low-overhead DOM traversal ABI and specialized Go any materializer - Research

**Researched:** 2026-04-23
**Domain:** Go/Rust/C++ FFI DOM traversal, internal materialization, benchmark validation
**Confidence:** HIGH for constraints and integration points; MEDIUM for exact benchmark gain until Phase 8 measurements exist

<user_constraints>
## User Constraints (from CONTEXT.md)

Source for this copied section: [VERIFIED: .planning/phases/08-low-overhead-dom-traversal-abi-and-specialized-go-any-materi/08-CONTEXT.md]

### Locked Decisions

### Traversal ABI Shape
- **D-01:** Use a bulk traversal/frame-style fast path as the core direction: native should walk a document or subtree once and return compact traversal data that Go can consume without a per-node FFI call.
- **D-02:** Treat the "expose the tape" idea as a strong research/planning vector, not a hard implementation mandate. Planning should explicitly compare a tape-like internal view over Rust-owned buffers against any separate frame-stream design before locking tasks.
- **D-03:** The new traversal/materialization ABI is internal first. It may be used by the Go wrapper and benchmark path, but it is not promised as a public C ABI surface in Phase 8.
- **D-04:** Support both whole-document and subtree materialization as the intended design envelope, while allowing the planner to choose the first implementation slice if doing both at once is too large.
- **D-05:** Preserve the current accessor ABI untouched and add the low-overhead path in parallel. Existing public DOM accessors, iterators, and error semantics remain stable.

### String And Key Handoff
- **D-06:** Object keys should be represented as slices or offsets inside the internal frame/tape view where possible. Go copies keys only when building the final `map[string]any`.
- **D-07:** String values are copied into Go only when materializing a string. Internal traversal may view Rust-owned bytes while the owning `Doc` is alive.
- **D-08:** Borrowed Rust memory must not escape into public Go values. User-visible strings, maps, and slices own Go memory by the time they escape the internal materializer.
- **D-09:** Lifetime safety for internal borrowed views should rely on explicit `Doc`/materializer ownership, `runtime.KeepAlive`, and existing `Close`/finalizer discipline. Debug live-view tracking is not required unless planning finds a cheap, useful way to add it.

### Go any Builder Semantics
- **D-10:** The specialized builder should match current accessor numeric semantics: preserve `int64`, `uint64`, and `float64` distinctions, and surface existing precision/range errors rather than collapsing everything to `float64`.
- **D-11:** Full `map[string]any` materialization uses ordinary Go map assignment semantics for duplicate keys, so the last duplicate key wins. This intentionally differs from `Object.GetField`, which remains a first-match DOM lookup.
- **D-12:** Arrays and objects should use exact or near-exact preallocation from traversal metadata instead of the current fixed small capacities.
- **D-13:** Materialization fails fast with existing typed errors. Wrong type, range, precision, invalid handle, and closed document cases should map to current public error behavior.

### Exposure And Proof Bar
- **D-14:** Keep the specialized materializer internal/benchmark-facing first. Do not add `Element.Interface()`, `Doc.Interface()`, or another public convenience API until the path is measured and validated.
- **D-15:** Phase 8 closeout must prove correctness plus benchmark delta: parity/oracle tests plus Tier 1 diagnostic evidence showing materialization improvement.
- **D-16:** The benchmark target is improvement over the Phase 7 baseline in both materialize-only and full Tier 1 paths. Phase 8 does not require beating `encoding/json + any` before closeout.
- **D-17:** Phase 8 documentation should be internal docs and benchmark notes only. Public README/result repositioning waits for Phase 9.

### Claude's Discretion

- Exact frame/tape struct names and layouts, as long as they preserve the decisions above and remain internal for Phase 8.
- Whether the first implementation slice targets whole-document materialization, subtree materialization, or both, based on risk and testability.
- Whether to add debug-only borrowed-view tracking, if it proves cheap and helpful.
- Exact benchmark command grouping and artifact naming, as long as Phase 7 baseline comparison remains clear.

### Deferred Ideas (OUT OF SCOPE)

- Public `Element.Interface()` / `Doc.Interface()` convenience APIs are deferred until after the internal materializer is measured.
- Borrowed-memory public/unsafe APIs are out of Phase 8. Public Go values must own Go memory.
- JSONPointer/path lookup helpers are not in Phase 8 unless planning proves they are required for the materializer. They may belong to later selective traversal or v0.2 work.
- Public README benchmark repositioning, headline claim changes, and any release decision are Phase 9 work.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| D-01 | Bulk traversal/frame-style fast path without per-node FFI. | Current benchmark materializer recursively calls public accessors and iterators per node; use one internal traversal handoff instead. [VERIFIED: benchmark_comparators_test.go:368] [VERIFIED: iterator.go:33] |
| D-02 | Compare tape-like internal view against frame-stream design. | simdjson uses a tape in document order, but this repo already reconstructs elements from `simdjson::internal::tape_ref`; prefer a repo-owned frame stream over exporting raw tape. [VERIFIED: third_party/simdjson/doc/tape.md] [VERIFIED: src/native/simdjson_bridge.cpp:212] |
| D-03 | Internal first, not public C ABI. | Public header and contract define the stable ABI; cbindgen/header audit rejects unexpected `pure_simdjson_*` symbols. [VERIFIED: docs/ffi-contract.md] [VERIFIED: tests/abi/check_header.py:168] |
| D-04 | Whole-document and subtree design envelope. | Existing `ValueView` can identify root and descendants via doc handle, generation, and index state, so the internal builder can accept a doc-tied view even if the first slice is root-only. [VERIFIED: src/runtime/registry.rs:641] [VERIFIED: src/runtime/registry.rs:669] |
| D-05 | Preserve public accessor ABI untouched. | `include/pure_simdjson.h` currently exposes parser/doc/accessor/iterator/object lookup symbols; Phase 8 should add parallel internal symbols rather than changing those. [VERIFIED: include/pure_simdjson.h] |
| D-06 | Key handoff by borrowed slice/offset until final Go map. | Current object iteration copies every key through `ElementGetString`; frame metadata can carry key pointer/length for one final Go string copy. [VERIFIED: iterator.go:82] [VERIFIED: internal/ffi/bindings.go:262] |
| D-07 | String values copy only when materialized. | Public string API currently allocates native bytes, copies to Go string, then frees; the internal path can read borrowed bytes while the doc is alive and copy once into Go. [VERIFIED: src/runtime/registry.rs:752] [VERIFIED: internal/ffi/bindings.go:262] |
| D-08 | Borrowed Rust/native memory must not escape. | Public contract says strings copy out and borrowed views are outside the public contract. [VERIFIED: docs/ffi-contract.md] |
| D-09 | Lifetime via Doc/materializer ownership and `runtime.KeepAlive`. | Existing Go bindings use `runtime.KeepAlive` for parse inputs; Go runtime documents it as keeping finalizers from running before the call site. [VERIFIED: internal/ffi/bindings.go:170] [VERIFIED: go doc runtime.KeepAlive] |
| D-10 | Preserve int64/uint64/float64 semantics and precision/range errors. | Existing accessor materializer switches by exact type and uses split numeric getters; float conversion rejects non-exact integer conversion. [VERIFIED: benchmark_comparators_test.go:418] [VERIFIED: element.go:168] |
| D-11 | Full map materialization last duplicate wins; `Object.GetField` first match remains. | Go map assignment overwrites earlier values, while public `Object.GetField` uses native DOM `at_key` semantics already tested as first duplicate. [VERIFIED: benchmark_comparators_test.go:457] [VERIFIED: iterator_test.go:311] |
| D-12 | Use exact or near-exact preallocation. | Current materializer uses fixed capacity 8 for arrays and maps; simdjson tape start words carry immediate child count with saturation, and a frame builder can compute exact counts while traversing. [VERIFIED: benchmark_comparators_test.go:442] [VERIFIED: third_party/simdjson/doc/tape.md] |
| D-13 | Fail fast with existing typed errors. | Public wrappers translate error codes into typed Go errors; internal fast path should route through the same error mapping. [VERIFIED: internal/ffi/bindings.go:115] |
| D-14 | No public `Interface` APIs in Phase 8. | Roadmap and context place public API exposure after measurement, and Phase 9 owns public benchmark repositioning. [VERIFIED: .planning/ROADMAP.md] [VERIFIED: 08-CONTEXT.md] |
| D-15 | Correctness plus benchmark delta required. | Existing diagnostic benchmarks isolate full, parse-only, and materialize-only cuts and raw Phase 7 artifacts provide the comparison point. [VERIFIED: benchmark_diagnostics_test.go:98] [VERIFIED: testdata/benchmark-results/v0.1.1/tier1-diagnostics.bench.txt] |
| D-16 | Improve Phase 7 materialize-only and full Tier 1; no need to beat stdlib. | Phase 7 baseline shows materialization dominates parse and current Tier 1 trails `encoding/json + any`. [VERIFIED: docs/benchmarks/results-v0.1.1.md] [VERIFIED: 07-LEARNINGS.md] |
| D-17 | Internal docs and benchmark notes only. | Phase 9 boundary explicitly owns public benchmark-story update and release decision. [VERIFIED: .planning/ROADMAP.md] [VERIFIED: 08-CONTEXT.md] |
</phase_requirements>

## Project Constraints (from Project Instructions)

- No repository-local `CLAUDE.md` file exists, and no repository-local `AGENTS.md` file exists. [VERIFIED: `test -f CLAUDE.md`, `test -f AGENTS.md`]
- Do not include conversation-supplied private repository hostnames or private repository details in commit messages, pull requests, generated planning artifacts, or related release artifacts. [VERIFIED: user prompt]
- The repo-local `pure-simdjson-release` skill applies only to tag-driven release operations; Phase 8 is not a release operation. [VERIFIED: .agents/skills/pure-simdjson-release/SKILL.md] [VERIFIED: 08-CONTEXT.md]
- `.planning/config.json` has `workflow.nyquist_validation: true`, `commit_docs: true`, and no explicit `security_enforcement: false`, so this research includes Validation Architecture and Security Domain sections. [VERIFIED: .planning/config.json]

## Summary

Phase 8 should implement a repo-owned internal frame-stream traversal ABI, not a public raw simdjson tape export. [VERIFIED: 08-CONTEXT.md] [VERIFIED: docs/ffi-contract.md] The frame stream should be generated by native code in one traversal of a `Doc` root or `ValueView` subtree, then consumed by a specialized Go builder that copies strings/keys only when constructing escaped Go values. [VERIFIED: src/native/simdjson_bridge.cpp:212] [VERIFIED: benchmark_comparators_test.go:418]

The key planning risk is lifetime and ABI leakage, not the existence of a traversal source. [VERIFIED: docs/ffi-contract.md] The C++ bridge already relies on `simdjson::internal::tape_ref` to reconstruct elements by tape index, so Phase 8 can use tape knowledge privately while exposing a stable internal frame format to Go. [VERIFIED: src/native/simdjson_bridge.cpp:212] [VERIFIED: third_party/simdjson/doc/tape.md]

Benchmark validation should compare against Phase 7 raw artifacts, especially materialize-only diagnostics. [VERIFIED: docs/benchmarks.md] [VERIFIED: testdata/benchmark-results/v0.1.1/tier1-diagnostics.bench.txt] Current Phase 7 materialize-only medians are much larger than parse-only diagnostics for twitter, citm, and canada fixtures, making traversal/materialization overhead the correct target. [VERIFIED: docs/benchmarks/results-v0.1.1.md]

**Primary recommendation:** implement an internal `psdj_internal_*` frame-stream ABI plus Go fast materializer, keep all public accessors unchanged, and gate closeout on parity tests plus benchstat-backed improvement over Phase 7 diagnostics. [VERIFIED: 08-CONTEXT.md] [ASSUMED]

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|--------------|----------------|-----------|
| Parse and DOM ownership | Rust runtime registry | C++ bridge | Rust owns parser/doc handles and copied padded input; C++ owns simdjson parser/document internals. [VERIFIED: src/runtime/registry.rs:432] [VERIFIED: src/native/simdjson_bridge.cpp:316] |
| Low-overhead traversal build | C++ bridge | Rust runtime registry | C++ can walk simdjson DOM/tape internals; Rust should expose only status/out-param ABI and doc lifetime validation. [VERIFIED: src/native/simdjson_bridge.cpp:212] [VERIFIED: src/runtime/mod.rs:254] |
| Internal dynamic ABI export | Rust cdylib | Go purego binding | Go loads cdylib symbols through purego; Rust exports public ABI today and can add excluded internal symbols. [VERIFIED: internal/ffi/bindings.go:53] [VERIFIED: src/lib.rs:263] |
| Go `any` tree construction | Go wrapper/benchmark layer | Internal ffi package | Go must allocate final maps, slices, and strings so public values own Go memory. [VERIFIED: benchmark_comparators_test.go:418] [VERIFIED: docs/ffi-contract.md] |
| Public DOM accessors | Go wrapper + public C ABI | Rust registry + C++ bridge | Existing typed accessors, iterators, and `GetField` semantics remain the compatibility surface. [VERIFIED: element.go:80] [VERIFIED: include/pure_simdjson.h] |
| Benchmark proof | Go benchmark tests | scripts/bench + docs/testdata | Existing diagnostics isolate parse/materialize/full paths and raw Phase 7 artifacts are committed as baseline. [VERIFIED: benchmark_diagnostics_test.go:98] [VERIFIED: scripts/bench/run_benchstat.sh:1] |

## Standard Stack

### Core

| Library/Tool | Version | Publish/Source Date | Purpose | Why Standard |
|--------------|---------|---------------------|---------|--------------|
| Go toolchain | go1.26.2 local; module requires Go 1.24 | 2026 local install observed | Go wrapper, purego binding, tests, benchmarks | The project is a Go module with `go 1.24`; local verification uses Go 1.26.2. [VERIFIED: go.mod] [VERIFIED: `go version`] |
| Rust toolchain | rustc/cargo 1.89.0 | 2025-08-04 from `rustc --version` | cdylib, registry, FFI exports | The project builds the native runtime as a Rust crate with C++ bridge. [VERIFIED: Cargo.toml] [VERIFIED: `rustc --version`] |
| simdjson | v4.6.1 vendored | v4.6.1 tag in `third_party/simdjson` | DOM parser and tape/document internals | Existing bridge compiles vendored simdjson and uses DOM/tape refs. [VERIFIED: `git -C third_party/simdjson describe --tags --always --dirty`] [VERIFIED: build.rs:11] |
| `github.com/ebitengine/purego` | v0.10.0 | 2026-02-23 | cgo-free dynamic symbol calls from Go | Existing binding registers cdylib symbols through purego. [VERIFIED: go.mod] [VERIFIED: `go list -m -json github.com/ebitengine/purego`] |
| `cc` crate | v1.2.60 | version verified; publish date not emitted by `cargo info` | Compile C++ bridge into Rust build | Existing `build.rs` uses `cc::Build` with C++17. [VERIFIED: Cargo.toml] [VERIFIED: build.rs:40] |
| cbindgen | 0.29.2 | version verified; publish date not emitted by `cargo info` | Generated public header guard | `make verify-contract` regenerates/diffs the header and header audit checks public symbol shape. [VERIFIED: cbindgen.toml] [VERIFIED: Makefile:6] |

### Supporting

| Library/Tool | Version | Publish/Source Date | Purpose | When to Use |
|--------------|---------|---------------------|---------|-------------|
| `github.com/minio/simdjson-go` | v0.4.5 | 2023-03-11 | Design reference for Go-side tape walking | Use as reference only; do not add a runtime dependency. [VERIFIED: go.mod] [VERIFIED: `go list -m -json github.com/minio/simdjson-go`] |
| `github.com/bytedance/sonic` | v1.15.0 | 2026-01-22 | Benchmark comparator | Keep existing benchmark comparator coverage. [VERIFIED: go.mod] [VERIFIED: `go list -m -json github.com/bytedance/sonic`] |
| `github.com/goccy/go-json` | v0.10.6 | 2025-10-28 | Benchmark comparator | Keep existing benchmark comparator coverage. [VERIFIED: go.mod] [VERIFIED: `go list -m -json github.com/goccy/go-json`] |
| benchstat | `golang.org/x/perf` pseudo-version `v0.0.0-20260209182753-b57e4e371b65` | 2026-02-09 pseudo-version | Benchmark delta analysis | Use `scripts/bench/run_benchstat.sh` to compare Phase 8 raw output to Phase 7 raw artifacts. [VERIFIED: `go version -m /Users/tazarov/go/bin/benchstat`] [VERIFIED: scripts/bench/run_benchstat.sh:66] |
| Python | 3.11.7 | local install observed | ABI/header validation scripts | Required by existing ABI header audit tooling. [VERIFIED: `python3 --version`] [VERIFIED: tests/abi/check_header.py] |
| Apple clang / C++ compiler | Apple clang 21.0.0 local | local install observed | Compile C++17 bridge locally | Required by `cc` crate for the native bridge. [VERIFIED: `cc --version`] [VERIFIED: build.rs:40] |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Repo-owned internal frame stream | Raw simdjson tape view exposed to Go | Raw tape minimizes translation, but leaks simdjson internal layout into Go and makes version/layout changes high-risk. [VERIFIED: third_party/simdjson/doc/tape.md] [VERIFIED: src/native/simdjson_bridge.cpp:212] |
| Native traversal build plus Go builder | Current accessor-shaped recursive Go materializer | Current path is correct but pays per-node FFI/registry/string-copy overhead and uses fixed small container capacities. [VERIFIED: benchmark_comparators_test.go:418] [VERIFIED: iterator.go:82] |
| Go-owned final values | Public borrowed-memory API | Borrowed public values would violate the public contract that strings copy out and Phase 8's deferred borrowed-memory API decision. [VERIFIED: docs/ffi-contract.md] [VERIFIED: 08-CONTEXT.md] |

**Installation:**

No new runtime dependency is recommended for Phase 8. [VERIFIED: go.mod] Use the existing toolchain setup:

```bash
go mod download
cargo fetch
```

**Version verification commands:**

```bash
go list -m -json github.com/ebitengine/purego github.com/minio/simdjson-go github.com/bytedance/sonic github.com/goccy/go-json golang.org/x/sys
git -C third_party/simdjson describe --tags --always --dirty
cargo info cc --quiet
cbindgen --version
go version -m /Users/tazarov/go/bin/benchstat
```

## Architecture Patterns

### System Architecture Diagram

The diagram shows the recommended Phase 8 data flow; it does not describe a new public API. [VERIFIED: 08-CONTEXT.md]

```mermaid
flowchart LR
    JSON[Go []byte input] --> Parse[Parser.Parse]
    Parse --> Registry[Rust registry validates parser/doc generation]
    Registry --> NativeParse[C++ simdjson parse_into_document]
    NativeParse --> Doc[Doc handle owns padded input and simdjson document]

    Doc --> FastCall[Internal psdj_internal traversal build]
    FastCall --> Validate{View live and doc generation valid?}
    Validate -- no --> TypedErr[Existing typed Go error mapping]
    Validate -- yes --> NativeWalk[C++ walks root/subtree once]
    NativeWalk --> Frames[Repo-owned frame stream: kind, counts, key/string spans, numeric payload]
    Frames --> GoBuilder[Go specialized builder]
    GoBuilder --> Copy{String or key?}
    Copy -- yes --> GoCopy[Copy bytes into Go string]
    Copy -- no --> Scalars[Use scalar payload]
    GoCopy --> Values[map[string]any / []any / scalar]
    Scalars --> Values
    Values --> KeepAlive[runtime.KeepAlive Doc after borrowed reads]
```

### Recommended Project Structure

```text
src/
|-- lib.rs                    # Add excluded internal psdj_internal_* exports with ffi_wrap.
|-- runtime/
|   |-- registry.rs           # Validate doc/view, own/reuse traversal frame scratch, bridge native errors.
|   `-- mod.rs                # Add Rust declarations for private C++ traversal builder hooks.
`-- native/
    |-- simdjson_bridge.h     # Add private psimdjson_* traversal builder structs/functions.
    `-- simdjson_bridge.cpp   # Walk simdjson DOM/tape once and fill frames.

internal/ffi/
|-- types.go                  # Add internal-only frame/view mirror types.
`-- bindings.go               # Register internal psdj_internal_* symbols and wrap status errors.

element.go / parser.go        # Keep public behavior stable; only add unexported fast materializer entry.
benchmark_*_test.go           # Switch Tier 1 comparator/diagnostics to measured fast materializer path.
tests/abi/check_header.py     # Ensure internal symbols do not enter generated public header.
```

Every listed file is an existing integration point except new internal frame symbols/types. [VERIFIED: src/lib.rs] [VERIFIED: internal/ffi/bindings.go] [VERIFIED: benchmark_comparators_test.go]

### Pattern 1: Internal Frame Stream, Not Raw Tape ABI

**What:** Native code converts a doc/subtree into a compact, repo-owned sequence of frames with explicit kind, child count, borrowed key/string spans, and scalar payload fields. [VERIFIED: third_party/simdjson/doc/tape.md] [ASSUMED]

**When to use:** Use for full `any` materialization and benchmark diagnostics where all or most nodes are consumed. [VERIFIED: docs/benchmarks.md]

**Example:**

```rust
// Source: synthesized from existing ffi_wrap/status/out-param patterns.
// Keep this internal and excluded from cbindgen's public header.
#[repr(C)]
pub struct PsdjInternalDomFrame {
    pub kind: u32,
    pub flags: u32,
    pub child_count: u32,
    pub reserved: u32,
    pub key_ptr: *const u8,
    pub key_len: usize,
    pub string_ptr: *const u8,
    pub string_len: usize,
    pub i64_value: i64,
    pub u64_value: u64,
    pub f64_value: f64,
}

#[no_mangle]
pub unsafe extern "C" fn psdj_internal_dom_frames_build(
    view: ValueView,
    out_ptr: *mut *const PsdjInternalDomFrame,
    out_len: *mut usize,
) -> i32 {
    ffi_wrap(|| {
        // Validate handle/generation, build frames into doc-owned scratch,
        // and return pointer/length through out params.
        Ok(())
    })
}
```

### Pattern 2: Go Builder Copies Only at Escape Boundary

**What:** Go reads borrowed frame spans while the `Doc` is alive, but converts keys and string values into owning Go strings before returning. [VERIFIED: docs/ffi-contract.md] [VERIFIED: 08-CONTEXT.md]

**When to use:** Use inside unexported materializer code only; never expose frame pointers or borrowed byte slices publicly. [VERIFIED: 08-CONTEXT.md]

**Example:**

```go
// Source: synthesized from internal/ffi bindings and runtime.KeepAlive guidance.
func copyCString(ptr uintptr, n uintptr) string {
	if ptr == 0 || n == 0 {
		return ""
	}
	b := unsafe.Slice((*byte)(unsafe.Pointer(ptr)), int(n))
	return string(b) // copy into Go-owned memory
}

func materializeFast(doc *Doc, root Element) (any, error) {
	frames, err := ffi.InternalDomFramesBuild(root.view)
	if err != nil {
		return nil, err
	}
	defer runtime.KeepAlive(doc)
	return buildAnyFromFrames(frames)
}
```

### Pattern 3: Error Mapping Uses Existing Typed Errors

**What:** Internal bindings should return native status codes and call the same Go error mapping path used by public accessors. [VERIFIED: internal/ffi/bindings.go:115] [VERIFIED: src/lib.rs:177]

**When to use:** Use for invalid handle, closed doc, precision/range, and traversal corruption checks. [VERIFIED: element.go:168] [VERIFIED: parser_test.go]

### Anti-Patterns to Avoid

- **Public raw tape exposure:** This would couple public or Go-visible internals to simdjson tape layout rather than the repo's own contract. [VERIFIED: third_party/simdjson/doc/tape.md] [VERIFIED: docs/ffi-contract.md]
- **Adding `pure_simdjson_*` internal fast-path symbols without contract updates:** Header audit rejects unexpected public symbols, and Phase 8 says the traversal ABI is internal first. [VERIFIED: tests/abi/check_header.py:168] [VERIFIED: 08-CONTEXT.md]
- **Using `unsafe.String` or string headers for returned values:** That can create Go strings backed by non-Go memory; Phase 8 requires escaped public values to own Go memory. [VERIFIED: 08-CONTEXT.md] [VERIFIED: go doc runtime.KeepAlive]
- **Caching built Go maps/slices on `Doc`:** This would make materialize-only diagnostics measure cache reuse rather than traversal/materialization cost. [VERIFIED: docs/benchmarks.md] [ASSUMED]
- **Collapsing numeric values to `float64`:** Existing `any` materialization preserves `int64`, `uint64`, and `float64` distinctions. [VERIFIED: benchmark_comparators_test.go:418] [VERIFIED: element.go:168]

## Design Comparison: Tape-like View vs Frame Stream

| Dimension | Tape-like internal borrowed view | Repo-owned frame stream | Recommendation |
|-----------|----------------------------------|--------------------------|----------------|
| Crossing count | One crossing can expose tape and string buffers to Go. [VERIFIED: third_party/simdjson/doc/tape.md] | One crossing can expose frame pointer/length to Go. [ASSUMED] | Equivalent on crossing count if designed with pointer/length out params. [ASSUMED] |
| Coupling | Go must understand simdjson tape words, tags, string tape layout, object/key alternation, and saturated counts. [VERIFIED: third_party/simdjson/doc/tape.md] | Go understands only repo-owned frame layout. [ASSUMED] | Prefer frame stream to keep simdjson internals private. [ASSUMED] |
| Existing bridge fit | Bridge already uses `simdjson::internal::tape_ref`, so native can access tape indices. [VERIFIED: src/native/simdjson_bridge.cpp:212] | Bridge can translate tape/DOM data into frames in C++ and Rust can validate handles. [VERIFIED: src/runtime/registry.rs:669] [ASSUMED] | Use tape internally to build frames, not as Go ABI. [ASSUMED] |
| Child counts | Tape start words carry immediate child count with saturation; exact counts need scan when saturated. [VERIFIED: third_party/simdjson/doc/tape.md] | Builder can compute exact count during traversal and write it into frame metadata. [ASSUMED] | Frame stream better satisfies D-12. [ASSUMED] |
| Lifetime | Go would hold borrowed pointers directly into native/Rust-owned buffers. [VERIFIED: 08-CONTEXT.md] | Go holds borrowed frame/key/string spans for the duration of one materializer call. [ASSUMED] | Frame stream narrows the lifetime window. [ASSUMED] |
| Public ABI leakage | High risk if tape structs or `pure_simdjson_*` exports enter cbindgen/header. [VERIFIED: cbindgen.toml] [VERIFIED: tests/abi/check_header.py:168] | Lower risk if exported as `psdj_internal_*` and excluded from cbindgen/header audit. [ASSUMED] | Prefer frame stream plus explicit header guard. [ASSUMED] |
| Benchmark fairness | Raw tape may be fastest but harder to keep stable across simdjson upgrades. [ASSUMED] | Frame build adds native work but removes per-node registry/FFI/string allocation cost. [ASSUMED] | Measure both full and materialize-only diagnostics. [VERIFIED: docs/benchmarks.md] |

**Conclusion:** Implement a frame-stream ABI whose builder may use simdjson tape internals privately. [VERIFIED: src/native/simdjson_bridge.cpp:212] [ASSUMED] Do not expose raw tape layout to Go as the Phase 8 ABI. [VERIFIED: 08-CONTEXT.md] [ASSUMED]

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| JSON parsing or UTF-8 validation | A Go parser or custom lexer for materialization | Existing simdjson DOM parse | The parser already validates JSON before DOM access and Phase 8 is DOM-era work. [VERIFIED: .planning/PROJECT.md] [VERIFIED: benchmark_oracle_test.go] |
| Numeric classification | String reparse of numbers in Go | Existing simdjson/Rust numeric kind and split getters, encoded into frame payloads | Current API distinguishes signed, unsigned, and floating values and rejects inexact float-to-int conversions. [VERIFIED: element.go:80] [VERIFIED: element.go:168] |
| Public zero-copy borrowed strings | Unsafe public byte/string aliases into native memory | Internal borrowed spans plus final Go string copy | Public contract requires copy-out strings and Phase 8 forbids borrowed memory escaping. [VERIFIED: docs/ffi-contract.md] [VERIFIED: 08-CONTEXT.md] |
| Per-node lifetime validation | Descendant hash/iterator lease checks on every materializer node | One root/subtree validation before native traversal, plus doc-owned frame lifetime | Existing registry checks are correct for public accessors but are the overhead Phase 8 is trying to bypass. [VERIFIED: src/runtime/registry.rs:669] [ASSUMED] |
| Benchmark comparison tooling | Ad hoc spreadsheet or manual percent math | Existing raw artifacts plus benchstat script | `scripts/bench/run_benchstat.sh` already validates old/new files and invokes benchstat. [VERIFIED: scripts/bench/run_benchstat.sh:1] |

**Key insight:** the custom work should be a small internal representation and builder, not a second JSON parser, public borrowed-memory API, or new benchmark methodology. [VERIFIED: 08-CONTEXT.md] [VERIFIED: docs/benchmarks.md]

## Common Pitfalls

### Pitfall 1: Leaking Internal Symbols Into the Public Header

**What goes wrong:** A Phase 8 helper appears in `include/pure_simdjson.h` as a public `pure_simdjson_*` symbol. [VERIFIED: include/pure_simdjson.h]  
**Why it happens:** Rust `#[no_mangle] extern "C"` items are visible to cbindgen unless excluded/configured. [VERIFIED: cbindgen.toml]  
**How to avoid:** Use an internal prefix such as `psdj_internal_*`, add cbindgen exclusions, and update header audit to assert internal symbols are absent. [ASSUMED]  
**Warning signs:** `make verify-contract` or `tests/abi/check_header.py include/pure_simdjson.h` reports unexpected symbols. [VERIFIED: Makefile:6] [VERIFIED: tests/abi/check_header.py:168]

### Pitfall 2: Borrowed Native Bytes Escaping Into Go Values

**What goes wrong:** Returned `map[string]any` or `[]any` contains strings backed by native memory that can be freed by `Doc.Close`. [VERIFIED: docs/ffi-contract.md]  
**Why it happens:** `unsafe.String` or string-header tricks avoid copying but violate Phase 8's ownership rule. [VERIFIED: 08-CONTEXT.md]  
**How to avoid:** Convert borrowed byte spans with `string(unsafe.Slice(...))` at the final value boundary and call `runtime.KeepAlive(doc)` after the last borrowed read. [VERIFIED: go doc runtime.KeepAlive] [ASSUMED]  
**Warning signs:** A materialized string changes, panics, or fails after `doc.Close` or GC pressure tests. [VERIFIED: element_scalar_test.go] [VERIFIED: parser_test.go]

### Pitfall 3: Changing Duplicate-key Semantics Accidentally

**What goes wrong:** Full materialization and `Object.GetField` end up using the same duplicate behavior. [VERIFIED: 08-CONTEXT.md]  
**Why it happens:** Native object lookup returns first match, while Go map assignment returns last value for repeated keys. [VERIFIED: src/native/simdjson_bridge.cpp:630] [VERIFIED: benchmark_comparators_test.go:457]  
**How to avoid:** Fast materializer should iterate all object entries in order and assign `m[key] = value`; public `Object.GetField` should remain untouched. [VERIFIED: iterator_test.go:311] [ASSUMED]  
**Warning signs:** A duplicate-key materialization test returns first value or a `GetField` test starts returning last value. [VERIFIED: iterator_test.go:311]

### Pitfall 4: Benchmarking a Cached Tree Instead of Materialization

**What goes wrong:** Materialize-only diagnostics improve because the first iteration cached a Go tree, not because traversal/materialization got cheaper. [ASSUMED]  
**Why it happens:** It is tempting to store final Go `any` on `Doc` for repeated benchmark iterations. [ASSUMED]  
**How to avoid:** Rebuild final maps/slices/strings every materializer call; reuse only native/Rust scratch capacity when it does not skip traversal or final Go allocation. [VERIFIED: docs/benchmarks.md] [ASSUMED]  
**Warning signs:** Go allocation counts collapse unrealistically or materialize-only time approaches zero compared with full Tier 1. [ASSUMED]

### Pitfall 5: Using Raw Tape Counts as Exact Capacity

**What goes wrong:** Arrays/objects allocate too small/large or corrupt traversal on large containers. [VERIFIED: third_party/simdjson/doc/tape.md]  
**Why it happens:** simdjson tape start words carry a 24-bit immediate child count that can saturate. [VERIFIED: third_party/simdjson/doc/tape.md]  
**How to avoid:** Have the frame builder compute exact counts while walking, or mark saturated counts and let Go fall back to near-exact growth. [ASSUMED]  
**Warning signs:** Large-object benchmarks still allocate heavily or fail parity against current materializer. [ASSUMED]

### Pitfall 6: Windows Calling Convention Drift

**What goes wrong:** An internal helper works on darwin/linux but fails on windows/amd64. [VERIFIED: .planning/PROJECT.md]  
**Why it happens:** purego calls dynamic functions without cgo and signatures must stay simple across platforms. [VERIFIED: /Users/tazarov/go/pkg/mod/github.com/ebitengine/purego@v0.10.0/README.md]  
**How to avoid:** Use integer/pointer out-param signatures, avoid callbacks, avoid struct-by-value returns, and mirror layouts with explicit tests. [VERIFIED: docs/ffi-contract.md] [ASSUMED]  
**Warning signs:** Windows smoke tests fail to register or call only the new internal symbols. [ASSUMED]

## Code Examples

Verified patterns from existing sources and recommended Phase 8 adaptations.

### Current Hot Path to Replace for Tier 1

```go
// Source: benchmark_comparators_test.go
func benchmarkMaterializePureElement(el Element) (any, error) {
	switch typ, err := el.Type(); typ {
	case TypeArray:
		out := make([]any, 0, 8)
		// Iteration calls FFI per node today.
	case TypeObject:
		out := make(map[string]any, 8)
		// Key materialization calls ElementGetString today.
	}
}
```

The current implementation recursively calls public typed accessors and iterators, and it uses fixed capacity 8 for arrays/maps. [VERIFIED: benchmark_comparators_test.go:418]

### Recommended Internal Binding Shape

```go
// Source: synthesized from internal/ffi/bindings.go purego registration pattern.
type internalDomFrame struct {
	Kind       uint32
	Flags      uint32
	ChildCount uint32
	Reserved   uint32
	KeyPtr     uintptr
	KeyLen     uintptr
	StringPtr  uintptr
	StringLen  uintptr
	I64        int64
	U64        uint64
	F64        float64
}

func InternalDomFramesBuild(view ValueView) ([]internalDomFrame, error) {
	var ptr uintptr
	var n uintptr
	code := bindings.internalDomFramesBuild(view, &ptr, &n)
	if code != ErrorOK {
		return nil, codeToError(code)
	}
	return unsafe.Slice((*internalDomFrame)(unsafe.Pointer(ptr)), int(n)), nil
}
```

This should remain in `internal/ffi` and should not define a public API. [VERIFIED: internal/ffi/bindings.go] [VERIFIED: 08-CONTEXT.md]

### Duplicate-key Materialization Test Pattern

```go
// Source: synthesized from iterator_test.go duplicate-key coverage and D-11.
func TestFastMaterializerDuplicateKeyLastWins(t *testing.T) {
	parser := NewParser()
	doc, err := parser.Parse([]byte(`{"dup":1,"dup":2}`))
	if err != nil {
		t.Fatal(err)
	}
	defer doc.Close()

	got, err := materializeFast(doc, doc.Root())
	if err != nil {
		t.Fatal(err)
	}
	if got.(map[string]any)["dup"] != int64(2) {
		t.Fatalf("full materialization must use last duplicate")
	}

	first, err := doc.Root().AsObject().GetField("dup")
	_ = first
	_ = err
	// Existing GetField first-match assertion remains in iterator tests.
}
```

The final test should use the repo's exact helper style and avoid changing public `GetField` tests. [VERIFIED: iterator_test.go:311] [ASSUMED]

## State of the Art

| Old Approach | Current Approach for Phase 8 | When Changed | Impact |
|--------------|------------------------------|--------------|--------|
| Recursive public accessor materialization | Internal frame stream plus specialized Go builder | Phase 8 planning, 2026-04-23 | Removes avoidable per-node FFI and key/string copy-out cost from Tier 1 materialization. [VERIFIED: 08-CONTEXT.md] [ASSUMED] |
| Fixed map/slice capacity 8 | Use frame child-count metadata for preallocation | Phase 8 planning, 2026-04-23 | Reduces Go allocation growth for large arrays/objects. [VERIFIED: benchmark_comparators_test.go:442] [ASSUMED] |
| Public string copy-out through native allocation/free | Internal borrowed spans copied once into Go strings | Phase 8 planning, 2026-04-23 | Preserves public ownership while avoiding native bytes allocation per internal string read. [VERIFIED: src/runtime/registry.rs:752] [ASSUMED] |
| Public ABI only | Public ABI plus internal Go-bound cdylib symbols excluded from header | Phase 8 planning, 2026-04-23 | Enables benchmark-facing fast path without promising new C ABI. [VERIFIED: 08-CONTEXT.md] [ASSUMED] |

**Deprecated/outdated for Phase 8 planning:**

- Treating Tier 1 as primarily parse-bound is outdated for this repo; Phase 7 diagnostics show materialization dominates parse-only time. [VERIFIED: 07-LEARNINGS.md] [VERIFIED: testdata/benchmark-results/v0.1.1/tier1-diagnostics.bench.txt]
- Using public DOM accessors as the Tier 1 materialization benchmark path is correct for v0.1.1 evidence but is the specific bottleneck Phase 8 is replacing. [VERIFIED: docs/benchmarks/results-v0.1.1.md] [VERIFIED: 08-CONTEXT.md]

## Exact Code Integration Points

| File | Current Role | Phase 8 Integration |
|------|--------------|---------------------|
| `src/lib.rs` | Public FFI exports, error codes, `ffi_wrap`, generated ABI version. [VERIFIED: src/lib.rs:13] [VERIFIED: src/lib.rs:177] | Add internal `psdj_internal_*` exports using `ffi_wrap`; avoid public `pure_simdjson_*` additions unless contract intentionally changes. [ASSUMED] |
| `src/runtime/registry.rs` | Parser/doc registry, copy-in padded input, generation validation, public accessor/iterator helpers. [VERIFIED: src/runtime/registry.rs:34] [VERIFIED: src/runtime/registry.rs:432] | Add one root/subtree validation path and doc-owned traversal frame scratch or per-call frame allocation ownership. [ASSUMED] |
| `src/runtime/mod.rs` | Rust declarations and wrappers for private C++ bridge symbols. [VERIFIED: src/runtime/mod.rs:44] | Add declarations for `psimdjson_*` traversal builder hooks and convert native errors to registry errors. [ASSUMED] |
| `src/native/simdjson_bridge.h` | Private C++ bridge ABI for Rust. [VERIFIED: src/native/simdjson_bridge.h:14] | Define private frame structs/functions; keep C-compatible layout and no throwing API. [ASSUMED] |
| `src/native/simdjson_bridge.cpp` | simdjson parse/accessor/iterator implementation and tape-ref helpers. [VERIFIED: src/native/simdjson_bridge.cpp:316] [VERIFIED: src/native/simdjson_bridge.cpp:569] | Walk element subtree once, fill frame sequence, attach key/string borrowed spans and numeric payloads. [ASSUMED] |
| `internal/ffi/types.go` | Go mirrors of public ABI structs and error constants. [VERIFIED: internal/ffi/types.go:11] | Add internal-only frame/view mirror types; verify `unsafe.Sizeof`/field offsets if tests are added. [ASSUMED] |
| `internal/ffi/bindings.go` | purego symbol registration, status wrappers, `runtime.KeepAlive` discipline. [VERIFIED: internal/ffi/bindings.go:53] [VERIFIED: internal/ffi/bindings.go:170] | Register internal symbols and expose unexported frame build wrapper with existing error mapping. [ASSUMED] |
| `element.go` | Public DOM element/accessor behavior. [VERIFIED: element.go:80] | Keep public API stable; optionally add unexported fast materializer entry that takes `Element`. [ASSUMED] |
| `iterator.go` | Public array/object iterators and key copy-out. [VERIFIED: iterator.go:33] [VERIFIED: iterator.go:82] | Leave public iterators unchanged; use tests to prove no regression. [ASSUMED] |
| `parser.go` / `doc.go` | Parser busy semantics, doc root, close/finalizer behavior. [VERIFIED: parser.go:62] [VERIFIED: doc.go:26] | Fast materializer must reject closed docs and keep doc alive through borrowed frame reads. [ASSUMED] |
| `benchmark_comparators_test.go` | Current Tier 1 pure materializer and comparator registry. [VERIFIED: benchmark_comparators_test.go:36] [VERIFIED: benchmark_comparators_test.go:368] | Route pure simdjson Tier 1 comparator to fast materializer or add an internal comparator variant for before/after proof. [ASSUMED] |
| `benchmark_diagnostics_test.go` | Full/parse-only/materialize-only diagnostics. [VERIFIED: benchmark_diagnostics_test.go:98] | Use the same diagnostic shape to prove materialize-only and full Tier 1 delta against Phase 7. [VERIFIED: docs/benchmarks.md] |
| `tests/abi/check_header.py` | Public header required/forbidden symbol audit. [VERIFIED: tests/abi/check_header.py:46] [VERIFIED: tests/abi/check_header.py:168] | Add guard that Phase 8 internal symbols remain absent from the generated public header. [ASSUMED] |

## Recommended Plan Breakdown

| Wave | Dependency | Tasks | Exit Criteria |
|------|------------|-------|---------------|
| Wave 0: Guardrails | None | Add parity/numeric/duplicate/lifetime/header tests as failing or skipped scaffolds where needed. [ASSUMED] | Tests define expected behavior before fast path is wired. [ASSUMED] |
| Wave 1: Native frame builder | Wave 0 | Add private C++ traversal frames and Rust registry validation/export; support root document first, with function shape capable of subtree input. [ASSUMED] | Go can call internal build and inspect frame count/kinds on small fixtures. [ASSUMED] |
| Wave 2: Go frame materializer | Wave 1 | Implement unexported Go builder preserving numeric types, string copy ownership, duplicate last-wins, and preallocation. [ASSUMED] | Parity tests pass against current accessor materializer for representative docs. [ASSUMED] |
| Wave 3: Subtree and lifecycle hardening | Wave 2 | Wire subtree materialization if not done in Wave 1, add closed-doc/GC/lifetime tests, and keep public accessors unchanged. [ASSUMED] | `go test ./...` and targeted lifecycle tests pass. [ASSUMED] |
| Wave 4: Benchmark integration | Wave 2 or 3 | Switch Tier 1 benchmark path or add a measured fast comparator; capture diagnostics and benchstat vs v0.1.1 artifacts. [ASSUMED] | Materialize-only and full Tier 1 improve over Phase 7 baseline. [VERIFIED: 08-CONTEXT.md] |
| Wave 5: Contract/documentation closeout | Wave 4 | Run `make verify-contract`, update internal benchmark notes only, and leave public README/result repositioning for Phase 9. [VERIFIED: Makefile:6] | Header unchanged except intentional generated stability, no public fast API added, internal docs created. [VERIFIED: 08-CONTEXT.md] |

**Dependency rule:** do not switch the existing Tier 1 comparator until parity tests prove the fast materializer matches the accessor-shaped materializer for supported values. [ASSUMED]

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | A repo-owned frame stream will produce a measurable materialize-only and full Tier 1 improvement after removing per-node FFI/key-string handoff. | Summary, Design Comparison, State of the Art | Phase 8 may need another optimization pass or a different frame ownership strategy. |
| A2 | Internal `psdj_internal_*` dynamic symbols can be exported for Go purego use while remaining out of the public generated header. | Architecture Patterns, Integration Points | cbindgen/export tooling may need additional configuration or a different symbol naming scheme. |
| A3 | Doc-owned scratch rebuilt per materializer call is a fair benchmark compromise if it does not cache final Go values. | Common Pitfalls, Plan Breakdown | Native allocation metrics or repeated materialize-only runs may look better than real first-call behavior. |
| A4 | Whole-document first implementation is acceptable if the ABI accepts a `ValueView` and subtree support is added in a follow-up wave. | Recommended Plan Breakdown | Planner may need to split root and subtree tasks explicitly to satisfy D-04. |

## Open Questions

1. **Should the first implementation slice include subtree materialization?**  
   What we know: D-04 wants whole-doc and subtree in the intended design envelope. [VERIFIED: 08-CONTEXT.md]  
   What's unclear: Whether adding subtree support during the first native builder task will slow the critical Tier 1 benchmark path. [ASSUMED]  
   Recommendation: Design the internal ABI around `ValueView`, implement whole-doc first only if the frame span/root validation makes subtree a small follow-up task. [ASSUMED]

2. **Should frame buffers be doc-owned scratch or explicitly allocated/freeable views?**  
   What we know: D-09 allows Doc/materializer ownership and existing `Close`/finalizer discipline. [VERIFIED: 08-CONTEXT.md]  
   What's unclear: Which option gives clearer benchmark fairness and simpler panic-safe cleanup. [ASSUMED]  
   Recommendation: Prefer doc-owned scratch rebuilt each call unless implementation reveals reentrancy or benchmark artifact issues; do not cache final Go values. [ASSUMED]

3. **What numeric threshold defines enough improvement?**  
   What we know: D-16 requires improvement over Phase 7 in materialize-only and full Tier 1, not beating `encoding/json + any`. [VERIFIED: 08-CONTEXT.md]  
   What's unclear: No minimum percentage is locked. [VERIFIED: 08-CONTEXT.md]  
   Recommendation: Use benchstat significance and report raw ns/op, B/op, allocs/op for the same fixtures; planner should not invent a public benchmark claim. [VERIFIED: docs/benchmarks.md] [ASSUMED]

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|-------------|-----------|---------|----------|
| Go | Go tests, benchmarks, purego binding | yes | go1.26.2 local; module requires Go 1.24 | none needed. [VERIFIED: `go version`] [VERIFIED: go.mod] |
| Rust/cargo | Native cdylib and tests | yes | 1.89.0 | none needed. [VERIFIED: `rustc --version`] [VERIFIED: `cargo --version`] |
| C++ compiler | simdjson bridge build | yes | Apple clang 21.0.0 local | CI target compilers remain required for non-darwin targets. [VERIFIED: `cc --version`] [VERIFIED: .planning/PROJECT.md] |
| cbindgen | Header generation/contract verification | yes | 0.29.2 | none needed. [VERIFIED: `cbindgen --version`] |
| benchstat | Benchmark deltas | yes | x/perf pseudo-version `v0.0.0-20260209182753-b57e4e371b65` | install with `go install golang.org/x/perf/cmd/benchstat@latest` if missing. [VERIFIED: `go version -m /Users/tazarov/go/bin/benchstat`] [VERIFIED: scripts/bench/run_benchstat.sh:66] |
| Python 3 | ABI header audit | yes | 3.11.7 | none needed. [VERIFIED: `python3 --version`] [VERIFIED: tests/abi/check_header.py] |

**Missing dependencies with no fallback:** none found. [VERIFIED: environment audit commands]

**Missing dependencies with fallback:** none found. [VERIFIED: environment audit commands]

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go test/bench with go1.26.2 local; Cargo test with Rust 1.89.0; Python header audit. [VERIFIED: `go version`] [VERIFIED: `rustc --version`] [VERIFIED: tests/abi/check_header.py] |
| Config file | `go.mod`, `Cargo.toml`, `Makefile`, `cbindgen.toml`. [VERIFIED: go.mod] [VERIFIED: Cargo.toml] [VERIFIED: Makefile] |
| Quick run command | `go test ./...` [VERIFIED: existing Go package tests] |
| Contract command | `make verify-contract` [VERIFIED: Makefile:6] |
| Benchmark delta command | `scripts/bench/run_benchstat.sh --old testdata/benchmark-results/v0.1.1/tier1-diagnostics.bench.txt --new <phase8-diagnostics-output>` [VERIFIED: scripts/bench/run_benchstat.sh:1] |
| Full suite command | `go test ./... && cargo test -- --test-threads=1 && make verify-contract` [VERIFIED: Makefile:6] [VERIFIED: Cargo.toml] |

### Phase Requirements -> Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|--------------|
| D-01/D-06/D-07 | Fast materializer avoids public per-node key/string accessor path and still returns correct values. | integration + benchmark | `go test ./... -run TestFastMaterializer` | no, Wave 0. [ASSUMED] |
| D-02/D-03/D-05/D-14 | Internal symbols do not enter public header and existing public ABI remains unchanged. | contract | `make verify-contract` | yes for existing guard; update likely needed. [VERIFIED: Makefile:6] [ASSUMED] |
| D-04 | Root and subtree materialization produce correct equivalent trees. | integration | `go test ./... -run TestFastMaterializerSubtree` | no, Wave 0. [ASSUMED] |
| D-08/D-09 | Materialized strings survive `Doc.Close`, GC pressure, and parser finalizer paths. | unit/integration | `go test ./... -run 'TestFastMaterializer.*(Lifetime|Close|GC)'` | no, Wave 0. [ASSUMED] |
| D-10/D-13 | Numeric int64/uint64/float64 preservation and range/precision errors match existing accessors. | unit | `go test ./... -run 'TestFastMaterializer.*Numeric'` | no, Wave 0. [ASSUMED] |
| D-11 | Duplicate-key full map materialization last wins while `Object.GetField` first match remains. | unit | `go test ./... -run 'TestFastMaterializerDuplicate|TestObjectGetFieldDuplicate'` | partial existing for `GetField`; fast test missing. [VERIFIED: iterator_test.go:311] [ASSUMED] |
| D-12 | Container preallocation uses frame metadata and allocation counts improve or do not regress. | benchmark | `go test ./... -run '^$' -bench 'BenchmarkTier1Diagnostics_' -benchmem -count=5` | existing diagnostics yes; allocation assertion no. [VERIFIED: benchmark_diagnostics_test.go:98] |
| D-15/D-16 | Phase 8 materialize-only and full Tier 1 improve over Phase 7 baseline. | benchmark + benchstat | `scripts/bench/run_benchstat.sh --old testdata/benchmark-results/v0.1.1/tier1-diagnostics.bench.txt --new <new>` | comparison script yes; new artifact no. [VERIFIED: scripts/bench/run_benchstat.sh:1] |
| D-17 | Only internal docs/benchmark notes updated. | review/contract | `git diff -- docs README.md .planning/phases/08-*` | manual review required. [VERIFIED: 08-CONTEXT.md] |

### Sampling Rate

- **Per task commit:** `go test ./...` for Go-facing changes; `cargo test -- --test-threads=1` for Rust/C++ registry changes. [VERIFIED: Cargo.toml] [ASSUMED]
- **Per wave merge:** `go test ./... && cargo test -- --test-threads=1 && make verify-contract`. [VERIFIED: Makefile:6] [ASSUMED]
- **Phase gate:** full suite plus Tier 1 diagnostics captured with `-benchmem -count=5` and benchstat against Phase 7 raw artifacts. [VERIFIED: docs/benchmarks.md] [VERIFIED: testdata/benchmark-results/v0.1.1/tier1-diagnostics.bench.txt]

### Wave 0 Gaps

- [ ] `materializer_fastpath_test.go` - parity, duplicate-key, numeric preservation, string ownership, closed-doc behavior. [ASSUMED]
- [ ] `internal/ffi` layout test - verifies Go frame struct size/offsets against native constants if exported. [ASSUMED]
- [ ] `tests/abi/check_header.py` extension - asserts internal Phase 8 symbols are absent from public header. [ASSUMED]
- [ ] Benchmark artifact naming - choose internal Phase 8 raw output path before closeout. [ASSUMED]

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|------------------|
| V2 Authentication | no | Phase 8 does not add identity/authentication behavior. [VERIFIED: 08-CONTEXT.md] |
| V3 Session Management | no | Phase 8 does not add sessions or cookies. [VERIFIED: 08-CONTEXT.md] |
| V4 Access Control | no | Phase 8 is an in-process parser/materializer change. [VERIFIED: 08-CONTEXT.md] |
| V5 Input Validation | yes | Keep simdjson parse validation and existing typed error mapping; do not add custom JSON parsing. [VERIFIED: benchmark_oracle_test.go] [VERIFIED: internal/ffi/bindings.go:115] |
| V6 Cryptography | no | Phase 8 does not add cryptography. [VERIFIED: 08-CONTEXT.md] |

### Known Threat Patterns for Go/Rust/C++ FFI Traversal

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Use-after-free of borrowed native bytes | Tampering/Denial of Service | Borrow only during materializer call, copy escaped strings into Go memory, call `runtime.KeepAlive(doc)` after final read. [VERIFIED: go doc runtime.KeepAlive] [ASSUMED] |
| Public ABI confusion through accidental header export | Information Disclosure/Tampering | Keep internal symbols out of `include/pure_simdjson.h`; enforce with cbindgen diff and header audit. [VERIFIED: tests/abi/check_header.py:168] [ASSUMED] |
| Numeric precision loss | Tampering | Preserve split int64/uint64/float64 frame tags and reuse existing precision/range error semantics. [VERIFIED: element.go:168] [ASSUMED] |
| Invalid JSON/UTF-8 acceptance | Tampering | Keep simdjson DOM parse as the only source of materializable values and retain oracle tests. [VERIFIED: benchmark_oracle_test.go] [VERIFIED: element_fuzz_test.go] |
| Cross-platform FFI call mismatch | Denial of Service | Use pointer/integer out-param signatures, no callbacks, no struct-by-value returns, and CI for five first-class targets. [VERIFIED: docs/ffi-contract.md] [VERIFIED: .planning/PROJECT.md] |

## Sources

### Primary (HIGH confidence)

- `.planning/phases/08-low-overhead-dom-traversal-abi-and-specialized-go-any-materi/08-CONTEXT.md` - Phase 8 decisions D-01 through D-17, discretion, deferred scope. [VERIFIED]
- `.planning/ROADMAP.md` - Phase 8 and Phase 9 boundaries. [VERIFIED]
- `.planning/PROJECT.md` - stack and project constraints. [VERIFIED]
- `.planning/REQUIREMENTS.md` - completed public ABI/API/benchmark requirements and Phase 8 TBD requirement status. [VERIFIED]
- `.planning/STATE.md` - current focus and Phase 7 handoff. [VERIFIED]
- `docs/ffi-contract.md` and `include/pure_simdjson.h` - normative public ABI behavior. [VERIFIED]
- `src/lib.rs`, `src/runtime/registry.rs`, `src/runtime/mod.rs`, `src/native/simdjson_bridge.cpp`, `src/native/simdjson_bridge.h` - current Rust/C++ implementation seams. [VERIFIED]
- `internal/ffi/bindings.go`, `internal/ffi/types.go`, `element.go`, `iterator.go`, `parser.go`, `doc.go` - current Go wrapper and purego binding seams. [VERIFIED]
- `benchmark_comparators_test.go`, `benchmark_diagnostics_test.go`, `docs/benchmarks.md`, `docs/benchmarks/results-v0.1.1.md`, `testdata/benchmark-results/v0.1.1/*.bench.txt` - current benchmark path and baseline. [VERIFIED]
- `third_party/simdjson/doc/tape.md`, `third_party/simdjson/doc/performance.md`, `third_party/simdjson` tag v4.6.1 - upstream vendored tape/performance context. [VERIFIED]

### Secondary (MEDIUM confidence)

- Local module cache for `github.com/minio/simdjson-go@v0.4.5` - reference Go tape and `Interface` implementation. [VERIFIED]
- Local module cache for `github.com/ebitengine/purego@v0.10.0` README - purego platform and dynamic FFI context. [VERIFIED]
- `go doc runtime.KeepAlive` - lifetime/finalizer guidance. [VERIFIED]
- `cargo info cc`, `cargo info cbindgen`, `go list -m -json`, `go version -m benchstat` - version provenance. [VERIFIED]

### Tertiary (LOW confidence)

- None. All low-confidence claims are captured as `[ASSUMED]` in the Assumptions Log. [VERIFIED: this document]

## Metadata

**Confidence breakdown:**

- Standard stack: HIGH - versions were verified from `go list -m`, `git describe`, `cargo info`, local tool commands, and existing manifests. [VERIFIED]
- Architecture: MEDIUM - integration points are verified, but frame-stream performance and exact ownership shape require implementation and benchmark proof. [VERIFIED] [ASSUMED]
- Pitfalls: HIGH for ABI/lifetime/numeric/header risks from existing docs and code; MEDIUM for benchmark-cache/fairness predictions until Phase 8 artifacts exist. [VERIFIED] [ASSUMED]

**Research date:** 2026-04-23  
**Valid until:** 2026-05-23 for codebase-local integration research; re-verify versions and purego/cbindgen behavior before implementation if delayed beyond 30 days. [ASSUMED]
