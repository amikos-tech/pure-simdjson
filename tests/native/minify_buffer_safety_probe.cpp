// Durable minify buffer-safety contract.
//
// For every simdjson implementation supported by the host, this probe runs
// the same 12 fixtures with an exact-size separate destination and with
// dst == src. ASan/UBSan guard the buffer boundary; semantic checks require
// the aliased result to match the separate-buffer result byte for byte.

#include "simdjson.h"

#include <algorithm>
#include <cstdio>
#include <cstring>
#include <string>
#include <utility>
#include <vector>

using namespace simdjson;

namespace {

constexpr size_t kCasesPerKernel = 12;

struct CaseResult {
  std::string kernel;
  std::string fixture;
  error_code separate_error = UNINITIALIZED;
  error_code aliased_error = UNINITIALIZED;
  size_t separate_len = 0;
  size_t aliased_len = 0;
  bool aliased_matches_reference = false;

  bool passed() const {
    return separate_error == SUCCESS && aliased_error == SUCCESS &&
           separate_len == aliased_len && aliased_matches_reference;
  }
};

// ASan places a redzone immediately after this exact-size allocation, so any
// write at or beyond len is detected instead of relying on spare capacity.
struct ExactBuffer {
  explicit ExactBuffer(size_t length)
      : data(new uint8_t[length == 0 ? 1 : length]), len(length) {}
  ~ExactBuffer() { delete[] data; }

  ExactBuffer(const ExactBuffer &) = delete;
  ExactBuffer &operator=(const ExactBuffer &) = delete;

  uint8_t *data;
  size_t len;
};

CaseResult run_case(const implementation *impl,
                    const std::string &fixture_name,
                    const std::string &json) {
  CaseResult result;
  result.kernel = impl->name();
  result.fixture = fixture_name;

  const auto *source = reinterpret_cast<const uint8_t *>(json.data());
  const size_t source_len = json.size();
  std::vector<uint8_t> reference;

  {
    ExactBuffer destination(source_len);
    result.separate_error =
        impl->minify(source, source_len, destination.data, result.separate_len);
    if (result.separate_error == SUCCESS) {
      reference.assign(destination.data,
                       destination.data + result.separate_len);
    }
  }

  {
    ExactBuffer buffer(source_len);
    std::memcpy(buffer.data, json.data(), source_len);
    result.aliased_error =
        impl->minify(buffer.data, source_len, buffer.data, result.aliased_len);
    if (result.aliased_error == SUCCESS &&
        result.aliased_len == reference.size() &&
        std::memcmp(buffer.data, reference.data(), result.aliased_len) == 0) {
      result.aliased_matches_reference = true;
    }
  }

  return result;
}

std::vector<std::pair<std::string, std::string>> fixtures() {
  std::string nested =
      R"({"a":[1,2,{"b":"c"},[true,false,null]],"d":{"e":{"f":1.5e10}}})";
  std::string escaped =
      R"({"s":"line1\nline2\ttab\\backslash\"quote","u":"éècafé"})";
  std::string whitespace_heavy =
      "{\n  \"a\"   :    1,\n  \"b\"  :   [   1,   2,   3   ]  \n}\n\n\n";
  std::string wide_unicode = R"({"emoji":"😀👍","cjk":"你好"})";

  auto pad_with_whitespace = [](std::string json, size_t target_len) {
    while (json.size() < target_len) {
      json.insert(json.size() - 1, " ");
    }
    return json;
  };

  std::vector<std::pair<std::string, std::string>> cases;
  cases.emplace_back("nested", nested);
  cases.emplace_back("escaped_strings", escaped);
  cases.emplace_back("whitespace_heavy", whitespace_heavy);
  cases.emplace_back("wide_unicode", wide_unicode);
  cases.emplace_back("boundary_63", pad_with_whitespace(R"({"a":1})", 63));
  cases.emplace_back("boundary_64", pad_with_whitespace(R"({"a":1})", 64));
  cases.emplace_back("boundary_65", pad_with_whitespace(R"({"a":1})", 65));
  cases.emplace_back("boundary_128", pad_with_whitespace(R"({"a":1})", 128));
  cases.emplace_back("empty_object", "{}");
  cases.emplace_back("single_char", "1");

  {
    std::string large = "[";
    for (int index = 0; index < 2000; ++index) {
      if (index != 0) {
        large += ",     ";
      }
      large += "   " + std::to_string(index) + "   ";
    }
    large += "]";
    cases.emplace_back("large_whitespace_array_~18kb", std::move(large));
  }

  {
    std::string large = "[";
    for (int index = 0; index < 500; ++index) {
      if (index != 0) {
        large += ",";
      }
      large += R"({"id":)" + std::to_string(index) +
               R"(,"name":"item\t)" + std::to_string(index) +
               R"(\n","tags":["a","b","c"]})";
    }
    large += "]";
    cases.emplace_back("large_nested_objects_~25kb", std::move(large));
  }

  return cases;
}

std::vector<const implementation *> supported_implementations() {
  std::vector<const implementation *> supported;
  for (const implementation *impl : get_available_implementations()) {
    if (impl->supported_by_runtime_system()) {
      supported.push_back(impl);
    }
  }
  std::sort(supported.begin(), supported.end(),
            [](const implementation *left, const implementation *right) {
              return left->name() < right->name();
            });
  return supported;
}

std::string kernel_list(
    const std::vector<const implementation *> &implementations) {
  std::string result;
  for (const implementation *impl : implementations) {
    if (!result.empty()) {
      result += ',';
    }
    result += impl->name();
  }
  return result.empty() ? "none" : result;
}

} // namespace

int main() {
  const auto cases = fixtures();
  const auto implementations = supported_implementations();
  size_t total = 0;
  size_t failures = 0;

  if (cases.size() != kCasesPerKernel) {
    std::fprintf(stderr, "expected %zu fixtures, found %zu\n",
                 kCasesPerKernel, cases.size());
    ++failures;
  }
  if (implementations.empty()) {
    std::fprintf(stderr, "no supported simdjson implementation found\n");
    ++failures;
  }

  for (const implementation *impl : implementations) {
    for (const auto &[name, json] : cases) {
      const CaseResult result = run_case(impl, name, json);
      ++total;
      if (!result.passed()) {
        ++failures;
      }
      std::printf(
          "RESULT kernel=%s fixture=%s separate_error=%d separate_len=%zu "
          "aliased_error=%d aliased_len=%zu aliased_matches_reference=%d "
          "passed=%d\n",
          result.kernel.c_str(), result.fixture.c_str(),
          static_cast<int>(result.separate_error), result.separate_len,
          static_cast<int>(result.aliased_error), result.aliased_len,
          result.aliased_matches_reference ? 1 : 0,
          result.passed() ? 1 : 0);
    }
  }

  const std::string kernels = kernel_list(implementations);
  std::printf(
      "SUMMARY kernels=%s cases_per_kernel=%zu total=%zu failures=%zu\n",
      kernels.c_str(), kCasesPerKernel, total, failures);
  return failures == 0 ? 0 : 1;
}
