Status: SOURCE READY — NOT RELEASED

# Phase 11 ABI 1.2 Artifact Readiness

This record covers local source readiness only. It does not claim that
`v0.1.6` exists at the default bootstrap URL, that hosted release jobs passed,
or that any artifact was published.

## Tested Source Identity

| Property | Observed value |
|---|---|
| Approved intermediate version | `0.1.6` |
| Public ABI | `0x00010002` |
| Audited upstream release | simdjson `v4.6.4` |
| Audited upstream commit | `1bcf71bd85059ab6574ea1159de9298dcc1212c5` |
| Tested source commit | `76303dcac30301f7a7d76cc4bacfe85a4915f3c7` |
| Host | `darwin/arm64` |
| Fresh release library | `/tmp/pure-simdjson-recovery-host-target/release/libpure_simdjson.dylib` |

The upstream commit equality check exited `0`. The tracked submodule remains
the audited official commit; the repository-owned positive-overflow patch is
applied only to the verified build-output copy by `build.rs`.

The exact serial source gate was rerun in full for the `0.1.6` recovery after
the build-only line-ending compatibility fix and the Alpine smoke dependency
fix. All commands below exited `0` at
`76303dcac30301f7a7d76cc4bacfe85a4915f3c7`; the ABI, audited upstream commit,
and patched native behavior are unchanged.

## Packaged Smoke Contract Feedback

The runnable smoke contains no `PURE_SIMDJSON_LIB_PATH` override logic. Local
feedback used the fresh release library explicitly:

| Command | Exit |
|---|---:|
| `cargo build --release --locked` | `0` |
| `PURE_SIMDJSON_LIB_PATH="$LOCAL_LIBRARY" go run ./tests/smoke/go_bootstrap_smoke.go` | `0` |
| `python3 scripts/release/test_release_workflow_contracts.py` | `0` |
| `python3 scripts/release/test_public_bootstrap_validation_contracts.py` | `0` |
| `bash scripts/release/run_alpine_smoke.sh --image-ref "$ALPINE_IMAGE_REF"` | `0` |

The smoke constructs a parser with `WithMaxCapacity(64)` and
`WithMaxDepth(8)`, requires a non-empty active kernel, parses
`18446744073709551616` as `TypeBigInt`, reads the exact copied text through
`GetBigInt`, reads a normal `int64` field, and closes the document and parser.
Contract tests trace `release.yml` through
`scripts/release/run_go_packaged_smoke.sh` to this smoke and trace both public
validation bands through `scripts/release/run_public_bootstrap_smoke.sh` to
the tag-owned copy.

## Exact Serial Source Gate

The following commands ran serially in this order. Every command exited `0`.
The release build completed before any Go runtime gate, and every local Go
runtime command received the same freshly built library through
`PURE_SIMDJSON_LIB_PATH`.

| # | Exact command | Exit | Evidence summary |
|---:|---|---:|---|
| 1 | `make verify-contract` | `0` | Rust unit/integration suites, generated-header diff, 25 ABI-header tests, header rules, and C layout compile passed. |
| 2 | `cargo build --release --locked` | `0` | Fresh host release library built successfully. |
| 3 | `PURE_SIMDJSON_LIB_PATH="$LOCAL_LIBRARY" go test ./... -race -count=1` | `0` | Root package plus bootstrap, CLI, and FFI packages passed under the race detector. |
| 4 | `python3 scripts/release/test_check_bootstrap_abi_state.py` | `0` | 11 bootstrap/ABI policy tests passed. |
| 5 | `python3 scripts/release/test_release_workflow_contracts.py` | `0` | 13 release workflow contract tests passed. |
| 6 | `python3 scripts/release/test_public_bootstrap_validation_contracts.py` | `0` | 11 public validation contract tests passed. |
| 7 | `python3 scripts/release/test_render_release_notes.py` | `0` | 10 release-note renderer tests passed. |
| 8 | `PURE_SIMDJSON_LIB_PATH="$LOCAL_LIBRARY" go test . -run '^TestJSONTestSuiteOracle$' -count=1` | `0` | JSONTestSuite oracle passed (`ok`, 0.453s). |
| 9 | `PHASE11_BENCH_DIR="$(mktemp -d "${TMPDIR:-/tmp}/pure-simdjson-phase11-13-bench.XXXXXX")"` | `0` | Created a new temporary benchmark directory. |
| 10 | `PURE_SIMDJSON_LIB_PATH="$LOCAL_LIBRARY" bash scripts/bench/run_pr_benchmark.sh --no-baseline --out-dir "$PHASE11_BENCH_DIR"` | `0` | Representative Tier 1/2/3 signal captured. |
| 11 | `bash scripts/release/check_readiness.sh` | `0` | Basic, non-strict release readiness passed. |
| 12 | `test "$(git -C third_party/simdjson rev-parse HEAD)" = "1bcf71bd85059ab6574ea1159de9298dcc1212c5"` | `0` | Exact v4.6.4 commit confirmed. |
| 13 | `rg -n 'NewParserPool\(' --glob '*.go' --glob '*.md' --glob '!.planning/**' --glob '!third_party/**' .` | `0` | Every source call handles `(*ParserPool, error)`; docs and examples handle `(pool, err)`. |
| 14 | `! rg -ni 'oversized literal rejected at parse|TestFastMaterializerOversizedLiteralParseRejected|want ErrInvalidJSON or ErrPrecisionLoss|ErrPrecisionLoss, or ErrPanic|native BIGINT classifications report ErrPrecisionLoss|BIGINT roots are unreachable|err_precision_loss\(\).*KIND_HINT_INVALID|kind cannot be classified \(for example, BIGINT\)|canonical precision-loss error surfaces|PRECISION_LOSS.*BIGINT|BIGINT.*precision.loss' element.go element_fuzz_test.go element_scalar_test.go materializer_fastpath_test.go src/native/simdjson_bridge.cpp src/runtime/registry.rs src/lib.rs include/pure_simdjson.h` | `0` | No obsolete BigInt rejection, unclassifiable-kind, or precision-loss contract remained in Go, C++, Rust, or the generated header. |

The platform-correct library resolver selected:

```text
LOCAL_LIBRARY=/tmp/pure-simdjson-recovery-host-target/release/libpure_simdjson.dylib
```

## Correctness and Benchmark Result

- Correctness oracle: `PASS`.
- Benchmark capture directory:
  `/tmp/pure-simdjson-phase11-recovery-bench.3JkgNb`.
- Capture: 60 result rows across 12 unique benchmark cases.
- Bands: Tier 1 full parse on twitter/canada, Tier 2 typed extraction on
  twitter/canada, and Tier 3 selective traversal on twitter.
- No-baseline summary: `bypassed=true`, `regression=false`,
  `flagged_rows=[]`, threshold `5.0%`, `p_max=0.05`.

The bypass is the intended no-baseline source signal: it proves all selected
benchmarks execute and records current measurements, but it makes no
head-versus-baseline performance claim.

## Failed Immutable Attempt

- `v0.1.5` remains an immutable annotated tag on
  `e9f3cea2bbe3b827f8b950c0aacd19640068d05b`.
- Release run `30028224205` passed tag ancestry plus all five platform
  build/native-smoke jobs, then failed the separate pinned Alpine smoke
  because its package list omitted `git`.
- The release publish job was skipped. No `v0.1.5` GitHub Release or public
  artifact was created.
- Recovery advanced to the unused patch version `0.1.6`; the failed tag was
  not moved, replaced, or reused.

## Outstanding Hosted Operator Gate

Default-bootstrap/public proof is still pending and is owned by
`11-14-T1`. Plan 11-14 must:

1. land the artifact-enabling source on `main` through the required squash
   merge;
2. run `bash scripts/release/check_readiness.sh --strict --version 0.1.6`
   from the approved `origin/main`-anchored commit;
3. create and push the annotated tag only after strict readiness succeeds;
4. require tag-driven `release.yml` to build, smoke, sign, and publish all
   five targets; and
5. run the Phase `06.1` public bootstrap validation for the five-target R2
   matrix and the documented three-target GitHub fallback subset.

`release.yml` expects the future tag commit to be anchored on `origin/main`.
Strict readiness was intentionally not run here because this source commit is
not yet that squash-merged commit.

No `v0.1.6` tag was created or pushed. No recovery release workflow was
dispatched, and no artifact was uploaded or published. The real no-override
default-bootstrap smoke has not been claimed.

## Readiness Boundary

The artifact-enabling source is ready for squash merge and the Plan 11-14
operator gate. Phase 11 is not complete, and `v0.1.6` is not a released
artifact.
