package purejson

import (
	"fmt"
	"sync"

	"github.com/amikos-tech/pure-simdjson/internal/ffi"
)

var (
	kernelMu                  sync.RWMutex
	kernelCond                = sync.NewCond(&kernelMu)
	kernelSelectionLocked     bool
	utilityKernelReservations uint

	// Test-only hooks make the reservation boundary observable without using
	// scheduler timing or a real bootstrap path. They are nil in production.
	utilityReservationHook func()
	setImplementationHook  func()
)

// Kernel returns the active process-wide native implementation name. It
// returns an empty string until a native library has already been loaded and
// never triggers library resolution or bootstrap work.
func Kernel() string {
	kernelMu.RLock()
	defer kernelMu.RUnlock()

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

	for utilityKernelReservations != 0 {
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
// The caller must hold kernelMu. CPU rejection, a too-small destination, and
// invalid arguments happen before native selection locks; every other returned
// status follows a successful gate.
func markKernelSelectionAfterUtility(rc int32) {
	if rc != int32(ffi.ErrCPUUnsupported) &&
		rc != int32(ffi.ErrBufferTooSmall) &&
		rc != int32(ffi.ErrInvalidArg) {
		kernelSelectionLocked = true
	}
}

// utilityKernelReservation keeps SetKernel out until a utility has published
// its native gate result. Multiple utilities may run concurrently; only
// SetKernel waits for all of them to finish.
type utilityKernelReservation struct {
	released bool
}

// beginUtilityKernel reserves process-wide implementation selection before a
// utility resolves the library or calls native code. The expensive work must
// happen after this function returns, with kernelMu unlocked.
func beginUtilityKernel() (reservation *utilityKernelReservation) {
	reservation = &utilityKernelReservation{}
	kernelMu.Lock()
	utilityKernelReservations++
	hook := utilityReservationHook
	kernelMu.Unlock()
	defer func() {
		if recovered := recover(); recovered != nil {
			reservation.cancel()
			panic(recovered)
		}
	}()
	if hook != nil {
		hook()
	}
	return reservation
}

// cancel releases a reservation when bootstrap or preflight fails before
// native code can pass its CPU gate. It is safe to defer after finish.
func (reservation *utilityKernelReservation) cancel() {
	kernelMu.Lock()
	if !reservation.released {
		reservation.released = true
		utilityKernelReservations--
		kernelCond.Broadcast()
	}
	kernelMu.Unlock()
}

// finish records the native utility status before allowing SetKernel to select
// an implementation. It is safe to pair with a deferred cancel.
func (reservation *utilityKernelReservation) finish(rc int32) {
	kernelMu.Lock()
	if !reservation.released {
		markKernelSelectionAfterUtility(rc)
		reservation.released = true
		utilityKernelReservations--
		kernelCond.Broadcast()
	}
	kernelMu.Unlock()
}
