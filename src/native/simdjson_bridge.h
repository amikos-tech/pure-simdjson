#ifndef PSIMDJSON_BRIDGE_H
#define PSIMDJSON_BRIDGE_H

#include "../../include/pure_simdjson.h"
#include "simdjson.h"

#ifdef __cplusplus
extern "C" {
#endif

typedef struct psimdjson_parser psimdjson_parser;
typedef struct psimdjson_doc psimdjson_doc;
typedef struct psimdjson_element psimdjson_element;

pure_simdjson_error_code_t psimdjson_get_implementation_name_len(size_t *out_len) noexcept;
pure_simdjson_error_code_t psimdjson_copy_implementation_name(
    uint8_t *dst,
    size_t dst_cap,
    size_t *out_written
) noexcept;
size_t psimdjson_padding_bytes(void) noexcept;

pure_simdjson_error_code_t psimdjson_parser_new(psimdjson_parser **out_parser) noexcept;
pure_simdjson_error_code_t psimdjson_parser_free(psimdjson_parser *parser) noexcept;
pure_simdjson_error_code_t psimdjson_parser_parse(
    psimdjson_parser *parser,
    const uint8_t *input_ptr,
    size_t input_len,
    psimdjson_doc **out_doc
) noexcept;
pure_simdjson_error_code_t psimdjson_parser_get_last_error_len(
    const psimdjson_parser *parser,
    size_t *out_len
) noexcept;
pure_simdjson_error_code_t psimdjson_parser_copy_last_error(
    const psimdjson_parser *parser,
    uint8_t *dst,
    size_t dst_cap,
    size_t *out_written
) noexcept;
pure_simdjson_error_code_t psimdjson_parser_get_last_error_offset(
    const psimdjson_parser *parser,
    uint64_t *out_offset
) noexcept;

pure_simdjson_error_code_t psimdjson_doc_free(psimdjson_doc *doc) noexcept;
pure_simdjson_error_code_t psimdjson_doc_root(
    psimdjson_doc *doc,
    const psimdjson_element **out_element
) noexcept;
pure_simdjson_error_code_t psimdjson_element_type(
    const psimdjson_element *element,
    pure_simdjson_value_kind_t *out_kind
) noexcept;
pure_simdjson_error_code_t psimdjson_element_get_int64(
    const psimdjson_element *element,
    int64_t *out_value
) noexcept;

pure_simdjson_error_code_t psimdjson_test_force_cpp_exception(void) noexcept;

#ifdef __cplusplus
}
#endif

#endif
