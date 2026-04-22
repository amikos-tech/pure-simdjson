# Phase 7: Benchmarks + v0.1 Release - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in `07-CONTEXT.md` — this log preserves the alternatives considered.

**Date:** 2026-04-22
**Phase:** 7-Benchmarks + v0.1 Release
**Areas discussed:** Benchmark fairness and tier definitions

---

## Benchmark Fairness and Tier Definitions

### What should Tier 1 "full parse" mean for `pure-simdjson`?

| Option | Description | Selected |
|--------|-------------|----------|
| Full materialization parity | `pure-simdjson` must materialize an equivalent Go tree before timing is counted. | ✓ |
| DOM parse + full walk parity | Parse to DOM and fully walk the document without a second Go tree. | |
| No Tier 1 headline | Drop the full-parse headline and keep typed extraction as the main story. | |

**User's choice:** Full materialization parity
**Notes:** Keep the headline comparison defensible even if it understates the library's natural strengths.

### What should Tier 2 "typed extraction" measure?

| Option | Description | Selected |
|--------|-------------|----------|
| Schema-shaped end-to-end extraction | Decode a fixed set of fields with the current public API and compare against typed decoding in baseline libraries. | ✓ |
| Accessor microbenchmarks | Benchmark isolated accessor calls on pre-parsed docs. | |
| App-style query tasks | Use custom per-corpus tasks such as extracting tweet fields or summing coordinates. | |

**User's choice:** Schema-shaped end-to-end extraction
**Notes:** Show the library's intended use without inventing a new API surface.

### What should Tier 3 "selective-path placeholder for v0.2" be in `v0.1`?

| Option | Description | Selected |
|--------|-------------|----------|
| Runnable placeholder on current DOM API | Extract only the target fields using today's DOM API and label it as a `v0.2` placeholder. | ✓ |
| Methodology-only placeholder | Describe the future selective benchmark but do not ship runnable numbers. | |
| Experimental branch inside the harness | Include a separate experimental section excluded from headline tables. | |

**User's choice:** Runnable placeholder on current DOM API
**Notes:** Satisfy the three-tier harness requirement without pretending On-Demand already ships in `v0.1`.

### How strict should comparator symmetry be in the public benchmark tables?

| Option | Description | Selected |
|--------|-------------|----------|
| Only publish head-to-head tables where the comparator set is actually available | Unsupported comparators are omitted from that exact target/toolchain table. | ✓ |
| Publish one full table with `N/A` cells | Keep every library listed even when some rows are unsupported or skipped. | |
| Always force one common baseline set only | Restrict public tables to the smallest universally supported subset. | |

**User's choice:** Only publish head-to-head tables where the comparator set is actually available
**Notes:** Keep the tables honest for the exact target/toolchain combination being reported.

## the agent's Discretion

- Exact benchmark environment and publication format
- Exact release threshold/tag strategy
- Exact `README.md` framing and information architecture

## Deferred Ideas

- None explicitly raised during discussion
