---
name: pure-simdjson-release
description: Repo-local release guidance for pure-simdjson. Use only for this repository's tag-driven CI release path.
---

# pure-simdjson release

This skill is narrow and repo-local. Use it only when working on release
operations for `pure-simdjson`.

## Required flow

1. Read `docs/releases.md` before suggesting any release action.
2. Run `bash scripts/release/check_readiness.sh --strict --version <semver-without-v>` before recommending a tag push.
3. Treat `main -> annotated tag -> CI publish` as the supported sequencing.
4. State explicitly that `release.yml` expects the tag commit to be anchored on `origin/main`.
5. Treat Phase `06.1` as the place for post-publish fresh-runner validation.

## Constraints

- CI is the only publish path.
- Do not hand-upload artifacts.
- Do not bypass CI publication.
- Do not reintroduce prep-branch checksum generation as a release dependency.
- Do not invent a generic multi-project release abstraction from this skill.
- If the runbook and a remembered procedure disagree, follow `docs/releases.md`.
