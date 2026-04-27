---
quick_id: 260427-gwy
status: complete
completed: 2026-04-27
related_pr: 22
commits:
  - fcc25b2  # task 1: ABI sync comments
  - 605cc16  # task 2: semver_tuple type fix
  - 86c0a93  # task 3: 0.1.1 boundary test
  - 0d45cdd  # task 4: pre-release doc + test
  - 90fd406  # task 5: check_readiness.sh comment
---

# Quick Task 260427-gwy — Summary

Applied 5 of the 9 PR-review items from the Claude bot's review on PR #22. The other 4 (#1, #3, #5, #7) were rejected as not-actionable in the prior `/pr-feedback` analysis (style nit, false alarm in scope, duplicate of #2, low-value coverage).

## Outcomes per task

| # | Item | Files touched | Commit |
|---|------|---------------|--------|
| 1 | Bidirectional ABI sync comments | `internal/bootstrap/abi_assertion.go`, `scripts/release/check_bootstrap_abi_state.py` | fcc25b2 |
| 2 | `semver_tuple` honors `tuple[int, int, int]` annotation | `scripts/release/check_bootstrap_abi_state.py` | 605cc16 |
| 3 | New `test_rejects_0_1_1_as_stale_for_current_abi` | `scripts/release/test_check_bootstrap_abi_state.py` | 86c0a93 |
| 4 | Document + test `0.1.2-dev` pre-release behavior | `scripts/release/check_bootstrap_abi_state.py`, `scripts/release/test_check_bootstrap_abi_state.py` | 0d45cdd |
| 5 | Inline comment on layered bootstrap.Version check | `scripts/release/check_readiness.sh` | 90fd406 |

## Verification

- `go build ./...` — clean (the ABI canary at `internal/bootstrap/abi_assertion.go:11-12` still compiles, confirming no semantic drift from the comment additions).
- `python3 -m unittest scripts.release.test_check_bootstrap_abi_state` — 7 tests pass (was 5 before this task; 2 new tests added).
- `bash -n scripts/release/check_readiness.sh` — syntax valid.

## Notes

- Task 4 added a comment that makes explicit a previously implicit contract: pre-release suffixes are accepted by `SEMVER_RE` and parsed as their base release. The new `test_accepts_prerelease_version_as_base_release` locks this so a future regex tightening can't silently change the behavior without breaking a test.
- Task 1 keeps `abiVersionForBootstrapVersion_0_1_2` named as-is — the bot also flagged the naming style as a nit (item #1) but the current name's link to `bootstrap.Version` is intentional and self-documenting.
- All 5 commits are functionally inert except the indirect type-correctness of #4 (runtime tuple is identical; only mypy/pyright behavior changes).
