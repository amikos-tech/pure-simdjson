package bootstrap

// Version is the library version pinned at compile time.
// CI-05 updates this constant at release time alongside checksums.go.
// scripts/release/update_bootstrap_release_state.py performs that rewrite.
// ldflags -X is explicitly rejected (D-06): consumer go build does not
// run our build flags.
const Version = "0.1.0"
