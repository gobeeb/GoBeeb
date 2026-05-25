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
	appendOperand(&b, mem, pc, meta.Mode)

	return b.String(), int(meta.Length)
}

func appendOperand(b *strings.Builder, mem Memory, pc uint16, mode AddressingMode) {
	switch mode {
	case ModeImplicit:
		return
	case ModeAccumulator:
		b.WriteString(" A")
	case ModeImmediate:
		b.WriteString(" #$")
		appendHex2(b, mem.Read(pc+1))
	case ModeZeroPage, ModeZeroPageX, ModeZeroPageY:
		b.WriteString(" $")
		appendHex2(b, mem.Read(pc+1))
		b.WriteString(zpSuffix(mode))
	case ModeRelative:
		offset := int8(mem.Read(pc + 1))
		dest := uint16(int32(pc) + 2 + int32(offset))
		b.WriteString(" $")
		appendHex4(b, dest)
	case ModeAbsolute, ModeAbsoluteX, ModeAbsoluteY, ModeIndirect:
		b.WriteString(absPrefix(mode))
		appendHex4(b, uint16(mem.Read(pc+1))|uint16(mem.Read(pc+2))<<8)
		b.WriteString(absSuffix(mode))
	case ModeIndexedIndirect:
		b.WriteString(" ($")
		appendHex2(b, mem.Read(pc+1))
		b.WriteString(",X)")
	case ModeIndirectIndexed:
		b.WriteString(" ($")
		appendHex2(b, mem.Read(pc+1))
		b.WriteString("),Y")
	}
}

func zpSuffix(mode AddressingMode) string {
	switch mode {
	case ModeZeroPageX:
		return ",X"
	case ModeZeroPageY:
		return ",Y"
	}
	return ""
}

func absPrefix(mode AddressingMode) string {
	if mode == ModeIndirect {
		return " ($"
	}
	return " $"
}

func absSuffix(mode AddressingMode) string {
	switch mode {
	case ModeAbsoluteX:
		return ",X"
	case ModeAbsoluteY:
		return ",Y"
	case ModeIndirect:
		return ")"
	}
	return ""
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
