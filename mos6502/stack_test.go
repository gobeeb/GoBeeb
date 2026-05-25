package mos6502

import "testing"

// TestStackWrapPHA exercises FR-015: with SP=$00, PHA writes to $0100
// and leaves SP=$FF.
func TestStackWrapPHA(t *testing.T) {
	cpu, ram := newTestCPU(Registers{A: 0xAB, SP: 0x00, PC: 0x0600})
	ram[0x0600] = 0x48 // PHA
	cpu.Step()
	if ram[0x0100] != 0xAB {
		t.Errorf("mem[$0100]=$%02X want $AB", ram[0x0100])
	}
	if cpu.SP != 0xFF {
		t.Errorf("SP=$%02X want $FF", cpu.SP)
	}
}

// TestStackWrapPLA exercises FR-015: with SP=$FF, PLA reads from $0100
// and leaves SP=$00.
func TestStackWrapPLA(t *testing.T) {
	cpu, ram := newTestCPU(Registers{SP: 0xFF, PC: 0x0600})
	ram[0x0600] = 0x68 // PLA
	ram[0x0100] = 0xCD
	cpu.Step()
	if cpu.A != 0xCD {
		t.Errorf("A=$%02X want $CD", cpu.A)
	}
	if cpu.SP != 0x00 {
		t.Errorf("SP=$%02X want $00", cpu.SP)
	}
}

// TestJSRRTSAcrossWrap: JSR/RTS work across the stack-wrap boundary.
func TestJSRRTSAcrossWrap(t *testing.T) {
	cpu, ram := newTestCPU(Registers{SP: 0x00, PC: 0x0600})
	ram[0x0600] = 0x20 // JSR $0700
	ram[0x0601] = 0x00
	ram[0x0602] = 0x07
	ram[0x0700] = 0x60 // RTS

	cpu.Step() // JSR
	if cpu.SP != 0xFE {
		t.Errorf("after JSR: SP=$%02X want $FE", cpu.SP)
	}
	if cpu.PC != 0x0700 {
		t.Errorf("after JSR: PC=$%04X want $0700", cpu.PC)
	}
	cpu.Step() // RTS
	if cpu.SP != 0x00 {
		t.Errorf("after RTS: SP=$%02X want $00", cpu.SP)
	}
	if cpu.PC != 0x0603 {
		t.Errorf("after RTS: PC=$%04X want $0603", cpu.PC)
	}
}
