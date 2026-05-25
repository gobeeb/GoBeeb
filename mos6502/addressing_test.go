package mos6502

import "testing"

// newTestCPU returns a CPU bound to a fresh flatRAM with the given
// register state (no RESET sequence).
func newTestCPU(r Registers) (*CPU, *flatRAM) {
	var ram flatRAM
	cpu := New(&ram)
	cpu.SetRegisters(r)
	return cpu, &ram
}

// TestZeroPageWrap exercises FR-014: zero-page indexed addressing wraps
// inside zero page when the index pushes the effective address past $FF.
func TestZeroPageWrap(t *testing.T) {
	cpu, ram := newTestCPU(Registers{X: 0x10, PC: 0x0600})
	// LDA $F8,X  with X=$10  →  effective $08
	ram[0x0600] = 0xB5 // LDA zp,X
	ram[0x0601] = 0xF8
	ram[0x0008] = 0x42

	cpu.Step()
	if cpu.A != 0x42 {
		t.Errorf("zp,X wrap: A=$%02X want $42", cpu.A)
	}
}

// TestZeroPageWrap_IndexedIndirect exercises FR-014 on the (zp,X)
// pointer fetch: the indexed pointer wraps in zero page.
func TestZeroPageWrap_IndexedIndirect(t *testing.T) {
	cpu, ram := newTestCPU(Registers{X: 0x10, PC: 0x0600})
	// LDA ($F0,X)  with X=$10  → pointer at $0000/$0001
	ram[0x0600] = 0xA1 // LDA (zp,X)
	ram[0x0601] = 0xF0
	ram[0x0000] = 0x34
	ram[0x0001] = 0x12
	ram[0x1234] = 0x77

	cpu.Step()
	if cpu.A != 0x77 {
		t.Errorf("(zp,X) wrap: A=$%02X want $77", cpu.A)
	}
}

// TestZeroPageWrap_IndirectIndexed exercises the pointer-high wrap on
// ($LL),Y: pointer high byte fetched from zero page wrap. Here the
// low byte of the pointer is at $FF, high byte at $00.
func TestZeroPageWrap_IndirectIndexedPointer(t *testing.T) {
	cpu, ram := newTestCPU(Registers{Y: 0x00, PC: 0x0600})
	ram[0x0600] = 0xB1 // LDA (zp),Y
	ram[0x0601] = 0xFF
	ram[0x00FF] = 0x34 // pointer low
	ram[0x0000] = 0x12 // pointer high (wrapped)
	ram[0x1234] = 0x99

	cpu.Step()
	if cpu.A != 0x99 {
		t.Errorf("(zp),Y pointer wrap: A=$%02X want $99", cpu.A)
	}
}

// TestJMPIndirectPageBug exercises FR-013: JMP ($xxFF) reads the high
// byte of the target from $xx00 of the same page, not the next page.
func TestJMPIndirectPageBug(t *testing.T) {
	cpu, ram := newTestCPU(Registers{PC: 0x0600})
	ram[0x0600] = 0x6C // JMP (abs)
	ram[0x0601] = 0xFF
	ram[0x0602] = 0x10
	ram[0x10FF] = 0x34 // target low
	ram[0x1000] = 0x12 // target high (NMOS bug: NOT $1100)
	ram[0x1100] = 0xAA // would-be-correct target high (must be ignored)

	cpu.Step()
	if cpu.PC != 0x1234 {
		t.Errorf("JMP ($10FF): PC=$%04X want $1234 (NMOS page bug)", cpu.PC)
	}
}

// TestAbsXPageCrossPenalty exercises FR-018: indexed-absolute reads
// pay a dummy-read cycle on page-cross.
func TestAbsXPageCrossPenalty(t *testing.T) {
	cpu, ram := newTestCPU(Registers{X: 0x01, PC: 0x0600})
	// LDA $00FF,X  → effective $0100, crosses page from $0000 to $0100
	ram[0x0600] = 0xBD
	ram[0x0601] = 0xFF
	ram[0x0602] = 0x00
	ram[0x0100] = 0x55

	start := cpu.cycles
	cpu.Step()
	delta := cpu.cycles - start
	if cpu.A != 0x55 {
		t.Errorf("A=$%02X want $55", cpu.A)
	}
	if delta != 5 {
		t.Errorf("cycles=%d want 5 (LDA abs,X with page-cross)", delta)
	}
}

// TestAbsXNoCrossNoPenalty: no page-cross → 4 cycles.
func TestAbsXNoCrossNoPenalty(t *testing.T) {
	cpu, ram := newTestCPU(Registers{X: 0x01, PC: 0x0600})
	ram[0x0600] = 0xBD
	ram[0x0601] = 0x10
	ram[0x0602] = 0x00
	ram[0x0011] = 0x55

	start := cpu.cycles
	cpu.Step()
	delta := cpu.cycles - start
	if cpu.A != 0x55 {
		t.Errorf("A=$%02X want $55", cpu.A)
	}
	if delta != 4 {
		t.Errorf("cycles=%d want 4 (no page-cross)", delta)
	}
}

// TestSTAAbsXAlwaysPenalty: stores always pay the indexed-penalty
// cycle, even without a page-cross. STA abs,X = 5 cycles always.
func TestSTAAbsXAlwaysPenalty(t *testing.T) {
	cpu, ram := newTestCPU(Registers{A: 0x77, X: 0x01, PC: 0x0600})
	ram[0x0600] = 0x9D // STA abs,X
	ram[0x0601] = 0x10
	ram[0x0602] = 0x00

	start := cpu.cycles
	cpu.Step()
	delta := cpu.cycles - start
	if ram[0x0011] != 0x77 {
		t.Errorf("mem[$0011]=$%02X want $77", ram[0x0011])
	}
	if delta != 5 {
		t.Errorf("STA abs,X cycles=%d want 5 (always-penalty)", delta)
	}
}

// TestBranchTakenPageCross: branch +1 on taken, +1 more on cross.
func TestBranchTakenPageCross(t *testing.T) {
	cases := []struct {
		name        string
		startPC     uint16
		offset      int8
		expectExtra uint64
	}{
		{"not taken", 0x0600, 0x10, 0},      // base 2 cycles
		{"taken no cross", 0x0600, 0x10, 1}, // +1 cycle on taken (no cross within page)
		{"taken cross", 0x06F0, 0x40, 2},    // +1 taken, +1 cross
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cpu, ram := newTestCPU(Registers{PC: c.startPC, P: FlagCarry})
			// BCS rel  — taken if C set; we set C above (for "taken" cases)
			// For "not taken" case, clear carry.
			taken := c.name != "not taken"
			if !taken {
				cpu.P = 0
			}
			ram[c.startPC] = 0xB0
			ram[c.startPC+1] = uint8(c.offset)
			start := cpu.cycles
			cpu.Step()
			delta := cpu.cycles - start
			want := uint64(2) + c.expectExtra
			if delta != want {
				t.Errorf("BCS %s cycles=%d want %d", c.name, delta, want)
			}
		})
	}
}
