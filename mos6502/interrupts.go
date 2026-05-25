package mos6502

// interruptKind tags the trigger for an interrupt-entry sequence.
type interruptKind uint8

const (
	irqInterrupt interruptKind = iota
	nmiInterrupt
	resetInterrupt
	brkInterrupt
)

// enterInterrupt implements the 7-cycle NMOS interrupt-entry routine
// shared by RESET, IRQ, NMI, and BRK.
//
// Calling convention:
//
//   - irqInterrupt / nmiInterrupt / resetInterrupt: the caller (Step
//     entry) has NOT advanced PC. enterInterrupt issues 2 dummy reads
//     at PC and runs all 7 cycles itself.
//   - brkInterrupt: the caller (brk opcode handler) has already
//     consumed 2 cycles (opcode fetch via dispatcher + padding fetch
//     via the handler). enterInterrupt runs only cycles 3–7.
//
// NMI-hijack (FR-022) is implemented by deferring the vector decision
// until after the cycle-5 push of P: if NMI is pending at that point
// (even one set mid-BRK or mid-IRQ), the vector used is $FFFA. The
// pushed B bit retains the original cause: 1 for BRK, 0 for IRQ/NMI.
func enterInterrupt(c *CPU, kind interruptKind) {
	// Cycles 1–2: dummy fetches at PC for IRQ/NMI/RESET. BRK's caller
	// has already paid these.
	if kind != brkInterrupt {
		_ = c.read(c.PC)
		_ = c.read(c.PC)
	}

	// Pushed P: B is set for BRK only; U is always set in pushed copy.
	pushedP := c.P | FlagUnused
	if kind == brkInterrupt {
		pushedP |= FlagBreak
	}

	// Cycles 3–5: push PCH, PCL, P. RESET fakes the pushes (read
	// instead of write) but still decrements SP.
	if kind == resetInterrupt {
		_ = c.read(0x0100 | uint16(c.SP))
		c.SP--
		_ = c.read(0x0100 | uint16(c.SP))
		c.SP--
		_ = c.read(0x0100 | uint16(c.SP))
		c.SP--
	} else {
		c.push(uint8(c.PC >> 8))
		c.push(uint8(c.PC))
		c.push(pushedP)
	}

	// Live I is set unconditionally on entry.
	c.setFlag(FlagInterrupt, true)
	c.setFlag(FlagUnused, true)

	// Determine vector (with NMI hijack of BRK / IRQ).
	var vec uint16
	switch kind {
	case nmiInterrupt:
		vec = 0xFFFA
	case resetInterrupt:
		vec = 0xFFFC
	case brkInterrupt, irqInterrupt:
		vec = 0xFFFE
		if c.nmiPending {
			vec = 0xFFFA // hijacked
			// The NMI edge is consumed by the hijack.
			c.nmiPending = false
		}
	}

	// Cycles 6–7: fetch vector lo, hi.
	lo := c.read(vec)
	hi := c.read(vec + 1)
	c.PC = uint16(lo) | uint16(hi)<<8

	// Edge-clear for serviced NMI.
	if kind == nmiInterrupt {
		c.nmiPending = false
	}
	if kind == resetInterrupt {
		c.resetPending = false
	}
}
