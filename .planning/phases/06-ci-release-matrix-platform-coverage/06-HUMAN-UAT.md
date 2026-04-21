---
status: partial
phase: 06-ci-release-matrix-platform-coverage
source: [06-VERIFICATION.md]
started: 2026-04-21T08:14:31Z
updated: 2026-04-21T08:48:32Z
---

## Current Test

[testing paused — 2 items outstanding]

## Tests

### 1. Download a published macOS dylib and clear quarantine
expected: After `xattr -d com.apple.quarantine <path-to-dylib>`, the downloaded dylib loads successfully on a fresh macOS host
result: blocked
blocked_by: release-build
reason: No published `v*` tag exists on `origin` as of 2026-04-21, so there is no released macOS dylib to download and validate against real Gatekeeper behavior.

### 2. Review the generated GitHub release notes for a real tag
expected: The published notes are acceptable for a public release and align with the prepared CHANGELOG entry
result: blocked
blocked_by: release-build
reason: No published `v*` tag exists on `origin` as of 2026-04-21, so there is no real GitHub release page or generated notes to review.

## Summary

total: 2
passed: 0
issues: 0
pending: 0
skipped: 0
blocked: 2

## Gaps
