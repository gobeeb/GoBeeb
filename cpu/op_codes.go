package cpu

import "fmt"

var opCodes = map[uint8]Instruction{}

func addIns(i Instruction) {
	found, ok := opCodes[i.OpCode]
	if ok {
		message := fmt.Sprintf("duplicate op code %x used for instructions %s & %s", i.OpCode, i.Symbol, found.Symbol)
		panic(message)
	}

	opCodes[i.OpCode] = i
}

// Based on the opcodes / instruction set here:
// https://www.masswerk.at/6502/6502_instruction_set.html
func init() {
	// ADC
	addIns(NewInstruction(0x69, "ADC", "Add memory to accululator", Immediate, 2, 2, adcExecute))
	addIns(NewInstruction(0x65, "ADC", "Add memory to accululator", ZeroPage, 2, 3, adcExecute))
	addIns(NewInstruction(0x75, "ADC", "Add memory to accululator", ZeroPageX, 2, 4, adcExecute))
	addIns(NewInstruction(0x6D, "ADC", "Add memory to accululator", Absolute, 3, 4, adcExecute))
	addIns(NewInstruction(0x7D, "ADC", "Add memory to accululator", AbsoluteX, 3, 4, adcExecute)) //*
	addIns(NewInstruction(0x79, "ADC", "Add memory to accululator", AbsoluteY, 3, 4, adcExecute)) //*
	addIns(NewInstruction(0x61, "ADC", "Add memory to accululator", IndirectX, 2, 6, adcExecute))
	addIns(NewInstruction(0x71, "ADC", "Add memory to accululator", IndirectY, 2, 5, adcExecute)) //*

	// AND
	addIns(NewInstruction(0x29, "AND", "AND memory with accumulator", Immediate, 2, 2, andExecute))
	addIns(NewInstruction(0x25, "AND", "AND memory with accumulator", ZeroPage, 2, 3, andExecute))
	addIns(NewInstruction(0x35, "AND", "AND memory with accumulator", ZeroPageX, 2, 4, andExecute))
	addIns(NewInstruction(0x2D, "AND", "AND memory with accumulator", Absolute, 3, 4, andExecute))
	addIns(NewInstruction(0x3D, "AND", "AND memory with accumulator", AbsoluteX, 3, 4, andExecute)) //*
	addIns(NewInstruction(0x39, "AND", "AND memory with accumulator", AbsoluteY, 3, 4, andExecute)) //*
	addIns(NewInstruction(0x21, "AND", "AND memory with accumulator", IndirectX, 2, 6, andExecute))
	addIns(NewInstruction(0x31, "AND", "AND memory with accumulator", IndirectY, 2, 5, andExecute)) //*

	// ASL
	addIns(NewInstruction(0x0A, "ASL", "Shift Left One Bit", Accumulator, 1, 2, aslExecute))
	addIns(NewInstruction(0x06, "ASL", "Shift Left One Bit", ZeroPage, 2, 5, aslExecute))
	addIns(NewInstruction(0x16, "ASL", "Shift Left One Bit", ZeroPageX, 2, 16, aslExecute))
	addIns(NewInstruction(0x0E, "ASL", "Shift Left One Bit", Absolute, 3, 6, aslExecute))
	addIns(NewInstruction(0x1E, "ASL", "Shift Left One Bit", AbsoluteX, 3, 7, aslExecute))

	// BCC
	addIns(NewInstruction(0x90, "BCC", "Branch on Carry Clear", Relative, 2, 2, bccExecute)) //**

	// BCS
	addIns(NewInstruction(0xB0, "BCS", "Branch on Carry Set", Relative, 2, 2, bcsExecute)) //**

	// BEQ
	addIns(NewInstruction(0xF0, "BEQ", "Branch on Result Zero", Relative, 2, 2, beqExecute)) //**

	// BIT
	addIns(NewInstruction(0x24, "BIT", "Test Bits in Memory With Accumulator", ZeroPage, 2, 3, bitExecute))
	addIns(NewInstruction(0x2C, "BIT", "Test Bits in Memory With Accumulator", Absolute, 3, 4, bitExecute))

	// BMI
	addIns(NewInstruction(0x30, "BMI", "Branch on Result Minus", Relative, 2, 2, bmiExecute)) //**

	// BNE
	addIns(NewInstruction(0xD0, "BNE", "Branch on Result not Zero", Relative, 2, 2, bneExecute)) //**

	// BPL
	addIns(NewInstruction(0x10, "BPL", "Branch on Result Plus", Relative, 2, 2, bplExecute)) //**

	// BRK
	addIns(NewInstruction(0x00, "BRK", "Force Break", Implied, 1, 7, brkExecute))

	// BVC
	addIns(NewInstruction(0x50, "BVC", "Branch on Overflow Clear", Relative, 2, 2, bvcExecute)) //**

	// BVS
	addIns(NewInstruction(0x70, "BVS", "Branch on Overflow Set", Relative, 2, 2, bvsExecute)) //**

	// CLC
	addIns(NewInstruction(0x18, "CLC", "Clear Carry Flag", Implied, 1, 2, clcExecute))

	// CLD
	addIns(NewInstruction(0xD8, "CLD", "Clear Decimal Mode", Implied, 1, 2, cldExecute))

	// CLI
	addIns(NewInstruction(0x58, "CLI", "Clear Interrupt Disable Bit", Implied, 1, 2, cliExecute))

	// CLV
	addIns(NewInstruction(0xB8, "CLV", "Clear Overflow Flag", Implied, 1, 2, clvExecute))

	// CMP
	addIns(NewInstruction(0xC9, "CMP", "Compare Memory with Accumulator", Immediate, 2, 2, cmpExecute))
	addIns(NewInstruction(0xC5, "CMP", "Compare Memory with Accumulator", ZeroPage, 2, 3, cmpExecute))
	addIns(NewInstruction(0xD5, "CMP", "Compare Memory with Accumulator", ZeroPageX, 2, 4, cmpExecute))
	addIns(NewInstruction(0xCD, "CMP", "Compare Memory with Accumulator", Absolute, 3, 4, cmpExecute))
	addIns(NewInstruction(0xDD, "CMP", "Compare Memory with Accumulator", AbsoluteX, 3, 4, cmpExecute)) //*
	addIns(NewInstruction(0xD9, "CMP", "Compare Memory with Accumulator", AbsoluteY, 3, 4, cmpExecute)) //*
	addIns(NewInstruction(0xC1, "CMP", "Compare Memory with Accumulator", IndirectX, 2, 6, cmpExecute))
	addIns(NewInstruction(0xD1, "CMP", "Compare Memory with Accumulator", IndirectY, 2, 5, cmpExecute)) //*

	// CPX
	addIns(NewInstruction(0xE0, "CPX", "Compare Memory and Index X", Immediate, 2, 2, cpxExecute))
	addIns(NewInstruction(0xE4, "CPX", "Compare Memory and Index X", ZeroPage, 2, 3, cpxExecute))
	addIns(NewInstruction(0xEC, "CPX", "Compare Memory and Index X", Absolute, 3, 4, cpxExecute))

	// CPY
	addIns(NewInstruction(0xC0, "CPY", "Compare Memory and Index Y", Immediate, 2, 2, cpyExecute))
	addIns(NewInstruction(0xC4, "CPY", "Compare Memory and Index Y", ZeroPage, 2, 3, cpyExecute))
	addIns(NewInstruction(0xCC, "CPY", "Compare Memory and Index Y", Absolute, 3, 4, cpyExecute))

	// DEC
	addIns(NewInstruction(0xC6, "DEC", "Decrement memory by One", ZeroPage, 2, 5, decExecute))
	addIns(NewInstruction(0xD6, "DEC", "Decrement memory by One", ZeroPageX, 2, 6, decExecute))
	addIns(NewInstruction(0xCE, "DEC", "Decrement memory by One", Absolute, 3, 3, decExecute))
	addIns(NewInstruction(0xDE, "DEC", "Decrement memory by One", AbsoluteX, 3, 7, decExecute))

	// DEX
	addIns(NewInstruction(0xCA, "DEX", "Decrement Index X by One", Implied, 1, 2, dexExecute))

	// DEY
	addIns(NewInstruction(0x88, "DEY", "Decrement Index Y by One", Implied, 1, 2, deyExecute))

	// EOR
	addIns(NewInstruction(0x49, "EOR", "Exclusive-OR Memory with Accumulator", Immediate, 2, 2, eorExecute))
	addIns(NewInstruction(0x45, "EOR", "Exclusive-OR Memory with Accumulator", ZeroPage, 2, 3, eorExecute))
	addIns(NewInstruction(0x55, "EOR", "Exclusive-OR Memory with Accumulator", ZeroPageX, 2, 4, eorExecute))
	addIns(NewInstruction(0x4D, "EOR", "Exclusive-OR Memory with Accumulator", Absolute, 3, 4, eorExecute))
	addIns(NewInstruction(0x5D, "EOR", "Exclusive-OR Memory with Accumulator", AbsoluteX, 3, 4, eorExecute)) //*
	addIns(NewInstruction(0x59, "EOR", "Exclusive-OR Memory with Accumulator", AbsoluteY, 3, 4, eorExecute)) //*
	addIns(NewInstruction(0x41, "EOR", "Exclusive-OR Memory with Accumulator", IndirectX, 2, 6, eorExecute))
	addIns(NewInstruction(0x51, "EOR", "Exclusive-OR Memory with Accumulator", IndirectY, 2, 5, eorExecute)) //*

	// INC
	addIns(NewInstruction(0xE6, "INC", "Increment memory by One", ZeroPage, 2, 5, incExecute))
	addIns(NewInstruction(0xF6, "INC", "Increment memory by One", ZeroPageX, 2, 6, incExecute))
	addIns(NewInstruction(0xEE, "INC", "Increment memory by One", Absolute, 3, 3, incExecute))
	addIns(NewInstruction(0xFE, "INC", "Increment memory by One", AbsoluteX, 3, 7, incExecute))

	// INX
	addIns(NewInstruction(0xE8, "INX", "Increment Index X by One", Implied, 1, 2, inxExecute))

	// INY
	addIns(NewInstruction(0xC8, "INY", "Increment Index Y by One", Implied, 1, 2, inyExecute))

	// JMP
	addIns(NewInstruction(0x4C, "JMP", "Jump to New Location", Absolute, 3, 3, jmpExecute))
	addIns(NewInstruction(0x6C, "JMP", "Jump to New Location", Indirect, 3, 5, jmpExecute))

	// JSR
	addIns(NewInstruction(0x20, "JSR", "Jump to New Location (Save ret addr)", Absolute, 3, 6, jsrExecute))

	// LDA
	addIns(NewInstruction(0xA9, "LDA", "Load Accumulator with Memory", Immediate, 2, 2, ldaExecute))
	addIns(NewInstruction(0xA5, "LDA", "Load Accumulator with Memory", ZeroPage, 2, 3, ldaExecute))
	addIns(NewInstruction(0xB5, "LDA", "Load Accumulator with Memory", ZeroPageX, 2, 4, ldaExecute))
	addIns(NewInstruction(0xAD, "LDA", "Load Accumulator with Memory", Absolute, 3, 4, ldaExecute))
	addIns(NewInstruction(0xBD, "LDA", "Load Accumulator with Memory", AbsoluteX, 3, 4, ldaExecute)) //*
	addIns(NewInstruction(0xB9, "LDA", "Load Accumulator with Memory", AbsoluteY, 3, 4, ldaExecute)) //*
	addIns(NewInstruction(0xA1, "LDA", "Load Accumulator with Memory", IndirectX, 2, 6, ldaExecute))
	addIns(NewInstruction(0xB1, "LDA", "Load Accumulator with Memory", IndirectY, 2, 5, ldaExecute)) //*

	// LDX
	addIns(NewInstruction(0xA2, "LDX", "Load Index X with Memory", Immediate, 2, 2, ldxExecute))
	addIns(NewInstruction(0xA6, "LDX", "Load Index X with Memory", ZeroPage, 2, 3, ldxExecute))
	addIns(NewInstruction(0xB6, "LDX", "Load Index X with Memory", ZeroPageY, 2, 4, ldxExecute))
	addIns(NewInstruction(0xAE, "LDX", "Load Index X with Memory", Absolute, 3, 4, ldxExecute))
	addIns(NewInstruction(0xBE, "LDX", "Load Index X with Memory", AbsoluteY, 3, 4, ldxExecute)) //*

	// LDY
	addIns(NewInstruction(0xA0, "LDY", "Load Index Y with Memory", Immediate, 2, 2, ldyExecute))
	addIns(NewInstruction(0xA4, "LDY", "Load Index Y with Memory", ZeroPage, 2, 3, ldyExecute))
	addIns(NewInstruction(0xB4, "LDY", "Load Index Y with Memory", ZeroPageX, 2, 4, ldyExecute))
	addIns(NewInstruction(0xAC, "LDY", "Load Index Y with Memory", Absolute, 3, 4, ldyExecute))
	addIns(NewInstruction(0xBC, "LDY", "Load Index Y with Memory", AbsoluteX, 3, 4, ldyExecute)) //*

	// LSR
	addIns(NewInstruction(0x4A, "LSR", "Shift One Bit Right", Accumulator, 1, 2, lsrExecute))
	addIns(NewInstruction(0x46, "LSR", "Shift One Bit Right", ZeroPage, 2, 5, lsrExecute))
	addIns(NewInstruction(0x56, "LSR", "Shift One Bit Right", ZeroPageX, 2, 6, lsrExecute))
	addIns(NewInstruction(0x4E, "LSR", "Shift One Bit Right", Absolute, 3, 6, lsrExecute))
	addIns(NewInstruction(0x5E, "LSR", "Shift One Bit Right", AbsoluteX, 3, 7, lsrExecute))

	// NOP
	addIns(NewInstruction(0xEA, "NOP", "No Operation", Implied, 1, 2, nopExecute))

	// ORA
	addIns(NewInstruction(0x09, "ORA", "OR Memory with Accumulator", Immediate, 2, 2, oraExecute))
	addIns(NewInstruction(0x05, "ORA", "OR Memory with Accumulator", ZeroPage, 2, 3, oraExecute))
	addIns(NewInstruction(0x15, "ORA", "OR Memory with Accumulator", ZeroPageX, 2, 4, oraExecute))
	addIns(NewInstruction(0x0D, "ORA", "OR Memory with Accumulator", Absolute, 3, 4, oraExecute))
	addIns(NewInstruction(0x1D, "ORA", "OR Memory with Accumulator", AbsoluteX, 3, 4, oraExecute)) //*
	addIns(NewInstruction(0x19, "ORA", "OR Memory with Accumulator", AbsoluteY, 3, 4, oraExecute)) //*
	addIns(NewInstruction(0x01, "ORA", "OR Memory with Accumulator", IndirectX, 2, 6, oraExecute))
	addIns(NewInstruction(0x11, "ORA", "OR Memory with Accumulator", IndirectY, 2, 5, oraExecute)) //*

	// PHA
	addIns(NewInstruction(0x48, "PHA", "Push Accumulator on Stack", Implied, 1, 3, phaExecute))

	// PHP
	addIns(NewInstruction(0x08, "PHP", "Push Processor Satus on Stack", Implied, 1, 3, phpExecute))

	// PLA
	addIns(NewInstruction(0x68, "PLA", "Pull Accumulator from Stack", Implied, 1, 4, plaExecute))

	// PLP
	addIns(NewInstruction(0x28, "PLP", "Pull Processor Status from Stack", Implied, 1, 4, plpExecute))

	// ROL
	addIns(NewInstruction(0x2A, "ROL", "Rotate One Bit Left", Accumulator, 1, 2, rolExecute))
	addIns(NewInstruction(0x26, "ROL", "Rotate One Bit Left", ZeroPage, 2, 5, rolExecute))
	addIns(NewInstruction(0x36, "ROL", "Rotate One Bit Left", ZeroPageX, 2, 6, rolExecute))
	addIns(NewInstruction(0x2E, "ROL", "Rotate One Bit Left", Absolute, 3, 6, rolExecute))
	addIns(NewInstruction(0x3E, "ROL", "Rotate One Bit Left", AbsoluteX, 3, 7, rolExecute))

	// ROR
	addIns(NewInstruction(0x6A, "ROR", "Rotate One Bit Right", Accumulator, 1, 2, rorExecute))
	addIns(NewInstruction(0x66, "ROR", "Rotate One Bit Right", ZeroPage, 2, 5, rorExecute))
	addIns(NewInstruction(0x76, "ROR", "Rotate One Bit Right", ZeroPageX, 2, 6, rorExecute))
	addIns(NewInstruction(0x6E, "ROR", "Rotate One Bit Right", Absolute, 3, 6, rorExecute))
	addIns(NewInstruction(0x7E, "ROR", "Rotate One Bit Right", AbsoluteX, 3, 7, rorExecute))

	// RTI
	addIns(NewInstruction(0x40, "RTI", "Return from Interrupt", Implied, 1, 6, rtiExecute))

	// RTS
	addIns(NewInstruction(0x60, "RTS", "Return from Subroutine", Immediate, 1, 6, rtsExecute))

	// SBC
	addIns(NewInstruction(0xE9, "SBC", "Subtract Memory from Accum with Borrow", Immediate, 2, 2, sbcExecute))
	addIns(NewInstruction(0xE5, "SBC", "Subtract Memory from Accum with Borrow", ZeroPage, 2, 3, sbcExecute))
	addIns(NewInstruction(0xF5, "SBC", "Subtract Memory from Accum with Borrow", ZeroPageX, 2, 4, sbcExecute))
	addIns(NewInstruction(0xED, "SBC", "Subtract Memory from Accum with Borrow", Absolute, 2, 4, sbcExecute))
	addIns(NewInstruction(0xFD, "SBC", "Subtract Memory from Accum with Borrow", AbsoluteX, 3, 4, sbcExecute)) //*
	addIns(NewInstruction(0xF9, "SBC", "Subtract Memory from Accum with Borrow", AbsoluteY, 3, 4, sbcExecute)) //*
	addIns(NewInstruction(0xE1, "SBC", "Subtract Memory from Accum with Borrow", IndirectX, 2, 6, sbcExecute))
	addIns(NewInstruction(0xF1, "SBC", "Subtract Memory from Accum with Borrow", IndirectY, 2, 5, sbcExecute)) //*

	// SEC
	addIns(NewInstruction(0x38, "SEC", "Set Carry Flag", Implied, 1, 2, secExecute))

	// SED
	addIns(NewInstruction(0xF8, "SED", "Set Decimal Flag", Implied, 1, 2, sedExecute))

	// SEI
	addIns(NewInstruction(0x7B, "SEI", "Set Interrupt Disable Status", Implied, 1, 2, seiExecute))

	// STA
	addIns(NewInstruction(0x85, "STA", "Store Accumulator in Memory", ZeroPage, 2, 3, staExecute))
	addIns(NewInstruction(0x95, "STA", "Store Accumulator in Memory", ZeroPageX, 2, 4, staExecute))
	addIns(NewInstruction(0x8D, "STA", "Store Accumulator in Memory", Absolute, 3, 4, staExecute))
	addIns(NewInstruction(0x9D, "STA", "Store Accumulator in Memory", AbsoluteX, 3, 5, staExecute))
	addIns(NewInstruction(0x99, "STA", "Store Accumulator in Memory", AbsoluteY, 3, 5, staExecute))
	addIns(NewInstruction(0x81, "STA", "Store Accumulator in Memory", IndirectX, 2, 6, staExecute))
	addIns(NewInstruction(0x91, "STA", "Store Accumulator in Memory", IndirectY, 2, 6, staExecute))

	// STX
	addIns(NewInstruction(0x86, "STX", "Store Index X in Memory", ZeroPage, 2, 3, stxExecute))
	addIns(NewInstruction(0x96, "STX", "Store Index X in Memory", ZeroPageY, 2, 4, stxExecute))
	addIns(NewInstruction(0x8E, "STX", "Store Index X in Memory", Absolute, 3, 4, stxExecute))

	// STY
	addIns(NewInstruction(0x84, "STY", "Store Index Y in Memory", ZeroPage, 2, 3, styExecute))
	addIns(NewInstruction(0x94, "STY", "Store Index Y in Memory", ZeroPageX, 2, 4, styExecute))
	addIns(NewInstruction(0x8C, "STY", "Store Index Y in Memory", Absolute, 3, 4, styExecute))

	// TAX
	addIns(NewInstruction(0xAA, "TAX", "Transfer Accumulator to Index X", Implied, 1, 2, taxExecute))

	// TAY
	addIns(NewInstruction(0xAB, "TAY", "Transfer Accumulator to Index Y", Implied, 1, 2, tayExecute))

	// TSX
	addIns(NewInstruction(0xBA, "TSX", "Transfer Stack Pointer to Index X", Implied, 1, 2, tsxExecute))

	// TXA
	addIns(NewInstruction(0x8A, "TXA", "Transfer Index X to Accumulator", Implied, 1, 2, txaExecute))

	// TXS
	addIns(NewInstruction(0x9A, "TXS", "Transfer Index X to Stack Register", Implied, 1, 2, txsExecute))

	// TYA
	addIns(NewInstruction(0x9B, "TYA", "Transfer Index Y to Accumulator", Implied, 1, 2, tyaExecute))

}
