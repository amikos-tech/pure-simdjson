#!/usr/bin/env python3
import csv
import json
import sys
from pathlib import Path


def load_rows(path: Path) -> list[dict[str, str]]:
    lines = [
        line
        for line in path.read_text(encoding="utf-8").splitlines()
        if line and not line.startswith("#")
    ]
    return list(csv.DictReader(lines, delimiter="\t"))


def main() -> int:
    if len(sys.argv) != 2:
        raise SystemExit("usage: verify.py <observations.tsv>")

    rows = load_rows(Path(sys.argv[1]))
    by_key = {(row["case"], row["mode"]): row for row in rows}
    required = {
        "empty",
        "invalid_utf8",
        "unclosed_string",
        "array_trailing_comma",
        "trailing_content",
        "missing_object_key",
        "unexpected_root_token",
        "extra_closing_bracket",
        "mismatched_container",
    }
    required_keys = {
        (name, mode)
        for name in required
        for mode in ("raw_json", "recursive_walk", "hybrid")
    }
    missing = sorted(required_keys - by_key.keys())
    if missing:
        raise SystemExit(
            "missing probe cases: "
            + ", ".join(f"{name}/{mode}" for name, mode in missing)
        )

    violations: list[str] = []
    known_rows: list[dict[str, str]] = []
    for row in rows:
        is_known = row["known"] == "true"
        if is_known:
            known_rows.append(row)
            if row["pointer_relation"] != "in_bounds":
                violations.append(
                    f"{row['case']}: known offset without in-bounds pointer"
                )
            offset = int(row["offset"])
            if offset < 0 or offset >= int(row["bytes"]):
                violations.append(f"{row['case']}: known offset outside input")
        elif row["pointer_relation"] == "at_end" and row["offset"] != "-":
            violations.append(f"{row['case']}: end pointer exposed as offset")

    for name in ("empty", "invalid_utf8", "unclosed_string"):
        for mode in ("raw_json", "recursive_walk", "hybrid"):
            row = by_key[(name, mode)]
            if row["replay_outcome"] == "iterate_failed":
                if row["location_status"] != "not_queried":
                    violations.append(
                        f"{name}/{mode}: current_location queried without a valid document"
                    )
                if row["known"] != "false":
                    violations.append(f"{name}/{mode}: iterate failure marked known")

    known_by_mode = {
        mode: [
            {"case": row["case"], "offset": int(row["offset"])}
            for row in known_rows
            if row["mode"] == mode
        ]
        for mode in ("raw_json", "recursive_walk", "hybrid")
    }
    false_negatives_by_mode = {
        mode: [
            row["case"]
            for row in rows
            if row["mode"] == mode
            and row["dom_error"] != "SUCCESS"
            and row["replay_outcome"] == "valid"
        ]
        for mode in ("raw_json", "recursive_walk", "hybrid")
    }

    if false_negatives_by_mode["hybrid"]:
        violations.append(
            "hybrid replay missed DOM failures: "
            + ", ".join(false_negatives_by_mode["hybrid"])
        )
    trailing = by_key[("trailing_content", "hybrid")]
    if trailing["known"] != "true" or trailing["offset"] != "8":
        violations.append(
            "hybrid replay did not preserve upstream trailing-content offset 8"
        )

    summary = {
        "cases": len(rows),
        "dom_failures": sum(row["dom_error"] != "SUCCESS" for row in rows),
        "iterate_failures": sum(
            row["replay_outcome"] == "iterate_failed" for row in rows
        ),
        "known_offsets_by_mode": known_by_mode,
        "false_negatives_by_mode": false_negatives_by_mode,
        "unknown_rows": [
            f"{row['case']}/{row['mode']}"
            for row in rows
            if row["known"] == "false"
        ],
        "safety_violations": violations,
    }
    print(json.dumps(summary, indent=2, sort_keys=True))
    return 1 if violations else 0


if __name__ == "__main__":
    raise SystemExit(main())
