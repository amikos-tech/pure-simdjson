#include <stddef.h>

#include "pure_simdjson.h"

#ifndef static_assert
#define static_assert _Static_assert
#endif

static_assert(sizeof(pure_simdjson_handle_t) == 8, "pure_simdjson_handle_t must stay 8 bytes");
static_assert(sizeof(pure_simdjson_handle_parts_t) == 8, "pure_simdjson_handle_parts_t must stay 8 bytes");
static_assert(offsetof(pure_simdjson_handle_parts_t, generation) == 4, "generation offset must stay stable");
static_assert(sizeof(pure_simdjson_value_view_t) == 32, "pure_simdjson_value_view_t must stay 32 bytes");
static_assert(sizeof(pure_simdjson_array_iter_t) == 32, "pure_simdjson_array_iter_t must stay 32 bytes");
static_assert(sizeof(pure_simdjson_object_iter_t) == 32, "pure_simdjson_object_iter_t must stay 32 bytes");
