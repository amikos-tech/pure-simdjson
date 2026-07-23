# Deferred Items

- Repository-wide `cargo fmt --check` reports pre-existing formatting drift in
  `src/runtime/registry.rs`, `tests/rust_shim_minimal.rs`, and
  `tests/rust_shim_native_alloc.rs`. Plan 11-05 formatted its new Rust code and
  left unrelated lines unchanged.
- `make verify-contract` prints the expected generated-header diff but returns
  success because the multi-command recipe does not stop after `diff` fails.
  Plan 11-07 left the pre-existing Makefile behavior unchanged; harden the gate
  after Plan 11-09 synchronizes the generated ABI contract.
