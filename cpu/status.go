package cpu

// Status represents the Processor Status Register and represents the Carry (C), Zero Result (Z),
// Interupt Disable (I), Decimal Mode (D), Break Command (B), Overflow (O), Negative Result (N) flags.
// Its represented as a 8-bit register with the following bits:
// N | V | | B | D | I | Z | C
type Status uint8

const (
	N Status = 1 << iota
	V
	Unused
	B
	D
	I
	Z
	C
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
