package purejson

import "testing"

const (
	benchmarkMetricNativeBytesPerOp  = "native-bytes/op"
	benchmarkMetricNativeAllocsPerOp = "native-allocs/op"
	benchmarkMetricNativeLiveBytes   = "native-live-bytes"
)

func benchmarkRunWithNativeAllocMetrics(b *testing.B, requireNativeAllocs bool, run func()) {
	b.Helper()

	library, err := activeLibrary()
	if err != nil {
		b.Fatalf("activeLibrary(): %v", err)
	}
	if err := wrapStatus(library.bindings.NativeAllocStatsReset()); err != nil {
		b.Fatalf("NativeAllocStatsReset(): %v", err)
	}

	// Why: benchmark bodies using this helper are single-threaded, so reset/run/snapshot forms
	// a closed native allocation window for the comparator under measurement.
	b.ResetTimer()
	run()
	b.StopTimer()

	stats, rc := library.bindings.NativeAllocStatsSnapshot()
	if err := wrapStatus(rc); err != nil {
		b.Fatalf("NativeAllocStatsSnapshot(): %v", err)
	}
	if requireNativeAllocs && b.N > 0 && stats.AllocCount == 0 {
		b.Fatalf("NativeAllocStatsSnapshot(): alloc_count = 0, want native allocation telemetry for this path")
	}

	if b.N > 0 {
		perOp := float64(b.N)
		b.ReportMetric(float64(stats.TotalAllocBytes)/perOp, benchmarkMetricNativeBytesPerOp)
		b.ReportMetric(float64(stats.AllocCount)/perOp, benchmarkMetricNativeAllocsPerOp)
	} else {
		b.ReportMetric(0, benchmarkMetricNativeBytesPerOp)
		b.ReportMetric(0, benchmarkMetricNativeAllocsPerOp)
	}
	b.ReportMetric(float64(stats.LiveBytes), benchmarkMetricNativeLiveBytes)
}
