#pragma once

#include <stdarg.h>
#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>
#include <stdlib.h>
#define PURE_SIMDJSON_ABI_VERSION 0x00010000

/**
 * Public error codes for the stable ABI v0.1 surface.
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
 * Diagnostic native allocator counters for the current telemetry epoch.
 *
 * This surface reports allocations routed through the native shim/simdjson cdylib path only.
 * It does not claim process-wide totals or Go heap activity.
 */
typedef struct pure_simdjson_native_alloc_stats_t {
  uint64_t live_bytes;
  uint64_t total_alloc_bytes;
  uint64_t alloc_count;
  uint64_t free_count;
} pure_simdjson_native_alloc_stats_t;

/**
 * Generic packed handle transport for the public ABI.
 *
 * The numeric value `0` is reserved as the invalid sentinel and is never produced by
 * successful constructors.
 */
typedef uint64_t pure_simdjson_handle_t;

/**
 * Opaque parser handle packed as `slot:u32 | generation:u32`.
 *
 * Parsers are thread-compatible, not thread-safe: one live parser/document graph must be
 * confined to one thread at a time.
 */
typedef pure_simdjson_handle_t pure_simdjson_parser_t;

/**
 * Opaque document handle packed as `slot:u32 | generation:u32`.
 *
 * The numeric value `0` is reserved as the invalid sentinel. Documents inherit the
 * thread-affinity of their owning parser.
 */
typedef pure_simdjson_handle_t pure_simdjson_doc_t;

/**
 * Lightweight document-tied node view used for roots, fields, and iterator results.
 */
typedef struct pure_simdjson_value_view_t {
  pure_simdjson_doc_t doc;
  uint64_t state0;
  uint64_t state1;
  uint32_t kind_hint;
  uint32_t reserved;
} pure_simdjson_value_view_t;

/**
 * Stateful array iterator tied to a live document handle.
 *
 * `state0`, `state1`, `index`, and `tag` are implementation-owned. `index` stays `u32`
 * because the ABI v0.1 layout only admits documents below the 4 GiB simdjson ceiling.
 * `reserved` stays pinned for future contract growth and callers must leave it untouched.
 */
typedef struct pure_simdjson_array_iter_t {
  pure_simdjson_doc_t doc;
  uint64_t state0;
  uint64_t state1;
  uint32_t index;
  uint16_t tag;
  uint16_t reserved;
} pure_simdjson_array_iter_t;

/**
 * Stateful object iterator tied to a live document handle.
 *
 * `state0`, `state1`, `index`, and `tag` are implementation-owned. `index` stays `u32`
 * because the ABI v0.1 layout only admits documents below the 4 GiB simdjson ceiling.
 * `reserved` stays pinned for future contract growth and callers must leave it untouched.
 */
typedef struct pure_simdjson_object_iter_t {
  pure_simdjson_doc_t doc;
  uint64_t state0;
  uint64_t state1;
  uint32_t index;
  uint16_t tag;
  uint16_t reserved;
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
 *
 * # Safety
 * `out_version` must be a valid writable pointer to a `u32`.
 */
pure_simdjson_error_code_t pure_simdjson_get_abi_version(uint32_t *out_version);

/**
 * Report the byte length of the active implementation name.
 *
 * # Safety
 * `out_len` must be a valid writable pointer to a `usize`.
 */
pure_simdjson_error_code_t pure_simdjson_get_implementation_name_len(size_t *out_len);

/**
 * Copy the active implementation name into caller-owned storage.
 *
 * `*out_written` is written with the required byte count whenever `out_written` itself is
 * non-null, regardless of the return code. Callers can read the size report on success, on
 * `PURE_SIMDJSON_ERR_BUFFER_TOO_SMALL`, and also on `PURE_SIMDJSON_ERR_INVALID_ARGUMENT`
 * caused by a null `dst` with sufficient `dst_cap`.
 *
 * # Safety
 * `out_written` must be a valid writable pointer to a `usize`. When `dst_cap` is large enough
 * to copy the implementation name, `dst` must point to writable storage for at least `dst_cap`
 * bytes.
 */
pure_simdjson_error_code_t pure_simdjson_copy_implementation_name(uint8_t *dst,
                                                                  size_t dst_cap,
                                                                  size_t *out_written);

/**
 * Reset the diagnostic native allocator telemetry epoch.
 *
 * Existing live native allocations remain valid, but future snapshots exclude them from the
 * reported counters until they are reallocated in the new epoch.
 */
pure_simdjson_error_code_t pure_simdjson_native_alloc_stats_reset(void);

/**
 * Snapshot the diagnostic native allocator counters for the current telemetry epoch.
 *
 * # Safety
 * `out_stats` must point to writable `pure_simdjson_native_alloc_stats_t` storage.
 */
pure_simdjson_error_code_t pure_simdjson_native_alloc_stats_snapshot(struct pure_simdjson_native_alloc_stats_t *out_stats);

/**
 * Allocate a parser handle.
 *
 * # Safety
 * `out_parser` must be a valid writable pointer to a `pure_simdjson_parser_t`.
 */
pure_simdjson_error_code_t pure_simdjson_parser_new(pure_simdjson_parser_t *out_parser);

/**
 * Release a parser handle after all associated documents have been freed.
 *
 * Returns `PURE_SIMDJSON_ERR_PARSER_BUSY` while a live document still belongs to `parser`.
 *
 * # Safety
 * `parser` must be a parser handle previously returned by this library. The sentinel `0` and
 * forged values are invalid.
 */
pure_simdjson_error_code_t pure_simdjson_parser_free(pure_simdjson_parser_t parser);

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
 *
 * # Safety
 * `parser` must be a live parser handle from this library. When `input_len` is non-zero,
 * `input_ptr` must be readable for `input_len` bytes. `out_doc` must be a valid writable pointer
 * to a `pure_simdjson_doc_t`.
 */
pure_simdjson_error_code_t pure_simdjson_parser_parse(pure_simdjson_parser_t parser,
                                                      const uint8_t *input_ptr,
                                                      size_t input_len,
                                                      pure_simdjson_doc_t *out_doc);

/**
 * Report the byte length of the parser's last diagnostic message.
 *
 * # Safety
 * `parser` must be a live parser handle from this library. `out_len` must be a valid writable
 * pointer to a `usize`.
 */
pure_simdjson_error_code_t pure_simdjson_parser_get_last_error_len(pure_simdjson_parser_t parser,
                                                                   size_t *out_len);

/**
 * Copy the parser's last diagnostic message into caller-owned storage.
 *
 * # Safety
 * `parser` must be a live parser handle from this library. `out_written` must be a valid
 * writable pointer to a `usize`. When `dst_cap` is large enough to copy the active diagnostic,
 * `dst` must point to writable storage for at least `dst_cap` bytes.
 */
pure_simdjson_error_code_t pure_simdjson_parser_copy_last_error(pure_simdjson_parser_t parser,
                                                                uint8_t *dst,
                                                                size_t dst_cap,
                                                                size_t *out_written);

/**
 * Report the byte offset associated with the parser's last failure.
 *
 * # Safety
 * `parser` must be a live parser handle from this library. `out_offset` must be a valid writable
 * pointer to a `u64`.
 */
pure_simdjson_error_code_t pure_simdjson_parser_get_last_error_offset(pure_simdjson_parser_t parser,
                                                                      uint64_t *out_offset);

/**
 * Release a live document handle.
 *
 * Contract:
 * - `pure_simdjson_doc_free` is the only operation that clears a parser's busy state.
 * - Parser reuse never happens implicitly from `pure_simdjson_parser_parse`.
 * - Generation checks remain the mechanism that turns stale parser/doc/view use into
 *   deterministic `PURE_SIMDJSON_ERR_INVALID_HANDLE` failures instead of undefined behavior.
 *
 * # Safety
 * `doc` must be a document handle previously returned by this library. The sentinel `0` and
 * forged values are invalid.
 */
pure_simdjson_error_code_t pure_simdjson_doc_free(pure_simdjson_doc_t doc);

/**
 * Resolve the root value view for a live document handle.
 *
 * The returned view's `kind_hint` is `PURE_SIMDJSON_VALUE_KIND_INVALID` for roots whose value
 * kind cannot be classified (for example, BIGINT). The canonical precision-loss error surfaces
 * at `pure_simdjson_element_type`, not here.
 *
 * # Safety
 * `doc` must be a live document handle from this library. `out_root` must be a valid writable
 * pointer to a `pure_simdjson_value_view_t`.
 */
pure_simdjson_error_code_t pure_simdjson_doc_root(pure_simdjson_doc_t doc,
                                                  struct pure_simdjson_value_view_t *out_root);

/**
 * Report the value kind for a document-tied view.
 *
 * Returns `PURE_SIMDJSON_ERR_PRECISION_LOSS` for BIGINT values and
 * `PURE_SIMDJSON_ERR_INVALID_HANDLE` when reserved bits are non-zero or the root tag is invalid.
 *
 * # Safety
 * `view` must point to a readable `pure_simdjson_value_view_t` derived from a live document and
 * `out_type` must point to writable `u32` storage.
 */
pure_simdjson_error_code_t pure_simdjson_element_type(const struct pure_simdjson_value_view_t *view,
                                                      uint32_t *out_type);

/**
 * Decode the referenced value as `int64_t`.
 *
 * # Safety
 * `view` must point to a readable `pure_simdjson_value_view_t` derived from a live document and
 * `out_value` must point to writable `i64` storage.
 */
pure_simdjson_error_code_t pure_simdjson_element_get_int64(const struct pure_simdjson_value_view_t *view,
                                                           int64_t *out_value);

/**
 * Decode the referenced value as `uint64_t`.
 *
 * Negative integers return `PURE_SIMDJSON_ERR_NUMBER_OUT_OF_RANGE`; non-uint64 kinds return
 * `PURE_SIMDJSON_ERR_WRONG_TYPE`.
 *
 * # Safety
 * `view` must point to a readable `pure_simdjson_value_view_t` derived from a live document and
 * `out_value` must point to writable `u64` storage.
 */
pure_simdjson_error_code_t pure_simdjson_element_get_uint64(const struct pure_simdjson_value_view_t *view,
                                                            uint64_t *out_value);

/**
 * Decode the referenced value as `double`.
 *
 * Integral values that cannot be represented exactly as `double` return
 * `PURE_SIMDJSON_ERR_PRECISION_LOSS`; non-numeric kinds return `PURE_SIMDJSON_ERR_WRONG_TYPE`.
 *
 * # Safety
 * `view` must point to a readable `pure_simdjson_value_view_t` derived from a live document and
 * `out_value` must point to writable `f64` storage.
 */
pure_simdjson_error_code_t pure_simdjson_element_get_float64(const struct pure_simdjson_value_view_t *view,
                                                             double *out_value);

/**
 * Copy the referenced string value into a newly allocated byte buffer.
 *
 * The caller receives `*out_ptr` plus `*out_len` and must release that allocation with
 * `pure_simdjson_bytes_free`. Borrowed string views are intentionally excluded from `v0.1`.
 *
 * # Safety
 * `view` must point to a readable `pure_simdjson_value_view_t` derived from a live document.
 * `out_ptr` and `out_len` must point to writable storage owned by the caller.
 */
pure_simdjson_error_code_t pure_simdjson_element_get_string(const struct pure_simdjson_value_view_t *view,
                                                            uint8_t **out_ptr,
                                                            size_t *out_len);

/**
 * Release memory previously returned by `pure_simdjson_element_get_string`.
 * The empty-string sentinel is `ptr == NULL && len == 0`.
 *
 * # Safety
 * `ptr` and `len` must describe an allocation previously returned by
 * `pure_simdjson_element_get_string`.
 */
pure_simdjson_error_code_t pure_simdjson_bytes_free(uint8_t *ptr, size_t len);

/**
 * Decode the referenced value as a C `uint8_t` boolean.
 *
 * # Safety
 * `view` must point to a readable `pure_simdjson_value_view_t` derived from a live document and
 * `out_value` must point to writable `u8` storage.
 */
pure_simdjson_error_code_t pure_simdjson_element_get_bool(const struct pure_simdjson_value_view_t *view,
                                                          uint8_t *out_value);

/**
 * Report whether the referenced value is JSON `null`.
 *
 * # Safety
 * `view` must point to a readable `pure_simdjson_value_view_t` derived from a live document and
 * `out_is_null` must point to writable `u8` storage.
 */
pure_simdjson_error_code_t pure_simdjson_element_is_null(const struct pure_simdjson_value_view_t *view,
                                                         uint8_t *out_is_null);

/**
 * Initialize array iterator state from an array-valued view.
 *
 * # Safety
 * `array_view` must point to a readable array-valued `pure_simdjson_value_view_t` derived from a
 * live document. `out_iter` must point to writable iterator storage.
 */
pure_simdjson_error_code_t pure_simdjson_array_iter_new(const struct pure_simdjson_value_view_t *array_view,
                                                        struct pure_simdjson_array_iter_t *out_iter);

/**
 * Advance an array iterator and return the next value view plus a done flag.
 *
 * # Safety
 * `iter` must point to readable and writable iterator state created by this library. `out_value`
 * and `out_done` must point to writable storage.
 */
pure_simdjson_error_code_t pure_simdjson_array_iter_next(struct pure_simdjson_array_iter_t *iter,
                                                         struct pure_simdjson_value_view_t *out_value,
                                                         uint8_t *out_done);

/**
 * Initialize object iterator state from an object-valued view.
 *
 * # Safety
 * `object_view` must point to a readable object-valued `pure_simdjson_value_view_t` derived from
 * a live document. `out_iter` must point to writable iterator storage.
 */
pure_simdjson_error_code_t pure_simdjson_object_iter_new(const struct pure_simdjson_value_view_t *object_view,
                                                         struct pure_simdjson_object_iter_t *out_iter);

/**
 * Advance an object iterator and return the next key/value pair plus a done flag.
 *
 * # Safety
 * `iter` must point to readable and writable iterator state created by this library. `out_key`,
 * `out_value`, and `out_done` must point to writable storage.
 */
pure_simdjson_error_code_t pure_simdjson_object_iter_next(struct pure_simdjson_object_iter_t *iter,
                                                          struct pure_simdjson_value_view_t *out_key,
                                                          struct pure_simdjson_value_view_t *out_value,
                                                          uint8_t *out_done);

/**
 * Look up one object field by key and return its value view through `out_value`.
 *
 * # Safety
 * `object_view` must point to a readable object-valued `pure_simdjson_value_view_t` derived from
 * a live document. When `key_len` is non-zero, `key_ptr` must be readable for `key_len` bytes.
 * `out_value` must point to writable storage.
 */
pure_simdjson_error_code_t pure_simdjson_object_get_field(const struct pure_simdjson_value_view_t *object_view,
                                                          const uint8_t *key_ptr,
                                                          size_t key_len,
                                                          struct pure_simdjson_value_view_t *out_value);

#ifdef __cplusplus
}  // extern "C"
#endif  // __cplusplus
