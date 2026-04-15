# Phase 1: FFI Contract Design - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-14T10:55:17Z
**Phase:** 1-FFI Contract Design
**Areas discussed:** Parser reuse semantics, Error and diagnostics surface, Value-handle shape, Read-side ABI shape

---

## Parser reuse semantics

| Option | Description | Selected |
|--------|-------------|----------|
| Auto-invalidate on re-parse | A second `Parse` replaces the old `Doc` implicitly and invalidates older views | |
| Hard-fail until prior `Doc` closes | Parser returns `ERR_PARSER_BUSY` while a live `Doc` exists; release is explicit | ✓ |
| Fresh parser/doc per parse | Avoid reuse entirely and treat parsers as single-shot objects | |

**User's choice:** Fast-path `recommended`, which accepted the suggested option: hard-fail with `ERR_PARSER_BUSY` until the live `Doc` is closed.
**Notes:** This keeps parser/doc lifetime explicit and avoids hidden invalidation at the contract layer.

---

## Error and diagnostics surface

| Option | Description | Selected |
|--------|-------------|----------|
| Numeric error codes only | Minimal ABI; no helper diagnostics beyond the code itself | |
| Numeric codes plus a small diagnostics surface | Stable `int32` codes remain primary, with minimal helpers for ABI version, implementation name, and parse context | ✓ |
| Rich error objects/messages | Dynamic strings or richer objects become part of the core ABI surface | |

**User's choice:** Fast-path `recommended`, which accepted the suggested option: stable numeric codes plus a small diagnostics surface.
**Notes:** Diagnostics stay advisory. Primary control flow remains portable numeric error codes.

---

## Value-handle shape

| Option | Description | Selected |
|--------|-------------|----------|
| Everything is an opaque handle | Uniform model, but extra allocations and more FFI lifecycle burden | |
| Opaque `Parser`/`Doc` plus lightweight value views | Long-lived owners stay opaque; `Element`/`Array`/`Object` stay cheap and view-like | ✓ |
| Copy out every value eagerly | Simplifies lifetime rules but defeats the parse-heavy performance goal | |

**User's choice:** Fast-path `recommended`, which accepted the suggested option: opaque owners plus lightweight value views.
**Notes:** This matches the performance target without reintroducing unsafe lifetime behavior.

---

## Read-side ABI shape

| Option | Description | Selected |
|--------|-------------|----------|
| Caller-managed scratch buffers and node indexes | Lowest-level ABI, but awkward for the Go wrapper and easy to misuse | |
| Copy-out strings plus stateful iterators | Strings are returned as owned copies; arrays/objects advance through explicit iterator state | ✓ |
| High-level batch materialization helpers | Fewer calls, but pushes policy and allocation into the ABI too early | |

**User's choice:** Fast-path `recommended`, which accepted the suggested option: copy-out strings plus stateful iterators.
**Notes:** Zero-copy strings stay deferred to `v0.2`; no callbacks or tree materialization are introduced.

---

## the agent's Discretion

- Precise error-code numbering
- Final function and iterator type names
- Exact diagnostic-helper transport shape, as long as it stays advisory

## Deferred Ideas

- Zero-copy string views tied to `Doc` lifetime
- Richer diagnostics / introspection beyond the minimal helper surface
