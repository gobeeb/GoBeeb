//go:build ignore

// Package bbc contract definitions.
//
// This file documents the MemoryMap type — the bridge between
// mos6502.Memory and the BBC's physical address layout. Spec
// references: FR-001..FR-004, FR-008..FR-014, FR-018; Research §1, §2,
// §4, §5.

package bbc

import "github.com/gobeeb/GoBeeb/mos6502"

// Compile-time assertion that MemoryMap satisfies mos6502.Memory.
var _ mos6502.Memory = (*MemoryMap)(nil)

// MemoryMap implements mos6502.Memory with the BBC Model B address
// layout:
//
//   $0000–$7FFF   32 KB main RAM (read/write)
//   $8000–$BFFF   16 KB sideways ROM bank window (read-only; bank
//                 selected by the latch at $FE30–$FE33)
//   $C000–$FBFF   OS ROM body (read-only)
//   $FC00–$FCFF   FRED I/O page (unmapped reads return $FF; writes
//                 drop)
//   $FD00–$FDFF   JIM I/O page (same default behaviour)
//   $FE00–$FEFF   SHEILA I/O page (decoded to stub peripherals; see
//                 SHEILA decoder section below)
//   $FF00–$FFFF   OS ROM vector region (read-only)
//
// MemoryMap is owned by Machine and is NOT intended for standalone
// construction. The exported type is documented here so that test
// harnesses, third-party debuggers, and the Phase 003 video package
// can name it in interface assertions; the public API for normal use
// goes through Machine.
//
// MemoryMap is NOT safe for concurrent use (inherits FR-028 from
// Machine).
type MemoryMap struct {
    // (Internal fields — see data-model.md §2 for the full layout.)
}

// Read returns the byte at addr per the BBC Model B memory map.
//
// Reads of unmapped addresses (FRED/JIM offsets not claimed by a
// peripheral; SHEILA offsets not in the routing table; sideways-window
// reads when no image is loaded in the selected bank) return $FF AND
// invoke the Machine's UnmappedAccessHook (if set) before returning.
//
// Read MUST NOT crash, panic, or surface an error for any address.
// (FR-002.)
func (m *MemoryMap) Read(addr uint16) uint8 { return 0 }

// Write stores value at addr per the BBC Model B memory map.
//
// Writes to read-only regions (OS ROM body $C000–$FBFF, OS ROM vector
// region $FF00–$FFFF, currently-paged sideways ROM at $8000–$BFFF) are
// silently dropped. (FR-003.)
//
// Writes to the SHEILA window are dispatched to the relevant stub
// peripheral via the decoder. Writes to FRED/JIM offsets not claimed
// by a peripheral are dropped AND invoke the Machine's
// UnmappedAccessHook (if set).
//
// The $FE30–$FE33 ROM-select latch is the single I/O write that has
// observable side effects in Phase 002: the low two bits of `value`
// become the new sideways bank index, and subsequent reads of
// $8000–$BFFF return bytes from the new bank. (FR-013.)
//
// Write MUST NOT crash, panic, or surface an error for any address.
func (m *MemoryMap) Write(addr uint16, value uint8) {}

// ──────────────────────────────────────────────────────────────────
// SHEILA decoder routing table (FR-008)
//
// The decoder dispatches I/O accesses in $FE00–$FEFF to one of:
//
//   $FE00–$FE07  CRTC        (Peripheral; A0 selects index vs data)
//   $FE08–$FE0F  ACIA
//   $FE10–$FE1F  SerialULA
//   $FE20–$FE2F  VideoULA
//   $FE30–$FE33  ROM-select latch (handled inline; updates RomBanks.bank)
//   $FE34–$FE37  ACCCON (Model B: always $FF on read; write drops)
//   $FE40–$FE5F  SystemVIA   (mirrors every 16 bytes)
//   $FE60–$FE7F  UserVIA     (mirrors every 16 bytes)
//   $FE80–$FE9F  FDC         (Phase 002: $FF on read; Phase 007 fills)
//   $FEA0–$FEBF  Econet      (Phase 002: $FF on read)
//   $FEC0–$FEDF  ADC
//   $FEE0–$FEFF  Tube
//
// Offsets within $FE00–$FEFF not listed above are unmapped. Reads
// return $FF, writes drop, both fire UnmappedAccessHook if set.
// ──────────────────────────────────────────────────────────────────
