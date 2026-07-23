# Concurrency

## Invariant

The single-doc invariant is simple: a `Parser` may own only one live `Doc` at a
time. Parsing again before that document is closed returns `ErrParserBusy`.

## Sharing Parsers

Parser methods serialize `Parse`, `Doc.Close`, `Parser.Close`, and `ParserPool.Put`
with a mutex, so concurrent calls do not race at the memory level. The real
constraint is logical ownership: one parser still owns at most one live
document graph at a time, and a concurrent caller can only observe that parser
as busy until the current document is closed.

## ParserPool Pattern

Use one parser per goroutine, and hand parsers across goroutines through
`ParserPool` rather than sharing one live parser concurrently.

```go
pool, err := purejson.NewParserPool(
	purejson.WithMaxCapacity(8 << 20),
	purejson.WithMaxDepth(256),
)
if err != nil {
	return err
}

parser, err := pool.Get()
if err != nil {
	return err
}

doc, err := parser.Parse(data)
if err != nil {
	return err
}

value, err := doc.Root().GetInt64()
if err != nil {
	return err
}

_ = value

if err := doc.Close(); err != nil {
	return err
}

if err := pool.Put(parser); err != nil {
	return err
}
```

Parser options are immutable. Omitted or zero capacity/depth options select the
defaults (`0xFFFFFFFF` bytes and depth `1024`), and repeated options use the
last value. The pool stores the normalized values and applies them to every
cache miss.

Constructing a pool only validates and stores those Go values; it does not
resolve, download, or load a native library. The first `Get` cache miss creates
a parser and may run the normal bootstrap path.

## Put Rejection Rules

`ParserPool.Put` rejects parsers that do not satisfy the parser-pool lifecycle
contract:

- `nil` parsers return `ErrInvalidHandle`
- closed parsers return `ErrClosed`
- parsers that still own a live document return `ErrParserBusy`
- parsers created with different capacity or depth options return
  `ErrInvalidOption`

Those failures are intentional. The pool does not auto-close documents, replace
parsers, mix parser configurations, or silently repair misuse.

## Pool Shutdown

`sync.Pool` cannot be drained, so there is no `ParserPool.Close`. When a
`ParserPool` goes out of scope, any parsers still held in it are released by
the GC finalizer. The same leak-warning rules apply: production builds are
quiet by default; set `PURE_SIMDJSON_WARN_LEAKS=1` or build with
`-tags purejson_testbuild` to surface warnings before cleanup runs.

## Leak Warnings

Explicit `Close` calls remain the primary cleanup path.

Production builds keep cleanup finalizers quiet by default. Setting
`PURE_SIMDJSON_WARN_LEAKS=1` emits the same `purejson leak:` warning prefix used
by test builds before leaked native resources are released.

Builds compiled with `-tags purejson_testbuild` attach the same cleanup
finalizers and add the warning prefix `purejson leak:` before cleanup so tests
can surface leaked parsers or docs.

The intended model remains goroutine-per-parser, with `ParserPool` providing the
handoff primitive when many goroutines need short-lived parser ownership.
