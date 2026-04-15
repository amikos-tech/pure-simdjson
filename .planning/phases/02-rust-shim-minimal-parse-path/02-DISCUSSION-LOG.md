# Phase 2: Rust Shim + Minimal Parse Path - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-15
**Phase:** 02-rust-shim-minimal-parse-path
**Areas discussed:** Surface depth, Safety foundation, Diagnostics and CPU gate, Second smoke platform

---

## Surface depth

| Option | Description | Selected |
|--------|-------------|----------|
| True minimum | Only the literal-`42` path is real in Phase 2: `parser_new -> parser_parse -> doc_root -> element_get_int64` | |
| Minimum path + diagnostics spine | Keep the phase narrow, but also make `element_type` and parser last-error helpers real for bring-up/debugging | ✓ |
| Most/all SHIM-06 exports real now | Implement most or all remaining accessors/iterators in Phase 2 instead of leaving them stubbed | |

**User's choice:** Accepted recommended option: `Minimum path + diagnostics spine`
**Notes:** The phase should remain narrow, but not so narrow that native build/parse failures are opaque during bring-up.

---

## Safety foundation

| Option | Description | Selected |
|--------|-------------|----------|
| Real safety core now, custom registry | Implement the locked generation-checked parser/doc model now using a small custom registry mapped directly to packed handles | ✓ |
| Real safety core now, `slotmap`-backed | Implement the real safety model now but rely on a helper crate for generational storage | |
| Thin temporary parse path now, real safety later | Use a minimal temporary shim first and retrofit the full safety model in a later phase | |

**User's choice:** Accepted recommended option: `Real safety core now, custom registry`
**Notes:** Phase 2 should prove the actual contract semantics rather than a temporary approximation that would need replacement in Phase 3.

---

## Diagnostics and CPU gate

| Option | Description | Selected |
|--------|-------------|----------|
| Kernel name + hard gate only | Only make implementation-name reporting and fallback rejection real | |
| Kernel name + thin parse diagnostics + hidden test-only bypass | Add selected-kernel reporting plus parser last-error helpers; keep the fallback bypass internal to tests/CI | ✓ |
| Full parser-scoped diagnostics now | Build a broader diagnostics subsystem across the native surface in Phase 2 | |
| Public kernel override/bypass now | Expose user-visible override/bypass controls in Phase 2 | |

**User's choice:** Accepted recommended option: `Kernel name + thin parse diagnostics + hidden test-only bypass`
**Notes:** Diagnostics remain advisory. The no-silent-fallback policy stays public; any bypass exists only to exercise the path in tests/CI.

---

## Second smoke platform

| Option | Description | Selected |
|--------|-------------|----------|
| `windows/amd64 (MSVC)` | Spend the second smoke slot burning down the explicit Windows/MSVC native build risk early | ✓ |
| `darwin/arm64` | Use the second smoke slot on Apple Silicon instead of Windows | |
| `darwin/amd64` | Use the second smoke slot on Intel macOS as the easiest non-Linux expansion | |

**User's choice:** Accepted recommended option: `windows/amd64 (MSVC)`
**Notes:** The repo and research both identify the Windows/MSVC + `cc` build path as the best early risk burn-down target for Phase 2.

---

## the agent's Discretion

- Exact internal registry implementation, as long as the packed handle ABI and contract semantics remain unchanged.
- Exact parser-diagnostics storage/copy-out mechanics, as long as they stay advisory and useful during bring-up.
- Exact hidden bypass wiring for fallback-kernel test coverage.

## Deferred Ideas

- Full typed accessor and iterator surface in the native shim.
- Public kernel override or broader diagnostics controls.
- Broader multi-platform smoke/release coverage and signing/distribution work.
