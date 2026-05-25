# Feature Specification: BBC Machine Layer

**Feature Branch**: `002-bbc-machine`

**Created**: 2026-05-25

**Status**: Draft

**Input**: User description: "Implement phase 002 from @docs/roadmap.md"

**Source**: [Roadmap §Phase 002 — BBC machine layer](../../docs/roadmap.md), [ADR-0001](../../docs/adr/0001-language-go-vs-zig.md), [Phase 001 mos6502 package](../../mos6502/).

## Clarifications

### Session 2026-05-25

- Q: Concurrency model for `Machine.Tick` and the control surface → A: Single-goroutine. `Machine` is not goroutine-safe; caller MUST drive `Tick`, control-surface methods (`Reset`, `AssertIRQ`, `AssertNMI`, `DeassertNMI`, `SetRDY`), ROM loaders, and `Snapshot`/`Restore` from one goroutine. No internal locking. Matches the Phase 001 `mos6502.CPU` contract.
- Q: Observability mechanism for unmapped bus accesses (needed for User Story 1 acceptance scenario 1) → A: Add `Machine.SetUnmappedAccessHook(func(addr uint16, write bool, value uint8))`. Fires on any read or write the BBC map classifies as unmapped (FRED/JIM unclaimed offsets, SHEILA unassigned offsets, sideways-window reads when no image is loaded in the selected bank). Nil hook = silent; hot path is a single nil-check. Matches the Phase 001 `IllegalOpcodeHook` pattern.
- Q: Cold reset vs soft reset API surface → A: Add `Machine.ColdReset()` distinct from `Machine.Reset()`. `ColdReset()` zeros main RAM, clears the sideways bank latch to 0, and asserts CPU reset (power-on semantics). `Reset()` remains the BREAK-key soft reset (preserves RAM, clears bank latch, asserts CPU reset). Avoids forcing a Phase 004 host to rebuild the `Machine` and re-load every ROM just to model a power-cycle.
- Q: Sideways / OS ROM image ownership on load → A: Copy on load. `LoadOSROM` and `LoadSidewaysROM` `copy()` the caller's slice into an internally-owned buffer. Caller may mutate, reuse, or free the slice afterwards with no effect on the machine. One-time 16 KB copy per ROM is invisible against the cycle budget; removes the aliasing-bug class for the lifetime of the project.

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Boot the BBC Operating System ROM (Priority: P1)

A consumer of the BBC machine layer (initially future Phase 003 video work, future Phase 004 SDL host, and developers writing integration tests) constructs a `Machine`, hands it a BBC OS ROM image (e.g. OS 1.20, 16 KB), asserts a hard RESET, and drives the machine forward in cycle-budgeted slices. The 6502 fetches the RESET vector from the OS ROM region, begins executing the MOS startup sequence, and the machine runs into the early MOS init code without crashing on any undefined or unmapped memory access.

**Why this priority**: This is the irreducible MVP of the BBC machine layer. Without it, no later subsystem (video, sound, input, storage) can be brought up against real MOS code paths. Every later phase consumes a `Machine` whose CPU has been correctly wired to a BBC-shaped memory map; this story is what proves that wiring is correct end-to-end against a real ROM.

**Independent Test**: Can be fully tested without any UI: load OS 1.20 (or any user-supplied 16 KB BBC OS ROM), call `Machine.Reset()`, then `Machine.Tick(N)` for a representative cycle budget. The test asserts (a) the first instruction executed was fetched from the OS ROM region, (b) the CPU's `PC` is inside the OS ROM region after N cycles, (c) no illegal-opcode hook fired, and (d) no read or write was issued to an address the BBC memory map does not define.

**Acceptance Scenarios**:

1. **Given** a fresh `Machine` with OS 1.20 loaded, `Reset()` asserted, and an `UnmappedAccessHook` installed, **When** the host drives `Tick(cycles)` for at least the documented number of cycles MOS spends in its boot-banner setup, **Then** the CPU's program counter ends inside the OS ROM region, no illegal-opcode hook fires, and the unmapped-access hook never fires.
2. **Given** the same machine, **When** the host inspects the bus trace of the first eight instruction fetches after RESET, **Then** the RESET vector at `$FFFC/$FFFD` was read from the OS ROM image and execution proceeded at the documented MOS entry point.
3. **Given** a machine constructed without an OS ROM, **When** the host calls `Reset()` and `Tick(1)`, **Then** the constructor or call surfaces a clear "no OS ROM loaded" error condition before any CPU cycles are spent.

---

### User Story 2 — Route Peripheral I/O Through SHEILA (Priority: P2)

A consumer needs to observe and (later) intercept the BBC's memory-mapped I/O region (`$FC00`–`$FEFF`, "FRED/JIM/SHEILA"). The machine layer decodes every read and write in the SHEILA window (`$FE00`–`$FEFF`) to the correct stub peripheral by address range — CRTC `$FE00`–`$FE07`, ACIA `$FE08`–`$FE0F`, serial ULA `$FE10`–`$FE1F`, video ULA `$FE20`–`$FE2F`, ROM-select latch `$FE30`, paged ROM ID `$FE34`, System VIA `$FE40`–`$FE4F`, User VIA `$FE60`–`$FE6F`, FDC `$FE80`–`$FE9F`, ADC `$FEC0`–`$FECF`, Tube `$FEE0`–`$FEFF`, plus the wider FRED/JIM windows (`$FC00`–`$FCFF`, `$FD00`–`$FDFF`). Each peripheral exposes a register file with observable read/write behaviour; no peripheral implements real behaviour yet (that is later-phase work).

**Why this priority**: Phase 003 (video) and Phase 004 (host/debugger) both depend on the decoder being correct before they wire in real CRTC, ULA, or VIA behaviour. Getting the decoder + register-file stubs right in Phase 002 means later phases plug in behaviour without rewriting the bus.

**Independent Test**: Without booting any ROM, the test plays a scripted sequence of CPU reads/writes (via a test harness that pokes memory and steps the CPU on synthetic programs) at canonical SHEILA addresses, then asserts (a) the right stub peripheral received the access, (b) the access landed in the right register offset within that peripheral, and (c) read-back returns the last value written (default register-file semantics).

**Acceptance Scenarios**:

1. **Given** a freshly-constructed machine, **When** the CPU writes `$5A` to `$FE40` (System VIA ORB), **Then** the System VIA stub records the write at register offset `0` with value `$5A`, and a subsequent read of `$FE40` returns `$5A`.
2. **Given** the same machine, **When** the CPU writes to `$FE21` (video ULA palette register), **Then** the video ULA stub records the write at register offset `1` and a subsequent read of `$FE21` returns the stored value (or the documented open-bus value if the stub declares the register write-only).
3. **Given** the same machine, **When** the CPU reads or writes any address in `$FC00`–`$FCFF` (FRED) or `$FD00`–`$FDFF` (JIM) that no peripheral has claimed, **Then** the machine returns the documented unmapped-I/O value (`$FF`) on read and silently drops writes, without crashing or surfacing a bus error.

---

### User Story 3 — Page Sideways ROM Banks via `$FE30` (Priority: P3)

A consumer loads one or more 16 KB sideways ROM images (BASIC, DFS, view, etc.) into specific bank slots, then expects MOS — or test code — to write a bank index to the `$FE30` latch and have subsequent reads in `$8000`–`$BFFF` return bytes from the selected bank. Writing a different bank index swaps the window; the previously-selected bank is now invisible at `$8000`–`$BFFF`.

**Why this priority**: BASIC, the language ROM, lives in a sideways bank on a stock Model B and is fetched by MOS shortly after boot. Without paging, MOS will crash or hang during the language-ROM bring-up phase. This story is the gating requirement for "MOS runs past the early init into a usable state".

**Independent Test**: Load two distinct 16 KB images into bank 0 and bank 1 (the images differ by a single byte at offset `$0000`). Write `0` to `$FE30`, read `$8000`; assert the byte matches bank 0's first byte. Write `1` to `$FE30`, read `$8000`; assert the byte now matches bank 1's first byte. Switch back to `0` and re-read; assert the original byte returns.

**Acceptance Scenarios**:

1. **Given** a machine with bank 0 loaded with byte `$AA` at offset `$0000`, **When** the CPU writes `$00` to `$FE30` and reads `$8000`, **Then** the read returns `$AA`.
2. **Given** the same machine with bank 1 additionally loaded with `$55` at offset `$0000`, **When** the CPU writes `$01` to `$FE30` and reads `$8000`, **Then** the read returns `$55`.
3. **Given** the same machine, **When** the CPU writes `$01` to `$FE30`, executes any instruction, then writes `$00` to `$FE30` and reads `$8000`, **Then** the read returns the bank 0 value, confirming the paging is observable on every subsequent read.
4. **Given** a machine where no image is loaded into bank `K`, **When** the CPU writes `K` to `$FE30` and reads any address in `$8000`–`$BFFF`, **Then** the read returns the documented unmapped-ROM value (`$FF`).
5. **Given** the same machine, **When** the CPU writes any value to any address in `$8000`–`$BFFF`, **Then** the write is silently dropped (sideways ROM is read-only) and the next read at the same address still returns the ROM byte.

---

### User Story 4 — Snapshot and Restore Machine State (Priority: P4)

A consumer (test harness, future debugger UI, future save-state feature) needs to capture the complete observable state of the machine — CPU registers, all 32 KB of main RAM, current sideways bank selection, every SHEILA stub register file — into a serialisable value, then restore it later or in a different `Machine` instance and have execution resume bit-identically.

**Why this priority**: Constitution Principle "No hidden state" requires every subsystem to expose `Snapshot`/`Restore`. Capturing this in Phase 002 means later phases (video, sound) can extend the same pattern instead of retrofitting it. Save-state is a deferred user-facing feature, but the seam needs to exist now.

**Independent Test**: Construct machine A, load OS ROM, RESET, run `Tick(N)`. Call `Snapshot()`, store. Run `Tick(M)` more; record CPU registers as `R_A`. Construct machine B, load same OS ROM, `Restore(snapshot)`, run `Tick(M)`; record CPU registers as `R_B`. Assert `R_A == R_B` and the byte-for-byte contents of main RAM match.

**Acceptance Scenarios**:

1. **Given** two distinct `Machine` instances loaded with the same OS ROM, **When** machine A is run for `N` cycles, snapshotted, machine B is restored from that snapshot, and both are run for an additional `M` cycles, **Then** the CPU registers and all of main RAM are bit-identical between A and B.
2. **Given** a machine with multiple sideways banks loaded and bank 3 currently selected, **When** the machine is snapshotted and a fresh `Machine` is restored from that snapshot (with the same sideways ROM images re-loaded), **Then** reading `$8000` on the restored machine returns bank 3's byte without needing another `$FE30` write.
3. **Given** a machine where the System VIA stub has accumulated written register values, **When** snapshot-then-restore is performed, **Then** the System VIA stub on the restored machine returns the same values for every register read.

---

### Edge Cases

- What happens when the OS ROM image supplied to the loader is not exactly 16 KB? → Loader MUST reject the image with a clear error; partial-length ROMs are not handled.
- What happens when a sideways ROM image is not exactly 16 KB? → Same: reject at load time.
- What happens when `$FE30` is written with a bank index higher than the number of supported banks? → The high bits of the bank index are masked to the supported width (mirrors the real Model B's 2-bit, 4-bank latch); no crash.
- What happens when the CPU reads `$FF00`–`$FFFF` (top of OS ROM, contains interrupt vectors)? → Reads return the OS ROM bytes at the corresponding offset (`OS[$3F00]`–`OS[$3FFF]`); writes are silently dropped.
- What happens when the CPU writes anywhere in `$C000`–`$FBFF` (OS ROM body)? → Writes are silently dropped; OS ROM is read-only.
- What happens when the CPU asserts IRQ from a stub peripheral? → Phase 002 does not source interrupts from any stub. IRQ handling on the CPU side already works (Phase 001); the machine layer simply does not pull `IRQ` low yet.
- What happens during a power-on cold start vs a soft `Reset()` call? → `ColdReset()` zeros main RAM and every SHEILA stub register file; `Reset()` preserves both. Both clear the sideways bank latch to 0 and assert `mos6502.CPU.AssertReset()`. Matches real Model B power-cycle vs BREAK-key semantics.
- What happens when an empty (no banks loaded) sideways window is read at boot before MOS has written `$FE30`? → Reads return `$FF`; the bank index defaults to `0` at cold start.

## Requirements *(mandatory)*

### Functional Requirements

#### Memory map

- **FR-001**: System MUST implement a BBC Model B memory map with the following regions:
  - `$0000`–`$7FFF`: 32 KB main RAM (read/write, byte-addressable).
  - `$8000`–`$BFFF`: 16 KB sideways ROM bank window (read-only, contents determined by current bank latch).
  - `$C000`–`$FBFF` and `$FF00`–`$FFFF`: OS ROM (read-only, single 16 KB image at fixed location).
  - `$FC00`–`$FCFF`: FRED I/O page (unmapped peripherals return `$FF`; writes drop).
  - `$FD00`–`$FDFF`: JIM I/O page (same default behaviour).
  - `$FE00`–`$FEFF`: SHEILA I/O page (decoded to stub peripherals — see FR-008).
- **FR-002**: Reads of unmapped addresses MUST return `$FF` (open-bus default) and MUST NOT crash, panic, or surface a Go error to the caller.
- **FR-003**: Writes to read-only regions (OS ROM body, currently-paged sideways ROM, the `$FF00`–`$FFFF` vector region) MUST be silently dropped without surfacing an error.
- **FR-004**: Every memory access (read or write) MUST be observable to the host via the existing `mos6502.Memory` interface, with no hidden side accesses.

#### Cycle accounting

- **FR-005**: The machine MUST expose a `Tick(cycles)` method that drives the CPU forward by approximately the requested number of cycles, returning the actual number consumed. The actual count MAY exceed the requested budget by at most one instruction (matching `mos6502.CPU.Run` semantics from Phase 001).
- **FR-006**: Cycle accounting MUST be driven entirely by `mos6502.CPU.Step()`'s return value; the machine MUST NOT introduce a second authoritative clock.
- **FR-007**: The machine MUST NOT add per-cycle allocations on its `Tick` hot path (zero-allocation policy carries forward from Phase 001).

#### SHEILA decoder

- **FR-008**: The SHEILA decoder MUST route reads and writes in `$FE00`–`$FEFF` to the correct stub peripheral by the following address-range table:
  - `$FE00`–`$FE07`: 6845 CRTC stub (8 addresses; A0 selects register-index vs register-data — see FR-010).
  - `$FE08`–`$FE0F`: 6850 ACIA stub.
  - `$FE10`–`$FE1F`: Serial ULA stub.
  - `$FE20`–`$FE2F`: Video ULA stub.
  - `$FE30`–`$FE33`: ROM-select latch (writes select sideways bank; reads MAY return open-bus `$FF`).
  - `$FE34`–`$FE37`: ACCCON / paged-ROM-ID stub (Model B treats as unmapped; Master only — Phase 002 returns `$FF`).
  - `$FE40`–`$FE5F`: System VIA stub (mirrors `$FE40`–`$FE4F` every 16 bytes).
  - `$FE60`–`$FE7F`: User VIA stub.
  - `$FE80`–`$FE9F`: 1770 FDC stub (Phase 002 returns `$FF`; full behaviour is Phase 007).
  - `$FEA0`–`$FEBF`: Econet stub (returns `$FF`).
  - `$FEC0`–`$FEDF`: ADC stub.
  - `$FEE0`–`$FEFF`: Tube stub.
- **FR-009**: Each stub peripheral MUST expose a register file whose default read returns the last value written to that register offset. Stubs MAY override individual register-file slots to declare write-only or read-only semantics, in which case reads return the documented open-bus value (`$FF`) for write-only registers and writes are dropped for read-only registers.
- **FR-010**: The CRTC stub MUST track the currently-selected register index (writes to `$FE00`) and route subsequent writes to `$FE01` to the indexed register slot; reads of `$FE01` MUST return the indexed register's stored value. This is the only piece of "real" CRTC behaviour Phase 002 implements; everything else (scanline timing, cursor blink, frame generation) is Phase 003.

#### Sideways ROM banking

- **FR-011**: The machine MUST support **four** sideways ROM bank slots, indexed 0–3, matching the stock Model B latch width.
- **FR-012**: A consumer MUST be able to load a 16 KB ROM image into any bank slot via a constructor or loader method (e.g. `LoadSidewaysROM(bank int, image []byte) error`). The loader MUST copy the supplied slice into internally-owned storage; the caller MUST be free to mutate, reuse, or release the slice immediately after the call returns.
- **FR-013**: The currently-selected bank MUST be determined by the low bits of the most-recent write to any address in `$FE30`–`$FE33`. The high bits of the written value are masked to the bank-index width (currently 2 bits, masking to `0`–`3`).
- **FR-014**: Reads in `$8000`–`$BFFF` MUST return bytes from the currently-selected bank. If no image is loaded in that bank slot, reads MUST return `$FF`.
- **FR-015**: The bank index MUST be observable via `Snapshot` and restorable via `Restore`.

#### OS ROM loader

- **FR-016**: The machine MUST provide a constructor or loader method that accepts a 16 KB byte slice as the OS ROM image (e.g. `bbc.New(os []byte)` or `Machine.LoadOSROM([]byte) error`). As with sideways ROM, the loader MUST copy the supplied slice into internally-owned storage.
- **FR-017**: The loader MUST reject ROM images that are not exactly 16384 bytes with a clear error; the machine MUST NOT silently pad, truncate, or repeat the image.
- **FR-018**: Reads in `$C000`–`$FFFF` MUST return bytes from the loaded OS ROM image at the corresponding offset (`address - 0xC000`).

#### Reset, interrupts, RDY

- **FR-019**: The machine MUST expose two reset methods:
  - `Reset()`: soft reset. Asserts CPU reset via `mos6502.CPU.AssertReset()`, clears the sideways bank latch to 0, and PRESERVES main RAM contents and all SHEILA stub register-file values (BBC BREAK-key semantics).
  - `ColdReset()`: power-on reset. Asserts CPU reset, clears the sideways bank latch to 0, ZEROS all 32 KB of main RAM, and zeros every SHEILA stub register file. Equivalent to constructing a fresh `Machine` with the same loaded ROM images.
- **FR-020**: The machine MUST expose pass-through control methods (or equivalent surface) for `AssertIRQ`, `AssertNMI`, `DeassertNMI`, and `SetRDY` so future phases can drive these signals from peripherals without bypassing the machine.
- **FR-021**: Phase 002 stubs MUST NOT source IRQ or NMI signals on their own; the machine layer wires the control surface but does not pull any line low automatically.

#### Snapshot / Restore

- **FR-022**: The machine MUST expose a `Snapshot() Snapshot` method that captures: CPU registers, full 32 KB of main RAM, current sideways bank index, and every stub peripheral's register file.
- **FR-023**: The machine MUST expose a `Restore(Snapshot) error` method that re-populates the captured state. Restored state MUST produce bit-identical subsequent execution, given the same loaded OS ROM and sideways ROM images.
- **FR-024**: The `Snapshot` type MUST NOT include OS ROM or sideways ROM image bytes (those are large and externally-supplied); the consumer is responsible for re-loading the same ROM images before calling `Restore`.

#### Constitutional gates (carry forward from Phase 001)

- **FR-025**: All requirements MUST be validated by deterministic Go unit tests. Test coverage on the `bbc/` package MUST meet the ≥ 80 % delta-coverage threshold from the project constitution.
- **FR-026**: The `Tick` hot path MUST report zero allocations under `go test -bench -benchmem` (consistent with Phase 001's 0 B/op result).
- **FR-027**: All public types and methods MUST have Go doc comments explaining their contract; lint and vet MUST pass.
- **FR-028**: `Machine` MUST be documented as single-goroutine. All public methods (`Tick`, `Reset`, `AssertIRQ`, `AssertNMI`, `DeassertNMI`, `SetRDY`, ROM loaders, `Snapshot`, `Restore`) MUST be driven from the same goroutine; the implementation MUST NOT take internal locks on the `Tick` hot path. Concurrent access is the caller's responsibility (e.g., by serialising through a single emulator goroutine and using a separate lock-free channel/ring-buffer for cross-goroutine handoff).
- **FR-029**: The machine MUST expose `SetUnmappedAccessHook(hook func(addr uint16, write bool, value uint8))`. The hook fires on every read or write whose target the BBC memory map classifies as unmapped: FRED/JIM offsets not claimed by a peripheral, SHEILA offsets not in the FR-008 table, and sideways-window reads (`$8000`–`$BFFF`) when no image is loaded in the currently-selected bank. For writes, `value` is the byte the CPU attempted to write; for reads, `value` is the open-bus byte returned (`$FF`). A nil hook MUST be silently ignored, and the hot path MUST cost no more than a single nil-check when the hook is unset.

### Key Entities

- **Machine**: Top-level container. Owns one `mos6502.CPU`, one BBC memory map, and the collection of stub peripherals. Exposes `Tick`, `Reset`, control-surface pass-throughs, ROM loaders, and snapshot/restore. This is the only public type a future host or test harness needs to interact with.
- **MemoryMap**: Implements `mos6502.Memory`. Owns the 32 KB main RAM array, holds the OS ROM image reference, holds the array of (up to four) sideways ROM image references, and dispatches I/O reads/writes to the SHEILA decoder. Hidden from direct external use; accessed through `Machine`.
- **Stub peripheral** (CRTC, ACIA, Serial ULA, Video ULA, ROM-select, System VIA, User VIA, FDC, ADC, Tube, Econet, ACCCON): A register-file struct that records reads and writes at fixed offsets. Phase 002 implements no real behaviour beyond register storage and (for CRTC) index-then-data addressing. Each stub exposes `Snapshot`/`Restore` of its register file.
- **Snapshot**: Plain-old-data value type carrying CPU registers, main RAM contents, current sideways bank index, and the per-stub register file dumps. Serialisable via standard Go means (the byte-level encoding format is not fixed by Phase 002; only round-trip fidelity through `Snapshot`/`Restore` is required).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user-supplied OS 1.20 ROM (or any 16 KB BBC OS ROM image) loads and the machine runs at least 1,000,000 emulated cycles from RESET without firing the illegal-opcode hook and without any read or write landing at an address outside the BBC-defined map.
- **SC-002**: The Phase 001 bus-trace harness (or its Phase 002 equivalent) records the first 256 bus accesses after RESET; every access matches an address in the documented BBC map and the first instruction is fetched from the OS ROM region.
- **SC-003**: Every SHEILA address range listed in FR-008 has at least one round-trip test (write a unique byte, read it back, assert it matches — or assert the documented open-bus value for write-only/unmapped slots).
- **SC-004**: A two-bank paging test demonstrates that writing different bank indices to `$FE30` swaps the visible bytes at `$8000`, and switching back returns the original bytes, with zero exceptions across at least 10 round trips.
- **SC-005**: A snapshot-then-restore round-trip on a machine that has run for ≥ 100,000 cycles produces bit-identical CPU registers, bit-identical 32 KB main RAM, and bit-identical SHEILA stub register files compared to running the same machine forward without the round-trip.
- **SC-006**: `go test -bench=. -benchmem ./bbc/...` reports `0 B/op` and `0 allocs/op` on the `Tick` hot path; per-emulated-cycle wall time stays within 50 % of the standalone `mos6502` benchmark (i.e. ≤ ~6.5 ns/cycle on amd64, allowing for decoder + dispatch overhead).
- **SC-007**: `make fmt vet lint test bench cover` passes on the `bbc/` package with delta line coverage ≥ 80 %.

## Assumptions

- **Standard Model B configuration.** Four sideways ROM bank slots, 32 KB main RAM, no shadow RAM, no second processor (Tube). Master 128 / B+ extensions (extra RAM, ACCCON, paged-ROM-ID at `$FE34`) are out of scope for Phase 002.
- **No real peripheral behaviour.** Stub peripherals implement register-file storage only. Video output (CRTC, ULA), VIA timers / shift registers, FDC, ACIA, ADC, Tube, Econet — all behaviour beyond "read returns last-written value" is deferred to later phases.
- **No interrupts sourced from peripherals.** Phase 002 wires the IRQ/NMI control surface but does not have any peripheral assert these lines. MOS init code that polls VIA timers will see whatever stub register values were last written; tests that need MOS to advance past a VIA-timer-poll loop must pre-populate the VIA timer register stubs accordingly (acceptable for Phase 002 acceptance tests; full VIA behaviour comes in a later phase).
- **OS ROM is externally supplied at load time.** Phase 002 does NOT bundle, fetch, or distribute any copyrighted BBC ROM image. Tests use either a user-provided OS 1.20 binary (path supplied via env var, skipped if absent) or a small hand-crafted ROM stub that exercises the boot path without infringing copyright.
- **Sideways ROM images are externally supplied.** Same policy as OS ROM. Tests for sideways paging use hand-crafted 16 KB images (e.g. all `$AA`, all `$55`) rather than copyrighted language ROMs.
- **Open-bus value is `$FF`.** Real BBC open-bus values depend on the last byte on the bus and whether DRAM precharge has decayed; emulators conventionally return `$FF` and this is sufficient for Phase 002 correctness.
- **No save-state file format is locked in.** Snapshot/Restore round-trip via in-process Go values is the Phase 002 deliverable; serialisation to disk (gob, JSON, custom binary) is deferred until a UX phase needs it.
- **Cycle-stuffing / 1 MHz bus contention is Phase 003.** Phase 002 does not drive `RDY` on its own; the `SetRDY` pass-through exists so Phase 003's video ULA can wire it in.
- **Reuses Phase 001 verbatim.** The `mos6502.Memory` interface, `mos6502.CPU` control surface, `mos6502.Disassemble`, and `mos6502.CPU.SetTrace` are consumed unchanged. No modifications to the `mos6502/` package are in scope for Phase 002; if a defect is uncovered, it is filed as a separate fix outside this spec.
