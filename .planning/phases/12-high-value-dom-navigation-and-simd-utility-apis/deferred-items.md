# Deferred Items

- Repository-wide `go vet ./...` reports the pre-existing
  `materializer_fastpath.go:217:37: possible misuse of unsafe.Pointer` warning
  and exits 1 under the current Go toolchain. Plans 12-11 and 12-08 left the
  unrelated materializer code unchanged.
