.PHONY: generate-header verify-contract verify-docs

generate-header:
	cbindgen --config cbindgen.toml --crate pure_simdjson --output include/pure_simdjson.h

verify-contract:
	cargo check
	tmp="$$(mktemp)"; trap 'rm -f "$$tmp"' EXIT; \
	cbindgen --config cbindgen.toml --crate pure_simdjson --output "$$tmp"; \
	diff -u include/pure_simdjson.h "$$tmp"
	python3 tests/abi/check_header.py --rule int32-outparams --rule no-mixed-float-int --rule required-symbols --rule string-copy-ownership --rule diag-surface include/pure_simdjson.h
	out="$$(mktemp /tmp/pure_simdjson_handle_layout.XXXXXX.o)"; trap 'rm -f "$$out"' EXIT; \
	cc -Iinclude tests/abi/handle_layout.c -c -o "$$out"

verify-docs:
	rg 'ffi_fn!|catch_unwind|panic = "abort"|\.get\(err\)|PURE_SIMDJSON_ERR_PARSER_BUSY|PURE_SIMDJSON_ERR_NUMBER_OUT_OF_RANGE|PURE_SIMDJSON_ERR_PRECISION_LOSS|pure_simdjson_element_get_int64|pure_simdjson_element_get_uint64|pure_simdjson_element_get_float64|SIMDJSON_PADDING|\^0\.1\.x' docs/ffi-contract.md
