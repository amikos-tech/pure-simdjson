# Deferred Items

- Repository-wide `go vet ./...` reports the pre-existing
  `materializer_fastpath.go:217:37: possible misuse of unsafe.Pointer` warning
  while still exiting successfully. Plan 12-11 left the unrelated materializer
  code unchanged.
