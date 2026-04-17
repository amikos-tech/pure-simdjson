# Phase 4: Full Typed Accessor Surface - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in `04-CONTEXT.md` — this log preserves the alternatives considered.

**Date:** 2026-04-17T09:35:49+0300
**Phase:** 04-full-typed-accessor-surface
**Areas discussed:** Type inspection surface, Iterator feel for arrays and objects, Optional helpers while the surface is open, Missing vs null vs wrong-type behavior

---

## Type inspection surface

### How should `Element.Type()` behave?

| Option | Description | Selected |
|--------|-------------|----------|
| 1 | `Type()` returns the full concrete kind set, including `Int64`, `Uint64`, and `Float64`. | ✓ |
| 2 | `Type()` returns broad JSON kinds (`Number`, `String`, `Bool`, `Null`, `Array`, `Object`), and `NumberKind()` is the separate way to distinguish numeric subtypes. | |
| 3 | `Type()` returns broad JSON kinds and there is no `NumberKind()` in `v0.1`. | |

**User's choice:** `Type()` returns the full concrete kind set, including `Int64`, `Uint64`, and `Float64`.
**Notes:** The user preferred exact numeric classification to be visible directly on `Type()`.

### What should `Type()` return after the owning doc is closed or stale?

| Option | Description | Selected |
|--------|-------------|----------|
| 1 | Return an explicit invalid sentinel such as `TypeInvalid`. | ✓ |
| 2 | Panic on misuse. | |
| 3 | Add a separate error-returning probe and keep `Type()` only for live values. | |

**User's choice:** Return an explicit invalid sentinel such as `TypeInvalid`.
**Notes:** The user preferred a total method with explicit invalid state over panic or extra probe API.

---

## Iterator feel for arrays and objects

### What should the iterator control flow look like in Go?

| Option | Description | Selected |
|--------|-------------|----------|
| 1 | `Next() bool`, then `Value()` / `Key()`, plus `Err()` for terminal failure. | ✓ |
| 2 | `Next() (bool, error)`, with `Value()` / `Key()` for the current item. | |
| 3 | `Next() error`, where `nil` means advanced and EOF-style means done. | |

**User's choice:** `Next() bool`, then `Value()` / `Key()`, plus `Err()` for terminal failure.
**Notes:** The user accepted the scanner-style iterator as the best Go fit for the locked ABI.

### What should `ObjectIter.Key()` expose?

| Option | Description | Selected |
|--------|-------------|----------|
| 1 | `Key() string` as a copied Go string. | ✓ |
| 2 | `Key() Element`, so callers inspect the key through the general accessor surface. | |
| 3 | Both: `Key() string` plus a lower-level key view accessor. | |

**User's choice:** `Key() string` as a copied Go string.
**Notes:** The user preferred JSON-object ergonomics over a more uniform-but-heavier generic key surface.

---

## Optional helpers while the surface is open

### Which optional helpers should Phase 4 include?

| Option | Description | Selected |
|--------|-------------|----------|
| 1 | Neither helper in `v0.1`. | |
| 2 | `NumberKind()` only. | |
| 3 | `GetStringField(name)` only. | ✓ |
| 4 | Both helpers. | |

**User's choice:** `GetStringField(name)` only.
**Notes:** The user explicitly probed the value of both helpers. The conclusion was that `GetStringField(name)` may matter for the hot object/string extraction path, while `NumberKind()` is redundant once `Type()` already exposes exact numeric kinds.

---

## Missing vs null vs wrong-type behavior

### How should missing, null, and wrong-type behavior split?

| Option | Description | Selected |
|--------|-------------|----------|
| 1 | Missing key -> `ErrElementNotFound`; present `null` -> valid `Element` with `IsNull()==true`; typed getter on `null` or wrong concrete type -> `ErrWrongType`. | ✓ |
| 2 | Collapse `null` and missing for convenience in field lookups. | |
| 3 | Treat `null` specially with a distinct typed-field error model. | |

**User's choice:** Missing key -> `ErrElementNotFound`; present `null` -> valid `Element` with `IsNull()==true`; typed getter on `null` or wrong concrete type -> `ErrWrongType`.
**Notes:** The user preferred preserving the JSON distinction between absent and explicitly null fields.

---

## the agent's Discretion

- Exact exported enum and invalid-sentinel names for `Type()`
- Exact file split for new iterator/accessor implementation
- Whether `GetStringField(name)` is implemented as composition or a dedicated native fast path

## Deferred Ideas

- `Element.NumberKind()`
- Broader typed field helper families beyond `GetStringField(name)`
