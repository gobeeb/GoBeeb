package bbc

// VideoULA is the video ULA stub. Two registers (control +
// palette latch), last-write-wins, mirrored across $FE20–$FE2F.
// Real palette decoding is Phase 003.
type VideoULA struct {
	regs [2]byte
}

// Read implements Peripheral.
func (v *VideoULA) Read(reg uint8) uint8 { return v.regs[reg&0x01] }

// Write implements Peripheral.
func (v *VideoULA) Write(reg uint8, value uint8) { v.regs[reg&0x01] = value }

// Zero clears the video ULA registers.
func (v *VideoULA) Zero() { v.regs = [2]byte{} }

// Snapshot captures the video ULA state.
func (v *VideoULA) Snapshot() VideoULASnapshot { return VideoULASnapshot{Regs: v.regs} }

// Restore re-populates the video ULA from a snapshot.
func (v *VideoULA) Restore(s VideoULASnapshot) { v.regs = s.Regs }
