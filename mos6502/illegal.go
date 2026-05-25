package mos6502

// IllegalOpcodeHook is invoked whenever the CPU executes a byte that is
// not one of the 151 documented NMOS 6502 opcodes (FR-019).
//
// The CPU treats the byte as a single-byte, 2-cycle NOP regardless of
// whether a hook is registered. The hook is purely observability: its
// return value (there is none) does not influence CPU behaviour and the
// CPU does not pause to wait for the hook.
//
// pc is the program counter at the address the opcode byte was fetched
// from — the address of the illegal byte itself. The CPU's live PC has
// already advanced past it by the time the hook is called.
type IllegalOpcodeHook func(pc uint16, opcode uint8)

// illegalNOP is the shared dispatch-table slot for every undocumented
// opcode. Single-byte, 2-cycle NOP: the opcode-fetch consumed cycle 1
// and the address (PC) and we still owe one further cycle of bus
// activity; on real NMOS the second cycle is a dummy fetch of the next
// byte without advancing PC.
func illegalNOP(c *CPU) {
	// We are called *after* the opcode fetch advanced PC by 1. The
	// pre-advance PC was therefore c.PC - 1. fetched holds the opcode
	// byte the dispatcher just consumed.
	if c.onIllegalOp != nil {
		c.onIllegalOp(c.PC-1, c.fetched)
	}
	// Second cycle: dummy read of PC without advancing it.
	_ = c.read(c.PC)
}
