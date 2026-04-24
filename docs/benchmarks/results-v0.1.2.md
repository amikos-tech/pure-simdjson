# Benchmark Results v0.1.2

This snapshot records the Phase 9 linux/amd64 benchmark evidence after the Phase 8 low-overhead materializer work. It is sourced from committed raw `go test -bench` output and the machine-readable claim gate summary.

## Status

- Claim mode: `tier1_headline`
- Tier 1 headline allowed: `true`
- Tier 2 headline allowed: `true`
- Tier 3 headline allowed: `true`
- Claim gate errors: none

Tier 1 headline wording is allowed for this linux/amd64 snapshot only. Headline numbers come from linux/amd64; other platforms may differ.

## Target and Raw Evidence

- Target: `linux/amd64`
- CPU: `AMD EPYC 7763 64-Core Processor`
- Go toolchain: `go version go1.24.13 linux/amd64`
- Rust toolchain: `rustc 1.89.0 (29483883e 2025-08-04)`
- Snapshot commit: `a47c561fd14ac1c580a38ba705ae7edab2debd1d`
- Captured at: `2026-04-24T12:53:27Z`

Durable evidence files:

- [`phase9.bench.txt`](../../testdata/benchmark-results/v0.1.2/phase9.bench.txt)
- [`coldwarm.bench.txt`](../../testdata/benchmark-results/v0.1.2/coldwarm.bench.txt)
- [`tier1-diagnostics.bench.txt`](../../testdata/benchmark-results/v0.1.2/tier1-diagnostics.bench.txt)
- [`phase9.benchstat.txt`](../../testdata/benchmark-results/v0.1.2/phase9.benchstat.txt)
- [`coldwarm.benchstat.txt`](../../testdata/benchmark-results/v0.1.2/coldwarm.benchstat.txt)
- [`tier1-diagnostics.benchstat.txt`](../../testdata/benchmark-results/v0.1.2/tier1-diagnostics.benchstat.txt)
- [`tier1-vs-stdlib.benchstat.txt`](../../testdata/benchmark-results/v0.1.2/tier1-vs-stdlib.benchstat.txt)
- [`tier2-vs-stdlib.benchstat.txt`](../../testdata/benchmark-results/v0.1.2/tier2-vs-stdlib.benchstat.txt)
- [`tier3-vs-stdlib.benchstat.txt`](../../testdata/benchmark-results/v0.1.2/tier3-vs-stdlib.benchstat.txt)
- [`metadata.json`](../../testdata/benchmark-results/v0.1.2/metadata.json)
- [`summary.json`](../../testdata/benchmark-results/v0.1.2/summary.json)

## Claim Gate Summary

The claim gate compares `v0.1.2` against the committed linux/amd64 baseline in [`testdata/benchmark-results/v0.1.1-linux-amd64`](../../testdata/benchmark-results/v0.1.1-linux-amd64/), not the older darwin/arm64 historical snapshot.

| Tier | Fixture | pure ns/op | stdlib ns/op | Ratio | Allowed |
| --- | --- | ---: | ---: | ---: | --- |
| Tier 1 | `twitter.json` | `2,044,151` | `6,443,140` | `3.15x` | yes |
| Tier 1 | `citm_catalog.json` | `5,113,116` | `17,338,004` | `3.39x` | yes |
| Tier 1 | `canada.json` | `14,159,130` | `34,934,366` | `2.47x` | yes |
| Tier 2 | `twitter.json` | `368,603` | `5,038,955` | `13.67x` | yes |
| Tier 2 | `citm_catalog.json` | `1,231,068` | `17,918,864` | `14.56x` | yes |
| Tier 2 | `canada.json` | `3,133,064` | `39,124,268` | `12.49x` | yes |
| Tier 3 | `twitter.json` | `313,001` | `4,999,739` | `15.97x` | yes |
| Tier 3 | `citm_catalog.json` | `1,175,722` | `17,844,443` | `15.18x` | yes |

## Tier 1: Full Parse + Full any Materialization

Tier 1 compares full parse plus full generic Go tree materialization. Ratios are versus `encoding/json` + `any`.

| Fixture | Comparator | ns/op | Ratio vs stdlib any | B/op | allocs/op | native-bytes/op | native-allocs/op | native-live-bytes |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| `twitter.json` | `pure-simdjson` | `2,044,151` | `3.15x` | `1,480,822` | `28,063` | `9,630,432` | `12` | `0` |
| `twitter.json` | `encoding-json-any` | `6,443,140` | `1.00x` | `2,076,696` | `32,125` | `0` | `0` | `0` |
| `twitter.json` | `encoding-json-struct` | `4,994,402` | n/a | `118,057` | `1,205` | `0` | `0` | `0` |
| `twitter.json` | `minio-simdjson-go` | `3,184,176` | n/a | `4,362,482` | `30,717` | `0` | `0` | `0` |
| `twitter.json` | `bytedance-sonic` | `2,008,950` | n/a | `2,533,101` | `13,002` | `0` | `0` | `0` |
| `twitter.json` | `goccy-go-json` | `3,510,296` | n/a | `2,681,140` | `40,562` | `0` | `0` | `0` |
| `citm_catalog.json` | `pure-simdjson` | `5,113,116` | `3.39x` | `4,600,980` | `76,325` | `23,749,544` | `13` | `0` |
| `citm_catalog.json` | `encoding-json-any` | `17,338,004` | `1.00x` | `5,130,765` | `95,865` | `0` | `0` | `0` |
| `citm_catalog.json` | `encoding-json-struct` | `17,532,250` | n/a | `1,232,839` | `16,920` | `0` | `0` | `0` |
| `citm_catalog.json` | `minio-simdjson-go` | `7,423,587` | n/a | `10,736,669` | `87,521` | `0` | `0` | `0` |
| `citm_catalog.json` | `bytedance-sonic` | `5,190,729` | n/a | `8,649,868` | `58,671` | `0` | `0` | `0` |
| `citm_catalog.json` | `goccy-go-json` | `8,553,034` | n/a | `7,423,603` | `124,071` | `0` | `0` | `0` |
| `canada.json` | `pure-simdjson` | `14,159,130` | `2.47x` | `4,952,800` | `223,253` | `50,018,288` | `15` | `0` |
| `canada.json` | `encoding-json-any` | `34,934,366` | `1.00x` | `10,583,553` | `392,516` | `0` | `0` | `0` |
| `canada.json` | `encoding-json-struct` | `38,768,708` | n/a | `6,267,800` | `114,178` | `0` | `0` | `0` |
| `canada.json` | `minio-simdjson-go` | `22,666,100` | n/a | `15,182,402` | `223,249` | `0` | `0` | `0` |
| `canada.json` | `bytedance-sonic` | `14,519,350` | n/a | `20,952,118` | `223,864` | `0` | `0` | `0` |
| `canada.json` | `goccy-go-json` | `28,912,204` | n/a | `11,366,098` | `446,471` | `0` | `0` | `0` |

## Tier 1 Diagnostics

The diagnostic family shows that the Phase 8 win comes from the Go materialization path while parse-only remains in the same range.

| Fixture | Diagnostic row | ns/op | B/op | allocs/op | native-bytes/op | native-allocs/op | native-live-bytes |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: |
| `twitter.json` | `pure-simdjson-full` | `2,057,104` | `1,480,823` | `28,063` | `9,630,432` | `12` | `0` |
| `twitter.json` | `pure-simdjson-parse-only` | `270,467` | `328` | `13` | `6,105,096` | `3` | `0` |
| `twitter.json` | `pure-simdjson-materialize-only` | `1,483,060` | `1,480,496` | `28,050` | `4,471` | `0` | `1,769,472` |
| `twitter.json` | `encoding-json-any-full` | `6,549,078` | `2,076,696` | `32,125` | `0` | `0` | `0` |
| `citm_catalog.json` | `pure-simdjson-full` | `5,245,686` | `4,601,000` | `76,325` | `23,749,544` | `13` | `0` |
| `citm_catalog.json` | `pure-simdjson-parse-only` | `788,324` | `328` | `13` | `16,696,712` | `3` | `0` |
| `citm_catalog.json` | `pure-simdjson-materialize-only` | `4,012,412` | `4,600,674` | `76,312` | `23,827` | `0` | `3,538,944` |
| `citm_catalog.json` | `encoding-json-any-full` | `17,323,110` | `5,130,770` | `95,865` | `0` | `0` | `0` |
| `canada.json` | `pure-simdjson-full` | `13,869,957` | `4,952,816` | `223,253` | `50,018,288` | `15` | `0` |
| `canada.json` | `pure-simdjson-parse-only` | `2,530,851` | `328` | `13` | `21,760,520` | `3` | `0` |
| `canada.json` | `pure-simdjson-materialize-only` | `8,816,500` | `4,952,475` | `223,240` | `210,098` | `0` | `14,155,776` |
| `canada.json` | `encoding-json-any-full` | `35,078,366` | `10,583,558` | `392,516` | `0` | `0` | `0` |

## Tier 2: Typed Extraction

Tier 2 measures schema-shaped typed extraction. Ratios are versus `encoding/json` + struct.

| Fixture | Comparator | ns/op | Ratio vs stdlib struct | B/op | allocs/op | native-bytes/op | native-allocs/op | native-live-bytes |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| `twitter.json` | `pure-simdjson` | `368,603` | `13.67x` | `23,169` | `692` | `6,105,096` | `3` | `0` |
| `twitter.json` | `encoding-json-struct` | `5,038,955` | `1.00x` | `118,056` | `1,205` | `0` | `0` | `0` |
| `twitter.json` | `bytedance-sonic` | `833,927` | n/a | `710,746` | `217` | `0` | `0` | `0` |
| `twitter.json` | `goccy-go-json` | `879,216` | n/a | `693,929` | `575` | `0` | `0` | `0` |
| `citm_catalog.json` | `pure-simdjson` | `1,231,068` | `14.56x` | `123,598` | `3,144` | `16,696,712` | `3` | `0` |
| `citm_catalog.json` | `encoding-json-struct` | `17,918,864` | `1.00x` | `1,234,427` | `16,923` | `0` | `0` | `0` |
| `citm_catalog.json` | `bytedance-sonic` | `2,461,716` | n/a | `2,699,156` | `5,443` | `0` | `0` | `0` |
| `citm_catalog.json` | `goccy-go-json` | `2,892,049` | n/a | `2,579,458` | `14,496` | `0` | `0` | `0` |
| `canada.json` | `pure-simdjson` | `3,133,064` | `12.49x` | `138,150` | `4,169` | `21,760,520` | `3` | `0` |
| `canada.json` | `encoding-json-struct` | `39,124,268` | `1.00x` | `6,267,791` | `114,177` | `0` | `0` | `0` |
| `canada.json` | `bytedance-sonic` | `7,117,254` | n/a | `7,121,287` | `58,101` | `0` | `0` | `0` |
| `canada.json` | `goccy-go-json` | `18,173,824` | n/a | `6,872,746` | `223,225` | `0` | `0` | `0` |

## Tier 3: Selective Traversal on the Current DOM API

Tier 3 measures selective reads on the current DOM API. It is not an On-Demand API claim. The current public selective-traversal snapshot covers `twitter_json` and `citm_catalog_json` only; it does not publish a `canada_json` Tier 3 claim.

| Fixture | Comparator | ns/op | Ratio vs stdlib struct | B/op | allocs/op | native-bytes/op | native-allocs/op | native-live-bytes |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| `twitter.json` | `pure-simdjson` | `313,001` | `15.97x` | `10,208` | `324` | `6,105,096` | `3` | `0` |
| `twitter.json` | `encoding-json-struct` | `4,999,739` | `1.00x` | `118,056` | `1,205` | `0` | `0` | `0` |
| `twitter.json` | `bytedance-sonic` | `801,321` | n/a | `709,712` | `217` | `0` | `0` | `0` |
| `twitter.json` | `goccy-go-json` | `856,472` | n/a | `693,519` | `574` | `0` | `0` | `0` |
| `citm_catalog.json` | `pure-simdjson` | `1,175,722` | `15.18x` | `88,156` | `2,873` | `16,696,712` | `3` | `0` |
| `citm_catalog.json` | `encoding-json-struct` | `17,844,443` | `1.00x` | `1,234,427` | `16,923` | `0` | `0` | `0` |
| `citm_catalog.json` | `bytedance-sonic` | `2,338,192` | n/a | `2,696,128` | `5,443` | `0` | `0` | `0` |
| `citm_catalog.json` | `goccy-go-json` | `2,879,005` | n/a | `2,580,485` | `14,496` | `0` | `0` | `0` |

## Cold vs Warm Parser Lifecycle

Cold start is the first parse after `NewParser` inside an already loaded process. It does not include bootstrap/download time.

| Fixture | Lifecycle | ns/op | B/op | allocs/op | native-bytes/op | native-allocs/op | native-live-bytes |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: |
| `twitter.json` | cold | `277,149` | `488` | `23` | `8,640,772` | `8` | `0` |
| `twitter.json` | warm | `270,798` | `328` | `13` | `6,105,096` | `3` | `0` |
| `citm_catalog.json` | cold | `813,034` | `488` | `23` | `23,615,108` | `8` | `0` |
| `citm_catalog.json` | warm | `791,390` | `328` | `13` | `16,696,712` | `3` | `0` |
| `canada.json` | cold | `2,575,436` | `488` | `23` | `30,774,276` | `8` | `0` |
| `canada.json` | warm | `2,539,128` | `328` | `13` | `21,760,520` | `3` | `0` |

## Comparator Notes

Comparator rows include only libraries that ran on this target. Unsupported comparators are omitted rather than rendered as synthetic `N/A`. README ratios remain stdlib-relative; full comparator rows live in this result document.

## Release Boundary

This benchmark snapshot is release-facing evidence for `v0.1.2`, but Phase 09.1 owns bootstrap artifact and default-install alignment before tagging.

release.yml expects the tag commit to be anchored on origin/main; before any later release tag, follow docs/releases.md and run bash scripts/release/check_readiness.sh --strict --version 0.1.2 from the prepared main commit.
