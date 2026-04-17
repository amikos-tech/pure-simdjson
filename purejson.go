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
// NewParserPool hands parsers across goroutines without weakening that
// lifecycle rule. See docs/concurrency.md in the repository for the
// concurrency and cleanup model.
package purejson
