// Spike 004: Minify buffer safety.
//
// Question: does simdjson::minify() (and each locally-buildable per-kernel
// implementation) write past dst[0, len(src)) when dst is allocated with
// EXACTLY len(src) bytes (no SIMDJSON_PADDING=64 slack), and does it produce
// correct output when dst aliases src (in-place minification)?
//
// This does not touch production source, the vendored gitlink, or any pinned
// version. It links directly against the untouched vendored singleheader
// amalgamation at third_party/simdjson/singleheader/{simdjson.h,simdjson.cpp}.
//
// Build with -fsanitize=address,undefined so any write outside an exact-size
// heap allocation is caught deterministically, not just "happened to not
// crash this run."

#include "simdjson.h"

#include <cstdio>
#include <cstdlib>
#include <cstring>
#include <string>
#include <vector>

using namespace simdjson;

namespace {

struct CaseResult {
  std::string kernel;
  std::string fixture;
  bool separate_ok = false;   // no ASan trap, dst sized exactly len(src)
  bool aliased_ok = false;    // no ASan trap, dst == src, sized exactly len(src)
  bool aliased_matches_reference = false; // in-place output byte-identical to separate-buffer output
  error_code separate_error = UNINITIALIZED;
  error_code aliased_error = UNINITIALIZED;
  size_t separate_len = 0;
  size_t aliased_len = 0;
};

std::vector<CaseResult> g_results;

// Exact-size heap buffer via `new`, so ASan's redzone sits immediately after
// byte len-1. Any write at or past `len` is a heap-buffer-overflow.
struct ExactBuffer {
  explicit ExactBuffer(size_t len) : data(new uint8_t[len == 0 ? 1 : len]), len(len) {}
  ~ExactBuffer() { delete[] data; }
  uint8_t *data;
  size_t len;
};

CaseResult run_case(const implementation *impl, const std::string &fixture_name,
                     const std::string &json) {
  CaseResult r;
  r.kernel = impl->name();
  r.fixture = fixture_name;

  const auto *src_bytes = reinterpret_cast<const uint8_t *>(json.data());
  const size_t len = json.size();

  // Case A: separate dst buffer, sized exactly len bytes.
  {
    ExactBuffer dst(len);
    size_t dst_len = 0;
    error_code err = impl->minify(src_bytes, len, dst.data, dst_len);
    r.separate_error = err;
    r.separate_len = dst_len;
    r.separate_ok = true; // if we get here, ASan did not trap
    if (!err) {
      // Store the reference output for the aliasing comparison below.
      r.separate_ok = true;
    }
  }

  // Case B: dst == src, a single buffer sized exactly len bytes, containing
  // the input. This is the actual Phase 12 MinifyInto(dst, src) shape when
  // the caller passes the same slice for both arguments.
  {
    ExactBuffer buf(len);
    std::memcpy(buf.data, json.data(), len);
    size_t dst_len = 0;
    error_code err = impl->minify(buf.data, len, buf.data, dst_len);
    r.aliased_error = err;
    r.aliased_len = dst_len;
    r.aliased_ok = true; // if we get here, ASan did not trap

    if (!err) {
      // Re-run the non-aliased case to get a trustworthy reference, then
      // compare byte-for-byte against what the aliased run produced.
      ExactBuffer ref(len);
      size_t ref_len = 0;
      error_code ref_err = impl->minify(src_bytes, len, ref.data, ref_len);
      if (!ref_err && ref_len == dst_len &&
          std::memcmp(ref.data, buf.data, dst_len) == 0) {
        r.aliased_matches_reference = true;
      }
    }
  }

  return r;
}

std::vector<std::pair<std::string, std::string>> fixtures() {
  std::string nested = R"({"a":[1,2,{"b":"c"},[true,false,null]],"d":{"e":{"f":1.5e10}}})";
  std::string escaped = R"({"s":"line1\nline2\ttab\\backslash\"quote","u":"éècafé"})";
  std::string whitespace_heavy =
      "{\n  \"a\"   :    1,\n  \"b\"  :   [   1,   2,   3   ]  \n}\n\n\n";
  std::string wide_unicode = R"({"emoji":"😀👍","cjk":"你好"})";

  // Sizes chosen to straddle SIMDJSON_PADDING (64) and typical SIMD widths
  // (16/32/64) boundaries, since a lookahead/overwrite bug would most likely
  // surface right at a chunk boundary.
  auto pad_with_ws = [](std::string base, size_t target_len) {
    while (base.size() < target_len) {
      // Insert whitespace before the final closing brace so it stays valid
      // JSON and still minifies down.
      base.insert(base.size() - 1, " ");
    }
    return base;
  };

  std::vector<std::pair<std::string, std::string>> out;
  out.emplace_back("nested", nested);
  out.emplace_back("escaped_strings", escaped);
  out.emplace_back("whitespace_heavy", whitespace_heavy);
  out.emplace_back("wide_unicode", wide_unicode);
  out.emplace_back("boundary_63", pad_with_ws(R"({"a":1})", 63));
  out.emplace_back("boundary_64", pad_with_ws(R"({"a":1})", 64));
  out.emplace_back("boundary_65", pad_with_ws(R"({"a":1})", 65));
  out.emplace_back("boundary_128", pad_with_ws(R"({"a":1})", 128));
  out.emplace_back("empty_object", "{}");
  out.emplace_back("single_char", "1");

  // Large, heavily-whitespace-padded array: this is the adversarial case for
  // in-place aliasing. High compression ratio means the write pointer lags
  // far behind the read pointer for long stretches, maximizing the window
  // in which a SIMD lookahead read could observe bytes the write side has
  // already clobbered, if such a bug existed.
  {
    std::string big = "[";
    for (int i = 0; i < 2000; ++i) {
      if (i) big += ",     ";
      big += "   " + std::to_string(i) + "   ";
    }
    big += "]";
    out.emplace_back("large_whitespace_array_~18kb", big);
  }

  // Large array of nested objects with escaped strings, no padding trick,
  // just raw size to cross many 64-byte SIMD chunks with real content.
  {
    std::string big = "[";
    for (int i = 0; i < 500; ++i) {
      if (i) big += ",";
      big += R"({"id":)" + std::to_string(i) + R"(,"name":"item\t)" + std::to_string(i) +
             R"(\n","tags":["a","b","c"]})";
    }
    big += "]";
    out.emplace_back("large_nested_objects_~25kb", big);
  }

  return out;
}

} // namespace

int main() {
  const auto &impls = get_available_implementations();

  bool any_failure = false;

  for (const implementation *impl : impls) {
    if (!impl->supported_by_runtime_system()) {
      std::fprintf(stderr, "skip:%s:not-supported-on-this-host\n", impl->name().c_str());
      continue;
    }
    for (const auto &[name, json] : fixtures()) {
      CaseResult r = run_case(impl, name, json);
      g_results.push_back(r);

      if (!r.separate_ok || !r.aliased_ok) {
        any_failure = true;
      }
      if (!r.aliased_error && !r.aliased_matches_reference) {
        any_failure = true;
      }

      std::printf(
          "RESULT kernel=%s fixture=%s separate_ok=%d separate_err=%d "
          "separate_len=%zu aliased_ok=%d aliased_err=%d aliased_len=%zu "
          "aliased_matches_reference=%d\n",
          r.kernel.c_str(), r.fixture.c_str(), r.separate_ok ? 1 : 0,
          static_cast<int>(r.separate_error), r.separate_len,
          r.aliased_ok ? 1 : 0, static_cast<int>(r.aliased_error),
          r.aliased_len, r.aliased_matches_reference ? 1 : 0);
    }
  }

  std::printf("SUMMARY total=%zu any_failure=%d\n", g_results.size(), any_failure ? 1 : 0);
  return any_failure ? 1 : 0;
}
