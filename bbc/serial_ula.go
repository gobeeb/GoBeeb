package bbc

// SerialULA is the serial ULA stub. One register, last-write-wins,
// mirrored across $FE10–$FE1F.
type SerialULA struct {
	regs [1]byte
}

// Read implements Peripheral.
func (s *SerialULA) Read(_ uint8) uint8 { return s.regs[0] }

// Write implements Peripheral.
func (s *SerialULA) Write(_ uint8, value uint8) { s.regs[0] = value }

// Zero clears the serial ULA register.
func (s *SerialULA) Zero() { s.regs = [1]byte{} }

// Snapshot captures the serial ULA state.
func (s *SerialULA) Snapshot() SerialULASnapshot { return SerialULASnapshot{Regs: s.regs} }

// Restore re-populates the serial ULA from a snapshot.
func (s *SerialULA) Restore(snap SerialULASnapshot) { s.regs = snap.Regs }
