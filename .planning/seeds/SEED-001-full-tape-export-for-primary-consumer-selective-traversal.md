---
id: SEED-001
status: dormant
planted: 2026-04-23T17:01:52Z
planted_during: Phase 08 - low-overhead-dom-traversal-abi-and-specialized-go-any-materializer
trigger_when: "ami-gin or another primary consumer needs co-designed selective traversal, path extraction, or v0.2 On-Demand/tape-walker performance beyond the internal frame-stream materializer"
scope: Large
---

# SEED-001: Full Tape Export for Primary-Consumer Selective Traversal

## Why This Matters

Phase 8 intentionally chooses a repo-owned internal frame stream instead of exposing raw tape. That is the right v0.1 tradeoff for a mixed-consumer library: it removes per-node FFI for full `any` materialization while keeping simdjson tape details private and preserving public ABI flexibility.

The full tape-export idea remains valuable if the product direction narrows around a primary consumer such as `ami-gin`, where pure-simdjson and the consumer can co-design the data format. In that case, exposing a stable repo-owned tape-like view to Go could minimize FFI further and support flexible selective traversal, path extraction, and custom indexing without forcing every use case through full Go tree materialization.

## When to Surface

**Trigger:** `ami-gin` or another primary consumer needs co-designed selective traversal, path extraction, or v0.2 On-Demand/tape-walker performance beyond the internal frame-stream materializer.

This seed should be presented during `$gsd-new-milestone` when the milestone scope matches any of these conditions:
- v0.2 On-Demand planning starts, especially `Parser.ParseOnDemand`, JSON Pointer, JSONPath subset, or selective-path extraction.
- `ami-gin` becomes the primary benchmark/consumer and needs a lower-level traversal contract rather than generic DOM or `any` materialization.
- Phase 8 frame-stream results show that frame construction still costs too much, or the next bottleneck is consumer-specific selective traversal rather than full materialization.
- A future milestone is willing to define and own a repo-specific tape format across Rust/C++ and Go, including versioning, lifetime, numeric semantics, and string arena ownership.

## Scope Estimate

**Large** - likely a full milestone. This is not just an optimization patch. It would define a new internal or public-adjacent data contract, Go-side tape walker, safety model, compatibility/versioning rules, benchmarks, and consumer integration tests.

## Breadcrumbs

Related code and decisions found in the current codebase:

- `.planning/PROJECT.md` - Documents `ami-gin` as the immediate consumer, v0.2 On-Demand as future scope, and selective-path extraction as a core performance story.
- `.planning/REQUIREMENTS.md` - Tracks v0.2 On-Demand requirements `OD-01` through `OD-04`, including path-set parsing, JSON Pointer, JSONPath subset, and single-consumption semantics.
- `.planning/ROADMAP.md` - Phase 8 targets frame-stream DOM materialization now; Phase 9 owns benchmark recalibration after Phase 8 lands.
- `.planning/STATE.md` - Records that Phase 8 owns low-overhead traversal/materialization follow-up after Phase 7 found materialization dominates the current Tier 1 path.
- `.planning/phases/08-low-overhead-dom-traversal-abi-and-specialized-go-any-materi/08-CONTEXT.md` - Decision D-02 treats tape exposure as a research vector, not a Phase 8 mandate; deferred ideas keep JSONPointer/path lookup out of Phase 8 unless required.
- `.planning/phases/08-low-overhead-dom-traversal-abi-and-specialized-go-any-materi/08-RESEARCH.md` - Explicitly compares raw tape exposure with repo-owned frame stream and chooses frame stream for Phase 8 to avoid simdjson internal layout coupling.
- `.planning/phases/07-benchmarks-v0.1-release/07-LEARNINGS.md` - Captures the materialization-dominates-parse finding and notes that Tier 3 selective benchmarks remain a DOM-era placeholder, not a shipped On-Demand API.
- `docs/benchmarks.md` - Defines Tier 1 full `any` materialization, Tier 2 typed extraction, and Tier 3 selective placeholder benchmark interpretation.
- `third_party/simdjson/doc/tape.md` - Upstream tape representation reference.
- `third_party/simdjson/doc/ondemand_design.md` - Upstream On-Demand design reference.
- `third_party/simdjson/doc/performance.md` - Upstream performance context for parsing and traversal decisions.

## Notes

The concrete idea is a full tape-style export, for example exposing a repo-owned equivalent of `(*u64, tape_len, *u8, string_arena_len)` to Go and letting Go walk it natively. That could mean raw simdjson tape, a normalized pure-simdjson tape, or a hybrid that preserves only the fields this project needs.

Do not treat this seed as a commitment to expose upstream simdjson internals directly. Before planning, re-evaluate:
- Whether the Phase 8 frame stream delivered enough benefit for generic full materialization.
- Whether `ami-gin` needs selective traversal/indexing that cannot be served cleanly through DOM accessors, frame streams, or v0.2 On-Demand APIs.
- Whether the tape format can be versioned and tested across all five supported platforms without leaking unstable upstream details into the public ABI.
- Whether borrowed string/tape lifetime rules can remain explicit enough for Go users and internal consumers.
