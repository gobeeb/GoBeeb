package mos6502

import "testing"

// TestNewZeroCycles: a fresh CPU has cycle count 0.
func TestNewZeroCycles(t *testing.T) {
	var ram flatRAM
	cpu := New(&ram)
	if cpu.Registers().Cycles != 0 {
		t.Errorf("fresh CPU cycles=%d want 0", cpu.Registers().Cycles)
	}
}

// TestSetRegistersRoundTrip: SetRegisters → Registers round-trips.
func TestSetRegistersRoundTrip(t *testing.T) {
	cpu, _ := newTestCPU(Registers{
		A: 0xAA, X: 0xBB, Y: 0xCC, SP: 0xDD, PC: 0x1234,
		P: FlagCarry | FlagZero, Cycles: 42,
	})
	r := cpu.Registers()
	if r.A != 0xAA || r.X != 0xBB || r.Y != 0xCC || r.SP != 0xDD || r.PC != 0x1234 || r.Cycles != 42 {
		t.Errorf("round-trip: %+v", r)
	}
}

// TestStepAdvancesOneInstruction: a single Step advances by exactly one
// instruction (and the documented cycle cost for that opcode).
func TestStepAdvancesOneInstruction(t *testing.T) {
	cpu, ram := newTestCPU(Registers{PC: 0x0600})
	ram[0x0600] = 0xA9 // LDA #$42
	ram[0x0601] = 0x42
	cpu.Step()
	if cpu.PC != 0x0602 {
		t.Errorf("PC=$%04X want $0602", cpu.PC)
	}
	if cpu.cycles != 2 {
		t.Errorf("cycles=%d want 2", cpu.cycles)
	}
}

// TestRunNoOvershootSplit: Run never returns mid-instruction.
func TestRunNoOvershootSplit(t *testing.T) {
	cpu, ram := newTestCPU(Registers{PC: 0x0600})
	// Two LDA #imm (2 cycles each).
	ram[0x0600] = 0xA9
	ram[0x0601] = 0x01
	ram[0x0602] = 0xA9
	ram[0x0603] = 0x02
	ram[0x0604] = 0xA9
	ram[0x0605] = 0x03

	cpu.Run(3) // budget of 3 cycles — should run 2 instructions (4 cycles, overshoot)
	if cpu.PC != 0x0604 {
		t.Errorf("Run(3) PC=$%04X want $0604 (2 instructions completed)", cpu.PC)
	}
	if cpu.cycles != 4 {
		t.Errorf("Run(3) cycles=%d want 4", cpu.cycles)
	}
}

// TestDeterminism: two independently-constructed CPUs running the same
// program produce byte-identical state. (SC-007)
func TestDeterminism(t *testing.T) {
	program := []byte{
		0xA9, 0x01, // LDA #$01
		0x69, 0x02, // ADC #$02
		0x8D, 0x00, 0x02, // STA $0200
		0xE6, 0x10, // INC $10
		0xA2, 0x05, // LDX #$05
		0xCA,       // DEX
		0xD0, 0xFD, // BNE -3
		0x00, // BRK
	}
	run := func() Registers {
		var ram flatRAM
		copy(ram[0x0600:], program)
		ram[0xFFFE] = 0x00
		ram[0xFFFF] = 0xFF // BRK vectors to $FF00 (will hit zeros)
		cpu := New(&ram)
		cpu.SetRegisters(Registers{PC: 0x0600, SP: 0xFD, P: FlagUnused})
		for i := 0; i < 50; i++ {
			cpu.Step()
		}
		return cpu.Registers()
	}
	r1 := run()
	r2 := run()
	if r1 != r2 {
		t.Errorf("non-deterministic: r1=%+v r2=%+v", r1, r2)
	}
}
