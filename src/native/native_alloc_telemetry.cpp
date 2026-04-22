#include "native_alloc_telemetry.h"

#include <atomic>
#include <cstddef>
#include <cstdint>
#include <cstdlib>
#include <functional>
#include <mutex>
#include <new>
#include <unordered_map>
#include <utility>

#ifdef _WIN32
#include <malloc.h>
#endif

namespace {

constexpr std::size_t kDefaultAlignment = alignof(std::max_align_t);

template <typename T>
class MallocAllocator {
 public:
  using value_type = T;

  MallocAllocator() noexcept = default;

  template <typename U>
  MallocAllocator(const MallocAllocator<U> &) noexcept {}

  [[nodiscard]] T *allocate(std::size_t count) {
    if (count > (static_cast<std::size_t>(-1) / sizeof(T))) {
      throw std::bad_alloc();
    }

    void *ptr = std::malloc(count * sizeof(T));
    if (ptr == nullptr) {
      throw std::bad_alloc();
    }

    return static_cast<T *>(ptr);
  }

  void deallocate(T *ptr, std::size_t) noexcept {
    std::free(ptr);
  }
};

template <typename T, typename U>
bool operator==(const MallocAllocator<T> &, const MallocAllocator<U> &) noexcept {
  return true;
}

template <typename T, typename U>
bool operator!=(const MallocAllocator<T> &, const MallocAllocator<U> &) noexcept {
  return false;
}

struct AllocationRecord {
  std::uint64_t size{0};
  std::uint64_t epoch{0};
};

using PointerRegistry = std::unordered_map<
    void *,
    AllocationRecord,
    std::hash<void *>,
    std::equal_to<void *>,
    MallocAllocator<std::pair<void *const, AllocationRecord>>>;

struct TelemetryState {
  std::mutex mutex{};
  PointerRegistry live_allocations{};
  std::uint64_t current_epoch{1};
  std::uint64_t live_bytes{0};
  std::atomic<std::uint64_t> total_alloc_bytes{0};
  std::atomic<std::uint64_t> alloc_count{0};
  std::atomic<std::uint64_t> free_count{0};
};

TelemetryState &telemetry_state() noexcept {
  static TelemetryState state;
  return state;
}

std::size_t normalize_size(std::size_t size) noexcept {
  return size == 0 ? std::size_t{1} : size;
}

std::size_t normalize_alignment(std::size_t alignment) noexcept {
  return alignment < kDefaultAlignment ? kDefaultAlignment : alignment;
}

void *raw_allocate(std::size_t size, std::size_t alignment) noexcept {
  const std::size_t aligned_size = normalize_size(size);
  const std::size_t aligned_alignment = normalize_alignment(alignment);

#ifdef _WIN32
  return _aligned_malloc(aligned_size, aligned_alignment);
#else
  void *ptr = nullptr;
  if (posix_memalign(&ptr, aligned_alignment, aligned_size) != 0) {
    return nullptr;
  }
  return ptr;
#endif
}

void raw_deallocate(void *ptr) noexcept {
  if (ptr == nullptr) {
    return;
  }

#ifdef _WIN32
  _aligned_free(ptr);
#else
  std::free(ptr);
#endif
}

void record_allocation(void *ptr, std::size_t size) {
  auto &state = telemetry_state();
  const auto tracked_size = static_cast<std::uint64_t>(normalize_size(size));
  AllocationRecord record{};
  record.size = tracked_size;
  record.epoch = state.current_epoch;

  std::lock_guard<std::mutex> lock(state.mutex);
  state.live_allocations.emplace(ptr, record);
  state.live_bytes += tracked_size;
  state.total_alloc_bytes.fetch_add(tracked_size, std::memory_order_relaxed);
  state.alloc_count.fetch_add(1, std::memory_order_relaxed);
}

void remove_allocation(void *ptr) noexcept {
  if (ptr == nullptr) {
    return;
  }

  auto &state = telemetry_state();

  {
    std::lock_guard<std::mutex> lock(state.mutex);
    const auto it = state.live_allocations.find(ptr);
    if (it != state.live_allocations.end()) {
      if (it->second.epoch == state.current_epoch) {
        state.live_bytes -= it->second.size;
        state.free_count.fetch_add(1, std::memory_order_relaxed);
      }
      state.live_allocations.erase(it);
    }
  }

  raw_deallocate(ptr);
}

void *allocate_or_throw(std::size_t size, std::size_t alignment) {
  void *ptr = raw_allocate(size, alignment);
  if (ptr == nullptr) {
    throw std::bad_alloc();
  }

  try {
    record_allocation(ptr, size);
    return ptr;
  } catch (...) {
    raw_deallocate(ptr);
    throw;
  }
}

void *allocate_or_null(std::size_t size, std::size_t alignment) noexcept {
  void *ptr = raw_allocate(size, alignment);
  if (ptr == nullptr) {
    return nullptr;
  }

  try {
    record_allocation(ptr, size);
    return ptr;
  } catch (...) {
    raw_deallocate(ptr);
    return nullptr;
  }
}

}  // namespace

namespace psimdjson::native_alloc_telemetry {

void reset() noexcept {
  auto &state = telemetry_state();
  std::lock_guard<std::mutex> lock(state.mutex);
  state.current_epoch += 1;
  if (state.current_epoch == 0) {
    state.current_epoch = 1;
  }
  state.live_bytes = 0;
  state.total_alloc_bytes.store(0, std::memory_order_relaxed);
  state.alloc_count.store(0, std::memory_order_relaxed);
  state.free_count.store(0, std::memory_order_relaxed);
}

pure_simdjson_error_code_t snapshot(pure_simdjson_native_alloc_stats_t *out_stats) noexcept {
  if (out_stats == nullptr) {
    return PURE_SIMDJSON_ERR_INVALID_ARGUMENT;
  }

  auto &state = telemetry_state();
  std::lock_guard<std::mutex> lock(state.mutex);
  out_stats->live_bytes = state.live_bytes;
  out_stats->total_alloc_bytes = state.total_alloc_bytes.load(std::memory_order_relaxed);
  out_stats->alloc_count = state.alloc_count.load(std::memory_order_relaxed);
  out_stats->free_count = state.free_count.load(std::memory_order_relaxed);
  return PURE_SIMDJSON_OK;
}

}  // namespace psimdjson::native_alloc_telemetry

void *operator new(std::size_t size) {
  return allocate_or_throw(size, kDefaultAlignment);
}

void *operator new[](std::size_t size) {
  return allocate_or_throw(size, kDefaultAlignment);
}

void *operator new(std::size_t size, const std::nothrow_t &) noexcept {
  return allocate_or_null(size, kDefaultAlignment);
}

void *operator new[](std::size_t size, const std::nothrow_t &) noexcept {
  return allocate_or_null(size, kDefaultAlignment);
}

void *operator new(std::size_t size, std::align_val_t alignment) {
  return allocate_or_throw(size, static_cast<std::size_t>(alignment));
}

void *operator new[](std::size_t size, std::align_val_t alignment) {
  return allocate_or_throw(size, static_cast<std::size_t>(alignment));
}

void *operator new(std::size_t size, std::align_val_t alignment, const std::nothrow_t &) noexcept {
  return allocate_or_null(size, static_cast<std::size_t>(alignment));
}

void *operator new[](
    std::size_t size,
    std::align_val_t alignment,
    const std::nothrow_t &
) noexcept {
  return allocate_or_null(size, static_cast<std::size_t>(alignment));
}

void operator delete(void *ptr) noexcept {
  remove_allocation(ptr);
}

void operator delete[](void *ptr) noexcept {
  remove_allocation(ptr);
}

void operator delete(void *ptr, std::size_t) noexcept {
  remove_allocation(ptr);
}

void operator delete[](void *ptr, std::size_t) noexcept {
  remove_allocation(ptr);
}

void operator delete(void *ptr, const std::nothrow_t &) noexcept {
  remove_allocation(ptr);
}

void operator delete[](void *ptr, const std::nothrow_t &) noexcept {
  remove_allocation(ptr);
}

void operator delete(void *ptr, std::align_val_t) noexcept {
  remove_allocation(ptr);
}

void operator delete[](void *ptr, std::align_val_t) noexcept {
  remove_allocation(ptr);
}

void operator delete(void *ptr, std::align_val_t, const std::nothrow_t &) noexcept {
  remove_allocation(ptr);
}

void operator delete[](void *ptr, std::align_val_t, const std::nothrow_t &) noexcept {
  remove_allocation(ptr);
}

void operator delete(void *ptr, std::size_t, std::align_val_t) noexcept {
  remove_allocation(ptr);
}

void operator delete[](void *ptr, std::size_t, std::align_val_t) noexcept {
  remove_allocation(ptr);
}
