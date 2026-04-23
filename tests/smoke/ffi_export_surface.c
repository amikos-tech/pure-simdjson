#include <inttypes.h>
#include <math.h>
#include <stdarg.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "pure_simdjson.h"

#ifdef _WIN32
#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#ifdef _MSC_VER
#pragma warning(disable : 4191)
#endif
#else
#include <dlfcn.h>
#endif

enum export_index {
  EXPORT_GET_ABI_VERSION = 0,
  EXPORT_GET_IMPLEMENTATION_NAME_LEN,
  EXPORT_COPY_IMPLEMENTATION_NAME,
  EXPORT_NATIVE_ALLOC_STATS_RESET,
  EXPORT_NATIVE_ALLOC_STATS_SNAPSHOT,
  EXPORT_PARSER_NEW,
  EXPORT_PARSER_FREE,
  EXPORT_PARSER_PARSE,
  EXPORT_PARSER_GET_LAST_ERROR_LEN,
  EXPORT_PARSER_COPY_LAST_ERROR,
  EXPORT_PARSER_GET_LAST_ERROR_OFFSET,
  EXPORT_DOC_FREE,
  EXPORT_DOC_ROOT,
  EXPORT_ELEMENT_TYPE,
  EXPORT_ELEMENT_GET_INT64,
  EXPORT_ELEMENT_GET_UINT64,
  EXPORT_ELEMENT_GET_FLOAT64,
  EXPORT_ELEMENT_GET_STRING,
  EXPORT_BYTES_FREE,
  EXPORT_ELEMENT_GET_BOOL,
  EXPORT_ELEMENT_IS_NULL,
  EXPORT_ARRAY_ITER_NEW,
  EXPORT_ARRAY_ITER_NEXT,
  EXPORT_OBJECT_ITER_NEW,
  EXPORT_OBJECT_ITER_NEXT,
  EXPORT_OBJECT_GET_FIELD,
  EXPORT_COUNT,
};

static const char *EXPORT_NAMES[EXPORT_COUNT] = {
    "pure_simdjson_get_abi_version",
    "pure_simdjson_get_implementation_name_len",
    "pure_simdjson_copy_implementation_name",
    "pure_simdjson_native_alloc_stats_reset",
    "pure_simdjson_native_alloc_stats_snapshot",
    "pure_simdjson_parser_new",
    "pure_simdjson_parser_free",
    "pure_simdjson_parser_parse",
    "pure_simdjson_parser_get_last_error_len",
    "pure_simdjson_parser_copy_last_error",
    "pure_simdjson_parser_get_last_error_offset",
    "pure_simdjson_doc_free",
    "pure_simdjson_doc_root",
    "pure_simdjson_element_type",
    "pure_simdjson_element_get_int64",
    "pure_simdjson_element_get_uint64",
    "pure_simdjson_element_get_float64",
    "pure_simdjson_element_get_string",
    "pure_simdjson_bytes_free",
    "pure_simdjson_element_get_bool",
    "pure_simdjson_element_is_null",
    "pure_simdjson_array_iter_new",
    "pure_simdjson_array_iter_next",
    "pure_simdjson_object_iter_new",
    "pure_simdjson_object_iter_next",
    "pure_simdjson_object_get_field",
};

typedef pure_simdjson_error_code_t (*fn_get_abi_version)(uint32_t *);
typedef pure_simdjson_error_code_t (*fn_get_implementation_name_len)(size_t *);
typedef pure_simdjson_error_code_t (*fn_copy_implementation_name)(uint8_t *, size_t, size_t *);
typedef pure_simdjson_error_code_t (*fn_native_alloc_stats_reset)(void);
typedef pure_simdjson_error_code_t (*fn_native_alloc_stats_snapshot)(pure_simdjson_native_alloc_stats_t *);
typedef pure_simdjson_error_code_t (*fn_parser_new)(pure_simdjson_parser_t *);
typedef pure_simdjson_error_code_t (*fn_parser_free)(pure_simdjson_parser_t);
typedef pure_simdjson_error_code_t (*fn_parser_parse)(pure_simdjson_parser_t,
                                                      const uint8_t *,
                                                      size_t,
                                                      pure_simdjson_doc_t *);
typedef pure_simdjson_error_code_t (*fn_parser_get_last_error_len)(pure_simdjson_parser_t, size_t *);
typedef pure_simdjson_error_code_t (*fn_parser_copy_last_error)(pure_simdjson_parser_t,
                                                                uint8_t *,
                                                                size_t,
                                                                size_t *);
typedef pure_simdjson_error_code_t (*fn_parser_get_last_error_offset)(pure_simdjson_parser_t,
                                                                      uint64_t *);
typedef pure_simdjson_error_code_t (*fn_doc_free)(pure_simdjson_doc_t);
typedef pure_simdjson_error_code_t (*fn_doc_root)(pure_simdjson_doc_t, pure_simdjson_value_view_t *);
typedef pure_simdjson_error_code_t (*fn_element_type)(const pure_simdjson_value_view_t *, uint32_t *);
typedef pure_simdjson_error_code_t (*fn_element_get_int64)(const pure_simdjson_value_view_t *, int64_t *);
typedef pure_simdjson_error_code_t (*fn_element_get_uint64)(const pure_simdjson_value_view_t *, uint64_t *);
typedef pure_simdjson_error_code_t (*fn_element_get_float64)(const pure_simdjson_value_view_t *,
                                                             double *);
typedef pure_simdjson_error_code_t (*fn_element_get_string)(const pure_simdjson_value_view_t *,
                                                            uint8_t **,
                                                            size_t *);
typedef pure_simdjson_error_code_t (*fn_bytes_free)(uint8_t *, size_t);
typedef pure_simdjson_error_code_t (*fn_element_get_bool)(const pure_simdjson_value_view_t *, uint8_t *);
typedef pure_simdjson_error_code_t (*fn_element_is_null)(const pure_simdjson_value_view_t *, uint8_t *);
typedef pure_simdjson_error_code_t (*fn_array_iter_new)(const pure_simdjson_value_view_t *,
                                                        pure_simdjson_array_iter_t *);
typedef pure_simdjson_error_code_t (*fn_array_iter_next)(pure_simdjson_array_iter_t *,
                                                         pure_simdjson_value_view_t *,
                                                         uint8_t *);
typedef pure_simdjson_error_code_t (*fn_object_iter_new)(const pure_simdjson_value_view_t *,
                                                         pure_simdjson_object_iter_t *);
typedef pure_simdjson_error_code_t (*fn_object_iter_next)(pure_simdjson_object_iter_t *,
                                                          pure_simdjson_value_view_t *,
                                                          pure_simdjson_value_view_t *,
                                                          uint8_t *);
typedef pure_simdjson_error_code_t (*fn_object_get_field)(const pure_simdjson_value_view_t *,
                                                          const uint8_t *,
                                                          size_t,
                                                          pure_simdjson_value_view_t *);

struct export_table {
#ifdef _WIN32
  HMODULE handle;
#else
  void *handle;
#endif
  unsigned char called[EXPORT_COUNT];

  fn_get_abi_version get_abi_version;
  fn_get_implementation_name_len get_implementation_name_len;
  fn_copy_implementation_name copy_implementation_name;
  fn_native_alloc_stats_reset native_alloc_stats_reset;
  fn_native_alloc_stats_snapshot native_alloc_stats_snapshot;
  fn_parser_new parser_new;
  fn_parser_free parser_free;
  fn_parser_parse parser_parse;
  fn_parser_get_last_error_len parser_get_last_error_len;
  fn_parser_copy_last_error parser_copy_last_error;
  fn_parser_get_last_error_offset parser_get_last_error_offset;
  fn_doc_free doc_free;
  fn_doc_root doc_root;
  fn_element_type element_type;
  fn_element_get_int64 element_get_int64;
  fn_element_get_uint64 element_get_uint64;
  fn_element_get_float64 element_get_float64;
  fn_element_get_string element_get_string;
  fn_bytes_free bytes_free;
  fn_element_get_bool element_get_bool;
  fn_element_is_null element_is_null;
  fn_array_iter_new array_iter_new;
  fn_array_iter_next array_iter_next;
  fn_object_iter_new object_iter_new;
  fn_object_iter_next object_iter_next;
  fn_object_get_field object_get_field;
};

static void *lookup_symbol(struct export_table *exports, const char *name)
{
#ifdef _WIN32
  return (void *)GetProcAddress(exports->handle, name);
#else
  return dlsym(exports->handle, name);
#endif
}

static int failf(const char *step, const char *fmt, ...)
{
  va_list args;
  fprintf(stderr, "ffi export smoke failed at %s: ", step);
  va_start(args, fmt);
  vfprintf(stderr, fmt, args);
  va_end(args);
  fputc('\n', stderr);
  return 1;
}

static int expect_status(const char *step,
                         pure_simdjson_error_code_t actual,
                         pure_simdjson_error_code_t expected)
{
  if (actual != expected) {
    return failf(step, "expected status %d, got %d", (int)expected, (int)actual);
  }
  return 0;
}

static int expect_string(const char *step, const uint8_t *ptr, size_t len, const char *expected)
{
  size_t expected_len = strlen(expected);
  if (len != expected_len || memcmp(ptr, expected, expected_len) != 0) {
    return failf(step, "expected %s, got len=%zu", expected, len);
  }
  return 0;
}

#ifdef _WIN32
static wchar_t *utf8_to_wide(const char *input)
{
  int wide_len = MultiByteToWideChar(CP_UTF8, 0, input, -1, NULL, 0);
  wchar_t *buffer;

  if (wide_len <= 0) {
    return NULL;
  }
  buffer = (wchar_t *)calloc((size_t)wide_len, sizeof(wchar_t));
  if (buffer == NULL) {
    return NULL;
  }
  if (MultiByteToWideChar(CP_UTF8, 0, input, -1, buffer, wide_len) <= 0) {
    free(buffer);
    return NULL;
  }
  return buffer;
}
#endif

static int resolve_exports(const char *library_path, struct export_table *exports)
{
  memset(exports, 0, sizeof(*exports));

#ifdef _WIN32
  wchar_t *wide_path = utf8_to_wide(library_path);
  if (wide_path == NULL) {
    return failf("utf8_to_wide", "failed to convert path %s", library_path);
  }
  exports->handle = LoadLibraryW(wide_path);
  free(wide_path);
  if (exports->handle == NULL) {
    return failf("LoadLibraryW", "failed to load %s (error=%lu)", library_path, GetLastError());
  }
#else
  exports->handle = dlopen(library_path, RTLD_NOW | RTLD_LOCAL);
  if (exports->handle == NULL) {
    return failf("dlopen", "failed to load %s: %s", library_path, dlerror());
  }
#endif

#define RESOLVE(field, index, type)                                                            \
  do {                                                                                         \
    void *symbol_ptr = lookup_symbol(exports, EXPORT_NAMES[index]);                            \
    if (symbol_ptr == NULL) {                                                                  \
      return failf("resolve", "failed to resolve %s", EXPORT_NAMES[index]);                    \
    }                                                                                          \
    exports->field = (type)symbol_ptr;                                                         \
  } while (0)

  RESOLVE(get_abi_version, EXPORT_GET_ABI_VERSION, fn_get_abi_version);
  RESOLVE(get_implementation_name_len,
          EXPORT_GET_IMPLEMENTATION_NAME_LEN,
          fn_get_implementation_name_len);
  RESOLVE(copy_implementation_name, EXPORT_COPY_IMPLEMENTATION_NAME, fn_copy_implementation_name);
  RESOLVE(native_alloc_stats_reset,
          EXPORT_NATIVE_ALLOC_STATS_RESET,
          fn_native_alloc_stats_reset);
  RESOLVE(native_alloc_stats_snapshot,
          EXPORT_NATIVE_ALLOC_STATS_SNAPSHOT,
          fn_native_alloc_stats_snapshot);
  RESOLVE(parser_new, EXPORT_PARSER_NEW, fn_parser_new);
  RESOLVE(parser_free, EXPORT_PARSER_FREE, fn_parser_free);
  RESOLVE(parser_parse, EXPORT_PARSER_PARSE, fn_parser_parse);
  RESOLVE(parser_get_last_error_len,
          EXPORT_PARSER_GET_LAST_ERROR_LEN,
          fn_parser_get_last_error_len);
  RESOLVE(parser_copy_last_error,
          EXPORT_PARSER_COPY_LAST_ERROR,
          fn_parser_copy_last_error);
  RESOLVE(parser_get_last_error_offset,
          EXPORT_PARSER_GET_LAST_ERROR_OFFSET,
          fn_parser_get_last_error_offset);
  RESOLVE(doc_free, EXPORT_DOC_FREE, fn_doc_free);
  RESOLVE(doc_root, EXPORT_DOC_ROOT, fn_doc_root);
  RESOLVE(element_type, EXPORT_ELEMENT_TYPE, fn_element_type);
  RESOLVE(element_get_int64, EXPORT_ELEMENT_GET_INT64, fn_element_get_int64);
  RESOLVE(element_get_uint64, EXPORT_ELEMENT_GET_UINT64, fn_element_get_uint64);
  RESOLVE(element_get_float64, EXPORT_ELEMENT_GET_FLOAT64, fn_element_get_float64);
  RESOLVE(element_get_string, EXPORT_ELEMENT_GET_STRING, fn_element_get_string);
  RESOLVE(bytes_free, EXPORT_BYTES_FREE, fn_bytes_free);
  RESOLVE(element_get_bool, EXPORT_ELEMENT_GET_BOOL, fn_element_get_bool);
  RESOLVE(element_is_null, EXPORT_ELEMENT_IS_NULL, fn_element_is_null);
  RESOLVE(array_iter_new, EXPORT_ARRAY_ITER_NEW, fn_array_iter_new);
  RESOLVE(array_iter_next, EXPORT_ARRAY_ITER_NEXT, fn_array_iter_next);
  RESOLVE(object_iter_new, EXPORT_OBJECT_ITER_NEW, fn_object_iter_new);
  RESOLVE(object_iter_next, EXPORT_OBJECT_ITER_NEXT, fn_object_iter_next);
  RESOLVE(object_get_field, EXPORT_OBJECT_GET_FIELD, fn_object_get_field);

#undef RESOLVE
  return 0;
}

static void close_exports(struct export_table *exports)
{
  if (exports->handle == NULL) {
    return;
  }
#ifdef _WIN32
  FreeLibrary(exports->handle);
#else
  dlclose(exports->handle);
#endif
  exports->handle = NULL;
}

static void mark_called(struct export_table *exports, enum export_index index)
{
  exports->called[index] = 1;
}

static int require_called(const struct export_table *exports)
{
  int index;
  for (index = 0; index < EXPORT_COUNT; ++index) {
    if (exports->called[index] == 0) {
      return failf("call coverage", "resolved export %s was never invoked", EXPORT_NAMES[index]);
    }
  }
  return 0;
}

int main(int argc, char **argv)
{
  static const uint8_t sample_json[] =
      "{\"int\":42,\"uint\":18446744073709551615,\"float\":3.5,\"str\":\"hello\","
      "\"bool\":true,\"null\":null,\"arr\":[1,2],\"obj\":{\"x\":7}}";
  static const uint8_t invalid_json[] = "{\"bad\":}";

  struct export_table exports;
  pure_simdjson_parser_t parser = 0;
  pure_simdjson_doc_t doc = 0;
  pure_simdjson_value_view_t root = {0};
  pure_simdjson_value_view_t view = {0};
  pure_simdjson_value_view_t iter_key = {0};
  pure_simdjson_value_view_t iter_value = {0};
  pure_simdjson_value_view_t array_value = {0};
  pure_simdjson_array_iter_t array_iter = {0};
  pure_simdjson_object_iter_t object_iter = {0};
  uint32_t abi_version = 0;
  size_t impl_len = 0;
  size_t impl_written = 0;
  uint8_t *impl_name = NULL;
  pure_simdjson_native_alloc_stats_t alloc_stats = {0};
  uint32_t root_kind = PURE_SIMDJSON_VALUE_KIND_INVALID;
  int64_t int_value = 0;
  uint64_t uint_value = 0;
  double float_value = 0.0;
  uint8_t *string_ptr = NULL;
  size_t string_len = 0;
  uint8_t bool_value = 0;
  uint8_t is_null = 0;
  uint8_t done = 0;
  size_t last_error_len = 0;
  size_t last_error_written = 0;
  uint8_t *last_error = NULL;
  uint64_t last_error_offset = 0;
  int rc = 1;

  if (argc != 2) {
    return failf("argv", "usage: ffi_export_surface <staged-library-path>");
  }
  if (resolve_exports(argv[1], &exports) != 0) {
    return 1;
  }

  mark_called(&exports, EXPORT_GET_ABI_VERSION);
  if (expect_status("pure_simdjson_get_abi_version",
                    exports.get_abi_version(&abi_version),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }
  if (abi_version != PURE_SIMDJSON_ABI_VERSION) {
    rc = failf("pure_simdjson_get_abi_version",
               "expected ABI 0x%08x, got 0x%08x",
               PURE_SIMDJSON_ABI_VERSION,
               abi_version);
    goto cleanup;
  }

  mark_called(&exports, EXPORT_GET_IMPLEMENTATION_NAME_LEN);
  if (expect_status("pure_simdjson_get_implementation_name_len",
                    exports.get_implementation_name_len(&impl_len),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }
  if (impl_len == 0) {
    rc = failf("pure_simdjson_get_implementation_name_len", "implementation name length was zero");
    goto cleanup;
  }

  impl_name = (uint8_t *)calloc(impl_len + 1, sizeof(uint8_t));
  if (impl_name == NULL) {
    rc = failf("calloc", "failed to allocate %zu bytes for implementation name", impl_len + 1);
    goto cleanup;
  }

  mark_called(&exports, EXPORT_COPY_IMPLEMENTATION_NAME);
  if (expect_status("pure_simdjson_copy_implementation_name",
                    exports.copy_implementation_name(impl_name, impl_len, &impl_written),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }
  if (impl_written == 0 || impl_written > impl_len) {
    rc = failf("pure_simdjson_copy_implementation_name",
               "unexpected written length %zu for implementation name len %zu",
               impl_written,
               impl_len);
    goto cleanup;
  }

  mark_called(&exports, EXPORT_NATIVE_ALLOC_STATS_RESET);
  if (expect_status("pure_simdjson_native_alloc_stats_reset(initial)",
                    exports.native_alloc_stats_reset(),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }

  mark_called(&exports, EXPORT_NATIVE_ALLOC_STATS_SNAPSHOT);
  if (expect_status("pure_simdjson_native_alloc_stats_snapshot(initial)",
                    exports.native_alloc_stats_snapshot(&alloc_stats),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }
  if (alloc_stats.epoch == 0 || alloc_stats.live_bytes != 0 ||
      alloc_stats.total_alloc_bytes != 0 || alloc_stats.alloc_count != 0 ||
      alloc_stats.free_count != 0 || alloc_stats.untracked_free_count != 0) {
    rc = failf("pure_simdjson_native_alloc_stats_snapshot(initial)",
               "expected nonzero epoch and zeroed counters, got epoch=%" PRIu64
               " live=%" PRIu64 " total=%" PRIu64 " allocs=%" PRIu64
               " frees=%" PRIu64 " untracked_frees=%" PRIu64,
               alloc_stats.epoch,
               alloc_stats.live_bytes,
               alloc_stats.total_alloc_bytes,
               alloc_stats.alloc_count,
               alloc_stats.free_count,
               alloc_stats.untracked_free_count);
    goto cleanup;
  }

  mark_called(&exports, EXPORT_PARSER_NEW);
  if (expect_status("pure_simdjson_parser_new",
                    exports.parser_new(&parser),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }
  if (parser == 0) {
    rc = failf("pure_simdjson_parser_new", "parser handle was zero");
    goto cleanup;
  }

  if (expect_status("pure_simdjson_native_alloc_stats_reset(parser-ready)",
                    exports.native_alloc_stats_reset(),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }
  if (expect_status("pure_simdjson_native_alloc_stats_snapshot(parser-ready)",
                    exports.native_alloc_stats_snapshot(&alloc_stats),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }
  if (alloc_stats.epoch == 0 || alloc_stats.live_bytes != 0 ||
      alloc_stats.total_alloc_bytes != 0 || alloc_stats.alloc_count != 0 ||
      alloc_stats.free_count != 0 || alloc_stats.untracked_free_count != 0) {
    rc = failf("pure_simdjson_native_alloc_stats_snapshot(parser-ready)",
               "expected nonzero epoch and zeroed parser-ready stats, got epoch=%" PRIu64
               " live=%" PRIu64 " total=%" PRIu64 " allocs=%" PRIu64
               " frees=%" PRIu64 " untracked_frees=%" PRIu64,
               alloc_stats.epoch,
               alloc_stats.live_bytes,
               alloc_stats.total_alloc_bytes,
               alloc_stats.alloc_count,
               alloc_stats.free_count,
               alloc_stats.untracked_free_count);
    goto cleanup;
  }

  mark_called(&exports, EXPORT_PARSER_PARSE);
  if (expect_status("pure_simdjson_parser_parse(valid)",
                    exports.parser_parse(parser, sample_json, sizeof(sample_json) - 1, &doc),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }
  if (doc == 0) {
    rc = failf("pure_simdjson_parser_parse(valid)", "document handle was zero");
    goto cleanup;
  }

  if (expect_status("pure_simdjson_native_alloc_stats_snapshot(after-parse)",
                    exports.native_alloc_stats_snapshot(&alloc_stats),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }
  if (alloc_stats.epoch == 0 || alloc_stats.live_bytes == 0 ||
      alloc_stats.total_alloc_bytes < alloc_stats.live_bytes ||
      alloc_stats.alloc_count == 0 || alloc_stats.untracked_free_count != 0) {
    rc = failf("pure_simdjson_native_alloc_stats_snapshot(after-parse)",
               "unexpected parse stats epoch=%" PRIu64 " live=%" PRIu64 " total=%" PRIu64
               " allocs=%" PRIu64 " untracked_frees=%" PRIu64,
               alloc_stats.epoch,
               alloc_stats.live_bytes,
               alloc_stats.total_alloc_bytes,
               alloc_stats.alloc_count,
               alloc_stats.untracked_free_count);
    goto cleanup;
  }

  mark_called(&exports, EXPORT_DOC_ROOT);
  if (expect_status("pure_simdjson_doc_root",
                    exports.doc_root(doc, &root),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }

  mark_called(&exports, EXPORT_ELEMENT_TYPE);
  if (expect_status("pure_simdjson_element_type",
                    exports.element_type(&root, &root_kind),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }
  if (root_kind != PURE_SIMDJSON_VALUE_KIND_OBJECT) {
    rc = failf("pure_simdjson_element_type",
               "expected root object kind %u, got %u",
               (unsigned)PURE_SIMDJSON_VALUE_KIND_OBJECT,
               (unsigned)root_kind);
    goto cleanup;
  }

  mark_called(&exports, EXPORT_OBJECT_GET_FIELD);
  if (expect_status("pure_simdjson_object_get_field(int)",
                    exports.object_get_field(&root, (const uint8_t *)"int", 3, &view),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }
  mark_called(&exports, EXPORT_ELEMENT_GET_INT64);
  if (expect_status("pure_simdjson_element_get_int64",
                    exports.element_get_int64(&view, &int_value),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }
  if (int_value != 42) {
    rc = failf("pure_simdjson_element_get_int64", "expected 42, got %lld", (long long)int_value);
    goto cleanup;
  }

  if (expect_status("pure_simdjson_object_get_field(uint)",
                    exports.object_get_field(&root, (const uint8_t *)"uint", 4, &view),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }
  mark_called(&exports, EXPORT_ELEMENT_GET_UINT64);
  if (expect_status("pure_simdjson_element_get_uint64",
                    exports.element_get_uint64(&view, &uint_value),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }
  if (uint_value != UINT64_MAX) {
    rc = failf("pure_simdjson_element_get_uint64",
               "expected %" PRIu64 ", got %" PRIu64,
               (uint64_t)UINT64_MAX,
               uint_value);
    goto cleanup;
  }

  if (expect_status("pure_simdjson_object_get_field(float)",
                    exports.object_get_field(&root, (const uint8_t *)"float", 5, &view),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }
  mark_called(&exports, EXPORT_ELEMENT_GET_FLOAT64);
  if (expect_status("pure_simdjson_element_get_float64",
                    exports.element_get_float64(&view, &float_value),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }
  if (fabs(float_value - 3.5) > 1e-12) {
    rc = failf("pure_simdjson_element_get_float64", "expected 3.5, got %.17g", float_value);
    goto cleanup;
  }

  if (expect_status("pure_simdjson_object_get_field(str)",
                    exports.object_get_field(&root, (const uint8_t *)"str", 3, &view),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }
  mark_called(&exports, EXPORT_ELEMENT_GET_STRING);
  if (expect_status("pure_simdjson_element_get_string(value)",
                    exports.element_get_string(&view, &string_ptr, &string_len),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }
  if (expect_string("pure_simdjson_element_get_string(value)", string_ptr, string_len, "hello") != 0) {
    goto cleanup;
  }
  mark_called(&exports, EXPORT_BYTES_FREE);
  if (expect_status("pure_simdjson_bytes_free(value)",
                    exports.bytes_free(string_ptr, string_len),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }
  string_ptr = NULL;
  string_len = 0;

  if (expect_status("pure_simdjson_object_get_field(bool)",
                    exports.object_get_field(&root, (const uint8_t *)"bool", 4, &view),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }
  mark_called(&exports, EXPORT_ELEMENT_GET_BOOL);
  if (expect_status("pure_simdjson_element_get_bool",
                    exports.element_get_bool(&view, &bool_value),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }
  if (bool_value != 1) {
    rc = failf("pure_simdjson_element_get_bool", "expected true, got %u", (unsigned)bool_value);
    goto cleanup;
  }

  if (expect_status("pure_simdjson_object_get_field(null)",
                    exports.object_get_field(&root, (const uint8_t *)"null", 4, &view),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }
  mark_called(&exports, EXPORT_ELEMENT_IS_NULL);
  if (expect_status("pure_simdjson_element_is_null",
                    exports.element_is_null(&view, &is_null),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }
  if (is_null != 1) {
    rc = failf("pure_simdjson_element_is_null", "expected null=1, got %u", (unsigned)is_null);
    goto cleanup;
  }

  if (expect_status("pure_simdjson_object_get_field(arr)",
                    exports.object_get_field(&root, (const uint8_t *)"arr", 3, &view),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }
  mark_called(&exports, EXPORT_ARRAY_ITER_NEW);
  if (expect_status("pure_simdjson_array_iter_new",
                    exports.array_iter_new(&view, &array_iter),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }
  mark_called(&exports, EXPORT_ARRAY_ITER_NEXT);
  if (expect_status("pure_simdjson_array_iter_next",
                    exports.array_iter_next(&array_iter, &array_value, &done),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }
  if (done != 0) {
    rc = failf("pure_simdjson_array_iter_next", "unexpected done flag %u", (unsigned)done);
    goto cleanup;
  }

  if (expect_status("pure_simdjson_object_get_field(obj)",
                    exports.object_get_field(&root, (const uint8_t *)"obj", 3, &view),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }
  mark_called(&exports, EXPORT_OBJECT_ITER_NEW);
  if (expect_status("pure_simdjson_object_iter_new",
                    exports.object_iter_new(&view, &object_iter),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }
  mark_called(&exports, EXPORT_OBJECT_ITER_NEXT);
  if (expect_status("pure_simdjson_object_iter_next",
                    exports.object_iter_next(&object_iter, &iter_key, &iter_value, &done),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }
  if (done != 0) {
    rc = failf("pure_simdjson_object_iter_next", "unexpected done flag %u", (unsigned)done);
    goto cleanup;
  }
  if (expect_status("pure_simdjson_element_get_string(key)",
                    exports.element_get_string(&iter_key, &string_ptr, &string_len),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }
  if (expect_string("pure_simdjson_element_get_string(key)", string_ptr, string_len, "x") != 0) {
    goto cleanup;
  }
  if (expect_status("pure_simdjson_bytes_free(key)",
                    exports.bytes_free(string_ptr, string_len),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }
  string_ptr = NULL;
  string_len = 0;
  if (expect_status("pure_simdjson_element_get_int64(iter-value)",
                    exports.element_get_int64(&iter_value, &int_value),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }
  if (int_value != 7) {
    rc = failf("pure_simdjson_element_get_int64(iter-value)",
               "expected 7, got %lld",
               (long long)int_value);
    goto cleanup;
  }

  mark_called(&exports, EXPORT_DOC_FREE);
  if (expect_status("pure_simdjson_doc_free(valid)",
                    exports.doc_free(doc),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }
  doc = 0;

  mark_called(&exports, EXPORT_PARSER_PARSE);
  if (expect_status("pure_simdjson_parser_parse(invalid)",
                    exports.parser_parse(parser, invalid_json, sizeof(invalid_json) - 1, &doc),
                    PURE_SIMDJSON_ERR_INVALID_JSON) != 0) {
    goto cleanup;
  }

  mark_called(&exports, EXPORT_PARSER_GET_LAST_ERROR_LEN);
  if (expect_status("pure_simdjson_parser_get_last_error_len",
                    exports.parser_get_last_error_len(parser, &last_error_len),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }
  if (last_error_len == 0) {
    rc = failf("pure_simdjson_parser_get_last_error_len", "expected non-empty diagnostic");
    goto cleanup;
  }

  last_error = (uint8_t *)calloc(last_error_len + 1, sizeof(uint8_t));
  if (last_error == NULL) {
    rc = failf("calloc", "failed to allocate %zu bytes for last error", last_error_len + 1);
    goto cleanup;
  }

  mark_called(&exports, EXPORT_PARSER_COPY_LAST_ERROR);
  if (expect_status("pure_simdjson_parser_copy_last_error",
                    exports.parser_copy_last_error(parser,
                                                  last_error,
                                                  last_error_len,
                                                  &last_error_written),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }
  if (last_error_written == 0 || last_error_written > last_error_len) {
    rc = failf("pure_simdjson_parser_copy_last_error",
               "unexpected copied last-error length %zu",
               last_error_written);
    goto cleanup;
  }

  mark_called(&exports, EXPORT_PARSER_GET_LAST_ERROR_OFFSET);
  if (expect_status("pure_simdjson_parser_get_last_error_offset",
                    exports.parser_get_last_error_offset(parser, &last_error_offset),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }
  if (last_error_offset != UINT64_MAX) {
    rc = failf("pure_simdjson_parser_get_last_error_offset",
               "expected unknown-offset sentinel, got %" PRIu64,
               last_error_offset);
    goto cleanup;
  }

  mark_called(&exports, EXPORT_PARSER_FREE);
  if (expect_status("pure_simdjson_parser_free",
                    exports.parser_free(parser),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }
  parser = 0;

  if (expect_status("pure_simdjson_native_alloc_stats_snapshot(after-free)",
                    exports.native_alloc_stats_snapshot(&alloc_stats),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }
  if (alloc_stats.live_bytes != 0 || alloc_stats.free_count == 0 ||
      alloc_stats.untracked_free_count != 0) {
    rc = failf("pure_simdjson_native_alloc_stats_snapshot(after-free)",
               "expected post-free live=0, frees>0, untracked_frees=0; got live=%" PRIu64
               " frees=%" PRIu64 " untracked_frees=%" PRIu64,
               alloc_stats.live_bytes,
               alloc_stats.free_count,
               alloc_stats.untracked_free_count);
    goto cleanup;
  }

  rc = require_called(&exports);
  if (rc != 0) {
    goto cleanup;
  }

  puts("ffi export surface smoke passed");
  rc = 0;

cleanup:
  free(impl_name);
  free(last_error);
  if (doc != 0) {
    exports.doc_free(doc);
  }
  if (parser != 0) {
    exports.parser_free(parser);
  }
  if (string_ptr != NULL) {
    exports.bytes_free(string_ptr, string_len);
  }
  close_exports(&exports);
  return rc;
}
