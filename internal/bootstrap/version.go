package bootstrap

// Version is the library version pinned at compile time.
// The release tag and this constant must match.
// ldflags -X is explicitly rejected (D-06): consumer go build does not
// run our build flags.
const Version = "0.1.0"
