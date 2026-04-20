package bootstrap

// Version is the library version pinned at compile time.
// CI-05 updates this constant at release time alongside checksums.go.
// ldflags -X is explicitly rejected (D-06): consumer go build does not
// run our build flags.
const Version = "0.1.0"
