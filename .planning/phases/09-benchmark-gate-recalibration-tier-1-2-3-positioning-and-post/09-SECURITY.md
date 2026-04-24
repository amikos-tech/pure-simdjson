---
phase: 09
slug: benchmark-gate-recalibration-tier-1-2-3-positioning-and-post
status: verified
threats_open: 0
threats_total: 15
threats_closed: 15
asvs_level: 1
created: 2026-04-24
---

# Phase 09 — Security

> Per-phase security contract: threat register, accepted risks, threat flags, and audit trail.
> Consolidated from the three plan-level threat models (`09-01` through `09-03`) and verified against the executed implementation plus the committed linux/amd64 benchmark evidence.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| raw benchmark files -> claim gate | Untrusted benchmark output is parsed into machine claim allowances and publishable status flags. | `phase9.bench.txt`, `coldwarm.bench.txt`, `tier1-diagnostics.bench.txt`, benchstat outputs |
| snapshot CLI input -> filesystem path | Snapshot labels and output paths decide where evidence is staged and promoted. | `--snapshot`, `--out-dir`, staged directory paths |
| workflow runner -> repository snapshot | CI-run evidence is transient until promoted into committed repository state. | raw benchmark output, `metadata.json`, `summary.json`, benchstat outputs |
| workflow token -> repository | The capture workflow must not have write scopes beyond reading contents and uploading artifacts. | GitHub Actions token permissions |
| GitHub Actions artifact -> local repository | Imported benchmark artifacts become durable evidence only after local verification and commit. | workflow artifact bundle, imported benchmark files |
| local host -> public claim evidence | Local machines may not satisfy the required public benchmark target. | target metadata, runner metadata, toolchain versions |
| `summary.json` -> public docs | Machine claim allowances and target facts become public-facing wording. | claim booleans, `readme_mode`, target metadata |
| benchmark docs -> release expectations | Benchmark evidence can be mistaken for release publication or default-install readiness if the boundary is unclear. | README copy, benchmark result docs, release guidance |

---

## Threat Register

| Threat ID | Category | Component | Disposition | Mitigation | Status |
|-----------|----------|-----------|-------------|------------|--------|
| T-09-01 | Tampering/Repudiation | `scripts/bench/check_benchmark_claims.py` | mitigate | Validate required files, required rows, metadata, target, and benchstat evidence; emit machine-readable `errors` and fail closed on malformed or incomplete evidence. | closed |
| T-09-02 | Tampering | `scripts/bench/capture_release_snapshot.sh` | mitigate | Reject invalid snapshot labels, stage into temp directories, preserve complete failed snapshots for diagnosis, and use the canonical benchmark commands. | closed |
| T-09-03 | Information Disclosure/Tampering | `.github/workflows/benchmark-capture.yml` token scope | mitigate | Limit permissions to `contents: read` and `actions: read`; no write, release, Pages, or `id-token` scopes. | closed |
| T-09-04 | Repudiation | GitHub Actions artifacts | mitigate | Upload 30-day artifacts as temporary transport only and require committed `testdata/benchmark-results/v0.1.2/` evidence. | closed |
| T-09-05 | Spoofing | target metadata | mitigate | Require `--require-target linux/amd64`, `metadata.json`, and raw benchmark metadata agreement before summary generation. | closed |
| T-09-06 | Spoofing | `metadata.json` and raw benchmark metadata | mitigate | Require `goos=linux`, `goarch=amd64`, raw metadata agreement, and a provenance gate before docs. | closed |
| T-09-07 | Tampering | imported benchmark artifacts | mitigate | Import only expected artifact files and rerun `check_benchmark_claims.py` locally after import. | closed |
| T-09-08 | Repudiation | benchmark run source | mitigate | Commit `commit`, runner metadata, toolchain versions, commands, raw output, benchstat output, and summary in the snapshot directory. | closed |
| T-09-09 | Information Disclosure | workflow artifact retention | mitigate | Treat Actions artifacts as retention-limited transport and keep committed `testdata/benchmark-results/v0.1.2/` as the durable source. | closed |
| T-09-10 | Tampering | public docs sequencing | mitigate | Keep docs work dependent on the evidence phase and only proceed after `summary.json` passes. | closed |
| T-09-11 | Tampering/Repudiation | `docs/benchmarks/results-v0.1.2.md` | mitigate | Derive public status and tables from committed raw files and `summary.json`; rerun the claim gate during verification. | closed |
| T-09-12 | Information Disclosure/Repudiation | README benchmark copy | mitigate | Keep README stdlib-relative only and send full comparator detail to the benchmark docs. | closed |
| T-09-13 | Spoofing | platform claims | mitigate | State the linux/amd64 source explicitly and warn that other platforms may differ. | closed |
| T-09-14 | Elevation of Privilege | release recommendation wording | mitigate | Keep any release recommendation gated by `docs/releases.md`, the strict readiness check, and the `origin/main` tag anchor; do not tag or push from this phase. | closed |
| T-09-15 | Tampering | artifact durability language | mitigate | Explain Actions artifacts as transport-only and committed evidence as durable source-of-truth. | closed |

*Status: open · closed*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Threat Flags

No explicit `## Threat Flags` sections were present in the Phase 09 summary files. The three Phase 09 plan-level threat models, the committed evidence set, and the recorded verification commands were the source of truth for this audit.

---

## Verification Notes

- `T-09-01` is closed by [scripts/bench/check_benchmark_claims.py](/Users/tazarov/experiments/amikos/pure-simdjson/scripts/bench/check_benchmark_claims.py:14), [scripts/bench/check_benchmark_claims.py](/Users/tazarov/experiments/amikos/pure-simdjson/scripts/bench/check_benchmark_claims.py:67), [scripts/bench/check_benchmark_claims.py](/Users/tazarov/experiments/amikos/pure-simdjson/scripts/bench/check_benchmark_claims.py:112), [scripts/bench/check_benchmark_claims.py](/Users/tazarov/experiments/amikos/pure-simdjson/scripts/bench/check_benchmark_claims.py:172), and [scripts/bench/check_benchmark_claims.py](/Users/tazarov/experiments/amikos/pure-simdjson/scripts/bench/check_benchmark_claims.py:230), which lock the required file set, CLI contract, required benchmark rows, raw-vs-JSON metadata agreement, and fail-closed `errors` flow.
- `T-09-02` is closed by [scripts/bench/capture_release_snapshot.sh](/Users/tazarov/experiments/amikos/pure-simdjson/scripts/bench/capture_release_snapshot.sh:54), [scripts/bench/capture_release_snapshot.sh](/Users/tazarov/experiments/amikos/pure-simdjson/scripts/bench/capture_release_snapshot.sh:67), [scripts/bench/capture_release_snapshot.sh](/Users/tazarov/experiments/amikos/pure-simdjson/scripts/bench/capture_release_snapshot.sh:95), and [scripts/bench/capture_release_snapshot.sh](/Users/tazarov/experiments/amikos/pure-simdjson/scripts/bench/capture_release_snapshot.sh:105), which reject malformed snapshot labels, stage to `mktemp`, promote atomically, and preserve the canonical benchmark commands verbatim.
- `T-09-03` is closed by [.github/workflows/benchmark-capture.yml](/Users/tazarov/experiments/amikos/pure-simdjson/.github/workflows/benchmark-capture.yml:16), which limits workflow permissions to `contents: read` and `actions: read` and does not request broader scopes.
- `T-09-04` is closed by [.github/workflows/benchmark-capture.yml](/Users/tazarov/experiments/amikos/pure-simdjson/.github/workflows/benchmark-capture.yml:59), [.github/workflows/benchmark-capture.yml](/Users/tazarov/experiments/amikos/pure-simdjson/.github/workflows/benchmark-capture.yml:66), and the committed snapshot files under [testdata/benchmark-results/v0.1.2](/Users/tazarov/experiments/amikos/pure-simdjson/testdata/benchmark-results/v0.1.2), which keep artifacts temporary and durable evidence in-repo.
- `T-09-05` is closed by [scripts/bench/check_benchmark_claims.py](/Users/tazarov/experiments/amikos/pure-simdjson/scripts/bench/check_benchmark_claims.py:45), [scripts/bench/check_benchmark_claims.py](/Users/tazarov/experiments/amikos/pure-simdjson/scripts/bench/check_benchmark_claims.py:230), and [testdata/benchmark-results/v0.1.2/summary.json](/Users/tazarov/experiments/amikos/pure-simdjson/testdata/benchmark-results/v0.1.2/summary.json:1), which require `linux/amd64` and emit a clean summary only when metadata aligns.
- `T-09-06` is closed by [testdata/benchmark-results/v0.1.2/metadata.json](/Users/tazarov/experiments/amikos/pure-simdjson/testdata/benchmark-results/v0.1.2/metadata.json:1), [testdata/benchmark-results/v0.1.2/summary.json](/Users/tazarov/experiments/amikos/pure-simdjson/testdata/benchmark-results/v0.1.2/summary.json:1), and [.planning/phases/09-benchmark-gate-recalibration-tier-1-2-3-positioning-and-post/09-02-SUMMARY.md](/Users/tazarov/experiments/amikos/pure-simdjson/.planning/phases/09-benchmark-gate-recalibration-tier-1-2-3-positioning-and-post/09-02-SUMMARY.md:59), which show the target is `linux/amd64`, the claim gate passes, and the docs plan was unblocked only after provenance checks.
- `T-09-07` is closed by [.planning/phases/09-benchmark-gate-recalibration-tier-1-2-3-positioning-and-post/09-02-SUMMARY.md](/Users/tazarov/experiments/amikos/pure-simdjson/.planning/phases/09-benchmark-gate-recalibration-tier-1-2-3-positioning-and-post/09-02-SUMMARY.md:34), [.planning/phases/09-benchmark-gate-recalibration-tier-1-2-3-positioning-and-post/09-02-SUMMARY.md](/Users/tazarov/experiments/amikos/pure-simdjson/.planning/phases/09-benchmark-gate-recalibration-tier-1-2-3-positioning-and-post/09-02-SUMMARY.md:35), and the rerun of `python3 scripts/bench/check_benchmark_claims.py --baseline-dir testdata/benchmark-results/v0.1.1-linux-amd64 --snapshot-dir testdata/benchmark-results/v0.1.2 --snapshot v0.1.2 --require-target linux/amd64`, which passes on current HEAD.
- `T-09-08` is closed by [scripts/bench/capture_release_snapshot.sh](/Users/tazarov/experiments/amikos/pure-simdjson/scripts/bench/capture_release_snapshot.sh:145), [testdata/benchmark-results/v0.1.2/metadata.json](/Users/tazarov/experiments/amikos/pure-simdjson/testdata/benchmark-results/v0.1.2/metadata.json:1), and the committed benchmark files under [testdata/benchmark-results/v0.1.2](/Users/tazarov/experiments/amikos/pure-simdjson/testdata/benchmark-results/v0.1.2), which preserve commands, toolchain versions, runner metadata, commit, raw outputs, benchstat outputs, and `summary.json`.
- `T-09-09` is closed by [.github/workflows/benchmark-capture.yml](/Users/tazarov/experiments/amikos/pure-simdjson/.github/workflows/benchmark-capture.yml:61), [.github/workflows/benchmark-capture.yml](/Users/tazarov/experiments/amikos/pure-simdjson/.github/workflows/benchmark-capture.yml:66), and [docs/benchmarks.md](/Users/tazarov/experiments/amikos/pure-simdjson/docs/benchmarks.md:3), which explicitly define workflow artifacts as retention-limited transport and the committed benchmark directory as the durable source.
- `T-09-10` is closed by [.planning/phases/09-benchmark-gate-recalibration-tier-1-2-3-positioning-and-post/09-03-PLAN.md](/Users/tazarov/experiments/amikos/pure-simdjson/.planning/phases/09-benchmark-gate-recalibration-tier-1-2-3-positioning-and-post/09-03-PLAN.md:5) and [.planning/phases/09-benchmark-gate-recalibration-tier-1-2-3-positioning-and-post/09-02-SUMMARY.md](/Users/tazarov/experiments/amikos/pure-simdjson/.planning/phases/09-benchmark-gate-recalibration-tier-1-2-3-positioning-and-post/09-02-SUMMARY.md:52), which keep Plan `09-03` dependent on `09-02` and record that docs only proceeded after the evidence summary passed.
- `T-09-11` is closed by [docs/benchmarks/results-v0.1.2.md](/Users/tazarov/experiments/amikos/pure-simdjson/docs/benchmarks/results-v0.1.2.md:5), [docs/benchmarks/results-v0.1.2.md](/Users/tazarov/experiments/amikos/pure-simdjson/docs/benchmarks/results-v0.1.2.md:24), [docs/benchmarks/results-v0.1.2.md](/Users/tazarov/experiments/amikos/pure-simdjson/docs/benchmarks/results-v0.1.2.md:38), and [.planning/phases/09-benchmark-gate-recalibration-tier-1-2-3-positioning-and-post/09-03-SUMMARY.md](/Users/tazarov/experiments/amikos/pure-simdjson/.planning/phases/09-benchmark-gate-recalibration-tier-1-2-3-positioning-and-post/09-03-SUMMARY.md:29), which link the committed evidence directly and record the claim-gate rerun during verification.
- `T-09-12` is closed by [README.md](/Users/tazarov/experiments/amikos/pure-simdjson/README.md:63), [README.md](/Users/tazarov/experiments/amikos/pure-simdjson/README.md:65), and [docs/benchmarks/results-v0.1.2.md](/Users/tazarov/experiments/amikos/pure-simdjson/docs/benchmarks/results-v0.1.2.md:144), which keep README benchmark wording stdlib-relative and push full comparator tables into the result document.
- `T-09-13` is closed by [README.md](/Users/tazarov/experiments/amikos/pure-simdjson/README.md:65) and [docs/benchmarks/results-v0.1.2.md](/Users/tazarov/experiments/amikos/pure-simdjson/docs/benchmarks/results-v0.1.2.md:13), which explicitly scope the published ratios to `linux/amd64` and warn that other platforms may differ.
- `T-09-14` is closed by [docs/benchmarks/results-v0.1.2.md](/Users/tazarov/experiments/amikos/pure-simdjson/docs/benchmarks/results-v0.1.2.md:148), [README.md](/Users/tazarov/experiments/amikos/pure-simdjson/README.md:67), and [.planning/phases/09-benchmark-gate-recalibration-tier-1-2-3-positioning-and-post/09-03-SUMMARY.md](/Users/tazarov/experiments/amikos/pure-simdjson/.planning/phases/09-benchmark-gate-recalibration-tier-1-2-3-positioning-and-post/09-03-SUMMARY.md:31), which retain the Phase `09.1` boundary, point later release work to the readiness gate, and confirm that no tag or publication action happened in Phase 09.
- `T-09-15` is closed by [docs/benchmarks.md](/Users/tazarov/experiments/amikos/pure-simdjson/docs/benchmarks.md:3), [docs/benchmarks/results-v0.1.2.md](/Users/tazarov/experiments/amikos/pure-simdjson/docs/benchmarks/results-v0.1.2.md:24), and [CHANGELOG.md](/Users/tazarov/experiments/amikos/pure-simdjson/CHANGELOG.md:14), which consistently describe committed benchmark files as the durable source and treat workflow artifacts as transport only.
- Fresh local verification on 2026-04-24 passed on current HEAD: `python3 scripts/bench/check_benchmark_claims.py --baseline-dir testdata/benchmark-results/v0.1.1-linux-amd64 --snapshot-dir testdata/benchmark-results/v0.1.2 --snapshot v0.1.2 --require-target linux/amd64`, `go test ./...`, `cargo test -- --test-threads=1`, `make verify-contract`, and `python3 tests/bench/test_check_benchmark_claims.py`.

---

## Accepted Risks Log

No accepted risks.

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-04-24 | 15 | 15 | 0 | Codex `/gsd-secure-phase` re-audit (State A — existing security file re-verified on current HEAD) |
| 2026-04-24 | 15 | 15 | 0 | Codex `/gsd-secure-phase` manual parity audit (State B — created from artifacts) |

### 2026-04-24 — Re-audit on current HEAD

- Input state: **A** (existing `09-SECURITY.md` present alongside the executed `*-PLAN.md` and `*-SUMMARY.md` files).
- Re-verified the existing 15-threat register against current HEAD `02a1591972db8df618776cc587f91322daa12828`; all threat dispositions remain closed and no accepted-risk or transfer entries were required.
- Re-ran the benchmark claim gate on the committed `v0.1.2` evidence with `python3 scripts/bench/check_benchmark_claims.py --baseline-dir testdata/benchmark-results/v0.1.1-linux-amd64 --snapshot-dir testdata/benchmark-results/v0.1.2 --snapshot v0.1.2 --require-target linux/amd64`; the command exited `0`, emitted an empty `errors` array, and kept `tier1_headline_allowed`, `tier2_headline_allowed`, and `tier3_headline_allowed` all `true`.
- Re-ran the verification command set on current HEAD: `go test ./...`, `cargo test -- --test-threads=1`, `make verify-contract`, and `python3 tests/bench/test_check_benchmark_claims.py`; all passed.
- Re-confirmed the Phase 09 security-sensitive controls remain intact: workflow permissions stay read-only, artifact retention remains 30 days, durable evidence still lives under `testdata/benchmark-results/v0.1.2/`, linux/amd64 caveats remain in README and benchmark docs, and Phase 09 still performs no tag, push, or release-publication action.
- Result: `## SECURED` — 15/15 threats remain closed, `threats_open: 0`, no accepted risks, no escalation required.

### 2026-04-24 — Initial audit

- Input state: **B** (no prior `09-SECURITY.md`; three `*-PLAN.md` files and three `*-SUMMARY.md` files present).
- Consolidated 15 unique plan-level threats from `09-01`, `09-02`, and `09-03`.
- Re-ran the committed claim gate against the linux/amd64 baseline and current `v0.1.2` evidence; the command exited `0` and produced an empty `errors` array with all three claim booleans `true`.
- Re-ran the Phase 09 verification command set on current HEAD: `go test ./...`, `cargo test -- --test-threads=1`, `make verify-contract`, and `python3 tests/bench/test_check_benchmark_claims.py` all passed.
- Verified the security-sensitive workflow and docs constraints remained intact: read-only workflow token scope, 30-day artifact retention, committed evidence under `testdata/benchmark-results/v0.1.2/`, linux/amd64 caveats in README/results docs, and no tag/push or release publication performed in this phase.
- Result: `## SECURED` — 15/15 threats closed, no accepted risks, no open threats, no escalation required.

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-04-24
