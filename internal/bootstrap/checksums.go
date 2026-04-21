package bootstrap

// Checksums maps path fragments "v<Version>/<os>-<arch>/<libname>" to their
// expected SHA-256 hex digests (D-08). CI-05 generates this map at release time.
// scripts/release/update_bootstrap_release_state.py performs that rewrite.
// During development the map is empty; BootstrapSync returns ErrNoChecksum when
// an entry is missing.
var Checksums = map[string]string{
	// "v0.1.0/linux-amd64/libpure_simdjson.so":      "<sha256>",
	// "v0.1.0/linux-arm64/libpure_simdjson.so":      "<sha256>",
	// "v0.1.0/darwin-amd64/libpure_simdjson.dylib":  "<sha256>",
	// "v0.1.0/darwin-arm64/libpure_simdjson.dylib":  "<sha256>",
	// "v0.1.0/windows-amd64/pure_simdjson-msvc.dll": "<sha256>",
}
