# Stack Research

**Domain:** Go library wrapping a native C++ dependency via a Rust FFI shim + purego; pre-built shared libraries distributed through CloudFlare R2
**Researched:** 2026-04-14
**Confidence:** HIGH on the pattern (three shipping reference repos validate it); MEDIUM on simdjson-specific API choice; LOW on `linux/arm` viability (see Blocker below)

## TL;DR (Prescriptive)

- **Go side:** `go 1.24`, `github.com/ebitengine/purego v0.10.0`, `github.com/Masterminds/semver/v3 v3.4.0`, `github.com/pkg/errors v0.9.1` — identical to the pure-tokenizers/pure-onnx dependency set.
- **Rust shim:** `edition = "2021"`, `crate-type = ["cdylib", "staticlib"]`, raw `#[no_mangle] extern "C"` + `#[repr(C)]` structs, `cbindgen 0.29.x` for header gen, `cc 1.2.x` to build bundled simdjson, `libc 0.2` for `c_char`/`c_int`. Follow pure-tokenizers `src/lib.rs` 1:1 for the error-code / handle-out-param convention.
- **C++ dep:** simdjson **v4.6.1**, compiled as a static archive in-tree via `cc` crate. Use the **On-Demand** API with a reused `ondemand::parser` + `padded_string` input. Require C++17 (simdjson's floor is C++11 but On-Demand reads cleaner in C++17).
- **Distribution:** Reuse pure-tokenizers' R2 layout exactly: `releases.amikos.tech/pure-simdjson/<rust-vX.Y.Z>/libsimdjson-<target>.tar.gz` + `SHA256SUMS` + `latest.json` + cosign keyless signatures. Use the same `rust-release.yml` + composite actions verbatim.
- **Build matrix:** `cross` on Linux (native/musl), native `cargo build --release` on macOS + Windows, reusing pure-tokenizers' composite actions. `cargo-zigbuild` is a fallback lever, not the default.
- **Benchmarking:** standard `testing.B`, `go test -bench=. -benchmem -count=10`, corpus from `simdjson/jsonexamples` (twitter.json, canada.json, citm_catalog.json, mesh.json, numbers.json). Compare against `encoding/json`, `minio/simdjson-go` (amd64 only), `bytedance/sonic`, `goccy/go-json`. `benchstat` for deltas.

---

## BLOCKER: linux/arm (32-bit) and pure-Go consumption are incompatible

This is the single most important finding — it directly contradicts a stated v0.1 requirement.

From the purego v0.10.0 README platform table (verified 2026-04):

> Tier 2 — Linux: 386, **arm**, loong64, ppc64le, riscv64, s390x¹
> ¹ These architectures require `CGO_ENABLED=1` to compile.

Meaning: on `linux/arm` (armv6/armv7 32-bit), purego itself needs cgo at **consumer** build time. The whole value proposition of the `pure-*` family — "downstream Go projects build without cgo" — does not hold on this target. Additionally, footnote 4 notes `windows/arm` "No longer supported as of Go 1.26" which is fine because it wasn't in our target list, but worth logging.

**Options (call this out in phase planning, don't silently pick):**

1. **Drop `linux/arm` from v0.1.** Match the pure-tokenizers release matrix, which already omits `linux/arm` and `linux/arm-musl`. Cleanest; keeps the no-cgo promise intact. Recommended.
2. **Ship `linux/arm` binary, document that consumers need `CGO_ENABLED=1` on that platform.** Breaks the "no cgo" promise for that one target. Acceptable only if a specific downstream consumer demands it.
3. **Wait for purego to drop the cgo requirement on linux/arm.** Not on their roadmap as of v0.10.0.

Confidence: HIGH. Sourced from purego's README platform table and footnotes.

## Recommended Stack

### Core Technologies

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| Go (toolchain) | 1.24.x | Consumer language & build | Matches pure-tokenizers (`go 1.24`) and pure-onnx (`go 1.24.0`). `runtime.Pinner` (Go 1.21+) is used in fast-distance for stable pointer passing across purego calls — we will need the same pattern for passing `[]byte` buffers to the parser. |
| Rust | stable (1.85+) | FFI shim language | Cross-rs MSRV is 1.85. cxx 1.0.194 requires recent stable. Matches pure-tokenizers toolchain. |
| simdjson (C++) | **v4.6.1** | Actual JSON parser | Latest stable (released April 2025). C++11 minimum, C++17 recommended. Released regularly (v4.5.0, v4.6.0, v4.6.1 all within one week in April 2025 — stable and active). |
| purego | **v0.10.0** | Go ↔ shared library calls without cgo | Adds struct-on-Linux support and Linux 386/arm/ppc64le/riscv64/s390x (Tier 2). **Upgrade from pure-tokenizers' v0.8.4 to v0.10.0**: struct passing was immature in 0.8.x, we need it for `Result`/`StringView` out-params. Reference: pure-onnx already pins v0.10.0. |
| cbindgen | 0.29.2 | Generate `libsimdjson.h` from the Rust shim | Used by both pure-tokenizers (0.29.0) and fast-distance (0.29). Latest is 0.29.2 (April 2024). Run from `build.rs`. |
| cc (crate) | 1.2.60+ | Compile bundled simdjson C++ sources in `build.rs` | Latest 1.2.60 (April 2025). Standard Rust toolchain for pulling in C/C++. Handles `-std=c++17`, optimization flags, and cross-toolchain detection automatically. |
| libc (crate) | 0.2.174+ | `c_char`, `c_int`, etc. in FFI signatures | Matches pure-tokenizers exactly. |

**Crate-type choice (HIGH confidence, validated by both reference repos):**

```toml
[lib]
crate-type = ["cdylib", "staticlib"]
```

`cdylib` produces the `.so`/`.dylib`/`.dll` that purego dlopens. `staticlib` is kept for musl (Linux musl targets **do not support cdylib** — they produce `.a` and the consumer is expected to link statically; this is how pure-tokenizers' release matrix handles musl). For pure-simdjson's no-cgo goal, dropping musl's `.a` from the shipped matrix is defensible since Alpine users can install glibc-compat or use the distroless glibc image. Decide in roadmap; don't inherit pure-tokenizers' musl story blindly.

### Supporting Libraries (Go side)

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/ebitengine/purego` | v0.10.0 | Dynamic library loading + symbol dispatch | Always. Core dependency. |
| `github.com/Masterminds/semver/v3` | v3.4.0 | ABI version compatibility checks | Always. pure-tokenizers uses it to enforce an `AbiCompatibilityConstraint = "^0.1.x"` between Go wrapper and shared library. Copy this pattern. |
| `github.com/pkg/errors` | v0.9.1 | `errors.Wrapf` with stack traces | Consistency with reference repos. Note: `pkg/errors` is archived but still used across the pure-* family; do not replace without a coordinated migration. |
| `golang.org/x/sys` | v0.35.0+ | OS-level syscalls (used by pure-tokenizers for cache dir handling on Windows) | As needed. Transitively pulled in anyway. |
| `github.com/stretchr/testify` | v1.10.0+ | Test assertions | Dev-only. Reference repos use this. |

### Supporting Crates (Rust side)

| Crate | Version | Purpose | When to Use |
|-------|---------|---------|-------------|
| `cc` | 1.2.60+ | Build bundled simdjson C++ sources | `build.rs`: `cc::Build::new().cpp(true).flag("-std=c++17").file("vendor/simdjson/simdjson.cpp").compile("simdjson")`. |
| `cbindgen` | 0.29.2 | Emit C header for purego symbol expectations + consumer docs | `build.rs`, post-compile. |
| `libc` | 0.2 | `c_char`, `size_t` | Always. |

**Explicitly NOT needed:** `cxx`, `autocxx`, `bindgen`. Rationale in "What NOT to Use" below.

### Development Tools

| Tool | Purpose | Notes |
|------|---------|-------|
| `cross` (cross-rs) | Cross-compile Rust cdylib for `linux/*` targets | v0.2.5 (stable; the team has been slow to cut new releases but main is active). Used on Linux runners to produce `x86_64-unknown-linux-gnu` and `aarch64-unknown-linux-gnu` via Docker images. Matches pure-tokenizers. |
| `cargo-zigbuild` | Fallback cross-compile (uses zig cc) | Keep as escape hatch, not default. pure-tokenizers keeps `use-zigbuild` as an opt-in flag on its composite action. Zig's cross-toolchain is more forgiving for linking against newer glibc but adds a dependency. |
| `cosign` | Keyless OIDC signing of release artifacts | sigstore/cosign-installer@v3. Both pure-tokenizers and pure-onnx sign `SHA256SUMS`, each `.tar.gz`, and `releases.json` this way. |
| `aws-cli` | Upload to CloudFlare R2 (S3-compatible) | Set `R2_ENDPOINT`, `R2_ACCESS_KEY_ID`, `R2_SECRET_ACCESS_KEY` secrets. Same pattern as reference repos. |
| `benchstat` | Statistical comparison of Go benchmarks | `go install golang.org/x/perf/cmd/benchstat@latest`. Canonical for `encoding/json` vs alternatives. |
| `lefthook` | Git hooks (format/lint pre-commit) | pure-tokenizers uses it (`lefthook.yml`). Optional. |
| `golangci-lint` | Linter | Both reference repos ship `.golangci.yml`. Copy one in. |
| `make` | Build orchestration | All three reference repos use a top-level `Makefile`. Canonical targets: `lib`, `test`, `bench`, `release`. |

## Installation

```bash
# Rust shim scaffold (run once at project init)
cargo init --lib
# In Cargo.toml: [lib] crate-type = ["cdylib", "staticlib"]

# Rust deps
cargo add libc@0.2
cargo add --build cbindgen@0.29 cc@1.2

# simdjson (vendored as a git submodule at vendor/simdjson, pinned to v4.6.1)
git submodule add -b v4.6.1 https://github.com/simdjson/simdjson vendor/simdjson

# Go deps
go mod init github.com/amikos-tech/pure-simdjson
go get github.com/ebitengine/purego@v0.10.0
go get github.com/Masterminds/semver/v3@v3.4.0
go get github.com/pkg/errors@v0.9.1
go get -t github.com/stretchr/testify@latest

# Cross-compile toolchain (CI)
cargo install cross --locked
cargo install cargo-zigbuild --locked  # optional escape hatch
```

## Alternatives Considered

### Rust → C++ FFI approach

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| Raw `extern "C"` + `cc` building simdjson | `cxx` crate (v1.0.194) | If the shim needs to pass Rust `UniquePtr<T>` / C++ exception handling across the boundary. Not our case — we expose a flat C ABI to Go, and Go can't consume `cxx`'s C++ side anyway. Using `cxx` would mean: Go → C ABI → `cxx`-generated C++ shim → C++-generated Rust shim → simdjson. Three layers to solve what one `#[no_mangle] extern "C"` function solves. |
| Raw `extern "C"` + `cc` | `autocxx` (v0.29.1, last release March 2025) | If we were wrapping 100+ C++ methods and header drift was a real concern. simdjson's On-Demand surface we need is small (~15 functions). autocxx adds a `bindgen` runtime dep and pulls in LLVM; overkill. |
| Raw `extern "C"` + `cc` | `bindgen` on `simdjson.h` | simdjson's C++ headers are template-heavy and do NOT expose a C API. `bindgen` would generate unusable output. simdjson has a `simdjson.h` C++ header and a single-file `simdjson.cpp` — the canonical integration is to compile the `.cpp` directly and write the C ABI by hand. |

**Confidence:** HIGH. Validated by pure-tokenizers (wraps Rust-native `tokenizers` crate, so no C++ involved but establishes the "hand-written extern C" pattern) and cbindgen as the header generator (same as fast-distance and pure-tokenizers).

### Go JSON parser alternatives (competitors, not dependencies)

| Recommended (what pure-simdjson ships) | Alternative | When to Use Alternative |
|---|---|---|
| pure-simdjson (this project) | **`minio/simdjson-go`** (pure-Go SIMD port) | amd64-only deployments where cgo-at-build-time-of-transitive-deps (purego's `linux/arm` footnote) would ever surface. `minio/simdjson-go` has **no ARM64 support** (AVX2 + CLMUL required), peaks at ~50-60% of C++ simdjson's throughput per their own README, cannot unmarshal into structs, and is lightly maintained (v0.4.5 from March 2023 is the latest visible release). **Its existence is the strongest argument FOR pure-simdjson:** the niche of "SIMD JSON on Go, including ARM64, with authoritative simdjson perf" is open. |
| pure-simdjson | **`bytedance/sonic`** (JIT + SIMD) | Struct-unmarshaling workloads. sonic is faster than simdjson-go for typed decoding via runtime JIT codegen, but it (a) is Linux/macOS/Windows amd64 + arm64 only, (b) requires Go 1.18–1.26 (incompatible with 1.24.0 per their README), (c) uses assembly + `golang-asm` (no cgo, but fragile against Go runtime changes; sonic has broken on Go version bumps repeatedly). Complementary, not competitive: pure-simdjson targets `any`-replacement + selective-path extraction, sonic targets `json.Unmarshal(&struct)`. |
| pure-simdjson | **`goccy/go-json`** (pure Go, codegen) | Drop-in `encoding/json` replacement with no SIMD. 2–5× faster than stdlib via reflection tricks and per-type codegen cache. No parse-speed claim at multi-GB/s. Good benchmark baseline. |
| pure-simdjson | **`sugawarayuuta/sonnet`** | Pure-Go, correctness-first (proper UTF8, RawMessage validation). Modest perf over stdlib. Mention as a citation in benchmarks; not a direct competitor. |
| pure-simdjson | `encoding/json` stdlib | Anything that isn't parse-dominated. Mandatory baseline in benchmark tables. |

**Why the amikos family picked purego-over-native instead of going pure-Go:** a pure-Go rewrite of simdjson's On-Demand API would be a year of work and would never match the C++ implementation's perf (minio/simdjson-go peaks at 60%, and that's only the DOM/tape style, not On-Demand). The shim approach ships upstream's actual parser — same kernels, same benchmarks, same bug fixes — at the cost of a build-matrix and a bootstrap download. The reference repos validate that cost is manageable.

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| `cxx` crate for this project | Adds a C++ → Rust → C layer for no gain. Go cannot consume `cxx`'s generated C++ bridge. We need a flat C ABI at the shim boundary. | Raw `#[no_mangle] extern "C"` + `#[repr(C)]` structs. Pattern in `pure-tokenizers/src/lib.rs`. |
| `autocxx` | Heavy (pulls LLVM via `bindgen`), generates `cxx`-style bridges we don't need, last release March 2025 is "active" but LLM-adjacent churn risk. Our simdjson surface is small. | Hand-write ~15 `extern "C"` functions wrapping simdjson calls. |
| `bindgen` on simdjson.h | simdjson is C++ templates. `bindgen` cannot parse templates meaningfully. | Write a tiny C++ file that exposes a C ABI over the parts of simdjson you need, compile it via `cc` crate, expose via Rust `extern "C"`. Or skip the C++ middleman entirely and call simdjson from Rust using `cc`-built objects + manual extern blocks. |
| purego v0.8.x | Struct passing on Linux was incomplete before v0.9/v0.10. You need structs to return `{result, error_code}` tuples cleanly. | v0.10.0. |
| cgo in the Rust shim's build | Would defeat the no-cgo consumer promise. The shim must produce a self-contained `.so` that purego dlopens without the consumer touching cgo. | Static-link simdjson into the cdylib (`cc` crate compiles `simdjson.cpp` into the cdylib; `-Wl,-Bsymbolic-functions` to avoid symbol leakage on Linux). |
| Separate SIMD-dispatch code in our shim | simdjson already does runtime kernel dispatch internally (icelake/haswell/westmere/arm64/fallback). Re-implementing it in the shim or Go layer would duplicate what simdjson handles and contradicts the "fail loudly" goal in PROJECT.md. | Trust `simdjson::get_active_implementation()`. Expose a `simdjson_implementation_name()` FFI function that Go surfaces in errors/diagnostics. `fast-distance`'s `dispatch.go` is structurally similar but operates at a level simdjson owns internally — don't re-create it. |
| `minio/simdjson-go` as a transitive dep of any kind | Pure-Go-only path exists, but bringing it in as a fallback would (a) double binary size on amd64, (b) still fail on arm64, (c) confuse the benchmark story. | Zero dependency on it. Benchmark against it, yes. Depend on it, no. |
| `github.com/pkg/errors` for NEW code | Archived upstream. Rough edges with Go 1.20+ error wrapping. | Use `fmt.Errorf("...: %w", err)` for new wrapping. Keep `pkg/errors` where the reference repos already have it to avoid a pointless diff. |
| musl cdylib | Rust's musl targets **only** produce `.a` (staticlib) — you cannot dlopen a musl cdylib. pure-tokenizers ships `libtokenizers.a` for musl and tells users to relink. | Either (a) skip musl from v0.1 and rely on glibc everywhere, or (b) keep static-archive distribution for musl only and document it. Do NOT try to produce a musl `.so`. |

## Reference-Repo Validation Matrix

For each stack choice, which pure-* repo already does this?

| Choice | pure-tokenizers | pure-onnx | fast-distance | Novel to pure-simdjson |
|---|---|---|---|---|
| `purego` dlopen + symbol bind | ✓ (v0.8.4) | ✓ (v0.10.0) | ✓ | — |
| Rust cdylib + staticlib crate-type | ✓ | — (consumes upstream onnxruntime, no shim) | ✓ | — |
| cbindgen header gen in `build.rs` | ✓ (0.29.0) | — | ✓ (0.29) | — |
| `cc` crate building vendored C++ | — (wraps pure Rust) | — | — | **Yes** |
| Handle-based FFI lifecycle (Result + error code) | ✓ | ✓ (different naming) | ✗ (stateless fns) | — |
| `Masterminds/semver/v3` ABI constraint | ✓ | — | — | Adopt from tokenizers |
| cross-rs on Linux, native cargo on macOS/Win | ✓ | — (no Rust build, downloads upstream) | — (?) | — |
| cargo-zigbuild opt-in | ✓ | — | — | — |
| R2 upload via aws-cli, cosign keyless sign | ✓ (rust-release.yml) | ✓ (release.yml) | — | — |
| `releases.amikos.tech/{project}/{version}/` path scheme | ✓ | ✓ | — | — |
| `latest.json` + `releases.json` metadata | ✓ | ✓ | — | — |
| Bootstrap-on-first-use download in Go layer | ✓ (`download.go`) | ✓ (`bootstrap.go`) | — | — |
| ABI verification on dlopen (symbol probe + version) | ✓ (`abi_test.go`, `verifyLibraryABICompatibilityHandle`) | ✓ (`abi_version.json`) | — | — |
| `runtime.Pinner` for stable byte-slice pointers | — (strings only) | ✓ | ✓ | Will need — we pass `[]byte` into simdjson. |
| Stateful handle with mutable C++ state (parser tape) | — (tokenizer is immutable after load) | ✓ (session) | — | **Wrinkle**: simdjson's `ondemand::parser` holds a tape that is invalidated on each new parse. Caller must not hold prior `Doc` references across re-parse. pure-onnx's `session` has similar lifecycle but simdjson's is more fragile. Document loudly; consider an in-shim generation counter that invalidates stale `Doc` handles with an error code. |

## Stack Patterns by Variant

**If you need to ship `linux/arm` (32-bit) in v0.1:**
- Cannot use purego without cgo. Must either (a) document the CGO_ENABLED=1 requirement on this target only, or (b) build a cgo fallback path specifically for this arch. Both are ugly. Strongly prefer dropping the target.
- Because: purego README footnote 1 on linux/arm.

**If you need Alpine/musl support:**
- Ship `libsimdjson.a` (staticlib) for `*-unknown-linux-musl` targets, not `.so`. Consumer must relink (breaks the no-cgo promise for them).
- Alternative: drop musl from v0.1, add it in v0.2 once the dlopen-of-staticlib problem is consciously addressed.
- Because: Rust's musl targets only emit staticlib; purego needs cdylib.

**If you need Windows stdcall or WinAPI signatures:**
- Doesn't apply — we're calling into our own cdylib, which uses the default calling convention on Windows x64 (same ABI for C and stdcall on amd64). Only matters on windows/386 or arm.
- Because: pure-tokenizers has a separate `library_windows.go` but only to handle `LoadLibrary` vs `dlopen`, not calling convention differences.

**If the simdjson C++ integration is too slow to compile in CI:**
- simdjson ships as a "single-header amalgamation" — `simdjson.h` (~200k lines) + `simdjson.cpp` (~400k lines). Compile time on arm64 via `cross` is ~2–3 minutes per target. Six targets + lto = long CI.
- Mitigation: `sccache` on the CI runners; commit the amalgamation at a pinned version rather than rebuilding from a submodule each time.

## Version Compatibility

| Package A | Compatible With | Notes |
|-----------|-----------------|-------|
| purego v0.10.0 | Go 1.21+ | Struct passing on Linux requires 0.10.0. Tier 1 covers linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64, windows/arm64. |
| purego v0.10.0 | Go 1.26+ | ⚠ windows/arm "No longer supported as of Go 1.26" (footnote 4). Not in our target list, so acceptable. |
| simdjson v4.6.1 | GCC 7+, Clang 5+, MSVC 2017+ | C++17 baseline. On older compilers simdjson degrades to fallback-only kernel (no SIMD). Our CI runners (ubuntu-latest = GCC 13+, macos-latest = Clang 15+, windows-latest = VS2022) are all fine. |
| cbindgen 0.29.2 | Rust 1.57+ | No issues on stable. |
| cc 1.2.60 | Rust 1.63+ | Handles C++17 flag across platforms. |
| cross-rs v0.2.5 | MSRV Rust 1.85 | Recent bump. Check at roadmap time. |
| `Masterminds/semver/v3` v3.4.0 | Go 1.20+ | Needed for ABI constraint check (pure-tokenizers pattern). |
| cosign v2.x | GitHub OIDC | Keyless signing. Validated by pure-tokenizers/pure-onnx workflows. |

## Benchmarking stack

| Tool/Corpus | Purpose | Source |
|---|---|---|
| `testing.B` | Go-native micro/macro benchmarks | stdlib |
| `benchstat` | Statistical comparison of bench runs | `golang.org/x/perf/cmd/benchstat` — canonical across Go ecosystem |
| `twitter.json` (631 KB) | Realistic mixed-type corpus | `github.com/simdjson/simdjson/tree/master/jsonexamples` (BSD-licensed, safe to vendor) |
| `canada.json` (2.2 MB) | Float-heavy (GeoJSON polygons) | Same. Canonical "nightmare float" case across JSON benchmarks. |
| `citm_catalog.json` (1.7 MB) | Object/key heavy | Same. |
| `mesh.json` (720 KB) | Nested numeric arrays | Same. |
| `numbers.json` | Pure numeric stress | Same. |
| Comparator set | `encoding/json`, `minio/simdjson-go`, `bytedance/sonic`, `goccy/go-json` | Use `t.Skip` to exclude `simdjson-go` on non-amd64 (no ARM64). Use `t.Skip` for sonic on Go 1.24.0 if hit. |

**Convention:**
```bash
go test -bench=. -benchmem -benchtime=5s -count=10 -run=^$ ./... | tee new.txt
benchstat old.txt new.txt
```

pure-tokenizers has `.github/workflows/benchmark.yml` — study it as the pattern to copy. HIGH confidence.

## Sources

| Source | Topics | Confidence |
|---|---|---|
| `github.com/amikos-tech/pure-tokenizers` (cloned, read `Cargo.toml`, `src/lib.rs`, `src/build.rs`, `tokenizers.go`, `library_loading.go`, `download.go`, `.github/workflows/rust-release.yml`, `.github/actions/*`) | Canonical pattern for Rust cdylib + cbindgen + purego + R2 release + cosign + cross-rs + composite actions + ABI version handshake | HIGH |
| `github.com/amikos-tech/pure-onnx` (cloned, read `go.mod`, `ort/bootstrap.go`, `.github/workflows/release.yml`) | R2 release automation for a non-Rust binary, bootstrap-on-first-use with SHA256 + lockfile + 1 GiB caps, purego v0.10.0 adoption | HIGH |
| `github.com/amikos-tech/fast-distance` (cloned, read `Cargo.toml`, `build.rs`, `cbindgen.toml`, `dispatch.go`) | `runtime.Pinner` + slice-pointer passing pattern, lto/opt-level=3 release profile, minimal cbindgen config | HIGH |
| `github.com/simdjson/simdjson` GitHub releases + docs | v4.6.1 (April 2025) is latest; C++11 minimum, C++17 recommended; kernels: icelake/haswell/westmere/arm64/ppc64/lasx/lsx/fallback; On-Demand is the recommended default API; `SIMDJSON_PADDING` on input buffers | HIGH |
| `github.com/ebitengine/purego` README + releases | v0.10.0 (Feb 2026); platform tiers + footnotes; **linux/arm requires CGO_ENABLED=1**; struct passing limitations (16-byte inline threshold on some platforms; Linux struct support added in 0.10); SyscallN warning about mixed int/float args | HIGH |
| `crates.io` / GitHub for `cxx` (v1.0.194), `cbindgen` (v0.29.2), `cc` (v1.2.60), `autocxx` (v0.29.1) | Latest versions as of early 2026 | HIGH |
| `github.com/minio/simdjson-go` | amd64-only (AVX2 + CLMUL required), no ARM64, ~40–60% of C++ perf, last visible release v0.4.5 March 2023 | HIGH |
| `github.com/bytedance/sonic` | amd64 + arm64, JIT + SIMD, no cgo but uses `golang-asm`; Go 1.18–1.26 incompat with 1.24.0 | MEDIUM (self-reported) |
| `github.com/sugawarayuuta/sonnet` | Pure Go, correctness-first, modest perf | MEDIUM |
| `github.com/cross-rs/cross` | v0.2.5 stable, MSRV Rust 1.85, supports `armv7-unknown-linux-*` (all four variants) via QEMU | HIGH |

### Open questions deferred to roadmap/phase research

1. **musl handling in v0.1** — ship `.a` only, or skip musl entirely? (Stack implication: affects whether we need the dual `cdylib`/`staticlib` in Cargo.toml to actually matter for distribution.)
2. **simdjson API style** — PROJECT.md defers "DOM vs On-Demand vs tape" to phase discussion. Strong prior: **On-Demand**, because it's simdjson's recommended default, delivers the "skip unused keys" property that maps directly to v0.2's "pre-declared path set" goal, and memory profile matches "reuse parser + Doc handle" in v0.1. But the shim surface differs materially between DOM and On-Demand; locking in On-Demand affects every `FEATURES.md` row.
3. **Amalgamation vs submodule for simdjson source** — does CI compile time force us to commit `simdjson.cpp` into the repo? Measure in Phase 1.
4. **Parser/Doc handle generation counter** — do we add an epoch-style invalidation to the shim so stale `Doc` pointers return `ERR_STALE` instead of segfaulting? (On-Demand specific wrinkle; not present in tokenizers/onnx.)

---
*Stack research for: Go library wrapping C++ via Rust shim + purego, distributed via R2*
*Researched: 2026-04-14*
