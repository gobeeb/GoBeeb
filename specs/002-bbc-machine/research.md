# Phase 0 Research: BBC Machine Layer

**Feature**: 002-bbc-machine

**Date**: 2026-05-25

**Purpose**: Pin down every design decision the spec deferred to plan time, settle the one Deferred item from the Clarifications coverage summary (Snapshot value-type shape), and record best-practice research that drives the Phase 1 design.

---

## 1. Memory-map dispatch shape

### Decision

A single `switch` on the high byte of the address inside `MemoryMap.Read` / `MemoryMap.Write`, with fast paths for the two dominant regions (main RAM and OS ROM) listed first:

```go
func (m *MemoryMap) Read(addr uint16) uint8 {
    switch {
    case addr < 0x8000:          // RAM  — most common
        return m.ram[addr]
    case addr >= 0xC000 && addr != 0xFC00, addr >= 0xFF00:  // OS ROM body + vectors
        return m.osROM[addr-0xC000]
    case addr < 0xC000:          // sideways window
        return m.sidewaysRead(addr)
    default:                     // $FC00–$FEFF I/O
        return m.ioRead(addr)
    }
}
```

(The above is illustrative; the production form will short-circuit unreachable cases and special-case the I/O page boundaries cleanly.)

### Rationale

- Branch-predictable on real workloads: a typical BBC workload spends > 95 % of cycles in RAM and OS ROM. Putting those two first inside the switch hands the branch predictor a stable target.
- Zero allocation: no maps, no interface dispatch on the hot path. The peripherals are dispatched only for the ~3 % of cycles that hit `$FC00`–`$FEFF`.
- O(1) and stays cache-resident: the `osROM` and `ram` slice headers fit alongside other `MemoryMap` fields in a single cache line.

### Alternatives considered

1. **`[256]func(addr uint16) uint8` table indexed by `addr >> 8`** — rejected. Higher constant overhead per access (table load + indirect call), and the table itself is 2 KB of pointers, mostly pointing at the same RAM/ROM handlers. Branch predictor on a switch is faster in measured Go micro-benchmarks for this access shape.
2. **Per-region `[]byte` indexed by `addr` with a flat `[64]byte*` page table** — considered for symmetry with how some other emulators do it. Rejected: the BBC has only a handful of distinct page classes (RAM, OS ROM, sideways window, three I/O pages), so a switch is more honest and avoids re-creating the page table on every bank switch.
3. **Interface-typed `region` per address class** — rejected: introduces a per-cycle interface call (boxing, vtable lookup), violates the zero-allocation budget.

---

## 2. SHEILA decoder shape

### Decision

A two-level switch inside `MemoryMap.ioRead` / `ioWrite`: outer switch on the high nibble of the low byte (so `$FE00`–`$FE0F` → CRTC/ACIA, `$FE10`–`$FE1F` → serial ULA, etc.), inner switch (or direct dispatch) on the low nibble when needed (e.g. `$FE00` vs `$FE01` for the CRTC). Each terminal arm calls a `Peripheral.Read(reg uint8)` / `Write(reg uint8, value uint8)` method, where `reg` is the address modulo the peripheral's stride (e.g. 2 for CRTC, 16 for VIA, 32 for video ULA).

### Rationale

- The SHEILA layout is geographically clustered by high nibble; the outer switch maps directly to BBC service-manual diagrams.
- The inner switch lets each peripheral state its stride once (a `const` in its file) rather than baking it into the decoder.
- `Peripheral` is an interface, but it is only dispatched for the cold I/O path (≤ 5 % of cycles in a typical workload); the cost is negligible against the ≤ ~6.5 ns/cycle budget.

### Alternatives considered

1. **Single flat switch on the full low byte** — rejected: produces a 256-arm switch that the compiler emits as a jump table; tedious to read, no measurable performance win, harder to test peripheral routing independently.
2. **One method per peripheral on `MemoryMap` directly** — rejected: tight coupling between the decoder and every peripheral; cannot test peripherals in isolation; bloats `memory_map.go` to thousands of lines.

---

## 3. `Snapshot` value-type shape (Deferred item from Clarifications)

### Decision

`Snapshot` is a plain Go struct with exported fields, holding by-value copies of every piece of state. No interface, no `encoding.BinaryMarshaler`, no `gob.GobEncoder` plumbing — the snapshot is in-process only, byte-for-byte serialisation to disk is deferred to a later phase that needs it.

```go
type Snapshot struct {
    CPU            mos6502.Registers          // architectural state via Phase 001 Registers
    RAM            [0x8000]byte               // 32 KB main RAM, copied by value
    SidewaysBank   uint8                      // currently-selected bank index (0..3)
    SidewaysLoaded [4]bool                    // which bank slots are populated
    Peripherals    PeripheralSnapshot         // see below
}

type PeripheralSnapshot struct {
    CRTC        CRTCSnapshot
    ACIA        ACIASnapshot
    SerialULA   SerialULASnapshot
    VideoULA    VideoULASnapshot
    SystemVIA   VIASnapshot
    UserVIA     VIASnapshot
    FDC         FDCSnapshot
    ADC         ADCSnapshot
    Tube        TubeSnapshot
    Econet      EconetSnapshot
    // (RomSelect/ACCCON are scalars carried in the outer Snapshot itself.)
}

// Per-stub snapshots are tiny:
type VIASnapshot     struct{ Regs [16]byte }
type CRTCSnapshot    struct{ Regs [18]byte; Index uint8 }
type ACIASnapshot    struct{ Regs [4]byte }
// (and so on per peripheral)
```

### Rationale

- **In-process round-trip is what the constitution requires.** The "no hidden state" rule from the roadmap demands a symmetrical capture/restore pair; it does NOT demand a disk format. Locking down disk encoding now would be premature optimisation that ties the hands of Phase 005+ when audio + video buffers may also need snapshotting.
- **Exported fields beat opaque accessors** for a debugging type. Tests, debuggers, and future save-state tools want to read `snap.RAM[0x1000]` without going through a getter. Go's convention for "data carrier" types is exported fields.
- **By-value copies kill aliasing.** Restoring a snapshot is `m.ram = snap.RAM; m.cpu.SetRegisters(snap.CPU); …` — no shared slices, no surprise mutation.
- **Tiny: ~32.5 KB per snapshot** dominated by main RAM. A debug session that snapshots once per second for an hour fits in ~115 MB; throwaway. If a future phase needs much smaller snapshots (e.g. for replay), a separate `CompactSnapshot` can be added without breaking this API.

### Alternatives considered

1. **Opaque `[]byte` returned by `Snapshot() []byte`** — rejected: forces a serialisation format decision now, hides the structure from tests, and the consumer has to write a parallel deserialiser to introspect the snapshot.
2. **`encoding.BinaryMarshaler` / `gob.GobEncoder`** — rejected for the same reason: not needed until disk persistence is on the table; adds a code-gen-ish dependency for no current consumer.
3. **JSON-friendly struct (lowercase tags, base64 RAM)** — rejected: the snapshot is meant to be touched by Go code, not piped through a config file. JSON serialisation of a 32 KB byte array via base64 is wasteful and slow.

### Forward-compatibility

If save-state to disk becomes a requirement, the chosen shape is trivially compatible with `encoding/gob`, `encoding/binary`, or a hand-rolled format — all the fields are `[N]byte` arrays, `byte`, `bool`, and the Phase 001 `Registers` struct (already a value type). The disk-format choice can be deferred without API change.

---

## 4. Sideways ROM bank storage

### Decision

Each bank slot is a fixed-size `[16384]byte` array inside `MemoryMap`, plus a parallel `[4]bool` "loaded" flag. Loading a ROM does `copy(m.sideways[bank][:], image)` (per Clarification Q4) and sets `m.sidewaysLoaded[bank] = true`. Reading the sideways window when `!sidewaysLoaded[currentBank]` returns `$FF` and fires the unmapped-access hook (per FR-014, FR-029).

### Rationale

- Fixed `[16384]byte` arrays sit inline in the parent struct: no heap allocation per bank, predictable cache layout.
- The `loaded` flags let us distinguish "bank exists, all bytes happen to be `$FF`" from "bank empty, observe an unmapped read" — the test in `unmapped_test.go` relies on this distinction.
- Copy-on-load (Clarification Q4) means the caller can free or mutate the input slice immediately after the loader returns.

### Alternatives considered

1. **`[4][]byte` slice headers pointing at caller's buffer** — rejected by Clarification Q4 (aliasing hazard).
2. **`map[uint8][]byte` keyed by bank** — rejected: allocation on map insertion, slower lookup, harder to snapshot.

---

## 5. OS ROM storage

### Decision

A single fixed-size `[16384]byte` array inside `MemoryMap`. Loaded once via `LoadOSROM([]byte) error` (copy-on-load), referenced by `Read(addr)` when `addr >= 0xC000` (excluding the I/O window).

### Rationale

- Same fixed-array reasoning as sideways banks: inline, cache-friendly, no allocation.
- Single image (Model B has exactly one OS ROM slot, unlike Master 128 with paged OS).
- A `loaded` bool gates the first `Tick` after construction: if no OS ROM has been loaded, `Reset()` / `Tick()` returns `ErrNoOSROM` rather than letting the CPU run into a zero-filled vector (User Story 1 acceptance scenario 3).

### Alternatives considered

1. **Lazy load on first Tick from a callback** — rejected: surprise I/O at the wrong moment; complicates the single-goroutine model.
2. **Allow nil OS ROM and return `$FF`** — rejected: hides the "forgot to load OS ROM" bug behind a working-looking machine that silently fails.

---

## 6. `Tick` cycle-budget semantics

### Decision

`Tick(cycles uint64) uint64` is a thin wrapper around `mos6502.CPU.Run(cycles)` (inherited from Phase 001). It returns the cumulative cycle count after the call; the actual cycles consumed during this `Tick` is `after - before`. The "may overshoot by one instruction" disclaimer from FR-005 inherits directly from `mos6502.CPU.Run`'s documented behaviour.

```go
func (m *Machine) Tick(cycles uint64) uint64 {
    return m.cpu.Run(cycles)
}
```

### Rationale

- Reuses Phase 001's `Run` budget logic verbatim — no second clock, no risk of drift (FR-006).
- Zero allocation, single function call: the BBC layer adds essentially no overhead to `Tick`; all the overhead is in the per-read/per-write decoder, which is on the `mos6502.Memory.Read/Write` path the CPU calls into.

### Alternatives considered

1. **`Tick` returns `(consumed, totalCycles)` tuple** — rejected: returning two values forces the hot path to construct a small struct in registers; cleaner to return just `totalCycles` and let the caller subtract.
2. **`Tick` drives the CPU one instruction at a time and re-checks an internal "stop" flag** — rejected: adds per-instruction branch overhead for a feature (early stop) no current consumer needs.

---

## 7. Peripheral interface shape

### Decision

```go
type Peripheral interface {
    Read(reg uint8) uint8
    Write(reg uint8, value uint8)
}
```

Each stub implements this interface; the decoder calls `peripheral.Read(reg)` / `peripheral.Write(reg, value)` after translating the bus address into a register offset. Snapshot/Restore is NOT on this interface — it would force allocation of an `any`/`interface{}` snapshot value per peripheral; instead, the parent `Snapshot` struct names each peripheral's snapshot type concretely (see §3).

### Rationale

- The decoder doesn't need to know what kind of peripheral it's talking to, only its register offset.
- Two-method interfaces are cheap in Go: the dynamic dispatch is a single indirect call, well-predicted because each address range always dispatches to the same peripheral.
- Snapshot stays off the interface to keep the hot path interface-call-free for the read/write hot path.

### Alternatives considered

1. **Single combined method `Access(reg uint8, write bool, value uint8) uint8`** — rejected: complicates the read-only / write-only register cases (need a sentinel for "not a write"); harder to test.
2. **No interface; the decoder calls each stub's method by name directly** — rejected: blows up `MemoryMap` with N hand-coded dispatch arms; loses the ability to plug in a mock peripheral for the SHEILA routing test.

---

## 8. Unmapped-access hook integration

### Decision

`UnmappedAccessHook` is a function type stored on `Machine` (and reachable from `MemoryMap` via a pointer set during construction). The decoder's "unmapped" arms invoke it via a single nil-check + indirect call:

```go
if m.unmappedHook != nil {
    m.unmappedHook(addr, isWrite, value)
}
```

The hook is invoked for: FRED/JIM slots not claimed by a (future) peripheral, SHEILA offsets not in the FR-008 table, and sideways-window reads when the selected bank has no loaded image. It is NOT invoked for: writes to RAM (always succeed), reads/writes to the OS ROM body (writes drop, reads succeed), and writes to a loaded sideways ROM (writes drop, no "unmapped" condition — see edge cases).

### Rationale

- Single nil-check on the cold path → zero overhead on the hot path (RAM/ROM reads + writes never touch the hook).
- Matches the Phase 001 `IllegalOpcodeHook` pattern: function type, single nil-check, invoked synchronously.
- A test installs a counter-collecting hook, runs the machine, asserts the counter stayed zero — directly satisfies User Story 1 acceptance scenario 1.

### Alternatives considered

1. **Channel-based event stream** — rejected: forces an allocation per event, requires a draining goroutine (violates single-goroutine model).
2. **Append to an in-`Machine` slice instead of calling a hook** — rejected: the consumer has to remember to clear it; allocations on append; less flexible than a closure.

---

## 9. CRTC index-then-data semantics (FR-010 implementation note)

### Decision

The CRTC stub keeps two state cells: a `selectedRegister` `uint8` (mod 18 — masked to 0..17 to model real 6845 behaviour) and a `[18]byte` register file. Write to `$FE00` (low bit clear) stores the value in `selectedRegister` (masked). Write to `$FE01` (low bit set) stores the value at `registers[selectedRegister]`. Read of `$FE01` returns `registers[selectedRegister]`. Read of `$FE00` returns the documented open-bus value (`$FF`).

### Rationale

- This is the one piece of "real" CRTC behaviour FR-010 mandates; everything else (scanline timing, etc.) is Phase 003.
- Masking to 0..17 means MOS code that probes the CRTC by writing index `>= 18` doesn't corrupt out-of-bounds memory.
- The index is part of the CRTC snapshot (see §3) so save/restore is faithful.

### Alternatives considered

1. **Store as a full `[256]byte` register file indexed by raw `selectedRegister`** — rejected: wastes 238 bytes per CRTC instance and doesn't reflect real hardware.
2. **Don't mask `selectedRegister`** — rejected: out-of-bounds access on a writeable indexed register is a memory-safety bug.

---

## 10. OS ROM availability for tests (no-redistribution policy)

### Decision

The unit test suite uses a hand-crafted 16 KB stub OS ROM (`testdata/stub_os_16k.bin`) that exercises just enough of the boot path to validate the memory map: a reset vector pointing at a deterministic 6502 sequence in the OS ROM region that touches RAM, writes a SHEILA register, pages a sideways bank, and runs a small loop. The OS-ROM smoke test (`boot_os_test.go`) is gated on the `BBC_OS_ROM` environment variable: if unset, the test is `t.Skip`ped; if set, the path is read at test time, the bytes are loaded into the machine, and the SC-001 assertion (≥ 1 000 000 cycles without firing the unmapped-access hook) runs.

### Rationale

- The project does not bundle or distribute any copyrighted BBC ROM (spec Assumptions).
- A hand-crafted stub gives deterministic, in-tree validation of the BBC memory map without leaning on a real ROM.
- The smoke test against the real OS 1.20 stays available to anyone who supplies the path, including CI configured by the project owner.

### Alternatives considered

1. **Bundle OS 1.20** — rejected: copyright.
2. **Generate an OS-ROM-shaped 6502 binary at test time** — over-engineered for the validation goal; the stub plus the gated smoke test is sufficient.
3. **Synthesize a "boot path" from a small assembler in tests** — appealing but would re-implement an assembler. The stub binary, committed once, is cheaper.

---

## 11. Bench shape & SC-006 verification

### Decision

Two benchmarks in `bench_test.go`:

- `BenchmarkTickNoop`: machine constructed with a stub OS ROM whose body is `EA EA EA …` (NOPs) with a reset vector pointing at it. Inner loop calls `m.Tick(1024)`. Asserts `0 B/op 0 allocs/op` and `ns/op / 1024` ≤ ~6.5 ns/cycle.
- `BenchmarkTickMixedWorkload`: machine with a synthetic ROM that loops `LDA $0000 ; STA $FE40 ; LDA $8000 ; INC $1000`. Exercises RAM, OS ROM, sideways ROM (bank 0 pre-loaded), and a SHEILA write per iteration. Same allocation gate; ns/cycle budget is the meaningful number for the wider emulator.

### Rationale

- `BenchmarkTickNoop` isolates the per-cycle overhead of the BBC layer (CPU + decoder + RAM read).
- `BenchmarkTickMixedWorkload` measures something closer to a real BBC workload, including the cold path through SHEILA.
- Both run on the standard `make bench` target — no special setup.

### Alternatives considered

1. **Re-use Phase 001's `BenchmarkRunNoop` directly** — rejected: that benchmark doesn't go through the BBC decoder; we need a Phase 002 number.
2. **Replay a captured BBC bus trace** — overkill for SC-006; can be added later if a regression analysis needs it.
