package mos6502

// dummyFetch issues a single bus read at PC without advancing PC. This
// reproduces the "internal operation" cycle a real NMOS 6502 spends on
// implicit / register-only / branch-fix-up operations.
func (c *CPU) dummyFetch() { _ = c.read(c.PC) }

// ──────────────────────────────────────────────────────────────────
// LDA / LDX / LDY  (load → setNZ)
// ──────────────────────────────────────────────────────────────────

func load(c *CPU, reg *uint8, addr uint16) {
	*reg = c.read(addr)
	c.setNZ(*reg)
}

func ldaImm(c *CPU)  { c.A = c.fetch(); c.setNZ(c.A) }
func ldaZp(c *CPU)   { load(c, &c.A, c.effZP()) }
func ldaZpX(c *CPU)  { load(c, &c.A, c.effZPX()) }
func ldaAbs(c *CPU)  { load(c, &c.A, c.effAbs()) }
func ldaAbsX(c *CPU) { load(c, &c.A, c.effAbsX(false)) }
func ldaAbsY(c *CPU) { load(c, &c.A, c.effAbsY(false)) }
func ldaIndX(c *CPU) { load(c, &c.A, c.effIndexedIndirect()) }
func ldaIndY(c *CPU) { load(c, &c.A, c.effIndirectIndexed(false)) }

func ldxImm(c *CPU)  { c.X = c.fetch(); c.setNZ(c.X) }
func ldxZp(c *CPU)   { load(c, &c.X, c.effZP()) }
func ldxZpY(c *CPU)  { load(c, &c.X, c.effZPY()) }
func ldxAbs(c *CPU)  { load(c, &c.X, c.effAbs()) }
func ldxAbsY(c *CPU) { load(c, &c.X, c.effAbsY(false)) }

func ldyImm(c *CPU)  { c.Y = c.fetch(); c.setNZ(c.Y) }
func ldyZp(c *CPU)   { load(c, &c.Y, c.effZP()) }
func ldyZpX(c *CPU)  { load(c, &c.Y, c.effZPX()) }
func ldyAbs(c *CPU)  { load(c, &c.Y, c.effAbs()) }
func ldyAbsX(c *CPU) { load(c, &c.Y, c.effAbsX(false)) }

// ──────────────────────────────────────────────────────────────────
// STA / STX / STY  (no flags)
// ──────────────────────────────────────────────────────────────────

func staZp(c *CPU)   { c.write(c.effZP(), c.A) }
func staZpX(c *CPU)  { c.write(c.effZPX(), c.A) }
func staAbs(c *CPU)  { c.write(c.effAbs(), c.A) }
func staAbsX(c *CPU) { c.write(c.effAbsX(true), c.A) }
func staAbsY(c *CPU) { c.write(c.effAbsY(true), c.A) }
func staIndX(c *CPU) { c.write(c.effIndexedIndirect(), c.A) }
func staIndY(c *CPU) { c.write(c.effIndirectIndexed(true), c.A) }

func stxZp(c *CPU)  { c.write(c.effZP(), c.X) }
func stxZpY(c *CPU) { c.write(c.effZPY(), c.X) }
func stxAbs(c *CPU) { c.write(c.effAbs(), c.X) }

func styZp(c *CPU)  { c.write(c.effZP(), c.Y) }
func styZpX(c *CPU) { c.write(c.effZPX(), c.Y) }
func styAbs(c *CPU) { c.write(c.effAbs(), c.Y) }

// ──────────────────────────────────────────────────────────────────
// Transfer  (TAX/TAY/TSX/TXA/TXS/TYA)
// ──────────────────────────────────────────────────────────────────

func tax(c *CPU) { c.dummyFetch(); c.X = c.A; c.setNZ(c.X) }
func tay(c *CPU) { c.dummyFetch(); c.Y = c.A; c.setNZ(c.Y) }
func tsx(c *CPU) { c.dummyFetch(); c.X = c.SP; c.setNZ(c.X) }
func txa(c *CPU) { c.dummyFetch(); c.A = c.X; c.setNZ(c.A) }
func txs(c *CPU) { c.dummyFetch(); c.SP = c.X } // TXS does not affect flags
func tya(c *CPU) { c.dummyFetch(); c.A = c.Y; c.setNZ(c.A) }

// ──────────────────────────────────────────────────────────────────
// Flag manipulation  (CLC/SEC/CLD/SED/CLI/SEI/CLV)
// ──────────────────────────────────────────────────────────────────

func clc(c *CPU) { c.dummyFetch(); c.setFlag(FlagCarry, false) }
func sec(c *CPU) { c.dummyFetch(); c.setFlag(FlagCarry, true) }
func cld(c *CPU) { c.dummyFetch(); c.setFlag(FlagDecimal, false) }
func sed(c *CPU) { c.dummyFetch(); c.setFlag(FlagDecimal, true) }
func cli(c *CPU) { c.dummyFetch(); c.setFlag(FlagInterrupt, false) }
func sei(c *CPU) { c.dummyFetch(); c.setFlag(FlagInterrupt, true) }
func clv(c *CPU) { c.dummyFetch(); c.setFlag(FlagOverflow, false) }

// ──────────────────────────────────────────────────────────────────
// NOP
// ──────────────────────────────────────────────────────────────────

func nop(c *CPU) { c.dummyFetch() }

// ──────────────────────────────────────────────────────────────────
// Logical  (AND / ORA / EOR / BIT)
// ──────────────────────────────────────────────────────────────────

func opAnd(c *CPU, m uint8) { c.A &= m; c.setNZ(c.A) }
func opOra(c *CPU, m uint8) { c.A |= m; c.setNZ(c.A) }
func opEor(c *CPU, m uint8) { c.A ^= m; c.setNZ(c.A) }

func andImm(c *CPU)  { opAnd(c, c.fetch()) }
func andZp(c *CPU)   { opAnd(c, c.read(c.effZP())) }
func andZpX(c *CPU)  { opAnd(c, c.read(c.effZPX())) }
func andAbs(c *CPU)  { opAnd(c, c.read(c.effAbs())) }
func andAbsX(c *CPU) { opAnd(c, c.read(c.effAbsX(false))) }
func andAbsY(c *CPU) { opAnd(c, c.read(c.effAbsY(false))) }
func andIndX(c *CPU) { opAnd(c, c.read(c.effIndexedIndirect())) }
func andIndY(c *CPU) { opAnd(c, c.read(c.effIndirectIndexed(false))) }

func oraImm(c *CPU)  { opOra(c, c.fetch()) }
func oraZp(c *CPU)   { opOra(c, c.read(c.effZP())) }
func oraZpX(c *CPU)  { opOra(c, c.read(c.effZPX())) }
func oraAbs(c *CPU)  { opOra(c, c.read(c.effAbs())) }
func oraAbsX(c *CPU) { opOra(c, c.read(c.effAbsX(false))) }
func oraAbsY(c *CPU) { opOra(c, c.read(c.effAbsY(false))) }
func oraIndX(c *CPU) { opOra(c, c.read(c.effIndexedIndirect())) }
func oraIndY(c *CPU) { opOra(c, c.read(c.effIndirectIndexed(false))) }

func eorImm(c *CPU)  { opEor(c, c.fetch()) }
func eorZp(c *CPU)   { opEor(c, c.read(c.effZP())) }
func eorZpX(c *CPU)  { opEor(c, c.read(c.effZPX())) }
func eorAbs(c *CPU)  { opEor(c, c.read(c.effAbs())) }
func eorAbsX(c *CPU) { opEor(c, c.read(c.effAbsX(false))) }
func eorAbsY(c *CPU) { opEor(c, c.read(c.effAbsY(false))) }
func eorIndX(c *CPU) { opEor(c, c.read(c.effIndexedIndirect())) }
func eorIndY(c *CPU) { opEor(c, c.read(c.effIndirectIndexed(false))) }

func doBit(c *CPU, m uint8) {
	c.setFlag(FlagZero, c.A&m == 0)
	c.setFlag(FlagOverflow, m&0x40 != 0)
	c.setFlag(FlagNegative, m&0x80 != 0)
}
func bitZp(c *CPU)  { doBit(c, c.read(c.effZP())) }
func bitAbs(c *CPU) { doBit(c, c.read(c.effAbs())) }

// ──────────────────────────────────────────────────────────────────
// Compare  (CMP / CPX / CPY)
// ──────────────────────────────────────────────────────────────────

func compare(c *CPU, reg uint8, m uint8) {
	c.setFlag(FlagCarry, reg >= m)
	c.setNZ(reg - m)
}

func cmpImm(c *CPU)  { compare(c, c.A, c.fetch()) }
func cmpZp(c *CPU)   { compare(c, c.A, c.read(c.effZP())) }
func cmpZpX(c *CPU)  { compare(c, c.A, c.read(c.effZPX())) }
func cmpAbs(c *CPU)  { compare(c, c.A, c.read(c.effAbs())) }
func cmpAbsX(c *CPU) { compare(c, c.A, c.read(c.effAbsX(false))) }
func cmpAbsY(c *CPU) { compare(c, c.A, c.read(c.effAbsY(false))) }
func cmpIndX(c *CPU) { compare(c, c.A, c.read(c.effIndexedIndirect())) }
func cmpIndY(c *CPU) { compare(c, c.A, c.read(c.effIndirectIndexed(false))) }

func cpxImm(c *CPU) { compare(c, c.X, c.fetch()) }
func cpxZp(c *CPU)  { compare(c, c.X, c.read(c.effZP())) }
func cpxAbs(c *CPU) { compare(c, c.X, c.read(c.effAbs())) }

func cpyImm(c *CPU) { compare(c, c.Y, c.fetch()) }
func cpyZp(c *CPU)  { compare(c, c.Y, c.read(c.effZP())) }
func cpyAbs(c *CPU) { compare(c, c.Y, c.read(c.effAbs())) }

// ──────────────────────────────────────────────────────────────────
// INX / INY / DEX / DEY  (implicit, 2 cycles)
// ──────────────────────────────────────────────────────────────────

func inx(c *CPU) { c.dummyFetch(); c.X++; c.setNZ(c.X) }
func iny(c *CPU) { c.dummyFetch(); c.Y++; c.setNZ(c.Y) }
func dex(c *CPU) { c.dummyFetch(); c.X--; c.setNZ(c.X) }
func dey(c *CPU) { c.dummyFetch(); c.Y--; c.setNZ(c.Y) }

// ──────────────────────────────────────────────────────────────────
// Branches  (FR-005)
//
// 2 cycles base; +1 on taken; +1 more on taken-AND-page-crossed.
// ──────────────────────────────────────────────────────────────────

func branch(c *CPU, take bool) {
	offset := int8(c.fetch())
	if !take {
		return
	}
	oldPC := c.PC
	c.dummyFetch() // +1 cycle on taken
	newPC := uint16(int32(c.PC) + int32(offset))
	if (oldPC & 0xFF00) != (newPC & 0xFF00) {
		// dummy read at the un-fixed-up address (high byte not yet
		// rippled) — +1 more cycle on page-cross
		_ = c.read((oldPC & 0xFF00) | (newPC & 0x00FF))
	}
	c.PC = newPC
}

func bcc(c *CPU) { branch(c, !c.flag(FlagCarry)) }
func bcs(c *CPU) { branch(c, c.flag(FlagCarry)) }
func beq(c *CPU) { branch(c, c.flag(FlagZero)) }
func bne(c *CPU) { branch(c, !c.flag(FlagZero)) }
func bmi(c *CPU) { branch(c, c.flag(FlagNegative)) }
func bpl(c *CPU) { branch(c, !c.flag(FlagNegative)) }
func bvc(c *CPU) { branch(c, !c.flag(FlagOverflow)) }
func bvs(c *CPU) { branch(c, c.flag(FlagOverflow)) }

// ──────────────────────────────────────────────────────────────────
// Stack  (PHA/PHP/PLA/PLP)
// ──────────────────────────────────────────────────────────────────

func pha(c *CPU) {
	c.dummyFetch()
	c.push(c.A)
}

// php pushes P with B and U bits both set in the pushed copy.
func php(c *CPU) {
	c.dummyFetch()
	c.push(c.P | FlagBreak | FlagUnused)
}

func pla(c *CPU) {
	c.dummyFetch()
	_ = c.read(0x0100 | uint16(c.SP)) // dummy stack peek
	c.A = c.pull()
	c.setNZ(c.A)
}

// plp pulls P; the B bit is ignored (does not exist in the live P) and U
// is forced set.
func plp(c *CPU) {
	c.dummyFetch()
	_ = c.read(0x0100 | uint16(c.SP))
	v := c.pull()
	c.P = (v &^ FlagBreak) | FlagUnused
}

// ──────────────────────────────────────────────────────────────────
// Jumps  (JMP / JSR / RTS)
// ──────────────────────────────────────────────────────────────────

func jmpAbs(c *CPU) { c.PC = c.effAbs() }
func jmpInd(c *CPU) { c.PC = c.effIndirect() }

// jsr is 6 cycles. The pushed return address is the address of the LAST
// byte of the JSR instruction (i.e. one before the next instruction):
// at the moment of push, PC has only advanced past the ADL byte, so
// PC == JSR_PC + 1 + 1 = JSR_PC + 2 ... wait, only ADL was fetched, so
// PC == JSR_PC + 2 ? No — ADL is one byte fetch, so PC = JSR_PC + 1 + 1
// (opcode + ADL) — yes, PC = JSR_PC + 2 at the push site.
func jsr(c *CPU) {
	lo := c.fetch()                   // cycle 2: ADL fetch (PC = JSR_PC + 2)
	_ = c.read(0x0100 | uint16(c.SP)) // cycle 3: internal / dummy stack peek
	c.push(uint8(c.PC >> 8))          // cycle 4: push PCH
	c.push(uint8(c.PC))               // cycle 5: push PCL
	hi := c.fetch()                   // cycle 6: ADH fetch
	c.PC = uint16(lo) | uint16(hi)<<8
}

// rts is 6 cycles.
func rts(c *CPU) {
	c.dummyFetch()                    // cycle 2: dummy fetch
	_ = c.read(0x0100 | uint16(c.SP)) // cycle 3: dummy stack peek
	lo := c.pull()                    // cycle 4
	hi := c.pull()                    // cycle 5
	c.PC = uint16(lo) | uint16(hi)<<8
	c.dummyFetch() // cycle 6: dummy fetch at PC
	c.PC++
}

// ──────────────────────────────────────────────────────────────────
// BRK / RTI
//
// BRK is 7 cycles. It pushes PC+2 (skipping the byte after the BRK
// opcode), pushes P with B set, sets I, vectors through $FFFE/$FFFF.
// The shared interrupt-entry routine in interrupts.go handles cycles
// 3–7 (push state + vector fetch) and the NMI-hijack window (FR-022).
// ──────────────────────────────────────────────────────────────────

func brk(c *CPU) {
	_ = c.fetch() // cycle 2: read padding byte, PC++
	enterInterrupt(c, brkInterrupt)
}

// rti is 6 cycles. Pulls P (B masked out, U forced), then PCL, then PCH.
func rti(c *CPU) {
	c.dummyFetch()
	_ = c.read(0x0100 | uint16(c.SP))
	v := c.pull()
	c.P = (v &^ FlagBreak) | FlagUnused
	lo := c.pull()
	hi := c.pull()
	c.PC = uint16(lo) | uint16(hi)<<8
}

// ──────────────────────────────────────────────────────────────────
// Accumulator-form shifts/rotates (memory form lives in rmw.go)
// ──────────────────────────────────────────────────────────────────

func aslA(c *CPU) {
	c.dummyFetch()
	c.setFlag(FlagCarry, c.A&0x80 != 0)
	c.A <<= 1
	c.setNZ(c.A)
}

func lsrA(c *CPU) {
	c.dummyFetch()
	c.setFlag(FlagCarry, c.A&0x01 != 0)
	c.A >>= 1
	c.setNZ(c.A)
}

func rolA(c *CPU) {
	c.dummyFetch()
	oldC := c.flag(FlagCarry)
	c.setFlag(FlagCarry, c.A&0x80 != 0)
	c.A <<= 1
	if oldC {
		c.A |= 0x01
	}
	c.setNZ(c.A)
}

func rorA(c *CPU) {
	c.dummyFetch()
	oldC := c.flag(FlagCarry)
	c.setFlag(FlagCarry, c.A&0x01 != 0)
	c.A >>= 1
	if oldC {
		c.A |= 0x80
	}
	c.setNZ(c.A)
}
