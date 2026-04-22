#include "simdjson_bridge.h"
#include "native_alloc_telemetry.h"

#include <cstring>
#include <memory>
#include <stdexcept>
#include <string>
#include <type_traits>

struct psimdjson_element {
  simdjson::dom::element value{};
};

struct psimdjson_doc {
  simdjson::dom::document document{};
  psimdjson_element root{};
};

struct psimdjson_parser {
  simdjson::dom::parser parser{};
  std::string last_error{};
  uint64_t last_error_offset{UINT64_MAX};
};

namespace {

pure_simdjson_error_code_t invalid_argument() noexcept {
  return PURE_SIMDJSON_ERR_INVALID_ARGUMENT;
}

pure_simdjson_error_code_t copy_bytes(
    const std::string &src,
    uint8_t *dst,
    size_t dst_cap,
    size_t *out_written
) noexcept {
  if (out_written == nullptr) {
    return invalid_argument();
  }

  *out_written = src.size();

  if (src.size() > dst_cap) {
    return PURE_SIMDJSON_ERR_BUFFER_TOO_SMALL;
  }

  if (!src.empty() && dst == nullptr) {
    return invalid_argument();
  }

  if (!src.empty()) {
    std::memcpy(dst, src.data(), src.size());
  }

  return PURE_SIMDJSON_OK;
}

pure_simdjson_error_code_t map_error(simdjson::error_code error) noexcept {
  switch (error) {
    case simdjson::SUCCESS:
      return PURE_SIMDJSON_OK;
    case simdjson::NO_SUCH_FIELD:
      return PURE_SIMDJSON_ERR_ELEMENT_NOT_FOUND;
    case simdjson::INCORRECT_TYPE:
      return PURE_SIMDJSON_ERR_WRONG_TYPE;
    case simdjson::NUMBER_OUT_OF_RANGE:
      return PURE_SIMDJSON_ERR_NUMBER_OUT_OF_RANGE;
    case simdjson::BIGINT_ERROR:
      return PURE_SIMDJSON_ERR_PRECISION_LOSS;
    case simdjson::TAPE_ERROR:
    case simdjson::DEPTH_ERROR:
    case simdjson::STRING_ERROR:
    case simdjson::T_ATOM_ERROR:
    case simdjson::F_ATOM_ERROR:
    case simdjson::N_ATOM_ERROR:
    case simdjson::NUMBER_ERROR:
    case simdjson::UTF8_ERROR:
    case simdjson::EMPTY:
    case simdjson::UNESCAPED_CHARS:
    case simdjson::UNCLOSED_STRING:
    case simdjson::INCOMPLETE_ARRAY_OR_OBJECT:
    case simdjson::TRAILING_CONTENT:
      return PURE_SIMDJSON_ERR_INVALID_JSON;
    case simdjson::CAPACITY:
    case simdjson::MEMALLOC:
    case simdjson::IO_ERROR:
    case simdjson::INVALID_JSON_POINTER:
    case simdjson::INVALID_URI_FRAGMENT:
    case simdjson::UNEXPECTED_ERROR:
    case simdjson::PARSER_IN_USE:
    case simdjson::UNINITIALIZED:
    case simdjson::INDEX_OUT_OF_BOUNDS:
    case simdjson::OUT_OF_ORDER_ITERATION:
    case simdjson::INSUFFICIENT_PADDING:
    case simdjson::SCALAR_DOCUMENT_AS_VALUE:
    case simdjson::OUT_OF_BOUNDS:
    case simdjson::OUT_OF_CAPACITY:
      return PURE_SIMDJSON_ERR_INTERNAL;
    case simdjson::UNSUPPORTED_ARCHITECTURE:
      return PURE_SIMDJSON_ERR_CPU_UNSUPPORTED;
    default:
      return PURE_SIMDJSON_ERR_INTERNAL;
  }
}

pure_simdjson_value_kind_t map_element_type(simdjson::dom::element_type type) noexcept {
  switch (type) {
    case simdjson::dom::element_type::ARRAY:
      return PURE_SIMDJSON_VALUE_KIND_ARRAY;
    case simdjson::dom::element_type::OBJECT:
      return PURE_SIMDJSON_VALUE_KIND_OBJECT;
    case simdjson::dom::element_type::INT64:
      return PURE_SIMDJSON_VALUE_KIND_INT64;
    case simdjson::dom::element_type::UINT64:
      return PURE_SIMDJSON_VALUE_KIND_UINT64;
    case simdjson::dom::element_type::DOUBLE:
      return PURE_SIMDJSON_VALUE_KIND_FLOAT64;
    case simdjson::dom::element_type::STRING:
      return PURE_SIMDJSON_VALUE_KIND_STRING;
    case simdjson::dom::element_type::BOOL:
      return PURE_SIMDJSON_VALUE_KIND_BOOL;
    case simdjson::dom::element_type::NULL_VALUE:
      return PURE_SIMDJSON_VALUE_KIND_NULL;
    case simdjson::dom::element_type::BIGINT:
      return PURE_SIMDJSON_VALUE_KIND_INVALID;
  }

  return PURE_SIMDJSON_VALUE_KIND_INVALID;
}

void clear_last_error(psimdjson_parser *parser) noexcept {
  parser->last_error.clear();
  parser->last_error_offset = UINT64_MAX;
}

void set_last_error_message(psimdjson_parser *parser, const std::string &message) noexcept {
  parser->last_error = message;
  parser->last_error_offset = UINT64_MAX;
}

void set_last_error(psimdjson_parser *parser, simdjson::error_code error) noexcept {
  set_last_error_message(parser, simdjson::error_message(error));
}

pure_simdjson_error_code_t map_cpp_exception(const std::bad_alloc &) noexcept {
  return PURE_SIMDJSON_ERR_INTERNAL;
}

pure_simdjson_error_code_t map_cpp_exception(const std::exception &) noexcept {
  return PURE_SIMDJSON_ERR_CPP_EXCEPTION;
}

pure_simdjson_error_code_t map_cpp_exception() noexcept {
  return PURE_SIMDJSON_ERR_CPP_EXCEPTION;
}

void capture_parser_exception(psimdjson_parser *parser, const std::bad_alloc &error) noexcept {
  set_last_error_message(parser, std::string("std::bad_alloc: ") + error.what());
}

void capture_parser_exception(psimdjson_parser *parser, const std::exception &error) noexcept {
  set_last_error_message(parser, error.what());
}

void capture_parser_exception(psimdjson_parser *parser) noexcept {
  set_last_error_message(parser, "unknown C++ exception");
}

std::string implementation_name() {
  return simdjson::get_active_implementation()->name();
}

simdjson::dom::element element_at(const psimdjson_doc *doc, uint64_t json_index) noexcept {
  static_assert(
      sizeof(simdjson::dom::element) == sizeof(simdjson::internal::tape_ref),
      "dom::element layout must stay tape_ref-sized for descendant reconstruction"
  );
  static_assert(
      std::is_trivially_copyable_v<simdjson::internal::tape_ref>,
      "tape_ref must remain trivially copyable for descendant reconstruction"
  );

  simdjson::dom::element element;
  auto *tape = reinterpret_cast<simdjson::internal::tape_ref *>(&element);
  *tape = simdjson::internal::tape_ref(&doc->document, size_t(json_index));
  return element;
}

simdjson::internal::tape_ref tape_ref_at(const psimdjson_doc *doc, uint64_t json_index) noexcept {
  return simdjson::internal::tape_ref(&doc->document, size_t(json_index));
}

simdjson::internal::tape_ref tape_ref_of(const simdjson::dom::element &element) noexcept {
  static_assert(
      sizeof(simdjson::dom::element) == sizeof(simdjson::internal::tape_ref),
      "dom::element layout must stay tape_ref-sized for descendant reconstruction"
  );
  static_assert(
      std::is_trivially_copyable_v<simdjson::internal::tape_ref>,
      "tape_ref must remain trivially copyable for descendant reconstruction"
  );

  simdjson::internal::tape_ref tape;
  std::memcpy(&tape, &element, sizeof(tape));
  return tape;
}

uint64_t element_json_index(const simdjson::dom::element &element) noexcept {
  return uint64_t(tape_ref_of(element).json_index);
}

}  // namespace

pure_simdjson_error_code_t psimdjson_get_implementation_name_len(size_t *out_len) noexcept {
  try {
    if (out_len == nullptr) {
      return invalid_argument();
    }

    *out_len = implementation_name().size();
    return PURE_SIMDJSON_OK;
  } catch (const std::bad_alloc &error) {
    return map_cpp_exception(error);
  } catch (const std::exception &error) {
    return map_cpp_exception(error);
  } catch (...) {
    return map_cpp_exception();
  }
}

pure_simdjson_error_code_t psimdjson_copy_implementation_name(
    uint8_t *dst,
    size_t dst_cap,
    size_t *out_written
) noexcept {
  try {
    return copy_bytes(implementation_name(), dst, dst_cap, out_written);
  } catch (const std::bad_alloc &error) {
    return map_cpp_exception(error);
  } catch (const std::exception &error) {
    return map_cpp_exception(error);
  } catch (...) {
    return map_cpp_exception();
  }
}

pure_simdjson_error_code_t psimdjson_native_alloc_stats_reset(void) noexcept {
  try {
    psimdjson::native_alloc_telemetry::reset();
    return PURE_SIMDJSON_OK;
  } catch (const std::bad_alloc &error) {
    return map_cpp_exception(error);
  } catch (const std::exception &error) {
    return map_cpp_exception(error);
  } catch (...) {
    return map_cpp_exception();
  }
}

pure_simdjson_error_code_t psimdjson_native_alloc_stats_snapshot(
    pure_simdjson_native_alloc_stats_t *out_stats
) noexcept {
  try {
    return psimdjson::native_alloc_telemetry::snapshot(out_stats);
  } catch (const std::bad_alloc &error) {
    return map_cpp_exception(error);
  } catch (const std::exception &error) {
    return map_cpp_exception(error);
  } catch (...) {
    return map_cpp_exception();
  }
}

size_t psimdjson_padding_bytes(void) noexcept {
  return simdjson::SIMDJSON_PADDING;
}

pure_simdjson_error_code_t psimdjson_parser_new(psimdjson_parser **out_parser) noexcept {
  try {
    if (out_parser == nullptr) {
      return invalid_argument();
    }

    auto parser = std::make_unique<psimdjson_parser>();
    *out_parser = parser.release();
    return PURE_SIMDJSON_OK;
  } catch (const std::bad_alloc &error) {
    return map_cpp_exception(error);
  } catch (const std::exception &error) {
    return map_cpp_exception(error);
  } catch (...) {
    return map_cpp_exception();
  }
}

pure_simdjson_error_code_t psimdjson_parser_free(psimdjson_parser *parser) noexcept {
  try {
    if (parser == nullptr) {
      return invalid_argument();
    }

    delete parser;
    return PURE_SIMDJSON_OK;
  } catch (const std::bad_alloc &error) {
    return map_cpp_exception(error);
  } catch (const std::exception &error) {
    return map_cpp_exception(error);
  } catch (...) {
    return map_cpp_exception();
  }
}

pure_simdjson_error_code_t psimdjson_parser_parse(
    psimdjson_parser *parser,
    const uint8_t *input_ptr,
    size_t input_len,
    psimdjson_doc **out_doc
) noexcept {
  try {
    if (parser == nullptr || out_doc == nullptr || (input_len != 0 && input_ptr == nullptr)) {
      return invalid_argument();
    }

    *out_doc = nullptr;
    auto doc = std::make_unique<psimdjson_doc>();
    simdjson::dom::element root;
    const auto error =
        parser->parser.parse_into_document(doc->document, input_ptr, input_len, false).get(root);
    if (error != simdjson::SUCCESS) {
      set_last_error(parser, error);
      return map_error(error);
    }

    clear_last_error(parser);
    doc->root.value = root;
    *out_doc = doc.release();
    return PURE_SIMDJSON_OK;
  } catch (const std::bad_alloc &error) {
    capture_parser_exception(parser, error);
    return map_cpp_exception(error);
  } catch (const std::exception &error) {
    capture_parser_exception(parser, error);
    return map_cpp_exception(error);
  } catch (...) {
    capture_parser_exception(parser);
    return map_cpp_exception();
  }
}

pure_simdjson_error_code_t psimdjson_parser_get_last_error_len(
    const psimdjson_parser *parser,
    size_t *out_len
) noexcept {
  try {
    if (parser == nullptr || out_len == nullptr) {
      return invalid_argument();
    }

    *out_len = parser->last_error.size();
    return PURE_SIMDJSON_OK;
  } catch (const std::bad_alloc &error) {
    return map_cpp_exception(error);
  } catch (const std::exception &error) {
    return map_cpp_exception(error);
  } catch (...) {
    return map_cpp_exception();
  }
}

pure_simdjson_error_code_t psimdjson_parser_copy_last_error(
    const psimdjson_parser *parser,
    uint8_t *dst,
    size_t dst_cap,
    size_t *out_written
) noexcept {
  try {
    if (parser == nullptr) {
      return invalid_argument();
    }

    return copy_bytes(parser->last_error, dst, dst_cap, out_written);
  } catch (const std::bad_alloc &error) {
    return map_cpp_exception(error);
  } catch (const std::exception &error) {
    return map_cpp_exception(error);
  } catch (...) {
    return map_cpp_exception();
  }
}

pure_simdjson_error_code_t psimdjson_parser_get_last_error_offset(
    const psimdjson_parser *parser,
    uint64_t *out_offset
) noexcept {
  try {
    if (parser == nullptr || out_offset == nullptr) {
      return invalid_argument();
    }

    *out_offset = parser->last_error_offset;
    return PURE_SIMDJSON_OK;
  } catch (const std::bad_alloc &error) {
    return map_cpp_exception(error);
  } catch (const std::exception &error) {
    return map_cpp_exception(error);
  } catch (...) {
    return map_cpp_exception();
  }
}

pure_simdjson_error_code_t psimdjson_doc_free(psimdjson_doc *doc) noexcept {
  try {
    if (doc == nullptr) {
      return invalid_argument();
    }

    delete doc;
    return PURE_SIMDJSON_OK;
  } catch (const std::bad_alloc &error) {
    return map_cpp_exception(error);
  } catch (const std::exception &error) {
    return map_cpp_exception(error);
  } catch (...) {
    return map_cpp_exception();
  }
}

pure_simdjson_error_code_t psimdjson_doc_root(
    psimdjson_doc *doc,
    const psimdjson_element **out_element
) noexcept {
  try {
    if (doc == nullptr || out_element == nullptr) {
      return invalid_argument();
    }

    *out_element = &doc->root;
    return PURE_SIMDJSON_OK;
  } catch (const std::bad_alloc &error) {
    return map_cpp_exception(error);
  } catch (const std::exception &error) {
    return map_cpp_exception(error);
  } catch (...) {
    return map_cpp_exception();
  }
}

pure_simdjson_error_code_t psimdjson_element_type(
    const psimdjson_element *element,
    pure_simdjson_value_kind_t *out_kind
) noexcept {
  try {
    if (element == nullptr || out_kind == nullptr) {
      return invalid_argument();
    }

    const auto type = element->value.type();
    if (type == simdjson::dom::element_type::BIGINT) {
      return PURE_SIMDJSON_ERR_PRECISION_LOSS;
    }

    *out_kind = map_element_type(type);
    return PURE_SIMDJSON_OK;
  } catch (const std::bad_alloc &error) {
    return map_cpp_exception(error);
  } catch (const std::exception &error) {
    return map_cpp_exception(error);
  } catch (...) {
    return map_cpp_exception();
  }
}

pure_simdjson_error_code_t psimdjson_element_get_int64(
    const psimdjson_element *element,
    int64_t *out_value
) noexcept {
  try {
    if (element == nullptr || out_value == nullptr) {
      return invalid_argument();
    }

    const auto error = element->value.get_int64().get(*out_value);
    return map_error(error);
  } catch (const std::bad_alloc &error) {
    return map_cpp_exception(error);
  } catch (const std::exception &error) {
    return map_cpp_exception(error);
  } catch (...) {
    return map_cpp_exception();
  }
}

pure_simdjson_error_code_t psimdjson_element_type_at(
    const psimdjson_doc *doc,
    uint64_t json_index,
    pure_simdjson_value_kind_t *out_kind
) noexcept {
  try {
    if (doc == nullptr || out_kind == nullptr) {
      return invalid_argument();
    }

    const auto type = element_at(doc, json_index).type();
    if (type == simdjson::dom::element_type::BIGINT) {
      return PURE_SIMDJSON_ERR_PRECISION_LOSS;
    }

    *out_kind = map_element_type(type);
    return PURE_SIMDJSON_OK;
  } catch (const std::bad_alloc &error) {
    return map_cpp_exception(error);
  } catch (const std::exception &error) {
    return map_cpp_exception(error);
  } catch (...) {
    return map_cpp_exception();
  }
}

pure_simdjson_error_code_t psimdjson_element_get_int64_at(
    const psimdjson_doc *doc,
    uint64_t json_index,
    int64_t *out_value
) noexcept {
  try {
    if (doc == nullptr || out_value == nullptr) {
      return invalid_argument();
    }

    const auto error = element_at(doc, json_index).get_int64().get(*out_value);
    return map_error(error);
  } catch (const std::bad_alloc &error) {
    return map_cpp_exception(error);
  } catch (const std::exception &error) {
    return map_cpp_exception(error);
  } catch (...) {
    return map_cpp_exception();
  }
}

pure_simdjson_error_code_t psimdjson_element_get_uint64_at(
    const psimdjson_doc *doc,
    uint64_t json_index,
    uint64_t *out_value
) noexcept {
  try {
    if (doc == nullptr || out_value == nullptr) {
      return invalid_argument();
    }

    const auto error = element_at(doc, json_index).get_uint64().get(*out_value);
    return map_error(error);
  } catch (const std::bad_alloc &error) {
    return map_cpp_exception(error);
  } catch (const std::exception &error) {
    return map_cpp_exception(error);
  } catch (...) {
    return map_cpp_exception();
  }
}

pure_simdjson_error_code_t psimdjson_element_get_float64_at(
    const psimdjson_doc *doc,
    uint64_t json_index,
    double *out_value
) noexcept {
  try {
    if (doc == nullptr || out_value == nullptr) {
      return invalid_argument();
    }

    const auto error = element_at(doc, json_index).get_double().get(*out_value);
    return map_error(error);
  } catch (const std::bad_alloc &error) {
    return map_cpp_exception(error);
  } catch (const std::exception &error) {
    return map_cpp_exception(error);
  } catch (...) {
    return map_cpp_exception();
  }
}

pure_simdjson_error_code_t psimdjson_element_get_string_view(
    const psimdjson_doc *doc,
    uint64_t json_index,
    const uint8_t **out_ptr,
    size_t *out_len
) noexcept {
  try {
    if (doc == nullptr || out_ptr == nullptr || out_len == nullptr) {
      return invalid_argument();
    }

    std::string_view value;
    const auto error = element_at(doc, json_index).get_string().get(value);
    if (error != simdjson::SUCCESS) {
      return map_error(error);
    }

    *out_len = value.size();
    *out_ptr = value.empty() ? nullptr : reinterpret_cast<const uint8_t *>(value.data());
    return PURE_SIMDJSON_OK;
  } catch (const std::bad_alloc &error) {
    return map_cpp_exception(error);
  } catch (const std::exception &error) {
    return map_cpp_exception(error);
  } catch (...) {
    return map_cpp_exception();
  }
}

pure_simdjson_error_code_t psimdjson_element_get_bool_at(
    const psimdjson_doc *doc,
    uint64_t json_index,
    uint8_t *out_value
) noexcept {
  try {
    if (doc == nullptr || out_value == nullptr) {
      return invalid_argument();
    }

    bool value = false;
    const auto error = element_at(doc, json_index).get_bool().get(value);
    if (error != simdjson::SUCCESS) {
      return map_error(error);
    }

    *out_value = value ? 1 : 0;
    return PURE_SIMDJSON_OK;
  } catch (const std::bad_alloc &error) {
    return map_cpp_exception(error);
  } catch (const std::exception &error) {
    return map_cpp_exception(error);
  } catch (...) {
    return map_cpp_exception();
  }
}

pure_simdjson_error_code_t psimdjson_element_is_null_at(
    const psimdjson_doc *doc,
    uint64_t json_index,
    uint8_t *out_is_null
) noexcept {
  try {
    if (doc == nullptr || out_is_null == nullptr) {
      return invalid_argument();
    }

    *out_is_null = element_at(doc, json_index).is_null() ? 1 : 0;
    return PURE_SIMDJSON_OK;
  } catch (const std::bad_alloc &error) {
    return map_cpp_exception(error);
  } catch (const std::exception &error) {
    return map_cpp_exception(error);
  } catch (...) {
    return map_cpp_exception();
  }
}

pure_simdjson_error_code_t psimdjson_element_after_index(
    const psimdjson_doc *doc,
    uint64_t json_index,
    uint64_t *out_after_json_index
) noexcept {
  try {
    if (doc == nullptr || out_after_json_index == nullptr) {
      return invalid_argument();
    }

    *out_after_json_index = uint64_t(tape_ref_at(doc, json_index).after_element());
    return PURE_SIMDJSON_OK;
  } catch (const std::bad_alloc &error) {
    return map_cpp_exception(error);
  } catch (const std::exception &error) {
    return map_cpp_exception(error);
  } catch (...) {
    return map_cpp_exception();
  }
}

pure_simdjson_error_code_t psimdjson_array_iter_bounds(
    const psimdjson_doc *doc,
    uint64_t json_index,
    uint64_t *out_state0,
    uint64_t *out_state1
) noexcept {
  try {
    if (doc == nullptr || out_state0 == nullptr || out_state1 == nullptr) {
      return invalid_argument();
    }

    const auto tape = tape_ref_at(doc, json_index);
    if (tape.tape_ref_type() != simdjson::internal::tape_type::START_ARRAY) {
      return PURE_SIMDJSON_ERR_WRONG_TYPE;
    }

    const auto after_json_index = uint64_t(tape.after_element());
    *out_state0 = json_index + 1;
    *out_state1 = after_json_index - 1;
    return PURE_SIMDJSON_OK;
  } catch (const std::bad_alloc &error) {
    return map_cpp_exception(error);
  } catch (const std::exception &error) {
    return map_cpp_exception(error);
  } catch (...) {
    return map_cpp_exception();
  }
}

pure_simdjson_error_code_t psimdjson_object_iter_bounds(
    const psimdjson_doc *doc,
    uint64_t json_index,
    uint64_t *out_state0,
    uint64_t *out_state1
) noexcept {
  try {
    if (doc == nullptr || out_state0 == nullptr || out_state1 == nullptr) {
      return invalid_argument();
    }

    const auto tape = tape_ref_at(doc, json_index);
    if (tape.tape_ref_type() != simdjson::internal::tape_type::START_OBJECT) {
      return PURE_SIMDJSON_ERR_WRONG_TYPE;
    }

    const auto after_json_index = uint64_t(tape.after_element());
    *out_state0 = json_index + 1;
    *out_state1 = after_json_index - 1;
    return PURE_SIMDJSON_OK;
  } catch (const std::bad_alloc &error) {
    return map_cpp_exception(error);
  } catch (const std::exception &error) {
    return map_cpp_exception(error);
  } catch (...) {
    return map_cpp_exception();
  }
}

pure_simdjson_error_code_t psimdjson_object_get_field_index(
    const psimdjson_doc *doc,
    uint64_t json_index,
    const uint8_t *key_ptr,
    size_t key_len,
    uint64_t *out_value_json_index
) noexcept {
  try {
    if (doc == nullptr || out_value_json_index == nullptr) {
      return invalid_argument();
    }
    if (key_len != 0 && key_ptr == nullptr) {
      return invalid_argument();
    }

    simdjson::dom::object object;
    const auto object_error = element_at(doc, json_index).get_object().get(object);
    if (object_error != simdjson::SUCCESS) {
      return map_error(object_error);
    }

    const auto key = key_len == 0
        ? std::string_view{}
        : std::string_view(reinterpret_cast<const char *>(key_ptr), key_len);
    simdjson::dom::element value;
    const auto field_error = object.at_key(key).get(value);
    if (field_error != simdjson::SUCCESS) {
      return map_error(field_error);
    }

    *out_value_json_index = element_json_index(value);
    return PURE_SIMDJSON_OK;
  } catch (const std::bad_alloc &error) {
    return map_cpp_exception(error);
  } catch (const std::exception &error) {
    return map_cpp_exception(error);
  } catch (...) {
    return map_cpp_exception();
  }
}

pure_simdjson_error_code_t psimdjson_test_force_cpp_exception(void) noexcept {
  try {
    throw std::runtime_error("forced cpp exception");
  } catch (const std::bad_alloc &error) {
    return map_cpp_exception(error);
  } catch (const std::exception &error) {
    return map_cpp_exception(error);
  } catch (...) {
    return map_cpp_exception();
  }
}
