# Phase 11: Upstream simdjson refresh, exact big integers, and production diagnostics - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-22
**Evidence follow-up:** 2026-07-23
**Phase:** 11-upstream-simdjson-refresh-bigint-and-diagnostics
**Areas discussed:** BigInt public contract, truthful error locations, parser controls and lifecycle, ABI and old-library compatibility, validated spike resolutions

---

## BigInt Public Contract

| Option | Description | Selected |
|--------|-------------|----------|
| Upstream-strict tagged kind | `GetBigInt` accepts only `TypeBigInt`; other numeric getters return `ErrWrongType`. | ✓ |
| Target-conversion semantics | Keep `GetBigInt` strict, but return conversion-specific range/precision errors from other numeric getters. | |
| Width-agnostic integer text | Let `GetBigInt` return text for int64, uint64, and BigInt kinds. | |
| Compatibility-first precision barrier | Keep `GetBigInt` strict and return `ErrPrecisionLoss` from every other numeric getter for BigInt. | |

**User's choice:** Upstream-strict tagged kind.

**Notes:** The user added “without breaking existing contracts.” Existing kind numbers, signatures, layouts, and behavior for already-accepted values remain unchanged; the new kind is appended. Current BigInt-related accessor comments are not treated as observable behavior because oversized integers currently fail before an `Element` exists.

---

## Truthful Error Locations

| Option | Description | Selected |
|--------|-------------|----------|
| Add `HasOffset() bool` | Preserve `Offset() uint64`, distinguish a known byte-zero location, and use only upstream-proven offsets. | ✓ |
| Add `ByteOffset() (uint64, bool)` | Preserve legacy `Offset()` and add a parallel comma-ok accessor. | |
| Add a Go fallback scanner | Try to locate more failures with a secondary parser after simdjson fails. | |
| Change `Offset()` | Replace it with `Offset() (uint64, bool)`, breaking existing callers. | |

**User's choice:** Preserve `Offset()` and add `HasOffset()`.

**Notes:** A real byte-zero failure is `(Offset()==0, HasOffset()==true)`; unknown is `(Offset()==0, HasOffset()==false)`. No secondary parser may manufacture an offset.

---

## Parser Controls and Lifecycle

| Option | Description | Selected |
|--------|-------------|----------|
| Immutable functional options | Configure immutable per-parser limits at construction; keep kernel control package-global. | ✓ |
| Additive `ParserConfig` constructors | Preserve exact constructor signatures and add parallel configured constructors. | |
| Idle-only setters | Permit limit changes only when the parser owns no live document. | |
| Per-parse limits | Apply capacity/depth policy independently to each parse call. | |
| Per-parser kernel pin | Allow different parsers in one process to force different kernels. | |

**User's choice:** Immutable functional options.

**Notes:** Kernel selection is process-global and locks after the first parser or pool. Capacity/depth are immutable per parser, zero means current defaults, invalid values fail, and the capacity check happens before the Rust-owned copy. Pools retain one normalized configuration.

### Pool constructor follow-up

| Option | Description | Selected |
|--------|-------------|----------|
| Add `NewParserPoolWithOptions` | Keep `NewParserPool()` exactly unchanged and add a validated configured constructor. | |
| Variadic constructor, same return | Use `NewParserPool(opts ...ParserOption) *ParserPool` and defer invalid-option errors until `Get()`. | |
| Variadic constructor returning error | Use `NewParserPool(opts ...ParserOption) (*ParserPool, error)` and validate immediately. | ✓ |

**User's choice:** Variadic pool constructor returning an error.

**Notes:** This is an explicit, user-approved source-breaking exception to the general compatibility preference. Planning must not preserve the old pool return shape by silently selecting the additive constructor alternative.

---

## ABI and Old-Library Compatibility

| Option | Description | Selected |
|--------|-------------|----------|
| ABI 1.2 with legacy mode | Accept ABI 1.1 for baseline parsing and return `ErrNotImplemented` for Phase 11 features. | |
| ABI 1.2 strict minimum | Reject ABI 1.1 and guarantee the complete Phase 11 surface for every parser. | ✓ |
| Keep ABI 1.1 and probe symbols | Leave the version unchanged while discovering capabilities symbol by symbol. | |
| ABI 2.0 hard boundary | Treat the additive work as a full incompatible ABI generation. | |

**User's choice:** ABI `0x00010002` as a strict native minimum.

**Notes:** There is no legacy capability mode. Bootstrap pin, Go/Rust/header constants, source canary, readiness policy, and five-platform artifacts move together. An ABI 1.2 artifact missing a mandatory Phase 11 symbol fails rather than degrading.

---

## Validated Spike Follow-Up

After approving Spikes 001, 002, and 003, the user asked to fold all validated findings into the context and decision log. The follow-up resolves implementation choices that were previously left to research or the agent's discretion.

### Error-location replay

| Option | Description | Selected |
|--------|-------------|----------|
| `raw_json()` replay only | Small first pass that preserves trailing-content location, but missed `[1,]`, a missing object key, and root token `x`. | |
| Recursive On-Demand replay only | Validates every key/container/scalar, but reduced the useful trailing-content result to byte zero. | |
| Two-fresh-parser hybrid | Use `raw_json()`/`at_end()` first; recursively traverse with a fresh parser only when the first pass reports valid. | ✓ |
| Secondary parser/scanner | Estimate more locations with non-simdjson logic. | |

**User's choice:** Fold the validated hybrid into the locked context.

**Notes:** Exact v4.6.4 repeatability proved unknown for empty input, invalid UTF-8, and unclosed string; known offsets are 3 (`[1,]`), 8 (trailing content), 16 (missing object key), 0 (`x`), 15 (extra closing bracket), and 9 (mismatched container). Only a successful `current_location()` pointer inside `[input,input+len)` becomes known. The hybrid uses two instances of simdjson's own upstream API, not a secondary parser or estimator.

### ABI binding order

| Option | Description | Selected |
|--------|-------------|----------|
| Bind every required symbol first | One pass, but collapses valid ABI 1.1 and corrupt ABI 1.2 into the same missing-symbol failure. | |
| Infer ABI from available symbols | Avoids a call-first stage, but symbol presence is not a reliable version contract. | |
| Probe ABI first, then bind the required surface | Resolve/call only the ABI getter, classify compatibility, then require every compatible ABI symbol before caching. | ✓ |

**User's choice:** Fold ABI-first staged binding into the locked context.

**Notes:** The purego spike proved ABI 1.1 reaches `ErrABIVersionMismatch` with zero ABI 1.2 lookups, complete ABI 1.2 succeeds, and incomplete ABI 1.2 fails as corrupt/load failure naming the omitted symbol.

### Capacity-gate order

| Option | Description | Selected |
|--------|-------------|----------|
| Rely on the C++ capacity limit | Native rejects eventually, after Rust has already prepared and copied the arena. | |
| Check after Rust resize/copy | Keeps the check near native parsing but does not bound wrapper-owned work. | |
| Clear diagnostics and check while the arena remains attached | Reject before `mem::take`, padding arithmetic, resize, or copy; leave the reusable arena unchanged. | ✓ |

**User's choice:** Fold the pre-copy, pre-detach capacity gate into the locked context.

**Notes:** The Rust spike accepted the exact boundary, rejected limit+1 with zero padding checks/resizes/copies, cleared stale details, preserved arena bytes/length/capacity, and avoided an 8,388,609-byte rejected copy.

---

## the agent's Discretion

- Exact functional-option names and internal representation.
- Duplicate-option handling, provided it is deterministic, documented, and tested.
- Exact existing or new typed error used for locked kernel selection and invalid options.
- Internal helper naming/decomposition and human-readable wording around the locked hybrid replay; replay order, pointer proof, and the selected corpus are no longer discretionary.
- Internal file and test decomposition.

## Deferred Ideas

None — discussion stayed within Phase 11 scope.
