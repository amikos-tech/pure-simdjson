package purejson

import (
	"fmt"
	"sync"

	"github.com/amikos-tech/pure-simdjson/internal/ffi"
)

var (
	kernelMu              sync.Mutex
	kernelCond            = sync.NewCond(&kernelMu)
	kernelSelectionLocked bool
	utilityKernelReserved bool

	// Test-only hooks make the reservation boundary observable without using
	// scheduler timing or a real bootstrap path. They are nil in production.
	utilityReservationHook func()
	setImplementationHook  func()
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
// immutable after the first parser or parser pool is created, or after a
// standalone utility's native CPU gate succeeds. A utility locks selection
// even when scanning later reports malformed content.
func SetKernel(name string) error {
	kernelMu.Lock()
	defer kernelMu.Unlock()

	for utilityKernelReserved {
		kernelCond.Wait()
	}
	if kernelSelectionLocked {
		return ErrKernelLocked
	}

	library, err := activeLibrary()
	if err != nil {
		return err
	}

	if setImplementationHook != nil {
		setImplementationHook()
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

// markKernelSelectionAfterUtility mirrors the native utility gate ordering.
// The caller must hold kernelMu. CPU rejection happens before native selection
// locks; every other returned status follows a successful gate.
func markKernelSelectionAfterUtility(rc int32) {
	if rc != int32(ffi.ErrCPUUnsupported) &&
		rc != int32(ffi.ErrBufferTooSmall) &&
		rc != int32(ffi.ErrInvalidArg) {
		kernelSelectionLocked = true
	}
}

// beginUtilityKernel reserves process-wide implementation selection before a
// utility resolves the library or calls native code. The expensive work must
// happen after this function returns, with kernelMu unlocked.
func beginUtilityKernel() {
	kernelMu.Lock()
	for utilityKernelReserved {
		kernelCond.Wait()
	}
	utilityKernelReserved = true
	hook := utilityReservationHook
	kernelMu.Unlock()
	if hook != nil {
		hook()
	}
}

// cancelUtilityKernel releases a reservation when bootstrap or preflight
// fails before native code can pass its CPU gate. Such failures never lock
// selection.
func cancelUtilityKernel() {
	kernelMu.Lock()
	utilityKernelReserved = false
	kernelCond.Broadcast()
	kernelMu.Unlock()
}

// finishUtilityKernel records the native utility status before allowing a
// waiter to select an implementation.
func finishUtilityKernel(rc int32) {
	kernelMu.Lock()
	markKernelSelectionAfterUtility(rc)
	utilityKernelReserved = false
	kernelCond.Broadcast()
	kernelMu.Unlock()
}
