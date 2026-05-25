# Quickstart: `mos6502` — Faithful NMOS 6502 CPU Emulator

**Feature**: 001-cpu-6502-core

**Audience**: A developer (the GoBeeb emulator itself, or a third party) who wants to instantiate a 6502, run a small program, and observe the result.

## Install

```sh
go get github.com/gobeeb/GoBeeb/mos6502@latest
```

No runtime dependencies. Go 1.22+ required.

## Minimal program

Run a tiny machine-code routine that loads `$42` into the accumulator and stores it at `$0200`:

```go
package main

import (
	"fmt"

	"github.com/gobeeb/GoBeeb/mos6502"
)

// FlatRAM is the simplest possible Memory: a 64 KB byte slice.
type FlatRAM [0x10000]byte

func (r *FlatRAM) Read(addr uint16) uint8         { return r[addr] }
func (r *FlatRAM) Write(addr uint16, value uint8) { r[addr] = value }

func main() {
	var ram FlatRAM

	// Program at $0600: LDA #$42 ; STA $0200 ; BRK
	copy(ram[0x0600:], []byte{0xA9, 0x42, 0x8D, 0x00, 0x02, 0x00})

	// Reset vector at $FFFC/$FFFD points at $0600.
	ram[0xFFFC] = 0x00
	ram[0xFFFD] = 0x06

	cpu := mos6502.New(&ram)
	cpu.AssertReset()

	// Run until BRK has been serviced (≈ 20 cycles).
	for i := 0; i < 5; i++ {
		cpu.Step()
	}

	r := cpu.Registers()
	fmt.Printf("A = $%02X, mem[$0200] = $%02X, cycles = %d\n",
		r.A, ram[0x0200], r.Cycles)
	// Output: A = $42, mem[$0200] = $42, cycles = 17
}
```

That's the entire integration surface for the no-frills case: implement `Memory`, call `New`, call `Step`.

## Wiring an interrupt source

The wider GoBeeb emulator will assert IRQs from the System VIA (50 Hz tick) and NMI from the 1770 disc controller. The CPU's view is:

```go
cpu.AssertIRQ(true)            // start of an IRQ pulse
// … host runs cycles, CPU acknowledges at next instruction boundary
cpu.AssertIRQ(false)           // peripheral has been serviced

cpu.AssertNMI()                // single edge — one service
// (the next AssertNMI without DeassertNMI between is NOT seen)
cpu.DeassertNMI()              // host clears its end of the NMI line
```

## Driving RDY for 1 MHz bus alignment

The BBC stretches the CPU when accessing `$FCxx`–`$FEFF` (Sheila). The host memory wrapper can implement this by toggling `RDY` per cycle:

```go
// Inside the host's per-cycle wrapper, before calling CPU.StepCycle():
if addrIsSheila(cpu.PendingAddress()) && !alignedTo1MHz(cpu.Cycles()) {
	cpu.SetRDY(false) // stall this read cycle
} else {
	cpu.SetRDY(true)
}
cpu.StepCycle()
```

NMOS `RDY` only stalls reads; writes proceed even when `RDY` is asserted. The CPU enforces this faithfully (FR-023).

## Observing illegal opcodes

Useful for spotting emulator runaway (a stray fetch of `$00`-padded ROM space, say) during development:

```go
cpu.SetIllegalOpcodeHook(func(pc uint16, op uint8) {
	log.Printf("illegal opcode $%02X at $%04X", op, pc)
})
```

The hook is purely observational; the CPU still treats the byte as a single-byte 2-cycle NOP (FR-019).

## Capturing a bus trace (tests / golden files)

For per-cycle verification against a reference trace:

```go
t := mos6502.NewTrace(4096) // 4 K-entry ring buffer
cpu.SetTrace(t)

cpu.Step()

for _, ev := range t.Snapshot() {
	kind := map[mos6502.BusEventKind]string{mos6502.BusRead: "R", mos6502.BusWrite: "W"}[ev.Kind]
	fmt.Printf("[%d] %s $%04X = $%02X\n", ev.Cycle, kind, ev.Addr, ev.Value)
}
```

For RMW instructions the trace will contain three consecutive cycles to the effective address: read, write of original value, write of modified value (FR-021).

## Running the Klaus Dormann test ROMs

Both `6502_functional_test` and `6502_decimal_test` are embedded in the package's `_test.go` files; they run as part of the standard `go test` suite. No additional download required.

```sh
go test ./mos6502/ -run TestFunctional
go test ./mos6502/ -run TestDecimal
```

A successful run prints the cycle count and the PC of the documented success trap.

## What to read next

- [`spec.md`](./spec.md) — what the CPU promises (FR-001..FR-023).
- [`data-model.md`](./data-model.md) — internal structure of the `CPU` type.
- [`contracts/`](./contracts/) — exported API surface in Go.
- [`research.md`](./research.md) — design decisions and their rationale.
