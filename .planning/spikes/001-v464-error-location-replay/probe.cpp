#include "simdjson.h"

#include <cstdint>
#include <iostream>
#include <string>
#include <string_view>
#include <utility>
#include <vector>

namespace {

struct ProbeCase {
  std::string name;
  std::string input;
};

enum class ReplayMode {
  raw_json,
  recursive_walk,
};

const char *error_name(simdjson::error_code error) {
  return error == simdjson::SUCCESS ? "SUCCESS" : simdjson::error_message(error);
}

const char *mode_name(ReplayMode mode) {
  return mode == ReplayMode::raw_json ? "raw_json" : "recursive_walk";
}

simdjson::error_code consume_value(simdjson::ondemand::value &value) {
  simdjson::ondemand::json_type type;
  simdjson::error_code error = value.type().get(type);
  if (error != simdjson::SUCCESS) {
    return error;
  }

  switch (type) {
    case simdjson::ondemand::json_type::array: {
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
        error = consume_value(child);
        if (error != simdjson::SUCCESS) {
          return error;
        }
      }
      return simdjson::SUCCESS;
    }
    case simdjson::ondemand::json_type::object: {
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
        error = consume_value(field.value());
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

simdjson::error_code consume_document(simdjson::ondemand::document &document) {
  simdjson::ondemand::json_type type;
  simdjson::error_code error = document.type().get(type);
  if (error != simdjson::SUCCESS) {
    return error;
  }

  switch (type) {
    case simdjson::ondemand::json_type::array: {
      simdjson::ondemand::array array;
      error = document.get_array().get(array);
      if (error != simdjson::SUCCESS) {
        return error;
      }
      for (auto child_result : array) {
        simdjson::ondemand::value child;
        error = child_result.get(child);
        if (error != simdjson::SUCCESS) {
          return error;
        }
        error = consume_value(child);
        if (error != simdjson::SUCCESS) {
          return error;
        }
      }
      return simdjson::SUCCESS;
    }
    case simdjson::ondemand::json_type::object: {
      simdjson::ondemand::object object;
      error = document.get_object().get(object);
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
        error = consume_value(field.value());
        if (error != simdjson::SUCCESS) {
          return error;
        }
      }
      return simdjson::SUCCESS;
    }
    case simdjson::ondemand::json_type::number: {
      simdjson::ondemand::number number;
      return document.get_number().get(number);
    }
    case simdjson::ondemand::json_type::string: {
      std::string_view string;
      return document.get_string().get(string);
    }
    case simdjson::ondemand::json_type::boolean: {
      bool boolean = false;
      return document.get_bool().get(boolean);
    }
    case simdjson::ondemand::json_type::null: {
      bool is_null = false;
      return document.is_null().get(is_null);
    }
    case simdjson::ondemand::json_type::unknown:
      return simdjson::TAPE_ERROR;
  }
  return simdjson::TAPE_ERROR;
}

struct ReplayObservation {
  std::string iterate_error;
  std::string consume_error;
  std::string replay_outcome;
  std::string replay_error;
  std::string location_status;
  std::string pointer_relation;
  std::string offset;
  bool known;
};

ReplayObservation replay(const ProbeCase &candidate, ReplayMode mode) {
  simdjson::padded_string replay_input(candidate.input.data(), candidate.input.size());
  simdjson::ondemand::parser replay_parser;
  simdjson::ondemand::document replay_doc;
  const simdjson::error_code iterate_error =
      replay_parser.iterate(replay_input).get(replay_doc);

  simdjson::error_code consume_error = simdjson::UNINITIALIZED;
  ReplayObservation observation{
      error_name(iterate_error),
      "not_run",
      "iterate_failed",
      error_name(iterate_error),
      "not_queried",
      "not_queried",
      "-",
      false,
  };

  if (iterate_error == simdjson::SUCCESS) {
    if (mode == ReplayMode::raw_json) {
      std::string_view raw_json;
      consume_error = replay_doc.raw_json().get(raw_json);
    } else {
      consume_error = consume_document(replay_doc);
    }
    observation.consume_error = error_name(consume_error);
    if (consume_error != simdjson::SUCCESS) {
      observation.replay_outcome = "consume_failed";
      observation.replay_error = error_name(consume_error);
    } else if (!replay_doc.at_end()) {
      observation.replay_outcome = "trailing_content";
      observation.replay_error = "TRAILING_CONTENT";
    } else {
      observation.replay_outcome = "valid";
      observation.replay_error = "SUCCESS";
    }

    if (observation.replay_outcome != "valid") {
      const char *location = nullptr;
      const simdjson::error_code location_error =
          replay_doc.current_location().get(location);
      observation.location_status = error_name(location_error);

      if (location_error == simdjson::SUCCESS && location != nullptr) {
        const std::uintptr_t begin =
            reinterpret_cast<std::uintptr_t>(replay_input.data());
        const std::uintptr_t end = begin + candidate.input.size();
        const std::uintptr_t observed =
            reinterpret_cast<std::uintptr_t>(location);

        if (observed >= begin && observed < end) {
          observation.pointer_relation = "in_bounds";
          observation.offset = std::to_string(observed - begin);
          observation.known = true;
        } else if (observed == end) {
          observation.pointer_relation = "at_end";
        } else {
          observation.pointer_relation = "out_of_range";
        }
      }
    }
  }
  return observation;
}

void print_observation(
    const ProbeCase &candidate,
    std::string_view mode,
    simdjson::error_code dom_error,
    const ReplayObservation &observation
) {
  std::cout << candidate.name << '\t' << mode << '\t'
            << candidate.input.size() << '\t' << error_name(dom_error) << '\t'
            << observation.iterate_error << '\t' << observation.consume_error
            << '\t' << observation.replay_outcome << '\t'
            << observation.replay_error << '\t' << observation.location_status
            << '\t' << observation.pointer_relation << '\t'
            << observation.offset << '\t'
            << (observation.known ? "true" : "false") << '\n';
}

void probe(const ProbeCase &candidate) {
  simdjson::padded_string dom_input(candidate.input.data(), candidate.input.size());
  simdjson::dom::parser dom_parser;
  simdjson::dom::element dom_root;
  const simdjson::error_code dom_error = dom_parser.parse(dom_input).get(dom_root);

  const ReplayObservation raw = replay(candidate, ReplayMode::raw_json);
  const ReplayObservation recursive = replay(candidate, ReplayMode::recursive_walk);
  const ReplayObservation hybrid =
      raw.replay_outcome == "valid" ? recursive : raw;

  print_observation(candidate, mode_name(ReplayMode::raw_json), dom_error, raw);
  print_observation(
      candidate, mode_name(ReplayMode::recursive_walk), dom_error, recursive);
  print_observation(candidate, "hybrid", dom_error, hybrid);
}

}  // namespace

int main() {
  std::string invalid_utf8 = "{\"x\":\"";
  invalid_utf8.push_back(static_cast<char>(0xff));
  invalid_utf8 += "\"}";

  const std::vector<ProbeCase> cases = {
      {"empty", ""},
      {"invalid_utf8", invalid_utf8},
      {"unclosed_string", "{\"x\":\"abc}"},
      {"array_trailing_comma", "[1,]"},
      {"trailing_content", "{\"a\":1} trailing"},
      {"missing_object_key", "{\"double\":13.06,false,\"integer\":-343}"},
      {"unexpected_root_token", "x"},
      {"extra_closing_bracket", "[\"extra close\"]]"},
      {"mismatched_container", "{\"a\":[1,2}"},
  };

  std::cout << "# simdjson_ref=v4.6.4\n";
  std::cout
      << "case\tmode\tbytes\tdom_error\titerate_error\tconsume_error"
         "\treplay_outcome\treplay_error\tlocation_status\tpointer_relation"
         "\toffset\tknown\n";
  for (const ProbeCase &candidate : cases) {
    probe(candidate);
  }
  return 0;
}
