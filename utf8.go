package purejson

// ValidateUTF8 reports whether data is valid UTF-8 using simdjson's standalone
// SIMD validator. Invalid UTF-8 returns (false, nil). A non-nil error may report
// bootstrap or library loading, an ABI mismatch, an unsupported CPU, a trapped
// Rust panic or C++ exception, or another native operational failure.
//
// ValidateUTF8 triggers the same CPU-unsupported rejection NewParser uses: on
// an unsupported CPU it returns ErrCPUUnsupported instead of silently running
// the slow fallback kernel. Once its native CPU gate succeeds it locks kernel
// selection, even when the data is invalid UTF-8, so SetKernel returns
// ErrKernelLocked afterward.
func ValidateUTF8(data []byte) (bool, error) {
	reservation := beginUtilityKernel()
	defer reservation.cancel()
	library, err := activeLibrary()
	if err != nil {
		return false, err
	}

	valid, rc := library.bindings.ValidateUTF8(data)
	reservation.finish(rc)
	if err := wrapStatus(rc); err != nil {
		return false, err
	}
	return valid, nil
}
