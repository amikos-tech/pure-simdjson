package purejson

import (
	"errors"
	"fmt"
	"strconv"
	"sync"
	"testing"
)

func TestParserPoolRoundTrip(t *testing.T) {
	pool := NewParserPool()

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
	pool := NewParserPool()
	if err := pool.Put(nil); !errors.Is(err, ErrInvalidHandle) {
		t.Fatalf("pool.Put(nil) error = %v, want ErrInvalidHandle", err)
	}
}

func TestParserPoolRejectsBusy(t *testing.T) {
	pool := NewParserPool()
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
	pool := NewParserPool()
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

	pool := NewParserPool()
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
	pool := NewParserPool()

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
