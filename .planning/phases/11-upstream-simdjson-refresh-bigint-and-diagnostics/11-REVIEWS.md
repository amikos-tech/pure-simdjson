---
phase: 11
reviewers: [claude, opencode]
reviewed_at: 2026-07-23T07:51:56Z
plans_reviewed:
  - 11-01-PLAN.md
  - 11-02-PLAN.md
  - 11-03-PLAN.md
  - 11-04-PLAN.md
  - 11-05-PLAN.md
  - 11-06-PLAN.md
  - 11-07-PLAN.md
  - 11-08-PLAN.md
  - 11-09-PLAN.md
  - 11-10-PLAN.md
  - 11-11-PLAN.md
  - 11-12-PLAN.md
  - 11-13-PLAN.md
  - 11-14-PLAN.md
---

# Cross-AI Plan Review — Phase 11

## Claude Review

Reviewed all 14 plans against the locked decisions (D-01–D-22), the three validated spikes, and the actual repository state (I verified the current bridge, registry, loader, bindings, error mapping, tests, ABI audits, and release workflows referenced by the plans).

### Summary

This is an unusually well-grounded plan set: every high-risk seam (error-location replay, staged ABI binding, pre-copy capacity gate) is backed by an executed spike, the ABI evolution is strictly additive with append-only numeric assertions, and publication is correctly fenced behind two human checkpoints (11-01 version decision, 11-14 operator-only release). The file-level claims match the codebase — the plans correctly identify the real defects they must fix (e.g., `hasOffset(offset) == offset != 0` losing known-zero in `errors.go:240-242`, capacity checked only after `Vec::resize` in `registry.rs:453-460`, bind-before-ABI-check ordering in `bindings.go`/`library_loading.go`). The main weakness is **cross-wave suite coherence**: enabling native BigInt in wave 1 breaks committed Go tests that are not updated until wave 8, and new Rust exports in waves 2–6 make `make verify-contract` red until the single header regeneration in wave 7 — both directly contradict the phase's own per-wave sampling contract in 11-VALIDATION.md, and no plan owns the reconciliation. There is also an unowned risk that the JSONTestSuite oracle pins the old oversized-integer rejection behavior.

### Strengths

- **Spike-to-plan traceability is exemplary.** D-19/D-20 (hybrid replay + pointer-range proof), D-21 (probe-then-bind), and D-22 (pre-copy gate with stale-diagnostic clearing) each map to a validated spike with machine-verified evidence, and plans 11-04/11-05/11-08 reproduce the exact validated ordering rather than re-deriving it.
- **The 11-01 version checkpoint is the right design.** It prevents every downstream plan from inventing a release version, records `publication_authorized: false`, and 11-07/11-13/11-14 all consume the recorded value with an explicit malformed/already-tagged abort path.
- **Additive-ABI discipline is enforced mechanically, not by convention** — kind 9 / status 9 / status 10 append-only, unchanged `psdj_internal_frame_t`/`ValueView` layouts, BigInt reusing the existing `string_ptr/string_len` and copied-bytes/`bytes_free` ownership path, and the audits (`check_header.py`, `handle_layout.c`, `types_test.go`) extended rather than replaced.
- **Correct dependency ordering for the ABI bump**: all five new native exports exist (waves 2–5) before the ABI constant moves to `0x00010002` (wave 6), before the loader requires them (wave 7). This avoids ever shipping a source state that claims 1.2 while missing symbols.
- **Characterize-then-assert for error offsets** (11-05 Binding Resolution) is the honest way to handle upstream-dependent behavior: the corpus observations are captured against pinned v4.6.4 before golden assertions are locked, and a contradiction with Spike 001 is a blocking investigation, not a silent patch.
- **The pre-copy capacity gate is placed at the only correct layer** (Rust registry, while `reusable_input` is still attached), with the white-box test proving no arena growth — the C++ limit alone would be too late, and the plans say so explicitly.
- **Threat models are concrete and testable** (unsupported kernel → illegal instructions, borrowed BigInt lifetime, fabricated offsets, heterogeneous pools), each with a named test obligation rather than aspirational language.
- **Release sequencing respects the repo's invariants**: squash to main → strict readiness → annotated tag → CI-only publish → Phase 06.1 public validation, with the "SOURCE READY — NOT RELEASED" state preventing local success being mistaken for shipped.

### Concerns

- **[HIGH] Cross-wave Go suite breakage is unowned (waves 1→8).** Plan 11-02 enables `number_as_string(true)` and removes the BIGINT→`ERR_INVALID_JSON` normalization in wave 1, and Rust tests are updated in the same plan — but the Go tests that pin the old behavior are not touched until Plan 11-10 (wave 8): `TestGetUint64/"oversized literal rejected at parse"` (`element_scalar_test.go:224-236`) and `TestFastMaterializerOversizedLiteralParseRejected` (`materializer_fastpath_test.go:108-123`) will fail as soon as the release library is rebuilt. 11-VALIDATION.md mandates `go test ./... -race` after every wave, so waves 1–7 are red by construction. An executor following the sampling contract will either stall or "fix" tests out-of-plan.
- **[MEDIUM] Interim `verify-contract` redness contradicts the sampling policy.** Plans 11-03 through 11-07 add Rust exports and bump the ABI constant but explicitly defer header regeneration to 11-09 ("do not hand-edit the generated header"). Until wave 7, `make verify-contract` fails on header diff, `check_header.py`'s hardcoded `0x00010001` regex, and the unexpected-symbol rule — yet the validation doc says to run `make verify-contract` "whenever ABI/header code changes." The single-sync approach is defensible, but no plan acknowledges or exempts the interim failures.
- **[MEDIUM] JSONTestSuite oracle expectations may pin the superseded rejection behavior.** The suite contains oversized-integer fixtures (e.g., `i_number_too_big_pos_int`-class inputs) that the current parser rejects and the Phase 11 parser will accept as BigInt. If the committed oracle table recorded those as rejections, `TestJSONTestSuiteOracle` goes red and **no plan owns the expectation update** — 11-13 only runs it as a gate. UP-01's "correctness oracle still agrees" success criterion may be unsatisfiable as written.
- **[MEDIUM] `NewParserPool(opts...)` becomes an eager native-load (potentially a network download).** Plan 11-12 has the pool constructor call native `LockImplementationSelection`, which requires `activeLibrary()` — so constructing an empty pool can now trigger the 5-minute bootstrap path. Today `NewParserPool()` is pure Go. D-12 requires the *lock*, but locking Go-side state at construction and deferring the native lock to first native touch would preserve laziness; if eager load is intended, it needs prominent documentation as part of the D-11 source break.
- **[LOW-MEDIUM] Accessor-fallback materializer BigInt branch is not specified.** Plan 11-10 adds kind 9 to `buildAnyFromFrame`, but `materializeElementViaAccessors` (`materializer_fastpath.go:51-120`) — the fallback when `psdj_internal_materialize_build` is absent, which remains a legitimately optional symbol — will hit `default: ErrInternal` on BigInt. The test helper `materializeViaAccessorsForTest` has the same gap.
- **[LOW] BigInt in the materialized `any` tree is indistinguishable from a JSON string.** Emitting plain `string` for kind 9 means `{"n": 123456789012345678901}` and `{"n": "123456789012345678901"}` materialize identically. The surface is unexported (Tier 1 internal), so this is tolerable, but it's an undocumented semantic choice.
- **[LOW] The `abiVersionOverride` test seam in `parser.go` is unaddressed.** Plan 11-08 removes NewParser's exact ABI check and moves compatibility into the loader, which orphans `setExpectedABIVersionForTest`/`TestABIMismatchAtNewParser`; the plan lists `parser.go` but not this seam's migration.
- **[LOW] Failure-path cost amplification is unmeasured.** The D-19 hybrid runs up to two additional On-Demand parses on every DOM parse failure (~3× work for adversarial malformed input). Bounded by input size and only on failures, but no benchmark or note covers it.
- **[LOW] Plan 11-02 doesn't explicitly name every BIGINT special case to remove**: the `type == BIGINT → PRECISION_LOSS` branches in `psimdjson_element_type`/`psimdjson_element_type_at` (`simdjson_bridge.cpp:712-714, 746-748`) and the now-dead `KIND_HINT_INVALID` precision-loss handling in `doc_root`/`encode_descendant_view_locked` (`registry.rs:650-654, 728-732`). The behavior tests would catch omissions, but naming them removes guesswork.
- **[LOW] Fuzz corpus.** Plan 11-10 updates the fuzz walker but doesn't add an oversized-integer seed (`f.Add([]byte("18446744073709551616"))`), and any cached fuzz corpus entries under `testdata/fuzz` that previously exercised the rejection path may need refresh.

### Suggestions

1. **Add an explicit "transitional expectations" step to Plan 11-02 Task 2**: update `element_scalar_test.go` and `materializer_fastpath_test.go`'s two oversized-literal tests in the same commit that changes native behavior (they can assert the new parse-succeeds behavior at the FFI level even before `TypeBigInt` exists, e.g., parse succeeds and `TypeErr()` returns no ErrInvalidJSON). Alternatively, amend 11-VALIDATION.md's per-wave gate to enumerate the tests expected red per wave — but fixing at the introducing commit is cleaner and matches the repo's own pre-commit philosophy.
2. **State the interim verify-contract policy once**: add a line to 11-VALIDATION.md (or each of 11-03..11-07) that `make verify-contract` is expected to fail between the first new export and Plan 11-09, and per-task gates for those plans substitute `cargo test`/`go test ./internal/ffi`.
3. **Characterize the oracle before wave 1 locks assertions**: add a check to Plan 11-02 (or 11-13's read_first) that greps the committed jsontestsuite expectations for oversized-integer fixtures and assigns the expectation update to a named plan if any flip from reject to accept.
4. **Split the kernel lock in Plan 11-12**: lock the Go lifecycle mutex at `NewParserPool` construction (sufficient to make `SetKernel` fail deterministically, since all supported kernel mutation flows through Go), and take the native lock at first native parser construction. If you keep the eager native lock, document "pool construction may trigger bootstrap download" in `docs/concurrency.md` and the changelog entry.
5. **Extend Plan 11-10 Task 2** to add the `TypeBigInt` case to `materializeElementViaAccessors` and `materializeViaAccessorsForTest`, and one parity test with the internal materializer forced unavailable.
6. **Name the seam migration in Plan 11-08 Task 2**: retire or relocate `abiVersionOverride`/`TestABIMismatchAtNewParser` to loader-level fixtures so the D-15 mismatch path keeps a first-class test.
7. Minor: add a BigInt fuzz seed in 11-10; consider a named type (`type BigInt string`) for the materialized tree if downstream consumers ever need to distinguish it — cheap now, a break later.

### Risk Assessment

**Overall: MEDIUM.**

The architectural risk is low — decisions are locked, spike-validated, additive, and the plans' file-level claims check out against the real codebase; the dangerous seams (memory ownership, ABI classification, capacity ordering, process-global kernel state) all have concrete, correctly-placed controls and executable proofs. The residual risk is *process* risk concentrated in sequencing: a 7-wave window of deliberately-red Go and contract gates that the phase's own validation contract forbids, plus one potentially unsatisfiable success criterion (oracle agreement) with no owning plan. These won't ship a wrong artifact — the 11-13/11-14 gates would catch everything — but they can stall autonomous execution, invite out-of-plan "fixes," and burn the operator checkpoint on avoidable churn. Addressing suggestions 1–3 before execution starts would drop this to LOW.

---

## OpenCode Review

### Summary

The plan set is unusually thorough and mostly achieves Phase 11’s goals: upstream pinning, BigInt propagation, diagnostics, limits, ABI gating, bootstrap readiness, and release proof are all covered with good source-to-test traceability. The biggest risks are not missing requirements, but sequencing/detail hazards in a few plans: public option API shape, kernel lifecycle ordering in the C smoke, error-location replay complexity, and the artifact/bootstrap gate potentially coupling source readiness too tightly to an unpublished artifact.

### Strengths

- Strong requirement coverage: UP-01, NUM-01/02, DIAG-01/02, LIMIT-01 are all mapped to concrete plans and tests.
- Good ABI discipline: append-only kind/status additions, ABI-first binding, mandatory 1.2 symbol checks, header audits, C smoke, and bootstrap canary all reinforce each other.
- Correct ownership model for BigInt: reuses existing copied-byte/free path instead of exposing simdjson-owned `string_view`.
- Excellent capacity-limit placement: explicitly gates before `mem::take`, padding, resize, or copy, which is the key DoS mitigation.
- Good release realism: Plan 14 correctly separates source readiness from CI-only artifact publication and public bootstrap validation.
- Good diagnostic truthfulness: known/unknown offset state is represented explicitly and avoids fabricated scanner offsets.
- Good kernel lifecycle model: process-global, exact-name, support-checked, locked on first parser/pool creation, with subprocess tests.

### Concerns

- HIGH: `11-09` C smoke has an ordering contradiction. It says to parse with a configured parser, then call `pure_simdjson_set_implementation` “before parser construction” and verify later lock behavior. Once the configured parser is created, kernel selection must already be locked, so the smoke cannot both parse first and set-before-construction in the same process.
- HIGH: `11-11` proposes `type ParserOption func(parserConfig) (parserConfig, error)` as a public exported type whose signature references an unexported type. This compiles, but makes custom external options impossible/awkward and leaks an internal type shape into docs. If custom options are intentionally disallowed, a sealed interface or unexported concrete option type is cleaner.
- MEDIUM: Plan 13 pins `internal/bootstrap/version.go` to an artifact version before that artifact exists, then runs source gates. This is intentional, but it means normal default-bootstrap behavior may fail until Plan 14. The plans mostly account for this, but every local smoke must consistently use `PURE_SIMDJSON_LIB_PATH` or a staged artifact before publication.
- MEDIUM: The On-Demand replay in `11-05` is complex and easy to get subtly wrong. The plan is careful, but the implementation should be treated as high-risk C++ logic despite being failure-only.
- MEDIUM: Status codes `9` and `10` are introduced in native/Rust before the public ABI synchronization. That is probably fine inside the staged plans, but intermediate states may break tests if any Go code sees new statuses before `errors.go` maps them.
- MEDIUM: Kernel state tests and Go package-level globals are inherently order-sensitive. The plan uses subprocesses, which is good, but any non-subprocess test that constructs a parser before kernel tests can make failures confusing.
- LOW: Plan 01 says “No tracked file changes” but also creates `11-01-SUMMARY.md`. The plan later clarifies the summary is allowed, but the acceptance wording is slightly inconsistent.
- LOW: `11-13` modifies `.planning/.../11-VALIDATION.md` and readiness artifacts as part of implementation. That is useful for GSD tracking, but it increases non-source churn and should not be mixed into public release commits if the project wants clean library history.
- LOW: BigInt materialization to Go `any` as string should be clearly documented. Returning `string` is probably right, but downstream users may assume all JSON numbers stay numeric in full materialization.
- LOW: The plan says no new dependencies, but `Plan 01` shell verification uses `grep`. That is fine operationally, just not relevant to source constraints.

### Suggestions

- Fix `11-09` C smoke ordering: call `pure_simdjson_set_implementation("", 0)` first, then construct the parser, parse BigInt, then assert a later setter returns kernel-locked.
- Rework the `ParserOption` API to avoid exposing unexported parameter types. For example: `type ParserOption interface{ applyParserOption(*parserConfig) error }` with unexported option structs, or `type ParserOption func(*parserConfig) error` if custom options are acceptable.
- Add an explicit note in `11-13` that any local Go smoke before publication must use the freshly built local library via `PURE_SIMDJSON_LIB_PATH`; default bootstrap is only proven in Plan 14.
- Add a targeted “no default bootstrap before publication” test or checklist item so source readiness cannot accidentally rely on a not-yet-published approved version.
- In `11-05`, require sanitizers or at least ASan/UBSan-capable CI/local run for the pointer-range replay code if available. The pointer arithmetic is safety-critical.
- In `11-06`/`11-12`, keep every kernel lifecycle test subprocess-isolated unless the test only calls `Kernel()` before loading. Avoid shared-process setter/parser tests.
- In `11-10`/docs, state explicitly that BigInt values materialized to `any` are represented as exact decimal `string`, not `int64`, `uint64`, `float64`, or `*big.Int`.
- Add a final grep/check in Plan 13 for stale `NewParserPool()` call sites without error handling.
- Add a final grep/check for old comments or tests expecting `ErrPrecisionLoss` or oversized integer `ErrInvalidJSON`.
- Ensure readiness/summary files are either excluded from public-facing release commits or intentionally included with a clean commit message, depending on repo practice.

### Risk Assessment

Overall risk: MEDIUM.

The plans are complete and well-validated, but Phase 11 touches almost every risky boundary in the project: vendored C++, Rust ownership, purego binding, public Go API, ABI versioning, loader behavior, release artifacts, and process-global SIMD state. The largest residual risks are implementation complexity in native diagnostics, global kernel lifecycle tests, and bootstrap/artifact sequencing. The plan quality is high enough to manage those risks if the ordering issues above are corrected before execution.

---

## Consensus Summary

Both reviewers rate the plan set **MEDIUM risk**. They agree that the architecture is well researched and the requirements are covered, while the remaining risk is concentrated in intermediate-wave coherence and a few lifecycle details rather than missing phase scope.

### Agreed Strengths

- The six Phase 11 requirements are mapped to concrete implementation tasks and tests with unusually strong plan-to-code traceability.
- ABI evolution is additive and mechanically guarded through staged binding, append-only values, header/layout audits, and release canaries.
- BigInt ownership is safely designed around exact copied bytes and the existing explicit free path.
- Capacity limits and diagnostic offsets are placed at the right layers and avoid both unnecessary allocation and fabricated precision.
- Source readiness, artifact publication, and public bootstrap proof are separated by explicit gates.

### Agreed Concerns

1. **[HIGH] Intermediate-wave coherence needs an explicit contract.** Native statuses, BigInt behavior, generated headers, Go mappings, and the unpublished bootstrap artifact become synchronized across different waves. Each wave needs named expected-green gates and any deliberate temporary failures must be documented; local validation before publication must consistently use the staged library.
2. **[MEDIUM] Kernel lifecycle ordering needs tighter specification.** Pool construction, native loading, setter locking, C smoke ordering, and test isolation interact through process-global state. The plans should choose whether pool creation eagerly loads native code, order the C smoke accordingly, and keep mutation tests subprocess-isolated.
3. **[MEDIUM] Error-location replay deserves extra safety and cost checks.** Both reviewers identify the failure-only On-Demand replay as complex. Add sanitizer coverage where available and document or measure the extra failure-path work.
4. **[LOW] BigInt materialization semantics need documentation.** Representing BigInt as a plain Go string preserves digits but makes it indistinguishable from a JSON string in a materialized `any` tree.

### Divergent Views

- Claude’s highest-priority findings are unowned red Go/contract gates, possible JSONTestSuite oracle drift, and missing fallback-materializer and ABI-test-seam work.
- OpenCode’s highest-priority findings are a contradictory C smoke order and the exported `ParserOption` signature exposing an unexported configuration type.
- The reviewers do not directly disagree; they identify different concrete manifestations of the same sequencing and API-detail risk.
