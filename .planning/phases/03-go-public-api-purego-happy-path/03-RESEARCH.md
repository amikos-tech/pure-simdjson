# Phase 3: Go Public API + purego Happy Path - Research

**Date:** 2026-04-15
**Status:** Ready for planning

## Objective

Define the first public Go wrapper over the already-working Rust ABI so a consumer can do:

`NewParser() -> Parse(data) -> doc.Root().GetInt64()`

The plan needs to establish the Go module, deterministic local library loading, ABI/version checks, typed error mapping, lifecycle correctness, `ParserPool`, and the test/doc surface for the Phase 3 happy path without dragging Phase 4 accessors or Phase 5 bootstrap behavior forward.

## Constraints That Are Already Locked

- The public package is `purejson` and must declare `Parser`, `Doc`, `Element`, `Array`, `Object`, and `ParserPool` in Phase 3.
- `Doc.Root()` returns `Element`, not `(Element, error)`.
- `Element` is a small value type tied to a live `Doc`, not a pointer-owning wrapper.
- Local library loading is Phase 3 scope; bootstrap/download remains Phase 5.
- The loader must be deterministic: `PURE_SIMDJSON_LIB_PATH` first, then repo-local build outputs only, never a recursive scan.
- One parser may own at most one live doc at a time; Go must preserve the locked native `ErrParserBusy` invariant instead of hiding it.
- `Parser.Close()` and `Doc.Close()` are idempotent; use-after-close returns `ErrClosed`.
- Finalizers are warning-only safety nets in test builds and silent in production builds.
- `RegisterFunc`/purego registration helpers are allowed; `SyscallN` is not.

## Current Repo Implications

- There is no Go module or Go package yet. Phase 3 has to create the initial Go layout from scratch.
- The Rust side already exposes the exact happy-path ABI Phase 3 needs now: ABI version helpers, implementation-name helpers, parser/doc lifecycle, `doc_root`, `element_type`, and `element_get_int64`.
- The remaining FFI exports (`element_get_uint64`, `element_get_float64`, `element_get_string`, bool/null helpers, iterators, field lookup) are still explicit stubs and must not be treated as Phase 3-ready behavior.
- The existing native smoke harness and workflow from Phase 2 are useful anchors, but they prove only the C ABI, not the Go wrapper contract.
- `src/lib.rs` already defines the authoritative error-code and value-kind constants. The Go layer should mirror those exactly once and translate them, not re-invent meanings at call sites.

## Recommended Technical Direction

### 1. Create a thin Go package with a small internal FFI layer

- Add `go.mod` at the repo root with module path `github.com/amikos-tech/pure-simdjson`.
- Keep the public API in the repo root as `package purejson` so exported docs are generated from the real consumer surface.
- Put the low-level purego mirror types and bound function pointers in an internal package such as `internal/ffi` or an equivalently hidden file set. The public package should not expose raw handles, C enums, or binding details.
- Recommended file split:
  - `go.mod`
  - `errors.go`
  - `library_loading.go`
  - `library_unix.go`
  - `library_windows.go`
  - `parser.go`
  - `doc.go`
  - `element.go`
  - `pool.go`
  - build-tagged finalizer logging files such as `finalizer_testbuild.go` and `finalizer_prod.go`
- `Array` and `Object` should be declared now as wrappers over `Element`, but their traversal behavior should remain intentionally minimal until Phase 4.

### 2. Make library resolution explicit and full-path only

- Resolve a concrete library path before binding any symbols.
- Honor `PURE_SIMDJSON_LIB_PATH` first.
- If the env var is unset, search only these repo-local candidates in order:
  1. `target/release`
  2. `target/debug`
  3. `target/<triple>/release`
  4. `target/<triple>/debug`
- Use platform-specific filenames and keep Windows on `LoadLibrary` with a fully qualified path, never a bare filename.
- If loading fails, return an error that includes the attempted paths. This is part of the Phase 3 developer ergonomics and will materially reduce bring-up time.

### 3. Perform ABI compatibility at `NewParser()`, not lazily later

- Bind and cache the required native symbols once the library path is resolved.
- `NewParser()` should call `pure_simdjson_get_abi_version` before `pure_simdjson_parser_new`.
- Compare the returned version with the Go-side constant derived from `PURE_SIMDJSON_ABI_VERSION` (`0x00010000` right now).
- Mismatch must return `ErrABIVersionMismatch` immediately from `NewParser()`.
- Do not defer the ABI check to `Parse()` or package init, because the roadmap explicitly wants the failure at parser construction time.

### 4. Use a two-layer error model: sentinels for control flow, structured detail for diagnosis

- Export sentinel errors for the requirement set:
  - `ErrInvalidHandle`
  - `ErrClosed`
  - `ErrParserBusy`
  - `ErrNumberOutOfRange`
  - `ErrPrecisionLoss`
  - `ErrCPUUnsupported`
  - `ErrABIVersionMismatch`
  - `ErrInvalidJSON`
  - `ErrElementNotFound`
  - `ErrWrongType`
- Add one public structured error type that carries:
  - native error code
  - byte offset
  - native message
  - wrapped sentinel
- `errors.Is`/`errors.As` must work against the sentinel set even when the richer error is returned.
- Parse and accessor failures should read parser diagnostics when the native code provides them.
- Load failures and ABI mismatch should use the same error type shape rather than a separate ad hoc format.
- `ErrClosed` is a Go wrapper invariant and will often be produced without a native round-trip. That is acceptable as long as it is consistent and deterministic.

### 5. Keep the lifecycle model explicit instead of hiding misuse

- `Parser` should own the native parser handle, a closed flag, and synchronization that prevents concurrent mutation of its Go-side state.
- `Doc` should keep a back-reference to its owning parser so `Doc.Close()` can release the native doc and clear Go-visible busy ownership state.
- `Parser.Parse` should reject a closed parser immediately and otherwise let the native layer remain the source of truth for busy-handle enforcement.
- `Doc.Root()` can stay allocation-free by returning an `Element` value that captures the root view. Accessor methods on `Element` must validate doc liveness and return `ErrClosed` once the doc is closed.
- `Parser.Close()` must not silently free a live doc. The native contract already says that parser free while a doc is live is busy-state misuse. Preserve that behavior through the Go API.
- `ParserPool.Put` must reject parsers that are closed or still own a live doc. Do not auto-close or auto-heal misuse inside the pool.

### 6. Scope finalizers narrowly and behind build tags

- Implement finalizer logging only in test builds or an explicit test-only build tag path so production builds stay silent.
- Finalizers should emit a warning when a parser or doc is garbage-collected without `Close()`.
- They must not become the primary cleanup path and must not mask deterministic lifecycle errors in tests.
- The plan should force verification that warnings appear for leak tests and do not appear in production-mode tests.

### 7. Keep Phase 3 verification honest without turning it into Phase 6

- Phase 3 should add Go-focused smoke verification, but not a full release matrix or bootstrap pipeline.
- The minimum honest proof is:
  - local/current-platform `cargo build --release`
  - `go test ./...`
  - targeted `go test ./... -race`
  - explicit smoke coverage for linux/amd64, darwin/arm64, and windows/amd64 using locally built shim artifacts
- The cross-platform proof can be a narrow manual or workflow-driven smoke path that exercises the Go wrapper only. It does not need artifact publishing, signing, or the full five-platform release process from Phase 6.
- Reuse the deterministic local-loader rules above so tests do not depend on bootstrap/download behavior that does not exist yet.

## Planning Risks To Address Explicitly

### Risk 1: First Go module and package shape

- There is no `go.mod`, no `package purejson`, and no existing internal FFI layer.
- The first plan must force the exact module path, package name, file layout, and symbol-binding direction so execution does not waste time rediscovering structure.

### Risk 2: ABI/version and loader errors becoming opaque

- If the Go layer defers ABI checks or returns generic load failures, Phase 3 will be painful to debug across platforms.
- The plan should require explicit attempted-path reporting and an early ABI handshake in `NewParser()`.

### Risk 3: Root view and closed-doc behavior

- `Doc.Root()` returns `Element` with no error channel, which creates a subtle closed-doc edge.
- The plan must say exactly where `ErrClosed` surfaces for the root/accessor path so the implementation stays predictable.

### Risk 4: Finalizer behavior drifting into production or becoming flaky

- Finalizer warnings need build-tag isolation and a test strategy that does not depend on fragile timing.
- The plan should make the warning path and the production-silent path separate, verifiable artifacts.

### Risk 5: Cross-platform Go smoke proof turning into premature CI scope

- The roadmap wants real proof on linux/amd64, darwin/arm64, and windows/amd64, but full CI productionization belongs later.
- The plan should keep this as a narrow wrapper-smoke verification layer rather than growing release automation now.

## Validation Architecture

Phase 3 should be planned around a fast local loop plus one explicit cross-platform wrapper proof layer:

- **Quick loop:** `cargo build --release && go test ./...`
- **Race loop:** `cargo build --release && go test ./... -race`
- **Current-platform smoke:** run the Go happy-path tests using the repo-local built library resolved through the deterministic loader
- **Cross-platform wrapper proof:** run the same Go smoke path on `linux/amd64`, `darwin/arm64`, and `windows/amd64` against locally built shim artifacts, without introducing bootstrap/download or release publishing

The plan should create or update any missing smoke infrastructure early enough that later tasks can reuse it instead of inventing one-off verification commands.

## Suggested Plan Shape

Phase 3 naturally breaks into three plans:

1. **Go module, loader, and FFI foundation**
   - create `go.mod`
   - add deterministic library loading files
   - bind the happy-path native symbols
   - add ABI version handshake and typed error translation scaffolding

2. **Public API lifecycle and pool behavior**
   - implement `Parser`, `Doc`, `Element`, `Array`, `Object`, `ParserPool`
   - wire `NewParser`, `Parse`, `Close`, `Root`, `GetInt64`
   - enforce idempotent close, `ErrClosed`, `ErrParserBusy`, and pool misuse rules
   - add build-tagged finalizer warning behavior

3. **Docs and verification proof**
   - add Go tests for happy path, close semantics, busy-state behavior, ABI mismatch, structured errors, and leak warnings
   - write `docs/concurrency.md`
   - add exported Go doc comments for every Phase 3 public type/function
   - add the narrow cross-platform smoke proof for the Go wrapper without pulling release automation forward

## Deliverables The Planner Should Force

- A concrete `go.mod` with the fixed module path `github.com/amikos-tech/pure-simdjson`
- Exact loader file names and the deterministic search order from the context
- One canonical Go-side error translation point from native code to sentinels + structured detail
- Explicit semantics for `Doc.Root()` on closed docs and for `Parser.Close()` while a doc is still live
- A `ParserPool` contract that rejects misuse instead of auto-repairing it
- Build-tagged finalizer logging files with tests proving "warnings in test builds, silent in production"
- `docs/concurrency.md` and exported Go doc comments as first-class outputs, not optional cleanup
- Explicit verification commands for current-platform `go test` / `go test -race` and the narrow linux/darwin/windows wrapper smoke path

## Research Conclusion

Phase 3 is the first public Go surface, so the plan cannot be a vague "add bindings" pass. The safest path is to treat it as three explicit deliverables: establish the module and deterministic loader, build the lifecycle-safe public API on top of the already-working ABI, and then prove the wrapper with race tests, docs, and narrow cross-platform smoke coverage. The phase should stay disciplined: only the happy path, only local loading, only the single real accessor, and no premature expansion into Phase 4 accessors or Phase 5 bootstrap logic.
