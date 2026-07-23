#include "simdjson_bridge.h"
#include "native_alloc_telemetry.h"

#include <cstddef>
#include <cstdint>
#include <cstdio>
#include <cstdlib>
#include <cstring>
#include <limits>
#include <memory>
#include <stdexcept>
#include <string>
#include <string_view>
#include <type_traits>
#include <vector>

namespace {

struct LastErrorBuffer {
  char *ptr{nullptr};
  size_t len{0};

  LastErrorBuffer() noexcept = default;
  LastErrorBuffer(const LastErrorBuffer &) = delete;
  LastErrorBuffer &operator=(const LastErrorBuffer &) = delete;

  ~LastErrorBuffer() {
    std::free(ptr);
  }

  void clear() noexcept {
    std::free(ptr);
    ptr = nullptr;
    len = 0;
  }

  void assign(std::string_view message) {
    char *next = nullptr;
    if (!message.empty()) {
      next = static_cast<char *>(std::malloc(message.size()));
      if (next == nullptr) {
        throw std::bad_alloc();
      }
      std::memcpy(next, message.data(), message.size());
    }

    std::free(ptr);
    ptr = next;
    len = message.size();
  }

  [[nodiscard]] const char *data() const noexcept {
    return ptr == nullptr ? "" : ptr;
  }

  [[nodiscard]] size_t size() const noexcept {
    return len;
  }
};

}  // namespace

struct psimdjson_element {
  simdjson::dom::element value{};
};

struct psimdjson_doc {
  simdjson::dom::document document{};
  psimdjson_element root{};
  // Borrowed spans returned by psimdjson_materialize_build point into this
  // scratch vector. The next build on the same doc clears/reuses it and
  // invalidates any prior span; callers must consume or copy before re-entry.
  std::vector<psdj_internal_frame_t> materialize_frames{};
  // Guard is per-doc because Go serializes all live document access with
  // Doc.mu; if that invariant changes, this scratch vector needs a wider guard.
  bool materialize_in_progress{false};
};

struct psimdjson_parser {
  simdjson::dom::parser parser{};
  LastErrorBuffer last_error{};
  uint64_t last_error_offset{UINT64_MAX};
  bool last_error_has_offset{false};
  psimdjson_test_replay_observation replay_observation{};
  // Plan 11-05 diagnostic replay consumes these exact normalized values.
  uint64_t max_capacity{0};
  uint32_t max_depth{0};
};

namespace {

constexpr uint64_t DEFAULT_MAX_CAPACITY = UINT64_C(0xFFFFFFFF);
constexpr uint32_t DEFAULT_MAX_DEPTH = UINT32_C(1024);
constexpr uint64_t MIN_MAX_CAPACITY = UINT64_C(32);
constexpr uint32_t REPLAY_PASS_RAW_JSON = 1;
constexpr uint32_t REPLAY_PASS_RECURSIVE = 2;
constexpr uint32_t POINTER_NOT_QUERIED = 0;
constexpr uint32_t POINTER_IN_BOUNDS = 1;
constexpr uint32_t POINTER_AT_END = 2;
constexpr uint32_t POINTER_OUT_OF_RANGE = 3;
constexpr uint32_t POINTER_ADDRESS_OVERFLOW = 4;

constexpr auto PURE_SIMDJSON_VALUE_KIND_BIGINT =
    static_cast<pure_simdjson_value_kind_t>(9);

pure_simdjson_error_code_t invalid_argument() noexcept {
  return PURE_SIMDJSON_ERR_INVALID_ARGUMENT;
}

pure_simdjson_error_code_t copy_bytes(
    std::string_view src,
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
    case simdjson::DEPTH_ERROR:
      return PURE_SIMDJSON_ERR_DEPTH_LIMIT;
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
      return PURE_SIMDJSON_VALUE_KIND_BIGINT;
  }

  return PURE_SIMDJSON_VALUE_KIND_INVALID;
}

void clear_last_error(psimdjson_parser *parser) noexcept {
  parser->last_error.clear();
  parser->last_error_offset = UINT64_MAX;
  parser->last_error_has_offset = false;
}

void set_last_error_message(psimdjson_parser *parser, std::string_view message) {
  parser->last_error.assign(message);
  parser->last_error_offset = UINT64_MAX;
  parser->last_error_has_offset = false;
}

void set_last_error(psimdjson_parser *parser, simdjson::error_code error) {
  set_last_error_message(parser, simdjson::error_message(error));
}

void try_set_last_error_message(psimdjson_parser *parser, std::string_view message) noexcept {
  try {
    set_last_error_message(parser, message);
  } catch (...) {
  }
}

void log_cpp_exception(const char *function_name, const char *message) noexcept {
  std::fprintf(stderr, "pure_simdjson C++ exception in %s: %s\n", function_name, message);
}

pure_simdjson_error_code_t map_cpp_exception(
    const char *function_name,
    const std::bad_alloc &error
) noexcept {
  log_cpp_exception(function_name, error.what());
  return PURE_SIMDJSON_ERR_INTERNAL;
}

pure_simdjson_error_code_t map_cpp_exception(
    const char *function_name,
    const std::exception &error
) noexcept {
  log_cpp_exception(function_name, error.what());
  return PURE_SIMDJSON_ERR_CPP_EXCEPTION;
}

pure_simdjson_error_code_t map_cpp_exception(const char *function_name) noexcept {
  log_cpp_exception(function_name, "unknown C++ exception");
  return PURE_SIMDJSON_ERR_CPP_EXCEPTION;
}

void capture_parser_exception(psimdjson_parser *parser, const std::bad_alloc &error) noexcept {
  try_set_last_error_message(parser, std::string("std::bad_alloc: ") + error.what());
}

void capture_parser_exception(psimdjson_parser *parser, const std::exception &error) noexcept {
  try_set_last_error_message(parser, error.what());
}

void capture_parser_exception(psimdjson_parser *parser) noexcept {
  try_set_last_error_message(parser, "unknown C++ exception");
}

#define PSIMDJSON_CATCH_CPP_EXCEPTIONS(function_name)                    \
  catch (const std::bad_alloc &error) {                                  \
    return map_cpp_exception(function_name, error);                       \
  }                                                                      \
  catch (const std::exception &error) {                                   \
    return map_cpp_exception(function_name, error);                       \
  }                                                                      \
  catch (...) {                                                          \
    return map_cpp_exception(function_name);                              \
  }

#define PSIMDJSON_CATCH_PARSER_CPP_EXCEPTIONS(function_name, parser_ptr) \
  catch (const std::bad_alloc &error) {                                  \
    capture_parser_exception(parser_ptr, error);                         \
    return map_cpp_exception(function_name, error);                       \
  }                                                                      \
  catch (const std::exception &error) {                                   \
    capture_parser_exception(parser_ptr, error);                         \
    return map_cpp_exception(function_name, error);                       \
  }                                                                      \
  catch (...) {                                                          \
    capture_parser_exception(parser_ptr);                                \
    return map_cpp_exception(function_name);                              \
  }

struct DiagnosticReplayLimits {
  uint64_t max_capacity;
  uint32_t max_depth;
};

enum class ReplayDisposition {
  valid,
  stopped,
};

struct ProvenDiagnosticOffset {
  uint64_t offset{UINT64_MAX};
  bool known{false};
  uint32_t pointer_relation{POINTER_NOT_QUERIED};
};

void reset_replay_observation(
    psimdjson_test_replay_observation *observation
) noexcept {
  *observation = {};
  observation->primary_error = static_cast<int32_t>(simdjson::UNINITIALIZED);
  observation->replay_error = static_cast<int32_t>(simdjson::UNINITIALIZED);
  observation->location_error = static_cast<int32_t>(simdjson::UNINITIALIZED);
  observation->derived_offset = UINT64_MAX;
}

ProvenDiagnosticOffset checked_diagnostic_offset(
    std::uintptr_t input_addr,
    size_t input_len,
    std::uintptr_t location_addr
) noexcept {
  ProvenDiagnosticOffset result;
  const auto max_addr = std::numeric_limits<std::uintptr_t>::max();
  if (input_len > max_addr - input_addr) {
    result.pointer_relation = POINTER_ADDRESS_OVERFLOW;
    return result;
  }

  const auto end_addr = input_addr + input_len;
  if (location_addr < input_addr || location_addr > end_addr) {
    result.pointer_relation = POINTER_OUT_OF_RANGE;
    return result;
  }
  if (location_addr == end_addr) {
    result.pointer_relation = POINTER_AT_END;
    return result;
  }

  const auto delta = location_addr - input_addr;
  if (delta > std::numeric_limits<uint64_t>::max()) {
    result.pointer_relation = POINTER_OUT_OF_RANGE;
    return result;
  }
  result.offset = static_cast<uint64_t>(delta);
  result.known = true;
  result.pointer_relation = POINTER_IN_BOUNDS;
  return result;
}

bool is_ordinary_diagnostic_error(simdjson::error_code error) noexcept {
  return map_error(error) == PURE_SIMDJSON_ERR_INVALID_JSON;
}

void record_replay_pass(
    psimdjson_test_replay_observation *observation,
    uint32_t replay_pass,
    DiagnosticReplayLimits limits
) noexcept {
  observation->replay_pass = replay_pass;
  observation->pass_count++;
  if (replay_pass == REPLAY_PASS_RAW_JSON) {
    observation->first_max_capacity = limits.max_capacity;
    observation->first_max_depth = limits.max_depth;
  } else {
    observation->second_max_capacity = limits.max_capacity;
    observation->second_max_depth = limits.max_depth;
  }
}

ReplayDisposition record_replay_failure(
    psimdjson_parser *parser,
    simdjson::ondemand::document &document,
    const uint8_t *input_ptr,
    size_t input_len,
    simdjson::error_code error,
    uint32_t replay_pass,
    psimdjson_test_replay_observation *observation
) noexcept {
  observation->replay_pass = replay_pass;
  observation->replay_error = static_cast<int32_t>(error);
  if (!is_ordinary_diagnostic_error(error)) {
    return ReplayDisposition::stopped;
  }

  const char *location = nullptr;
  const auto location_error = document.current_location().get(location);
  observation->location_error = static_cast<int32_t>(location_error);
  if (location_error != simdjson::SUCCESS || location == nullptr) {
    return ReplayDisposition::stopped;
  }

  const auto proven = checked_diagnostic_offset(
      reinterpret_cast<std::uintptr_t>(input_ptr),
      input_len,
      reinterpret_cast<std::uintptr_t>(location)
  );
  observation->pointer_relation = proven.pointer_relation;
  observation->derived_offset = proven.offset;
  if (proven.known) {
    parser->last_error_offset = proven.offset;
    parser->last_error_has_offset = true;
  }
  return ReplayDisposition::stopped;
}

template <typename OnDemandValue>
simdjson::error_code consume_ondemand_value(
    OnDemandValue &value,
    size_t current_depth,
    size_t max_depth
) {
  simdjson::ondemand::json_type type;
  auto error = value.type().get(type);
  if (error != simdjson::SUCCESS) {
    return error;
  }

  switch (type) {
    case simdjson::ondemand::json_type::array: {
      if (current_depth == std::numeric_limits<size_t>::max()) {
        return simdjson::DEPTH_ERROR;
      }
      const size_t container_depth = current_depth + 1;
      if (container_depth >= max_depth) {
        return simdjson::DEPTH_ERROR;
      }

      simdjson::ondemand::array array;
      error = value.get_array().get(array);
      if (error != simdjson::SUCCESS) {
        return error;
      }
      for (auto child_result : array) {
        simdjson::ondemand::value child;
        error = child_result.get(child);
        if (error != simdjson::SUCCESS) {
          return error;
        }
        error = consume_ondemand_value(child, container_depth, max_depth);
        if (error != simdjson::SUCCESS) {
          return error;
        }
      }
      return simdjson::SUCCESS;
    }
    case simdjson::ondemand::json_type::object: {
      if (current_depth == std::numeric_limits<size_t>::max()) {
        return simdjson::DEPTH_ERROR;
      }
      const size_t container_depth = current_depth + 1;
      if (container_depth >= max_depth) {
        return simdjson::DEPTH_ERROR;
      }

      simdjson::ondemand::object object;
      error = value.get_object().get(object);
      if (error != simdjson::SUCCESS) {
        return error;
      }
      for (auto field_result : object) {
        simdjson::ondemand::field field;
        error = std::move(field_result).get(field);
        if (error != simdjson::SUCCESS) {
          return error;
        }
        std::string_view key;
        error = field.unescaped_key().get(key);
        if (error != simdjson::SUCCESS) {
          return error;
        }
        error = consume_ondemand_value(field.value(), container_depth, max_depth);
        if (error != simdjson::SUCCESS) {
          return error;
        }
      }
      return simdjson::SUCCESS;
    }
    case simdjson::ondemand::json_type::number: {
      simdjson::ondemand::number number;
      return value.get_number().get(number);
    }
    case simdjson::ondemand::json_type::string: {
      std::string_view string;
      return value.get_string().get(string);
    }
    case simdjson::ondemand::json_type::boolean: {
      bool boolean = false;
      return value.get_bool().get(boolean);
    }
    case simdjson::ondemand::json_type::null: {
      bool is_null = false;
      return value.is_null().get(is_null);
    }
    case simdjson::ondemand::json_type::unknown:
      return simdjson::TAPE_ERROR;
  }
  return simdjson::TAPE_ERROR;
}

ReplayDisposition replay_raw_json_location(
    psimdjson_parser *parser,
    const uint8_t *input_ptr,
    size_t input_len,
    DiagnosticReplayLimits limits,
    psimdjson_test_replay_observation *observation
) {
  if (limits.max_capacity > std::numeric_limits<size_t>::max()) {
    observation->replay_error = static_cast<int32_t>(simdjson::CAPACITY);
    return ReplayDisposition::stopped;
  }

  const auto replay_capacity = static_cast<size_t>(limits.max_capacity);
  simdjson::ondemand::parser replay_parser(replay_capacity);
  record_replay_pass(observation, REPLAY_PASS_RAW_JSON, limits);

  observation->allocation_count++;
  const auto allocate_error =
      replay_parser.allocate(input_len, static_cast<size_t>(limits.max_depth));
  if (allocate_error != simdjson::SUCCESS) {
    observation->replay_error = static_cast<int32_t>(allocate_error);
    return ReplayDisposition::stopped;
  }
  if (input_len > std::numeric_limits<size_t>::max() - simdjson::SIMDJSON_PADDING) {
    observation->replay_error = static_cast<int32_t>(simdjson::OUT_OF_CAPACITY);
    return ReplayDisposition::stopped;
  }

  simdjson::ondemand::document document;
  observation->iterate_count++;
  const auto iterate_error =
      replay_parser
          .iterate(input_ptr, input_len, input_len + simdjson::SIMDJSON_PADDING)
          .get(document);
  if (iterate_error != simdjson::SUCCESS) {
    // Upstream cannot provide current_location() when iterate() fails, including
    // EMPTY, UTF8_ERROR, UNESCAPED_CHARS, and UNCLOSED_STRING. No broader
    // malformed-input location coverage is inferred here.
    observation->replay_error = static_cast<int32_t>(iterate_error);
    return ReplayDisposition::stopped;
  }

  std::string_view raw_json;
  const auto consume_error = document.raw_json().get(raw_json);
  if (consume_error != simdjson::SUCCESS) {
    return record_replay_failure(
        parser,
        document,
        input_ptr,
        input_len,
        consume_error,
        REPLAY_PASS_RAW_JSON,
        observation
    );
  }
  if (!document.at_end()) {
    return record_replay_failure(
        parser,
        document,
        input_ptr,
        input_len,
        simdjson::TRAILING_CONTENT,
        REPLAY_PASS_RAW_JSON,
        observation
    );
  }

  observation->replay_error = static_cast<int32_t>(simdjson::SUCCESS);
  return ReplayDisposition::valid;
}

ReplayDisposition replay_recursive_location(
    psimdjson_parser *parser,
    const uint8_t *input_ptr,
    size_t input_len,
    DiagnosticReplayLimits limits,
    psimdjson_test_replay_observation *observation
) {
  if (limits.max_capacity > std::numeric_limits<size_t>::max()) {
    observation->replay_error = static_cast<int32_t>(simdjson::CAPACITY);
    return ReplayDisposition::stopped;
  }

  const auto replay_capacity = static_cast<size_t>(limits.max_capacity);
  simdjson::ondemand::parser replay_parser(replay_capacity);
  record_replay_pass(observation, REPLAY_PASS_RECURSIVE, limits);

  observation->allocation_count++;
  const auto allocate_error =
      replay_parser.allocate(input_len, static_cast<size_t>(limits.max_depth));
  if (allocate_error != simdjson::SUCCESS) {
    observation->replay_error = static_cast<int32_t>(allocate_error);
    return ReplayDisposition::stopped;
  }
  if (input_len > std::numeric_limits<size_t>::max() - simdjson::SIMDJSON_PADDING) {
    observation->replay_error = static_cast<int32_t>(simdjson::OUT_OF_CAPACITY);
    return ReplayDisposition::stopped;
  }

  simdjson::ondemand::document document;
  observation->iterate_count++;
  const auto iterate_error =
      replay_parser
          .iterate(input_ptr, input_len, input_len + simdjson::SIMDJSON_PADDING)
          .get(document);
  if (iterate_error != simdjson::SUCCESS) {
    observation->replay_error = static_cast<int32_t>(iterate_error);
    return ReplayDisposition::stopped;
  }

  const auto consume_error =
      consume_ondemand_value(document, 0, static_cast<size_t>(limits.max_depth));
  if (consume_error != simdjson::SUCCESS) {
    return record_replay_failure(
        parser,
        document,
        input_ptr,
        input_len,
        consume_error,
        REPLAY_PASS_RECURSIVE,
        observation
    );
  }
  if (!document.at_end()) {
    return record_replay_failure(
        parser,
        document,
        input_ptr,
        input_len,
        simdjson::TRAILING_CONTENT,
        REPLAY_PASS_RECURSIVE,
        observation
    );
  }

  observation->replay_error = static_cast<int32_t>(simdjson::SUCCESS);
  return ReplayDisposition::valid;
}

void capture_diagnostic_location(
    psimdjson_parser *parser,
    const uint8_t *input_ptr,
    size_t input_len,
    simdjson::error_code primary_error,
    psimdjson_test_replay_observation *observation
) {
  reset_replay_observation(observation);
  observation->primary_error = static_cast<int32_t>(primary_error);
  if (!is_ordinary_diagnostic_error(primary_error)) {
    return;
  }

  const DiagnosticReplayLimits limits{
      parser->max_capacity,
      parser->max_depth,
  };
  if (replay_raw_json_location(parser, input_ptr, input_len, limits, observation) !=
      ReplayDisposition::valid) {
    return;
  }
  static_cast<void>(
      replay_recursive_location(parser, input_ptr, input_len, limits, observation)
  );
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

class materialize_build_guard {
 public:
  explicit materialize_build_guard(psimdjson_doc *doc) noexcept : doc_(doc) {
    if (doc_ != nullptr && !doc_->materialize_in_progress) {
      doc_->materialize_in_progress = true;
      acquired_ = true;
    }
  }

  materialize_build_guard(const materialize_build_guard &) = delete;
  materialize_build_guard &operator=(const materialize_build_guard &) = delete;

  ~materialize_build_guard() noexcept {
    if (acquired_) {
      doc_->materialize_in_progress = false;
    }
  }

  bool acquired() const noexcept {
    return acquired_;
  }

 private:
  psimdjson_doc *doc_{nullptr};
  bool acquired_{false};
};

void clear_materialize_outputs(
    const psdj_internal_frame_t **out_frames,
    size_t *out_frame_count
) noexcept {
  if (out_frames != nullptr) {
    *out_frames = nullptr;
  }
  if (out_frame_count != nullptr) {
    *out_frame_count = 0;
  }
}

void set_frame_key(psdj_internal_frame_t &frame, std::string_view key) noexcept {
  frame.key_len = key.size();
  frame.key_ptr = key.empty() ? nullptr : reinterpret_cast<const uint8_t *>(key.data());
}

void reserve_materialize_frames(psimdjson_doc *doc, size_t child_hint) {
  const size_t required = doc->materialize_frames.size() + 1 + child_hint;
  if (required <= doc->materialize_frames.capacity()) {
    return;
  }

  size_t new_capacity = doc->materialize_frames.capacity();
  if (new_capacity == 0) {
    new_capacity = required;
  }
  while (new_capacity < required) {
    if (new_capacity > (std::numeric_limits<size_t>::max() / 2)) {
      new_capacity = required;
      break;
    }
    new_capacity *= 2;
  }
  doc->materialize_frames.reserve(new_capacity);
}

// simdjson saturates tape scope counts at 0xFFFFFF; saturated hints no longer
// reflect the real child count and should not drive scratch-vector reserves.
bool has_unsaturated_child_hint(size_t child_hint) noexcept {
  constexpr size_t SATURATED_SCOPE_COUNT = 0xFFFFFF;
  return child_hint < SATURATED_SCOPE_COUNT;
}

pure_simdjson_error_code_t append_materialize_frame(
    psimdjson_doc *doc,
    simdjson::dom::element element,
    std::string_view key,
    size_t depth
) {
  // Defense-in-depth: simdjson's parser cap catches user input first today.
  // Keep the materializer bound in case future refactors decouple those depths.
  constexpr size_t MAX_MATERIALIZE_FRAME_DEPTH = 1024;
  if (depth > MAX_MATERIALIZE_FRAME_DEPTH) {
    return PURE_SIMDJSON_ERR_DEPTH_LIMIT;
  }

  psdj_internal_frame_t frame{};
  // Go reads flags as the bool payload for ValueKindBool; stale high bits would
  // misdecode if a future kind reused this field.
  frame.flags = 0;
  set_frame_key(frame, key);

  const auto type = element.type();
  frame.kind = map_element_type(type);

  switch (type) {
    case simdjson::dom::element_type::ARRAY: {
      simdjson::dom::array array;
      const auto error = element.get_array().get(array);
      if (error != simdjson::SUCCESS) {
        return map_error(error);
      }

      const auto frame_index = doc->materialize_frames.size();
      const auto child_hint = array.size();
      if (has_unsaturated_child_hint(child_hint)) {
        reserve_materialize_frames(doc, child_hint);
      }
      doc->materialize_frames.push_back(frame);

      uint64_t child_count = 0;
      for (simdjson::dom::element child : array) {
        const auto child_rc =
            append_materialize_frame(doc, child, std::string_view{}, depth + 1);
        if (child_rc != PURE_SIMDJSON_OK) {
          return child_rc;
        }
        child_count++;
        if (child_count > std::numeric_limits<uint32_t>::max()) {
          return PURE_SIMDJSON_ERR_INTERNAL;
        }
      }
      doc->materialize_frames[frame_index].child_count = uint32_t(child_count);
      return PURE_SIMDJSON_OK;
    }
    case simdjson::dom::element_type::OBJECT: {
      simdjson::dom::object object;
      const auto error = element.get_object().get(object);
      if (error != simdjson::SUCCESS) {
        return map_error(error);
      }

      const auto frame_index = doc->materialize_frames.size();
      const auto child_hint = object.size();
      if (has_unsaturated_child_hint(child_hint)) {
        reserve_materialize_frames(doc, child_hint);
      }
      doc->materialize_frames.push_back(frame);

      uint64_t child_count = 0;
      for (simdjson::dom::key_value_pair field : object) {
        const auto child_rc = append_materialize_frame(doc, field.value, field.key, depth + 1);
        if (child_rc != PURE_SIMDJSON_OK) {
          return child_rc;
        }
        child_count++;
        if (child_count > std::numeric_limits<uint32_t>::max()) {
          return PURE_SIMDJSON_ERR_INTERNAL;
        }
      }
      doc->materialize_frames[frame_index].child_count = uint32_t(child_count);
      return PURE_SIMDJSON_OK;
    }
    case simdjson::dom::element_type::INT64: {
      const auto error = element.get_int64().get(frame.int64_value);
      if (error != simdjson::SUCCESS) {
        return map_error(error);
      }
      doc->materialize_frames.push_back(frame);
      return PURE_SIMDJSON_OK;
    }
    case simdjson::dom::element_type::UINT64: {
      const auto error = element.get_uint64().get(frame.uint64_value);
      if (error != simdjson::SUCCESS) {
        return map_error(error);
      }
      doc->materialize_frames.push_back(frame);
      return PURE_SIMDJSON_OK;
    }
    case simdjson::dom::element_type::DOUBLE: {
      const auto error = element.get_double().get(frame.float64_value);
      if (error != simdjson::SUCCESS) {
        return map_error(error);
      }
      doc->materialize_frames.push_back(frame);
      return PURE_SIMDJSON_OK;
    }
    case simdjson::dom::element_type::STRING: {
      std::string_view value;
      const auto error = element.get_string().get(value);
      if (error != simdjson::SUCCESS) {
        return map_error(error);
      }
      frame.string_len = value.size();
      frame.string_ptr =
          value.empty() ? nullptr : reinterpret_cast<const uint8_t *>(value.data());
      doc->materialize_frames.push_back(frame);
      return PURE_SIMDJSON_OK;
    }
    case simdjson::dom::element_type::BOOL: {
      bool value = false;
      const auto error = element.get_bool().get(value);
      if (error != simdjson::SUCCESS) {
        return map_error(error);
      }
      frame.flags = value ? 1 : 0;
      doc->materialize_frames.push_back(frame);
      return PURE_SIMDJSON_OK;
    }
    case simdjson::dom::element_type::NULL_VALUE:
      doc->materialize_frames.push_back(frame);
      return PURE_SIMDJSON_OK;
    case simdjson::dom::element_type::BIGINT: {
      std::string_view value;
      const auto error = element.get_bigint().get(value);
      if (error != simdjson::SUCCESS) {
        return map_error(error);
      }
      frame.string_len = value.size();
      frame.string_ptr =
          value.empty() ? nullptr : reinterpret_cast<const uint8_t *>(value.data());
      doc->materialize_frames.push_back(frame);
      return PURE_SIMDJSON_OK;
    }
  }

  return PURE_SIMDJSON_ERR_INTERNAL;
}

}  // namespace

pure_simdjson_error_code_t psimdjson_get_implementation_name_len(size_t *out_len) noexcept {
  try {
    if (out_len == nullptr) {
      return invalid_argument();
    }

    *out_len = implementation_name().size();
    return PURE_SIMDJSON_OK;
  } PSIMDJSON_CATCH_CPP_EXCEPTIONS(__func__)
}

pure_simdjson_error_code_t psimdjson_copy_implementation_name(
    uint8_t *dst,
    size_t dst_cap,
    size_t *out_written
) noexcept {
  try {
    return copy_bytes(implementation_name(), dst, dst_cap, out_written);
  } PSIMDJSON_CATCH_CPP_EXCEPTIONS(__func__)
}

pure_simdjson_error_code_t psimdjson_native_alloc_stats_reset(void) noexcept {
  try {
    psimdjson::native_alloc_telemetry::reset();
    return PURE_SIMDJSON_OK;
  } PSIMDJSON_CATCH_CPP_EXCEPTIONS(__func__)
}

pure_simdjson_error_code_t psimdjson_native_alloc_stats_snapshot(
    pure_simdjson_native_alloc_stats_t *out_stats
) noexcept {
  try {
    return psimdjson::native_alloc_telemetry::snapshot(out_stats);
  } PSIMDJSON_CATCH_CPP_EXCEPTIONS(__func__)
}

size_t psimdjson_padding_bytes(void) noexcept {
  return simdjson::SIMDJSON_PADDING;
}

pure_simdjson_error_code_t psimdjson_parser_new(psimdjson_parser **out_parser) noexcept {
  return psimdjson_parser_new_configured(
      DEFAULT_MAX_CAPACITY,
      DEFAULT_MAX_DEPTH,
      out_parser
  );
}

pure_simdjson_error_code_t psimdjson_parser_new_configured(
    uint64_t max_capacity,
    uint32_t max_depth,
    psimdjson_parser **out_parser
) noexcept {
  try {
    if (out_parser == nullptr) {
      return invalid_argument();
    }
    if (max_capacity != 0 &&
        (max_capacity < MIN_MAX_CAPACITY || max_capacity > DEFAULT_MAX_CAPACITY)) {
      return invalid_argument();
    }

    const uint64_t effective_max_capacity =
        max_capacity == 0 ? DEFAULT_MAX_CAPACITY : max_capacity;
    const uint32_t effective_max_depth =
        max_depth == 0 ? DEFAULT_MAX_DEPTH : max_depth;
    auto parser = std::make_unique<psimdjson_parser>();
    parser->max_capacity = effective_max_capacity;
    parser->max_depth = effective_max_depth;
    parser->parser.set_max_capacity(static_cast<size_t>(parser->max_capacity));
    const auto allocate_error =
        parser->parser.allocate(0, static_cast<size_t>(parser->max_depth));
    if (allocate_error != simdjson::SUCCESS) {
      return map_error(allocate_error);
    }
    parser->parser.number_as_string(true);
    *out_parser = parser.release();
    return PURE_SIMDJSON_OK;
  } PSIMDJSON_CATCH_CPP_EXCEPTIONS(__func__)
}

pure_simdjson_error_code_t psimdjson_parser_free(psimdjson_parser *parser) noexcept {
  try {
    if (parser == nullptr) {
      return invalid_argument();
    }

    delete parser;
    return PURE_SIMDJSON_OK;
  } PSIMDJSON_CATCH_CPP_EXCEPTIONS(__func__)
}

pure_simdjson_error_code_t psimdjson_parser_reset_diagnostics(
    psimdjson_parser *parser
) noexcept {
  try {
    if (parser == nullptr) {
      return invalid_argument();
    }

    clear_last_error(parser);
    return PURE_SIMDJSON_OK;
  } PSIMDJSON_CATCH_CPP_EXCEPTIONS(__func__)
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
    reset_replay_observation(&parser->replay_observation);
    auto doc = std::make_unique<psimdjson_doc>();
    simdjson::dom::element root;
    const auto error =
        parser->parser.parse_into_document(doc->document, input_ptr, input_len, false).get(root);
    if (error != simdjson::SUCCESS) {
      set_last_error(parser, error);
      // Valid input pays no replay cost. Eligible DOM syntax failures use at
      // most two fresh, caller-bounded O(input length) On-Demand scans.
      capture_diagnostic_location(
          parser,
          input_ptr,
          input_len,
          error,
          &parser->replay_observation
      );
      return map_error(error);
    }

    parser->replay_observation.primary_error =
        static_cast<int32_t>(simdjson::SUCCESS);
    clear_last_error(parser);
    doc->root.value = root;
    *out_doc = doc.release();
    return PURE_SIMDJSON_OK;
  } PSIMDJSON_CATCH_PARSER_CPP_EXCEPTIONS(__func__, parser)
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
  } PSIMDJSON_CATCH_CPP_EXCEPTIONS(__func__)
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

    return copy_bytes(
        std::string_view(parser->last_error.data(), parser->last_error.size()),
        dst,
        dst_cap,
        out_written
    );
  } PSIMDJSON_CATCH_CPP_EXCEPTIONS(__func__)
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
  } PSIMDJSON_CATCH_CPP_EXCEPTIONS(__func__)
}

pure_simdjson_error_code_t psimdjson_parser_get_last_error_has_offset(
    const psimdjson_parser *parser,
    uint8_t *out_has_offset
) noexcept {
  try {
    if (parser == nullptr || out_has_offset == nullptr) {
      return invalid_argument();
    }

    *out_has_offset = parser->last_error_has_offset ? 1 : 0;
    return PURE_SIMDJSON_OK;
  } PSIMDJSON_CATCH_CPP_EXCEPTIONS(__func__)
}

pure_simdjson_error_code_t psimdjson_doc_free(psimdjson_doc *doc) noexcept {
  try {
    if (doc == nullptr) {
      return invalid_argument();
    }

    delete doc;
    return PURE_SIMDJSON_OK;
  } PSIMDJSON_CATCH_CPP_EXCEPTIONS(__func__)
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
  } PSIMDJSON_CATCH_CPP_EXCEPTIONS(__func__)
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
  } PSIMDJSON_CATCH_CPP_EXCEPTIONS(__func__)
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
  } PSIMDJSON_CATCH_CPP_EXCEPTIONS(__func__)
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

    *out_kind = map_element_type(element_at(doc, json_index).type());
    return PURE_SIMDJSON_OK;
  } PSIMDJSON_CATCH_CPP_EXCEPTIONS(__func__)
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
  } PSIMDJSON_CATCH_CPP_EXCEPTIONS(__func__)
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
  } PSIMDJSON_CATCH_CPP_EXCEPTIONS(__func__)
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
  } PSIMDJSON_CATCH_CPP_EXCEPTIONS(__func__)
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
  } PSIMDJSON_CATCH_CPP_EXCEPTIONS(__func__)
}

pure_simdjson_error_code_t psimdjson_element_get_bigint_view(
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
    const auto error = element_at(doc, json_index).get_bigint().get(value);
    if (error != simdjson::SUCCESS) {
      return map_error(error);
    }

    *out_len = value.size();
    *out_ptr = value.empty() ? nullptr : reinterpret_cast<const uint8_t *>(value.data());
    return PURE_SIMDJSON_OK;
  } PSIMDJSON_CATCH_CPP_EXCEPTIONS(__func__)
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
  } PSIMDJSON_CATCH_CPP_EXCEPTIONS(__func__)
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
  } PSIMDJSON_CATCH_CPP_EXCEPTIONS(__func__)
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
  } PSIMDJSON_CATCH_CPP_EXCEPTIONS(__func__)
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
  } PSIMDJSON_CATCH_CPP_EXCEPTIONS(__func__)
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
  } PSIMDJSON_CATCH_CPP_EXCEPTIONS(__func__)
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
  } PSIMDJSON_CATCH_CPP_EXCEPTIONS(__func__)
}

pure_simdjson_error_code_t psimdjson_materialize_build(psimdjson_doc *doc,
                                                       uint64_t json_index,
                                                       const psdj_internal_frame_t **out_frames,
                                                       size_t *out_frame_count) noexcept {
  try {
    if (doc == nullptr || out_frames == nullptr || out_frame_count == nullptr) {
      clear_materialize_outputs(out_frames, out_frame_count);
      return invalid_argument();
    }

    clear_materialize_outputs(out_frames, out_frame_count);
    materialize_build_guard guard(doc);
    if (!guard.acquired()) {
      return PURE_SIMDJSON_ERR_PARSER_BUSY;
    }

    doc->materialize_frames.clear();
    const auto rc =
        append_materialize_frame(doc, element_at(doc, json_index), std::string_view{}, 0);
    if (rc != PURE_SIMDJSON_OK) {
      doc->materialize_frames.clear();
      clear_materialize_outputs(out_frames, out_frame_count);
      return rc;
    }

    if (doc->materialize_frames.empty()) {
      return PURE_SIMDJSON_ERR_INTERNAL;
    }

    // Frame pointers are returned only after traversal completes. The returned
    // span remains valid until the next psimdjson_materialize_build call on the
    // same doc, which clears/reuses doc->materialize_frames.
    *out_frames = doc->materialize_frames.data();
    *out_frame_count = doc->materialize_frames.size();
    return PURE_SIMDJSON_OK;
  } PSIMDJSON_CATCH_CPP_EXCEPTIONS(__func__)
}

// Test scaffolding: always returns PURE_SIMDJSON_ERR_PARSER_BUSY when doc
// is valid. We acquire the materialize_build_guard, then call
// psimdjson_materialize_build which re-acquires the same guard and fails
// fast. This simulates the "materialize in progress" state so tests can
// exercise the reentry guard without racing real workloads.
pure_simdjson_error_code_t psimdjson_test_hold_materialize_guard(psimdjson_doc *doc,
                                                                 uint64_t json_index) noexcept {
  try {
    if (doc == nullptr) {
      return invalid_argument();
    }

    materialize_build_guard guard(doc);
    if (!guard.acquired()) {
      return PURE_SIMDJSON_ERR_PARSER_BUSY;
    }

    const psdj_internal_frame_t *frames = nullptr;
    size_t frame_count = 0;
    return psimdjson_materialize_build(doc, json_index, &frames, &frame_count);
  } PSIMDJSON_CATCH_CPP_EXCEPTIONS(__func__)
}

pure_simdjson_error_code_t psimdjson_test_characterize_diagnostic(
    const uint8_t *input_ptr,
    size_t input_len,
    uint64_t max_capacity,
    uint32_t max_depth,
    int32_t *out_parse_status,
    uint64_t *out_offset,
    uint8_t *out_has_offset,
    psimdjson_test_replay_observation *out_observation
) noexcept {
  try {
    if ((input_len != 0 && input_ptr == nullptr) ||
        out_parse_status == nullptr ||
        out_offset == nullptr ||
        out_has_offset == nullptr ||
        out_observation == nullptr) {
      return invalid_argument();
    }
    if (input_len > std::numeric_limits<size_t>::max() - simdjson::SIMDJSON_PADDING) {
      return invalid_argument();
    }

    std::vector<uint8_t> padded_input(
        input_len + simdjson::SIMDJSON_PADDING,
        uint8_t{0}
    );
    if (input_len != 0) {
      std::memcpy(padded_input.data(), input_ptr, input_len);
    }

    psimdjson_parser *parser = nullptr;
    const auto new_rc =
        psimdjson_parser_new_configured(max_capacity, max_depth, &parser);
    if (new_rc != PURE_SIMDJSON_OK) {
      return new_rc;
    }

    psimdjson_doc *doc = nullptr;
    const auto parse_rc =
        psimdjson_parser_parse(parser, padded_input.data(), input_len, &doc);
    if (doc != nullptr) {
      const auto doc_free_rc = psimdjson_doc_free(doc);
      if (doc_free_rc != PURE_SIMDJSON_OK) {
        static_cast<void>(psimdjson_parser_free(parser));
        return doc_free_rc;
      }
    }

    *out_parse_status = static_cast<int32_t>(parse_rc);
    *out_offset = parser->last_error_offset;
    *out_has_offset = parser->last_error_has_offset ? 1 : 0;
    *out_observation = parser->replay_observation;

    const auto parser_free_rc = psimdjson_parser_free(parser);
    return parser_free_rc;
  } PSIMDJSON_CATCH_CPP_EXCEPTIONS(__func__)
}

pure_simdjson_error_code_t psimdjson_test_recursive_replay_observation(
    const uint8_t *input_ptr,
    size_t input_len,
    uint64_t max_capacity,
    uint32_t max_depth,
    uint64_t *out_offset,
    uint8_t *out_has_offset,
    psimdjson_test_replay_observation *out_observation
) noexcept {
  try {
    if ((input_len != 0 && input_ptr == nullptr) ||
        out_offset == nullptr ||
        out_has_offset == nullptr ||
        out_observation == nullptr) {
      return invalid_argument();
    }
    if (input_len > std::numeric_limits<size_t>::max() - simdjson::SIMDJSON_PADDING) {
      return invalid_argument();
    }

    const uint64_t effective_max_capacity =
        max_capacity == 0 ? DEFAULT_MAX_CAPACITY : max_capacity;
    const uint32_t effective_max_depth =
        max_depth == 0 ? DEFAULT_MAX_DEPTH : max_depth;
    std::vector<uint8_t> padded_input(
        input_len + simdjson::SIMDJSON_PADDING,
        uint8_t{0}
    );
    if (input_len != 0) {
      std::memcpy(padded_input.data(), input_ptr, input_len);
    }

    psimdjson_parser parser;
    parser.max_capacity = effective_max_capacity;
    parser.max_depth = effective_max_depth;
    clear_last_error(&parser);
    reset_replay_observation(out_observation);
    static_cast<void>(replay_recursive_location(
        &parser,
        padded_input.data(),
        input_len,
        DiagnosticReplayLimits{effective_max_capacity, effective_max_depth},
        out_observation
    ));

    *out_offset = parser.last_error_offset;
    *out_has_offset = parser.last_error_has_offset ? 1 : 0;
    return PURE_SIMDJSON_OK;
  } PSIMDJSON_CATCH_CPP_EXCEPTIONS(__func__)
}

pure_simdjson_error_code_t psimdjson_test_terminal_diagnostic_observation(
    uint32_t terminal_case,
    uint64_t *out_offset,
    uint8_t *out_has_offset,
    psimdjson_test_replay_observation *out_observation
) noexcept {
  try {
    if (out_offset == nullptr || out_has_offset == nullptr || out_observation == nullptr) {
      return invalid_argument();
    }

    simdjson::error_code error;
    switch (terminal_case) {
      case 1:
        error = simdjson::CAPACITY;
        break;
      case 2:
        error = simdjson::DEPTH_ERROR;
        break;
      case 3:
        error = simdjson::MEMALLOC;
        break;
      case 4:
        error = simdjson::UNEXPECTED_ERROR;
        break;
      default:
        return invalid_argument();
    }

    psimdjson_parser parser;
    parser.max_capacity = DEFAULT_MAX_CAPACITY;
    parser.max_depth = DEFAULT_MAX_DEPTH;
    clear_last_error(&parser);
    uint8_t padded_input[simdjson::SIMDJSON_PADDING]{};
    capture_diagnostic_location(
        &parser,
        padded_input,
        0,
        error,
        out_observation
    );

    *out_offset = parser.last_error_offset;
    *out_has_offset = parser.last_error_has_offset ? 1 : 0;
    return PURE_SIMDJSON_OK;
  } PSIMDJSON_CATCH_CPP_EXCEPTIONS(__func__)
}

pure_simdjson_error_code_t psimdjson_test_checked_diagnostic_offset(
    uintptr_t input_addr,
    size_t input_len,
    uintptr_t location_addr,
    uint64_t *out_offset,
    uint8_t *out_has_offset
) noexcept {
  try {
    if (out_offset == nullptr || out_has_offset == nullptr) {
      return invalid_argument();
    }

    const auto proven =
        checked_diagnostic_offset(input_addr, input_len, location_addr);
    *out_offset = proven.offset;
    *out_has_offset = proven.known ? 1 : 0;
    return PURE_SIMDJSON_OK;
  } PSIMDJSON_CATCH_CPP_EXCEPTIONS(__func__)
}

pure_simdjson_error_code_t psimdjson_test_force_cpp_exception(void) noexcept {
  try {
    throw std::runtime_error("forced cpp exception");
  } PSIMDJSON_CATCH_CPP_EXCEPTIONS(__func__)
}
