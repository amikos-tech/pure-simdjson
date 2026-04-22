.PHONY: generate-header verify-contract verify-docs phase2-smoke-linux phase2-smoke-windows phase2-verify-exports phase3-go-test phase3-go-race phase3-go-wrapper-remote bench-phase7 bench-phase7-cold bench-phase7-compare

generate-header:
	cbindgen --config cbindgen.toml --crate pure_simdjson --output include/pure_simdjson.h

verify-contract:
	cargo check
	cargo test
	tmp_dir="$$(mktemp -d)"; \
	tmp="$$tmp_dir/pure_simdjson.h"; \
	out="$$tmp_dir/pure_simdjson_handle_layout.o"; \
	trap 'rm -rf "$$tmp_dir"' EXIT; \
	cbindgen --config cbindgen.toml --crate pure_simdjson --output "$$tmp"; \
	diff -u include/pure_simdjson.h "$$tmp"; \
	python3 tests/abi/test_check_header.py; \
	python3 tests/abi/check_header.py --rule error-code-outparams --rule no-mixed-float-int --rule required-symbols --rule string-copy-ownership --rule diag-surface include/pure_simdjson.h; \
	cc -Iinclude tests/abi/handle_layout.c -c -o "$$out"

verify-docs:
	@for pattern in 'ffi_wrap' 'catch_unwind' 'panic = "abort"' '\.get\(err\)' 'PURE_SIMDJSON_ERR_PARSER_BUSY' 'PURE_SIMDJSON_ERR_NUMBER_OUT_OF_RANGE' 'PURE_SIMDJSON_ERR_PRECISION_LOSS' 'pure_simdjson_element_get_int64' 'pure_simdjson_element_get_uint64' 'pure_simdjson_element_get_float64' 'SIMDJSON_PADDING' '\^0\.1\.x'; do \
		rg -q "$$pattern" docs/ffi-contract.md || { echo "verify-docs: docs/ffi-contract.md missing required pattern: $$pattern" >&2; exit 1; }; \
	done

phase2-smoke-linux:
	mkdir -p target/phase2-smoke
	cc -Iinclude tests/smoke/minimal_parse.c -Ltarget/release -lpure_simdjson -Wl,-rpath,$$PWD/target/release -o target/phase2-smoke/minimal_parse
	target/phase2-smoke/minimal_parse

phase2-smoke-windows:
ifeq ($(OS),Windows_NT)
	if not exist target\phase2-smoke mkdir target\phase2-smoke
	cl /nologo /Iinclude tests\smoke\minimal_parse.c /link /LIBPATH:target\release pure_simdjson.dll.lib /OUT:target\phase2-smoke\minimal_parse.exe
	set "PATH=$(CURDIR)\target\release;%PATH%" && target\phase2-smoke\minimal_parse.exe
else
	@echo "Run from Windows/MSVC:"
	@echo "  if not exist target\\phase2-smoke mkdir target\\phase2-smoke"
	@echo "  cl /nologo /Iinclude tests\\smoke\\minimal_parse.c /link /LIBPATH:target\\release pure_simdjson.dll.lib /OUT:target\\phase2-smoke\\minimal_parse.exe"
	@echo "  set PATH=$(CURDIR)\\target\\release;%PATH%"
	@echo "  target\\phase2-smoke\\minimal_parse.exe"
endif

phase2-verify-exports:
ifeq ($(OS),Windows_NT)
	dumpbin /EXPORTS target\release\pure_simdjson.dll
else
	nm -D --defined-only target/release/libpure_simdjson.so
endif

phase3-go-test:
	cargo build --release
	go test ./...

phase3-go-race:
	cargo build --release
	go test ./... -race

phase3-go-wrapper-remote:
	./scripts/phase3-go-wrapper-smoke.sh

bench-phase7:
	go test ./... -run '^$$' -bench 'Benchmark(Tier1|Tier2|Tier3)_' -benchmem -count=5

bench-phase7-cold:
	go test ./... -run '^$$' -bench 'Benchmark(ColdStart|Warm)_' -benchmem -count=5

bench-phase7-compare:
	./scripts/bench/run_benchstat.sh --old "$(OLD)" --new "$(NEW)"
