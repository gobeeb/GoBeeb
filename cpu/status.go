package cpu

// Status represents the Processor Status Register
type Status uint8

const (
	C Status = 1 << iota // Carry
	Z                    // Zero Result
	I                    // Interrupt disable
	D                    // Decimal mode
	B                    // Break command
	Unused
	V // Overflow O?
	N // Negative Result
)

func (s Status) Set(flag Status) {
	s = s | flag
}

func (s Status) Clear(flag Status) {
	s = s &^ flag
}

func (s Status) Toggle(flag Status) {
	s = s ^ flag
}

func (s Status) Has(flag Status) bool {
	return s&flag != 0
}
