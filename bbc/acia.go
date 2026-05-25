package bbc

// ACIA is the 6850 ACIA stub. Four registers, last-write-wins.
// Real ACIA decoding (status / data / control / transmit) is
// Phase 005+ work.
type ACIA struct {
	regs [4]byte
}

// Read implements Peripheral.
func (a *ACIA) Read(reg uint8) uint8 { return a.regs[reg&0x03] }

// Write implements Peripheral.
func (a *ACIA) Write(reg uint8, value uint8) { a.regs[reg&0x03] = value }

// Zero clears the ACIA's register file.
func (a *ACIA) Zero() { a.regs = [4]byte{} }

// Snapshot captures the ACIA state.
func (a *ACIA) Snapshot() ACIASnapshot { return ACIASnapshot{Regs: a.regs} }

// Restore re-populates the ACIA from a snapshot.
func (a *ACIA) Restore(s ACIASnapshot) { a.regs = s.Regs }
