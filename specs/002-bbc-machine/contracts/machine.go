//go:build ignore

// Package bbc contract definitions.
//
// This file documents the public surface of the Machine type as it will
// be exposed in `bbc/machine.go`. It is the authoritative reference for
// downstream consumers (Phase 003 video, Phase 004 SDL host, third-party
// debuggers). Spec references: FR-005, FR-006, FR-019, FR-020, FR-022,
// FR-023, FR-028; Clarifications session 2026-05-25 Q1/Q3; Research §1,
// §6, §10.

package bbc

import "github.com/gobeeb/GoBeeb/mos6502"

// Machine is one emulated BBC Model B. It owns one mos6502.CPU, a BBC
// memory map, OS ROM + sideways ROM banks, and the SHEILA stub
// peripherals.
//
// A Machine is NOT safe for concurrent use by multiple goroutines
// (FR-028). The caller MUST drive every public method — Tick, Reset,
// ColdReset, the IRQ/NMI/RDY pass-throughs, the ROM loaders, Snapshot,
// and Restore — from the same goroutine. The implementation does not
// take internal locks on the Tick hot path.
type Machine struct {
    // (Internal fields — see data-model.md §1 for the full struct
    // layout. The public surface is method-based.)
}

// New constructs a fresh Machine with no ROMs loaded. The CPU is
// reset-pending at construction; the first Tick (after LoadOSROM) will
// perform the 7-cycle RESET sequence and fetch from $FFFC/$FFFD.
//
// Calling Tick or Reset before LoadOSROM returns ErrNoOSROM.
func New() *Machine { return nil /* contract only */ }

// ──────────────────────────────────────────────────────────────────
// ROM loading (FR-012, FR-016, FR-017) — Clarification Q4: copy-on-load
// ──────────────────────────────────────────────────────────────────

// LoadOSROM installs the 16 KB OS ROM image at $C000–$FFFF (excluding
// the I/O window at $FC00–$FEFF). The image MUST be exactly 16384 bytes
// or LoadOSROM returns ErrInvalidROMSize.
//
// The image is COPIED into internally-owned storage; the caller may
// mutate, reuse, or release the slice immediately after the call
// returns. (Clarification Q4.)
//
// LoadOSROM may be called more than once (e.g. to swap ROM versions).
// The current Machine state (CPU registers, RAM, peripheral registers)
// is preserved; only the OS ROM bytes change. The caller is responsible
// for issuing Reset() after a hot-swap if guest software is running.
func (m *Machine) LoadOSROM(image []byte) error { return nil }

// LoadSidewaysROM installs a 16 KB ROM image into the given sideways
// bank slot (0..3 on Model B). The image MUST be exactly 16384 bytes
// or LoadSidewaysROM returns ErrInvalidROMSize. `bank` MUST be 0..3
// or LoadSidewaysROM returns ErrBankOutOfRange.
//
// Like LoadOSROM, the image is COPIED into internally-owned storage.
// (Clarification Q4.)
//
// Loading a bank does NOT change the currently-selected bank latch
// (which is at $FE30). MOS code or the host MUST write to $FE30 to
// make the new bank visible at $8000–$BFFF.
func (m *Machine) LoadSidewaysROM(bank int, image []byte) error { return nil }

// ──────────────────────────────────────────────────────────────────
// Reset surface (FR-019, Clarification Q3)
// ──────────────────────────────────────────────────────────────────

// Reset performs a BBC BREAK-key soft reset. Preserves main RAM and
// every peripheral register file; clears the sideways bank latch to 0;
// asserts mos6502.CPU.AssertReset() so the next Tick re-fetches the
// RESET vector at $FFFC/$FFFD.
//
// Returns ErrNoOSROM if no OS ROM has been loaded.
func (m *Machine) Reset() error { return nil }

// ColdReset performs a power-on reset. Zeros main RAM, zeros every
// peripheral register file, clears the sideways bank latch to 0,
// asserts CPU reset. Equivalent to constructing a fresh Machine with
// the same loaded ROM images.
//
// Returns ErrNoOSROM if no OS ROM has been loaded.
func (m *Machine) ColdReset() error { return nil }

// ──────────────────────────────────────────────────────────────────
// CPU control pass-throughs (FR-020)
// ──────────────────────────────────────────────────────────────────
//
// These are thin pass-throughs to the underlying mos6502.CPU. They
// exist on Machine so future phases (Phase 003 video pulling RDY low;
// Phase 005 sound + Phase 007 storage asserting IRQ/NMI from their
// peripherals) can drive these lines without bypassing the machine.
// Phase 002 itself never invokes them automatically (FR-021).

// AssertIRQ sets the IRQ line on the CPU. Level-sensitive; deassert
// with AssertIRQ(false).
func (m *Machine) AssertIRQ(level bool) {}

// AssertNMI raises an edge on the CPU's NMI line. Edge-triggered.
func (m *Machine) AssertNMI() {}

// DeassertNMI clears the host-side NMI line.
func (m *Machine) DeassertNMI() {}

// SetRDY drives the CPU's RDY input. ready=false stalls READ cycles.
// Default after construction is ready=true.
func (m *Machine) SetRDY(ready bool) {}

// ──────────────────────────────────────────────────────────────────
// Execution surface (FR-005, FR-006)
// ──────────────────────────────────────────────────────────────────

// Tick drives the CPU forward by approximately `cycles` bus cycles and
// returns the new cumulative cycle count. Tick never splits an
// instruction across calls — it may overshoot the requested budget by
// up to the cost of the last instruction (the same semantics as
// mos6502.CPU.Run, inherited verbatim per Research §6).
//
// Tick must NOT be called from a goroutine other than the one that
// constructed the Machine (FR-028).
//
// Returns 0 and an error condition (signalled via panic on a Machine
// constructed without ROM) only at the first cycle after construction
// if no OS ROM has been loaded. Once Tick has run successfully at
// least once, subsequent Tick calls never error.
func (m *Machine) Tick(cycles uint64) uint64 { return 0 }

// ──────────────────────────────────────────────────────────────────
// Observability (FR-029)
// ──────────────────────────────────────────────────────────────────

// SetUnmappedAccessHook installs (or removes, with nil) a callback
// invoked on every read or write to an address the BBC memory map
// classifies as unmapped: FRED/JIM offsets not claimed by a
// peripheral, SHEILA offsets not in the FR-008 routing table, and
// sideways-window reads ($8000–$BFFF) when the currently-selected
// bank has no loaded image.
//
// The hook is purely observational. The CPU still sees the
// open-bus value ($FF) on the read; the write is silently dropped.
// A nil hook is a no-op (the default after New).
//
// The hook is called synchronously on the calling goroutine; it
// MUST return quickly. (See data-model.md §7.)
func (m *Machine) SetUnmappedAccessHook(hook UnmappedAccessHook) {}

// ──────────────────────────────────────────────────────────────────
// Snapshot / Restore (FR-022, FR-023, FR-024) — Research §3
// ──────────────────────────────────────────────────────────────────

// Snapshot captures the complete observable state of the Machine: CPU
// registers, main RAM, sideways bank index + loaded-flags, and every
// peripheral's register file. The returned value is suitable for
// in-process round-trip via Restore.
//
// Snapshot does NOT include the OS ROM or sideways ROM image bytes
// (those are large and externally-supplied; the caller is responsible
// for re-loading the same images before Restore).
func (m *Machine) Snapshot() Snapshot { return Snapshot{} }

// Restore re-populates Machine state from snap. The Machine MUST
// already have an OS ROM loaded; for any sideways bank where
// snap.SidewaysLoaded[i] is true, the Machine MUST also have a
// sideways ROM loaded in that bank.
//
// Returns ErrRestoreMismatch if the loaded-bank set differs between
// snap and the Machine. Returns ErrNoOSROM if no OS ROM has been
// loaded.
//
// On success, subsequent Tick calls produce bit-identical execution
// to a Machine that had run the original sequence directly.
func (m *Machine) Restore(snap Snapshot) error { return nil }

// ──────────────────────────────────────────────────────────────────
// Access to the inner CPU (debug / disassembly)
// ──────────────────────────────────────────────────────────────────

// CPU returns the underlying mos6502.CPU pointer for direct
// observation (e.g. mos6502.Disassemble at PC, attaching a Trace
// recorder, reading Registers, installing an IllegalOpcodeHook).
//
// Production hosts should treat the returned pointer as read-only:
// mutating CPU state directly bypasses the Machine's invariants
// (e.g. cycle counting, snapshot consistency). Use SetRegisters /
// AssertReset via the Machine surface where possible.
func (m *Machine) CPU() *mos6502.CPU { return nil }
