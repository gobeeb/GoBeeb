//go:build ignore

// Package bbc contract definitions.
//
// This file documents the Snapshot value type returned by
// Machine.Snapshot and consumed by Machine.Restore. Spec references:
// FR-022, FR-023, FR-024; Clarifications session 2026-05-25
// (Deferred → Resolved here); Research §3.

package bbc

import "github.com/gobeeb/GoBeeb/mos6502"

// Snapshot captures the complete observable state of a Machine.
//
// Snapshot is a plain value type with exported fields: tests,
// debuggers, and future save-state tooling can read fields directly
// without going through accessors. The byte-level serialisation
// format (gob, JSON, custom binary) is NOT fixed by Phase 002 —
// in-process round-trip via Restore is the only contract.
//
// Snapshot does NOT carry OS ROM or sideways ROM image bytes (FR-024).
// The consumer of Restore is responsible for re-loading the same
// images on the destination Machine first. Restore returns
// ErrRestoreMismatch if the loaded-bank set differs.
//
// Snapshot is value-copyable; copying a Snapshot is ~32.5 KB of byte
// data and one mos6502.Registers struct. No internal pointers to
// shared mutable state.
type Snapshot struct {
    // CPU is the architectural register state captured at the moment
    // Snapshot was called. Equivalent to mos6502.CPU.Registers().
    CPU mos6502.Registers

    // RAM is a by-value copy of the 32 KB main RAM ($0000–$7FFF).
    RAM [0x8000]byte

    // SidewaysBank is the currently-selected sideways bank index
    // (0..3) at snapshot time.
    SidewaysBank uint8

    // SidewaysLoaded[i] is true iff bank slot i had a ROM image
    // loaded at snapshot time. Restore uses this to assert the
    // destination Machine has the same set of loaded banks.
    SidewaysLoaded [4]bool

    // Peripherals captures every SHEILA stub's register file.
    Peripherals PeripheralSnapshot
}

// PeripheralSnapshot is the by-value capture of every SHEILA stub.
type PeripheralSnapshot struct {
    CRTC      CRTCSnapshot
    ACIA      ACIASnapshot
    SerialULA SerialULASnapshot
    VideoULA  VideoULASnapshot
    SystemVIA VIASnapshot
    UserVIA   VIASnapshot
    FDC       FDCSnapshot
    ADC       ADCSnapshot
    Tube      TubeSnapshot
    Econet    EconetSnapshot
}

// CRTCSnapshot is the captured state of the 6845 CRTC stub. Selected
// is the currently-pointed register index (0..17); Regs is the
// per-register file.
type CRTCSnapshot struct {
    Regs     [18]byte
    Selected uint8
}

// VIASnapshot is the captured register file of a 6522 VIA stub
// (System VIA or User VIA).
type VIASnapshot struct {
    Regs [16]byte
}

// ACIASnapshot is the captured register file of the 6850 ACIA stub.
type ACIASnapshot struct {
    Regs [4]byte
}

// SerialULASnapshot is the captured register state of the serial
// ULA stub.
type SerialULASnapshot struct {
    Regs [1]byte
}

// VideoULASnapshot is the captured register state of the video ULA
// stub (control + palette latch).
type VideoULASnapshot struct {
    Regs [2]byte
}

// FDCSnapshot is the captured register state of the 1770 FDC stub.
type FDCSnapshot struct {
    Regs [4]byte
}

// ADCSnapshot is the captured register state of the µPD7002 ADC stub.
type ADCSnapshot struct {
    Regs [4]byte
}

// TubeSnapshot is the captured register state of the Tube stub.
type TubeSnapshot struct {
    Regs [8]byte
}

// EconetSnapshot is the captured register state of the ADLC / Econet
// stub.
type EconetSnapshot struct {
    Regs [4]byte
}
