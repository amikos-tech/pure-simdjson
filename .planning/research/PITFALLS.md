# Pitfalls Research

**Domain:** Go library consuming a Rust shim over C++ simdjson via purego, distributed as prebuilt shared libraries across six platforms
**Researched:** 2026-04-14
**Confidence:** HIGH for FFI/simdjson invariants (verified against upstream docs and purego source), MEDIUM for cross-platform build specifics (verified against kernel/toolchain docs), MEDIUM for distribution strategy (no canonical source — derived from pure-onnx/pure-tokenizers precedent)

---

## Critical Pitfalls (P0 — data corruption, UB, or crash)

### Pitfall 1: Reusing `dom::parser` while document views are still live

**What goes wrong:**
Calling `parser.parse(buf2)` — or destroying/moving the parser — while any `dom::element`, `dom::object`, `dom::array`, or `std::string_view` from a previous `parse(buf1)` is still alive. The second parse overwrites the tape and string buffer in place. Reads through the old element now return garbage (valid-looking JSON values from the second document, or corrupted strings if the tape layout changed).

**Why it happens:**
simdjson documents this as "the JSON document still lives in the parser" — which is the whole performance story — but it is not type-system enforced. From a Go side, handles hide the invariant entirely: a caller holds a `Doc` handle, calls `parser.Parse()` again to reuse the parser, and every `Doc` handle previously handed out now silently points at the new document's tape.

**How to avoid:**
- FFI surface MUST NOT expose a "reuse parser for next doc" call that does not also invalidate the prior `Doc` handle. Enforce 1-parser-holds-1-live-doc at the Rust shim: `parser_parse(parser_handle, input) -> doc_handle` consumes (invalidates) any prior doc_handle owned by that parser. Track the generation counter per parser and stamp it into every doc/element handle; reject operations whose stamp doesn't match current generation with `ERR_DOC_INVALIDATED`.
- Wrapper `Doc.Close()` must be idempotent. Parser reuse is an explicit operation on the parser handle, not an implicit side effect of `Parse`.
- Document in Go doc comments: "A `Doc` from `Parser.Parse` is invalidated by the next `Parser.Parse` call on the same parser. Clone values you need to retain."

**Warning signs:**
- Integration test: parse doc A, extract string view, parse doc B on same parser, assert view still matches A → should fail fast with `ERR_DOC_INVALIDATED`, not return silent garbage.
- Fuzz test: round-robin parse of two different JSONs on one parser, read random fields, diff against independent parse. Any mismatch = invariant leak.
- ASAN/Valgrind run over the integration suite — catches the tape-overwrite as a use-after-scope.

**Phase to address:** FFI design phase (before implementation). Handle-generation scheme must be in the FFI contract before a single Rust function is written.

**Severity:** P0

---

### Pitfall 2: Input buffer freed while Doc is alive (ownership split across FFI)

**What goes wrong:**
Go caller passes `[]byte` to `Parse`. Rust shim keeps a pointer into that buffer (for zero-copy string views). Go's GC moves or frees the backing array. Subsequent `GetString` reads return garbage or segfault.

**Why it happens:**
purego does not apply cgo's pointer-passing rules. There is no runtime check that Go memory is pinned. A naive shim stores the pointer in the parser/doc state assuming "the caller will keep it alive" — but Go conventions don't make that obvious to the caller either.

**How to avoid:**
- Rust shim must **copy the input into a Rust-owned buffer** (with SIMDJSON_PADDING already added) on every `Parse` call. The cost of a single `memcpy` is negligible relative to parse time and is the only memory-safe design given Go's GC.
- Alternatively, expose a `ParsePinned(ptr, len)` only if the Go wrapper holds a `runtime.Pinner` for the slice for the full `Doc` lifetime. This is the advanced API for v0.2; v0.1 copies.
- Never store a raw Go pointer in Rust state that outlives a single FFI call.

**Warning signs:**
- Stress test: parse in a tight loop with `GOGC=1` forcing frequent GC cycles between parse and read. Any crash = ownership bug.
- Parse a buffer from a `sync.Pool`, release to pool before reading from `Doc`, read. Must either copy or hold the buffer — corruption here = design bug.

**Phase to address:** FFI design phase. The "does Rust own the input buffer" decision gates the entire API.

**Severity:** P0

---

### Pitfall 3: SIMDJSON_PADDING not applied to input

**What goes wrong:**
simdjson reads 8–64 bytes at a time including past the logical end of the input. If the input ends within `SIMDJSON_PADDING` bytes of an unmapped page boundary, the parser segfaults. If you're lucky, you get `INSUFFICIENT_PADDING`; if unlucky, SIGSEGV.

**Why it happens:**
It's a performance-motivated invariant that most JSON test fixtures never hit — small, heap-allocated test inputs always have padding by accident. Production traffic with mmap'd files or exact-sized slab allocations trips it.

**How to avoid:**
- Rust shim allocates `len + SIMDJSON_PADDING` bytes for every parse and copies the input in (see Pitfall 2; same mechanism solves both).
- Use `simdjson::padded_string::load()` or `padded_string_view` at the C++ call site — never raw pointers.
- CI test specifically includes a JSON file page-aligned to the end of a single-page `mmap` (verify manually; fuzz harness should generate boundary cases).

**Warning signs:**
- Page-boundary fuzz: allocate one page via `mmap`, write `len - PAGE_PADDING` bytes JSON at offset `PAGE_PADDING`, parse. Must succeed without the Rust shim silently padding for you (i.e., verify the copy-in path is actually triggered).

**Phase to address:** FFI implementation phase (shim design).

**Severity:** P0

---

### Pitfall 4: Rust panic crossing FFI without `catch_unwind`

**What goes wrong:**
Any panic (array index OOB, `unwrap` on `None`, allocation failure, integer overflow in debug) in the Rust shim unwinds into Go's runtime. With `extern "C"` (not `"C-unwind"`) this is "safe abort" on modern Rust — the whole process dies — but process death in a long-running Go server is exactly the failure we're trying to prevent. With `"C-unwind"`, unwinding into Go stack frames is undefined behavior.

**Why it happens:**
Easy to forget `catch_unwind` on every single `#[no_mangle] pub extern "C"` function. simdjson's API returns errors, but the Rust code around it can still panic (allocation, index, cxx bridge failures).

**How to avoid:**
- Every exported function wraps its body in `std::panic::catch_unwind(AssertUnwindSafe(|| { ... }))`. On `Err`, return a designated error code like `ERR_PANIC`. Log the panic message into a per-thread-last-error buffer that Go can retrieve.
- Build with `panic = "abort"` in `Cargo.toml` for release — makes the failure mode deterministic. `catch_unwind` still needed because debug/test builds may unwind.
- Add a lint or clippy rule (`disallowed_methods` for `unwrap`/`expect` in FFI layer) or a wrapper macro `ffi_fn!` that enforces the pattern at compile time.

**Warning signs:**
- Test that deliberately triggers a panic (e.g., `get_field_by_index(999)` on a 2-field object, if the shim panics on OOB) and verifies the process does not abort and an error code is returned.
- Grep CI: `pub extern "C" fn` in shim source without a `catch_unwind` in its body should fail the build.

**Phase to address:** FFI implementation phase. Ship the `ffi_fn!` wrapper macro on day one — retrofitting catch_unwind later means auditing every function.

**Severity:** P0

---

### Pitfall 5: C++ exceptions leaking from simdjson into Rust

**What goes wrong:**
simdjson-the-library prefers `error_code` returns, but C++ can still throw — `std::bad_alloc` from internal allocation, or implicit exceptions from `operator[]` variants that throw on missing keys (`document["foo"]` throws; `document["foo"].get()` returns error code). If the Rust shim uses `cxx` or raw bindings, a C++ exception crossing into Rust via a non-`C-unwind` boundary is UB; via `C-unwind` it's implementation-defined per the Nomicon.

**Why it happens:**
Easy to use the throwing form of the simdjson API by accident (it's the ergonomic form in C++ examples). Rust shim authors often don't realize the wrapped C++ call can throw.

**How to avoid:**
- Use only the **error-code form** of every simdjson call: `auto err = doc["foo"].get(value)`, never `doc["foo"]`.
- If using `cxx`, mark all extern functions to return `Result<T>` (cxx translates C++ exceptions into `cxx::Exception`). Never call a throwing C++ function from a non-`Result` cxx bridge.
- Wrap every C++ call site in a C shim with a top-level `try { ... } catch (...) { return ERR_CPP_EXCEPTION; }`. The Rust side never sees an in-flight exception.
- Compile simdjson with exceptions disabled where possible (`-fno-exceptions`) — not fully supported by simdjson but worth investigating as a belt-and-suspenders measure.

**Warning signs:**
- Grep shim C++ source: any `operator[]` on `dom::element`/`ondemand::document` without `.get(...)` is a latent throw. CI grep check.
- Allocation-failure injection test (override `operator new` to throw `bad_alloc` on the Nth call) — shim must return an error code, not abort.

**Phase to address:** FFI implementation phase.

**Severity:** P0

---

### Pitfall 6: On-Demand API — reading the same field twice aborts

**What goes wrong:**
With simdjson's On-Demand API, attempting to read the same node twice **aborts the program** (not "returns an error" — `abort()`). This is documented as preventing string double-unescaping. Out-of-order field access is slower but not fatal; double-read is fatal.

**Why it happens:**
DOM API is re-readable; On-Demand is not. Developers who prototype with DOM and switch to On-Demand for perf don't realize the semantics changed.

**How to avoid:**
- If v0.1 uses DOM, this is deferred to v0.2. When On-Demand lands, the Go API must either:
  (a) Model single-use explicitly — `v, err := field.AsString()` consumes the field value handle, `field.AsString()` called twice returns `ErrAlreadyConsumed`, OR
  (b) Force-copy on first access into a Rust-owned scratch (kills the On-Demand advantage — avoid).
- Wrap every On-Demand value access in a "consumed" flag at the Rust layer, return `ERR_ALREADY_CONSUMED` on second access. **Never let a double-read reach simdjson's abort path.**

**Warning signs:**
- Integration test specifically reads a scalar field twice through the Go API — must return error, not crash.
- Document in Go doc comments with an example: "On-Demand values are single-shot. Clone with `.Bytes()` if you need to re-examine."

**Phase to address:** v0.2 On-Demand phase — but the handle-consumption design must be sketched in v0.1 FFI so the v0.2 addition fits cleanly.

**Severity:** P0

---

### Pitfall 7: Go callback from native code — goroutine stack + scheduler interaction

**What goes wrong:**
If the visitor API uses `purego.NewCallback` so that simdjson's iteration calls back into Go per element, several issues:
1. The callback runs on the calling OS thread's stack, not a goroutine's resizable stack. Deep recursion or large stack frames overflow.
2. Panics in the Go callback propagate back into C++ code, which has no unwinding support → UB/abort.
3. The callback M is pinned; goroutine scheduler cannot preempt it. High-rate callbacks (millions/sec during SIMD parse) stall the P.
4. purego documents "at least 2000 callbacks" can be created process-wide and **allocated memory is never released**. Creating a callback per parse leaks.

**Why it happens:**
Visitor pattern is the natural way to expose "walk a document without allocating a Go tree," but `NewCallback` has strict limits that aren't in the README.

**How to avoid:**
- **Do not register a callback per parse or per Doc.** Register a small, fixed set of trampoline callbacks at package init (e.g., one per iteration type: object-entry, array-element, scalar). Callback body dispatches to a Go-side visitor stored in a handle table keyed by uintptr.
- Consider an alternative to `NewCallback` entirely: **cursor/pull model**. Go calls `Next()`/`EnterObject()`/`GetKey()` in a loop. Each call is a plain FFI invocation, no callback. Avoids stack switching, panic propagation, callback table exhaustion. For simdjson On-Demand this maps naturally.
- If callbacks are unavoidable, recover panics at the top of the Go callback: `defer func() { if r := recover(); r != nil { /* stash, set error flag, return */ } }()`.
- Constrain callback arguments to `uintptr`-sized per purego limits; pass complex data via a handle + separate FFI getters.

**Warning signs:**
- Benchmark 10M-element JSON parse through visitor API — measure allocations attributable to callback trampolines. Any linear growth in callback count = leak.
- Test: panic from inside a Go visitor callback must surface as a returned error from `Parse`, not kill the process.

**Phase to address:** API design phase (pre-FFI). The visitor-vs-cursor decision is the biggest API shape question.

**Severity:** P0

---

### Pitfall 8: Struct-by-value return across purego FFI

**What goes wrong:**
purego's struct support is limited: docs state structs-by-value return works only on darwin amd64/arm64 and linux amd64/arm64, with **manual padding**. Windows amd64, linux arm, linux arm64 edge cases, and floating-point struct members all have pitfalls. A `simdjson_number_t { type: u8, i64_val, u64_val, f64_val }` returned by value compiles and runs on dev Mac, segfaults on Windows.

**Why it happens:**
Easy FFI shape. Feels natural. Unsupported on two of six target platforms.

**How to avoid:**
- **Hard rule: no struct returns across the FFI boundary.** Every function returns a single integer (error code or small scalar). Multi-value returns use out-parameters (pointer to caller-allocated struct).
- For number access, expose separate entry points: `doc_get_int64(handle, path, *i64_out) -> err`, `doc_get_uint64(...)`, `doc_get_double(...)`, plus `doc_get_number_type(handle, path) -> u8`. More calls, simpler ABI.
- Write a CI matrix check: build and run a minimal "all FFI signatures smoke test" on every one of the six target triples. Struct-return bugs manifest as immediate crashes.

**Warning signs:**
- Any `extern "C"` function whose return type is not `i32`/`i64`/`u32`/`u64`/`*mut c_void`/`()` fails code review.
- CI smoke test on windows/amd64 and linux/arm specifically — these are where struct-return bugs hide.

**Phase to address:** FFI design phase (ABI document).

**Severity:** P0

---

### Pitfall 9: Floating-point + integer mixed argument bug on purego amd64

**What goes wrong:**
purego docs: "`SyscallN` does not properly call functions with both integer AND float parameters" and "on amd64: if there are more than 8 floats, the 9th+ will be placed incorrectly on the stack." `RegisterFunc` is better but the underlying issue reflects the System V ABI classification complexity.

**Why it happens:**
FFI convenience functions that mix `float64` args with `uintptr` args hit this. Signatures like `fn(handle uintptr, threshold float64, count int)` are exactly the shape that breaks.

**How to avoid:**
- **Never mix float and int args in an FFI function signature.** If a function needs a double parameter, pass it as a pointer to a `f64` buffer, or bit-cast to `uint64` and unpack Rust-side.
- Prefer: all FFI args are `uintptr` (pointers or indices). Payload details live in structs on the caller side, passed via pointer.
- CI smoke test must exercise any float-taking/returning FFI symbol on all platforms.

**Warning signs:**
- Code review: any `extern "C"` signature with both `f64`/`f32` and non-float args = block.

**Phase to address:** FFI design phase (ABI document).

**Severity:** P0

---

### Pitfall 10: Handle double-free / use-after-free from Go side

**What goes wrong:**
Go's GC can run a finalizer on a handle wrapper concurrently with an explicit `Close()`. Or the caller passes a handle to two goroutines, both call `Close()`. Rust side's `Box::from_raw` panics on double free, or — worse — the slot gets re-allocated between close and the stale use, and the stale handle now points at an unrelated object.

**Why it happens:**
Handle-as-opaque-pointer is the ergonomic choice but fragile. pure-tokenizers convention uses handles; the pitfall is shared.

**How to avoid:**
- Handles are **generation-stamped indices into a registry**, not raw pointers. `Handle = { slot: u32, generation: u32 }` packed into `u64`. Rust-side registry stores the object and a per-slot generation counter. Every FFI op checks `slot.generation == handle.generation`; mismatch = `ERR_INVALID_HANDLE`. On free, increment generation. Double free = `ERR_INVALID_HANDLE`, not abort.
- Go wrapper uses `atomic.CompareAndSwap` on the handle field to make `Close()` idempotent and race-free. `SetFinalizer` only for leak detection (warn in test builds), never as primary cleanup.
- Document: callers must call `Close()`; do not share handles across goroutines without external synchronization.

**Warning signs:**
- Race-detector CI run of tests including a `TestDoubleClose` that calls `Close()` 10x in parallel.
- `go test -race` on a fuzz harness that randomly clones and frees handles.

**Phase to address:** FFI design phase.

**Severity:** P0

---

### Pitfall 11: Invalid UTF-8 handling — simdjson validates, but via the error channel

**What goes wrong:**
simdjson **does validate UTF-8** during parse and returns `UTF8_ERROR`. But the On-Demand API validates lazily — you may not see the error until you access the bad value. If the Go side assumes "parse succeeded = all strings valid," it can pass a later-returned invalid UTF-8 `[]byte` to Go code that assumes string validity (e.g., storing in a `map[string]struct{}`, which is fine, but printing through `fmt` may re-scan).

**Why it happens:**
The mental model "parser error means parse failed" doesn't hold for On-Demand. And at the FFI level, if the Rust shim does eager validation for safety it costs the lazy-eval win; if it defers, Go side gets late errors.

**How to avoid:**
- Decision in FFI design: v0.1 DOM API does full validation at parse time — fewer surprises. v0.2 On-Demand exposes `ERR_UTF8_LATE` from every string accessor.
- Every `GetString`/`GetBytes` in Go returns `(value, error)`. Callers cannot forget to check.
- Do not expose raw pointers to unvalidated string content. Always go through a validating accessor.

**Warning signs:**
- Fuzz corpus includes truncated UTF-8, overlong sequences, lone surrogates. Parser must return a documented error, never corrupt or panic.

**Phase to address:** FFI design phase (v0.1 sets the contract).

**Severity:** P0

---

### Pitfall 12: Number type ambiguity — int64 vs uint64 vs double

**What goes wrong:**
`{"x": 9999999999999999999}` fits in `u64` but not `i64`. `{"x": 1e20}` is a double but looks like an integer. `{"x": 9007199254740993}` is exactly `2^53 + 1` — loses precision as a `double`, exact as `u64`. If Go wrapper always goes through `GetInt64`, overflow silently truncates.

**Why it happens:**
The library's core value proposition ("distinct int64/uint64/float64") is easy to undermine with a lazy API. `GetNumber() any` or `GetInt64` as the default getter erases the win.

**How to avoid:**
- No generic `GetNumber`. API is `GetInt64(h, path) (int64, err)`, `GetUint64`, `GetFloat64`, `GetNumberType(h, path) (NumberType, err)`. Callers pick.
- `GetInt64` on a value that overflows returns `ErrNumberOutOfRange` — never silent truncation.
- `GetFloat64` on a value with an exact integer representation that exceeds 2^53 returns `ErrPrecisionLoss` — at least in strict mode, configurable.
- Fuzz: generate numbers at boundary values (`i64::MAX`, `u64::MAX`, `2^53`, negative zero, subnormals, `Inf`-like strings if the parser accepts them) and assert round-trip semantics.

**Warning signs:**
- Unit test matrix specifically covers the boundary values above against simdjson's `get_int64`/`get_uint64`/`get_double` semantics.

**Phase to address:** API design phase (v0.1 — this is in the core value prop).

**Severity:** P0

---

## High-severity pitfalls (P1 — UX-breaking, release-blocking, or hard to diagnose)

### Pitfall 13: CPU kernel dispatch on older/virtualized hardware

**What goes wrong:**
simdjson picks an implementation (westmere, haswell, icelake, arm64, fallback) at runtime via CPUID. On VMs or emulators CPUID may lie, report unsupported features, or vary per boot. On CPUs older than westmere (no SSE4.2), simdjson falls back to a slow scalar path — but the Go consumer may not know and benchmarks look terrible.

**Why it happens:**
`-march=native` during build pins to builder CPU and fails everywhere else (kernel dispatch goes away). Correct build uses runtime dispatch, but users don't realize they're on the slow path.

**How to avoid:**
- Build simdjson **without** `-march=native`; let runtime dispatch select.
- Expose `Kernel()` / `ImplementationName()` in the Go API so users can log which kernel is active in production.
- Project README docs the CPU floor (SSE4.2 on x86_64, NEON on arm64) and what "fallback" means perf-wise.
- Decision per PROJECT.md: fail loudly rather than silently use fallback. On load, `simdjson::get_active_implementation()` — if name equals "fallback", return `ERR_CPU_UNSUPPORTED` from `New()` unless user explicitly opts in.

**Warning signs:**
- Run CI benchmark on an explicitly-older runner (e.g., old GH actions image, or a VM with `-cpu qemu64`). Throughput must stay within an order of magnitude of reference.
- Integration test that checks `Kernel()` returns one of the expected names on each platform.

**Phase to address:** Build/release phase (toolchain flags) + API phase (expose `Kernel()`).

**Severity:** P1

---

### Pitfall 14: Cross-compile Docker toolchain mismatches for linux/arm and linux/arm64

**What goes wrong:**
Cross-compiling C++ from GH Actions ubuntu-latest to `aarch64-unknown-linux-gnu` works. But producing a `.so` that runs on Ubuntu 18.04 (glibc 2.27) requires targeting that old glibc, not the builder's glibc 2.39. Symbol like `__libc_single_threaded` or `fcntl64` pulled in from newer glibc = runtime load failure with no obvious diagnostic.

linux/arm (32-bit) adds: soft-float vs hard-float ABI. `armv7-unknown-linux-gnueabi` (soft) and `armv7-unknown-linux-gnueabihf` (hard) are **not binary compatible**. Raspberry Pi OS is hard-float; some embedded targets soft-float.

**Why it happens:**
Default toolchains target the newest available glibc. Hard-float is standard on modern ARM but the ABI confusion is a classic footgun.

**How to avoid:**
- Build inside a **manylinux-style** docker image (e.g., `quay.io/pypa/manylinux2014_x86_64` equivalent) that pins old glibc, or use `zig cc` as cross compiler (`-target aarch64-linux-gnu.2.17`) to explicitly set glibc version.
- Ship linux/arm **only as armv7-unknown-linux-gnueabihf** (hard-float). Document the ABI. Do not ship a soft-float variant — would require two artifacts and most users don't know which they need.
- CI has a "compatibility test" job: load the built `.so` on Ubuntu 18.04, Debian 11, RHEL 8 containers. Any `GLIBC_2.xx not found` fails the release.

**Warning signs:**
- `objdump -T libsimdjson_shim.so | grep GLIBC_` reveals the max glibc symbol requirement. CI asserts `≤ 2.17` (manylinux2014 baseline).
- `readelf -A` on the arm binary shows `Tag_ABI_VFP_args: VFP registers` (hard-float) or missing (soft).

**Phase to address:** Build-matrix phase.

**Severity:** P1

---

### Pitfall 15: Darwin arm64 code signing / notarization / Gatekeeper for dylibs

**What goes wrong:**
Unsigned `.dylib` downloaded at runtime from CloudFlare R2 on darwin arm64 + Sequoia may be quarantined by Gatekeeper, triggering "cannot be opened because the developer cannot be verified." Worse, on arm64 the dynamic loader enforces signatures more strictly than x86_64 — an unsigned arm64 dylib can fail to load with a cryptic `EBADMACHO` or similar.

**Why it happens:**
The download-at-first-use pattern worked fine on Linux/Windows with pure-onnx/pure-tokenizers. macOS arm64 is stricter. Apple's policy has tightened since 2022.

**How to avoid:**
- Sign the darwin dylibs with an Apple Developer ID (`codesign -s "Developer ID Application: ..." --options runtime lib*.dylib`) before uploading to R2. Notarization (`xcrun notarytool`) for downloaded-from-internet binaries.
- Strip the quarantine xattr on download: after saving the file, run `xattr -d com.apple.quarantine` before `dlopen`. Purego-side can do this with a Go `syscall` wrapper.
- Fallback: provide a mechanism to use a user-bundled dylib at a known path so enterprises can pre-sign and install via their MDM.

**Warning signs:**
- End-to-end test on a clean macOS arm64 VM (or GH Actions `macos-14`) that downloads and loads the dylib without `sudo xattr -cr`.
- `codesign -dv libsimdjson_shim.dylib` in CI — fails build if not signed.

**Phase to address:** Release/distribution phase.

**Severity:** P1

---

### Pitfall 16: Windows DLL search path hijacking

**What goes wrong:**
Go app downloads `simdjson_shim.dll` into `$TEMP` and calls `LoadLibraryW`. Windows search order searches the current working directory. If an attacker drops a `simdjson_shim.dll` in the working directory first, it's loaded instead.

**Why it happens:**
Default `LoadLibraryW("simdjson_shim.dll")` uses unsafe search order. Developers rarely think about CWD-based attacks.

**How to avoid:**
- Always pass a **fully qualified path** to `LoadLibrary` — the path where the downloader stored the DLL. `purego.Dlopen(absPath, ...)`.
- Call `SetDefaultDllDirectories(LOAD_LIBRARY_SEARCH_DEFAULT_DIRS)` or use `LoadLibraryExW` with `LOAD_LIBRARY_SEARCH_SYSTEM32 | LOAD_LIBRARY_SEARCH_USER_DIRS` — cannot do this from purego directly, so document and encourage full paths instead.
- The bootstrap downloader stores the DLL in a **per-user, non-world-writable directory** (e.g., `%LOCALAPPDATA%\amikos\pure-simdjson\<version>\`). Verify directory ACLs.

**Warning signs:**
- Manual security review: any `purego.Dlopen` call that takes a bare filename (not an absolute path) fails review.
- CI test on Windows: download a real DLL and a decoy `simdjson_shim.dll` in CWD, verify the real one loads.

**Phase to address:** Distribution phase (bootstrap downloader).

**Severity:** P1

---

### Pitfall 17: Binary integrity — no checksum/signature verification on download

**What goes wrong:**
`GET https://r2.amikos.tech/pure-simdjson/v0.1.0/linux-amd64/libsimdjson_shim.so` → write to disk → `dlopen`. If the R2 bucket is compromised, or a MITM (rare with TLS, but corporate proxies do MITM), the attacker ships a malicious `.so` that runs in-process with full permissions.

**Why it happens:**
Easy to skip. "TLS is enough." It isn't, because corporate firewalls replace certs.

**How to avoid:**
- Ship a **SHA-256 of each artifact baked into the Go code** (generated at release time, committed to the source tree). Verify post-download, before `dlopen`. Mismatch = refuse to load, surface an error.
- Optionally: sign the manifest with minisign / sigstore, embed public key in Go, verify signature. Higher effort, stronger guarantee.
- Pin `User-Agent` + include version in URL so R2 logs attribute downloads to specific library versions.

**Warning signs:**
- CI test: serve a corrupted binary from a test server, ensure the downloader refuses to load and errors cleanly.
- Release checklist item: sha256 of every artifact matches hash table in Go source before tagging.

**Phase to address:** Distribution phase.

**Severity:** P1

---

### Pitfall 18: Version pinning — client pulls "latest" breaks on server update

**What goes wrong:**
URL schema `r2.amikos.tech/pure-simdjson/latest/...` — every client auto-upgrades. A breaking change in the shim ABI between v0.1.3 and v0.1.4 means an existing Go binary compiled against v0.1.3's ABI downloads v0.1.4 and segfaults at the first FFI call.

**Why it happens:**
Too-easy distribution pattern. Matches the pure-onnx model if it's been careless.

**How to avoid:**
- The download URL includes the **exact semver of the Go package**: `r2.amikos.tech/pure-simdjson/<GoPackageVersion>/<os>-<arch>/...`. Generated from `debug/buildinfo` or a compile-time constant in Go source.
- Go code embeds expected ABI version; shim exports `get_abi_version()`; Go verifies match after load. Any mismatch = `ErrABIVersionMismatch`, refuse to use.
- Never expose a `latest` alias in the URL schema. Every Go library release maps 1:1 to an artifact URL.

**Warning signs:**
- Integration test: run v0.1.3 Go code against v0.1.4 artifacts, expect `ErrABIVersionMismatch` (not a segfault).
- Release checklist: bumping the Go module version requires uploading artifacts at the new path, not overwriting.

**Phase to address:** Distribution phase + API versioning (v0.1 contract).

**Severity:** P1

---

### Pitfall 19: Cold-start latency from runtime binary download

**What goes wrong:**
First invocation of `pure-simdjson.Parse(...)` blocks for 2–10 seconds while 2–5 MB is pulled from R2. In a serverless or autoscaling context, this is catastrophic — the first request of every cold container times out.

**Why it happens:**
Download-on-first-use is convenient but is a hidden cost invisible in dev-box benchmarks.

**How to avoid:**
- Provide a `pure-simdjson.BootstrapSync()` function that's safe to call from `init()` or a preflight step, with context/timeout. Applications that care call it at startup.
- Document: "For production, pre-download the binary at container build time." Provide a CLI tool `pure-simdjson-bootstrap` that downloads artifacts for `$GOOS/$GOARCH` into a user-specified directory. Users pre-download in their Dockerfile.
- Support an env var `PURE_SIMDJSON_LIB_PATH=/app/lib/libsimdjson_shim.so` that bypasses download entirely.
- Cache downloaded binary in a user-cache directory; second invocation does a single stat + hash check, not a re-download.

**Warning signs:**
- Benchmark "cold start": `time go run main.go` where main.go does a single Parse. Must stay under a documented ceiling (e.g., 500ms on fast network, with fallback to "you need to bootstrap" error after 10s).
- Integration test with air-gapped container — `PURE_SIMDJSON_LIB_PATH` must be the intended escape hatch and it must work.

**Phase to address:** Distribution phase.

**Severity:** P1

---

### Pitfall 20: Corporate egress blocking R2

**What goes wrong:**
Enterprise user's firewall blocks `r2.amikos.tech`. Library load fails with a network error. No offline path.

**Why it happens:**
R2 uses Cloudflare's edge. Some corporate allowlists don't include it. Air-gapped environments have no egress at all.

**How to avoid:**
- `PURE_SIMDJSON_LIB_PATH` override (see Pitfall 19) is the primary escape hatch — document loudly.
- Publish SHA-256 checksums in the GitHub release page as well as R2, so users can verify artifacts downloaded from any mirror.
- Provide a `go run github.com/amikos-tech/pure-simdjson/cmd/bootstrap --output ./lib` CLI that IT can run once on a build machine with egress and ship the binary with the application.
- Support a mirror URL override: `PURE_SIMDJSON_BINARY_MIRROR=https://mycorp.example.com/pure-simdjson/`.

**Warning signs:**
- Integration test on a container with `--network none`: must fail with actionable error pointing to `PURE_SIMDJSON_LIB_PATH`.

**Phase to address:** Distribution phase.

**Severity:** P1

---

### Pitfall 21: glibc vs musl (Alpine) — shipping only one

**What goes wrong:**
A `.so` built against glibc does not run on Alpine (musl). Alpine is the default base image for minimal Docker containers; "works on my Ubuntu, crashes on Alpine with no error" is a classic issue.

**Why it happens:**
Six platforms in PROJECT.md are OS/arch, not libc-specific. Alpine is a libc split, not an OS.

**How to avoid:**
- **Decision**: Do we target Alpine?
  - YES: build linux/amd64 and linux/arm64 for both glibc and musl. Add two more artifacts (8 total linux artifacts). Build via `x86_64-unknown-linux-musl` and `aarch64-unknown-linux-musl` Rust targets + static-linked C++ runtime.
  - NO: document clearly in README. `PURE_SIMDJSON_LIB_PATH` for Alpine users to bring their own.
- Recommend: **YES for v0.1**. Alpine is too common in containerized Go deployments to ignore.
- Build matrix test: load the binary in an Alpine container as a CI job. Catches glibc-symbol leaks the manylinux check would miss.

**Warning signs:**
- `ldd libsimdjson_shim.so` on a linux artifact: must show only `libc.so.6`+`libstdc++.so.6`+`libm.so.6`+`libgcc_s.so.1`+`libpthread.so.0` (or fewer with static-link). Any extras = portability risk.
- Alpine smoke-test job in CI.

**Phase to address:** Build-matrix phase.

**Severity:** P1

---

### Pitfall 22: libstdc++ ABI / static vs shared C++ runtime

**What goes wrong:**
simdjson uses `std::string`, `std::vector`, exception types. If the shim dynamically links `libstdc++.so.6` and the host system has an older version without the `CXX_ABI_1.3.11` symbols we need, load fails. If we static-link, but the user's other loaded library dynamically links a different `libstdc++`, mixing the two can corrupt exception tables.

**Why it happens:**
C++ ABI across distros is a minefield. glibcxx symbol versioning + the 2011-era ABI break (old vs new `std::string`) + distro pinning.

**How to avoid:**
- **Static-link `libstdc++` and `libgcc`** into the shim `.so`: `-static-libstdc++ -static-libgcc`. Bloats the artifact ~1-2 MB but fixes ABI portability in one shot.
- Use `-D_GLIBCXX_USE_CXX11_ABI=1` consistently (default on modern toolchains; worth pinning explicitly).
- Since the shim does not **export** any C++ symbols (only `extern "C"` functions), static-linking the C++ runtime is safe — no risk of mixing runtimes with user code. Make this an explicit build invariant.

**Warning signs:**
- `nm libsimdjson_shim.so | grep ' T '` should show only the `extern "C"` entry points. Any `_Z` (mangled C++) symbol as `T` exported = leak.
- CI compat job: load the shim on Debian oldstable, Ubuntu LTS, Amazon Linux 2, RHEL 8 containers.

**Phase to address:** Build-matrix phase.

**Severity:** P1

---

### Pitfall 23: arm64 page-size divergence (4K vs 16K vs 64K)

**What goes wrong:**
macOS arm64 uses 16 KB pages. Linux arm64 defaults to 4 KB on Debian/Ubuntu but 64 KB on some RHEL/CentOS kernels. If the shim (or the simdjson mmap path for `load()`) assumes 4 KB alignment for page-boundary tricks, it breaks on 64 KB systems with alignment faults.

**Why it happens:**
simdjson's page-boundary guards are based on `getpagesize()` so they're fine. But any custom mmap-based buffer management in the Rust/C++ shim that bakes in `4096` is wrong.

**How to avoid:**
- Do not bake in `4096`. Use `sysconf(_SC_PAGESIZE)` at runtime or Rust `page_size::get()`.
- If the shim memory-maps input files (v0.2 zero-copy path), it must round up to actual page size.
- CI job on a 64K-page kernel — not free, but AWS Graviton instances with CentOS Stream 9 or a qemu-aarch64 + custom kernel does it.

**Warning signs:**
- Grep source: any literal `4096` in mmap-related code = review.
- Test on macOS arm64 (16 KB page) covers this partially for free.

**Phase to address:** FFI implementation phase (if mmap is used) / v0.2 zero-copy phase.

**Severity:** P1 (P0 if triggered — crashes; P1 likelihood)

---

### Pitfall 24: Benchmark vs `encoding/json` comparing different shapes

**What goes wrong:**
Benchmarks show "pure-simdjson is 10× faster than `encoding/json`!" — but the simdjson benchmark uses a visitor that touches 3 fields, while the encoding/json benchmark does `json.Unmarshal(&map[string]any{})` materializing the entire tree. Reviewers call out the unfair comparison; project credibility takes a hit.

**Why it happens:**
The two libraries have fundamentally different APIs. Constructing a truly fair comparison requires care.

**How to avoid:**
- Benchmark suite has **three tiers** for every fixture:
  1. **Apples-to-apples full parse**: `encoding/json` → `map[string]any`, vs `pure-simdjson` → equivalent Go tree materialization (if we offer one) or `minio/simdjson-go`'s `ParseBytes`. Same shape of work.
  2. **Apples-to-apples typed**: `encoding/json` → `type Foo struct{...}`, vs `pure-simdjson` typed accessors for the same fields.
  3. **Apples-to-apples selective**: parse only N fields. Measure. Clearly labeled as "selective path — not equivalent to a full Unmarshal."
- Publish benchmark harness source with README explaining what each tier measures.
- Include allocations (`-benchmem`) in every benchmark. Surface allocs/op alongside ns/op.
- Compare against `minio/simdjson-go` on identical hardware and fixtures — this is the honest peer comparison.

**Warning signs:**
- Peer review of benchmark PRs by a second person. No benchmarks land without review.

**Phase to address:** Benchmarking phase (dedicated v0.1 deliverable per PROJECT.md).

**Severity:** P1 (credibility)

---

### Pitfall 25: Warm-up effect skews micro-benchmarks

**What goes wrong:**
First `Parse` call on a fresh `Parser` triggers simdjson runtime CPU dispatch (loads the implementation pointer), allocates tape buffer at peak size. Go benchmark framework amortizes by running `b.N` iterations — but `b.N=1` runs (small fixtures) show an artificial spike. Also: the first run of a `.so` across `dlopen` has symbol resolution cost counted in the benchmark.

**Why it happens:**
JIT-like one-time costs. Standard for SIMD libraries.

**How to avoid:**
- Benchmark convention: `b.ResetTimer()` after a warm-up `Parse` call on each `Parser`. Document this in the benchmark file header.
- Separate "cold start" benchmark that measures `dlopen` + first-parse explicitly. Report separately.
- Use `testing.B.ReportAllocs()` and `b.ReportMetric(bytesPerSec, "MB/s")` for GB/s throughput claims.

**Warning signs:**
- Benchmarks with `b.N=1` spiking reveal missing warm-up.

**Phase to address:** Benchmarking phase.

**Severity:** P1

---

### Pitfall 26: Go allocation counters misleading when work is in C

**What goes wrong:**
`-benchmem` shows `0 allocs/op` for pure-simdjson because all work is in Rust/C++ heap, invisible to Go's allocator. Reader thinks "zero allocations!" — actually hundreds of MB churned per second by simdjson's arena. Misleading.

**Why it happens:**
Go's `runtime.MemStats` doesn't see cgo/purego-side allocations.

**How to avoid:**
- Surface native allocator stats: expose `pure-simdjson.NativeAllocStats()` if the Rust shim tracks them (implement via a counting allocator or periodic mallinfo).
- Benchmark output includes both Go allocs/op AND a note: "Native allocator activity: X MB churned per sec (measured separately)."
- Comparison with `minio/simdjson-go` helps here — that library is pure Go so its allocs are visible. Apples-to-apples forces honesty.

**Phase to address:** Benchmarking phase.

**Severity:** P1

---

## Moderate pitfalls (P2 — quality-of-life, future maintenance burden)

### Pitfall 27: `SyscallN` vs `RegisterFunc` performance differential

**What goes wrong:**
`purego.SyscallN` involves per-call reflection. `purego.RegisterFunc` pre-compiles a typed trampoline. The difference matters at parser hot-loop rates (millions of FFI calls/sec in visitor mode).

**How to avoid:**
- Use `RegisterFunc` for every FFI entry point. Never raw `SyscallN`.
- Benchmark the FFI overhead itself (a no-op shim function) to quantify per-call cost.

**Warning signs:**
- Grep: any `purego.SyscallN` in non-test code = review.

**Phase to address:** FFI implementation phase.

**Severity:** P2

---

### Pitfall 28: cmake generator mismatch on Windows

**What goes wrong:**
Building simdjson with the default cmake generator on Windows picks whatever MSVC it finds. CI job uses `Visual Studio 17 2022`, developer uses `Ninja`, third person uses MinGW — three different `.dll`s with three different ABIs. Artifacts stored in R2 may not match what a local dev gets.

**How to avoid:**
- Pin cmake generator in CI: `cmake -G "Visual Studio 17 2022" -A x64` for windows/amd64. Document in CONTRIBUTING.
- MSVC (not MinGW) for Windows release artifacts — broader ABI compatibility with Go's Windows build environment.
- Release artifact naming includes the toolchain: `libsimdjson_shim-windows-amd64-msvc.dll`. Clear provenance.

**Warning signs:**
- CI release job fails if toolchain env variables drift.

**Phase to address:** Build-matrix phase.

**Severity:** P2

---

### Pitfall 29: Windows long paths in CI build

**What goes wrong:**
Rust + cargo + cmake on GH Actions Windows runner with deep nested `target/` dirs hits the 260-char `MAX_PATH` limit. Errors are cryptic (`The system cannot find the path specified`).

**How to avoid:**
- Enable long paths: `git config --global core.longpaths true`, set `LongPathsEnabled` registry key on CI runner, use `cargo --target-dir C:\t` short path.
- Document in CONTRIBUTING.

**Phase to address:** Build-matrix phase.

**Severity:** P2

---

### Pitfall 30: macOS universal vs thin binaries

**What goes wrong:**
Shipping a universal binary (amd64 + arm64 in one `.dylib` via `lipo`) doubles artifact size. Shipping thin binaries per arch matches the six-platform matrix cleanly but is two downloads.

**How to avoid:**
- Ship **thin** dylibs per architecture (matches PROJECT.md's six-platform matrix). The download path selects by runtime `runtime.GOARCH`.
- Do not use `lipo` — simpler and smaller.

**Phase to address:** Build-matrix phase.

**Severity:** P2

---

### Pitfall 31: Finalizer relied on as primary cleanup

**What goes wrong:**
PROJECT.md already calls this out ("no finalizer-only reliance"), but it's easy to slip: a `Doc` wrapper that only has `runtime.SetFinalizer`. Finalizers run at unpredictable times — handles can outlive GC in ways that starve the native allocator.

**How to avoid:**
- Every native-resource wrapper has an explicit `Close()`. Finalizer is a safety net that logs a warning ("Doc closed by finalizer — likely leak") in test builds, no-op in release, never the primary path.
- Linter: any `SetFinalizer` without a corresponding `Close` method in the same type = warning.

**Phase to address:** API design phase.

**Severity:** P2

---

### Pitfall 32: Visitor panic leaves parser state corrupted

**What goes wrong:**
Go user's visitor callback panics mid-traversal (e.g., panic in their own code). Shim catches it, returns error, but the parser's internal iterator is now in an undefined state. Next `Parse` call on that parser may misbehave.

**How to avoid:**
- After any visitor-bubbled error, mark the parser "poisoned" at the FFI level. Further operations return `ERR_PARSER_POISONED`. User must `Close()` and create a new parser.
- Alternative: reset parser state on error — but this is simdjson-version-dependent and easy to get wrong. Poison is safer.

**Phase to address:** FFI implementation phase.

**Severity:** P2

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Single opaque `GetNumber()` returning `any` | Simple API, one getter | Erases the precision-preserving value prop (Pitfall 12); migrating callers later = breaking change | Never |
| `runtime.SetFinalizer` as primary cleanup | No `Close()` boilerplate for users | Pitfall 31; handles outlive GC; native allocator pressure | Never for v0.1 |
| Direct C++ binding (skip Rust shim) | One less layer | C++ name mangling unstable; no `catch_unwind`; C++ exceptions leak (Pitfall 5); PROJECT.md already decided against | Never — decided |
| Download to `$TEMP` with predictable name | Simple | DLL hijacking (Pitfall 16); TOCTOU on shared systems | Never |
| Ship `latest` URL alias | Zero-ceremony user upgrades | Pitfall 18; breaks existing deploys | Never |
| `-march=native` at build time | Max single-box perf | Pitfall 13; binary only runs on builder CPU family | Never for distributed artifacts |
| Skip checksum verification | Simpler bootstrap code | Pitfall 17; MITM + compromised bucket = RCE | Never |
| Hand-written FFI signatures (no generated bindings) | Fewer build-time deps | Signature drift between Rust and Go; Pitfall 8, 9 silent on mismatches | Acceptable if CI signature-match test exists |
| Ship glibc-only Linux | Smaller matrix (6 platforms) | Pitfall 21; Alpine users bring-your-own | Acceptable only if clearly documented + `PURE_SIMDJSON_LIB_PATH` works |
| No `Kernel()` introspection API | Smaller surface | Pitfall 13; users can't diagnose slow runs | Acceptable for v0.1 if clearly roadmapped |

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| CloudFlare R2 | Assume R2 URL is stable — bake CNAME into URL | Use a custom domain (`binaries.amikos.tech`) CNAME'd to R2; switchable if R2 has an outage |
| CloudFlare Workers (if fronting R2) | Worker returns HTML error pages on failure | Worker returns 404/5xx with no body; Go client checks status code, not body content |
| GH Actions macOS runners | Use `macos-latest` | Pin `macos-14` (arm64) and `macos-13` (amd64); `latest` drift breaks reproducibility |
| GH Actions Windows runners | Use default `bash` which is MSYS | Use `pwsh` for native tooling, `bash` only for cross-compatible scripts |
| pure-tokenizers conventions | Copy the error-code values exactly | Define pure-simdjson's own error-code space; shared error codes across libraries invite confusion |
| `go.work` in downstream consumers | Replace `pure-simdjson` with local fork, but forget to update artifact download path | Go replace directive + override `PURE_SIMDJSON_LIB_PATH` at the same time; document together |

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Per-parse parser allocation | Throughput plateaus well below simdjson's claimed GB/s | Parser pool / per-goroutine parser; expose `sync.Pool`-ready `Parser` | >100 parses/sec of small docs |
| Unnecessary string copies from FFI | CPU time in Go `copy`, not parse | Zero-copy string view tied to `Doc` lifetime (v0.2); copy only on `.String()` coercion | Any parse-heavy hot path |
| Creating `purego.NewCallback` per visit | Callback table leaks (Pitfall 7) | One callback per type, registered at init | Exhausts after ~2000 parses |
| Single-threaded parse of huge corpus | One core saturated, cluster idle | NDJSON streaming parallel parse (v0.2) or caller shards | >1 GB JSON stream, multi-core machine |
| Regex-based path extraction | CPU in Go regex instead of SIMD | On-Demand pre-declared paths (v0.2) | Complex path queries |
| Re-parsing the same buffer for different fields | 2× parse cost | Parse once, use selective accessors; or visitor | Large doc, multiple consumers |

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Dlopen unsigned dylib on macOS arm64 | Gatekeeper refuses to load OR (if bypassed) user runs unattested code | Pitfall 15: sign + notarize OR strip quarantine xattr from known-good-hash download |
| Load DLL from bare filename on Windows | Pitfall 16: DLL hijacking via CWD | Always full path; per-user non-world-writable directory |
| Skip SHA-256 on downloaded binary | Pitfall 17: MITM via corporate proxy, compromised bucket | Embed hash table in Go source, verify pre-dlopen |
| Accept `http://` URL for binary mirror | Plaintext MITM trivial | Force `https://` for `PURE_SIMDJSON_BINARY_MIRROR`; reject http schemes |
| Use `os.TempDir()` for binary storage | World-readable, symlink attacks, hijacking | Use user-cache dir (`os.UserCacheDir`) with perm 0700 on unix |
| Parse untrusted JSON without bounded input size | OOM via 10 GB input → tape buffer blows up | Expose max-input-size guard; return `ErrInputTooLarge` above threshold |
| Pass Go slice to FFI without copy, rely on GC | Pitfall 2 | Copy into Rust-owned buffer |
| Log raw user JSON in errors | PII leak in error strings | Error messages reference offset/type only, never content |

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| "CPU not supported" as only error | User has no idea what to do | Error includes detected CPU class and required minimum (e.g., "need SSE4.2, got: no SSE4.2 on this CPUID") |
| Binary download failure with `dial tcp: i/o timeout` | User can't tell if it's DNS, firewall, R2 outage | Error wraps with "could not download <URL>; set PURE_SIMDJSON_LIB_PATH to use a pre-downloaded binary" |
| Silent fallback to slow kernel | Users benchmark and find pure-simdjson "barely faster than encoding/json" | Log `Kernel=fallback` loudly on first parse; or fail per PROJECT.md decision |
| `Doc` methods return `(value, error)` but user ignores error | Silent garbage values | Make missing-field return a sentinel `ErrKeyNotFound` that vets visibly; do not return zero-value + nil |
| Upgrade path: new version requires re-downloading binary without warning | First parse after upgrade pauses 5s | Changelog entry + docstring on new version explicitly notes binary change |
| Handle used after `Close()` | Panic or, worse, use-after-free | Pitfall 10: generation-stamped handles return `ErrHandleClosed` cleanly |

## "Looks Done But Isn't" Checklist

- [ ] **Handle lifecycle**: often missing generation-stamp double-free protection — verify `TestDoubleClose` + `go test -race` clean
- [ ] **FFI boundary**: often missing `catch_unwind` on one function — verify grep finds every `pub extern "C" fn` wrapped
- [ ] **Number API**: often missing overflow errors on `GetInt64` of `u64::MAX`-value — verify boundary-value unit tests for all getters
- [ ] **Binary distribution**: often missing SHA-256 verification — verify corrupted-download test fails closed
- [ ] **Platform matrix**: often missing Alpine / musl runtime test — verify Alpine CI job runs and passes
- [ ] **Platform matrix**: often missing old-glibc compat check — verify `objdump -T` asserts glibc ≤ 2.17
- [ ] **macOS signing**: often missing `codesign -dv` verification on release — verify CI gate
- [ ] **CPU dispatch**: often missing `Kernel()` returned-name assertion per platform — verify kernel-name integration test
- [ ] **UTF-8**: often missing malformed-UTF-8 corpus fuzz — verify seeded corpus + error-path coverage
- [ ] **On-Demand** (v0.2): often missing double-read guard — verify explicit test reads same field twice, expects error
- [ ] **Visitor API**: often missing panic-in-callback recovery — verify test panics inside visitor, expects error from `Parse`
- [ ] **Benchmarks**: often missing apples-to-apples tier labeling — verify README explains what each bench measures
- [ ] **Benchmarks**: often missing `b.ResetTimer()` after warm-up — verify every Benchmark file has it
- [ ] **Cold start**: often missing `BootstrapSync()` + `PURE_SIMDJSON_LIB_PATH` — verify air-gapped container integration test
- [ ] **Version pinning**: often missing ABI-version check at load — verify mismatched artifacts error cleanly, no segfault
- [ ] **Documentation**: often missing explicit "Parser reuse invalidates prior Doc" warning — verify doc-comment on `Parse()`
- [ ] **Input ownership**: often missing "we copy your input" line in docs — verify doc-comment on `Parse([]byte)`
- [ ] **linux/arm ABI**: often missing EABIHF assertion — verify `readelf -A` check in release pipeline

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Parser reuse corrupts Doc (Pitfall 1) | HIGH | Re-parse; invalidate all prior handles; audit calling code for retained Docs |
| Input buffer freed while Doc alive (Pitfall 2) | HIGH | Re-parse with fresh buffer; ship hotfix that force-copies if shim was relying on pinning |
| Missing SIMDJSON_PADDING (Pitfall 3) | MEDIUM | Hotfix: Rust shim copies with padding; re-release |
| Unhandled panic crosses FFI (Pitfall 4) | LOW | Add `catch_unwind` wrapper; re-release patch |
| On-Demand double-read abort (Pitfall 6) | MEDIUM | Document "consumed on first read"; add consumption tracking; re-release |
| Callback leak (Pitfall 7) | MEDIUM | Switch to cursor API; deprecate visitor; migration guide |
| Struct return breaks Windows (Pitfall 8) | HIGH | Break FFI ABI to remove struct returns; bump artifact version path; require client upgrade |
| Handle double-free (Pitfall 10) | MEDIUM | Add generation counter; re-release; older clients get `ErrInvalidHandle` not crash |
| Kernel=fallback on user machine (Pitfall 13) | LOW | Expose `Kernel()`; document CPU floor; user can detect |
| Old glibc load failure (Pitfall 14) | HIGH | Rebuild with older glibc target; re-release; old binaries remain at old URL |
| macOS notarization fail (Pitfall 15) | MEDIUM | Sign + notarize; users re-download; provide `xattr -d` escape hatch |
| R2 outage | MEDIUM | Docs point users to `PURE_SIMDJSON_LIB_PATH`; stand up mirror |
| Corporate firewall blocks R2 (Pitfall 20) | LOW | `PURE_SIMDJSON_LIB_PATH` + `BINARY_MIRROR` env vars already exist |
| Alpine users fail to load (Pitfall 21) | HIGH if not shipped | Ship musl artifacts (1-sprint addition); interim: `PURE_SIMDJSON_LIB_PATH` |
| Benchmark unfair comparison (Pitfall 24) | MEDIUM | Publish revised benchmark with tiers; post-mortem in blog |
| ABI version mismatch at runtime (Pitfall 18) | LOW | Error message already clear; user re-installs matching binary |

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| 1 Parser-reuse invalidates Doc | FFI design | Handle-generation scheme in ABI contract; integration test for invalidation |
| 2 Input buffer ownership | FFI design | Rust-owned copy is default path; stress test with `GOGC=1` |
| 3 SIMDJSON_PADDING | FFI implementation | Shim allocates `len + PADDING`; page-boundary fuzz |
| 4 Rust panic across FFI | FFI implementation | `ffi_fn!` macro; grep CI check |
| 5 C++ exceptions | FFI implementation | Only `.get(err)` form; top-level `catch (...)`; allocation-failure test |
| 6 On-Demand double-read | v0.2 On-Demand | Consumption tracking; double-read test |
| 7 Callback / visitor | API design | Cursor API choice OR fixed callback pool; leak benchmark |
| 8 Struct-by-value return | FFI design | No struct returns rule; 6-platform FFI smoke test |
| 9 FP+int mixed args | FFI design | Signature review rule; 6-platform FFI smoke test |
| 10 Double-free handles | FFI design | Generation stamps; `TestDoubleClose -race` |
| 11 UTF-8 validation | FFI design | Validating accessors; malformed fuzz corpus |
| 12 Number type ambiguity | API design (v0.1) | No `GetNumber`; boundary-value unit tests |
| 13 CPU kernel dispatch | Build / API | No `-march=native`; `Kernel()` API; per-platform kernel integration test |
| 14 glibc / ABI compat | Build-matrix | manylinux/zig cc; `objdump -T` glibc ≤ 2.17; EABIHF assertion |
| 15 macOS codesign | Release | `codesign -s` + notarize; CI gate on `codesign -dv`; clean-VM e2e |
| 16 DLL hijacking | Distribution (bootstrap) | Full-path `LoadLibrary`; per-user non-world-writable dir; Windows decoy test |
| 17 Binary integrity | Distribution | SHA-256 table in Go source; verify pre-dlopen; corrupted-download test |
| 18 Version pinning | Distribution + API versioning | Semver-in-URL; ABI version check at load; version-mismatch test |
| 19 Cold-start latency | Distribution | `BootstrapSync()`; `PURE_SIMDJSON_LIB_PATH`; cold-start benchmark |
| 20 Corporate egress | Distribution | `PURE_SIMDJSON_LIB_PATH` + `BINARY_MIRROR`; air-gapped integration test |
| 21 glibc vs musl | Build-matrix | Alpine smoke test; decide YES to shipping musl artifacts |
| 22 libstdc++ ABI | Build-matrix | `-static-libstdc++ -static-libgcc`; `nm` exports only `extern "C"` |
| 23 arm64 page size | FFI implementation / v0.2 | No literal 4096; macOS 16K page coverage; optional 64K-page CI |
| 24 Unfair benchmarks | Benchmarking | Three-tier harness; peer review; compare vs minio/simdjson-go |
| 25 Warm-up skew | Benchmarking | `b.ResetTimer()` convention; separate cold-start bench |
| 26 Go alloc counters | Benchmarking | Native alloc stats surfaced; comparison with minio/simdjson-go honest |
| 27 SyscallN vs RegisterFunc | FFI implementation | Use `RegisterFunc`; grep for `SyscallN` |
| 28 cmake generator | Build-matrix | Pin generator; artifact name includes toolchain |
| 29 Windows long paths | Build-matrix | Enable longpaths on CI; document |
| 30 Universal dylibs | Build-matrix | Thin per-arch; no `lipo` |
| 31 Finalizer as primary | API design | Explicit `Close()`; finalizer = warn only |
| 32 Visitor poisons parser | FFI implementation | Poison flag on FFI error; test |

## Phase Groupings (for roadmap authoring)

- **API design phase**: Pitfalls 7, 12, 24-relevant, 31
- **FFI design phase**: Pitfalls 1, 2, 8, 9, 10, 11
- **FFI implementation phase**: Pitfalls 3, 4, 5, 23, 27, 32
- **Build-matrix phase**: Pitfalls 14, 21, 22, 28, 29, 30
- **Release/distribution phase**: Pitfalls 15, 16, 17, 18, 19, 20
- **Benchmarking phase**: Pitfalls 24, 25, 26
- **v0.2 (deferred) phase**: Pitfalls 6, 23 (mmap path)
- **Cross-cutting** (every phase touches): 13 CPU dispatch

## Sources

- [simdjson: basics.md — parser reuse, document lifetime, padding, UTF-8, runtime dispatch](https://github.com/simdjson/simdjson/blob/master/doc/basics.md)
- [simdjson: On-Demand design — single-shot iteration, number types, ordering](https://simdjson.org/api/0.6.0/md_doc_ondemand.html)
- [simdjson Issue #1246 — moving parser invalidates elements](https://github.com/simdjson/simdjson/issues/1246)
- [simdjson Issue #938 — segfault when reusing parser inside loop](https://github.com/simdjson/simdjson/issues/938)
- [simdjson Discussion #2195 — what happens without padding](https://github.com/simdjson/simdjson/discussions/2195)
- [simdjson Issue #906 — buffer overrun on incomplete document](https://github.com/simdjson/simdjson/issues/906)
- [purego pkg.go.dev — RegisterFunc and NewCallback limitations](https://pkg.go.dev/github.com/ebitengine/purego)
- [purego HN comment from contributor on goroutine stack handling](https://news.ycombinator.com/item?id=34764450)
- [Rust Nomicon — FFI unwinding, catch_unwind, C-unwind ABI](https://doc.rust-lang.org/nomicon/ffi.html)
- [RFC 2945 — C-unwind ABI semantics](https://rust-lang.github.io/rfcs/2945-c-unwind-abi.html)
- [cxx.rs — result / exception bridging](https://cxx.rs/binding/result.html)
- [Microsoft Learn — DLL search order and LoadLibraryEx safe flags](https://learn.microsoft.com/en-us/windows/win32/dlls/dynamic-link-library-search-order)
- [Microsoft Support — secure loading of libraries](https://support.microsoft.com/en-us/topic/secure-loading-of-libraries-to-prevent-dll-preloading-attacks-d41303ec-0748-9211-f317-2edc819682e1)
- [Linux Kernel docs — AArch64 memory layout and page sizes](https://docs.kernel.org/arch/arm64/memory.html)
- [pypa/manylinux Issue #735 — inconsistent page size on arm64](https://github.com/pypa/manylinux/issues/735)
- pure-tokenizers, pure-onnx, fast-distance reference repos (per PROJECT.md) for established handle/lifecycle/distribution conventions
- Training knowledge cross-referenced against above — anywhere training alone supported a claim, it was flagged or dropped

---
*Pitfalls research for: pure-simdjson (Go↔Rust↔C++ FFI + purego + multi-platform binary distribution)*
*Researched: 2026-04-14*
