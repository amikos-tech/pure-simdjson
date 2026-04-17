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
pool := purejson.NewParserPool()

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

## Put Rejection Rules

`ParserPool.Put` rejects parsers that do not satisfy the parser-pool lifecycle
contract:

- `nil` parsers return `ErrInvalidHandle`
- closed parsers return `ErrClosed`
- parsers that still own a live document return `ErrParserBusy`

Those failures are intentional. The pool does not auto-close documents, replace
parsers, or silently repair misuse.

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
