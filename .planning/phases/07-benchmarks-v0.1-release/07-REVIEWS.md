---
phase: 7
reviewers: [gemini, claude]
reviewed_at: 2026-04-22T17:03:32Z
plans_reviewed:
  - 07-01-PLAN.md
  - 07-02-PLAN.md
  - 07-03-PLAN.md
  - 07-04-PLAN.md
  - 07-05-PLAN.md
---

# Cross-AI Plan Review - Phase 7

## Gemini Review

### 1. Summary

The phase plans provide a well-structured approach to fulfilling the v0.1 benchmark, correctness, and release requirements. The division of benchmark workloads into Tier 1 (full materialization), Tier 2 (typed extraction), and a clearly marked Tier 3 (placeholder for On-Demand) correctly aligns with the project's DOM-based capabilities and ensures fair comparisons. The plans also correctly respect the immutability of the existing `v0.1.0` tag by explicitly driving a `v0.1.1` patch release through the established CI sequence. However, there are significant execution risks regarding how the autonomous agent will source the external benchmark corpora and how it will technically fulfill the native C++ allocation tracking requirements in the Rust shim.

### 2. Strengths

- **Fairness and Parity:** Tier 1 mandates strict full-materialization for `pure-simdjson`, eliminating the common pitfall of comparing a lazy DOM parse to standard library struct unmarshaling.
- **Release Safety:** Plan 07-05 explicitly prevents rewriting the existing `v0.1.0` tag and relies strictly on the existing `check_readiness.sh` and CI-driven `release.yml` workflows.
- **Deterministic Oracle:** Plan 07-01 isolates the correctness oracle from upstream mutability by vendoring an exact `expectations.tsv` manifest and the corresponding test cases.
- **Comparator Discipline:** Plan 07-02 mandates structural omission of unsupported comparators per target rather than emitting misleading `N/A` rows.

### 3. Concerns

- **HIGH: Ambiguous External Data Sourcing (07-01):** The plan instructs the agent to "Vendor canada.json, mesh.json, and numbers.json... from a single pinned upstream snapshot" and to "Create testdata/jsontestsuite/cases/...". Autonomous agents cannot reliably intuit the correct URLs, submodule paths, or raw GitHub endpoints for these files without explicit instructions. This is highly likely to result in a hallucinated or failed execution.
- **HIGH: Technical Feasibility of Native Allocator Hooks (07-03):** The plan requires tracking native allocations to capture "simdjson's tape allocs", instructing the agent to "Implement the counters in Rust". `simdjson` allocates its tape and structural memory using C++ allocators (`malloc`/`aligned_alloc`). A Rust global allocator will not intercept these C++ allocations unless `malloc` is overridden across the C ABI or `simdjson` is patched to use an injected Rust allocator. The agent will likely fail or only count the Rust-side padded input arena.
- **MEDIUM: Comparator Build Constraints (07-02):** The `minio/simdjson-go` comparator does not compile on `arm64`. If the agent implements the availability gates using only runtime checks (`runtime.GOARCH`), the benchmark suite will fail to compile on ARM targets.
- **LOW: Insufficient Benchstat Sampling (07-02):** The validation commands and standard `go test` runs are configured with `-count=1`. `benchstat` requires statistically significant sample sizes (typically `-count=5` or `-count=10`) to compute the Mann-Whitney U test and render variance margins.

### 4. Suggestions

- **Provide Explicit Fetch Instructions (07-01):** Update the plan to provide exact `curl` commands pointing to the `simdjson/simdjson` `v4.6.1` raw GitHub URLs for the missing JSON files and the JSONTestSuite directory, ensuring the agent has deterministic access to the required data.
- **Scope or Redesign BENCH-05 (07-03):** Either explicitly scope the native allocator tracking to the Rust-owned padded input arena (and document the limitation regarding C++ tape allocations), or provide the agent with a specific architectural pattern for intercepting C++ `malloc` within the `cdylib` without destabilizing the Go consumer process.
- **Mandate Go Build Tags (07-02):** Explicitly instruct the use of Go build tags (e.g. `//go:build amd64`) for the `minio-simdjson-go` comparator wrapper file to ensure the test suite cleanly compiles across all targets in the Phase 6 CI matrix.
- **Update Benchstat Execution (07-02):** Modify `scripts/bench/run_benchstat.sh` and the corresponding `Makefile` targets to execute benchmark suites with `-count=5` or `-count=10` to ensure actionable `benchstat` output.

### 5. Risk Assessment

**HIGH**

**Justification:** While the architectural intent and CI sequencing are sound, the combination of ambiguous data-sourcing instructions in 07-01 and the deep technical complexity of intercepting C++ allocations from a Rust shim in 07-03 poses a severe risk of autonomous execution failure. If the agent cannot locate the benchmark files or fails to build the allocator intercept, the core deliverables for Phase 7 (BENCH-02 and BENCH-05) will be blocked.

---

## Claude Review

`★ Insight ─────────────────────────────────────`
- Phase 7's plans follow a clean "vendor → measure → document → release" progression that mirrors how credible benchmark stories are built: inputs frozen first, claims last. That ordering is the right shape for a release-close phase.
- The biggest structural risk sits where the plans hand off between measurement (02/03) and public claim (04): there's no explicit step that captures/commits *the actual numbers* or gates README language on whether the ≥3x claim survived measurement.
`─────────────────────────────────────────────────`

# Phase 7 Plan Review

## 1. Summary

The plan set is well-structured and well-sequenced: 07-01 freezes the benchmark and oracle inputs, 07-02 builds the Tier 1 harness and comparator surface, 07-03 layers Tier 2/3 and native allocator telemetry, 07-04 turns results into public docs and legal artifacts, and 07-05 closes the release through the existing CI-only path using `v0.1.1` so `v0.1.0` stays immutable. The plans honor the locked CONTEXT decisions (D-01..D-04), respect existing release/bootstrap anchors, and avoid smuggling v0.2 On-Demand scope into v0.1. The main weaknesses are around BENCH-05 (native allocator instrumentation depth), comparator availability on non-x86 targets, how measured numbers feed README claims, and a few acceptance-criteria regexes that are too permissive.

## 2. Strengths

- **Input-first sequencing.** 07-01 pins the benchmark corpus and oracle manifest before any measurement exists, which is exactly the discipline BENCH-02/BENCH-06 demand.
- **Release-close respects immutability.** 07-05 explicitly routes the phase to `v0.1.1` rather than moving `v0.1.0`; it reuses `check_readiness.sh` and `public-bootstrap-validation.yml` rather than inventing a new path, and the SKILL constraints are honored.
- **Comparator honesty by design.** The canonical comparator keys and the "omit unsupported, don't fake N/A" rule are threaded through 07-02 and surface again in 07-04's methodology doc.
- **Tier 1 fairness baked into acceptance.** Plan 07-02 requires `pure-simdjson` to keep the timer running through full Go-value materialization, not just DOM parse - the single biggest fairness trap for this kind of library is pre-empted.
- **Cold-start / warm split is lexical, not stylistic.** Distinct `BenchmarkColdStart_` / `BenchmarkWarm_` family prefixes make benchstat comparisons safe against accidental cross-family diffs.
- **FFI surface growth is disciplined.** 07-03 updates `include/pure_simdjson.h`, Rust tests, and `tests/smoke/ffi_export_surface.c` in the same task - matching the Phase 1/2/4 precedent for ABI additions.
- **Autonomy boundary is correct.** Only 07-05 is marked `autonomous: false`, and the reasons (tag push, live workflow observation) match reality.

## 3. Concerns

### HIGH

- **BENCH-05 instrumentation depth is under-specified.** [07-03 Task 1] Most of `pure-simdjson`'s real allocation cost is inside C++ simdjson (tape, string buffers, implementation kernels), not Rust shim code. "Measure native allocations performed by the shim path" is ambiguous: a Rust `GlobalAlloc` wrapper will miss C++ `new`/`malloc`; instrumenting only Rust-side `Vec` allocations will systematically under-report the very thing the requirement exists to expose. Plan must specify whether C++ allocations are counted (requires overriding operator new/delete or hooking malloc/free at link time) and how the counters remain zero-overhead in non-benchmark callers.
- **Comparator availability on non-amd64 targets is handwaved.** [07-02 Task 1] `minio/simdjson-go` historically requires AVX2/CLMUL (linux/arm64, darwin/arm64 unsupported). `bytedance/sonic` has architecture and Go-version constraints. "Availability gates" as runtime skips inside a root-package `_test.go` won't help if the import itself fails to compile on arm64. The plan needs explicit build-tag strategy (e.g. per-comparator `_amd64.go` / `_noop.go` pairs) or the `go test ./...` step on linux/arm64 will fail outright, which blocks 07-02's own acceptance command.
- **No step captures and commits the measured numbers that back the README claim.** [07-04 Task 1] The README is instructed to state `>=3x vs encoding/json + any` and `within 2x of minio/simdjson-go` "only if measured evidence supports it" - but no plan actually captures a snapshot (e.g. `docs/benchmarks/results-0.1.1.md` or `testdata/benchmark-results/`) or gates README language programmatically. The reader of the final release cannot reproduce the claim from committed inputs; BENCH-07 is effectively verified by grep-for-substring, not by evidence-that-matches.

### MEDIUM

- **ABI contract doc and verify-contract workflow are not updated.** [07-03] Adding `pure_simdjson_native_alloc_stats_t` + two exports changes the committed C header; `Makefile` `verify-contract` already diffs the header byte-for-byte and `verify-docs` greps `docs/ffi-contract.md` for contract patterns. Plan 07-03 doesn't list `docs/ffi-contract.md`, `cbindgen.toml`, or `tests/abi/check_header.py` in `files_modified`, which means either the diagnostic surface isn't documented (contract drift), or those files get modified ad hoc without being in the plan envelope.
- **Oracle corpus size and upstream pin are not specified.** [07-01 Task 1] The plan requires a manifest and cases directory but sets no minimum case count and no upstream snapshot commit/hash for `JSONTestSuite`. BENCH-06 reads "parse every file in simdjson's `jsontestsuite`" - a 5-case oracle technically satisfies the plan's structural checks but not the requirement's intent. Recommend pinning the upstream snapshot ref in `testdata/jsontestsuite/README.md` and asserting a floor (e.g. ≥ 300 cases) in the test.
- **Cold-start vs bootstrap semantics are conflated.** [07-02 Task 2] BENCH-04 defines cold-start as "first `Parse` after `NewParser`". Process-level library bootstrap (download/dlopen) only happens once per process regardless of `b.N`, so a benchmark named `BenchmarkColdStart_` will measure per-iteration `NewParser()` + first parse, not the one-time bootstrap cost. That's a legitimate metric, but the plan's phrasing risks the README implying something stronger. README text and `docs/benchmarks.md` should explicitly state what "cold" means.
- **Tier 2 fairness across comparators is not constrained.** [07-03 Task 2] Exact workloads are specified for `pure-simdjson`, but "typed decoding in the baseline libraries" is not. Without committed comparator adapter schemas, `encoding/json` + struct, `sonic`, and `goccy/go-json` may decode more or fewer fields than the `pure-simdjson` path, which is the classic Tier 2 unfairness trap. Add: a single shared schema definition (already introduced in 07-02 `benchmark_schema_test.go`) is the *only* struct shape allowed for every Tier 2 comparator.
- **Acceptance regexes are too permissive.** [07-01 Task 1] `rg '^relative_path\\texpect\\tnote$|accept|reject'` passes as long as *any* of the three alternatives is found - a file containing only the word `accept` satisfies it. Several acceptance checks in 07-02 and 07-04 use similar OR-separated greps. Consider AND-style checks (separate `rg` invocations joined by `&&`) or stricter regexes.
- **README claim failure path is unhandled.** [07-04 / 07-05] If measurements don't hit ≥3x on three of five corpus files (success criterion 1) or don't match the `within 2x of minio/simdjson-go` line, BENCH-07 / DOC-01 are partially unmet. Plan 04 says "only if supported," but 07-05 proceeds to tag `v0.1.1` regardless. A short "if evidence doesn't support the headline, here's the fallback phrasing and whether we still ship the patch release" branch would make the phase robust.

### LOW

- **LICENSE copyright holder unspecified.** [07-04 Task 2] "Full MIT license text" - real MIT text needs `Copyright (c) <year> <holder>`. Worth nailing the authorship line.
- **NOTICE cites only Apache-2.0.** [07-04 Task 2] Upstream simdjson is dual-licensed (Apache-2.0 + MIT per `third_party/simdjson/LICENSE-MIT`). Citing both matches upstream intent.
- **README doesn't point at cosign verification.** [07-04 Task 1] `docs/releases.md` and `docs/bootstrap.md` both cover cosign; README should at minimum link to them from a "Verifying downloads" note.
- **`scripts/bench/run_benchstat.sh` doesn't check for `benchstat` availability.** [07-02 Task 2] First-time runners will get an opaque "command not found" instead of a pointer to `go install golang.org/x/perf/cmd/benchstat@latest`.
- **Docs/releases does not yet mention `v0.1.1` flow.** [07-05] The runbook is generic over `<version>`, which is fine, but the CHANGELOG and `docs/releases.md` don't cross-link to the Phase 7 patch release rationale anywhere. Low-impact but would help future archaeology.
- **No explicit note that development-time benchmarks use local `target/release/`, not the published artifact.** Research explicitly flagged this ("credible benchmarks need the final `.so` on the final channel"). Plans do not re-run benchmarks against the published `v0.1.1` artifact before claiming numbers.

## 4. Suggestions

- **Make BENCH-05 instrumentation concrete.** Specify the mechanism: either (a) override C++ `operator new`/`operator delete` in the shim with counting wrappers behind a compile-time feature flag, or (b) use a Rust-side `GlobalAlloc` wrapper plus an explicit caveat that simdjson's C++ allocations are not captured. Either choice is defensible - ambiguity is not.
- **Add a "capture + commit benchmark results" task to 07-04.** Vendor an actual benchstat-formatted snapshot under `docs/benchmarks-0.1.1.md` or `testdata/benchmark-results/0.1.1/`, and quote from it in the README. That's what makes BENCH-07 auditable instead of assertive.
- **Pin the JSONTestSuite snapshot.** Put the upstream commit hash and date in `testdata/jsontestsuite/README.md`, and add a `TestJSONTestSuiteOracleCorpusFloor` that asserts the case count is within a sane floor.
- **Introduce explicit comparator build tags.** Something like `benchmark_comparators_minio_amd64.go` / `benchmark_comparators_minio_stub.go` so unsupported-target builds compile cleanly and omission is a compile-time truth, not a runtime branch.
- **Tighten acceptance-criteria regexes.** Prefer multiple small `rg` calls joined with `&&` over one grep with an OR-alternative that is satisfied by matching any single alternative.
- **Update 07-03 `files_modified` to include the ABI contract doc and `cbindgen.toml`** (and any `tests/abi/*` pattern rules that need to allow the new exports). Even "no change needed" is valuable as a committed statement.
- **Add a README-claim fallback branch to 07-04 or 07-05.** Explicitly state: "if the ≥3x claim doesn't hold, README says X; patch release still ships because docs+benchmarks+legal are independently valuable."
- **Plan 07-05 could note a post-publish benchmark re-run** against the real `v0.1.1` `.so` pulled via `BootstrapSync`, to confirm CI-produced artifacts yield equivalent numbers to the pre-tag local build.

## 5. Risk Assessment

**Overall: MEDIUM**

The phase sequencing and release-close are low-risk - the plans correctly avoid mutating `v0.1.0` and route through the existing CI path. The medium-risk concentration is in three places: (1) BENCH-05's native allocator instrumentation is specified by shape but not by mechanism, and getting it wrong silently under-reports; (2) comparator portability on arm64 targets could break `go test ./...` during 07-02 before any benchmarks run; (3) the handoff from measured results to public README claims has no gate that prevents overstatement. None of these are structural blockers, but each materially affects whether Phase 7 actually ships a *credible* benchmark story vs. just a *complete* one. Addressing the three HIGH concerns would drop the phase to LOW risk.

---

## Consensus Summary

Both reviewers agreed that the overall phase shape is strong: freeze inputs first, then measure, then document, then ship through a patch release that keeps `v0.1.0` immutable. They also agreed that the current plans are not yet tight enough around measurement credibility and portability, which are the two areas most likely to undermine a "credible benchmark story" even if the phase executes end-to-end.

### Agreed Strengths

- The plan sequencing is disciplined and matches the phase goal: vendored inputs and oracle first, benchmark harness second, public docs only after measurement, and release-close last.
- The release strategy is correct: `v0.1.0` remains immutable, and any public Phase 7 ship path goes through `v0.1.1` plus the existing CI-only release and bootstrap-validation workflows.
- Tier 1 fairness and comparator omission are treated seriously. Both reviews called out the full-materialization requirement and the rule to omit unsupported comparators rather than publishing fake `N/A` rows.

### Agreed Concerns

- **Allocator telemetry needs a concrete mechanism.** Both reviewers flagged BENCH-05 as under-specified because most meaningful allocation cost sits in C++ simdjson, not the Rust shim. The plan should explicitly choose whether it captures true C++ allocations or only Rust-side allocations, and document the limitation if it chooses the narrower path.
- **Comparator portability must be compile-time safe.** Both reviewers flagged the risk that unsupported comparators, especially `minio/simdjson-go` on non-amd64 targets, will break compilation before any runtime omission logic can help. The plan should require build-tagged comparator wrappers or equivalent compile-time isolation.
- **README benchmark claims need committed evidence.** Both reviewers flagged the lack of an explicit "capture results and feed docs from them" step. A committed benchmark snapshot or results artifact should exist before `README.md` makes public performance claims.

### Divergent Views

- Gemini treated missing upstream fetch details for the corpus and JSONTestSuite as a high-risk execution blocker; Claude treated the same area as a pinning/completeness issue and focused more on corpus size floors and provenance.
- Gemini rated the overall phase risk **HIGH** because it expects autonomous execution to fail on data sourcing or allocator instrumentation; Claude rated it **MEDIUM** because it sees the weaknesses as fixable without restructuring the phase.
- Claude surfaced additional plan-hygiene issues not mentioned by Gemini: ABI contract drift risk from the new FFI exports, overly permissive acceptance regexes, cold-start wording ambiguity, and missing fallback wording if the headline benchmark claim does not hold.
