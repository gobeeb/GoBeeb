package cpu

type ldaInstruction struct{}

func (i *ldaInstruction) Execute(cpu *CPU) error {
	return nil
}
