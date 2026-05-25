package bbc

import "github.com/gobeeb/GoBeeb/mos6502"

// Machine is one emulated BBC Model B. It owns one mos6502.CPU,
// a BBC memory map, OS ROM + four sideways ROM banks, and the
// SHEILA stub peripherals.
//
// Machine is NOT goroutine-safe (FR-028). All public methods MUST
// be driven from the same goroutine. The Tick hot path takes no
// internal locks.
type Machine struct {
	cpu    *mos6502.CPU
	mmap   MemoryMap
	rom    RomBanks
	periph Peripherals

	unmappedHook UnmappedAccessHook
}

// New constructs a fresh Machine with no ROMs loaded. The
// underlying CPU is reset-pending; the first Tick (after
// LoadOSROM) performs the 7-cycle RESET sequence and fetches from
// $FFFC/$FFFD.
//
// Calling Reset, ColdReset, or Tick before LoadOSROM has no
// effect on a Machine without an OS ROM beyond returning
// ErrNoOSROM (Reset/ColdReset) or skipping the cycle (Tick — the
// CPU never advances because the RESET vector read goes through
// the memory map regardless; Reset() must be called first).
func New() *Machine {
	m := &Machine{}
	m.mmap.rom = &m.rom
	m.mmap.periph = &m.periph
	m.mmap.parent = m
	m.cpu = mos6502.New(&m.mmap)
	return m
}

// Reset performs a BBC BREAK-key soft reset. Preserves main RAM
// and every peripheral register file; clears the sideways bank
// latch to 0; asserts CPU reset so the next Tick re-fetches the
// RESET vector at $FFFC/$FFFD.
//
// Returns ErrNoOSROM if no OS ROM has been loaded.
func (m *Machine) Reset() error {
	if !m.rom.osLoaded {
		return ErrNoOSROM
	}
	m.rom.bank = 0
	m.cpu.AssertReset()
	return nil
}

// ColdReset performs a power-on reset. Zeros main RAM, zeros every
// peripheral register file, clears the sideways bank latch to 0,
// asserts CPU reset. Equivalent to constructing a fresh Machine
// with the same loaded ROM images.
//
// Returns ErrNoOSROM if no OS ROM has been loaded.
func (m *Machine) ColdReset() error {
	if !m.rom.osLoaded {
		return ErrNoOSROM
	}
	for i := range m.mmap.ram {
		m.mmap.ram[i] = 0
	}
	m.rom.bank = 0
	m.periph.Zero()
	m.cpu.AssertReset()
	return nil
}

// Tick drives the CPU forward by approximately cycles bus cycles
// and returns the new cumulative cycle count. Tick never splits an
// instruction across calls — the cumulative count may exceed the
// budget by up to one instruction (inherits mos6502.CPU.Run
// semantics).
func (m *Machine) Tick(cycles uint64) uint64 {
	return m.cpu.Run(cycles)
}

// AssertIRQ sets the CPU's IRQ line (level-sensitive). Phase 002
// stubs never call this on their own (FR-021).
func (m *Machine) AssertIRQ(level bool) { m.cpu.AssertIRQ(level) }

// AssertNMI raises an edge on the CPU's NMI line (edge-triggered).
func (m *Machine) AssertNMI() { m.cpu.AssertNMI() }

// DeassertNMI clears the host-side NMI line so the next AssertNMI
// is observed as a fresh edge.
func (m *Machine) DeassertNMI() { m.cpu.DeassertNMI() }

// SetRDY drives the CPU's RDY input. ready=false stalls read
// cycles. Default after construction is ready=true.
func (m *Machine) SetRDY(ready bool) { m.cpu.SetRDY(ready) }

// CPU returns the underlying mos6502.CPU for direct observation
// (attaching a Trace recorder, reading Registers, installing an
// IllegalOpcodeHook, calling Disassemble at PC). Production hosts
// should treat the returned pointer as read-only: mutating CPU
// state directly bypasses Machine invariants.
func (m *Machine) CPU() *mos6502.CPU { return m.cpu }

// Memory returns the underlying MemoryMap. It satisfies
// mos6502.Memory and is exported so debug helpers like
// mos6502.Disassemble can be called directly from a host.
// Production hosts should not mutate the returned map's RAM
// through this handle — go through Snapshot/Restore or normal
// CPU instructions instead.
func (m *Machine) Memory() *MemoryMap { return &m.mmap }
