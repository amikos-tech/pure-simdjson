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

// ElementType reports the concrete JSON value kind for an Element.
type ElementType uint32

const (
	// TypeInvalid reports a closed, invalid, or otherwise unusable element view.
	TypeInvalid ElementType = ElementType(ffi.ValueKindInvalid)
	// TypeNull reports a JSON null value.
	TypeNull ElementType = ElementType(ffi.ValueKindNull)
	// TypeBool reports a JSON boolean value.
	TypeBool ElementType = ElementType(ffi.ValueKindBool)
	// TypeInt64 reports a JSON number classified as int64.
	TypeInt64 ElementType = ElementType(ffi.ValueKindInt64)
	// TypeUint64 reports a JSON number classified as uint64.
	TypeUint64 ElementType = ElementType(ffi.ValueKindUint64)
	// TypeFloat64 reports a JSON number classified as float64.
	TypeFloat64 ElementType = ElementType(ffi.ValueKindFloat64)
	// TypeString reports a JSON string value.
	TypeString ElementType = ElementType(ffi.ValueKindString)
	// TypeArray reports a JSON array value.
	TypeArray ElementType = ElementType(ffi.ValueKindArray)
	// TypeObject reports a JSON object value.
	TypeObject ElementType = ElementType(ffi.ValueKindObject)
)

// Array wraps an Element verified to represent a JSON array. Construct via
// Element.AsArray; the unexported field prevents callers from creating an
// unverified instance. Traversal methods will be added in a later phase.
type Array struct{ element Element }

// Object wraps an Element verified to represent a JSON object. Construct via
// Element.AsObject; the unexported field prevents callers from creating an
// unverified instance. Traversal methods will be added in a later phase.
type Object struct{ element Element }

// GetInt64 reads the current element as an int64 and returns ErrClosed when the
// owning document has already been released. Element accessors are not safe for
// concurrent use with Doc.Close.
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

// Type reports the concrete JSON value kind for the current element. Closed,
// invalid, or tampered views collapse to TypeInvalid instead of returning an
// error.
func (e Element) Type() ElementType {
	if e.doc == nil || e.doc.isClosed() {
		return TypeInvalid
	}

	kind, rc := e.doc.parser.library.bindings.ElementType(&e.view)
	runtime.KeepAlive(e.doc)
	if rc != int32(ffi.OK) {
		return TypeInvalid
	}

	switch ffi.ValueKind(kind) {
	case ffi.ValueKindNull:
		return TypeNull
	case ffi.ValueKindBool:
		return TypeBool
	case ffi.ValueKindInt64:
		return TypeInt64
	case ffi.ValueKindUint64:
		return TypeUint64
	case ffi.ValueKindFloat64:
		return TypeFloat64
	case ffi.ValueKindString:
		return TypeString
	case ffi.ValueKindArray:
		return TypeArray
	case ffi.ValueKindObject:
		return TypeObject
	default:
		return TypeInvalid
	}
}

// GetUint64 reads the current element as a uint64 and returns ErrClosed when
// the owning document has already been released.
func (e Element) GetUint64() (uint64, error) {
	if e.doc == nil || e.doc.isClosed() {
		return 0, ErrClosed
	}

	value, rc := e.doc.parser.library.bindings.ElementGetUint64(&e.view)
	runtime.KeepAlive(e.doc)
	if err := wrapStatus(rc); err != nil {
		return 0, err
	}
	return value, nil
}

// GetFloat64 reads the current element as a float64 and returns ErrClosed when
// the owning document has already been released.
func (e Element) GetFloat64() (float64, error) {
	if e.doc == nil || e.doc.isClosed() {
		return 0, ErrClosed
	}

	value, rc := e.doc.parser.library.bindings.ElementGetFloat64(&e.view)
	runtime.KeepAlive(e.doc)
	if err := wrapStatus(rc); err != nil {
		return 0, err
	}
	return value, nil
}

// GetString reads the current element as a copied Go string and returns
// ErrClosed when the owning document has already been released.
func (e Element) GetString() (string, error) {
	if e.doc == nil || e.doc.isClosed() {
		return "", ErrClosed
	}

	value, rc := e.doc.parser.library.bindings.ElementGetString(&e.view)
	runtime.KeepAlive(e.doc)
	if err := wrapStatus(rc); err != nil {
		return "", err
	}
	return value, nil
}

// GetBool reads the current element as a bool and returns ErrClosed when the
// owning document has already been released.
func (e Element) GetBool() (bool, error) {
	if e.doc == nil || e.doc.isClosed() {
		return false, ErrClosed
	}

	value, rc := e.doc.parser.library.bindings.ElementGetBool(&e.view)
	runtime.KeepAlive(e.doc)
	if err := wrapStatus(rc); err != nil {
		return false, err
	}
	return value, nil
}

// IsNull reports whether the current element is a JSON null value. Closed,
// invalid, or tampered views return false.
func (e Element) IsNull() bool {
	if e.doc == nil || e.doc.isClosed() {
		return false
	}

	value, rc := e.doc.parser.library.bindings.ElementIsNull(&e.view)
	runtime.KeepAlive(e.doc)
	if rc != int32(ffi.OK) {
		return false
	}
	return value
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
