package bbc

// Tube is the Tube parasite-bus stub. Eight registers,
// last-write-wins. Real Tube semantics (FIFO ports, status
// registers, interrupts to the host) are out of scope for
// Phase 002.
type Tube struct {
	regs [8]byte
}

// Read implements Peripheral.
func (tu *Tube) Read(reg uint8) uint8 { return tu.regs[reg&0x07] }

// Write implements Peripheral.
func (tu *Tube) Write(reg uint8, value uint8) { tu.regs[reg&0x07] = value }

// Zero clears the Tube register file.
func (tu *Tube) Zero() { tu.regs = [8]byte{} }

// Snapshot captures the Tube register file.
func (tu *Tube) Snapshot() TubeSnapshot { return TubeSnapshot{Regs: tu.regs} }

// Restore re-populates the Tube from a snapshot.
func (tu *Tube) Restore(s TubeSnapshot) { tu.regs = s.Regs }
