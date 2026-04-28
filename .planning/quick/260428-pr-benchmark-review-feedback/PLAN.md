---
status: complete
created: 2026-04-28
---

# Quick Task: PR Benchmark Review Feedback

Address Phase 10 PR review feedback for the lightweight PR benchmark regression signal.

## Tasks

1. Harden benchmark evidence handling.
   - Fail the PR benchmark orchestrator when the selected benchmark regex produces no benchmark rows.
   - Fail the parser closed when a tracked tier row appears under an unrecognized metric header.
   - Remove the unused parser import sentinel.

2. Tighten workflow behavior and comments.
   - Save the rolling main baseline cache only on successful baseline capture.
   - Clarify the deliberate cache restore fallback key.
   - Replace the internal planning-id comment on the blocking flip with behavioral wording.

3. Expand regression coverage.
   - Add p-value boundary, non-tier exclusion, unrecognized metric, empty capture, failure cleanup, stale-output replacement, tier-list sync, and YAML parse coverage.

