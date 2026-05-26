# GoBeeb Roadmap

This roadmap captures the phase plan for GoBeeb on the **Go + SDL2 + cimgui-go** stack, per [ADR-0001](./adr/0001-language-go-vs-zig.md).

## Phase status

| Phase | Name                                 | Status      | Branch                       |
|-------|--------------------------------------|-------------|------------------------------|
| 001   | 6502 CPU core                        | ✅ Complete  | `001-cpu-6502-core`          |
| 002   | BBC machine layer                    | ✅ Complete  | `002-bbc-machine`            |
| 003   | CPU bus-cycle validation (Tom Harte) | ✅ Complete  | `003-cpu-processor-tests`    |
| 004   | Video ULA + framebuffer              | 🟡 Planned  | `004-video-ula`              |
| 005   | SDL2 host + ImGui debug              | 🟡 Planned  | `005-sdl-host`               |
| 006   | Sound (SN76489)                      | ⚪ Backlog   | `006-sound-sn76489`          |
| 007   | Keyboard / joystick                  | ⚪ Backlog   | `007-input`                  |
| 008   | Disc (1770 FDC) + tape               | ⚪ Backlog   | `008-storage`                |

Legend: ✅ complete · 🟡 planned (spec required next) · ⚪ backlog (not yet specified).

---

## Phase 002 — BBC machine layer (`bbc/`)

**Goal**: wire the CPU to a BBC memory map, page in the OS ROM, decode SHEILA (`$FC00`–`$FEFF`) to stub peripherals, and provide a host-callable `Tick(cycles)` entry point. **No UI yet.**

**Scope**:

- `bbc.Machine` struct owning a `*mos6502.CPU` and a `bbc.MemoryMap` implementing `mos6502.Memory`.
- BBC memory map: 32 KB main RAM (`$0000`–`$7FFF`), 16 KB sideways ROM bank window (`$8000`–`$BFFF`, MOS-paged), 16 KB OS ROM (`$C000`–`$FBFF` + `$FF00`–`$FFFF`), SHEILA I/O (`$FC00`–`$FEFF`).
- Sideways ROM bank latch at `$FE30`.
- SHEILA decoder routes addresses to stub peripherals (no behaviour yet, just observable reads/writes for tests).
- 6845 CRTC stub: register file only, no rendering.
- VIA stub (System VIA `$FE40`–`$FE4F`, User VIA `$FE60`–`$FE6F`): register file only.
- `bbc.Machine.LoadOSROM([]byte)` constructor helper.
- Cycle accounting respects `mos6502.CPU.Step()` return value; no separate clock yet.

**Reuses**: `mos6502.Memory` interface verbatim. `mos6502.Disassemble` for any future test-time tracing.

**Exit criteria**:

- OS 1.20 ROM loaded, RESET vector fetched correctly, CPU runs into MOS startup code without crashing on undefined memory.
- Bus-trace integration test confirms the BBC memory map routes a representative MOS-init instruction stream to the right (stub) peripherals.
- ≥ 80 % delta coverage on `bbc/` per constitution Principle II.

**Status — complete (2026-05-25)**: 78 deterministic tests pass on `bbc/`; 97.2 % coverage; `BenchmarkTickNoop` ~5.4 ns/cycle and `BenchmarkTickMixedWorkload` ~5.3 ns/cycle on amd64 (both 0 B/op, 0 allocs/op — comfortably under the ≤ ~6.5 ns/cycle SC-006 budget). Golden bus traces locked for reset, CRTC index/data, VIA round-trip, ROM-select swap. OS 1.20 smoke test gated on `BBC_OS_ROM` (not redistributed). Deferred items handed off to later phases: real CRTC scanline timing (Phase 003), VIA timers / interrupts (Phase 005/007 when peripherals come online), real 1770 FDC behaviour (Phase 007). See [specs/002-bbc-machine/](../specs/002-bbc-machine/).

---

## Phase 003 — CPU bus-cycle validation (Tom Harte ProcessorTests)

**Goal**: validate cycle-accurate bus behaviour of every documented NMOS 6502 opcode against the [SingleStepTests/ProcessorTests](https://github.com/SingleStepTests/ProcessorTests) corpus — 10,000 JSON cases per opcode, ~1.51M tests across the 151 documented opcodes. Closes the validation gap left by the Klaus Dormann ROM, which only checks final state and cannot see per-cycle read/write order.

**Scope**:

- New test file `mos6502/processortests_test.go` consuming the upstream JSON corpus.
- `go:generate` directive (e.g. on a new `mos6502/gen.go`) that fetches `SingleStepTests/ProcessorTests` subdir `6502/v1/*.json` into `mos6502/testdata/processortests/` at a pinned commit SHA. Directory is gitignored; fetch script is idempotent.
- Sparse-memory test adapter (`map[uint16]uint8` implementing `mos6502.Memory`) populated from each JSON case's `initial.ram` entries.
- Per-test flow per JSON case:
  1. Parse `{name, initial: {pc, s, a, x, y, p, ram}, final: {...}, cycles: [[addr, value, "read"|"write"]]}`.
  2. Construct CPU on sparse memory; `SetRegisters(initial)`; populate RAM; attach `Trace`.
  3. `cpu.Step()` once.
  4. Assert `Registers()` == `final`; assert RAM mutations == `final.ram`; assert `Trace.Snapshot()` matches `cycles` exactly (cycle count + each `{addr, value, kind}`).
- One Go subtest per opcode (`t.Run("0xA9_LDA_imm", ...)`) with `t.Parallel()`; sampled subset under `testing.Short()`.
- Skip list covers the 105 illegal NMOS opcodes (constant in test file) — deferred to a future phase.
- Pin upstream commit SHA in test source for reproducibility.

**Reuses** (zero new public API on `mos6502`):

- `mos6502.Trace` / `BusEvent` / `Trace.Snapshot()` — already records every bus cycle.
- `mos6502.CPU.Registers()` / `SetRegisters()` — state setup + assert.
- `mos6502.Memory` interface — sparse-map adapter is trivial.
- Klaus Dormann `functional_test.go` pattern for harness shape (corpus itself is fetched, not embedded).

**Exit criteria**:

- All 151 documented opcodes pass 10,000 cases each (~1.51M passing tests).
- Pre-existing tests (functional ROM, golden traces, unit tests) still pass; coverage on `mos6502/` does not regress below current 99.3 %.
- `go generate ./mos6502/` reproducibly fetches the pinned corpus; documented in `mos6502/` quickstart or README.
- `go test -short ./mos6502/` runs a representative sampled subset in CI-acceptable time; full corpus runs under a non-short tag or build flag.
- Pinned `SingleStepTests/ProcessorTests` commit SHA recorded in test source.

**Out of scope** (tracked as follow-ups, not in this phase):

- Implementing the 105 undocumented NMOS opcodes (LAX, SAX, DCP, ISB, RLA, RRA, SLO, SRE, ANC, ARR, ASR, LAS, XAA, AHX, SHX, SHY, TAS, KIL). Current stub treats them as 2-cycle NOP and would fail Tom Harte expectations — explicitly skipped here.
- 65C02 / 65816 variants.
- Reset / IRQ / NMI sequences (not part of this corpus; covered by existing `interrupts_test.go`).

**Status — complete (2026-05-26)**: 151 documented NMOS opcodes pass 10,000 cases each against `SingleStepTests/ProcessorTests` pinned at SHA `bb117564…` (10K × 151 ≈ 1.51M assertions). 105 undocumented opcodes reported as SKIP via auditable `skipList`. Full corpus run: ~4 s wall-time on amd64 (SC-008 budget ≤ 5 min). Short mode (`-short`, 100 cases/opcode): ~4 s. `mos6502/` coverage holds at 99.3 %. Bench `BenchmarkRunMixedWorkload` ~4.4 ns/cycle (within margin of pre-phase 4.24 ns/cycle baseline; 0 allocs). Zero new public API, zero new runtime deps. See [specs/003-cpu-processor-tests/](../specs/003-cpu-processor-tests/).

---

## Phase 004 — Video ULA + framebuffer (`video/`)

**Goal**: produce a `[]uint8` 8-bit-indexed framebuffer matching what a real BBC Model B would render for MODE 0–7 from a representative MOS init. **Still no SDL.**

**Scope**:

- 6845 CRTC: register file → scanline + cursor + frame timing.
- Video ULA at `$FE20`–`$FE21`: palette latch, control register (`$FE20`), framebuffer mode decode.
- Pixel pipeline: bitmap modes 0–6 + teletext mode 7 (SAA5050-equivalent rendering).
- 1 MHz / 2 MHz bus contention model exposed via `RDY` on the CPU (uses `mos6502.CPU.SetRDY` already implemented).
- `video.Renderer.Frame() []uint8` returns the current 640×512 8-bit-indexed framebuffer.
- Palette is the standard 8-colour BBC palette; mode 7 uses the SAA5050 ROM glyphs.

**Reuses**: `mos6502.CPU.SetRDY` for bus contention. `bbc.MemoryMap` extended to route `$FE20`–`$FE21` to the ULA and `$FE00`–`$FE07` to the CRTC.

**Exit criteria**:

- MOS 1.20 boot renders the `BBC Computer 32K` banner in MODE 7 to the framebuffer.
- A `golden_frames/` set of `.png` references for MODE 0–7 boot screens matches the generated framebuffer pixel-for-pixel.
- Sub-cycle bus contention test: timing-sensitive demo (e.g. a known cycle-stuffing test ROM) produces the expected scanline pattern.

---

## Phase 005 — SDL2 host + ImGui debugger (`host/sdl/`)

**Goal**: first time a user sees pixels on screen. SDL2 window blitting the Phase 004 framebuffer, audio callback wired up for Phase 006, ImGui overlay showing CPU/memory state.

**Scope**:

- `host/sdl/` package owns the main thread (`runtime.LockOSThread` on `main`).
- Bindings: `github.com/AllenDang/cimgui-go` (uses cgo) + cimgui-go's bundled `backend/sdlbackend` (SDL2 + OpenGL).
- Window: 1280×1024 (2× the BBC's 640×512), resizable, integer scaling.
- Render loop: 50 Hz emulator step → framebuffer → SDL_Texture upload → ImGui draw → swap.
- Audio: SDL2 audio callback pulls from a preallocated lock-free ring buffer. **No allocations in the audio callback.** Audio goroutine pinned via `runtime.LockOSThread`. (Audio source itself lands in Phase 006; Phase 005 wires the callback with a silent stub.)
- ImGui overlay (toggle via F12):
  - CPU register window (live `A`, `X`, `Y`, `SP`, `PC`, `P` flags).
  - Disassembly window using `mos6502.Disassemble` from the current `PC`.
  - Memory hex view (1 KB pages, jump-to-address).
  - Cycle counter + emulated MHz.
- Input: SDL keyboard events → BBC keyboard matrix stub (real mapping is Phase 007).

**Reuses**: `mos6502.Disassemble`, `mos6502.CPU.Registers`, `bbc.Machine.Tick`, `video.Renderer.Frame`.

**Dependencies introduced** (first non-stdlib runtime deps in the project):

- `github.com/AllenDang/cimgui-go` (cgo, SDL2 backend).
- Transitively SDL2 system library; documented build requirement in `README` + `Makefile`.

**Exit criteria** (also serve as ADR-0001 verification):

- Window opens, MOS boot banner visible in MODE 7.
- ImGui overlay renders at ≥ 60 fps with emulator running.
- `GODEBUG=gctrace=1` shows no GC pause > 5 ms during a 60-second run.
- CPU throughput within 10 % of standalone `mos6502` benchmark (≤ 138 ns/emulated cycle).
- F12 toggles overlay; reg/disasm/memory views all live.
- Clean shutdown on window close.

**ADR re-evaluation gate**: completion of Phase 005 is the explicit point to re-read ADR-0001 and decide whether the SDL2 + cgo + GC stack is meeting the verification criteria. After Phase 005 the cost of reversing the language decision rises sharply.

---

## Phase 006+ — backlog (out of scope for this roadmap pass)

- **Phase 006 — Sound**: SN76489 emulation + audio source for the Phase 005 ring buffer.
- **Phase 007 — Input**: full BBC keyboard matrix, optional joystick.
- **Phase 008 — Storage**: 1770 FDC for SSD/DSD discs, UEF tape loader.

These will get their own spec / plan / tasks via the standard spec-kit flow (`/speckit-specify` → `/speckit-plan` → `/speckit-tasks`) before promotion to "planned".

---

## Cross-cutting concerns (apply to every phase)

- **Constitution gates** (`fmt vet lint test bench cover`) MUST pass per phase; ≥ 80 % delta coverage; deterministic tests only.
- **Performance budgets**: emulator hot path stays zero-allocation; ≤ 125 ns per emulated CPU cycle on amd64 (SC-006 from Phase 001 carries forward).
- **No hidden state**: every subsystem exposes its registers via a `Snapshot() T` / `Restore(T)` pair for future save-state support.
- **Spec-kit flow**: each phase begins with `/speckit-specify`, never with code.
