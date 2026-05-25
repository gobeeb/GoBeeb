# ADR 0001: Use Go (not Zig) as the implementation language for GoBeeb

- **Status**: Accepted
- **Date**: 2026-05-25
- **Deciders**: Richard Case
- **Related**: [Roadmap](../roadmap.md), [mos6502 package](../../mos6502/), [Constitution](../../.specify/memory/constitution.md)

## Context

GoBeeb is a hobby BBC Model B emulator. Phase 001 (`mos6502/`, NMOS 6502 CPU core) is complete in Go: 99.3 % test coverage, Klaus Dormann `6502_functional_test` passing, ~4.3 ns/emulated cycle on amd64 (≈ 30× the 8 MHz target), zero allocations on hot path, golden bus-trace harness, 91 unit tests. Constitution and spec-kit planning artifacts are written around Go.

The next major subsystem after CPU is the host shell: window, input, audio, and a Dear ImGui debugger overlay. The original intent was SDL3 + Dear ImGui. The question raised: would Zig be a better implementation language given that target stack?

The investigation produced four relevant findings:

1. **`cimgui-go` ships an SDL2 backend, not SDL3** (verified against the repo README and `pkg.go.dev`, May 2026). Backends offered: GLFW, SDL2, Ebitengine, DRM/EGL, RayLib. Upstream Dear ImGui has `imgui_impl_sdl3.cpp`, but `cimgui-go` has not wrapped it.
2. **SDL3 is not load-bearing** for a BBC Model B emulator — no HDR, no GPU-API requirement, no SDL3-only feature is needed. SDL2 covers the use case fully and will be supported for years.
3. **Zig's `@cImport` is genuinely cleaner** for SDL3 + Dear ImGui than Go's `cgo` + `cimgui-go`. This is the one place Zig wins on practical grounds for *this exact stack*, not theoretical grounds.
4. **The cost of switching is non-trivial**: throwing away a validated cycle-accurate CPU implementation, re-deriving NMI hijack / RMW double-write / BCD `N`/`V`/`Z` edge cases, rewriting the spec-kit constitution and CI gates to be Zig-flavored, and learning Zig from zero against a pre-1.0 language with churning APIs.

## Decision

**Stay on Go. Use SDL2 (via `cimgui-go`'s bundled SDL2 backend) for windowing/input/audio. Use `cimgui-go` for the Dear ImGui debugger overlay.**

Revisit only if (a) a concrete SDL3-only feature becomes load-bearing for the emulator, *or* (b) measured audio jitter under Go's GC cannot be mitigated by preallocation + `runtime.LockOSThread` + `GOGC` tuning.

## Considered Alternatives

### Alt A — Switch entire project to Zig + SDL3 + cimgui (via `@cImport`)

- ✅ First-class C interop: no `cgo`, trivial SDL3 and ImGui binding via `@cImport`.
- ✅ No GC: audio callbacks become trivially jitter-free.
- ✅ `comptime` opcode dispatch could replace the 256-entry function-pointer table.
- ✅ Models hardware idiomatically (packed structs, explicit `u8`/`u16`).
- ❌ Throw away a passing Klaus Dormann implementation (~6–10 days to re-derive).
- ❌ Zero existing Zig experience; pre-1.0 language with breaking changes between minor versions.
- ❌ Spec-kit / constitution / CI all Go-flavored; rewrite required.
- ❌ Thinner tooling: no `pprof`, no `go test -fuzz`, no race detector, custom coverage gate needed.

### Alt B — Stay on Go, build a custom SDL3 backend for cimgui-go

- ✅ Get SDL3 features in Go.
- ❌ ~600 LOC of C++ shim ported to Go, ongoing maintenance burden.
- ❌ Not justified by any concrete SDL3-only requirement of a BBC emulator.

### Alt C — Stay on Go, drop SDL entirely, use GLFW + miniaudio/oto

- ✅ `cimgui-go`'s GLFW backend is well-trodden.
- ✅ No SDL version question at all.
- ❌ Splits audio and windowing across two libraries.
- ❌ Loses SDL's mature gamepad / haptics / event-pump conveniences (relevant for BBC keyboard mapping, joystick).

### Alt D (chosen) — Stay on Go, use SDL2 + cimgui-go

- ✅ Zero friction with `cimgui-go`'s bundled `backend/sdlbackend`.
- ✅ Phase 001 CPU work preserved unchanged.
- ✅ Spec-kit / constitution / CI unchanged.
- ✅ SDL2 is mature, supported, and meets every requirement of a BBC Model B emulator.
- ❌ Not SDL3. (Acceptable: no SDL3-only feature is needed.)
- ❌ `cgo` overhead on every ImGui call. (Acceptable: ImGui calls are bounded by ~60 Hz frame rate; aggregate cost negligible.)
- ❌ Go GC jitter is theoretically possible in the audio callback. (Mitigable: preallocate ring buffer, `runtime.LockOSThread` the audio goroutine, profile with `GODEBUG=gctrace=1`.)

## Consequences

### Positive

- **Zero rework on Phase 001.** The validated `mos6502/` package is the canonical 6502 implementation for the life of the project.
- **Spec-kit framework unchanged.** Constitution Principle II (≥ 80 % delta line coverage, `golangci-lint`, `go test` determinism) continues to apply to all future phases without modification.
- **CI gates unchanged.** `make fmt vet lint test bench cover` already encodes the constitution; no new toolchain to introduce.
- **Standard library suffices.** Tests use only `testing`, `testing/quick`, `embed`, `bytes` — no external dependency burden until Phase 004 (SDL host).
- **Cross-platform reach for free.** Go cross-compiles to Linux/macOS/Windows on amd64/arm64 with no platform-specific code.
- **Hire-no-one risk.** Single-developer hobby project: stays in a language the developer already knows fluently.

### Negative / accepted limitations

- **SDL2, not SDL3.** Accepted because the BBC Model B emulator needs no SDL3-only feature. Re-evaluate if cimgui-go ships an SDL3 backend (likely within 12 months given upstream pressure), at which point the migration is a contained Phase 004 sub-task.
- **`cgo` is required.** `CGO_ENABLED=1` for any build that links cimgui-go and SDL2. Cross-compilation needs a C toolchain for the target. Documented in the Phase 004 plan; not a blocker.
- **GC jitter is theoretically possible.** Mitigation pattern documented up-front: preallocated audio ring buffer + `runtime.LockOSThread` on the audio goroutine + zero per-frame allocations on the emulator hot path (already proven achievable — `mos6502` hits 0 B/op). Worst-case escape valve: `GOGC=off` during the run loop with a manual `runtime.GC()` between frames.
- **No `comptime` opcode dispatch.** The 256-entry function-pointer table stays. Already shown to be O(1) and branch-predictable; current 4.3 ns/cycle leaves a 30× margin against the SC-006 target.
- **No first-class packed structs.** The P register is hand-coded bit operations in `status.go`. Already implemented and tested; not a recurring cost.

### Neutral

- **Decision is reversible** at modest cost while the project is still single-package. The longer the host shell, video, sound, and machine layers grow in Go, the higher the cost of reversing. After Phase 004 (SDL host integrated), reversal cost rises sharply; re-evaluate before Phase 004 begins, not after.

## Verification

Decision is validated if all of the following remain true 90 days after Phase 004 (SDL host) lands:

- Audio callback under `GODEBUG=gctrace=1` shows no GC pause exceeding half the audio buffer period (≤ 5 ms for a 1024-sample buffer at 48 kHz).
- ImGui debugger overlay renders at ≥ 60 fps with the emulator running a representative BBC workload.
- CPU throughput on the integrated emulator stays within 10 % of the standalone `mos6502` benchmark (`≤ 138 ns/emulated cycle` allowing for host-loop overhead).
- No SDL3-only feature has become a hard requirement.

If any condition fails, open ADR-0002 to re-evaluate.

## References

- Plan document this ADR was extracted from: `~/.claude/plans/would-it-be-better-glowing-glade.md`
- `cimgui-go` backend list: https://github.com/AllenDang/cimgui-go
- Dear ImGui SDL3 example (upstream, not yet in cimgui-go): https://github.com/ocornut/imgui/blob/master/examples/example_sdl3_opengl3/main.cpp
- Phase 001 spec: `specs/001-cpu-6502-core/spec.md`
- Phase 001 plan: `specs/001-cpu-6502-core/plan.md`
