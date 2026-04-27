---
phase: 9
phase_name: "Benchmark gate recalibration, Tier 1/2/3 positioning, and post-ABI evidence refresh"
project: "pure-simdjson"
generated: "2026-04-24"
counts:
  decisions: 5
  lessons: 5
  patterns: 5
  surprises: 4
missing_artifacts:
  - "09-VERIFICATION.md"
---

# Phase 9 Learnings: Benchmark gate recalibration, Tier 1/2/3 positioning, and post-ABI evidence refresh

## Decisions

### Gate Public Benchmark Wording from Committed Evidence
Phase 9 made benchmark claim allowances a generated artifact derived from committed benchmark evidence instead of hand-written documentation judgment.

**Rationale:** This makes unsupported benchmark wording mechanically difficult and forces README/docs copy to follow the measured snapshot.
**Source:** 09-01-PLAN.md

---

### Use linux/amd64 CI Evidence as the Durable Public Baseline
The phase standardized on linux/amd64 GitHub Actions evidence as the canonical baseline and snapshot target for public benchmark comparisons.

**Rationale:** Future gates run on GitHub Actions, so the public comparison target needed to match the environment where repeatable evidence is captured.
**Source:** 09-02-SUMMARY.md

---

### Keep Noisy Tier 1 Results Publishable Only Through Conservative Modes
Phase 9 allowed `tier1_headline`, `tier1_improved_but_tier2_tier3_headline`, and `conservative_current_strengths` as the only publishable claim modes.

**Rationale:** Complete evidence should still unlock truthful docs, but noisy or non-significant Tier 1 runs must fail closed rather than being turned into unsupported headline language.
**Source:** 09-01-PLAN.md

---

### Keep Full Comparator Tables Out of README
README benchmark copy was intentionally limited to stdlib-relative framing, while full comparator tables were pushed into the release-scoped benchmark result document.

**Rationale:** The README should stay bounded to the claim-gated story, while detailed competitor context belongs in the evidence-backed docs page.
**Source:** 09-03-PLAN.md

---

### Preserve the Phase 09.1 Release Boundary
Phase 9 explicitly stopped at evidence, docs, and changelog updates and left bootstrap artifact/default-install alignment to Phase 09.1.

**Rationale:** Benchmark evidence alone is not enough to justify tagging or release claims while the default bootstrap path still points at older published artifacts.
**Source:** 09-03-SUMMARY.md

---

## Lessons

### Same-Target Baselines Matter More Than Historical Convenience
The old darwin/arm64 `v0.1.1` evidence could not serve as a trustworthy gate for the new linux/amd64 snapshot.

**Context:** The phase had to capture a new pre-Phase-8 linux/amd64 baseline before the claim gate could compare old and new results honestly.
**Source:** 09-02-SUMMARY.md

---

### Claim Gates Need to Understand Real benchstat Output, Not Idealized Fixtures
The initial claim-gate logic needed follow-up fixes because real benchstat output omitted the `Benchmark` prefix used in synthetic assumptions.

**Context:** The gate had to be tightened against actual captured files before `summary.json` could be trusted as a public input.
**Source:** 09-02-SUMMARY.md

---

### Metadata Validation Is Part of the Benchmark Contract
The benchmark gate had to verify `goos`, `goarch`, `pkg`, and `cpu` from both raw benchmark files and `metadata.json`.

**Context:** Without target and metadata agreement checks, cross-target evidence could masquerade as valid old/new benchmark proof.
**Source:** 09-01-PLAN.md

---

### Public Benchmark Docs Need Their Own Contract Tests
The phase validation audit added dedicated doc/workflow/evidence contract checks instead of relying only on functional benchmark tests.

**Context:** Phase 9 touched workflow configuration, committed evidence, and public benchmark wording, so correctness depended on more than runtime tests.
**Source:** 09-VALIDATION.md

---

### Complete Evidence Can Still Lead to Conservative Copy
Passing evidence does not automatically mean the most aggressive benchmark headline is justified.

**Context:** The workflow explicitly treats conservative publishable modes as valid outcomes when the evidence is complete but some public headline allowance is false or noisy.
**Source:** 09-01-PLAN.md

---

## Patterns

### Machine-Readable Claim Summary
Generate a `summary.json` file with fixed top-level keys for snapshot, target, thresholds, claims, fixtures, and errors.

**When to use:** Use when public wording, docs, or release notes must be driven from a stable machine-readable interpretation of benchmark evidence.
**Source:** 09-01-PLAN.md

---

### Tested Same-Snapshot Normalization Helper
Normalize benchmark inputs for stdlib comparisons with a dedicated helper script instead of ad hoc shell rewriting.

**When to use:** Use when two benchmark inputs need deterministic name normalization so benchstat compares like-for-like rows without fragile text munging.
**Source:** 09-01-SUMMARY.md

---

### Workflow Artifact as Transport, Repository Snapshot as Source of Truth
Capture benchmark results in GitHub Actions, but treat uploaded workflow artifacts as temporary transport only and commit the durable evidence under `testdata/benchmark-results/`.

**When to use:** Use when CI is the right execution target, but long-lived evidence must survive artifact retention windows and support later audits.
**Source:** 09-01-SUMMARY.md

---

### Upload Evidence Even on Gate Failure
The benchmark capture path preserves a complete snapshot even when the claim gate exits nonzero.

**When to use:** Use when failed benchmark gates still produce valuable diagnostic evidence that should not be discarded.
**Source:** 09-02-PLAN.md

---

### Release-Scoped Benchmark Documents
Publish benchmark result pages under versioned names like `results-v0.1.2.md` instead of phase-scoped documents.

**When to use:** Use when benchmark evidence is intended to describe a release candidate snapshot rather than an internal implementation phase.
**Source:** 09-03-PLAN.md

---

## Surprises

### The First Claim Gate Misclassified Noisy Tier 1 Evidence
Early claim-mode selection mapped noisy Tier 1 evidence to the wrong publishable mode.

**Impact:** The gate had to be tightened before it could safely control README and docs wording.
**Source:** 09-01-SUMMARY.md

---

### Real linux/amd64 Capture Required More Than One Workflow Run
Phase 9 needed both a new snapshot run and a same-target pre-Phase-8 baseline capture before the gate could pass.

**Impact:** The evidence pipeline was more operationally involved than a single benchmark capture, but it produced a defensible old/new comparison.
**Source:** 09-02-SUMMARY.md

---

### Human Review Still Matters After the Gate Passes
The validation contract kept manual-only checks for provenance, public wording quality, and release-boundary discipline even after automated checks were green.

**Impact:** The phase reinforced that machine gating is necessary but not sufficient for public benchmark claims.
**Source:** 09-VALIDATION.md

---

### Phase 9 Finished the Benchmark Story but Not the Release Story
The phase completed evidence capture and benchmark-positioning updates, yet the project state still had to queue Phase 09.1 immediately afterward.

**Impact:** The work clarified that benchmark proof and default-install readiness are separate delivery milestones.
**Source:** STATE.md
