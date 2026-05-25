package mos6502

// CPU is one emulated NMOS 6502 processor. Create via [New], drive via
// [CPU.Step] / [CPU.StepCycle] / [CPU.Run], control via [CPU.AssertReset],
// [CPU.AssertIRQ], [CPU.AssertNMI], [CPU.DeassertNMI], [CPU.SetRDY].
//
// The architectural register fields are exported so consumers can
// inspect them directly when convenient (debuggers, tests). Production
// hosts should prefer [CPU.Registers] / [CPU.SetRegisters].
//
// A CPU is NOT safe for concurrent use by multiple goroutines
// (Assumption A8). Callers are responsible for serialisation.
type CPU struct {
	A  uint8  // Accumulator
	X  uint8  // Index X
	Y  uint8  // Index Y
	SP uint8  // Stack pointer (live stack address = $0100 | SP)
	PC uint16 // Program counter
	P  uint8  // Processor status (layout NV-BDIZC; see Flag* constants)

	mem Memory

	// Pending signals. (FR-008, FR-010, FR-011)
	irqLine      bool
	nmiPending   bool
	nmiPrev      bool
	resetPending bool

	// Ready line. true = ready (default), false = stalls reads. (FR-023)
	rdy bool

	// Cumulative bus-cycle counter. (FR-005, FR-018)
	cycles uint64

	// Optional observers. nil = zero cost.
	trace       *Trace
	onIllegalOp IllegalOpcodeHook

	// Internal scratch for sub-cycle execution. Not part of public
	// architectural state.
	addr    uint16
	fetched uint8
}

// Registers is a snapshot of the architectural register state at an
// instruction boundary. (FR-007)
type Registers struct {
	A      uint8  // Accumulator
	X      uint8  // Index X
	Y      uint8  // Index Y
	SP     uint8  // Stack pointer
	PC     uint16 // Program counter
	P      uint8  // Processor status (see Flag* constants)
	Cycles uint64 // Cumulative bus-cycle count
}

// New constructs a fresh CPU bound to mem. The CPU is returned with a
// pending RESET: the next call to [CPU.Step], [CPU.StepCycle], or
// [CPU.Run] will perform the 7-cycle RESET sequence and start fetching
// from the vector at $FFFC/$FFFD.
//
// New does not invoke any methods on mem; Memory is only touched once
// the CPU is stepped.
func New(mem Memory) *CPU {
	return &CPU{
		mem:          mem,
		rdy:          true,
		resetPending: true,
	}
}

// AssertReset latches a pending RESET. The reset is processed at the
// start of the next bus cycle the CPU runs. RESET takes priority over
// IRQ and NMI.
func (c *CPU) AssertReset() { c.resetPending = true }

// AssertIRQ sets the IRQ line to the given level. IRQ is
// level-sensitive: while the line is true and the I flag is clear, the
// CPU will service the IRQ at the next instruction boundary.
// AssertIRQ(false) deasserts the line.
func (c *CPU) AssertIRQ(level bool) { c.irqLine = level }

// AssertNMI raises an edge on the NMI line. NMI is edge-triggered; a
// single call latches one pending service. The CPU clears the latch
// once it services the NMI. A second AssertNMI before the host has
// deasserted (via [CPU.DeassertNMI]) and re-asserted will NOT queue a
// second service. (FR-011, FR-022)
func (c *CPU) AssertNMI() {
	if !c.nmiPrev {
		c.nmiPending = true
	}
	c.nmiPrev = true
}

// DeassertNMI clears the host-side NMI line. The next [CPU.AssertNMI]
// will be observed as a new edge.
func (c *CPU) DeassertNMI() { c.nmiPrev = false }

// SetRDY drives the RDY input. ready=false stalls READ bus cycles: the
// CPU will repeat the current read on the next call to
// [CPU.StepCycle]/[CPU.Step]/[CPU.Run] without advancing any
// architectural state. Writes proceed even when RDY is asserted — this
// is the documented NMOS limitation. The default (post-[New]) state is
// ready=true. (FR-023)
func (c *CPU) SetRDY(ready bool) { c.rdy = ready }

// SetIllegalOpcodeHook registers a hook invoked when the CPU executes
// an undocumented NMOS opcode byte. The CPU will still behave as a
// single-byte, 2-cycle NOP for the byte; the hook is purely
// observational. Pass nil to remove the hook. (FR-019)
func (c *CPU) SetIllegalOpcodeHook(h IllegalOpcodeHook) { c.onIllegalOp = h }

// SetTrace attaches an optional bus-trace recorder. While attached,
// every bus cycle is appended to t. Pass nil to detach; detached is
// the zero-cost default. (SC-008)
func (c *CPU) SetTrace(t *Trace) { c.trace = t }

// Registers returns a snapshot of the architectural register state at
// the most recent instruction boundary.
func (c *CPU) Registers() Registers {
	return Registers{
		A:      c.A,
		X:      c.X,
		Y:      c.Y,
		SP:     c.SP,
		PC:     c.PC,
		P:      c.P,
		Cycles: c.cycles,
	}
}

// SetRegisters overwrites the architectural register state. Intended
// for test harnesses and debuggers; production hosts should use
// [CPU.AssertReset] for fresh-state initialisation. Calling
// SetRegisters in the middle of a multi-cycle instruction is undefined.
//
// Setting registers also clears any pending RESET, so a host that
// hand-initialises state can bypass the 7-cycle reset sequence.
func (c *CPU) SetRegisters(r Registers) {
	c.A = r.A
	c.X = r.X
	c.Y = r.Y
	c.SP = r.SP
	c.PC = r.PC
	c.P = r.P
	c.cycles = r.Cycles
	c.resetPending = false
}

// ──────────────────────────────────────────────────────────────────
// Bus-cycle primitives (FR-006, FR-018, FR-023)
//
// RDY honouring note (v1): the NMOS RDY pin can stretch any read
// cycle. In this implementation the entire sub-cycle sequence of an
// instruction runs back-to-back inside Step / StepCycle, so the
// natural enforcement point is the instruction boundary (Step entry).
// Sub-instruction RDY stretching is out of scope for v1 — Step / Run
// gate on RDY at entry; read / write / fetch assume rdy=true once
// dispatched.
// ──────────────────────────────────────────────────────────────────

// read performs one bus read cycle at addr.
func (c *CPU) read(addr uint16) uint8 {
	v := c.mem.Read(addr)
	c.cycles++
	if c.trace != nil {
		c.trace.append(BusEvent{Cycle: c.cycles, Addr: addr, Value: v, Kind: BusRead})
	}
	c.fetched = v
	return v
}

// write performs one bus write cycle at addr. Writes proceed even when
// RDY is asserted-low — this is the documented NMOS limitation. (FR-023)
func (c *CPU) write(addr uint16, value uint8) {
	c.mem.Write(addr, value)
	c.cycles++
	if c.trace != nil {
		c.trace.append(BusEvent{Cycle: c.cycles, Addr: addr, Value: value, Kind: BusWrite})
	}
}

// fetch reads at PC and post-increments PC.
func (c *CPU) fetch() uint8 {
	v := c.read(c.PC)
	c.PC++
	return v
}

// push writes value to $0100|SP then decrements SP.
func (c *CPU) push(value uint8) {
	c.write(0x0100|uint16(c.SP), value)
	c.SP--
}

// pull increments SP then reads from $0100|SP.
func (c *CPU) pull() uint8 {
	c.SP++
	return c.read(0x0100 | uint16(c.SP))
}

// ──────────────────────────────────────────────────────────────────
// Execution surface (FR-008)
// ──────────────────────────────────────────────────────────────────

// Step executes exactly one full 6502 instruction (or, if a RESET,
// NMI, or IRQ is pending, services the corresponding interrupt vector
// — also one "instruction" from the architectural point of view).
// Returns the cumulative cycle count after the instruction.
// (FR-007, FR-008, FR-010, FR-011)
//
// Pending signals are evaluated at the instruction boundary in
// priority order: RESET ≻ NMI ≻ IRQ (the latter only if the I flag
// is clear).
//
// If RDY is asserted-low at entry, Step returns immediately without
// advancing any architectural state. Sub-instruction RDY stretching
// is not supported in v1.
func (c *CPU) Step() uint64 {
	if !c.rdy {
		return c.cycles
	}
	switch {
	case c.resetPending:
		enterInterrupt(c, resetInterrupt)
		return c.cycles
	case c.nmiPending:
		enterInterrupt(c, nmiInterrupt)
		return c.cycles
	case c.irqLine && !c.flag(FlagInterrupt):
		enterInterrupt(c, irqInterrupt)
		return c.cycles
	}
	op := c.fetch()
	opcodeTable[op](c)
	return c.cycles
}

// StepCycle in v1 is a synonym for Step (one whole instruction). A
// future revision may slice instructions into per-bus-cycle steps;
// existing callers will continue to work because the cumulative cycle
// count remains correct.
func (c *CPU) StepCycle() uint64 { return c.Step() }

// Run executes whole instructions until at least cycleBudget bus
// cycles have been consumed since the call. Run never splits an
// instruction across calls — once started, an instruction always
// runs to completion, so the final cumulative cycle count may exceed
// (start + cycleBudget) by up to the cost of the last instruction.
func (c *CPU) Run(cycleBudget uint64) uint64 {
	deadline := c.cycles + cycleBudget
	for c.cycles < deadline {
		if !c.rdy {
			return c.cycles
		}
		c.Step()
	}
	return c.cycles
}
