package bbc

import "testing"

// Acceptance scenario 1: bank 0 read.
func TestSideways_Bank0Read(t *testing.T) {
	m := newWithStubOS(t)
	if err := m.LoadSidewaysROM(0, stubSidewaysAA); err != nil {
		t.Fatalf("LoadSidewaysROM(0, AA): %v", err)
	}
	// Default bank after construction is 0.
	if got := m.mmap.Read(0x8000); got != 0xAA {
		t.Fatalf("read $8000 bank 0 = $%02X, want $AA", got)
	}
}

// Acceptance scenario 2: bank switch read.
func TestSideways_BankSwitchRead(t *testing.T) {
	m := newWithStubOS(t)
	_ = m.LoadSidewaysROM(0, stubSidewaysAA)
	_ = m.LoadSidewaysROM(1, stubSideways55)

	m.mmap.Write(0xFE30, 0x01)
	if got := m.mmap.Read(0x8000); got != 0x55 {
		t.Fatalf("read $8000 bank 1 = $%02X, want $55", got)
	}
}

// Acceptance scenario 3: swap-back.
func TestSideways_SwapBack(t *testing.T) {
	m := newWithStubOS(t)
	_ = m.LoadSidewaysROM(0, stubSidewaysAA)
	_ = m.LoadSidewaysROM(1, stubSideways55)

	m.mmap.Write(0xFE30, 0x01)
	_ = m.mmap.Read(0x8000)
	m.mmap.Write(0xFE30, 0x00)
	if got := m.mmap.Read(0x8000); got != 0xAA {
		t.Fatalf("read $8000 after swap-back = $%02X, want $AA", got)
	}
}

// Acceptance scenario 4: empty-bank reads return $FF and fire
// the unmapped hook.
func TestSideways_EmptyBankReturnsOpenBus(t *testing.T) {
	m := newWithStubOS(t)
	events := captureUnmapped(m)

	// Bank 2 has no image loaded.
	m.mmap.Write(0xFE30, 0x02)
	for _, addr := range []uint16{0x8000, 0x9234, 0xBFFF} {
		if got := m.mmap.Read(addr); got != 0xFF {
			t.Fatalf("empty bank read $%04X = $%02X, want $FF", addr, got)
		}
	}
	if len(*events) != 3 {
		t.Fatalf("expected 3 unmapped events for 3 empty-bank reads, got %d", len(*events))
	}
}

// Acceptance scenario 5: writes to $8000–$BFFF silently dropped.
func TestSideways_WritesDrop(t *testing.T) {
	m := newWithStubOS(t)
	_ = m.LoadSidewaysROM(0, stubSidewaysAA)

	m.mmap.Write(0x8000, 0xCC)
	if got := m.mmap.Read(0x8000); got != 0xAA {
		t.Fatalf("write to sideways must drop; read = $%02X, want $AA", got)
	}
	// Also assert writes don't reach the underlying buffer.
	if m.rom.sideways[0][0] != 0xAA {
		t.Fatal("sideways buffer mutated by write — expected silent drop")
	}
}

// SC-004 in full: ≥ 10 alternating bank-swap round trips with no
// mismatches.
func TestSideways_RoundTripStress(t *testing.T) {
	m := newWithStubOS(t)
	_ = m.LoadSidewaysROM(0, stubSidewaysAA)
	_ = m.LoadSidewaysROM(1, stubSideways55)

	for i := 0; i < 25; i++ {
		want := byte(0xAA)
		bank := byte(0)
		if i%2 == 1 {
			want = 0x55
			bank = 1
		}
		m.mmap.Write(0xFE30, bank)
		if got := m.mmap.Read(0x8000); got != want {
			t.Fatalf("iter %d bank %d: got $%02X, want $%02X", i, bank, got, want)
		}
	}
}

// FR-013: high bits of the $FE30 write are masked to the 2-bit
// bank-index width.
func TestSideways_BankMaskedTo2Bits(t *testing.T) {
	m := newWithStubOS(t)
	_ = m.LoadSidewaysROM(1, stubSideways55)

	// $FF & 0x03 == 3. We didn't load bank 3, so reads return
	// $FF and fire the unmapped hook. The important assertion
	// is that the latch stored 3, not 0xFF.
	m.mmap.Write(0xFE30, 0xFF)
	if m.rom.bank != 0x03 {
		t.Fatalf("bank latch = %d, want 3 (masked low 2 bits)", m.rom.bank)
	}

	// $F5 & 0x03 == 1. Bank 1 is loaded with $55.
	m.mmap.Write(0xFE30, 0xF5)
	if got := m.mmap.Read(0x8000); got != 0x55 {
		t.Fatalf("after masked write 0xF5: read $8000 = $%02X, want $55", got)
	}
}
