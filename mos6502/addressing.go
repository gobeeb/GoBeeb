package mos6502

// AddressingMode is one of the thirteen NMOS 6502 addressing modes.
type AddressingMode uint8

// Addressing mode enum values. (FR-003) See spec.md §FR-003 and
// data-model.md §5 for the per-mode semantics.
const (
	ModeImplicit        AddressingMode = iota // operates on registers only
	ModeAccumulator                           // operand is A
	ModeImmediate                             // operand is the byte after opcode
	ModeZeroPage                              // $LL
	ModeZeroPageX                             // $LL,X (wraps in zero page)
	ModeZeroPageY                             // $LL,Y (wraps in zero page)
	ModeRelative                              // signed 8-bit offset from PC
	ModeAbsolute                              // $LLHH
	ModeAbsoluteX                             // $LLHH,X (+1 cycle on page cross)
	ModeAbsoluteY                             // $LLHH,Y (+1 cycle on page cross)
	ModeIndirect                              // ($LLHH) — JMP only, with NMOS page bug
	ModeIndexedIndirect                       // ($LL,X) — wraps in zero page
	ModeIndirectIndexed                       // ($LL),Y — +1 cycle on page cross
)

// modeBytes is the documented byte length (opcode + operand bytes) for
// each addressing mode. Used by the disassembler.
var modeBytes = [...]uint8{
	ModeImplicit:        1,
	ModeAccumulator:     1,
	ModeImmediate:       2,
	ModeZeroPage:        2,
	ModeZeroPageX:       2,
	ModeZeroPageY:       2,
	ModeRelative:        2,
	ModeAbsolute:        3,
	ModeAbsoluteX:       3,
	ModeAbsoluteY:       3,
	ModeIndirect:        3,
	ModeIndexedIndirect: 2,
	ModeIndirectIndexed: 2,
}

// ──────────────────────────────────────────────────────────────────
// Effective-address helpers (FR-003, FR-013, FR-014, FR-018)
//
// Each helper performs the bus cycles a real NMOS 6502 performs to
// compute the effective address, including dummy reads on indexed
// page-cross (FR-018) and the JMP-indirect page bug (FR-013).
// ──────────────────────────────────────────────────────────────────

// effZP returns a zero-page effective address. 1 cycle (the operand fetch).
func (c *CPU) effZP() uint16 {
	return uint16(c.fetch())
}

// effZPX returns a zero-page,X effective address. 2 cycles: operand
// fetch + dummy read of the un-indexed zero-page address. Wraps in
// zero page (FR-014).
func (c *CPU) effZPX() uint16 {
	base := c.fetch()
	_ = c.read(uint16(base)) // dummy read of un-indexed address
	return uint16(base + c.X)
}

// effZPY returns a zero-page,Y effective address. 2 cycles: operand
// fetch + dummy read. Wraps in zero page (FR-014).
func (c *CPU) effZPY() uint16 {
	base := c.fetch()
	_ = c.read(uint16(base))
	return uint16(base + c.Y)
}

// effAbs returns an absolute effective address. 2 cycles (operand fetch
// of low + high byte).
func (c *CPU) effAbs() uint16 {
	lo := c.fetch()
	hi := c.fetch()
	return uint16(lo) | uint16(hi)<<8
}

// effAbsX returns an absolute,X effective address. Always issues 2
// operand-fetch cycles. On page-cross, also issues one dummy read at
// the un-fixed-up address (FR-018). The caller indicates whether the
// access is for a "store" or RMW (which always pays the dummy-read
// cycle, even without a page-cross) by setting alwaysPenalty=true.
func (c *CPU) effAbsX(alwaysPenalty bool) uint16 {
	lo := c.fetch()
	hi := c.fetch()
	base := uint16(lo) | uint16(hi)<<8
	addr := base + uint16(c.X)
	if alwaysPenalty || (base&0xFF00) != (addr&0xFF00) {
		_ = c.read((base & 0xFF00) | (addr & 0x00FF))
	}
	return addr
}

// effAbsY returns an absolute,Y effective address. Same semantics as
// effAbsX with Y instead of X.
func (c *CPU) effAbsY(alwaysPenalty bool) uint16 {
	lo := c.fetch()
	hi := c.fetch()
	base := uint16(lo) | uint16(hi)<<8
	addr := base + uint16(c.Y)
	if alwaysPenalty || (base&0xFF00) != (addr&0xFF00) {
		_ = c.read((base & 0xFF00) | (addr & 0x00FF))
	}
	return addr
}

// effIndirect implements the JMP ($LLHH) indirect addressing mode with
// the NMOS page-bug: when the pointer's low byte is $FF the high byte
// of the target is fetched from $xx00 of the same page rather than
// $xx00 of the next page. (FR-013)
func (c *CPU) effIndirect() uint16 {
	ptrLo := c.fetch()
	ptrHi := c.fetch()
	ptr := uint16(ptrLo) | uint16(ptrHi)<<8
	lo := c.read(ptr)
	hi := c.read((ptr & 0xFF00) | uint16(uint8(ptr)+1))
	return uint16(lo) | uint16(hi)<<8
}

// effIndexedIndirect implements ($LL,X). The X-indexed zero-page
// pointer wraps in zero page (FR-014). 4 cycles: operand fetch + dummy
// read at $LL + read of pointer-low + read of pointer-high.
func (c *CPU) effIndexedIndirect() uint16 {
	base := c.fetch()
	_ = c.read(uint16(base)) // dummy read of un-indexed pointer
	ptr := base + c.X        // wraps in zero page (uint8 arithmetic)
	lo := c.read(uint16(ptr))
	hi := c.read(uint16(ptr + 1))
	return uint16(lo) | uint16(hi)<<8
}

// effIndirectIndexed implements ($LL),Y. 3 cycles plus an optional
// dummy read on page-cross (FR-018). Set alwaysPenalty=true for store /
// RMW variants that always pay the penalty cycle.
func (c *CPU) effIndirectIndexed(alwaysPenalty bool) uint16 {
	ptr := c.fetch()
	lo := c.read(uint16(ptr))
	hi := c.read(uint16(ptr + 1)) // pointer-high read wraps in zero page
	base := uint16(lo) | uint16(hi)<<8
	addr := base + uint16(c.Y)
	if alwaysPenalty || (base&0xFF00) != (addr&0xFF00) {
		_ = c.read((base & 0xFF00) | (addr & 0x00FF))
	}
	return addr
}
