package bbc

// crtcRegCount is the number of addressable 6845 registers (R0..R17).
const crtcRegCount = 18

// CRTC is the 6845 CRTC stub. Phase 002 implements only the
// index-then-data addressing required by FR-010: writes to $FE00
// (reg==0) latch the register index; writes to $FE01 (reg==1) store
// into the indexed register; reads of $FE01 return the indexed
// register. Scanline timing, cursor blink, and frame generation are
// Phase 003.
type CRTC struct {
	selected uint8 // 0..17
	regs     [crtcRegCount]byte
}

// Read implements Peripheral. reg==0 returns open-bus $FF
// (write-only address-register slot); reg==1 returns the byte at
// the currently-selected register index.
func (c *CRTC) Read(reg uint8) uint8 {
	if reg&1 == 0 {
		return openBus
	}
	return c.regs[c.selected]
}

// Write implements Peripheral. reg==0 latches the register index
// masked to 0..17; reg==1 stores value at the indexed register.
func (c *CRTC) Write(reg uint8, value uint8) {
	if reg&1 == 0 {
		c.selected = value & 0x1F
		if c.selected >= crtcRegCount {
			c.selected %= crtcRegCount
		}
		return
	}
	c.regs[c.selected] = value
}

// Zero clears the CRTC's register file and index.
func (c *CRTC) Zero() {
	c.selected = 0
	for i := range c.regs {
		c.regs[i] = 0
	}
}

// Snapshot captures the CRTC's register state.
func (c *CRTC) Snapshot() CRTCSnapshot {
	return CRTCSnapshot{Regs: c.regs, Selected: c.selected}
}

// Restore re-populates the CRTC from a snapshot.
func (c *CRTC) Restore(s CRTCSnapshot) {
	c.regs = s.Regs
	c.selected = s.Selected
	if c.selected >= crtcRegCount {
		c.selected %= crtcRegCount
	}
}
