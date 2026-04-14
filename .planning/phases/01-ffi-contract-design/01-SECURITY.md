---
phase: 01
slug: ffi-contract-design
status: verified
threats_open: 0
asvs_level: 1
created: 2026-04-14
---

# Phase 01 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| local toolchain -> generated ABI artifacts | `cbindgen` output becomes the committed public C contract | Generated header names, types, comments, and layout surface |
| Rust ABI source -> future Go bindings | Drift between Rust exports and the committed header would propagate ABI breakage downstream | FFI signatures, handle/view layouts, diagnostics, and ownership rules |
| Go / purego callers -> C ABI | Incorrect low-level ABI shape would turn into runtime corruption across supported targets | Integer returns, pointer out-params, handles, views, and iterator state |
| contract doc -> downstream implementers | Ambiguous policy text would create unsafe implementation assumptions in later phases | Lifecycle, panic/exception policy, copy semantics, and compatibility rules |

---

## Threat Register

| Threat ID | Category | Component | Disposition | Mitigation | Status |
|-----------|----------|-----------|-------------|------------|--------|
| T-01-01 | T | `cbindgen` pipeline | mitigate | `Makefile` `verify-contract` regenerates the header with `cbindgen` and diffs it against `include/pure_simdjson.h`, preventing hand-edited drift from surviving | closed |
| T-01-02 | D | Wave 0 bootstrap | mitigate | Fresh `make verify-contract` passed this audit, proving both `cargo check` and the `cbindgen`-driven verification path are hard execution gates | closed |
| T-02-01 | T | public function signatures | mitigate | `tests/abi/check_header.py` enforces `int32-outparams` and `no-mixed-float-int` against `include/pure_simdjson.h` during `verify-contract` | closed |
| T-02-02 | T | handle and value model | mitigate | `src/lib.rs` and `include/pure_simdjson.h` define packed handles plus doc-tied view/iterator structs, and `tests/abi/handle_layout.c` locks their sizes and offsets at compile time | closed |
| T-02-03 | D | parser lifecycle surface | mitigate | `src/lib.rs`, `include/pure_simdjson.h`, and `docs/ffi-contract.md` explicitly fix parser-busy and `doc_free` lifecycle semantics; `make verify-docs` confirms the policy text remains present | closed |
| T-03-01 | R | panic/exception policy docs | mitigate | `docs/ffi-contract.md` explicitly requires `ffi_fn!`, `catch_unwind`, `.get(err)`, and separately states that release `panic = "abort"` remains fatal | closed |
| T-03-02 | T | header/doc drift | mitigate | `make verify-contract` and `make verify-docs` jointly gate regenerated-header parity, ABI linting, layout assertions, and required contract clauses | closed |
| T-03-03 | D | missing verification coverage | mitigate | `tests/abi/README.md` maps the static checks to `FFI-01` through `FFI-08` and `DOC-02`, and the mapped commands passed in this audit | closed |

*Status: open · closed*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

No accepted risks.

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-04-14 | 7 | 7 | 0 | Codex |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-04-14
