# Phase 6: CI Release Matrix + Platform Coverage - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-20
**Phase:** 06-ci-release-matrix-platform-coverage
**Areas discussed:** Alpine/musl strategy, Alpine gate mode, release-state update flow, Linux build baseline, release gate depth, release-process guidance, sequencing with promoted Phase 06.1

---

## Alpine / musl strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Smoke-test only | Alpine validated via `PURE_SIMDJSON_LIB_PATH`; no shipped musl artifact in `v0.1` | ✓ |
| Ship musl static `.a` | Publish musl static archives as an escape hatch | |
| First-class musl runtime artifacts | Treat musl as a shipped runtime target | |

**User's choice:** Smoke-test only
**Notes:** Keeps the release set aligned with the five shipped runtime targets and avoids widening scope into musl distribution.

---

## Alpine release gate

| Option | Description | Selected |
|--------|-------------|----------|
| Hard gate | Release blocks if Alpine smoke fails | ✓ |
| Advisory only | Release publishes but Alpine result is reported | |
| Hard on tags only | Branch CI can be advisory; releases block | |

**User's choice:** Hard gate
**Notes:** The documented Alpine story should count as release quality, not post-hoc information.

---

## Release-state update flow

| Option | Description | Selected |
|--------|-------------|----------|
| Prep commit, then tag | Prepare release metadata in source control first, then tag/publish that exact commit | ✓ |
| Tag first, then follow-up PR | Generate/update checksums and metadata after the tag | |
| Hybrid retag flow | Generate metadata after first build, then retag/re-publish | |

**User's choice:** Prep commit, then tag
**Notes:** Tagged source should already be bootstrap-correct and release-ready.

---

## Linux build baseline

| Option | Description | Selected |
|--------|-------------|----------|
| Manylinux-style containers | Containerized Linux builds with explicit glibc proof; zig/cross only as fallback | ✓ |
| Zig/cross primary | Use zig/cross as the standard Linux build route | |
| Native runners | Build on GitHub runners and inspect afterwards | |

**User's choice:** Manylinux-style containers
**Notes:** `objdump` remains a required verification tool to prove glibc `<= 2.17`.

---

## Release gate depth

| Option | Description | Selected |
|--------|-------------|----------|
| Native artifact gate only | Native compile/smoke/sign/package checks block release | |
| Native + Go consumer verification | Block publish on native checks plus Go bootstrap/consumer verification using packaged artifacts | ✓ |
| Two-tier staged release | Publish to staging first, then gate public release on higher-level verification | |

**User's choice:** Native + Go consumer verification
**Notes:** The release gate should prove the shipped module actually boots and parses through the real consumer path.

---

## Release-process guidance

| Option | Description | Selected |
|--------|-------------|----------|
| Runbook only | In-repo Markdown guidance without enforcement | |
| Runbook + scriptable gate | In-repo runbook plus one scriptable readiness gate | ✓ |
| Repo-local skill + runbook | Guidance delivered as a runbook and repo-local agent skill | ✓ |
| Global reusable skill | Broader cross-repo skill created as part of this phase | |

**User's choice:** Runbook + scriptable gate, plus a repo-local skill
**Notes:** The user explicitly wants the release process guided and gated for both humans and agents. A global reusable skill was not chosen for this phase.

---

## Sequencing with promoted Phase 06.1

| Option | Description | Selected |
|--------|-------------|----------|
| Hard gate before final release | Phase `06.1` must pass before final `v0.1` release closeout | ✓ |
| Post-release validation | Run Phase `06.1` after final release | |
| Milestone-close gate only | Required before milestone close, but not before final tag | |

**User's choice:** Hard gate before final release
**Notes:** The user also clarified that "fresh machine" can be satisfied by a clean CI runner rather than only by a manually prepared workstation.

---

## the agent's Discretion

- Exact release workflow decomposition between workflow YAML, composite actions, and helper scripts
- Exact implementation shape of the Go-side packaged-artifact release smoke gate
- Exact file layout and naming of the runbook and repo-local release skill

## Deferred Ideas

- Global reusable release skill beyond this repository
- First-class musl runtime artifact distribution
