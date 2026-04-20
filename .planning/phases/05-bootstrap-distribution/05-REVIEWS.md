---
phase: 5
reviewers: [gemini, codex]
reviewed_at: 2026-04-20T10:41:03Z
plans_reviewed:
  - 05-01-PLAN.md
  - 05-02-PLAN.md
  - 05-03-PLAN.md
  - 05-04-PLAN.md
  - 05-05-PLAN.md
  - 05-06-PLAN.md
---

# Cross-AI Plan Review — Phase 5 (Bootstrap + Distribution)

## Gemini Review

This review evaluates the implementation plans for **Phase 5: Bootstrap + Distribution** of the `pure-simdjson` project.

### 1. Summary
The implementation plans for Phase 5 are of exceptional quality, demonstrating a deep understanding of the cross-platform FFI distribution pattern established in the `pure-*` family. The strategy effectively balances the "no-cgo" requirement with robust security (SHA-256 verification before `dlopen`), concurrency (flock-based process synchronization), and network resilience (full-jitter exponential backoff). By lifting and adapting proven patterns from `pure-onnx` while stripping unnecessary archive-extraction complexity, the plans provide a surgical and highly reliable path to achieving the DIST-01..10 requirements.

### 2. Strengths
- **Security-First Architecture:** The mandate for SHA-256 verification *before* any `dlopen` call (Pitfall #17) is consistently enforced across the download and loading tiers.
- **Comprehensive Fault Injection:** The inclusion of an explicit 11-scenario Fault Injection Matrix (Plan 06/Validation) ensures that edge cases like HTTPS-to-HTTP redirects, checksum mismatches, and lock contention are not just handled but verified.
- **Platform Integrity:** The plans correctly address Windows DLL hijacking (Pitfall #29) by enforcing absolute path resolution in the loader and cache path construction.
- **Dependency Hygiene:** Adherence to the `net/http` stdlib-first convention for the library while justifying `spf13/cobra` for the CLI demonstrates pragmatic architectural discipline.
- **Resilient Networking:** The transition to Full-Jitter exponential backoff with context-aware sleep is a significant upgrade over the linear sleep patterns in previous `pure-*` reference implementations.

### 3. Concerns
- **Windows `os.Rename` Atomicity (LOW):** While the plans correctly identify that `os.Rename` is generally atomic on the same volume in Go 1.22+, Windows environments can occasionally have unconventional volume mount points for `%LocalAppData%`.
    - *Mitigation:* The planned use of `flock` (Plan 02) effectively mitigates the risk of torn files during concurrent writes, even if the rename were non-atomic.
- **Empty `checksums.go` (LOW):** Plan 01 notes that `checksums.go` will be empty during initial development.
    - *Mitigation:* The inclusion of `ErrNoChecksum` provides a graceful failure mode for developers before the CI-05 release-time population occurs.
- **Circular Error Dependency (LOW):** Plan 01 revised the error sentinel approach to avoid circular imports between root `purejson` and `internal/bootstrap`.
    - *Mitigation:* Defining sentinels in `internal/bootstrap` and aliasing/wrapping them in the root package is the correct idiomatic solution for Go modules.

### 4. Suggestions
- **Cache Directory Override:** While the plans focus on `os.UserCacheDir()`, explicitly implementing the `PURE_SIMDJSON_CACHE_DIR` environment variable (mentioned as agent's discretion) is highly recommended for CI/CD environments where home directories are ephemeral or read-only.
- **User-Agent String:** In `download.go`, consider including the library version in the `User-Agent` (e.g., `pure-simdjson-go/v0.1.0`) to assist in R2/GitHub server-side telemetry and troubleshooting.
- **Progress Feedback:** For the CLI `fetch` verb, ensure that when `--all-platforms` is used, the progress output clearly distinguishes which platform is currently being processed to avoid "silent hang" perceptions during slow network conditions.

### 5. Risk Assessment: LOW
The risk is **LOW**. The plans are grounded in empirical research and proven patterns. The 4-stage loader chain is a robust evolution of the existing Phase 3 logic. The decoupling of the distribution machinery into `internal/bootstrap` protects the core library surface while providing the necessary hooks for the bootstrap CLI and auto-loader.

### Conclusion
The plans are **APPROVED** for execution. The wave-based approach (Scaffold → Core → Integration → Verification) is sound and maximizes parallelization potential between the library work and the CLI development.

---

## Codex Review

### Summary

The plan set is strong on intent and coverage: it has a clear phase boundary, concrete security rules, explicit success criteria, and good reuse of prior `pure-*` patterns. The biggest problems are not architectural, but contract-level inconsistencies that will cause churn if execution starts as written: GitHub fallback artifact naming is unresolved, bootstrap error ownership is contradictory across plans, the test/package/export story does not line up, and the loader rewrite preserves a mutex scope that the context explicitly says should not cover downloads. Fix those first and the rest of the phase looks tractable.

### Strengths

- The phase boundary is crisp. It stays focused on bootstrap/distribution and avoids dragging in unrelated API work.
- Security-critical requirements are correctly prioritized: SHA-256 before `dlopen`, absolute paths on Windows, and non-world-writable cache directories are all treated as load-bearing rules.
- The plans do a good job of turning research into implementation constraints instead of vague guidance.
- Reusing `pure-onnx` for lock/download/cache structure is the right call; it reduces invention risk.
- Requirement traceability is unusually good. The roadmap, context, research, and per-plan acceptance criteria line up well.
- The fault-injection mindset is strong. The plan explicitly targets checksum corruption, retry behavior, cancellation, and fallback behavior rather than only happy-path tests.
- The loader precedence rules are simple and defensible: env override, cache, bootstrap, fail.

### Concerns

- **[HIGH] `05-05`'s GitHub fallback URL scheme looks wrong for multi-platform assets.** GitHub release assets are flat, but the plan uses the same filename for multiple targets (`libpure_simdjson.so` for both Linux targets, `libpure_simdjson.dylib` for both macOS targets). As written, `githubArtifactURL(version, goos, goarch)` ignores `goarch` in the final asset name, so DIST-02 is not actually specified in an implementable way.
- **[HIGH] `05-01` and `05-04` are contradictory about bootstrap error ownership.** The plan cycles through "put sentinels in root," "put them in internal/bootstrap," "define both," and "alias from root." That is a real execution blocker, not just wording noise.
- **[HIGH] `05-03` is marked parallel with `05-02`, but its tests depend on APIs that `05-02` introduces.** As written, Wave 2 parallelism is overstated. The tests mention `BootstrapSync`, `resolveConfig`, `withHTTPClient`, and `withGitHubBaseURL` before those contracts are actually stabilized.
- **[MEDIUM] `05-04` says to keep `activeLibrary()` verbatim, but the context says downloads should happen before `libraryMu` is acquired.** Today `activeLibrary()` takes the mutex before `resolveLibraryPath()`. If preserved, first-run bootstrap will hold the process-wide loader mutex across network I/O, directly contradicting the design note.
- **[MEDIUM] The plans do not fully satisfy the roadmap's real-world success criteria yet.** `05-01` intentionally starts with an empty `Checksums` map, and no plan in this phase actually stages real published artifacts/checksums. That means success criterion 1 ("fresh machine with internet works") is only simulated unless another phase or manual release step exists.
- **[MEDIUM] The test/package/export story is inconsistent.** Several plans want `package bootstrap_test` while also calling unexported helpers like `resolveConfig`, `withHTTPClient`, `withGitHubBaseURL`, `defaultCacheDir`, and URL helpers. That will not compile as written.
- **[MEDIUM] Auto-bootstrap latency on `NewParser()` can be very painful on blocked networks.** With no caller context and sequential retries/fallbacks, a bad network path can stall parser creation for minutes, and the plans do not mention failure memoization to avoid repeating that delay.
- **[MEDIUM] The CLI `verify` scope is too narrow for the offline story.** In `05-05`, it only checks current platform in the default cache. That does not match the `--dest` / `--all-platforms` offline bundle flow particularly well.
- **[LOW] `05-02`'s fallback to `os.TempDir()` weakens the cache-security model.** It may still be workable with a private subdirectory, but it is a deviation from the "OS user cache dir with 0700 perms" decision and deserves explicit justification.
- **[LOW] Cross-process locking is not really proven by goroutine-only tests.** The flock/`LockFileEx` behavior that matters most is inter-process, not just concurrent goroutines inside one process.

### Suggestions

- Lock the GitHub Releases asset naming scheme before implementation starts. Make the asset name itself platform-specific, not just the URL path.
- Resolve error-sentinel ownership in `05-01` and keep it one-way. The cleanest option is usually: define canonical sentinels in `internal/bootstrap`, alias them from root `errors.go`, and never duplicate them.
- Change `05-03` to depend on `05-02`, or explicitly recast it as "write failing tests only." Right now the wave structure overpromises parallelism.
- Decide whether bootstrap tests need package-private hooks. If yes, keep them in `package bootstrap`; if no, export the required test seams intentionally.
- Rework `activeLibrary()` so bootstrap/path resolution happens outside `libraryMu`, then recheck/cache under the lock.
- Either add a staged-artifact E2E task to this phase or narrow the success criteria to "validated against staged/httptest artifacts" until the release pipeline exists.
- Extend CLI `verify` to accept `--dest` and target selection, or explicitly document that it only verifies the current-platform runtime cache.
- Add a plan note for repeated failure behavior on `NewParser()`. Even a simple "memoize last bootstrap failure per process for N seconds" would avoid repeated long stalls.

### Risk Assessment

**Overall risk: MEDIUM**

The core design is sound and the requirement coverage is better than average, so this is not a fundamentally broken phase. The risk comes from a small number of high-impact inconsistencies in plan contracts and wave sequencing. If those are cleaned up before execution, the remaining work is mostly straightforward implementation and test hardening.

---

## Consensus Summary

Both reviewers converge on **"the design is sound, pattern reuse is smart, and requirement coverage is strong"**, but they diverge sharply on residual risk: Gemini calls the set APPROVED at LOW risk; Codex flags three HIGH-severity contract-level problems that would cause real churn mid-execution and rates the set MEDIUM. The divergence matters because it points at the single conclusion that actually changes what you do next — the HIGH items from Codex should be resolved before `/gsd-execute-phase`.

### Agreed Strengths

- **Security-first** — SHA-256-before-`dlopen`, Windows absolute-path loading, and `0700` cache perms are treated as load-bearing invariants by both reviewers (Pitfalls #17, #16, #29).
- **Proven pattern reuse** — Lifting lock/download/cache scaffolding from `pure-onnx` is called out by both as the right call that reduces invention risk.
- **Requirement traceability** — Roadmap → CONTEXT → RESEARCH → per-plan acceptance criteria are aligned cleanly, which both reviewers note is unusually good.
- **Fault-injection mindset** — Explicit 11-scenario matrix (plan 06 / 05-VALIDATION.md) for checksum corruption, retry, cancellation, and fallback is praised by both.
- **Dependency hygiene** — stdlib-first for the library, `cobra` scoped to the CLI only, is highlighted as sound architectural discipline.

### Agreed Concerns

Only two items appear (in different framings) in both reviews:

- **Error sentinel ownership (HIGH per Codex / LOW per Gemini)** — Both reviewers noticed the back-and-forth between `internal/bootstrap` and root `purejson` on where sentinels live. Gemini considered it resolved; Codex reads the current plan text as still cycling across plans. Worth a single concrete decision in `05-01` with an `errors.go` alias pattern before execution.
- **Empty `checksums.go` at development time (LOW both)** — Neither reviewer considers it a blocker, but both call out that success criterion 1 ("fresh-machine bootstrap") is not actually exercised by this phase because no real artifacts/hashes exist until Phase 6. Codex frames this more sharply as a success-criteria honesty problem.

### Divergent Views (Codex-only, worth investigating)

These are Codex catches that Gemini missed entirely. Codex's depth on plan-level contracts surfaced concrete inconsistencies Gemini's high-level assessment smoothed over:

- **[HIGH] GitHub fallback asset naming** — Codex claims `libpure_simdjson.so` would collide across linux/amd64 and linux/arm64 (same for `.dylib` across macOS arches). Verify: does `05-05` / `05-VALIDATION.md` actually encode architecture into the GitHub asset filename, or only into the URL path? If the latter, DIST-02's fallback spec is unimplementable as written.
- **[HIGH] Wave 2 parallelism overstated** — Codex argues `05-03` tests reference `BootstrapSync`, `resolveConfig`, `withHTTPClient`, `withGitHubBaseURL` (APIs introduced by `05-02`). Either recast `05-03` as "TDD-first, expect to fail until 05-02 lands" or serialize it after `05-02` in the wave map.
- **[MEDIUM] `libraryMu` scope across network I/O** — Codex asserts `activeLibrary()` acquires `libraryMu` before `resolveLibraryPath()`, meaning if `05-04` preserves its shape verbatim the first-run bootstrap blocks every parser creation for the whole download window. Context explicitly said downloads should happen outside the lock. Needs either an explicit `05-04` rewrite note or a re-examination of `activeLibrary()`'s current shape.
- **[MEDIUM] Bootstrap latency on blocked networks** — No failure memoization means every `NewParser()` on a blocked corporate network re-attempts the full retry cascade. Worth a short mention in `05-02` or `05-04` about caching the last bootstrap failure per process for N seconds.
- **[MEDIUM] Package/export test story** — Codex claims `package bootstrap_test` files reference unexported helpers (`resolveConfig`, `withHTTPClient`, `withGitHubBaseURL`, `defaultCacheDir`). Verify compile correctness: either stay in `package bootstrap` for internal tests or export test seams via `export_test.go`.
- **[MEDIUM] CLI `verify` scope** — Codex argues it only covers current-platform default cache, missing the `--dest` / `--all-platforms` offline bundle flow that is this CLI's whole point.
- **[LOW] Cross-process locking not actually proven** — Goroutine-only tests cover intra-process contention; inter-process flock/`LockFileEx` is the more interesting case and isn't explicitly exercised.

Gemini added three suggestions Codex didn't mention — worth considering on their own merits:
- `PURE_SIMDJSON_CACHE_DIR` env override for ephemeral CI home directories.
- Version-stamped `User-Agent` header on downloads.
- Per-platform progress feedback in the CLI `fetch --all-platforms` flow.

### Recommendation

Before running `/gsd-execute-phase 05`, resolve Codex's three HIGH items. They're plan-contract fixes, not design changes — most can land as targeted edits in `05-01`, `05-03`, `05-04`, and `05-05` via `/gsd-plan-phase 05 --reviews`. The MEDIUM items are worth a second pass but don't block execution.
