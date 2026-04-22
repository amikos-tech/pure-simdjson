package purejson

import "testing"

const (
	benchmarkMetricNativeBytesPerOp  = "native-bytes/op"
	benchmarkMetricNativeAllocsPerOp = "native-allocs/op"
	benchmarkMetricNativeLiveBytes   = "native-live-bytes"
)

func benchmarkRunWithNativeAllocMetrics(b *testing.B, run func()) {
	b.Helper()

	library, err := activeLibrary()
	if err != nil {
		b.Fatalf("activeLibrary(): %v", err)
	}
	if err := wrapStatus(library.bindings.NativeAllocStatsReset()); err != nil {
		b.Fatalf("NativeAllocStatsReset(): %v", err)
	}

	b.ResetTimer()
	run()
	b.StopTimer()

	stats, rc := library.bindings.NativeAllocStatsSnapshot()
	if err := wrapStatus(rc); err != nil {
		b.Fatalf("NativeAllocStatsSnapshot(): %v", err)
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
