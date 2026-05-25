package bbc

// Econet is the 68B54 ADLC / Econet stub. Phase 002 returns $FF
// for all reads (FR-008); writes are captured for snapshot but
// have no observable effect.
type Econet struct {
	regs [4]byte
}

// Read implements Peripheral; always returns $FF in Phase 002.
func (e *Econet) Read(_ uint8) uint8 { return openBus }

// Write implements Peripheral.
func (e *Econet) Write(reg uint8, value uint8) { e.regs[reg&0x03] = value }

// Zero clears the Econet register file.
func (e *Econet) Zero() { e.regs = [4]byte{} }

// Snapshot captures the Econet register file.
func (e *Econet) Snapshot() EconetSnapshot { return EconetSnapshot{Regs: e.regs} }

// Restore re-populates the Econet from a snapshot.
func (e *Econet) Restore(s EconetSnapshot) { e.regs = s.Regs }
