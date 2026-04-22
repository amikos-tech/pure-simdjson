# JSONTestSuite Snapshot Provenance

This directory vendors the correctness-oracle snapshot used for `TestJSONTestSuiteOracle`.

- Upstream snapshot: `nst/JSONTestSuite@1ef36fa01286573e846ac449e8683f8833c5b26a`
- Archive URL: `https://github.com/nst/JSONTestSuite/archive/1ef36fa01286573e846ac449e8683f8833c5b26a.tar.gz`
- Extracted subset: `test_parsing/` only, copied into `testdata/jsontestsuite/cases/`
- Expected snapshot size: `318` files under `testdata/jsontestsuite/cases/`
- Extraction command: `tar -xzf JSONTestSuite-1ef36fa01286573e846ac449e8683f8833c5b26a.tar.gz && cp JSONTestSuite-1ef36fa01286573e846ac449e8683f8833c5b26a/test_parsing/*.json testdata/jsontestsuite/cases/`

`expectations.tsv` is generated deterministically at vendoring time and becomes the only runtime source of truth:

- `y_*.json` rows are recorded as `accept`
- `n_*.json` rows are recorded as `reject`
- `i_*.json` rows are classified once with the vendored `simdjson v4.6.1` parser and recorded as the observed `accept` or `reject`

The runtime oracle must not shell out, inspect git state, infer expectations from filename prefixes, or fetch new upstream data. It must validate the committed manifest against the committed `cases/` snapshot.
