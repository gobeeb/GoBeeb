package mos6502

import "testing"

// TestTraceRingWrap: writing more events than capacity wraps the
// ring buffer; Snapshot returns events in chronological order.
func TestTraceRingWrap(t *testing.T) {
	tr := NewTrace(3)
	for i := 0; i < 5; i++ {
		tr.append(BusEvent{Cycle: uint64(i + 1), Addr: uint16(i), Value: uint8(i), Kind: BusRead})
	}
	snap := tr.Snapshot()
	if len(snap) != 3 {
		t.Fatalf("len=%d want 3", len(snap))
	}
	// After wrap we should have the most recent 3: cycles 3, 4, 5.
	if snap[0].Cycle != 3 || snap[1].Cycle != 4 || snap[2].Cycle != 5 {
		t.Errorf("snap cycles=%v want [3 4 5]",
			[]uint64{snap[0].Cycle, snap[1].Cycle, snap[2].Cycle})
	}
}

// TestTraceSnapshotChronological (no wrap): events come back in order.
func TestTraceSnapshotChronological(t *testing.T) {
	tr := NewTrace(8)
	tr.append(BusEvent{Cycle: 1})
	tr.append(BusEvent{Cycle: 2})
	tr.append(BusEvent{Cycle: 3})
	snap := tr.Snapshot()
	if len(snap) != 3 || snap[0].Cycle != 1 || snap[2].Cycle != 3 {
		t.Errorf("unexpected snap: %+v", snap)
	}
}

// TestSetTraceNilDetaches: nil detach is zero-overhead and produces
// no further events.
func TestSetTraceNilDetaches(t *testing.T) {
	cpu, ram := newTestCPU(Registers{PC: 0x0600})
	ram[0x0600] = 0xEA // NOP
	ram[0x0601] = 0xEA
	ram[0x0602] = 0xEA

	tr := NewTrace(64)
	cpu.SetTrace(tr)
	cpu.Step()
	first := tr.Len()
	if first == 0 {
		t.Fatal("expected events captured during attached step")
	}

	cpu.SetTrace(nil)
	cpu.Step()
	cpu.Step()
	if tr.Len() != first {
		t.Errorf("events after detach changed from %d to %d", first, tr.Len())
	}
}

// TestTraceReset: Reset clears the trace without allocating.
func TestTraceReset(t *testing.T) {
	tr := NewTrace(4)
	tr.append(BusEvent{Cycle: 1})
	tr.append(BusEvent{Cycle: 2})
	tr.Reset()
	if tr.Len() != 0 {
		t.Errorf("after Reset, Len=%d want 0", tr.Len())
	}
	if len(tr.Snapshot()) != 0 {
		t.Errorf("Snapshot after Reset non-empty")
	}
}
