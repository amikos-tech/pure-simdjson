---
phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics
plan: 14
subsystem: release-validation
tags: [abi-1.2, bigint, release, r2, github-actions, bootstrap]

# Dependency graph
requires:
  - phase: 11-13
    provides: Fully tested ABI 1.2 source state and packaged/public smoke contracts
  - phase: 06.1
    provides: Fresh-runner R2 and GitHub-fallback public bootstrap validation workflow
provides:
  - Main-anchored annotated v0.1.7 tag and successful CI-only five-platform release evidence
  - Published ABI 1.2 assets, signatures, certificates, and checksum metadata
  - Successful five-target R2 and three-target GitHub-fallback bootstrap proof
  - Anonymous public endpoint evidence for SHA256SUMS and latest.json
affects: [12-dom-navigation-and-utilities, 16-v0.2-release, public-bootstrap-validation]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Immutable superseding patch versions for release recovery
    - Tag-driven CI publication followed by a separate fresh-runner public bootstrap gate
    - Operator transcript plus independently auditable hosted and public endpoint evidence

key-files:
  created:
    - .planning/phases/11-upstream-simdjson-refresh-bigint-and-diagnostics/11-14-SUMMARY.md
  modified:
    - .planning/STATE.md
    - .planning/ROADMAP.md

key-decisions:
  - "The final approved Phase 11 compatibility release is v0.1.7 with ABI 0x00010002; it supersedes the plan-time v0.1.5 identity and is not the Phase 16 v0.2 release."
  - "Release recovery preserved tag immutability by using new patch versions for corrected source; no published tag was moved or replaced."
  - "The operator-provided strict-readiness success on the exact tag target is retained as evidence and is not rerun after main advanced, because the script's depth-1 fetch can no longer prove older ancestry reliably."
  - "Phase 11 closes only after both the CI-only release workflow and the separate Phase 06.1 public bootstrap workflow pass."

patterns-established:
  - "Hosted release proof: exact tag target -> successful release matrix -> published signed assets."
  - "Public bootstrap proof: published tag source -> five R2 targets -> one fallback target per OS family."

requirements-completed: [UP-01, NUM-01, NUM-02, DIAG-01, DIAG-02, LIMIT-01]

# Metrics
duration: 9min
completed: 2026-07-29
---

# Phase 11 Plan 14: ABI 1.2 Publication and Public Bootstrap Summary

**Outcome: PASSED**

**Annotated v0.1.7 on the squash-merged ABI 1.2 source produced a signed five-platform CI release and passed fresh-runner R2 plus forced GitHub-fallback bootstrap validation**

## Performance

- **Duration:** 9 min for post-checkpoint evidence audit and closeout
- **Started:** 2026-07-29T11:45:58Z
- **Completed:** 2026-07-29T11:54:45Z
- **Tasks:** 1
- **Files modified:** 3 planning files

## Accomplishments

- Confirmed the annotated `v0.1.7` tag object
  `cd153ae770745dad124750ec8dd765eb1afdb83e` resolves to the squash commit
  `ab86c2e1e666c6c313d1dd951c37a8c43538c407`, which is an ancestor of
  `origin/main` at `7c1051b8758139645ce437a8ae38ca75fc8f2174`.
- Recorded the operator's zero-exit strict-readiness result on that exact tag
  target, including
  `bootstrap ABI state ok: version 0.1.7, abi 0x00010002`.
- Audited successful tag-driven publication across linux/amd64, linux/arm64,
  darwin/amd64, darwin/arm64, and windows/amd64, including native and packaged
  smoke gates, Alpine smoke, signatures, certificates, checksum metadata, R2
  publication, and the GitHub Release.
- Audited successful public bootstrap from R2 on all five targets and forced
  GitHub fallback on linux/amd64, darwin/arm64, and windows/amd64.
- Confirmed anonymous clients receive the real `v0.1.7` checksum manifest and
  `latest.json` after the Access Bypass correction.

## Task Commits

Task 1 was an operator-only publication checkpoint. It changed no repository
source and therefore has no production task commit; its evidence and the
required tracking updates are recorded in the plan metadata commit.

## Publication Identity

| Property | Verified value |
|---|---|
| Final intermediate release | `v0.1.7` |
| Public ABI | `0x00010002` |
| Squash/tag target | `ab86c2e1e666c6c313d1dd951c37a8c43538c407` |
| Annotated tag object | `cd153ae770745dad124750ec8dd765eb1afdb83e` |
| `v0.1.7^{commit}` | `ab86c2e1e666c6c313d1dd951c37a8c43538c407` |
| Audited `origin/main` | `7c1051b8758139645ce437a8ae38ca75fc8f2174` |
| Main ancestry | PASS — the tag target is an ancestor of `origin/main` |
| Artifact role | Intermediate Phase 11 ABI 1.2 compatibility release |
| Phase 16 boundary | This is not the final v0.2 release |

The tagged source itself pins bootstrap version `0.1.7`; its generated C
header, Go FFI mirror, and bootstrap ABI canary all identify ABI
`0x00010002`.

## Strict Readiness Evidence

The operator ran strict readiness on the exact future tag target before tag
publication. The command exited `0` and included:

```text
bootstrap ABI state ok: version 0.1.7, abi 0x00010002
strict release readiness checks passed for version 0.1.7
```

This closeout intentionally does not rerun
`bash scripts/release/check_readiness.sh --strict --version 0.1.7`.
That script performs a depth-1 `origin/main` fetch, so after `main` advances it
can produce a false negative for a valid older ancestor. The hosted
`verify tag source state` job independently fetched full history and passed
both tag ancestry and committed release-version checks on the exact tag.

## Release Workflow Evidence

- **Run:** https://github.com/amikos-tech/pure-simdjson/actions/runs/30030435051
- **Workflow:** `release`
- **Trigger/head:** tag push at
  `ab86c2e1e666c6c313d1dd951c37a8c43538c407`
- **Result:** `completed / success`
- **Started:** `2026-07-23T17:40:47Z`
- **Completed:** `2026-07-23T17:44:57Z`

Successful jobs:

| Job | Result |
|---|---|
| verify tag source state | success |
| linux build (linux-amd64) | success |
| linux build (linux-arm64) | success |
| darwin build (darwin-amd64) | success |
| darwin build (darwin-arm64) | success |
| windows build (windows-amd64) | success |
| alpine smoke (`PURE_SIMDJSON_LIB_PATH` escape hatch) | success |
| release publish | success |

The job-step audit also confirmed success for every five-target native
`ffi_export_surface.c + minimal_parse.c` smoke, the assembled Go
packaged-artifact smoke, raw-asset and `SHA256SUMS` signing, cosign
verification, immutable R2 publication, and GitHub Release publication.

## Published Release Evidence

- **Release:** https://github.com/amikos-tech/pure-simdjson/releases/tag/v0.1.7
- **State:** published, non-draft, non-prerelease
- **Published:** `2026-07-23T17:44:54Z`
- **Assets:** 18

The asset set is complete: five platform binaries, a `.sig` and `.pem`
sidecar for each binary, and the `SHA256SUMS`, `SHA256SUMS.sig`, and
`SHA256SUMS.pem` trio.

Anonymous endpoint audit on 2026-07-29:

| Endpoint | HTTP | Content-Type | Bytes | Evidence |
|---|---:|---|---:|---|
| `https://releases.amikos.tech/pure-simdjson/v0.1.7/SHA256SUMS` | 200 | `text/plain` | 538 | Five real SHA-256 rows, one per supported target |
| `https://releases.amikos.tech/pure-simdjson/latest.json` | 200 | `application/json` | 202 | `version` is `v0.1.7`; checksum URL points at the immutable manifest |

## Public Bootstrap Validation Evidence

- **Run:** https://github.com/amikos-tech/pure-simdjson/actions/runs/30448288140
- **Workflow:** `public bootstrap validation`
- **Trigger:** `workflow_dispatch`
- **Head:** `7c1051b8758139645ce437a8ae38ca75fc8f2174`
- **Result:** `completed / success`
- **Started:** `2026-07-29T11:37:34Z`
- **Completed:** `2026-07-29T11:39:06Z`

| Target | Release build/native smoke | R2 public bootstrap | Forced GitHub fallback |
|---|---|---|---|
| linux/amd64 | success | success | success |
| linux/arm64 | success | success | not in documented fallback subset |
| darwin/amd64 | success | success | not in documented fallback subset |
| darwin/arm64 | success | success | success |
| windows/amd64 | success | success | success |

Every validation job first checked out and validated the published
`v0.1.7` tag source, then ran the corresponding public bootstrap smoke. The
scheduled-failure notification job was skipped as expected because this
manual validation succeeded.

## Release Recovery Narrative

Plan 11-01 originally selected `v0.1.5` before publication work began. During
operator-led recovery, Alpine smoke and release-source path corrections
changed source after the earlier patch attempts. Recovery followed the plan's
immutability rule: the existing tags were not moved or replaced; each
corrected source state received a new patch version, first `v0.1.6` and then
`v0.1.7`.

Only `v0.1.7` is the approved Plan 11-14 outcome because it is the version for
which strict readiness, the complete tag-driven release workflow, anonymous
public endpoints, and the separate Phase 06.1 validation all passed. It still
identifies ABI `0x00010002` and remains an intermediate compatibility release,
leaving the final v0.2 release to Phase 16.

## Decisions Made

- Accepted `v0.1.7` as the final Phase 11 compatibility publication instead of
  carrying the stale plan-time `v0.1.5` identity into closeout.
- Counted hosted `verify tag source state` as the independent ancestry/source
  proof after `main` advanced; did not manufacture a new strict-readiness
  result from a shallow fetch.
- Required both publication and public-bootstrap runs to pass before setting
  `Outcome: PASSED`.

## Deviations from Plan

None - release recovery used the plan's explicit new-version contingency, kept
tags immutable, and repeated the required publish/validate path through the
successful `v0.1.7` outcome.

## Issues Encountered

- Earlier patch attempts were superseded while correcting the Alpine
  release-smoke/source-path flow. New immutable patch versions were used
  instead of changing existing tags.
- Anonymous release endpoints were initially affected by Access policy. The
  operator completed the Access Bypass correction before closeout; both
  endpoints now return the expected public responses and the fresh-runner
  validation is green.

## Authentication Gates

None during closeout. The external publication and Access configuration were
completed by the operator before this continuation resumed.

## User Setup Required

None - the required CI and Access configuration is complete.

## Known Stubs

None. This plan created planning evidence only and introduced no unfinished
production behavior.

## Threat Model Verification

- **T-11-03 (tag/artifact ABI identity):** mitigated by exact annotated-tag
  identity, main ancestry, strict readiness, hosted source verification, ABI
  1.2 native/packaged smokes, and public bootstrap validation.
- **T-11-SC (release supply chain):** mitigated by CI-only publication,
  immutable superseding tags, keyless signatures/certificates, checksum
  metadata, and no manual artifact upload.
- No new endpoint, authentication path, file-access pattern, or schema trust
  boundary was introduced by this evidence-only plan.

## Verification

- `git rev-parse v0.1.7^{tag}` matched the approved annotated tag object.
- `git rev-parse v0.1.7^{commit}` matched the squash release commit.
- `git merge-base --is-ancestor v0.1.7^{commit} origin/main` exited `0`.
- GitHub Actions API inspection reported both workflow runs
  `completed / success` and every required matrix job successful.
- GitHub Releases API inspection reported `draft=false`,
  `prerelease=false`, and 18 expected assets.
- Anonymous HTTP checks returned the expected status, media type, byte length,
  checksum rows, and `latest.json` version.
- The unrelated modified `.planning/config.json` and untracked Phase 10
  learnings file were not altered, staged, deleted, or reverted.

## Next Phase Readiness

- Phase 11 is complete and ready for phase-level verification/transition.
- Phase 12 may build on a publicly available, default-bootstrap-tested ABI 1.2
  compatibility foundation.
- Phase 16 retains ownership of final v0.2 stabilization, evidence, and
  publication.

## Self-Check: PASSED

- This summary exists and contains `Outcome: PASSED`.
- The annotated tag object, tag target, release run, release record, endpoint
  responses, and public-validation run all match the recorded evidence.
- The summary distinguishes the final `v0.1.7` outcome from superseded patch
  attempts and from the future Phase 16 v0.2 release.
- No production source or release artifact was modified during closeout.

---
*Phase: 11-upstream-simdjson-refresh-bigint-and-diagnostics*
*Completed: 2026-07-29*
