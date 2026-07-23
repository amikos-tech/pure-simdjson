# Deferred Items

- Repository-wide `cargo fmt --check` reports pre-existing formatting drift in
  `src/runtime/registry.rs`, `tests/rust_shim_minimal.rs`, and
  `tests/rust_shim_native_alloc.rs`. Plan 11-05 formatted its new Rust code and
  left unrelated lines unchanged.
