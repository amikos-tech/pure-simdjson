#include <stdint.h>

#define EXPORT __attribute__((visibility("default")))

EXPORT int32_t pure_simdjson_get_abi_version(uint32_t *out_version) {
  if (out_version == 0) {
    return 1;
  }
  *out_version = UINT32_C(0x00010001);
  return 0;
}
