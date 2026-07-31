package bootstrap

// Version is the unreleased source-tree pin for the ABI 1.3 wrapper.
// No 0.2.0-dev artifact is published; repository runtime tests must use a
// freshly built library through PURE_SIMDJSON_LIB_PATH.
// ldflags -X remains explicitly rejected (D-06): consumer go build does not
// run our build flags.
const Version = "0.2.0-dev"
