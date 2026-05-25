# Phase 0 Research: 6502 CPU Core

**Feature**: 001-cpu-6502-core

**Date**: 2026-05-25

**Purpose**: Resolve all `NEEDS CLARIFICATION` items left after `/speckit-clarify`, settle the two deferred items from the spec's Coverage Summary, and record the best-practice research that drives the Phase 1 design.

## 1. Deferred item — Reset post-state for `A`, `X`, `Y`, `SP`, `P` (Assumption A7)

### Decision

On RESET the CPU initialises architectural state to a fixed, deterministic value set:

| Register | Post-RESET value | Source                                                                  |
|----------|------------------|-------------------------------------------------------------------------|
| `A`      | `$00`            | NMOS leaves it undefined; project picks zero for determinism (FR-007 / SC-007). |
| `X`      | `$00`            | Same rationale.                                                         |
| `Y`      | `$00`            | Same rationale.                                                         |
| `SP`     | `$FD`            | NMOS reset sequence performs three "fake stack pushes" that decrement `SP` three times from its uninitialised value. Stable, well-documented post-reset value of `$FD` (also what `6502_functional_test` expects). |
| `P`      | `0x24` (`--1--I--` plus unused bit) | Reset sets the `I` (interrupt disable) flag (FR-009). The unused bit (bit 5) reads as 1 on a real chip; `B` is cleared in the live `P` (it has no real flag — see below). `D` is *not* cleared by reset on NMOS 6502 (contrast: 65C02 clears `D`). The wider BBC emulator MOS sets `D` to a known state itself before any BCD use. |
| `PC`     | `mem[$FFFC] \| (mem[$FFFD]<<8)` | FR-009.                                                            |
| Cumulative cycles | reset to 0 *after* the 7-cycle reset sequence (so first executed instruction starts at cycle 7) | Matches `6502_functional_test` expectations and gives a clean boundary. |

### Rationale

- The constitution requires deterministic tests; uninitialised state is the worst kind of test flake.
- `SP = $FD` is what virtually every published NMOS reference and every passing emulator uses; choosing anything else would break `6502_functional_test`'s expectations for the early instructions that touch the stack.
- Leaving `D` *uncleared* on reset is the documented NMOS behaviour — the BBC MOS does not rely on reset clearing `D`, so doing the right thing here is free.
- The `B` bit is not a real flag in `P`; it only exists in the *pushed* status byte (set for BRK, clear for IRQ/NMI). Storing it as `0` in the live register is conventional and consistent with what `PHP` will push (which uses a per-instruction "push-with-B-set" mask).

### Alternatives considered

1. **All-zeros including `SP = $00`** — rejected: breaks the first stack operation after reset, fails the functional test.
2. **Leave registers `$FF`** (some toy emulators do this) — rejected: arbitrary, doesn't match real silicon, makes RESET-state assertions fragile across emulators.
3. **Expose a `WithResetState(...)` option** — rejected as YAGNI for v1; can be added later without an API break because the constructor will accept functional options.

## 2. Deferred item — Memory interface error semantics

### Decision

The `Memory` interface is **infallible** — both `Read` and `Write` return without an error and without panicking:

```go
type Memory interface {
    Read(addr uint16) uint8
    Write(addr uint16, value uint8)
}
```

The host implementation is responsible for handling any "internal" failure (unmapped address, slow peripheral, bad ROM bank) and returning *some* byte for reads (the de-facto convention is `$FF` for "open bus" on a BBC, but the CPU does not care — that is purely a host concern).

### Rationale

- A real NMOS 6502 has no concept of a bus error. Every clock cycle, ALE rises, an address is on the bus, and data either flows in (read) or out (write). There is no protocol path back into the CPU for "this access failed".
- Returning `error` from `Read`/`Write` would force the CPU's hot path to allocate (interface satisfying `error` boxes the dynamic value) and to branch on error every cycle — both fatal for SC-006's ≤ 125 ns/cycle budget.
- The host has perfect control: BBC MMIO that wants to signal an internal fault can record it in its own state, expose a separate error channel to the emulator front-end, and still return `$FF` to the CPU. This keeps the CPU's contract clean and the bus traffic faithful to real hardware.
- Panics inside `Memory` propagate up through `Step`/`Run`. That is acceptable for genuine programming errors in the host (e.g. nil-deref) — the constitution allows panics for unrecoverable host bugs — but a *well-behaved* `Memory` will never panic.

### Alternatives considered

1. **`Read(addr) (uint8, error)`** — rejected: forces allocation per cycle, breaks the zero-allocation perf budget, has no real-hardware analogue.
2. **`Read(addr) uint8; Err() error` (deferred error)** — rejected: hides the error site, complicates the CPU loop with an after-step error check on every cycle.
3. **Two-tier interface (`Memory` + `MemoryWithError`)** — rejected: API surface bloat for no concrete user.

## 3. Tech-choice research

### 3.1 Opcode dispatch: function-pointer table vs `switch` statement

**Decision**: 256-entry function-pointer table indexed by the opcode byte.

```go
var opcodeTable = [256]opcodeFn{
    0x00: brk,
    0x01: oraIndX,
    /* … 256 entries; illegals point at illegalNOP */
}
```

**Rationale**:

- O(1) dispatch, branch-predictable on amd64 (indirect call through a hot, fully-populated function-pointer table is well-predicted in modern CPUs after warm-up).
- Each opcode handler is small enough to fit in i-cache; the table itself is 2 KB (256 × 8 bytes), trivially L1-resident.
- The big-`switch` approach also compiles to a jump table in Go for dense byte cases, but obscures the structure and makes per-opcode benchmarking and instrumentation harder. The function-pointer table is also what `Fake6502` (C reference) and `MAME m6502` use.
- Illegal-opcode handling is one trivial bullet: every illegal slot in the table is initialised to a shared `illegalNOP` function pointer, so FR-019 needs no extra branch on the hot path.

**Alternatives considered**: `switch` (rejected — harder to instrument), `map[uint8]opcodeFn` (rejected — allocation, slow), code generation from a CSV (deferred — not needed for v1; the 256-entry literal is acceptable LOC and obvious to readers).

### 3.2 Sub-cycle execution model

**Decision**: each opcode is implemented as an explicit sequence of bus-cycle method calls on the CPU. The CPU has a `tick()` primitive that performs one bus access (read or write) through the `Memory` interface and increments the cycle counter. Each opcode handler invokes `tick()` exactly as many times as a real 6502 would, in the correct order, with the correct address each time.

```go
// example: LDA absolute (4 cycles)
func ldaAbs(c *CPU) {
    lo := c.fetch()                    // cycle 2: read PC+1
    hi := c.fetch()                    // cycle 3: read PC+2
    addr := uint16(lo) | uint16(hi)<<8
    c.A = c.read(addr)                 // cycle 4: read effective addr
    c.setNZ(c.A)
}
```

`c.read` / `c.write` / `c.fetch` are the three "tick"-bearing primitives. They:

1. Check `RDY` first (for reads); if asserted-low, return the previously-latched value and do not advance state (FR-023).
2. Invoke the host `Memory` interface.
3. Increment `c.cycles` by 1.
4. Optionally emit a `BusEvent` to the bus-trace recorder (only if a recorder is attached; zero cost otherwise).

The CPU also exposes a public `StepCycle()` that runs *one* bus cycle, used by the host emulator when it needs to interleave CPU and video-ULA timing precisely.

**Rationale**: This is the only model that simultaneously satisfies FR-018 (sub-cycle accuracy), FR-006 (every access through the interface), and FR-023 (RDY checked per-cycle). It also keeps each opcode handler obvious and reviewable — the cycle structure is the code structure.

**Alternatives considered**:

1. **Whole-instruction handlers + post-hoc cycle inflation** — rejected; cannot reproduce per-cycle bus order required by FR-006/FR-018.
2. **Microcode interpreter (each opcode is data-driven)** — rejected as over-engineered for 151 opcodes; obscures NMOS quirks that are easier to express directly.

### 3.3 BCD implementation

**Decision**: Implement Bruce Clark's NMOS algorithm directly (no lookup table). For `ADC`: compute the binary intermediate, derive `N`, `V`, `Z` from it, then BCD-correct the low and high nibbles to produce the final `A` and `C`. For `SBC`: same shape, using the 6502's `A + ~M + C` model so that decimal and binary `SBC` share their first half.

**Rationale**:

- Bruce Clark's algorithm is the reference for "what real NMOS silicon does"; it is what `6502_decimal_test` verifies (SC-001).
- A lookup table would be 64 KB (every `(A, M, C)` triple) — not worth it for the constant cost of a half-dozen comparisons.
- Sharing the binary path with the decimal-correction step keeps the code small and the `N`/`V`/`Z` derivation in one place.

**Reference**: Clark, B. *Decimal Mode in NMOS 6502 Processors*, http://www.6502.org/tutorials/decimal_mode.html (the de-facto specification for NMOS BCD behaviour).

### 3.4 Test ROM hosting

**Decision**: embed `6502_functional_test.bin` and `6502_decimal_test.bin` from Klaus Dormann's test suite (https://github.com/Klaus2m5/6502_65C02_functional_tests) via `//go:embed` into the package's `_test.go` file. The license (BSD 3-Clause) permits redistribution with attribution; an `mos6502/testdata/LICENSE_KLAUS_DORMANN.txt` accompanies the binaries.

**Rationale**: `go test` works against a freshly-`go get`-ed module with no extra setup. Reproducible across CI and developer machines. ~64 KB per binary, trivially small.

**Alternatives considered**: download at test time (rejected — flaky CI, network dependency), require user to provide (rejected — friction for contributors).

### 3.5 Bus-trace recorder API

**Decision**: optional, attached via `cpu.SetTrace(t *Trace)`. `nil` (default) means no tracing and zero overhead in the hot path. When attached, every bus cycle appends a `BusEvent{Cycle, Addr, Value, Kind}` to a fixed-capacity ring buffer (no allocation). Golden tests assert `expectedTrace.Equal(actualTrace)` against a `.trace` text file.

**Rationale**: SC-008 demands per-cycle bus-trace conformance; this gives the test layer a clean way to capture and compare without weighing down production runs. Ring-buffer + pre-allocated slice means tracing itself is zero-alloc.

## 4. Validation

- All `NEEDS CLARIFICATION` markers in `spec.md`: **0** (verified by grep, see `checklists/requirements.md`).
- Deferred items from spec Coverage Summary: **both resolved above** (§ 1 Reset post-state; § 2 Memory interface error semantics).
- Outstanding research questions: **none**.

Phase 0 complete. Proceed to Phase 1 design.
