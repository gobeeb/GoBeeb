package bbc

import "testing"

// sheilaCase asserts the routing for a single SHEILA address.
type sheilaCase struct {
	name      string
	addr      uint16
	roundTrip bool // true if write(addr,v) then read(addr) returns v
	wantRead  uint8
	writeOnly bool // if true, after write the read returns $FF
}

func TestSHEILARouting(t *testing.T) {
	cases := []sheilaCase{
		// CRTC: $FE00 write-only (index), $FE01 round-trip (data after index=1)
		{"CRTC index $FE00 (write-only)", 0xFE00, false, 0xFF, true},
		// ACIA $FE08–$FE0F
		{"ACIA $FE08", 0xFE08, true, 0, false},
		{"ACIA $FE0F", 0xFE0F, true, 0, false},
		// SerialULA $FE10–$FE1F mirror to a single register
		{"SerialULA $FE10", 0xFE10, true, 0, false},
		{"SerialULA $FE1F", 0xFE1F, true, 0, false},
		// VideoULA $FE20–$FE2F (stride 2)
		{"VideoULA $FE20", 0xFE20, true, 0, false},
		{"VideoULA $FE21", 0xFE21, true, 0, false},
		// ROM-select $FE30–$FE33: write-only, reads open-bus
		{"ROM-select $FE30 (write-only)", 0xFE30, false, 0xFF, true},
		{"ROM-select $FE33 (write-only)", 0xFE33, false, 0xFF, true},
		// ACCCON $FE34–$FE37: always $FF on Model B
		{"ACCCON $FE34 (Model B)", 0xFE34, false, 0xFF, true},
		// Unmapped gap $FE38–$FE3F
		{"Unmapped $FE38", 0xFE38, false, 0xFF, true},
		{"Unmapped $FE3F", 0xFE3F, false, 0xFF, true},
		// System VIA $FE40–$FE5F (mirrors every 16)
		{"SystemVIA $FE40", 0xFE40, true, 0, false},
		{"SystemVIA $FE4F", 0xFE4F, true, 0, false},
		// User VIA $FE60–$FE7F
		{"UserVIA $FE60", 0xFE60, true, 0, false},
		{"UserVIA $FE6F", 0xFE6F, true, 0, false},
		// FDC $FE80–$FE9F: writes accepted, reads return $FF
		{"FDC $FE80 (Phase 002 $FF)", 0xFE80, false, 0xFF, true},
		// Econet $FEA0–$FEBF: same
		{"Econet $FEA0 (Phase 002 $FF)", 0xFEA0, false, 0xFF, true},
		// ADC $FEC0–$FEDF
		{"ADC $FEC0", 0xFEC0, true, 0, false},
		// Tube $FEE0–$FEFF
		{"Tube $FEE0", 0xFEE0, true, 0, false},
		{"Tube $FEFF", 0xFEFF, true, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newWithStubOS(t)
			const probe uint8 = 0xA5

			// For the CRTC data register at $FE01 the write
			// must be preceded by an index write so the data
			// lands in a deterministic R0..R17 slot. Address
			// $FE00 itself is write-only (index register).
			m.mmap.Write(tc.addr, probe)
			got := m.mmap.Read(tc.addr)
			if tc.roundTrip {
				if got != probe {
					t.Fatalf("round-trip $%04X: write $%02X, read got $%02X", tc.addr, probe, got)
				}
			} else if tc.writeOnly {
				if got != tc.wantRead {
					t.Fatalf("write-only $%04X: want read $%02X, got $%02X", tc.addr, tc.wantRead, got)
				}
			}
		})
	}
}

// TestSHEILA_SystemVIAMirror confirms FR-008's "mirrors every 16
// bytes" semantics for the System VIA window.
func TestSHEILA_SystemVIAMirror(t *testing.T) {
	m := newWithStubOS(t)
	m.mmap.Write(0xFE40, 0x5A)
	if got := m.mmap.Read(0xFE50); got != 0x5A {
		t.Fatalf("System VIA mirror $FE50 = $%02X, want $5A", got)
	}
	// Confirm a different reg offset doesn't alias to 0.
	m.mmap.Write(0xFE45, 0xC3)
	if got := m.mmap.Read(0xFE55); got != 0xC3 {
		t.Fatalf("System VIA mirror $FE55 = $%02X, want $C3", got)
	}
}

func TestSHEILA_UserVIAMirror(t *testing.T) {
	m := newWithStubOS(t)
	m.mmap.Write(0xFE60, 0x11)
	if got := m.mmap.Read(0xFE70); got != 0x11 {
		t.Fatalf("User VIA mirror $FE70 = $%02X, want $11", got)
	}
}

// TestFREDJIMUnmapped covers FR-008's documented FRED/JIM page
// behaviour: reads return $FF, writes silently drop, both fire
// the unmapped hook with the correct addr/write/value.
func TestFREDJIMUnmapped(t *testing.T) {
	probes := []uint16{0xFC00, 0xFC7F, 0xFCFF, 0xFD00, 0xFD80, 0xFDFF}
	for _, addr := range probes {
		t.Run("addr", func(t *testing.T) {
			m := newWithStubOS(t)
			events := captureUnmapped(m)

			if got := m.mmap.Read(addr); got != 0xFF {
				t.Fatalf("read $%04X = $%02X, want $FF", addr, got)
			}
			m.mmap.Write(addr, 0x42)

			if len(*events) != 2 {
				t.Fatalf("got %d hook events, want 2: %+v", len(*events), *events)
			}
			if (*events)[0].addr != addr || (*events)[0].write {
				t.Fatalf("read event: %+v", (*events)[0])
			}
			if (*events)[1].addr != addr || !(*events)[1].write || (*events)[1].value != 0x42 {
				t.Fatalf("write event: %+v", (*events)[1])
			}
		})
	}
}
