package purejson

import (
	"runtime"

	"github.com/amikos-tech/pure-simdjson/internal/ffi"
)

type Element struct {
	doc  *Doc
	view ffi.ValueView
}

type Array struct{ Element }

type Object struct{ Element }

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
