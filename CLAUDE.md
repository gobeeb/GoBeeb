<!-- SPECKIT START -->
For additional context about technologies to be used, project structure,
shell commands, and other important information, read the current plan:

- Current feature: CPU Bus-Cycle Validation — Tom Harte ProcessorTests (branch `003-cpu-processor-tests`)
- Plan: [specs/003-cpu-processor-tests/plan.md](specs/003-cpu-processor-tests/plan.md)
- Spec: [specs/003-cpu-processor-tests/spec.md](specs/003-cpu-processor-tests/spec.md)
- Research / decisions: [specs/003-cpu-processor-tests/research.md](specs/003-cpu-processor-tests/research.md)
- Data model: [specs/003-cpu-processor-tests/data-model.md](specs/003-cpu-processor-tests/data-model.md)
- Contracts: [specs/003-cpu-processor-tests/contracts/](specs/003-cpu-processor-tests/contracts/)
- Quickstart: [specs/003-cpu-processor-tests/quickstart.md](specs/003-cpu-processor-tests/quickstart.md)

Language: Go 1.22+. Module: `github.com/gobeeb/GoBeeb`. Target package: `mos6502/` (test-only additions; no new public API). Phase 002 (`bbc/`) artifacts at [specs/002-bbc-machine/](specs/002-bbc-machine/) remain canonical for that layer.
<!-- SPECKIT END -->

## Implemented packages

### `mos6502/` — NMOS 6502 CPU core (sub-cycle accurate)

Public API:

- `mos6502.New(mem Memory) *CPU` — construct a CPU bound to host memory.
- `CPU.Step() uint64` / `CPU.StepCycle() uint64` / `CPU.Run(budget uint64) uint64` — drive execution.
- `CPU.AssertReset()` / `CPU.AssertIRQ(level bool)` / `CPU.AssertNMI()` / `CPU.DeassertNMI()` / `CPU.SetRDY(ready bool)` — control surface.
- `CPU.Registers()` / `CPU.SetRegisters(Registers)` — state snapshot/restore.
- `CPU.SetTrace(*Trace)` / `CPU.SetIllegalOpcodeHook(IllegalOpcodeHook)` — observability hooks.
- `Memory` interface (infallible, host-supplied).
- `Disassemble(mem Memory, pc uint16) (string, int)` — debug helper.

Validation status:
- Klaus Dormann `6502_functional_test` ROM: **PASS** (exercises every documented opcode + BCD).
- Tom Harte SingleStepTests/ProcessorTests corpus (Phase 003): **PASS** for all 151 documented NMOS opcodes (10,000 cases each ≈ 1.51M assertions); 105 undocumented opcodes reported as SKIP via the `skipList` contract. Pinned upstream SHA `bb11756436da8fd16cce86aef63dc6725f48836f`. `-short` mode samples 100 cases per opcode (~15,100 total) for the CI inner loop; full corpus run completes in ~4 s on amd64 (SC-008 budget ≤ 5 min). Corpus is gitignored; fetched on demand via `go generate ./mos6502/`.
- 91 unit tests covering addressing modes, RMW double-write trace, BCD edges, stack wrap, interrupts (RESET/IRQ/NMI/BRK/RTI), NMI hijack, illegal opcode hook, RDY gating, golden bus traces.
- Coverage: 99.3% on `mos6502/` (gate: ≥80%).
- Benchmarks: ~4.3 ns/cycle on amd64, 0 B/op, 0 allocs/op (gate: ≤125 ns/cycle).

Known v1 limitations:
- RDY gated at instruction boundary, not per bus cycle.
- `Run(budget)` may overshoot the budget by up to one instruction (never splits mid-instruction).
- Decimal-mode validation via unit tests + functional ROM's internal BCD suite (separate `6502_decimal_test.bin` ROM not pre-built upstream).
- `StepCycle()` runs one whole instruction (true per-cycle scheduling deferred).

Make targets: `make fmt vet lint test bench cover`.

### `bbc/` — BBC Model B machine layer

Public API:

- `bbc.New() *Machine` — construct a Machine with no ROMs loaded.
- `Machine.LoadOSROM([]byte) error` / `Machine.LoadSidewaysROM(bank int, []byte) error` — copy-on-load ROM installers.
- `Machine.Tick(cycles uint64) uint64` — drive the CPU forward, returns cumulative cycle count.
- `Machine.Reset() error` / `Machine.ColdReset() error` — soft (BREAK-key) vs power-on resets.
- `Machine.AssertIRQ(level bool)` / `AssertNMI()` / `DeassertNMI()` / `SetRDY(ready bool)` — control pass-throughs.
- `Machine.CPU() *mos6502.CPU` — debug access (Trace, Registers, Disassemble, IllegalOpcodeHook).
- `Machine.SetUnmappedAccessHook(UnmappedAccessHook)` — observability for FRED/JIM/SHEILA + empty-bank reads.
- `Machine.Snapshot() Snapshot` / `Machine.Restore(Snapshot) error` — in-process round-trip of CPU + RAM + peripherals.
- `Peripheral` interface, `MemoryMap` (implements `mos6502.Memory`), per-stub snapshot types.
- Errors: `ErrNoOSROM`, `ErrInvalidROMSize`, `ErrBankOutOfRange`, `ErrRestoreMismatch`.

Validation status:
- 78 unit tests covering OS-ROM/sideways loaders, Reset vs ColdReset, RAM round-trip, SHEILA decoder routing for every address range in FR-008, CRTC index-then-data semantics, System/User VIA round-trip + 16-byte mirroring, FRED/JIM unmapped behaviour, sideways paging across 4 banks with empty-bank open-bus, Snapshot/Restore byte-identical round-trip after 100k+ cycles, UnmappedAccessHook semantics, FR-028 no-locks reflection guard.
- Golden bus traces: `reset_first256.trace`, `crtc_index_then_data.trace`, `via_register_round_trip.trace`, `rom_select_swap.trace`.
- OS 1.20 smoke test: gated on `BBC_OS_ROM` env var (not redistributed); when set, asserts ≥ 1 000 000 cycles run without firing the illegal-opcode or unmapped-access hooks.
- Coverage: 97.2% on `bbc/` (gate: ≥ 80%).
- Benchmarks: `BenchmarkTickNoop` ~5.4 ns/cycle, `BenchmarkTickMixedWorkload` ~5.3 ns/cycle on amd64; 0 B/op, 0 allocs/op (gate: ≤ ~6.5 ns/cycle).

Known v1 limitations:
- Stub peripherals implement register-file storage only — no CRTC scanline timing, no VIA timers/shift registers/interrupt flags, no real ACIA/FDC/ADC/Tube/Econet behaviour. Real semantics are owned by later phases.
- ACCCON ($FE34–$FE37) is a Model B no-op (always returns $FF on read); Master 128 semantics are deferred.
- No save-state file format — `Snapshot`/`Restore` round-trip is in-process Go values only.
- No interrupts sourced from peripherals (FR-021). The IRQ/NMI control surface is wired but never pulled by Phase 002 code.
- Single-goroutine contract carries forward from `mos6502.CPU` — no internal locking on `Tick`.
