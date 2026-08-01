---
spike: 005
name: wildcard-path-semantics
type: standard
validates: "Given the vendored simdjson v4.6.4 DOM API and fixed documents, when element::at_path_with_wildcard is called across wildcard, non-wildcard, scalar-receiver, and malformed paths, then the exact (error_code, result_count, ordering) is pinned and D-02's claimed error surface is either confirmed or refuted"
verdict: PARTIAL
related: [004]
tags: [simdjson, dom, jsonpath, wildcard, error-taxonomy, phase-12]
---

# Spike 005: Wildcard Path Semantics

## What This Validates

**Given** the vendored simdjson v4.6.4 DOM API and a fixed set of documents,
**when** `element::at_path_with_wildcard` is called across wildcard, non-wildcard,
scalar-receiver, and malformed paths,
**then** the exact `(error_code, result_count, ordering)` per case is pinned, and
D-02's claimed error surface is confirmed or refuted against the real implementation.

Phase 12 plans 12-03 and 12-06 delegate `AtPathAll` directly to
`at_path_with_wildcard` and claim decision D-02's error surface
(`ErrInvalidPath` / `ErrElementNotFound` / `ErrIndexOutOfRange` / `ErrWrongType`).
Cross-AI review (`12-REVIEWS.md`, codex) asserted thin delegation cannot produce
that surface. This spike settles it executably.

## Research

No external research required — this is pure upstream behavior against an
already-pinned dependency. Prior spikes established the vendored version
(simdjson v4.6.4, commit `1bcf71bd85059ab6574ea1159de9298dcc1212c5`).

Source read before probing, to shape the case list rather than replace it:

- `dom/element-inl.h:450-459` — scalar receiver falls to `default: return std::vector<element>{}`
- `dom/object-inl.h:175-243` — regime split on `json_path.find("*")`; `else` branch propagates `at_path` errors
- `dom/object-inl.h:154-172` — `process_json_path_of_child_elements` does `if (error) { continue; }`
- `dom/array-inl.h:162-224` — array variant is symmetric

Reading settled the single-level cases. It could not settle the multi-level
recursion (element → object/array → element, with error-swallowing at each hop),
which is what the probe covers.

Deviation from spike 004: that probe looped every `implementation` because minify
is SIMD kernel code. Path resolution lives in `dom/*-inl.h` and is
kernel-independent, so this probe records the active implementation but does not
iterate. `IMPL` is excluded from the golden diff for host portability.

## How to Run

```bash
.planning/spikes/005-wildcard-path-semantics/verify.sh
```

Regenerate the pinned table deliberately (e.g. after an upstream bump):

```bash
.planning/spikes/005-wildcard-path-semantics/verify.sh --update
```

## What to Expect

`==> VERIFIED: 35 cases match pinned semantics, deterministic across 3 runs`

Builds the vendored singleheader in a `mktemp` dir with ASan+UBSan, runs 3×,
requires byte-identical output, then diffs against `expected.txt`. Any semantics
drift fails with a per-case diff.

## Investigation Trail

**Round 1 (27 cases).** Probed no-wildcard delegation, scalar receivers,
exact-suffix wildcards, mid-path wildcards with partial matches, and malformed
grammar. Both codex claims confirmed immediately. The unexpected result was the
`.z.b` / `.z.*` pair: identical missing key, opposite outcomes.

**Round 2 (+8 boundary cases).** The regime split looked like it might be keyed
on *where* the wildcard sits or *what* the receiver is. It is neither — added
cases isolating a missing prefix *before* a later wildcard, `[*]` against
objects, `.*` against arrays, index-on-object, and trailing dots. This produced
two findings round 1 had missed (`[*]`/`.*` aliasing; trailing-dot behavior).

**Drift-detection self-test.** Mutated one golden line and re-ran; the verifier
failed with a readable per-case diff. Restored and re-verified clean.

## Results

**Verdict: PARTIAL.** Ordering and grammar hold. D-02's error surface does not.

### What D-02 gets right

- **Ordering is document order.** `.*` on `{"p":1,"q":2,"r":3}` → `[1,2,3]`.
  Nested and array-of-object wildcards likewise preserve source order.
- **Grammar is consistent between `AtPath` and `AtPathAll`.** All four malformed
  inputs return `INVALID_JSON_POINTER(22)` in *both* columns, so a single
  `ErrInvalidPath` mapping is correct for both APIs.
- **Exact known-rejected strings** (closes the "malformed cases underspecified"
  review finding): `a.b`, `*`, `.a[0`, `""`.

### What D-02 gets wrong

**Upstream selects its error regime by substring-testing the path for `*`, not by
what the document contains.**

| Path | Document | Result |
|---|---|---|
| `.z.b` | `{"a":{"b":1}}` | `NO_SUCH_FIELD(20)` |
| `.z.*` | `{"a":{"b":1}}` | `SUCCESS`, 0 results |
| `.z.*.b` | `{"a":{"b":1}}` | `SUCCESS`, 0 results |

Adding a `*` anywhere — even *after* the segment that fails — converts a hard
error into a silent empty result. Consequences:

- **With a wildcard present, no path error is ever reachable.** Missing keys,
  out-of-range indices, and non-container branches are all silently dropped
  (`wild_mid_partial` → 1 of 2; `wild_mid_hetero` → 1 of 2;
  `wild_arr_of_obj_partial` → 1 of 2).
- **`ErrWrongType` is regime-conditional, not unreachable.** `.a.b` on
  `{"a":[10,20]}` *does* return `INCORRECT_TYPE(17)` — but only without a
  wildcard. Codex's "not generally available" is right; "unavailable" would be wrong.
- **Scalar receivers never error.** Root `42` with `.a` or `.*` →
  `SUCCESS`, 0 results. **Plans 12-03 and 12-06 expect `ErrWrongType` here and
  will fail as written.**

### Surprises

1. **`.*` and `[*]` are interchangeable aliases, neither type-checked.**
   `[*]` on an object returns its values (`.a[*]` on `{"a":{"b":1}}` → `[1]`);
   `.*` on an array returns its elements (`.a.*` on `{"a":[10,20]}` → `[10,20]`).
   Callers will reasonably assume bracket-star means "array" — it does not. Must
   be documented on the Go API.
2. **Trailing dot is an empty-key lookup, not a syntax error.** `.a.` →
   `NO_SUCH_FIELD(20)`. This independently confirms, through the path API, the
   review finding that `AtPointer("/a/")` on `{"a":1}` yields `ErrElementNotFound`
   rather than `ErrWrongType` — 12-06's planned assertion is wrong.
3. **`[0]` on an object is `NO_SUCH_FIELD`, not `INCORRECT_TYPE`** — it degrades
   to a lookup of key `"0"`.
4. **`AtPath` and `AtPathAll` disagree on nearly every wildcard path.** `at_path`
   treats `*` as a literal key, so `.*` → `NO_SUCH_FIELD` there while
   `at_path_with_wildcard` returns every value. Same string, different API,
   different meaning — worth an explicit doc note.

## Implementation Consequence

**Adopt disposition (a): require at least one `*` in `AtPathAll`, rejecting
wildcard-free paths with `ErrInvalidPath`.**

Of the three dispositions cross-AI review proposed, this is the only one that
removes the regime split rather than documenting or papering over it. With a
wildcard mandatory, only the wildcard regime is reachable, so the contract
collapses to one honest sentence:

> `AtPathAll` requires at least one `*`. It returns ordered, document-tied
> matches — possibly empty. The only path error is `ErrInvalidPath`; missing
> keys, out-of-range indices, and non-container branches yield no match rather
> than an error.

- Cost is a `strings.ContainsRune(path, '*')` guard before the FFI call.
- `AtPath` already covers the wildcard-free case *with* full error reporting, so
  nothing is lost.
- Disposition (b) keeps `.z.b` erroring while `.z.*` does not — the inconsistency
  becomes the public contract. Disposition (c) needs native error interception
  and still leaves `AtPathAll` ≠ `AtPath` for identical input.

**D-02 must be amended either way.** Its claimed `ErrElementNotFound`,
`ErrIndexOutOfRange`, and `ErrWrongType` are not reachable through `AtPathAll`
under disposition (a), and are only conditionally reachable under (b).

### Plan changes required

| Plan | Change |
|---|---|
| 12-03 | Add the wildcard-required guard; drop `ErrWrongType`-on-scalar-receiver expectation |
| 12-06 | Fix scalar-receiver test (expects `ErrWrongType`, gets `SUCCESS`+empty); fix `AtPointer("/a/")` test (expects `ErrWrongType`, gets `ErrElementNotFound`); document `.*`/`[*]` aliasing |
| D-02 | Amend error surface to `ErrInvalidPath` only |

`expected.txt` is directly reusable as the test fixture table for both plans.
