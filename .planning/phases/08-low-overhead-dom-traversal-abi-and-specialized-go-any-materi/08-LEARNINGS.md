---
phase: 8
phase_name: "Low-overhead DOM traversal ABI and specialized Go any materializer"
project: "pure-simdjson"
generated: "2026-04-24"
counts:
  decisions: 7
  lessons: 7
  patterns: 7
  surprises: 5
missing_artifacts:
  - "08-UAT.md"
---

# Phase 8 Learnings: Low-overhead DOM traversal ABI and specialized Go any materializer

## Decisions

### Keep the Traversal ABI Internal
Phase 8 added `psdj_internal_materialize_build` and related frame-stream machinery as private dynamic symbols only, while keeping the generated public header free of `psdj_internal_` and `psimdjson_` exports.

**Rationale:** The phase needed one low-overhead handoff for benchmarks without committing to a public `Interface()`-style API or widening the v0.1 C ABI surface.
**Source:** 08-02-SUMMARY.md

---

### Normalize Oversized Integer Literals at Parse Time
Oversized integer literals are mapped to parse-time `PURE_SIMDJSON_ERR_INVALID_JSON`, so rejected JSON never reaches the internal materializer as BIGINT frames or partial frame streams.

**Rationale:** The public parser already rejects these inputs before a usable `Doc` or `Element` exists; preserving that behavior avoids unreachable materialization-time cases and keeps public semantics stable.
**Source:** 08-02-SUMMARY.md

---

### Use Doc-Owned Native Frame Scratch with Reentrancy Protection
The native frame stream lives in doc-owned scratch guarded by `materialize_in_progress`; nested materialization attempts return `PURE_SIMDJSON_ERR_PARSER_BUSY`.

**Rationale:** Borrowed frame pointers are only valid for the live owning document and current scratch span, so concurrent or nested builds must fail deterministically instead of corrupting shared scratch.
**Source:** 08-02-SUMMARY.md

---

### Materialize Go Values Under the Doc Mutex and Copy at Escape Boundaries
`fastMaterializeElement` holds `doc.mu` while consuming borrowed frames, copies strings and object keys into Go-owned values before return, and keeps the materializer unexported.

**Rationale:** The internal ABI returns borrowed pointers; keeping the doc locked during consumption and copying bytes at the value boundary prevents borrowed native memory from escaping to callers.
**Source:** 08-03-SUMMARY.md

---

### Treat Frame Stream Desynchronization as Internal Failure
The Go materializer rejects empty streams, under-consumed containers, and trailing frames with `ErrInternal`.

**Rationale:** Native and Go must agree exactly on the preorder frame contract; failing loudly is safer than returning a partially materialized tree when the frame stream is malformed.
**Source:** 08-03-SUMMARY.md

---

### Preserve Benchmark Row Names While Swapping the Implementation
Tier 1 pure-simdjson full and materialize-only benchmark paths now delegate to `fastMaterializeElement`, but comparator keys and diagnostic row labels remain Phase 7-compatible.

**Rationale:** Keeping row names stable lets benchstat compare Phase 8 results against committed Phase 7 diagnostics without conflating naming churn with performance changes.
**Source:** 08-04-SUMMARY.md

---

### Keep Phase 8 Evidence Internal Until Phase 9
Phase 8 committed raw diagnostics, benchstat output, and machine-gated improvement proof, but deliberately left README, public benchmark docs, changelog, release workflow, and release decisions untouched.

**Rationale:** The phase proved the internal fast path; Phase 9 owns public benchmark repositioning and any release decision based on the new evidence.
**Source:** 08-05-SUMMARY.md

---

## Lessons

### Explicit Makefile Rule Lists Can Bypass New Default Checks
Adding `no-internal-symbols` to `check_header.py` default rules was not enough because `make verify-contract` passed an explicit `--rule` list.

**Context:** Plan 08-01 had to add `--rule no-internal-symbols` to the Makefile target so the internal-prefix guard actually ran in the contract gate.
**Source:** 08-01-SUMMARY.md

---

### Fixture Boundaries Must Match Actual Parser Behavior
An initial oversized-literal fixture returned `ErrPrecisionLoss`; the test had to use `18446744073709551616` for the current parse-time `ErrInvalidJSON` boundary.

**Context:** The Wave 0 materializer tests were designed to pin existing public behavior before implementation, so the fixture had to reflect the parser's actual boundary.
**Source:** 08-01-SUMMARY.md

---

### Nested Oversized Integers Exposed a Contract Gap
Parsing `{"ok":1,"big":99999999999999999999999}` initially returned `PURE_SIMDJSON_ERR_PRECISION_LOSS`, contradicting the Phase 8 contract.

**Context:** Plan 08-02 added a parse-specific mapper from `simdjson::BIGINT_ERROR` to `PURE_SIMDJSON_ERR_INVALID_JSON`, tightening behavior before internal materialization could see such nodes.
**Source:** 08-02-SUMMARY.md

---

### Stale Native Artifacts Can Masquerade as Binding Failures
Root-package tests initially loaded an older `target/release` library that did not export `psdj_internal_materialize_build`.

**Context:** Rebuilding the native release artifact with `cargo build --release` resolved the symbol-bind failure without source changes.
**Source:** 08-03-SUMMARY.md

---

### Existing Lifecycle Checks Can Block New Busy Semantics
`fastMaterializeElement` used `doc.mu.TryLock()`, but `element.usableDoc()` called `Doc.isClosed()` with a blocking mutex lock and deadlocked under the concurrent-close test.

**Context:** Plan 08-03 changed `Doc.isClosed()` to use `TryLock()` so lock contention reaches the materializer busy guard and returns `ErrParserBusy`.
**Source:** 08-03-SUMMARY.md

---

### Benchmark Evidence Gates Need to Fail Before Closeout
The first Phase 8 same-host benchmark gate failed on the Canada rows and left evidence uncommitted.

**Context:** The failure prevented positive closeout notes until the structural performance bug was fixed and the full evidence capture was rerun.
**Source:** 08-05-SUMMARY.md

---

### Same-Host Metadata Is Part of the Benchmark Contract
The Phase 8 gate compares medians only after `goos`, `goarch`, `pkg`, and `cpu` match exactly.

**Context:** This prevents cross-host or cross-package variance from being accepted as valid Phase 7 vs Phase 8 improvement evidence.
**Source:** 08-05-SUMMARY.md

---

## Patterns

### Internal Symbol Leakage Guard
Parse public header prototypes and reject any symbol beginning with `psdj_internal_` or `psimdjson_` through a dedicated `no-internal-symbols` rule.

**When to use:** Use whenever private dynamic symbols are added for Go bindings but must not become public C ABI promises.
**Source:** 08-01-SUMMARY.md

---

### Wave 0 Behavior Tests Before Linking Production Code
Create named tests with public-accessor baseline assertions before the fast path is linked, then remove the skip guard when implementation lands.

**When to use:** Use for risky internal rewrites where behavior parity, edge cases, and review concerns need executable names before implementation begins.
**Source:** 08-01-SUMMARY.md

---

### Validate Once, Traverse Once
Validate a `ValueView` through the Rust registry, then pass the resolved doc pointer and JSON index to native traversal for a single frame-stream build.

**When to use:** Use when an internal path needs to support both root and descendant views while reusing existing handle, generation, tag, and reserved-bit validation.
**Source:** 08-02-SUMMARY.md

---

### Borrowed Frame Slice Binding
Bind the internal frame builder in `internal/ffi`, return a borrowed `[]InternalFrame` without copying, and consume it immediately under the caller's lifetime guard.

**When to use:** Use for internal-only, same-call FFI spans where copying the metadata would erase the performance win but public callers must never receive borrowed storage.
**Source:** 08-03-SUMMARY.md

---

### Full Frame Consumption Invariant
Build `any` values recursively from preorder frames and reject both under-consumed containers and trailing frames.

**When to use:** Use for stream-shaped native contracts where the consumer must detect producer/consumer desynchronization instead of accepting partial data.
**Source:** 08-03-SUMMARY.md

---

### Stable Benchmark Label Swap
Keep comparator registry keys and diagnostic row labels unchanged while swapping only the implementation under test.

**When to use:** Use when a new implementation must be compared to an existing committed benchmark baseline with benchstat.
**Source:** 08-04-SUMMARY.md

---

### Same-Host Improvement Gate
Use a script that parses raw benchmark rows, verifies host metadata identity, compares medians, and prints explicit PASS/FAIL lines for every required row.

**When to use:** Use when benchmark evidence must be committed as machine-checkable proof rather than manually interpreted benchstat output.
**Source:** 08-05-SUMMARY.md

---

## Surprises

### Larger BIGINT-Style Fixture Did Not Hit the Expected Error
The initial oversized integer test fixture returned `ErrPrecisionLoss` instead of the planned parse-time `ErrInvalidJSON`.

**Impact:** The test fixture had to be corrected before Wave 0 could accurately pin current public parser behavior.
**Source:** 08-01-SUMMARY.md

---

### Cargo Test Commands Contended on the Build Directory
Parallel targeted `cargo test` runs briefly contended on Cargo's build-directory lock.

**Impact:** The checks had to be rerun serially; no source change was needed.
**Source:** 08-02-SUMMARY.md

---

### Busy-Guard Testing Exposed a Preexisting Blocking Lock Path
The first all-tests fast-materializer run appeared hung because the busy-guard path deadlocked in `Doc.isClosed()`.

**Impact:** `Doc.isClosed()` needed a non-blocking mutex check so materializer contention could surface `ErrParserBusy` deterministically.
**Source:** 08-03-SUMMARY.md

---

### Canada Regressed by Orders of Magnitude on the First Evidence Run
The first Phase 8 evidence attempt regressed Canada rows to roughly `4.2 s/op`, with `37,569` native allocations and more than `232 GB` cumulative native allocation traffic.

**Impact:** The gate correctly blocked closeout until native scratch growth was fixed; final Canada diagnostics dropped to `6.14 ms` full and `4.43 ms` materialize-only.
**Source:** 08-05-SUMMARY.md

---

### Per-Container Reserve Churn Was the Real Large-Fixture Bottleneck
The Canada regression came from per-container `reserve(size + 1 + child_hint)` churn in the native frame vector, not from the fast-materializer API shape.

**Impact:** Replacing per-container reserve churn with geometric frame scratch growth fixed the large-tree case without changing the public or internal ABI.
**Source:** 08-05-SUMMARY.md
