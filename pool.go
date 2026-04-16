package purejson

import "sync"

type ParserPool struct {
	pool sync.Pool
}

func NewParserPool() *ParserPool {
	return &ParserPool{}
}

func (p *ParserPool) Get() (*Parser, error) {
	if value := p.pool.Get(); value != nil {
		if parser, ok := value.(*Parser); ok && parser != nil {
			return parser, nil
		}
	}

	return NewParser()
}

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
