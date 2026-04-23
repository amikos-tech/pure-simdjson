#ifndef PSIMDJSON_NATIVE_ALLOC_TELEMETRY_H
#define PSIMDJSON_NATIVE_ALLOC_TELEMETRY_H

#include "../../include/pure_simdjson.h"

namespace psimdjson::native_alloc_telemetry {

void reset() noexcept;
pure_simdjson_error_code_t snapshot(pure_simdjson_native_alloc_stats_t *out_stats) noexcept;

}  // namespace psimdjson::native_alloc_telemetry

#endif
