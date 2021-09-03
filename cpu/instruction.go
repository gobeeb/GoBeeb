package cpu

type Instruction interface {
	Execute(cpu *CPU) error
}
