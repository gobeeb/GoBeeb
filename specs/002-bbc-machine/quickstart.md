# Quickstart: `bbc` — BBC Model B Machine Layer

**Feature**: 002-bbc-machine

**Audience**: A developer (the GoBeeb host shell, future Phase 003/004 work, or a third-party debugger) who wants to stand up a BBC machine, load an OS ROM, run cycles, and observe state.

## Install

```sh
go get github.com/gobeeb/GoBeeb/bbc@latest
```

No runtime dependencies beyond `github.com/gobeeb/GoBeeb/mos6502` (Phase 001). Go 1.22+ required.

## Minimal program — boot an OS ROM and run one frame's worth of cycles

```go
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/gobeeb/GoBeeb/bbc"
)

func main() {
    osROM, err := os.ReadFile(os.Getenv("BBC_OS_ROM"))
    if err != nil {
        log.Fatalf("read OS ROM: %v", err)
    }

    m := bbc.New()
    if err := m.LoadOSROM(osROM); err != nil {
        log.Fatalf("load OS ROM: %v", err)
    }

    // (Optionally:) load BASIC into sideways bank 0.
    // basic, _ := os.ReadFile("basic2.rom")
    // _ = m.LoadSidewaysROM(0, basic)

    // Drive the machine forward by one "frame" worth of 2 MHz cycles.
    if err := m.Reset(); err != nil {
        log.Fatalf("reset: %v", err)
    }
    cycles := m.Tick(40_000) // ~50 Hz frame @ 2 MHz

    regs := m.CPU().Registers()
    fmt.Printf("after %d cycles: PC=$%04X A=$%02X X=$%02X Y=$%02X SP=$%02X P=$%02X\n",
        cycles, regs.PC, regs.A, regs.X, regs.Y, regs.SP, regs.P)
}
```

## Observe unmapped bus accesses

```go
m := bbc.New()
_ = m.LoadOSROM(osROM)

var unmapped []string
m.SetUnmappedAccessHook(func(addr uint16, write bool, value uint8) {
    kind := "read"
    if write {
        kind = "write"
    }
    unmapped = append(unmapped, fmt.Sprintf("$%04X %s $%02X", addr, kind, value))
})

_ = m.Reset()
m.Tick(1_000_000)

if len(unmapped) > 0 {
    fmt.Println("unmapped accesses during boot:")
    for _, u := range unmapped {
        fmt.Println("  ", u)
    }
}
```

A healthy OS-ROM boot path should produce zero entries in `unmapped` (SC-001).

## Snapshot and restore

```go
// Run some cycles.
_ = m.Reset()
m.Tick(100_000)

snap := m.Snapshot()

// Build a parallel Machine, reload the same ROMs, restore.
m2 := bbc.New()
_ = m2.LoadOSROM(osROM)
// (Re-load any sideways ROMs the snapshot recorded as loaded.)
for bank, loaded := range snap.SidewaysLoaded {
    if loaded {
        _ = m2.LoadSidewaysROM(bank, sidewaysImages[bank])
    }
}
if err := m2.Restore(snap); err != nil {
    log.Fatalf("restore: %v", err)
}

// m and m2 now execute identically.
_ = m.Tick(50_000)
_ = m2.Tick(50_000)
// m.CPU().Registers() == m2.CPU().Registers()
```

## Reset vs ColdReset

```go
// BREAK-key reset — preserves RAM, peripheral registers.
_ = m.Reset()

// Power-on reset — zeros RAM, zeros peripheral registers,
// clears sideways bank latch to 0.
_ = m.ColdReset()
```

Both leave the ROM images intact. Neither changes which sideways banks are loaded.

## Attach a bus-trace recorder (Phase 001 surface)

The Machine's underlying `mos6502.CPU` is reachable via `Machine.CPU()`. Use the Phase 001 trace recorder to capture every bus cycle:

```go
trace := mos6502.NewTrace(8) // pre-allocated ring buffer
m.CPU().SetTrace(trace)
_ = m.Reset()
m.Tick(8) // first 8 cycles after RESET

for _, ev := range trace.Snapshot() {
    fmt.Printf("cycle %d: $%04X %v $%02X\n", ev.Cycle, ev.Addr, ev.Kind, ev.Value)
}
```

## Disassemble at the current PC

```go
// MemoryMap implements mos6502.Memory. Construct a Machine and
// pass its underlying map (exposed via a small accessor or by
// installing a peripheral mock for richer tests).
disasm, n := mos6502.Disassemble(m.Memory(), m.CPU().Registers().PC)
fmt.Printf("next instr (%d bytes): %s\n", n, disasm)
```

Note `Disassemble` takes a `mos6502.Memory`. `Machine.Memory()` returns the underlying `*MemoryMap`, which already satisfies the interface.

## Concurrency

`Machine` is single-goroutine (FR-028, Clarification Q1). Drive every public method — `Tick`, `Reset`, `ColdReset`, the control pass-throughs, ROM loaders, `Snapshot`, `Restore`, `SetUnmappedAccessHook` — from one goroutine. Cross-goroutine communication (e.g. a render thread reading frames) should go through a separate channel or ring buffer, not by sharing the `Machine`.

## Testing your own peripheral against the BBC layer

The `Peripheral` interface (`Read(reg uint8) uint8`, `Write(reg uint8, value uint8)`) is the seam Phase 003+ phases plug into. In Phase 002 the SHEILA decoder owns the peripheral set; in Phase 003 the real `VideoULA` will replace the stub via a planned constructor option. Tests in this phase can install a mock peripheral by constructing a `Machine` and inspecting the routing tests in `sheila_test.go` for the exact expected (addr → reg) mapping.

## Performance

`Tick` is the hot path. On Linux amd64 (developer workstation), the target is ≤ ~6.5 ns per emulated CPU cycle and `0 B/op 0 allocs/op` (SC-006). To measure locally:

```sh
make bench
# or
go test -bench=BenchmarkTick -benchmem ./bbc/...
```

Any benchmark regression > 5 % blocks merge per the constitution.

## What's NOT in this phase

- No window, no audio, no input — that lands in Phase 004 (SDL host).
- No real CRTC scanline timing, no video ULA pixel pipeline — that lands in Phase 003.
- No VIA timers, no FDC, no ACIA, no ADC, no Tube — those land in their respective later phases.
- No sound — Phase 005.
- No save-state file format — Snapshot/Restore is in-process only.

If you need any of these now, see the [roadmap](../../docs/roadmap.md).
