# Implementation Plan: BBC Machine Layer

**Branch**: `002-bbc-machine` | **Date**: 2026-05-25 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/002-bbc-machine/spec.md`

## Summary

Wire the existing `mos6502.CPU` (Phase 001, validated, ≥ 99 % covered) into a BBC Model B-shaped address space so the wider GoBeeb emulator and its tests can boot a real OS 1.20 ROM, exercise sideways ROM paging via the `$FE30` latch, and observe every SHEILA I/O access through stub peripherals — all without any UI dependency yet. The deliverable is a single Go package `bbc/` exposing one user-visible type (`Machine`) with a small control surface (`Tick`, `Reset`, `ColdReset`, IRQ/NMI/RDY pass-through, ROM loaders, snapshot/restore, unmapped-access hook). Peripheral stubs (CRTC, ACIA, serial ULA, video ULA, ROM-select, System VIA, User VIA, FDC, ADC, Tube, Econet) implement register-file storage only; real behaviour is deferred to later phases. Single-goroutine contract carries forward from Phase 001 (Clarification Q1). Zero-allocation hot path and ≥ 80 % delta coverage carry forward from the constitution.

## Technical Context

**Language/Version**: Go 1.22+ (module already pinned to `1.22`; developed against the system toolchain at `go1.26.2`).

**Primary Dependencies**: Go standard library only. The package consumes the Phase 001 `github.com/gobeeb/GoBeeb/mos6502` package verbatim — no modifications to that package are in scope. Tests use only `testing`, `testing/quick`, `embed`, `bytes`, `errors`, `fmt`.

**Storage**: N/A — no on-disk state managed by this package. ROM images are supplied by the caller as `[]byte`; tests embed small hand-crafted stub ROMs via `//go:embed` and optionally read a path to a user-supplied OS 1.20 binary via environment variable (`BBC_OS_ROM`, see `quickstart.md`). No save-state file format is locked in this phase (Snapshot/Restore is in-process; see Research §3).

**Testing**: `go test` with table-driven unit tests, golden bus-trace tests built on the Phase 001 `mos6502.Trace` recorder, an OS-ROM smoke test that runs the machine for ≥ 1 000 000 cycles asserting the unmapped-access hook never fires (skipped when `BBC_OS_ROM` is unset), benchmarks under `testing.B` for SC-006 throughput and zero-allocation gates.

**Target Platform**: Pure Go, cross-platform. Tier-1: Linux x86-64. Tier-2: macOS arm64, Windows x86-64, Linux arm64. No platform-specific code.

**Project Type**: Library — a single importable Go package `github.com/gobeeb/GoBeeb/bbc`. No CLI, no service, no UI. Consumers are (a) the future Phase 003 video package, (b) the future Phase 004 SDL host, and (c) developers writing integration tests.

**Performance Goals**:
- SC-006 — `Tick` hot path: ≤ ~6.5 ns per emulated CPU cycle on Linux amd64 (within 50 % of the Phase 001 standalone `mos6502` benchmark of ~4.3 ns/cycle, accounting for the BBC memory-map decoder dispatch overhead).
- Zero allocations per emulated cycle (`0 B/op 0 allocs/op` reported by `go test -bench -benchmem`).
- Unmapped-access hook adds at most a single nil-check to the unmapped cold path; mapped reads/writes (the overwhelming majority) take zero hook overhead.

**Constraints**:
- Single-goroutine: `Machine` is not goroutine-safe; the implementation MUST NOT take internal locks on the hot path (Clarification Q1, FR-028).
- Deterministic: given the same loaded ROMs and the same sequence of `Tick` / control-surface calls, the machine MUST produce identical bus traces and identical `Snapshot()` byte output across runs and platforms (SC-005).
- Zero hidden state: every subsystem MUST round-trip through `Snapshot`/`Restore` (constitution cross-cutting concern from the roadmap).

**Scale/Scope**: Estimated ~1 500–2 500 LOC of production code (memory map + decoder + 12 peripheral stubs + machine façade + snapshot plumbing), ~2 500–4 000 LOC of tests. Single Go package. No goroutine concurrency.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### I. Code Quality — PASS

- `gofmt`, `go vet`, `golangci-lint run` (already wired in Phase 001 CI) gate every PR.
- Cyclomatic complexity floor (≤ 10) is preserved by splitting peripherals across one file per device (`crtc.go`, `via_system.go`, `via_user.go`, `acia.go`, `serial_ula.go`, `video_ula.go`, `rom_select.go`, `fdc.go`, `adc.go`, `tube.go`, `econet.go`) and keeping the SHEILA decoder a single `switch` keyed on the high nibble + low nibble of the address.
- FR-027 already mandates doc comments on every exported identifier; that satisfies the constitution's public-API doc-comment rule.
- No dead code; the package starts from an empty directory and grows by atomic commits per task.

### II. Testing Standards (NON-NEGOTIABLE) — PASS

- Unit tests per peripheral stub (write → read-back assertion; write-only and read-only register cases where declared).
- Memory-map tests per region (RAM, OS ROM, sideways ROM, FRED, JIM, SHEILA, vector region).
- Sideways ROM banking integration test (User Story 3).
- Snapshot/Restore round-trip test (User Story 4): run N cycles, snapshot, run M more, capture state; replay snapshot, run M, compare byte-for-byte.
- OS-ROM smoke test (User Story 1) gated on `BBC_OS_ROM` environment variable; CI sets this to a known-OK 16 KB image (provided out-of-band, not redistributed).
- Golden bus traces for representative MOS-init instruction streams using the Phase 001 `mos6502.Trace` recorder.
- Delta line coverage ≥ 80 % enforced via `make cover`.
- All tests deterministic — no `time.Now`, no goroutines, no real I/O.

### III. User Experience Consistency — EXEMPT (no user surface)

This is a Go library. No UI, no design-token consumer, no accessibility surface. The package's developer UX (API ergonomics) follows Go idioms — small interfaces, value-receiver methods where safe, errors only at construction / loader entry points, no global state. The quickstart serves as the analogue "first-run experience".

### IV. Performance Requirements — PASS

- SC-006 declares the throughput budget before implementation: ≤ ~6.5 ns/cycle on Linux amd64 in `Tick`, zero allocations per cycle.
- A `bench_test.go` will measure `ns/op` for `BenchmarkTickNoop` (machine running NOPs from main RAM) and `BenchmarkTickMixedWorkload` (a synthetic ROM exercising RAM, OS ROM, sideways ROM, and a couple of SHEILA writes per iteration). Both must meet the budget; regression > 5 % blocks merge.
- Allocation budget: zero allocations per emulated cycle on the hot path. Asserted by `testing.B.ReportAllocs()` in the benchmark wrapper.
- Worst-case complexity: memory-map dispatch is O(1) (high-byte switch); SHEILA decoder dispatch is O(1) (range table); peripheral register read/write is O(1) (array index). No algorithm in this phase processes > 1 000 items per call.

**Result**: No constitution violations. `## Complexity Tracking` section below is empty.

## Project Structure

### Documentation (this feature)

```text
specs/002-bbc-machine/
├── plan.md                       # This file
├── spec.md                       # Feature specification (complete)
├── research.md                   # Phase 0 output (created by this command)
├── data-model.md                 # Phase 1 output (created by this command)
├── quickstart.md                 # Phase 1 output (created by this command)
├── contracts/                    # Phase 1 output (created by this command)
│   ├── machine.go                # Machine type + control/ROM/snapshot surface
│   ├── memory_map.go             # bbc.MemoryMap implementing mos6502.Memory
│   ├── peripheral.go             # Peripheral interface contract for SHEILA stubs
│   ├── snapshot.go               # Snapshot value type contract
│   └── unmapped_hook.go          # UnmappedAccessHook function-type contract
├── checklists/
│   └── requirements.md           # Validation checklist (from /speckit-specify)
└── tasks.md                      # Phase 2 output (created by /speckit-tasks, not here)
```

### Source Code (repository root)

```text
github.com/gobeeb/GoBeeb (module root)
├── go.mod                        # Unchanged (Go 1.22+, no runtime deps)
├── go.sum                        # Unchanged (empty)
├── LICENSE
├── CLAUDE.md                     # SPECKIT block updated to point at this plan
├── mos6502/                      # Phase 001 — UNCHANGED IN THIS PHASE
│   └── …
└── bbc/                          # Phase 002 — this feature
    ├── doc.go                    # Package-level doc comment + overview
    ├── machine.go                # Machine struct, New(), Tick, Reset/ColdReset, control pass-throughs
    ├── memory_map.go             # MemoryMap implementing mos6502.Memory; dispatches I/O to peripherals
    ├── rom.go                    # OS ROM + sideways ROM loaders (copy-on-load)
    ├── snapshot.go               # Snapshot type, Snapshot()/Restore(), per-stub round-trip plumbing
    ├── unmapped.go               # UnmappedAccessHook function type + invocation helper
    ├── peripheral.go             # Peripheral interface + helpers (RegisterFile base struct)
    ├── crtc.go                   # 6845 CRTC stub (register-index + indexed-data semantics — FR-010)
    ├── acia.go                   # 6850 ACIA stub
    ├── serial_ula.go             # Serial ULA stub
    ├── video_ula.go              # Video ULA stub
    ├── rom_select.go             # $FE30–$FE33 ROM-select latch (drives MemoryMap's bank index)
    ├── via.go                    # Shared 16-register VIA stub type; System + User VIA both instantiate it
    ├── errors.go                 # Exported error vars (ErrNoOSROM, ErrInvalidROMSize, ErrBankOutOfRange, ErrRestoreMismatch)
    ├── fdc.go                    # 1770 FDC stub (returns $FF; Phase 007 fills behaviour)
    ├── adc.go                    # ADC stub
    ├── tube.go                   # Tube stub
    ├── econet.go                 # Econet stub
    ├── acccon.go                 # ACCCON / paged-ROM-ID stub ($FE34; returns $FF on Model B)
    ├── machine_test.go           # Construction, Tick budget semantics, Reset vs ColdReset, control pass-throughs
    ├── memory_map_test.go        # Per-region read/write tests (RAM, OS ROM, sideways window, FRED, JIM, vector region)
    ├── rom_test.go               # OS ROM + sideways ROM loader: 16 KB size enforcement, copy-on-load isolation
    ├── snapshot_test.go          # Snapshot/Restore round-trip (User Story 4 acceptance scenarios)
    ├── unmapped_test.go          # UnmappedAccessHook fires on every unmapped slot, never on mapped slots
    ├── sheila_test.go            # SHEILA decoder routing test — write/read each address range, assert correct peripheral + offset
    ├── crtc_test.go              # CRTC index-then-data semantics (FR-010)
    ├── via_test.go               # System VIA + User VIA register-file round-trip; mirror-every-16-bytes test
    ├── sideways_test.go          # User Story 3 acceptance scenarios (paging across banks, empty-bank reads, write-drop)
    ├── boot_os_test.go           # OS-ROM smoke test, gated on $BBC_OS_ROM (skipped if env var unset)
    ├── golden_trace_test.go      # Golden bus traces for representative MOS-init instruction streams
    ├── bench_test.go             # SC-006 throughput + zero-alloc benchmarks
    └── testdata/
        ├── stub_os_16k.bin       # Hand-crafted 16 KB ROM that exercises the boot path (no copyright)
        ├── stub_sideways_aa.bin  # 16 KB of $AA — for bank 0 in sideways_test
        ├── stub_sideways_55.bin  # 16 KB of $55 — for bank 1 in sideways_test
        └── golden_traces/
            ├── reset_first256.trace        # First 256 bus cycles after RESET against stub OS ROM (SC-002)
            ├── rom_select_swap.trace       # CPU writes to $FE30 + subsequent $8000 reads
            ├── via_register_round_trip.trace
            └── crtc_index_then_data.trace
```

**Structure Decision**: Single Go module unchanged from Phase 001. The BBC machine layer is a single sibling package `bbc/` at the module root (not under `pkg/` or `internal/`) so external consumers — the future host shell, third-party debuggers, or anyone wanting a Model B memory map around the validated `mos6502` package — can `import "github.com/gobeeb/GoBeeb/bbc"` with the most natural path. Test data lives in `bbc/testdata/` and is embedded with `//go:embed` so the unit test suite needs nothing on disk to run (the OS-ROM smoke test is the only test that consults the filesystem, and it skips when no path is supplied).

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

No violations. (The "≤ 5 days" branch lifetime constraint is comfortably met — the phase scope is bounded by a fixed peripheral count, and the implementation is per-peripheral plus a memory map.)

## Post-Design Constitution Re-check

*Performed after Phase 1 design artifacts (research.md, data-model.md, contracts/, quickstart.md) were drafted.*

- **Code Quality**: Phase 1 design splits the package across one file per peripheral (with `via.go` shared by System + User VIA via a common type) plus five core files (machine, memory_map, rom, snapshot, errors). Public API surface (`contracts/`) is seven exported identifiers (`Machine`, `New`, `MemoryMap`, `Peripheral`, `Snapshot`, `UnmappedAccessHook`, `ErrInvalidROMSize`). No principle violation introduced.
- **Testing Standards**: Phase 1 added `golden_traces/` and the OS-ROM smoke gate; coverage path is intact, all tests remain deterministic. The OS-ROM smoke test is gated on an environment variable to honour the no-redistribution policy without weakening coverage of the other tests.
- **UX Consistency**: still exempt — Phase 1 introduces no user surface.
- **Performance**: Phase 1 design selected a switch-on-high-byte memory-map dispatcher (O(1), branch-predictable, no map lookup) and a per-peripheral array-indexed register file (O(1)). Confirms the ≤ ~6.5 ns/cycle budget remains achievable.

**Result**: Constitution Check re-confirmed PASS after Phase 1. No new entries in `## Complexity Tracking`.
