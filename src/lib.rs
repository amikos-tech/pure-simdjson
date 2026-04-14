#![allow(non_camel_case_types)]

pub type pure_simdjson_handle_t = u64;
pub type pure_simdjson_err_t = i32;

pub const PURE_SIMDJSON_ABI_VERSION: u32 = 0x0001_0000;

#[no_mangle]
pub unsafe extern "C" fn pure_simdjson_get_abi_version(
    out_version: *mut u32,
) -> pure_simdjson_err_t {
    if out_version.is_null() {
        return -1;
    }

    unsafe {
        *out_version = PURE_SIMDJSON_ABI_VERSION;
    }

    0
}
