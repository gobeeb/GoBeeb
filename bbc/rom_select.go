package bbc

// writeRomSelect handles writes to the ROM-select latch at
// $FE30–$FE33 (FR-013). The low two bits select one of the four
// sideways banks; the high bits are masked off (matches Model B's
// 2-bit latch width).
func (m *MemoryMap) writeRomSelect(value uint8) {
	m.rom.bank = value & 0x03
}
