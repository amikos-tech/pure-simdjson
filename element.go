package purejson

import (
	"runtime"

	"github.com/amikos-tech/pure-simdjson/internal/ffi"
)

const maxExactFloat64Integer = 1 << 53

// Element is the public value-view wrapper for a document root or child value.
type Element struct {
	doc  *Doc
	view ffi.ValueView
}

// ElementType reports the concrete JSON value kind for an Element, preserving
// the distinct int64, uint64, and float64 classifications from simdjson's DOM.
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
// unverified instance.
type Array struct{ element Element }

// Object wraps an Element verified to represent a JSON object. Construct via
// Element.AsObject; the unexported field prevents callers from creating an
// unverified instance.
type Object struct{ element Element }

// GetInt64 reads the current element as an int64 and returns ErrClosed when the
// owning document has already been released. Uint64 values larger than max
// int64 report ErrNumberOutOfRange, while float-kind values report ErrWrongType.
// Element accessors are not safe for concurrent use with Doc.Close.
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
// the owning document has already been released. Negative integers report
// ErrNumberOutOfRange and non-uint64 kinds report ErrWrongType.
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
// the owning document has already been released. Large int64 and uint64 values
// that would lose precision report ErrPrecisionLoss instead of rounding.
func (e Element) GetFloat64() (float64, error) {
	if e.doc == nil || e.doc.isClosed() {
		return 0, ErrClosed
	}

	switch ffi.ValueKind(e.view.KindHint) {
	case ffi.ValueKindInt64:
		value, rc := e.doc.parser.library.bindings.ElementGetInt64(&e.view)
		runtime.KeepAlive(e.doc)
		if err := wrapStatus(rc); err != nil {
			return 0, err
		}
		if value < -maxExactFloat64Integer || value > maxExactFloat64Integer {
			return 0, wrapStatus(int32(ffi.ErrPrecisionLoss))
		}
		return float64(value), nil
	case ffi.ValueKindUint64:
		value, rc := e.doc.parser.library.bindings.ElementGetUint64(&e.view)
		runtime.KeepAlive(e.doc)
		if err := wrapStatus(rc); err != nil {
			return 0, err
		}
		if value > maxExactFloat64Integer {
			return 0, wrapStatus(int32(ffi.ErrPrecisionLoss))
		}
		return float64(value), nil
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

// Iter returns a scanner-style iterator over the array contents in document
// order.
func (a Array) Iter() *ArrayIter {
	it := &ArrayIter{doc: a.element.doc}
	if a.element.doc == nil || a.element.doc.isClosed() {
		it.err = ErrClosed
		return it
	}
	if ffi.ValueKind(a.element.view.KindHint) != ffi.ValueKindArray {
		it.err = ErrWrongType
		return it
	}

	iter, rc := a.element.doc.parser.library.bindings.ArrayIterNew(&a.element.view)
	runtime.KeepAlive(a.element.doc)
	if err := normalizeIteratorError(a.element.doc, rc); err != nil {
		it.err = err
		return it
	}

	it.iter = iter
	return it
}

// Iter returns a scanner-style iterator over the object fields in document
// order.
func (o Object) Iter() *ObjectIter {
	it := &ObjectIter{doc: o.element.doc}
	if o.element.doc == nil || o.element.doc.isClosed() {
		it.err = ErrClosed
		return it
	}
	if ffi.ValueKind(o.element.view.KindHint) != ffi.ValueKindObject {
		it.err = ErrWrongType
		return it
	}

	iter, rc := o.element.doc.parser.library.bindings.ObjectIterNew(&o.element.view)
	runtime.KeepAlive(o.element.doc)
	if err := normalizeIteratorError(o.element.doc, rc); err != nil {
		it.err = err
		return it
	}

	it.iter = iter
	return it
}

// GetField returns the element for the given object key. Missing fields return
// ErrElementNotFound, while present null fields return a valid Element whose
// IsNull method reports true.
func (o Object) GetField(key string) (Element, error) {
	if o.element.doc == nil || o.element.doc.isClosed() {
		return Element{}, ErrClosed
	}
	if ffi.ValueKind(o.element.view.KindHint) != ffi.ValueKindObject {
		return Element{}, ErrWrongType
	}

	view, rc := o.element.doc.parser.library.bindings.ObjectGetField(&o.element.view, key)
	runtime.KeepAlive(o.element.doc)
	if err := normalizeIteratorError(o.element.doc, rc); err != nil {
		return Element{}, err
	}

	return Element{doc: o.element.doc, view: view}, nil
}

// GetStringField returns the named field as a copied Go string using the same
// semantics as GetField followed by Element.GetString, including
// ErrElementNotFound for missing fields and ErrWrongType for present non-string
// values.
func (o Object) GetStringField(name string) (string, error) {
	field, err := o.GetField(name)
	if err != nil {
		return "", err
	}
	return field.GetString()
}
