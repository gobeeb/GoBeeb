// Package mos6502 contract definitions.
//
// This file documents the illegal-opcode notification hook contract.
// Spec references: FR-019; Clarifications session 2026-05-25 Q4.

package mos6502

// IllegalOpcodeHook is invoked whenever the CPU executes a byte that
// is not one of the 151 documented NMOS 6502 opcodes (FR-019).
//
// The CPU treats the byte as a single-byte, 2-cycle NOP regardless of
// whether a hook is registered. The hook is purely an observability
// signal: its return value (there is none) does not influence CPU
// behaviour, and the CPU does not pause to wait for the hook to
// complete (the hook is called synchronously and is expected to
// return quickly).
//
// Parameters:
//
//   - pc: the value of the program counter AT WHICH THE OPCODE WAS
//     FETCHED, i.e. the address of the illegal byte itself. The
//     CPU's live PC has already advanced by 1 by the time the hook
//     is called.
//   - opcode: the byte that was fetched.
//
// Typical hook implementations:
//
//   - Test harness: append (pc, opcode) to a slice for later
//     assertion ("did our code path ever hit an illegal opcode?").
//   - Debugger: log a warning, optionally set a breakpoint.
//   - Wider GoBeeb emulator: count occurrences for the front-end
//     status bar; this should be very rare on a healthy MOS.
//
// The hook is invoked at most once per illegal opcode executed. It
// is NOT invoked for legal opcodes, RESET, IRQ, NMI, or BRK.
type IllegalOpcodeHook func(pc uint16, opcode uint8)
