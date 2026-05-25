package bbc

// FDC is the 1770 FDC stub. Phase 002 returns $FF for all reads
// and silently drops writes (FR-008). Full WD1770 behaviour
// (commands, status flags, DRQ, etc.) lands in Phase 007.
type FDC struct {
	regs [4]byte
}

// Read implements Peripheral; always returns $FF in Phase 002.
func (f *FDC) Read(_ uint8) uint8 { return openBus }

// Write implements Peripheral; stores the byte for snapshot
// purposes but applies no FDC semantics in Phase 002.
func (f *FDC) Write(reg uint8, value uint8) { f.regs[reg&0x03] = value }

// Zero clears the FDC register file.
func (f *FDC) Zero() { f.regs = [4]byte{} }

// Snapshot captures the FDC register file.
func (f *FDC) Snapshot() FDCSnapshot { return FDCSnapshot{Regs: f.regs} }

// Restore re-populates the FDC from a snapshot.
func (f *FDC) Restore(s FDCSnapshot) { f.regs = s.Regs }
