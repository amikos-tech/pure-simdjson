package purejson

import (
	"runtime"

	"github.com/amikos-tech/pure-simdjson/internal/ffi"
)

// Element is the public value-view wrapper used for the Phase 3 happy path.
type Element struct {
	doc  *Doc
	view ffi.ValueView
}

// Array is the Phase 3 public array placeholder and wraps an Element view.
type Array struct{ Element }

// Object is the Phase 3 public object placeholder and wraps an Element view.
type Object struct{ Element }

// GetInt64 reads the current element as an int64 and returns ErrClosed when the
// owning document has already been released.
func (e Element) GetInt64() (int64, error) {
	if e.doc == nil || e.doc.isClosed() {
		return 0, ErrClosed
	}

	value, rc := e.doc.parser.library.bindings.ElementGetInt64(&e.view)
	runtime.KeepAlive(e.doc)
	if err := wrapStatus(rc); err != nil {
		return 0, err
	}
	return value, nil
}
