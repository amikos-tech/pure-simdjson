package purejson

import "unsafe"

// Minify returns a SIMD-accelerated minified copy of data without modifying
// data. It removes insignificant JSON whitespace, but it does not validate
// JSON: only unterminated strings are detected, so a nil error does not mean
// data was valid JSON.
//
// Minify triggers the same CPU-unsupported rejection NewParser uses: on an
// unsupported CPU it returns ErrCPUUnsupported instead of silently running the
// slow fallback kernel. Once its native CPU gate succeeds it locks kernel
// selection, even if scanning later reports malformed content, so SetKernel
// returns ErrKernelLocked afterward.
func Minify(data []byte) ([]byte, error) {
	dst := make([]byte, len(data))
	written, err := MinifyInto(dst, data)
	if err != nil {
		return nil, err
	}
	return dst[:written], nil
}

// MinifyInto writes the SIMD-accelerated minified form of src into dst and
// returns the number of bytes written. Exact same-start aliasing (dst and src
// start at the same byte) and disjoint buffers are supported. Any partial
// overlap is rejected with ErrInvalidOption before writing. If dst is shorter
// than src, MinifyInto returns ErrBufferTooSmall before writing.
//
// MinifyInto removes insignificant JSON whitespace, but it does not validate
// JSON: only unterminated strings are detected, so a nil error does not mean
// src was valid JSON.
//
// MinifyInto triggers the same CPU-unsupported rejection NewParser uses: on an
// unsupported CPU it returns ErrCPUUnsupported instead of silently running the
// slow fallback kernel. Once its native CPU gate succeeds it locks kernel
// selection, even if scanning later reports malformed content, so SetKernel
// returns ErrKernelLocked afterward.
func MinifyInto(dst, src []byte) (int, error) {
	if len(dst) < len(src) {
		return 0, ErrBufferTooSmall
	}
	if byteSlicesPartiallyOverlap(dst, src) {
		return 0, ErrInvalidOption
	}

	beginUtilityKernel()
	library, err := activeLibrary()
	if err != nil {
		cancelUtilityKernel()
		return 0, err
	}

	written, rc := library.bindings.Minify(dst, src)
	finishUtilityKernel(rc)
	if err := wrapStatus(rc); err != nil {
		return 0, err
	}
	return written, nil
}

func byteSlicesPartiallyOverlap(dst, src []byte) bool {
	if len(dst) == 0 || len(src) == 0 {
		return false
	}

	dstStart := uintptr(unsafe.Pointer(unsafe.SliceData(dst)))
	srcStart := uintptr(unsafe.Pointer(unsafe.SliceData(src)))
	if dstStart == srcStart {
		return false
	}
	if dstStart < srcStart {
		return srcStart-dstStart < uintptr(len(dst))
	}
	return dstStart-srcStart < uintptr(len(src))
}
