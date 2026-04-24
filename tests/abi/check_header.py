#!/usr/bin/env python3

from __future__ import annotations

import argparse
import pathlib
import re
import sys
from typing import Callable, NoReturn


PROTO_RE = re.compile(
    r"(?s)([A-Za-z_][\w\s\*]*?)\s+"
    r"((?:pure_simdjson_|psdj_internal_|psimdjson_)[A-Za-z0-9_]+)\s*"
    r"\((.*?)\);"
)
COMMENT_RE = re.compile(r"(?s)/\*.*?\*/|//[^\n]*")
ABI_VERSION_DEFINE_RE = re.compile(
    r"(?m)^#define\s+PURE_SIMDJSON_ABI_VERSION\s+0x00010001\s*$"
)
FORBIDDEN_INTERNAL_SYMBOL_PREFIXES = ("psdj_internal_", "psimdjson_")
HEADER_SYMBOL_PREFIXES = ("pure_simdjson_",) + FORBIDDEN_INTERNAL_SYMBOL_PREFIXES

STRUCT_TYPES = (
    "struct pure_simdjson_value_view_t",
    "struct pure_simdjson_array_iter_t",
    "struct pure_simdjson_object_iter_t",
    "struct pure_simdjson_native_alloc_stats_t",
    "pure_simdjson_value_view_t",
    "pure_simdjson_array_iter_t",
    "pure_simdjson_object_iter_t",
    "pure_simdjson_native_alloc_stats_t",
    "pure_simdjson_handle_parts_t",
)

NATIVE_ALLOC_STATS_STRUCT_RE = re.compile(
    r"typedef\s+struct\s+pure_simdjson_native_alloc_stats_t\s*\{\s*"
    r"uint64_t\s+epoch;\s*"
    r"uint64_t\s+live_bytes;\s*"
    r"uint64_t\s+total_alloc_bytes;\s*"
    r"uint64_t\s+alloc_count;\s*"
    r"uint64_t\s+free_count;\s*"
    r"uint64_t\s+untracked_free_count;\s*"
    r"\}\s+pure_simdjson_native_alloc_stats_t\s*;",
    re.S,
)

REQUIRED_SYMBOLS = (
    "pure_simdjson_get_abi_version",
    "pure_simdjson_get_implementation_name_len",
    "pure_simdjson_copy_implementation_name",
    "pure_simdjson_native_alloc_stats_reset",
    "pure_simdjson_native_alloc_stats_snapshot",
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


def strip_comments(header_text: str) -> str:
    return COMMENT_RE.sub("", header_text)


def iter_prototype_statements(header_text: str) -> list[str]:
    statements = []
    current: list[str] = []

    for raw_line in strip_comments(header_text).splitlines():
        line = raw_line.strip()
        if not line or line.startswith("#"):
            continue

        if current:
            current.append(line)
            if ";" in line:
                statements.append(normalize_space(" ".join(current)))
                current = []
            continue

        if not any(prefix in line for prefix in HEADER_SYMBOL_PREFIXES) or "(" not in line:
            continue

        current = [line]
        if ";" in line:
            statements.append(normalize_space(" ".join(current)))
            current = []

    return statements


def parse_prototypes(header_text: str) -> dict[str, tuple[str, list[str]]]:
    prototypes: dict[str, tuple[str, list[str]]] = {}
    for statement in iter_prototype_statements(header_text):
        match = PROTO_RE.fullmatch(statement)
        if match is None:
            symbol_match = re.search(
                r"\b((?:pure_simdjson_|psdj_internal_|psimdjson_)[A-Za-z0-9_]+)\b",
                statement,
            )
            symbol = symbol_match.group(1) if symbol_match else statement
            fail(f"unparseable exported prototype: {symbol}: {statement}")
        return_type = normalize_space(match.group(1))
        name = match.group(2)
        params_blob = normalize_space(match.group(3))
        params = [] if params_blob == "void" else [normalize_space(part) for part in params_blob.split(",")]
        prototypes[name] = (return_type, params)
    return prototypes


def fail(message: str) -> NoReturn:
    raise SystemExit(message)


def require_symbol(
    prototypes: dict[str, tuple[str, list[str]]],
    symbol: str,
) -> tuple[str, list[str]]:
    if symbol not in prototypes:
        fail(f"missing required symbol: {symbol}")
    return prototypes[symbol]


def rule_error_code_outparams(
    prototypes: dict[str, tuple[str, list[str]]], _: str
) -> None:
    for name, (return_type, params) in prototypes.items():
        if return_type not in (
            "pure_simdjson_error_code_t",
            "enum pure_simdjson_error_code_t",
        ):
            fail(
                f"{name}: expected pure_simdjson_error_code_t return, found {return_type}"
            )
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
    unexpected = sorted(
        name
        for name in prototypes
        if name.startswith("pure_simdjson_") and name not in REQUIRED_SYMBOLS
    )
    if unexpected:
        fail("unexpected exported symbols: " + ", ".join(unexpected))


def rule_no_internal_symbols(
    prototypes: dict[str, tuple[str, list[str]]], _: str
) -> None:
    internal_symbols = sorted(
        name
        for name in prototypes
        if name.startswith(FORBIDDEN_INTERNAL_SYMBOL_PREFIXES)
    )
    if internal_symbols:
        fail(
            "internal symbols must not appear in public header: "
            + ", ".join(internal_symbols)
        )


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


def rule_diag_surface(
    prototypes: dict[str, tuple[str, list[str]]], header_text: str
) -> None:
    if not ABI_VERSION_DEFINE_RE.search(header_text):
        fail("missing ABI version macro: #define PURE_SIMDJSON_ABI_VERSION 0x00010001")

    expected_signatures = {
        "pure_simdjson_get_abi_version": ["uint32_t *out_version"],
        "pure_simdjson_get_implementation_name_len": ["size_t *out_len"],
        "pure_simdjson_copy_implementation_name": [
            "uint8_t *dst",
            "size_t dst_cap",
            "size_t *out_written",
        ],
        "pure_simdjson_parser_new": ["pure_simdjson_parser_t *out_parser"],
        "pure_simdjson_parser_free": ["pure_simdjson_parser_t parser"],
        "pure_simdjson_parser_parse": [
            "pure_simdjson_parser_t parser",
            "const uint8_t *input_ptr",
            "size_t input_len",
            "pure_simdjson_doc_t *out_doc",
        ],
        "pure_simdjson_parser_get_last_error_len": [
            "pure_simdjson_parser_t parser",
            "size_t *out_len",
        ],
        "pure_simdjson_parser_copy_last_error": [
            "pure_simdjson_parser_t parser",
            "uint8_t *dst",
            "size_t dst_cap",
            "size_t *out_written",
        ],
        "pure_simdjson_parser_get_last_error_offset": [
            "pure_simdjson_parser_t parser",
            "uint64_t *out_offset",
        ],
        "pure_simdjson_doc_free": ["pure_simdjson_doc_t doc"],
        "pure_simdjson_doc_root": [
            "pure_simdjson_doc_t doc",
            "struct pure_simdjson_value_view_t *out_root",
        ],
    }

    for symbol, expected_params in expected_signatures.items():
        _, params = require_symbol(prototypes, symbol)
        if params != expected_params:
            fail(f"{symbol}: expected {expected_params}, found {params}")

    require_symbol(prototypes, "pure_simdjson_bytes_free")


def rule_native_alloc_surface(
    prototypes: dict[str, tuple[str, list[str]]], header_text: str
) -> None:
    if not NATIVE_ALLOC_STATS_STRUCT_RE.search(strip_comments(header_text)):
        fail(
            "pure_simdjson_native_alloc_stats_t: expected fields "
            "[epoch, live_bytes, total_alloc_bytes, alloc_count, free_count, "
            "untracked_free_count] in order"
        )

    _, reset_params = require_symbol(prototypes, "pure_simdjson_native_alloc_stats_reset")
    if reset_params != []:
        fail(
            "pure_simdjson_native_alloc_stats_reset: expected no parameters, "
            f"found {reset_params}"
        )

    _, snapshot_params = require_symbol(
        prototypes, "pure_simdjson_native_alloc_stats_snapshot"
    )
    expected_snapshot = ["struct pure_simdjson_native_alloc_stats_t *out_stats"]
    if snapshot_params != expected_snapshot:
        fail(
            "pure_simdjson_native_alloc_stats_snapshot: expected "
            f"{expected_snapshot}, found {snapshot_params}"
        )


RULES: dict[str, Callable[[dict[str, tuple[str, list[str]]], str], None]] = {
    "error-code-outparams": rule_error_code_outparams,
    "no-mixed-float-int": rule_no_mixed_float_int,
    "no-internal-symbols": rule_no_internal_symbols,
    "required-symbols": rule_required_symbols,
    "string-copy-ownership": rule_string_copy_ownership,
    "diag-surface": rule_diag_surface,
    "native-alloc-surface": rule_native_alloc_surface,
}


def main() -> int:
    parser = argparse.ArgumentParser(description="Lint the generated public ABI header.")
    parser.add_argument("header", type=pathlib.Path)
    parser.add_argument(
        "--rule",
        dest="rules",
        action="append",
        choices=sorted(RULES),
        required=False,
        help="Rule name to execute; may be passed multiple times.",
    )
    args = parser.parse_args()

    try:
        header_text = args.header.read_text(encoding="utf-8")
    except OSError as error:
        fail(f"read {args.header}: {error}")
    prototypes = parse_prototypes(header_text)
    selected_rules = args.rules or list(RULES)
    for rule_name in selected_rules:
        RULES[rule_name](prototypes, header_text)

    print(f"ok: {args.header} ({', '.join(selected_rules)})")
    return 0


if __name__ == "__main__":
    sys.exit(main())
