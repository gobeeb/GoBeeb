# Feature Specification: CPU Bus-Cycle Validation (Tom Harte ProcessorTests)

**Feature Branch**: `003-cpu-processor-tests`

**Created**: 2026-05-25

**Status**: Draft

**Input**: User description: "Phase 003 — validate cycle-accurate bus behaviour of every documented NMOS 6502 opcode against the SingleStepTests/ProcessorTests corpus (10,000 JSON cases per opcode, ~1.51M tests across the 151 documented opcodes). Closes the validation gap left by the Klaus Dormann ROM, which only checks final state and cannot see per-cycle read/write order."

## Clarifications

### Session 2026-05-25

- Q: Corpus fetch mechanism for `go generate`? → A: `git clone --depth 1` of the pinned SHA with sparse-checkout of `6502/v1/`. Single tool dependency (`git`), native SHA verification by checkout, no GitHub API rate limits.
- Q: `-short` mode sampling depth per documented opcode? → A: 100 cases per opcode (15,100 total). Stronger regression-detection across operand/flag variance than a smoke sample, at a few-second CI cost.
- Q: Behaviour when `testdata/processortests/` is absent at test time? → A: Hard-fail (`t.Fatal`) with a message naming the missing path and the `go generate ./mos6502/` remediation command. Silent skip would defeat the validation purpose.
- Q: Full-corpus invocation mechanism? → A: Default `go test ./mos6502/` runs the full 10,000-case-per-opcode corpus; `-short` runs the 100-case sample. No new build tag. Matches the existing Klaus Dormann functional-ROM pattern.
- Q: Full-run wall-time upper bound? → A: ≤ 5 minutes on a typical amd64 developer machine using `go test ./mos6502/` with default GOMAXPROCS and per-opcode `t.Parallel()`. Realistic for nightly CI; allows naive per-case JSON parsing without optimisation work.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Validate documented opcodes against per-cycle ground truth (Priority: P1)

A GoBeeb maintainer needs cryptographic-grade confidence that the `mos6502/` core is not just functionally correct on final state, but emits the **exact** sequence of bus reads and writes that a real NMOS 6502 chip emits — in the right order, on the right cycles, against the right addresses. They run the new test suite against the upstream SingleStepTests/ProcessorTests corpus and the result is binary: every documented opcode either matches a real chip cycle-for-cycle or it does not.

**Why this priority**: This is the entire phase. Without per-cycle bus validation, downstream phases (Video ULA contention, Phase 004; 1 MHz bus alignment, Phase 005) build on an unverified foundation. Bugs in CPU bus order are the single highest-impact defect class for an emulator because they corrupt every subsystem that synchronises on the bus. This must land before any cycle-sensitive peripheral work begins.

**Independent Test**: A maintainer runs `go generate ./mos6502/` to fetch the pinned corpus, then `go test ./mos6502/ -run TestProcessorTests`, and observes pass/fail per opcode. The suite delivers value standalone — it neither needs nor changes any other package.

**Acceptance Scenarios**:

1. **Given** the pinned corpus has been fetched and the `mos6502/` core is unmodified, **When** the maintainer runs the full suite, **Then** all 151 documented opcodes pass 10,000 cases each and the suite reports `PASS`.
2. **Given** a regression is introduced into the CPU core (e.g., an addressing-mode read order is silently swapped), **When** the suite runs, **Then** the affected opcode's subtest fails with a diff that names the differing cycle index, the expected `{addr, value, kind}`, and the observed `{addr, value, kind}`.
3. **Given** the maintainer is on a developer machine and wants quick feedback, **When** they run `go test -short ./mos6502/`, **Then** the suite executes a representative sampled subset of cases and completes in CI-acceptable time without sacrificing coverage of every documented opcode.

---

### User Story 2 - Reproducibly fetch the upstream corpus (Priority: P2)

A maintainer (or a fresh CI runner) needs to obtain the ~1.51M-case corpus on demand without bloating the GoBeeb repository. They invoke `go generate ./mos6502/` and the corpus is fetched, at a pinned upstream commit SHA, into a local `testdata/` directory that is gitignored. Re-running the generator is a no-op when the corpus is already present at the correct SHA.

**Why this priority**: The test suite is worthless if the corpus cannot be obtained reliably and reproducibly. Pinning the SHA means a future upstream change to the corpus cannot silently re-flake GoBeeb's CI. Gitignoring keeps repository size sane (the corpus is ~hundreds of MB unpacked).

**Independent Test**: From a clean checkout with no `testdata/processortests/` directory, run `go generate ./mos6502/` and confirm the corpus appears at the pinned SHA; re-run and confirm no work is done.

**Acceptance Scenarios**:

1. **Given** `mos6502/testdata/processortests/` does not exist, **When** the maintainer runs `go generate ./mos6502/`, **Then** the directory is populated with `6502/v1/*.json` from the pinned upstream commit and no other files are touched.
2. **Given** the corpus is already present at the pinned SHA, **When** the maintainer re-runs `go generate ./mos6502/`, **Then** the command exits cleanly without re-downloading anything.
3. **Given** the corpus is present but at the wrong SHA (e.g., a prior partial fetch), **When** the maintainer re-runs the generator, **Then** the corpus is re-fetched to match the pinned SHA, or the discrepancy is reported clearly enough that the maintainer can resolve it.
4. **Given** a project member inspects `git status` after running the generator, **When** they look for new files, **Then** the corpus directory does not appear because it is gitignored.

---

### User Story 3 - Explicitly defer illegal opcodes without poisoning the suite (Priority: P3)

A maintainer knows the current `mos6502/` core treats the 105 undocumented NMOS opcodes as 2-cycle NOP stubs and would fail Tom Harte expectations for those. They need the corpus harness to skip those opcodes by an explicit, auditable allowlist rather than by silently ignoring failures, so a future "implement undocumented opcodes" phase can remove the skip entries one at a time and watch them go green.

**Why this priority**: Defining the boundary of this phase precisely. Without explicit skip handling, either the suite fails (and hides real regressions in documented opcodes under noise), or the suite passes by accident (and stays passing if a documented opcode silently regresses to NOP behaviour). The skip list is the contract between this phase and the future undocumented-opcode phase.

**Independent Test**: Inspect the skip list constant in the test source; confirm exactly 105 entries, each cross-referenced to its undocumented mnemonic. Remove any one entry locally and confirm the suite then fails for that opcode (proving the harness was actually running it, not silently skipping a different way).

**Acceptance Scenarios**:

1. **Given** the skip list contains all 105 illegal NMOS opcodes, **When** the suite runs, **Then** documented opcodes are tested and undocumented opcodes are reported as skipped (not passed, not failed).
2. **Given** a maintainer removes an entry from the skip list, **When** the suite runs, **Then** that opcode is exercised against its 10,000 cases and reports a real pass/fail result.
3. **Given** a future contributor adds an undocumented opcode implementation, **When** they remove the matching skip entry and run the suite, **Then** the suite either confirms their implementation matches a real chip or pinpoints the divergence.

---

### Edge Cases

- A JSON case sets the initial PC to fetch from an unmapped sparse-memory address — the sparse adapter MUST return `$00` (consistent with `initial.ram` being authoritative; addresses absent from `initial.ram` are implicitly zero unless the test depends otherwise).
- A documented opcode produces an extra or missing bus cycle versus the corpus — the failure message MUST localise the divergence to the specific cycle index, not just say "trace differs".
- The pinned upstream SHA is deleted or rewritten on GitHub — the generator MUST fail loudly with a clear error pointing the maintainer at the pinned SHA constant, not silently fetch a different commit.
- The corpus is partially downloaded (network drop) — re-running the generator MUST recover cleanly without leaving the test suite in a state where some opcodes silently skip due to missing input files.
- A documented opcode's RMW double-write semantics differ between the corpus's expectation and the existing core — the failure MUST be expressed in terms of the bus event sequence, not as a register diff, so the maintainer can diagnose without guessing.
- Running the full suite on a constrained machine takes too long — the `-short` mode MUST still exercise every documented opcode (sampled), not skip whole opcodes, so regressions cannot hide behind sampling.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The CPU core MUST be validated against the SingleStepTests/ProcessorTests corpus (`6502/v1/*.json`) for all 151 documented NMOS 6502 opcodes, with 10,000 cases each (~1.51M total cases).
- **FR-002**: Each test case MUST initialise the CPU with the `initial` registers and RAM from the JSON case, execute exactly one `Step()`, and assert (a) `Registers()` matches `final`, (b) RAM mutations match `final.ram`, and (c) the bus-cycle trace exactly matches the `cycles` array (count and each `{addr, value, kind}`).
- **FR-003**: The harness MUST emit one Go subtest per opcode (e.g., `0xA9_LDA_imm`) and run subtests in parallel where safe, so a single failing opcode produces a localised failure message rather than aborting the suite.
- **FR-004**: The suite MUST skip exactly the 105 undocumented NMOS opcodes via an explicit, auditable constant in the test source, with each entry traceable to its undocumented mnemonic; skipped opcodes MUST be reported as skipped, not as passes.
- **FR-005**: The corpus MUST NOT be committed to the GoBeeb repository; the local `testdata/processortests/` directory MUST be gitignored.
- **FR-006**: A `go generate` directive MUST be present in the `mos6502/` package which, when invoked, fetches the corpus from the pinned upstream commit SHA into `mos6502/testdata/processortests/` using `git clone --depth 1` of the pinned SHA combined with sparse-checkout of the `6502/v1/` subtree. The only required external tool is `git`. The fetch operation MUST be idempotent (re-running with the corpus already present at the pinned SHA is a no-op) and MUST pin the upstream commit SHA in source so the corpus version is reproducible across machines and across time. SHA verification is performed natively by the checkout; if the pinned SHA cannot be reached or checked out, the generator MUST fail with a clear error referencing the pinned-SHA constant.
- **FR-007**: The default invocation (`go test ./mos6502/`) MUST run the full corpus (all 10,000 cases per documented opcode). The `-short` flag MUST run exactly 100 cases per documented opcode (~15,100 cases total) in CI-acceptable time while still exercising every documented opcode (no whole-opcode skipping under `-short`). No additional build tag is introduced for either mode.
- **FR-008**: Failures MUST localise to the divergent cycle index when the trace mismatches, naming both expected and observed `{addr, value, kind}` for that cycle.
- **FR-009**: Pre-existing `mos6502/` validation surface (Klaus Dormann functional ROM, golden bus traces, unit tests covering addressing modes, BCD, stack wrap, interrupts, NMI hijack, illegal-opcode hook, RDY gating) MUST continue to pass; coverage on `mos6502/` MUST NOT regress below the current 99.3%.
- **FR-010**: The new test harness MUST NOT introduce any new public API on the `mos6502` package; it MUST consume only the existing exports (`Memory`, `CPU.SetRegisters`/`Registers`, `Trace`/`BusEvent`/`Trace.Snapshot`, `CPU.Step`).
- **FR-011**: The maintainer-facing instructions for fetching and running the corpus MUST be documented in the `mos6502/` quickstart or README, including the pinned SHA reference and how to switch between `-short` and full runs.
- **FR-012**: When `mos6502/testdata/processortests/6502/v1/` is absent at test time, the harness MUST hard-fail (`t.Fatal`) with a message naming the missing path and the `go generate ./mos6502/` remediation command. The harness MUST NOT skip silently and MUST NOT auto-fetch.

### Out of Scope

- **OOS-001**: Implementing any of the 105 undocumented NMOS opcodes (LAX, SAX, DCP, ISB, RLA, RRA, SLO, SRE, ANC, ARR, ASR, LAS, XAA, AHX, SHX, SHY, TAS, KIL). These are explicitly skipped in this phase and tracked as a future phase.
- **OOS-002**: 65C02 or 65816 variants. The corpus subdirs for those CPUs are not consumed in this phase.
- **OOS-003**: RESET / IRQ / NMI sequences. These are not part of the Tom Harte single-step corpus and remain covered by the existing `interrupts_test.go`.

### Key Entities

- **Test Case (JSON)**: One entry in a corpus file. Carries `name`, an `initial` state (`pc`, `s`, `a`, `x`, `y`, `p`, `ram` as `[addr, value]` pairs), a `final` state in the same shape, and a `cycles` array of `[addr, value, "read"|"write"]` triples.
- **Sparse Memory Adapter**: A test-only `Memory` implementation backed by a sparse address-to-byte map, populated from a case's `initial.ram`. Returns `$00` for unmapped reads; records writes back into the map so post-run RAM mutations can be compared against `final.ram`.
- **Bus Cycle Trace**: A sequence of `BusEvent` records (each carrying address, data byte, and kind = read or write). Already produced by the existing `mos6502.Trace`; this phase asserts its content matches the corpus `cycles` array verbatim.
- **Skip List**: A constant in the test source enumerating the 105 undocumented opcodes deferred from this phase, each annotated with its undocumented mnemonic.
- **Pinned Corpus SHA**: A constant in the test source naming the exact upstream commit of `SingleStepTests/ProcessorTests` that the suite was authored against. Drives reproducibility of the generator and locks the test oracle to a known revision.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All 151 documented NMOS 6502 opcodes pass their 10,000 corpus cases (~1.51M passing assertions in aggregate) on the unmodified `mos6502/` core.
- **SC-002**: Exactly 105 opcodes (the undocumented set) are reported as skipped, not as passes or failures; total documented + skipped + (none other) = 256 opcodes.
- **SC-003**: `mos6502/` test coverage stays at or above its current 99.3% after the new suite lands; no pre-existing test regresses.
- **SC-004**: A clean checkout reaches a green `go test ./mos6502/` run in two commands (`go generate ./mos6502/` then `go test ./mos6502/`), with no manual fetching, unpacking, or environment configuration beyond network access on the generator step.
- **SC-005**: `go test -short ./mos6502/` completes in a time budget compatible with CI (comparable to the existing `mos6502/` short-mode runtime) while still exercising every one of the 151 documented opcodes through a 100-case sample.
- **SC-006**: When a deliberate single-cycle perturbation is injected into a documented opcode (test harness self-check), the suite fails for exactly that opcode and the failure message names the divergent cycle index along with expected and observed bus events.
- **SC-007**: A fresh runner with no prior corpus data can reproduce the same pass/skip counts as any other runner because the corpus is pinned to a specific upstream commit SHA recorded in source.
- **SC-008**: `go test ./mos6502/` (full corpus, no `-short`) completes within 5 minutes on a typical amd64 developer machine using default GOMAXPROCS and per-opcode `t.Parallel()`.

## Assumptions

- The upstream `SingleStepTests/ProcessorTests` repository remains available on GitHub at a stable URL and the pinned commit SHA remains reachable for the duration of this phase. If GitHub mirroring is needed, that is a separate concern tracked outside this phase.
- The corpus directory naming and JSON schema as documented upstream (`6502/v1/*.json` with the `{name, initial, final, cycles}` shape) is treated as stable for the pinned SHA. Schema changes in unpinned future revisions are not this phase's concern.
- The current `mos6502/` core, having passed the Klaus Dormann ROM and the existing 91-test unit suite, is expected to pass the Tom Harte corpus for documented opcodes without significant rework; any divergences uncovered are real CPU bugs to be fixed under this phase's scope (a fix is in scope; an opcode rewrite is not unless required for documented behaviour).
- Re-running `go generate` requires network access; the suite itself, once the corpus is present, MUST run fully offline (no network at test time).
- The sparse-memory adapter is acceptable to allocate per test case; per-case heap activity in tests is not subject to the production zero-allocation budget on the hot path (the production budget applies to `CPU.Step` itself, which is what is being measured, not the test scaffolding around it).
- Full-run wall-time target (SC-008, ≤ 5 minutes) assumes a "typical amd64 developer machine" — a modern multi-core x86-64 box with NVMe storage. Runs on slower hardware (constrained CI tiers, ARM laptops, spinning disks) may exceed the budget; mitigations (smaller sampling under `-short`) are already provided for those environments.
- The 151 / 105 opcode partition reflects the standard documented-vs-undocumented NMOS 6502 split (151 documented + 105 undocumented = 256 total). If the upstream corpus disagrees on count for the pinned SHA, the skip list MUST be reconciled with the corpus's actual file set rather than with the textbook number.
