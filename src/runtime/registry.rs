use std::{
    ptr,
    sync::{Mutex, MutexGuard, OnceLock},
};

use crate::{
    pure_simdjson_doc_t, pure_simdjson_error_code_t, pure_simdjson_parser_t,
    pure_simdjson_value_view_t,
};

use super::ROOT_VIEW_TAG;

/// Public parser/doc handles share one packed `u64` wire format, so the registry must enforce
/// these invariants:
/// - slot `0` is never returned;
/// - parser/doc generations never collide numerically for the same slot;
/// - parser busy state is cleared only by the matching document free path;
/// - root views are tagged with [`ROOT_VIEW_TAG`] and reject non-zero reserved bits.
const MAX_SLOT_COUNT: usize = u32::MAX as usize - 1;
const PARSER_GENERATION_START: u32 = 1;
const DOC_GENERATION_START: u32 = 2;

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(crate) enum ParserState {
    Idle,
    Busy { doc_slot: u32, doc_generation: u32 },
}

#[derive(Clone, Debug)]
struct ParserEntry {
    generation: u32,
    native_ptr: usize,
    state: ParserState,
}

#[derive(Clone, Debug)]
struct DocEntry {
    generation: u32,
    native_ptr: usize,
    root_ptr: usize,
    owner_slot: u32,
    owner_generation: u32,
    input_storage: Vec<u8>,
}

#[derive(Clone, Debug)]
enum Slot<T> {
    Vacant { generation: u32 },
    Occupied(T),
}

#[derive(Default)]
struct Registry {
    parsers: Vec<Slot<ParserEntry>>,
    docs: Vec<Slot<DocEntry>>,
}

static REGISTRY: OnceLock<Mutex<Registry>> = OnceLock::new();

#[inline]
fn err_invalid_handle() -> pure_simdjson_error_code_t {
    pure_simdjson_error_code_t::PURE_SIMDJSON_ERR_INVALID_HANDLE
}

#[inline]
fn err_invalid_argument() -> pure_simdjson_error_code_t {
    pure_simdjson_error_code_t::PURE_SIMDJSON_ERR_INVALID_ARGUMENT
}

#[inline]
fn err_internal() -> pure_simdjson_error_code_t {
    pure_simdjson_error_code_t::PURE_SIMDJSON_ERR_INTERNAL
}

#[inline]
fn err_ok() -> pure_simdjson_error_code_t {
    pure_simdjson_error_code_t::PURE_SIMDJSON_OK
}

#[inline]
fn err_parser_busy() -> pure_simdjson_error_code_t {
    pure_simdjson_error_code_t::PURE_SIMDJSON_ERR_PARSER_BUSY
}

#[inline]
fn err_precision_loss() -> pure_simdjson_error_code_t {
    pure_simdjson_error_code_t::PURE_SIMDJSON_ERR_PRECISION_LOSS
}

/// Coarse value kind sentinel for views whose backing element cannot be classified
/// (e.g. BIGINT elements, where the canonical error surfaces at `pure_simdjson_element_type`).
const KIND_HINT_INVALID: u32 = 0;

#[inline]
fn registry() -> &'static Mutex<Registry> {
    REGISTRY.get_or_init(|| Mutex::new(Registry::default()))
}

#[inline]
fn registry_guard() -> MutexGuard<'static, Registry> {
    registry()
        .lock()
        .unwrap_or_else(|poisoned| poisoned.into_inner())
}

#[inline]
fn next_generation(current: u32, restart: u32) -> u32 {
    let next = current.wrapping_add(2);
    if next == 0 {
        restart
    } else {
        next
    }
}

#[inline]
fn next_parser_generation(current: u32) -> u32 {
    next_generation(current, PARSER_GENERATION_START)
}

#[inline]
fn next_doc_generation(current: u32) -> u32 {
    next_generation(current, DOC_GENERATION_START)
}

#[inline]
fn pack_handle(slot: u32, generation: u32) -> u64 {
    u64::from(slot) | (u64::from(generation) << 32)
}

#[inline]
fn unpack_handle(handle: u64) -> Result<(usize, u32, u32), pure_simdjson_error_code_t> {
    if handle == 0 {
        return Err(err_invalid_handle());
    }
    let slot = handle as u32;
    if slot == 0 {
        return Err(err_invalid_handle());
    }
    Ok(((slot - 1) as usize, slot, (handle >> 32) as u32))
}

impl Registry {
    fn alloc_parser(
        &mut self,
        native_ptr: usize,
    ) -> Result<pure_simdjson_parser_t, pure_simdjson_error_code_t> {
        for (index, slot) in self.parsers.iter_mut().enumerate() {
            if let Slot::Vacant { generation } = slot {
                let generation = *generation;
                let slot_number = (index + 1) as u32;
                *slot = Slot::Occupied(ParserEntry {
                    generation,
                    native_ptr,
                    state: ParserState::Idle,
                });
                return Ok(pack_handle(slot_number, generation));
            }
        }

        if self.parsers.len() >= MAX_SLOT_COUNT {
            return Err(err_internal());
        }

        let generation = PARSER_GENERATION_START;
        self.parsers.push(Slot::Occupied(ParserEntry {
            generation,
            native_ptr,
            state: ParserState::Idle,
        }));
        Ok(pack_handle(self.parsers.len() as u32, generation))
    }

    fn alloc_doc(
        &mut self,
        native_ptr: usize,
        root_ptr: usize,
        owner_slot: u32,
        owner_generation: u32,
        input: Vec<u8>,
    ) -> Result<pure_simdjson_doc_t, pure_simdjson_error_code_t> {
        for (index, slot) in self.docs.iter_mut().enumerate() {
            if let Slot::Vacant { generation } = slot {
                let generation = *generation;
                let slot_number = (index + 1) as u32;
                *slot = Slot::Occupied(DocEntry {
                    generation,
                    native_ptr,
                    root_ptr,
                    owner_slot,
                    owner_generation,
                    input_storage: input,
                });
                return Ok(pack_handle(slot_number, generation));
            }
        }

        if self.docs.len() >= MAX_SLOT_COUNT {
            return Err(err_internal());
        }

        let generation = DOC_GENERATION_START;
        self.docs.push(Slot::Occupied(DocEntry {
            generation,
            native_ptr,
            root_ptr,
            owner_slot,
            owner_generation,
            input_storage: input,
        }));
        Ok(pack_handle(self.docs.len() as u32, generation))
    }

    fn parser_entry(&self, handle: pure_simdjson_parser_t) -> Result<&ParserEntry, pure_simdjson_error_code_t> {
        let (index, _, generation) = unpack_handle(handle)?;
        match self.parsers.get(index) {
            Some(Slot::Occupied(entry)) if entry.generation == generation => Ok(entry),
            _ => Err(err_invalid_handle()),
        }
    }

    fn doc_entry(&self, handle: pure_simdjson_doc_t) -> Result<&DocEntry, pure_simdjson_error_code_t> {
        let (index, _, generation) = unpack_handle(handle)?;
        match self.docs.get(index) {
            Some(Slot::Occupied(entry)) if entry.generation == generation => Ok(entry),
            _ => Err(err_invalid_handle()),
        }
    }
}

pub(crate) fn parser_new() -> Result<pure_simdjson_parser_t, pure_simdjson_error_code_t> {
    let native_ptr = super::native_parser_new()?;
    let mut registry = registry_guard();
    match registry.alloc_parser(native_ptr) {
        Ok(handle) => Ok(handle),
        Err(rc) => {
            let free_rc = super::native_parser_free(native_ptr);
            if free_rc != err_ok() {
                eprintln!(
                    "pure_simdjson cleanup failure in parser_new/alloc_parser: {:?}",
                    free_rc
                );
            }
            Err(rc)
        }
    }
}

pub(crate) fn parser_free(handle: pure_simdjson_parser_t) -> pure_simdjson_error_code_t {
    let mut registry = registry_guard();
    let (index, _, generation) = match unpack_handle(handle) {
        Ok(parts) => parts,
        Err(rc) => return rc,
    };

    let native_ptr = match registry.parsers.get(index) {
        Some(Slot::Occupied(entry)) if entry.generation == generation => {
            if !matches!(entry.state, ParserState::Idle) {
                return err_parser_busy();
            }
            entry.native_ptr
        }
        _ => return err_invalid_handle(),
    };

    let rc = super::native_parser_free(native_ptr);
    if rc != err_ok() {
        return rc;
    }

    registry.parsers[index] = Slot::Vacant {
        generation: next_parser_generation(generation),
    };
    err_ok()
}

pub(crate) fn parser_parse(
    handle: pure_simdjson_parser_t,
    input: &[u8],
) -> Result<pure_simdjson_doc_t, pure_simdjson_error_code_t> {
    // The registry mutex is held across `native_parser_parse` deliberately: the parser slot's
    // Idle->Busy transition must be atomic with the doc allocation that owns the busy state, and
    // simdjson parsers are thread-compatible (one parser per thread). Multi-parser throughput is
    // not a Phase 02 goal; revisit if cross-parser contention becomes a measured bottleneck.
    let mut registry = registry_guard();
    let (index, slot, generation) = unpack_handle(handle)?;

    let native_ptr = match registry.parsers.get(index) {
        Some(Slot::Occupied(entry)) if entry.generation == generation => {
            if !matches!(entry.state, ParserState::Idle) {
                return Err(err_parser_busy());
            }
            entry.native_ptr
        }
        _ => return Err(err_invalid_handle()),
    };

    let padding = super::padding_bytes()?;
    let total_len = input
        .len()
        .checked_add(padding)
        .ok_or_else(err_invalid_argument)?;
    let mut owned_input = vec![0u8; total_len]; // padding bytes stay zero-initialized
    owned_input[..input.len()].copy_from_slice(input);

    let parsed = super::native_parser_parse(native_ptr, &owned_input[..], input.len())?;
    let doc_handle = match registry.alloc_doc(
        parsed.doc_ptr,
        parsed.root_ptr,
        slot,
        generation,
        owned_input,
    ) {
        Ok(handle) => handle,
        Err(rc) => {
            let free_rc = super::native_doc_free(parsed.doc_ptr);
            if free_rc != err_ok() {
                eprintln!(
                    "pure_simdjson cleanup failure in parser_parse/alloc_doc: {:?}",
                    free_rc
                );
            }
            return Err(rc);
        }
    };

    let (_, doc_slot, doc_generation) = unpack_handle(doc_handle)?;
    if let Some(Slot::Occupied(entry)) = registry.parsers.get_mut(index) {
        entry.state = ParserState::Busy {
            doc_slot,
            doc_generation,
        };
        Ok(doc_handle)
    } else {
        Err(err_internal())
    }
}

pub(crate) fn parser_last_error_len(
    handle: pure_simdjson_parser_t,
) -> Result<usize, pure_simdjson_error_code_t> {
    let registry = registry_guard();
    let entry = registry.parser_entry(handle)?;
    super::native_parser_get_last_error_len(entry.native_ptr)
}

pub(crate) fn parser_copy_last_error(
    handle: pure_simdjson_parser_t,
    dst: *mut u8,
    dst_cap: usize,
    out_written: *mut usize,
) -> pure_simdjson_error_code_t {
    let registry = registry_guard();
    let entry = match registry.parser_entry(handle) {
        Ok(entry) => entry,
        Err(rc) => return rc,
    };
    super::native_parser_copy_last_error(entry.native_ptr, dst, dst_cap, out_written)
}

pub(crate) fn parser_last_error_offset(
    handle: pure_simdjson_parser_t,
) -> Result<u64, pure_simdjson_error_code_t> {
    let registry = registry_guard();
    let entry = registry.parser_entry(handle)?;
    super::native_parser_get_last_error_offset(entry.native_ptr)
}

pub(crate) fn doc_free(handle: pure_simdjson_doc_t) -> pure_simdjson_error_code_t {
    let mut registry = registry_guard();
    let (doc_index, _, doc_generation) = match unpack_handle(handle) {
        Ok(parts) => parts,
        Err(rc) => return rc,
    };

    let (native_ptr, owner_slot, owner_generation) = match registry.docs.get(doc_index) {
        Some(Slot::Occupied(entry)) if entry.generation == doc_generation => (
            entry.native_ptr,
            entry.owner_slot,
            entry.owner_generation,
        ),
        _ => return err_invalid_handle(),
    };

    let parser_index = owner_slot.checked_sub(1).map(|value| value as usize);
    let parser_index = match parser_index {
        Some(index) => index,
        None => return err_invalid_handle(),
    };

    match registry.parsers.get(parser_index) {
        Some(Slot::Occupied(entry)) if entry.generation == owner_generation => match entry.state {
            ParserState::Busy {
                doc_slot,
                doc_generation: busy_doc_generation,
            } if doc_slot == (doc_index + 1) as u32 && busy_doc_generation == doc_generation => {}
            ParserState::Busy { .. } => return err_invalid_handle(),
            ParserState::Idle => return err_invalid_handle(),
        },
        _ => return err_invalid_handle(),
    }

    let rc = super::native_doc_free(native_ptr);
    if rc != err_ok() {
        return rc;
    }

    if let Some(Slot::Occupied(entry)) = registry.parsers.get_mut(parser_index) {
        entry.state = ParserState::Idle;
    }

    registry.docs[doc_index] = Slot::Vacant {
        generation: next_doc_generation(doc_generation),
    };
    err_ok()
}

pub(crate) fn doc_root(
    handle: pure_simdjson_doc_t,
) -> Result<pure_simdjson_value_view_t, pure_simdjson_error_code_t> {
    let registry = registry_guard();
    let entry = registry.doc_entry(handle)?;
    // BIGINT roots are unreachable today (bridge does not enable bigint storage), but the bridge's
    // `psimdjson_element_type` would surface PRECISION_LOSS for them. Per the header contract that
    // error must surface at `pure_simdjson_element_type`, not at `pure_simdjson_doc_root`, so we
    // hand back a view with an invalid kind hint and let the canonical error fire downstream.
    let kind_hint = match super::native_element_type(entry.root_ptr) {
        Ok(kind) => kind,
        Err(rc) if rc == err_precision_loss() => KIND_HINT_INVALID,
        Err(rc) => return Err(rc),
    };
    Ok(pure_simdjson_value_view_t {
        doc: handle,
        state0: entry.root_ptr as u64,
        state1: ROOT_VIEW_TAG,
        kind_hint,
        reserved: 0,
    })
}

fn with_validated_view<T, F>(
    view: *const pure_simdjson_value_view_t,
    action: F,
) -> Result<T, pure_simdjson_error_code_t>
where
    F: FnOnce(usize) -> Result<T, pure_simdjson_error_code_t>,
{
    if view.is_null() {
        return Err(err_invalid_argument());
    }

    let view = unsafe { ptr::read_unaligned(view) };
    if view.state1 != ROOT_VIEW_TAG || view.state0 == 0 || view.reserved != 0 {
        return Err(err_invalid_handle());
    }

    let registry = registry_guard();
    let entry = registry.doc_entry(view.doc)?;
    if entry.root_ptr != view.state0 as usize {
        return Err(err_invalid_handle());
    }

    let root_ptr = entry.root_ptr;
    drop(registry);
    action(root_ptr)
}

pub(crate) fn element_type(
    view: *const pure_simdjson_value_view_t,
) -> Result<u32, pure_simdjson_error_code_t> {
    with_validated_view(view, super::native_element_type)
}

pub(crate) fn element_get_int64(
    view: *const pure_simdjson_value_view_t,
) -> Result<i64, pure_simdjson_error_code_t> {
    with_validated_view(view, super::native_element_get_int64)
}
