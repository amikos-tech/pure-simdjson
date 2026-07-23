---
spike: 001
name: v464-error-location-replay
type: standard
validates: "Given malformed inputs and pinned simdjson v4.6.4, when an upstream-only On-Demand replay runs after DOM failure, then only stable in-range pointers become known offsets"
verdict: VALIDATED
related: []
tags: [simdjson, diagnostics, offsets, phase-11]
---

# Spike 001: v4.6.4 Error-Location Replay

## What This Validates

Given malformed inputs rejected by the DOM parser, when the same bytes are replayed through simdjson v4.6.4 On-Demand and fully consumed, then the wrapper can distinguish:

- failures where `iterate()` never produced a valid document and location must remain unknown;
- recoverable traversal/trailing-content failures with an upstream pointer inside the original input;
- end or out-of-range pointers that must remain unknown.

No secondary parser, scanner, message parsing, or estimated byte index is used.

## Research

| Approach | Tool/Library | Pros | Cons | Status |
|----------|--------------|------|------|--------|
| DOM parse result only | simdjson DOM | Same path as production parsing | A failed DOM parse does not expose a document-level `current_location()` API | Rejected as insufficient |
| `raw_json()` replay | simdjson v4.6.4 | Small and consumes the root span | Uses an internal skip path and does not fully validate every malformed object, array, or root token | Rejected after experiment |
| Recursive On-Demand replay | simdjson v4.6.4 | Exercises every key, container, and scalar through public typed accessors | More code; on the trailing-content fixture it reports a less useful container error at byte zero | Useful fallback |
| Hybrid: raw root/trailing probe, then recursive fallback | simdjson v4.6.4 | Preserves the documented trailing-content location while recursively validating cases `raw_json()` skips | Requires two isolated On-Demand parser instances on the failure path | Chosen |
| Secondary JSON scanner | Any other parser/scanner | Could report more locations | Would fabricate a location under Phase 11 decision D-08 | Prohibited |

The pinned [simdjson v4.6.4 error-location documentation](https://github.com/simdjson/simdjson/blob/v4.6.4/doc/basics.md#current-location-in-document) says `current_location()` requires a valid On-Demand document, remains usable for several traversal errors, returns `OUT_OF_BOUNDS` at the end, and is unavailable when `iterate()` itself fails. The probe therefore calls `current_location()` only after a successful `iterate()`, and accepts a location only after an explicit in-range pointer check.

## How to Run

If the v4.6.4 tag object is not present in the submodule object store, fetch it without changing the gitlink:

```bash
git -C third_party/simdjson fetch --no-tags origin \
  refs/tags/v4.6.4:refs/tags/v4.6.4
```

Then compile the official v4.6.4 single-header source in a temporary directory and verify the observations:

```bash
bash .planning/spikes/001-v464-error-location-replay/verify.sh
```

## What to Expect

The command prints one TSV row per malformed input, followed by a JSON safety summary. Known offsets must have `pointer_relation=in_bounds` and `0 <= offset < bytes`. An `iterate_failed` row must have `location_status=not_queried`.

## Investigation Trail

1. Confirmed the tag resolves to commit `1bcf71bd85059ab6574ea1159de9298dcc1212c5` without checking out or modifying the submodule.
2. Initially used `document.raw_json()` followed by `at_end()`. The first run exposed a false-negative trap: `[1,]`, a missing object key, and an unexpected root token were reported as valid replay results even though DOM rejected all three.
3. Added a recursive public-API walk that consumes every object key, array element, and scalar through the appropriate typed accessor. It removed those false negatives, but the trailing-content fixture produced a less useful error location at byte zero.
4. Added a hybrid candidate: use a fresh `raw_json()` replay only to capture an explicit upstream trailing-content/error result; if it reports the document as valid, run a second fresh recursive replay. The final probe retains all three modes for comparison.
5. Added both the fixed Phase 11 corpus and extra structural/root-token cases to find whether stable byte-zero or nonzero locations actually exist.

## Results

**Verdict: VALIDATED.**

Three consecutive runs against the exact v4.6.4 commit produced byte-for-byte identical output and no safety violations.

| Case | Hybrid result | Known offset |
|------|---------------|--------------|
| Empty input | `iterate()` failed | Unknown |
| Invalid UTF-8 | `iterate()` failed | Unknown |
| Unclosed string | `iterate()` failed | Unknown |
| `[1,]` | Recursive `TAPE_ERROR` | 3 |
| `{"a":1} trailing` | Explicit trailing content | 8 |
| Missing object key | Recursive `TAPE_ERROR` | 16 |
| Unexpected root token | Recursive `TAPE_ERROR` | 0 |
| Extra closing bracket | Explicit trailing content | 15 |
| Mismatched container | Upstream `TAPE_ERROR` | 9 |

The known-zero representation is therefore not merely a transport seam: v4.6.4 supplies a real in-bounds byte-zero location for the unexpected-root-token fixture.

The important implementation finding is that neither candidate alone is sufficient:

- `raw_json()` missed three DOM failures.
- Recursive traversal caught every DOM failure in the corpus, but gave the trailing-content fixture a less useful container error at byte zero.
- A two-parser hybrid preserved the documented trailing-content offset and used recursive traversal only when the raw-root pass found no error/trailing content.

Phase 11 should characterize and implement this hybrid failure-only replay, while retaining unknown for every `iterate()` failure. The production path must still gate every pointer with the same in-range check.
