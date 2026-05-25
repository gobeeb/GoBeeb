// Package bbc wires the validated github.com/gobeeb/GoBeeb/mos6502
// CPU core into a BBC Model B memory map. The exported surface is
// small: one type (Machine) that owns the CPU, the memory map, the
// OS ROM, four sideways ROM banks, and the SHEILA stub peripherals.
//
// # Memory map
//
//	$0000–$7FFF   32 KB main RAM (read/write)
//	$8000–$BFFF   16 KB sideways ROM bank window (read-only; bank
//	              selected by the latch at $FE30–$FE33)
//	$C000–$FBFF   OS ROM body (read-only)
//	$FC00–$FCFF   FRED I/O page (unmapped reads return $FF)
//	$FD00–$FDFF   JIM I/O page (unmapped reads return $FF)
//	$FE00–$FEFF   SHEILA I/O page (decoded to stub peripherals per
//	              FR-008)
//	$FF00–$FFFF   OS ROM vector region (read-only)
//
// Reads of unmapped addresses return $FF (open-bus); writes to
// read-only regions silently drop. Both fire the optional
// UnmappedAccessHook when the host installs one. Mapped accesses
// (RAM, OS ROM, loaded sideways bank, mapped SHEILA peripheral)
// never invoke the hook.
//
// # Tick semantics
//
// Machine.Tick(cycles) is a thin pass-through to mos6502.CPU.Run.
// The cumulative cycle count may exceed the requested budget by
// up to one instruction; Tick never splits an instruction across
// calls. The Tick hot path performs zero allocations and takes no
// internal locks.
//
// # Single-goroutine contract
//
// Machine is NOT safe for concurrent use. Drive Tick, Reset,
// ColdReset, the IRQ/NMI/RDY pass-throughs, the ROM loaders,
// Snapshot, Restore, and SetUnmappedAccessHook from one
// goroutine. Cross-goroutine handoff (e.g. a render thread
// reading frames) must go through a separate channel or ring
// buffer, not by sharing the Machine.
//
// # Snapshot / Restore
//
// Machine.Snapshot() captures CPU registers, all 32 KB of main
// RAM, the sideways bank index + loaded-flags, and every
// peripheral's register file as a plain Go value. Restore
// re-populates the state. ROM image bytes are NOT included in
// the snapshot (FR-024); the consumer is responsible for
// re-loading the same OS ROM and sideways ROM images on the
// destination Machine before calling Restore. Restore returns
// ErrRestoreMismatch if the loaded-bank set differs.
//
// # Stub peripherals
//
// CRTC, ACIA, Serial ULA, Video ULA, System VIA, User VIA, FDC,
// ADC, Tube, Econet, and ACCCON implement register-file storage
// only. Real behaviour (CRTC scanline timing, VIA timers /
// shift registers / interrupt flags, ACIA / FDC / ADC / Tube /
// Econet semantics) is owned by later phases. The exception is
// the CRTC's index-then-data addressing required by FR-010,
// which Phase 002 does implement so MOS code that probes the
// CRTC during boot behaves correctly.
//
// See specs/002-bbc-machine/ for the feature specification,
// plan, research notes, data model, and quickstart.
package bbc
