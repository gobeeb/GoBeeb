# Phase 1 Data Model: 6502 CPU Core

**Feature**: 001-cpu-6502-core

**Date**: 2026-05-25

This document specifies the *internal* data model of the `mos6502` package and the precise field-level shape of every type that crosses the package boundary. Field types are Go types; everything is little-endian (matching the 6502 itself).

## 1. CPU

The central type. One instance per emulated 6502.

```go
type CPU struct {
    // Architectural registers (FR-001).
    A  uint8   // accumulator
    X  uint8   // index X
    Y  uint8   // index Y
    SP uint8   // stack pointer (low byte; the stack is always $0100 | SP)
    PC uint16  // program counter
    P  uint8   // processor status — bit layout: NV-BDIZC (see §2)

    // Bus collaborator (FR-006).
    mem Memory

    // Pending signals (latched, processed at the next instruction boundary
    // unless the NMI-hijack window applies — see §6 Interrupts).
    irqLine     bool   // level-sensitive
    nmiPending  bool   // edge-triggered: set by AssertNMI, cleared on service
    nmiPrev     bool   // last-seen NMI line state, for edge detection
    resetPending bool

    // RDY (FR-023). Level-sensitive: the host re-evaluates it each cycle.
    rdy bool          // false = asserted (i.e. CPU stalls on reads)

    // Cumulative bus-cycle counter (FR-005, FR-018). Wraps at uint64 max
    // (≥ 5,000 years of emulation at 2 MHz — non-issue).
    cycles uint64

    // Optional observers. Both nil-checked once per cycle; nil = zero cost.
    trace          *Trace                 // bus-trace recorder (§7)
    onIllegalOp    IllegalOpcodeHook      // FR-019 hook; default nil

    // Internal scratch for sub-cycle execution. Not part of the public
    // architectural state; reset to zero between instructions. Held here
    // rather than as locals to keep the hot path allocation-free.
    addr   uint16   // current effective address
    fetched uint8   // last byte read by fetch()
}
```

### Field invariants

- `SP` is the 8-bit low byte; the live stack address is always `0x0100 | uint16(SP)`. Pushes decrement `SP` after writing; pulls increment `SP` before reading.
- `P` bit layout (LSB first): `C, Z, I, D, B, U, V, N`. Bit 5 (`U`, unused) is conceptually always 1 on a real 6502 but is not enforced live; it is asserted to 1 only when `P` is *pushed* to the stack.
- `cycles` is monotonically non-decreasing across `Step()`, `StepCycle()`, `Run()`. RESET re-initialises it to 0 (see Research §1).
- Exactly one of `resetPending`, `(nmiPending → service)`, `(irqLine ∧ !I-flag → service)` may be acted upon at the start of an instruction; ordering: RESET ≻ NMI ≻ IRQ.

## 2. Processor status (`P`) flag layout

| Bit | Name | Constant       | Meaning                                                                |
|-----|------|----------------|------------------------------------------------------------------------|
| 0   | C    | `FlagCarry`    | Carry / borrow.                                                        |
| 1   | Z    | `FlagZero`     | Last result was zero.                                                  |
| 2   | I    | `FlagInterrupt`| IRQ disabled when set.                                                 |
| 3   | D    | `FlagDecimal`  | BCD mode for `ADC`/`SBC` when set.                                     |
| 4   | B    | `FlagBreak`    | Not a real flag in `P`; appears only in pushed-status copies (see §6). |
| 5   | U    | `FlagUnused`   | Reads as 1 in pushed status; ignored in live `P`.                      |
| 6   | V    | `FlagOverflow` | Signed overflow on the last `ADC`/`SBC`/`BIT`.                         |
| 7   | N    | `FlagNegative` | Bit 7 of the last result.                                              |

Helper methods on `*CPU`: `setFlag(mask uint8, on bool)`, `flag(mask uint8) bool`, `setNZ(v uint8)`.

## 3. Memory interface

See `contracts/memory.go`. Field-level summary:

```go
type Memory interface {
    Read(addr uint16) uint8
    Write(addr uint16, value uint8)
}
```

Infallible (Research §2). No batch / range methods — the CPU only ever issues one access per bus cycle.

## 4. Instruction descriptor table

The 256-entry dispatch table indexed by opcode byte. Each entry is a function pointer; the function performs the full sub-cycle execution of that opcode.

```go
type opcodeFn func(c *CPU)

var opcodeTable = [256]opcodeFn{ … }
```

Auxiliary table (used by `Disasm` only, never on the hot path):

```go
type opcodeMeta struct {
    Mnemonic   string         // "LDA", "BRK", …
    Mode       AddressingMode // §5
    BaseCycles uint8          // documented base cost
    Length     uint8          // 1, 2, or 3 bytes
    Illegal    bool           // true for undocumented opcodes
}

var opcodeMetaTable = [256]opcodeMeta{ … }
```

The two tables are independent so that production binaries that strip the disassembler (`-trimpath -ldflags "-X mos6502.disableMeta=1"` — TBD) pay no metadata cost.

## 5. Addressing modes

```go
type AddressingMode uint8

const (
    ModeImplicit AddressingMode = iota   // operates on registers only
    ModeAccumulator                      // operand is A
    ModeImmediate                        // operand is the byte after opcode
    ModeZeroPage                         // $LL
    ModeZeroPageX                        // $LL,X (wraps in zero page)
    ModeZeroPageY                        // $LL,Y (wraps in zero page)
    ModeRelative                         // signed 8-bit offset from PC after operand
    ModeAbsolute                         // $LLHH
    ModeAbsoluteX                        // $LLHH,X (+1 cycle on page cross)
    ModeAbsoluteY                        // $LLHH,Y (+1 cycle on page cross)
    ModeIndirect                         // ($LLHH) — JMP only; with NMOS page-bug
    ModeIndexedIndirect                  // ($LL,X) — wraps in zero page
    ModeIndirectIndexed                  // ($LL),Y — +1 cycle on page cross
)
```

Each mode has a corresponding *effective-address helper* used internally by opcode handlers (`effZP`, `effZPX`, `effAbs`, `effAbsX`, etc.). Helpers are responsible for the dummy reads on page-cross (FR-018, Story-2 acceptance scenario 2), and for the zero-page wrap (FR-014).

## 6. Interrupts (RESET / IRQ / NMI / BRK)

Three latched signals on `CPU` plus the on-instruction `BRK` opcode all share the "interrupt entry sequence" code path. The sequence is implemented as a 7-cycle micro-routine, with the NMI-hijack check between cycles 4 and 5 (FR-022):

| Cycle | Action (for IRQ / BRK)                                                                   | Hijack check |
|-------|-------------------------------------------------------------------------------------------|--------------|
| 1     | Dummy read of PC (BRK) or dummy read of next opcode (IRQ).                                | n/a          |
| 2     | Dummy read of PC+1 (BRK skips a byte; IRQ does not).                                      | n/a          |
| 3     | Push `PCH` to stack.                                                                       | n/a          |
| 4     | Push `PCL` to stack.                                                                       | n/a          |
| 5     | Push `P` with `B` set (BRK) or `B` clear (IRQ); set `I` in live `P`. *Latch vector address.* | **here**: if NMI is asserted at this point, latch the NMI vector instead. |
| 6     | Read PCL from latched vector (`$FFFE` for IRQ/BRK, `$FFFA` for NMI, `$FFFC` for RESET).   | n/a          |
| 7     | Read PCH from latched vector + 1.                                                          | n/a          |

RESET shares the same vector-read tail (cycles 6 & 7) but the first five cycles are dummy reads (the silicon does not push the stack on RESET, it just decrements `SP` three times — hence the post-RESET `SP = $FD`, see Research §1).

NMI is edge-triggered. `AssertNMI()` sets `nmiPending = true`. The CPU clears `nmiPending` only after servicing it; a still-asserted NMI line (`nmiPrev == true`) does *not* re-trigger.

## 7. Bus-trace recorder

Optional observability hook for tests.

```go
type BusEventKind uint8

const (
    BusRead BusEventKind = iota
    BusWrite
)

type BusEvent struct {
    Cycle uint64
    Addr  uint16
    Value uint8
    Kind  BusEventKind
}

type Trace struct {
    // pre-allocated ring buffer; configurable size; default 4096 entries
    events []BusEvent
    head   int
    full   bool
}
```

`(*CPU).SetTrace(t *Trace)` attaches; `t.Snapshot()` returns the current contents in chronological order. `t == nil` (default) → no allocation, no branch beyond a single nil-check per cycle.

## 8. Illegal-opcode hook

```go
type IllegalOpcodeHook func(pc uint16, opcode uint8)
```

Registered via `(*CPU).SetIllegalOpcodeHook(h IllegalOpcodeHook)`. Invoked whenever the CPU executes an opcode whose dispatch slot is the shared `illegalNOP` handler (FR-019). `h == nil` → not invoked; single nil-check overhead per illegal opcode (which is itself off the hot path in normal BBC software).

## 9. State transitions

Top-level CPU lifecycle:

```text
[constructed]  ── AssertReset() ──▶  [reset-pending]
                                     │
                                     ▼ (next Step / StepCycle / Run)
                                  [running] ◀──────────────┐
                                     │                     │
                                     ├── AssertIRQ() ──▶  irqLine = true (level)
                                     ├── AssertNMI() ──▶  nmiPending = true (edge-latched)
                                     ├── SetRDY(false) ──▶  reads stall, writes proceed
                                     └── AssertReset() ──▶  back to [reset-pending]
```

There is no explicit "halt" state — illegal opcodes do *not* halt (FR-019 chose option B). RESET is the only way to return a CPU to a known initial state.

## 10. Concurrency model

A single `CPU` value is **not** safe for concurrent use (Assumption A8). The package documents this in `doc.go`. If the host wants to step the CPU from one goroutine and assert IRQ from another, it must serialise via its own mutex; the package does not add internal locking because the locking overhead is prohibitive on the hot path.

Two distinct `CPU` instances with two distinct `Memory` implementations *are* safe to run from two goroutines (no shared state inside the package).
