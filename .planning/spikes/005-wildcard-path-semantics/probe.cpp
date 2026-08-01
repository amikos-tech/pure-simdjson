// Spike 005: Wildcard path semantics truth table.
//
// Pins the exact (error_code, result_count, ordering) of
// simdjson::dom::element::at_path_with_wildcard against the vendored
// simdjson v4.6.4, and diffs it against element::at_path on identical
// (document, path) pairs.
//
// Phase 12 plans 12-03 and 12-06 delegate AtPathAll directly to
// at_path_with_wildcard and claim decision D-02's error surface
// (ErrInvalidPath / ErrElementNotFound / ErrIndexOutOfRange / ErrWrongType).
// This probe establishes what that delegation actually produces.
//
// Output is deterministic and machine-readable; verify.sh diffs it against
// expected.txt so any upstream semantics drift fails the build.

#include <cstdio>
#include <string>
#include <utility>
#include <vector>

#include "simdjson.h"

using namespace simdjson;

namespace {

struct Fixture {
  const char *name;
  const char *json;
};

// Documents chosen so each wildcard branch can independently hit, miss, or
// be a non-container -- that is where the two error regimes diverge.
const std::vector<Fixture> &fixtures() {
  static const std::vector<Fixture> f = {
      {"obj_nested", R"({"a":{"b":1}})"},
      {"obj_miss_key", R"({"a":{"c":1}})"},
      {"obj_arr", R"({"a":[10,20]})"},
      {"scalar_root", R"(42)"},
      {"string_root", R"("hi")"},
      {"flat_obj", R"({"p":1,"q":2,"r":3})"},
      {"arr_root", R"([1,2,3])"},
      {"wild_all_hit", R"({"a":{"x":{"b":1},"y":{"b":2}}})"},
      {"wild_partial", R"({"a":{"x":{"b":1},"y":{"c":2}}})"},
      {"wild_hetero", R"({"a":{"x":{"b":1},"y":5}})"},
      {"arr_of_obj", R"({"a":[{"b":1},{"b":2}]})"},
      {"arr_of_obj_partial", R"({"a":[{"b":1},{"c":2}]})"},
      {"empty_obj", R"({})"},
      {"empty_arr", R"([])"},
      {"deep", R"({"a":{"x":{"y":{"b":1}},"z":{"y":{"b":2}}}})"},
  };
  return f;
}

struct Case {
  const char *name;
  const char *doc;
  const char *path;
};

// Ordered by the question each answers: no-wildcard delegation, scalar
// receivers, exact-suffix wildcards, mid-path wildcards with partial
// matches, then grammar/malformed input.
const std::vector<Case> &cases() {
  static const std::vector<Case> c = {
      // -- no wildcard present: does the path hard-error or return empty?
      {"nowild_hit_obj", "obj_nested", ".a.b"},
      {"nowild_miss_key", "obj_miss_key", ".a.b"},
      {"nowild_hit_arr", "obj_arr", ".a[1]"},
      {"nowild_index_oob", "obj_arr", ".a[5]"},
      {"nowild_miss_prefix", "obj_nested", ".z.b"},
      {"nowild_type_mismatch", "obj_arr", ".a.b"},

      // -- scalar / string receiver with and without a wildcard
      {"scalar_recv_path", "scalar_root", ".a"},
      {"scalar_recv_wild", "scalar_root", ".*"},
      {"string_recv_wild", "string_root", ".*"},

      // -- exact-suffix wildcard shortcut
      {"wild_obj_all", "flat_obj", ".*"},
      {"wild_arr_all", "arr_root", "[*]"},
      {"wild_empty_obj", "empty_obj", ".*"},
      {"wild_empty_arr", "empty_arr", "[*]"},

      // -- mid-path wildcard: all branches hit, some miss, some non-container
      {"wild_mid_all_hit", "wild_all_hit", ".a.*.b"},
      {"wild_mid_partial", "wild_partial", ".a.*.b"},
      {"wild_mid_hetero", "wild_hetero", ".a.*.b"},
      {"wild_arr_of_obj", "arr_of_obj", ".a[*].b"},
      {"wild_arr_of_obj_partial", "arr_of_obj_partial", ".a[*].b"},
      {"wild_missing_prefix", "obj_nested", ".z.*"},
      {"wild_deep_two_levels", "deep", ".a.*.y.b"},
      {"wild_double_star", "wild_all_hit", ".a.*.*"},

      // -- grammar and malformed input
      {"grammar_dollar_prefix", "obj_nested", "$.a.b"},
      {"grammar_dollar_wild", "flat_obj", "$.*"},
      {"malformed_no_leading_dot", "obj_nested", "a.b"},
      {"malformed_bare_star", "flat_obj", "*"},
      {"malformed_unterminated", "obj_nested", ".a[0"},
      {"malformed_empty_path", "obj_nested", ""},

      // -- regime boundary: the same failure with and without a '*' later in
      // the path. Upstream selects its error regime by substring-testing for
      // '*', not by what the document contains, so these pairs isolate it.
      {"boundary_miss_prefix_then_wild", "obj_nested", ".z.*.b"},
      {"boundary_obj_indexed_wild", "obj_nested", ".a[*]"},
      {"boundary_scalar_branch_wild", "obj_arr", ".a[0].*"},
      {"boundary_star_on_array", "obj_arr", ".a.*"},
      {"boundary_star_on_root_obj", "flat_obj", "[*]"},
      {"boundary_index_on_root_obj", "flat_obj", "[0]"},
      {"boundary_trailing_dot", "obj_nested", ".a."},
      {"boundary_trailing_dot_wild", "wild_all_hit", ".a.*."},
  };
  return c;
}

const char *find_json(const char *doc_name) {
  for (const auto &f : fixtures()) {
    if (std::string(f.name) == doc_name) { return f.json; }
  }
  return nullptr;
}

// Stable "<code>:<name>" so a drift in either the numeric value or the
// meaning shows up in the golden diff.
std::string err_tag(error_code e) {
  return std::to_string(static_cast<int>(e)) + ":" + error_message(e);
}

std::string join_values(const std::vector<dom::element> &els) {
  std::string out = "[";
  for (size_t i = 0; i < els.size(); ++i) {
    if (i) { out += ","; }
    out += to_string(els[i]);
  }
  out += "]";
  return out;
}

} // namespace

int main() {
  const implementation *active = get_active_implementation();
  std::printf("IMPL name=%s\n", active ? active->name().c_str() : "unknown");

  dom::parser parser;
  size_t total = 0;
  bool any_parse_failure = false;

  for (const auto &c : cases()) {
    const char *json = find_json(c.doc);
    if (json == nullptr) {
      std::printf("CASE name=%s ERROR=fixture-not-found\n", c.name);
      any_parse_failure = true;
      continue;
    }

    dom::element root;
    auto parse_err = parser.parse(padded_string(std::string(json))).get(root);
    if (parse_err) {
      std::printf("CASE name=%s ERROR=parse:%s\n", c.name, error_message(parse_err));
      any_parse_failure = true;
      continue;
    }

    // AtPathAll delegation target.
    std::vector<dom::element> matches;
    auto all_err = root.at_path_with_wildcard(c.path).get(matches);

    // AtPath delegation target, same inputs, for divergence comparison.
    dom::element one;
    auto one_err = root.at_path(c.path).get(one);

    std::printf(
        "CASE name=%-24s doc=%-18s path=%-10s | all_err=%-28s all_n=%zu all_vals=%s | one_err=%-28s one_val=%s\n",
        c.name, c.doc, (std::string("'") + c.path + "'").c_str(),
        err_tag(all_err).c_str(), all_err ? size_t(0) : matches.size(),
        all_err ? "-" : join_values(matches).c_str(),
        err_tag(one_err).c_str(), one_err ? "-" : to_string(one).c_str());

    ++total;
  }

  std::printf("SUMMARY total=%zu parse_failures=%d\n", total, any_parse_failure ? 1 : 0);
  return any_parse_failure ? 1 : 0;
}
