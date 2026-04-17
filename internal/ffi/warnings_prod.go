//go:build !purejson_testbuild

package ffi

import "os"

// leakWarningsEnabled turns on stderr warnings in normal builds when
// PURE_SIMDJSON_WARN_LEAKS=1 is set.
func leakWarningsEnabled() bool {
	return os.Getenv("PURE_SIMDJSON_WARN_LEAKS") == "1"
}
