package bootstrap

import "github.com/amikos-tech/pure-simdjson/internal/ffi"

// Version == "0.1.5" is expected to publish ABI 0x00010002.
// Future ABI bumps must update the bootstrap release pin and this canary together.
// Must stay in sync with scripts/release/check_bootstrap_abi_state.py:ABI_MINIMUM_VERSION.
const abiVersionForBootstrapVersion_0_1_5 uint32 = 0x00010002

var _ [int64(ffi.ABIVersion) - int64(abiVersionForBootstrapVersion_0_1_5)]struct{}
var _ [int64(abiVersionForBootstrapVersion_0_1_5) - int64(ffi.ABIVersion)]struct{}
