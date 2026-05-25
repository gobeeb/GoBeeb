package mos6502

import (
	_ "embed"
	"testing"
)

//go:embed testdata/6502_functional_test.bin
var functionalROM []byte

// flatRAM is the simplest possible Memory: a 64 KB byte slice mapped
// directly to the 16-bit address space.
type flatRAM [0x10000]byte

func (r *flatRAM) Read(addr uint16) uint8         { return r[addr] }
func (r *flatRAM) Write(addr uint16, value uint8) { r[addr] = value }

// TestFunctional runs Klaus Dormann's 6502_functional_test ROM and
// asserts it reaches the documented success trap at $3469. Covers
// every documented NMOS opcode + flag behaviour + decimal mode.
// (SC-001, FR-002, FR-016)
func TestFunctional(t *testing.T) {
	if len(functionalROM) != 0x10000 {
		t.Fatalf("functional ROM size = %d, want 65536", len(functionalROM))
	}
	const (
		entryPC       = 0x0400
		successTrapPC = 0x3469
		maxCycles     = 200_000_000
	)

	var ram flatRAM
	copy(ram[:], functionalROM)
	cpu := New(&ram)
	cpu.SetRegisters(Registers{SP: 0xFD, PC: entryPC, P: FlagInterrupt | FlagUnused})

	prevPC := uint16(0xFFFF)
	for cpu.cycles < maxCycles {
		startPC := cpu.PC
		cpu.Step()
		// Trap detection: PC unchanged AND identical to previous PC
		// (i.e. the same JMP * has fired twice).
		if cpu.PC == startPC && cpu.PC == prevPC {
			if cpu.PC == successTrapPC {
				return // pass
			}
			t.Fatalf("trap at $%04X (expected success trap at $%04X); cycles=%d, regs=%+v",
				cpu.PC, successTrapPC, cpu.cycles, cpu.Registers())
		}
		prevPC = startPC
	}
	t.Fatalf("ran out of cycle budget (%d cycles) without reaching success trap; PC=$%04X", maxCycles, cpu.PC)
}
