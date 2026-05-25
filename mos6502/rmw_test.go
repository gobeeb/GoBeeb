package mos6502

import "testing"

// TestRMWDoubleWrite verifies FR-021: RMW instructions emit three
// consecutive bus cycles to the effective address — read, dummy-write
// of original, write of modified — captured via Trace.
func TestRMWDoubleWrite(t *testing.T) {
	cpu, ram := newTestCPU(Registers{PC: 0x0600})
	// INC $1234
	ram[0x0600] = 0xEE
	ram[0x0601] = 0x34
	ram[0x0602] = 0x12
	ram[0x1234] = 0x41

	tr := NewTrace(64)
	cpu.SetTrace(tr)

	cpu.Step()

	if ram[0x1234] != 0x42 {
		t.Fatalf("mem[$1234]=$%02X want $42", ram[0x1234])
	}

	events := tr.Snapshot()
	// Expected events: read $0600 (opcode), read $0601 (lo), read $0602 (hi),
	// then RMW: read $1234, write $1234 = $41 (dummy of original), write $1234 = $42.
	want := []BusEvent{
		{Cycle: 1, Addr: 0x0600, Value: 0xEE, Kind: BusRead},
		{Cycle: 2, Addr: 0x0601, Value: 0x34, Kind: BusRead},
		{Cycle: 3, Addr: 0x0602, Value: 0x12, Kind: BusRead},
		{Cycle: 4, Addr: 0x1234, Value: 0x41, Kind: BusRead},
		{Cycle: 5, Addr: 0x1234, Value: 0x41, Kind: BusWrite},
		{Cycle: 6, Addr: 0x1234, Value: 0x42, Kind: BusWrite},
	}
	if len(events) != len(want) {
		t.Fatalf("got %d events, want %d: %+v", len(events), len(want), events)
	}
	for i, w := range want {
		if events[i] != w {
			t.Errorf("event[%d]=%+v want %+v", i, events[i], w)
		}
	}
}

// TestRMWAccumulator: ASL A is not a memory RMW — it operates on the
// accumulator with no memory cycles for the operand.
func TestRMWAccumulator(t *testing.T) {
	cpu, ram := newTestCPU(Registers{A: 0x42, PC: 0x0600})
	ram[0x0600] = 0x0A // ASL A

	tr := NewTrace(8)
	cpu.SetTrace(tr)

	cpu.Step()

	if cpu.A != 0x84 {
		t.Errorf("A=$%02X want $84", cpu.A)
	}
	events := tr.Snapshot()
	// Expect 2 events: opcode fetch + dummy fetch at PC.
	if len(events) != 2 {
		t.Errorf("got %d events, want 2: %+v", len(events), events)
	}
}
