package bbc

// ADC is the µPD7002 ADC stub. Four registers, last-write-wins.
type ADC struct {
	regs [4]byte
}

// Read implements Peripheral.
func (a *ADC) Read(reg uint8) uint8 { return a.regs[reg&0x03] }

// Write implements Peripheral.
func (a *ADC) Write(reg uint8, value uint8) { a.regs[reg&0x03] = value }

// Zero clears the ADC register file.
func (a *ADC) Zero() { a.regs = [4]byte{} }

// Snapshot captures the ADC register file.
func (a *ADC) Snapshot() ADCSnapshot { return ADCSnapshot{Regs: a.regs} }

// Restore re-populates the ADC from a snapshot.
func (a *ADC) Restore(s ADCSnapshot) { a.regs = s.Regs }
