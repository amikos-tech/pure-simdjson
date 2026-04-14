# Architecture Research

**Domain:** Three-layer SIMD JSON parser — Go (purego) → Rust FFI shim → C++ simdjson
**Researched:** 2026-04-14
**Confidence:** HIGH for layering and FFI pattern (verbatim reuse of pure-tokenizers); MEDIUM for simdjson-specific choices (verified against simdjson docs but not yet prototyped)

## Standard Architecture

### System Overview

```
┌─────────────────────────────────────────────────────────────────┐
│  Layer 1 — Go Public API (package purejson)                     │
│                                                                 │
│  ┌─────────┐  ┌──────┐  ┌─────────┐  ┌──────────┐  ┌────────┐   │
│  │ Parser  │  │ Doc  │  │ Element │  │ Array    │  │ Object │   │
│  └────┬────┘  └──┬───┘  └────┬────┘  └────┬─────┘  └───┬────┘   │
│       │         │            │             │           │        │
│       └─────────┴────────────┴─────────────┴───────────┘        │
│                          │                                      │
│  ┌───────────────────────┴───────────────────────────┐          │
│  │ internal/ffi: purego.RegisterLibFunc bindings,    │          │
│  │                   C-struct layout mirrors         │          │
│  │                   sync.RWMutex lifecycle          │          │
│  └───────────────────────┬───────────────────────────┘          │
│                          │                                      │
│  ┌───────────────────────┴───────────────────────────┐          │
│  │ internal/bootstrap: download+verify+cache         │          │
│  │                     R2/GitHub fallback, semver    │          │
│  └───────────────────────┬───────────────────────────┘          │
├──────────────────────────┼──────────────────────────────────────┤
│                          │  C ABI (extern "C", #[repr(C)])      │
├──────────────────────────┼──────────────────────────────────────┤
│  Layer 2 — Rust FFI Shim (crate pure_simdjson, cdylib)          │
│                                                                 │
│  ┌───────────────────────┴────────────────────────────┐         │
│  │ lib.rs: #[no_mangle] exports                       │         │
│  │   parser_new / parser_free                         │         │
│  │   parser_parse(bytes, len) → doc handle            │         │
│  │   doc_root / doc_free                              │         │
│  │   element_type / element_get_* / array_iter / ... │         │
│  │   get_version / get_abi_version                   │         │
│  └───────────────┬───────────────┬────────────────────┘         │
│                  │               │                              │
│         catch_unwind()   Box<Parser> / Box<OwnedDoc>            │
│                  │               │                              │
│  ┌───────────────┴───────────────┴────────────────────┐         │
│  │ cxx / bindgen-generated simdjson bindings          │         │
│  └───────────────┬────────────────────────────────────┘         │
├──────────────────┼──────────────────────────────────────────────┤
│                  │  C++ ABI (unstable, name-mangled)            │
├──────────────────┼──────────────────────────────────────────────┤
│  Layer 3 — C++ simdjson (vendored submodule or fetchcontent)    │
│                                                                 │
│  ┌───────────────┴────────────────────────────────────┐         │
│  │ simdjson::dom::parser  (reusable, owns tape)       │         │
│  │ simdjson::dom::element (borrows from parser tape)  │         │
│  │ runtime kernel dispatch: haswell/westmere/arm64/…  │         │
│  └────────────────────────────────────────────────────┘         │
└─────────────────────────────────────────────────────────────────┘
```

### Component Responsibilities

| Component | Responsibility | Implementation |
|-----------|----------------|----------------|
| `Parser` (Go) | User-facing handle; owns shared-library handle + native `simdjson::dom::parser`; guards lifecycle with `sync.RWMutex` | Thin Go struct wrapping opaque `unsafe.Pointer` |
| `Doc` (Go) | Per-parse result handle; owns tape lifetime; yields a root `Element` | Struct holding `unsafe.Pointer` + back-ref to `Parser` for mutex |
| `Element` (Go) | Stack-allocated view over a single tape node; **never owns** native memory | `struct { doc *Doc; node uintptr }` — cheap, copyable, lifetime = `Doc` |
| `Array`, `Object` (Go) | Typed Element wrappers; iteration API | Structs wrapping Element + cursor state |
| `internal/ffi` | purego `RegisterLibFunc` bindings; `#[repr(C)]` struct mirrors; error-code → error translation | Go; mirrors `pure-tokenizers/tokenizers.go` |
| `internal/bootstrap` | Download shared lib from R2 with GitHub fallback; sha256 verify; cache in OS cache dir; ABI semver check | Go; adapted from `pure-tokenizers/download.go` + `library_loading.go` |
| `lib.rs` (Rust) | `#[no_mangle] extern "C"` exports; opaque-pointer boxing; `catch_unwind` panic shield; error-code returns | Rust cdylib |
| `build.rs` (Rust) | Drive simdjson C++ build via `cc` or `cmake` crate; cbindgen header generation | Rust build script |
| simdjson C++ | SIMD JSON parsing; tape production; runtime kernel dispatch | Vendored as submodule at `third_party/simdjson` |

### Layer deviation note vs pure-onnx

`pure-onnx` binds ONNX Runtime **directly** from Go via purego — it does not use a Rust shim because onnxruntime ships a stable C ABI (`onnxruntime_c_api.h`). simdjson has **no stable C ABI**; its public API is templated C++. We therefore follow the **`pure-tokenizers` + `fast-distance` pattern**: Rust cdylib wraps the C++/Rust core and exposes a hand-written C ABI. This is the correct precedent.

## Recommended Project Structure

```
pure-simdjson/
├── Cargo.toml                    # [lib] crate-type=["cdylib","staticlib"]
├── build.rs                      # cbindgen + simdjson C++ build driver
├── cbindgen.toml                 # C header generation config
├── third_party/
│   └── simdjson/                 # git submodule pinned to vX.Y.Z
├── src/                          # Rust FFI shim
│   ├── lib.rs                    # #[no_mangle] extern "C" exports
│   ├── handles.rs                # Opaque handle types + catch_unwind helpers
│   ├── errors.rs                 # Error-code constants (mirror pure-tokenizers)
│   └── ffi/
│       ├── parser.rs             # parser_new/parse/free
│       ├── element.rs            # element_type/get_*/iter
│       └── simdjson_sys.rs       # cxx or bindgen-generated bindings
├── include/
│   └── pure_simdjson.h           # Generated by cbindgen (committed for CI sanity)
│
├── go.mod                        # github.com/amikos-tech/pure-simdjson
├── purejson.go                   # Public API: Parser, Doc, Element types
├── parser.go                     # Parser lifecycle + RegisterLibFunc
├── doc.go                        # Doc lifecycle + root element
├── element.go                    # Typed accessors (GetInt64, GetString, ...)
├── array.go                      # Array iteration
├── object.go                     # Object field access
├── errors.go                     # Error-code → error mapping
├── library.go                    # purego.Dlopen wrapper (non-windows)
├── library_windows.go            # purego LoadLibrary wrapper (windows)
├── library_loading.go            # LoadLibrary orchestration (env → cache → download)
├── download.go                   # R2 + GitHub fallback, sha256 verify
├── abi.go                        # Semver ABI compatibility check
├── internal/
│   └── cbind/                    # Generated or hand-written purego struct mirrors
│       └── types.go              #   #[repr(C)]  →  Go struct
│
├── examples/
│   ├── basic/main.go             # parse twitter.json → walk keys
│   └── benchmark/main.go         # vs encoding/json, minio/simdjson-go
│
├── scripts/
│   ├── build-local.sh            # Local Rust build for dev
│   ├── prepare-release.sh
│   └── build_releases_index.sh
│
├── .github/
│   ├── workflows/
│   │   ├── rust-ci.yml           # Cargo test on all 6 targets
│   │   ├── go-ci.yml             # Go test against pre-built local lib
│   │   ├── rust-release.yml      # Cross-build cdylibs → R2 + GH releases
│   │   └── benchmark.yml
│   └── actions/
│       ├── build-rust-library/
│       ├── get-rust-library/
│       └── setup-cross-compilation/
│
├── .planning/                    # GSD
├── abi_version.json              # Current ABI contract snapshot
├── Makefile
└── README.md
```

### Structure Rationale

- **Flat Go package at root** — matches pure-tokenizers/fast-distance. A single `purejson` package keeps imports simple (`purejson.Parser`, not `purejson/parser.Parser`). Subpackages only for truly internal concerns (`internal/cbind`).
- **`internal/cbind/types.go`** — consolidates `#[repr(C)]` mirror structs in one place. pure-tokenizers scatters these across `tokenizers.go`; treat it as a lesson learned.
- **`third_party/simdjson` submodule, not package manager** — pure-onnx downloads a pre-built onnxruntime binary; simdjson has no equivalent pre-built distribution. vcpkg/Conan add a build-time dependency we don't want. Submodule pinned to a known tag is simplest and matches simdjson's own recommendation for embedders.
- **Generated header committed** — cbindgen regenerates `include/pure_simdjson.h` on every build, but committing it lets CI diff verify ABI hasn't silently drifted.
- **`library_windows.go` split** — purego's `Dlopen` vs `LoadLibrary` differ enough that a build-tagged file is cleaner than `runtime.GOOS` switching. Verbatim reuse of pure-tokenizers.

## Architectural Patterns

### Pattern 1: Opaque-Pointer Handles Across FFI

**What:** Every Rust-owned object (`Parser`, `Doc`) crosses the ABI as `*mut c_void` — an opaque pointer Go stores as `unsafe.Pointer`. Go never dereferences it; only passes it back to Rust functions.

**When to use:** Any object with a lifetime that must outlive a single FFI call. Default for our three main handle types.

**Trade-offs:**
- **+** No struct-layout synchronization between Rust and Go
- **+** Rust can freely evolve internals without ABI break
- **+** Clean Drop semantics on Rust side via `Box::from_raw`
- **−** Each handle costs one heap allocation
- **−** Go can't inspect the object; all access goes through FFI calls

**Example (mirrors `pure-tokenizers/src/lib.rs:154`):**
```rust
#[repr(C)]
pub struct ParserResult {
    parser: *mut Parser,   // opaque
    error_code: i32,
}

#[no_mangle]
pub unsafe extern "C" fn parser_new(out: *mut ParserResult) -> i32 {
    let p = Box::new(simdjson_bridge::Parser::new());
    (*out).parser = Box::into_raw(p) as *mut _;
    (*out).error_code = 0;
    0
}

#[no_mangle]
pub unsafe extern "C" fn parser_free(p: *mut Parser) {
    if !p.is_null() { drop(Box::from_raw(p)); }
}
```

### Pattern 2: Error-Code Return + Out-Param Result

**What:** Every fallible FFI function returns `i32` (0 = success, negative = error) and writes output to a caller-provided `*mut OutStruct`. No exceptions, no thread-local errno.

**When to use:** Universal — applied to every FFI boundary function.

**Trade-offs:**
- **+** Simple, unambiguous, no hidden state
- **+** Works identically on all platforms
- **+** Easy to translate to idiomatic Go errors at the wrapper
- **−** More verbose call sites vs Result-style
- **−** Error payload is fixed (code + optional out-message pointer), not rich

**Example (verbatim from `pure-tokenizers/tokenizers.go:42`):**
```go
const (
    SUCCESS            = 0
    ErrParseFailed     = -1
    ErrInvalidHandle   = -2
    ErrInvalidUTF8     = -3
    ErrOutOfMemory     = -4
    ErrTypeMismatch    = -5  // new: Element wrong JSON type
    // ...
)

func (p *Parser) Parse(data []byte) (*Doc, error) {
    var r DocResult
    rc := p.parse(p.handle, data, uint32(len(data)), &r)
    if rc != SUCCESS {
        return nil, mapError(rc)
    }
    return &Doc{parent: p, handle: r.doc}, nil
}
```

### Pattern 3: RWMutex-Guarded Handle Lifecycle

**What:** Every handle wraps a `sync.RWMutex` + `closed bool`. Every operation takes `RLock` + checks `closed`; `Close()` takes `Lock`, sets `closed`, zeros out function pointers, frees handle.

**When to use:** Universal for any Go object that owns an FFI handle.

**Trade-offs:**
- **+** Close is race-safe against concurrent reads
- **+** Double-Close is cheap and correct
- **+** Clear panic location if someone Uses-after-Close
- **−** RLock/RUnlock overhead per call (negligible vs FFI cost)

**Example (verbatim `pure-tokenizers/tokenizers.go:393`):**
```go
func (p *Parser) beginOperation() (func(), error) {
    p.lifecycleMu.RLock()
    if p.closed {
        p.lifecycleMu.RUnlock()
        return nil, ErrParserClosed
    }
    return p.lifecycleMu.RUnlock, nil
}
```

### Pattern 4: Element as Stack-Allocated View (simdjson-specific)

**What:** `Element` is NOT a heap-allocated FFI handle. It's a tiny Go struct holding `{doc *Doc; tape_node uintptr}` — where `tape_node` is an index into the parser's tape, passed as an integer across FFI.

**When to use:** For anything whose lifetime is strictly ≤ Doc's lifetime — i.e., every tape-derived value. Applies to `Element`, `Array`, `Object`, and returned string slices.

**Trade-offs:**
- **+** Zero-allocation Element traversal (critical for parse-heavy workloads)
- **+** Matches simdjson's own "borrowed view" model
- **+** Compiles to what's basically a uint64 + pointer — register-sized
- **−** If user stores an Element past `Doc.Close()`, every method returns error (not a segfault, because we check `doc.closed`)
- **−** Go doesn't enforce the lifetime — user must not outlive Doc

**Example:**
```go
type Element struct {
    doc  *Doc
    node uint64   // tape index; opaque to Go
}

func (e Element) GetInt64() (int64, error) {
    unlock, err := e.doc.beginOperation()
    if err != nil { return 0, err }
    defer unlock()
    var v int64
    rc := e.doc.parent.elementGetInt64(e.doc.handle, e.node, &v)
    if rc != SUCCESS { return 0, mapError(rc) }
    return v, nil
}
```

This is a **deliberate deviation** from `pure-tokenizers`, whose returned `EncodeResult` copies all bytes out of Rust memory into Go slices. For simdjson, every parse produces a tree — copying the whole tree into Go defeats the performance story. We must return tape-bound views.

### Pattern 5: Vec-into-Raw for Owned Output Buffers

**What:** When Rust must return an array of values that outlive the FFI call (e.g., a decoded string the user holds for a while), it builds `Vec<u8>`, calls `shrink_to_fit()`, gets pointer+len, then `mem::forget`s the Vec. A matching `free_*` function reconstructs the Vec and drops it.

**When to use:** For **copy-out** paths only — decoded strings the user asked for by value. The zero-copy string view path uses tape-bound views instead (Pattern 4).

**Trade-offs:**
- **+** Clear ownership: Rust allocates, Rust frees (matching allocator)
- **−** Must never cross allocators — Go mustn't `free()` a Rust Vec pointer

**Example (verbatim `pure-tokenizers/src/lib.rs:248`):**
```rust
let mut vec = encoding.get_ids().to_vec();
vec.shrink_to_fit();
let ptr = vec.as_mut_ptr();
let len = vec.len();
std::mem::forget(vec);
// ptr/len written into out struct; freed later by free_buffer
```

### Pattern 6: Per-Operation Catch-Unwind Shield

**What:** Every `#[no_mangle] extern "C" fn` wraps its body in `std::panic::catch_unwind`. Unwinding into a C stack is UB; this returns an error code instead.

**When to use:** Mandatory for every exported function. `pure-tokenizers` is inconsistent here; we should be stricter.

**Example:**
```rust
#[no_mangle]
pub unsafe extern "C" fn parser_parse(...) -> i32 {
    std::panic::catch_unwind(|| {
        // real work
    }).unwrap_or(ERROR_PANIC)
}
```

### Pattern 7: ABI Semver Gate at Library Load

**What:** On `Parser` creation, call `get_abi_version()` from the loaded .so; compare against compile-time constraint `^0.1.x`. Refuse to proceed if mismatch.

**When to use:** Mandatory once multiple shared-lib versions exist in the wild.

**Example (verbatim `pure-tokenizers/tokenizers.go:362`):**
```go
const AbiCompatibilityConstraint = "^0.1.x"

func (p *Parser) abiCheck(c *semver.Constraints) error {
    v, err := semver.NewVersion(p.getVersion())
    if err != nil { return err }
    if !c.Check(v) { return ErrAbiMismatch }
    return nil
}
```

## Data Flow

### Hot Path: `parser.Parse(data)` End-to-End

```
Go:   data []byte                                           // zero-cost
 │
 │    purejson.(*Parser).Parse(data)
 │      beginOperation()  → RLock + closed check            // ~20ns
 │
 ▼
Go/FFI boundary (purego) — direct syscall register marshaling
 │    parser_parse(ctx, *u8, u32 len, *DocResult) → i32
 │    (purego avoids cgo runtime; close to ~50-200ns per call)
 │
 ▼
Rust:  parser_parse()
 │       catch_unwind { ... }                               // ~0 (no unwind)
 │       slice::from_raw_parts(ptr, len) → &[u8]
 │       SIMDJSON_PADDING check — pad if caller didn't       ⚠️ see pitfall
 │
 ▼
C++:   simdjson::dom::parser::parse(data)
 │       runtime-dispatched kernel (haswell/arm64/...)
 │       produces internal TAPE — binary tree encoding
 │       Returns simdjson::dom::element (borrowed view)     // THE HOT WORK
 │
 ▲
Rust:  Box::new(OwnedDoc { element, parser_ref }) → ptr
 │       Write ptr into DocResult struct
 │
 ▲
Go/FFI boundary — return i32 error code
 │
 ▲
Go:    Wrap in Doc{handle: ptr, parent: p}                 // 1 small alloc
       unlock()                                              // ~20ns
       return doc
```

**Cost centers (descending):**
1. **C++ simdjson parse** — the actual SIMD work; dominates at multi-GB/s
2. **Go → Rust FFI call** (~50-200ns per call via purego) — **matters for element-level access, not for Parse**; hence we must NOT make an FFI call per tape node
3. **Padding copy** if caller gave unpadded input — up to one memcpy of input size
4. **Box allocation** for the returned handle — single small alloc

### Visitor/Iterator Choice — Pull-Based Go-Driven

**Option A — C++ calls Go callback (REJECTED):**
Would require marshaling a Go function pointer across FFI, which purego supports but with significant overhead (~1μs per call). For a twitter.json with ~5000 tape nodes, that's 5ms of callback overhead alone — defeats simdjson's perf story.

**Option B — Go drives iteration, Rust yields one tape step at a time (CHOSEN):**
Go calls `element_next_child(doc, parent_node) → (child_node, type)`. Each call is one FFI round-trip. For object iteration, Go calls `object_iter_next(doc, iter_state) → (key_ptr, value_node)`.

**Option C — Bulk extract a path set (v0.2 On-Demand API):**
For `v0.2`, compile a pre-declared path set in Rust; single FFI call yields all matching values as a Vec → pointer + length. This is where the 10-100× selective-extract win lives.

**v0.1 locks in Option B. v0.2 adds Option C as a fast-path overlay.**

### Handle Lifecycle

```
Parser                Doc                   Element
  │                    │                       │
  ▼                    │                       │
 New()                 │                       │
  │─heap─▶ Box<Parser> │                       │
  │                    ▼                       │
  │                 Parse(b)                   │
  │       ──────────▶  │                       │
  │                    │─heap─▶ Box<Doc>       │
  │                    │         │             ▼
  │                    │         │         Root()
  │                    │         │         (stack view)
  │                    │         │             │
  │                    │         │         GetField("x")
  │                    │         │           ──FFI──▶ Rust walks tape
  │                    │         │                    returns tape_idx
  │                    │         │             │
  │                    │         │         GetInt64()
  │                    │         │           ──FFI──▶ reads tape node
  │                    │         │                    returns int64
  │                    │         │             │
  │                    │        Close()       │
  │                    │         │             │
  │                    │    Box::from_raw      │ ← Element now stale
  │                    │                      │   (doc.closed detected
  │                    │                      │    on next method call)
  │                    │                      │
  │                  Close()                  │
  │           (no-op, already closed)         │
  │                                           │
 Close()                                      │
  │                                           │
 Box::from_raw(parser) — Drops simdjson::dom::parser
 purego.Dlclose(libh)
```

**Invariants:**
1. `Doc` outlives all `Element`s derived from it (Go can't enforce this statically; we detect-and-error dynamically via `doc.closed`).
2. `Parser` outlives all `Doc`s (Close'ing the Parser while Docs are open panics at FFI layer — we enforce this with a parent ref count in v0.2 if needed; v0.1 documents the rule).
3. Calling `Parser.Parse()` a second time **invalidates the previous Doc** per simdjson's one-doc-per-parser invariant. **Therefore: a single Parser supports at most one live Doc at a time.**

### The Per-Parser Single-Doc Invariant (simdjson-specific)

> "A parser may have at most one document open at a time, since it holds allocated memory used for the parsing." — simdjson docs

**Consequence for the API:** If a user wants to parse concurrently from multiple goroutines, **each goroutine needs its own Parser**. This is the same rule simdjson's C++ API enforces.

**Recommended API shape:**
- `Parser` is NOT safe for concurrent `Parse()` calls. Document this prominently.
- Provide `ParserPool` utility that uses `sync.Pool` to amortize Parser allocation across goroutines.
- Every `Parse()` returns a Doc; calling `Parse()` again on the same Parser before `Close()`-ing the previous Doc must return `ErrParserBusy` (not silently invalidate).

**Why not just auto-serialize inside Parser with a Mutex?** Because that hides a latent serial bottleneck that looks like a perf bug from the outside. Explicit is better; the pool pattern solves real concurrent use.

### Finalizer Strategy

- **Primary:** Explicit `Close()`. All examples and docs teach `defer p.Close()`.
- **Safety net:** `runtime.SetFinalizer` on `Parser` and `Doc` that logs a warning ("leaked — Close() was not called") and frees the handle. This catches bugs in long-lived processes without silently leaking native memory.

`pure-onnx` uses finalizers extensively (see `ort/finalizer_log.go`); `pure-tokenizers` does not. For pure-simdjson, we want the safety net because Doc/Parser leaks are easy to introduce in tight parse loops where error paths forget `defer`. **Adopt finalizer safety net — lesson from pure-onnx.**

## Scaling Considerations

| Scale | Architecture Adjustments |
|-------|--------------------------|
| Single-doc parse | No adjustments; v0.1 API is sufficient |
| Parse-per-request web servers | Recommend `ParserPool` (ship as part of v0.1 API) |
| NDJSON high-throughput (≥1 GB/s) | v0.2 parallel streaming parse using simdjson's built-in NDJSON support |
| 10k+ concurrent parses | ParserPool with sized `MaxParsers` cap; each Parser holds ~1MB of tape memory |

### Scaling Priorities

1. **First bottleneck:** FFI call count per tape node. **Mitigation:** batch accessors — e.g., `GetStringField(name) → string` in one FFI call rather than `GetField(name).GetString()` in two. Apply systematically for v0.1's hot paths.
2. **Second bottleneck:** Goroutine contention on a shared Parser. **Mitigation:** `ParserPool` in v0.1.
3. **Third bottleneck (v0.2):** Whole-tape materialization when user only needs 3 fields. **Mitigation:** On-Demand path-set API.

## Anti-Patterns

### Anti-Pattern 1: Materializing the Tree as `map[string]any`

**What people do:** "Port" the `encoding/json → any` pattern to this library — parse, then traverse the tree calling `GetString`, `GetArray`, recursively, building a `map[string]any` to pass around.
**Why it's wrong:** Defeats the entire performance story. Each tape node becomes a Go map entry + heap alloc + interface box. On a 5000-node doc, that's ~300μs of Go allocation vs ~5μs of actual parse work.
**Do this instead:** Extract specifically the fields the application needs via typed accessors; for v0.2, pre-declare the path set.

### Anti-Pattern 2: Shared Parser Across Goroutines

**What people do:** Create one global `Parser`; call `Parse` from many goroutines.
**Why it's wrong:** simdjson's Parser is single-threaded by design. Sharing requires serializing all parses through a mutex — silent perf bottleneck.
**Do this instead:** One Parser per goroutine, or `ParserPool`.

### Anti-Pattern 3: Storing Elements Past Doc.Close()

**What people do:** `Parse` into a struct field of type `Element`, `Close()` the doc, then try to `GetString()` later.
**Why it's wrong:** Element is a view into the parser's tape; the tape is gone after Close.
**Do this instead:** Extract all needed values as owned Go types (`string`, `int64`) before `Close`. Document this prominently. The `doc.closed` guard turns UB into a clean error.

### Anti-Pattern 4: Returning `[]byte` Views Without Explicit Opt-In

**What people do:** Default string accessors return `[]byte` views into native memory to look fast.
**Why it's wrong:** Ties user code to Doc lifetime in a way that's easy to get wrong. `encoding/json`-familiar users will store views and hit UB.
**Do this instead:** Default `GetString()` returns `string` (copy). Add `GetStringView()` as an explicit zero-copy opt-in for v0.2 with documented lifetime rules.

### Anti-Pattern 5: cmake-Inside-Cargo-Build for simdjson

**What people do:** Use `cmake` crate to invoke simdjson's CMake build from `build.rs`.
**Why it's wrong:** Pulls in a cmake build-time dependency, slow, platform-flaky (especially on Windows MSVC). simdjson is a single-header amalgamation — `cc` crate compiles it directly.
**Do this instead:** Use `cc` crate to compile `simdjson.cpp` (amalgamated) directly; two-file (`simdjson.h` + `simdjson.cpp`) include in `third_party/simdjson/`. No cmake involvement. Verified approach per simdjson's "integrating" docs.

## Integration Points

### External Services

| Service | Integration Pattern | Notes |
|---------|---------------------|-------|
| CloudFlare R2 (binary distribution) | HTTPS GET + sha256 manifest, mirror of `pure-tokenizers` layout at `releases.amikos.tech/pure-simdjson/vX.Y.Z/` | Already operational infra |
| GitHub Releases (fallback) | Tag-based asset download when R2 unreachable | Same pattern as pure-onnx/pure-tokenizers |
| simdjson upstream | git submodule at `third_party/simdjson`, pinned tag | MIT-safe: simdjson is Apache-2.0 |

### Internal Boundaries

| Boundary | Communication | Notes |
|----------|---------------|-------|
| Go ↔ Rust | C ABI (`extern "C"`), `#[repr(C)]` structs, opaque pointers, error-code returns | Bound at runtime via purego.RegisterLibFunc; no cgo |
| Rust ↔ simdjson C++ | `cxx` crate for safe bridging, OR hand-written `bindgen`+`cc` for the narrow set of C++ calls we need | **Recommendation: `cxx` crate** — safer, handles string_view/unique_ptr cleanly, well-suited for our narrow surface (parser_new, parse, tape navigation) |
| Handle → tape index | Opaque Box pointer (Parser, Doc) + integer index (Element nodes) | Parser/Doc heap, Element stack |

## Build Order / Dependency Graph

Phases must land in this order because each depends on the previous:

```
Phase 1: FFI error-code convention + ABI version handshake
  │
  │   (blocks: every subsequent phase — error codes are the contract)
  ▼
Phase 2: Rust cdylib skeleton + cxx+simdjson compile + Parse(bytes) minimal path
  │     • parser_new / parser_free / parser_parse (returning doc handle)
  │     • doc_free / doc_root (returning an Element tape_idx)
  │     • element_type + element_get_int64 as the first typed accessor
  │     • build.rs drives cc crate over amalgamated simdjson
  │
  │   (blocks: Go side can't meaningfully test until Parse returns something)
  ▼
Phase 3: Go purejson package skeleton + purego load + Parse happy path
  │     • library_unix.go + library_windows.go + library_loading.go
  │     • Parser struct + New()/Close()
  │     • Doc struct + Parse()/Close()
  │     • single GetInt64 accessor as smoke test
  │
  │   (blocks: meaningful benchmarks)
  ▼
Phase 4: Full typed accessor surface
  │     • element_get_uint64 / _float64 / _string / _bool / _null
  │     • element_array / array_iter / array_len
  │     • element_object / object_iter_next / object_get_field
  │     • Go wrappers for all
  │
  ▼
Phase 5: Bootstrap pipeline — R2 download, sha256 verify, OS cache dir
  │     • download.go + library_loading.go orchestration
  │     • GitHub fallback path
  │     (can start in parallel with Phase 4 — independent code)
  │
  ▼
Phase 6: CI release matrix — 6 targets × rust-release.yml
  │     • Cross-compilation: ubuntu-latest (linux/amd64,arm64,arm),
  │       macos-latest (darwin/amd64,arm64), windows-latest (windows/amd64)
  │     • sha256 manifests + GH release upload + R2 push
  │
  │   (risky: Windows MSVC simdjson compile; arm7 float ABI; macOS signing)
  ▼
Phase 7: Benchmarks + docs + v0.1 release
```

### Risky Integration Points (phase markers)

| Risk | Phase | Mitigation |
|------|-------|------------|
| Windows MSVC compile of simdjson via `cc` crate | Phase 2/6 | Test Windows build path as part of Phase 2 exit criteria, not deferred to Phase 6 |
| arm7 (32-bit) float ABI ("hard" vs "soft") | Phase 6 | Pin to `arm-unknown-linux-gnueabihf` (hardfloat); confirm simdjson has ARMv7 fallback kernel (it has a generic scalar kernel) |
| macOS codesigning / Gatekeeper on dylib download | Phase 5 | Ad-hoc sign the dylib in rust-release.yml; fast-distance and pure-tokenizers already handle this — copy their approach |
| purego struct layout mismatch on windows (calling convention) | Phase 3 | windows/amd64 uses Microsoft x64 cc — same as every other lib we use; structured test of every FFI function on Windows in Phase 3 |
| simdjson kernel dispatch failing on CI runners with old CPUs | Phase 2/6 | simdjson has a fallback kernel; verify CI runners hit the expected kernel or the fallback; log detected kernel in `get_simd_info` for diagnostics |

### What Ships Before What (component-level)

- **Error codes (errors.rs + errors.go)** → before any function that uses them
- **ABI version handshake** → before first public release
- **Parser.New + Parse + single accessor** → before full accessor surface
- **Local-file library loading** → before bootstrap/download
- **Bootstrap/download** → before CI release pipeline (CI needs something to upload)
- **CI release pipeline** → before v0.1 tag

## simdjson-Specific Wrinkles (Not Found In Reference Repos)

These require new thinking beyond what pure-tokenizers/pure-onnx/fast-distance teach:

1. **Padded input requirement (SIMDJSON_PADDING).** simdjson requires input buffer to have `SIMDJSON_PADDING` (64) extra accessible bytes past the logical end. Two options: (a) Rust side allocates a padded copy always (simple, memcpy tax), (b) Go side offers a `ParseUnsafe` that trusts caller's buffer has padding. **Recommendation v0.1: always pad in Rust. Revisit in v0.2 if memcpy shows up in profiles.**

2. **Per-parser single-doc invariant.** Not a concept in tokenizers or onnx. Must be surfaced in the API (`ErrParserBusy`), documented, and tested.

3. **Tape views vs owned copies.** pure-tokenizers always copies out. For simdjson, default copy-out for strings/numbers, opt-in views for bulk scan use cases.

4. **C++ build from Rust.** fast-distance and pure-tokenizers are pure-Rust crates. pure-onnx downloads a pre-built library. pure-simdjson is the first in the family to compile C++ during Rust build. The `cc` crate + amalgamated simdjson.cpp is the validated approach per simdjson's own integration docs.

5. **Runtime kernel dispatch.** simdjson auto-dispatches at runtime. We expose `GetSimdInfo()` for diagnostics (same pattern as fast-distance's `get_simd_info`) but don't let the user force a kernel — simdjson's own heuristics are battle-tested.

## Sources

- **pure-tokenizers** (github.com/amikos-tech/pure-tokenizers, cloned 2026-04-14): `tokenizers.go`, `src/lib.rs`, `src/build.rs`, `library.go`, `library_loading.go`, `download.go`, `.kiro/steering/structure.md` — HIGH confidence, primary pattern
- **pure-onnx** (github.com/amikos-tech/pure-onnx, cloned 2026-04-14): `ort/bootstrap.go`, `ort/session.go`, `ort/finalizer_log.go`, `internal/c_api/` — HIGH confidence for bootstrap + finalizer patterns
- **fast-distance** (github.com/amikos-tech/fast-distance, cloned 2026-04-14): `Cargo.toml`, `build.rs`, `src/lib.rs`, `dispatch.go`, `distance.go` — HIGH confidence for cbindgen + SIMD-dispatch patterns
- **simdjson docs** (github.com/simdjson/simdjson/blob/master/doc/basics.md, fetched 2026-04-14): Parser/Doc/Element lifecycle, thread-safety, padding, kernel dispatch — HIGH confidence
- **simdjson DOM docs** (same repo, dom.md path, fetched 2026-04-14): DOM tree stability after parse, tape borrowing rules — HIGH confidence
- **simdjson On-Demand docs** (fetched 2026-04-14): forward-only iteration + single-active-element constraint — MEDIUM confidence, drives v0.1 decision to start with DOM API not On-Demand
- **purego** (github.com/ebitengine/purego): `RegisterLibFunc` semantics, Dlopen/LoadLibrary abstraction — inferred from pure-tokenizers usage, HIGH confidence
- **cxx crate** (cxx.rs): Rust ↔ C++ bridging for our Rust shim — MEDIUM confidence, not yet prototyped against simdjson specifically

---
*Architecture research for: Go↔Rust↔C++ SIMD JSON parser library*
*Researched: 2026-04-14*
