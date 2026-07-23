# Phase 11: Upstream simdjson Refresh, BigInt, and Diagnostics - Pattern Map

**Mapped:** 2026-07-22
**Scope:** simdjson v4.6.4, additive ABI `0x00010002`, BigInt, parser options, kernel selection, reliable error offsets, bootstrap/release coordination
**Planned targets classified:** 54 source, test, documentation, and release-policy files
**Existing gates mapped for reuse:** 7
**Strong analogs read:** 32

## Locked Implementation Order

Use the repository's existing layers in this order. Each layer should stay a thin translation of the one below it:

```text
Go API
  -> purego binding and ABI gate
  -> public Rust C exports
  -> Rust registry/runtime
  -> C++ bridge
  -> simdjson v4.6.4 singleheader

Rust public exports
  -> cbindgen configuration
  -> generated include/pure_simdjson.h
  -> ABI/header/native smoke tests
  -> five-platform tag-driven release artifacts
```

Do not hand-maintain behavior in the generated header or add a second release path. The source-of-truth flow is Rust declarations -> cbindgen -> committed header, and source state on `main` -> strict readiness -> annotated tag -> release workflow -> public bootstrap validation.

## File Classification

| New/Modified File | Action | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|---|
| `third_party/simdjson` | update gitlink to v4.6.4 | dependency/config | file-I/O/build | current gitlink plus `build.rs` | exact mechanism |
| `build.rs` | verify paths/flags; change only if v4.6.4 requires it | config | file-I/O/build | `build.rs:10-29,41-49` | exact |
| `src/native/simdjson_bridge.h` | modify | native interface | request-response | its current declarations, `src/native/simdjson_bridge.h:83-174` | exact |
| `src/native/simdjson_bridge.cpp` | modify | native service | request-response/transform | current type map, accessors, diagnostics, materializer | exact |
| `src/runtime/mod.rs` | modify | FFI service | request-response | current C++ wrapper functions, `src/runtime/mod.rs:277-480` | exact |
| `src/runtime/registry.rs` | modify | store/service | CRUD/request-response | parser registry and copied-string allocation paths | exact |
| `src/lib.rs` | modify | controller/FFI exports | request-response | current `ffi_wrap` exports, `src/lib.rs:311-696` | exact |
| `cbindgen.toml` | modify | generated-code config | transform | current ABI macro/exclusion list, `cbindgen.toml:7-58` | exact |
| `include/pure_simdjson.h` | regenerate, never hand-edit | generated config/API | transform | `Makefile:3-18` | exact |
| `internal/ffi/types.go` | modify | model/config | transform | current ABI, status, kind, and pinned layout mirrors | exact |
| `internal/ffi/bindings.go` | modify | service | request-response | copied-string binding and mandatory registration | exact |
| `library_loading.go` | modify | provider | request-response | current double-checked active-library load | exact role; staged bind is new |
| `parser_options.go` | create | config/utility | transform | no repository functional-option analog; use immutable config decisions from research | no exact analog |
| `kernel.go` | create | config/provider | event-driven/process-global | `library_loading.go:37-97` and test-only runtime override in `src/runtime/mod.rs:24-27,634-684` | partial |
| `parser.go` | modify | service/model | request-response | current constructor/lifecycle, `parser.go:23-105` | exact |
| `pool.go` | modify | provider/store | request-response | current `sync.Pool` wrapper, `pool.go:10-49` | exact |
| `element.go` | modify | model/service | request-response | strict scalar getters and copied string getter | exact |
| `materializer_fastpath.go` | modify | utility | batch/transform | current frame switch and copied string frame | exact |
| `errors.go` | modify | model/utility | transform | current native-detail wrapping and offset heuristic | exact |
| `purejson.go` | modify | documentation/config | transform | current package contract, `purejson.go:1-14` | exact |
| `internal/bootstrap/version.go` | modify after version choice | config | request-response | current single version constant, `internal/bootstrap/version.go:1-7` | exact |
| `internal/bootstrap/abi_assertion.go` | modify | config/test guard | transform | current two-sided compile-time ABI canary | exact |
| `internal/bootstrap/checksums.go` | update examples only after version choice, if retained | config/documentation | file-I/O | current empty published-checksum override map | exact |
| `scripts/release/check_bootstrap_abi_state.py` | modify | release policy | batch/validation | current ABI-to-minimum-bootstrap policy map | exact |
| `docs/ffi-contract.md` | modify | documentation | transform | current normative ABI contract | exact |
| `docs/concurrency.md` | modify | documentation | request-response | current parser-pool example and lifecycle rules | exact |
| `docs/bootstrap.md` | update versioned examples only after version choice | documentation | request-response | current tagged artifact examples | exact |
| `CHANGELOG.md` | modify | documentation/release config | batch | existing `Changed — ABI` entry at `CHANGELOG.md:40-47` | exact |
| `example_test.go` | modify | test/documentation | request-response | `ExampleParserPool_Get`, `example_test.go:320-352` | exact |
| `tests/rust_shim_bigint.rs` | create | integration test | request-response | `tests/rust_shim_accessors.rs:1-49,98-165` | exact role/data flow |
| `tests/rust_shim_diagnostics.rs` | create | integration test | request-response | `tests/rust_shim_minimal.rs:83-103,197-259` | exact role/data flow |
| `tests/rust_shim_kernel.rs` | create | integration test | event-driven/process-global | `tests/rust_shim_fallback_gate.rs:1-62` | role-match |
| `tests/rust_shim_minimal.rs` | modify | integration test | request-response | its ABI/type/boundary assertions | exact |
| `tests/rust_shim_fast_materializer.rs` | modify | integration test | batch/transform | its preorder frame tests, `tests/rust_shim_fast_materializer.rs:91-170` | exact |
| `tests/abi/check_header.py` | modify | contract test utility | batch/validation | required-symbol/signature rules | exact |
| `tests/abi/test_check_header.py` | modify | test | batch/validation | fixture builder and missing-symbol tests | exact |
| `tests/abi/handle_layout.c` | modify | ABI test | transform | current numeric kind and layout static assertions | exact |
| `tests/smoke/ffi_export_surface.c` | modify | smoke test | request-response | current typedef/export-table/resolve/call-all pattern | exact |
| `tests/smoke/go_bootstrap_smoke.go` | modify | release smoke test | request-response | current real bootstrap parse, `tests/smoke/go_bootstrap_smoke.go:17-44` | exact |
| `internal/ffi/types_test.go` | modify | unit/ABI test | transform | current layout assertions | exact |
| `internal/ffi/bindings_test.go` | modify | unit test | request-response | current copied-string fake binding/free assertion | exact |
| `internal/ffi/bindings_optional_test.go` | modify or copy into a focused ABI-binding test | unit test | request-response | optional/mandatory registration failure tests | role-match |
| `library_loading_test.go` | modify | unit/integration test | request-response | loader caching, ABI mismatch, and constructor signature assertions | exact |
| `bigint_test.go` (recommended new focused file) | create | API test | request-response | `element_scalar_test.go:40-64,205-266,490-513` | exact role/data flow |
| `element_scalar_test.go` | modify old boundary expectations | API test | request-response | its scalar classification/boundary tables | exact |
| `element_fuzz_test.go` | modify walker and accepted-error set | fuzz test | streaming/transform | its type switch, `element_fuzz_test.go:49-124` | exact |
| `materializer_fastpath_test.go` | modify | API test | batch/transform | parity, numeric semantics, ownership, and adversarial frames | exact |
| `parser_options_test.go` | create | API test | request-response | parser constructor/error tests in `parser_test.go:19-75` | role-match |
| `pool_test.go` | modify | API/concurrency test | request-response | current pool lifecycle and concurrent reuse tests | exact |
| `kernel_test.go` | create | API/subprocess test | event-driven/process-global | subprocess isolation in `parser_test.go:340-514` | role-match |
| `parser_test.go` | modify | API test | request-response | current ABI mismatch and diagnostic detail tests | exact |
| `errors_test.go` | modify | unit test | transform | current status-to-sentinel table | exact |
| `scripts/release/test_check_bootstrap_abi_state.py` | modify | release policy test | batch/validation | current stale/mismatch/unknown-policy fixtures | exact |
| `Makefile` | modify only to add new contract rules; keep generation path | config | batch/validation | `generate-header` and `verify-contract`, `Makefile:3-18` | exact |

### Existing Gate Files to Reuse

These are part of the phase path, but current mechanisms already do the right work. Prefer proving they consume the updated header/smoke source over editing them:

| File | Existing behavior to preserve |
|---|---|
| `.github/workflows/release.yml` | tag-only trigger, tag anchored on `origin/main`, five build targets, native smoke, packaged Go smoke, signing, CI publication |
| `.github/workflows/public-bootstrap-validation.yml` | five-platform R2 validation plus selected fallback validation from the published tag source |
| `scripts/release/check_readiness.sh` | strict bootstrap/ABI policy, locked Cargo metadata, and `origin/main` ancestry gate |
| `scripts/release/verify_glibc_floor.sh` | derives the exact expected public export set from the generated header; new public symbols flow through automatically |
| `scripts/release/test_release_workflow_contracts.py` | protects release ordering and artifact behavior; extend only if Phase 11 changes a workflow contract |
| `scripts/release/test_public_bootstrap_validation_contracts.py` | protects published-tag checkout and public validation matrix; extend only if that contract changes |
| `docs/releases.md` | canonical operator runbook; do not invent another publish sequence |

## Pattern Assignments

### Upstream gitlink and native build

**Targets:** `third_party/simdjson`, `build.rs`, `src/native/simdjson_bridge.{h,cpp}`

**Analog:** current singleheader build in `build.rs:10-29,41-49`.

```rust
// build.rs: retain the existing build shape and source ownership.
cc::Build::new()
    .cpp(true)
    .std("c++17")
    .file("third_party/simdjson/singleheader/simdjson.cpp")
    .file("src/native/simdjson_bridge.cpp");
```

Update the gitlink to the locked v4.6.4 commit. First try the existing singleheader paths and C++17 flags unchanged. Adapt only compile errors caused by the pinned upstream API; do not vendor a second copy or introduce a package-manager path.

The C++ bridge is the only layer that should know simdjson's concrete C++ types. Continue using non-throwing `.get(err)` access and the existing exception guards at `src/native/simdjson_bridge.cpp:258-281`.

### Additive ABI propagation

**Targets:** `src/lib.rs`, `internal/ffi/types.go`, `cbindgen.toml`, generated header, ABI tests, bootstrap canary.

**Analogs:**

```rust
// src/lib.rs:13-17
pub const PURE_SIMDJSON_ABI_VERSION: u32 = 0x0001_0001;
```

```toml
# cbindgen.toml:7
after_includes = "#define PURE_SIMDJSON_ABI_VERSION 0x00010001"
```

`internal/bootstrap/abi_assertion.go:1-11` uses two opposite zero-length array expressions as a compile-time equality canary. Keep both directions and update its expected value in the same change as the Go and Rust ABI constants.

Change all three ABI sources together to `0x00010002`. Append BigInt as value kind `9`; never renumber kinds `0..8`. Preserve the exact sizes and offsets of `pure_simdjson_value_view_t` and `psdj_internal_frame_t`; BigInt frame text belongs in the existing `string_ptr/string_len` fields.

Every new public Rust export must follow the existing boundary at `src/lib.rs:187-202,393-415`:

```rust
#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_parser_new(/* out params */) -> pure_simdjson_error_code_t {
    ffi_wrap(|| {
        // validate pointers, call registry/runtime, write outputs only on success
    })
}
```

Then regenerate, do not edit, `include/pure_simdjson.h`:

```make
# Makefile:3-18
generate-header:
	cbindgen --config cbindgen.toml --crate pure_simdjson --output include/pure_simdjson.h

verify-contract:
	# regenerate to a temporary path and diff against the committed header
```

Extend `cbindgen.toml:22-58` for any new internal `psimdjson_*` bridge helpers so only intended `pure_simdjson_*` symbols enter the public header.

### ABI-first purego loading

**Targets:** `library_loading.go`, `internal/ffi/bindings.go`, loader/binding tests.

**Analog:** `library_loading.go:37-97` already centralizes and caches one loaded library. Keep that locking and caching, but split binding into two stages:

1. Open the library and bind only `pure_simdjson_get_abi_version`.
2. Read the reported version and reject anything other than `0x00010002`.
3. Only then bind the full ABI 1.2 mandatory symbol set.
4. Cache the library only after every mandatory binding succeeds.

Do not model Phase 11 symbols with `registerOptionalFunc`. The optional pattern at `internal/ffi/bindings_optional_test.go:9-31` is useful only as a negative example. Mandatory functions use the current fail-closed registration path at `internal/ffi/bindings.go:113-149`.

Required fixture cases:

- library reports 1.1: return `ErrABIMismatch` without probing 1.2-only symbols;
- library reports 1.2 and supplies every mandatory symbol: load succeeds;
- library reports 1.2 but omits one mandatory symbol: load fails deterministically and is not cached.

### BigInt type and copied accessor

**Targets:** C++ type/accessor, Rust runtime/registry/export, Go binding/model, materializer, and their tests.

Use the existing copied-string route as the exact ownership pattern. The public header currently expresses it at `include/pure_simdjson.h:368-390`:

```c
pure_simdjson_error_code_t pure_simdjson_element_get_string(
  const pure_simdjson_value_view_t *view,
  uint8_t **out_ptr,
  size_t *out_len);

pure_simdjson_error_code_t pure_simdjson_bytes_free(uint8_t *ptr, size_t len);
```

The Rust registry already allocates copied scalar bytes and tracks them for `bytes_free` at `src/runtime/registry.rs:793-870`. The Go binding already copies and frees the native allocation at `internal/ffi/bindings.go:319-343`. Add BigInt beside those paths rather than introducing borrowed memory or another allocator.

The Go API should mirror the strict getter shape in `element.go:209-223`:

```go
func (e Element) GetString() (string, error) {
	// validate the document-tied view
	// call the copied FFI accessor
	// runtime.KeepAlive(e.doc)
	// wrap the numeric status; return Go-owned text on success
}
```

Apply the locked classification exactly:

- integer syntax below `math.MinInt64` or above `math.MaxUint64` is `TypeBigInt` / kind `9`;
- values at both boundaries remain `TypeInt64` or `TypeUint64`;
- decimal/exponent syntax is not BigInt;
- `GetBigInt` returns copied decimal spelling and remains valid after `Doc.Close`;
- `GetInt64`, `GetUint64`, and `GetFloat64` on BigInt return `ErrWrongType`;
- root, descendant lookup, array iteration, and object iteration must all carry kind `9`.

Replace the current upstream rejection points, not just the root hint: `src/native/simdjson_bridge.cpp:175-198,702-821`, `src/runtime/registry.rs:641-661,720-742`, and the old rejection assertions in `tests/rust_shim_minimal.rs:503-520`, `tests/rust_shim_fast_materializer.rs:209-223`, `element_scalar_test.go:205-266`, and `materializer_fastpath_test.go:108-123`.

### BigInt in the fast materializer

**Targets:** `src/native/simdjson_bridge.cpp`, `materializer_fastpath.go`, fast-materializer tests.

Use the current string frame without changing the pinned frame layout:

```go
// materializer_fastpath.go:154-159,199-206
case TypeString:
	value, err := copyFrameString(frame)
	// assign the copied Go string into the materialized result
```

The native builder already puts borrowed string spans into `string_ptr/string_len` at `src/native/simdjson_bridge.cpp:504-514`; emit BigInt's decimal span the same way. The Go frame consumer must copy it before the native frame guard is released. Add parity tests for root and nested BigInt and an adversarial kind-9 frame, while retaining the document lock/`runtime.KeepAlive` pattern at `materializer_fastpath.go:15-48`.

### Immutable parser options and homogeneous pools

**Targets:** new `parser_options.go`, `parser.go`, `pool.go`, Rust configured constructor/registry, tests/docs/examples.

There is no repository functional-option analog. Keep the new surface small: an unexported config value, `ParserOption` functions that return a new config or a validation error, and one validation pass inside construction. Options must not retain caller-owned mutable state.

Preserve the lifecycle and ABI checks in `parser.go:31-60`; insert validated config and the ABI 1.2 configured constructor into that sequence. Preserve the parser mutex and one-live-document guard in `parser.go:62-105`.

The current pool is deliberately thin (`pool.go:10-28`):

```go
type ParserPool struct {
	pool sync.Pool
}

func NewParserPool() *ParserPool { return &ParserPool{} }

func (p *ParserPool) Get() (*Parser, error) {
	if parser, ok := p.pool.Get().(*Parser); ok {
		return parser, nil
	}
	return NewParser()
}
```

Keep `sync.Pool`, but store the validated immutable config/factory on `ParserPool`. Change the constructor to the locked `NewParserPool(opts ...ParserOption) (*ParserPool, error)` and make every miss call `NewParser` with that same config. `Put` keeps its current nil/closed/busy rejection at `pool.go:30-49`; additionally reject a parser whose immutable configuration does not match the pool. Update all call sites, including `example_test.go:320-352`, `docs/concurrency.md:16-48`, and `pool_test.go`.

Capacity belongs before allocation/copy in `src/runtime/registry.rs:453-460`, where input is currently padded and copied:

```rust
let required_len = input_len.checked_add(padding).ok_or(/* status */)?;
if parser_entry.reusable_input.len() < required_len {
    parser_entry.reusable_input.resize(required_len, 0);
}
// copy input only after the configured capacity check has passed
```

Depth belongs in the native traversal/materialization recursion at `src/native/simdjson_bridge.cpp:397-534`. Store both values on the parser entry created at `src/runtime/registry.rs:212-242`; never mutate them after construction.

### Process-global kernel selection

**Targets:** new `kernel.go`, Rust public/runtime/native kernel helpers, subprocess tests.

The closest locking pattern is the repository's process-global active-library state (`library_loading.go:37-97`) and the isolated test override state (`src/runtime/mod.rs:24-27,634-684`). Use one package/runtime state machine, not per-parser mutable kernel fields:

```text
unlocked -> SetKernel may validate/set -> first NewParser or NewParserPool -> locked forever
```

The pool constructor itself locks selection, even before its first `Get`. `Kernel` is a read-only diagnostic. Concurrent `SetKernel` versus parser/pool construction must have one linearized winner and no data race. Test each irreversible scenario in a subprocess; do not depend on test order or attempt to unlock production state.

### Error offsets: known zero versus unknown

**Targets:** native diagnostic state/getters, Rust registry/export, Go bindings and `Error`, diagnostics tests.

The current Go heuristic loses a valid zero offset. It is concentrated in `errors.go:108-114,233-242`:

```go
func hasOffset(offset uint64) bool {
	return offset != 0 && offset != math.MaxUint64
}
```

Replace inference with explicit state propagated from C++ to Go. Store `(offset, hasOffset)` on the native parser error buffer, expose both through the additive ABI, cache both in `Error`, and provide the locked separate `Offset` and `HasOffset` accessors. Unknown may retain the wire sentinel for backward diagnostics, but callers must branch on the explicit boolean.

Use only an upstream-proven pointer/location that is within the copied input range. Pointer subtraction without range proof, a guessed token index, and converting an unknown sentinel to zero are forbidden. Preserve the current clear-after-success behavior demonstrated by `tests/rust_shim_minimal.rs:238-259`.

Test all five cases independently:

- known non-zero offset;
- known offset zero;
- unknown offset;
- success after failure clears stale text, offset, and known flag;
- stale/invalid handles still return the existing handle status rather than fabricated diagnostics.

### Go error/status mapping

**Targets:** `errors.go`, `errors_test.go`, parse/materializer option failures.

Keep one numeric-status mapping table. `errors_test.go:10-93` is the existing table-driven pattern. Extend it for any newly exposed capacity/depth/kernel statuses required by the locked research, while preserving `errors.Is` behavior and treating native text as advisory. Do not branch on message strings.

### Rust shim test layout

**Targets:** three new Rust test binaries and existing minimal/materializer tests.

Copy the parse/root/cleanup helpers from `tests/rust_shim_accessors.rs:1-49` and the copied ownership/free assertions at `tests/rust_shim_accessors.rs:98-138`. Separate test binaries are useful here because kernel selection is process-global and diagnostic state belongs to a parser.

Use `tests/rust_shim_fast_materializer.rs:91-102` for borrowed-frame safety:

```rust
let rc = unsafe { psdj_internal_materialize_build(view, &mut frames, &mut frame_count) };
assert_eq!(rc, PURE_SIMDJSON_OK);
// Copy the borrowed span while its document remains live.
unsafe { slice::from_raw_parts(frames, frame_count).to_vec() }
```

The new tests should establish the low-level contract before Go wrapper tests depend on it:

- `rust_shim_bigint.rs`: classification boundaries, copied text/free, wrong-type getters, root and descendants;
- `rust_shim_diagnostics.rs`: known non-zero/zero/unknown and stale-state clearing;
- `rust_shim_kernel.rs`: available/selected kernel reporting and isolated lock scenarios.

### ABI and native smoke tests

**Targets:** `tests/abi/*`, `tests/smoke/ffi_export_surface.c`, `internal/ffi/types_test.go`.

Follow the current three-level contract:

1. `tests/abi/check_header.py:18-20,48-75,197-267` parses the committed header and checks version, required symbols, ownership, and signatures.
2. `tests/abi/test_check_header.py:10-107,125-213` builds synthetic headers and proves missing/unexpected symbols fail.
3. `tests/abi/handle_layout.c:9-85` pins numeric values, sizes, and offsets in a real C compile.

Append kind `9` assertions and all ABI 1.2 mandatory symbols/signatures. Add a BigInt copy/free rule parallel to `string-copy-ownership`. Do not weaken the existing exact layout checks.

`tests/smoke/ffi_export_surface.c` already has function typedefs (`81-126`), one export table (`128-162`), one resolver (`224-295`), and a call-all assertion (`315-324`). Add every mandatory ABI 1.2 symbol to all four places and actually exercise BigInt, configured construction, kernel diagnostics, and explicit offset-known reporting. This same C source is compiled by every release platform, so no platform-specific duplicate smoke is needed.

### Go API, fuzz, and public bootstrap tests

**Targets:** focused BigInt/options/kernel tests plus existing scalar/fuzz/materializer/pool/parser tests and public smoke.

Reuse `mustNewParser` from `parser_test.go:19-27` and the copied-lifetime pattern in `element_scalar_test.go:490-513`. Extend `element_fuzz_test.go:49-124` with `TypeBigInt -> GetBigInt`, and remove `ErrPrecisionLoss` as an accepted parse result for valid oversized integer syntax.

The public bootstrap smoke at `tests/smoke/go_bootstrap_smoke.go:17-44` is intentionally tiny. Extend it just enough to prove that the downloaded artifact exposes ABI 1.2 behavior (at minimum one BigInt/configured-parser path); leave artifact download and platform selection to the existing release scripts/workflows.

### Bootstrap ABI policy and release source state

**Targets:** bootstrap version/canary, policy checker/tests, changelog, versioned docs.

`scripts/release/check_bootstrap_abi_state.py:72-104` is the canonical source-state check: it compares Go and Rust ABI values, looks up the ABI in a minimum-bootstrap-version policy map, and checks the requested release version. Add `0x00010002` to that map and update tests for accepted 1.2, rejected 1.1, Go/Rust mismatch, unknown ABI, stale bootstrap version, and requested-version mismatch.

Do not guess the release version. Once the operator chooses it, update together:

- `internal/bootstrap/version.go`;
- versioned examples/comments in `internal/bootstrap/checksums.go` and `docs/bootstrap.md`, if kept;
- the matching `CHANGELOG.md` section, including the ABI bump and rebuild notice.

Keep the source checksum override map empty unless the existing runbook explicitly calls for committed overrides; published `SHA256SUMS` remains the runtime source.

### Tag-driven release path

**Targets:** primarily verification of existing workflows and runbook; source edits only where tests expose a missing contract.

The workflow already encodes the required release path:

```yaml
# .github/workflows/release.yml:3-6,47-51
on:
  push:
    tags:
      - "v*"

- name: Verify tag commit is anchored on origin/main
  run: |
    git fetch --no-tags origin main:refs/remotes/origin/main
    git merge-base --is-ancestor "$GITHUB_SHA" "origin/main"
```

It already builds the five locked platforms at `.github/workflows/release.yml:72-85,178-195,298-307`, runs the native smoke at `158-163,278-283,412-417`, and runs packaged Go bootstrap smoke at `579-584`. `scripts/release/verify_glibc_floor.sh:15-39,71-99` derives the public export allowlist from the generated header, so adding a parallel hard-coded export list would be redundant.

The pre-tag gate remains:

```bash
# scripts/release/check_readiness.sh:92-95
python3 scripts/release/check_bootstrap_abi_state.py --version "$version"
cargo metadata --format-version 1 --locked >/dev/null
git fetch origin main --depth=1
git merge-base --is-ancestor HEAD origin/main
```

After CI publishes, `.github/workflows/public-bootstrap-validation.yml:53-122` validates the public artifact matrix using the published tag source. Phase execution may prepare and verify source state, but must stop for the operator before creating a tag or publishing. Source changes reach `main` through a squash merge, matching project instructions.

## Shared Patterns

### Ownership

**Source:** `internal/ffi/bindings.go:319-343`, `src/runtime/registry.rs:793-870`

**Apply to:** BigInt public accessor and any materializer copy-out.

Native allocates -> Go copies -> Go calls the existing `bytes_free`. Never retain a C++/Rust pointer after the call or after document close.

### FFI safety

**Source:** `src/lib.rs:187-202`, `src/native/simdjson_bridge.cpp:258-281`

**Apply to:** every new public export/native helper.

Rust public functions go through `ffi_wrap`; C++ uses non-throwing simdjson access plus the established exception-to-status boundary. Outputs are written only on success unless a size probe explicitly permits otherwise.

### Document lifecycle and synchronization

**Source:** `parser.go:62-105`, `materializer_fastpath.go:15-48`, `pool.go:30-49`

**Apply to:** getters, materialization, configured parsers, and pools.

Validate live handles under the existing parser/document lock, keep Go owners alive across purego calls, and never let pooling weaken the one-live-document invariant.

### Layout compatibility

**Source:** `tests/abi/handle_layout.c:60-85`, `internal/ffi/types_test.go:8-47`

**Apply to:** BigInt kind and configured parser additions.

Append enum values and add functions; do not resize or reinterpret existing public structs. Reuse the materializer frame's string span for BigInt text.

### Fail-closed ABI loading

**Source:** `library_loading.go:37-97`, `internal/ffi/bindings.go:53-149`

**Apply to:** all bootstrap-loaded artifacts.

Read ABI minimally, require exact 1.2, bind every 1.2 symbol, then cache. A missing mandatory symbol is a load failure, not an optional feature downgrade.

### Release automation

**Source:** `.github/workflows/release.yml`, `.github/workflows/public-bootstrap-validation.yml`, `docs/releases.md`

**Apply to:** final Phase 11 source state and artifact proof.

CI is the sole publisher. Do not manually upload artifacts or create an alternative workflow. Preserve the five-platform matrix and require public-bootstrap evidence after publication.

## No Analog Found

| File/Concern | Role | Data Flow | Closest Partial Pattern | Planner Guidance |
|---|---|---|---|---|
| `parser_options.go` | config/utility | transform | constructor validation in `parser.go` | implement one immutable config value and the smallest option surface from locked research |
| production `kernel.go` state machine | provider/config | process-global | active library lock plus test-only runtime override | isolate irreversible tests in subprocesses; no reset API |
| ABI-first minimal bind followed by full bind | provider/service | request-response | current monolithic `ffi.Bind` plus loader ABI check | split the binding sequence; never probe 1.2-only functions from a 1.1 library |
| known-offset boolean across ABI | diagnostic model | request-response | current offset sentinel getter | add explicit state end to end; do not reinterpret zero |

## Planner Guardrails

- v4.6.4 and ABI `0x00010002` are locked; do not reopen them.
- BigInt is only out-of-range integer syntax and is copied text; do not use `math/big.Int` implicitly or widen other numeric getters.
- Capacity must be checked before input allocation/copy.
- Capacity and depth are immutable per parser; pools are configuration-homogeneous.
- Kernel choice is process-global and locks on the first parser **or pool** construction.
- Reject ABI 1.1 before looking up ABI 1.2-only symbols.
- Preserve existing wire layouts and numeric values; append only.
- Prefer the current header generator, export audit, native smoke, packaged smoke, and public validation instead of new duplicate mechanisms.
- The release version remains an operator decision. No plan may assume a particular next version.
- Do not create a tag or publish without the explicit operator checkpoint.

## Metadata

**Analog search scope:** root Go API, `internal/{ffi,bootstrap}`, `src/{runtime,native}`, `tests/{abi,smoke}`, Rust shim tests, release scripts/workflows, docs, build/generation config

**Current upstream gitlink inspected:** `third_party/simdjson` is a gitlink; Phase 11 replaces it with the locked v4.6.4 pin

**Pattern extraction date:** 2026-07-22
