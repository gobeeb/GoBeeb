package mos6502

// Processor status (P) bit constants. (FR-001, FR-004)
//
// The P register is a single byte with the layout NV-BDIZC (LSB first:
// C, Z, I, D, B, U, V, N). Bit 5 (U) is conceptually always 1 on a real
// 6502 but is enforced live only when P is pushed to the stack. Bit 4
// (B) is not a real flag in the live register; it only takes a value in
// the pushed copy (set for BRK, clear for IRQ/NMI).
const (
	FlagCarry     uint8 = 1 << 0
	FlagZero      uint8 = 1 << 1
	FlagInterrupt uint8 = 1 << 2
	FlagDecimal   uint8 = 1 << 3
	FlagBreak     uint8 = 1 << 4
	FlagUnused    uint8 = 1 << 5
	FlagOverflow  uint8 = 1 << 6
	FlagNegative  uint8 = 1 << 7
)

// flag returns true when the bit(s) named by mask are all set in P.
func (c *CPU) flag(mask uint8) bool { return c.P&mask != 0 }

// setFlag sets or clears the bit(s) named by mask in P.
func (c *CPU) setFlag(mask uint8, on bool) {
	if on {
		c.P |= mask
	} else {
		c.P &^= mask
	}
}

// setNZ sets the N and Z bits in P based on the byte v, the same way
// every load / arithmetic / logic instruction does.
func (c *CPU) setNZ(v uint8) {
	c.setFlag(FlagZero, v == 0)
	c.setFlag(FlagNegative, v&0x80 != 0)
}
