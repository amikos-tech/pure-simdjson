package purejson

import (
	"fmt"
	"sync"
)

// ParserPool reuses Parser instances across goroutines while preserving the
// one-live-doc-per-parser invariant. There is no Close method: sync.Pool
// cannot be drained deterministically. Parsers left in the pool when it is
// discarded are reclaimed by the GC finalizer; the same leak-warning rules
// that apply to standalone parsers apply here.
type ParserPool struct {
	pool   sync.Pool
	config parserConfig
}

// NewParserPool validates immutable parser options and constructs an empty
// parser pool. Native library resolution is deferred until the first Get miss.
func NewParserPool(opts ...ParserOption) (*ParserPool, error) {
	config, err := normalizeParserOptions(opts...)
	if err != nil {
		return nil, err
	}
	lockKernelSelection()
	return &ParserPool{config: config}, nil
}

// Get returns a reusable parser or allocates a new one on a pool miss.
func (p *ParserPool) Get() (*Parser, error) {
	if value := p.pool.Get(); value != nil {
		if parser, ok := value.(*Parser); ok {
			return parser, nil
		}
	}

	return newParserWithConfig(p.config)
}

// Put returns a parser to the pool and rejects nil, closed, still-busy, or
// differently configured parsers instead of silently repairing misuse. The
// parser's mutex is held across the pool insert so a racing Close cannot stash
// a just-closed parser.
func (p *ParserPool) Put(parser *Parser) error {
	if parser == nil {
		return ErrInvalidHandle
	}

	parser.mu.Lock()
	defer parser.mu.Unlock()

	switch {
	case parser.closed:
		return ErrClosed
	case parser.liveDoc != 0:
		return ErrParserBusy
	case parser.config != p.config:
		return fmt.Errorf("%w: parser configuration does not match pool", ErrInvalidOption)
	default:
		p.pool.Put(parser)
		return nil
	}
}
