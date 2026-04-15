# Phase 3: Go Public API + purego Happy Path - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-15T20:05:42Z
**Phase:** 3-Go Public API + purego Happy Path
**Areas discussed:** Public API shape, Local library loading contract, Error and diagnostics surface, Lifecycle and pool ergonomics

---

## Public API shape

### Root exposure

| Option | Description | Selected |
|--------|-------------|----------|
| `Root() Element` | Keep the happy path as `doc.Root().GetInt64()` and surface stale/closed errors at accessor time. | ✓ |
| `Root() (Element, error)` | Make root resolution explicit with an extra error return. | |
| `RootElement() Element` | Same shape as option 1 with a more verbose name. | |

**User's choice:** `Root() Element`
**Notes:** The happy path should stay minimal and chainable.

### Public `Element` representation

| Option | Description | Selected |
|--------|-------------|----------|
| `Element` as a small value type | Cheap to copy, matches the native view model, and supports chained accessors naturally. | ✓ |
| `*Element` pointer wrapper | Add pointer semantics and possible `nil` checks. | |
| Interface abstraction | Hide representation behind an interface. | |

**User's choice:** `Element` as a small value type
**Notes:** Avoid extra allocation and indirection for a view-like type.

### Phase 3 surface breadth

| Option | Description | Selected |
|--------|-------------|----------|
| Expose `Parser`, `Doc`, `Element`, `Array`, `Object`, and `ParserPool` now | Lock the public v0.1 shape early, even if only the happy-path methods are wired in Phase 3. | ✓ |
| Expose only `Parser`, `Doc`, `Element`, and `ParserPool` in Phase 3 | Delay `Array` and `Object` as public types until Phase 4. | |
| Keep future types internal until the full DOM surface is ready | Avoid a partial public API in Phase 3. | |

**User's choice:** Expose the full type skeleton now
**Notes:** Phase 4 should add behavior rather than introduce core public types.

---

## Local library loading contract

### Loader policy

| Option | Description | Selected |
|--------|-------------|----------|
| `PURE_SIMDJSON_LIB_PATH` first, then local build outputs, then fail with attempted paths | Deterministic override and good repo-local development ergonomics. | ✓ |
| Require an explicit path only | Keep the loader minimal and fully manual. | |
| Hardcode one repo-local path | Minimal code with a brittle default. | |

**User's choice:** Env override first, then deterministic repo-local discovery
**Notes:** Phase 3 should stay local-only; bootstrap/download remains a later phase.

### Repo-local search order

| Option | Description | Selected |
|--------|-------------|----------|
| `target/release`, then `target/debug`, then fail | Simple native-only discovery. | |
| `target/release`, `target/debug`, then `target/<triple>/{release,debug}` | Cover both native and explicit-target builds without requiring env setup. | ✓ |
| Broad recursive scan under `target/` | Automatically search all candidate outputs. | |

**User's choice:** `target/release`, `target/debug`, then `target/<triple>/{release,debug}`
**Notes:** Discovery should stay deterministic and report attempted paths on failure.

---

## Error and diagnostics surface

### Error wrapping strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Sentinel errors plus richer wrapped errors only for ABI/load failures | Keep detail mostly for startup failures. | |
| Sentinel errors plus richer wrapped errors for every failure when detail exists | Preserve debug detail while still supporting idiomatic sentinel matching. | ✓ |
| Sentinel errors only | Drop native detail from returned errors. | |

**User's choice:** Sentinel errors plus richer wrapped errors for all failures
**Notes:** The native diagnostics already exist and should not be discarded at the Go boundary.

### Public detail shape

| Option | Description | Selected |
|--------|-------------|----------|
| One public structured error type carrying code, offset, message, and wrapped sentinel | Stable and inspectable detail for tests and debugging. | ✓ |
| Keep richer detail internal | Smaller public surface, but harder to inspect. | |
| Public structured type only for load/ABI failures | Mixed error models depending on failure source. | |

**User's choice:** One public structured error type
**Notes:** The package should keep one consistent error model across load, parse, and accessor failures.

---

## Lifecycle and pool ergonomics

| Option | Description | Selected |
|--------|-------------|----------|
| Strict API: idempotent `Close()`, `ErrClosed` after close, `ParserPool.Put` rejects live-doc parsers, test-build finalizer warnings | Deterministic misuse handling that matches the roadmap safety model. | ✓ |
| Minimal API: document misuse but do not enforce much beyond idempotent close | Smaller implementation, weaker safety surface. | |
| Auto-healing API: pool silently repairs parser/doc misuse | More forgiving, but hides lifecycle bugs. | |

**User's choice:** Strict API
**Notes:** Lifecycle mistakes should surface explicitly rather than be papered over by the pool or finalizers.

## the agent's Discretion

- Exact root-package file split for the Go implementation.
- Exact naming of the exported error-code helper type behind the structured error.
- Exact `ParserPool` method signatures, so long as misuse is explicit and deterministic.
- Exact wording and logging mechanism for test-build finalizer warnings.

## Deferred Ideas

None.
