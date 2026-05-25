# Phase 1 Data Model: BBC Machine Layer

**Feature**: 002-bbc-machine

**Date**: 2026-05-25

This document specifies the *internal* data model of the `bbc` package and the field-level shape of every type that crosses the package boundary. All field types are Go types; arrays are little-endian (matching the 6502 itself).

---

## 1. `Machine`

The central façade. One instance per emulated BBC Model B.

```go
type Machine struct {
    cpu       *mos6502.CPU      // owned; constructed in New, bound to &mmap
    mmap      MemoryMap         // owned by value; the CPU calls into it
    rom       RomBanks          // owned by value; OS ROM + 4 sideways slots
    periph    Peripherals       // owned by value; SHEILA stubs

    // Hook surface (FR-029). nil = silent; zero overhead on hot path.
    unmappedHook UnmappedAccessHook
}
```

### Field invariants

- `cpu` is non-nil after `New`; the Phase 001 `mos6502.CPU` is reset-pending at construction (so the first `Tick` triggers the BBC RESET vector fetch from `$FFFC/$FFFD`).
- `mmap` is constructed in-place; it holds a back-pointer to `&periph` and `&rom` (or, equivalently, the parent `*Machine`) so it can dispatch I/O and sideways reads without ownership ambiguity.
- `Machine` is NOT goroutine-safe (FR-028). All methods are called from one goroutine.

---

## 2. `MemoryMap`

Implements `mos6502.Memory`. Owns no state of its own beyond pointers to the parent's components; lives by-value inside `Machine`.

```go
type MemoryMap struct {
    rom    *RomBanks            // points to Machine.rom
    periph *Peripherals         // points to Machine.periph
    parent *Machine             // for unmappedHook dispatch

    ram [0x8000]byte            // 32 KB main RAM ($0000–$7FFF)
}
```

### Field invariants

- `ram` is byte-addressable for the full `$0000`–`$7FFF` range. Reads and writes go straight to the slice with no bounds check (compiler elides; index masked by `& 0x7FFF` against the slice length to allow the compiler's BCE to confirm safety).
- `rom`, `periph`, `parent` are set once in `Machine.init()` and never mutated.
- The struct is placed inline inside `Machine` so cache locality is high: the `cpu` (in its own struct) reads from `m.ram` on every fetch; the two structs share L1 in practice.

---

## 3. `RomBanks`

Owns the OS ROM image and the four sideways ROM bank slots.

```go
type RomBanks struct {
    os        [0x4000]byte      // 16 KB OS ROM ($C000–$FBFF + $FF00–$FFFF)
    osLoaded  bool              // gated; Reset returns ErrNoOSROM if false

    sideways       [4][0x4000]byte
    sidewaysLoaded [4]bool      // true iff a real image was loaded into the slot

    bank uint8                  // currently-selected bank index (0..3)
}
```

### Field invariants

- `bank` is always in the range `0..3`. Writes to `$FE30`–`$FE33` apply `value & 0x03` before storing.
- `osLoaded` is set true by `LoadOSROM`, never cleared.
- `sidewaysLoaded[i]` is set true by `LoadSidewaysROM(i, …)`, never cleared (consumers wanting to "unload" a bank can write zeros via a future API; not in scope).
- All ROM bytes are owned by the `Machine` after load (Clarification Q4, FR-012, FR-016). The caller's slice is copied via `copy(dst[:], src)`.

---

## 4. `Peripherals`

Holds every SHEILA stub by value. Owned by `Machine`.

```go
type Peripherals struct {
    CRTC      CRTC              // $FE00–$FE07
    ACIA      ACIA              // $FE08–$FE0F
    SerialULA SerialULA         // $FE10–$FE1F
    VideoULA  VideoULA          // $FE20–$FE2F
    // $FE30–$FE33 is the ROM-select latch — lives on RomBanks; no stub.
    ACCCON    ACCCON            // $FE34–$FE37 (returns $FF on Model B)
    SystemVIA VIA               // $FE40–$FE5F (mirrors every 16)
    UserVIA   VIA               // $FE60–$FE7F (mirrors every 16)
    FDC       FDC               // $FE80–$FE9F
    Econet    Econet            // $FEA0–$FEBF
    ADC       ADC               // $FEC0–$FEDF
    Tube      Tube              // $FEE0–$FEFF
}
```

Each field is a small struct (≤ 32 bytes) implementing the `Peripheral` interface (Read/Write by register offset).

---

## 5. Peripheral stubs

All stubs share a common shape: a `[N]byte` register file plus zero or one small auxiliary fields. They implement `Peripheral`:

```go
type Peripheral interface {
    Read(reg uint8) uint8
    Write(reg uint8, value uint8)
}
```

### 5.1 `CRTC` (6845, indexed)

```go
type CRTC struct {
    selected uint8        // 0..17 (masked); register index
    regs     [18]byte     // R0..R17
}
```

- `reg == 0` → write stores to `selected = value & 0x1F` then re-masks via `selected %= 18`; reads return `$FF`.
- `reg == 1` → write stores to `regs[selected]`; reads return `regs[selected]`.
- Other reg values: mirror of 0/1 within the 8-byte CRTC window (real hardware decodes only A0).

### 5.2 `VIA` (System + User, identical shape)

```go
type VIA struct {
    regs [16]byte
}
```

- 16 registers, last-write-wins for all 16 (Phase 002 stub).
- The decoder masks the SHEILA offset by 16 before calling `Write`/`Read`, which is what produces the "mirrors every 16 bytes" behaviour from a single stub.

### 5.3 Other stubs (ACIA, SerialULA, VideoULA, FDC, ADC, Tube, Econet, ACCCON)

```go
type ACIA      struct{ regs [4]byte }      // mirrors over the 8-byte ACIA window
type SerialULA struct{ regs [1]byte }      // single register; mirrors over $FE10–$FE1F
type VideoULA  struct{ regs [2]byte }      // control + palette; mirrors over $FE20–$FE2F
type FDC       struct{ regs [4]byte }      // 1770 has 4 registers
type ADC       struct{ regs [4]byte }      // µPD7002 has 4 registers
type Tube      struct{ regs [8]byte }      // Tube has 8 registers
type Econet    struct{ regs [4]byte }      // ADLC has 4 registers
type ACCCON    struct{}                    // unmapped on Model B; always $FF
```

All non-ACCCON stubs follow last-write-wins. ACCCON's `Read` returns `$FF` and `Write` is a no-op.

---

## 6. `Snapshot`

Exported value type used by `Machine.Snapshot()` / `Machine.Restore(Snapshot) error`.

```go
type Snapshot struct {
    CPU            mos6502.Registers           // Phase 001 type, by value
    RAM            [0x8000]byte                // 32 KB main RAM
    SidewaysBank   uint8                       // currently-selected bank (0..3)
    SidewaysLoaded [4]bool                     // which slots were populated
    Peripherals    PeripheralSnapshot          // per-stub state
}

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

type CRTCSnapshot      struct { Regs [18]byte; Selected uint8 }
type VIASnapshot       struct { Regs [16]byte }
type ACIASnapshot      struct { Regs [4]byte }
type SerialULASnapshot struct { Regs [1]byte }
type VideoULASnapshot  struct { Regs [2]byte }
type FDCSnapshot       struct { Regs [4]byte }
type ADCSnapshot       struct { Regs [4]byte }
type TubeSnapshot      struct { Regs [8]byte }
type EconetSnapshot    struct { Regs [4]byte }
```

### Invariants

- `Snapshot` does NOT carry OS ROM or sideways ROM image bytes (FR-024). Caller is responsible for re-loading the same ROM images before calling `Restore`.
- `Snapshot.SidewaysLoaded` is captured for diagnostic round-tripping (`Restore` returns an error if the machine being restored into has different banks loaded than the snapshot recorded).
- All fields are exported so tests and debuggers can read them directly.

---

## 7. `UnmappedAccessHook`

Function type registered on `Machine` via `SetUnmappedAccessHook` (FR-029).

```go
type UnmappedAccessHook func(addr uint16, write bool, value uint8)
```

### Invocation rules

- Called from the cold path of `MemoryMap.Read` / `MemoryMap.Write` when no peripheral or memory region claims the address.
- `addr` is the original 16-bit bus address (not the post-mask register offset).
- `write` is `true` for write accesses, `false` for reads.
- For reads, `value` is the open-bus byte the machine will return to the CPU (always `$FF` in Phase 002).
- For writes, `value` is the byte the CPU attempted to write.
- The hook is called synchronously, on the calling goroutine, BEFORE the read returns `$FF` (so the hook observes both the access and the about-to-be-returned value).
- The hook MUST return quickly; the CPU is paused inside `mos6502.CPU.Step` until the hook returns.

---

## 8. Errors

```go
var (
    ErrNoOSROM         = errors.New("bbc: no OS ROM loaded")
    ErrInvalidROMSize  = errors.New("bbc: ROM image must be exactly 16384 bytes")
    ErrBankOutOfRange  = errors.New("bbc: sideways bank index must be 0..3")
    ErrRestoreMismatch = errors.New("bbc: restored snapshot has different sideways banks loaded than the current machine")
)
```

- `ErrNoOSROM`: returned by `Reset()` and the first `Tick()` after construction if no OS ROM was loaded.
- `ErrInvalidROMSize`: returned by `LoadOSROM` and `LoadSidewaysROM` for any image not exactly 16384 bytes.
- `ErrBankOutOfRange`: returned by `LoadSidewaysROM(bank, …)` for `bank < 0` or `bank > 3`.
- `ErrRestoreMismatch`: returned by `Restore` if `snap.SidewaysLoaded != m.rom.sidewaysLoaded`.

---

## 9. Lifecycle / state transitions

```text
                ┌─────────────┐
                │  New(mmap)  │
                └──────┬──────┘
                       │  osLoaded=false, sidewaysLoaded=[false×4]
                       ▼
                ┌─────────────────────┐
                │  LoadOSROM(image)   │  ────► osLoaded=true
                └──────────┬──────────┘
                           │
              ┌────────────┼────────────┐
              ▼            ▼            ▼
   LoadSidewaysROM      Tick(N)        Reset() | ColdReset()
   (any slot, any        ▲              │
    time)                │              │
                         └──────────────┘
                          CPU runs, peripherals see writes,
                          unmappedHook fires on bad accesses

   Snapshot()  ────►  Snapshot{...}  ────►  Restore(snap)
                                             (consumer must
                                              re-load same ROMs
                                              before calling)
```

Reset semantics:

- `Reset()` (BREAK-key): preserves `RAM`, peripheral register files, and sideways bank latch; asserts `mos6502.CPU.AssertReset()`.
- `ColdReset()` (power-on): zeros `RAM`, zeros every peripheral register file, sets bank latch to 0; asserts CPU reset.
- Both check `osLoaded`; if false, return `ErrNoOSROM` and do not touch CPU state.
