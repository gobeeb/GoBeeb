package cpu

var opCodes = map[uint8]Instruction{}

func addIns(i Instruction) {
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
	addIns(NewInstruction(0xB4, "LDY", "Load Index Y with Memory", ZeroPageY, 2, 4, ldyExecute))
	addIns(NewInstruction(0xAC, "LDY", "Load Index Y with Memory", Absolute, 3, 4, ldyExecute))
	addIns(NewInstruction(0xBC, "LDY", "Load Index Y with Memory", AbsoluteY, 3, 4, ldyExecute)) //*
}
