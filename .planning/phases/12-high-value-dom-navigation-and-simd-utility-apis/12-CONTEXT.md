# Phase 12: High-value DOM navigation and SIMD utility APIs - Context

**Gathered:** 2026-07-30
**Updated:** 2026-07-30 — Spike 004 findings folded in (D-13/D-14); Minify aliasing decision confirmed, no amendment needed
**Updated:** 2026-07-30 — D-15 added at plan-phase time, resolving RESEARCH.md Open Question 1 (standalone entry points gate on CPU support and lock kernel selection)
**Status:** Ready for planning

<domain>
## Phase Boundary

Expose the mature, high-value parts of simdjson's DOM and implementation APIs as thin Go wrappers over the existing `Element`/`Array`/`Object` types: RFC 6901 JSON Pointer navigation (`AtPointer`), the documented simdjson dot/index path subset (`AtPath`), ordered wildcard path selection (`AtPathAll`), indexed array access plus constant-time array/object size helpers, an allocation-conscious `Minify`, and a standalone `ValidateUTF8`.

Requirements: DOM-01, DOM-02, DOM-03, DOM-04, UTIL-01, UTIL-02.

No JSON encoder/builder, reflection-based `Unmarshal`, full JSONPath (RFC 9535) engine, file-loading wrapper, or mutable DOM API is added. Zero-copy/borrowed views remain Phase 14 scope — the DOM navigation methods here return normal copied `Element` views, same as existing accessors.

</domain>

<decisions>
## Implementation Decisions

### Navigation error taxonomy (DOM-01/02/03)
- **D-01:** Add two new typed error sentinels: `ErrInvalidPath` (malformed JSON Pointer / path syntax — upstream `INVALID_JSON_POINTER`) and `ErrIndexOutOfRange` (a syntactically valid index that exceeds array bounds — upstream `INDEX_OUT_OF_BOUNDS`). Reuse the existing `ErrElementNotFound` for missing object keys / missing pointer segments (upstream `NO_SUCH_FIELD`), and the existing `ErrWrongType` for traversal type mismatches (upstream `INCORRECT_TYPE`). Do not invent a third or fourth sentinel, and do not conflate invalid-path with out-of-bounds into one error — they must be distinguishable via `errors.Is`.
- **D-02 (AMENDED 2026-07-31 — original contract refuted by spike 005):** `Element.AtPathAll` **requires at least one `*` in the path**; a wildcard-free path returns `ErrInvalidPath` before reaching the FFI boundary. It returns `([]Element{}, nil)` when the wildcard legitimately matches zero elements, matching upstream's `at_path_with_wildcard` behavior and Go idiom (map lookups, `filepath.Glob`). Missing keys, out-of-range indices, and non-container branches yield **no match rather than an error**. **The only path error is `ErrInvalidPath`** — `ErrWrongType`, `ErrElementNotFound`, and `ErrIndexOutOfRange` are NOT reachable through `AtPathAll` and must not be asserted in its tests.
  - *Why the original was wrong:* upstream selects its error regime by substring-testing the path for `*`, not by document content. Accepting wildcard-free paths therefore makes `.z.b` return `NO_SUCH_FIELD` while `.z.*` silently returns an empty set on the same document. Requiring a wildcard removes the split; `AtPath` already covers wildcard-free paths with full error reporting, so nothing is lost.
  - *Evidence:* `.planning/spikes/005-wildcard-path-semantics/` — 35-case pinned truth table, verdict PARTIAL.
- **D-03:** `ErrIndexOutOfRange` is shared between `AtPointer`/`AtPath` (pointer/path segments that resolve to an out-of-range array index) and the new indexed `Array.At` (see below) — same upstream failure, same sentinel, no duplication.

### Indexed array access and size helpers (DOM-04)
- **D-04:** `Array.At(index int) (Element, error)` — no panic-safe/dual-method twin (no bare `Array.At(index) Element`). This mirrors the existing `Object.GetField` precedent, which also has no panic-safe twin: a zero-value `Element` returned on failure would silently launder the real out-of-range error behind a second, unrelated `ErrInvalidHandle` the next time it's touched.
- **D-05:** `index` is a Go `int`, not `uint64` — matches the existing codebase convention of casting native unsigned counts to `int` at Go boundaries (e.g. `frame.ChildCount` in the materializer). Negative indices are NOT supported (no Python-style `-1` = last element); upstream has no such concept and this project mirrors upstream's unsigned-index semantics faithfully elsewhere (e.g. `GetUint64` rejects negative reinterpretation).
- **D-06:** `Array.Len() int` / `Array.LenErr() (int, error)` and equivalent `Object` size helper follow the existing dual-method convention already used by `Type()/TypeErr()` and `IsNull()/IsNullErr()` — `Len()` is panic-safe (returns `0` on failure), `LenErr()` preserves the real error. This fits because, unlike `At`, `Len` returns a simple scalar with an unambiguous, safe zero-value fallback.
- **D-07:** Out-of-range `Array.At` returns `ErrIndexOutOfRange` (not `ErrElementNotFound`) — see D-01/D-03.

### Minify (UTIL-01)
- **D-08:** Ship a dual API: `Minify(data []byte) ([]byte, error)` — the simple allocating convenience for the common case, matching the project's established copy-by-default precedent (`Parser.Parse`, `Element.GetString`) — plus `MinifyInto(dst, src []byte) (int, error)` — the allocation-conscious, buffer-reuse variant where `dst` may alias `src` (in-place minification), mirroring upstream's `minify(buf, len, dst, dst_len)` contract directly.
- **D-09:** `MinifyInto` is the variant that must be exercised by the required "overlap" tests (success criterion 4) — `dst == src` aliasing must be tested as a first-class supported case, not just an internal FFI-level proof. `dst` must be sized `>= len(src)` (minified output is never longer than input); undersized `dst` returns an error rather than silently truncating.
- **D-10:** Minify's buffer handling is NOT gated by Phase 14's zero-copy benchmark-gating rule. That rule exists for document-lifetime-tied borrowed views with invalidation hazards; `Minify`/`MinifyInto` operate on plain, always-caller-owned byte slices with no Doc/Parser lifetime coupling, so it is in-scope for Phase 12 as stated in UTIL-01, not deferred.

### Validated Spike Resolution — Minify Aliasing Safety
- **D-13 (post-discussion, spike-validated 2026-07-30):** D-08/D-09's `dst == src` aliasing contract is confirmed memory-safe and correct, not just theoretically sound. `.planning/spikes/004-minify-buffer-safety/` built a standalone ASan+UBSan probe against the untouched vendored simdjson v4.6.4 singleheader (matching production's `SIMDJSON_IMPLEMENTATION_FALLBACK=1` build.rs flag) and found: (a) upstream's own doc comments contradict each other on whether `dst` needs `len + SIMDJSON_PADDING` slack or just `len` bytes, and upstream's own multi-year fuzz harness never tests the `dst == src` case at all; (b) empirically, across the `arm64` and `fallback` kernels, `dst` sized exactly `len(src)` (no padding) triggered zero ASan/UBSan traps, and in-place (`dst == src`) output was byte-identical to the non-aliased reference across 12 fixtures including two adversarial high-compression-ratio inputs (~18KB, ~25KB), with identical results across 3 repeated runs. **No CONTEXT.md amendment to D-08/D-09 was needed — the locked decision stands as originally written.**
- **D-14 (residual, non-blocking):** The spike ran on an Apple Silicon (arm64) host and could not exercise the `haswell`/`westmere`/`icelake` x86-64 kernels. Before Phase 12 claims the aliasing guarantee across all five shipped platforms, re-run the same probe (or an equivalent contract test) on an x86-64 CI runner — fold this into the existing Rust/C++ contract test suite or five-platform smoke matrix rather than treating it as a one-off. This is a plan-phase task, not a blocker to starting the plan.

### Standalone entry-point CPU policy (UTIL-01/UTIL-02)
- **D-15 (post-research, 2026-07-30):** `Minify`, `MinifyInto`, and `ValidateUTF8` enforce the same CPU-unsupported rejection that `NewParser`/`NewParserPool` already apply — they call the `reject_fallback_implementation()`-equivalent check and return `ErrCPUUnsupported` on unsupported CPUs rather than silently running the `fallback` kernel. They also call `lockKernelSelection()` on first successful native call, mirroring `NewParser`, so a `SetKernel` call after any of these three returns `ErrKernelLocked`. This resolves RESEARCH.md Open Question 1: the project's stated "unsupported CPUs fail loudly, not silently" philosophy wins over "utilities are best-effort, just slower". The behavior change (`SetKernel` newly rejected after a `Minify`/`ValidateUTF8` call) must be stated explicitly in the doc comments for all three functions.

### ValidateUTF8 (UTIL-02)
- **D-11:** `func ValidateUTF8(data []byte) (bool, error)` — not a bare bool, and not a `Type()/TypeErr()`-style dual method. The error return exists because, unlike upstream's pure C++ function, the Go entry point can genuinely fail on first call (native library not yet loaded/bootstrapped, ABI mismatch) — every other function that touches `activeLibrary()` (`NewParser`, `SetKernel`) already returns `error` for exactly this reason, and `ValidateUTF8` is the first standalone entry point that can trigger first-time library resolution without a prior `NewParser` call.
- **D-12:** No dual-method (`ValidateUTF8()`/`ValidateUTF8Err()`) — that convention exists for transient Doc/Parser handle validity, which `ValidateUTF8` has none of; its only failure mode (library load) is not a "safe to hide" condition, it's a real error every caller needs to see.

### Claude's Discretion
- Exact FFI status-code numeric values for `ErrInvalidPath` and `ErrIndexOutOfRange`, provided they are additive to the existing status-code block and follow the established 1:1 native-code-to-Go-sentinel convention.
- Internal decomposition of the dot/index path parser for `AtPath` (whether parsing happens in Go before the FFI call, or is delegated entirely to upstream's `at_path`), provided the documented subset and error behavior above are preserved.
- Whether `Object` gets a `Size()/SizeErr()` pair with the same name as `Array.Len()/LenErr()` or a distinct name — both are constant-time size accessors per DOM-04; naming should stay internally consistent between the two types.
- Internal file organization and test decomposition across Go, Rust, C++, and FFI layers for the new exports.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope and prior decisions
- `.planning/ROADMAP.md` — Phase 12 goal, requirements (DOM-01..04, UTIL-01/02), success criteria, and explicit scope boundary (no encoder, no full JSONPath, no mutable DOM).
- `.planning/REQUIREMENTS.md` — v2 "High-value DOM navigation and SIMD utilities" section is the complete Phase 12 requirement set.
- `.planning/PROJECT.md` — copied-DOM-by-default philosophy, precision-preservation goal, five-platform requirement, no-cgo constraint, FFI safety rules (`ffi_wrap`/`catch_unwind` on every export).
- `.planning/phases/11-upstream-simdjson-refresh-bigint-and-diagnostics/11-CONTEXT.md` — current ABI (0x00010002), truthful-diagnostics precedent that this phase's error-taxonomy decisions extend, and the established pattern of additive-only ABI growth.
- `.planning/phases/08-low-overhead-dom-traversal-abi-and-specialized-go-any-materi/08-CONTEXT.md` — existing frame/materializer ABI that indexed-array-access work must not destabilize.

### Validated spike evidence
- `.planning/spikes/MANIFEST.md` — Phase 12 spike contract and validated verdict.
- `.planning/spikes/004-minify-buffer-safety/README.md` — ASan/UBSan evidence behind D-13 that `MinifyInto`'s `dst == src` aliasing is memory-safe and correct on the `arm64`/`fallback` kernels, plus the D-14 residual x86-64 verification follow-up.
- `.planning/spikes/004-minify-buffer-safety/probe.cpp`, `verify.sh` — reusable probe; re-run on x86-64 CI per D-14 before claiming the aliasing guarantee across all five platforms.
- `.planning/spikes/CONVENTIONS.md` — isolation, exact-pin, machine-readable evidence, repeatability, and promotion rules these probes follow.

### Public Go and C contracts
- `element.go` — current `Element`/`Array`/`Object` types, existing accessor methods, and the `Type()/TypeErr()`, `IsNull()/IsNullErr()` dual-method convention that `Len()/LenErr()` extends (D-06).
- `errors.go` — current typed error sentinel list and native-status-to-Go-error mapping; add `ErrInvalidPath` and `ErrIndexOutOfRange` here per D-01.
- `doc.go` — existing `Doc.Root() Element` entry point; `AtPointer`/`AtPath`/`AtPathAll` attach to `Element` (root or any descendant), consistent with this existing pattern.
- `internal/ffi/types.go` — Go ABI constants and error-code numbers; add the two new status codes as an additive block per D-01/D-10 (see Phase 11's additive-ABI-growth precedent).
- `internal/ffi/bindings.go` — purego symbol binding table to extend with the new exports (pointer/path navigation, wildcard, indexed access, size, minify, validate-utf8).
- `include/pure_simdjson.h` — generated public ABI header; new exports must appear here via `cbindgen` and pass the existing diff-check.
- `docs/ffi-contract.md` — normative FFI ownership/error/versioning rules the new exports must follow.

### Native implementation and upstream reference
- `src/lib.rs`, `src/native/simdjson_bridge.h`, `src/native/simdjson_bridge.cpp` — Rust/C++ bridge surface to extend with `at_pointer`, `at_path`, `at_path_with_wildcard`, indexed `array::at`/`size()`, `object::size()`, `minify`, and standalone `validate_utf8` calls into vendored upstream simdjson.
- `third_party/simdjson/singleheader/simdjson.h` — vendored upstream signatures confirmed during this discussion: `element::at_pointer(string_view) -> simdjson_result<element>` (~L5357/L6986/L7136/L7340/L7448), `element::at_path(string_view)` and `at_path_with_wildcard(string_view) -> simdjson_result<vector<element>>` (~L5368-5383, L6988, L7137-7139, L7350-7365, L7450), `array::at(size_t) -> simdjson_result<element>` (~L5401, L7036), `array::size()`/`object::size() -> size_t` (~L5329, L5447, L7281, L7459), `simdjson::minify(const char*, size_t, char*, size_t&) -> error_code` (~L4453), `simdjson::validate_utf8(const char*, size_t) noexcept -> bool` (~L3852).
- `third_party/simdjson/include/simdjson/dom/element-inl.h`, `array-inl.h`, `object-inl.h` — upstream error-code taxonomy behind D-01/D-02 (`NO_SUCH_FIELD`, `INVALID_JSON_POINTER`, `INDEX_OUT_OF_BOUNDS`, `INCORRECT_TYPE`; wildcard empty-match-is-not-an-error behavior).
- `third_party/simdjson/fuzz/fuzz_minifyimpl.cpp`, `third_party/simdjson/tools/minify.cpp` — upstream evidence that `dst == src` in-place minify aliasing is safe and fuzzed, informing D-08/D-10.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `Element.usableDoc()` (element.go) — existing closed/invalid-handle guard reused by every accessor; new navigation/indexed methods should call it the same way before touching native state.
- `wrapStatus(rc)` (errors.go) — existing native-status-to-Go-error mapping; extend its switch/table with the two new status codes rather than inventing a parallel error path.
- `Array`/`Object` wrapper types (element.go) already exist (`Array{element Element}`, `Object{element Element}`) with unexported fields preventing unverified construction — `At`/`Len`/`Size` attach directly to these.
- `activeLibrary()` / `activeLibraryWithOps()` (library_loading.go) — existing library-resolution path already used by `NewParser`/`SetKernel`; `ValidateUTF8`/`Minify`/`MinifyInto` are the first standalone (no-Doc/Parser) callers and should reuse this same resolution path rather than requiring a `Parser` to exist first.

### Established Patterns
- Dual-method convention: a panic-safe zero-value getter (`Type()`, `IsNull()`) paired with an Err-suffixed variant (`TypeErr()`, `IsNullErr()`) — used for simple-scalar-returning accessors with an unambiguous zero value. NOT used for `Element`-returning accessors (`GetField` has no twin) — extended to `Len()/LenErr()` (D-06) but explicitly NOT to `At` (D-04).
- One sentinel per distinct native status code — `ErrNumberOutOfRange`, `ErrPrecisionLoss`, and `ErrWrongType` are already kept separate rather than merged; D-01's two new sentinels follow this same granularity.
- Copied-by-default: `Parser.Parse` copies input into a Rust-owned arena, `GetString`/`GetBigInt` return copied Go strings. `Minify`'s convenience path follows this; `MinifyInto` is an explicit opt-in for buffer reuse, not a silent default.
- Every `extern "C"` export returns an int32 error code with out-parameters, wrapped in `ffi_wrap`/`catch_unwind`, and reflected in the generated header — new exports for this phase follow the same shape.

### Integration Points
- `element.go` gets the new navigation methods (`AtPointer`, `AtPath`, `AtPathAll`) on `Element`, and indexed/size methods on `Array`/`Object`.
- `errors.go` gets the two new sentinels and their native-status mapping.
- A new top-level file (e.g. `minify.go`, `utf8.go` — naming left to Claude's discretion) hosts the standalone `Minify`/`MinifyInto`/`ValidateUTF8` functions, since these are the first public functions with no `Doc`/`Element`/`Parser` receiver.
- Native side: `src/native/simdjson_bridge.cpp` gains new bridge functions calling upstream `at_pointer`/`at_path`/`at_path_with_wildcard`/`array::at`/`size()`/`minify`/`validate_utf8`; `src/lib.rs` exports them through `ffi_wrap`; `include/pure_simdjson.h` is regenerated via `cbindgen`.

</code_context>

<specifics>
## Specific Ideas

- The user confirmed every recommended option across all four researched gray areas (error taxonomy, indexed access, minify shape, ValidateUTF8 shape) without modification — the research syntheses above are locked as-is, not starting points for further negotiation.
- "Allocation-conscious" for `Minify` was resolved concretely as a dual API (simple allocating convenience + explicit buffer-reuse variant) rather than left as a vague aspiration — this directly satisfies the roadmap's "explicit input/output aliasing behavior" wording, which a hidden-copy-only design would not have delivered.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 12-high-value-dom-navigation-and-simd-utility-apis*
*Context gathered: 2026-07-30*
