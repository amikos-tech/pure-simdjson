# Benchmark Corpus Provenance

Phase 7 benchmark execution must read only `testdata/bench/*.json`. It must never read `third_party/simdjson/*` or fetch benchmark inputs from the network at runtime.

All five corpus files in this directory are vendored from `simdjson/simdjson@19c3b1315a2a6b8ab0a6b7335bb97269cbd0a448`. This commit is intentionally pre-`714f0ba2226cf996fdd575f8c0e7a3c092c194bd`, which deleted most of the historical benchmark JSON files from the upstream repository.

| filename | source | upstream_ref | sha256 | notes |
| --- | --- | --- | --- | --- |
| `twitter.json` | `https://raw.githubusercontent.com/simdjson/simdjson/19c3b1315a2a6b8ab0a6b7335bb97269cbd0a448/jsonexamples/twitter.json` | `19c3b1315a2a6b8ab0a6b7335bb97269cbd0a448` | `30721e496a8d73cfc50658923c34eb2c0fbe15ee6835005e43ee624d8dedf200` | Historical simdjson benchmark fixture. |
| `citm_catalog.json` | `https://raw.githubusercontent.com/simdjson/simdjson/19c3b1315a2a6b8ab0a6b7335bb97269cbd0a448/jsonexamples/citm_catalog.json` | `19c3b1315a2a6b8ab0a6b7335bb97269cbd0a448` | `a73e7a883f6ea8de113dff59702975e60119b4b58d451d518a929f31c92e2059` | Historical simdjson benchmark fixture. |
| `canada.json` | `https://raw.githubusercontent.com/simdjson/simdjson/19c3b1315a2a6b8ab0a6b7335bb97269cbd0a448/jsonexamples/canada.json` | `19c3b1315a2a6b8ab0a6b7335bb97269cbd0a448` | `f83b3b354030d5dd58740c68ac4fecef64cb730a0d12a90362a7f23077f50d78` | Historical simdjson benchmark fixture. |
| `mesh.json` | `https://raw.githubusercontent.com/simdjson/simdjson/19c3b1315a2a6b8ab0a6b7335bb97269cbd0a448/jsonexamples/mesh.json` | `19c3b1315a2a6b8ab0a6b7335bb97269cbd0a448` | `45bc8bf429340a874a7af8ea7056d60497402f80f55dba1e6ecc4ca8f1e46aff` | Historical simdjson benchmark fixture. |
| `numbers.json` | `https://raw.githubusercontent.com/simdjson/simdjson/19c3b1315a2a6b8ab0a6b7335bb97269cbd0a448/jsonexamples/numbers.json` | `19c3b1315a2a6b8ab0a6b7335bb97269cbd0a448` | `82e9ddfe00963110ed8a0704e7df4d1ad1af9c0f336d1b24431ebc63cf430a2b` | Historical simdjson benchmark fixture. |
