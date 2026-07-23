---
spike: 002
name: abi-first-staged-binding
type: standard
validates: "Given ABI 1.1, complete ABI 1.2, and incomplete ABI 1.2 libraries, when the loader probes ABI before mandatory symbols, then mismatch and corruption remain distinguishable"
verdict: VALIDATED
related: []
tags: [purego, abi, loader, phase-11]
---

# Spike 002: ABI-First Staged Binding

## What This Validates

Given three tiny synthetic libraries, prove that the Phase 11 loader can preserve three distinct outcomes:

- an older ABI is a version mismatch;
- an ABI 1.2 library missing a required ABI 1.2 symbol is corrupt;
- a complete ABI 1.2 library loads successfully.

The experiment compares the current bind-everything-first order with an ABI-first staged order. It does not change the production loader or native library.

## Research

| Approach | Pros | Cons | Status |
|----------|------|------|--------|
| Resolve every required symbol, then call the ABI function | One binding pass | An older valid library and a corrupt current library both fail as a missing symbol | Rejected |
| Infer ABI from the symbols that happen to exist | Avoids calling native code first | Symbol sets are not a version contract and can be incomplete | Rejected |
| Resolve and call only `pure_simdjson_get_abi_version`, compare it, then resolve the matching required surface | Preserves version mismatch and corruption as separate error classes | Requires a small probe binding stage | Chosen |

The pinned [purego `Dlsym` documentation](https://pkg.go.dev/github.com/ebitengine/purego#Dlsym) exposes lookup as an explicit operation, while [`RegisterFunc`](https://pkg.go.dev/github.com/ebitengine/purego#RegisterFunc) binds a resolved address to a Go function. That supports a narrow ABI probe before any ABI 1.2 symbol is touched.

## How to Run

```bash
bash .planning/spikes/002-abi-first-staged-binding/verify.sh
```

The script builds all three libraries in a temporary directory, runs the probe with the repository's pinned purego dependency, verifies the result rows, and removes the temporary build products.

## Fixtures

| Fixture | Reported ABI | Phase 11 symbols |
|---------|--------------|------------------|
| `abi11` | `0x00010001` | None |
| `abi12_complete` | `0x00010002` | All five planned mandatory symbols |
| `abi12_missing` | `0x00010002` | Omits `pure_simdjson_element_get_bigint` |

## Results

**Verdict: VALIDATED.**

Three consecutive verifier runs produced the same SHA-256 digest.

| Fixture | Bind-everything-first | ABI-first staged |
|---------|-----------------------|------------------|
| ABI 1.1 | `missing_symbol` before reading the ABI | `abi_mismatch`; zero ABI 1.2 lookups |
| Complete ABI 1.2 | `ok` | `ok`; all five mandatory symbols resolved |
| Incomplete ABI 1.2 | `missing_symbol` | `corrupt_abi12`; omitted symbol named |

The naive order collapses a valid old library and a corrupt current library into the same missing-symbol class. The staged order reads the ABI safely through the real pinned purego API, rejects ABI 1.1 without touching any ABI 1.2 symbol, and checks the full mandatory surface only after confirming ABI 1.2.

Phase 11 should split loading into a minimal ABI-probe binding and a required-surface binding. The ABI function must remain the only symbol required before the version comparison.
