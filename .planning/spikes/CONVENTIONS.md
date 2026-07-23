# Spike Conventions

These conventions were established by Phase 11 spikes 001–003.

## Shape

- Keep each experiment self-contained under `.planning/spikes/NNN-name/`.
- Put the question, alternatives, run command, evidence, verdict, and implementation consequence in `README.md`.
- Use frontmatter with `spike`, `name`, `type`, `validates`, `verdict`, `related`, and `tags`.
- Keep the cross-spike status in `.planning/spikes/MANIFEST.md`.

## Isolation

- Spike code must not change production source, dependency pins, submodule gitlinks, versions, tags, or release state.
- Build generated sources, fixtures, libraries, and binaries in a `mktemp` directory and remove them on exit.
- When a dependency version matters, verify the exact commit or module version before running.
- Prefer the repository's pinned tool or dependency over a lookalike reimplementation.

## Evidence

- State the invariant as given/when/then before implementing the probe.
- Compare the risky/current order with the proposed order when the difference is otherwise easy to hide.
- Emit a small machine-readable result in addition to human-readable output.
- Provide one verifier command that fails on every contract violation.
- Include boundary, failure, stale-state, and false-positive/false-negative cases relevant to the decision.
- Repeat deterministic probes three times before marking them validated.

## Promotion

- Use `PENDING`, `VALIDATED`, `INVALIDATED`, or `INCONCLUSIVE` consistently in the README and manifest.
- Record surprises and rejected approaches, not only the final successful path.
- Promote the finding into phase research or an execution plan; do not copy spike code into production blindly.
- Commit each completed spike separately with only its directory and manifest row.
