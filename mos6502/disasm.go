package mos6502

import "strings"

// Disassemble returns the human-readable mnemonic + operand for the
// instruction at pc in mem, plus the byte length the instruction
// consumes. The function makes no use of CPU state and performs no
// allocation beyond the returned string. (FR-020 observability)
//
// Format is the standard NMOS 6502 syntax: immediate operands as
// `#$NN`, addresses as `$NNNN` or `$NN`, indexed as `,X` / `,Y`,
// indirect as `($NNNN)` / `($NN,X)` / `($NN),Y`. Branch targets are
// expressed as their absolute destination address.
func Disassemble(mem Memory, pc uint16) (text string, length int) {
	op := mem.Read(pc)
	meta := opcodeMetaTable[op]

	var b strings.Builder
	b.Grow(16)
	b.WriteString(meta.Mnemonic)

	switch meta.Mode {
	case ModeImplicit:
		// no operand
	case ModeAccumulator:
		b.WriteString(" A")
	case ModeImmediate:
		b.WriteString(" #$")
		appendHex2(&b, mem.Read(pc+1))
	case ModeZeroPage:
		b.WriteString(" $")
		appendHex2(&b, mem.Read(pc+1))
	case ModeZeroPageX:
		b.WriteString(" $")
		appendHex2(&b, mem.Read(pc+1))
		b.WriteString(",X")
	case ModeZeroPageY:
		b.WriteString(" $")
		appendHex2(&b, mem.Read(pc+1))
		b.WriteString(",Y")
	case ModeRelative:
		offset := int8(mem.Read(pc + 1))
		dest := uint16(int32(pc) + 2 + int32(offset))
		b.WriteString(" $")
		appendHex4(&b, dest)
	case ModeAbsolute:
		lo := mem.Read(pc + 1)
		hi := mem.Read(pc + 2)
		b.WriteString(" $")
		appendHex4(&b, uint16(lo)|uint16(hi)<<8)
	case ModeAbsoluteX:
		lo := mem.Read(pc + 1)
		hi := mem.Read(pc + 2)
		b.WriteString(" $")
		appendHex4(&b, uint16(lo)|uint16(hi)<<8)
		b.WriteString(",X")
	case ModeAbsoluteY:
		lo := mem.Read(pc + 1)
		hi := mem.Read(pc + 2)
		b.WriteString(" $")
		appendHex4(&b, uint16(lo)|uint16(hi)<<8)
		b.WriteString(",Y")
	case ModeIndirect:
		lo := mem.Read(pc + 1)
		hi := mem.Read(pc + 2)
		b.WriteString(" ($")
		appendHex4(&b, uint16(lo)|uint16(hi)<<8)
		b.WriteByte(')')
	case ModeIndexedIndirect:
		b.WriteString(" ($")
		appendHex2(&b, mem.Read(pc+1))
		b.WriteString(",X)")
	case ModeIndirectIndexed:
		b.WriteString(" ($")
		appendHex2(&b, mem.Read(pc+1))
		b.WriteString("),Y")
	}

	return b.String(), int(meta.Length)
}

func appendHex2(b *strings.Builder, v uint8) {
	const hex = "0123456789ABCDEF"
	b.WriteByte(hex[v>>4])
	b.WriteByte(hex[v&0x0F])
}

func appendHex4(b *strings.Builder, v uint16) {
	const hex = "0123456789ABCDEF"
	b.WriteByte(hex[(v>>12)&0x0F])
	b.WriteByte(hex[(v>>8)&0x0F])
	b.WriteByte(hex[(v>>4)&0x0F])
	b.WriteByte(hex[v&0x0F])
}
