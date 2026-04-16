package purejson

import (
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

type Parser struct {
	mu      sync.Mutex
	library *loadedLibrary
	handle  ffi.ParserHandle
	closed  bool
	liveDoc ffi.DocHandle
}

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
	runtime.KeepAlive(parser)
	return parser, nil
}

func (p *Parser) Parse(data []byte) (*Doc, error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil, ErrClosed
	}

	handle := p.handle
	library := p.library
	p.mu.Unlock()

	docHandle, rc := library.bindings.ParserParse(handle, data)
	runtime.KeepAlive(data)
	runtime.KeepAlive(p)
	if err := wrapParserStatus(library.bindings, handle, rc); err != nil {
		return nil, err
	}

	root, rc := library.bindings.DocRoot(docHandle)
	runtime.KeepAlive(p)
	if err := wrapStatus(rc); err != nil {
		_ = library.bindings.DocFree(docHandle)
		return nil, err
	}

	doc := &Doc{
		parser: p,
		handle: docHandle,
		root:   root,
	}
	attachDocFinalizer(doc)

	p.mu.Lock()
	p.liveDoc = docHandle
	p.mu.Unlock()

	return doc, nil
}

func (p *Parser) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}

	handle := p.handle
	library := p.library
	p.mu.Unlock()

	clearParserFinalizer(p)
	rc := library.bindings.ParserFree(handle)
	runtime.KeepAlive(p)
	if err := wrapStatus(rc); err != nil {
		attachParserFinalizer(p)
		return err
	}

	p.mu.Lock()
	p.closed = true
	p.handle = 0
	p.liveDoc = 0
	p.mu.Unlock()
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

func (p *Parser) finalizeLeaked() bool {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return false
	}

	handle := p.handle
	liveDoc := p.liveDoc
	library := p.library

	p.closed = true
	p.handle = 0
	p.liveDoc = 0
	p.mu.Unlock()

	if liveDoc != 0 {
		if rc := library.bindings.DocFree(liveDoc); rc == int32(ffi.OK) {
			docFinalizerCount.Add(1)
		}
	}

	if handle != 0 {
		if rc := library.bindings.ParserFree(handle); rc == int32(ffi.OK) {
			parserFinalizerCount.Add(1)
			return true
		}
	}

	return false
}

func expectedABIVersion() uint32 {
	if abiVersionOverrideSet.Load() {
		return abiVersionOverride.Load()
	}
	return ffi.ABIVersion
}

func setExpectedABIVersionForTest(version uint32) func() {
	previousSet := abiVersionOverrideSet.Load()
	previousValue := abiVersionOverride.Load()

	abiVersionOverride.Store(version)
	abiVersionOverrideSet.Store(true)

	return func() {
		if previousSet {
			abiVersionOverride.Store(previousValue)
			abiVersionOverrideSet.Store(true)
			return
		}

		abiVersionOverride.Store(0)
		abiVersionOverrideSet.Store(false)
	}
}

func resetFinalizerCountsForTest() {
	parserFinalizerCount.Store(0)
	docFinalizerCount.Store(0)
}

func parserFinalizerCountForTest() int64 {
	return parserFinalizerCount.Load()
}

func docFinalizerCountForTest() int64 {
	return docFinalizerCount.Load()
}
