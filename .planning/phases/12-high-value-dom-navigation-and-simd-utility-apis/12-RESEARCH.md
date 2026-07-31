# Phase 12: High-value DOM navigation and SIMD utility APIs - Research

**Researched:** 2026-07-30
**Domain:** Four-layer FFI extension (Go purejson -> purego -> Rust `ffi_wrap`/registry -> C++ `psimdjson_bridge` -> vendored simdjson v4.6.4 DOM) adding RFC 6901/dot-path navigation, wildcard selection, indexed/size container helpers, minification, and UTF-8 validation.
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Phase Boundary:** Expose the mature, high-value parts of simdjson's DOM and implementation APIs as thin Go wrappers over the existing `Element`/`Array`/`Object` types: RFC 6901 JSON Pointer navigation (`AtPointer`), the documented simdjson dot/index path subset (`AtPath`), ordered wildcard path selection (`AtPathAll`), indexed array access plus constant-time array/object size helpers, an allocation-conscious `Minify`, and a standalone `ValidateUTF8`.

Requirements: DOM-01, DOM-02, DOM-03, DOM-04, UTIL-01, UTIL-02.

No JSON encoder/builder, reflection-based `Unmarshal`, full JSONPath (RFC 9535) engine, file-loading wrapper, or mutable DOM API is added. Zero-copy/borrowed views remain Phase 14 scope — the DOM navigation methods here return normal copied `Element` views, same as existing accessors.

**Navigation error taxonomy (DOM-01/02/03):**
- **D-01:** Add two new typed error sentinels: `ErrInvalidPath` (malformed JSON Pointer / path syntax — upstream `INVALID_JSON_POINTER`) and `ErrIndexOutOfRange` (a syntactically valid index that exceeds array bounds — upstream `INDEX_OUT_OF_BOUNDS`). Reuse the existing `ErrElementNotFound` for missing object keys / missing pointer segments (upstream `NO_SUCH_FIELD`), and the existing `ErrWrongType` for traversal type mismatches (upstream `INCORRECT_TYPE`). Do not invent a third or fourth sentinel, and do not conflate invalid-path with out-of-bounds into one error — they must be distinguishable via `errors.Is`.
- **D-02 (AMENDED 2026-07-31 — original contract refuted by spike 005):** `Element.AtPathAll` **requires at least one `*` in the path**; a wildcard-free path returns `ErrInvalidPath` before reaching the FFI boundary. It returns `([]Element{}, nil)` when the wildcard legitimately matches zero elements. Missing keys, out-of-range indices, and non-container branches yield **no match rather than an error**. **The only path error is `ErrInvalidPath`** — `ErrWrongType`, `ErrElementNotFound`, and `ErrIndexOutOfRange` are NOT reachable through `AtPathAll`. See "Upstream Wildcard Semantics" below for the pinned truth table.
- **D-03:** `ErrIndexOutOfRange` is shared between `AtPointer`/`AtPath` (pointer/path segments that resolve to an out-of-range array index) and the new indexed `Array.At` (see below) — same upstream failure, same sentinel, no duplication.

**Indexed array access and size helpers (DOM-04):**
- **D-04:** `Array.At(index int) (Element, error)` — no panic-safe/dual-method twin (no bare `Array.At(index) Element`). This mirrors the existing `Object.GetField` precedent, which also has no panic-safe twin: a zero-value `Element` returned on failure would silently launder the real out-of-range error behind a second, unrelated `ErrInvalidHandle` the next time it's touched.
- **D-05:** `index` is a Go `int`, not `uint64` — matches the existing codebase convention of casting native unsigned counts to `int` at Go boundaries (e.g. `frame.ChildCount` in the materializer). Negative indices are NOT supported (no Python-style `-1` = last element); upstream has no such concept and this project mirrors upstream's unsigned-index semantics faithfully elsewhere (e.g. `GetUint64` rejects negative reinterpretation).
- **D-06:** `Array.Len() int` / `Array.LenErr() (int, error)` and equivalent `Object` size helper follow the existing dual-method convention already used by `Type()/TypeErr()` and `IsNull()/IsNullErr()` — `Len()` is panic-safe (returns `0` on failure), `LenErr()` preserves the real error. This fits because, unlike `At`, `Len` returns a simple scalar with an unambiguous, safe zero-value fallback.
- **D-07:** Out-of-range `Array.At` returns `ErrIndexOutOfRange` (not `ErrElementNotFound`) — see D-01/D-03.

**Minify (UTIL-01):**
- **D-08:** Ship a dual API: `Minify(data []byte) ([]byte, error)` — the simple allocating convenience for the common case, matching the project's established copy-by-default precedent (`Parser.Parse`, `Element.GetString`) — plus `MinifyInto(dst, src []byte) (int, error)` — the allocation-conscious, buffer-reuse variant where `dst` may alias `src` (in-place minification), mirroring upstream's `minify(buf, len, dst, dst_len)` contract directly.
- **D-09:** `MinifyInto` is the variant that must be exercised by the required "overlap" tests (success criterion 4) — `dst == src` aliasing must be tested as a first-class supported case, not just an internal FFI-level proof. `dst` must be sized `>= len(src)` (minified output is never longer than input); undersized `dst` returns an error rather than silently truncating.
- **D-10:** Minify's buffer handling is NOT gated by Phase 14's zero-copy benchmark-gating rule. That rule exists for document-lifetime-tied borrowed views with invalidation hazards; `Minify`/`MinifyInto` operate on plain, always-caller-owned byte slices with no Doc/Parser lifetime coupling, so it is in-scope for Phase 12 as stated in UTIL-01, not deferred.

**Validated Spike Resolution — Minify Aliasing Safety:**
- **D-13 (post-discussion, spike-validated 2026-07-30):** D-08/D-09's `dst == src` aliasing contract is confirmed memory-safe and correct, not just theoretically sound. `.planning/spikes/004-minify-buffer-safety/` built a standalone ASan+UBSan probe against the untouched vendored simdjson v4.6.4 singleheader (matching production's `SIMDJSON_IMPLEMENTATION_FALLBACK=1` build.rs flag) and found: (a) upstream's own doc comments contradict each other on whether `dst` needs `len + SIMDJSON_PADDING` slack or just `len` bytes, and upstream's own multi-year fuzz harness never tests the `dst == src` case at all; (b) empirically, across the `arm64` and `fallback` kernels, `dst` sized exactly `len(src)` (no padding) triggered zero ASan/UBSan traps, and in-place (`dst == src`) output was byte-identical to the non-aliased reference across 12 fixtures including two adversarial high-compression-ratio inputs (~18KB, ~25KB), with identical results across 3 repeated runs. **No CONTEXT.md amendment to D-08/D-09 was needed — the locked decision stands as originally written.**
- **D-14 (residual, non-blocking):** The spike ran on an Apple Silicon (arm64) host and could not exercise the `haswell`/`westmere`/`icelake` x86-64 kernels. Before Phase 12 claims the aliasing guarantee across all five shipped platforms, re-run the same probe (or an equivalent contract test) on an x86-64 CI runner — fold this into the existing Rust/C++ contract test suite or five-platform smoke matrix rather than treating it as a one-off. This is a plan-phase task, not a blocker to starting the plan.

**ValidateUTF8 (UTIL-02):**
- **D-11:** `func ValidateUTF8(data []byte) (bool, error)` — not a bare bool, and not a `Type()/TypeErr()`-style dual method. The error return exists because, unlike upstream's pure C++ function, the Go entry point can genuinely fail on first call (native library not yet loaded/bootstrapped, ABI mismatch) — every other function that touches `activeLibrary()` (`NewParser`, `SetKernel`) already returns `error` for exactly this reason, and `ValidateUTF8` is the first standalone entry point that can trigger first-time library resolution without a prior `NewParser` call.
- **D-12:** No dual-method (`ValidateUTF8()`/`ValidateUTF8Err()`) — that convention exists for transient Doc/Parser handle validity, which `ValidateUTF8` has none of; its only failure mode (library load) is not a "safe to hide" condition, it's a real error every caller needs to see.

### Claude's Discretion
- Exact FFI status-code numeric values for `ErrInvalidPath` and `ErrIndexOutOfRange`, provided they are additive to the existing status-code block and follow the established 1:1 native-code-to-Go-sentinel convention.
- Internal decomposition of the dot/index path parser for `AtPath` (whether parsing happens in Go before the FFI call, or is delegated entirely to upstream's `at_path`), provided the documented subset and error behavior above are preserved.
- Whether `Object` gets a `Size()/SizeErr()` pair with the same name as `Array.Len()/LenErr()` or a distinct name — both are constant-time size accessors per DOM-04; naming should stay internally consistent between the two types.
- Internal file organization and test decomposition across Go, Rust, C++, and FFI layers for the new exports.

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| DOM-01 | `Element.AtPointer(ptr string) (Element, error)` implements RFC 6901 JSON Pointer using upstream DOM navigation | Exact grammar characterized from `element::at_pointer`/`array::at_pointer`/`object::at_pointer` source (escaping, array-index digit rules, "-" handling, trailing-separator behavior); recommended resolve-then-register bridge pattern (Pattern 1); new `ErrInvalidPath`/`ErrIndexOutOfRange` sentinels and status codes 11/12 |
| DOM-02 | `Element.AtPath(path string) (Element, error)` exposes the documented simdjson dot/index path subset without claiming full RFC 9535 support | Exact grammar characterized from `json_path_to_pointer_conversion`; two non-obvious pitfalls documented (leading `.`/`[` requirement, bracket-key quote-non-awareness) that must appear in Go doc comments and tests |
| DOM-03 | `Element.AtPathAll(path string) ([]Element, error)` exposes upstream wildcard matching with ordered, document-tied results and explicit lifetime semantics | New bulk-transport design: C++ doc-owned scratch `uint64_t` index list (transient) -> Rust-owned `Vec<pure_simdjson_value_view_t>` (each entry registered via existing `encode_descendant_view_locked`) -> new `pure_simdjson_value_views_free`; explicitly contrasted with and rejected the Phase 8 borrowed-frame-span pattern as unsuitable |
| DOM-04 | Add indexed array access plus constant-time array/object size helpers where upstream provides them | `array::at` is O(n) not O(1) (documented pitfall); `array::size()`/`object::size()` are O(1) but saturate at `0xFFFFFF` (documented pitfall, cross-referenced against this codebase's own existing internal saturation handling) |
| UTIL-01 | Add a thin, allocation-conscious `Minify` API over simdjson's SIMD minifier with explicit input/output aliasing behavior | Critical finding: upstream `simdjson::minify` has no `dst_cap` parameter and cannot itself detect undersized `dst` — the C++ bridge must add this check (mirroring the existing `copy_bytes` pattern); minify's real error surface is `UNCLOSED_STRING`-only, documented as a non-validating pass |
| UTIL-02 | Add a standalone `ValidateUTF8` API over simdjson's SIMD validator; parsing continues to validate JSON strings normally | Confirmed `simdjson::validate_utf8` is a pure bool free function with no native error surface; D-11's "genuinely fail on first call" is entirely about `activeLibrary()` bootstrap, not the UTF-8 check itself; flagged the CPU-unsupported-gate bypass gap (Pitfall 7 / Open Question 1) |
</phase_requirements>

## Summary

Phase 12's API *shapes* are already locked by `12-CONTEXT.md` (D-01..D-14) — this research focuses entirely on the mechanics of wiring six new capabilities through the existing four-layer stack without breaking any of the invariants that stack already enforces (generation-checked handles, `descendant_indices`-tracked lifetime, additive-only ABI growth, `ffi_wrap`/`catch_unwind` on every export).

Every new capability maps cleanly onto a pattern the codebase already uses. `AtPointer`/`AtPath`/`Array.At` are one-shot "resolve to a `json_index`, then register it as a descendant view" operations — structurally identical to the existing `object_get_field` path (`src/runtime/registry.rs::object_get_field` -> `encode_descendant_view_locked`). `Array.Len`/`Object.Size` are trivial O(1) tape reads, structurally identical to `element_type`. `AtPathAll` (wildcard) is the one genuinely new transport shape: it returns an *ordered set* of long-lived, document-tied `Element`s, which rules out reusing Phase 8's borrowed/invalidated-on-next-call frame-stream pattern (that pattern is correct only when the caller consumes-and-discards synchronously; wildcard results must survive and be independently usable afterward, just like any other `Element`). The recommended transport is a single FFI call that returns a Rust-owned, heap-allocated array of `pure_simdjson_value_view_t` (each entry already validated and registered via the *existing* `encode_descendant_view_locked` machinery), freed by a new companion function that mirrors the existing `bytes_free` tracked-allocation discipline.

Two upstream discoveries materially change what "thin wrapper" must mean here. First, `simdjson::minify`'s C++ signature (`minify(buf, len, dst, dst_len)`) has **no destination-capacity parameter at all** — it is architecturally incapable of detecting an undersized `dst`; a caller who under-sizes `dst` gets silent heap corruption, not an error. D-09's "undersized `dst` returns an error" requirement is therefore not optional polish, it is a **mandatory pre-check that the Rust/C++ bridge itself must own** (mirroring the existing `copy_bytes` bridge helper's `dst_cap` pattern), not something that can be left to upstream. Second, `array::size()`/`object::size()` are O(1) but **saturate at `0xFFFFFF` (16,777,215)** — this project's own `simdjson_bridge.cpp` already has a comment acknowledging this for internal materializer reserve-hinting; DOM-04's public `Len()`/`Size()` need the same honesty in their doc comments, since a container with more than ~16.7M direct children will silently report a wrong, capped count.

**Primary recommendation:** Resolve every new navigation/indexed-access operation to a `json_index` in a *new, single-purpose* C++ bridge function (mirroring `psimdjson_object_get_field_index`'s exact shape), convert that index to a registered `pure_simdjson_value_view_t` via the *existing* `encode_descendant_view_locked` in Rust, bump the ABI to `0x00010003` as a required (not optional) additive symbol block per the Phase 11 precedent, and add two new status codes (`PURE_SIMDJSON_ERR_INVALID_PATH = 11`, `PURE_SIMDJSON_ERR_INDEX_OUT_OF_RANGE = 12`) into the explicitly-reserved `11-31` gap in `docs/ffi-contract.md`'s error-code space.

## Architectural Responsibility Map

This project has no browser/server/CDN tiers; its real tiers are the FFI stack layers. Every DOM-01..04/UTIL-01/02 capability's *algorithm* lives in vendored C++; every capability's *safety contract* (handle validity, lifetime, ABI shape) lives in Rust; Go only translates and types the result.

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| RFC 6901 `AtPointer` (DOM-01) | C++ bridge (`element::at_pointer`) | Rust registry (`encode_descendant_view_locked`) | Upstream owns pointer grammar/traversal; Rust owns turning the resulting `json_index` into a lifetime-safe handle |
| Dot/index `AtPath` (DOM-02) | C++ bridge (`element::at_path` -> `json_path_to_pointer_conversion` -> `at_pointer`) | Rust registry | Same as above; upstream's own path-to-pointer conversion is the single source of truth for the "documented subset" |
| Wildcard `AtPathAll` (DOM-03) | C++ bridge (`at_path_with_wildcard`) + Rust (result transport) | Go (`[]Element` construction, empty-slice normalization) | Upstream builds the match set; Rust must invent new bulk-transport plumbing (no existing precedent covers "N long-lived handles out of one call") |
| Indexed `Array.At` (DOM-04) | C++ bridge (`array::at`, O(n) walk) | Rust registry | One FFI call replaces what would otherwise be `index` FFI round-trips if done as a Rust-side loop over `after_index` |
| `Array.Len`/`Object.Size` (DOM-04) | C++ bridge (`tape.scope_count()`, O(1)) | Go (doc comment on the `0xFFFFFF` saturation caveat) | Trivial tape read; the only real "logic" is documenting the saturation honestly |
| `Minify`/`MinifyInto` (UTIL-01) | C++ bridge (new `dst_cap` pre-check) + vendored `simdjson::minify` | Go (`activeLibrary()` first-call bootstrap, dual allocating/buffer-reuse API) | Upstream's `minify` cannot self-defend against undersized `dst` — the bridge must add the check upstream lacks |
| `ValidateUTF8` (UTIL-02) | Vendored `simdjson::validate_utf8` (pure bool, no error surface) | Go (`activeLibrary()` first-call bootstrap, the only real failure mode) | Native call essentially cannot fail; all of D-11's "genuinely fail on first call" is about library loading, not the UTF-8 check itself |

## Standard Stack

No new third-party dependencies. This phase extends the existing four-layer stack in place.

### Core
| Component | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| vendored simdjson | 4.6.4 (submodule `1bcf71bd85059ab6574ea1159de9298dcc1212c5`) | Source of `at_pointer`, `at_path`, `at_path_with_wildcard`, `array::at`, `array::size`/`object::size`, `minify`, `validate_utf8` | [VERIFIED: repo submodule] Already pinned by Phase 11; Phase 12 adds no version bump |
| cbindgen | 0.29.2 locally; pinned via `cargo install cbindgen --locked` in CI | Regenerates `include/pure_simdjson.h` from `src/lib.rs` `#[no_mangle]` exports | [VERIFIED: local `cbindgen --version`, `.github/workflows/phase2-rust-shim-smoke.yml`] |
| purego | v0.10.0 (go.mod) | Go <-> native symbol binding, no cgo | [VERIFIED: go.mod] Already pinned; no purego API changes needed for this phase |
| Rust | edition 2021, `panic = "abort"` in both dev/release profiles | `ffi_wrap`/`catch_unwind` FFI boundary | [VERIFIED: Cargo.toml] |

### Supporting
None — this phase adds zero new crates and zero new Go modules. `Cargo.toml` currently has exactly one dependency (`cc = "1"`, build-only) and no new one is needed; the wildcard bulk-transport design uses only `std::vec::Vec` and existing registry primitives.

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Delegating `AtPath`'s dot/index parsing to upstream `element::at_path` | Reimplementing `json_path_to_pointer_conversion` in Go (zero extra FFI call) | Faster (no bridge round-trip) but creates permanent drift risk against upstream's exact grammar on every future simdjson version bump; violates the phase's "thin wrapper" framing. Recommended: delegate to upstream (see Pitfall 1). |
| Rust-owned heap array for wildcard results | Doc-owned borrowed scratch buffer (Phase 8 materializer pattern) | Materializer pattern requires synchronous consume-and-discard before the next call; wildcard results must be long-lived, independently-held `Element`s (D-03), which the borrowed-span pattern cannot provide. |
| Rust-owned heap array for wildcard results | New opaque `WildcardResult` handle in the parser/doc registry (own slot/generation) | Adds a third handle *kind* to a registry currently scoped to parsers+docs; heavier than needed for a bounded, ordered, fire-and-forget struct array. |

**Installation:** No `cargo add` / `go get` changes required.

## Package Legitimacy Audit

**Not applicable.** This phase adds no external packages to any ecosystem (Rust, Go, or otherwise) — it is pure internal FFI-surface growth over already-vendored, already-pinned simdjson source. `slopcheck`/registry verification were not run because there is nothing to verify.

**Packages removed due to slopcheck [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

## Architecture Patterns

### System Architecture Diagram

```
Go caller
  │
  ▼
purejson public API (element.go / minify.go / utf8.go)      [NEW: AtPointer, AtPath, AtPathAll,
  │  Element.usableDoc() guard, wrapStatus() error mapping    Array.At/Len, Object.Size, Minify,
  │                                                            MinifyInto, ValidateUTF8]
  ▼
internal/ffi.Bindings (purego symbol table)                  [NEW: 8-9 registered function pointers,
  │  registerFuncWithRegistrar() -- REQUIRED, not optional     required per docs/ffi-contract.md]
  ▼
Rust src/lib.rs  pure_simdjson_* extern "C" exports           [NEW: wrapped in ffi_wrap(), each calls
  │  ffi_wrap() -> catch_unwind() -> err_panic() on unwind      into registry::*]
  ▼
Rust src/runtime/registry.rs                                  [NEW: element_at_pointer(), element_at_path(),
  │  with_resolved_view() validates {doc generation, state1     element_at_path_wildcard(), array_at(),
  │  tag, descendant_indices membership} BEFORE any native      array_size(), object_size()]
  │  call; encode_descendant_view_locked() registers new        -- reuses existing validation, does NOT
  │  json_index results into entry.descendant_indices            duplicate it
  ▼
Rust src/runtime/mod.rs  unsafe extern "C" { ... }             [NEW: psimdjson_* FFI declarations mirroring
  │  thin usize-pointer wrappers, no logic                      the C++ bridge 1:1]
  ▼
C++ src/native/simdjson_bridge.cpp  psimdjson_*                [NEW: psimdjson_element_at_pointer_index,
  │  map_error() translates simdjson::error_code ->             psimdjson_element_at_path_index,
  │  pure_simdjson_error_code_t (NEEDS 2 new cases)              psimdjson_element_at_path_wildcard_indices,
  │  try/catch -> PSIMDJSON_CATCH_CPP_EXCEPTIONS                 psimdjson_array_at_index, psimdjson_array_size,
  │                                                               psimdjson_object_size, psimdjson_minify,
  │                                                               psimdjson_validate_utf8]
  ▼
third_party/simdjson (v4.6.4, untouched)                       element::at_pointer / at_path /
                                                                 at_path_with_wildcard, array::at,
                                                                 array::size/object::size, simdjson::minify,
                                                                 simdjson::validate_utf8
```

Data flow for the two shapes that matter most:

- **Single-result navigation** (`AtPointer`, `AtPath`, `Array.At`): Go call -> purego -> Rust `ffi_wrap` -> `with_resolved_view` (validates the *input* view) -> C++ bridge resolves one `json_index` via upstream -> Rust `encode_descendant_view_locked` (registers the *output* view) -> `ValueView` struct returned by value (32 bytes, no allocation) -> Go wraps it as `Element{doc, view}`.
- **Bulk-result navigation** (`AtPathAll`): same as above through the C++ bridge, except the bridge produces N `json_index` values into a doc-owned scratch `std::vector<uint64_t>` (transient, valid only for the duration of this one call) -> Rust reads that borrowed slice *synchronously within the same call*, converts each index to a registered `ValueView` via the same `encode_descendant_view_locked`, and copies the N `ValueView`s into a freshly Rust-heap-allocated, **owned** `Vec<pure_simdjson_value_view_t>` before returning -> Go copies that into a Go slice of `Element` and immediately calls the new free function.
- **Standalone utilities** (`Minify`, `MinifyInto`, `ValidateUTF8`): no `Doc`/handle at all. Go calls `activeLibrary()` (same double-checked-locking bootstrap `NewParser`/`SetKernel` already use) -> purego -> Rust `ffi_wrap` -> C++ bridge calls the vendored free function directly, no registry involvement.

### Recommended Project Structure

No new directories. New files follow the existing flat-package convention (see `kernel.go` for the exact style precedent — a small top-level file with `activeLibrary()`-first standalone functions):

```
element.go          # + AtPointer, AtPath, AtPathAll on Element; At, Len/LenErr on Array; Size/SizeErr on Object
errors.go            # + ErrInvalidPath, ErrIndexOutOfRange sentinels + sentinelForStatus() cases
minify.go            # NEW: Minify, MinifyInto (mirrors kernel.go's activeLibrary()-first style)
utf8.go               # NEW: ValidateUTF8
internal/ffi/types.go   # + ErrInvalidPath=11, ErrIndexOutOfRange=12; ABIVersion -> 0x00010003
internal/ffi/bindings.go  # + 8 new required function-pointer fields + registration + typed wrapper methods
src/lib.rs                # + 8 new pure_simdjson_* extern "C" exports, each ffi_wrap-wrapped
src/runtime/mod.rs          # + 8 new psimdjson_* unsafe extern "C" declarations + thin wrappers
src/runtime/registry.rs      # + element_at_pointer/at_path/at_path_wildcard, array_at, array_size, object_size
src/native/simdjson_bridge.h   # + 8 new psimdjson_* declarations
src/native/simdjson_bridge.cpp  # + implementations; map_error() gains 2 new cases; psimdjson_doc gains
                                  #   wildcard_indices scratch vector + wildcard_in_progress guard
include/pure_simdjson.h          # regenerated via `make generate-header` (cbindgen picks up new exports
                                  #   automatically; the two new structs/enums need no manual cbindgen.toml change)
docs/ffi-contract.md              # + error codes 11/12, ABI 1.3 section, value-views-free ownership note
tests/abi/check_header.py          # + 8 new names appended to REQUIRED_SYMBOLS tuple (see Pitfall-adjacent note below)
```

### Pattern 1: Resolve-then-register (single result)

**What:** Every new single-result accessor follows the exact shape `object_get_field` already established: validate the input view, ask the C++ bridge for a target `json_index`, register that index as a descendant, return a `ValueView`.
**When to use:** `AtPointer`, `AtPath`, `Array.At`.
**Example (Rust registry, modeled directly on the existing `object_get_field`):**
```rust
// Source: src/runtime/registry.rs (existing object_get_field, lines 1154-1168) — the
// pattern to replicate for element_at_pointer / element_at_path / array_at.
pub(crate) fn object_get_field(
    object_view: *const pure_simdjson_value_view_t,
    key: &[u8],
) -> Result<pure_simdjson_value_view_t, pure_simdjson_error_code_t> {
    with_resolved_view(object_view, |entry, json_index, doc| {
        let kind = super::native_element_type_at(entry.native_ptr, json_index)?;
        if kind != KIND_HINT_OBJECT {
            return Err(err_wrong_type());
        }
        let value_json_index =
            super::native_object_get_field_index(entry.native_ptr, json_index, key)?;
        encode_descendant_view_locked(entry, doc, value_json_index)
    })
}

// NEW, same shape, no kind pre-check needed (at_pointer/at_path dispatch on kind
// internally in C++ and work on scalars too when the pointer is empty):
pub(crate) fn element_at_pointer(
    view: *const pure_simdjson_value_view_t,
    pointer: &[u8],
) -> Result<pure_simdjson_value_view_t, pure_simdjson_error_code_t> {
    with_resolved_view(view, |entry, json_index, doc| {
        let value_json_index =
            super::native_element_at_pointer_index(entry.native_ptr, json_index, pointer)?;
        encode_descendant_view_locked(entry, doc, value_json_index)
    })
}
```

### Pattern 2: C++ bridge dst_cap pre-check (mandatory for Minify)

**What:** `copy_bytes()` (already in `simdjson_bridge.cpp`, used by `psimdjson_copy_implementation_name`/`psimdjson_parser_copy_last_error`) already implements exactly the "write required size, then reject if `dst_cap` too small, before touching `dst`" contract that `MinifyInto` needs. Reuse this shape verbatim rather than inventing a new one.
**When to use:** `psimdjson_minify` — this is not optional, since upstream's own `simdjson::minify(buf, len, dst, dst_len)` has no `dst_cap` parameter and cannot detect an undersized `dst` itself (see Pitfall 5).
**Example:**
```cpp
// Source: src/native/simdjson_bridge.cpp lines 125-150 (existing copy_bytes) — the exact
// pattern psimdjson_minify's dst_cap pre-check must follow.
pure_simdjson_error_code_t copy_bytes(
    std::string_view src, uint8_t *dst, size_t dst_cap, size_t *out_written
) noexcept {
  if (out_written == nullptr) return invalid_argument();
  *out_written = src.size();
  if (src.size() > dst_cap) return PURE_SIMDJSON_ERR_BUFFER_TOO_SMALL;
  if (!src.empty() && dst == nullptr) return invalid_argument();
  if (!src.empty()) std::memcpy(dst, src.data(), src.size());
  return PURE_SIMDJSON_OK;
}

// NEW: psimdjson_minify must pre-check dst_cap >= src_len BEFORE calling
// simdjson::minify at all, because minify's own signature has no dst_cap.
pure_simdjson_error_code_t psimdjson_minify(
    const uint8_t *src_ptr, size_t src_len,
    uint8_t *dst_ptr, size_t dst_cap,
    size_t *out_written
) noexcept {
  try {
    if (out_written == nullptr) return invalid_argument();
    *out_written = src_len; // conservative upper bound: minified len is never > src_len
    if (dst_cap < src_len) return PURE_SIMDJSON_ERR_BUFFER_TOO_SMALL;
    if (src_len != 0 && (src_ptr == nullptr || dst_ptr == nullptr)) return invalid_argument();
    if (src_len == 0) { *out_written = 0; return PURE_SIMDJSON_OK; }

    size_t written = 0;
    auto err = simdjson::minify(reinterpret_cast<const char *>(src_ptr), src_len,
                                 reinterpret_cast<char *>(dst_ptr), written);
    if (err != simdjson::SUCCESS) { *out_written = 0; return map_error(err); }
    *out_written = written;
    return PURE_SIMDJSON_OK;
  } PSIMDJSON_CATCH_CPP_EXCEPTIONS(__func__)
}
```

### Anti-Patterns to Avoid

- **Trusting `simdjson::minify`'s doc comment ("`dst` MUST be allocated up to `len` bytes") as sufficient documentation without an enforced check:** the comment describes a *caller obligation*, not a *runtime-checked invariant*. Every layer of this codebase currently treats undersized-buffer as a checked, returned error (`PURE_SIMDJSON_ERR_BUFFER_TOO_SMALL`) — minify must not be the one silent exception.
- **Reusing Phase 8's `psdj_internal_materialize_build` borrowed-span pattern for `AtPathAll`:** that pattern's contract ("valid until the next materialize-build call on the same doc") is fundamentally incompatible with returning long-lived `Element`s the caller may hold indefinitely.
- **Registering the new DOM-nav/minify/validate-utf8 symbols as *optional* (`registerOptionalFuncWithRegistrar`)**, the way `psdj_internal_materialize_build`/`native_alloc_stats_*` are: those are internal-diagnostic surfaces this project deliberately allows released artifacts to lack. DOM-01..04/UTIL-01/02 are public, documented API — a native artifact claiming ABI 1.3 without them is corrupt per `docs/ffi-contract.md`'s existing rule ("An artifact claiming compatible ABI ... but missing any required symbol is corrupt/incomplete and must fail closed").

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| RFC 6901 pointer parsing (escaping, array-index digit rules, "-" append sentinel) | A Go or Rust reimplementation of JSON Pointer parsing | Upstream `element::at_pointer`/`array::at_pointer`/`object::at_pointer` (already vendored, already fuzzed by upstream's own test suite) | Escaping (`~0`/`~1`), leading-zero rejection, and the deliberate "-"-is-out-of-bounds choice are exact, non-obvious upstream decisions (see Pitfall 2) that a reimplementation would have to track forever across every future simdjson version bump |
| Dot/index path -> JSON Pointer conversion | A second parser for `.a[0]` syntax | Upstream `json_path_to_pointer_conversion` (`third_party/simdjson/include/simdjson/jsonpathutil.h`) via `element::at_path` | This is the literal, single, exact grammar the phase must claim compliance with; reimplementing it risks silent divergence, and the phase's own framing is "thin Go wrappers ... over upstream" |
| Wildcard match collection | Hand-rolled recursive descent over the DOM tree | Upstream `at_path_with_wildcard` | Already handles `.*`/`[*]` and nested wildcard-then-suffix traversal (`process_json_path_of_child_elements`); reimplementing this in Rust/Go duplicates real logic for no benefit |
| Undersized-destination detection for minify | A Go-only length check before the FFI call, trusting the native side is safe | A Rust/C++-level `dst_cap` check via the `copy_bytes` pattern | Defense-in-depth: this project's established idiom is that every buffer-write FFI boundary checks its own destination capacity, not just the Go caller. A Go-only check would be bypassed by any future direct Rust/C caller (fuzzers, other bindings) |
| Bulk-array ownership tracking for wildcard results | A second, ad hoc `unsafe` free path outside the existing allocation-tracking discipline | Extend the existing tracked-allocation pattern (`registry.byte_allocations`-equivalent map keyed by pointer) for the new `Vec<pure_simdjson_value_view_t>` allocation | `bytes_free`'s registered-allocation check (exact pointer+length match before `Vec::from_raw_parts`) is what prevents double-free/wrong-size-free bugs; a parallel ad hoc free path would reintroduce exactly the class of bug that discipline exists to prevent |

**Key insight:** every "don't hand-roll" item in this phase reduces to the same rule: **the algorithm belongs to vendored simdjson; this project's own code must only ever translate results and enforce lifetime/ownership**, never reimplement JSON-path semantics or memory-safety checks that upstream already got right (or, in minify's case, checks that upstream deliberately left to the caller).

## Common Pitfalls

### Pitfall 1: `AtPath`'s bracket-key form is NOT quote-aware (silent wrong-key lookup)

**What goes wrong:** `element::at_path`/`array::at_path`/`object::at_path` (the *non*-wildcard variants) all convert via `json_path_to_pointer_conversion` (`third_party/simdjson/include/simdjson/jsonpathutil.h:15-64`). That function's bracket-handling loop copies characters verbatim between `[` and `]` (only escaping literal `~`/`/`) — it does **not** strip surrounding quote characters. So `AtPath(".obj['foo']")` produces the JSON Pointer segment `'foo'` (11 characters, including both single-quote chars), which looks up the *literal* object key `'foo'` — almost certainly not the key the caller meant (`foo`). Unquoted bracket form, `AtPath(".obj[foo]")`, correctly looks up key `foo` (since object pointer segments are literal-string lookups with no digit requirement).
**Why it happens:** The quote-stripping logic (`get_next_key_and_json_path` in the same header, lines 66-106) exists **only** in the wildcard code path (`at_path_with_wildcard`), not in the plain `at_path` -> `json_path_to_pointer_conversion` path used by `AtPath`/`AtPointer`'s dot form. This is an asymmetry in upstream itself, not a bug this project introduces.
**How to avoid:** Document this precisely in `Element.AtPath`'s Go doc comment (state that bracket-quoted string keys, `['key']`/`["key"]`, are **not** supported in `AtPath` and will look up the literal quoted string as a key — recommend dot form or unquoted bracket form for object keys). Add an explicit unit test asserting this exact (surprising) behavior so it can never be "fixed" as an unnoticed regression later, and cannot be mistaken for a wrapper bug by a future contributor.
**Warning signs:** A user reports `AtPath` returning `ErrElementNotFound` for a key that visibly exists in the JSON, when they used bracket-quoted syntax.

### Pitfall 2: `AtPath` requires a leading `.` or `[` — bare field names are always invalid

**What goes wrong:** `json_path_to_pointer_conversion` requires, after an optional leading `$`, that the very next character be `.` or `[`; otherwise it returns the `"-1"` sentinel, mapped to `INVALID_JSON_POINTER` (-> new `ErrInvalidPath`). `AtPath("name")` is **always** invalid; it must be `AtPath(".name")` or `AtPath("$.name")`.
**Why it happens:** upstream's grammar was designed to mirror JSONPath's `$.field` convention, not bare dotted-path notation like `gjson`/many other Go JSON-path libraries use.
**How to avoid:** State the exact required-leading-character rule in the Go doc comment with a working example; add boundary tests for `""`(empty, invalid), `"name"` (no leading separator, invalid), `".name"` (valid), `"$.name"` (valid).
**Warning signs:** Every `AtPath` call in a test suite failing with `ErrInvalidPath` despite looking syntactically reasonable to someone coming from a different path-query library.

### Pitfall 3: Trailing separator navigates into an empty-string key on the child, not a no-op

**What goes wrong:** `AtPointer("/a/")` (trailing slash) or `AtPath(".a.")` (trailing dot) does not error and does not just return `a`'s value — `object::at_pointer` recurses into `a`'s value with the *remaining* pointer `"/"`, which resolves to looking up the literal empty-string key `""` inside `a`'s value. This returns `ErrElementNotFound` unless `a`'s value is an object that happens to have a `""` key, or `ErrWrongType` if `a`'s value isn't an object at all.
**Why it happens:** RFC 6901 treats `/` as a segment separator, and a pointer ending in `/` has an empty final segment by definition; simdjson implements this literally rather than special-casing a trailing separator as "stop here."
**How to avoid:** Document explicitly; include a test asserting `AtPointer("/a/")` on `{"a": 1}` returns `ErrWrongType` (since `1` is not an object) and `AtPointer("/a/")` on `{"a": {"": "x"}}` returns the string `"x"`.
**Warning signs:** Off-by-one-looking failures when path strings are built by string concatenation and accidentally retain a trailing separator.

### Pitfall 4: `Array.At`/`array::at` is O(n), not O(1) — despite being "indexed access"

**What goes wrong:** `array::at(size_t index)` (`third_party/simdjson/include/simdjson/dom/array-inl.h:228-236`) is a **linear scan** via the array iterator (`for (auto element : *this) { if (i == index) return element; i++; }`), because simdjson's DOM tape has no random-access index into array elements — only sequential `after_element()` stepping. Calling `Array.At(i)` in a loop over all `i` makes the whole traversal O(n^2), unlike `Array.Iter()` which is O(n) total.
**Why it happens:** The tape format is a flat sequence with only "how far to skip to get past this element" markers, not a per-element offset table.
**How to avoid:** State the O(n) cost explicitly in `Array.At`'s Go doc comment and explicitly recommend `Array.Iter()` for full traversal. This project's whole positioning is "selective traversal is the performance story" (`PROJECT.md`) — an unlabeled O(n) `At` is exactly the kind of surprise that undermines that positioning.
**Warning signs:** Benchmark regressions on any workload that walks an array via repeated `At(i)` calls instead of `Iter()`.

### Pitfall 5: `simdjson::minify` cannot detect an undersized destination buffer — the bridge must

**What goes wrong:** `simdjson::minify(const char *buf, size_t len, char *dst, size_t &dst_len)` (`third_party/simdjson/include/simdjson/minify.h:26`) has **no destination-capacity parameter**. Its only documented contract is a caller obligation ("`dst` *MUST* be allocated up to `len` bytes"). The actual minifier implementation (`third_party/simdjson/src/generic/stage1/json_minifier.h`) writes via `dst += in.compress(mask, dst)` with zero bounds checking against any caller-supplied capacity — because there is no such parameter to check against. An undersized `dst` is a straight heap-buffer-overflow (verified by reading the implementation directly, not inferred).
**Why it happens:** upstream's public API predates (and is orthogonal to) this project's `dst_cap`-checked-copy convention; it was designed for callers who always pre-size `dst` correctly (e.g. `len(src)` bytes), with no defensive layer.
**How to avoid:** The new `psimdjson_minify` bridge function must accept an explicit `dst_cap` parameter distinct from `dst`, and must check `dst_cap >= src_len` **before** calling `simdjson::minify` at all — mirroring the exact `copy_bytes` pattern this codebase already uses everywhere else a caller-provided buffer exists (see Pattern 2 above). This is D-09's error requirement, but the *research* finding is that it cannot be satisfied by a Go-only pre-check; it must live at the C++/Rust boundary as well, consistent with every other buffer-write export in this codebase.
**Warning signs:** ASan/heap-corruption crashes only under fuzzing or adversarial input that deliberately under-sizes `dst` — will not reproduce in normal correct-usage testing, making this exactly the kind of bug that ships silently without the explicit check.

### Pitfall 6: `simdjson::minify` validates almost nothing — it is not a JSON validator

**What goes wrong:** The only error `json_minifier::minify`'s underlying scanner can produce is `UNCLOSED_STRING` (confirmed directly in `third_party/simdjson/src/generic/stage1/json_scanner.h:111`: "Returns either `UNCLOSED_STRING` or `SUCCESS`"). Mismatched braces, trailing garbage, invalid tokens (`{"a":tru}`), and other malformed-but-string-terminated JSON will "successfully" minify without error.
**Why it happens:** `minify` is a whitespace-stripping pass over a tokenizer, not a parser; the doc comment says this explicitly ("does not parse or validate").
**How to avoid:** Document clearly that `Minify`/`MinifyInto` success does **not** imply the input was valid JSON, only that string boundaries were well-formed. Do not let README/godoc language imply `Minify` can be used as a cheap validity pre-check.
**Warning signs:** A caller assumes `err == nil` from `Minify` means the JSON was valid, then is surprised when `Parser.Parse` on the same input later returns `ErrInvalidJSON`.

### Pitfall 7: Standalone `Minify`/`ValidateUTF8` bypass the existing CPU-unsupported ("fallback kernel") gate

**What goes wrong:** `PURE_SIMDJSON_ERR_CPU_UNSUPPORTED` is currently only returned from `psimdjson_parser_new`/`psimdjson_parser_new_configured` (via `reject_fallback_implementation()` in Rust and the mirrored check in `parser_new_configured_with_selection_lock` in C++, `simdjson_bridge.cpp:716-758`). `simdjson::minify`/`simdjson::validate_utf8` are free functions that call `get_active_implementation()` directly and never touch parser construction. If a caller's very first library interaction is `purejson.ValidateUTF8(data)` (never having called `NewParser`), and the host CPU's best-supported kernel is `fallback`, the call will silently succeed using the slow scalar kernel — with **no** `ErrCPUUnsupported`, contradicting this project's stated "unsupported CPUs fail loudly, not silently" policy (`REQUIREMENTS.md` Out of Scope table).
**Why it happens:** This is the first phase to expose SIMD-kernel-dependent functionality with *no* parser construction anywhere in the call path — a genuinely new code shape for this codebase, not a regression.
**How to avoid:** This is a real, currently-unresolved design gap — see Open Questions. Recommended resolution: have `psimdjson_minify`/`psimdjson_validate_utf8` call the same fallback-rejection check `parser_new_configured_with_selection_lock` uses (and correspondingly lock kernel selection, matching `NewParser`'s existing `kernelSelectionLocked = true` behavior on the Go side), so the *first* SIMD-touching call of any kind — parser or utility — is the one that locks in kernel selection and enforces the CPU-support policy.
**Warning signs:** A CI job or user machine running solely `Minify`/`ValidateUTF8` (no `NewParser`) on an unsupported CPU produces correct-but-unexpectedly-slow results instead of a clear startup error.

### Pitfall 8: `array::size()`/`object::size()` silently saturate at 16,777,215 direct children

**What goes wrong:** `array::size()`/`object::size()` read `tape.scope_count()` (`third_party/simdjson/include/simdjson/internal/tape_ref-inl.h:80-82`), a 24-bit field (`JSON_COUNT_MASK = 0xFFFFFF`) baked into the tape word at parse time. Upstream's own tape-dump debug code literally labels this "saturated count" (`third_party/simdjson/include/simdjson/dom/document-inl.h:125,134`). A container with more than 16,777,215 direct children reports a **capped, wrong** count — silently, with no error at parse time or at `.size()` call time.
**Why it happens:** The tape's per-scope word packs `next_tape_location:u32` and `count:u24` into one 64-bit word by design; simdjson accepts the tradeoff since >16.7M-direct-child containers are extremely rare in practice.
**How to avoid:** State this exact numeric threshold in `Array.Len`/`Object.Size`'s Go doc comments. This codebase already discovered and handled this exact issue internally (`simdjson_bridge.cpp`'s `has_unsaturated_child_hint`, used only for internal materializer scratch-vector reserve sizing) — DOM-04 is the first phase to make it a *public*, user-facing correctness caveat, and it must be documented with the same honesty this project applies to BigInt/precision-loss elsewhere.
**Warning signs:** None observable in typical testing — this only manifests on adversarial or genuinely huge flat containers, which is exactly why it needs proactive documentation rather than reactive bug reports.

### Pitfall 9: `REQUIRED_SYMBOLS` in `tests/abi/check_header.py` is a closed allowlist that WILL fail the build

**What goes wrong:** `tests/abi/check_header.py`'s `rule_required_symbols` (lines 48-79, 189-199) both requires every name in its hardcoded `REQUIRED_SYMBOLS` tuple to be present AND fails on any `pure_simdjson_*`-prefixed symbol in the generated header that is *not* in that tuple ("unexpected exported symbols"). `make verify-contract` runs this rule. Adding any new public Rust export (e.g. `pure_simdjson_element_at_pointer`) without also adding its name to this Python tuple will fail CI immediately, even though the Rust/cbindgen/header side is otherwise correct.
**Why it happens:** This is a deliberate closed-world check to prevent accidental/undocumented ABI growth — exactly the kind of guard that is easy to forget when adding a batch of new symbols, since nothing about the Rust or C++ side hints that this file exists.
**How to avoid:** Treat updating `tests/abi/check_header.py`'s `REQUIRED_SYMBOLS` tuple as a mandatory, explicit task in the plan, verified by running `make verify-contract` locally before considering any wave "done."
**Warning signs:** `make verify-contract` failing with "unexpected exported symbols: pure_simdjson_..." after otherwise-correct Rust/C++ work.

## Code Examples

### Wildcard bulk-transport: C++ bridge shape (new)

```cpp
// Source: modeled on the existing borrowed-scratch-buffer pattern in
// src/native/simdjson_bridge.cpp (psimdjson_materialize_build, lines 1543-1579)
// and psimdjson_object_get_field_index (lines 1508-1541). psimdjson_doc gains:
//   std::vector<uint64_t> wildcard_indices{};
//   bool wildcard_in_progress{false};
// guarded the same way materialize_frames/materialize_in_progress already are.
pure_simdjson_error_code_t psimdjson_element_at_path_wildcard_indices(
    psimdjson_doc *doc,
    uint64_t json_index,
    const uint8_t *path_ptr,
    size_t path_len,
    const uint64_t **out_indices,
    size_t *out_count
) noexcept {
  try {
    if (doc == nullptr || out_indices == nullptr || out_count == nullptr) {
      return invalid_argument();
    }
    *out_indices = nullptr;
    *out_count = 0;

    wildcard_build_guard guard(doc);  // same shape as materialize_build_guard
    if (!guard.acquired()) return PURE_SIMDJSON_ERR_PARSER_BUSY;

    std::string_view path(reinterpret_cast<const char *>(path_ptr), path_len);
    std::vector<simdjson::dom::element> matches;
    auto error = element_at(doc, json_index).at_path_with_wildcard(path).get(matches);
    if (error != simdjson::SUCCESS) return map_error(error);

    doc->wildcard_indices.clear();
    doc->wildcard_indices.reserve(matches.size());
    for (const auto &el : matches) {
      doc->wildcard_indices.push_back(element_json_index(el));
    }
    if (!doc->wildcard_indices.empty()) {
      *out_indices = doc->wildcard_indices.data();
      *out_count = doc->wildcard_indices.size();
    }
    return PURE_SIMDJSON_OK;
  } PSIMDJSON_CATCH_CPP_EXCEPTIONS(__func__)
}
```

### Wildcard bulk-transport: Rust owned-array handoff (new)

```rust
// Source: modeled on the existing tracked-allocation pattern in
// src/runtime/registry.rs (element_get_bytes_copy, lines 837-881, and
// bytes_free, lines 895-927). The u64 scratch slice from C++ is read
// synchronously and converted into an OWNED Vec<ValueView> before this
// function returns -- unlike materialize_build, nothing borrowed escapes.
pub(crate) fn element_at_path_wildcard(
    view: *const pure_simdjson_value_view_t,
    path: &[u8],
) -> Result<(*mut pure_simdjson_value_view_t, usize), pure_simdjson_error_code_t> {
    let views: Vec<pure_simdjson_value_view_t> = with_resolved_view(view, |entry, json_index, doc| {
        let (indices_ptr, count) =
            super::native_element_at_path_wildcard_indices(entry.native_ptr, json_index, path)?;
        // SAFETY: indices_ptr/count describe the doc-owned scratch vector filled
        // synchronously by the call above; read it before returning.
        let indices: &[u64] = if count == 0 { &[] } else {
            unsafe { slice::from_raw_parts(indices_ptr, count) }
        };
        let mut out = Vec::with_capacity(indices.len());
        for &json_index in indices {
            out.push(encode_descendant_view_locked(entry, doc, json_index)?);
        }
        Ok(out)
    })?;

    if views.is_empty() {
        return Ok((ptr::null_mut(), 0));
    }
    let mut boxed = views.into_boxed_slice().into_vec();
    let ptr = boxed.as_mut_ptr();
    let len = boxed.len();
    mem::forget(boxed);
    // Register (ptr, len) in a tracked-allocation table the same way
    // element_get_bytes_copy registers byte allocations, so
    // pure_simdjson_value_views_free can validate before freeing.
    Ok((ptr, len))
}
```

### Go public surface (new)

```go
// AtPathAll returns ordered, document-tied elements matching a wildcard path
// query, per the documented simdjson dot/index subset with '*'/'[*]' wildcard
// segments. Zero matches is not an error: it returns ([]Element{}, nil).
// AtPathAll does not implement full RFC 9535 JSONPath.
func (e Element) AtPathAll(path string) ([]Element, error) {
	doc, err := e.usableDoc()
	if err != nil {
		return nil, err
	}
	views, rc := doc.parser.library.bindings.ElementAtPathWildcard(&e.view, path)
	runtime.KeepAlive(doc)
	if err := wrapStatus(rc); err != nil {
		return nil, err
	}
	out := make([]Element, len(views))
	for i, v := range views {
		out[i] = Element{doc: doc, view: v}
	}
	return out, nil // len(out)==0 -> non-nil empty slice, matches D-02
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| Deprecated `element::at(string_view)` (pre-RFC-6901 pointer syntax, no leading `/` required) | `element::at_pointer(string_view)` (RFC 6901-compliant) | simdjson v0.4+ (long predates the v4.6.4 pin) | `element::at` is `[[deprecated]]` in the vendored header (`element-inl.h:144-153`); do not build on it even internally — `AtPointer` must map to `at_pointer`, never the deprecated `at` |

**Deprecated/outdated:** `simdjson::dom::element::at(std::string_view)` — superseded by `at_pointer`; the vendored header carries a compiler deprecation attribute with the message "For standard compliance, use at_pointer instead."

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | New status codes should be placed at values 11/12 (immediately after `PURE_SIMDJSON_ERR_KERNEL_LOCKED = 10`) rather than elsewhere in the reserved `11-31` band | Common Pitfalls / Architecture (error code space) | Low — `docs/ffi-contract.md` already documents `11-31` as reserved-for-future-additive with no further constraint, and CONTEXT.md D-01/discretion explicitly leaves exact numeric placement to Claude's discretion; any value in that band is contract-compliant, this is a stylistic recommendation only |
| A2 | ABI should bump to `0x00010003` (next minor) rather than staying at `0x00010002` | Architecture Patterns / State of the Art | Medium — if the planner instead treats this as non-breaking-enough to skip the bump, `docs/ffi-contract.md`'s own required-symbol-set-per-ABI-version rule and `tests/abi/check_header.py`'s closed `REQUIRED_SYMBOLS` allowlist would still force *some* explicit acknowledgment (the test fails either way until updated), so this is verifiable rather than purely assumed — but the exact new numeric value (`0x00010003` vs. some other minor) is an inference from the Phase 11 precedent (0x00010001 -> 0x00010002 for a comparable-sized additive surface), not a fixed rule stated anywhere |
| A3 | Standalone `Minify`/`ValidateUTF8` should also gate on the CPU-unsupported fallback check and lock kernel selection, matching `NewParser` | Pitfall 7 / Open Questions | Medium — this is a genuine open design question not resolved by CONTEXT.md; if the planner decides the opposite (utilities intentionally bypass the gate, since correctness is unaffected, only speed), doc comments and tests need the opposite framing |

## Open Questions (ALL RESOLVED at plan-phase, 2026-07-30)

> Q1 resolved by the user → recorded as CONTEXT.md **D-15**, implemented in plans 12-04 / 12-07.
> Q2 resolved by adopting the recommendation (separate `view_array_allocations` map) → implemented in plan 12-03.
> Q3 resolved by adopting the recommendation (new step in the existing `linux-smoke` job) → implemented in plan 12-04 Task 3.
> No live ambiguity reaches the executor: every consuming plan encodes one consistent answer.

1. **(RESOLVED — see D-15)** **Should `Minify`/`MinifyInto`/`ValidateUTF8` trigger the same CPU-unsupported ("fallback kernel") rejection that `NewParser`/`NewParserPool` already enforce?**
   - What we know: today, `ERR_CPU_UNSUPPORTED` is only reachable via parser construction; the new standalone entry points call into SIMD-accelerated upstream code without ever constructing a parser, so they would silently use the `fallback` kernel on unsupported CPUs today.
   - What's unclear: whether this is acceptable (utilities are "best effort, just slower" and correctness is unaffected) or a policy violation (this project's stated philosophy is "unsupported CPUs fail loudly, not silently").
   - Recommendation: gate all three new standalone entry points behind the same `reject_fallback_implementation()`-equivalent check used by parser construction, and have them call `lockKernelSelection()` on the Go side (mirroring `NewParser`) on first successful native call, so `SetKernel` after a `Minify`/`ValidateUTF8` call is treated identically to `SetKernel` after `NewParser`.

2. **(RESOLVED — recommendation adopted, plan 12-03)** **Exact bulk-array free-function naming and whether it shares an allocation-tracking table with `bytes_free`.**
   - What we know: the existing `registry.byte_allocations: HashMap<usize, usize>` tracks byte-buffer allocations by pointer, keyed to an exact length, and `bytes_free` rejects any pointer/length pair that doesn't match.
   - What's unclear: whether the wildcard result array should reuse that exact map (values would then be ambiguous between "N bytes" and "N structs" unless tagged) or use a second, parallel map scoped to `pure_simdjson_value_view_t` arrays.
   - Recommendation: use a second, separate map (e.g. `view_array_allocations: HashMap<usize, usize>`, storing element *count* not byte length) — simpler to reason about than overloading one map's units, and this is explicitly left to Claude's discretion in CONTEXT.md ("internal file organization and test decomposition ... for the new exports").

3. **(RESOLVED — recommendation adopted, plan 12-04 Task 3)** **Whether the D-14 x86-64 aliasing verification should run as a dedicated new CI job or as an added step inside the existing `linux-smoke` job in `phase2-rust-shim-smoke.yml`.**
   - What we know: `verify.sh` already loops over every kernel `get_available_implementations()` reports as `supported_by_runtime_system()`, so running it unmodified on `ubuntu-latest` (an x86-64 runner) will naturally exercise `haswell`/`westmere`/`fallback` (and `icelake` if the runner CPU happens to support AVX-512, gracefully skipped otherwise) with zero probe changes. This is corroborated by `build.rs` unconditionally defining `SIMDJSON_IMPLEMENTATION_FALLBACK=1` (all target artifacts, including the ones built for x86-64 CI, already compile the fallback kernel alongside whatever x86 kernels the target architecture enables by default).
   - What's unclear: whether promotion means "copy `verify.sh` invocation into a new permanent CI step" (recommended, per spike CONVENTIONS.md's promotion rule "promote the finding into phase research or an execution plan; do not copy spike code into production blindly" — the *spike script itself* stays in `.planning/spikes/`, only its *invocation* gets promoted into CI) or "port `probe.cpp`'s logic into a permanent Rust/C++ contract test using the project's own `cc`-compiled bridge instead of the untouched singleheader."
   - Recommendation: add a new step to the existing `linux-smoke` job (`.github/workflows/phase2-rust-shim-smoke.yml`) that runs `bash .planning/spikes/004-minify-buffer-safety/verify.sh` — this is the minimal, lowest-risk way to satisfy D-14's "fold this into the existing Rust/C++ contract test suite or five-platform smoke matrix rather than treating it as a one-off" without introducing a second probe implementation to maintain.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Rust toolchain + cargo | All new bridge/registry/lib.rs code | ✓ (local + CI, `.github/workflows/phase2-rust-shim-smoke.yml`) | cargo 1.89.0 local; `dtolnay/rust-toolchain@stable` in CI | — |
| cbindgen | Header regeneration (`make generate-header`) | ✓ (local + CI, `cargo install cbindgen --locked`) | 0.29.2 local | — |
| clang++ (or `$CXX`) | D-14 x86-64 ASan/UBSan probe (`.planning/spikes/004-minify-buffer-safety/verify.sh`) | ✓ local (Apple clang 21.0.0); assumed present on `ubuntu-latest` GH runners (standard image) | — | `apt-get install clang` if the runner image lacks it |
| Go toolchain | Public API + tests | ✓ local | go1.26.5 darwin/arm64; project targets Go per go.mod | — |
| purego v0.10.0 | Symbol binding, no new API surface needed | ✓ (go.mod, unchanged) | v0.10.0 | — |

**Missing dependencies with no fallback:** none identified.

**Missing dependencies with fallback:** clang on the CI runner image (low risk — `ubuntu-latest` ships clang by default; only a concern if a future minimal image is substituted).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` (stdlib, no testify — confirmed via `go.mod` and existing `*_test.go` files) for Go-layer tests; Rust `cargo test` (existing `tests/rust_shim_*.rs` integration-test convention) for Rust/C++-layer tests |
| Config file | none — both frameworks use zero-config convention-based discovery |
| Quick run command | `go test ./... -run TestAtPointer\|TestAtPath\|TestMinify\|TestValidateUTF8` (Go); `cargo test --test rust_shim_navigation` (new Rust file) |
| Full suite command | `make verify-contract && go test ./... -race` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| DOM-01 | `AtPointer` RFC 6901 conformance (escaping, array index, empty pointer, missing field, wrong type, out-of-range) | unit | `go test -run TestElement_AtPointer ./...` | ❌ Wave 0 |
| DOM-02 | `AtPath` dot/index subset (leading `.`/`[`/`$`, bracket-key non-quote-awareness, trailing separator) | unit | `go test -run TestElement_AtPath ./...` | ❌ Wave 0 |
| DOM-03 | `AtPathAll` wildcard zero/one/many matches, ordering, non-error empty result | unit | `go test -run TestElement_AtPathAll ./...` | ❌ Wave 0 |
| DOM-03 | Bulk-array Rust ownership: allocate N views, free correctly, reject double-free/mismatched-length free | Rust integration | `cargo test --test rust_shim_navigation` | ❌ Wave 0 |
| DOM-04 | `Array.At` valid/out-of-range/wrong-type; `Array.Len`/`Object.Size` including saturation-documented behavior | unit | `go test -run TestArray_At\|TestArray_Len\|TestObject_Size ./...` | ❌ Wave 0 |
| UTIL-01 | `Minify`/`MinifyInto`: overlap (`dst==src`), empty input, malformed (unclosed string) input, undersized `dst` | unit + Rust contract | `go test -run TestMinify ./...`; `cargo test --test rust_shim_minify` | ❌ Wave 0 |
| UTIL-01 | Undersized `dst_cap` pre-check at the C++ bridge boundary (must reject before calling `simdjson::minify`) | C++/Rust contract | `cargo test --test rust_shim_minify -- undersized` | ❌ Wave 0 |
| UTIL-02 | `ValidateUTF8` valid/invalid UTF-8, empty input | unit | `go test -run TestValidateUTF8 ./...` | ❌ Wave 0 |
| D-14 | `dst==src` aliasing safety on x86-64 (haswell/westmere/fallback kernels) | CI-matrix (ASan/UBSan) | `bash .planning/spikes/004-minify-buffer-safety/verify.sh` invoked as a new step in `.github/workflows/phase2-rust-shim-smoke.yml`'s `linux-smoke` job | ❌ Wave 0 (CI wiring) |

### Sampling Rate
- **Per task commit:** `go test ./... -run <focused-pattern>` and `cargo check` (fast feedback, seconds).
- **Per wave merge:** `make verify-contract` (includes `cargo test -- --test-threads=1`, header diff check, `check_header.py` all rules including `required-symbols`) + `go test ./... -race`.
- **Phase gate:** Full suite green (`make verify-contract`, `go test ./... -race`, `make phase2-smoke-linux`) plus the new D-14 x86-64 ASan step green in CI before `/gsd:verify-work`.

### Wave 0 Gaps
- [ ] `internal/ffi/types_test.go` or equivalent — extend the existing offset-by-offset ABI layout check for any new/changed struct (none expected; `pure_simdjson_value_view_t` stays 32 bytes unchanged, only new *functions* are added).
- [ ] New Rust integration test file (`tests/rust_shim_navigation.rs`) — covers `element_at_pointer`, `element_at_path`, `element_at_path_wildcard`, `array_at`, `array_size`, `object_size` at the registry/FFI layer, following the existing `tests/rust_shim_accessors.rs`/`tests/rust_shim_iterators.rs` naming and structure.
- [ ] New Rust integration test file (`tests/rust_shim_minify.rs`) — overlap, empty, malformed, undersized-dst cases at the FFI layer (Go-level tests alone cannot exercise the `dst_cap` pre-check's C++/Rust boundary in isolation from Go's own slice-length invariants).
- [ ] `tests/abi/check_header.py`'s `REQUIRED_SYMBOLS` tuple — must gain the 8 new exported names or `make verify-contract` fails closed (Pitfall 9).
- [ ] `.github/workflows/phase2-rust-shim-smoke.yml` — new step in `linux-smoke` invoking the D-14 x86-64 probe.
- [ ] `docs/ffi-contract.md` — new error codes (11/12), ABI 1.3 section update, `pure_simdjson_value_views_free` ownership note in a new "Worked call sequences" subsection (mirroring the existing "String copy and free" worked example).

## Security Domain

`security_enforcement` is not set in `.planning/config.json`; treated as enabled per policy. This is a native-FFI parsing library with no network/auth/session surface — most ASVS categories do not apply. The relevant categories are input validation (untrusted JSON/path-string handling) and the memory-safety-equivalent of "cryptography" (buffer bounds enforcement at a hard FFI boundary).

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | No auth surface in this library |
| V3 Session Management | No | No session concept |
| V4 Access Control | No | No access-control concept; all data is process-local caller-supplied bytes |
| V5 Input Validation | Yes | Untrusted path/pointer strings and untrusted JSON bytes must be rejected via typed errors (`ErrInvalidPath`, `ErrIndexOutOfRange`, `ErrInvalidJSON`), never via panics or silent truncation; every new bridge function follows the existing `try {} PSIMDJSON_CATCH_CPP_EXCEPTIONS(...)` + `ffi_wrap`/`catch_unwind` double containment already mandated project-wide (`docs/ffi-contract.md`) |
| V6 Cryptography | No (not literally applicable) | The closest analogue is memory-safety at the buffer-write boundary (`MinifyInto`'s `dst_cap` check) — never hand-roll bounds checking; reuse the existing `copy_bytes` pattern (Pattern 2 above) rather than inventing new bounds-check logic per call site |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Undersized `dst` buffer passed to `MinifyInto` (heap buffer overflow, since upstream `simdjson::minify` has no self-defense) | Tampering / Denial of Service | Mandatory `dst_cap >= src_len` pre-check in the C++ bridge before calling into vendored `minify` (Pitfall 5) |
| Adversarial deeply-nested or huge-fanout JSON fed to `AtPathAll`, producing an unexpectedly large `Vec<ValueView>` allocation | Denial of Service (resource exhaustion) | No new mitigation beyond the existing depth-limit (`ErrDepthLimitExceeded`, already enforced at parse time by Phase 11's configured max-depth) and the existing capacity-limit; wildcard match count is bounded by the number of nodes already accepted at parse time, no unbounded amplification |
| Malformed/hostile `AtPointer`/`AtPath` strings (e.g. deeply repeated escape sequences, adversarial bracket nesting) | Denial of Service | Upstream's own string-based parsing is linear in path length, not exponential; no new amplification vector introduced by this phase |
| Stale/forged `Element` handle passed into a new navigation method | Spoofing / Tampering | Already covered by the existing generation-checked handle + `descendant_indices` membership validation in `with_resolved_view`; every new accessor reuses this unmodified |

## Upstream Wildcard Semantics (spike 005, verdict PARTIAL)

Executable 35-case truth table pinned at
`.planning/spikes/005-wildcard-path-semantics/expected.txt`, regenerated and defended by
`.planning/spikes/005-wildcard-path-semantics/verify.sh`. **Reuse it directly as the fixture
table for 12-03 and 12-06 rather than inventing cases.** Run against vendored simdjson v4.6.4
with ASan+UBSan, deterministic across 3 runs.

**Core finding — `at_path_with_wildcard` selects its error regime by substring-testing the path
for `*`, not by document content:**

| Path | Document | Result |
|---|---|---|
| `.z.b` | `{"a":{"b":1}}` | `NO_SUCH_FIELD(20)` |
| `.z.*` | `{"a":{"b":1}}` | `SUCCESS`, 0 results |
| `.z.*.b` | `{"a":{"b":1}}` | `SUCCESS`, 0 results |

A `*` anywhere in the path — including *after* the failing segment — converts a hard error into a
silent empty result. This is what refuted the original D-02 and drives the amended contract above.

**Behaviors the plans must account for:**

1. **Scalar/string receivers never error.** Root `42` with `.a` or `.*` → `SUCCESS`, 0 results.
   Plans 12-03 and 12-06 currently expect `ErrWrongType` here — that assertion will fail.
2. **With a wildcard present, misses are silently dropped.** `.a.*.b` on
   `{"a":{"x":{"b":1},"y":{"c":2}}}` → 1 result, not an error. Same for non-container branches
   (`{"a":{"x":{"b":1},"y":5}}` → 1 result).
3. **`.*` and `[*]` are interchangeable aliases, neither type-checked.** `[*]` on an object
   returns its values (`.a[*]` on `{"a":{"b":1}}` → `[1]`); `.*` on an array returns its elements
   (`.a.*` on `{"a":[10,20]}` → `[10,20]`). Callers will assume bracket-star implies an array —
   it does not. Must be stated in the `AtPathAll` doc comment.
4. **Trailing dot is an empty-key lookup, not a syntax error.** `.a.` → `NO_SUCH_FIELD(20)`.
   This independently confirms that `AtPointer("/a/")` on `{"a":1}` yields `ErrElementNotFound`,
   **not** `ErrWrongType` — 12-06's planned assertion is wrong and must be corrected.
5. **`[0]` on an object is `NO_SUCH_FIELD`, not `INCORRECT_TYPE`** — it degrades to a lookup of
   key `"0"`.
6. **Ordering is document order**, confirmed for flat, nested, and array-of-object wildcards.
   D-02's ordering guarantee holds.
7. **Grammar is identical between `at_path` and `at_path_with_wildcard`** — all malformed inputs
   return `INVALID_JSON_POINTER(22)` in both, so a single `ErrInvalidPath` mapping is correct for
   both APIs. Exact confirmed-rejected strings for tests: `a.b`, `*`, `.a[0`, `""`.
8. **`ErrWrongType` (`INCORRECT_TYPE(17)`) is still reachable via `AtPath`** — e.g. `.a.b` on
   `{"a":[10,20]}`. D-01 and D-03 remain valid for `AtPointer` / `AtPath` / `Array.At`; only
   `AtPathAll`'s surface is narrowed.

## Sources

### Primary (HIGH confidence — direct reading of pinned vendored source and this repository's own code, git commit `1bcf71bd85059ab6574ea1159de9298dcc1212c5`, simdjson v4.6.4)
- `third_party/simdjson/include/simdjson/dom/element-inl.h` (lines 129-461) — `at_pointer`, `at_path`, `at_path_with_wildcard`, `is_pointer_well_formed`, deprecated `at()`
- `third_party/simdjson/include/simdjson/dom/array-inl.h` (lines 44-245) — `array::at_pointer`, `array::at_path`, `array::at_path_with_wildcard`, `array::at(size_t)` O(n) scan, `array::size()`
- `third_party/simdjson/include/simdjson/dom/object-inl.h` (lines 34-244) — `object::at_pointer` escaping logic, `object::at_path`, `object::at_path_with_wildcard`, `object::size()`
- `third_party/simdjson/include/simdjson/jsonpathutil.h` — `json_path_to_pointer_conversion` (used by plain `at_path`), `get_next_key_and_json_path` (used only by wildcard) — the exact grammar asymmetry behind Pitfall 1
- `third_party/simdjson/include/simdjson/internal/tape_ref-inl.h` (lines 14, 80-82) — `JSON_COUNT_MASK = 0xFFFFFF`, `scope_count()` saturation
- `third_party/simdjson/include/simdjson/dom/document-inl.h` (lines 121-136) — upstream's own "saturated count" tape-dump label, corroborating the saturation finding
- `third_party/simdjson/include/simdjson/minify.h` — `simdjson::minify` signature, no `dst_cap` parameter
- `third_party/simdjson/src/generic/stage1/json_minifier.h` — actual minify implementation, confirms no bounds check against any capacity
- `third_party/simdjson/src/generic/stage1/json_scanner.h` (line 111) — confirms `UNCLOSED_STRING`/`SUCCESS` are the only two possible minify error outcomes
- `third_party/simdjson/include/simdjson/implementation.h`, `third_party/simdjson/src/implementation.cpp` — `validate_utf8` free-function signature and dispatch through `get_active_implementation()`
- `third_party/simdjson/fuzz/fuzz_minifyimpl.cpp` — confirms upstream's own fuzz harness never exercises `dst == src` aliasing
- `element.go`, `errors.go`, `doc.go`, `library_loading.go`, `kernel.go`, `parser.go` — existing Go public API, error-mapping, and `activeLibrary()`-first patterns
- `internal/ffi/types.go`, `internal/ffi/bindings.go` — existing ABI constants and purego symbol-registration patterns (required vs. optional)
- `src/lib.rs`, `src/runtime/mod.rs`, `src/runtime/registry.rs` — existing `ffi_wrap`, handle registry, `with_resolved_view`/`encode_descendant_view_locked`, `object_get_field`, `materialize_build` patterns
- `src/native/simdjson_bridge.h`, `src/native/simdjson_bridge.cpp` — existing bridge signatures, `map_error()` (confirms `INVALID_JSON_POINTER`/`INDEX_OUT_OF_BOUNDS` currently fall through to `PURE_SIMDJSON_ERR_INTERNAL`), `copy_bytes` pattern, `object_get_field_index`, `psimdjson_doc`'s existing borrowed-scratch-buffer discipline
- `docs/ffi-contract.md` — normative ABI contract, confirms reserved error-code bands (11-31), 32-byte `pure_simdjson_value_view_t` pin, ABI-version-handshake rules
- `tests/abi/check_header.py` — confirms the closed `REQUIRED_SYMBOLS` allowlist that must be updated for new exports to pass `make verify-contract`
- `cbindgen.toml`, `Makefile`, `build.rs` — header regeneration mechanics, `SIMDJSON_IMPLEMENTATION_FALLBACK=1` build flag
- `.github/workflows/phase2-rust-shim-smoke.yml` — existing `linux-smoke`/`windows-smoke`/`darwin-build-only` jobs, confirms `ubuntu-latest` (x86-64) already runs Rust tests, the natural home for D-14
- `.planning/spikes/004-minify-buffer-safety/probe.cpp`, `verify.sh`, `.planning/spikes/CONVENTIONS.md` — D-13/D-14 spike evidence and promotion rules
- `.planning/phases/12-high-value-dom-navigation-and-simd-utility-apis/12-CONTEXT.md`, `.planning/REQUIREMENTS.md`, `.planning/ROADMAP.md`, `.planning/STATE.md`, `.planning/config.json` — phase scope, requirements, and workflow configuration

### Secondary (MEDIUM confidence)
None used — all findings were verifiable directly against the pinned vendored source or this repository's own committed code; no web search was needed or performed (`brave_search`/`exa_search`/`firecrawl` are all `false` in `.planning/config.json`, and the domain is fully covered by local, authoritative source).

### Tertiary (LOW confidence)
None.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependencies, all versions confirmed via `go.mod`/`Cargo.toml`/local tool checks
- Architecture: HIGH — every proposed new function/struct is modeled directly on an existing, working precedent in this exact codebase
- Pitfalls: HIGH — every pitfall was independently verified by reading the exact pinned vendored source (not training-data recollection of simdjson's general API), including several pitfalls (bracket-key quote-handling, minify's missing `dst_cap`, `check_header.py`'s closed allowlist) that are non-obvious enough to not be findable without direct source inspection

**Research date:** 2026-07-30
**Valid until:** Stable for the life of the vendored simdjson v4.6.4 pin; re-verify the `at_path`/`at_pointer`/`minify` source excerpts above if `third_party/simdjson` is ever bumped past this commit before Phase 12 implementation lands.
