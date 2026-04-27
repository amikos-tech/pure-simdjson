package bootstrap

// Checksums optionally overrides published SHA-256 digests for
// "v<Version>/<os>-<arch>/<libname>" path fragments.
//
// Production bootstrap resolves digests from published SHA256SUMS metadata under
// the release tag. Tests and controlled local flows may inject overrides here to
// avoid network metadata lookups.
var Checksums = map[string]string{
	// "v0.1.3/linux-amd64/libpure_simdjson.so":      "<sha256>",
	// "v0.1.3/linux-arm64/libpure_simdjson.so":      "<sha256>",
	// "v0.1.3/darwin-amd64/libpure_simdjson.dylib":  "<sha256>",
	// "v0.1.3/darwin-arm64/libpure_simdjson.dylib":  "<sha256>",
	// "v0.1.3/windows-amd64/pure_simdjson-msvc.dll": "<sha256>",
}
