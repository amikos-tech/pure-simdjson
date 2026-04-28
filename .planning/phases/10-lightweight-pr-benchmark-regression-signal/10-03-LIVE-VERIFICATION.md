# Phase 10 Plan 03 Live Verification Checklist

Use this checklist after the Phase 10 workflow files are merged to `main`. These checks validate live GitHub Actions behavior that cannot be proven fully from local linting.

## PR Surface Block

Add this block to the Phase 10 PR description or as a follow-up PR comment:

```markdown
## Phase 10 Live Verification

This PR adds two workflow files whose local gates pass (`actionlint`, `yq`, parser tests, orchestrator tests). Final completion requires live GitHub Actions verification after merge to `main`.

- [ ] `main benchmark baseline` appears in the Actions sidebar.
- [ ] `pr benchmark` appears in the Actions sidebar.
- [ ] Run `main benchmark baseline` via `workflow_dispatch` on `main`.
- [ ] Baseline run finishes green.
- [ ] A `pr-bench-baseline-<sha>` cache entry exists.
- [ ] Baseline artifact contains `head.bench.txt`.
- [ ] Open or update a small non-doc PR.
- [ ] `pr benchmark` runs on that PR.
- [ ] `Restore main-baseline cache` has non-empty `cache-matched-key`.
- [ ] Step summary shows the PR benchmark result.
- [ ] Sticky PR comment is posted or updated, or fork-token denial is harmless because the step is `continue-on-error`.
- [ ] PR benchmark job finishes green in advisory mode.
- [ ] Delete `pr-bench-baseline-*` cache and rerun a PR benchmark.
- [ ] Cache-miss run reports `advisory bypass` and still exits green.
- [ ] Cache-miss artifacts include `head.bench.txt`, `summary.json`, and `markdown.md`.
- [ ] Push two commits quickly to the same PR.
- [ ] Earlier run is cancelled and latest run updates the sticky comment.
- [ ] `grep -nE "REQUIRE_NO_REGRESSION" .github/workflows/pr-benchmark.yml CHANGELOG.md scripts/bench/check_pr_regression.py` returns exactly three matches.
```

## Evidence To Record

When the checklist is complete, record:

- Baseline workflow run URL.
- Baseline cache key observed.
- PR benchmark workflow run URL with cache hit.
- PR URL where the sticky comment appeared or was intentionally unavailable due to fork permissions.
- Cache-miss workflow run URL.
- Concurrency cancellation run pair URLs.
- Output of the `REQUIRE_NO_REGRESSION` grep command.

## Completion Rule

Only create `10-03-SUMMARY.md` and mark Phase 10 complete after the checklist above is verified. If any live check fails, keep Plan 10-03 open and capture the failing run URL plus the failing step name before applying a follow-up fix.
