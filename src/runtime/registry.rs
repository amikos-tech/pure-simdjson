use std::{
    collections::{HashMap, HashSet},
    mem, ptr, slice,
    sync::{Mutex, MutexGuard, OnceLock},
};

use crate::{
    pure_simdjson_array_iter_t, pure_simdjson_doc_t, pure_simdjson_error_code_t,
    pure_simdjson_object_iter_t, pure_simdjson_parser_t, pure_simdjson_value_kind_t,
    pure_simdjson_value_view_t,
};

use super::{ARRAY_ITER_TAG, DESC_VIEW_TAG, OBJECT_ITER_TAG, ROOT_VIEW_TAG};

/// Public parser/doc handles share one packed `u64` wire format, so the registry must enforce
/// these invariants:
/// - slot `0` is never returned;
/// - parser/doc generations never collide numerically for the same slot;
/// - parser busy state is cleared only by the matching document free path;
/// - root views are tagged with [`ROOT_VIEW_TAG`], descendants with [`DESC_VIEW_TAG`], and both
///   reject non-zero reserved bits.
const MAX_SLOT_COUNT: usize = u32::MAX as usize - 1;
const PARSER_GENERATION_START: u32 = 1;
const DOC_GENERATION_START: u32 = 2;
const ROOT_JSON_INDEX: u64 = 1;
const ITER_LEASE_START: u32 = 1;

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
    reusable_input: Vec<u8>,
}

#[derive(Clone, Debug)]
struct DocEntry {
    generation: u32,
    native_ptr: usize,
    root_ptr: usize,
    root_after_index: u64,
    owner_slot: u32,
    owner_generation: u32,
    #[allow(dead_code)]
    // Pinned: simdjson's parsed tape and borrowed string views remain tied to this owned buffer
    // for the lifetime of the document entry, even though Rust never reads the field directly.
    input_storage: Vec<u8>,
    descendant_indices: HashSet<u64>,
    iter_leases: HashMap<u32, IteratorLease>,
    next_iter_lease: u32,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
struct IteratorLease {
    // Current iterator cursor, or the exhausted sentinel once `state0 == state1`.
    state0: u64,
    // Paired end bound carried in the public iterator struct and re-validated on each call.
    state1: u64,
    // Distinguishes array iterator leases from object iterator leases.
    tag: u16,
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
    string_allocations: HashMap<usize, usize>,
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

#[inline]
fn err_wrong_type() -> pure_simdjson_error_code_t {
    pure_simdjson_error_code_t::PURE_SIMDJSON_ERR_WRONG_TYPE
}

/// Coarse value kind sentinel for views whose backing element cannot be classified
/// (e.g. BIGINT elements, where the canonical error surfaces at `pure_simdjson_element_type`).
const KIND_HINT_INVALID: u32 = 0;
const KIND_HINT_STRING: u32 = pure_simdjson_value_kind_t::PURE_SIMDJSON_VALUE_KIND_STRING as u32;
const KIND_HINT_ARRAY: u32 = pure_simdjson_value_kind_t::PURE_SIMDJSON_VALUE_KIND_ARRAY as u32;
const KIND_HINT_OBJECT: u32 = pure_simdjson_value_kind_t::PURE_SIMDJSON_VALUE_KIND_OBJECT as u32;

#[inline]
fn uint64_is_exact_float64(value: u64) -> bool {
    if value == 0 {
        return true;
    }
    let significant = value >> value.trailing_zeros();
    (u64::BITS - significant.leading_zeros()) <= 53
}

#[inline]
fn int64_is_exact_float64(value: i64) -> bool {
    let magnitude = if value < 0 {
        value.wrapping_neg() as u64
    } else {
        value as u64
    };
    uint64_is_exact_float64(magnitude)
}

#[inline]
fn registry() -> &'static Mutex<Registry> {
    REGISTRY.get_or_init(|| Mutex::new(Registry::default()))
}

#[inline]
fn registry_guard() -> MutexGuard<'static, Registry> {
    registry()
        .lock()
        // Intentional poison-heal: every exported FFI entrypoint wraps its body in `catch_unwind`,
        // so reachable callers should never observe a poisoned registry mutex.
        .unwrap_or_else(|poisoned| poisoned.into_inner())
}

#[inline]
fn next_generation(current: u32, restart: u32) -> u32 {
    // Parsers start at 1 (odd) and docs start at 2 (even); wrapping by 2 preserves parity so a
    // parser and a doc sharing the same slot index can never share the same generation number.
    // This is the numeric-collision guarantee referenced in the module-level invariants above.
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
fn next_iter_lease(current: u32) -> u32 {
    let next = current.wrapping_add(1);
    if next == 0 {
        ITER_LEASE_START
    } else {
        next
    }
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
    // Linear scan acceptable at the current ABI v0.1 scope (few parsers, short lifetimes).
    // Switch to a free-list of vacant indices if parser churn grows.
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
                    reusable_input: Vec::new(),
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
            reusable_input: Vec::new(),
        }));
        Ok(pack_handle(self.parsers.len() as u32, generation))
    }

    // Linear scan acceptable at the current ABI v0.1 scope (few docs, short lifetimes).
    // Switch to a free-list of vacant indices if doc churn grows.
    fn alloc_doc(
        &mut self,
        native_ptr: usize,
        root_ptr: usize,
        root_after_index: u64,
        owner_slot: u32,
        owner_generation: u32,
        input: Vec<u8>,
    ) -> Result<pure_simdjson_doc_t, (pure_simdjson_error_code_t, Vec<u8>)> {
        for (index, slot) in self.docs.iter_mut().enumerate() {
            if let Slot::Vacant { generation } = slot {
                let generation = *generation;
                let slot_number = (index + 1) as u32;
                *slot = Slot::Occupied(DocEntry {
                    generation,
                    native_ptr,
                    root_ptr,
                    root_after_index,
                    owner_slot,
                    owner_generation,
                    input_storage: input,
                    descendant_indices: HashSet::new(),
                    iter_leases: HashMap::new(),
                    next_iter_lease: ITER_LEASE_START,
                });
                return Ok(pack_handle(slot_number, generation));
            }
        }

        if self.docs.len() >= MAX_SLOT_COUNT {
            return Err((err_internal(), input));
        }

        let generation = DOC_GENERATION_START;
        self.docs.push(Slot::Occupied(DocEntry {
            generation,
            native_ptr,
            root_ptr,
            root_after_index,
            owner_slot,
            owner_generation,
            input_storage: input,
            descendant_indices: HashSet::new(),
            iter_leases: HashMap::new(),
            next_iter_lease: ITER_LEASE_START,
        }));
        Ok(pack_handle(self.docs.len() as u32, generation))
    }

    fn parser_entry(
        &self,
        handle: pure_simdjson_parser_t,
    ) -> Result<&ParserEntry, pure_simdjson_error_code_t> {
        let (index, _, generation) = unpack_handle(handle)?;
        match self.parsers.get(index) {
            Some(Slot::Occupied(entry)) if entry.generation == generation => Ok(entry),
            _ => Err(err_invalid_handle()),
        }
    }

    fn doc_entry(
        &self,
        handle: pure_simdjson_doc_t,
    ) -> Result<&DocEntry, pure_simdjson_error_code_t> {
        let (index, _, generation) = unpack_handle(handle)?;
        match self.docs.get(index) {
            Some(Slot::Occupied(entry)) if entry.generation == generation => Ok(entry),
            _ => Err(err_invalid_handle()),
        }
    }
}

impl DocEntry {
    fn alloc_iter_lease(
        &mut self,
        tag: u16,
        state0: u64,
        state1: u64,
    ) -> Result<u32, pure_simdjson_error_code_t> {
        if self.iter_leases.len() >= u32::MAX as usize {
            return Err(err_internal());
        }

        let mut lease_id = if self.next_iter_lease == 0 {
            ITER_LEASE_START
        } else {
            self.next_iter_lease
        };
        for _ in 0..u32::MAX {
            if let std::collections::hash_map::Entry::Vacant(slot) =
                self.iter_leases.entry(lease_id)
            {
                slot.insert(IteratorLease {
                    state0,
                    state1,
                    tag,
                });
                self.next_iter_lease = next_iter_lease(lease_id);
                return Ok(lease_id);
            }
            lease_id = next_iter_lease(lease_id);
        }
        Err(err_internal())
    }

    fn validate_iter_lease(
        &self,
        lease_id: u32,
        state0: u64,
        state1: u64,
        tag: u16,
    ) -> Result<(), pure_simdjson_error_code_t> {
        match self.iter_leases.get(&lease_id) {
            Some(lease) if lease.state0 == state0 && lease.state1 == state1 && lease.tag == tag => {
                Ok(())
            }
            _ => Err(err_invalid_handle()),
        }
    }

    fn update_iter_lease(
        &mut self,
        lease_id: u32,
        state0: u64,
        state1: u64,
        tag: u16,
    ) -> Result<(), pure_simdjson_error_code_t> {
        match self.iter_leases.get_mut(&lease_id) {
            Some(lease) if lease.tag == tag => {
                lease.state0 = state0;
                // Thread `state1` back through the registry copy even when it is unchanged so the
                // lease always mirrors the caller-visible iterator header exactly.
                lease.state1 = state1;
                Ok(())
            }
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
    // not an ABI v0.1 throughput goal; revisit if cross-parser contention becomes a measured bottleneck.
    let mut registry = registry_guard();
    let (index, slot, generation) = unpack_handle(handle)?;

    let (native_ptr, mut owned_input) = match registry.parsers.get_mut(index) {
        Some(Slot::Occupied(entry)) if entry.generation == generation => {
            if !matches!(entry.state, ParserState::Idle) {
                return Err(err_parser_busy());
            }
            (entry.native_ptr, mem::take(&mut entry.reusable_input))
        }
        _ => return Err(err_invalid_handle()),
    };

    let padding = super::padding_bytes()?;
    let total_len = input
        .len()
        .checked_add(padding)
        .ok_or_else(err_invalid_argument)?;
    owned_input.resize(total_len, 0);
    owned_input[..input.len()].copy_from_slice(input);
    owned_input[input.len()..].fill(0);

    let parsed = match super::native_parser_parse(native_ptr, &owned_input[..], input.len()) {
        Ok(parsed) => parsed,
        Err(rc) => {
            restore_parser_input_buffer(&mut registry, index, generation, owned_input);
            return Err(rc);
        }
    };
    let root_after_index = match super::native_element_after_index(parsed.doc_ptr, ROOT_JSON_INDEX)
    {
        Ok(root_after_index) => root_after_index,
        Err(rc) => {
            let free_rc = super::native_doc_free(parsed.doc_ptr);
            if free_rc != err_ok() {
                eprintln!(
                    "pure_simdjson cleanup failure in parser_parse/root_after_index: {:?}",
                    free_rc
                );
            }
            restore_parser_input_buffer(&mut registry, index, generation, owned_input);
            return Err(rc);
        }
    };
    let doc_handle = match registry.alloc_doc(
        parsed.doc_ptr,
        parsed.root_ptr,
        root_after_index,
        slot,
        generation,
        owned_input,
    ) {
        Ok(handle) => handle,
        Err((rc, owned_input)) => {
            let free_rc = super::native_doc_free(parsed.doc_ptr);
            if free_rc != err_ok() {
                eprintln!(
                    "pure_simdjson cleanup failure in parser_parse/alloc_doc: {:?}",
                    free_rc
                );
            }
            restore_parser_input_buffer(&mut registry, index, generation, owned_input);
            return Err(rc);
        }
    };

    let (doc_index_usize, doc_slot, doc_generation) = unpack_handle(doc_handle)?;
    match registry.parsers.get_mut(index) {
        Some(Slot::Occupied(entry)) if entry.generation == generation => {
            entry.state = ParserState::Busy {
                doc_slot,
                doc_generation,
            };
            Ok(doc_handle)
        }
        _ => {
            let free_rc = super::native_doc_free(parsed.doc_ptr);
            if free_rc != err_ok() {
                eprintln!(
                    "pure_simdjson cleanup failure in parser_parse/parser_vanished: {:?}",
                    free_rc
                );
            }
            let next_gen = next_doc_generation(doc_generation);
            if let Some(Slot::Occupied(entry)) = registry.docs.get(doc_index_usize) {
                if entry.generation == doc_generation {
                    let _ = mem::replace(
                        &mut registry.docs[doc_index_usize],
                        Slot::Vacant { generation: next_gen },
                    );
                }
            }
            Err(err_internal())
        }
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
        Some(Slot::Occupied(entry)) if entry.generation == doc_generation => {
            (entry.native_ptr, entry.owner_slot, entry.owner_generation)
        }
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

    match registry.docs.get(doc_index) {
        Some(Slot::Occupied(entry)) if entry.generation == doc_generation => {}
        _ => return err_internal(),
    }
    let input_storage = match mem::replace(
        &mut registry.docs[doc_index],
        Slot::Vacant {
            generation: next_doc_generation(doc_generation),
        },
    ) {
        Slot::Occupied(entry) => entry.input_storage,
        _ => return err_internal(),
    };

    match registry.parsers.get_mut(parser_index) {
        Some(Slot::Occupied(entry)) if entry.generation == owner_generation => {
            entry.state = ParserState::Idle;
            entry.reusable_input = input_storage;
            err_ok()
        }
        _ => err_internal(),
    }
}

fn restore_parser_input_buffer(
    registry: &mut Registry,
    index: usize,
    generation: u32,
    input_storage: Vec<u8>,
) {
    if let Some(Slot::Occupied(entry)) = registry.parsers.get_mut(index) {
        if entry.generation == generation && matches!(entry.state, ParserState::Idle) {
            entry.reusable_input = input_storage;
        }
    }
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

#[inline]
fn doc_contains_json_index(entry: &DocEntry, json_index: u64) -> bool {
    json_index >= ROOT_JSON_INDEX && json_index < entry.root_after_index
}

fn validate_descendant(
    view: &pure_simdjson_value_view_t,
    entry: &DocEntry,
) -> Result<u64, pure_simdjson_error_code_t> {
    let json_index = view.state0;
    if json_index == 0
        || !doc_contains_json_index(entry, json_index)
        || !entry.descendant_indices.contains(&json_index)
    {
        return Err(err_invalid_handle());
    }
    Ok(json_index)
}

fn with_resolved_view<T, F>(
    view: *const pure_simdjson_value_view_t,
    action: F,
) -> Result<T, pure_simdjson_error_code_t>
where
    F: FnOnce(&mut DocEntry, u64, pure_simdjson_doc_t) -> Result<T, pure_simdjson_error_code_t>,
{
    if view.is_null() {
        return Err(err_invalid_argument());
    }

    // SAFETY: `view` was checked for null above, and the ABI permits callers to provide an
    // unaligned pointer to a `pure_simdjson_value_view_t`.
    let view = unsafe { ptr::read_unaligned(view) };
    if view.state0 == 0 || view.reserved != 0 {
        return Err(err_invalid_handle());
    }

    let (doc_index, _, doc_generation) = unpack_handle(view.doc)?;
    let mut registry = registry_guard();
    let entry = match registry.docs.get_mut(doc_index) {
        Some(Slot::Occupied(entry)) if entry.generation == doc_generation => entry,
        _ => return Err(err_invalid_handle()),
    };
    let json_index = match view.state1 {
        ROOT_VIEW_TAG => {
            if entry.root_ptr != view.state0 as usize {
                return Err(err_invalid_handle());
            }
            ROOT_JSON_INDEX
        }
        DESC_VIEW_TAG => validate_descendant(&view, entry)?,
        _ => return Err(err_invalid_handle()),
    };
    action(entry, json_index, view.doc)
}

fn encode_descendant_view_locked(
    entry: &mut DocEntry,
    handle: pure_simdjson_doc_t,
    json_index: u64,
) -> Result<pure_simdjson_value_view_t, pure_simdjson_error_code_t> {
    if json_index == 0 || json_index >= entry.root_after_index {
        return Err(err_invalid_handle());
    }
    let kind_hint = match super::native_element_type_at(entry.native_ptr, json_index) {
        Ok(kind) => kind,
        Err(rc) if rc == err_precision_loss() => KIND_HINT_INVALID,
        Err(rc) => return Err(rc),
    };
    entry.descendant_indices.insert(json_index);

    Ok(pure_simdjson_value_view_t {
        doc: handle,
        state0: json_index,
        state1: DESC_VIEW_TAG,
        kind_hint,
        reserved: 0,
    })
}

pub(crate) fn element_type(
    view: *const pure_simdjson_value_view_t,
) -> Result<u32, pure_simdjson_error_code_t> {
    with_resolved_view(view, |entry, json_index, _| {
        super::native_element_type_at(entry.native_ptr, json_index)
    })
}

pub(crate) fn element_get_int64(
    view: *const pure_simdjson_value_view_t,
) -> Result<i64, pure_simdjson_error_code_t> {
    with_resolved_view(view, |entry, json_index, _| {
        super::native_element_get_int64_at(entry.native_ptr, json_index)
    })
}

pub(crate) fn element_get_uint64(
    view: *const pure_simdjson_value_view_t,
) -> Result<u64, pure_simdjson_error_code_t> {
    with_resolved_view(view, |entry, json_index, _| {
        super::native_element_get_uint64_at(entry.native_ptr, json_index)
    })
}

pub(crate) fn element_get_float64(
    view: *const pure_simdjson_value_view_t,
) -> Result<f64, pure_simdjson_error_code_t> {
    with_resolved_view(
        view,
        |entry, json_index, _| match super::native_element_type_at(entry.native_ptr, json_index)? {
            kind if kind == pure_simdjson_value_kind_t::PURE_SIMDJSON_VALUE_KIND_INT64 as u32 => {
                let value = super::native_element_get_int64_at(entry.native_ptr, json_index)?;
                if !int64_is_exact_float64(value) {
                    return Err(err_precision_loss());
                }
                Ok(value as f64)
            }
            kind if kind == pure_simdjson_value_kind_t::PURE_SIMDJSON_VALUE_KIND_UINT64 as u32 => {
                let value = super::native_element_get_uint64_at(entry.native_ptr, json_index)?;
                if !uint64_is_exact_float64(value) {
                    return Err(err_precision_loss());
                }
                Ok(value as f64)
            }
            _ => super::native_element_get_float64_at(entry.native_ptr, json_index),
        },
    )
}

pub(crate) fn element_get_string(
    view: *const pure_simdjson_value_view_t,
) -> Result<(*mut u8, usize), pure_simdjson_error_code_t> {
    let (ptr, len) = with_resolved_view(view, |entry, json_index, _| {
        let (borrowed_ptr, len) =
            super::native_element_get_string_view(entry.native_ptr, json_index)?;
        if len == 0 {
            return Ok((ptr::null_mut(), 0));
        }
        if borrowed_ptr == 0 {
            return Err(err_internal());
        }

        // SAFETY: the native bridge returned a non-null pointer for a non-empty string view, and
        // the accompanying `len` bounds the borrowed bytes for the duration of this copy.
        let bytes = unsafe { slice::from_raw_parts(borrowed_ptr as *const u8, len) };
        let mut owned = bytes.to_vec().into_boxed_slice().into_vec();
        let ptr = owned.as_mut_ptr();
        let len = owned.len();
        debug_assert_eq!(owned.len(), owned.capacity());
        mem::forget(owned);
        Ok((ptr, len))
    })?;

    if ptr.is_null() {
        return Ok((ptr, len));
    }

    let mut registry = registry_guard();
    if registry
        .string_allocations
        .insert(ptr as usize, len)
        .is_some()
    {
        // SAFETY: `ptr`/`len` came from `owned.as_mut_ptr()` and `owned.len()` immediately before
        // `mem::forget(owned)`, and `debug_assert_eq!(owned.len(), owned.capacity())` established
        // that reconstructing with `(ptr, len, len)` matches the original allocation.
        unsafe {
            drop(Vec::from_raw_parts(ptr, len, len));
        }
        return Err(err_internal());
    }

    Ok((ptr, len))
}

pub(crate) fn bytes_free(ptr: *mut u8, len: usize) -> pure_simdjson_error_code_t {
    if ptr.is_null() {
        return if len == 0 {
            err_ok()
        } else {
            err_invalid_argument()
        };
    }
    if len == 0 {
        return err_invalid_argument();
    }

    {
        let mut registry = registry_guard();
        match registry.string_allocations.remove(&(ptr as usize)) {
            Some(registered_len) if registered_len == len => {}
            Some(registered_len) => {
                registry
                    .string_allocations
                    .insert(ptr as usize, registered_len);
                return err_invalid_handle();
            }
            None => return err_invalid_handle(),
        }
    }

    // SAFETY: successful allocations are registered with exact pointer/length pairs, so this
    // reconstructs the original Vec allocation exactly once after removing its registry entry.
    unsafe {
        drop(Vec::from_raw_parts(ptr, len, len));
    }
    err_ok()
}

pub(crate) fn element_get_bool(
    view: *const pure_simdjson_value_view_t,
) -> Result<u8, pure_simdjson_error_code_t> {
    with_resolved_view(view, |entry, json_index, _| {
        super::native_element_get_bool_at(entry.native_ptr, json_index)
    })
}

pub(crate) fn element_is_null(
    view: *const pure_simdjson_value_view_t,
) -> Result<u8, pure_simdjson_error_code_t> {
    with_resolved_view(view, |entry, json_index, _| {
        super::native_element_is_null_at(entry.native_ptr, json_index)
    })
}

#[derive(Clone, Copy, Debug)]
pub(crate) struct ArrayIterStep {
    pub(crate) iter: pure_simdjson_array_iter_t,
    pub(crate) value: pure_simdjson_value_view_t,
    pub(crate) done: u8,
}

#[derive(Clone, Copy, Debug)]
pub(crate) struct ObjectIterStep {
    pub(crate) iter: pure_simdjson_object_iter_t,
    pub(crate) key: pure_simdjson_value_view_t,
    pub(crate) value: pure_simdjson_value_view_t,
    pub(crate) done: u8,
}

#[inline]
fn validate_iter_index(
    index: u64,
    root_after_index: u64,
) -> Result<(), pure_simdjson_error_code_t> {
    if index == 0 || index >= root_after_index {
        Err(err_invalid_handle())
    } else {
        Ok(())
    }
}

fn with_iter_doc<T, F>(
    doc: pure_simdjson_doc_t,
    state0: u64,
    state1: u64,
    lease_id: u32,
    tag: u16,
    reserved: u16,
    expected_tag: u16,
    action: F,
) -> Result<T, pure_simdjson_error_code_t>
where
    F: FnOnce(&mut DocEntry) -> Result<T, pure_simdjson_error_code_t>,
{
    // `state0 == state1` is the exhausted sentinel; callers handle that case
    // explicitly and still need the lease/tag checks below to run first.
    if reserved != 0 || tag != expected_tag || state0 > state1 {
        return Err(err_invalid_handle());
    }

    let (doc_index, _, doc_generation) = unpack_handle(doc)?;
    let mut registry = registry_guard();
    let entry = match registry.docs.get_mut(doc_index) {
        Some(Slot::Occupied(entry)) if entry.generation == doc_generation => entry,
        _ => return Err(err_invalid_handle()),
    };
    entry.validate_iter_lease(lease_id, state0, state1, expected_tag)?;
    validate_iter_index(state0, entry.root_after_index)?;
    validate_iter_index(state1, entry.root_after_index)?;
    action(entry)
}

pub(crate) fn array_iter_new(
    array_view: *const pure_simdjson_value_view_t,
) -> Result<pure_simdjson_array_iter_t, pure_simdjson_error_code_t> {
    with_resolved_view(array_view, |entry, json_index, doc| {
        let kind = super::native_element_type_at(entry.native_ptr, json_index)?;
        if kind != KIND_HINT_ARRAY {
            return Err(err_wrong_type());
        }

        let (state0, state1) = super::native_array_iter_bounds(entry.native_ptr, json_index)?;
        let lease_id = entry.alloc_iter_lease(ARRAY_ITER_TAG, state0, state1)?;
        Ok(pure_simdjson_array_iter_t {
            doc,
            state0,
            state1,
            index: lease_id,
            tag: ARRAY_ITER_TAG,
            reserved: 0,
        })
    })
}

pub(crate) fn array_iter_next(
    iter: *const pure_simdjson_array_iter_t,
) -> Result<ArrayIterStep, pure_simdjson_error_code_t> {
    if iter.is_null() {
        return Err(err_invalid_argument());
    }

    // SAFETY: `iter` was checked for null above, and the ABI permits callers to provide an
    // unaligned pointer to a `pure_simdjson_array_iter_t`.
    let iter = unsafe { ptr::read_unaligned(iter) };
    with_iter_doc(
        iter.doc,
        iter.state0,
        iter.state1,
        iter.index,
        iter.tag,
        iter.reserved,
        ARRAY_ITER_TAG,
        |entry| {
            if iter.state0 == iter.state1 {
                return Ok(ArrayIterStep {
                    iter,
                    value: pure_simdjson_value_view_t::default(),
                    done: 1,
                });
            }

            let value = encode_descendant_view_locked(entry, iter.doc, iter.state0)?;
            let next_state0 = super::native_element_after_index(entry.native_ptr, iter.state0)?;
            if next_state0 <= iter.state0 || next_state0 > iter.state1 {
                return Err(err_invalid_handle());
            }
            entry.update_iter_lease(iter.index, next_state0, iter.state1, ARRAY_ITER_TAG)?;

            Ok(ArrayIterStep {
                iter: pure_simdjson_array_iter_t {
                    state0: next_state0,
                    ..iter
                },
                value,
                done: 0,
            })
        },
    )
}

pub(crate) fn object_iter_new(
    object_view: *const pure_simdjson_value_view_t,
) -> Result<pure_simdjson_object_iter_t, pure_simdjson_error_code_t> {
    with_resolved_view(object_view, |entry, json_index, doc| {
        let kind = super::native_element_type_at(entry.native_ptr, json_index)?;
        if kind != KIND_HINT_OBJECT {
            return Err(err_wrong_type());
        }

        let (state0, state1) = super::native_object_iter_bounds(entry.native_ptr, json_index)?;
        let lease_id = entry.alloc_iter_lease(OBJECT_ITER_TAG, state0, state1)?;
        Ok(pure_simdjson_object_iter_t {
            doc,
            state0,
            state1,
            index: lease_id,
            tag: OBJECT_ITER_TAG,
            reserved: 0,
        })
    })
}

pub(crate) fn object_iter_next(
    iter: *const pure_simdjson_object_iter_t,
) -> Result<ObjectIterStep, pure_simdjson_error_code_t> {
    if iter.is_null() {
        return Err(err_invalid_argument());
    }

    // SAFETY: `iter` was checked for null above, and the ABI permits callers to provide an
    // unaligned pointer to a `pure_simdjson_object_iter_t`.
    let iter = unsafe { ptr::read_unaligned(iter) };
    with_iter_doc(
        iter.doc,
        iter.state0,
        iter.state1,
        iter.index,
        iter.tag,
        iter.reserved,
        OBJECT_ITER_TAG,
        |entry| {
            if iter.state0 == iter.state1 {
                return Ok(ObjectIterStep {
                    iter,
                    key: pure_simdjson_value_view_t::default(),
                    value: pure_simdjson_value_view_t::default(),
                    done: 1,
                });
            }

            let value_json_index = iter.state0.checked_add(1).ok_or_else(err_invalid_handle)?;
            if value_json_index >= iter.state1 {
                return Err(err_invalid_handle());
            }
            validate_iter_index(value_json_index, entry.root_after_index)?;

            let key_kind = super::native_element_type_at(entry.native_ptr, iter.state0)?;
            if key_kind != KIND_HINT_STRING {
                return Err(err_invalid_handle());
            }

            let key = encode_descendant_view_locked(entry, iter.doc, iter.state0)?;
            let value = encode_descendant_view_locked(entry, iter.doc, value_json_index)?;
            let next_state0 =
                super::native_element_after_index(entry.native_ptr, value_json_index)?;
            if next_state0 <= value_json_index || next_state0 > iter.state1 {
                return Err(err_invalid_handle());
            }
            entry.update_iter_lease(iter.index, next_state0, iter.state1, OBJECT_ITER_TAG)?;

            Ok(ObjectIterStep {
                iter: pure_simdjson_object_iter_t {
                    state0: next_state0,
                    ..iter
                },
                key,
                value,
                done: 0,
            })
        },
    )
}

pub(crate) fn object_get_field(
    object_view: *const pure_simdjson_value_view_t,
    key: &[u8],
) -> Result<pure_simdjson_value_view_t, pure_simdjson_error_code_t> {
    with_resolved_view(object_view, |entry, json_index, doc| {
        let kind = super::native_element_type_at(entry.native_ptr, json_index)?;
        if kind != KIND_HINT_OBJECT {
            return Err(err_wrong_type());
        }

        let value_json_index =
            super::native_object_get_field_index(entry.native_ptr, json_index, key)?;
        encode_descendant_view_locked(entry, doc, value_json_index)
    })
}

#[cfg(test)]
mod materialize_tests {
    use super::*;
    use crate::{
        runtime::psdj_internal_frame_t,
        pure_simdjson_value_kind_t::{
            PURE_SIMDJSON_VALUE_KIND_ARRAY, PURE_SIMDJSON_VALUE_KIND_BOOL,
            PURE_SIMDJSON_VALUE_KIND_INT64, PURE_SIMDJSON_VALUE_KIND_NULL,
            PURE_SIMDJSON_VALUE_KIND_OBJECT, PURE_SIMDJSON_VALUE_KIND_STRING,
            PURE_SIMDJSON_VALUE_KIND_UINT64,
        },
    };
    use std::{ptr, slice};

    fn parse_root(json: &[u8]) -> (pure_simdjson_parser_t, pure_simdjson_doc_t, pure_simdjson_value_view_t) {
        let parser = parser_new().expect("parser_new should succeed");
        let doc = parser_parse(parser, json).expect("parser_parse should succeed");
        let root = doc_root(doc).expect("doc_root should succeed");
        (parser, doc, root)
    }

    fn cleanup(parser: pure_simdjson_parser_t, doc: pure_simdjson_doc_t) {
        assert_eq!(doc_free(doc), err_ok());
        assert_eq!(parser_free(parser), err_ok());
    }

    fn build_frames(view: &pure_simdjson_value_view_t) -> Vec<psdj_internal_frame_t> {
        let (frames, frame_count) =
            materialize_build(view as *const _).expect("materialize_build should succeed");
        assert!(!frames.is_null());
        assert_ne!(frame_count, 0);

        // SAFETY: materialize_build returns a doc-owned frame span that remains valid while the
        // owning doc is live; callers copy it before freeing the doc in these tests.
        unsafe { slice::from_raw_parts(frames, frame_count).to_vec() }
    }

    fn frame_key(frame: &psdj_internal_frame_t) -> &[u8] {
        if frame.key_len == 0 {
            assert!(frame.key_ptr.is_null());
            return &[];
        }
        assert!(!frame.key_ptr.is_null());
        // SAFETY: non-empty key spans are borrowed from the live simdjson document.
        unsafe { slice::from_raw_parts(frame.key_ptr, frame.key_len) }
    }

    fn frame_string(frame: &psdj_internal_frame_t) -> &[u8] {
        if frame.string_len == 0 {
            assert!(frame.string_ptr.is_null());
            return &[];
        }
        assert!(!frame.string_ptr.is_null());
        // SAFETY: non-empty string spans are borrowed from the live simdjson document.
        unsafe { slice::from_raw_parts(frame.string_ptr, frame.string_len) }
    }

    #[test]
    fn materialize_build_returns_preorder_root_frame_stream() {
        let (parser, doc, root) =
            parse_root(br#"{"a":[1,true,null,"x"],"n":18446744073709551615}"#);

        let frames = build_frames(&root);

        assert_eq!(frames.len(), 7);
        assert_eq!(
            frames[0].kind,
            PURE_SIMDJSON_VALUE_KIND_OBJECT as u32,
            "root frame kind"
        );
        assert_eq!(frames[0].child_count, 2);
        assert_eq!(frames[0].reserved, 0);

        assert_eq!(frame_key(&frames[1]), b"a");
        assert_eq!(frames[1].kind, PURE_SIMDJSON_VALUE_KIND_ARRAY as u32);
        assert_eq!(frames[1].child_count, 4);

        assert_eq!(frames[2].kind, PURE_SIMDJSON_VALUE_KIND_INT64 as u32);
        assert_eq!(frames[2].int64_value, 1);
        assert_eq!(frames[3].kind, PURE_SIMDJSON_VALUE_KIND_BOOL as u32);
        assert_eq!(frames[3].flags, 1);
        assert_eq!(frames[4].kind, PURE_SIMDJSON_VALUE_KIND_NULL as u32);
        assert_eq!(frames[5].kind, PURE_SIMDJSON_VALUE_KIND_STRING as u32);
        assert_eq!(frame_string(&frames[5]), b"x");

        assert_eq!(frame_key(&frames[6]), b"n");
        assert_eq!(frames[6].kind, PURE_SIMDJSON_VALUE_KIND_UINT64 as u32);
        assert_eq!(frames[6].uint64_value, u64::MAX);

        cleanup(parser, doc);
    }

    #[test]
    fn materialize_build_accepts_descendant_subtree_views() {
        let (parser, doc, root) =
            parse_root(br#"{"a":[1,true,null,"x"],"n":18446744073709551615}"#);
        let array_view = object_get_field(&root as *const _, b"a")
            .expect("object_get_field should resolve a descendant view");

        let frames = build_frames(&array_view);

        assert_eq!(frames.len(), 5);
        assert_eq!(frames[0].kind, PURE_SIMDJSON_VALUE_KIND_ARRAY as u32);
        assert_eq!(frames[0].child_count, 4);
        assert_eq!(frame_key(&frames[0]), b"");
        assert_eq!(frames[4].kind, PURE_SIMDJSON_VALUE_KIND_STRING as u32);
        assert_eq!(frame_string(&frames[4]), b"x");

        cleanup(parser, doc);
    }

    #[test]
    fn materialize_build_rejects_null_outside_registry_validation() {
        let result = materialize_build(ptr::null());

        assert_eq!(result, Err(err_invalid_argument()));
    }
}
