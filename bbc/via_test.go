package bbc

import "testing"

// FR-008: System VIA $FE40–$FE5F and User VIA $FE60–$FE7F each
// mirror their 16-register file every 16 bytes.
func TestVIA_SystemRoundTrip(t *testing.T) {
	m := newWithStubOS(t)
	// Write every register, then read back from the mirrored
	// half of the window to assert both addressing and storage.
	for reg := uint8(0); reg < 16; reg++ {
		m.mmap.Write(0xFE40+uint16(reg), 0x10+reg)
	}
	for reg := uint8(0); reg < 16; reg++ {
		got := m.mmap.Read(0xFE50 + uint16(reg))
		want := 0x10 + reg
		if got != want {
			t.Errorf("System VIA reg $%02X: got $%02X, want $%02X", reg, got, want)
		}
	}
}

func TestVIA_UserRoundTrip(t *testing.T) {
	m := newWithStubOS(t)
	for reg := uint8(0); reg < 16; reg++ {
		m.mmap.Write(0xFE60+uint16(reg), 0xA0+reg)
	}
	for reg := uint8(0); reg < 16; reg++ {
		got := m.mmap.Read(0xFE70 + uint16(reg))
		want := 0xA0 + reg
		if got != want {
			t.Errorf("User VIA reg $%02X: got $%02X, want $%02X", reg, got, want)
		}
	}
}

// System VIA and User VIA are distinct register files; writing to
// one MUST NOT change the other (decoder regression guard).
func TestVIA_SystemAndUserAreIndependent(t *testing.T) {
	m := newWithStubOS(t)
	m.mmap.Write(0xFE40, 0xAA)
	m.mmap.Write(0xFE60, 0x55)
	if got := m.mmap.Read(0xFE40); got != 0xAA {
		t.Fatalf("System VIA reg 0 = $%02X, want $AA", got)
	}
	if got := m.mmap.Read(0xFE60); got != 0x55 {
		t.Fatalf("User VIA reg 0 = $%02X, want $55", got)
	}
}
