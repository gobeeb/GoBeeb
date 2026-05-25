// Package mos6502 contract definitions.
//
// This file documents the public control + state surface of the CPU
// type as it will be exposed in `mos6502/cpu.go`. It is the
// authoritative reference for downstream consumers (the wider GoBeeb
// emulator, third-party imports, test harnesses). Spec references:
// FR-007, FR-008, FR-018, FR-022, FR-023; Research §1, §3.2.

package mos6502

// CPU is one emulated NMOS 6502 processor. It is created via New, fed
// instructions through a host-supplied Memory, and driven by a small
// set of control methods.
//
// A CPU is NOT safe for concurrent use by multiple goroutines (Assumption
// A8). The host is responsible for serialising calls.
type CPU struct {
	// (Internal fields — see data-model.md §1 for the full struct
	// layout. The public surface is method-based.)
}

// New constructs a fresh CPU bound to the given Memory. The CPU is
// returned with reset_pending = true: the *next* call to Step,
// StepCycle, or Run will perform the 7-cycle RESET sequence and start
// fetching from the vector at $FFFC/$FFFD.
//
// New does not invoke any methods on mem. Memory is only touched once
// the CPU is stepped.
func New(mem Memory) *CPU { return nil /* contract only */ }

// ──────────────────────────────────────────────────────────────────
// Control surface (FR-008)
// ──────────────────────────────────────────────────────────────────

// AssertReset latches a pending RESET. The reset is processed at the
// start of the next bus cycle the CPU runs. RESET takes priority over
// IRQ and NMI.
func (c *CPU) AssertReset() {}

// AssertIRQ sets the IRQ line to the given level. IRQ is
// level-sensitive: while the line is true and the I flag is clear,
// the CPU will service the IRQ at the next instruction boundary.
// Calling AssertIRQ(false) deasserts the line.
func (c *CPU) AssertIRQ(level bool) {}

// AssertNMI raises an edge on the NMI line. NMI is edge-triggered;
// a single call latches one pending service. The CPU clears the latch
// once it services the NMI. A second AssertNMI before the host has
// deasserted (via DeassertNMI) and re-asserted will NOT queue a
// second service — this matches real NMOS behaviour and is the
// foundation of the NMI-hijack semantics (FR-022).
func (c *CPU) AssertNMI() {}

// DeassertNMI clears the host-side NMI line. The next AssertNMI
// will be observed as a new edge.
func (c *CPU) DeassertNMI() {}

// SetRDY drives the RDY input (FR-023). ready=false (the asserted-low
// real-pin equivalent) stalls READ bus cycles: the CPU will repeat
// the current read on the next call to StepCycle/Step/Run without
// advancing any architectural state. Writes proceed even when RDY is
// asserted — this is the documented NMOS limitation. RDY is
// level-sensitive; the host re-evaluates it every cycle.
//
// The default (post-New) RDY state is ready=true (not stalled).
func (c *CPU) SetRDY(ready bool) {}

// SetIllegalOpcodeHook registers a hook invoked when the CPU executes
// an undocumented NMOS opcode byte (FR-019). The CPU will still
// behave as a single-byte, 2-cycle NOP for the byte; the hook is
// purely observational. Pass nil to remove the hook.
func (c *CPU) SetIllegalOpcodeHook(h IllegalOpcodeHook) {}

// SetTrace attaches an optional bus-trace recorder. While attached,
// every bus cycle is appended to t. Pass nil to detach; detached is
// the zero-cost default.
func (c *CPU) SetTrace(t *Trace) {}

// ──────────────────────────────────────────────────────────────────
// Execution surface (FR-008)
// ──────────────────────────────────────────────────────────────────

// StepCycle executes exactly one bus cycle (or repeats it if RDY is
// stalling a read). This is the foundational primitive that all other
// execution methods are built upon. It is what the host emulator
// will call when it needs to interleave CPU and video-ULA timing
// cycle-by-cycle. Returns the cumulative cycle count after the step.
func (c *CPU) StepCycle() uint64 { return 0 }

// Step executes exactly one full 6502 instruction (or, if a RESET or
// interrupt is pending, services that vector — also one "instruction"
// from the architectural point of view). Equivalent to calling
// StepCycle repeatedly until an instruction boundary is reached.
// Returns the cumulative cycle count after the instruction.
func (c *CPU) Step() uint64 { return 0 }

// Run executes whole instructions until at least cycleBudget bus
// cycles have been consumed since the call. Run never splits an
// instruction across calls — if the next instruction would push the
// consumed count above cycleBudget, Run stops at the current
// instruction boundary instead. Returns the cumulative cycle count
// after the call.
func (c *CPU) Run(cycleBudget uint64) uint64 { return 0 }

// ──────────────────────────────────────────────────────────────────
// State surface (FR-007)
// ──────────────────────────────────────────────────────────────────

// Registers returns a snapshot of the architectural register state
// at the most recent instruction boundary. The cycle field is the
// cumulative bus-cycle count.
func (c *CPU) Registers() Registers { return Registers{} }

// SetRegisters overwrites the architectural register state. This is
// intended for test harnesses and debuggers; production hosts should
// use AssertReset for fresh-state initialisation. Calling
// SetRegisters in the middle of a multi-cycle instruction is
// undefined.
func (c *CPU) SetRegisters(r Registers) {}

// Registers is the architectural state of a 6502.
type Registers struct {
	A      uint8
	X      uint8
	Y      uint8
	SP     uint8
	PC     uint16
	P      uint8  // status: NV-BDIZC
	Cycles uint64 // cumulative bus-cycle count
}
