---
description: "Task list for 001-cpu-6502-core implementation"
---

# Tasks: 6502 CPU Core

**Input**: Design documents from `/specs/001-cpu-6502-core/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md (all present)

**Tests**: Tests are REQUIRED for this feature (user requested unit tests; constitution Principle II is non-negotiable). All test tasks below are mandatory, not optional.

**Organization**: Tasks are grouped by user story (US1, US2, US3, US4 from spec.md) to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story?] Description`

- **[P]**: Task can run in parallel with other [P]-marked tasks in the same phase (different files, no in-phase dependencies)
- **[Story]**: Maps the task to its user story (US1, US2, US3, US4)
- Every task carries an exact file path

## Path Conventions

- Module root: `/home/richard/code/personal/GoBeeb/` (module `github.com/gobeeb/GoBeeb`)
- Production code: `mos6502/`
- Tests live alongside production code as `*_test.go` (Go convention)
- Embedded test ROMs: `mos6502/testdata/`
- Per-opcode golden bus traces: `mos6502/testdata/golden_traces/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Initialise the Go module, create the package directory, and wire formatter + linter + CI gates the constitution requires.

- [ ] T001 Initialise Go module at repo root: `go mod init github.com/gobeeb/GoBeeb` (creates `go.mod` pinned to `go 1.22`); add `go.sum` entry on first use.
- [ ] T002 [P] Create the package directory layout per plan.md: `mos6502/`, `mos6502/testdata/`, `mos6502/testdata/golden_traces/`.
- [ ] T003 [P] Add `.golangci.yml` at repo root configuring `gofmt`, `govet`, `staticcheck`, `revive`, `errcheck`, `gocyclo` (limit 10 per constitution), `goconst`, `misspell`.
- [ ] T004 [P] Add `Makefile` at repo root with targets `fmt`, `vet`, `lint`, `test`, `bench`, `cover` (writes `cover.out`).
- [ ] T005 [P] Add GitHub Actions workflow at `.github/workflows/ci.yml` running `make fmt vet lint test cover` on `ubuntu-latest` with Go 1.22 + 1.26; enforce `cover` ≥ 80 % on `mos6502/` delta lines.
- [ ] T006 [P] Add `mos6502/doc.go` with a one-paragraph package overview, link to the spec, and a usage example matching `quickstart.md`.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Types and primitives every user story depends on — the `Memory` interface, the `CPU` struct skeleton, the bus-cycle `tick()` primitives with `RDY` handling, the `Trace` recorder, and the illegal-opcode hook plumbing.

**⚠️ CRITICAL**: No user-story work begins until this phase is complete.

- [ ] T007 Implement `mos6502/memory.go`: declare the `Memory` interface with `Read(uint16) uint8` and `Write(uint16, uint8)` methods, plus the contract doc comment from `contracts/memory.go`. (FR-006)
- [ ] T008 [P] Implement `mos6502/status.go`: `FlagCarry`/`FlagZero`/`FlagInterrupt`/`FlagDecimal`/`FlagBreak`/`FlagUnused`/`FlagOverflow`/`FlagNegative` bit constants for the `P` register; helper methods `(*CPU).flag(mask) bool`, `(*CPU).setFlag(mask, on)`, `(*CPU).setNZ(v uint8)`. (FR-001, FR-004)
- [ ] T009 [P] Implement `mos6502/illegal.go`: declare `IllegalOpcodeHook` type, the shared `illegalNOP` dispatch-table slot handler (single-byte, 2-cycle NOP, invokes hook if registered). (FR-019)
- [ ] T010 [P] Implement `mos6502/trace.go`: `BusEventKind` (`BusRead`/`BusWrite`), `BusEvent{Cycle,Addr,Value,Kind}`, `Trace` ring buffer with `NewTrace(capacity int)`, `(*Trace).Snapshot() []BusEvent`. Pre-allocated, zero-alloc on append. (SC-008)
- [ ] T011 Implement `mos6502/cpu.go` SKELETON: `CPU` struct with all fields from `data-model.md` §1 (`A`, `X`, `Y`, `SP`, `PC`, `P`, `mem`, `irqLine`, `nmiPending`, `nmiPrev`, `resetPending`, `rdy`, `cycles`, `trace`, `onIllegalOp`, `addr`, `fetched`); constructor `New(mem Memory) *CPU` returning a CPU with `resetPending=true`, `rdy=true`; public control methods `AssertReset/AssertIRQ(level bool)/AssertNMI/DeassertNMI/SetRDY/SetIllegalOpcodeHook/SetTrace`; public state methods `Registers()`/`SetRegisters(Registers)`. **All implementations are stubs** — they only set/read fields. (FR-007, FR-008)
- [ ] T012 Implement the bus-cycle primitives in `mos6502/cpu.go`: unexported `(*CPU).read(addr) uint8`, `(*CPU).write(addr, value)`, `(*CPU).fetch() uint8` (reads at `PC` and post-increments). Each MUST: (a) on read, if `!c.rdy`, repeat the cycle without advancing state (FR-023); (b) call `c.mem.Read`/`c.mem.Write`; (c) increment `c.cycles`; (d) if `c.trace != nil`, append a `BusEvent`. Writes proceed even when `!c.rdy` (FR-023 NMOS limitation). These primitives are the foundation every opcode handler uses. (FR-006, FR-018, FR-023, SC-008)

**Checkpoint**: Foundation ready — user story implementation can now begin in parallel.

---

## Phase 3: User Story 1 — Execute a 6502 Program to a Correct Final State (Priority: P1) 🎯 MVP

**Goal**: A consumer can load a program into memory, set `PC`, step the CPU, and read out architecturally correct final state — registers, flags, memory, cycle count — matching what real NMOS 6502 silicon would produce.

**Independent Test**: Run Klaus Dormann's `6502_functional_test` and `6502_decimal_test` end-to-end; both reach the documented success trap with zero failed sub-tests. (SC-001)

### Tests for User Story 1 (write FIRST; assert they fail before implementing the matching code)

- [ ] T013 [P] [US1] Create `mos6502/addressing_test.go` with table-driven tests for every one of the 13 addressing modes: `ModeImmediate`, `ModeZeroPage`, `ModeZeroPageX` (incl. zero-page wrap, FR-014), `ModeZeroPageY`, `ModeRelative`, `ModeAbsolute`, `ModeAbsoluteX` (incl. page-cross dummy-read on bus trace, FR-018), `ModeAbsoluteY` (page-cross), `ModeIndirect` (incl. NMOS page-bug `JMP ($xxFF)`, FR-013), `ModeIndexedIndirect` (zero-page wrap on pointer fetch), `ModeIndirectIndexed` (page-cross on `(IND),Y`), `ModeImplicit`, `ModeAccumulator`. (FR-003, FR-014)
- [ ] T014 [P] [US1] Create `mos6502/opcodes_test.go` with one sub-test per documented NMOS opcode (151 cases). Each asserts: registers/memory/SP effect, every documented flag change, total cycle count incl. conditional extras. (FR-002, FR-004, FR-005)
- [ ] T014a [P] [US1] Add `mos6502/stack_test.go` asserting FR-015 stack-wrap invariants: with `SP=$00`, executing `PHA` writes to `$0100` and leaves `SP=$FF`; with `SP=$FF`, executing `PLA` reads from `$0100` and leaves `SP=$00`. Also covers `JSR`/`RTS` and `BRK`/`RTI` across the wrap boundary. (FR-015)
- [ ] T015 [P] [US1] Create `mos6502/arith_test.go` covering: binary `ADC`/`SBC` with every carry/overflow corner; BCD `ADC`/`SBC` with NMOS-faithful `N`/`V`/`Z` derived from the binary intermediate (FR-016); Klaus Dormann decimal-test edge cases ported into table form for fast unit runs. (FR-016, SC-002)
- [ ] T016 [P] [US1] Create `mos6502/rmw_test.go` verifying the read-modify-write **three-cycle** bus pattern for `ASL`/`LSR`/`ROL`/`ROR`/`INC`/`DEC` against memory operands — read, dummy-write-of-original-value, write-of-modified-value, on consecutive cycles to the same address (FR-021). Assert via `Trace` snapshot.
- [ ] T017 [US1] Create `mos6502/functional_test.go` embedding `testdata/6502_functional_test.bin` via `//go:embed`. Test sets up a flat 64 KB `FlatMemory` (test helper), loads the ROM, sets `PC` to the documented entry, and runs until either the success trap address is hit or a 200 M-cycle budget is exhausted. Assert success. (SC-001)
- [ ] T018 [US1] Create `mos6502/decimal_test.go` embedding `testdata/6502_decimal_test.bin` via `//go:embed`; same shape as T017, asserting decimal-test success. (SC-001)

### Implementation for User Story 1

- [ ] T019 [P] [US1] Implement `mos6502/addressing.go`: effective-address helpers `effZP`, `effZPX`, `effZPY` (zero-page wrap), `effAbs`, `effAbsX`, `effAbsY` (with the dummy-read on page-cross emitted via `c.read`), `effIndirect` (with NMOS page-bug — high byte from `$xx00` on `JMP ($xxFF)`, FR-013), `effIndexedIndirect` (wrap on `($LL,X)`), `effIndirectIndexed` (page-cross extra cycle on `($LL),Y`). Plus the `AddressingMode` enum and a per-mode byte-length table. (FR-003, FR-013, FR-014, FR-018)
- [ ] T020 [P] [US1] Implement `mos6502/instructions.go`: shared instruction primitives — load (`LDA`/`LDX`/`LDY`), store (`STA`/`STX`/`STY`), transfer (`TAX`/`TAY`/`TSX`/`TXA`/`TXS`/`TYA`), compare (`CMP`/`CPX`/`CPY`), branch (`BCC`/`BCS`/`BEQ`/`BMI`/`BNE`/`BPL`/`BVC`/`BVS` — incl. +1 cycle on taken, +1 more on taken-and-page-crossed, FR-005), flag manipulation (`CLC`/`SEC`/`CLD`/`SED`/`CLI`/`SEI`/`CLV`), logical (`AND`/`ORA`/`EOR`/`BIT`), stack (`PHA`/`PHP`/`PLA`/`PLP`), jumps (`JMP`/`JSR`/`RTS`), `NOP`. (FR-002, FR-004, FR-005, FR-015)
- [ ] T021 [P] [US1] Implement `mos6502/arith.go`: binary `ADC` and `SBC` (sharing `A + ~M + C` for `SBC`); NMOS-faithful BCD `ADC`/`SBC` per Bruce Clark's algorithm — `C` is BCD-correct, `N`/`V`/`Z` are derived from the binary pre-correction intermediate (FR-016); cycle counts identical to binary `ADC`/`SBC` (no 65C02 extra cycle). (FR-016)
- [ ] T022 [P] [US1] Implement `mos6502/rmw.go`: a single `rmw(c *CPU, addr uint16, op func(uint8) uint8)` helper that performs read → dummy-write-of-original → write-of-modified across three calls to `c.read`/`c.write`, then specific wrappers for `ASL`/`LSR`/`ROL`/`ROR`/`INC`/`DEC` against memory. Accumulator-form `ASL A`/`LSR A`/`ROL A`/`ROR A` is implemented inline in `instructions.go` with no memory cycles for the operand. (FR-021)
- [ ] T023 [US1] Implement `mos6502/opcodes.go`: the 256-entry `opcodeTable [256]opcodeFn` literal. Wire all 151 documented opcodes to their handlers from T020/T021/T022; every other slot points at `illegalNOP` (from T009). Also implement the per-opcode `opcodeMetaTable [256]opcodeMeta` with mnemonic / mode / base cycles / length / illegal flag, used by the disassembler (US2). (FR-002, FR-019)
- [ ] T024 [US1] Wire instruction dispatch into `(*CPU).Step` in `mos6502/cpu.go`: fetch opcode byte (one `c.fetch()` call), dispatch through `opcodeTable[op](c)`, return updated cycle count. Note: pre-instruction RESET/IRQ/NMI gating is added in US3 — for US1 the consumer uses `SetRegisters` to bypass RESET (Story-1 acceptance scenario 1). (FR-002, FR-007)
- [ ] T025 [US1] Add `mos6502/testdata/6502_functional_test.bin` (Klaus Dormann ROM, BSD 3-Clause; include `mos6502/testdata/LICENSE_KLAUS_DORMANN.txt`).
- [ ] T026 [US1] Add `mos6502/testdata/6502_decimal_test.bin` (Klaus Dormann decimal-mode ROM; same licence header in the same `LICENSE_KLAUS_DORMANN.txt`).

**Checkpoint**: User Story 1 fully functional — `go test -run TestFunctional ./mos6502/` and `go test -run TestDecimal ./mos6502/` both green. MVP achieved.

---

## Phase 4: User Story 2 — Step, Inspect, and Single-Step Debugging (Priority: P2)

**Goal**: A developer can drive the CPU one instruction (or one bus cycle) at a time and observe the full architectural state between steps, with cycle counts that match documented behaviour including conditional extras.

**Independent Test**: Hand-assemble a 6-instruction routine including a branch-taken-across-page-cross; step the CPU one instruction at a time; assert that after each step the register/flag/cycle values match a reference disassembly. (Story-2 acceptance scenarios 1–3)

### Tests for User Story 2

- [ ] T027 [P] [US2] Add `mos6502/cpu_test.go` covering: `New(mem).Registers().Cycles == 0`; `SetRegisters` round-trip; single `Step()` advances by exactly one instruction; `Run(N)` never splits an instruction; branch-taken-page-cross adds the documented extra cycle and branch-not-taken does not. (Story-2 scenarios 1–3; FR-005, FR-007)
- [ ] T027a [P] [US2] Add a determinism assertion to `mos6502/cpu_test.go`: run a 5 000-instruction synthetic program twice against two independently-constructed `CPU` + `FlatMemory` pairs, capture both `Trace` snapshots and final `Registers`, assert byte-for-byte and cycle-for-cycle equality. (SC-007)
- [ ] T028 [P] [US2] Add `mos6502/disasm_test.go` exercising `Disassemble(mem, pc) (string, length)` against a fixture program covering every addressing mode. Assert the human-readable form and the reported byte length match a golden string per opcode. (Code-quality observability seam)
- [ ] T029 [P] [US2] Add `mos6502/bench_test.go` with `BenchmarkRunNoop` (tight `NOP` loop) and `BenchmarkRunMixedWorkload` (a synthetic mix exercising loads, stores, branches, RMW). Both use `b.ReportAllocs()` and assert `≤ 125 ns/cycle` on amd64 (skipped on non-amd64). (SC-006)

### Implementation for User Story 2

- [ ] T030 [P] [US2] Implement `mos6502/disasm.go`: `Disassemble(mem Memory, pc uint16) (text string, length int)` using `opcodeMetaTable` from T023; emits the mnemonic, operand in canonical 6502 syntax (`$1234`, `$LL,X`, `($LL),Y`, …), and the consumed byte length. Pure function — no CPU state, no allocation in the hot loop (uses a `strconv.AppendUint`-style approach into a re-usable buffer). (FR-020 observability)
- [ ] T031 [US2] Verify `StepCycle()` semantics (test only — implementation lives in T012) and tighten `Run(cycleBudget)` in `mos6502/cpu.go` so that `Run` accumulates whole instructions up to the budget and never overshoots. (FR-008)

**Checkpoint**: User Stories 1 and 2 both work independently. SC-006 throughput benchmark passes.

---

## Phase 5: User Story 4 — Memory and Address-Bus Abstraction for Banked ROM and MMIO (Priority: P2)

**Goal**: Every CPU memory access routes through the host-supplied `Memory` interface; a recording mock memory can verify that the bus trace matches what a real NMOS 6502 would emit for any given instruction sequence.

**Independent Test**: Run a program exercising every addressing mode against a recording mock; assert the bus trace contains exactly the reads and writes the 6502 reference performs, in the correct order, on the correct cycles. (US4 acceptance scenarios 1–3, SC-008)

### Tests for User Story 4

- [ ] T032 [P] [US4] Add `mos6502/trace_test.go` verifying: `Trace` ring-buffer wraps correctly when capacity exceeded; `Snapshot` returns events in chronological order; `(*CPU).SetTrace(nil)` cleanly detaches with zero subsequent overhead. (SC-008)
- [ ] T033 [P] [US4] Add `mos6502/recording_memory_test.go` defining a `recordingMemory` helper (a test-only `Memory` implementation that records `(cycle, addr, kind, value)` tuples) and a `compareTrace` assertion. (US4 scenario 1)
- [ ] T034 [P] [US4] Generate `mos6502/testdata/golden_traces/*.trace` reference files for at least: `lda_imm.trace`, `lda_abs.trace`, `lda_abs_x_nopagecross.trace`, `lda_abs_x_pagecross.trace`, `sta_abs_x.trace` (no extra cycle on store-indexed), `lda_ind_y_pagecross.trace`, `jmp_indirect_normal.trace`, `jmp_indirect_pagebug.trace`, `inc_abs.trace` (RMW double-write), `inc_abs_x.trace`, `asl_a.trace` (accumulator form, no memory cycles), `brk_normal.trace`, `irq_entry.trace`, `nmi_entry.trace`, `rmw_then_branch.trace`. Each `.trace` file is a plain-text sequence of `cycle: R|W $addr = $value` lines. (SC-008)
- [ ] T035 [US4] Add `mos6502/golden_trace_test.go`: a test-loop reading every `testdata/golden_traces/*.trace` file, running the matching setup program, and asserting the recorded `Trace` equals the golden trace byte-for-byte (extra/missing access fails the test with a precise diff). (US4 scenario 1, SC-008)
- [ ] T036 [P] [US4] Add `mos6502/rdy_test.go` verifying `RDY` semantics: `SetRDY(false)` stalls the current read cycle (same access repeats next cycle, no state advance), `SetRDY(false)` does NOT stall writes (the write proceeds), per-cycle re-evaluation works (toggle RDY between cycles and observe). (FR-023, US4 scenario 2)
- [ ] T037 [P] [US4] Add `mos6502/illegal_test.go` verifying: illegal opcodes execute as single-byte 2-cycle NOP; `SetIllegalOpcodeHook(h)` invokes `h(pc, op)` exactly once per illegal opcode with the pre-advance `PC`; `SetIllegalOpcodeHook(nil)` clears the hook and zero further invocations occur. (FR-019)

### Implementation for User Story 4

- [ ] T038 [US4] No new production code required — US4 verifies that the abstraction from Phase 2 (FR-006) holds across the implementation from US1. If T035 reveals any opcode that bypasses `c.read`/`c.write`/`c.fetch` (e.g. an opcode handler that touches `c.mem` directly), fix the offending opcode handler in `mos6502/{addressing,instructions,arith,rmw,opcodes}.go`. (FR-006)

**Checkpoint**: All bus traffic is observable through `Memory` and `Trace`. Hooked illegal-opcode behaviour verified.

---

## Phase 6: User Story 3 — Reset and Interrupt Handling (IRQ, NMI, BRK) (Priority: P3)

**Goal**: RESET, IRQ, NMI, and `BRK` all enter the CPU through the correct vector with the correct pushed state, including the NMOS NMI-hijack-of-BRK/IRQ quirk.

**Independent Test**: Install a known IRQ handler at `$FFFE/$FFFF`, enable interrupts, raise IRQ from the host, step; assert PCH/PCL/P-with-B-clear are pushed in order, `I` is set in the live register, `PC` is loaded from the IRQ vector, exactly 7 cycles are consumed, and `RTI` resumes correctly. (Story-3 acceptance scenarios 1–5)

### Tests for User Story 3

- [ ] T039 [P] [US3] Add `mos6502/interrupts_test.go` — RESET sub-suite: after `New(mem); AssertReset(); Step()`, assert `A==0, X==0, Y==0, SP==0xFD, P==0x24 (I set, U set, B clear, D unchanged-from-construction-of-zero)`, `PC == mem[$FFFC] | mem[$FFFD]<<8`, `Cycles == 7`. (FR-009, Research §1)
- [ ] T040 [P] [US3] In `mos6502/interrupts_test.go` — IRQ sub-suite: with `I` clear and IRQ asserted, the CPU pushes PCH/PCL/P (B clear in pushed copy, U set), sets live `I`, loads PC from `$FFFE/$FFFF`, consumes 7 cycles. With `I` set, IRQ is ignored. (FR-010, Story-3 scenarios 2–3)
- [ ] T041 [P] [US3] In `mos6502/interrupts_test.go` — NMI sub-suite: a single `AssertNMI` is serviced exactly once even if the line stays asserted; `DeassertNMI` + `AssertNMI` re-fires; pushed status has B clear. (FR-011, Story-3 scenario 4)
- [ ] T042 [P] [US3] In `mos6502/interrupts_test.go` — BRK sub-suite: `BRK` pushes `PC+2` (not `PC+1`), pushes `P` with B *set* in pushed copy, sets live `I`, vectors through `$FFFE/$FFFF`, consumes 7 cycles. `RTI` restores the pushed PC and status (B and U masked out of live `P`). (FR-012, Story-3 scenario 5)
- [ ] T043 [P] [US3] In `mos6502/interrupts_test.go` — NMI-hijack sub-suite: assert NMI mid-BRK before vector low-byte latch; vector fetched is `$FFFA/$FFFB`; pushed `B` bit retains the BRK value (1); same scenario for in-flight IRQ → hijacked vector but pushed `B` retains IRQ value (0). (FR-022)

### Implementation for User Story 3

- [ ] T044 [US3] Implement `mos6502/interrupts.go`: a single shared `enterInterrupt(c *CPU, kind interruptKind)` routine encoding the 7-cycle entry sequence from `data-model.md` §6. `kind` is one of `irqInterrupt`, `nmiInterrupt`, `brkInterrupt`, `resetInterrupt`. The routine performs the cycle-correct dummy reads, stack pushes (or fake-pushes on RESET), the NMI-hijack check between cycles 4 and 5 (vector latched at cycle 5; if `c.nmiPending` is true at that point, latch `$FFFA` instead — FR-022), and the two vector-fetch reads on cycles 6 and 7. (FR-009–FR-012, FR-022)
- [ ] T045 [US3] Implement `brk` opcode handler in `mos6502/interrupts.go` (replacing the placeholder slot in `opcodes.go`): it calls `enterInterrupt(c, brkInterrupt)`, which handles the PC+2 push, B=set in pushed P, vector fetch.
- [ ] T046 [US3] In `mos6502/cpu.go` `Step()`: BEFORE opcode fetch, check pending signals in priority order — RESET, then NMI, then IRQ-if-I-clear — and dispatch to `enterInterrupt` if any is pending. On NMI service, clear `nmiPending` (edge-clear). On IRQ service, do NOT clear `irqLine` (level-sensitive — host clears via `AssertIRQ(false)`). (FR-008, FR-010, FR-011)
- [ ] T047 [US3] Implement `RTI` in `mos6502/instructions.go` (was a stub in T020): pull `P` (mask out B and U bits when loading into live `P` per architecture), pull `PCL`, pull `PCH`, consume 6 cycles. (FR-012)

**Checkpoint**: All user stories complete. Klaus Dormann functional + decimal tests still pass; new interrupt + RDY + illegal-opcode + golden-trace tests pass.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Documentation completeness, performance verification, constitution-gate satisfaction.

- [ ] T048 [P] Expand `mos6502/doc.go` to the final package overview: feature summary, the `Memory` interface contract, the control surface, a copy of the quickstart example. (FR-020, constitution Principle I)
- [ ] T049 [P] Audit every exported identifier in `mos6502/` for a doc comment (`golint`/`revive` rule); add missing ones. (FR-020)
- [ ] T050 [P] Add a `mos6502/quickstart_test.go` that runs the exact program from `specs/001-cpu-6502-core/quickstart.md` end-to-end and asserts the expected output (`A = $42`, `mem[$0200] = $42`, `cycles = 17`). Keeps the quickstart honest under future refactors.
- [ ] T050a [P] Add an external-consumer example program at `examples/external/main.go` (separate Go module `github.com/gobeeb/GoBeeb/examples/external` with its own `go.mod` that depends on `github.com/gobeeb/GoBeeb`) that imports `mos6502`, supplies a minimal `FlatRAM`, and runs a ≥ 1 000-instruction routine (a hand-assembled tight loop will do — e.g. counted-down `DEX`/`BNE` with periodic side effects), printing the final cycle count. Verified by a CI step `go run ./examples/external/`. (SC-005)
- [ ] T051 Run full coverage: `go test -coverprofile=cover.out -covermode=atomic ./mos6502/`; assert ≥ 80 % delta line coverage; raise to ≥ 90 % on `opcodes.go`/`addressing.go`/`arith.go`/`rmw.go`/`interrupts.go`. (Constitution Principle II)
- [ ] T052 Run benchmarks: `go test -bench=. -benchmem -benchtime=5s ./mos6502/`; record `ns/op` for both benchmarks; assert ≤ 125 ns/cycle on amd64, 0 B/op, 0 allocs/op. Record numbers in a `BENCHMARKS.md` table under `mos6502/` for future regression comparison. (SC-006, constitution Principle IV)
- [ ] T053 [P] Run `make fmt vet lint` over the new package; resolve any findings; ensure CI workflow from T005 stays green.
- [ ] T054 Update `CLAUDE.md` (and any README that lands in this phase) to reflect the now-implemented `mos6502/` package and its public API.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately.
- **Foundational (Phase 2)**: Depends on Setup completion. **BLOCKS all user stories.**
- **User Story 1 (Phase 3, P1)**: Depends on Foundational. Delivers the MVP — pass Klaus Dormann tests.
- **User Story 2 (Phase 4, P2)**: Depends on Foundational. Independent of US1's opcode implementation in principle, but the test fixtures it uses (Story-2 acceptance scenarios) need real opcodes — so in practice schedule US2 after US1 unless a developer wants to scaffold opcodes alongside.
- **User Story 4 (Phase 5, P2)**: Depends on Foundational + US1 (golden traces need real opcode behaviour to compare against). Schedule after US1.
- **User Story 3 (Phase 6, P3)**: Depends on Foundational + US1 (interrupt entry uses the same opcode-dispatch machinery; `BRK` and `RTI` live in the instructions file populated by US1).
- **Polish (Phase 7)**: Depends on all user stories complete.

### Within Each User Story

- Tests are written **first** and asserted to fail before the matching production code lands. The constitution treats tests as proof-of-behaviour, not after-the-fact decoration.
- Models / interfaces before services / handlers (within `mos6502/` this means: `addressing.go` before opcode handlers in `instructions.go`/`arith.go`/`rmw.go`; the dispatch table in `opcodes.go` is wired last in US1).

### Parallel Opportunities

- **Phase 1**: T002–T006 are all [P] — different files.
- **Phase 2**: T008/T009/T010 are [P]; T007/T011/T012 are sequential (`cpu.go` skeleton must exist before bus-cycle primitives plug into it).
- **Phase 3 (US1)**: All six test tasks T013–T018 [P]. All four implementation tasks T019–T022 [P]. T023 (dispatch table wire-up) depends on T019–T022. T024 (Step wiring) depends on T023. T025/T026 (test ROMs) [P] with anything.
- **Phase 4 (US2)**: T027/T028/T029 are [P]. T030/T031 are independent of each other.
- **Phase 5 (US4)**: All five test tasks T032/T033/T034/T036/T037 are [P]. T035 depends on T033/T034. T038 is conditional remediation only.
- **Phase 6 (US3)**: T039–T043 are [P] (separate sub-test functions in the same file — Go conventions allow this in a single `_test.go` file; if a strict-different-file interpretation of [P] is preferred, split into `interrupts_reset_test.go`/`interrupts_irq_test.go`/`interrupts_nmi_test.go`/`interrupts_brk_test.go`/`interrupts_hijack_test.go`). T044 enables T045/T046/T047 (sequential).
- **Phase 7**: T048/T049/T050/T053 are [P]. T051/T052/T054 are sequential after the rest.

---

## Parallel Example: User Story 1

```bash
# All US1 tests first, in parallel (different files):
Task: "T013 [P] [US1] Per-mode addressing tests in mos6502/addressing_test.go"
Task: "T014 [P] [US1] Per-opcode tests in mos6502/opcodes_test.go"
Task: "T015 [P] [US1] ADC/SBC binary + BCD tests in mos6502/arith_test.go"
Task: "T016 [P] [US1] RMW double-write tests in mos6502/rmw_test.go"
Task: "T017 [US1] Klaus Dormann functional ROM runner in mos6502/functional_test.go"
Task: "T018 [US1] Klaus Dormann decimal ROM runner in mos6502/decimal_test.go"

# Then the implementation, in parallel (different files):
Task: "T019 [P] [US1] Addressing-mode helpers in mos6502/addressing.go"
Task: "T020 [P] [US1] Shared instruction primitives in mos6502/instructions.go"
Task: "T021 [P] [US1] Binary + BCD arithmetic in mos6502/arith.go"
Task: "T022 [P] [US1] RMW helpers in mos6502/rmw.go"

# Then wire-up (sequential):
Task: "T023 [US1] Dispatch table in mos6502/opcodes.go"
Task: "T024 [US1] Step dispatch in mos6502/cpu.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 only)

1. Complete Phase 1: Setup.
2. Complete Phase 2: Foundational (CRITICAL — blocks every story).
3. Complete Phase 3: User Story 1.
4. **STOP and VALIDATE**: `go test -run 'TestFunctional|TestDecimal' ./mos6502/` must be green.
5. Tag a `v0.1.0` pre-release. MVP shipped.

### Incremental Delivery

1. Setup + Foundational → foundation ready.
2. Add User Story 1 → MVP (Klaus Dormann passes).
3. Add User Story 2 → step/inspect API + disassembler + benchmark.
4. Add User Story 4 → bus-trace conformance + recording mock + golden traces.
5. Add User Story 3 → interrupts + RDY + NMI hijack.
6. Polish phase → doc-completeness, coverage gates, benchmark records.

### Single-Developer Sequencing (this project's expected mode)

This is a one-developer project (per the GoBeeb constitution's preamble and the existing solo `git log`). The "parallel team" model in the template does not apply. Single-dev order:

1. Phase 1 (Setup) — half-day.
2. Phase 2 (Foundational) — one day.
3. Phase 3 (US1) — two-to-three days; the bulk of the work is the 151 opcode handlers + their unit tests, but ~80 % of opcodes are mechanical once the first ten are done.
4. Phase 4 (US2) — half-day (disassembler is the only new code; the Step API was scaffolded in Phase 2).
5. Phase 5 (US4) — one day (golden-trace generation is fiddly but mostly data).
6. Phase 6 (US3) — one-to-two days (interrupt entry routine plus NMI-hijack tests).
7. Phase 7 (Polish) — half-day.

Total estimate: 6–8 working days for the full feature.

---

## Notes

- `[P]` tasks operate on different files and have no in-phase ordering dependency. Two `[P]` tasks in the same phase may proceed in any order or concurrently.
- `[Story]` label is mandatory inside user-story phases; absent from Setup, Foundational, and Polish per the template rules.
- Tests are written first within each user story; the test must fail before the matching code lands (constitution Principle II).
- Atomic commits per task or per logical group (constitution Workflow rule).
- Stop at any checkpoint to validate independently; do not push partial-story commits to `master` without their accompanying tests green.
