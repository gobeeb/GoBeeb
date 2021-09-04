package cpu

// Docs: http://www.obelisk.me.uk/6502/addressing.html

type AddressingMode int

const (
	Implied AddressingMode = iota + 1
	Accumulator
	Immediate
	ZeroPage
	ZeroPageX
	ZeroPageY
	Relative
	Absolute
	AbsoluteX
	AbsoluteY
	Indirect
	IndirectX
	IndirectY
)

// Implicit

// Accumulator
// Operate directly on accumulatpr. Special operand A for example:@
// LSR A or ROR A

// Immediate
// Specify an 8-bit constant value. Normally indicated by =
// LDA #10
// LDX #LO LABEL

// Zero page
// Only has 8 bit address operand (i.e. 1 byte). Limits to addressoing first 256 bytes ($0000-$00FF)
// LDA $00

// Zero Page,X
// Zero Page,Y
// Relative
// Absolute
// Absolute,X
// Absolute,Y
// Indirect
// Indexd indirect
// Indirect indexed
