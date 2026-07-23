#include <stdint.h>

#define EXPORT __attribute__((visibility("default")))

EXPORT int32_t pure_simdjson_get_abi_version(uint32_t *out_version) {
  if (out_version == 0) {
    return 1;
  }
  *out_version = UINT32_C(0x00010002);
  return 0;
}

EXPORT int32_t pure_simdjson_parser_new_configured(void) { return 0; }
EXPORT int32_t pure_simdjson_parser_get_last_error_has_offset(void) { return 0; }
EXPORT int32_t pure_simdjson_element_get_bigint(void) { return 0; }
EXPORT int32_t pure_simdjson_set_implementation(void) { return 0; }
EXPORT int32_t pure_simdjson_lock_implementation_selection(void) { return 0; }
