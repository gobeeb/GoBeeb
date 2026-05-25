package bbc

// VIA is the 6522 VIA stub used by both the System VIA and the
// User VIA. 16 registers, last-write-wins for all 16. Phase 002
// does NOT implement timers, shift registers, port-direction
// registers, or interrupt flags; those land in a later phase
// when VIA-driven peripherals come online.
type VIA struct {
	regs [16]byte
}

// Read implements Peripheral.
func (v *VIA) Read(reg uint8) uint8 { return v.regs[reg&0x0F] }

// Write implements Peripheral.
func (v *VIA) Write(reg uint8, value uint8) { v.regs[reg&0x0F] = value }

// Zero clears the VIA register file.
func (v *VIA) Zero() { v.regs = [16]byte{} }

// Snapshot captures the VIA register file.
func (v *VIA) Snapshot() VIASnapshot { return VIASnapshot{Regs: v.regs} }

// Restore re-populates the VIA from a snapshot.
func (v *VIA) Restore(s VIASnapshot) { v.regs = s.Regs }
