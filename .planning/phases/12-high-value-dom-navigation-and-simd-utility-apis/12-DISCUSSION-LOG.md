# Phase 12: High-value DOM navigation and SIMD utility APIs - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-30
**Phase:** 12-high-value-dom-navigation-and-simd-utility-apis
**Areas discussed:** Navigation error taxonomy, Indexed array access semantics, Minify aliasing behavior, ValidateUTF8 return shape, AtPathAll empty-match behavior

Advisor mode was active (USER-PROFILE.md present, calibration tier: full_maturity / thorough-evaluator). Each area below was researched by an independent `gsd-advisor-researcher` agent, which returned a comparison table and rationale before the user picked.

---

## Navigation error taxonomy

| Option | Description | Selected |
|--------|-------------|----------|
| Two new sentinels (ErrInvalidPath, ErrIndexOutOfRange; reuse ErrElementNotFound for missing) | Matches upstream's 3-way error split 1:1 | ✓ |
| Three new sentinels mirroring upstream names exactly | Perfect naming parity, but duplicates ErrElementNotFound's job | |
| Reuse only (ErrElementNotFound + ErrWrongType) | Simplest, but fails the roadmap's 3-category success criterion | |
| One catch-all ErrInvalidPath for both invalid-syntax and out-of-bounds | Fewer sentinels, still conflates two categories | |

**User's choice:** Two new sentinels — `ErrInvalidPath` and `ErrIndexOutOfRange`, reusing `ErrElementNotFound` for missing and `ErrWrongType` for type mismatches.
**Notes:** Matches upstream's own `NO_SUCH_FIELD`/`INVALID_JSON_POINTER`/`INDEX_OUT_OF_BOUNDS` split exactly; `ErrIndexOutOfRange` is shared with the indexed-array-access decision below.

---

## Indexed array access semantics

| Option | Description | Selected |
|--------|-------------|----------|
| At(int) + Len/LenErr dual-method, new ErrIndexOutOfRange | int matches Go idiom; Len/LenErr slots into existing Type/TypeErr, IsNull/IsNullErr convention | ✓ |
| At(uint64) + Size() direct return | Exact C-type parity with upstream size_t, but forces casts at call sites | |
| Reuse ErrElementNotFound for out-of-range | Smallest diff, but conflates two upstream-distinct failures | |
| Dual-method on At too (panic-safe At + AtErr) | Surface consistency, but a zero-value Element on failure launders the real error | |

**User's choice:** `Array.At(index int) (Element, error)` with new `ErrIndexOutOfRange`; `Array.Len() int` / `Array.LenErr() (int, error)` dual-method.
**Notes:** No panic-safe twin for `At` — mirrors `Object.GetField`'s existing precedent of no dual-method for Element-returning accessors. Negative/Python-style indexing was considered and rejected (upstream has no such concept).

---

## Minify aliasing behavior

| Option | Description | Selected |
|--------|-------------|----------|
| Dual API: Minify(data) ([]byte, error) + MinifyInto(dst, src) (int, error), dst may alias src | Keeps simple default; MinifyInto satisfies the roadmap's explicit aliasing/overlap requirement | ✓ |
| Allocate-only: Minify(data) ([]byte, error) | Simplest, matches Parse/GetString precedent, but can't exercise aliasing publicly | |
| Buffer-supplied only: Minify(dst, src) (int, error) | Mirrors upstream exactly, but breaks the safe-by-default precedent | |
| bytes.Buffer style (json.Compact-like) | Idiomatic vs stdlib, but can't alias src's backing array | |

**User's choice:** Dual API — `Minify(data []byte) ([]byte, error)` plus `MinifyInto(dst, src []byte) (int, error)`.
**Notes:** `MinifyInto` is not gated by Phase 14's zero-copy benchmark rule — it's a stateless buffer transform with no Doc/Parser lifetime coupling, unlike the borrowed-view work Phase 14 gates.

---

## ValidateUTF8 return shape

| Option | Description | Selected |
|--------|-------------|----------|
| (bool, error) | Consistent with every other activeLibrary()-calling function (NewParser, SetKernel) | ✓ |
| Bare bool matching upstream 1:1 | Simplest, but silently collapses a real load/ABI failure into false | |
| Dual-method (ValidateUTF8/ValidateUTF8Err) | Surface consistency, but wrong analogy — no handle/lifecycle to go stale | |

**User's choice:** `func ValidateUTF8(data []byte) (bool, error)`.
**Notes:** ValidateUTF8 is the first standalone entry point that can trigger first-time native library resolution without a prior `NewParser` call — the error return surfaces that real failure mode.

---

## AtPathAll empty-match behavior

| Option | Description | Selected |
|--------|-------------|----------|
| Empty slice, nil error | Matches upstream (empty vector, no error) and Go idiom (map lookups, filepath.Glob) | ✓ |
| ErrElementNotFound on zero matches | Treats empty match set as a missing-element error | |

**User's choice:** `([]Element{}, nil)` on zero matches; errors only for malformed wildcard syntax or traversal type mismatches.
**Notes:** Follow-up question asked after the main four areas, since this specific case wasn't part of the original vote options.

---

## Claude's Discretion

- Exact FFI status-code numeric values for the two new sentinels (additive to the existing block).
- Internal decomposition of the `AtPath` dot/index parser (Go-side pre-parse vs. full delegation to upstream `at_path`).
- Naming for `Object`'s size accessor (`Size()/SizeErr()` vs. matching `Array.Len()/LenErr()` naming) — left open, should stay internally consistent between the two types.
- Internal file organization and test decomposition across Go, Rust, C++, and FFI layers.

## Deferred Ideas

None — discussion stayed within phase scope.
