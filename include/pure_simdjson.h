#ifndef PURE_SIMDJSON_H
#define PURE_SIMDJSON_H

#pragma once

#include <stdarg.h>
#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>
#include <stdlib.h>

/**
 * Public error codes for the stable Phase 1 ABI.
 */
enum pure_simdjson_error_code_t
#ifdef __cplusplus
  : int32_t
#endif // __cplusplus
 {
  PURE_SIMDJSON_OK = 0,
  PURE_SIMDJSON_ERR_INVALID_ARGUMENT = 1,
  PURE_SIMDJSON_ERR_INVALID_HANDLE = 2,
  PURE_SIMDJSON_ERR_PARSER_BUSY = 3,
  PURE_SIMDJSON_ERR_WRONG_TYPE = 4,
  PURE_SIMDJSON_ERR_ELEMENT_NOT_FOUND = 5,
  PURE_SIMDJSON_ERR_BUFFER_TOO_SMALL = 6,
  PURE_SIMDJSON_ERR_INVALID_JSON = 32,
  PURE_SIMDJSON_ERR_NUMBER_OUT_OF_RANGE = 33,
  PURE_SIMDJSON_ERR_PRECISION_LOSS = 34,
  PURE_SIMDJSON_ERR_CPU_UNSUPPORTED = 64,
  PURE_SIMDJSON_ERR_ABI_MISMATCH = 65,
  PURE_SIMDJSON_ERR_PANIC = 96,
  PURE_SIMDJSON_ERR_CPP_EXCEPTION = 97,
  PURE_SIMDJSON_ERR_INTERNAL = 127,
};
#ifndef __cplusplus
typedef int32_t pure_simdjson_error_code_t;
#endif // __cplusplus

/**
 * Coarse value kind tags used by `pure_simdjson_value_view_t.kind_hint`.
 */
enum pure_simdjson_value_kind_t
#ifdef __cplusplus
  : uint32_t
#endif // __cplusplus
 {
  PURE_SIMDJSON_VALUE_KIND_INVALID = 0,
  PURE_SIMDJSON_VALUE_KIND_NULL = 1,
  PURE_SIMDJSON_VALUE_KIND_BOOL = 2,
  PURE_SIMDJSON_VALUE_KIND_INT64 = 3,
  PURE_SIMDJSON_VALUE_KIND_UINT64 = 4,
  PURE_SIMDJSON_VALUE_KIND_FLOAT64 = 5,
  PURE_SIMDJSON_VALUE_KIND_STRING = 6,
  PURE_SIMDJSON_VALUE_KIND_ARRAY = 7,
  PURE_SIMDJSON_VALUE_KIND_OBJECT = 8,
};
#ifndef __cplusplus
typedef uint32_t pure_simdjson_value_kind_t;
#endif // __cplusplus

/**
 * Opaque parser and document handles are packed `slot:u32 | generation:u32`.
 */
typedef uint64_t pure_simdjson_handle_t;

/**
 * Lightweight document-tied node view used for roots, fields, and iterator results.
 */
typedef struct pure_simdjson_value_view_t {
  pure_simdjson_handle_t doc;
  uint64_t state0;
  uint64_t state1;
  uint32_t kind_hint;
  uint32_t reserved;
} pure_simdjson_value_view_t;

/**
 * Stateful array iterator tied to a live document handle.
 */
typedef struct pure_simdjson_array_iter_t {
  pure_simdjson_handle_t doc;
  uint64_t state0;
  uint64_t state1;
  uint64_t index;
} pure_simdjson_array_iter_t;

/**
 * Stateful object iterator tied to a live document handle.
 */
typedef struct pure_simdjson_object_iter_t {
  pure_simdjson_handle_t doc;
  uint64_t state0;
  uint64_t state1;
  uint64_t index;
} pure_simdjson_object_iter_t;

/**
 * Split view of a packed `pure_simdjson_handle_t`.
 */
typedef struct pure_simdjson_handle_parts_t {
  uint32_t slot;
  uint32_t generation;
} pure_simdjson_handle_parts_t;

#ifdef __cplusplus
extern "C" {
#endif // __cplusplus

/**
 * Write the packed ABI version expected by Go-side compatibility checks.
 */
int32_t pure_simdjson_get_abi_version(uint32_t *out_version);

/**
 * Report the byte length of the active implementation name.
 */
int32_t pure_simdjson_get_implementation_name_len(size_t *out_len);

/**
 * Copy the active implementation name into caller-owned storage.
 */
int32_t pure_simdjson_copy_implementation_name(uint8_t *dst, size_t dst_cap, size_t *out_written);

/**
 * Allocate a parser handle.
 */
int32_t pure_simdjson_parser_new(pure_simdjson_handle_t *out_parser);

/**
 * Release a parser handle after all associated documents have been freed.
 */
int32_t pure_simdjson_parser_free(pure_simdjson_handle_t parser);

/**
 * Parse one JSON buffer into a new document handle.
 *
 * Contract:
 * - Every call copies `input_ptr[..input_len]` into Rust-owned padded storage before simdjson
 *   sees it, with enough trailing capacity for `SIMDJSON_PADDING`.
 * - A parser owns at most one live document at a time. If `parser` already owns a live document,
 *   this function returns `PURE_SIMDJSON_ERR_PARSER_BUSY`.
 * - Re-parse never implicitly invalidates an existing document. The busy state remains until
 *   `pure_simdjson_doc_free` succeeds for that document.
 */
int32_t pure_simdjson_parser_parse(pure_simdjson_handle_t parser,
                                   const uint8_t *input_ptr,
                                   size_t input_len,
                                   pure_simdjson_handle_t *out_doc);

/**
 * Report the byte length of the parser's last diagnostic message.
 */
int32_t pure_simdjson_parser_get_last_error_len(pure_simdjson_handle_t parser, size_t *out_len);

/**
 * Copy the parser's last diagnostic message into caller-owned storage.
 */
int32_t pure_simdjson_parser_copy_last_error(pure_simdjson_handle_t parser,
                                             uint8_t *dst,
                                             size_t dst_cap,
                                             size_t *out_written);

/**
 * Report the byte offset associated with the parser's last failure.
 */
int32_t pure_simdjson_parser_get_last_error_offset(pure_simdjson_handle_t parser,
                                                   uint64_t *out_offset);

/**
 * Release a live document handle.
 *
 * Contract:
 * - `pure_simdjson_doc_free` is the only operation that clears a parser's busy state.
 * - Parser reuse never happens implicitly from `pure_simdjson_parser_parse`.
 * - Generation checks remain the mechanism that turns stale parser/doc/view use into
 *   deterministic `PURE_SIMDJSON_ERR_INVALID_HANDLE` failures instead of undefined behavior.
 */
int32_t pure_simdjson_doc_free(pure_simdjson_handle_t doc);

/**
 * Resolve the root value view for a live document handle.
 */
int32_t pure_simdjson_doc_root(pure_simdjson_handle_t doc,
                               struct pure_simdjson_value_view_t *out_root);

/**
 * Report the value kind for a document-tied view.
 */
int32_t pure_simdjson_element_type(const struct pure_simdjson_value_view_t *view,
                                   uint32_t *out_type);

/**
 * Decode the referenced value as `int64_t`.
 */
int32_t pure_simdjson_element_get_int64(const struct pure_simdjson_value_view_t *view,
                                        int64_t *out_value);

/**
 * Decode the referenced value as `uint64_t`.
 */
int32_t pure_simdjson_element_get_uint64(const struct pure_simdjson_value_view_t *view,
                                         uint64_t *out_value);

/**
 * Decode the referenced value as `double`.
 */
int32_t pure_simdjson_element_get_float64(const struct pure_simdjson_value_view_t *view,
                                          double *out_value);

/**
 * Copy the referenced string value into a newly allocated byte buffer.
 *
 * The caller receives `*out_ptr` plus `*out_len` and must release that allocation with
 * `pure_simdjson_bytes_free`. Borrowed string views are intentionally excluded from `v0.1`.
 */
int32_t pure_simdjson_element_get_string(const struct pure_simdjson_value_view_t *view,
                                         uint8_t **out_ptr,
                                         size_t *out_len);

/**
 * Release memory previously returned by `pure_simdjson_element_get_string`.
 */
int32_t pure_simdjson_bytes_free(uint8_t *ptr, size_t len);

/**
 * Decode the referenced value as a C `uint8_t` boolean.
 */
int32_t pure_simdjson_element_get_bool(const struct pure_simdjson_value_view_t *view,
                                       uint8_t *out_value);

/**
 * Report whether the referenced value is JSON `null`.
 */
int32_t pure_simdjson_element_is_null(const struct pure_simdjson_value_view_t *view,
                                      uint8_t *out_is_null);

/**
 * Initialize array iterator state from an array-valued view.
 */
int32_t pure_simdjson_array_iter_new(const struct pure_simdjson_value_view_t *array_view,
                                     struct pure_simdjson_array_iter_t *out_iter);

/**
 * Advance an array iterator and return the next value view plus a done flag.
 */
int32_t pure_simdjson_array_iter_next(struct pure_simdjson_array_iter_t *iter,
                                      struct pure_simdjson_value_view_t *out_value,
                                      uint8_t *out_done);

/**
 * Initialize object iterator state from an object-valued view.
 */
int32_t pure_simdjson_object_iter_new(const struct pure_simdjson_value_view_t *object_view,
                                      struct pure_simdjson_object_iter_t *out_iter);

/**
 * Advance an object iterator and return the next key/value pair plus a done flag.
 */
int32_t pure_simdjson_object_iter_next(struct pure_simdjson_object_iter_t *iter,
                                       struct pure_simdjson_value_view_t *out_key,
                                       struct pure_simdjson_value_view_t *out_value,
                                       uint8_t *out_done);

/**
 * Look up one object field by key and return its value view through `out_value`.
 */
int32_t pure_simdjson_object_get_field(const struct pure_simdjson_value_view_t *object_view,
                                       const uint8_t *key_ptr,
                                       size_t key_len,
                                       struct pure_simdjson_value_view_t *out_value);

#ifdef __cplusplus
}  // extern "C"
#endif  // __cplusplus

#endif  /* PURE_SIMDJSON_H */
