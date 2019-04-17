package cpu

// little endian
// first 256 ($0000-$00FF) bytes are zero page
// second page ($0100-$01FF) system stack cannot relocated
// last 6 bytes of memory $FFAA to $FFFF
//                - NMI handler $FFFA/B
//                - power on reset location $FFFC/D
//                - BRK/IRQ handler $FFFE/F

// CPU represents a 6502 processor
type CPU struct {
	databus    uint8
	addressbus uint16

	PC uint16 // program counter
	SP byte   // stack pointer

	A byte // accumulator
	X byte // index register x
	Y byte // index register y

	// Status represents the Processor Status Register and represents the Carry (C), Zero Result (Z),
	// Interupt Disable (I), Decimal Mode (D), Break Command (B), Overflow (O), Negative Result (N) flags.
	// Its represented as a 8-bit register with the following bits:
	// N | V | | B | D | I | Z | C
	Status byte

	RAM *[]byte
	ROM *[]byte
}

func NewCPU(memory *[]byte) *CPU {
	return &CPU{Memory: memory}
}

//func (c *CPU) Read(address uint16) byte {
//	return 1
//}

func (c *CPU) Reset() {

}
