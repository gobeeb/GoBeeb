# Implementation Plan: 6502 CPU Core

**Branch**: `001-cpu-6502-core` | **Date**: 2026-05-25 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/001-cpu-6502-core/spec.md`

## Summary

Build a sub-cycle / bus-cycle accurate NMOS 6502 CPU emulator as a self-contained, reusable Go package (`github.com/gobeeb/GoBeeb/mos6502`) for use by the wider GoBeeb BBC Model B emulator. The package exposes a `CPU` type, a small `Memory` interface that the host supplies, an optional illegal-opcode notification hook, and a control surface for RESET/IRQ/NMI/RDY signalling plus single-cycle and single-instruction stepping. All 151 documented opcodes across all 13 addressing modes, faithful NMOS quirks (indirect-JMP page-bug, RMW double-write, NMI hijack of BRK/IRQ, RDY-stalls-reads-only, NMOS-style BCD flag derivation), and a recording bus-trace harness for golden tests. Validated by Klaus Dormann's `6502_functional_test` and `6502_decimal_test` plus per-opcode bus-trace golden files.

## Technical Context

**Language/Version**: Go 1.22+ (developed against the system toolchain at `go1.26.2`; module `go` directive pinned to `1.22` for broad consumer compatibility).

**Primary Dependencies**: Go standard library only for the CPU core. Tests use only `testing`, `testing/quick`, `embed` (for the Klaus Dormann ROMs), and `bytes`. No external runtime dependencies — this is a non-negotiable design constraint for a reusable emulator core.

**Storage**: N/A — the CPU holds only register/flag state in memory. Test ROMs are embedded via `//go:embed`.

**Testing**: `go test` with table-driven unit tests, golden-file bus traces for every opcode/addressing-mode combination, two functional-test ROM runners (Klaus Dormann `6502_functional_test`, `6502_decimal_test`), benchmarks under `testing.B` for the SC-006 throughput target, and `testing/quick` property tests for arithmetic identities (ADC/SBC round-trip, rotate identity, etc.).

**Target Platform**: Pure Go, cross-platform. Tier-1: Linux x86-64 (Arch, per the developer environment). Tier-2: macOS arm64, Windows x86-64, Linux arm64. No platform-specific code.

**Project Type**: Library — a single importable Go package. No CLI, no service, no UI. The GoBeeb top-level emulator (a later phase) will import this package; standalone test harnesses (Klaus Dormann runner, golden-trace runner) live inside `mos6502/` as `_test.go` files.

**Performance Goals**: SC-006 — ≥ 4 × real BBC speed (≥ 8 MHz effective 6502 throughput) on a contemporary developer laptop. Concretely: average ≤ 125 ns per emulated CPU cycle on amd64 Linux at `GOAMD64=v3`. Hot path must be zero-allocation per emulated cycle (verified by `-benchmem` showing `0 B/op 0 allocs/op` on `BenchmarkRunNoop`, `BenchmarkRunMixedWorkload`).

**Constraints**: Sub-cycle / bus-cycle accuracy (FR-018) is the hard correctness floor — every memory access lands on the cycle a real NMOS 6502 issues it. Deterministic across runs and platforms (SC-007). Doc comments on every exported identifier (FR-020, constitution Principle I). ≥ 80 % delta line coverage on the new package (constitution Principle II).

**Scale/Scope**: Estimated 3 000–5 000 LOC for production code (151 opcode entries + 13 addressing-mode helpers + interrupt + RMW + BCD + disassembler), 5 000–8 000 LOC of tests. Single Go package. No goroutine concurrency (Assumption A8).

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

The GoBeeb constitution defines four Core Principles. Each is addressed below.

### I. Code Quality — PASS

- `gofmt` is the project formatter; `go vet` and `golangci-lint run` (config to land alongside this feature) are the linter gate. Both will be wired into CI.
- Cyclomatic complexity floor (≤ 10) is preserved by splitting the opcode set across mnemonic-grouped files (`arith.go`, `logical.go`, `branch.go`, `transfer.go`, `stack.go`, `flags.go`, `rmw.go`, `interrupts.go`) and using one function per addressing-mode/opcode behaviour.
- FR-020 already mandates doc comments on every exported identifier; that satisfies the constitution's public-API doc-comment rule.
- No dead code, no commented-out blocks. Any TODO has an issue link.

### II. Testing Standards (NON-NEGOTIABLE) — PASS

- Unit tests per opcode (happy path + at least one failure-mode / flag-edge case).
- Klaus Dormann ROM tests act as integration tests for the published 6502 contract.
- Delta line coverage ≥ 80 % is enforced via `go test -coverprofile=cover.out -covermode=atomic` and a CI threshold check.
- All tests are deterministic — the CPU itself is deterministic per SC-007, and tests use no `time.Now`, no goroutines, no real I/O.
- Test pyramid shape: very many fast unit tests (per opcode × per mode), a moderate set of integration tests (Klaus Dormann + golden bus traces), a single end-to-end benchmark.

### III. User Experience Consistency — EXEMPT (no user surface)

This is a Go library. It has no UI, no design-token consumer, no copy, no accessibility surface. The principle's gate is met by the absence of in-scope user-facing surfaces.

The package's *developer* UX (API ergonomics) is treated as the relevant analogue: the API surface (`CPU`, `Memory`, `IllegalOpcodeHook`) follows Go idioms — small interfaces, value-receiver methods where safe, errors only at construction, no global state. Quickstart (`quickstart.md`) is the equivalent of a "first-run experience" and is part of the deliverable.

### IV. Performance Requirements — PASS

- SC-006 declares the throughput budget *before* implementation: ≥ 8 MHz effective 6502 throughput; ≤ 125 ns per emulated CPU cycle on Linux amd64.
- A `bench_test.go` will measure `ns/op` for `BenchmarkRunNoop` (tight `NOP` loop) and `BenchmarkRunMixedWorkload` (representative BBC OS-style trace). Both must meet the budget; regression > 5 % blocks merge.
- Allocation budget: zero allocations per emulated CPU cycle on the hot path. Enforced via `-benchmem` and asserted by a benchmark wrapper that calls `testing.B.ReportAllocs()`.
- Worst-case complexity: instruction dispatch is O(1) (151-entry function-pointer table); per-cycle state machine is O(1); no algorithm in this feature processes > 1 000 items per call.

**Result**: No constitution violations. `## Complexity Tracking` section below is empty.

## Project Structure

### Documentation (this feature)

```text
specs/001-cpu-6502-core/
├── plan.md                       # This file
├── spec.md                       # Feature specification (already complete)
├── research.md                   # Phase 0 output (created by this command)
├── data-model.md                 # Phase 1 output (created by this command)
├── quickstart.md                 # Phase 1 output (created by this command)
├── contracts/                    # Phase 1 output (created by this command)
│   ├── cpu.go                    # CPU control + state API (Go interface form)
│   ├── memory.go                 # Memory interface contract
│   └── illegal.go                # IllegalOpcodeHook interface contract
├── checklists/
│   └── requirements.md           # Validation checklist (from /speckit-specify)
└── tasks.md                      # Phase 2 output (created by /speckit-tasks, not here)
```

### Source Code (repository root)

```text
github.com/gobeeb/GoBeeb (module root)
├── go.mod                        # Module declaration, Go 1.22+, no runtime deps
├── go.sum                        # Empty until a first test-time dep is added (none expected)
├── LICENSE
├── CLAUDE.md
├── mos6502/                      # the CPU core (this feature)
│   ├── doc.go                    # Package-level doc comment + overview
│   ├── cpu.go                    # CPU struct, New(), Reset/IRQ/NMI/RDY control, Step/StepCycle/Run
│   ├── status.go                 # Processor status (P) flag bit operations
│   ├── memory.go                 # Memory interface declaration
│   ├── addressing.go             # 13 addressing-mode effective-address helpers
│   ├── opcodes.go                # 256-entry dispatch table (151 official + illegal slots)
│   ├── instructions.go           # Shared instruction primitives (load/store/transfer/compare/branch)
│   ├── arith.go                  # ADC, SBC — binary and NMOS-faithful BCD
│   ├── rmw.go                    # ASL/LSR/ROL/ROR/INC/DEC with double-write semantics
│   ├── interrupts.go             # RESET/IRQ/NMI/BRK, NMI-hijack, RDY-aware vector fetch
│   ├── illegal.go                # Illegal-opcode hook plumbing, single-byte 2-cycle NOP
│   ├── disasm.go                 # Lightweight per-instruction disassembler (debug observability)
│   ├── trace.go                  # Optional bus-trace recorder used by golden tests
│   ├── cpu_test.go               # CPU construction, Step/StepCycle/Run semantics, Registers/SetRegisters round-trip, determinism re-run (RESET-state tests live in interrupts_test.go)
│   ├── addressing_test.go        # Per-mode effective-address tests incl. wrap + page-cross
│   ├── opcodes_test.go           # One test per opcode (happy path + flag effects + cycle count)
│   ├── arith_test.go             # ADC/SBC binary + BCD edge cases
│   ├── rmw_test.go               # RMW double-write trace verification
│   ├── interrupts_test.go        # RESET vector, IRQ, NMI edge, BRK, NMI-hijack-of-BRK/IRQ
│   ├── illegal_test.go           # NOP behaviour + hook invocation
│   ├── rdy_test.go               # RDY stalls reads only; writes proceed; per-cycle re-evaluation
│   ├── trace_test.go             # Bus-trace recorder fidelity
│   ├── disasm_test.go            # Round-trip: opcode bytes → mnemonic+operand string
│   ├── functional_test.go        # Klaus Dormann 6502_functional_test runner (long)
│   ├── decimal_test.go           # Klaus Dormann 6502_decimal_test runner (long)
│   ├── bench_test.go             # SC-006 throughput + zero-alloc benchmarks
│   └── testdata/
│       ├── 6502_functional_test.bin
│       ├── 6502_decimal_test.bin
│       └── golden_traces/
│           ├── lda_imm.trace
│           ├── lda_abs_x_pagecross.trace
│           ├── inc_abs.trace               # RMW double-write
│           ├── jmp_indirect_pagebug.trace
│           ├── brk_normal.trace
│           ├── brk_nmi_hijack.trace
│           └── ...                         # one per opcode × interesting mode
└── (future phases — out of scope for this feature)
    bbc/        # the wider BBC machine
    video/      # video ULA
    sound/      # SN76489
    ...
```

**Structure Decision**: Single Go module rooted at `github.com/gobeeb/GoBeeb`. The CPU core is a single Go package `mos6502` placed at the module root (not under `pkg/` or `internal/`) so external consumers — including, eventually, the GoBeeb top-level emulator and any third party who wants a faithful NMOS 6502 — can `import "github.com/gobeeb/GoBeeb/mos6502"` with the most natural path. Test ROMs live in `mos6502/testdata/` and are embedded with `//go:embed` so a `go get` consumer needs nothing on disk to run the full test suite.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation                                                                                                                                                | Why Needed                                                                                                                                                                                                                                                                              | Simpler Alternative Rejected Because                                                                                                                                                                                                                                                                       |
|----------------------------------------------------------------------------------------------------------------------------------------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Workflow rule "Feature branches MUST be short-lived (≤ 5 days)" vs single-developer estimate of 6–8 working days for the full feature on branch `001-cpu-6502-core`. | A faithful sub-cycle-accurate 6502 with all 151 opcodes + Klaus Dormann conformance + NMI-hijack + RDY + BCD is irreducibly large; it cannot be split into shippable sub-features without leaving the CPU in a useless intermediate state (e.g. half the opcodes implemented does not run any real BBC program). | **Per-story PRs (preferred fallback)**: cut a PR at each Phase-3 / 4 / 5 / 6 checkpoint. Each sub-PR is ≤ 3 days and ships a green-tests increment (US1 = Klaus Dormann pass; US2 = step API + benchmark; etc.). This is the *operational* approach this plan will follow. The 6–8 day figure refers to the total feature, not any single branch. |

## Post-Design Constitution Re-check

*Performed after Phase 1 design artifacts (research.md, data-model.md, contracts/, quickstart.md) were drafted.*

- **Code Quality**: Phase 1 design splits the package into ten small files of one responsibility each. Public API surface (Phase 1 `contracts/`) is six exported identifiers (`CPU`, `New`, `Memory`, `IllegalOpcodeHook`, `BusEvent`, `Trace`). No principle violation introduced.
- **Testing Standards**: Phase 1 added `golden_traces/` and the Klaus Dormann runners; coverage path is intact, all tests remain deterministic.
- **UX Consistency**: still exempt — Phase 1 introduces no user surface.
- **Performance**: Phase 1 design selected a function-pointer dispatch table (O(1), branch-predictable, no map lookup). Confirms ≤ 125 ns/cycle budget remains achievable.

**Result**: Constitution Check re-confirmed PASS after Phase 1. No new entries in `## Complexity Tracking`.
