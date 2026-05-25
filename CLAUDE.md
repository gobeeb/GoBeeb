<!-- SPECKIT START -->
For additional context about technologies to be used, project structure,
shell commands, and other important information, read the current plan:

- Current feature: 6502 CPU Core (branch `001-cpu-6502-core`)
- Plan: [specs/001-cpu-6502-core/plan.md](specs/001-cpu-6502-core/plan.md)
- Spec: [specs/001-cpu-6502-core/spec.md](specs/001-cpu-6502-core/spec.md)
- Research / decisions: [specs/001-cpu-6502-core/research.md](specs/001-cpu-6502-core/research.md)
- Data model: [specs/001-cpu-6502-core/data-model.md](specs/001-cpu-6502-core/data-model.md)
- API contracts (Go): [specs/001-cpu-6502-core/contracts/](specs/001-cpu-6502-core/contracts/)
- Quickstart: [specs/001-cpu-6502-core/quickstart.md](specs/001-cpu-6502-core/quickstart.md)

Language: Go 1.22+. Module: `github.com/gobeeb/GoBeeb`. Target package: `mos6502/`.
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
- 91 unit tests covering addressing modes, RMW double-write trace, BCD edges, stack wrap, interrupts (RESET/IRQ/NMI/BRK/RTI), NMI hijack, illegal opcode hook, RDY gating, golden bus traces.
- Coverage: 99.3% on `mos6502/` (gate: ≥80%).
- Benchmarks: ~4.3 ns/cycle on amd64, 0 B/op, 0 allocs/op (gate: ≤125 ns/cycle).

Known v1 limitations:
- RDY gated at instruction boundary, not per bus cycle.
- `Run(budget)` may overshoot the budget by up to one instruction (never splits mid-instruction).
- Decimal-mode validation via unit tests + functional ROM's internal BCD suite (separate `6502_decimal_test.bin` ROM not pre-built upstream).
- `StepCycle()` runs one whole instruction (true per-cycle scheduling deferred).

Make targets: `make fmt vet lint test bench cover`.
