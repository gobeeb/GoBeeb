//go:build ignore

// Package bbc contract definitions.
//
// This file documents the UnmappedAccessHook function type registered
// via Machine.SetUnmappedAccessHook. Spec references: FR-029;
// Clarifications session 2026-05-25 Q2.

package bbc

// UnmappedAccessHook is invoked on every read or write the BBC memory
// map classifies as unmapped:
//
//   - FRED ($FC00–$FCFF) offsets not claimed by a peripheral.
//   - JIM ($FD00–$FDFF) offsets not claimed by a peripheral.
//   - SHEILA ($FE00–$FEFF) offsets not in the FR-008 routing table.
//   - Sideways-window reads ($8000–$BFFF) when the currently-selected
//     bank has no loaded image.
//
// The hook is purely observational. The CPU still sees the open-bus
// value ($FF) on the read; the write is silently dropped. The hook is
// called synchronously on the same goroutine that called Tick, BEFORE
// the underlying read returns; the hook MUST return quickly because
// the CPU is paused inside mos6502.CPU.Step until it returns.
//
// Parameters:
//
//   - addr: the original 16-bit bus address (not the post-mask
//     register offset).
//   - write: true for write accesses, false for reads.
//   - value: for writes, the byte the CPU attempted to write; for
//     reads, the open-bus byte the machine is about to return ($FF).
//
// Typical hook implementations:
//
//   - Acceptance test: increment a counter; assert the counter
//     stayed zero across an OS-ROM boot run (User Story 1 acceptance
//     scenario 1).
//   - Debugger overlay: log the address + direction + value for
//     display in a Phase 004 ImGui window.
//   - Forensic snapshot: capture the CPU PC at the time of the
//     access (via Machine.CPU().Registers().PC) to attribute the
//     access back to the calling instruction.
//
// A nil hook is the default after New and is a no-op (single
// nil-check on the cold path).
type UnmappedAccessHook func(addr uint16, write bool, value uint8)
