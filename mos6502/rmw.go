package mos6502

// Read-modify-write (RMW) memory operations. The NMOS 6502 implements
// every memory-form RMW as three consecutive bus cycles to the same
// effective address: a read, a dummy write of the original byte, then
// the write of the modified byte. (FR-021)
//
// Accumulator-form versions of ASL/LSR/ROL/ROR are implemented in
// instructions.go (no memory cycles for the operand).

func (c *CPU) rmw(addr uint16, op func(uint8) uint8) {
	v := c.read(addr)
	c.write(addr, v) // dummy write of original value (NMOS quirk)
	c.write(addr, op(v))
}

// Per-op transforms (set carry/N/Z on the CPU).

func (c *CPU) opAsl(v uint8) uint8 {
	c.setFlag(FlagCarry, v&0x80 != 0)
	v <<= 1
	c.setNZ(v)
	return v
}

func (c *CPU) opLsr(v uint8) uint8 {
	c.setFlag(FlagCarry, v&0x01 != 0)
	v >>= 1
	c.setNZ(v)
	return v
}

func (c *CPU) opRol(v uint8) uint8 {
	oldC := c.flag(FlagCarry)
	c.setFlag(FlagCarry, v&0x80 != 0)
	v <<= 1
	if oldC {
		v |= 0x01
	}
	c.setNZ(v)
	return v
}

func (c *CPU) opRor(v uint8) uint8 {
	oldC := c.flag(FlagCarry)
	c.setFlag(FlagCarry, v&0x01 != 0)
	v >>= 1
	if oldC {
		v |= 0x80
	}
	c.setNZ(v)
	return v
}

func (c *CPU) opInc(v uint8) uint8 { v++; c.setNZ(v); return v }
func (c *CPU) opDec(v uint8) uint8 { v--; c.setNZ(v); return v }

// ASL memory
func aslZp(c *CPU)   { c.rmw(c.effZP(), c.opAsl) }
func aslZpX(c *CPU)  { c.rmw(c.effZPX(), c.opAsl) }
func aslAbs(c *CPU)  { c.rmw(c.effAbs(), c.opAsl) }
func aslAbsX(c *CPU) { c.rmw(c.effAbsX(true), c.opAsl) }

// LSR memory
func lsrZp(c *CPU)   { c.rmw(c.effZP(), c.opLsr) }
func lsrZpX(c *CPU)  { c.rmw(c.effZPX(), c.opLsr) }
func lsrAbs(c *CPU)  { c.rmw(c.effAbs(), c.opLsr) }
func lsrAbsX(c *CPU) { c.rmw(c.effAbsX(true), c.opLsr) }

// ROL memory
func rolZp(c *CPU)   { c.rmw(c.effZP(), c.opRol) }
func rolZpX(c *CPU)  { c.rmw(c.effZPX(), c.opRol) }
func rolAbs(c *CPU)  { c.rmw(c.effAbs(), c.opRol) }
func rolAbsX(c *CPU) { c.rmw(c.effAbsX(true), c.opRol) }

// ROR memory
func rorZp(c *CPU)   { c.rmw(c.effZP(), c.opRor) }
func rorZpX(c *CPU)  { c.rmw(c.effZPX(), c.opRor) }
func rorAbs(c *CPU)  { c.rmw(c.effAbs(), c.opRor) }
func rorAbsX(c *CPU) { c.rmw(c.effAbsX(true), c.opRor) }

// INC memory
func incZp(c *CPU)   { c.rmw(c.effZP(), c.opInc) }
func incZpX(c *CPU)  { c.rmw(c.effZPX(), c.opInc) }
func incAbs(c *CPU)  { c.rmw(c.effAbs(), c.opInc) }
func incAbsX(c *CPU) { c.rmw(c.effAbsX(true), c.opInc) }

// DEC memory
func decZp(c *CPU)   { c.rmw(c.effZP(), c.opDec) }
func decZpX(c *CPU)  { c.rmw(c.effZPX(), c.opDec) }
func decAbs(c *CPU)  { c.rmw(c.effAbs(), c.opDec) }
func decAbsX(c *CPU) { c.rmw(c.effAbsX(true), c.opDec) }
