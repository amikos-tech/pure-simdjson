---
phase: 6
reviewers: [gemini, claude]
reviewer_status:
  gemini: completed
  claude: completed
reviewed_at: 2026-04-21T04:16:15Z
plans_reviewed:
  - 06-01-PLAN.md
  - 06-02-PLAN.md
  - 06-03-PLAN.md
  - 06-04-PLAN.md
  - 06-05-PLAN.md
  - 06-06-PLAN.md
---

# Cross-AI Plan Review — Phase 6 (CI Release Matrix + Platform Coverage)

## Gemini Review

### Gemini Output

The implementation plans for Phase 6 provide a robust and highly automated release pipeline for `pure-simdjson`. The strategy centers on a **two-workflow model** (Prepare and Publish) that elegantly solves the circular dependency between artifact checksums and source-controlled metadata. The inclusion of a loopback HTTP mirror for smoke testing ensures that the "real" bootstrap path is verified before any library reaches production, making this a high-integrity design.

## Strengths

- **Two-Workflow Architecture**: Separating state preparation (writing version/checksums to source) from publication (tagging and uploading) ensures that the tagged commit is always "correct" and coherent with the distributed binaries.
- **Loopback Bootstrap Smoke**: Using `PURE_SIMDJSON_BINARY_MIRROR` to point at a local HTTP server serving staged artifacts is a brilliant verification strategy. it exercises the full HTTP stack, checksum verification, and OS-native caching logic exactly as a consumer would.
- **ABI & Compatibility Guards**: The use of `manylinux2014` containers combined with `objdump -T` glibc-floor checks and `nm`/`dumpbin` export audits provides strong guarantees against runtime regressions on older Linux systems.
- **Deterministic Tooling**: Python-based state management and manifest-driven packaging reduce the risk of manual errors in the release process.
- **Alpine Escape-Hatch Validation**: The plans fulfill the Alpine requirement by verifying the documented `PURE_SIMDJSON_LIB_PATH` bypass in a real container, ensuring users on musl-based systems have a verified path forward.

## Concerns

- **[MEDIUM] Digest Drift Risk**: `release.yml` rebuilds artifacts and asserts their digests match the `checksums.go` written by `release-prepare.yml`. While necessary for security, subtle changes in CI runner environments (especially macOS/Windows) could cause rebuilds to produce slightly different binaries, failing the release.
- **[LOW] R2 Prefix Collision**: The `publish_r2.sh` script uses a "list-then-upload" approach for immutability. While effective, a failed/retried job might leave a partial release that blocks subsequent attempts without manual cleanup.
- **[LOW] Python Environment**: The plans rely on `python3` for critical path tasks (state rewrite, server). While standard, ensuring the correct version is available on all runners (especially Windows) is a minor setup detail to watch.

## Suggestions

1. **Pin Build Toolchains**: Use exact versions/hashes for `dtolnay/rust-toolchain` and `manylinux` images (rather than `@stable` or `latest`) to maximize binary reproducibility between the preparation and publication runs.
2. **Digest Mismatch Diagnostics**: In `assert_prepared_state.py`, ensure that if a mismatch occurs, both the "Expected (Committed)" and "Actual (Rebuilt)" digests are printed to the GitHub Actions `GITHUB_STEP_SUMMARY` for immediate debugging.
3. **Artifact Retention**: In `release-prepare.yml`, ensure the `manifest.json` and built artifacts are uploaded as GitHub Actions artifacts. This allows a human or agent to verify the state manually before the preparation PR is merged.
4. **Cosign Verification Examples**: In the runbook, include a specific example of how to verify the `SHA256SUMS` file itself, in addition to the raw library blobs.

## Risk Assessment: LOW

The design is defensive and prioritizes correctness over simplicity. By gating the release on three different smoke tests (Native, Go Bootstrap, and Alpine) and enforcing digest coherence, the plan minimizes the risk of shipping broken or incompatible binaries. The "prepare-then-publish" pattern is a proven industry standard for this type of distribution model.

---

## the agent Review

### the agent Output

# Phase 6 Plan Review

## 1. Summary

The six plans cleanly decompose the phase into: shared tooling (06-01), platform builds split GNU/linux (06-02) and darwin+windows (06-03), hard verification gates (06-04), the two-workflow release path (06-05), and operator surface (06-06). The dependency graph is sensible (01 → {02,03} → 04 → 05 → 06) and the "prepare-then-tag" model with a rebuild-and-assert digest gate is the right shape. The package-from-one-source-of-truth scaffold in 06-01 is strong: it prevents the R2/GitHub-name divergence that is the single most common release-matrix footgun.

The plan is well-scoped in surface area but under-scoped on two load-bearing properties: **build reproducibility** and **prep-branch → main → tag flow**. Both of these can silently invalidate the whole release gate.

`★ Insight ─────────────────────────────────────`
- The "rebuild at tag time and assert digests match source-committed checksums" pattern is elegant because it makes tampering after prep visible — but it *only* works if builds are reproducible. Rust release builds are not byte-reproducible by default across runners; this is the phase's biggest risk.
- Splitting the public asset name (`libpure_simdjson-linux-amd64.so`) from the cache/R2 filename (`libpure_simdjson.so`) in one packaging script is correct — bootstrap downloads URL-layout paths, humans download flat-namespace assets. Getting those from two sources of truth is how naming drift enters the repo.
`─────────────────────────────────────────────────`

## 2. Strengths

- **One packaging source of truth** (06-01 Task 1). R2 key + GitHub asset name + manifest row all emitted by one helper. This eliminates the top cause of matrix drift.
- **Deterministic, tested bootstrap-state rewrite** (06-01 Task 2) with explicit sort order, idempotence test, and comment-block preservation. Exactly the right rigor for generated-into-source code.
- **manylinux2014 as the default** (not zig) with an explicit `objdump -T`/`nm -D` gate (06-02). The glibc floor is proven, not assumed.
- **Real bootstrap smoke via loopback mirror** (06-04). Testing the actual download-verify-load path — not a `PURE_SIMDJSON_LIB_PATH` shortcut — is the honest gate. Segregating Alpine to the escape-hatch path is correct.
- **Prep-then-tag with digest assertion** (06-05). The tag workflow cannot publish if its rebuilt manifest disagrees with committed `checksums.go`. This makes the tagged commit self-coherent.
- **Cosign + immutable R2 prefix** (06-05). Provenance + append-only public store.
- **Single runbook backing a single agent skill** (06-06). Prevents the "agent invents its own release process" failure mode.

## 3. Concerns

### HIGH

**H1 — Build reproducibility is assumed, not engineered (06-05).** Plan 06-05 Task 2 requires that the tag workflow's freshly built artifacts produce digests matching the committed `checksums.go`. Rust release builds are *not* bit-reproducible across runner instances by default: embedded rustc version metadata, path remapping, codegen-units ordering, debuginfo, linker timestamps (MSVC, mach-o), and build-time env (e.g. `HOSTNAME`) all perturb the hash. No plan specifies the reproducibility controls (pinned toolchain channel + commit, `RUSTFLAGS="--remap-path-prefix=... -C codegen-units=1"`, `-C strip=symbols` or post-link strip, `SOURCE_DATE_EPOCH`, deterministic archive/linker flags per platform, locked `Cargo.lock` + vendored deps). As written, the publish gate will almost certainly fail on first use, and the natural fix ("just relax the gate") destroys the guarantee.

**H2 — Prep branch → main → tag flow is undefined (06-05 Task 1).** The prep workflow pushes to `release-prep/v<version>` and "does not create the final tag here." Nothing says how that branch reaches `main`, whether `main` is protected, whether tags must be on the merge commit, or who opens/reviews the PR. If an operator tags `release-prep/v0.1.0` directly, `main` diverges from the released source. The runbook in 06-06 must pin this down, and probably the readiness gate should refuse to bless a tag target that is not a descendant of `origin/main`.

**H3 — Bootstrap env-var contract is assumed to exist.** Plan 06-04 uses `PURE_SIMDJSON_BINARY_MIRROR` and `PURE_SIMDJSON_DISABLE_GH_FALLBACK`. Plan 06-05's locked-decisions list in the phase context mentions `PURE_SIMDJSON_LIB_PATH` for Alpine but the mirror/fallback envs are not confirmed against Phase 5 `internal/bootstrap`. If those aren't the actual env names, the "real bootstrap path" smoke silently falls through to the default GitHub URL and tests a different artifact than the one staged. 06-01 should add an explicit consistency grep/test between the scripts and `internal/bootstrap/url.go`.

### MEDIUM

**M1 — `check_readiness.sh` and `assert_prepared_state.py` can drift.** Two gates encode overlapping rules (version match, five keys present). No plan makes one the source of truth for the other. Recommend `check_readiness.sh --strict` invokes the Python script in a "dry" mode rather than reimplementing the checks.

**M2 — Runner labels not validated.** `macos-15-intel` (06-03) and `ubuntu-24.04-arm` (06-02) should be cross-checked against currently available GHA labels at the time the workflow lands. GHA runner labels change; a label typo fails the workflow with `No runner matching the specified labels`.

**M3 — Cosign mode (keyless-OIDC vs key-pair) not specified.** Affects which secrets are required, what `cosign verify-blob` args end up in the runbook, and whether the job needs `id-token: write` permissions. 06-05 should call this out; 06-06's runbook needs the exact verify command.

**M4 — MSVC CRT runtime dependency is not audited (06-03).** `dumpbin /EXPORTS` only inspects the export table. `dumpbin /DEPENDENTS` (or `dumpbin /IMPORTS`) should also be run to confirm what Visual C++ runtime the DLL requires, and that list should be documented. Consumers without `vcruntime140.dll` will see a load failure that bootstrap cannot diagnose.

**M5 — Artifact flow across jobs is implicit in 06-04.** The Go packaged-artifact smoke needs all five built libraries assembled into one staged tree on one job. The plan does not explicitly call out `actions/upload-artifact` / `download-artifact` wiring, digest preservation across upload, or what happens if an artifact is missing. Add an explicit "assemble staged root" task.

**M6 — No concurrency guard on release workflows (06-05).** Two concurrent `workflow_dispatch` runs of `release-prepare.yml` for the same version can push conflicting commits. Add `concurrency: group: release-prepare-${{ inputs.version }}, cancel-in-progress: false`.

**M7 — Action versions pinned by major only (06-05).** `sigstore/cosign-installer@v3`, `action-gh-release@v2`, `ilammy/msvc-dev-cmd@v1` are moving refs. For release-path workflows, pin by commit SHA; otherwise a supply-chain compromise of any one action publishes arbitrary binaries signed by your OIDC identity.

**M8 — Alpine image unpinned (06-04).** `alpine:latest` defeats determinism for a hard gate. Pin to a digest.

### LOW

**L1 — Test pollutes working tree (06-01 Task 2).** `test_update_bootstrap_release_state.py` should copy `internal/bootstrap/*.go` into a tempdir and point the script at it, not mutate the real files during the unit test.

**L2 — `check_readiness.sh` "exactly five keys" is brittle (06-06).** If the matrix ever adds musl or another target, this check silently blocks. Prefer "at least the five required keys" or drive the expected set from one constant shared with `update_bootstrap_release_state.py`.

**L3 — No GitHub build-provenance attestation alongside cosign.** `actions/attest-build-provenance` is cheap, orthogonal to cosign, and useful for downstream verifiers. Optional.

**L4 — Retention of `release-prep/v*` branches is not specified.** They accumulate. A branch-cleanup note in the runbook is enough.

**L5 — `nm -D --defined-only` export audit pattern is `pure_simdjson_` (06-02).** This matches by prefix; good. But it doesn't bound the surface (anything that happens to start with that prefix passes). Consider comparing against a checked-in expected-exports list generated from the Rust shim's `#[no_mangle]` symbols.

## 4. Suggestions

1. **Add a dedicated reproducibility task to 06-02/06-03 (or a new 06-01 task).** Pin toolchain (`rust-toolchain.toml` with SHA), set `RUSTFLAGS="-C codegen-units=1 --remap-path-prefix=$PWD=. -C strip=symbols"` (plus per-platform linker flags: `/Brepro` on MSVC, `-Wl,--build-id=none` on Linux, `-Wl,-no_uuid` on macOS), export `SOURCE_DATE_EPOCH`, strip post-link, and prove reproducibility by building twice in the prep workflow and asserting the digests match *before* committing them. Failing fast in prep beats failing at tag.
2. **Tighten 06-05 with an explicit merge + tag-source-gate.** Require the prep branch to land on `main` via PR; require `release.yml` to verify `git merge-base --is-ancestor HEAD origin/main` on the tagged SHA before publishing.
3. **Make `check_readiness.sh --strict` shell out to `assert_prepared_state.py`** rather than reimplement checks. Single source of truth.
4. **Add a 06-01 consistency test** that greps `PURE_SIMDJSON_BINARY_MIRROR` etc. against `internal/bootstrap/` to fail if the env-var names drift from Phase 5 reality.
5. **Pin actions by SHA** in release workflows; add `concurrency:` block to both; add `permissions:` block scoped per job.
6. **Specify cosign mode** (keyless OIDC recommended) and pre-write the verify commands into 06-06's runbook.
7. **06-03: add `dumpbin /DEPENDENTS`** and document the VC++ Redistributable expectation in `docs/bootstrap.md`.
8. **06-04: add explicit staged-tree assembly step** (download all per-platform artifacts into the `v<version>/...` layout) before the Go loopback smoke, and fail if any of the five expected files is missing.
9. **06-02: pin manylinux container by digest**, not tag.

## 5. Overall Risk Assessment

**Moderate-High, driven almost entirely by H1 (reproducibility) and H2 (prep→tag flow).**

The scaffolding, gating, and naming contracts are thoughtful and the plan is in good shape structurally. But the tag-time digest-assert gate — which is the *core* integrity guarantee of the whole phase — will not survive contact with real Rust release builds unless reproducibility is engineered explicitly. If that gate is softened to unblock shipping (e.g., "only compare filenames, not digests"), the phase delivers no more assurance than a traditional single-workflow release. And without a defined PR/merge/tag flow, a well-intentioned operator can publish a commit that does not exist on `main`.

Fix H1 and H2, pin action versions and runner/image labels (M7, M8, M2), and the phase is solidly shippable. Leave them open and the first real release attempt will stall at the digest-assert step and force a rushed process redesign under tag-time pressure — exactly the kind of "scope creep under deadline" the user's profile flags as a frustration.

Scope verdict: **appropriately sized, slightly under-scoped on reproducibility and branching** — add one focused task for each rather than expanding every plan.
