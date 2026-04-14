#!/usr/bin/env python3

from __future__ import annotations

import importlib.util
import pathlib
import unittest


REPO_ROOT = pathlib.Path(__file__).resolve().parents[2]
CHECK_HEADER_PATH = REPO_ROOT / "tests" / "abi" / "check_header.py"

SPEC = importlib.util.spec_from_file_location("check_header", CHECK_HEADER_PATH)
assert SPEC is not None and SPEC.loader is not None
check_header = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(check_header)


def make_required_symbols_header(*, extra_symbols: list[str] | None = None) -> str:
    prototypes = [f"int32_t {symbol}(void);" for symbol in check_header.REQUIRED_SYMBOLS]
    if extra_symbols:
        prototypes.extend(f"int32_t {symbol}(void);" for symbol in extra_symbols)
    return "\n".join(prototypes)


class RequiredSymbolsRuleTests(unittest.TestCase):
    def test_accepts_exact_required_surface(self) -> None:
        header_text = make_required_symbols_header()
        prototypes = check_header.parse_prototypes(header_text)
        check_header.rule_required_symbols(prototypes, header_text)

    def test_rejects_unexpected_pure_simdjson_export(self) -> None:
        header_text = make_required_symbols_header(
            extra_symbols=["pure_simdjson_internal_helper"]
        )
        prototypes = check_header.parse_prototypes(header_text)

        with self.assertRaises(SystemExit) as excinfo:
            check_header.rule_required_symbols(prototypes, header_text)

        self.assertEqual(
            str(excinfo.exception),
            "unexpected exported symbols: pure_simdjson_internal_helper",
        )


if __name__ == "__main__":
    unittest.main()
