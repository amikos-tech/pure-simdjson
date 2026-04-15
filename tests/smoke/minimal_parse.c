#include <stdint.h>
#include <stdarg.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "pure_simdjson.h"

static int failf(const char *step, const char *fmt, ...)
{
  va_list args;
  fprintf(stderr, "phase2 smoke failed at %s: ", step);
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

int main(void)
{
  static const uint8_t literal_42[] = "42";
  static const uint8_t invalid_json[] = "{\"x\":}";

  pure_simdjson_parser_t parser = 0;
  pure_simdjson_doc_t doc = 0;
  pure_simdjson_value_view_t root = {0};
  uint32_t abi_version = 0;
  uint32_t value_kind = PURE_SIMDJSON_VALUE_KIND_INVALID;
  int64_t int_value = 0;
  size_t last_error_len = 0;
  size_t copied_len = 0;
  uint8_t *last_error = NULL;
  int rc = 1;

  if (expect_status("pure_simdjson_get_abi_version",
                    pure_simdjson_get_abi_version(&abi_version),
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

  if (expect_status("pure_simdjson_parser_new",
                    pure_simdjson_parser_new(&parser),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }

  if (expect_status("pure_simdjson_parser_parse(42)",
                    pure_simdjson_parser_parse(parser, literal_42, sizeof(literal_42) - 1, &doc),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }

  if (expect_status("pure_simdjson_doc_root",
                    pure_simdjson_doc_root(doc, &root),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }

  if (expect_status("pure_simdjson_element_type",
                    pure_simdjson_element_type(&root, &value_kind),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }
  if (value_kind != PURE_SIMDJSON_VALUE_KIND_INT64) {
    rc = failf("pure_simdjson_element_type",
               "expected value kind %u, got %u",
               (unsigned)PURE_SIMDJSON_VALUE_KIND_INT64,
               (unsigned)value_kind);
    goto cleanup;
  }

  if (expect_status("pure_simdjson_element_get_int64",
                    pure_simdjson_element_get_int64(&root, &int_value),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }
  if (int_value != 42) {
    rc = failf("pure_simdjson_element_get_int64", "expected 42, got %lld", (long long)int_value);
    goto cleanup;
  }

  if (expect_status("pure_simdjson_doc_free(first)",
                    pure_simdjson_doc_free(doc),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }
  doc = 0;
  memset(&root, 0, sizeof(root));

  if (expect_status("pure_simdjson_parser_parse(42 after doc_free)",
                    pure_simdjson_parser_parse(parser, literal_42, sizeof(literal_42) - 1, &doc),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }

  if (expect_status("pure_simdjson_doc_free(second)",
                    pure_simdjson_doc_free(doc),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }
  doc = 0;

  if (expect_status("pure_simdjson_parser_parse({\"x\":})",
                    pure_simdjson_parser_parse(parser, invalid_json, sizeof(invalid_json) - 1, &doc),
                    PURE_SIMDJSON_ERR_INVALID_JSON) != 0) {
    goto cleanup;
  }

  if (expect_status("pure_simdjson_parser_get_last_error_len",
                    pure_simdjson_parser_get_last_error_len(parser, &last_error_len),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }
  if (last_error_len == 0) {
    rc = failf("pure_simdjson_parser_get_last_error_len", "expected a non-empty diagnostic");
    goto cleanup;
  }

  last_error = (uint8_t *)calloc(last_error_len + 1, sizeof(uint8_t));
  if (last_error == NULL) {
    rc = failf("calloc", "unable to allocate %zu bytes for diagnostic buffer", last_error_len + 1);
    goto cleanup;
  }

  if (expect_status("pure_simdjson_parser_copy_last_error",
                    pure_simdjson_parser_copy_last_error(parser, last_error, last_error_len, &copied_len),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }
  if (copied_len == 0 || copied_len > last_error_len || last_error[0] == '\0') {
    rc = failf("pure_simdjson_parser_copy_last_error",
               "expected a non-empty diagnostic copy, got len=%zu first=%u",
               copied_len,
               (unsigned)(copied_len == 0 ? 0 : last_error[0]));
    goto cleanup;
  }

  if (expect_status("pure_simdjson_parser_free",
                    pure_simdjson_parser_free(parser),
                    PURE_SIMDJSON_OK) != 0) {
    goto cleanup;
  }
  parser = 0;

  puts("phase2 smoke passed");
  rc = 0;

cleanup:
  free(last_error);
  if (doc != 0) {
    pure_simdjson_doc_free(doc);
  }
  if (parser != 0) {
    pure_simdjson_parser_free(parser);
  }
  return rc;
}
