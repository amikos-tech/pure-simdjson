# Concurrency

## Invariant

The single-doc invariant is simple: a `Parser` may own only one live `Doc` at a
time. Parsing again before that document is closed returns `ErrParserBusy`.

## Why Parsers Are Not Shareable

The native shim preserves the Phase 2 lifecycle rule that one parser owns one
live document graph. Sharing a parser concurrently would turn that invariant
into a race between `Parse`, `Doc.Close`, and `Parser.Close`, so Phase 3 keeps
the contract explicit instead of trying to hide misuse.

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

`ParserPool.Put` rejects parsers that do not satisfy the Phase 3 lifecycle
contract:

- `nil` parsers return `ErrInvalidHandle`
- closed parsers return `ErrClosed`
- parsers that still own a live document return `ErrParserBusy`

Those failures are intentional. The pool does not auto-close documents, replace
parsers, or silently repair misuse.

## Leak Warnings

Explicit `Close` calls remain the primary cleanup path.

Production builds keep cleanup finalizers silent. They still release leaked
native resources, including parsers evicted from `sync.Pool`, but they do not
emit warning text.

Builds compiled with `-tags purejson_testbuild` attach the same cleanup
finalizers and add the warning prefix `purejson leak:` before cleanup so tests
can surface leaked parsers or docs.

The intended model remains goroutine-per-parser, with `ParserPool` providing the
handoff primitive when many goroutines need short-lived parser ownership.
