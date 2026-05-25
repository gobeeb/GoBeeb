// Package mos6502 emulates the NMOS MOS Technology 6502 microprocessor
// with sub-cycle / bus-cycle accuracy.
//
// It is the CPU core of the wider GoBeeb BBC Model B emulator but has no
// dependency on any of the BBC-specific peripherals: a consumer supplies a
// [Memory] implementation, instantiates a [CPU] with [New], and drives it
// either one full instruction at a time via [CPU.Step] or one bus cycle at
// a time via [CPU.StepCycle].
//
// See specs/001-cpu-6502-core/spec.md for the full functional contract
// (FR-001..FR-023) and specs/001-cpu-6502-core/quickstart.md for an
// end-to-end example.
//
// Minimal usage:
//
//	var ram mos6502.FlatRAM // any type satisfying Memory
//	cpu := mos6502.New(&ram)
//	cpu.AssertReset()
//	for !done {
//	    cpu.Step()
//	}
package mos6502
