package cpu

type ExecuteFunc func(i *Instruction, cpu *CPU) error

type Instruction struct {
	OpCode      uint8
	Symbol      string
	Description string
	Mode        AddressingMode
	Cycles      int
	Bytes       int
	Execute     ExecuteFunc
}

func NewInstruction(opcode uint8, symbol string, desc string, mode AddressingMode, bytes int, cycles int, exec ExecuteFunc) Instruction {
	return Instruction{
		OpCode:      opcode,
		Symbol:      symbol,
		Description: desc,
		Mode:        mode,
		Cycles:      cycles,
		Bytes:       bytes,
		Execute:     exec,
	}
}

func adcExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func andExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func aslExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func bccExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func bcsExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func beqExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func bitExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func bmiExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func bneExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func bplExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func brkExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func bvcExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func bvsExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func clcExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func cldExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func cliExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func clvExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func cmpExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func cpxExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func cpyExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func decExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func dexExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func deyExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func eorExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func incExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func inxExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func inyExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func jmpExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func jsrExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func ldaExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func ldxExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func ldyExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func lsrExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func nopExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func oraExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func phaExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func phpExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func plaExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func plpExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func rolExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func rorExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func rtiExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func rtsExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func sbcExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func secExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func sedExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func seiExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func staExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func stxExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func styExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func taxExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func tayExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func tsxExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func txaExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func txsExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}

func tyaExecute(i *Instruction, cpu *CPU) error {
	return ErrNotImplemented
}
