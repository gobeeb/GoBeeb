# Implementation Plan: CPU Bus-Cycle Validation (Tom Harte ProcessorTests)

**Branch**: `003-cpu-processor-tests` | **Date**: 2026-05-25 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/003-cpu-processor-tests/spec.md`

## Summary

Drop the upstream [SingleStepTests/ProcessorTests](https://github.com/SingleStepTests/ProcessorTests) JSON corpus on top of the existing `mos6502/` core and assert per-cycle bus equivalence with a real NMOS 6502 across all 151 documented opcodes (10,000 cases each, ~1.51M assertions). The harness is test-only: zero new public API on `mos6502`, no new runtime dependencies, no source code changes to the core unless a documented-opcode divergence is uncovered. The corpus is fetched on demand via `go generate` (`git clone --depth 1` of a pinned SHA + sparse-checkout of `6502/v1/`), gitignored, and never committed.

Two run modes share the same harness: default `go test ./mos6502/` runs the full 10,000 cases per opcode (≤ 5 min on amd64); `go test -short ./mos6502/` runs the first 100 cases per opcode (~15,100 total) for CI / dev-loop feedback. The 105 undocumented NMOS opcodes are explicitly skipped via a `map[uint8]string` constant in test source, each entry annotated with its undocumented mnemonic — deferred to a future "implement undocumented opcodes" phase.

## Technical Context

**Language/Version**: Go 1.22+ (toolchain pinned by repo; mise-managed)

**Primary Dependencies**: stdlib only (`encoding/json`, `testing`, `os`, `path/filepath`, `bytes`, `fmt`). No new runtime deps. Generator step shells out to `git` (system tool).

**Storage**: filesystem — corpus lives at `mos6502/testdata/processortests/6502/v1/*.json`, gitignored. ~hundreds of MB unpacked.

**Testing**: `go test`. New file `mos6502/processortests_test.go` + new generator `mos6502/gen.go` (build-tag isolated so it doesn't enter normal compilation). Reuses existing `mos6502.Memory`, `mos6502.CPU.Step`, `mos6502.CPU.Registers`/`SetRegisters`, `mos6502.Trace`/`BusEvent`/`Snapshot`.

**Target Platform**: any host that compiles `mos6502/` (Linux/macOS/Windows on amd64/arm64). Generator step requires POSIX `git` in PATH.

**Project Type**: emulator core library; this phase adds a validation harness only — no binary, no service, no UI.

**Performance Goals**:
- Full corpus run (`go test ./mos6502/`): ≤ 5 minutes on typical amd64 dev machine, default GOMAXPROCS, per-opcode `t.Parallel()`.
- `-short` run: comparable to current `mos6502/` short-mode runtime (single-digit seconds).
- Existing zero-allocation hot path on `CPU.Step` is untouched and MUST stay within the ≤ 125 ns/cycle SC-006 budget post-phase.

**Constraints**:
- Zero new public API on `mos6502` (FR-010).
- No new runtime deps.
- Test code MUST NOT introduce hidden state that would alter the behaviour of pre-existing tests (`functional_test.go`, `golden_trace_test.go`, etc.) running in the same `go test` invocation.
- Corpus is gitignored (FR-005); a fresh clone must not contain `testdata/processortests/`.

**Scale/Scope**:
- 151 documented opcodes × 10,000 cases = ~1.51M case-level assertions on full run.
- 151 documented opcodes × 100 cases = ~15,100 on `-short` run.
- 105 documented + 105 undocumented + 46 unassigned = wait — actually 151 + 105 = 256, no unassigned (every byte is either documented or one of the 105 illegals). Skip-list contains exactly 105 entries.
- Test code volume: estimate ~400 lines across `processortests_test.go` + `gen.go` + a small fixture helper.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### I. Code Quality
- **PASS planned**: New files (`processortests_test.go`, `gen.go`) will go through the same `make fmt vet lint test bench cover` gate as the rest of `mos6502/`. The harness is one `TestProcessorTests` driver + one sparse-memory adapter + one JSON case struct + one skip-list constant — each below complexity 10. Doc comments on the file-level driver explain JSON schema source-of-truth and skip-list governance.

### II. Testing Standards (NON-NEGOTIABLE)
- **PASS planned, with explicit note**: This phase IS tests. There is no "new feature code" requiring its own unit-test coverage — the harness's correctness is exercised by every JSON case it consumes and is double-checked by SC-006 (deliberate single-cycle perturbation must fail). FR-009 forbids `mos6502/` line coverage regression below 99.3%; harness code lives in `_test.go` and does not enter the coverage denominator. The skip-list constant gets a sanity unit test (length == 105, no overlap with documented opcodes, all keys are NMOS-illegal). The sparse-memory adapter gets a focused unit test (round-trip, default-zero for unmapped reads).

### III. User Experience Consistency
- **N/A**: No user-facing surface introduced. Test output formatting follows Go's standard `t.Errorf` / `t.Fatalf` conventions used elsewhere in `mos6502/`.

### IV. Performance Requirements
- **PASS planned**: SC-008 declares the full-run wall-time budget (≤ 5 min) up front. Per-opcode `t.Parallel()` is the parallelism plan. The harness adds zero overhead to `CPU.Step` itself (sparse-memory adapter is allocated outside the timed cycle path; trace is reused per-case via `Trace.Reset()`). Existing `BenchmarkInstrMix` etc. will be re-run post-implementation to confirm no regression on the ≤ 125 ns/cycle SC-006 budget for the core.

**Verdict**: All four principles compatible. No Complexity Tracking entries required.

## Project Structure

### Documentation (this feature)

```text
specs/003-cpu-processor-tests/
├── plan.md                       # This file
├── spec.md                       # Already exists
├── research.md                   # Phase 0 output
├── data-model.md                 # Phase 1 output
├── quickstart.md                 # Phase 1 output
├── contracts/
│   ├── corpus-schema.md          # JSON case shape contract with upstream
│   └── generator-contract.md     # go generate / fetch tool contract
├── checklists/
│   └── requirements.md           # Already exists
└── tasks.md                      # Phase 2 — created by /speckit-tasks
```

### Source Code (repository root)

```text
mos6502/
├── …existing files unchanged…
├── gen.go                        # NEW: //go:generate directive + fetch entrypoint (build-tag gated)
├── processortests_test.go        # NEW: harness — corpus loader, sparse adapter,
│                                 #      skip list, per-opcode parallel subtests
└── testdata/
    ├── 6502_functional_test.bin  # Existing
    ├── LICENSE_KLAUS_DORMANN.txt # Existing
    └── processortests/           # NEW, gitignored
        └── 6502/
            └── v1/
                ├── 00.json
                ├── 01.json
                └── …             # 256 files, one per opcode byte
```

Plus two repo-level touches:
- `.gitignore` — add `mos6502/testdata/processortests/`
- `docs/roadmap.md` — flip Phase 003 status from 🟡 Planned to ✅ Complete on phase exit (out of plan scope — owned by the phase-completion commit)

**Structure Decision**: Pure-additive layout. Tests + generator live alongside the existing `mos6502/` files; testdata extends the existing `testdata/` directory. No new packages, no new directories outside `mos6502/`. The fetcher script (`mos6502/gen.go`) is a tiny self-contained Go program guarded by `//go:build ignore` (so it doesn't enter normal package compilation) and invoked via `go run gen.go` from a `//go:generate` directive at the top of `processortests_test.go`.

## Complexity Tracking

> No constitution violations to justify. Section intentionally empty.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| (none)    | —          | —                                    |
