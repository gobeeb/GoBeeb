# GoBeeb Roadmap

This roadmap captures the phase plan for GoBeeb on the **Go + SDL2 + cimgui-go** stack, per [ADR-0001](./adr/0001-language-go-vs-zig.md).

## Phase status

| Phase | Name                    | Status      | Branch                  |
|-------|-------------------------|-------------|-------------------------|
| 001   | 6502 CPU core           | ✅ Complete  | `001-cpu-6502-core`     |
| 002   | BBC machine layer       | ✅ Complete  | `002-bbc-machine`       |
| 003   | Video ULA + framebuffer | 🟡 Planned  | `003-video-ula`         |
| 004   | SDL2 host + ImGui debug | 🟡 Planned  | `004-sdl-host`          |
| 005   | Sound (SN76489)         | ⚪ Backlog   | `005-sound-sn76489`     |
| 006   | Keyboard / joystick     | ⚪ Backlog   | `006-input`             |
| 007   | Disc (1770 FDC) + tape  | ⚪ Backlog   | `007-storage`           |

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

## Phase 003 — Video ULA + framebuffer (`video/`)

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

## Phase 004 — SDL2 host + ImGui debugger (`host/sdl/`)

**Goal**: first time a user sees pixels on screen. SDL2 window blitting the Phase 003 framebuffer, audio callback wired up for Phase 005, ImGui overlay showing CPU/memory state.

**Scope**:

- `host/sdl/` package owns the main thread (`runtime.LockOSThread` on `main`).
- Bindings: `github.com/AllenDang/cimgui-go` (uses cgo) + cimgui-go's bundled `backend/sdlbackend` (SDL2 + OpenGL).
- Window: 1280×1024 (2× the BBC's 640×512), resizable, integer scaling.
- Render loop: 50 Hz emulator step → framebuffer → SDL_Texture upload → ImGui draw → swap.
- Audio: SDL2 audio callback pulls from a preallocated lock-free ring buffer. **No allocations in the audio callback.** Audio goroutine pinned via `runtime.LockOSThread`. (Audio source itself lands in Phase 005; Phase 004 wires the callback with a silent stub.)
- ImGui overlay (toggle via F12):
  - CPU register window (live `A`, `X`, `Y`, `SP`, `PC`, `P` flags).
  - Disassembly window using `mos6502.Disassemble` from the current `PC`.
  - Memory hex view (1 KB pages, jump-to-address).
  - Cycle counter + emulated MHz.
- Input: SDL keyboard events → BBC keyboard matrix stub (real mapping is Phase 006).

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

**ADR re-evaluation gate**: completion of Phase 004 is the explicit point to re-read ADR-0001 and decide whether the SDL2 + cgo + GC stack is meeting the verification criteria. After Phase 004 the cost of reversing the language decision rises sharply.

---

## Phase 005+ — backlog (out of scope for this roadmap pass)

- **Phase 005 — Sound**: SN76489 emulation + audio source for the Phase 004 ring buffer.
- **Phase 006 — Input**: full BBC keyboard matrix, optional joystick.
- **Phase 007 — Storage**: 1770 FDC for SSD/DSD discs, UEF tape loader.

These will get their own spec / plan / tasks via the standard spec-kit flow (`/speckit-specify` → `/speckit-plan` → `/speckit-tasks`) before promotion to "planned".

---

## Cross-cutting concerns (apply to every phase)

- **Constitution gates** (`fmt vet lint test bench cover`) MUST pass per phase; ≥ 80 % delta coverage; deterministic tests only.
- **Performance budgets**: emulator hot path stays zero-allocation; ≤ 125 ns per emulated CPU cycle on amd64 (SC-006 from Phase 001 carries forward).
- **No hidden state**: every subsystem exposes its registers via a `Snapshot() T` / `Restore(T)` pair for future save-state support.
- **Spec-kit flow**: each phase begins with `/speckit-specify`, never with code.
