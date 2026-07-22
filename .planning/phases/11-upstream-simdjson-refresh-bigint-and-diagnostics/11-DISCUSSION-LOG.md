# Phase 11: Upstream simdjson refresh, exact big integers, and production diagnostics - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-22
**Phase:** 11-upstream-simdjson-refresh-bigint-and-diagnostics
**Areas discussed:** BigInt public contract, truthful error locations, parser controls and lifecycle, ABI and old-library compatibility

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

## the agent's Discretion

- Exact functional-option names and internal representation.
- Duplicate-option handling, provided it is deterministic, documented, and tested.
- Exact existing or new typed error used for locked kernel selection and invalid options.
- Exact native technique for retrieving upstream-proven parse locations and the human-readable unknown-location wording.
- Internal file and test decomposition.

## Deferred Ideas

None — discussion stayed within Phase 11 scope.
