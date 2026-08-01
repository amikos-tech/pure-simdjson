package bootstrap

import "github.com/amikos-tech/pure-simdjson/internal/ffi"

// Version == "0.2.0-dev" is the unreleased source-tree identity for ABI 0x00010003.
// Release-readiness policy must adopt this pair before publication; until then,
// this canary prevents the wrapper from claiming the historical ABI 1.2 artifact.
const abiVersionForBootstrapVersion_0_2_0_dev uint32 = 0x00010003

var _ [int64(ffi.ABIVersion) - int64(abiVersionForBootstrapVersion_0_2_0_dev)]struct{}
var _ [int64(abiVersionForBootstrapVersion_0_2_0_dev) - int64(ffi.ABIVersion)]struct{}
