// Package purejson exposes the Go wrapper for the pure-simdjson native library.
//
// NewParser creates one reusable native parser handle. Each Parser may own only
// one live Doc at a time, so callers must close the current document before
// parsing again or before closing or pooling the parser.
//
// Parsed documents expose typed Element accessors that preserve simdjson's
// int64/uint64/float64 split, copy strings into Go-owned memory, and surface
// arrays and objects through scanner-style iterators plus direct field lookup
// helpers.
//
// Parser limits are immutable after construction:
//
//	parser, err := NewParser(
//		WithMaxCapacity(8<<20),
//		WithMaxDepth(128),
//	)
//
// TypeBigInt values keep their exact decimal spelling. GetBigInt returns a
// copied Go string, so the text remains owned by Go after the document closes:
//
//	digits, err := element.GetBigInt()
//
// Parse locations are not guessed. Check HasOffset before using Offset,
// including when byte zero may be the known location:
//
//	if parseErr.HasOffset() {
//		log.Printf("invalid JSON at byte %d", parseErr.Offset())
//	}
//
// NewParserPool hands parsers across goroutines without weakening the
// lifecycle rule. Kernel selection is process-global and diagnostic-only:
// SetKernel must run before the first parser or parser pool is created, after
// which it returns ErrKernelLocked. See docs/concurrency.md in the repository
// for the concurrency and cleanup model.
package purejson
