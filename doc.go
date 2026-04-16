package purejson

import (
	"runtime"
	"sync"

	"github.com/amikos-tech/pure-simdjson/internal/ffi"
)

// Doc wraps one live native document handle plus its cached root view.
type Doc struct {
	mu     sync.Mutex
	parser *Parser
	handle ffi.DocHandle
	root   ffi.ValueView
	closed bool
}

// Root returns the cached root element view for the live document.
func (d *Doc) Root() Element {
	return Element{doc: d, view: d.root}
}

// Close releases the native document and clears the owning parser's busy
// state. It is idempotent.
func (d *Doc) Close() error {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return nil
	}

	handle := d.handle
	parser := d.parser
	library := parser.library

	clearDocFinalizer(d)
	rc := library.bindings.DocFree(handle)
	runtime.KeepAlive(d)
	runtime.KeepAlive(d.parser)
	if err := wrapStatus(rc); err != nil {
		attachDocFinalizer(d)
		d.mu.Unlock()
		return err
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
	d.mu.Unlock()

	rc := library.bindings.DocFree(handle)
	docFreed := rc == int32(ffi.OK)
	if docFreed {
		docFinalizerCount.Add(1)
	}

	d.mu.Lock()
	if docFreed {
		d.closed = true
		d.handle = 0
	}
	d.mu.Unlock()

	if docFreed {
		parser.clearLiveDoc(handle)
	}
	return docFreed
}
