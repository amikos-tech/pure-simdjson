#include <stddef.h>

#include "pure_simdjson.h"

#ifndef static_assert
#define static_assert _Static_assert
#endif

static_assert(PURE_SIMDJSON_OK == 0, "PURE_SIMDJSON_OK must stay pinned");
static_assert(PURE_SIMDJSON_ERR_INVALID_ARGUMENT == 1,
              "PURE_SIMDJSON_ERR_INVALID_ARGUMENT must stay pinned");
static_assert(PURE_SIMDJSON_ERR_INVALID_HANDLE == 2,
              "PURE_SIMDJSON_ERR_INVALID_HANDLE must stay pinned");
static_assert(PURE_SIMDJSON_ERR_PARSER_BUSY == 3,
              "PURE_SIMDJSON_ERR_PARSER_BUSY must stay pinned");
static_assert(PURE_SIMDJSON_ERR_WRONG_TYPE == 4,
              "PURE_SIMDJSON_ERR_WRONG_TYPE must stay pinned");
static_assert(PURE_SIMDJSON_ERR_ELEMENT_NOT_FOUND == 5,
              "PURE_SIMDJSON_ERR_ELEMENT_NOT_FOUND must stay pinned");
static_assert(PURE_SIMDJSON_ERR_BUFFER_TOO_SMALL == 6,
              "PURE_SIMDJSON_ERR_BUFFER_TOO_SMALL must stay pinned");
static_assert(PURE_SIMDJSON_ERR_INVALID_JSON == 32,
              "PURE_SIMDJSON_ERR_INVALID_JSON must stay pinned");
static_assert(PURE_SIMDJSON_ERR_NUMBER_OUT_OF_RANGE == 33,
              "PURE_SIMDJSON_ERR_NUMBER_OUT_OF_RANGE must stay pinned");
static_assert(PURE_SIMDJSON_ERR_PRECISION_LOSS == 34,
              "PURE_SIMDJSON_ERR_PRECISION_LOSS must stay pinned");
static_assert(PURE_SIMDJSON_ERR_CPU_UNSUPPORTED == 64,
              "PURE_SIMDJSON_ERR_CPU_UNSUPPORTED must stay pinned");
static_assert(PURE_SIMDJSON_ERR_ABI_MISMATCH == 65,
              "PURE_SIMDJSON_ERR_ABI_MISMATCH must stay pinned");
static_assert(PURE_SIMDJSON_ERR_PANIC == 96,
              "PURE_SIMDJSON_ERR_PANIC must stay pinned");
static_assert(PURE_SIMDJSON_ERR_CPP_EXCEPTION == 97,
              "PURE_SIMDJSON_ERR_CPP_EXCEPTION must stay pinned");
static_assert(PURE_SIMDJSON_ERR_INTERNAL == 127,
              "PURE_SIMDJSON_ERR_INTERNAL must stay pinned");
static_assert(PURE_SIMDJSON_ABI_VERSION == 0x00010000u,
              "PURE_SIMDJSON_ABI_VERSION must stay pinned");

static_assert(PURE_SIMDJSON_VALUE_KIND_INVALID == 0,
              "PURE_SIMDJSON_VALUE_KIND_INVALID must stay pinned");
static_assert(PURE_SIMDJSON_VALUE_KIND_NULL == 1,
              "PURE_SIMDJSON_VALUE_KIND_NULL must stay pinned");
static_assert(PURE_SIMDJSON_VALUE_KIND_BOOL == 2,
              "PURE_SIMDJSON_VALUE_KIND_BOOL must stay pinned");
static_assert(PURE_SIMDJSON_VALUE_KIND_INT64 == 3,
              "PURE_SIMDJSON_VALUE_KIND_INT64 must stay pinned");
static_assert(PURE_SIMDJSON_VALUE_KIND_UINT64 == 4,
              "PURE_SIMDJSON_VALUE_KIND_UINT64 must stay pinned");
static_assert(PURE_SIMDJSON_VALUE_KIND_FLOAT64 == 5,
              "PURE_SIMDJSON_VALUE_KIND_FLOAT64 must stay pinned");
static_assert(PURE_SIMDJSON_VALUE_KIND_STRING == 6,
              "PURE_SIMDJSON_VALUE_KIND_STRING must stay pinned");
static_assert(PURE_SIMDJSON_VALUE_KIND_ARRAY == 7,
              "PURE_SIMDJSON_VALUE_KIND_ARRAY must stay pinned");
static_assert(PURE_SIMDJSON_VALUE_KIND_OBJECT == 8,
              "PURE_SIMDJSON_VALUE_KIND_OBJECT must stay pinned");

static_assert(sizeof(pure_simdjson_handle_t) == 8, "pure_simdjson_handle_t must stay 8 bytes");
static_assert(sizeof(pure_simdjson_parser_t) == 8, "pure_simdjson_parser_t must stay 8 bytes");
static_assert(sizeof(pure_simdjson_doc_t) == 8, "pure_simdjson_doc_t must stay 8 bytes");
static_assert(sizeof(pure_simdjson_handle_parts_t) == 8, "pure_simdjson_handle_parts_t must stay 8 bytes");
static_assert(offsetof(pure_simdjson_handle_parts_t, generation) == 4, "generation offset must stay stable");
static_assert(sizeof(pure_simdjson_value_view_t) == 32, "pure_simdjson_value_view_t must stay 32 bytes");
static_assert(sizeof(pure_simdjson_array_iter_t) == 32, "pure_simdjson_array_iter_t must stay 32 bytes");
static_assert(offsetof(pure_simdjson_array_iter_t, index) == 24,
              "pure_simdjson_array_iter_t.index offset must stay stable");
static_assert(offsetof(pure_simdjson_array_iter_t, tag) == 28,
              "pure_simdjson_array_iter_t.tag offset must stay stable");
static_assert(offsetof(pure_simdjson_array_iter_t, reserved) == 30,
              "pure_simdjson_array_iter_t.reserved offset must stay stable");
static_assert(sizeof(pure_simdjson_object_iter_t) == 32, "pure_simdjson_object_iter_t must stay 32 bytes");
static_assert(offsetof(pure_simdjson_object_iter_t, index) == 24,
              "pure_simdjson_object_iter_t.index offset must stay stable");
static_assert(offsetof(pure_simdjson_object_iter_t, tag) == 28,
              "pure_simdjson_object_iter_t.tag offset must stay stable");
static_assert(offsetof(pure_simdjson_object_iter_t, reserved) == 30,
              "pure_simdjson_object_iter_t.reserved offset must stay stable");
