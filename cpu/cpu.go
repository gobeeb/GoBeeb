package cpu

// little endian
// $0000-$00FF (256 bytes) bytes are zero page
// $0100-$01FF system stack cannot relocated
// $0200 - $FFF9 unused
// $FFFA - B - NMI handler
// $FFFC - D - power on reset location
// $FFFE - F - BRK/IRQ handler

// CPU represents a 6502 processor
type CPU struct {
	//databus    uint8
	//addressbus uint16

	PC uint16 // program counter
	SP byte   // stack pointer

	A byte // accumulator
	X byte // index register x or 1
	Y byte // index register y or 2

	// Status flags
	P Status

	memory []uint8
}

func NewCPU(memory []uint8) *CPU {
	c := &CPU{memory: memory}
	c.Reset()

	return c
}

func (c *CPU) Read(address uint16) uint8 {
	return c.memory[address]
}

func (c *CPU) Write(address uint16, value uint8) {
	c.memory[address] = value
}

func (c *CPU) Reset() {
	c.A = 0
	c.X = 0
	c.Y = 0

	c.P.Clear(C)
	c.P.Clear(Z)
	c.P.Set(I)
	c.P.Clear(D)
	c.P.Clear(B)
	c.P.Clear(V)
	c.P.Clear(N)

	c.SP = 0xfd
	c.PC = 0xfffc
}

func (c *CPU) Execute() error {
	for {
		c.Read(c.PC)
		c.PC++
		// TODO: handle opcode
	}
}
