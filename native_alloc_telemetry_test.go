package purejson

import "testing"

// TestUntrackedFreeCountStaysZero guards the public NativeAllocStats.UntrackedFreeCount
// counter: a healthy parse/close cycle must never increment it. The counter is incremented
// in native code when remove_allocation receives a pointer that is not in the live
// allocations map (double free, stray free, bookkeeping bug). Making this assertion explicit
// means any future change that introduces such a path fails this test instead of silently
// bleeding into production telemetry.
func TestUntrackedFreeCountStaysZero(t *testing.T) {
	library, err := activeLibrary()
	if err != nil {
		t.Fatalf("activeLibrary(): %v", err)
	}

	if err := wrapStatus(library.bindings.NativeAllocStatsReset()); err != nil {
		t.Fatalf("NativeAllocStatsReset(): %v", err)
	}

	for i := 0; i < 16; i++ {
		parser := mustNewParser(t)
		doc, err := parser.Parse([]byte(`{"a":1,"b":[true,false,null],"c":"hello"}`))
		if err != nil {
			_ = parser.Close()
			t.Fatalf("Parse() error = %v", err)
		}
		if err := doc.Close(); err != nil {
			_ = parser.Close()
			t.Fatalf("doc.Close() error = %v", err)
		}
		if err := parser.Close(); err != nil {
			t.Fatalf("parser.Close() error = %v", err)
		}
	}

	stats, rc := library.bindings.NativeAllocStatsSnapshot()
	if err := wrapStatus(rc); err != nil {
		t.Fatalf("NativeAllocStatsSnapshot(): %v", err)
	}

	if stats.UntrackedFreeCount != 0 {
		t.Fatalf("UntrackedFreeCount = %d, want 0 (indicates stray free in native telemetry)", stats.UntrackedFreeCount)
	}
	if stats.AllocCount == 0 {
		t.Fatalf("AllocCount = 0, want > 0 (telemetry path not exercised)")
	}
}
