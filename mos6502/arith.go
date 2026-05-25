package mos6502

// ADC / SBC — binary and NMOS-faithful BCD (FR-016).
//
// On NMOS in decimal mode, the C flag is BCD-correct but the N, V, and
// Z flags are derived from the binary intermediate result (or, for N
// and V on ADC, from the partially-corrected intermediate). This is
// what the Klaus Dormann 6502_decimal_test verifies.

func opAdc(c *CPU, m uint8) {
	if c.flag(FlagDecimal) {
		adcBCD(c, m)
		return
	}
	cin := uint16(0)
	if c.flag(FlagCarry) {
		cin = 1
	}
	sum := uint16(c.A) + uint16(m) + cin
	result := uint8(sum)
	c.setFlag(FlagCarry, sum > 0xFF)
	c.setFlag(FlagOverflow, (c.A^result)&(m^result)&0x80 != 0)
	c.A = result
	c.setNZ(c.A)
}

func opSbc(c *CPU, m uint8) {
	if c.flag(FlagDecimal) {
		sbcBCD(c, m)
		return
	}
	// SBC = A + ~M + C
	cin := uint16(0)
	if c.flag(FlagCarry) {
		cin = 1
	}
	m2 := ^m
	sum := uint16(c.A) + uint16(m2) + cin
	result := uint8(sum)
	c.setFlag(FlagCarry, sum > 0xFF)
	c.setFlag(FlagOverflow, (c.A^result)&(m2^result)&0x80 != 0)
	c.A = result
	c.setNZ(c.A)
}

// adcBCD implements NMOS decimal-mode ADC per Bruce Clark. N, V are
// computed from the partially-corrected high-nibble intermediate; Z is
// computed from the pure binary result; C is BCD-correct.
func adcBCD(c *CPU, m uint8) {
	cin := uint8(0)
	if c.flag(FlagCarry) {
		cin = 1
	}
	// Z derives from pure binary result.
	binResult := c.A + m + cin
	c.setFlag(FlagZero, binResult == 0)

	// Low nibble with BCD correction.
	al := uint16(c.A&0x0F) + uint16(m&0x0F) + uint16(cin)
	if al >= 0x0A {
		al = ((al + 0x06) & 0x0F) + 0x10
	}
	// Intermediate high nibble (uncorrected).
	ah := uint16(c.A&0xF0) + uint16(m&0xF0) + al

	// N from bit 7 of intermediate.
	c.setFlag(FlagNegative, ah&0x80 != 0)
	// V from intermediate as if it were binary.
	aInt := uint8(ah)
	c.setFlag(FlagOverflow, (c.A^aInt)&(m^aInt)&0x80 != 0)

	// High-nibble BCD correction.
	if ah >= 0xA0 {
		ah += 0x60
	}
	c.setFlag(FlagCarry, ah >= 0x100)
	c.A = uint8(ah)
}

// sbcBCD implements NMOS decimal-mode SBC. All four flags (N, V, Z, C)
// are computed from the binary intermediate (A + ~M + C); only A is
// BCD-corrected.
func sbcBCD(c *CPU, m uint8) {
	cin := uint16(0)
	if c.flag(FlagCarry) {
		cin = 1
	}

	// Binary intermediate first — flags come from this.
	m2 := ^m
	binSum := uint16(c.A) + uint16(m2) + cin
	binResult := uint8(binSum)
	c.setFlag(FlagZero, binResult == 0)
	c.setFlag(FlagNegative, binResult&0x80 != 0)
	c.setFlag(FlagOverflow, (c.A^binResult)&(m2^binResult)&0x80 != 0)
	c.setFlag(FlagCarry, binSum > 0xFF)

	// BCD result for A.
	// AL = (A & 0x0F) - (M & 0x0F) - (1 - C)
	al := int16(c.A&0x0F) - int16(m&0x0F) - int16(1-uint8(cin))
	if al < 0 {
		al = ((al - 0x06) & 0x0F) - 0x10
	}
	aInt := int16(c.A&0xF0) - int16(m&0xF0) + al
	if aInt < 0 {
		aInt -= 0x60
	}
	c.A = uint8(aInt)
}

// ADC opcode handlers
func adcImm(c *CPU)  { opAdc(c, c.fetch()) }
func adcZp(c *CPU)   { opAdc(c, c.read(c.effZP())) }
func adcZpX(c *CPU)  { opAdc(c, c.read(c.effZPX())) }
func adcAbs(c *CPU)  { opAdc(c, c.read(c.effAbs())) }
func adcAbsX(c *CPU) { opAdc(c, c.read(c.effAbsX(false))) }
func adcAbsY(c *CPU) { opAdc(c, c.read(c.effAbsY(false))) }
func adcIndX(c *CPU) { opAdc(c, c.read(c.effIndexedIndirect())) }
func adcIndY(c *CPU) { opAdc(c, c.read(c.effIndirectIndexed(false))) }

// SBC opcode handlers
func sbcImm(c *CPU)  { opSbc(c, c.fetch()) }
func sbcZp(c *CPU)   { opSbc(c, c.read(c.effZP())) }
func sbcZpX(c *CPU)  { opSbc(c, c.read(c.effZPX())) }
func sbcAbs(c *CPU)  { opSbc(c, c.read(c.effAbs())) }
func sbcAbsX(c *CPU) { opSbc(c, c.read(c.effAbsX(false))) }
func sbcAbsY(c *CPU) { opSbc(c, c.read(c.effAbsY(false))) }
func sbcIndX(c *CPU) { opSbc(c, c.read(c.effIndexedIndirect())) }
func sbcIndY(c *CPU) { opSbc(c, c.read(c.effIndirectIndexed(false))) }
