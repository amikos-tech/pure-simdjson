#!/usr/bin/env python3

from __future__ import annotations

import importlib.util
import pathlib
import unittest


REPO_ROOT = pathlib.Path(__file__).resolve().parents[2]
CHECK_HEADER_PATH = REPO_ROOT / "tests" / "abi" / "check_header.py"
ABI_VERSION_DEFINE = "#define PURE_SIMDJSON_ABI_VERSION 0x00010000"
SURFACE_SIGNATURES = {
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
    "pure_simdjson_element_type": [
        "const struct pure_simdjson_value_view_t *view",
        "uint32_t *out_type",
    ],
    "pure_simdjson_element_get_int64": [
        "const struct pure_simdjson_value_view_t *view",
        "int64_t *out_value",
    ],
    "pure_simdjson_element_get_uint64": [
        "const struct pure_simdjson_value_view_t *view",
        "uint64_t *out_value",
    ],
    "pure_simdjson_element_get_float64": [
        "const struct pure_simdjson_value_view_t *view",
        "double *out_value",
    ],
    "pure_simdjson_element_get_string": [
        "const struct pure_simdjson_value_view_t *view",
        "uint8_t **out_ptr",
        "size_t *out_len",
    ],
    "pure_simdjson_bytes_free": ["uint8_t *ptr", "size_t len"],
    "pure_simdjson_element_get_bool": [
        "const struct pure_simdjson_value_view_t *view",
        "uint8_t *out_value",
    ],
    "pure_simdjson_element_is_null": [
        "const struct pure_simdjson_value_view_t *view",
        "uint8_t *out_is_null",
    ],
    "pure_simdjson_array_iter_new": [
        "const struct pure_simdjson_value_view_t *array_view",
        "struct pure_simdjson_array_iter_t *out_iter",
    ],
    "pure_simdjson_array_iter_next": [
        "struct pure_simdjson_array_iter_t *iter",
        "struct pure_simdjson_value_view_t *out_value",
        "uint8_t *out_done",
    ],
    "pure_simdjson_object_iter_new": [
        "const struct pure_simdjson_value_view_t *object_view",
        "struct pure_simdjson_object_iter_t *out_iter",
    ],
    "pure_simdjson_object_iter_next": [
        "struct pure_simdjson_object_iter_t *iter",
        "struct pure_simdjson_value_view_t *out_key",
        "struct pure_simdjson_value_view_t *out_value",
        "uint8_t *out_done",
    ],
    "pure_simdjson_object_get_field": [
        "const struct pure_simdjson_value_view_t *object_view",
        "const uint8_t *key_ptr",
        "size_t key_len",
        "struct pure_simdjson_value_view_t *out_value",
    ],
}

SPEC = importlib.util.spec_from_file_location("check_header", CHECK_HEADER_PATH)
assert SPEC is not None and SPEC.loader is not None
check_header = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(check_header)


def make_required_symbols_header(*, extra_symbols: list[str] | None = None) -> str:
    prototypes = [
        f"pure_simdjson_error_code_t {symbol}(void);"
        for symbol in check_header.REQUIRED_SYMBOLS
    ]
    if extra_symbols:
        prototypes.extend(
            f"pure_simdjson_error_code_t {symbol}(void);"
            for symbol in extra_symbols
        )
    return "\n".join(prototypes)


def make_surface_header(
    *,
    include_abi_version_macro: bool = True,
    return_type: str = "pure_simdjson_error_code_t",
    overrides: dict[str, list[str]] | None = None,
    omit_symbols: set[str] | None = None,
    extra_lines: list[str] | None = None,
) -> str:
    lines: list[str] = []
    if include_abi_version_macro:
        lines.append(ABI_VERSION_DEFINE)

    signatures = dict(SURFACE_SIGNATURES)
    if overrides:
        signatures.update(overrides)
    if omit_symbols:
        for symbol in omit_symbols:
            signatures.pop(symbol, None)

    for symbol in check_header.REQUIRED_SYMBOLS:
        if symbol not in signatures:
            continue
        params = ", ".join(signatures[symbol])
        lines.append(f"{return_type} {symbol}({params});")

    if extra_lines:
        lines.extend(extra_lines)

    return "\n".join(lines)


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

        self.assertIn("unexpected exported symbols", str(excinfo.exception))

    def test_rejects_missing_required_symbol(self) -> None:
        header_text = make_surface_header(
            omit_symbols={"pure_simdjson_object_get_field"}
        )
        prototypes = check_header.parse_prototypes(header_text)

        with self.assertRaises(SystemExit) as excinfo:
            check_header.rule_required_symbols(prototypes, header_text)

        self.assertIn(
            "missing required symbols: pure_simdjson_object_get_field",
            str(excinfo.exception),
        )


class ErrorCodeOutparamsRuleTests(unittest.TestCase):
    def test_accepts_error_code_returns_with_pointer_transport(self) -> None:
        header_text = make_surface_header()
        prototypes = check_header.parse_prototypes(header_text)

        check_header.rule_error_code_outparams(prototypes, header_text)

    def test_rejects_plain_int32_return_type(self) -> None:
        header_text = make_surface_header(return_type="int32_t")
        prototypes = check_header.parse_prototypes(header_text)

        with self.assertRaises(SystemExit) as excinfo:
            check_header.rule_error_code_outparams(prototypes, header_text)

        self.assertIn(
            "expected pure_simdjson_error_code_t return",
            str(excinfo.exception),
        )

    def test_rejects_struct_value_transport(self) -> None:
        header_text = make_surface_header(
            overrides={
                "pure_simdjson_doc_root": [
                    "pure_simdjson_doc_t doc",
                    "struct pure_simdjson_value_view_t out_root",
                ]
            }
        )
        prototypes = check_header.parse_prototypes(header_text)

        with self.assertRaises(SystemExit) as excinfo:
            check_header.rule_error_code_outparams(prototypes, header_text)

        self.assertIn("struct transport must use pointer out-params", str(excinfo.exception))


class RealHeaderRuleTests(unittest.TestCase):
    def test_real_public_header_matches_contract_rules(self) -> None:
        header_text = (REPO_ROOT / "include" / "pure_simdjson.h").read_text()
        prototypes = check_header.parse_prototypes(header_text)

        check_header.rule_required_symbols(prototypes, header_text)
        check_header.rule_error_code_outparams(prototypes, header_text)


class NoMixedFloatIntRuleTests(unittest.TestCase):
    def test_accepts_pointer_based_float_outparams(self) -> None:
        header_text = make_surface_header()
        prototypes = check_header.parse_prototypes(header_text)

        check_header.rule_no_mixed_float_int(prototypes, header_text)

    def test_rejects_scalar_double_parameter(self) -> None:
        header_text = make_surface_header(
            overrides={
                "pure_simdjson_element_get_float64": [
                    "const struct pure_simdjson_value_view_t *view",
                    "double out_value",
                ]
            }
        )
        prototypes = check_header.parse_prototypes(header_text)

        with self.assertRaises(SystemExit) as excinfo:
            check_header.rule_no_mixed_float_int(prototypes, header_text)

        self.assertIn(
            "scalar float/double parameter is forbidden",
            str(excinfo.exception),
        )


class StringCopyOwnershipRuleTests(unittest.TestCase):
    def test_accepts_copy_out_string_surface(self) -> None:
        header_text = make_surface_header()
        prototypes = check_header.parse_prototypes(header_text)

        check_header.rule_string_copy_ownership(prototypes, header_text)

    def test_rejects_wrong_bytes_free_signature(self) -> None:
        header_text = make_surface_header(
            overrides={"pure_simdjson_bytes_free": ["void *ptr", "size_t len"]}
        )
        prototypes = check_header.parse_prototypes(header_text)

        with self.assertRaises(SystemExit) as excinfo:
            check_header.rule_string_copy_ownership(prototypes, header_text)

        self.assertIn("pure_simdjson_bytes_free: expected", str(excinfo.exception))


class DiagSurfaceRuleTests(unittest.TestCase):
    def test_accepts_expected_surface_and_abi_macro(self) -> None:
        header_text = make_surface_header()
        prototypes = check_header.parse_prototypes(header_text)

        check_header.rule_diag_surface(prototypes, header_text)

    def test_rejects_missing_abi_version_macro(self) -> None:
        header_text = make_surface_header(include_abi_version_macro=False)
        prototypes = check_header.parse_prototypes(header_text)

        with self.assertRaises(SystemExit) as excinfo:
            check_header.rule_diag_surface(prototypes, header_text)

        self.assertIn("missing ABI version macro", str(excinfo.exception))

    def test_rejects_wrong_parser_diagnostic_signature(self) -> None:
        header_text = make_surface_header(
            overrides={
                "pure_simdjson_parser_get_last_error_offset": [
                    "pure_simdjson_parser_t parser",
                    "uint32_t *out_offset",
                ]
            }
        )
        prototypes = check_header.parse_prototypes(header_text)

        with self.assertRaises(SystemExit) as excinfo:
            check_header.rule_diag_surface(prototypes, header_text)

        self.assertIn(
            "pure_simdjson_parser_get_last_error_offset: expected",
            str(excinfo.exception),
        )


if __name__ == "__main__":
    unittest.main()
