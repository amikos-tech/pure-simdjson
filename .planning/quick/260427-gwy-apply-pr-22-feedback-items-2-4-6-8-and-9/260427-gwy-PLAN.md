---
quick_id: 260427-gwy
description: Apply PR #22 feedback items #2, #4, #6, #8, and #9-comment
created: 2026-04-27
status: planned
related_pr: 22
---

# Quick Task 260427-gwy: Apply PR #22 feedback

Five surgical fixes from the Claude-bot code review on PR #22. Each maps to one feedback item; items #1, #3, #5, #7 were rejected as not-actionable in the prior `/pr-feedback` analysis.

## Tasks

### Task 1 — #2: Bidirectional ABI sync comments

**Files:**
- `internal/bootstrap/abi_assertion.go`
- `scripts/release/check_bootstrap_abi_state.py`

**Action:**
- In `abi_assertion.go`, extend the existing comment block with a pointer to `scripts/release/check_bootstrap_abi_state.py:ABI_MINIMUM_VERSION`.
- Above the `ABI_MINIMUM_VERSION` dict in the Python script, add a comment pointing back to `internal/bootstrap/abi_assertion.go`.

**Verify:** Both files reference each other. `pytest scripts/release/test_check_bootstrap_abi_state.py` still passes. `go build ./...` still passes.

**Done when:** Both sites discoverable from each other via grep on the partner path.

---

### Task 2 — #4: Fix `semver_tuple` return-type lie

**File:** `scripts/release/check_bootstrap_abi_state.py`

**Action:** Replace the generator-based tuple construction with explicit indexing so the runtime return matches the `tuple[int, int, int]` annotation:

```python
return int(match.group(1)), int(match.group(2)), int(match.group(3))
```

**Verify:** `pytest scripts/release/test_check_bootstrap_abi_state.py` passes. Optional: `mypy --strict scripts/release/check_bootstrap_abi_state.py` (if mypy available).

**Done when:** Return statement uses explicit positional `match.group(N)` calls, not a generator.

---

### Task 3 — #6: Add stale-version boundary test for `0.1.1`

**File:** `scripts/release/test_check_bootstrap_abi_state.py`

**Action:** Add a test method `test_rejects_0_1_1_as_stale_for_current_abi` that runs the checker with `bootstrap_version="0.1.1"` and `requested_version="0.1.1"` (so the version-mismatch check doesn't short-circuit first), expecting "stale bootstrap.Version" in stderr.

**Verify:** New test passes. All existing tests still pass.

**Done when:** Boundary case `0.1.1 < 0.1.2 minimum for ABI 0x00010001` has explicit coverage.

---

### Task 4 — #8: Document and test pre-release semver parsing

**Files:**
- `scripts/release/check_bootstrap_abi_state.py` (one-line comment on `SEMVER_RE`)
- `scripts/release/test_check_bootstrap_abi_state.py` (one new test)

**Action:**
- Add a comment above `SEMVER_RE` explaining that pre-release suffixes (`-dev`, `-rc.1`, etc.) are accepted and parsed as their base release. The exact-string `bootstrap_version != requested_version` check upstream handles dev-snapshot rejection separately.
- Add `test_accepts_prerelease_version_as_base_release` that uses `bootstrap_version="0.1.2-dev"` and `requested_version="0.1.2-dev"` (so the exact-match check passes), verifying the checker accepts it.

**Verify:** New test passes. All existing tests still pass.

**Done when:** Behavior is documented in source AND locked by a test.

---

### Task 5 — #9-comment: Clarify layered version check in `check_readiness.sh`

**File:** `scripts/release/check_readiness.sh`

**Action:** Add a single comment line above the `source_version=` extraction (line ~80) explaining that this is the fast-fail pre-gate; the canonical check is the Python script invoked below.

**Verify:** `bash -n scripts/release/check_readiness.sh` (syntax check) passes. Behavior unchanged.

**Done when:** Reviewer confusion about the apparent duplication is preempted by an inline comment.

---

## must_haves

- **Truths:**
  - Items #1, #3, #5, #7 from the PR review are explicitly out of scope and not addressed here.
  - All five edits are non-functional (item #4 is a type-level fix; the runtime tuple is identical).
- **Artifacts:** `internal/bootstrap/abi_assertion.go`, `scripts/release/check_bootstrap_abi_state.py`, `scripts/release/test_check_bootstrap_abi_state.py`, `scripts/release/check_readiness.sh`.
- **Key links:** PR #22 (https://github.com/amikos-tech/pure-simdjson/pull/22), prior `/pr-feedback` analysis in this conversation thread.
