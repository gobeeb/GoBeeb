# Feature Specification: 6502 CPU Core

**Feature Branch**: `001-cpu-6502-core`

**Created**: 2026-05-25

**Status**: Draft

**Input**: User description: "This project is an emulator for a BBC Model B. The first part of the implementation is emulating the 6502 CPU. Please implement the 6502 CPU including its registers, instructions, memory (and address bus). It needs to support the various addressing modes as well. The resulting package will be used in the wider emulator for the BBC Model B. References: https://6502.org/users/obelisk/6502/architecture.html, registers.html, instructions.html, addressing.html, algorithms.html."

## Clarifications

### Session 2026-05-25

- Q: Read-modify-write "double-write" semantics on memory operands → A: Faithfully emulate NMOS RMW (read → dummy-write-of-old-value → write-of-new-value, three consecutive bus cycles on the same address). Host observes all three accesses.
- Q: NMI hijack of in-progress BRK / IRQ vector fetch → A: Emulate the hijack faithfully. If NMI asserts before the low-byte vector latch (pre-cycle-5) of a BRK or IRQ sequence, the vector fetch is redirected to `$FFFA/$FFFB`; the pushed PC and pushed status byte (including the original B bit) are unchanged from what BRK/IRQ would normally push.
- Q: Clock-stretching / `RDY` for 1 MHz peripheral access → A: CPU exposes an `RDY`-equivalent input with faithful NMOS semantics — `RDY` asserted stalls a read cycle (CPU repeats the cycle, no internal state advances); writes proceed unaffected. Host (BBC machine layer) owns 1 MHz / 2 MHz alignment by driving `RDY` per cycle.
- Q: Behaviour on undocumented / illegal NMOS opcodes → A: Treat as single-byte, 2-cycle NOP **and** expose an optional host hook ("illegal-opcode encountered") that receives the `PC` and opcode byte. Hook is no-op if unregistered. Guest software continues; debugging surfaces the event.
- Q: BCD-mode `N`, `V`, `Z` flag behaviour for `ADC`/`SBC` (FR-016) → A: NMOS-faithful. `N`, `V`, `Z` derived from the **binary (pre-BCD-correction)** result; only `C` is BCD-correct. Cycle count is identical to binary `ADC`/`SBC` (no 65C02-style extra cycle). Reference: Bruce Clark's 6502 decimal-mode document; verified by Klaus Dormann's `6502_decimal_test`.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Execute a 6502 Program to a Correct Final State (Priority: P1)

A consumer of the CPU core (initially the wider BBC Model B emulator, but also developers writing unit tests and small test ROMs) loads a 6502 program into memory at a known address, points the program counter at it, and runs the CPU. After the program halts (or the requested number of cycles elapses), the consumer can read the final register values, processor status flags, and any memory the program touched, and these MUST match the behaviour of a real NMOS 6502 executing the same program with the same input memory.

**Why this priority**: This is the irreducible MVP. Without a correct, observable execution loop, the CPU core cannot be used to bring up any other emulator subsystem (video, sound, I/O, OS ROM). Every later capability — interrupts, debugging, integration with the BBC machine — depends on the execute-and-inspect cycle being trustworthy first.

**Independent Test**: Can be fully tested by loading well-known 6502 functional test suites (e.g. Klaus Dormann's `6502_functional_test`) into a flat 64 KB memory image, running the CPU until the program reaches its documented success trap, and verifying that the final program counter and registers indicate success and that no test sub-routine wrote a failure marker.

**Acceptance Scenarios**:

1. **Given** a freshly-constructed CPU and a 64 KB memory containing a known machine-code routine at $0600 that loads `A` with $42 and stores it at $0200, **When** the consumer sets `PC = $0600` and steps the CPU until the routine completes, **Then** `A == $42`, memory at $0200 == $42, and the consumed cycle count matches the documented total for those instructions.
2. **Given** the Klaus Dormann 6502 functional test image loaded at its expected address, **When** the CPU is run until it reaches the documented success trap address, **Then** the CPU reports success and no instruction took an incorrect cycle count, no flag was set wrongly, and no memory location outside the test scratch area was modified.
3. **Given** a program that exercises every official instruction at least once across every applicable addressing mode, **When** executed against the CPU, **Then** every opcode dispatches to the correct operation, every addressing-mode calculation produces the documented effective address, and the processor status flags after each instruction match the documented behaviour for that instruction (including the NMOS-specific quirks).

---

### User Story 2 - Step, Inspect, and Single-Step Debugging (Priority: P2)

A developer building or debugging the wider emulator, OS ROM bring-up, or a piece of BBC software needs to drive the CPU one instruction (or a fixed number of cycles) at a time and read the full architectural state between steps: all registers, the processor status flags, the stack pointer, and the cycle counter. They also need to be able to write to that state to set up regression tests and reproduce bug reports.

**Why this priority**: Without programmatic step-and-inspect, debugging the emulator itself (and any guest software it runs) becomes guesswork. This is a hard requirement for the test strategy in User Story 1 and for every later subsystem that needs to observe CPU behaviour around a specific cycle.

**Independent Test**: Construct a CPU with a memory containing a short hand-assembled sequence, set initial register state explicitly, step the CPU one instruction at a time, and after each step assert that the exposed register/flag/cycle values match the expected post-instruction state from a reference disassembly.

**Acceptance Scenarios**:

1. **Given** a CPU whose registers have been set to known values via the public state API, **When** the consumer calls "step one instruction", **Then** exactly one instruction is fetched, decoded, and executed, the reported cycle count advances by exactly the cycles documented for that instruction (including any conditional extra cycles for page crossing or branch-taken), and every register/flag is readable in its post-instruction state.
2. **Given** a running CPU, **When** the consumer requests "run for at most N cycles", **Then** the CPU executes whole instructions until the next instruction would push the consumed cycle count above N, then stops with the CPU in a clean inter-instruction state.
3. **Given** an instruction stream containing a branch that crosses a page boundary, **When** the consumer steps that branch, **Then** the cycle count reported includes the documented extra cycle for the page crossing on branches that are taken, and does not include it for branches not taken.

---

### User Story 3 - Reset and Interrupt Handling (IRQ, NMI, BRK) (Priority: P3)

The wider BBC emulator needs to assert RESET on power-on, raise NMI on certain hardware events, and raise IRQ from peripherals (timers, ACIA, etc.). The 6502 core MUST honour these signals using the standard 6502 vector locations and timing, and MUST cooperate with the `I` (interrupt-disable) and `B` (break) status flags as a real 6502 does.

**Why this priority**: Required for any non-trivial BBC software (the MOS itself uses IRQs heavily), but the core execute-and-inspect loop in P1 is independently useful for testing pure arithmetic / logic / addressing-mode behaviour before interrupts are introduced. Hence P3 rather than P1.

**Independent Test**: Run a small program that enables interrupts, install a known IRQ handler at the vector address, raise an IRQ from the host side, and verify that the CPU pushes PC and P to the stack in the correct order, sets the `I` flag, fetches the new PC from $FFFE/$FFFF, executes the handler, and resumes via `RTI` with the correct restored state.

**Acceptance Scenarios**:

1. **Given** a CPU that has been signalled RESET, **When** the next step is taken, **Then** the CPU loads PC from the reset vector at $FFFC/$FFFD, sets the `I` flag, leaves other registers in their documented post-reset state, and consumes the documented number of cycles for reset.
2. **Given** a CPU running with `I` clear and an IRQ asserted by the host, **When** the next instruction boundary is reached, **Then** the CPU pushes PCH, PCL, and the status byte (with `B` clear in the pushed copy) to the stack, sets `I`, loads PC from $FFFE/$FFFF, and consumes 7 cycles.
3. **Given** a CPU running with `I` set and an IRQ asserted, **When** stepped, **Then** the CPU ignores the IRQ and continues normal execution.
4. **Given** an NMI is asserted while the CPU is running, **When** the next instruction boundary is reached, **Then** the CPU services the NMI via the $FFFA/$FFFB vector regardless of the `I` flag, and a second assertion of NMI without it first being de-asserted does not re-trigger (edge-triggered behaviour).
5. **Given** a program executes a `BRK` instruction, **When** stepped, **Then** PC+2 (not PC+1) is pushed, the status byte is pushed with `B` *set* in the pushed copy, `I` is set in the running CPU, PC is loaded from $FFFE/$FFFF, and the documented 7 cycles are consumed.

---

### User Story 4 - Memory and Address-Bus Abstraction for Banked ROM and Memory-Mapped I/O (Priority: P2)

The BBC Model B is not flat RAM: it has a paged OS/language ROM, sideways ROM banks, and memory-mapped peripherals (SHEILA: VIA, ACIA, video ULA, etc.) in the $FC00–$FEFF region. The CPU core MUST route every read and write through a host-supplied memory interface so that the wider emulator can install ROM regions, intercept I/O accesses, and observe bus traffic.

**Why this priority**: P1 only requires a flat 64 KB array to validate the CPU itself. But the moment the wider emulator integrates this core, every non-RAM access has side-effects (e.g. reading `$FE44` ticks the VIA timer, writing `$FE21` programs the video palette). Without this abstraction, the core is unshippable into the BBC machine.

**Independent Test**: Provide a mock memory implementation that records every read and write (address, value, read/write, cycle number). Run a program that exercises every addressing mode and verify that the recorded bus trace contains exactly the reads and writes the 6502 reference performs for those instructions, in the correct order, with no spurious or missing accesses.

**Acceptance Scenarios**:

1. **Given** a host-supplied memory implementation, **When** the CPU performs any read or write — instruction fetch, operand fetch, effective-address calculation, indirect pointer fetch, stack operation, vector fetch — **Then** the access is delegated to the host memory interface; the CPU never bypasses it.
2. **Given** the CPU executes an indexed addressing mode (e.g. `LDA $1234,X`) where the effective address crosses a page boundary, **When** stepped, **Then** the documented "dummy read" of the un-fixed-up address is issued to the host memory interface on the exact bus cycle a real NMOS 6502 would issue it, followed on the next cycle by the corrected read.
3. **Given** the host installs a region as read-only ROM, **When** the CPU writes to an address in that region, **Then** the write is delivered to the host memory interface (which discards it), the CPU itself does not enforce read-only semantics, and the write consumes the same cycles as a write to RAM.

### Edge Cases

- **Indirect JMP page-boundary bug**: `JMP ($xxFF)` on the NMOS 6502 reads the high byte of the target from `$xx00`, not `$(xx+1)00`. The core MUST reproduce this bug; emulating the "fixed" CMOS behaviour for this case is incorrect for a BBC Model B.
- **RMW double-write**: `ASL`, `LSR`, `ROL`, `ROR`, `INC`, and `DEC` against a memory operand issue three bus cycles to the effective address: (1) read original value, (2) write the *original* (un-modified) value back, (3) write the modified value. The dummy write of the original value MUST be visible on the host memory bus. BBC software (and the MOS) relies on this when manipulating VIA registers — e.g. clearing IFR bits while preserving still-asserted bits — and `INC`/`DEC` against video-ULA-style write-only registers depends on the same pattern.
- **Stack wrap**: the stack lives in `$0100`–`$01FF` and the stack pointer is 8 bits. Pushing when `SP == $00` MUST wrap `SP` to `$FF` (writing to `$0100`); pulling when `SP == $FF` MUST wrap to `$00` (reading from `$0100`).
- **Zero-page indexed wrap**: `LDA $80,X` with `X == $90` MUST read from `$0010`, not `$0110`. All zero-page indexed modes wrap within the zero page.
- **Branch range**: relative branches are signed 8-bit (`-128`..`+127`) from the byte *after* the branch operand. Branches that cross a page boundary cost one extra cycle when taken.
- **Decimal mode**: with the `D` flag set, `ADC` and `SBC` MUST perform BCD arithmetic. NMOS 6502 leaves `N`, `V`, and `Z` undefined as a software contract, but real silicon derives them from the binary (pre-BCD-correction) intermediate; the core MUST do the same so it passes Klaus Dormann's `6502_decimal_test`. Only `C` is BCD-correct.
- **BRK PC offset**: `BRK` is a 1-byte instruction but pushes PC+2, leaving a 1-byte "signature" gap. Handlers that read the signature MUST see the correct byte.
- **IRQ vs BRK distinguishing**: an IRQ pushes the status byte with `B` clear; a `BRK` pushes it with `B` set. Both vector through `$FFFE/$FFFF`. Handlers rely on this bit to distinguish; the core MUST preserve the distinction.
- **NMI edge-triggering**: NMI is edge-triggered. The core MUST not re-fire NMI on a still-asserted line; the host must de-assert and re-assert.
- **NMI hijack of BRK / IRQ**: see FR-022. (Normative statement; full timing and pushed-state contract live there.)
- **Reset latches initial PC from vector, not from $0000**: a freshly-constructed CPU MUST NOT start executing at `$0000`; it MUST read the reset vector at `$FFFC/$FFFD` on its first cycle after RESET.
- **Undefined opcodes**: any byte the CPU fetches as an opcode that is not one of the 151 documented NMOS opcodes is out of scope for v1 (see Assumption A3). The core's behaviour on such bytes MUST be documented (e.g. "treated as NOP" or "returns an explicit illegal-opcode error to the host") and MUST be deterministic.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The CPU core MUST implement an 8-bit accumulator `A`, two 8-bit index registers `X` and `Y`, an 8-bit stack pointer `SP`, a 16-bit program counter `PC`, and an 8-bit processor status register `P` with bits Negative, Overflow, (unused), Break, Decimal, Interrupt-disable, Zero, Carry — matching the NMOS 6502.
- **FR-002**: The CPU core MUST decode and execute all 151 documented NMOS 6502 opcodes, covering the full set of 56 mnemonics from the Obelisk reference (ADC, AND, ASL, BCC, BCS, BEQ, BIT, BMI, BNE, BPL, BRK, BVC, BVS, CLC, CLD, CLI, CLV, CMP, CPX, CPY, DEC, DEX, DEY, EOR, INC, INX, INY, JMP, JSR, LDA, LDX, LDY, LSR, NOP, ORA, PHA, PHP, PLA, PLP, ROL, ROR, RTI, RTS, SBC, SEC, SED, SEI, STA, STX, STY, TAX, TAY, TSX, TXA, TXS, TYA).
- **FR-003**: The CPU core MUST support all 13 NMOS 6502 addressing modes: Implicit, Accumulator, Immediate, Zero Page, Zero Page,X, Zero Page,Y, Relative, Absolute, Absolute,X, Absolute,Y, Indirect, Indexed Indirect (Indirect,X), and Indirect Indexed (Indirect),Y — each routed through the correct effective-address calculation and consuming the documented number of bytes and cycles.
- **FR-004**: For each executed instruction, the CPU core MUST update the processor status flags (`N`, `V`, `Z`, `C`) exactly as documented for that instruction; flags that an instruction does not affect MUST be left unchanged.
- **FR-005**: For each executed instruction, the CPU core MUST report (or accumulate) a cycle count equal to the documented base cost plus the documented conditional extras for: page-boundary crossing on indexed reads (`ABS,X`, `ABS,Y`, `(IND),Y`), branch taken, and branch taken across a page boundary.
- **FR-006**: All memory accesses — opcode fetch, operand fetch, indirect pointer fetch, effective-address read/write, stack push/pull, vector fetch on RESET/IRQ/NMI/BRK, dummy reads on indexed-with-page-cross addressing, and the dummy write of the original value during read-modify-write instructions (see FR-021) — MUST be performed through a host-supplied memory interface that exposes 8-bit read and 8-bit write operations over a 16-bit address. The CPU MUST NOT hold its own private copy of the address space. Per FR-018, every such access MUST occur on the bus cycle a real NMOS 6502 would issue it, in the documented order.
- **FR-021**: Read-modify-write instructions against memory operands (`ASL`, `LSR`, `ROL`, `ROR`, `INC`, `DEC`) MUST issue three bus cycles to the effective address: a read of the current value, a write of that **original un-modified value** back to the same address, then a write of the modified value. All three accesses MUST be visible to the host memory interface on consecutive bus cycles. RMW against the accumulator addressing-mode form (e.g. `ASL A`) MUST NOT issue any memory cycles for the operand itself.
- **FR-022**: The CPU core MUST emulate the NMOS "NMI hijack" quirk. If the NMI line is asserted (low transition latched) **between cycles 4 and 5 of an in-progress BRK or IRQ sequence — i.e. before the low-byte vector read that occurs on cycle 5** — the vector fetch MUST be redirected to `$FFFA/$FFFB` (the NMI vector). The pushed PC and the pushed status byte (including the `B` bit — `1` for BRK, `0` for IRQ) MUST remain exactly what the original BRK/IRQ would have pushed; only the address from which the new `PC` is loaded changes. After the hijacked vector is fetched, the NMI is considered serviced and the NMI edge MUST be cleared so that the line must be de-asserted and re-asserted before another NMI fires.
- **FR-023**: The CPU core MUST expose an `RDY` input with faithful NMOS 6502 semantics. When `RDY` is asserted (low) at the start of a **read** bus cycle, the CPU MUST repeat that read cycle on the next tick without advancing any internal architectural state (registers, flags, PC, instruction phase). When `RDY` is asserted at the start of a **write** bus cycle, the CPU MUST proceed with the write normally (the NMOS-6502 RDY-only-stalls-reads limitation). `RDY` is level-sensitive: the host re-evaluates it on every bus cycle. The BBC Model B's 1 MHz/2 MHz peripheral alignment is therefore implemented in the host layer by holding `RDY` low for the appropriate number of cycles when a Sheila address is on the bus; the CPU core itself contains no BBC-specific clock logic.
- **FR-007**: The CPU core MUST expose a public state API allowing callers to read and write every architectural register (`A`, `X`, `Y`, `SP`, `PC`, `P`) and to read the current cumulative cycle count.
- **FR-008**: The CPU core MUST expose a control API allowing callers to: (a) assert RESET, (b) assert IRQ (level-sensitive), (c) assert NMI (edge-triggered), (d) drive the `RDY` line (level-sensitive, see FR-023), (e) step exactly one bus cycle (the sub-cycle primitive required by FR-018), (f) step exactly one instruction (a convenience that internally runs whole-instruction worth of bus cycles), and (g) run until a caller-specified cycle budget is exhausted. The bus-cycle step MUST be the foundational primitive; instruction-step MUST be implemented on top of it so that bus traffic remains observable.
- **FR-009**: On RESET, the CPU core MUST load `PC` from the reset vector at `$FFFC/$FFFD`, set the `I` flag, leave the other state in its documented post-reset condition (`A`, `X`, `Y` are unspecified by the hardware spec — the core MUST document its chosen behaviour), and consume the documented reset cycles before executing the first instruction.
- **FR-010**: On IRQ entry (when `I` is clear), the CPU core MUST push `PCH`, `PCL`, then `P` (with the `B` bit clear in the pushed copy and the unused bit set), set `I`, load `PC` from `$FFFE/$FFFF`, and consume 7 cycles. With `I` set, IRQ MUST be ignored.
- **FR-011**: On NMI, the CPU core MUST push `PCH`, `PCL`, then `P` (with the `B` bit clear in the pushed copy), set `I`, load `PC` from `$FFFA/$FFFB`, and consume 7 cycles. NMI MUST be edge-triggered: a single transition causes one service; the line must be de-asserted before the next assertion can fire.
- **FR-012**: `BRK` MUST push `PC+2` (not `PC+1`), push `P` with the `B` bit *set* in the pushed copy, set `I`, vector through `$FFFE/$FFFF`, and consume 7 cycles. `RTI` MUST restore the pushed status byte (ignoring the `B` and unused bits in the live register per architecture) and the pushed PC and resume.
- **FR-013**: The CPU core MUST reproduce the NMOS 6502 indirect-JMP page-boundary bug: `JMP ($xxFF)` reads the high byte of the target from `$xx00`, not `$(xx+1)00`.
- **FR-014**: All zero-page indexed addressing modes (`Zero Page,X`, `Zero Page,Y`, `(Indirect,X)`, and the pointer-fetch half of `(Indirect),Y`) MUST wrap the effective address calculation within the zero page (modulo 256).
- **FR-015**: Stack push/pull operations MUST wrap `SP` modulo 256 and always address `$0100 + SP`.
- **FR-016**: Decimal mode: when the `D` flag is set, `ADC` and `SBC` MUST perform Binary-Coded-Decimal arithmetic per Bruce Clark's NMOS 6502 decimal-mode algorithm. `C` MUST reflect the BCD carry/borrow. `N`, `V`, and `Z` MUST be derived from the **binary (pre-BCD-correction)** result — matching real NMOS hardware behaviour as verified by Klaus Dormann's `6502_decimal_test`. Cycle counts for `ADC`/`SBC` in decimal mode MUST be identical to their binary-mode cycle counts (the extra-cycle behaviour is a 65C02 feature and is out of scope per Assumption A1).
- **FR-017**: The CPU core MUST be importable as a self-contained Go package whose only required collaborator is the host-supplied memory interface — no global state, no required side-effects on construction beyond what the caller drives.
- **FR-018**: The CPU core MUST be **sub-cycle / bus-cycle accurate**. Every read and write a real NMOS 6502 would perform — including instruction fetch, operand fetch, indirect-pointer fetch, both halves of a 16-bit pointer read, dummy reads on indexed addressing with page crossing, dummy reads/writes on read-modify-write instructions, stack pushes/pulls, and vector fetches during RESET/IRQ/NMI/BRK — MUST be issued to the host memory interface on the exact cycle a real 6502 would issue it, in the documented order. The host MUST be able to drive the CPU one bus cycle at a time and observe the resulting access (or absence of access) on that cycle. Per-instruction cycle counts MUST be byte-for-byte and tick-for-tick consistent with the sub-cycle timing.
- **FR-019**: On encountering an undocumented NMOS opcode byte, the CPU MUST treat it as a single-byte, 2-cycle `NOP`: `PC` advances by 1, the cumulative cycle count advances by 2, no register or flag is modified, and no operand bytes are consumed. In addition, if the host has registered an "illegal-opcode" notification hook, the CPU MUST invoke that hook with the value of `PC` *at which the opcode was fetched* and the opcode byte itself. The hook is purely observational — its return value does not influence CPU behaviour — and it MUST NOT be invoked if no hook has been registered (zero cost in production).
- **FR-020**: Every public type, function, and method exported by the CPU package MUST carry a doc comment that states its purpose, inputs, outputs, and — where applicable — error or panic conditions, satisfying the project's Code Quality principle. (Most identifiers will have no error mode: the package's only errors arise in `New` for constructor-time validation; the `Memory` interface is infallible per `research.md` §2.)

### Key Entities

- **CPU**: the architectural processor — owns the registers (`A`, `X`, `Y`, `SP`, `PC`, `P`), the cumulative cycle counter, and the pending interrupt lines (`RESET`, `IRQ`, `NMI`). Holds a reference to a memory interface but never to a concrete memory implementation.
- **Memory / Address Bus**: a host-supplied capability that maps a 16-bit address to an 8-bit value for both reads and writes. The CPU treats it as opaque; the host is free to implement flat RAM (for the CPU's own test suite), banked ROM, or fully decoded BBC SHEILA I/O.
- **Instruction**: the static description of one of the 151 NMOS opcodes — its mnemonic, addressing mode, base cycle cost, and the conditions under which it consumes extra cycles. Used by the decoder/dispatcher and by any disassembly the package exposes for debugging.
- **Addressing Mode**: the rule for computing the effective address (or operand) for an instruction. One of the 13 NMOS modes. Each mode has a documented byte length, base cycle cost contribution, and (where applicable) page-cross extra-cycle rule.
- **Processor Status (`P`)**: an 8-bit register with bit positions `N V - B D I Z C`. The `B` bit is a software-visible signal of how the status was pushed (BRK vs IRQ/NMI), not a real flag; the unused bit (bit 5) is conventionally read as 1.
- **Interrupt Vector Table**: the three fixed locations `$FFFA/$FFFB` (NMI), `$FFFC/$FFFD` (RESET), `$FFFE/$FFFF` (IRQ/BRK) in the address space the CPU reads on the corresponding event.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: The CPU core passes Klaus Dormann's `6502_functional_test` and `6502_decimal_test` end-to-end with zero failed sub-tests, with each test image loaded into a flat 64 KB memory and the CPU run to the documented success trap.
- **SC-002**: For every one of the 151 documented NMOS opcodes, the project test suite contains at least one test case that asserts (a) correct effect on `A`, `X`, `Y`, memory, and `SP` where applicable; (b) correct flag updates for every flag the instruction documents; (c) correct cycle count including any conditional extras. Coverage of opcode-times-flag-effect combinations is ≥ 95%.
- **SC-003**: All 13 NMOS addressing modes are exercised by at least one test per mode that asserts the effective address computed matches a hand-derived reference, including the page-cross, zero-page-wrap, indirect-JMP-bug, and `(IND),Y` page-cross edge cases.
- **SC-004**: Delta line coverage on the new CPU package is ≥ 80% (constitution's Testing Standards floor), and the package has no flaky tests over 100 consecutive CI runs.
- **SC-005**: The CPU package is consumable as an isolated Go module: a consumer can import it, supply a minimal memory implementation, and run a non-trivial program (≥ 1000 instructions) without depending on anything else in the GoBeeb codebase.
- **SC-006**: For a representative BBC workload (e.g. running OS 1.20 to the BASIC prompt, once the wider emulator integrates this core), the CPU sustains at least 4 × real-BBC speed (≥ 8 MHz effective 6502 throughput). Concretely: average ≤ 125 ns per emulated bus cycle on Linux amd64 with `GOAMD64=v3`, measured by `go test -bench` over a ≥ 10-second run with zero allocations per emulated cycle (`-benchmem` reports `0 B/op, 0 allocs/op`).
- **SC-007**: The CPU's observable behaviour (final register state, final flag state, total cycle count) on every test program in the suite is byte-for-byte and cycle-for-cycle deterministic across repeated runs and across supported host platforms.
- **SC-008**: For every NMOS opcode/addressing-mode combination, the bus trace produced by the CPU (the ordered sequence of `<cycle, address, read|write, value>` tuples) matches a reference bus trace derived from the published NMOS 6502 cycle-by-cycle behaviour. Verified by a recording mock memory and a golden-trace test set.

## Assumptions

- **A1 (variant)**: The target is the NMOS 6502 as fitted to the BBC Model B (e.g. Synertek/Rockwell SY6502A). 65C02 / 65816 instructions and CMOS bug-fixes are explicitly out of scope. The wider emulator MAY one day need a 65C02 mode (BBC Master), but v1 is NMOS only.
- **A2 (clock speed agnostic)**: The CPU core counts cycles but does not own a real-time clock or sleep. The wider emulator is responsible for pacing execution to 2 MHz (or whatever speed it chooses); the CPU only guarantees that the cycle counts it reports match a real 6502's.
- **A3 (undocumented opcodes)**: Undocumented / "illegal" NMOS opcodes are out of scope for v1. BBC MOS 1.20 and most commercial BBC software use only documented opcodes; the small minority that uses illegal opcodes will not work and that is an accepted v1 limitation. FR-019 still requires the core's behaviour on such bytes to be deterministic and documented.
- **A4 (cycle granularity — resolved)**: FR-018 is resolved as **sub-cycle / bus-cycle accurate**. FR-006's memory-access ordering is therefore a hard cycle-level requirement: bus traffic the CPU emits is observable on the exact cycle a real 6502 would emit it. This is the accuracy floor required to reproduce BBC video-ULA bus contention (the CPU and ULA share the bus on alternate 2 MHz/1 MHz cycles in MODE 0–3), cycle-stuffing tricks, and timing-sensitive copy protection.
- **A5 (no separate decimal-mode disable)**: The 6502 supports BCD; the CPU core implements it. The wider BBC emulator does *not* need to mask BCD out (unlike the NES, which uses a 6502 variant with BCD disabled).
- **A6 (host owns I/O)**: The CPU core does not implement any BBC-specific behaviour — no SHEILA, no VIA, no ACIA, no video ULA. Those live in the host memory implementation that the wider emulator will supply.
- **A7 (RESET post-state — resolved)**: NMOS hardware leaves `A`, `X`, `Y` undefined on RESET. The core picks the following deterministic post-RESET state (chosen for compatibility with Klaus Dormann's `6502_functional_test` and for clean test boundaries — see `research.md` §1): `A = X = Y = $00`, `SP = $FD` (matches real NMOS three-cycle fake-push), `P = $24` (`I` set, unused bit set, `B` clear, `D` left at construction-time value), `PC = mem[$FFFC] | mem[$FFFD]<<8`, cumulative cycle counter reset to `0` immediately *after* the 7-cycle RESET sequence.
- **A8 (single CPU, no multicore)**: There is exactly one 6502 in a BBC Model B and exactly one in the emulator. The CPU package is not designed for concurrent use from multiple goroutines on the same instance; safe sharing across goroutines is the host's responsibility.
- **A9 (reference document)**: Where the Obelisk reference and Klaus Dormann's functional test disagree on a corner case, the functional test wins (it is the de-facto behaviour-of-real-silicon reference).
