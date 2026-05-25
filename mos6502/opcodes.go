package mos6502

// opcodeFn is the signature every entry in the dispatch table satisfies.
type opcodeFn func(c *CPU)

// opcodeMeta carries the human-readable metadata for one opcode byte.
// Only used by the disassembler / debug surface; not on the hot path.
type opcodeMeta struct {
	Mnemonic   string
	Mode       AddressingMode
	BaseCycles uint8
	Length     uint8
	Illegal    bool
}

// opcodeTable is the 256-entry function-pointer dispatch table indexed
// by the opcode byte. The 151 documented NMOS 6502 opcodes have
// concrete entries; the remaining 105 illegal slots default to the
// shared illegalNOP handler. (FR-002, FR-019)
var opcodeTable [256]opcodeFn

// opcodeMetaTable mirrors opcodeTable, carrying disassembly metadata.
var opcodeMetaTable [256]opcodeMeta

func init() {
	// Default every slot to illegalNOP and mark every meta entry as
	// illegal; documented opcodes overwrite below.
	for i := 0; i < 256; i++ {
		opcodeTable[i] = illegalNOP
		opcodeMetaTable[i] = opcodeMeta{
			Mnemonic: "???",
			Mode:     ModeImplicit,
			Length:   1,
			Illegal:  true,
		}
	}

	type entry struct {
		op       uint8
		fn       opcodeFn
		mnemonic string
		mode     AddressingMode
		cycles   uint8
	}

	entries := []entry{
		// ── ORA / ASL / PHP / BPL / CLC (0x00..0x1F) ─────────────────
		{0x00, brk, "BRK", ModeImplicit, 7},
		{0x01, oraIndX, "ORA", ModeIndexedIndirect, 6},
		{0x05, oraZp, "ORA", ModeZeroPage, 3},
		{0x06, aslZp, "ASL", ModeZeroPage, 5},
		{0x08, php, "PHP", ModeImplicit, 3},
		{0x09, oraImm, "ORA", ModeImmediate, 2},
		{0x0A, aslA, "ASL", ModeAccumulator, 2},
		{0x0D, oraAbs, "ORA", ModeAbsolute, 4},
		{0x0E, aslAbs, "ASL", ModeAbsolute, 6},
		{0x10, bpl, "BPL", ModeRelative, 2},
		{0x11, oraIndY, "ORA", ModeIndirectIndexed, 5},
		{0x15, oraZpX, "ORA", ModeZeroPageX, 4},
		{0x16, aslZpX, "ASL", ModeZeroPageX, 6},
		{0x18, clc, "CLC", ModeImplicit, 2},
		{0x19, oraAbsY, "ORA", ModeAbsoluteY, 4},
		{0x1D, oraAbsX, "ORA", ModeAbsoluteX, 4},
		{0x1E, aslAbsX, "ASL", ModeAbsoluteX, 7},

		// ── JSR / AND / BIT / ROL / PLP / BMI / SEC (0x20..0x3F) ─────
		{0x20, jsr, "JSR", ModeAbsolute, 6},
		{0x21, andIndX, "AND", ModeIndexedIndirect, 6},
		{0x24, bitZp, "BIT", ModeZeroPage, 3},
		{0x25, andZp, "AND", ModeZeroPage, 3},
		{0x26, rolZp, "ROL", ModeZeroPage, 5},
		{0x28, plp, "PLP", ModeImplicit, 4},
		{0x29, andImm, "AND", ModeImmediate, 2},
		{0x2A, rolA, "ROL", ModeAccumulator, 2},
		{0x2C, bitAbs, "BIT", ModeAbsolute, 4},
		{0x2D, andAbs, "AND", ModeAbsolute, 4},
		{0x2E, rolAbs, "ROL", ModeAbsolute, 6},
		{0x30, bmi, "BMI", ModeRelative, 2},
		{0x31, andIndY, "AND", ModeIndirectIndexed, 5},
		{0x35, andZpX, "AND", ModeZeroPageX, 4},
		{0x36, rolZpX, "ROL", ModeZeroPageX, 6},
		{0x38, sec, "SEC", ModeImplicit, 2},
		{0x39, andAbsY, "AND", ModeAbsoluteY, 4},
		{0x3D, andAbsX, "AND", ModeAbsoluteX, 4},
		{0x3E, rolAbsX, "ROL", ModeAbsoluteX, 7},

		// ── RTI / EOR / LSR / PHA / JMP / BVC / CLI (0x40..0x5F) ─────
		{0x40, rti, "RTI", ModeImplicit, 6},
		{0x41, eorIndX, "EOR", ModeIndexedIndirect, 6},
		{0x45, eorZp, "EOR", ModeZeroPage, 3},
		{0x46, lsrZp, "LSR", ModeZeroPage, 5},
		{0x48, pha, "PHA", ModeImplicit, 3},
		{0x49, eorImm, "EOR", ModeImmediate, 2},
		{0x4A, lsrA, "LSR", ModeAccumulator, 2},
		{0x4C, jmpAbs, "JMP", ModeAbsolute, 3},
		{0x4D, eorAbs, "EOR", ModeAbsolute, 4},
		{0x4E, lsrAbs, "LSR", ModeAbsolute, 6},
		{0x50, bvc, "BVC", ModeRelative, 2},
		{0x51, eorIndY, "EOR", ModeIndirectIndexed, 5},
		{0x55, eorZpX, "EOR", ModeZeroPageX, 4},
		{0x56, lsrZpX, "LSR", ModeZeroPageX, 6},
		{0x58, cli, "CLI", ModeImplicit, 2},
		{0x59, eorAbsY, "EOR", ModeAbsoluteY, 4},
		{0x5D, eorAbsX, "EOR", ModeAbsoluteX, 4},
		{0x5E, lsrAbsX, "LSR", ModeAbsoluteX, 7},

		// ── RTS / ADC / ROR / PLA / JMP-ind / BVS / SEI (0x60..0x7F) ─
		{0x60, rts, "RTS", ModeImplicit, 6},
		{0x61, adcIndX, "ADC", ModeIndexedIndirect, 6},
		{0x65, adcZp, "ADC", ModeZeroPage, 3},
		{0x66, rorZp, "ROR", ModeZeroPage, 5},
		{0x68, pla, "PLA", ModeImplicit, 4},
		{0x69, adcImm, "ADC", ModeImmediate, 2},
		{0x6A, rorA, "ROR", ModeAccumulator, 2},
		{0x6C, jmpInd, "JMP", ModeIndirect, 5},
		{0x6D, adcAbs, "ADC", ModeAbsolute, 4},
		{0x6E, rorAbs, "ROR", ModeAbsolute, 6},
		{0x70, bvs, "BVS", ModeRelative, 2},
		{0x71, adcIndY, "ADC", ModeIndirectIndexed, 5},
		{0x75, adcZpX, "ADC", ModeZeroPageX, 4},
		{0x76, rorZpX, "ROR", ModeZeroPageX, 6},
		{0x78, sei, "SEI", ModeImplicit, 2},
		{0x79, adcAbsY, "ADC", ModeAbsoluteY, 4},
		{0x7D, adcAbsX, "ADC", ModeAbsoluteX, 4},
		{0x7E, rorAbsX, "ROR", ModeAbsoluteX, 7},

		// ── STA / STY / STX / DEY / TXA / BCC / TYA / TXS (0x80..0x9F)
		{0x81, staIndX, "STA", ModeIndexedIndirect, 6},
		{0x84, styZp, "STY", ModeZeroPage, 3},
		{0x85, staZp, "STA", ModeZeroPage, 3},
		{0x86, stxZp, "STX", ModeZeroPage, 3},
		{0x88, dey, "DEY", ModeImplicit, 2},
		{0x8A, txa, "TXA", ModeImplicit, 2},
		{0x8C, styAbs, "STY", ModeAbsolute, 4},
		{0x8D, staAbs, "STA", ModeAbsolute, 4},
		{0x8E, stxAbs, "STX", ModeAbsolute, 4},
		{0x90, bcc, "BCC", ModeRelative, 2},
		{0x91, staIndY, "STA", ModeIndirectIndexed, 6},
		{0x94, styZpX, "STY", ModeZeroPageX, 4},
		{0x95, staZpX, "STA", ModeZeroPageX, 4},
		{0x96, stxZpY, "STX", ModeZeroPageY, 4},
		{0x98, tya, "TYA", ModeImplicit, 2},
		{0x99, staAbsY, "STA", ModeAbsoluteY, 5},
		{0x9A, txs, "TXS", ModeImplicit, 2},
		{0x9D, staAbsX, "STA", ModeAbsoluteX, 5},

		// ── LDY / LDA / LDX / TAY / TAX / BCS / CLV / TSX (0xA0..0xBF)
		{0xA0, ldyImm, "LDY", ModeImmediate, 2},
		{0xA1, ldaIndX, "LDA", ModeIndexedIndirect, 6},
		{0xA2, ldxImm, "LDX", ModeImmediate, 2},
		{0xA4, ldyZp, "LDY", ModeZeroPage, 3},
		{0xA5, ldaZp, "LDA", ModeZeroPage, 3},
		{0xA6, ldxZp, "LDX", ModeZeroPage, 3},
		{0xA8, tay, "TAY", ModeImplicit, 2},
		{0xA9, ldaImm, "LDA", ModeImmediate, 2},
		{0xAA, tax, "TAX", ModeImplicit, 2},
		{0xAC, ldyAbs, "LDY", ModeAbsolute, 4},
		{0xAD, ldaAbs, "LDA", ModeAbsolute, 4},
		{0xAE, ldxAbs, "LDX", ModeAbsolute, 4},
		{0xB0, bcs, "BCS", ModeRelative, 2},
		{0xB1, ldaIndY, "LDA", ModeIndirectIndexed, 5},
		{0xB4, ldyZpX, "LDY", ModeZeroPageX, 4},
		{0xB5, ldaZpX, "LDA", ModeZeroPageX, 4},
		{0xB6, ldxZpY, "LDX", ModeZeroPageY, 4},
		{0xB8, clv, "CLV", ModeImplicit, 2},
		{0xB9, ldaAbsY, "LDA", ModeAbsoluteY, 4},
		{0xBA, tsx, "TSX", ModeImplicit, 2},
		{0xBC, ldyAbsX, "LDY", ModeAbsoluteX, 4},
		{0xBD, ldaAbsX, "LDA", ModeAbsoluteX, 4},
		{0xBE, ldxAbsY, "LDX", ModeAbsoluteY, 4},

		// ── CPY / CMP / DEC / INY / DEX / BNE / CLD (0xC0..0xDF) ─────
		{0xC0, cpyImm, "CPY", ModeImmediate, 2},
		{0xC1, cmpIndX, "CMP", ModeIndexedIndirect, 6},
		{0xC4, cpyZp, "CPY", ModeZeroPage, 3},
		{0xC5, cmpZp, "CMP", ModeZeroPage, 3},
		{0xC6, decZp, "DEC", ModeZeroPage, 5},
		{0xC8, iny, "INY", ModeImplicit, 2},
		{0xC9, cmpImm, "CMP", ModeImmediate, 2},
		{0xCA, dex, "DEX", ModeImplicit, 2},
		{0xCC, cpyAbs, "CPY", ModeAbsolute, 4},
		{0xCD, cmpAbs, "CMP", ModeAbsolute, 4},
		{0xCE, decAbs, "DEC", ModeAbsolute, 6},
		{0xD0, bne, "BNE", ModeRelative, 2},
		{0xD1, cmpIndY, "CMP", ModeIndirectIndexed, 5},
		{0xD5, cmpZpX, "CMP", ModeZeroPageX, 4},
		{0xD6, decZpX, "DEC", ModeZeroPageX, 6},
		{0xD8, cld, "CLD", ModeImplicit, 2},
		{0xD9, cmpAbsY, "CMP", ModeAbsoluteY, 4},
		{0xDD, cmpAbsX, "CMP", ModeAbsoluteX, 4},
		{0xDE, decAbsX, "DEC", ModeAbsoluteX, 7},

		// ── CPX / SBC / INC / INX / NOP / BEQ / SED (0xE0..0xFF) ─────
		{0xE0, cpxImm, "CPX", ModeImmediate, 2},
		{0xE1, sbcIndX, "SBC", ModeIndexedIndirect, 6},
		{0xE4, cpxZp, "CPX", ModeZeroPage, 3},
		{0xE5, sbcZp, "SBC", ModeZeroPage, 3},
		{0xE6, incZp, "INC", ModeZeroPage, 5},
		{0xE8, inx, "INX", ModeImplicit, 2},
		{0xE9, sbcImm, "SBC", ModeImmediate, 2},
		{0xEA, nop, "NOP", ModeImplicit, 2},
		{0xEC, cpxAbs, "CPX", ModeAbsolute, 4},
		{0xED, sbcAbs, "SBC", ModeAbsolute, 4},
		{0xEE, incAbs, "INC", ModeAbsolute, 6},
		{0xF0, beq, "BEQ", ModeRelative, 2},
		{0xF1, sbcIndY, "SBC", ModeIndirectIndexed, 5},
		{0xF5, sbcZpX, "SBC", ModeZeroPageX, 4},
		{0xF6, incZpX, "INC", ModeZeroPageX, 6},
		{0xF8, sed, "SED", ModeImplicit, 2},
		{0xF9, sbcAbsY, "SBC", ModeAbsoluteY, 4},
		{0xFD, sbcAbsX, "SBC", ModeAbsoluteX, 4},
		{0xFE, incAbsX, "INC", ModeAbsoluteX, 7},
	}

	for _, e := range entries {
		opcodeTable[e.op] = e.fn
		opcodeMetaTable[e.op] = opcodeMeta{
			Mnemonic:   e.mnemonic,
			Mode:       e.mode,
			BaseCycles: e.cycles,
			Length:     modeBytes[e.mode],
			Illegal:    false,
		}
	}
}
