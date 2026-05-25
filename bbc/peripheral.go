package bbc

// Peripheral is implemented by every SHEILA stub. The MemoryMap
// decoder translates the bus address ($FE00–$FEFF) into the
// peripheral's local register offset before calling Read or Write;
// implementations do not re-mask.
//
// Implementations are deterministic, non-allocating, and
// single-goroutine (FR-026, FR-028). Phase 002 stubs never source
// IRQ or NMI signals (FR-021).
type Peripheral interface {
	// Read returns the byte at the given register offset.
	Read(reg uint8) uint8

	// Write stores value at the given register offset.
	Write(reg uint8, value uint8)
}

// Peripherals is the by-value container of every SHEILA stub on
// the Model B. The ROM-select latch ($FE30–$FE33) lives on
// RomBanks, not here; ACCCON ($FE34–$FE37) is included even
// though Model B treats it as unmapped so the decoder always has
// a destination.
type Peripherals struct {
	CRTC      CRTC
	ACIA      ACIA
	SerialULA SerialULA
	VideoULA  VideoULA
	ACCCON    ACCCON
	SystemVIA VIA
	UserVIA   VIA
	FDC       FDC
	Econet    Econet
	ADC       ADC
	Tube      Tube
}

// Zero clears every stub's register file. ColdReset (FR-019) calls
// Zero so the machine reaches power-on register state without
// rebuilding the Peripherals container.
func (p *Peripherals) Zero() {
	p.CRTC.Zero()
	p.ACIA.Zero()
	p.SerialULA.Zero()
	p.VideoULA.Zero()
	p.ACCCON.Zero()
	p.SystemVIA.Zero()
	p.UserVIA.Zero()
	p.FDC.Zero()
	p.Econet.Zero()
	p.ADC.Zero()
	p.Tube.Zero()
}
