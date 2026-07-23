package purejson

import (
	"fmt"
	"sync"

	"github.com/amikos-tech/pure-simdjson/internal/ffi"
)

var (
	kernelMu              sync.Mutex
	kernelSelectionLocked bool
)

// Kernel returns the active process-wide native implementation name. It
// returns an empty string until a native library has already been loaded and
// never triggers library resolution or bootstrap work.
func Kernel() string {
	kernelMu.Lock()
	defer kernelMu.Unlock()

	libraryMu.Lock()
	library := cachedLibrary
	libraryMu.Unlock()
	if library == nil {
		return ""
	}

	name, rc := library.bindings.ImplementationName()
	if rc != int32(ffi.OK) {
		return ""
	}
	return name
}

// SetKernel selects an exact process-wide native implementation name for
// diagnostics. An empty name restores automatic selection. Selection becomes
// immutable after the first parser or parser pool is created.
func SetKernel(name string) error {
	kernelMu.Lock()
	defer kernelMu.Unlock()

	if kernelSelectionLocked {
		return ErrKernelLocked
	}

	library, err := activeLibrary()
	if err != nil {
		return err
	}

	rc := library.bindings.SetImplementation(name)
	if statusErr := wrapStatus(rc); statusErr != nil {
		if rc == int32(ffi.ErrInvalidArg) {
			return fmt.Errorf("%w: %v", ErrInvalidOption, statusErr)
		}
		return statusErr
	}

	selected, rc := library.bindings.ImplementationName()
	if err := wrapStatus(rc); err != nil {
		return err
	}

	libraryMu.Lock()
	library.implementationName = selected
	libraryMu.Unlock()
	return nil
}

func lockKernelSelection() {
	kernelMu.Lock()
	kernelSelectionLocked = true
	kernelMu.Unlock()
}
