package bbc

import "github.com/gobeeb/GoBeeb/mos6502"

// openBus is the byte returned for any read the BBC memory map
// classifies as unmapped. Real hardware values depend on bus
// state; $FF is the conventional emulator default.
const openBus uint8 = 0xFF

// Compile-time assertion that *MemoryMap satisfies mos6502.Memory.
var _ mos6502.Memory = (*MemoryMap)(nil)

// MemoryMap implements mos6502.Memory with the BBC Model B layout:
//
//	$0000–$7FFF   32 KB main RAM (read/write)
//	$8000–$BFFF   16 KB sideways ROM bank window (read-only; bank
//	              selected by the latch at $FE30–$FE33)
//	$C000–$FBFF   OS ROM body (read-only)
//	$FC00–$FCFF   FRED I/O page (unmapped reads return $FF)
//	$FD00–$FDFF   JIM I/O page (unmapped reads return $FF)
//	$FE00–$FEFF   SHEILA I/O page (decoded to stub peripherals)
//	$FF00–$FFFF   OS ROM vector region (read-only)
//
// MemoryMap is owned by Machine; the rom/periph/parent pointers are
// set once during New and never mutated. NOT goroutine-safe (FR-028).
type MemoryMap struct {
	rom    *RomBanks
	periph *Peripherals
	parent *Machine

	ram [0x8000]byte
}

// Read returns the byte at addr per the BBC Model B memory map.
// Reads of unmapped addresses return $FF and invoke the parent
// Machine's UnmappedAccessHook before returning (FR-002, FR-029).
func (m *MemoryMap) Read(addr uint16) uint8 {
	switch {
	case addr < 0x8000:
		return m.ram[addr]
	case addr < 0xC000:
		return m.sidewaysRead(addr)
	case addr < 0xFC00:
		return m.rom.os[addr-0xC000]
	case addr < 0xFD00:
		// FRED — Phase 002 has no claimants.
		m.fireUnmapped(addr, false, openBus)
		return openBus
	case addr < 0xFE00:
		// JIM — Phase 002 has no claimants.
		m.fireUnmapped(addr, false, openBus)
		return openBus
	case addr < 0xFF00:
		return m.ioRead(addr)
	default:
		// $FF00–$FFFF — OS ROM vector region.
		return m.rom.os[addr-0xC000]
	}
}

// Write stores value at addr per the BBC Model B memory map.
// Writes to read-only regions are silently dropped. Writes to
// unmapped FRED/JIM/SHEILA offsets fire the UnmappedAccessHook
// (FR-003, FR-029).
func (m *MemoryMap) Write(addr uint16, value uint8) {
	switch {
	case addr < 0x8000:
		m.ram[addr] = value
	case addr < 0xC000:
		// Sideways ROM is read-only; drop writes silently.
	case addr < 0xFC00:
		// OS ROM body — drop writes silently (FR-003).
	case addr < 0xFD00:
		// FRED — drop, but observe.
		m.fireUnmapped(addr, true, value)
	case addr < 0xFE00:
		// JIM — drop, but observe.
		m.fireUnmapped(addr, true, value)
	case addr < 0xFF00:
		m.ioWrite(addr, value)
	default:
		// $FF00–$FFFF — OS ROM vector region; drop writes.
	}
}

// sideways reads the byte at addr ($8000–$BFFF) from the
// currently-selected bank, or returns $FF and fires the unmapped
// hook if no image is loaded in that bank slot (FR-014).
func (m *MemoryMap) sidewaysRead(addr uint16) uint8 {
	bank := m.rom.bank
	if !m.rom.sidewaysLoaded[bank] {
		m.fireUnmapped(addr, false, openBus)
		return openBus
	}
	return m.rom.sideways[bank][addr-0x8000]
}

// ioRead dispatches a SHEILA read ($FE00–$FEFF) to the relevant
// peripheral per FR-008. The dispatch splits at $FE40 into low
// (CRTC..ACCCON gap) and high (VIAs..Tube) halves to keep the
// per-function cyclomatic complexity inside the project gate.
func (m *MemoryMap) ioRead(addr uint16) uint8 {
	low := uint8(addr)
	if low < 0x40 {
		return m.ioReadLow(addr, low)
	}
	return m.ioReadHigh(low)
}

// ioReadLow covers $FE00–$FE3F: CRTC, ACIA, Serial ULA, Video
// ULA, the ROM-select latch (write-only), ACCCON, and the gap
// at $FE38–$FE3F.
func (m *MemoryMap) ioReadLow(addr uint16, low uint8) uint8 {
	switch {
	case low < 0x08: // $FE00–$FE07: CRTC
		return m.periph.CRTC.Read(low & 0x01)
	case low < 0x10: // $FE08–$FE0F: ACIA
		return m.periph.ACIA.Read(low & 0x03)
	case low < 0x20: // $FE10–$FE1F: Serial ULA
		return m.periph.SerialULA.Read(0)
	case low < 0x30: // $FE20–$FE2F: Video ULA
		return m.periph.VideoULA.Read(low & 0x01)
	case low < 0x34: // $FE30–$FE33: ROM-select latch (write-only)
		m.fireUnmapped(addr, false, openBus)
		return openBus
	case low < 0x38: // $FE34–$FE37: ACCCON
		return m.periph.ACCCON.Read(low & 0x03)
	default: // $FE38–$FE3F: unmapped
		m.fireUnmapped(addr, false, openBus)
		return openBus
	}
}

// ioReadHigh covers $FE40–$FEFF: System VIA, User VIA, FDC,
// Econet, ADC, Tube.
func (m *MemoryMap) ioReadHigh(low uint8) uint8 {
	switch {
	case low < 0x60: // $FE40–$FE5F: System VIA (mirrors every 16)
		return m.periph.SystemVIA.Read(low & 0x0F)
	case low < 0x80: // $FE60–$FE7F: User VIA (mirrors every 16)
		return m.periph.UserVIA.Read(low & 0x0F)
	case low < 0xA0: // $FE80–$FE9F: FDC
		return m.periph.FDC.Read(low & 0x03)
	case low < 0xC0: // $FEA0–$FEBF: Econet
		return m.periph.Econet.Read(low & 0x03)
	case low < 0xE0: // $FEC0–$FEDF: ADC
		return m.periph.ADC.Read(low & 0x03)
	default: // $FEE0–$FEFF: Tube
		return m.periph.Tube.Read(low & 0x07)
	}
}

// ioWrite dispatches a SHEILA write ($FE00–$FEFF). Symmetric
// with ioRead — split into low/high halves at $FE40.
func (m *MemoryMap) ioWrite(addr uint16, value uint8) {
	low := uint8(addr)
	if low < 0x40 {
		m.ioWriteLow(addr, low, value)
		return
	}
	m.ioWriteHigh(low, value)
}

func (m *MemoryMap) ioWriteLow(addr uint16, low, value uint8) {
	switch {
	case low < 0x08:
		m.periph.CRTC.Write(low&0x01, value)
	case low < 0x10:
		m.periph.ACIA.Write(low&0x03, value)
	case low < 0x20:
		m.periph.SerialULA.Write(0, value)
	case low < 0x30:
		m.periph.VideoULA.Write(low&0x01, value)
	case low < 0x34:
		m.writeRomSelect(value)
	case low < 0x38:
		m.periph.ACCCON.Write(low&0x03, value)
	default: // $FE38–$FE3F unmapped
		m.fireUnmapped(addr, true, value)
	}
}

func (m *MemoryMap) ioWriteHigh(low, value uint8) {
	switch {
	case low < 0x60:
		m.periph.SystemVIA.Write(low&0x0F, value)
	case low < 0x80:
		m.periph.UserVIA.Write(low&0x0F, value)
	case low < 0xA0:
		m.periph.FDC.Write(low&0x03, value)
	case low < 0xC0:
		m.periph.Econet.Write(low&0x03, value)
	case low < 0xE0:
		m.periph.ADC.Write(low&0x03, value)
	default:
		m.periph.Tube.Write(low&0x07, value)
	}
}
