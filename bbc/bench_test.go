package bbc

import "testing"

// BenchmarkTickNoop measures the per-cycle overhead of the BBC
// machine layer against an OS ROM whose body is one big NOP loop.
// Asserts zero allocations on the hot path (SC-006).
func BenchmarkTickNoop(b *testing.B) {
	m := New()
	if err := m.LoadOSROM(stubOSROM); err != nil {
		b.Fatalf("LoadOSROM: %v", err)
	}
	if err := m.Reset(); err != nil {
		b.Fatalf("Reset: %v", err)
	}
	// Warm-up so the RESET sequence isn't part of the timed slice.
	m.Tick(1024)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Tick(1024)
	}
}

// BenchmarkTickMixedWorkload measures a synthetic ROM that
// exercises RAM, OS ROM, sideways ROM, and a SHEILA write per
// iteration. Gives the more realistic ns/emulated-cycle number
// for SC-006.
func BenchmarkTickMixedWorkload(b *testing.B) {
	// $C000: LDA $1000      ; AD 00 10  (RAM read)
	//        STA $FE40      ; 8D 40 FE  (SHEILA write — System VIA)
	//        LDA $8000      ; AD 00 80  (sideways ROM read)
	//        INC $1000      ; EE 00 10  (RAM RMW)
	//        JMP $C000      ; 4C 00 C0
	prog := []byte{
		0xAD, 0x00, 0x10,
		0x8D, 0x40, 0xFE,
		0xAD, 0x00, 0x80,
		0xEE, 0x00, 0x10,
		0x4C, 0x00, 0xC0,
	}
	m := New()
	if err := m.LoadOSROM(buildStubOSROM(prog)); err != nil {
		b.Fatalf("LoadOSROM: %v", err)
	}
	if err := m.LoadSidewaysROM(0, stubSidewaysAA); err != nil {
		b.Fatalf("LoadSidewaysROM: %v", err)
	}
	if err := m.Reset(); err != nil {
		b.Fatalf("Reset: %v", err)
	}
	m.Tick(1024)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Tick(1024)
	}
}
