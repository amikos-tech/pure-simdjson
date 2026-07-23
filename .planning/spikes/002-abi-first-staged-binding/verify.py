#!/usr/bin/env python3
import json
import sys


def fail(message: str) -> None:
    raise SystemExit(f"verification failed: {message}")


rows = [json.loads(line) for line in sys.stdin if line.strip()]
if len(rows) != 6:
    fail(f"expected 6 result rows, got {len(rows)}")

by_key = {(row["fixture"], row["strategy"]): row for row in rows}
if len(by_key) != 6:
    fail("fixture/strategy rows are not unique")

abi11_naive = by_key[("abi11", "naive")]
if abi11_naive["outcome"] != "missing_symbol":
    fail("naive ABI 1.1 loading did not collapse to missing_symbol")
if "abi" in abi11_naive:
    fail("naive ABI 1.1 loading unexpectedly reached the ABI probe")

abi11_staged = by_key[("abi11", "staged")]
if abi11_staged["outcome"] != "abi_mismatch":
    fail("staged ABI 1.1 loading did not preserve abi_mismatch")
if abi11_staged["abi"] != "0x00010001":
    fail("staged ABI 1.1 loading reported the wrong ABI")
if abi11_staged["mandatory_lookups"] != 0:
    fail("staged ABI 1.1 loading touched mandatory ABI 1.2 symbols")

complete = by_key[("abi12_complete", "staged")]
if complete["outcome"] != "ok":
    fail("complete ABI 1.2 surface did not load")
if complete["mandatory_lookups"] != 5:
    fail("complete ABI 1.2 surface did not resolve every mandatory symbol")

missing_naive = by_key[("abi12_missing", "naive")]
if missing_naive["outcome"] != "missing_symbol":
    fail("naive incomplete ABI 1.2 loading did not report missing_symbol")

missing_staged = by_key[("abi12_missing", "staged")]
if missing_staged["outcome"] != "corrupt_abi12":
    fail("staged incomplete ABI 1.2 loading did not report corrupt_abi12")
if missing_staged.get("missing") != "pure_simdjson_element_get_bigint":
    fail("staged incomplete ABI 1.2 loading did not identify the omitted symbol")
if missing_staged.get("abi") != "0x00010002":
    fail("staged incomplete ABI 1.2 loading did not probe ABI first")

for row in rows:
    if not row["trace"]:
        fail(f"{row['fixture']}/{row['strategy']} has no lookup trace")
    if row["strategy"] == "staged" and row["trace"][0] != (
        "lookup:pure_simdjson_get_abi_version"
    ):
        fail(f"{row['fixture']}/staged did not look up the ABI probe first")

print(
    json.dumps(
        {
            "verified": True,
            "rows": len(rows),
            "abi11_mandatory_lookups_before_mismatch": abi11_staged[
                "mandatory_lookups"
            ],
            "incomplete_abi12_missing": missing_staged["missing"],
        },
        sort_keys=True,
    )
)
