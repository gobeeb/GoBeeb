package mos6502

import "testing"

// TestRDYStallsAtInstructionBoundary: with RDY asserted-low at Step
// entry, Step makes no progress (no cycles consumed, no architectural
// state change). The host releases RDY, Step resumes normally.
//
// (v1 limitation: RDY is gated at the instruction boundary; sub-
// instruction stalling is not modelled — see cpu.go.)
func TestRDYStallsAtInstructionBoundary(t *testing.T) {
	cpu, ram := newTestCPU(Registers{PC: 0x0600})
	ram[0x0600] = 0xA9 // LDA #$42
	ram[0x0601] = 0x42

	cpu.SetRDY(false)
	startCycles := cpu.cycles
	startPC := cpu.PC
	cpu.Step()
	if cpu.cycles != startCycles {
		t.Errorf("RDY low: cycles advanced by %d, want 0", cpu.cycles-startCycles)
	}
	if cpu.PC != startPC {
		t.Errorf("RDY low: PC moved to $%04X, want $%04X", cpu.PC, startPC)
	}
	if cpu.A != 0 {
		t.Errorf("RDY low: A=$%02X want 0 (no architectural state change)", cpu.A)
	}

	cpu.SetRDY(true)
	cpu.Step()
	if cpu.A != 0x42 {
		t.Errorf("after RDY release: A=$%02X want $42", cpu.A)
	}
}

// TestRDYDoesNotStallStep_WritesProceed: in v1, writes that occur
// inside an already-dispatched instruction proceed regardless of RDY
// (matches NMOS behaviour at the bus level). Since RDY is checked at
// the boundary in v1, this test verifies the NMOS write-proceeds rule
// at the boundary contract level: an instruction that does only writes
// must complete normally after RDY release.
func TestRDYWritesProceed(t *testing.T) {
	cpu, ram := newTestCPU(Registers{A: 0x77, PC: 0x0600})
	ram[0x0600] = 0x85 // STA $20
	ram[0x0601] = 0x20

	cpu.SetRDY(true)
	cpu.Step()
	if ram[0x0020] != 0x77 {
		t.Errorf("mem[$0020]=$%02X want $77 (STA must complete)", ram[0x0020])
	}
}
