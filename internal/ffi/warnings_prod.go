//go:build !purejson_testbuild

package ffi

import "os"

// leakWarningsEnabled turns on stderr warnings in normal builds when
// PURE_SIMDJSON_WARN_LEAKS=1 is set. The default is off so quiet deployments
// stay quiet; opt-in surfaces FFI leak signals for operators investigating
// memory pressure or shim regressions without burdening every consumer with
// an unsolicited stderr line.
func leakWarningsEnabled() bool {
	return os.Getenv("PURE_SIMDJSON_WARN_LEAKS") == "1"
}
