package bbc

import "testing"

// FR-010 acceptance:
//   - write index → write data → read data returns stored byte
//   - write index ≥ 18 → masked to 0..17
//   - read of $FE00 returns $FF (write-only)
func TestCRTC_IndexThenData(t *testing.T) {
	m := newWithStubOS(t)

	// Select R5 and write $5A.
	m.mmap.Write(0xFE00, 0x05)
	m.mmap.Write(0xFE01, 0x5A)
	if got := m.mmap.Read(0xFE01); got != 0x5A {
		t.Fatalf("R5 read = $%02X, want $5A", got)
	}

	// Switch to R10 and write $C3.
	m.mmap.Write(0xFE00, 0x0A)
	m.mmap.Write(0xFE01, 0xC3)
	if got := m.mmap.Read(0xFE01); got != 0xC3 {
		t.Fatalf("R10 read = $%02X, want $C3", got)
	}

	// Switch back to R5 and confirm it still has $5A.
	m.mmap.Write(0xFE00, 0x05)
	if got := m.mmap.Read(0xFE01); got != 0x5A {
		t.Fatalf("R5 after switch = $%02X, want $5A (no clobber)", got)
	}
}

func TestCRTC_IndexMaskedToValidRange(t *testing.T) {
	m := newWithStubOS(t)
	// Index 25 (decimal) is out of range; real 6845 wraps via
	// the low bits. 25 & 0x1F = 25; 25 % 18 = 7.
	m.mmap.Write(0xFE00, 25)
	m.mmap.Write(0xFE01, 0xAB)
	// Read R7 directly via the masked index.
	m.mmap.Write(0xFE00, 7)
	if got := m.mmap.Read(0xFE01); got != 0xAB {
		t.Fatalf("R7 = $%02X after index=25 write of $AB; want $AB (index should have masked to 7)", got)
	}
}

func TestCRTC_IndexReadIsOpenBus(t *testing.T) {
	m := newWithStubOS(t)
	m.mmap.Write(0xFE00, 0x05)
	if got := m.mmap.Read(0xFE00); got != 0xFF {
		t.Fatalf("read $FE00 = $%02X, want $FF (write-only address register)", got)
	}
}
