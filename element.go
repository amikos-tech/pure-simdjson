package purejson

import (
	"math/bits"
	"runtime"

	"github.com/amikos-tech/pure-simdjson/internal/ffi"
)

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

func exactFloat64Uint64(value uint64) bool {
	if value == 0 {
		return true
	}
	significant := value >> bits.TrailingZeros64(value)
	return bits.Len64(significant) <= 53
}

func exactFloat64Int64(value int64) bool {
	magnitude := uint64(value)
	if value < 0 {
		magnitude = uint64(^value) + 1
	}
	return exactFloat64Uint64(magnitude)
}

func (e Element) usableDoc() (*Doc, error) {
	if e.doc == nil {
		return nil, ErrInvalidHandle
	}
	if e.doc.parser == nil || e.doc.parser.library == nil || e.doc.parser.library.bindings == nil {
		return nil, ErrInvalidHandle
	}
	if e.doc.isClosed() {
		return nil, ErrClosed
	}
	return e.doc, nil
}

// GetInt64 reads the current element as an int64 and returns ErrClosed when the
// owning document has already been released. Uint64 values larger than max
// int64 report ErrNumberOutOfRange, float-kind values report ErrWrongType, and
// native BIGINT classifications report ErrPrecisionLoss. Element accessors are
// not safe for concurrent use with Doc.Close.
func (e Element) GetInt64() (int64, error) {
	doc, err := e.usableDoc()
	if err != nil {
		return 0, err
	}

	value, rc := doc.parser.library.bindings.ElementGetInt64(&e.view)
	runtime.KeepAlive(doc)
	if err := wrapStatus(rc); err != nil {
		return 0, err
	}
	return value, nil
}

// Type reports the concrete JSON value kind for the current element. Closed,
// invalid, or tampered views collapse to TypeInvalid instead of returning an
// error.
func (e Element) Type() ElementType {
	kind, err := e.TypeErr()
	if err != nil {
		return TypeInvalid
	}
	return kind
}

// TypeErr reports the concrete JSON value kind for the current element while
// preserving native failures such as ErrClosed, ErrInvalidHandle,
// ErrPrecisionLoss, or ErrPanic.
func (e Element) TypeErr() (ElementType, error) {
	doc, err := e.usableDoc()
	if err != nil {
		return TypeInvalid, err
	}

	kind, rc := doc.parser.library.bindings.ElementType(&e.view)
	runtime.KeepAlive(doc)
	if err := wrapStatus(rc); err != nil {
		return TypeInvalid, err
	}

	switch ffi.ValueKind(kind) {
	case ffi.ValueKindNull:
		return TypeNull, nil
	case ffi.ValueKindBool:
		return TypeBool, nil
	case ffi.ValueKindInt64:
		return TypeInt64, nil
	case ffi.ValueKindUint64:
		return TypeUint64, nil
	case ffi.ValueKindFloat64:
		return TypeFloat64, nil
	case ffi.ValueKindString:
		return TypeString, nil
	case ffi.ValueKindArray:
		return TypeArray, nil
	case ffi.ValueKindObject:
		return TypeObject, nil
	default:
		return TypeInvalid, nil
	}
}

// GetUint64 reads the current element as a uint64 and returns ErrClosed when
// the owning document has already been released. Negative integers report
// ErrNumberOutOfRange, non-uint64 kinds report ErrWrongType, and native BIGINT
// classifications report ErrPrecisionLoss.
func (e Element) GetUint64() (uint64, error) {
	doc, err := e.usableDoc()
	if err != nil {
		return 0, err
	}

	value, rc := doc.parser.library.bindings.ElementGetUint64(&e.view)
	runtime.KeepAlive(doc)
	if err := wrapStatus(rc); err != nil {
		return 0, err
	}
	return value, nil
}

// GetFloat64 reads the current element as a float64 and returns ErrClosed when
// the owning document has already been released. Large int64 and uint64 values
// that would lose precision report ErrPrecisionLoss instead of rounding.
func (e Element) GetFloat64() (float64, error) {
	doc, err := e.usableDoc()
	if err != nil {
		return 0, err
	}

	// Int64/Uint64 hints stay on the integer accessors so Go can enforce the
	// exact-float64 contract without routing integer-backed values through the
	// native float accessor. Invalid hints still fall through to Rust, which
	// re-checks the native kind before applying the same precision-loss rule.
	switch ffi.ValueKind(e.view.KindHint) {
	case ffi.ValueKindInt64:
		value, rc := doc.parser.library.bindings.ElementGetInt64(&e.view)
		runtime.KeepAlive(doc)
		if err := wrapStatus(rc); err != nil {
			return 0, err
		}
		if !exactFloat64Int64(value) {
			return 0, wrapStatus(int32(ffi.ErrPrecisionLoss))
		}
		return float64(value), nil
	case ffi.ValueKindUint64:
		value, rc := doc.parser.library.bindings.ElementGetUint64(&e.view)
		runtime.KeepAlive(doc)
		if err := wrapStatus(rc); err != nil {
			return 0, err
		}
		if !exactFloat64Uint64(value) {
			return 0, wrapStatus(int32(ffi.ErrPrecisionLoss))
		}
		return float64(value), nil
	}

	value, rc := doc.parser.library.bindings.ElementGetFloat64(&e.view)
	runtime.KeepAlive(doc)
	if err := wrapStatus(rc); err != nil {
		return 0, err
	}
	return value, nil
}

// GetString reads the current element as a copied Go string and returns
// ErrClosed when the owning document has already been released.
func (e Element) GetString() (string, error) {
	doc, err := e.usableDoc()
	if err != nil {
		return "", err
	}

	value, rc := doc.parser.library.bindings.ElementGetString(&e.view)
	runtime.KeepAlive(doc)
	if err := wrapStatus(rc); err != nil {
		return "", err
	}
	return value, nil
}

// GetBool reads the current element as a bool and returns ErrClosed when the
// owning document has already been released.
func (e Element) GetBool() (bool, error) {
	doc, err := e.usableDoc()
	if err != nil {
		return false, err
	}

	value, rc := doc.parser.library.bindings.ElementGetBool(&e.view)
	runtime.KeepAlive(doc)
	if err := wrapStatus(rc); err != nil {
		return false, err
	}
	return value, nil
}

// IsNull reports whether the current element is a JSON null value. Closed,
// invalid, or tampered views return false.
func (e Element) IsNull() bool {
	value, err := e.IsNullErr()
	if err != nil {
		return false
	}
	return value
}

// IsNullErr reports whether the current element is a JSON null value while
// preserving native failures such as ErrClosed or ErrInvalidHandle.
func (e Element) IsNullErr() (bool, error) {
	doc, err := e.usableDoc()
	if err != nil {
		return false, err
	}

	value, rc := doc.parser.library.bindings.ElementIsNull(&e.view)
	runtime.KeepAlive(doc)
	if err := wrapStatus(rc); err != nil {
		return false, err
	}
	return value, nil
}

// AsArray returns a typed Array view when the element represents a JSON array.
// Returns ErrClosed when the owning document is released and ErrWrongType when
// the underlying value kind is not an array.
func (e Element) AsArray() (Array, error) {
	if _, err := e.usableDoc(); err != nil {
		return Array{}, err
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
	if _, err := e.usableDoc(); err != nil {
		return Object{}, err
	}
	if ffi.ValueKind(e.view.KindHint) != ffi.ValueKindObject {
		return Object{}, ErrWrongType
	}
	return Object{element: e}, nil
}

// Iter returns a scanner-style iterator over the array contents in document
// order. Successful descendant traversal records document-tied bookkeeping in
// the native registry that is released when Doc.Close runs.
func (a Array) Iter() *ArrayIter {
	it := &ArrayIter{doc: a.element.doc}
	doc, err := a.element.usableDoc()
	if err != nil {
		it.err = err
		return it
	}
	if ffi.ValueKind(a.element.view.KindHint) != ffi.ValueKindArray {
		it.err = ErrWrongType
		return it
	}

	iter, rc := doc.parser.library.bindings.ArrayIterNew(&a.element.view)
	runtime.KeepAlive(doc)
	if err := normalizeIteratorError(doc, rc); err != nil {
		it.err = err
		return it
	}

	it.iter = iter
	it.doc = doc
	return it
}

// Iter returns a scanner-style iterator over the object fields in document
// order. Successful descendant traversal records document-tied bookkeeping in
// the native registry that is released when Doc.Close runs.
func (o Object) Iter() *ObjectIter {
	it := &ObjectIter{doc: o.element.doc}
	doc, err := o.element.usableDoc()
	if err != nil {
		it.err = err
		return it
	}
	if ffi.ValueKind(o.element.view.KindHint) != ffi.ValueKindObject {
		it.err = ErrWrongType
		return it
	}

	iter, rc := doc.parser.library.bindings.ObjectIterNew(&o.element.view)
	runtime.KeepAlive(doc)
	if err := normalizeIteratorError(doc, rc); err != nil {
		it.err = err
		return it
	}

	it.iter = iter
	it.doc = doc
	return it
}

// GetField returns the element for the given object key. Missing fields return
// ErrElementNotFound, while present null fields return a valid Element whose
// IsNull method reports true. When duplicate keys are present, GetField returns
// the first matching field. Successful lookups also record document-tied
// descendant bookkeeping in the native registry that is released when
// Doc.Close runs.
func (o Object) GetField(key string) (Element, error) {
	doc, err := o.element.usableDoc()
	if err != nil {
		return Element{}, err
	}
	if ffi.ValueKind(o.element.view.KindHint) != ffi.ValueKindObject {
		return Element{}, ErrWrongType
	}

	view, rc := doc.parser.library.bindings.ObjectGetField(&o.element.view, key)
	runtime.KeepAlive(doc)
	if err := normalizeIteratorError(doc, rc); err != nil {
		return Element{}, err
	}

	return Element{doc: doc, view: view}, nil
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
