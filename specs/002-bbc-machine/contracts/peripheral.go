//go:build ignore

// Package bbc contract definitions.
//
// This file documents the Peripheral interface — the contract every
// SHEILA stub implements so the MemoryMap decoder can dispatch I/O
// accesses uniformly. Spec references: FR-008, FR-009, FR-010;
// Research §7.

package bbc

// Peripheral is implemented by every SHEILA stub (CRTC, ACIA, SerialULA,
// VideoULA, SystemVIA, UserVIA, FDC, ADC, Tube, Econet, ACCCON).
//
// The MemoryMap decoder calls Read or Write on a Peripheral after
// translating the bus address $FE00–$FEFF into the peripheral's local
// register offset. The offset is already masked to the peripheral's
// stride (e.g. 0..1 for CRTC, 0..15 for VIA), so implementations do
// not need to re-mask.
//
// Implementations MUST:
//
//   - Default to last-write-wins for any register that is not declared
//     write-only or read-only.
//   - Return the documented open-bus value ($FF) for write-only
//     registers on a Read call.
//   - Silently drop writes to read-only registers.
//   - Be deterministic: given the same sequence of Read and Write
//     calls, return the same sequence of bytes and reach the same
//     internal state.
//   - NOT be goroutine-safe. The single-goroutine contract from
//     Machine (FR-028) carries down to every Peripheral.
//
// Implementations MUST NOT:
//
//   - Allocate on Read or Write (zero-allocation hot path, FR-026).
//   - Source IRQ or NMI signals on their own in Phase 002 (FR-021).
//     Future phases that implement real peripheral behaviour will
//     drive those signals via the parent Machine's AssertIRQ /
//     AssertNMI pass-throughs.
type Peripheral interface {
    // Read returns the byte at the given register offset. Called by
    // MemoryMap.Read for any access in the peripheral's SHEILA window.
    Read(reg uint8) uint8

    // Write stores value at the given register offset. Called by
    // MemoryMap.Write for any access in the peripheral's SHEILA window.
    Write(reg uint8, value uint8)
}

// ──────────────────────────────────────────────────────────────────
// Phase 002 stub semantics summary
// ──────────────────────────────────────────────────────────────────
//
// CRTC (6845, $FE00–$FE07; stride 2):
//   - reg==0 (write):     selectedRegister = value & 0x1F, masked to 0..17
//   - reg==0 (read):      $FF (write-only)
//   - reg==1 (write):     regs[selectedRegister] = value
//   - reg==1 (read):      regs[selectedRegister]
//   - FR-010: this is the only piece of "real" CRTC behaviour in
//     Phase 002. Scanline timing, cursor blink, frame generation are
//     Phase 003.
//
// VIA (SystemVIA + UserVIA, stride 16):
//   - 16 registers, last-write-wins for all 16.
//   - Phase 002 does NOT implement timers, shift registers, port
//     direction registers, or interrupt flags; those land in a later
//     phase when VIA-driven peripherals come online.
//
// ACIA (6850, $FE08–$FE0F; stride 4):
//   - 4 registers, last-write-wins. Real ACIA register decoding is
//     Phase 005+ work.
//
// SerialULA ($FE10–$FE1F; stride 1):
//   - 1 register, last-write-wins.
//
// VideoULA ($FE20–$FE2F; stride 2):
//   - 2 registers (control + palette latch), last-write-wins. Real
//     palette decoding is Phase 003.
//
// FDC ($FE80–$FE9F; stride 4):
//   - 4 registers, reads return $FF, writes drop in Phase 002.
//     Full 1770 behaviour lands in Phase 007.
//
// ADC ($FEC0–$FEDF; stride 4):
//   - 4 registers, last-write-wins.
//
// Tube ($FEE0–$FEFF; stride 8):
//   - 8 registers, last-write-wins.
//
// Econet ($FEA0–$FEBF; stride 4):
//   - 4 registers, reads return $FF, writes drop in Phase 002.
//
// ACCCON ($FE34–$FE37):
//   - Reads always return $FF, writes drop. Model B has no ACCCON;
//     this stub exists for Master 128 compatibility in a future phase.
