---
phase: 3
reviewers: [gemini, claude]
reviewed_at: 2026-04-16T06:10:42Z
plans_reviewed:
  - 03-01-PLAN.md
  - 03-02-PLAN.md
  - 03-03-PLAN.md
---

# Cross-AI Plan Review — Phase 3

## Gemini Review

### Summary

The Phase 3 plans are well-structured and aligned with the project's architectural constraints. The split between FFI/loader foundation, public API/lifecycle, and docs/verification keeps the phase disciplined and focused on the happy path without dragging in Phase 4 or Phase 5 work.

### Strengths

- Deterministic loading strategy with a strict env-var and repo-local search order reduces wrong-library and DLL-hijack risk.
- ABI validation at `NewParser()` is an early, high-value failure point.
- Lifecycle integrity is preserved by keeping the native one-parser/one-doc invariant visible in the Go API and `ParserPool`.
- Build-tagged finalizers provide diagnostics without polluting the default production path.
- The FFI surface is intentionally narrow and avoids pulling Phase 4 symbols forward.

### Concerns

- `MEDIUM`: Plan `03-03` depends on GitHub workflow dispatch and remote permissions; missing auth or push access could block the final proof step.
- `LOW`: The module path/package-name split (`pure-simdjson` module, `purejson` package) needs clear docs to avoid consumer confusion.
- `LOW`: Native diagnostic handling should avoid stale parser error text bleeding into unrelated later failures.

### Suggestions

- Cache the implementation/kernel name for easier troubleshooting.
- Use atomic close-state handling to avoid racey double-close behavior.
- Add a Godoc or example file that demonstrates the `ParserPool` pattern directly.

### Risk Assessment

`LOW` — The decomposition is solid, the native ABI is already proven, and the plans directly address the main loader and lifecycle hazards for this phase.

## Claude Review

### Summary

The three plans form a coherent Phase 3 deliverable and follow the research and context documents closely. The decomposition is sound, but there are several important execution and correctness gaps that should be tightened before implementation, especially around purego/GC interaction, finalizer strategy, and the remote workflow verification mechanics in Plan `03-03`.

### Strengths

- Strong phase decomposition: foundation, lifecycle API, then docs and wrapper proof.
- Good scope discipline: no premature Phase 4 accessors or Phase 5 bootstrap work.
- Deterministic loader order matches the locked context decisions.
- The FFI allowlist and negative checks in `03-01` are precise and prevent accidental scope expansion.
- `ParserPool` misuse rejection and build-tagged finalizer split are directionally correct.
- `03-03` requires an observed green workflow result rather than treating YAML as proof.

### Concerns

- `HIGH`: The plans never mention `runtime.KeepAlive`, which is a real correctness risk when purego calls pass handle values into native code.
- `HIGH`: Plan `03-03` is marked autonomous even though it pushes to remote and dispatches GitHub Actions; this should be split or checkpointed rather than granting unattended push authority.
- `HIGH`: The `sleep 10` plus `gh run list --limit 1` workflow observation pattern is racy and can fail for timing reasons.
- `MEDIUM`: Production builds appear to attach no-op finalizers rather than omitting finalizers entirely, which still carries GC/finalizer overhead.
- `MEDIUM`: `sync.Pool` eviction of parsers is not addressed and can leak native resources.
- `MEDIUM`: Windows job setup does not mention MSVC environment preparation.
- `MEDIUM`: `macos-latest` is not pinned to guarantee the intended ARM64 proof target.
- `MEDIUM`: Some acceptance criteria are overly syntax-strict and assert parameter names instead of semantics.
- `LOW`: The docs example is checked textually but not for actual compilability.

### Suggestions

- Add an explicit `runtime.KeepAlive` requirement and acceptance check anywhere Go objects hand native handles to purego-bound functions.
- Make finalizer attachment itself build-tag-dependent rather than using a production no-op finalizer body.
- Clarify or redesign the `sync.Pool` eviction story for parser resources.
- Split remote workflow proof from local code changes, or gate push/dispatch behind a human checkpoint.
- Replace fixed `sleep 10` logic with bounded polling for the actual run ID.
- Add MSVC setup to the Windows workflow and pin the macOS runner version.
- Use an `example_test.go` for the concurrency example and loosen token-level signature greps into semantic checks.

### Risk Assessment

`MEDIUM` — The phase shape is right, but the current plans should address the high-severity purego/finalizer and remote-verification issues before execution.

## Consensus Summary

### Agreed Strengths

- The phase decomposition is strong: `03-01` establishes the module/loader/FFI foundation, `03-02` implements the public lifecycle API, and `03-03` handles docs and proof.
- The plans show good scope discipline by keeping Phase 3 on the happy path and explicitly avoiding broader Phase 4 accessor work and Phase 5 bootstrap/distribution work.
- Deterministic local loading and early ABI validation are both strong design choices.
- Preserving the native lifecycle invariant and making `ParserPool` reject misuse is the right direction for the public API.

### Agreed Concerns

- The remote GitHub workflow proof in `03-03` is execution-sensitive. Both reviewers called out environment and verification fragility there, whether from auth/push permissions or from the workflow observation mechanics themselves.

### Divergent Views

- Gemini sees the plans as broadly ready for execution with low overall risk; Claude rates the phase medium risk until several execution hazards are fixed first.
- Claude identifies `runtime.KeepAlive`, production finalizer attachment, and `sync.Pool` eviction behavior as meaningful correctness concerns. Gemini does not raise those issues.
- Gemini raises lower-severity usability concerns around package naming and stale native diagnostic text that Claude does not treat as central.

### High-Signal Follow-Ups

- Investigate Claude's `runtime.KeepAlive` concern before execution even though it was raised by only one reviewer; it is the highest-severity technical issue in the review set.
- Rework Plan `03-03` so the remote workflow proof is robust and does not depend on brittle `sleep` timing or unattended push authority.
- Tighten finalizer and resource-lifecycle assumptions in `03-02` before implementation starts.
