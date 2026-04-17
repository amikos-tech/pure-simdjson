//go:build !purejson_testbuild

package ffi

import "os"

func leakWarningsEnabled() bool {
	return os.Getenv("PURE_SIMDJSON_WARN_LEAKS") == "1"
}
