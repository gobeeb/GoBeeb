package mos6502

import "testing"

// TestIllegalOpcodeNOPBehaviour: an undocumented opcode executes as a
// single-byte, 2-cycle NOP (FR-019). The CPU does not halt.
func TestIllegalOpcodeNOPBehaviour(t *testing.T) {
	cpu, ram := newTestCPU(Registers{PC: 0x0600})
	ram[0x0600] = 0x02 // illegal (KIL on NMOS, but we treat as NOP)
	ram[0x0601] = 0xA9 // LDA #$33  — the byte after the illegal
	ram[0x0602] = 0x33

	startCycles := cpu.cycles
	cpu.Step()
	if cpu.cycles-startCycles != 2 {
		t.Errorf("illegal opcode cycles=%d want 2", cpu.cycles-startCycles)
	}
	if cpu.PC != 0x0601 {
		t.Errorf("PC=$%04X want $0601 (single-byte advance)", cpu.PC)
	}

	// Next Step should execute the LDA normally.
	cpu.Step()
	if cpu.A != 0x33 {
		t.Errorf("A=$%02X want $33 (LDA after illegal)", cpu.A)
	}
}

// TestIllegalOpcodeHook: the hook is invoked exactly once per illegal
// opcode with pre-advance PC and the offending opcode byte.
func TestIllegalOpcodeHook(t *testing.T) {
	cpu, ram := newTestCPU(Registers{PC: 0x0600})
	ram[0x0600] = 0x02 // illegal #1
	ram[0x0601] = 0xEA // NOP (documented)
	ram[0x0602] = 0x12 // illegal #2

	type call struct {
		pc uint16
		op uint8
	}
	var calls []call
	cpu.SetIllegalOpcodeHook(func(pc uint16, op uint8) {
		calls = append(calls, call{pc, op})
	})

	cpu.Step() // illegal at $0600
	cpu.Step() // NOP at $0601 (no hook)
	cpu.Step() // illegal at $0602

	want := []call{{0x0600, 0x02}, {0x0602, 0x12}}
	if len(calls) != len(want) {
		t.Fatalf("hook called %d times, want %d (%+v)", len(calls), len(want), calls)
	}
	for i, w := range want {
		if calls[i] != w {
			t.Errorf("call[%d] = %+v want %+v", i, calls[i], w)
		}
	}
}

// TestIllegalOpcodeHookNilClears: SetIllegalOpcodeHook(nil) clears the
// hook; subsequent illegal opcodes do not invoke the previous callback.
func TestIllegalOpcodeHookNilClears(t *testing.T) {
	cpu, ram := newTestCPU(Registers{PC: 0x0600})
	ram[0x0600] = 0x02
	ram[0x0601] = 0x02

	calls := 0
	cpu.SetIllegalOpcodeHook(func(uint16, uint8) { calls++ })
	cpu.Step()
	if calls != 1 {
		t.Fatalf("after 1 illegal, calls=%d want 1", calls)
	}

	cpu.SetIllegalOpcodeHook(nil)
	cpu.Step()
	if calls != 1 {
		t.Errorf("after nil-clear and second illegal, calls=%d want 1 (no further invocation)", calls)
	}
}
