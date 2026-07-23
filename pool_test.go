package purejson

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
)

func mustNewParserPool(t *testing.T, opts ...ParserOption) *ParserPool {
	t.Helper()

	pool, err := NewParserPool(opts...)
	if err != nil {
		t.Fatalf("NewParserPool() error = %v", err)
	}
	return pool
}

func TestParserPoolOptionConstructorSignature(t *testing.T) {
	var constructor func(...ParserOption) (*ParserPool, error) = NewParserPool
	_ = constructor
}

func TestParserPoolOptionValidation(t *testing.T) {
	restore := withLibraryCacheClearedForTest(t)
	defer restore()

	t.Setenv(libraryEnvPath, filepath.Join(t.TempDir(), "missing-native-library"))
	pool, err := NewParserPool(ParserOption{})
	if pool != nil {
		t.Fatal("NewParserPool(ParserOption{}) returned a pool")
	}
	if !errors.Is(err, ErrInvalidOption) {
		t.Fatalf("NewParserPool(ParserOption{}) error = %v, want ErrInvalidOption", err)
	}
	if cachedLibrary != nil {
		t.Fatal("NewParserPool(ParserOption{}) loaded a native library")
	}
}

func TestParserPoolConstructionDoesNotLoad(t *testing.T) {
	restore := withLibraryCacheClearedForTest(t)
	defer restore()

	t.Setenv(libraryEnvPath, filepath.Join(t.TempDir(), "missing-native-library"))
	pool, err := NewParserPool(WithMaxCapacity(32), WithMaxDepth(4))
	if err != nil {
		t.Fatalf("NewParserPool() error = %v", err)
	}
	if cachedLibrary != nil {
		t.Fatal("NewParserPool() loaded a native library")
	}

	parser, err := pool.Get()
	if parser != nil {
		_ = parser.Close()
		t.Fatal("pool.Get() with missing library returned a parser")
	}
	if !errors.Is(err, errLoadLibrary) {
		t.Fatalf("pool.Get() error = %v, want library load failure on first miss", err)
	}
}

func TestParserPoolConfigDefaultsEquivalent(t *testing.T) {
	omitted := mustNewParserPool(t)
	explicit := mustNewParserPool(t, WithMaxCapacity(0), WithMaxDepth(0))

	if omitted.config != explicit.config {
		t.Fatalf("omitted config = %+v, explicit config = %+v; want equal", omitted.config, explicit.config)
	}
	if omitted.config != defaultParserConfig {
		t.Fatalf("pool default config = %+v, want %+v", omitted.config, defaultParserConfig)
	}
}

func TestParserPoolConfigAppliedOnMiss(t *testing.T) {
	pool := mustNewParserPool(t, WithMaxCapacity(32), WithMaxDepth(4))
	parser, err := pool.Get()
	if err != nil {
		t.Fatalf("pool.Get() error = %v", err)
	}
	if parser.config != pool.config {
		t.Fatalf("parser config = %+v, pool config = %+v; want equal", parser.config, pool.config)
	}
	if err := parser.Close(); err != nil {
		t.Fatalf("parser.Close() error = %v", err)
	}
}

func TestParserPoolConfigRejectsMismatchedPut(t *testing.T) {
	testCases := []struct {
		name string
		opts []ParserOption
	}{
		{name: "capacity", opts: []ParserOption{WithMaxCapacity(96), WithMaxDepth(8)}},
		{name: "depth", opts: []ParserOption{WithMaxCapacity(64), WithMaxDepth(9)}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pool := mustNewParserPool(t, WithMaxCapacity(64), WithMaxDepth(8))
			foreign, err := NewParser(tc.opts...)
			if err != nil {
				t.Fatalf("NewParser(foreign config) error = %v", err)
			}

			if err := pool.Put(foreign); !errors.Is(err, ErrInvalidOption) {
				t.Fatalf("pool.Put(mismatched parser) error = %v, want ErrInvalidOption", err)
			}

			got, err := pool.Get()
			if err != nil {
				t.Fatalf("pool.Get() after rejected Put error = %v", err)
			}
			if got == foreign {
				t.Fatal("pool.Get() returned the parser rejected by Put")
			}
			if got.config != pool.config {
				t.Fatalf("pool.Get() config = %+v, pool config = %+v; want equal", got.config, pool.config)
			}

			if err := got.Close(); err != nil {
				t.Fatalf("pooled parser Close() error = %v", err)
			}
			if err := foreign.Close(); err != nil {
				t.Fatalf("foreign parser Close() error = %v", err)
			}
		})
	}
}

func TestParserPoolReuseRoundTrip(t *testing.T) {
	pool := mustNewParserPool(t)

	firstDone := make(chan struct{})
	errs := make(chan error, 2)

	go func() {
		parser, err := pool.Get()
		if err != nil {
			errs <- err
			return
		}

		doc, err := parser.Parse([]byte("42"))
		if err == nil {
			_, err = doc.Root().GetInt64()
		}
		if err == nil {
			err = doc.Close()
		}
		if err == nil {
			err = pool.Put(parser)
		}
		errs <- err
		close(firstDone)
	}()

	<-firstDone

	go func() {
		parser, err := pool.Get()
		if err != nil {
			errs <- err
			return
		}

		doc, err := parser.Parse([]byte("43"))
		if err == nil {
			_, err = doc.Root().GetInt64()
		}
		if err == nil {
			err = doc.Close()
		}
		if err == nil {
			err = pool.Put(parser)
		}
		errs <- err
	}()

	for i := 0; i < 2; i++ {
		if err := <-errs; err != nil {
			t.Fatalf("goroutine %d error = %v", i, err)
		}
	}
}

func TestParserPoolRejectsNil(t *testing.T) {
	pool := mustNewParserPool(t)
	if err := pool.Put(nil); !errors.Is(err, ErrInvalidHandle) {
		t.Fatalf("pool.Put(nil) error = %v, want ErrInvalidHandle", err)
	}
}

func TestParserPoolRejectsBusy(t *testing.T) {
	pool := mustNewParserPool(t)
	parser, err := pool.Get()
	if err != nil {
		t.Fatalf("pool.Get() error = %v", err)
	}

	doc, err := parser.Parse([]byte("42"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if err := pool.Put(parser); !errors.Is(err, ErrParserBusy) {
		t.Fatalf("pool.Put(busy parser) error = %v, want ErrParserBusy", err)
	}

	if err := doc.Close(); err != nil {
		t.Fatalf("doc.Close() error = %v", err)
	}
	if err := parser.Close(); err != nil {
		t.Fatalf("parser.Close() error = %v", err)
	}
}

func TestParserPoolRejectsClosed(t *testing.T) {
	pool := mustNewParserPool(t)
	parser, err := pool.Get()
	if err != nil {
		t.Fatalf("pool.Get() error = %v", err)
	}

	if err := parser.Close(); err != nil {
		t.Fatalf("parser.Close() error = %v", err)
	}
	if err := pool.Put(parser); !errors.Is(err, ErrClosed) {
		t.Fatalf("pool.Put(closed parser) error = %v, want ErrClosed", err)
	}
}

func TestPooledParserEvictionCleansUp(t *testing.T) {
	resetFinalizerCountsForTest()

	pool := mustNewParserPool(t)
	parser, err := pool.Get()
	if err != nil {
		t.Fatalf("pool.Get() error = %v", err)
	}

	if err := pool.Put(parser); err != nil {
		t.Fatalf("pool.Put() error = %v", err)
	}
	parser = nil

	waitForFinalizers(t, func() bool {
		return parserFinalizerCountForTest() >= 1
	})
}

func TestParserPoolConcurrentGetParsePut(t *testing.T) {
	pool := mustNewParserPool(t)

	const goroutines = 12
	const iterations = 25

	var (
		wg    sync.WaitGroup
		mu    sync.Mutex
		inUse = make(map[*Parser]int)
	)

	errs := make(chan error, goroutines)
	start := make(chan struct{})

	for worker := 0; worker < goroutines; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			<-start

			for iteration := 0; iteration < iterations; iteration++ {
				parser, err := pool.Get()
				if err != nil {
					errs <- fmt.Errorf("worker %d pool.Get(): %w", worker, err)
					return
				}

				mu.Lock()
				if previous, exists := inUse[parser]; exists {
					mu.Unlock()
					errs <- fmt.Errorf("parser %p reused concurrently by workers %d and %d", parser, previous, worker)
					return
				}
				inUse[parser] = worker
				mu.Unlock()

				doc, err := parser.Parse([]byte(strconv.Itoa(worker*iterations + iteration)))
				if err == nil {
					_, err = doc.Root().GetInt64()
				}
				if err == nil {
					err = doc.Close()
				}

				mu.Lock()
				delete(inUse, parser)
				mu.Unlock()

				if err == nil {
					err = pool.Put(parser)
				}

				if err != nil {
					errs <- fmt.Errorf("worker %d iteration %d error: %w", worker, iteration, err)
					return
				}
			}
		}(worker)
	}

	close(start)
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
}
