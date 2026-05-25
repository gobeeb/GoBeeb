package mos6502

import "testing"

// TestReset covers FR-009: after AssertReset + Step, the CPU loads PC
// from $FFFC/$FFFD, sets the I flag, and consumes exactly 7 cycles.
// SP ends at $FD (three decrements from $00).
func TestReset(t *testing.T) {
	var ram flatRAM
	ram[0xFFFC] = 0x00
	ram[0xFFFD] = 0x80
	cpu := New(&ram)
	cpu.Step() // services pending RESET

	r := cpu.Registers()
	if r.PC != 0x8000 {
		t.Errorf("PC=$%04X want $8000", r.PC)
	}
	if r.SP != 0xFD {
		t.Errorf("SP=$%02X want $FD", r.SP)
	}
	if r.Cycles != 7 {
		t.Errorf("Cycles=%d want 7", r.Cycles)
	}
	if r.P&FlagInterrupt == 0 {
		t.Errorf("I flag not set after RESET")
	}
	if r.P&FlagUnused == 0 {
		t.Errorf("U flag not set after RESET")
	}
}

// TestIRQServicedWhenIClear covers FR-010: IRQ asserted with I clear
// pushes PCH/PCL/P (B clear, U set), sets live I, vectors via $FFFE,
// consumes 7 cycles.
func TestIRQServicedWhenIClear(t *testing.T) {
	var ram flatRAM
	ram[0xFFFE] = 0x00
	ram[0xFFFF] = 0x90
	cpu := New(&ram)
	cpu.SetRegisters(Registers{PC: 0x0600, SP: 0xFD, P: FlagUnused}) // I clear
	cpu.AssertIRQ(true)

	startCycles := cpu.cycles
	cpu.Step()

	r := cpu.Registers()
	if r.PC != 0x9000 {
		t.Errorf("PC=$%04X want $9000", r.PC)
	}
	if r.Cycles-startCycles != 7 {
		t.Errorf("IRQ cycles=%d want 7", r.Cycles-startCycles)
	}
	if r.P&FlagInterrupt == 0 {
		t.Errorf("I flag not set after IRQ entry")
	}
	// Stack: SP was $FD, pushes leave SP at $FA. $01FD=PCH, $01FC=PCL, $01FB=P.
	if ram[0x01FD] != 0x06 {
		t.Errorf("stack PCH=$%02X want $06", ram[0x01FD])
	}
	if ram[0x01FC] != 0x00 {
		t.Errorf("stack PCL=$%02X want $00", ram[0x01FC])
	}
	// Pushed P: U set, B clear (IRQ, not BRK).
	gotP := ram[0x01FB]
	if gotP&FlagBreak != 0 {
		t.Errorf("pushed P has B set (got $%02X) — IRQ must push B clear", gotP)
	}
	if gotP&FlagUnused == 0 {
		t.Errorf("pushed P missing U (got $%02X)", gotP)
	}
}

// TestIRQIgnoredWhenISet: IRQ does not fire while I is set.
func TestIRQIgnoredWhenISet(t *testing.T) {
	var ram flatRAM
	ram[0x0600] = 0xEA // NOP
	cpu := New(&ram)
	cpu.SetRegisters(Registers{PC: 0x0600, SP: 0xFD, P: FlagInterrupt | FlagUnused})
	cpu.AssertIRQ(true)

	cpu.Step()
	if cpu.PC != 0x0601 {
		t.Errorf("IRQ should have been ignored; PC=$%04X want $0601 (NOP advanced)", cpu.PC)
	}
}

// TestNMIServicedOnce covers FR-011: a single AssertNMI is serviced
// exactly once even if the host doesn't deassert.
func TestNMIServicedOnce(t *testing.T) {
	var ram flatRAM
	ram[0xFFFA] = 0x00
	ram[0xFFFB] = 0xA0
	// At $A000 just NOP forever.
	ram[0xA000] = 0xEA
	ram[0xA001] = 0xEA
	cpu := New(&ram)
	cpu.SetRegisters(Registers{PC: 0x0600, SP: 0xFD, P: FlagUnused})
	cpu.AssertNMI()

	cpu.Step() // services NMI
	if cpu.PC != 0xA000 {
		t.Fatalf("first NMI: PC=$%04X want $A000", cpu.PC)
	}

	startSP := cpu.SP
	cpu.Step() // should run NOP, NOT re-service NMI
	if cpu.PC != 0xA001 {
		t.Errorf("after NMI service, NOP advances: PC=$%04X want $A001", cpu.PC)
	}
	if cpu.SP != startSP {
		t.Errorf("SP changed (NMI should NOT re-fire while still asserted): SP=$%02X want $%02X", cpu.SP, startSP)
	}
}

// TestNMIRefiresAfterDeassertAssert: a second NMI service requires a
// fresh edge: deassert, then assert.
func TestNMIRefiresAfterDeassertAssert(t *testing.T) {
	var ram flatRAM
	ram[0xFFFA] = 0x00
	ram[0xFFFB] = 0xA0
	ram[0xA000] = 0xEA
	cpu := New(&ram)
	cpu.SetRegisters(Registers{PC: 0x0600, SP: 0xFD, P: FlagUnused})
	cpu.AssertNMI()
	cpu.Step()
	cpu.DeassertNMI()
	cpu.AssertNMI()
	cpu.Step() // services 2nd NMI

	if cpu.SP != 0xF7 {
		// First NMI: SP $FD → $FA. Second NMI: $FA → $F7.
		t.Errorf("SP=$%02X want $F7 (2 NMIs serviced)", cpu.SP)
	}
}

// TestBRKPushedB covers FR-012: BRK pushes PC+2, pushes P with B set
// in the pushed copy, sets I, vectors via $FFFE, consumes 7 cycles.
func TestBRKPushedB(t *testing.T) {
	var ram flatRAM
	ram[0x0600] = 0x00 // BRK
	ram[0x0601] = 0x00 // padding
	ram[0xFFFE] = 0x00
	ram[0xFFFF] = 0x80
	cpu := New(&ram)
	cpu.SetRegisters(Registers{PC: 0x0600, SP: 0xFD, P: FlagUnused})

	cpu.Step()

	if cpu.PC != 0x8000 {
		t.Errorf("PC=$%04X want $8000", cpu.PC)
	}
	if cpu.cycles != 7 {
		t.Errorf("BRK cycles=%d want 7", cpu.cycles)
	}
	if cpu.P&FlagInterrupt == 0 {
		t.Errorf("I flag not set after BRK")
	}
	// Pushed return address = $0602 (PC+2).
	if ram[0x01FD] != 0x06 {
		t.Errorf("stack PCH=$%02X want $06", ram[0x01FD])
	}
	if ram[0x01FC] != 0x02 {
		t.Errorf("stack PCL=$%02X want $02", ram[0x01FC])
	}
	// Pushed P has B set.
	if ram[0x01FB]&FlagBreak == 0 {
		t.Errorf("pushed P missing B (got $%02X) — BRK must push B set", ram[0x01FB])
	}
}

// TestRTIRestores: RTI pulls P (B masked, U forced), then PCL, then
// PCH. Round-trips a BRK.
func TestRTIRestores(t *testing.T) {
	var ram flatRAM
	ram[0x0600] = 0x00 // BRK
	ram[0x0601] = 0x00 // padding (skipped)
	ram[0x0602] = 0xA9 // LDA #$AA (resumed)
	ram[0x0603] = 0xAA
	// IRQ handler at $9000: RTI immediately
	ram[0xFFFE] = 0x00
	ram[0xFFFF] = 0x90
	ram[0x9000] = 0x40 // RTI
	cpu := New(&ram)
	cpu.SetRegisters(Registers{PC: 0x0600, SP: 0xFD, P: FlagUnused})

	cpu.Step() // BRK → vectors to $9000
	if cpu.PC != 0x9000 {
		t.Fatalf("after BRK: PC=$%04X want $9000", cpu.PC)
	}
	cpu.Step() // RTI → restores to $0602
	if cpu.PC != 0x0602 {
		t.Fatalf("after RTI: PC=$%04X want $0602", cpu.PC)
	}
	cpu.Step() // LDA #$AA
	if cpu.A != 0xAA {
		t.Errorf("A=$%02X want $AA", cpu.A)
	}
}

// TestNMIHijackOfBRK covers FR-022: NMI asserted mid-BRK before vector
// latch hijacks the vector to $FFFA, but the pushed B bit retains the
// BRK value (1).
func TestNMIHijackOfBRK(t *testing.T) {
	// hijackMemory asserts NMI on its very first read (the opcode
	// fetch). The BRK then proceeds: padding fetch, three pushes,
	// then the hijack window. By the time vector decision happens,
	// nmiPending is true → vector becomes $FFFA.
	var ram flatRAM
	ram[0x0600] = 0x00 // BRK
	ram[0xFFFA] = 0x00 // NMI vector
	ram[0xFFFB] = 0xC0
	ram[0xFFFE] = 0x00 // BRK vector (must NOT be used)
	ram[0xFFFF] = 0xB0

	cpu := New(&ram)
	cpu.SetRegisters(Registers{PC: 0x0600, SP: 0xFD, P: FlagUnused})
	// Assert NMI before stepping BRK. The Step() priority check would
	// service NMI first if checked before BRK dispatch — but we want
	// to test mid-BRK hijack. To get that: clear nmiPending after the
	// Step priority check fires NMI, OR test with the alternative
	// scenario: set nmiPending mid-BRK by hooking memory.
	//
	// Simpler scenario for this v1 implementation: directly set
	// nmiPending mid-handler. We do this by triggering BRK first
	// (priority check sees no NMI), then via a memory hook we'd
	// assert NMI before cycle 5. The cleanest workable test in our
	// model is: assert NMI between Step's pending-check and the
	// opcode dispatch. We achieve this by injecting NMI through a
	// memory hook on the BRK opcode fetch.
	hijackingMem := &nmiTriggerMem{
		inner:     &ram,
		cpu:       cpu,
		triggerOn: 0x0600, // assert NMI when CPU reads the BRK opcode
	}
	cpu.mem = hijackingMem

	cpu.Step()

	if cpu.PC != 0xC000 {
		t.Errorf("hijacked PC=$%04X want $C000 (NMI vector)", cpu.PC)
	}
	// Pushed B bit must retain BRK value (1).
	if ram[0x01FB]&FlagBreak == 0 {
		t.Errorf("pushed P=$%02X — B must remain set (BRK origin) after NMI hijack", ram[0x01FB])
	}
}

// nmiTriggerMem wraps a Memory and asserts NMI on the first read at a
// trigger address. Used to inject mid-instruction NMI for hijack tests.
type nmiTriggerMem struct {
	inner     Memory
	cpu       *CPU
	triggerOn uint16
	fired     bool
}

func (m *nmiTriggerMem) Read(addr uint16) uint8 {
	if !m.fired && addr == m.triggerOn {
		m.fired = true
		m.cpu.AssertNMI()
	}
	return m.inner.Read(addr)
}
func (m *nmiTriggerMem) Write(addr uint16, value uint8) { m.inner.Write(addr, value) }
