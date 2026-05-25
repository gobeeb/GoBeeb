# Tasks: BBC Machine Layer

**Input**: Design documents from `/specs/002-bbc-machine/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Included — feature spec FR-025/FR-026 mandate Go unit tests + benchmarks; plan §Testing Standards lists per-region, per-peripheral, golden-trace, snapshot round-trip, and OS-ROM smoke tests as gates.

**Organization**: Tasks grouped by user story so each story can be implemented, validated, and committed independently.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: Which user story this task belongs to (US1, US2, US3, US4)
- All paths are repository-root-relative

## Path Conventions

Single Go module at repository root. The Phase 002 package is `bbc/` (sibling to `mos6502/`). Tests live alongside production code (`bbc/*_test.go`) per Go convention. Test fixtures live in `bbc/testdata/`.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Stand up the empty `bbc/` package and the test-data fixtures every later phase consumes.

- [X] T001 Create `bbc/` package directory and `bbc/doc.go` with a package-level doc comment naming the package, citing `mos6502`, and pointing at `specs/002-bbc-machine/`
- [X] T002 Create `bbc/testdata/` and `bbc/testdata/golden_traces/` directories (committed via a `.gitkeep` if empty)
- [X] T003 [P] Generate `bbc/testdata/stub_os_16k.bin` — hand-crafted 16 KB OS ROM stub with a reset vector at `$3FFC/$3FFD` pointing at a deterministic 6502 sequence (NOPs + a tiny loop) that touches RAM but no SHEILA addresses
- [X] T004 [P] Generate `bbc/testdata/stub_sideways_aa.bin` (16 KB of `$AA`)
- [X] T005 [P] Generate `bbc/testdata/stub_sideways_55.bin` (16 KB of `$55`)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Land the shared types (`Peripheral`, `UnmappedAccessHook`, errors), the `RomBanks`/`MemoryMap`/`Machine` skeletons, and the RAM-only `Read`/`Write` fast path. No user story can compile against the package until this phase lands.

**Critical**: No US task may begin until Phase 2 is complete.

- [X] T006 Define `Peripheral` interface (`Read(reg uint8) uint8`, `Write(reg uint8, value uint8)`) with doc comments in `bbc/peripheral.go`
- [X] T007 [P] Define `UnmappedAccessHook func(addr uint16, write bool, value uint8)` type with doc comments in `bbc/unmapped.go`
- [X] T008 [P] Define exported error vars `ErrNoOSROM`, `ErrInvalidROMSize`, `ErrBankOutOfRange`, `ErrRestoreMismatch` (per data-model §8) in `bbc/errors.go`
- [X] T009 Define `RomBanks` struct (`os [0x4000]byte`, `osLoaded bool`, `sideways [4][0x4000]byte`, `sidewaysLoaded [4]bool`, `bank uint8`) in `bbc/rom.go`
- [X] T010 Define `MemoryMap` struct (`ram [0x8000]byte`, plus `rom *RomBanks`, `periph *Peripherals`, `parent *Machine` pointers) in `bbc/memory_map.go`
- [X] T010a Define `Peripherals` container struct skeleton (empty fields — populated in US2) + a no-op `Zero()` method satisfying `ColdReset`'s FR-019 contract from the start in `bbc/peripheral.go`
- [X] T011 Define `Machine` struct + `New() *Machine` constructor that constructs a `mos6502.CPU` bound to `&Machine.mmap`, wires `mmap.rom = &m.rom` / `mmap.periph = &m.periph` / `mmap.parent = m`, and leaves `osLoaded=false` in `bbc/machine.go`
- [X] T012 Implement RAM-only `MemoryMap.Read`/`MemoryMap.Write` for `$0000`–`$7FFF` (every other address returns `$FF` from `Read`, drops on `Write` — proper region dispatch lands in later phases) in `bbc/memory_map.go`

**Checkpoint**: Package compiles, `Machine` instantiates, CPU can step against RAM-only memory, `Peripherals.Zero()` exists as a no-op seam ready for US2 to fill.

---

## Phase 3: User Story 1 — Boot the BBC Operating System ROM (Priority: P1) 🎯 MVP

**Goal**: A `Machine` can be constructed, given a 16 KB OS ROM image, reset, and ticked. The CPU fetches its reset vector from the OS ROM region, executes from there, and the `UnmappedAccessHook` never fires for OS-ROM-region accesses or RAM accesses.

**Independent Test**: `go test ./bbc/ -run TestBoot` runs against `bbc/testdata/stub_os_16k.bin` and asserts (a) PC inside OS ROM region after N cycles, (b) zero illegal-opcode hook fires, (c) zero unmapped-access hook fires for any access to a mapped region. The OS 1.20 smoke test is additionally available when `BBC_OS_ROM` env var is set.

### Tests for User Story 1

> Write these tests FIRST, ensure they fail, then implement.

- [X] T013 [P] [US1] OS ROM loader tests in `bbc/rom_test.go` — accepts exactly 16384 bytes, rejects shorter/longer with `ErrInvalidROMSize`, copies-on-load (mutating caller's slice after load does not change machine behaviour)
- [X] T014 [P] [US1] Machine construction + Reset/ColdReset tests in `bbc/machine_test.go` — `Reset()` and first `Tick()` return `ErrNoOSROM` when no ROM loaded; `Reset()` preserves RAM; `ColdReset()` zeros RAM; both clear bank latch to 0 and assert CPU reset; also assert idle stubs never pull `IRQ`/`NMI` low across ≥ 10 000 cycles of `Tick` with no peripheral interaction (satisfies FR-021)
- [X] T015 [P] [US1] `UnmappedAccessHook` tests in `bbc/unmapped_test.go` — hook fires on FRED/JIM/SHEILA reads/writes the map does not claim; hook silent on RAM and OS-ROM reads/writes; nil hook is silently ignored; `value` carries the open-bus `$FF` on reads and the attempted byte on writes
- [X] T016 [P] [US1] Golden bus trace test in `bbc/golden_trace_test.go` — captures the first **256** bus cycles after `Reset()` against `stub_os_16k.bin` via `mos6502.Trace`, compares to `testdata/golden_traces/reset_first256.trace`, AND asserts every captured access targets an address in the documented BBC memory map (satisfies SC-002 in full)
- [X] T017 [P] [US1] OS-ROM smoke test in `bbc/boot_os_test.go` — gated on `BBC_OS_ROM` env var (`t.Skip` if unset); loads the user-supplied OS 1.20 image, runs `Tick(1_000_000)`, asserts neither `IllegalOpcodeHook` nor `UnmappedAccessHook` ever fires (SC-001)

### Implementation for User Story 1

- [X] T018 [US1] Implement `Machine.LoadOSROM(image []byte) error` with 16384-byte size check + `copy(m.rom.os[:], image)` + `m.rom.osLoaded = true` in `bbc/rom.go`
- [X] T019 [US1] Extend `MemoryMap.Read` to serve OS ROM region `$C000`–`$FBFF` and `$FF00`–`$FFFF` (`return m.rom.os[addr-0xC000]`) in `bbc/memory_map.go`
- [X] T020 [US1] Extend `MemoryMap.Write` to silently drop writes targeting the OS ROM region in `bbc/memory_map.go`
- [X] T021 [US1] Implement `Machine.Reset()` — returns `ErrNoOSROM` if `!m.rom.osLoaded`; else clears `m.rom.bank = 0` and calls `m.cpu.AssertReset()`; RAM and peripheral state preserved — in `bbc/machine.go`
- [X] T022 [US1] Implement `Machine.ColdReset()` — same gate as `Reset()`, additionally zeros `m.mmap.ram`, clears the sideways bank latch, and calls `m.periph.Zero()` (seam landed in T010a; real per-stub zeroing arrives with T045 in US2) in `bbc/machine.go`
- [X] T023 [US1] Implement `Machine.Tick(cycles uint64) uint64` as a thin pass-through to `m.cpu.Run(cycles)` in `bbc/machine.go`
- [X] T024 [US1] Implement control-surface pass-throughs `AssertIRQ(bool)`, `AssertNMI()`, `DeassertNMI()`, `SetRDY(bool)` on `Machine` delegating to `m.cpu.*` in `bbc/machine.go`
- [X] T025 [US1] Implement `Machine.SetUnmappedAccessHook(UnmappedAccessHook)` + an internal `(m *MemoryMap) fireUnmapped(addr uint16, write bool, value uint8)` helper (single nil-check) in `bbc/unmapped.go`
- [X] T026 [US1] Wire `MemoryMap.Read`/`Write` to call `fireUnmapped` on accesses to `$FC00`–`$FEFF` (currently every byte there is unmapped — SHEILA decoder lands in US2) in `bbc/memory_map.go`
- [X] T027 [US1] Expose `Machine.CPU() *mos6502.CPU` accessor so tests and consumers can attach the Phase 001 `Trace` recorder + call `Registers()` (per quickstart) in `bbc/machine.go`
- [X] T028 [US1] Record `bbc/testdata/golden_traces/reset_first256.trace` fixture (256 bus cycles) by running the trace recorder once against `stub_os_16k.bin` after T026 lands, then commit the resulting file

**Checkpoint**: `make test` passes for US1 tests. Stub OS ROM boots, PC reaches OS-ROM region, zero unmapped hits during boot.

---

## Phase 4: User Story 2 — Route Peripheral I/O Through SHEILA (Priority: P2)

**Goal**: Every read/write in `$FE00`–`$FEFF` routes to the correct stub peripheral at the correct register offset; FRED/JIM unmapped slots return `$FF` and fire the hook; per-peripheral register-file round-trip works.

**Independent Test**: `go test ./bbc/ -run 'TestSHEILA|TestCRTC|TestVIA'` plays scripted CPU writes (via `Machine.CPU()` running synthetic NOP+STA programs assembled into `stub_os_16k.bin`-like fixtures, or by calling unexported `MemoryMap.Read`/`Write` from same-package tests) and asserts the correct stub registers received the access.

### Tests for User Story 2

- [X] T029 [P] [US2] SHEILA decoder routing tests in `bbc/sheila_test.go` — table-driven across every address range in FR-008; writes a unique byte, reads it back, asserts equality or open-bus `$FF` for write-only/unmapped slots
- [X] T030 [P] [US2] CRTC index-then-data tests in `bbc/crtc_test.go` — FR-010 acceptance: write index → write data → read data returns stored byte; write index ≥ 18 → masked to `0..17`; read of `$FE00` returns `$FF`
- [X] T031 [P] [US2] System VIA + User VIA round-trip and 16-byte mirror tests in `bbc/via_test.go` — write at `$FE40`, read at `$FE50` returns same byte; same for `$FE60` and `$FE70`
- [X] T032 [P] [US2] FRED ($FC00–$FCFF) + JIM ($FD00–$FDFF) unmapped behaviour tests in `bbc/sheila_test.go` (or a sibling `bbc/fred_jim_test.go`) — reads return `$FF`, writes silently drop, hook fires with correct `addr`/`write`/`value`
- [X] T033 [P] [US2] Golden trace `testdata/golden_traces/crtc_index_then_data.trace` regression test in `bbc/golden_trace_test.go`
- [X] T034 [P] [US2] Golden trace `testdata/golden_traces/via_register_round_trip.trace` regression test in `bbc/golden_trace_test.go`

### Implementation for User Story 2

- [X] T035 [P] [US2] CRTC stub (`type CRTC struct { selected uint8; regs [18]byte }` + index/data semantics, FR-010) in `bbc/crtc.go`
- [X] T036 [P] [US2] ACIA stub (`type ACIA struct { regs [4]byte }`, mirrored over 8 bytes) in `bbc/acia.go`
- [X] T037 [P] [US2] SerialULA stub (`type SerialULA struct { regs [1]byte }`, mirrored over 16 bytes) in `bbc/serial_ula.go`
- [X] T038 [P] [US2] VideoULA stub (`type VideoULA struct { regs [2]byte }`, mirrored over 16 bytes) in `bbc/video_ula.go`
- [X] T039 [P] [US2] ACCCON stub (`type ACCCON struct{}`, `Read` returns `$FF`, `Write` is a no-op — Model B does not implement) in `bbc/acccon.go`
- [X] T040 [P] [US2] VIA stub (`type VIA struct { regs [16]byte }`, last-write-wins) in `bbc/via.go` — used by both System and User VIA
- [X] T041 [P] [US2] FDC stub (`type FDC struct { regs [4]byte }`, all reads return `$FF` per FR-008 — full behaviour deferred to Phase 007) in `bbc/fdc.go`
- [X] T042 [P] [US2] Econet stub (`type Econet struct { regs [4]byte }`, all reads return `$FF`) in `bbc/econet.go`
- [X] T043 [P] [US2] ADC stub (`type ADC struct { regs [4]byte }`, last-write-wins) in `bbc/adc.go`
- [X] T044 [P] [US2] Tube stub (`type Tube struct { regs [8]byte }`, last-write-wins) in `bbc/tube.go`
- [X] T045 [US2] Populate the `Peripherals` container (skeleton landed in T010a) by adding fields for every stub by value (per data-model §4) + extend the existing `Zero()` helper to zero every stub's register file in `bbc/peripheral.go` — depends on T035–T044
- [X] T046 [US2] Implement SHEILA decoder (`(m *MemoryMap) ioRead(addr uint16) uint8`, `ioWrite(addr uint16, value uint8)`) as a two-level switch — outer on high nibble of low byte, inner on low nibble / peripheral stride — dispatching to `m.periph.*` per FR-008 in `bbc/memory_map.go`; depends on T045
- [X] T047 [US2] Wire FRED ($FC00–$FCFF) + JIM ($FD00–$FDFF) — every offset returns `$FF` on read, drops writes, fires `fireUnmapped` — and replace the placeholder US1 unmapped-everything block in `bbc/memory_map.go` so only genuinely unmapped SHEILA/FRED/JIM offsets fire the hook
- [X] T048 [US2] Record `bbc/testdata/golden_traces/crtc_index_then_data.trace` and `bbc/testdata/golden_traces/via_register_round_trip.trace` fixtures by running the trace recorder against synthetic test programs

**Checkpoint**: Every SHEILA address range has a passing round-trip test; FRED/JIM behave per spec; ColdReset's `Peripherals.Zero()` now does real work.

---

## Phase 5: User Story 3 — Page Sideways ROM Banks via `$FE30` (Priority: P3)

**Goal**: Loading two distinct 16 KB images into banks 0 and 1, writing the bank index to `$FE30`, and reading `$8000` returns the selected bank's byte. Empty banks return `$FF`. Writes to `$8000`–`$BFFF` are silently dropped.

**Independent Test**: `go test ./bbc/ -run TestSideways` against the `stub_sideways_aa.bin` and `stub_sideways_55.bin` fixtures asserts every acceptance scenario in spec User Story 3.

### Tests for User Story 3

- [X] T049 [P] [US3] Sideways paging tests in `bbc/sideways_test.go` — covers all 5 acceptance scenarios (bank 0 read, bank-switch read, swap-back, empty-bank `$FF`, write-drops) PLUS an explicit loop that performs ≥ 10 alternating bank-swap round trips and asserts zero mismatches across all iterations (satisfies SC-004 in full)
- [X] T050 [P] [US3] Sideways ROM loader tests in `bbc/rom_test.go` (extending T013) — accepts exactly 16 KB; rejects wrong size with `ErrInvalidROMSize`; rejects `bank < 0`/`> 3` with `ErrBankOutOfRange`; copy-on-load isolation
- [X] T051 [P] [US3] Golden trace `testdata/golden_traces/rom_select_swap.trace` regression test in `bbc/golden_trace_test.go`

### Implementation for User Story 3

- [X] T052 [US3] Implement `Machine.LoadSidewaysROM(bank int, image []byte) error` with size check, bank range check, `copy(m.rom.sideways[bank][:], image)`, set `m.rom.sidewaysLoaded[bank] = true` in `bbc/rom.go`
- [X] T053 [US3] Implement `(m *MemoryMap) writeRomSelect(value uint8)` that stores `value & 0x03` into `m.rom.bank` in `bbc/rom_select.go`
- [X] T054 [US3] Extend `MemoryMap.Read` to serve `$8000`–`$BFFF` from `m.rom.sideways[m.rom.bank]` when `sidewaysLoaded[bank]` is true in `bbc/memory_map.go`
- [X] T055 [US3] Extend `MemoryMap.Read`/`Write` for `$8000`–`$BFFF`: writes silently dropped, reads on an empty bank return `$FF` and fire `fireUnmapped` in `bbc/memory_map.go`
- [X] T056 [US3] Wire `$FE30`–`$FE33` writes through the SHEILA decoder to `writeRomSelect` (`$FE30` reads return open-bus `$FF`) in `bbc/memory_map.go`; remove any placeholder in T046
- [X] T057 [US3] Record `bbc/testdata/golden_traces/rom_select_swap.trace` fixture

**Checkpoint**: Paging round-trips work across all 4 banks; empty-bank reads fire the unmapped hook; writes to sideways window drop.

---

## Phase 6: User Story 4 — Snapshot and Restore Machine State (Priority: P4)

**Goal**: Two `Machine` instances loaded with the same ROMs converge to bit-identical CPU + RAM + peripheral state after `Snapshot` → `Restore` → `Tick`.

**Independent Test**: `go test ./bbc/ -run TestSnapshot` runs the full round-trip across all three acceptance scenarios.

### Tests for User Story 4

- [X] T058 [P] [US4] Snapshot/Restore round-trip tests in `bbc/snapshot_test.go` — all 3 acceptance scenarios + SC-005 (≥ 100 000 cycles before snapshot; bit-identical CPU registers, 32 KB RAM, every peripheral register file after restore + further Tick); also assert via `reflect.TypeOf(bbc.Snapshot{})` that no field carries the OS-ROM or sideways-ROM image bytes (satisfies FR-024)

### Implementation for User Story 4

- [X] T059 [P] [US4] Define `Snapshot`, `PeripheralSnapshot`, and per-stub `*Snapshot` types (per data-model §6) in `bbc/snapshot.go`
- [X] T060 [US4] Add `Snapshot()`/`Restore()` methods on every peripheral stub (CRTC, ACIA, SerialULA, VideoULA, VIA, FDC, Econet, ADC, Tube, ACCCON) — each lives in the stub's own file (`crtc.go`, `acia.go`, …) for ownership clarity
- [X] T061 [US4] Implement `Machine.Snapshot() Snapshot` collecting CPU registers (`m.cpu.Registers()`), RAM (`m.mmap.ram`), bank index, `sidewaysLoaded`, and every peripheral snapshot in `bbc/snapshot.go`; depends on T060
- [X] T062 [US4] Implement `Machine.Restore(s Snapshot) error` — returns `ErrRestoreMismatch` if `s.SidewaysLoaded != m.rom.sidewaysLoaded`; else restores CPU via `m.cpu.SetRegisters`, RAM via copy, bank index, every peripheral via its `Restore` method in `bbc/snapshot.go`

**Checkpoint**: Round-trip yields bit-identical state across two `Machine` instances over 100 K+ cycles.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Hit the constitutional gates (zero-alloc benchmarks, coverage, fmt/vet/lint) and update project docs to reflect Phase 002 completion.

- [X] T063 [P] Benchmarks in `bbc/bench_test.go` — `BenchmarkTickNoop` (NOP-filled stub OS ROM) and `BenchmarkTickMixedWorkload` (synthetic ROM exercising RAM + OS ROM + sideways + a SHEILA write per iteration); both assert `0 B/op 0 allocs/op` via `b.ReportAllocs()` and meet the ≤ ~6.5 ns/cycle target on amd64 (SC-006)
- [X] T064 [P] Flesh out `bbc/doc.go` with the full package-level API overview (memory map, Tick semantics, single-goroutine contract, snapshot rules)
- [X] T065 [P] Verify `make fmt vet lint test bench cover` passes with ≥ 80 % delta line coverage on `bbc/` (SC-007); if a file dips below threshold, add the gap-filling test in the same commit. Additionally add a reflection-based test (in `bbc/machine_test.go`) asserting `Machine` and `MemoryMap` contain no `sync.Mutex`, `sync.RWMutex`, `atomic.*`, or channel fields — codifies FR-028's no-internal-locks contract.
- [X] T066 Update `CLAUDE.md` "Implemented packages" section with a `bbc/` entry mirroring the `mos6502/` entry's shape (public API summary, validation status, known limitations, benchmarks)
- [X] T067 Update `docs/roadmap.md` Phase 002 section — mark complete, record bench numbers, link to `specs/002-bbc-machine/`, note any deferred items handed off to Phase 003+ (real VIA timers, CRTC scanline timing, FDC behaviour)
- [X] T068 Run the `quickstart.md` examples end-to-end against the built package (minimal program, unmapped hook, snapshot round-trip) and resolve any drift between quickstart sample code and the actual API

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)** → no dependencies, can start immediately
- **Foundational (Phase 2)** → depends on Setup; BLOCKS every user story
- **US1 (Phase 3)** → depends on Foundational
- **US2 (Phase 4)** → depends on Foundational; benefits from US1 (its tests use `Machine.CPU()` to drive synthetic programs) but the peripheral stubs themselves can be implemented in parallel with US1 once Phase 2 is done
- **US3 (Phase 5)** → depends on Foundational + US2's `Peripherals` container (so the SHEILA decoder has somewhere to route `$FE30`)
- **US4 (Phase 6)** → depends on US1 + US2 + US3 (every piece of state Snapshot captures must exist first)
- **Polish (Phase 7)** → depends on US1–US4

### Within Each User Story

- Tests written and failing before implementation
- Stubs (US2 T035–T044) parallelizable; container (T045) and decoder wiring (T046–T048) serialize behind them
- Implementation depends only on Foundational + intra-story prerequisites

### Parallel Opportunities

- T003/T004/T005 (test data fixtures) — different files, all parallel
- T007/T008 (foundational type defs in different files) — parallel
- T013–T017 (US1 tests in five different files) — parallel
- T029–T034 (US2 tests in different files) — parallel
- T035–T044 (US2 peripheral stubs, one file per peripheral) — fully parallel; the decoder (T046) blocks behind them
- T049–T051 (US3 tests) — parallel
- T063/T064/T065 (polish across different files) — parallel

---

## Parallel Example: User Story 2 implementation

```bash
# After Phase 2 + the US2 tests (T029–T034) are red, launch every peripheral stub in parallel:
Task: "Implement CRTC stub in bbc/crtc.go"          # T035
Task: "Implement ACIA stub in bbc/acia.go"          # T036
Task: "Implement SerialULA stub in bbc/serial_ula.go"  # T037
Task: "Implement VideoULA stub in bbc/video_ula.go" # T038
Task: "Implement ACCCON stub in bbc/acccon.go"      # T039
Task: "Implement VIA stub in bbc/via.go"            # T040
Task: "Implement FDC stub in bbc/fdc.go"            # T041
Task: "Implement Econet stub in bbc/econet.go"      # T042
Task: "Implement ADC stub in bbc/adc.go"            # T043
Task: "Implement Tube stub in bbc/tube.go"          # T044
```

---

## Implementation Strategy

### MVP First (US1 only)

1. Phase 1 (Setup) + Phase 2 (Foundational) → package compiles, RAM works.
2. Phase 3 (US1) → stub OS ROM boots, hooks land, control surface wired.
3. STOP and VALIDATE: `go test ./bbc/ -run TestBoot`. Optionally run the OS 1.20 smoke test with `BBC_OS_ROM=…`.

### Incremental Delivery

1. Setup + Foundational → foundation ready.
2. Add US1 → stub OS ROM smoke + Reset/ColdReset/control surface (commit).
3. Add US2 → SHEILA decoder + every peripheral stub (commit; OS 1.20 smoke now passes further into MOS init).
4. Add US3 → sideways paging; BASIC ROM bring-up unblocks (commit).
5. Add US4 → snapshot/restore (commit).
6. Polish (Phase 7) → benchmarks, coverage, roadmap doc update, CLAUDE.md (commit).

### Parallel Team Strategy

With multiple contributors:

- One pair completes Phase 1 + Phase 2 together.
- After Foundational lands:
  - Dev A: US1 (boot path, hooks, control surface)
  - Dev B: US2 peripheral stubs (T035–T044 in parallel)
  - Dev C: prepare US3 sideways tests + loader against Foundational
- US4 (snapshot) is the integrative phase — one dev pulls it together once US1–US3 are merged.

---

## Notes

- Tests live alongside production code per Go convention (`bbc/*_test.go`), not under `tests/`.
- The `Peripherals` skeleton (T010a) lets US1's `ColdReset` (T022) compile and call `Zero()` from day one; T045 then populates the container with the real stub fields and extends `Zero()` to actually clear them. The SHEILA decoder (T046) depends on T045 having added the fields.
- T028, T048, T057 capture golden trace fixtures *after* the corresponding implementation lands — they are the "regression lock" the matching golden-trace test asserts against.
- ROM-select latch (T053/T056) lives on `RomBanks`, not as a SHEILA peripheral stub — `$FE30`–`$FE33` writes are decoded straight to `writeRomSelect` (per data-model §4 callout).
- `Peripherals.Zero()` (T010a no-op → T045 real impl) honours FR-019 incrementally: US1's `ColdReset` zeros RAM and calls the no-op `Zero()`; once US2 lands the populated container, `Zero()` clears every stub's register file without any change to `ColdReset` itself.
- Commit after each task or logical group; stop at any checkpoint to validate independently.
- Avoid: same-file conflicts on `bbc/memory_map.go` and `bbc/machine.go` (these two files are touched across most stories — serialize edits to them).
