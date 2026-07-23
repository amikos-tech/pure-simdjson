---
spike: 003
name: pre-copy-capacity-proof
type: standard
validates: "Given an oversized input and stale parser diagnostics, when parsing starts, then capacity rejection occurs before Rust buffer growth/copy and clears stale details"
verdict: VALIDATED
related: []
tags: [rust, capacity, memory, diagnostics, phase-11]
---

# Spike 003: Pre-Copy Capacity Proof

## What This Validates

Prove the exact order required for Phase 11's Rust-owned reusable input arena:

1. clear diagnostic state for the new parse attempt;
2. compare input length with the parser's immutable maximum;
3. reject an oversized input;
4. only for accepted input, calculate padded length, resize, and copy.

The probe is a standalone instrumented model of the relevant `registry::parser_parse` operations. It compares a late capacity gate with the proposed pre-copy gate without changing production Rust code.

## Research

| Approach | Pros | Cons | Status |
|----------|------|------|--------|
| Rely on the C++ parser's maximum capacity | Native parser still rejects | Rust has already resized its arena and copied the input | Rejected |
| Compare after Rust resize/copy | Simple placement near the native call | The configured bound does not prevent the largest wrapper-owned work | Rejected |
| Clear diagnostics, compare length, then calculate padding/resize/copy | Rejects before every wrapper-owned size-dependent mutation and prevents stale error details | Requires the registry to own the authoritative early gate | Chosen |

The official Rust documentation confirms that [`Vec::resize`](https://doc.rust-lang.org/std/vec/struct.Vec.html#method.resize) changes the vector length and may extend it, [`copy_from_slice`](https://doc.rust-lang.org/std/primitive.slice.html#method.copy_from_slice) performs the byte copy, and [`checked_add`](https://doc.rust-lang.org/std/primitive.usize.html#method.checked_add) is the padding-overflow operation. The capacity check must textually precede all three.

## How to Run

```bash
bash .planning/spikes/003-pre-copy-capacity-proof/verify.sh
```

The script compiles and runs four Rust tests, executes the optimized comparison probe, verifies its JSON result, and removes all temporary binaries.

## Cases

| Case | Required observation |
|------|----------------------|
| Exact configured limit | Accepted; one resize and one input copy |
| Limit + 1 | Capacity status; no padding arithmetic, resize, or copy |
| Rejection after stale syntax detail | Message and offset cleared before return |
| 8 MiB rejected input | Late-gate model copies all bytes; pre-copy model copies zero |

## Results

**Verdict: VALIDATED.**

All four Rust tests and the machine-readable invariant verifier passed. Three consecutive result streams had the same SHA-256 digest.

| Observation | Late gate | Pre-copy gate |
|-------------|-----------|---------------|
| Exact 32-byte limit | — | Accepted; 32 bytes copied once |
| Rejected 33-byte input | Would prepare the arena | Zero padding checks, resizes, copies, or arena changes |
| Rejection after stale detail | — | Message and offset cleared |
| Rejected 8,388,609-byte input | 8,388,609 bytes copied | Zero bytes copied |

The ordering is sufficient to turn the configured maximum into a real wrapper-owned work bound. It also shows why the gate should run while the reusable arena is still attached to the parser entry: an early return then needs no buffer restoration path and cannot accidentally discard or alter the arena.

Phase 11 should read the parser's stored capacity and clear native diagnostics before taking the reusable arena. Only an accepted input should proceed to `mem::take`, padding `checked_add`, `Vec::resize`, and `copy_from_slice`.
