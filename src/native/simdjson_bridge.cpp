#include "simdjson_bridge.h"

#include <cstring>
#include <memory>
#include <stdexcept>
#include <string>

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
    case simdjson::CAPACITY:
    case simdjson::MEMALLOC:
    case simdjson::UNEXPECTED_ERROR:
    case simdjson::PARSER_IN_USE:
      return PURE_SIMDJSON_ERR_INTERNAL;
    case simdjson::UNSUPPORTED_ARCHITECTURE:
      return PURE_SIMDJSON_ERR_CPU_UNSUPPORTED;
    default:
      return PURE_SIMDJSON_ERR_INVALID_JSON;
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

void set_last_error(psimdjson_parser *parser, simdjson::error_code error) noexcept {
  parser->last_error = simdjson::error_message(error);
  parser->last_error_offset = UINT64_MAX;
}

std::string implementation_name() {
  return simdjson::get_active_implementation()->name();
}

}  // namespace

pure_simdjson_error_code_t psimdjson_get_implementation_name_len(size_t *out_len) noexcept {
  try {
    if (out_len == nullptr) {
      return invalid_argument();
    }

    *out_len = implementation_name().size();
    return PURE_SIMDJSON_OK;
  } catch (...) {
    return PURE_SIMDJSON_ERR_CPP_EXCEPTION;
  }
}

pure_simdjson_error_code_t psimdjson_copy_implementation_name(
    uint8_t *dst,
    size_t dst_cap,
    size_t *out_written
) noexcept {
  try {
    return copy_bytes(implementation_name(), dst, dst_cap, out_written);
  } catch (...) {
    return PURE_SIMDJSON_ERR_CPP_EXCEPTION;
  }
}

size_t psimdjson_padding_bytes(void) noexcept {
  try {
    return simdjson::SIMDJSON_PADDING;
  } catch (...) {
    return 0;
  }
}

pure_simdjson_error_code_t psimdjson_parser_new(psimdjson_parser **out_parser) noexcept {
  try {
    if (out_parser == nullptr) {
      return invalid_argument();
    }

    *out_parser = new psimdjson_parser();
    return PURE_SIMDJSON_OK;
  } catch (...) {
    return PURE_SIMDJSON_ERR_CPP_EXCEPTION;
  }
}

pure_simdjson_error_code_t psimdjson_parser_free(psimdjson_parser *parser) noexcept {
  try {
    if (parser == nullptr) {
      return invalid_argument();
    }

    delete parser;
    return PURE_SIMDJSON_OK;
  } catch (...) {
    return PURE_SIMDJSON_ERR_CPP_EXCEPTION;
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
  } catch (...) {
    return PURE_SIMDJSON_ERR_CPP_EXCEPTION;
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
  } catch (...) {
    return PURE_SIMDJSON_ERR_CPP_EXCEPTION;
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
  } catch (...) {
    return PURE_SIMDJSON_ERR_CPP_EXCEPTION;
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
  } catch (...) {
    return PURE_SIMDJSON_ERR_CPP_EXCEPTION;
  }
}

pure_simdjson_error_code_t psimdjson_doc_free(psimdjson_doc *doc) noexcept {
  try {
    if (doc == nullptr) {
      return invalid_argument();
    }

    delete doc;
    return PURE_SIMDJSON_OK;
  } catch (...) {
    return PURE_SIMDJSON_ERR_CPP_EXCEPTION;
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
  } catch (...) {
    return PURE_SIMDJSON_ERR_CPP_EXCEPTION;
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

    *out_kind = map_element_type(element->value.type());
    return PURE_SIMDJSON_OK;
  } catch (...) {
    return PURE_SIMDJSON_ERR_CPP_EXCEPTION;
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
  } catch (...) {
    return PURE_SIMDJSON_ERR_CPP_EXCEPTION;
  }
}

pure_simdjson_error_code_t psimdjson_test_force_cpp_exception(void) noexcept {
  try {
    throw std::runtime_error("forced cpp exception");
  } catch (...) {
    return PURE_SIMDJSON_ERR_CPP_EXCEPTION;
  }
}
