#ifndef PURE_SIMDJSON_H
#define PURE_SIMDJSON_H

#pragma once

#include <stdarg.h>
#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>
#include <stdlib.h>
#include <stddef.h>
#include <stdint.h>
#include <stdbool.h>

#define PURE_SIMDJSON_ABI_VERSION 65536

typedef int32_t pure_simdjson_err_t;

#ifdef __cplusplus
extern "C" {
#endif // __cplusplus

pure_simdjson_err_t pure_simdjson_get_abi_version(uint32_t *out_version);

#ifdef __cplusplus
}  // extern "C"
#endif  // __cplusplus

#endif  /* PURE_SIMDJSON_H */
