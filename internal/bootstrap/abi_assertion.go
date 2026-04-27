package bootstrap

import "github.com/amikos-tech/pure-simdjson/internal/ffi"

// Version == "0.1.2" is expected to publish ABI 0x00010001.
// Future ABI bumps must update the bootstrap release pin and this canary together.
// Must stay in sync with scripts/release/check_bootstrap_abi_state.py:ABI_MINIMUM_VERSION.
const abiVersionForBootstrapVersion_0_1_2 uint32 = 0x00010001

var _ [int64(ffi.ABIVersion) - int64(abiVersionForBootstrapVersion_0_1_2)]struct{}
var _ [int64(abiVersionForBootstrapVersion_0_1_2) - int64(ffi.ABIVersion)]struct{}
