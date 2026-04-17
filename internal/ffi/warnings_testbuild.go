//go:build purejson_testbuild

package ffi

// leakWarningsEnabled is forced on in purejson_testbuild builds so leak-path
// tests do not depend on PURE_SIMDJSON_WARN_LEAKS.
func leakWarningsEnabled() bool {
	return true
}
