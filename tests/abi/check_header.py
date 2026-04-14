#!/usr/bin/env python3

from __future__ import annotations

import argparse
import pathlib
import re
import sys
from typing import Callable


PROTO_RE = re.compile(
    r"(?ms)^([A-Za-z_][\w\s\*]*?)\s+"
    r"(pure_simdjson_[A-Za-z0-9_]+)\s*"
    r"\((.*?)\);"
)

STRUCT_TYPES = (
    "struct pure_simdjson_value_view_t",
    "struct pure_simdjson_array_iter_t",
    "struct pure_simdjson_object_iter_t",
    "pure_simdjson_value_view_t",
    "pure_simdjson_array_iter_t",
    "pure_simdjson_object_iter_t",
    "pure_simdjson_handle_parts_t",
)

REQUIRED_SYMBOLS = (
    "pure_simdjson_get_abi_version",
    "pure_simdjson_get_implementation_name_len",
    "pure_simdjson_copy_implementation_name",
    "pure_simdjson_parser_new",
    "pure_simdjson_parser_free",
    "pure_simdjson_parser_parse",
    "pure_simdjson_parser_get_last_error_len",
    "pure_simdjson_parser_copy_last_error",
    "pure_simdjson_parser_get_last_error_offset",
    "pure_simdjson_doc_free",
    "pure_simdjson_doc_root",
    "pure_simdjson_element_type",
    "pure_simdjson_element_get_int64",
    "pure_simdjson_element_get_uint64",
    "pure_simdjson_element_get_float64",
    "pure_simdjson_element_get_string",
    "pure_simdjson_bytes_free",
    "pure_simdjson_element_get_bool",
    "pure_simdjson_element_is_null",
    "pure_simdjson_array_iter_new",
    "pure_simdjson_array_iter_next",
    "pure_simdjson_object_iter_new",
    "pure_simdjson_object_iter_next",
    "pure_simdjson_object_get_field",
)


def normalize_space(value: str) -> str:
    return " ".join(value.split())


def parse_prototypes(header_text: str) -> dict[str, tuple[str, list[str]]]:
    prototypes: dict[str, tuple[str, list[str]]] = {}
    for match in PROTO_RE.finditer(header_text):
        return_type = normalize_space(match.group(1))
        name = match.group(2)
        params_blob = normalize_space(match.group(3))
        params = [] if params_blob == "void" else [normalize_space(part) for part in params_blob.split(",")]
        prototypes[name] = (return_type, params)
    return prototypes


def fail(message: str) -> None:
    raise SystemExit(message)


def require_symbol(
    prototypes: dict[str, tuple[str, list[str]]],
    symbol: str,
) -> tuple[str, list[str]]:
    if symbol not in prototypes:
        fail(f"missing required symbol: {symbol}")
    return prototypes[symbol]


def rule_int32_outparams(prototypes: dict[str, tuple[str, list[str]]], _: str) -> None:
    for name, (return_type, params) in prototypes.items():
        if return_type != "int32_t":
            fail(f"{name}: expected int32_t return, found {return_type}")
        for param in params:
            if any(struct_type in param for struct_type in STRUCT_TYPES) and "*" not in param:
                fail(f"{name}: struct transport must use pointer out-params, found by-value parameter {param}")


def rule_no_mixed_float_int(prototypes: dict[str, tuple[str, list[str]]], _: str) -> None:
    scalar_float = re.compile(r"\b(?:float|double)\b(?!\s*\*)")
    for name, (_, params) in prototypes.items():
        for param in params:
            if scalar_float.search(param):
                fail(f"{name}: scalar float/double parameter is forbidden: {param}")


def rule_required_symbols(prototypes: dict[str, tuple[str, list[str]]], _: str) -> None:
    missing = [symbol for symbol in REQUIRED_SYMBOLS if symbol not in prototypes]
    if missing:
        fail("missing required symbols: " + ", ".join(missing))


def rule_string_copy_ownership(prototypes: dict[str, tuple[str, list[str]]], _: str) -> None:
    _, params = require_symbol(prototypes, "pure_simdjson_element_get_string")
    expected = [
        "const struct pure_simdjson_value_view_t *view",
        "uint8_t **out_ptr",
        "size_t *out_len",
    ]
    if params != expected:
        fail(
            "pure_simdjson_element_get_string: expected "
            f"{expected}, found {params}"
        )

    _, free_params = require_symbol(prototypes, "pure_simdjson_bytes_free")
    expected_free = ["uint8_t *ptr", "size_t len"]
    if free_params != expected_free:
        fail(
            "pure_simdjson_bytes_free: expected "
            f"{expected_free}, found {free_params}"
        )


def rule_diag_surface(prototypes: dict[str, tuple[str, list[str]]], _: str) -> None:
    expected_signatures = {
        "pure_simdjson_get_abi_version": ["uint32_t *out_version"],
        "pure_simdjson_get_implementation_name_len": ["size_t *out_len"],
        "pure_simdjson_copy_implementation_name": [
            "uint8_t *dst",
            "size_t dst_cap",
            "size_t *out_written",
        ],
        "pure_simdjson_parser_get_last_error_len": [
            "pure_simdjson_handle_t parser",
            "size_t *out_len",
        ],
        "pure_simdjson_parser_copy_last_error": [
            "pure_simdjson_handle_t parser",
            "uint8_t *dst",
            "size_t dst_cap",
            "size_t *out_written",
        ],
        "pure_simdjson_parser_get_last_error_offset": [
            "pure_simdjson_handle_t parser",
            "uint64_t *out_offset",
        ],
    }

    for symbol, expected_params in expected_signatures.items():
        _, params = require_symbol(prototypes, symbol)
        if params != expected_params:
            fail(f"{symbol}: expected {expected_params}, found {params}")

    require_symbol(prototypes, "pure_simdjson_bytes_free")


RULES: dict[str, Callable[[dict[str, tuple[str, list[str]]], str], None]] = {
    "int32-outparams": rule_int32_outparams,
    "no-mixed-float-int": rule_no_mixed_float_int,
    "required-symbols": rule_required_symbols,
    "string-copy-ownership": rule_string_copy_ownership,
    "diag-surface": rule_diag_surface,
}


def main() -> int:
    parser = argparse.ArgumentParser(description="Lint the generated public ABI header.")
    parser.add_argument("header", type=pathlib.Path)
    parser.add_argument(
        "--rule",
        dest="rules",
        action="append",
        choices=sorted(RULES),
        required=True,
        help="Rule name to execute; may be passed multiple times.",
    )
    args = parser.parse_args()

    header_text = args.header.read_text(encoding="utf-8")
    prototypes = parse_prototypes(header_text)
    for rule_name in args.rules:
        RULES[rule_name](prototypes, header_text)

    print(f"ok: {args.header} ({', '.join(args.rules)})")
    return 0


if __name__ == "__main__":
    sys.exit(main())
