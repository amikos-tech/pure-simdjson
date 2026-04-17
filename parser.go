package purejson

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/amikos-tech/pure-simdjson/internal/ffi"
)

var (
	abiVersionOverride    atomic.Uint32
	abiVersionOverrideSet atomic.Bool
	parserFinalizerCount  atomic.Int64
	docFinalizerCount     atomic.Int64
)

// Parser owns one live native parser handle and enforces a one-document-at-a-
// time lifecycle.
type Parser struct {
	mu      sync.Mutex
	library *loadedLibrary
	handle  ffi.ParserHandle
	closed  bool
	liveDoc ffi.DocHandle
}

// NewParser resolves the local shared library, verifies the ABI, and allocates
// a reusable native parser.
func NewParser() (*Parser, error) {
	library, err := activeLibrary()
	if err != nil {
		return nil, err
	}

	actualABI, rc := library.bindings.ABI()
	if err := wrapStatus(rc); err != nil {
		return nil, err
	}

	expectedABI := expectedABIVersion()
	if actualABI != expectedABI {
		return nil, wrapABIMismatch(expectedABI, actualABI, library.path)
	}

	handle, rc := library.bindings.ParserNew()
	if err := wrapStatus(rc); err != nil {
		return nil, err
	}

	parser := &Parser{
		library: library,
		handle:  handle,
	}
	attachParserFinalizer(parser)
	return parser, nil
}

// Parse copies one JSON buffer into the native parser and returns a live Doc on
// success.
func (p *Parser) Parse(data []byte) (*Doc, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.library == nil {
		return nil, ErrInvalidHandle
	}
	if p.closed {
		return nil, ErrClosed
	}
	if p.liveDoc != 0 {
		return nil, ErrParserBusy
	}

	handle := p.handle
	library := p.library

	docHandle, rc := library.bindings.ParserParse(handle, data)
	runtime.KeepAlive(data)
	runtime.KeepAlive(p)
	if err := wrapParserStatus(library.bindings, handle, rc); err != nil {
		return nil, err
	}

	root, rc := library.bindings.DocRoot(docHandle)
	runtime.KeepAlive(p)
	if err := wrapStatus(rc); err != nil {
		if freeErr := wrapStatus(library.bindings.DocFree(docHandle)); freeErr != nil {
			err = errors.Join(err, freeErr)
		}
		return nil, err
	}

	doc := &Doc{
		parser: p,
		handle: docHandle,
		root:   root,
	}
	attachDocFinalizer(doc)

	p.liveDoc = docHandle
	return doc, nil
}

// Close releases the native parser. While a live document still belongs to the
// parser, Close returns ErrParserBusy and leaves the parser usable. Subsequent
// calls after a successful Close return nil.
func (p *Parser) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.library == nil {
		return ErrInvalidHandle
	}
	if p.closed {
		return nil
	}
	if p.liveDoc != 0 {
		return ErrParserBusy
	}

	handle := p.handle
	library := p.library

	clearParserFinalizer(p)
	rc := library.bindings.ParserFree(handle)
	runtime.KeepAlive(p)
	if err := wrapStatus(rc); err != nil {
		attachParserFinalizer(p)
		if leakWarningsEnabled() {
			fmt.Fprintf(os.Stderr, "purejson close-failed: parser %v\n", err)
		}
		return err
	}

	p.closed = true
	p.handle = 0
	p.liveDoc = 0
	return nil
}

func (p *Parser) clearLiveDoc(doc ffi.DocHandle) {
	p.mu.Lock()
	if p.liveDoc == doc {
		p.liveDoc = 0
	}
	p.mu.Unlock()
}

func (p *Parser) hasLeakedState() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return !p.closed && (p.handle != 0 || p.liveDoc != 0)
}

// finalizeLeaked frees native resources from the GC finalizer. It releases the
// mutex around FFI calls so a stuck native side cannot hold the runtime, then
// re-acquires it to commit only the state for handles that actually freed.
func (p *Parser) finalizeLeaked() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}

	handle := p.handle
	liveDoc := p.liveDoc
	library := p.library
	p.mu.Unlock()

	docFreed := liveDoc == 0
	if liveDoc != 0 {
		if rc := library.bindings.DocFree(liveDoc); rc == int32(ffi.OK) {
			docFinalizerCount.Add(1)
			docFreed = true
		}
	}

	parserFreed := handle == 0
	if handle != 0 {
		if rc := library.bindings.ParserFree(handle); rc == int32(ffi.OK) {
			parserFinalizerCount.Add(1)
			parserFreed = true
		}
	}

	p.mu.Lock()
	if docFreed {
		p.liveDoc = 0
	}
	if parserFreed {
		p.handle = 0
	}
	if p.handle == 0 && p.liveDoc == 0 {
		p.closed = true
	}
	p.mu.Unlock()
}

func expectedABIVersion() uint32 {
	if abiVersionOverrideSet.Load() {
		return abiVersionOverride.Load()
	}
	return ffi.ABIVersion
}
