package mos6502

import "testing"

// TestADCBinary covers ADC corner cases: carry-in, overflow-set,
// overflow-clear, zero result, negative result.
func TestADCBinary(t *testing.T) {
	cases := []struct {
		name         string
		a, m         uint8
		cIn          bool
		wantA        uint8
		wantC, wantV bool
		wantZ, wantN bool
	}{
		{"0+0", 0, 0, false, 0, false, false, true, false},
		{"127+1 overflow", 127, 1, false, 128, false, true, false, true},
		{"-1 + 1 carry, no overflow", 0xFF, 1, false, 0, true, false, true, false},
		{"-128 + -1 overflow", 0x80, 0xFF, false, 0x7F, true, true, false, false},
		{"carry-in propagates", 0x40, 0x40, true, 0x81, false, true, false, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cpu, _ := newTestCPU(Registers{A: c.a})
			cpu.setFlag(FlagCarry, c.cIn)
			opAdc(cpu, c.m)
			if cpu.A != c.wantA {
				t.Errorf("A=$%02X want $%02X", cpu.A, c.wantA)
			}
			if cpu.flag(FlagCarry) != c.wantC {
				t.Errorf("C=%v want %v", cpu.flag(FlagCarry), c.wantC)
			}
			if cpu.flag(FlagOverflow) != c.wantV {
				t.Errorf("V=%v want %v", cpu.flag(FlagOverflow), c.wantV)
			}
			if cpu.flag(FlagZero) != c.wantZ {
				t.Errorf("Z=%v want %v", cpu.flag(FlagZero), c.wantZ)
			}
			if cpu.flag(FlagNegative) != c.wantN {
				t.Errorf("N=%v want %v", cpu.flag(FlagNegative), c.wantN)
			}
		})
	}
}

// TestSBCBinary covers SBC corner cases.
func TestSBCBinary(t *testing.T) {
	cases := []struct {
		name         string
		a, m         uint8
		cIn          bool // SBC: carry-in means "no borrow"
		wantA        uint8
		wantC, wantV bool
		wantZ, wantN bool
	}{
		{"5 - 3", 5, 3, true, 2, true, false, false, false},
		{"5 - 5 zero", 5, 5, true, 0, true, false, true, false},
		{"borrow propagates", 5, 5, false, 0xFF, false, false, false, true},
		{"-128 - 1 overflow", 0x80, 1, true, 0x7F, true, true, false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cpu, _ := newTestCPU(Registers{A: c.a})
			cpu.setFlag(FlagCarry, c.cIn)
			opSbc(cpu, c.m)
			if cpu.A != c.wantA {
				t.Errorf("A=$%02X want $%02X", cpu.A, c.wantA)
			}
			if cpu.flag(FlagCarry) != c.wantC {
				t.Errorf("C=%v want %v", cpu.flag(FlagCarry), c.wantC)
			}
			if cpu.flag(FlagOverflow) != c.wantV {
				t.Errorf("V=%v want %v", cpu.flag(FlagOverflow), c.wantV)
			}
			if cpu.flag(FlagZero) != c.wantZ {
				t.Errorf("Z=%v want %v", cpu.flag(FlagZero), c.wantZ)
			}
			if cpu.flag(FlagNegative) != c.wantN {
				t.Errorf("N=%v want %v", cpu.flag(FlagNegative), c.wantN)
			}
		})
	}
}

// TestADCBCD covers FR-016: BCD ADC with NMOS-style C from BCD result
// and Z from pure binary.
func TestADCBCD(t *testing.T) {
	cases := []struct {
		name  string
		a, m  uint8
		cIn   bool
		wantA uint8
		wantC bool
	}{
		{"05+05=10", 0x05, 0x05, false, 0x10, false},
		{"50+50=100 (carry)", 0x50, 0x50, false, 0x00, true},
		{"99+01=00 carry", 0x99, 0x01, false, 0x00, true},
		{"99+00+C=00 carry", 0x99, 0x00, true, 0x00, true},
		{"19+28=47", 0x19, 0x28, false, 0x47, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cpu, _ := newTestCPU(Registers{A: c.a, P: FlagDecimal})
			cpu.setFlag(FlagCarry, c.cIn)
			opAdc(cpu, c.m)
			if cpu.A != c.wantA {
				t.Errorf("A=$%02X want $%02X", cpu.A, c.wantA)
			}
			if cpu.flag(FlagCarry) != c.wantC {
				t.Errorf("C=%v want %v", cpu.flag(FlagCarry), c.wantC)
			}
		})
	}
}

// TestSBCBCD covers FR-016: BCD SBC.
func TestSBCBCD(t *testing.T) {
	cases := []struct {
		name  string
		a, m  uint8
		cIn   bool // SBC: C=1 means "no borrow"
		wantA uint8
		wantC bool
	}{
		{"50-25=25", 0x50, 0x25, true, 0x25, true},
		{"50-50=00", 0x50, 0x50, true, 0x00, true},
		{"00-01 borrow", 0x00, 0x01, true, 0x99, false},
		{"99-00=99", 0x99, 0x00, true, 0x99, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cpu, _ := newTestCPU(Registers{A: c.a, P: FlagDecimal})
			cpu.setFlag(FlagCarry, c.cIn)
			opSbc(cpu, c.m)
			if cpu.A != c.wantA {
				t.Errorf("A=$%02X want $%02X", cpu.A, c.wantA)
			}
			if cpu.flag(FlagCarry) != c.wantC {
				t.Errorf("C=%v want %v", cpu.flag(FlagCarry), c.wantC)
			}
		})
	}
}
