# Phase 8: Low-overhead DOM traversal ABI and specialized Go any materializer - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in `08-CONTEXT.md`; this log preserves the alternatives considered.

**Date:** 2026-04-23
**Phase:** 08-low-overhead-dom-traversal-abi-and-specialized-go-any-materializer
**Areas discussed:** Traversal ABI Shape, String And Key Handoff, Go any Builder Semantics, Exposure And Proof Bar

---

## Traversal ABI Shape

### Core fast path

| Option | Description | Selected |
| --- | --- | --- |
| Bulk traversal frames | Native walks a subtree once and returns compact pre-order frames; Go builds `any`. | yes |
| Iterator v2 | Keep pull iteration, but add lower-overhead size/type/value calls. | no |
| Planner decides | Research the fastest maintainable shape before locking. | no |

**User's choice:** Bulk traversal frames.
**Notes:** User also provided exploratory research favoring tape-like exposure: Rust-owned arena, Go read-only slices over tape/string data, Go-side walking, explicit lifetime ownership, and strings copied only when materialized. This was captured as a research vector, not a hard mandate.

### ABI scope

| Option | Description | Selected |
| --- | --- | --- |
| Internal ABI only | Used by Go wrapper/benchmarks, not promised as public C API yet. | yes |
| Public minor ABI extension | Add documented public symbols. | no |
| Prototype behind tests first | Document later if it wins. | no |

**User's choice:** Internal ABI only.
**Notes:** Keeps Phase 8 focused on proving the path before committing public API shape.

### Traversal granularity

| Option | Description | Selected |
| --- | --- | --- |
| Whole-document materialization only | Optimize only full-document Tier 1. | no |
| Any subtree materialization | Materialize from an `Element`. | no |
| Both, planner chooses first slice | Design envelope includes both, but planning can sequence delivery. | yes |

**User's choice:** Both, planner chooses first implementation slice.
**Notes:** This preserves subtree capability without forcing one oversized implementation plan.

### Compatibility stance

| Option | Description | Selected |
| --- | --- | --- |
| Add parallel fast path | Preserve current accessor ABI untouched and add a parallel fast path. | yes |
| Replace internals only | Replace some current internal iterator implementation while keeping public Go API stable. | no |
| Allow internal layout break | Permit breakage if header/ABI versioning handles it. | no |

**User's choice:** Add a parallel fast path.
**Notes:** Public DOM accessor behavior remains stable.

---

## String And Key Handoff

### Object key handling

| Option | Description | Selected |
| --- | --- | --- |
| Key slices in internal view | Go copies only when building `map[string]any`. | yes |
| Batched key strings | Keep current key-as-string behavior but batch extraction. | no |
| Planner decides after measuring | Let fixture data decide. | no |

**User's choice:** Key slices inside the internal frame/tape view.
**Notes:** Optimizes the key-heavy object path where current iteration copies each key through `ElementGetString`.

### String value copy rule

| Option | Description | Selected |
| --- | --- | --- |
| Copy on materialization | Internal traversal may view Rust-owned bytes; public result owns Go strings. | yes |
| Copy every string during traversal | Simpler but copies values that may not be needed. | no |
| Keep existing string getter path | Preserve per-string allocation/free and optimize elsewhere first. | no |

**User's choice:** Initially chose copy every string during traversal, then accepted the recommendation to copy only when materializing.
**Notes:** Borrowed Rust memory is acceptable internally; the hard boundary is that borrowed memory must not escape into public Go values.

### Borrowed memory escape

| Option | Description | Selected |
| --- | --- | --- |
| No public escape | Borrowed slices are internal only; user-visible values own Go memory. | yes |
| Unsafe public API | Expose borrowed memory behind advanced API. | no |
| Defer to v0.2 | Do not decide in Phase 8. | no |

**User's choice:** No borrowed Rust memory escapes to public Go callers.
**Notes:** User asked whether internal borrowed Rust memory makes sense. Clarification: yes, for internal ABI paths; the risk is public lifetime proof.

### Lifetime safety

| Option | Description | Selected |
| --- | --- | --- |
| Ownership and KeepAlive discipline | Explicit `Doc`/materializer ownership plus `runtime.KeepAlive`, `Close`, and finalizer discipline. | yes |
| Debug live-view tracking | Panic/fail if Rust is freed while views are active. | no |
| Both if cheap | Normal ownership plus debug tracking if inexpensive. | no |

**User's choice:** Ownership and `KeepAlive` discipline.
**Notes:** Debug tracking remains discretionary if planning finds a cheap implementation.

---

## Go any Builder Semantics

### Numeric parity

| Option | Description | Selected |
| --- | --- | --- |
| Match current accessors | Preserve int64/uint64/float64 distinctions and precision/range errors. | yes |
| Match `encoding/json` | Collapse to default float64 behavior for closer comparator shape. | no |
| Planner decides | Check oracle expectations first. | no |

**User's choice:** Match current accessor semantics.
**Notes:** This keeps public correctness behavior ahead of benchmark convenience.

### Duplicate object keys

| Option | Description | Selected |
| --- | --- | --- |
| Last duplicate wins | Full `map[string]any` keeps the last value because Go map assignment collapses duplicates. | yes |
| Preserve `GetField` first-match | Force full map materialization to mirror current direct lookup. | no |
| Document natural behavior | Add tests for whichever behavior falls out. | no |

**User's choice:** Last duplicate wins.
**Notes:** User asked what is lost with `GetField`. Clarification: `GetField` remains first-match DOM lookup; full map materialization cannot represent duplicate entries and uses ordinary Go map assignment semantics.

### Container sizing

| Option | Description | Selected |
| --- | --- | --- |
| Exact or near-exact preallocation | Use traversal metadata instead of fixed `8` capacities. | yes |
| Conservative hints | Keep implementation simpler. | no |
| Planner decides per type | Choose based on available metadata. | no |

**User's choice:** Exact or near-exact preallocation.
**Notes:** Directly addresses current fixed-capacity slices/maps in the recursive benchmark materializer.

### Error behavior

| Option | Description | Selected |
| --- | --- | --- |
| Fail fast with existing typed errors | Wrong type/range/precision/invalid-handle map to current public errors. | yes |
| Best effort | Build placeholders for unsupported values. | no |
| Panic internally | Panic in internal fast path and recover at outer API boundary. | no |

**User's choice:** Fail fast with existing typed errors.
**Notes:** Keeps correctness and error matching testable.

---

## Exposure And Proof Bar

### Materializer exposure

| Option | Description | Selected |
| --- | --- | --- |
| Internal first | Benchmark/wrapper fast path only; no public API until measured. | yes |
| Public convenience API now | Add `Element.Interface()` / `Doc.Interface()` immediately. | no |
| Experimental public API | Add unstable public naming. | no |

**User's choice:** Internal benchmark/wrapper fast path first.
**Notes:** Public API remains deferred until the implementation proves itself.

### Proof before Phase 9

| Option | Description | Selected |
| --- | --- | --- |
| Correctness plus benchmark delta | Oracle/parity tests plus Tier 1 diagnostics showing improvement. | yes |
| Benchmarks only | Existing tests are enough. | no |
| Correctness only | Phase 9 can decide benchmark story later. | no |

**User's choice:** Correctness plus benchmark delta.
**Notes:** Phase 9 should consume measured evidence, not design intent.

### Benchmark closeout target

| Option | Description | Selected |
| --- | --- | --- |
| Improvement over Phase 7 | Show materialize-only and full Tier 1 improvement without requiring a headline ratio. | yes |
| Beat `encoding/json + any` | Require Tier 1 faster than stdlib before closeout. | no |
| No Tier 2/3 regression only | Require only that other tiers do not regress. | no |

**User's choice:** Show materialize-only and full Tier 1 improvement over the Phase 7 baseline.
**Notes:** Beating `encoding/json + any` is not required for Phase 8 completion.

### Documentation scope

| Option | Description | Selected |
| --- | --- | --- |
| Internal docs and notes | Public README/result repositioning waits for Phase 9. | yes |
| README immediately | Update README if numbers improve. | no |
| Design doc only | No public benchmark docs. | no |

**User's choice:** Internal docs and benchmark notes only.
**Notes:** Keeps public benchmark repositioning in Phase 9.

---

## the agent's Discretion

- Exact internal traversal/frame/tape type names and layouts.
- Whether the first implementation slice targets whole-document materialization, subtree materialization, or both.
- Whether to add debug-only borrowed-view tracking if cheap.
- Exact benchmark command grouping and artifact naming.

## Deferred Ideas

- Public `Element.Interface()` / `Doc.Interface()` convenience APIs.
- Public borrowed-memory or unsafe APIs.
- JSONPointer/path lookup helpers unless required for the materializer.
- Public benchmark repositioning and release decisions.
