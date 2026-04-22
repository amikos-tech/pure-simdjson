#ifndef PSIMDJSON_BRIDGE_H
#define PSIMDJSON_BRIDGE_H

#include "../../include/pure_simdjson.h"
#include "simdjson.h"

#ifdef __cplusplus
extern "C" {
#define PSIMDJSON_NOEXCEPT noexcept
#else
#define PSIMDJSON_NOEXCEPT
#endif

typedef struct psimdjson_parser psimdjson_parser;
typedef struct psimdjson_doc psimdjson_doc;
typedef struct psimdjson_element psimdjson_element;
struct pure_simdjson_native_alloc_stats_t;

pure_simdjson_error_code_t psimdjson_get_implementation_name_len(size_t *out_len) PSIMDJSON_NOEXCEPT;
pure_simdjson_error_code_t psimdjson_copy_implementation_name(
    uint8_t *dst,
    size_t dst_cap,
    size_t *out_written
) PSIMDJSON_NOEXCEPT;
pure_simdjson_error_code_t psimdjson_native_alloc_stats_reset(void) PSIMDJSON_NOEXCEPT;
pure_simdjson_error_code_t psimdjson_native_alloc_stats_snapshot(
    struct pure_simdjson_native_alloc_stats_t *out_stats
) PSIMDJSON_NOEXCEPT;
size_t psimdjson_padding_bytes(void) PSIMDJSON_NOEXCEPT;

pure_simdjson_error_code_t psimdjson_parser_new(psimdjson_parser **out_parser) PSIMDJSON_NOEXCEPT;
pure_simdjson_error_code_t psimdjson_parser_free(psimdjson_parser *parser) PSIMDJSON_NOEXCEPT;
pure_simdjson_error_code_t psimdjson_parser_parse(
    psimdjson_parser *parser,
    const uint8_t *input_ptr,
    size_t input_len,
    psimdjson_doc **out_doc
) PSIMDJSON_NOEXCEPT;
pure_simdjson_error_code_t psimdjson_parser_get_last_error_len(
    const psimdjson_parser *parser,
    size_t *out_len
) PSIMDJSON_NOEXCEPT;
pure_simdjson_error_code_t psimdjson_parser_copy_last_error(
    const psimdjson_parser *parser,
    uint8_t *dst,
    size_t dst_cap,
    size_t *out_written
) PSIMDJSON_NOEXCEPT;
pure_simdjson_error_code_t psimdjson_parser_get_last_error_offset(
    const psimdjson_parser *parser,
    uint64_t *out_offset
) PSIMDJSON_NOEXCEPT;

pure_simdjson_error_code_t psimdjson_doc_free(psimdjson_doc *doc) PSIMDJSON_NOEXCEPT;
pure_simdjson_error_code_t psimdjson_doc_root(
    psimdjson_doc *doc,
    const psimdjson_element **out_element
) PSIMDJSON_NOEXCEPT;
pure_simdjson_error_code_t psimdjson_element_type(
    const psimdjson_element *element,
    pure_simdjson_value_kind_t *out_kind
) PSIMDJSON_NOEXCEPT;
pure_simdjson_error_code_t psimdjson_element_get_int64(
    const psimdjson_element *element,
    int64_t *out_value
) PSIMDJSON_NOEXCEPT;
pure_simdjson_error_code_t psimdjson_element_type_at(
    const psimdjson_doc *doc,
    uint64_t json_index,
    pure_simdjson_value_kind_t *out_kind
) PSIMDJSON_NOEXCEPT;
pure_simdjson_error_code_t psimdjson_element_get_int64_at(
    const psimdjson_doc *doc,
    uint64_t json_index,
    int64_t *out_value
) PSIMDJSON_NOEXCEPT;
pure_simdjson_error_code_t psimdjson_element_get_uint64_at(
    const psimdjson_doc *doc,
    uint64_t json_index,
    uint64_t *out_value
) PSIMDJSON_NOEXCEPT;
pure_simdjson_error_code_t psimdjson_element_get_float64_at(
    const psimdjson_doc *doc,
    uint64_t json_index,
    double *out_value
) PSIMDJSON_NOEXCEPT;
pure_simdjson_error_code_t psimdjson_element_get_string_view(
    const psimdjson_doc *doc,
    uint64_t json_index,
    const uint8_t **out_ptr,
    size_t *out_len
) PSIMDJSON_NOEXCEPT;
pure_simdjson_error_code_t psimdjson_element_get_bool_at(
    const psimdjson_doc *doc,
    uint64_t json_index,
    uint8_t *out_value
) PSIMDJSON_NOEXCEPT;
pure_simdjson_error_code_t psimdjson_element_is_null_at(
    const psimdjson_doc *doc,
    uint64_t json_index,
    uint8_t *out_is_null
) PSIMDJSON_NOEXCEPT;
pure_simdjson_error_code_t psimdjson_element_after_index(
    const psimdjson_doc *doc,
    uint64_t json_index,
    uint64_t *out_after_json_index
) PSIMDJSON_NOEXCEPT;
pure_simdjson_error_code_t psimdjson_array_iter_bounds(
    const psimdjson_doc *doc,
    uint64_t json_index,
    uint64_t *out_state0,
    uint64_t *out_state1
) PSIMDJSON_NOEXCEPT;
pure_simdjson_error_code_t psimdjson_object_iter_bounds(
    const psimdjson_doc *doc,
    uint64_t json_index,
    uint64_t *out_state0,
    uint64_t *out_state1
) PSIMDJSON_NOEXCEPT;
pure_simdjson_error_code_t psimdjson_object_get_field_index(
    const psimdjson_doc *doc,
    uint64_t json_index,
    const uint8_t *key_ptr,
    size_t key_len,
    uint64_t *out_value_json_index
) PSIMDJSON_NOEXCEPT;

pure_simdjson_error_code_t psimdjson_test_force_cpp_exception(void) PSIMDJSON_NOEXCEPT;

#ifdef __cplusplus
}
#endif

#undef PSIMDJSON_NOEXCEPT

#endif
