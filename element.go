package purejson

import (
	"runtime"

	"github.com/amikos-tech/pure-simdjson/internal/ffi"
)

// Element is the public value-view wrapper for a document root or child value.
type Element struct {
	doc  *Doc
	view ffi.ValueView
}

// Array wraps an Element verified to represent a JSON array. Construct via
// Element.AsArray; the unexported field prevents callers from creating an
// unverified instance.
type Array struct{ element Element }

// Object wraps an Element verified to represent a JSON object. Construct via
// Element.AsObject; the unexported field prevents callers from creating an
// unverified instance.
type Object struct{ element Element }

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

// AsArray returns a typed Array view when the element represents a JSON array.
// Returns ErrClosed when the owning document is released and ErrWrongType when
// the underlying value kind is not an array.
func (e Element) AsArray() (Array, error) {
	if e.doc == nil || e.doc.isClosed() {
		return Array{}, ErrClosed
	}
	if ffi.ValueKind(e.view.KindHint) != ffi.ValueKindArray {
		return Array{}, ErrWrongType
	}
	return Array{element: e}, nil
}

// AsObject returns a typed Object view when the element represents a JSON
// object. Returns ErrClosed when the owning document is released and
// ErrWrongType when the underlying value kind is not an object.
func (e Element) AsObject() (Object, error) {
	if e.doc == nil || e.doc.isClosed() {
		return Object{}, ErrClosed
	}
	if ffi.ValueKind(e.view.KindHint) != ffi.ValueKindObject {
		return Object{}, ErrWrongType
	}
	return Object{element: e}, nil
}
