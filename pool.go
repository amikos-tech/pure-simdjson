package purejson

import "sync"

// ParserPool reuses Parser instances across goroutines while preserving the
// one-live-doc-per-parser invariant.
type ParserPool struct {
	pool sync.Pool
}

// NewParserPool constructs an empty parser pool.
func NewParserPool() *ParserPool {
	return &ParserPool{}
}

// Get returns a reusable parser or allocates a new one on a pool miss.
func (p *ParserPool) Get() (*Parser, error) {
	if value := p.pool.Get(); value != nil {
		if parser, ok := value.(*Parser); ok && parser != nil {
			return parser, nil
		}
	}

	return NewParser()
}

// Put returns a parser to the pool and rejects nil, closed, or still-busy
// parsers instead of silently repairing misuse.
func (p *ParserPool) Put(parser *Parser) error {
	if parser == nil {
		return ErrInvalidHandle
	}

	parser.mu.Lock()
	closed := parser.closed
	liveDoc := parser.liveDoc
	parser.mu.Unlock()

	switch {
	case closed:
		return ErrClosed
	case liveDoc != 0:
		return ErrParserBusy
	default:
		p.pool.Put(parser)
		return nil
	}
}
