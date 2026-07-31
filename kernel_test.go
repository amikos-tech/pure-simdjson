package purejson

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/amikos-tech/pure-simdjson/internal/ffi"
)

const kernelScenarioEnv = "PUREJSON_KERNEL_SCENARIO"

func TestKernelUnloadedReadDoesNotLoad(t *testing.T) {
	runKernelScenario(t, "unloaded-read", func(t *testing.T) {
		t.Setenv(libraryEnvPath, filepath.Join(t.TempDir(), "missing-native-library"))
		t.Setenv("PURE_SIMDJSON_CACHE_DIR", t.TempDir())

		if got := Kernel(); got != "" {
			t.Fatalf("Kernel() = %q, want empty before native load", got)
		}
		if library := cachedLibraryForKernelTest(); library != nil {
			t.Fatalf("Kernel() installed cached library %#v", library)
		}
	})
}

func TestKernelSetAndRead(t *testing.T) {
	runKernelScenario(t, "set-and-read", func(t *testing.T) {
		if err := SetKernel(""); err != nil {
			t.Fatalf("SetKernel(automatic) error = %v", err)
		}
		current := Kernel()
		if current == "" {
			t.Fatal("Kernel() is empty after automatic selection")
		}
		if err := SetKernel(current); err != nil {
			t.Fatalf("SetKernel(%q) error = %v", current, err)
		}
		if got := Kernel(); got != current {
			t.Fatalf("Kernel() = %q, want %q", got, current)
		}
	})
}

func TestKernelInvalidNamePreservesSelection(t *testing.T) {
	runKernelScenario(t, "invalid-name", func(t *testing.T) {
		if err := SetKernel(""); err != nil {
			t.Fatalf("SetKernel(automatic) error = %v", err)
		}
		before := Kernel()

		for _, name := range []string{"not-a-compiled-implementation", strings.ToUpper(before)} {
			if name == before {
				continue
			}
			if err := SetKernel(name); !errors.Is(err, ErrInvalidOption) {
				t.Fatalf("SetKernel(%q) error = %v, want ErrInvalidOption", name, err)
			}
			if got := Kernel(); got != before {
				t.Fatalf("Kernel() after SetKernel(%q) = %q, want unchanged %q", name, got, before)
			}
		}
	})
}

func TestKernelUnsupportedNameWhenObservable(t *testing.T) {
	runKernelScenario(t, "unsupported-name", func(t *testing.T) {
		if err := SetKernel(""); err != nil {
			t.Fatalf("SetKernel(automatic) error = %v", err)
		}

		candidates := []string{"icelake", "haswell", "westmere", "arm64", "ppc64", "lasx", "lsx"}
		for _, candidate := range candidates {
			before := Kernel()
			err := SetKernel(candidate)
			switch {
			case errors.Is(err, ErrCPUUnsupported):
				if got := Kernel(); got != before {
					t.Fatalf("Kernel() after unsupported %q = %q, want %q", candidate, got, before)
				}
				return
			case errors.Is(err, ErrInvalidOption):
				if got := Kernel(); got != before {
					t.Fatalf("Kernel() after invalid %q = %q, want %q", candidate, got, before)
				}
			case err == nil:
				if err := SetKernel(""); err != nil {
					t.Fatalf("restore automatic selection after %q: %v", candidate, err)
				}
			default:
				t.Fatalf("SetKernel(%q) error = %v, want nil, ErrInvalidOption, or ErrCPUUnsupported", candidate, err)
			}
		}
		t.Log("no compiled-but-unsupported implementation is observable on this host")
	})
}

func TestKernelExplicitFallback(t *testing.T) {
	runKernelScenario(t, "explicit-fallback", func(t *testing.T) {
		if err := SetKernel("fallback"); err != nil {
			t.Fatalf("SetKernel(fallback) error = %v", err)
		}
		if got := Kernel(); got != "fallback" {
			t.Fatalf("Kernel() = %q, want fallback", got)
		}
		if got := cachedLibraryForKernelTest().implementationName; got != "fallback" {
			t.Fatalf("cached implementation name = %q, want fallback", got)
		}

		parser, err := NewParser()
		if err != nil {
			t.Fatalf("NewParser() with explicit fallback error = %v", err)
		}
		if err := parser.Close(); err != nil {
			t.Fatalf("parser.Close() error = %v", err)
		}
	})
}

func TestKernelAutomaticFallbackRejected(t *testing.T) {
	runKernelScenario(t, "automatic-fallback", func(t *testing.T) {
		t.Setenv("PURE_SIMDJSON_TEST_FORCE_IMPLEMENTATION", "fallback")

		parser, err := NewParser()
		if parser != nil {
			_ = parser.Close()
			t.Fatal("NewParser() returned a parser for automatic fallback")
		}
		if !errors.Is(err, ErrCPUUnsupported) {
			t.Fatalf("NewParser() error = %v, want ErrCPUUnsupported", err)
		}
	})
}

func TestKernelInvalidOptionsDoNotLock(t *testing.T) {
	runKernelScenario(t, "invalid-options", func(t *testing.T) {
		if _, err := NewParser(ParserOption{}); !errors.Is(err, ErrInvalidOption) {
			t.Fatalf("NewParser(invalid option) error = %v, want ErrInvalidOption", err)
		}
		if _, err := NewParserPool(ParserOption{}); !errors.Is(err, ErrInvalidOption) {
			t.Fatalf("NewParserPool(invalid option) error = %v, want ErrInvalidOption", err)
		}
		if library := cachedLibraryForKernelTest(); library != nil {
			t.Fatalf("invalid options installed cached library %#v", library)
		}
		if err := SetKernel(""); err != nil {
			t.Fatalf("SetKernel() after invalid options error = %v", err)
		}
	})
}

func TestKernelUtilityStatusGateExcludesOnlyPreGateStatuses(t *testing.T) {
	runKernelScenario(t, "utility-status-gate", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			rc   int32
			want bool
		}{
			{name: "cpu unsupported", rc: int32(ffi.ErrCPUUnsupported), want: false},
			{name: "buffer too small", rc: int32(ffi.ErrBufferTooSmall), want: false},
			{name: "invalid argument", rc: int32(ffi.ErrInvalidArg), want: false},
			{name: "invalid JSON after gate", rc: int32(ffi.ErrInvalidJSON), want: true},
			{name: "success", rc: int32(ffi.OK), want: true},
		} {
			t.Run(tc.name, func(t *testing.T) {
				kernelMu.Lock()
				kernelSelectionLocked = false
				markKernelSelectionAfterUtility(tc.rc)
				got := kernelSelectionLocked
				kernelMu.Unlock()
				if got != tc.want {
					t.Fatalf("markKernelSelectionAfterUtility(%d) locked = %t, want %t", tc.rc, got, tc.want)
				}
			})
		}
	})
}

func TestKernelUtilityReservationBlocksSetKernelUntilFinalStatus(t *testing.T) {
	runKernelScenario(t, "utility-reservation", func(t *testing.T) {
		if err := SetKernel(""); err != nil {
			t.Fatalf("SetKernel(automatic) error = %v", err)
		}

		reserved := make(chan struct{})
		releaseUtility := make(chan struct{})
		setBindingEntered := make(chan struct{}, 1)
		utilityReservationHook = func() {
			close(reserved)
			<-releaseUtility
		}
		setImplementationHook = func() { setBindingEntered <- struct{}{} }
		defer func() {
			utilityReservationHook = nil
			setImplementationHook = nil
		}()

		utilityResult := make(chan error, 1)
		go func() {
			_, err := Minify([]byte(` { "x" : 1 } `))
			utilityResult <- err
		}()
		<-reserved

		setResult := make(chan error, 1)
		go func() { setResult <- SetKernel("") }()
		select {
		case <-setBindingEntered:
			t.Fatal("SetKernel entered native binding while utility reservation was active")
		case <-time.After(50 * time.Millisecond):
		}

		close(releaseUtility)
		if err := <-utilityResult; err != nil {
			t.Fatalf("Minify() error = %v", err)
		}
		if err := <-setResult; !errors.Is(err, ErrKernelLocked) {
			t.Fatalf("SetKernel() after successful reserved utility error = %v, want ErrKernelLocked", err)
		}
		select {
		case <-setBindingEntered:
			t.Fatal("SetKernel entered native binding after the utility locked selection")
		default:
		}
	})
}

func TestKernelUtilityReservationReleasesAfterPanic(t *testing.T) {
	runKernelScenario(t, "utility-reservation-panic", func(t *testing.T) {
		utilityReservationHook = func() { panic("test utility reservation panic") }
		defer func() { utilityReservationHook = nil }()

		func() {
			defer func() {
				if recovered := recover(); recovered == nil {
					t.Fatal("beginUtilityKernel() did not propagate hook panic")
				}
			}()
			_ = beginUtilityKernel()
		}()

		kernelMu.Lock()
		reservations := utilityKernelReservations
		kernelMu.Unlock()
		if reservations != 0 {
			t.Fatalf("utility reservations after panic = %d, want 0", reservations)
		}

		setResult := make(chan error, 1)
		go func() { setResult <- SetKernel("") }()
		select {
		case err := <-setResult:
			if err != nil {
				t.Fatalf("SetKernel() after utility panic error = %v", err)
			}
		case <-time.After(time.Second):
			t.Fatal("SetKernel() remained blocked after utility panic")
		}
	})
}

func TestKernelUtilityReservationsPermitConcurrentUtilities(t *testing.T) {
	runKernelScenario(t, "utility-reservations-concurrent", func(t *testing.T) {
		first := beginUtilityKernel()
		defer first.cancel()
		second := beginUtilityKernel()
		defer second.cancel()

		kernelMu.Lock()
		reservations := utilityKernelReservations
		kernelMu.Unlock()
		if reservations != 2 {
			t.Fatalf("concurrent utility reservations = %d, want 2", reservations)
		}

		setBindingEntered := make(chan struct{}, 1)
		setImplementationHook = func() { setBindingEntered <- struct{}{} }
		defer func() { setImplementationHook = nil }()
		setResult := make(chan error, 1)
		go func() { setResult <- SetKernel("") }()
		select {
		case <-setBindingEntered:
			t.Fatal("SetKernel entered native binding before concurrent utilities finished")
		case <-time.After(50 * time.Millisecond):
		}

		first.finish(int32(ffi.ErrCPUUnsupported))
		select {
		case <-setBindingEntered:
			t.Fatal("SetKernel entered native binding while one utility remained reserved")
		case <-time.After(50 * time.Millisecond):
		}
		second.finish(int32(ffi.ErrCPUUnsupported))
		select {
		case err := <-setResult:
			if err != nil {
				t.Fatalf("SetKernel() after concurrent pre-gate utilities error = %v", err)
			}
		case <-time.After(time.Second):
			t.Fatal("SetKernel() remained blocked after all utilities finished")
		}
	})
}

func TestKernelParserCreationLocksSelection(t *testing.T) {
	runKernelScenario(t, "parser-lock", func(t *testing.T) {
		parser, err := NewParser()
		if err != nil {
			t.Fatalf("NewParser() error = %v", err)
		}
		defer func() {
			if err := parser.Close(); err != nil {
				t.Fatalf("parser.Close() error = %v", err)
			}
		}()

		if err := SetKernel(""); !errors.Is(err, ErrKernelLocked) {
			t.Fatalf("SetKernel() after NewParser() error = %v, want ErrKernelLocked", err)
		}
	})
}

func TestKernelNativeLockedStatusMapsToSentinel(t *testing.T) {
	runKernelScenario(t, "native-locked-status", func(t *testing.T) {
		parser, err := NewParser()
		if err != nil {
			t.Fatalf("NewParser() error = %v", err)
		}
		defer func() {
			if err := parser.Close(); err != nil {
				t.Fatalf("parser.Close() error = %v", err)
			}
		}()

		if err := SetKernel(""); !errors.Is(err, ErrKernelLocked) {
			t.Fatalf("public SetKernel() error = %v, want Go ErrKernelLocked guard", err)
		}

		raw := cachedBindingsForKernelTest(t).SetImplementation("")
		if raw != int32(ffi.ErrKernelLocked) {
			t.Fatalf("native SetImplementation() status = %d, want %d", raw, ffi.ErrKernelLocked)
		}
		err = wrapStatus(raw)
		if !errors.Is(err, ErrKernelLocked) {
			t.Fatalf("wrapStatus(%d) error = %v, want ErrKernelLocked", raw, err)
		}
		var nativeErr *Error
		if !errors.As(err, &nativeErr) {
			t.Fatalf("wrapStatus(%d) error = %v, want *Error", raw, err)
		}
		if nativeErr.Code() != int32(ffi.ErrKernelLocked) {
			t.Fatalf("wrapped native code = %d, want %d", nativeErr.Code(), ffi.ErrKernelLocked)
		}
	})
}

func TestKernelConcurrentSetAndConstructor(t *testing.T) {
	runKernelScenario(t, "set-parser-race", func(t *testing.T) {
		if err := SetKernel(""); err != nil {
			t.Fatalf("SetKernel(automatic) error = %v", err)
		}
		current := Kernel()

		start := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(2)

		setErr := make(chan error, 1)
		go func() {
			defer wg.Done()
			<-start
			setErr <- SetKernel(current)
		}()

		type parserResult struct {
			parser *Parser
			err    error
		}
		parserResultCh := make(chan parserResult, 1)
		go func() {
			defer wg.Done()
			<-start
			parser, err := NewParser()
			parserResultCh <- parserResult{parser: parser, err: err}
		}()

		close(start)
		wg.Wait()

		if err := <-setErr; err != nil && !errors.Is(err, ErrKernelLocked) {
			t.Fatalf("SetKernel() race error = %v, want nil or ErrKernelLocked", err)
		}
		result := <-parserResultCh
		if result.err != nil {
			t.Fatalf("NewParser() race error = %v", result.err)
		}
		if err := result.parser.Close(); err != nil {
			t.Fatalf("parser.Close() error = %v", err)
		}
	})
}

func TestParserPoolKernelLock(t *testing.T) {
	runKernelScenario(t, "pool-kernel-lock", func(t *testing.T) {
		t.Setenv(libraryEnvPath, filepath.Join(t.TempDir(), "missing-native-library"))

		pool, err := NewParserPool(WithMaxCapacity(32), WithMaxDepth(4))
		if err != nil {
			t.Fatalf("NewParserPool() error = %v", err)
		}
		if pool == nil {
			t.Fatal("NewParserPool() returned nil pool")
		}
		if library := cachedLibraryForKernelTest(); library != nil {
			t.Fatalf("NewParserPool() installed cached library %#v", library)
		}
		if err := SetKernel(""); !errors.Is(err, ErrKernelLocked) {
			t.Fatalf("SetKernel() after NewParserPool() error = %v, want ErrKernelLocked", err)
		}
		if library := cachedLibraryForKernelTest(); library != nil {
			t.Fatalf("locked SetKernel() installed cached library %#v", library)
		}
	})
}

func TestParserPoolLazyNativeLock(t *testing.T) {
	runKernelScenario(t, "pool-lazy-native-lock", func(t *testing.T) {
		pool, err := NewParserPool()
		if err != nil {
			t.Fatalf("NewParserPool() error = %v", err)
		}
		if library := cachedLibraryForKernelTest(); library != nil {
			t.Fatalf("NewParserPool() installed cached library %#v", library)
		}

		parser, err := pool.Get()
		if err != nil {
			t.Fatalf("pool.Get() error = %v", err)
		}
		defer func() {
			if err := parser.Close(); err != nil {
				t.Fatalf("parser.Close() error = %v", err)
			}
		}()

		raw := cachedBindingsForKernelTest(t).SetImplementation("")
		if raw != int32(ffi.ErrKernelLocked) {
			t.Fatalf("native SetImplementation() after pool.Get() = %d, want %d", raw, ffi.ErrKernelLocked)
		}
	})
}

func TestParserPoolKernelLockRace(t *testing.T) {
	runKernelScenario(t, "set-pool-race", func(t *testing.T) {
		if err := SetKernel(""); err != nil {
			t.Fatalf("SetKernel(automatic) error = %v", err)
		}
		current := Kernel()

		start := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(2)

		setErr := make(chan error, 1)
		go func() {
			defer wg.Done()
			<-start
			setErr <- SetKernel(current)
		}()

		type poolResult struct {
			pool *ParserPool
			err  error
		}
		poolResultCh := make(chan poolResult, 1)
		go func() {
			defer wg.Done()
			<-start
			pool, err := NewParserPool()
			poolResultCh <- poolResult{pool: pool, err: err}
		}()

		close(start)
		wg.Wait()

		if err := <-setErr; err != nil && !errors.Is(err, ErrKernelLocked) {
			t.Fatalf("SetKernel() race error = %v, want nil or ErrKernelLocked", err)
		}
		result := <-poolResultCh
		if result.err != nil || result.pool == nil {
			t.Fatalf("NewParserPool() race = (%#v, %v), want non-nil pool and nil error", result.pool, result.err)
		}
	})
}

func runKernelScenario(t *testing.T, scenario string, child func(*testing.T)) {
	t.Helper()

	if activeScenario := os.Getenv(kernelScenarioEnv); activeScenario != "" {
		if activeScenario != scenario {
			t.Skipf("kernel scenario %q is active", activeScenario)
		}
		child(t)
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^"+t.Name()+"$", "-test.count=1")
	cmd.Env = kernelScenarioEnvironment(scenario)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("kernel scenario %q failed: %v\n%s", scenario, err, output)
	}
}

func kernelScenarioEnvironment(scenario string) []string {
	prefix := kernelScenarioEnv + "="
	env := make([]string, 0, len(os.Environ())+1)
	for _, value := range os.Environ() {
		if !strings.HasPrefix(value, prefix) {
			env = append(env, value)
		}
	}
	return append(env, prefix+scenario)
}

func cachedLibraryForKernelTest() *loadedLibrary {
	libraryMu.Lock()
	defer libraryMu.Unlock()
	return cachedLibrary
}

func cachedBindingsForKernelTest(t *testing.T) *ffi.Bindings {
	t.Helper()
	library := cachedLibraryForKernelTest()
	if library == nil || library.bindings == nil {
		t.Fatal("native library is not cached")
	}
	return library.bindings
}
