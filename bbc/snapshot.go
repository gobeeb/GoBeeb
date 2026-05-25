package bbc

import "github.com/gobeeb/GoBeeb/mos6502"

// Snapshot captures the complete observable state of a Machine
// for in-process round-trip via Restore. ROM image bytes are NOT
// included (FR-024); the consumer of Restore re-loads the same
// images on the destination Machine before calling.
type Snapshot struct {
	// CPU is the architectural register state at snapshot time.
	CPU mos6502.Registers

	// RAM is a by-value copy of the 32 KB main RAM ($0000–$7FFF).
	RAM [0x8000]byte

	// SidewaysBank is the currently-selected sideways bank
	// index (0..3) at snapshot time.
	SidewaysBank uint8

	// SidewaysLoaded[i] is true iff bank slot i had a ROM image
	// loaded at snapshot time. Restore returns ErrRestoreMismatch
	// if the destination Machine has a different loaded set.
	SidewaysLoaded [4]bool

	// Peripherals captures every SHEILA stub's register file.
	Peripherals PeripheralSnapshot
}

// PeripheralSnapshot is the by-value capture of every SHEILA stub.
type PeripheralSnapshot struct {
	CRTC      CRTCSnapshot
	ACIA      ACIASnapshot
	SerialULA SerialULASnapshot
	VideoULA  VideoULASnapshot
	SystemVIA VIASnapshot
	UserVIA   VIASnapshot
	FDC       FDCSnapshot
	ADC       ADCSnapshot
	Tube      TubeSnapshot
	Econet    EconetSnapshot
}

// CRTCSnapshot is the captured state of the 6845 CRTC stub.
type CRTCSnapshot struct {
	Regs     [crtcRegCount]byte
	Selected uint8
}

// VIASnapshot is the captured register file of a 6522 VIA stub.
type VIASnapshot struct {
	Regs [16]byte
}

// ACIASnapshot is the captured register file of the 6850 ACIA stub.
type ACIASnapshot struct {
	Regs [4]byte
}

// SerialULASnapshot is the captured register state of the serial
// ULA stub.
type SerialULASnapshot struct {
	Regs [1]byte
}

// VideoULASnapshot is the captured register state of the video
// ULA stub.
type VideoULASnapshot struct {
	Regs [2]byte
}

// FDCSnapshot is the captured register state of the 1770 FDC stub.
type FDCSnapshot struct {
	Regs [4]byte
}

// ADCSnapshot is the captured register state of the µPD7002 ADC stub.
type ADCSnapshot struct {
	Regs [4]byte
}

// TubeSnapshot is the captured register state of the Tube stub.
type TubeSnapshot struct {
	Regs [8]byte
}

// EconetSnapshot is the captured register state of the ADLC /
// Econet stub.
type EconetSnapshot struct {
	Regs [4]byte
}

// Snapshot captures the complete observable state of the Machine
// (CPU registers, main RAM, sideways bank index + loaded-flags,
// every peripheral's register file). ROM image bytes are NOT
// included (FR-024); callers are responsible for re-loading the
// same images on the destination Machine before Restore.
func (m *Machine) Snapshot() Snapshot {
	return Snapshot{
		CPU:            m.cpu.Registers(),
		RAM:            m.mmap.ram,
		SidewaysBank:   m.rom.bank,
		SidewaysLoaded: m.rom.sidewaysLoaded,
		Peripherals: PeripheralSnapshot{
			CRTC:      m.periph.CRTC.Snapshot(),
			ACIA:      m.periph.ACIA.Snapshot(),
			SerialULA: m.periph.SerialULA.Snapshot(),
			VideoULA:  m.periph.VideoULA.Snapshot(),
			SystemVIA: m.periph.SystemVIA.Snapshot(),
			UserVIA:   m.periph.UserVIA.Snapshot(),
			FDC:       m.periph.FDC.Snapshot(),
			ADC:       m.periph.ADC.Snapshot(),
			Tube:      m.periph.Tube.Snapshot(),
			Econet:    m.periph.Econet.Snapshot(),
		},
	}
}

// Restore re-populates Machine state from snap. The Machine MUST
// already have an OS ROM loaded; for any sideways bank where
// snap.SidewaysLoaded[i] is true, the Machine MUST also have a
// sideways ROM loaded in that bank.
//
// Returns ErrNoOSROM if no OS ROM has been loaded;
// ErrRestoreMismatch if the loaded-bank set differs.
//
// On success, subsequent Tick calls produce bit-identical
// execution to a Machine that had run the original sequence
// directly.
func (m *Machine) Restore(snap Snapshot) error {
	if !m.rom.osLoaded {
		return ErrNoOSROM
	}
	if snap.SidewaysLoaded != m.rom.sidewaysLoaded {
		return ErrRestoreMismatch
	}
	m.cpu.SetRegisters(snap.CPU)
	m.mmap.ram = snap.RAM
	m.rom.bank = snap.SidewaysBank & 0x03
	m.periph.CRTC.Restore(snap.Peripherals.CRTC)
	m.periph.ACIA.Restore(snap.Peripherals.ACIA)
	m.periph.SerialULA.Restore(snap.Peripherals.SerialULA)
	m.periph.VideoULA.Restore(snap.Peripherals.VideoULA)
	m.periph.SystemVIA.Restore(snap.Peripherals.SystemVIA)
	m.periph.UserVIA.Restore(snap.Peripherals.UserVIA)
	m.periph.FDC.Restore(snap.Peripherals.FDC)
	m.periph.ADC.Restore(snap.Peripherals.ADC)
	m.periph.Tube.Restore(snap.Peripherals.Tube)
	m.periph.Econet.Restore(snap.Peripherals.Econet)
	return nil
}
