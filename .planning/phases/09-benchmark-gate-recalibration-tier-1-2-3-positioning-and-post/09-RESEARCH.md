# Phase 9: Benchmark Gate Recalibration, Tier 1/2/3 Positioning, and Post-ABI Evidence Refresh - Research

**Researched:** 2026-04-24 [VERIFIED: current_date prompt]
**Domain:** Go benchmark evidence capture, benchstat claim gating, GitHub Actions artifact handling, and release-facing benchmark documentation [VERIFIED: .planning/phases/09-benchmark-gate-recalibration-tier-1-2-3-positioning-and-post/09-CONTEXT.md]
**Confidence:** HIGH [VERIFIED: local planning files, benchmark harness files, existing evidence files, official GitHub docs, pkg.go.dev benchstat docs]

<user_constraints>
## User Constraints (from CONTEXT.md)

All locked decisions, discretion areas, and deferred ideas in this section are copied from `.planning/phases/09-benchmark-gate-recalibration-tier-1-2-3-positioning-and-post/09-CONTEXT.md`. [VERIFIED: .planning/phases/09-benchmark-gate-recalibration-tier-1-2-3-positioning-and-post/09-CONTEXT.md]

### Locked Decisions

#### Evidence Scope
- **D-01:** Rerun the full public benchmark evidence set: Tier 1/2/3, Tier 1 diagnostics, and cold/warm parser lifecycle rows.
- **D-02:** Public positioning compares against current industry comparators, especially `encoding/json`, not against the older weaker pure-simdjson implementation. Phase 7 and Phase 8 evidence remain historical context and regression baselines, not the public competitive bar.
- **D-03:** Require real `linux/amd64` evidence before changing public benchmark wording. Existing workflows include dispatchable smoke/release-validation jobs, but no benchmark-capture workflow; Phase 9 should add or use a dedicated `workflow_dispatch` benchmark job on real `linux/amd64`.
- **D-04:** Commit raw `.bench.txt`, `benchstat` output, and a machine-readable gate/summary output for the release-scoped benchmark snapshot.

#### Public Positioning
- **D-05:** If real `linux/amd64` evidence shows a benchstat-significant Tier 1 win over `encoding/json + any` on every published Tier 1 fixture, README may lead with Tier 1 as a supported headline.
- **D-06:** If Tier 1 greatly improves but does not beat `encoding/json + any` under the headline gate, README should say Tier 1 greatly improved while typed extraction and selective traversal remain the headline.
- **D-07:** Use moderate platform caveats in public copy: headline numbers come from `linux/amd64`; other platforms may differ. Exact GOOS/GOARCH/CPU/toolchain metadata belongs in the results document.
- **D-08:** README should show only stdlib-relative ratios. Full comparator tables, including named industry comparators, belong in benchmark docs.

#### Result Artifact Shape
- **D-09:** Detach benchmark docs from phase numbering. Do not create `results-phase9.md`; public benchmark snapshots are tied to a release or upcoming release.
- **D-10:** Use `v0.1.2` as the working next benchmark snapshot label for planning: `docs/benchmarks/results-v0.1.2.md` and `testdata/benchmark-results/v0.1.2/`. If planning discovers the release train requires a different semver label, update every docs/raw/workflow path consistently before capture.
- **D-11:** Investigate GitHub-hosted benchmark history as an auxiliary surface, not the durable source of truth. Native GitHub Actions artifacts are useful for workflow bundles but retention-limited. `benchmark-action/github-action-benchmark` is the benchmark-specific option for Go benchmark history/charts through GitHub Pages.
- **D-12:** The machine-readable summary must include ratios, target/toolchain metadata, thresholds, and which public claims are allowed.
- **D-13:** README should link to `docs/benchmarks.md`; that methodology page owns the pointer to the current benchmark snapshot.

#### Release Decision Boundary
- **D-14:** Phase 9 locks the next benchmark snapshot version label so docs and evidence paths are release/upcoming-release scoped.
- **D-15:** Phase 9 may recommend a patch release, but Phase 09.1 performs release/bootstrap artifact alignment first.
- **D-16:** Benchmark capture should run before release tagging against a release-candidate commit. A release workflow may verify, attach, or publish already-produced benchmark artifacts, but it should not be the first place where public benchmark claims are discovered.
- **D-17:** If benchmark evidence is strong while the default bootstrap path is still pinned to old artifacts, publish benchmark docs but keep README language framed as an upcoming-release claim until Phase 09.1 validates default installs.
- **D-18:** Update `CHANGELOG.md` under `Unreleased` with benchmark-positioning notes during Phase 9.

#### Benchmark Gate Policy
- **D-19:** Tier 1 headline claims require a benchstat-significant win over `encoding/json + any` on every published Tier 1 fixture.
- **D-20:** Tier 2 and Tier 3 claims require both no material regression versus the current public snapshot and continued wins over `encoding/json + struct` on every published fixture.
- **D-21:** Noisy headline runs fail closed: rerun until significance or keep older/more conservative claims.
- **D-22:** Gate output should emit per-fixture statuses plus generated README/doc claim allowances, not just a single pass/fail bit.

### Claude's Discretion

- Exact workflow filename and script names for benchmark capture.
- Exact machine-readable gate output format, as long as it is committed, deterministic, and includes the fields in D-12 and D-22.
- Exact wording of README/changelog/result docs, as long as the claim gates above are enforced.

### Deferred Ideas (OUT OF SCOPE)

- Publishing or aligning native release artifacts is deferred to Phase 09.1.
- Default-install bootstrap validation is deferred to Phase 09.1.
- New parser/API/materializer optimization work discovered during benchmarking belongs in a later phase or backlog item.
- Treating GitHub Pages benchmark history as the primary public source of truth is deferred unless planning explicitly accepts the third-party action and its write/Pages tradeoffs.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| BENCH-01 | Three-tier benchmark harness: Tier 1 full parse/walk, Tier 2 typed extraction, Tier 3 selective-path placeholder [VERIFIED: .planning/REQUIREMENTS.md] | Existing benchmark families already cover all three tiers through `BenchmarkTier1FullParse_*`, `BenchmarkTier2Typed_*`, and `BenchmarkTier3SelectivePlaceholder_*`; Phase 9 should capture them unchanged and gate claims from their rows. [VERIFIED: benchmark_fullparse_test.go, benchmark_typed_test.go, benchmark_selective_test.go] |
| BENCH-02 | Canonical corpus includes `twitter.json`, `canada.json`, `citm_catalog.json`, `mesh.json`, and `numbers.json` [VERIFIED: .planning/REQUIREMENTS.md] | Published Tier 1/2/3 rows currently use `twitter.json`, `citm_catalog.json`, and `canada.json`; Phase 9 should not expand public claims to fixtures without published rows unless it captures and gates them consistently. [VERIFIED: benchmark_comparators_test.go, docs/benchmarks/results-v0.1.1.md] |
| BENCH-03 | Comparison baselines include `encoding/json` + `any`, `encoding/json` + struct, `minio/simdjson-go`, `bytedance/sonic`, and `goccy/go-json`. [VERIFIED: .planning/REQUIREMENTS.md] | Comparator registry already encodes these baselines and omits unavailable target-specific comparators structurally; Phase 9 docs should preserve omission rules. [VERIFIED: benchmark_comparators_test.go, benchmark_comparators_minio_amd64_test.go, benchmark_comparators_minio_stub_test.go, benchmark_comparators_sonic_supported_test.go, benchmark_comparators_sonic_stub_test.go] |
| BENCH-04 | Results reported via `benchstat`; cold-start reported separately from warm. [VERIFIED: .planning/REQUIREMENTS.md] | `scripts/bench/run_benchstat.sh` wraps `benchstat`, and `BenchmarkColdStart_*` / `BenchmarkWarm_*` are separate benchmark families. [VERIFIED: scripts/bench/run_benchstat.sh, benchmark_coldstart_test.go] |
| BENCH-05 | Native allocator stats reported beside Go allocation counts. [VERIFIED: .planning/REQUIREMENTS.md] | Benchmark helpers report `native-bytes/op`, `native-allocs/op`, and `native-live-bytes` through `b.ReportMetric`; docs already explain why native metrics accompany Go `benchmem`. [VERIFIED: benchmark_native_alloc_test.go, docs/benchmarks.md; CITED: https://pkg.go.dev/testing#B.ReportMetric] |
| BENCH-07 | README benchmark positioning links committed evidence, labels Tier 1 as full `any`, positions Tier 2/Tier 3 honestly, and avoids unsupported Tier 1/x86_64 claims. [VERIFIED: .planning/REQUIREMENTS.md] | Phase 9 must replace the v0.1.1 README/results snapshot with claim-gated v0.1.2-or-equivalent wording based on real `linux/amd64` evidence. [VERIFIED: README.md, docs/benchmarks.md, 09-CONTEXT.md] |
| DOC-01 | README includes installation, quick start, platform matrix, and benchmark snapshot. [VERIFIED: .planning/REQUIREMENTS.md] | README already has all required sections; Phase 9 should edit only the benchmark snapshot/pointer and any claim wording affected by the new gate. [VERIFIED: README.md] |
| DOC-06 | CHANGELOG follows Keep a Changelog. [VERIFIED: .planning/REQUIREMENTS.md] | `CHANGELOG.md` has an `Unreleased` section and Keep-a-Changelog/SemVer references; Phase 9 should add benchmark-positioning notes under `Unreleased`. [VERIFIED: CHANGELOG.md] |
</phase_requirements>

## Project Constraints

- Do not include internal repository hostnames or internal company information in generated artifacts such as commits, PRs, docs, or release notes. [VERIFIED: user-provided AGENTS.md instruction]
- No root `AGENTS.md` or `CLAUDE.md` file exists in this checkout; the prompt-supplied instruction is the active project-specific rule for this research. [VERIFIED: `find . -maxdepth 3 \( -name AGENTS.md -o -name CLAUDE.md \) -print`]
- Before recommending a release action, read `docs/releases.md`; this was done for this research. [VERIFIED: .agents/skills/pure-simdjson-release/SKILL.md, docs/releases.md]
- The supported release sequence is `main` commit, strict readiness gate, annotated tag, tag push, then CI publish. [VERIFIED: docs/releases.md]
- `release.yml` expects the tag commit to be anchored on `origin/main`; do not recommend a tag from a non-main or unprepared commit. [VERIFIED: docs/releases.md, .github/workflows/release.yml]
- CI is the only supported publish path; do not hand-upload release artifacts, bypass CI, or reintroduce prep-branch checksum generation as a release dependency. [VERIFIED: .agents/skills/pure-simdjson-release/SKILL.md, docs/releases.md]
- Phase 09.1 owns bootstrap artifact alignment and default-install validation after Phase 9 locks benchmark/docs evidence. [VERIFIED: 09-CONTEXT.md, .planning/ROADMAP.md]

## Summary

Phase 9 should be planned as an evidence-capture and claim-gating phase, not a performance-optimization phase. [VERIFIED: 09-CONTEXT.md] The existing harness already has the needed benchmark families for Tier 1, Tier 2, Tier 3, diagnostics, and cold/warm lifecycle rows. [VERIFIED: benchmark_fullparse_test.go, benchmark_typed_test.go, benchmark_selective_test.go, benchmark_diagnostics_test.go, benchmark_coldstart_test.go] The missing piece is a real `linux/amd64` capture path plus a deterministic gate that translates benchmark evidence into allowed public claims. [VERIFIED: 09-CONTEXT.md, .github/workflows]

The current public snapshot is `v0.1.1` and was captured on `darwin/arm64` Apple M3 Max, while Phase 8 internal evidence shows same-host Tier 1 diagnostic improvement but remains internal and non-public-positioning evidence. [VERIFIED: docs/benchmarks/results-v0.1.1.md, .planning/phases/08-low-overhead-dom-traversal-abi-and-specialized-go-any-materi/08-BENCHMARK-NOTES.md] Phase 9 must capture release-scoped raw benchmark output, benchstat output, and a machine-readable summary under `testdata/benchmark-results/v0.1.2/` unless the planner consistently changes the release label. [VERIFIED: 09-CONTEXT.md]

GitHub Actions artifacts are useful for run bundles, but they are retention-limited and should not replace committed benchmark evidence. [CITED: https://docs.github.com/en/actions/tutorials/store-and-share-data; CITED: https://docs.github.com/en/organizations/managing-organization-settings/configuring-the-retention-period-for-github-actions-artifacts-and-logs-in-your-organization] `benchmark-action/github-action-benchmark` can be evaluated as an auxiliary GitHub Pages/chart surface, but it requires write/deploy tradeoffs and should not become the durable source of truth in Phase 9. [CITED: https://github.com/benchmark-action/github-action-benchmark; VERIFIED: 09-CONTEXT.md]

**Primary recommendation:** Add a dispatchable `linux/amd64` benchmark-capture workflow, generalize the Phase 8 Python gate into a release-snapshot claim gate, commit `v0.1.2` raw/benchstat/summary artifacts, then update README, docs, and changelog strictly from the generated claim allowances. [VERIFIED: 09-CONTEXT.md, scripts/bench/check_phase8_tier1_improvement.py, docs/benchmarks.md, README.md, CHANGELOG.md]

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|--------------|----------------|-----------|
| Benchmark execution on real target | GitHub Actions runner | Go test binary | `linux/amd64` evidence must be captured on a real GitHub-hosted or equivalent Linux amd64 runner; `go test -bench` is the execution mechanism. [VERIFIED: 09-CONTEXT.md; VERIFIED: go help testflag] |
| Comparator benchmark semantics | Go benchmark code | Test fixtures | Comparator selection, fixture loading, and row names live in benchmark test files and committed `testdata/bench` inputs. [VERIFIED: benchmark_comparators_test.go, benchmark_fixtures_test.go] |
| Statistical comparison | Scripts/tooling | Go x/perf benchstat | Existing wrapper delegates to `benchstat`, which summarizes and compares Go benchmark files. [VERIFIED: scripts/bench/run_benchstat.sh; CITED: https://pkg.go.dev/golang.org/x/perf/cmd/benchstat] |
| Claim gating | Scripts/tooling | Docs/README | The gate should parse raw benchmark files and emit deterministic claim allowances that docs consume. [VERIFIED: 09-CONTEXT.md, scripts/bench/check_phase8_tier1_improvement.py] |
| Public benchmark story | Repository documentation | README/changelog | README should stay concise and stdlib-relative; methodology/results docs own detailed comparator tables and target metadata. [VERIFIED: 09-CONTEXT.md, README.md, docs/benchmarks.md] |
| Release recommendation boundary | Release runbook | Phase 09.1 | Phase 9 may recommend a patch release, but publication/bootstrap alignment is deferred. [VERIFIED: 09-CONTEXT.md, docs/releases.md, .agents/skills/pure-simdjson-release/SKILL.md] |

## Standard Stack

### Core

| Library / Tool | Version | Purpose | Why Standard |
|----------------|---------|---------|--------------|
| Go benchmark framework | module declares Go 1.24; local toolchain `go1.26.2 darwin/arm64` [VERIFIED: go.mod, `go version`] | Run `go test -bench`, `-benchmem`, `-count`, and benchmark subtests. [VERIFIED: go help testflag] | Existing harness is Go benchmark-native and already produces benchstat-compatible rows. [VERIFIED: benchmark_*_test.go] |
| `golang.org/x/perf/cmd/benchstat` | installed pseudo-version `v0.0.0-20260209182753-b57e4e371b65`; built with Go 1.25.7 [VERIFIED: `go version -m $(command -v benchstat)`] | Statistical summaries and A/B comparisons for Go benchmark output. [CITED: https://pkg.go.dev/golang.org/x/perf/cmd/benchstat] | Existing script already requires `benchstat`; Phase 9 gate policy depends on significance. [VERIFIED: scripts/bench/run_benchstat.sh, 09-CONTEXT.md] |
| Python 3 | local `Python 3.11.7` [VERIFIED: `python3 --version`] | Deterministic parsing and claim-gate summary generation. [VERIFIED: scripts/bench/check_phase8_tier1_improvement.py] | Existing Phase 8 machine gate is Python and has unit tests. [VERIFIED: scripts/bench/check_phase8_tier1_improvement.py, tests/bench/test_check_phase8_improvement.py] |
| GitHub Actions workflow dispatch | existing workflows use `workflow_dispatch`; benchmark capture workflow is absent. [VERIFIED: .github/workflows/public-bootstrap-validation.yml, .github/workflows] | Run capture on real `linux/amd64` and upload temporary bundles. [VERIFIED: 09-CONTEXT.md] | Phase 9 explicitly requires a dispatchable benchmark-capture job instead of relying on release workflow discovery. [VERIFIED: 09-CONTEXT.md] |

### Supporting

| Library / Tool | Version | Purpose | When to Use |
|----------------|---------|---------|-------------|
| `actions/upload-artifact` | existing repo pins SHA `ea165f8d65b6e75b540449e92b4886f43607fa02`, tag-verified as `v4.6.2`; latest remote tag seen was `v7.0.1` [VERIFIED: .github/workflows/release.yml, `git ls-remote --tags https://github.com/actions/upload-artifact.git`] | Upload raw `.bench.txt`, benchstat, summary JSON, and metadata bundle from workflow run. [CITED: https://docs.github.com/en/actions/tutorials/store-and-share-data] | Use as ephemeral workflow bundle; committed files remain durable source. [VERIFIED: 09-CONTEXT.md; CITED: https://docs.github.com/en/actions/tutorials/store-and-share-data] |
| `actions/download-artifact` | latest remote tag seen was `v8.0.1`; docs examples show download usage. [VERIFIED: `git ls-remote --tags https://github.com/actions/download-artifact.git`; CITED: https://docs.github.com/en/actions/tutorials/store-and-share-data] | Optional if a multi-job benchmark workflow needs to collect outputs from earlier jobs. [CITED: https://docs.github.com/en/actions/tutorials/store-and-share-data] | Use only if the workflow separates capture, benchstat, and summary jobs; a single job can avoid it. [VERIFIED: local workflow patterns] |
| `benchmark-action/github-action-benchmark` | latest remote tag seen was `v1.9.0`; README documents semver usage and `@v1` behavior. [VERIFIED: `git ls-remote --tags https://github.com/benchmark-action/github-action-benchmark.git`; CITED: https://github.com/benchmark-action/github-action-benchmark] | Auxiliary trend chart/history surface for Go benchmark output. [CITED: https://github.com/benchmark-action/github-action-benchmark] | Investigate only as optional GitHub Pages history; do not replace committed release-scoped evidence. [VERIFIED: 09-CONTEXT.md] |
| `encoding/json` | Go standard library, tied to selected Go toolchain. [VERIFIED: go.mod, go help testflag] | Public stdlib baseline for Tier 1 `any` and Tier 2/3 struct claims. [VERIFIED: benchmark_comparators_test.go, benchmark_typed_test.go] | Always include in public ratios and README wording. [VERIFIED: 09-CONTEXT.md] |
| `github.com/bytedance/sonic` | `v1.15.0`, published `2026-01-22T12:41:14Z` [VERIFIED: `go list -m -u -json github.com/bytedance/sonic`] | Industry comparator where target/toolchain supports it. [VERIFIED: go.mod, benchmark_comparators_sonic_supported_test.go] | Include in full comparator docs, subject to omission rules. [VERIFIED: 09-CONTEXT.md, benchmark_comparators_test.go] |
| `github.com/goccy/go-json` | `v0.10.6`, published `2025-10-28T00:14:29Z` [VERIFIED: `go list -m -u -json github.com/goccy/go-json`] | Industry comparator in Tier 1 and typed decode paths. [VERIFIED: go.mod, benchmark_comparators_test.go, benchmark_typed_test.go] | Include in full comparator docs. [VERIFIED: 09-CONTEXT.md] |
| `github.com/minio/simdjson-go` | `v0.4.5`, published `2023-03-11T19:16:56Z` [VERIFIED: `go list -m -u -json github.com/minio/simdjson-go`] | Industry x86_64 comparator. [VERIFIED: go.mod, benchmark_comparators_minio_amd64_test.go] | Include only when it actually runs on the target; omit structurally otherwise. [VERIFIED: benchmark_comparators_minio_stub_test.go, docs/benchmarks.md] |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Committed release-scoped evidence | Native GitHub Actions artifacts only | Artifacts are useful for sharing workflow-produced files, but retention is limited and configurable; this fails the durable source-of-truth requirement. [CITED: https://docs.github.com/en/actions/tutorials/store-and-share-data; CITED: https://docs.github.com/en/organizations/managing-organization-settings/configuring-the-retention-period-for-github-actions-artifacts-and-logs-in-your-organization; VERIFIED: 09-CONTEXT.md] |
| In-repo deterministic gate | `benchmark-action/github-action-benchmark` failure threshold only | The action supports Go benchmark output, historical comparison, alerts, and GitHub Pages charts, but Phase 9 needs release-specific claim allowances with project-specific Tier 1/2/3 semantics. [CITED: https://github.com/benchmark-action/github-action-benchmark; VERIFIED: 09-CONTEXT.md] |
| `benchstat` significance gate | Median-ratio-only script | Median ratios are useful for docs, but the locked Tier 1 headline requires benchstat-significant wins. [VERIFIED: 09-CONTEXT.md; CITED: https://pkg.go.dev/golang.org/x/perf/cmd/benchstat] |
| Existing release workflow | First-discovery benchmark run inside `release.yml` | Release workflow should not be the first place public claims are discovered; benchmark capture should happen before tagging. [VERIFIED: 09-CONTEXT.md, docs/releases.md] |

**Installation / availability:**

```bash
go install golang.org/x/perf/cmd/benchstat@latest
python3 --version
go version
```

Version verification performed before writing this section: Go, Python, local benchstat binary, Go module comparator dependencies, and GitHub action tags were checked with local commands. [VERIFIED: `go version`, `python3 --version`, `go version -m $(command -v benchstat)`, `go list -m -u -json ...`, `git ls-remote --tags ...`]

## Architecture Patterns

### System Architecture Diagram

```text
workflow_dispatch input: snapshot label/version
        |
        v
GitHub Actions linux/amd64 benchmark job
        |
        +--> cargo build --release / local native library availability
        |
        +--> go test -bench Tier1/2/3 -benchmem -count=N
        |        |
        |        v
        |   raw phase benchmark file
        |
        +--> go test -bench ColdStart/Warm -benchmem -count=N
        |        |
        |        v
        |   raw coldwarm file
        |
        +--> go test -bench Tier1Diagnostics -benchmem -count=N
                 |
                 v
            raw diagnostics file
                 |
                 v
scripts/bench/run_benchstat.sh compares v0.1.1 baseline to new snapshot
                 |
                 v
scripts/bench/check_benchmark_claims.py
        |
        +--> parse metadata: goos/goarch/pkg/cpu/toolchain
        +--> parse rows: Tier 1 vs encoding-json-any, Tier 2/3 vs encoding-json-struct
        +--> require benchstat significance where headline gates demand it
        +--> emit per-fixture PASS/FAIL + claim allowances
                 |
                 v
committed release-scoped evidence
        |
        +--> testdata/benchmark-results/v0.1.2/*.bench.txt
        +--> testdata/benchmark-results/v0.1.2/*.benchstat.txt
        +--> testdata/benchmark-results/v0.1.2/summary.json
        +--> docs/benchmarks/results-v0.1.2.md
        +--> docs/benchmarks.md pointer
        +--> README benchmark snapshot
        +--> CHANGELOG.md Unreleased note
```

The diagram reflects the locked requirement that benchmark discovery happens before release tagging and that committed evidence remains the durable source of truth. [VERIFIED: 09-CONTEXT.md, docs/releases.md]

### Recommended Project Structure

```text
.github/workflows/
  benchmark-capture.yml                 # workflow_dispatch linux/amd64 capture job [VERIFIED: 09-CONTEXT.md]
scripts/bench/
  run_benchstat.sh                       # existing wrapper [VERIFIED: scripts/bench/run_benchstat.sh]
  check_benchmark_claims.py              # new generalized gate, derived from Phase 8 gate [VERIFIED: scripts/bench/check_phase8_tier1_improvement.py]
tests/bench/
  test_check_benchmark_claims.py         # new unit tests for claim gates [VERIFIED: tests/bench/test_check_phase8_improvement.py pattern]
testdata/benchmark-results/v0.1.2/
  phase9.bench.txt                       # Tier 1/2/3 raw linux/amd64 evidence [VERIFIED: 09-CONTEXT.md]
  coldwarm.bench.txt                     # cold/warm raw linux/amd64 evidence [VERIFIED: 09-CONTEXT.md]
  tier1-diagnostics.bench.txt            # diagnostic raw linux/amd64 evidence [VERIFIED: 09-CONTEXT.md]
  phase9.benchstat.txt                   # benchstat vs current public snapshot [VERIFIED: BENCH-04]
  coldwarm.benchstat.txt                 # lifecycle comparison output [VERIFIED: BENCH-04]
  tier1-diagnostics.benchstat.txt        # diagnostic comparison output [VERIFIED: BENCH-04]
  summary.json                           # machine-readable metadata, thresholds, ratios, claim allowances [VERIFIED: 09-CONTEXT.md]
docs/benchmarks/
  results-v0.1.2.md                      # public release/upcoming-release snapshot [VERIFIED: 09-CONTEXT.md]
```

Use `phase9.bench.txt` or `tier123.bench.txt` as the main raw file name, but keep every path under the release snapshot directory and update names consistently if the planner chooses a different label. [VERIFIED: 09-CONTEXT.md]

### Pattern 1: Reuse Stable Benchmark Row Names

**What:** Benchmark rows should remain the existing `BenchmarkTier1FullParse_*`, `BenchmarkTier2Typed_*`, `BenchmarkTier3SelectivePlaceholder_*`, `BenchmarkTier1Diagnostics_*`, `BenchmarkColdStart_*`, and `BenchmarkWarm_*` names. [VERIFIED: benchmark_fullparse_test.go, benchmark_typed_test.go, benchmark_selective_test.go, benchmark_diagnostics_test.go, benchmark_coldstart_test.go]

**When to use:** Use these exact families for the public snapshot so benchstat can compare historical and new output and docs can map rows to existing methodology. [VERIFIED: docs/benchmarks.md, scripts/bench/run_benchstat.sh]

**Example:**

```bash
# Source: docs/benchmarks.md + go help testflag [VERIFIED]
go test ./... -run '^$' -bench 'Benchmark(Tier1FullParse|Tier2Typed|Tier3SelectivePlaceholder)_' -benchmem -count=10 -timeout 1200s > testdata/benchmark-results/v0.1.2/phase9.bench.txt
go test ./... -run '^$' -bench 'Benchmark(ColdStart|Warm)_' -benchmem -count=10 -timeout 1200s > testdata/benchmark-results/v0.1.2/coldwarm.bench.txt
go test ./... -run '^$' -bench 'BenchmarkTier1Diagnostics_' -benchmem -count=10 -timeout 1200s > testdata/benchmark-results/v0.1.2/tier1-diagnostics.bench.txt
```

The count should be at least 10 for public gating because benchstat guidance recommends choosing at least 10 benchmark runs and sticking to it. [CITED: https://pkg.go.dev/golang.org/x/perf/cmd/benchstat]

### Pattern 2: Deterministic Claim Gate Summary

**What:** Generalize the Phase 8 Python parser into a claim gate that emits deterministic machine-readable output. [VERIFIED: scripts/bench/check_phase8_tier1_improvement.py, 09-CONTEXT.md]

**When to use:** Use it after raw capture and benchstat comparison, before any README/docs/changelog edit. [VERIFIED: 09-CONTEXT.md]

**Example output shape:**

```json
{
  "snapshot": "v0.1.2",
  "target": {
    "goos": "linux",
    "goarch": "amd64",
    "pkg": "github.com/amikos-tech/pure-simdjson",
    "cpu": "..."
  },
  "thresholds": {
    "tier1_headline": "benchstat_significant_win_vs_encoding_json_any_every_fixture",
    "tier2_tier3": "no_material_regression_vs_v0.1.1_and_win_vs_encoding_json_struct_every_fixture"
  },
  "claims": {
    "tier1_headline_allowed": false,
    "tier2_headline_allowed": true,
    "tier3_headline_allowed": true,
    "readme_mode": "tier1_improved_but_tier2_tier3_headline"
  },
  "fixtures": []
}
```

This structure covers ratios, metadata, thresholds, and allowed public claims required by D-12 and D-22. [VERIFIED: 09-CONTEXT.md]

### Pattern 3: Use Workflow Artifacts as a Transport, Not the Record

**What:** The benchmark workflow may upload the generated raw, benchstat, and summary files for review. [CITED: https://docs.github.com/en/actions/tutorials/store-and-share-data]

**When to use:** Use artifacts to retrieve run output from `workflow_dispatch`; after review, commit the release-scoped evidence into `testdata/benchmark-results/<label>/`. [VERIFIED: 09-CONTEXT.md]

**Example:**

```yaml
# Source: GitHub Actions artifact docs + existing release workflow pattern [CITED/VERIFIED]
- name: Upload benchmark evidence
  uses: actions/upload-artifact@ea165f8d65b6e75b540449e92b4886f43607fa02
  with:
    name: benchmark-evidence-v0.1.2-linux-amd64
    path: testdata/benchmark-results/v0.1.2/
    if-no-files-found: error
    retention-days: 30
```

`retention-days` cannot exceed the repository, organization, or enterprise retention limit. [CITED: https://docs.github.com/en/actions/tutorials/store-and-share-data]

### Anti-Patterns to Avoid

- **Promoting Phase 8 internal diagnostics to public headline copy:** Phase 8 evidence is same-host `darwin/arm64` diagnostic evidence, not real `linux/amd64` public positioning evidence. [VERIFIED: 08-BENCHMARK-NOTES.md, 09-CONTEXT.md]
- **Rerunning until benchstat says significant without a fixed run count:** Benchstat warns that repeated reruns looking for significance are multiple testing; set a run count and fail closed on noisy headline results. [CITED: https://pkg.go.dev/golang.org/x/perf/cmd/benchstat; VERIFIED: 09-CONTEXT.md]
- **Using GitHub Actions artifacts as durable benchmark history:** Public artifact/log retention defaults to 90 days and public repositories are capped at 1 to 90 days. [CITED: https://docs.github.com/en/organizations/managing-organization-settings/configuring-the-retention-period-for-github-actions-artifacts-and-logs-in-your-organization]
- **Adding benchmark discovery to `release.yml`:** Release tagging must happen after claim evidence exists; `release.yml` should not be where public claims are first discovered. [VERIFIED: 09-CONTEXT.md, docs/releases.md]
- **Showing full competitor tables in README:** README should show only stdlib-relative ratios and link to methodology/results docs for named comparators. [VERIFIED: 09-CONTEXT.md]
- **Changing benchmark fixture or comparator semantics while recalibrating claims:** Phase 9 is not an optimization or harness-redesign phase; changing semantics invalidates comparison to `v0.1.1`. [VERIFIED: 09-CONTEXT.md, 07-LEARNINGS.md]

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Statistical benchmark comparison | Custom p-value or significance implementation | `benchstat` via `scripts/bench/run_benchstat.sh` | `benchstat` is the Go ecosystem tool already in the repo and uses median summaries plus Mann-Whitney U-test by default. [VERIFIED: scripts/bench/run_benchstat.sh; CITED: https://pkg.go.dev/golang.org/x/perf/cmd/benchstat] |
| Benchmark execution harness | Custom timing loops outside Go benchmarks | `go test -bench` with existing `testing.B` rows | Existing benchmark code already reports bytes, allocs, native metrics, and stable sub-benchmark names. [VERIFIED: benchmark_*_test.go] |
| Artifact transport | Custom curl/upload scripts for workflow bundles | `actions/upload-artifact` | GitHub documents artifact upload/download for workflow-produced files. [CITED: https://docs.github.com/en/actions/tutorials/store-and-share-data] |
| Long-term benchmark history UI | Custom chart generator in Phase 9 | Optional `benchmark-action/github-action-benchmark` | The action supports Go output, GitHub Pages charts, comparison, and alerts; Phase 9 should only investigate it as auxiliary. [CITED: https://github.com/benchmark-action/github-action-benchmark; VERIFIED: 09-CONTEXT.md] |
| Release publication | Manual artifact uploads or local release commands | Existing tag-driven `release.yml` after Phase 09.1 | The release runbook says CI is the only supported publish path. [VERIFIED: docs/releases.md, .agents/skills/pure-simdjson-release/SKILL.md] |

**Key insight:** The hard part is not running benchmarks; it is preventing docs and release copy from outrunning what real `linux/amd64` evidence permits. [VERIFIED: 09-CONTEXT.md, docs/benchmarks/results-v0.1.1.md, 08-BENCHMARK-NOTES.md]

## Common Pitfalls

### Pitfall 1: Benchstat Significance Versus Ratio-Only Claiming

**What goes wrong:** A large median ratio is treated as a headline win even when benchstat marks the row as noise or lacks enough samples. [VERIFIED: 09-CONTEXT.md; CITED: https://pkg.go.dev/golang.org/x/perf/cmd/benchstat]

**Why it happens:** Existing Phase 8 gate uses medians and a 10% improvement threshold, while Phase 9 Tier 1 headline policy explicitly requires benchstat-significant wins over `encoding/json + any`. [VERIFIED: scripts/bench/check_phase8_tier1_improvement.py, 09-CONTEXT.md]

**How to avoid:** Make the new gate parse benchstat output or compute a conservative significance status from benchstat-formatted comparison output, and fail closed when significance is absent. [VERIFIED: 09-CONTEXT.md; CITED: https://pkg.go.dev/golang.org/x/perf/cmd/benchstat]

**Warning signs:** Summary output contains only medians and ratios, or README wording is edited before the gate emits claim allowances. [VERIFIED: 09-CONTEXT.md]

### Pitfall 2: Mixing Baseline Purposes

**What goes wrong:** Phase 7/8 evidence becomes the public competitive bar instead of historical/regression context. [VERIFIED: 09-CONTEXT.md]

**Why it happens:** Phase 8 diagnostics show dramatic improvement versus Phase 7 on the same `darwin/arm64` host, but the public gate is against current `encoding/json` rows on real `linux/amd64`. [VERIFIED: 08-BENCHMARK-NOTES.md, docs/benchmarks/results-v0.1.1.md, 09-CONTEXT.md]

**How to avoid:** Separate gate outputs into `regression_vs_public_snapshot` and `public_claim_vs_stdlib` sections. [VERIFIED: 09-CONTEXT.md]

**Warning signs:** README says "96% faster" without clarifying that the comparison is versus the old pure-simdjson diagnostic path. [VERIFIED: 08-BENCHMARK-NOTES.md]

### Pitfall 3: Target Metadata Drift

**What goes wrong:** Docs present ratios without exact GOOS/GOARCH/CPU/toolchain, making platform caveats impossible to verify. [VERIFIED: 09-CONTEXT.md, docs/benchmarks/results-v0.1.1.md]

**Why it happens:** Go benchmark output includes `goos`, `goarch`, `pkg`, and `cpu`, but toolchain and OS metadata require explicit capture. [VERIFIED: raw benchmark files, docs/benchmarks/results-v0.1.1.md]

**How to avoid:** Workflow should write a metadata file with `go version`, `rustc --version`, OS release info, runner name, commit SHA, and benchmark command lines into the snapshot directory. [VERIFIED: docs/benchmarks/results-v0.1.1.md pattern]

**Warning signs:** `summary.json` lacks `linux/amd64`, Go/Rust versions, or commit SHA. [VERIFIED: 09-CONTEXT.md]

### Pitfall 4: Optional Comparator Availability Misreported

**What goes wrong:** Unsupported comparators appear as fake zero, `N/A`, or failed rows in public tables. [VERIFIED: 07-CONTEXT.md, docs/benchmarks.md]

**Why it happens:** `minio/simdjson-go` and `sonic` have target/toolchain constraints; this repo handles those with build-tag availability/omission files. [VERIFIED: benchmark_comparators_minio_amd64_test.go, benchmark_comparators_minio_stub_test.go, benchmark_comparators_sonic_supported_test.go, benchmark_comparators_sonic_stub_test.go]

**How to avoid:** Generate docs only from rows that actually ran and list omission reasons separately if needed. [VERIFIED: benchmark_comparators_test.go, docs/benchmarks.md]

**Warning signs:** Result doc has `N/A` cells for comparators or README names non-stdlib competitors. [VERIFIED: 09-CONTEXT.md]

### Pitfall 5: Release Boundary Inversion

**What goes wrong:** A tag or release recommendation is made before benchmark evidence is committed and before Phase 09.1 validates default-install artifact alignment. [VERIFIED: 09-CONTEXT.md, docs/releases.md]

**Why it happens:** Benchmark docs and release preparation are adjacent, but this phase deliberately stops before bootstrap artifact alignment. [VERIFIED: 09-CONTEXT.md, .planning/ROADMAP.md]

**How to avoid:** Phase 9 can recommend whether evidence supports a patch release, but any tag push must wait for Phase 09.1 and the strict release readiness gate. [VERIFIED: .agents/skills/pure-simdjson-release/SKILL.md, docs/releases.md]

**Warning signs:** Plan includes `git tag`, `git push`, artifact publication, or default-install validation work. [VERIFIED: 09-CONTEXT.md]

## Code Examples

### Capture Commands

```bash
# Source: docs/benchmarks.md, go help testflag, 09-CONTEXT.md [VERIFIED]
SNAPSHOT=v0.1.2
OUT="testdata/benchmark-results/${SNAPSHOT}"
mkdir -p "$OUT"

go test ./... -run '^$' -bench 'Benchmark(Tier1FullParse|Tier2Typed|Tier3SelectivePlaceholder)_' -benchmem -count=10 -timeout 1200s > "${OUT}/phase9.bench.txt"
go test ./... -run '^$' -bench 'Benchmark(ColdStart|Warm)_' -benchmem -count=10 -timeout 1200s > "${OUT}/coldwarm.bench.txt"
go test ./... -run '^$' -bench 'BenchmarkTier1Diagnostics_' -benchmem -count=10 -timeout 1200s > "${OUT}/tier1-diagnostics.bench.txt"
```

### Benchstat Comparison

```bash
# Source: scripts/bench/run_benchstat.sh [VERIFIED]
scripts/bench/run_benchstat.sh \
  --old testdata/benchmark-results/v0.1.1/phase7.bench.txt \
  --new testdata/benchmark-results/v0.1.2/phase9.bench.txt \
  > testdata/benchmark-results/v0.1.2/phase9.benchstat.txt
```

### Claim Gate Invocation

```bash
# Source: scripts/bench/check_phase8_tier1_improvement.py pattern + 09-CONTEXT.md [VERIFIED]
python3 scripts/bench/check_benchmark_claims.py \
  --baseline-dir testdata/benchmark-results/v0.1.1 \
  --snapshot-dir testdata/benchmark-results/v0.1.2 \
  --snapshot v0.1.2 \
  --require-target linux/amd64 \
  > testdata/benchmark-results/v0.1.2/summary.json
```

### Workflow Skeleton

```yaml
# Source: existing workflows + GitHub artifact docs [VERIFIED/CITED]
name: benchmark capture

on:
  workflow_dispatch:
    inputs:
      snapshot:
        description: benchmark snapshot label, for example v0.1.2
        required: true
        type: string

permissions:
  contents: read
  actions: read

jobs:
  linux-amd64:
    runs-on: ubuntu-latest
    timeout-minutes: 60
    steps:
      - uses: actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683
        with:
          submodules: recursive
      - uses: actions/setup-go@40f1582b2485089dde7abd97c1529aa768e1baff
        with:
          go-version-file: go.mod
      - name: Install benchstat
        run: go install golang.org/x/perf/cmd/benchstat@latest
      - name: Capture benchmark evidence
        run: bash scripts/bench/capture_release_snapshot.sh "${{ inputs.snapshot }}"
      - name: Upload benchmark evidence
        uses: actions/upload-artifact@ea165f8d65b6e75b540449e92b4886f43607fa02
        with:
          name: benchmark-evidence-${{ inputs.snapshot }}-linux-amd64
          path: testdata/benchmark-results/${{ inputs.snapshot }}/
          if-no-files-found: error
          retention-days: 30
```

The upload-artifact SHA above matches the existing repo-pinned action in `release.yml`. [VERIFIED: .github/workflows/release.yml]

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Phase-numbered benchmark artifacts | Release/upcoming-release snapshot paths such as `docs/benchmarks/results-v0.1.2.md` and `testdata/benchmark-results/v0.1.2/` | Locked in Phase 9 context on 2026-04-24 [VERIFIED: 09-CONTEXT.md] | Plans should not create `results-phase9.md`. [VERIFIED: 09-CONTEXT.md] |
| Tier 1 as assumed `>=3x` headline versus `encoding/json + any` | Tier 1 headline only if every published Tier 1 fixture has benchstat-significant `linux/amd64` win versus `encoding/json + any` | Locked in Phase 9 context on 2026-04-24 [VERIFIED: 09-CONTEXT.md] | README may need conservative wording even after large internal improvement. [VERIFIED: 09-CONTEXT.md, 08-BENCHMARK-NOTES.md] |
| Phase 7 public `darwin/arm64` snapshot | Phase 9 requires real `linux/amd64` evidence before public wording changes | Locked in Phase 9 context on 2026-04-24 [VERIFIED: 09-CONTEXT.md] | Planner needs a dispatchable Linux benchmark workflow. [VERIFIED: .github/workflows] |
| GitHub Actions artifact as possible storage | Artifacts as temporary workflow bundles; committed evidence as durable truth | Verified during Phase 9 research [CITED: GitHub artifact docs] | Benchmark result files must be committed after review. [VERIFIED: 09-CONTEXT.md] |
| Optional chart/history not present | `benchmark-action/github-action-benchmark` can provide auxiliary GitHub Pages charts | Verified during Phase 9 research [CITED: https://github.com/benchmark-action/github-action-benchmark] | Treat as optional and gated by write/Pages permissions. [CITED: https://docs.github.com/en/pages/getting-started-with-github-pages/using-custom-workflows-with-github-pages] |

**Deprecated/outdated:**
- `results-phase9.md`: explicitly disallowed; use release/upcoming-release snapshot naming. [VERIFIED: 09-CONTEXT.md]
- README named-comparator tables: explicitly disallowed; README should show stdlib-relative ratios and link to docs for full comparator tables. [VERIFIED: 09-CONTEXT.md]
- First-discovery release benchmark: explicitly disallowed; benchmark capture must precede tagging. [VERIFIED: 09-CONTEXT.md]

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The planner may choose whether the main raw Tier 1/2/3 file is named `phase9.bench.txt` or `tier123.bench.txt`, as long as paths remain release-scoped. [ASSUMED] | Recommended Project Structure | Low; naming affects docs/scripts consistency only. |
| A2 | Maintainers may prefer manual artifact retrieval and commit over a bot-created benchmark evidence commit/PR. [ASSUMED] | Open Questions | Medium; affects workflow permissions and implementation task shape. |
| A3 | The repo may or may not grant Pages/write permissions for auxiliary benchmark history. [ASSUMED] | Open Questions, Security Domain | Medium; determines whether `benchmark-action/github-action-benchmark` is implemented or deferred. |
| A4 | Phase 09.1 may require a different patch label before tagging. [ASSUMED] | Open Questions | Low; using a centralized snapshot input keeps renaming contained. |
| A5 | Ubuntu GitHub-hosted runners have bash available. [ASSUMED] | Environment Availability | Low; GitHub-hosted Linux runners normally provide bash, and scripts already use bash. |
| A6 | GitHub UI can serve as workflow dispatch/download fallback if `gh` auth is unavailable. [ASSUMED] | Environment Availability | Low; affects operator ergonomics, not benchmark correctness. |
| A7 | A `scripts/bench/capture_release_snapshot.sh` helper is optional but likely useful. [ASSUMED] | Validation Architecture | Low; the planner can inline commands in the workflow instead. |
| A8 | The selected validity windows for this research are estimates. [ASSUMED] | Metadata | Low; planner should recheck action versions before implementation if delayed. |
| A9 | GitHub Actions runner availability must be validated by dispatching the new workflow. [ASSUMED] | Environment Availability | Medium; a missing runner blocks remote capture and requires an alternate runner. |
| A10 | Adding workflow write tokens should require explicit approval. [ASSUMED] | Security Domain | Medium; affects whether optional Pages/history work is in scope. |
| A11 | A strict snapshot input regex and confined output path will be implemented in new scripts. [ASSUMED] | Security Domain | Medium; missing validation could permit path mistakes in workflow inputs. |
| A12 | Optional GitHub Pages benchmark history adoption depends on project permission decisions. [ASSUMED] | Metadata | Medium; affects whether auxiliary benchmark history is planned. |

## Open Questions (RESOLVED)

1. **RESOLVED - Should the benchmark-capture workflow commit results automatically or only upload artifacts?** [VERIFIED: 09-CONTEXT.md]
   - What we know: Phase 9 requires committed evidence, but existing workflows do not auto-commit benchmark captures. [VERIFIED: .github/workflows, 09-CONTEXT.md]
   - Resolution: Use upload-only workflow artifacts as transport, then import and commit reviewed evidence under `testdata/benchmark-results/v0.1.2/`. Do not grant workflow write permissions or create bot commits in the core Phase 9 plan. [VERIFIED: 09-01-PLAN.md, 09-02-PLAN.md]
   - Rationale: This satisfies durable committed evidence while avoiding unnecessary write tokens and preserving a human provenance checkpoint before public docs change. [VERIFIED: 09-VALIDATION.md]

2. **RESOLVED - Should `benchmark-action/github-action-benchmark` be implemented in Phase 9 or deferred?** [VERIFIED: 09-CONTEXT.md]
   - What we know: It supports Go benchmark output, trend storage, comparisons, and GitHub Pages charts. [CITED: https://github.com/benchmark-action/github-action-benchmark]
   - Resolution: Defer auxiliary benchmark history/charts from the core Phase 9 plan. Phase 9 uses committed release-scoped evidence as the durable source of truth and workflow artifacts as transport only. [VERIFIED: 09-01-PLAN.md, 09-03-PLAN.md]
   - Rationale: GitHub Pages/history would require additional write/deploy permissions and is explicitly auxiliary to the durable evidence requirement. [VERIFIED: 09-CONTEXT.md, 09-VALIDATION.md]

3. **RESOLVED - Should the next snapshot stay `v0.1.2`?** [VERIFIED: 09-CONTEXT.md]
   - What we know: Phase 9 context locks `v0.1.2` as the working label unless release-train planning discovers a different semver. [VERIFIED: 09-CONTEXT.md]
   - Resolution: Use `v0.1.2` consistently for Phase 9 docs, raw evidence, workflow input examples, and summary paths. If the release train later requires a different semver, rename every docs/raw/workflow path consistently before capture. [VERIFIED: 09-01-PLAN.md, 09-02-PLAN.md, 09-03-PLAN.md]
   - Rationale: A single release/upcoming-release label keeps benchmark evidence detached from phase numbering while preserving a clear Phase 09.1 release/bootstrap boundary. [VERIFIED: 09-CONTEXT.md]

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|-------------|-----------|---------|----------|
| Go | benchmark capture/tests | yes [VERIFIED: `go version`] | `go1.26.2 darwin/arm64` local; module declares `go 1.24` [VERIFIED: go.mod] | GitHub Actions should use `actions/setup-go` with `go-version-file: go.mod`. [VERIFIED: public-bootstrap-validation.yml] |
| Rust/Cargo | local native build before benchmark tests | yes [VERIFIED: `rustc --version`, `cargo --version`] | `rustc 1.89.0`, `cargo 1.89.0`; repo pins Rust `1.89.0` [VERIFIED: rust-toolchain.toml] | GitHub Actions should use existing repo Rust setup pattern before benchmarks. [VERIFIED: release.yml] |
| Python 3 | claim gate scripts/tests | yes [VERIFIED: `python3 --version`] | `Python 3.11.7` | None needed. [VERIFIED: scripts/bench/check_phase8_tier1_improvement.py] |
| Bash | shell wrappers/workflow scripts | yes [VERIFIED: `bash --version`] | GNU bash `5.3.9` local | Use POSIX-compatible shell only if GitHub runner lacks bash, but Ubuntu has bash by default. [ASSUMED] |
| benchstat | BENCH-04 comparisons | yes [VERIFIED: `command -v benchstat`, `benchstat -h`] | `golang.org/x/perf v0.0.0-20260209182753-b57e4e371b65` installed [VERIFIED: `go version -m`] | Workflow installs with `go install golang.org/x/perf/cmd/benchstat@latest`. [VERIFIED: scripts/bench/run_benchstat.sh] |
| GitHub CLI `gh` | optional workflow dispatch/manual retrieval | yes [VERIFIED: `gh --version`] | `gh version 2.91.0 (2026-04-22)` | Use GitHub UI for workflow dispatch/download if CLI auth is unavailable. [ASSUMED] |
| Git | release/runbook checks and source metadata | yes [VERIFIED: `git --version`] | `git version 2.54.0` | None. [VERIFIED: docs/releases.md] |

**Missing dependencies with no fallback:** None found locally for research and planning. [VERIFIED: environment probes]

**Missing dependencies with fallback:** No blocking local dependency was missing; GitHub Actions runner availability is external to the local machine and must be validated by dispatching the new workflow. [VERIFIED: environment probes; ASSUMED: GitHub runner availability]

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go `testing` package for project tests/benchmarks; Python `unittest` for benchmark gate scripts. [VERIFIED: benchmark_*_test.go, tests/bench/test_check_phase8_improvement.py] |
| Config file | No dedicated Go test config; Python tests are direct files under `tests/bench/`. [VERIFIED: repo file scan] |
| Quick run command | `python3 tests/bench/test_check_benchmark_claims.py && go test ./... -run 'TestTierNComparatorsAgree|TestJSONTestSuiteOracle' -count=1` [VERIFIED: existing test patterns] |
| Full suite command | `go test ./... && cargo test -- --test-threads=1 && make verify-contract && python3 tests/bench/test_check_benchmark_claims.py` [VERIFIED: Makefile, Phase 8 notes] |

### Phase Requirements -> Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|--------------|
| BENCH-01 | Tier 1/2/3 benchmark rows are present and stable. [VERIFIED: benchmark_*_test.go] | smoke/unit | `go test ./... -run '^$' -list 'Benchmark(Tier1FullParse|Tier2Typed|Tier3SelectivePlaceholder)_'` | yes [VERIFIED: command run] |
| BENCH-02 | Fixture set used by public benchmark rows loads from committed testdata. [VERIFIED: benchmark_fixtures_test.go] | unit | `go test ./... -run TestTierNComparatorsAgree -count=1` | yes [VERIFIED: benchmark_comparators_test.go] |
| BENCH-03 | Comparator registry omits unsupported libraries structurally. [VERIFIED: benchmark_comparators_test.go] | unit | `go test ./... -run TestTierNComparatorsAgree -count=1` | yes [VERIFIED: benchmark_comparators_test.go] |
| BENCH-04 | Benchstat wrapper rejects missing files/tool and compares raw outputs. [VERIFIED: scripts/bench/run_benchstat.sh] | script smoke | `scripts/bench/run_benchstat.sh --old testdata/benchmark-results/v0.1.1/tier1-diagnostics.bench.txt --new testdata/benchmark-results/phase8/tier1-diagnostics.bench.txt` | yes [VERIFIED: command run] |
| BENCH-05 | Native allocation metrics remain in benchmark output. [VERIFIED: benchmark_native_alloc_test.go] | benchmark smoke | `go test ./... -run '^$' -bench 'BenchmarkTier1Diagnostics_twitter_json/pure-simdjson-full$' -benchmem -count=1` | yes [VERIFIED: benchmark_native_alloc_test.go] |
| BENCH-07 | Claim gate emits allowed README/docs claims and fails closed on unsupported/noisy Tier 1 headline. [VERIFIED: 09-CONTEXT.md] | unit/script | `python3 tests/bench/test_check_benchmark_claims.py` | no, Wave 0 [VERIFIED: file scan] |
| DOC-01 | README links methodology and uses gate-generated stdlib-relative benchmark wording. [VERIFIED: README.md, 09-CONTEXT.md] | doc grep + review | `rg 'docs/benchmarks.md|results-v0.1.2|encoding/json' README.md` after implementation | yes for README, no for v0.1.2 content yet [VERIFIED: README.md] |
| DOC-06 | CHANGELOG has `Unreleased` benchmark-positioning note. [VERIFIED: CHANGELOG.md, 09-CONTEXT.md] | doc grep | `rg 'benchmark|Tier 1|Tier 2|Tier 3' CHANGELOG.md` | yes [VERIFIED: CHANGELOG.md] |

### Sampling Rate

- **Per task commit:** Run the Python gate tests for script changes and targeted `go test` for benchmark-row or comparator changes. [VERIFIED: tests/bench/test_check_phase8_improvement.py, benchmark_*_test.go]
- **Per wave merge:** Run `go test ./...`, `cargo test -- --test-threads=1`, `make verify-contract`, and the new gate tests. [VERIFIED: Makefile, 08-BENCHMARK-NOTES.md]
- **Phase gate:** Dispatch or run the benchmark capture on real `linux/amd64`, commit raw/benchstat/summary outputs, and verify docs match `summary.json`. [VERIFIED: 09-CONTEXT.md]

### Wave 0 Gaps

- [ ] `scripts/bench/check_benchmark_claims.py` - generalized Tier 1/2/3 claim gate with machine-readable summary. [VERIFIED: missing by file scan]
- [ ] `tests/bench/test_check_benchmark_claims.py` - unit tests for metadata mismatch, missing rows, non-significant Tier 1, Tier 2/3 regression, claim allowance output, and malformed benchmark rows. [VERIFIED: missing by file scan]
- [ ] `.github/workflows/benchmark-capture.yml` or equivalent - dispatchable `linux/amd64` capture workflow. [VERIFIED: .github/workflows scan]
- [ ] Optional `scripts/bench/capture_release_snapshot.sh` - one command to run capture, benchstat, metadata, and gate locally or in CI. [ASSUMED]

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|------------------|
| V2 Authentication | no for benchmark scripts; yes only if optional GitHub Pages/comment automation uses tokens. [VERIFIED: 09-CONTEXT.md; CITED: https://github.com/benchmark-action/github-action-benchmark] | Prefer upload-only workflow with `contents: read`; require explicit approval before adding write tokens. [VERIFIED: existing workflows; ASSUMED: approval policy] |
| V3 Session Management | no [VERIFIED: phase scope has no user sessions] | Not applicable. [VERIFIED: 09-CONTEXT.md] |
| V4 Access Control | yes for workflow permissions. [VERIFIED: .github/workflows/release.yml, public-bootstrap-validation.yml] | Use least-privilege workflow permissions; benchmark capture can use `contents: read` and `actions: read` unless it writes Pages or commits. [VERIFIED: existing workflows; CITED: GitHub Pages custom workflow docs] |
| V5 Input Validation | yes [VERIFIED: workflow inputs and script CLI inputs] | Validate snapshot label as semver-like `v<major>.<minor>.<patch>` or explicit accepted label before using it in paths. [VERIFIED: public-bootstrap-validation.yml pattern] |
| V6 Cryptography | no for benchmark evidence itself; release artifact signing remains in `release.yml`. [VERIFIED: docs/releases.md, 09-CONTEXT.md] | Do not move cosign/signing concerns into Phase 9 benchmark capture. [VERIFIED: docs/releases.md] |

### Known Threat Patterns for This Stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Over-broad workflow token permissions for benchmark history/Pages | Elevation of privilege | Default benchmark-capture workflow to `contents: read`; add `pages: write` / `id-token: write` only if implementing GitHub Pages deployment. [CITED: https://docs.github.com/en/pages/getting-started-with-github-pages/using-custom-workflows-with-github-pages] |
| Unvalidated snapshot input used as a filesystem path | Tampering | Validate input against a strict snapshot regex and write only under `testdata/benchmark-results/<snapshot>/`. [VERIFIED: public-bootstrap-validation.yml pattern; ASSUMED: script implementation] |
| Public docs overclaiming unsupported performance | Repudiation / integrity | Generate docs from committed raw evidence and machine-readable gate output. [VERIFIED: 09-CONTEXT.md] |
| Artifact retention mistaken for durable audit evidence | Repudiation | Commit raw benchmark evidence and summaries to git; use workflow artifacts only as transport. [CITED: GitHub artifact retention docs; VERIFIED: 09-CONTEXT.md] |

## Sources

### Primary (HIGH confidence)

- `.planning/phases/09-benchmark-gate-recalibration-tier-1-2-3-positioning-and-post/09-CONTEXT.md` - Phase 9 locked decisions, claim gates, artifact shape, release boundary. [VERIFIED]
- `.planning/REQUIREMENTS.md` - BENCH-01/02/03/04/05/07 and DOC-01/DOC-06 requirement text. [VERIFIED]
- `.planning/ROADMAP.md`, `.planning/PROJECT.md`, `.planning/STATE.md` - phase boundary, current state, Phase 09.1 boundary. [VERIFIED]
- `.planning/phases/07-benchmarks-v0.1-release/07-CONTEXT.md` and `07-LEARNINGS.md` - benchmark tier definitions and prior lessons. [VERIFIED]
- `.planning/phases/08-low-overhead-dom-traversal-abi-and-specialized-go-any-materi/08-CONTEXT.md` and `08-BENCHMARK-NOTES.md` - Phase 8 internal evidence and handoff. [VERIFIED]
- `docs/benchmarks.md`, `docs/benchmarks/results-v0.1.1.md`, `docs/releases.md`, `README.md`, `CHANGELOG.md` - current public docs and release constraints. [VERIFIED]
- `benchmark_comparators_test.go`, `benchmark_diagnostics_test.go`, `benchmark_coldstart_test.go`, `benchmark_typed_test.go`, `benchmark_selective_test.go`, `benchmark_fullparse_test.go`, `benchmark_native_alloc_test.go` - benchmark families and metrics. [VERIFIED]
- `scripts/bench/run_benchstat.sh`, `scripts/bench/check_phase8_tier1_improvement.py`, `tests/bench/test_check_phase8_improvement.py` - existing comparison/gate patterns. [VERIFIED]
- `.github/workflows/release.yml`, `.github/workflows/public-bootstrap-validation.yml` - existing workflow and release/baseline permission patterns. [VERIFIED]
- `go help testflag`, `go doc testing.B`, `benchstat -h`, `go version -m $(command -v benchstat)` - local tool behavior and versions. [VERIFIED]

### Secondary (MEDIUM/HIGH confidence)

- GitHub Actions artifact docs: https://docs.github.com/en/actions/tutorials/store-and-share-data - upload/download artifacts and retention-days behavior. [CITED]
- GitHub artifact/log retention docs: https://docs.github.com/en/organizations/managing-organization-settings/configuring-the-retention-period-for-github-actions-artifacts-and-logs-in-your-organization - default and max retention limits. [CITED]
- GitHub Pages custom workflow docs: https://docs.github.com/en/pages/getting-started-with-github-pages/using-custom-workflows-with-github-pages - minimum Pages deployment permissions. [CITED]
- `benchmark-action/github-action-benchmark` README: https://github.com/benchmark-action/github-action-benchmark - Go benchmark support, comparisons, GitHub Pages chart behavior, and versioning. [CITED]
- `benchstat` docs: https://pkg.go.dev/golang.org/x/perf/cmd/benchstat - statistical comparison behavior and noise guidance. [CITED]

### Tertiary (LOW confidence)

- None used as authoritative source. [VERIFIED: research source log]

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - versions and availability were verified via local module/tool commands and Git remote tag checks. [VERIFIED]
- Architecture: HIGH - direct mapping from Phase 9 decisions and existing benchmark/workflow files. [VERIFIED]
- Pitfalls: HIGH - grounded in prior phase learnings, locked Phase 9 gates, and official benchstat/GitHub documentation. [VERIFIED/CITED]
- Optional GitHub Pages benchmark history: MEDIUM - official repo docs confirm capabilities, but project adoption depends on write/Pages permission decisions not yet made. [CITED/ASSUMED]

**Research date:** 2026-04-24 [VERIFIED: current_date prompt]
**Valid until:** 2026-05-01 for GitHub Actions/action-version recommendations; 2026-05-24 for local benchmark architecture and repo-specific patterns. [ASSUMED]
