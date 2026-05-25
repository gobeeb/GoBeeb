package bbc

// UnmappedAccessHook is invoked on every read or write the BBC
// memory map classifies as unmapped: FRED/JIM offsets not claimed
// by a peripheral, SHEILA offsets not in the FR-008 routing table,
// and sideways-window reads ($8000–$BFFF) when the selected bank
// has no loaded image.
//
// For writes, value is the byte the CPU attempted to write. For
// reads, value is the open-bus byte the machine is about to return
// ($FF in Phase 002). The hook is called synchronously and must
// return quickly (the CPU is paused inside mos6502.CPU.Step until
// it returns).
type UnmappedAccessHook func(addr uint16, write bool, value uint8)

// SetUnmappedAccessHook installs (or removes, with nil) the hook
// invoked on every unmapped bus access. The default after New is
// nil (silent). A nil hook costs a single nil-check on the cold
// path; mapped accesses pay nothing (FR-029).
func (m *Machine) SetUnmappedAccessHook(hook UnmappedAccessHook) {
	m.unmappedHook = hook
}

// fireUnmapped dispatches an unmapped access to the parent
// Machine's hook, if one is set. The single nil-check keeps the
// cold path branch-predictable and zero-allocation.
func (m *MemoryMap) fireUnmapped(addr uint16, write bool, value uint8) {
	if m.parent != nil && m.parent.unmappedHook != nil {
		m.parent.unmappedHook(addr, write, value)
	}
}
