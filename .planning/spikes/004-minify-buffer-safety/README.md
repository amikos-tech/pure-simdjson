---
spike: 004
name: minify-buffer-safety
type: standard
validates: "Given dst allocated exactly len(src) bytes (no SIMDJSON_PADDING slack), when simdjson::minify() runs directly and with dst aliasing src, then no out-of-bounds write occurs and in-place output matches the non-aliased reference"
verdict: VALIDATED
related: []
tags: [simdjson, minify, memory-safety, ffi, phase-12]
---

# Spike 004: Minify Buffer Safety

## What This Validates

Phase 12's locked `12-CONTEXT.md` decision (D-08/D-09) ships `MinifyInto(dst, src []byte) (int, error)` where `dst` may alias `src` for in-place minification, sized only `>= len(src)` — no extra padding required from the caller.

Upstream's own doc comments disagree on the buffer contract:

- The internal per-kernel virtual (`third_party/simdjson/singleheader/simdjson.h:3946`): *"`dst` MUST be allocated up to `len + SIMDJSON_PADDING` bytes."*
- The public free function `simdjson::minify()` (same file, ~line 4451): *"`dst` MUST be allocated up to `len` bytes."*

This spike determines which contract the actual v4.6.4 implementation honors, and specifically whether `dst == src` (a case upstream's own fuzzers never test — see Research) is safe and correct.

## Research

| Approach | Pros | Cons | Status |
|----------|------|------|--------|
| Trust the internal virtual's doc comment (`len + SIMDJSON_PADDING`) and always require callers to over-allocate | Conservative, can't be wrong | Breaks the "dst may alias src, sized exactly len(src)" story the roadmap explicitly asks for; forces every caller to think about padding for a supposedly simple utility | Rejected without evidence |
| Trust the public free function's doc comment (`len` only) at face value | Matches the roadmap's aliasing goal | Just as unverified as the other comment — the two upstream comments directly contradict each other | Insufficient alone |
| Read upstream's own fuzz harnesses | `fuzz/fuzz_minifyimpl.cpp` calls `impl->minify(Data, Size, ret.data(), retsize)` — the exact *internal* virtual the padding comment warns about — against a `std::vector<uint8_t> ret(Size)`, i.e. a buffer of **exactly** `Size` bytes, not `Size + SIMDJSON_PADDING`. This has been OSS-Fuzzed for years across all compiled-in kernels (see the harness's own multi-implementation consistency check) | Doesn't cover the `dst == src` aliasing case at all — `Data` (src) is `const uint8_t*` and is never passed as `dst` in that harness | Strong prior evidence the padding comment is stale, but the aliasing case remained genuinely untested by upstream |
| Build our own probe, both non-aliased exact-size and `dst == src` aliased, under ASan+UBSan | Directly answers the open question (aliasing), independently confirms the non-aliased case, uses the exact vendored v4.6.4 pin | Only covers kernels buildable on this host (see Limitations) | **Chosen** |

## How to Run

```bash
bash .planning/spikes/004-minify-buffer-safety/verify.sh
```

Builds `probe.cpp` against the **untouched** vendored `third_party/simdjson/singleheader/{simdjson.h,simdjson.cpp}` in a `mktemp` scratch directory, compiled with `-fsanitize=address,undefined` and `-DSIMDJSON_IMPLEMENTATION_FALLBACK=1` (the same macro `build.rs` uses in production, so the scalar `fallback` kernel is included alongside the host's native kernel). Runs the resulting binary 3 times and fails if any run traps, any case fails, or the three runs aren't byte-identical.

## What to Expect

```
==> VERIFIED: 24/24 cases safe, correct, and deterministic across 3 runs
```

12 fixtures × 2 kernels (`arm64`, `fallback`, both buildable/runnable on this Apple Silicon host) = 24 cases. Each case runs both the non-aliased (separate exact-size `dst`) and aliased (`dst == src`, same exact-size buffer) calls, plus a correctness comparison between them.

## Investigation Trail

1. **Read upstream doc comments** — found the internal-virtual-vs-public-function contradiction described above.
2. **Read upstream's fuzz harnesses** (`fuzz/fuzz_minifyimpl.cpp`, `fuzz/fuzz_minify.cpp`) — found years of OSS-Fuzz coverage calling the internal virtual with an exact-`Size` buffer, but confirmed neither harness ever aliases `dst` with the source bytes. This reframed the real open question from "does minify overflow at all" (upstream's fuzzer already suggests no) to "is `dst == src` specifically safe" (upstream never tests this).
3. **Built `probe.cpp`** — links directly against the vendored singleheader, calls `implementation::minify` (the internal virtual, matching the fuzz harness's own target) across every kernel `get_available_implementations()` reports as supported on the host, with 10 initial fixtures (nested structures, escaped strings including a backslash-quote-tab-newline mix, wide Unicode/emoji, and sizes straddling the `SIMDJSON_PADDING = 64` boundary at 63/64/65/128 bytes).
4. **First run**: only `arm64` appeared in `get_available_implementations()` — `fallback` was silently absent. Traced this to a missing `SIMDJSON_IMPLEMENTATION_FALLBACK` define; checked `build.rs` and found production explicitly sets `.define("SIMDJSON_IMPLEMENTATION_FALLBACK", "1")`. Recompiled with the same macro so the spike doesn't test a different implementation surface than production ships. `fallback` then appeared and passed identically.
5. **Stress-tested the aliasing hypothesis harder**: added two large, high-compression-ratio fixtures (~18KB heavily-whitespace-padded array, ~25KB array of escaped nested objects) specifically because in-place minification is riskiest when the write pointer lags far behind the read pointer for long stretches — that's the scenario where a SIMD lookahead read could plausibly observe bytes the write side already clobbered, if such a bug existed. Both kernels handled both large fixtures with zero traps and byte-identical aliased-vs-reference output.
6. **Repeated 3x** per `CONVENTIONS.md` — identical SHA-256 digest (`f777b8ef...`) across all three runs, confirming no data-race or uninitialized-memory nondeterminism.

No surprising failures were found. The one real surprise was upstream's own doc-comment contradiction and the fact that its multi-year fuzz coverage never exercised the aliasing case we actually need.

## Limitations

- Only `arm64` and `fallback` kernels were tested — this is an Apple Silicon (M3 Max) host, so `haswell`/`westmere`/`icelake` (x86-64 kernels) could not be compiled or executed here. Production ships all five platforms; the x86-64 kernels use different SIMD widths (AVX2/SSE) than NEON, so this residual risk should be closed by running the same `probe.cpp`/`verify.sh` (or an equivalent CI job) on an x86-64 runner during Phase 12 execution, ideally as part of the existing five-platform CI matrix rather than as a one-off.
- ASan/UBSan detect this class of memory-safety bug reliably (redzone-based heap-buffer-overflow detection is deterministic, not probabilistic like TSan for races), so a clean run here is strong evidence, not merely "didn't crash this time."

## Results

**Verdict: VALIDATED.**

| Observation | Evidence |
|---|---|
| Non-aliased, exact-size `dst` (no `SIMDJSON_PADDING` slack) | 0 ASan/UBSan traps across 12 fixtures × 2 kernels |
| Aliased `dst == src`, exact-size buffer | 0 ASan/UBSan traps across the same 12 fixtures × 2 kernels |
| Aliased output correctness | Byte-identical to the non-aliased reference in all 24 cases, including the two large/adversarial fixtures |
| Determinism | 3 consecutive runs produced an identical SHA-256 digest |

The public `simdjson::minify()` contract's `len`-only requirement (not `len + SIMDJSON_PADDING`) holds for both kernels tested, and `dst == src` in-place aliasing is safe and produces correct output. **Phase 12's `MinifyInto(dst, src []byte) (int, error)` decision (D-08/D-09) stands as locked — no CONTEXT.md amendment needed** for the arm64/fallback kernels tested here.

**Follow-up for the execution plan, not a blocker:** re-run this same probe (or fold it into the Rust/C++ contract test suite) on an x86-64 CI runner before claiming the aliasing guarantee across all five shipped platforms — see Limitations.
