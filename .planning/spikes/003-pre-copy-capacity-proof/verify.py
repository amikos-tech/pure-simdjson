#!/usr/bin/env python3
import json
import sys


def fail(message: str) -> None:
    raise SystemExit(f"verification failed: {message}")


json_lines = [line for line in sys.stdin if line.startswith("{")]
if len(json_lines) != 1:
    fail(f"expected one JSON result row, got {len(json_lines)}")

result = json.loads(json_lines[0])
if result.get("verified") is not True:
    fail("probe did not mark the result verified")
if result["exact_copied_bytes"] != result["exact_limit"]:
    fail("exact-limit input was not copied exactly once")
if result["rejected_bytes"] != result["exact_limit"] + 1:
    fail("rejected boundary was not limit+1")
for field in (
    "rejected_padding_checks",
    "rejected_resize_calls",
    "rejected_copy_calls",
    "pre_copy_gate_copied_bytes",
):
    if result[field] != 0:
        fail(f"{field} must be zero, got {result[field]}")
if result.get("rejected_arena_unchanged") is not True:
    fail("rejected input changed the reusable arena")
if result.get("rejected_diagnostics_cleared") is not True:
    fail("capacity rejection retained stale diagnostics")
if result["late_gate_copied_bytes"] != result["large_input_bytes"]:
    fail("late-gate baseline did not demonstrate the rejected copy")

print(
    json.dumps(
        {
            "verified": True,
            "tests": 4,
            "avoided_copy_bytes": result["late_gate_copied_bytes"],
            "pre_copy_mutations": 0,
            "stale_diagnostics_cleared": True,
        },
        sort_keys=True,
    )
)
