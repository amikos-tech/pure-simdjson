package purejson

import (
	"errors"
	"runtime"

	"github.com/amikos-tech/pure-simdjson/internal/ffi"
)

// ArrayIter scans array values one element at a time in document order.
type ArrayIter struct {
	doc          *Doc
	iter         ffi.ArrayIter
	currentValue Element
	done         bool
	err          error
}

// ObjectIter scans object entries one field at a time in document order. Keys
// are exposed as copied Go strings.
type ObjectIter struct {
	doc          *Doc
	iter         ffi.ObjectIter
	currentValue Element
	currentKey   string
	done         bool
	err          error
}

// Next advances the iterator and reports whether another value is available. It
// returns false after the iterator is exhausted or when Err reports a terminal
// failure.
func (it *ArrayIter) Next() bool {
	if it == nil || it.done || it.err != nil {
		return false
	}
	if it.doc == nil || it.doc.isClosed() {
		it.currentValue = Element{}
		it.done = true
		it.err = ErrClosed
		return false
	}

	value, done, rc := it.doc.parser.library.bindings.ArrayIterNext(&it.iter)
	runtime.KeepAlive(it.doc)
	if err := normalizeIteratorError(it.doc, rc); err != nil {
		it.currentValue = Element{}
		it.done = true
		it.err = err
		return false
	}
	if done {
		it.currentValue = Element{}
		it.done = true
		return false
	}

	it.currentValue = Element{doc: it.doc, view: value}
	return true
}

// Value returns the current array element after Next reports true.
func (it *ArrayIter) Value() Element {
	if it == nil {
		return Element{}
	}
	return it.currentValue
}

// Err reports the terminal iterator error, if any.
func (it *ArrayIter) Err() error {
	if it == nil {
		return nil
	}
	return it.err
}

// Next advances the iterator and reports whether another object entry is
// available. It caches the current key as a copied Go string for Key and the
// current value view for Value.
func (it *ObjectIter) Next() bool {
	if it == nil || it.done || it.err != nil {
		return false
	}
	if it.doc == nil || it.doc.isClosed() {
		it.currentValue = Element{}
		it.currentKey = ""
		it.done = true
		it.err = ErrClosed
		return false
	}

	keyView, valueView, done, rc := it.doc.parser.library.bindings.ObjectIterNext(&it.iter)
	runtime.KeepAlive(it.doc)
	if err := normalizeIteratorError(it.doc, rc); err != nil {
		it.currentValue = Element{}
		it.currentKey = ""
		it.done = true
		it.err = err
		return false
	}
	if done {
		it.currentValue = Element{}
		it.currentKey = ""
		it.done = true
		return false
	}

	key, rc := it.doc.parser.library.bindings.ElementGetString(&keyView)
	runtime.KeepAlive(it.doc)
	if err := normalizeIteratorError(it.doc, rc); err != nil {
		it.currentValue = Element{}
		it.currentKey = ""
		it.done = true
		it.err = err
		return false
	}

	it.currentKey = key
	it.currentValue = Element{doc: it.doc, view: valueView}
	return true
}

// Key returns the current object key after Next reports true.
func (it *ObjectIter) Key() string {
	if it == nil {
		return ""
	}
	return it.currentKey
}

// Value returns the current object value after Next reports true.
func (it *ObjectIter) Value() Element {
	if it == nil {
		return Element{}
	}
	return it.currentValue
}

// Err reports the terminal iterator error, if any.
func (it *ObjectIter) Err() error {
	if it == nil {
		return nil
	}
	return it.err
}

func normalizeIteratorError(doc *Doc, code int32) error {
	err := wrapStatus(code)
	if err == nil {
		return nil
	}
	if doc != nil && doc.isClosed() && errors.Is(err, ErrInvalidHandle) {
		return ErrClosed
	}
	return err
}
