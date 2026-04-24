---
quick_id: 20260424
slug: phase8-followup-feedback
status: in_progress
created_at: 2026-04-24T06:16:57Z
---

# Phase 8 Follow-Up Feedback

Address the convergent follow-up feedback from the Phase 8 materializer review.

## Scope

- Add an observable depth-limit status/sentinel and regression coverage.
- Tighten `Doc.isClosed` and materializer comments.
- Fill the missing string-span adversarial frame test and warning test cleanup.
- Preserve the Phase 8 benchmark improvement gate.

## Acceptance

- Depth-limit errors map to `ErrDepthLimitExceeded`.
- A deeply nested array regression test exercises the depth-limit path.
- Fast materializer adversarial-frame tests cover both key and string nil-pointer spans.
- `go test ./...`, `make verify-contract`, and the Phase 8 benchmark improvement gate pass.
