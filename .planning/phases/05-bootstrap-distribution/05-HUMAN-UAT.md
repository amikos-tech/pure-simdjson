---
status: complete
phase: 05-bootstrap-distribution
source: [05-VERIFICATION.md]
started: 2026-04-20T15:45:00Z
updated: 2026-04-23T12:29:00Z
resolution: live-bootstrap-passed; corporate-firewall-moved-to-backlog
---

## Current Test

[testing complete — moved to backlog]

## Tests

### 1. Fresh-machine end-to-end bootstrap against live R2 + GitHub Releases
expected: rm -rf ~/Library/Caches/pure-simdjson; NewParser() downloads from releases.amikos.tech, verifies SHA-256, caches, and parses successfully on all 5 target platforms
why_human: Required published artifacts plus public `SHA256SUMS` metadata and hosted runners across the supported target matrix.
result: pass
evidence: GitHub Actions run `24835017953` dispatched `public-bootstrap-validation.yml` for `version=0.1.0`; all five R2 hosted-runner jobs and all three GitHub fallback jobs passed.

### 2. Corporate-firewall workaround against a real proxy blocking releases.amikos.tech
expected: With PURE_SIMDJSON_BINARY_MIRROR set to internal mirror, bootstrap succeeds; with GH fallback reachable, R2-blocked environment still bootstraps
why_human: Requires corporate network environment and cannot be automated meaningfully in CI. Documented in 05-VALIDATION.md Manual-Only Verifications section. Deferred to Phase 7 or user-reported validation.
result: blocked
blocked_by: other
reason: Tracked as Phase 999.5 in ROADMAP.md backlog. Requires real corporate-network environment or proxy emulation.

## Summary

total: 2
passed: 1
issues: 0
pending: 0
skipped: 0
blocked: 1

## Gaps
