package bbc

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/gobeeb/GoBeeb/mos6502"
)

// goldenTraceCapacity is the number of bus cycles captured by the
// reset-fixture test (SC-002).
const goldenTraceCapacity = 256

// encodeTrace serialises a slice of BusEvents to a deterministic
// byte form: 8 bytes per event (cycle uint32 BE, addr uint16 BE,
// value uint8, kind uint8). Used by both the comparison test and
// the regeneration path.
func encodeTrace(events []mos6502.BusEvent) []byte {
	out := make([]byte, 0, 8*len(events))
	var scratch [8]byte
	for _, ev := range events {
		binary.BigEndian.PutUint32(scratch[0:4], uint32(ev.Cycle))
		binary.BigEndian.PutUint16(scratch[4:6], ev.Addr)
		scratch[6] = ev.Value
		scratch[7] = uint8(ev.Kind)
		out = append(out, scratch[:]...)
	}
	return out
}

// captureFirstN runs the machine after a Reset and returns the
// first n bus events.
func captureFirstN(t *testing.T, m *Machine, n int) []mos6502.BusEvent {
	t.Helper()
	tr := mos6502.NewTrace(n)
	m.CPU().SetTrace(tr)
	if err := m.Reset(); err != nil {
		t.Fatalf("Reset: %v", err)
	}
	for tr.Len() < n {
		m.Tick(1)
	}
	events := tr.Snapshot()
	if len(events) > n {
		events = events[:n]
	}
	return events
}

// compareOrRegen writes encoded into testdata/golden_traces/name
// when BBC_REGEN_GOLDEN=1; otherwise reads the fixture and
// fails if encoded doesn't match.
func compareOrRegen(t *testing.T, name string, encoded []byte) {
	t.Helper()
	fixture := filepath.Join("testdata", "golden_traces", name)
	if os.Getenv("BBC_REGEN_GOLDEN") == "1" {
		if err := os.WriteFile(fixture, encoded, 0o644); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
		t.Logf("regenerated %s (%d bytes)", fixture, len(encoded))
		return
	}
	want, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("read fixture %q: %v (set BBC_REGEN_GOLDEN=1 to regenerate)", fixture, err)
	}
	if !bytes.Equal(encoded, want) {
		t.Fatalf("trace does not match %s; rerun with BBC_REGEN_GOLDEN=1 to regenerate after intentional changes", fixture)
	}
}

// assertResetVectorFetched scans events for $FFFC/$FFFD vector
// reads and the first instruction fetch from the OS ROM region
// that follows (acceptance scenario 2 of US1).
func assertResetVectorFetched(t *testing.T, events []mos6502.BusEvent) {
	t.Helper()
	var sawVectorLo, sawVectorHi bool
	var firstFetchAfterVector uint16
	for i, ev := range events {
		switch ev.Addr {
		case 0xFFFC:
			sawVectorLo = true
		case 0xFFFD:
			sawVectorHi = true
		}
		if sawVectorHi && firstFetchAfterVector == 0 && i+1 < len(events) {
			firstFetchAfterVector = events[i+1].Addr
			break
		}
	}
	if !sawVectorLo || !sawVectorHi {
		t.Errorf("RESET trace missing vector fetches: lo=%v hi=%v", sawVectorLo, sawVectorHi)
	}
	if firstFetchAfterVector < 0xC000 {
		t.Errorf("first fetch after vector load was $%04X; expected OS ROM region", firstFetchAfterVector)
	}
}

// TestGoldenTrace_ResetFirst256 captures the first 256 bus cycles
// after Reset and locks the trace + asserts SC-002 (every access
// in the documented BBC map; vector fetched from OS ROM).
func TestGoldenTrace_ResetFirst256(t *testing.T) {
	m := newWithStubOS(t)
	events := captureFirstN(t, m, goldenTraceCapacity)

	for i, ev := range events {
		if !inMappedRegion(ev.Addr) {
			t.Errorf("event %d at $%04X is in unmapped region", i, ev.Addr)
		}
	}
	assertResetVectorFetched(t, events)
	compareOrRegen(t, "reset_first256.trace", encodeTrace(events))
}

// inMappedRegion is true for every address that the BBC Model B
// memory map serves with deterministic byte storage (RAM or OS
// ROM body/vector region). FRED/JIM/SHEILA addresses are also
// "in the BBC map" but in a clean stub-OS boot they should NOT
// appear.
func inMappedRegion(addr uint16) bool {
	switch {
	case addr < 0x8000: // RAM
		return true
	case addr >= 0xC000 && addr < 0xFC00: // OS ROM body
		return true
	case addr >= 0xFF00: // OS ROM vector region
		return true
	}
	return false
}

// buildStubOSROM crafts a 16 KB OS ROM image whose body starts
// with the supplied 6502 byte sequence at $C000 and whose RESET,
// NMI, and IRQ vectors all point at $C000.
func buildStubOSROM(prog []byte) []byte {
	img := make([]byte, romImageSize)
	for i := range img {
		img[i] = 0xEA // NOP
	}
	copy(img, prog)
	for _, off := range []int{0x3FFA, 0x3FFC, 0x3FFE} {
		img[off] = 0x00
		img[off+1] = 0xC0
	}
	return img
}

// assertInExpectedRegions fails for any event addr outside RAM,
// OS ROM, SHEILA, or the sideways window.
func assertInExpectedRegions(t *testing.T, events []mos6502.BusEvent) {
	t.Helper()
	for i, ev := range events {
		mapped := inMappedRegion(ev.Addr) ||
			inSHEILA(ev.Addr) ||
			(ev.Addr >= 0x8000 && ev.Addr < 0xC000)
		if !mapped {
			t.Errorf("event %d $%04X outside expected regions", i, ev.Addr)
		}
	}
}

// captureGoldenTrace runs a stub-OS machine through n bus cycles
// and compares-or-regens the result against the named fixture.
func captureGoldenTrace(t *testing.T, name string, prog []byte, n int) {
	t.Helper()
	m := New()
	if err := m.LoadOSROM(buildStubOSROM(prog)); err != nil {
		t.Fatalf("LoadOSROM: %v", err)
	}
	events := captureFirstN(t, m, n)
	assertInExpectedRegions(t, events)
	compareOrRegen(t, name, encodeTrace(events))
}

func inSHEILA(addr uint16) bool { return addr >= 0xFE00 && addr < 0xFF00 }

// TestGoldenTrace_CRTCIndexThenData scripts the canonical FR-010
// sequence (select R5, store $5A to data port, read back) and
// locks the bus trace.
func TestGoldenTrace_CRTCIndexThenData(t *testing.T) {
	prog := []byte{
		0xA9, 0x05, // LDA #$05
		0x8D, 0x00, 0xFE, // STA $FE00
		0xA9, 0x5A, // LDA #$5A
		0x8D, 0x01, 0xFE, // STA $FE01
		0xAD, 0x01, 0xFE, // LDA $FE01
		0x4C, 0x00, 0xC0, // JMP $C000
	}
	captureGoldenTrace(t, "crtc_index_then_data.trace", prog, 128)
}

// TestGoldenTrace_VIARegisterRoundTrip scripts a write + read at
// $FE40 (System VIA reg 0) plus a mirror access at $FE50.
func TestGoldenTrace_VIARegisterRoundTrip(t *testing.T) {
	prog := []byte{
		0xA9, 0xA5,
		0x8D, 0x40, 0xFE,
		0xAD, 0x40, 0xFE,
		0x8D, 0x50, 0xFE,
		0xAD, 0x50, 0xFE,
		0x4C, 0x00, 0xC0,
	}
	captureGoldenTrace(t, "via_register_round_trip.trace", prog, 128)
}

// TestGoldenTrace_ROMSelectSwap scripts US3 paging: write bank 0
// to $FE30, read $8000; write bank 1, read $8000.
func TestGoldenTrace_ROMSelectSwap(t *testing.T) {
	prog := []byte{
		0xA9, 0x00,
		0x8D, 0x30, 0xFE,
		0xAD, 0x00, 0x80,
		0xA9, 0x01,
		0x8D, 0x30, 0xFE,
		0xAD, 0x00, 0x80,
		0x4C, 0x00, 0xC0,
	}
	m := New()
	if err := m.LoadOSROM(buildStubOSROM(prog)); err != nil {
		t.Fatalf("LoadOSROM: %v", err)
	}
	if err := m.LoadSidewaysROM(0, stubSidewaysAA); err != nil {
		t.Fatalf("LoadSidewaysROM 0: %v", err)
	}
	if err := m.LoadSidewaysROM(1, stubSideways55); err != nil {
		t.Fatalf("LoadSidewaysROM 1: %v", err)
	}

	events := captureFirstN(t, m, 128)
	assertInExpectedRegions(t, events)
	compareOrRegen(t, "rom_select_swap.trace", encodeTrace(events))
}
