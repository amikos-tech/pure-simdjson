package purejson

import (
	"runtime"
	"sync"

	"github.com/amikos-tech/pure-simdjson/internal/ffi"
)

type Doc struct {
	mu     sync.Mutex
	parser *Parser
	handle ffi.DocHandle
	root   ffi.ValueView
	closed bool
}

func (d *Doc) Root() Element {
	return Element{doc: d, view: d.root}
}

func (d *Doc) Close() error {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return nil
	}

	handle := d.handle
	parser := d.parser
	library := parser.library
	d.mu.Unlock()

	clearDocFinalizer(d)
	rc := library.bindings.DocFree(handle)
	runtime.KeepAlive(d)
	runtime.KeepAlive(d.parser)
	if err := wrapStatus(rc); err != nil {
		attachDocFinalizer(d)
		return err
	}

	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return nil
	}
	d.closed = true
	d.handle = 0
	d.mu.Unlock()

	parser.clearLiveDoc(handle)
	return nil
}

func (d *Doc) isClosed() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.closed
}

func (d *Doc) hasLeakedState() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return !d.closed && d.handle != 0
}

func (d *Doc) finalizeLeaked() bool {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return false
	}

	handle := d.handle
	parser := d.parser
	library := parser.library

	d.closed = true
	d.handle = 0
	d.mu.Unlock()

	rc := library.bindings.DocFree(handle)
	if rc == int32(ffi.OK) {
		docFinalizerCount.Add(1)
	}

	parser.clearLiveDoc(handle)
	return rc == int32(ffi.OK)
}
