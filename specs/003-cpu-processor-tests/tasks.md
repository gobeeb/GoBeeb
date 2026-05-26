---
description: "Task list for Phase 003 — CPU Bus-Cycle Validation (Tom Harte ProcessorTests)"
---

# Tasks: CPU Bus-Cycle Validation (Tom Harte ProcessorTests)

**Input**: Design documents from `/specs/003-cpu-processor-tests/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Phase 003 IS tests. There is no production code under test here — the harness, generator, and skip list constitute the deliverable. Self-checks (unmarshal round-trip, skip-list invariants, sparse-adapter round-trip, deliberate-perturbation self-check) are included as harness validation, not as optional add-ons.

**Organization**: Tasks grouped by user story (US1, US2, US3) to enable independent slices. Story dependency note: the spec's "independent test" for US1 invokes US2 (`go generate` then `go test`) and US1's per-case loop consults the `skipList` produced by US3. The harness *code* for each story is authorable in any order; the green run of US1 requires US2 corpus + US3 skip list in place. MVP scope (US1) therefore ships only when all three slices land.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: different file, no dependency on incomplete tasks → can run in parallel
- **[Story]**: which user story (US1, US2, US3)
- File paths are exact

## Path Conventions

Single Go module at repo root (`github.com/gobeeb/GoBeeb`). New code lives in `mos6502/`:

- `mos6502/processortests_test.go` — NEW harness (one file, all `_test.go` symbols)
- `mos6502/gen.go` — NEW `//go:build ignore` fetcher
- `mos6502/testdata/processortests/` — NEW gitignored corpus root

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Repo-level wiring and reproducibility pin discovery. No code yet.

- [X] T001 Add line `mos6502/testdata/processortests/` to `.gitignore` at repo root (FR-005, US2 acceptance scenario 4)
- [X] T002 Discover pinned upstream commit SHA: run `git ls-remote https://github.com/SingleStepTests/ProcessorTests.git refs/heads/master`, capture the 40-char hex SHA; record it in the implementation commit body and in the planning notes for use as `pinnedCorpusSHA` in T015 + T022 (R1)
- [X] T003 [P] Confirm `git --version` present on dev shell PATH (quickstart prereq; smoke check, no file output)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Capture the pre-phase baseline so regression gates (FR-009, SC-003) have a reference point.

**⚠️ CRITICAL**: No story work changes the baseline numbers until this is captured.

- [X] T004 Run `make fmt vet lint test bench cover` on the unmodified branch tip and record current `mos6502/` test count (91), pass status (PASS), coverage (≥ 99.3%), and `BenchmarkInstrMix` ns/cycle in the implementation commit body for post-phase comparison (regression gate for FR-009 / SC-003)

**Checkpoint**: Baseline captured — story work can begin.

---

## Phase 3: User Story 1 - Validate documented opcodes against per-cycle ground truth (Priority: P1) 🎯 MVP

**Goal**: Per-cycle bus-equivalence assertion for all 151 documented NMOS 6502 opcodes (10,000 cases each, ~1.51M assertions on full run; 100 cases each on `-short`). Failure messages localise to the divergent cycle index.

**Independent Test**: With corpus present at `mos6502/testdata/processortests/6502/v1/` and skip list populated, run `go test ./mos6502/ -run TestProcessorTests` — all 151 documented opcode subtests report PASS, all 105 undocumented report SKIP, full run completes in ≤ 5 min on amd64 default GOMAXPROCS.

### Implementation for User Story 1

- [X] T005 [P] [US1] Add `processorCase`, `processorState`, `ramEntry`, `busCycle` types (unexported, struct-tagged) plus `(*ramEntry).UnmarshalJSON` and `(*busCycle).UnmarshalJSON` decoding upstream's 2- and 3-tuple JSON-array shapes in `mos6502/processortests_test.go` (data-model E1, contracts/corpus-schema.md)
- [X] T006 [P] [US1] Add `processorState.toRegisters()` helper mapping JSON state → `mos6502.Registers` (Cycles=0) in `mos6502/processortests_test.go` (data-model E1)
- [X] T007 [P] [US1] Add `ramMap` type (`map[uint16]uint8`) implementing `mos6502.Memory.Read`/`Write` with default-zero on unmapped reads, plus `newRAMMap(initial []ramEntry) ramMap` constructor with `make(ramMap, len(initial)*2)` capacity hint in `mos6502/processortests_test.go` (data-model E2)
- [X] T008 [US1] Add `TestRAMMapRoundTrip` covering: populate from `initial.ram`, default-zero on unmapped read, write-then-read round-trip, final-state comparison against `final.ram` (both directions: every `final` entry present, every map key not in `final` is unchanged-or-zero) in `mos6502/processortests_test.go`
- [X] T009 [US1] Add `TestRAMEntryUnmarshal` and `TestBusCycleUnmarshal` covering positive 2- and 3-tuple decode, malformed-input → error, range-overflow → error, case-sensitive `"read"`/`"write"` kind decoding in `mos6502/processortests_test.go` (contracts/corpus-schema.md decoding subtleties)
- [X] T010 [US1] Add `subtestName(op uint8) string` helper producing `0x<HH>_<MNEMONIC>_<addr_mode>` (e.g., `0xA9_LDA_imm`); derive mnemonic + addr-mode from existing `mos6502/opcodes.go` introspection (no parallel hand-list) in `mos6502/processortests_test.go` (quickstart "Running a single opcode")
- [X] T011 [US1] Add `runCase(t *testing.T, c *processorCase, trace *mos6502.Trace)` performing: build `ramMap`, construct `mos6502.New(ramMap)`, `cpu.SetRegisters(c.Initial.toRegisters())`, `cpu.SetTrace(trace)`, `trace.Reset()`, `cpu.Step()`, then assert (a) `cpu.Registers()` matches `c.Final` (ignoring `Cycles`), (b) `ramMap` matches `c.Final.RAM`, (c) `trace.Snapshot()` matches `c.Cycles` exactly in `mos6502/processortests_test.go` (FR-002; data-model "state transitions")
- [X] T012 [US1] Add failure formatter `reportDiff(t *testing.T, c *processorCase, gotRegs, gotRAM, gotCycles ...)` printing the case `name` plus (i) full 6-field register block, (ii) divergent `(addr, got, want)` RAM cells only, (iii) first divergent cycle index with expected and observed `{addr,value,kind}`; uses `t.Errorf` for first 5 failures per subtest then `t.Fatalf("skipped remaining N cases")` in `mos6502/processortests_test.go` (FR-008, SC-006, R8)
- [X] T013 [US1] Add `TestProcessorTests` driver: iterate opcode bytes 0x00–0xFF, skip those present in `skipList` with `t.Run(name, func(t *testing.T) { t.Skip("undocumented: " + mnemonic) })`, otherwise call `t.Run(subtestName(op), func(t *testing.T) { t.Parallel(); ... })` which (a) parses `<HH>.json` once via `os.ReadFile` + `json.Unmarshal` into `[]processorCase`, (b) allocates one `mos6502.NewTrace(16)` per subtest goroutine, (c) iterates cases calling `runCase`, with case count capped at 100 when `testing.Short()` else len in `mos6502/processortests_test.go` (FR-003, FR-007, R3, R4, R7)
- [X] T014 [US1] Add corpus-presence preflight in `TestProcessorTests` before iteration: `os.Stat("testdata/processortests/6502/v1")` — on error, `t.Fatalf("corpus not found at mos6502/testdata/processortests/6502/v1/; run \`go generate ./mos6502/\` to fetch")` in `mos6502/processortests_test.go` (FR-012, quickstart "CI integration")
- [X] T015 [US1] Hex-pad and lowercase the opcode-to-filename mapping (`fmt.Sprintf("%02x.json", op)`) and confirm against an upstream file listing of `6502/v1/` in `mos6502/processortests_test.go` (contracts/corpus-schema.md "Per-opcode file")

**Checkpoint**: US1 harness code complete. Cannot exercise until US2 + US3 land.

---

## Phase 4: User Story 2 - Reproducibly fetch the upstream corpus (Priority: P2)

**Goal**: `go generate ./mos6502/` populates `mos6502/testdata/processortests/6502/v1/` at the pinned SHA from a clean checkout; re-runs are no-ops; partial downloads self-recover.

**Independent Test**: From a clean checkout with no `mos6502/testdata/processortests/` directory, run `go generate ./mos6502/` — directory appears populated with 256 `.json` files plus `.fetched-sha` matching `pinnedCorpusSHA`; re-run prints "already at <SHA>; skipping" and touches nothing; `git status` shows no new tracked files.

### Implementation for User Story 2

- [X] T016 [P] [US2] Create `mos6502/gen.go` with `//go:build ignore` header, `package main`, declared constants `pinnedCorpusSHA = "<SHA from T002>"` and `upstreamRepoURL = "https://github.com/SingleStepTests/ProcessorTests.git"`, plus a doc comment naming this file as the fetcher invoked via `go generate` (data-model E4, contracts/generator-contract.md inputs)
- [X] T017 [US2] Implement `main()` happy-path-fresh sequence in `mos6502/gen.go`: pre-flight check `exec.LookPath("git")` → exit 1 on miss; `os.RemoveAll("testdata/processortests")`; `git init testdata/processortests`; `git -C ... remote add origin <upstreamRepoURL>`; `git -C ... sparse-checkout init --cone`; `git -C ... sparse-checkout set 6502/v1`; `git -C ... fetch --depth 1 origin <pinnedCorpusSHA>`; `git -C ... checkout FETCH_HEAD`; write `testdata/processortests/.fetched-sha` containing `<pinnedCorpusSHA>\n`; print success line (contracts/generator-contract.md "Happy path — fresh")
- [X] T018 [US2] Implement idempotency branch in `mos6502/gen.go`: at startup `os.ReadFile("testdata/processortests/.fetched-sha")`, compare trimmed contents to `pinnedCorpusSHA`; if equal AND `len(filepath.Glob("testdata/processortests/6502/v1/*.json")) >= 200`, print "already at <SHA>; skipping fetch" and `os.Exit(0)` (contracts/generator-contract.md "Happy path — already fetched"; R6 idempotency)
- [X] T019 [US2] Implement partial-download recovery in `mos6502/gen.go`: any mismatch (missing marker, wrong SHA, thin `6502/v1/`) falls through to the fresh-path `RemoveAll`+re-fetch sequence; no clever repair (contracts/generator-contract.md "Partial-download recovery")
- [X] T020 [US2] Implement failure-mode reporting in `mos6502/gen.go`: every `git`/`os` error → `fmt.Fprintln(os.Stderr, ...)` with the underlying error AND a hint naming `pinnedCorpusSHA` for fetch/checkout errors; on checkout failure do NOT write `.fetched-sha`; `os.Exit(1)` (contracts/generator-contract.md "Failure modes" + invariants)
- [X] T021 [US2] Add `//go:generate go run gen.go` directive at the very top of `mos6502/processortests_test.go` (above the package declaration's leading comment, per Go generate semantics) (contracts/generator-contract.md preamble)
- [X] T022 [US2] Declare `const pinnedCorpusSHA = "<SHA from T002>"` mirror in `mos6502/processortests_test.go` with a one-line doc comment noting it duplicates `mos6502/gen.go` for documentation parity; add `TestPinnedSHADocumented` asserting the constant is exactly 40 lowercase hex chars (catches typos / drift) (data-model E4)
- [X] T023 [US2] Manual smoke verification: `rm -rf mos6502/testdata/processortests && go generate ./mos6502/`, confirm `6502/v1/` populated and `.fetched-sha` written; re-run `go generate ./mos6502/`, confirm no-op message; record outcome in implementation commit body (US2 acceptance scenarios 1, 2, 4; SC-004, SC-007)

**Checkpoint**: Corpus fetcher landed. US1 can now run for any opcode not in the skip list.

---

## Phase 5: User Story 3 - Explicitly defer illegal opcodes without poisoning the suite (Priority: P3)

**Goal**: The 105 undocumented NMOS opcodes are reported as SKIP with auditable mnemonics, never as PASS or FAIL. The skip list is the contract between Phase 003 and the future "implement undocumented opcodes" phase.

**Independent Test**: `go test -v ./mos6502/ -run TestProcessorTests` reports exactly 105 SKIP lines (each carrying the undocumented mnemonic) and 151 PASS lines under `TestProcessorTests`; running `TestSkipListInvariants` independently passes (length, no overlap with documented set, non-empty values).

### Implementation for User Story 3

- [X] T024 [P] [US3] Add `var skipList = map[uint8]string{ ... }` to `mos6502/processortests_test.go` enumerating all 105 NMOS undocumented opcode bytes with values `<MNEMONIC>_<addr_mode>` (e.g., `0x02: "KIL"`, `0x07: "SLO_zp"`, `0xA3: "LAX_indx"`, `0xFB: "ISB_aby"`, `0xFF: "ISB_abx"`). Authoritative sources for the 105-entry table (cite in a doc comment above the map literal): (a) NESdev wiki `https://www.nesdev.org/wiki/CPU_unofficial_opcodes`, (b) `http://www.oxyron.de/html/opcodes02.html`, (c) the `64doc.txt` reference (`https://www.atarihq.com/danb/files/64doc.txt`). Cross-check the resulting set against `mos6502/illegal.go` / current `opcodeTable` stubs to confirm parity with existing NOP-stub coverage (data-model E3, R5)
- [X] T025 [P] [US3] Add `documentedOpcodes() map[uint8]bool` helper introspecting `mos6502/opcodes.go` (the `opcodeTable` / handler-function identity) to derive the documented set without a parallel hand-list; if introspection isn't feasible without exporting symbols, use a build-tagged internal helper that imports `mos6502` and walks the table via reflection on test-time exported names (data-model E3 "Authoritative documented-opcode source")
- [X] T026 [US3] Add `TestSkipListInvariants` asserting: (a) `len(skipList) == 105`, (b) every key NOT in `documentedOpcodes()`, (c) every value non-empty, (d) documented + skipped = 256, (e) no key duplicates (map shape enforces but assert via `len` invariant) in `mos6502/processortests_test.go` (data-model E3 invariants, FR-004, SC-002)
- [X] T027 [US3] Wire `TestProcessorTests` opcode loop to consult `skipList`: for any opcode byte present in `skipList`, `t.Run(subtestName(op), func(t *testing.T) { t.Skipf("undocumented: %s (Phase 003 OOS-001)", skipList[op]) })`; documented opcodes proceed to the parallel subtest path from T013 in `mos6502/processortests_test.go` (FR-004, FR-007 no-whole-opcode-skipping-under-short still applies only to documented set)

**Checkpoint**: All three slices land. Full suite is now runnable end-to-end.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Verify success criteria, document, and update tracking artefacts.

- [X] T028 [P] Update `mos6502/` quickstart pointer (top-of-file doc comment in `processortests_test.go` or `mos6502/README.md` if present) per FR-011: name the pinned SHA, link to `specs/003-cpu-processor-tests/quickstart.md`, document `-short` vs full mode
- [X] T029 Run `make fmt vet lint test cover` and verify: full `TestProcessorTests` PASS for all 151 documented opcodes, 105 SKIPs reported, `mos6502/` line coverage ≥ 99.3% (SC-001, SC-002, SC-003)
- [X] T030 Run `make bench` and verify `BenchmarkInstrMix` ns/cycle within the ≤ 125 ns/cycle SC-006 budget and within margin of the T004-recorded baseline (no hot-path regression from harness — the harness should not affect bench code at all, but assert this rather than assume)
- [X] T031 Run `go test ./mos6502/` (full corpus, no `-short`) on a typical amd64 dev box and verify wall-time ≤ 5 minutes (SC-008); record actual time in commit body for nightly-CI tuning
- [X] T032 Run `go test -short ./mos6502/` and verify single-digit-second runtime exercising all 151 documented opcodes via 100-case sample (SC-005)
- [X] T033 SC-006 self-check (deliberate perturbation): on a throwaway branch, inject a single-cycle-order swap into one addressing-mode helper in `mos6502/addressing.go`, rerun `TestProcessorTests`, confirm exactly the affected opcode subtest fails with a failure message naming the divergent cycle index plus expected and observed `{addr,value,kind}`; revert the perturbation; record the demonstration in the commit body
- [X] T034 [P] Verify FR-009: confirm `TestFunctionalROM` (Klaus Dormann), `TestGoldenTraces`, and the 91 pre-existing unit tests still PASS post-harness — explicit grep over `go test -v ./mos6502/` output for the historic test names
- [X] T035 [P] Update `docs/roadmap.md` Phase 003 row: `🟡 Planned` → `✅ Complete`; append one-line completion summary (documented opcodes passing, undocumented skipped, full-run time achieved, coverage delta) (R10)
- [X] T036 [P] Update `CLAUDE.md` `mos6502/` "Validation status" section to record Tom Harte corpus passing — line under the existing Klaus Dormann entry naming pinned SHA, opcode counts, and short/full mode behaviour
- [X] T037 Run `quickstart.md` end-to-end on a fresh checkout (or simulated clean state via `rm -rf mos6502/testdata/processortests`): `go generate ./mos6502/` then `go test ./mos6502/` — confirm the documented two-command path completes green (SC-004)
- [X] T038 [P] Verify FR-010 (zero new public API, zero new runtime deps): (a) run `go list -json ./mos6502 | jq '.Export, .CompiledGoFiles'` (or equivalent `go doc -all ./mos6502`) and diff the exported-symbol surface against the T004 baseline — MUST be empty diff; (b) run `go mod tidy` and confirm `git diff go.mod go.sum` is empty; (c) grep new files for `import "github.com/...non-stdlib..."` — only stdlib + `github.com/gobeeb/GoBeeb/mos6502` allowed. Record outcome in commit body (FR-010)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: no dependencies; start immediately
- **Foundational (Phase 2)**: depends on Setup; blocks all stories
- **US1 (Phase 3)**: code authorable after Foundational; green run blocked on US2 + US3
- **US2 (Phase 4)**: independent of US1/US3 code-wise; produces corpus consumed by US1
- **US3 (Phase 5)**: independent of US2 code-wise; produces `skipList` consumed by US1
- **Polish (Phase 6)**: depends on US1 + US2 + US3 all complete

### User Story Dependencies (data-flow)

- US2 corpus output → consumed by US1 runtime preflight (T014) and per-opcode JSON read (T013)
- US3 `skipList` symbol → consumed by US1 driver loop (T013, T027)
- US2 `pinnedCorpusSHA` constant → mirrored in US2 test-side declaration (T022)

These are *symbol/file-path* dependencies, not story-completion blockers — US1 harness code compiles standalone because `skipList` and corpus reads are name-resolved at test time, not at compile time of an isolated file. The green-run gate is what couples them.

### Within Each User Story

- T005–T007 (US1 types + adapter) before T008/T009 (their unit tests)
- T010 (subtestName) before T013 (driver consumes it)
- T011 (runCase) before T013 (driver consumes it)
- T012 (failure formatter) before T013 (runCase forwards diffs to it)
- T013 (driver) before T014 (preflight injects into the driver)
- T016 (gen.go skeleton) before T017–T020 (fill behaviours)
- T017–T020 (gen.go logic) before T021 (`//go:generate` directive) — directive useless without working binary
- T024 (skipList literal) before T026 (invariant test consumes it)
- T025 (documentedOpcodes helper) before T026 (invariant test consumes it)
- T027 (driver wiring) after T013 + T024 (both targets must exist)

### Parallel Opportunities

Within Phase 3 (US1):
- T005, T006, T007 parallel (different symbols, same file; commit boundary serialises edits but logical work is independent)

Within Phase 4 (US2):
- T016 is a single new file → T017–T020 are sequential within the file

Within Phase 5 (US3):
- T024 and T025 parallel (different symbols)

Within Phase 6 (Polish):
- T028, T034, T035, T036, T038 parallel (different files)

Cross-story (after Phase 2):
- US1 harness implementation (T005–T015), US2 fetcher (T016–T023), and US3 skip list (T024–T027) can be authored by three contributors in parallel because they touch disjoint files (US2) or disjoint symbols in the shared `processortests_test.go` (US1 + US3).

---

## Parallel Example: US1 Harness Bootstrapping

```text
# After Foundational (T004) completes:
Task T005: Add processorCase/processorState/ramEntry/busCycle types + custom UnmarshalJSON in mos6502/processortests_test.go
Task T006: Add processorState.toRegisters() helper in mos6502/processortests_test.go
Task T007: Add ramMap sparse memory adapter in mos6502/processortests_test.go
# Run in parallel — three contributors can stage these as disjoint diffs against the new file.
```

---

## Implementation Strategy

### MVP First (US1 — but coupled to US2 + US3 for green run)

Phase 003 doesn't split MVP-fashion the way most features do — US1 is the entire deliverable; US2 + US3 are the supporting infrastructure that US1's independent test scenario explicitly invokes. The natural sequence is:

1. Complete Phase 1 (Setup) + Phase 2 (Foundational baseline capture)
2. Land US2 first (T016–T023): now `go generate ./mos6502/` works and the corpus is on disk
3. Land US3 next (T024–T027): `skipList` and invariants in place
4. Land US1 (T005–T015): driver consumes both — green run achievable
5. **STOP and VALIDATE**: `go test ./mos6502/` full run + `go test -short ./mos6502/`
6. Polish (Phase 6) — verify SC-* and update tracking

This ordering means US1 commits don't have to introduce dead code or `t.Skip`s waiting for corpus arrival. Alternative ordering (US1 code first against placeholder corpus) is supported by the design but adds a temporary `t.Skip` you'd remove on US2 landing — extra churn for no benefit.

### Incremental Delivery (per-commit cadence)

Each task above is a clean atomic commit boundary:

- T001 + T002 → setup commit
- T004 → baseline commit
- T016 → empty `gen.go` shell commit; T017–T020 → fetcher logic commit; T021 → directive commit; T022 → SHA-mirror commit; T023 → manual smoke commit body
- T024 + T026 → skip list + invariants commit; T025 → documentedOpcodes helper commit; T027 → wiring commit
- T005–T015 → either one large harness commit or splittable along the natural boundaries (types/adapter, runCase/formatter, driver, preflight)
- Polish (T028–T037) → individual small commits

### Parallel Team Strategy

With three contributors after Phase 2:

1. Contributor A: US2 (`mos6502/gen.go` + `//go:generate` directive)
2. Contributor B: US3 (`skipList` + `TestSkipListInvariants` + `documentedOpcodes`)
3. Contributor C: US1 (`processortests_test.go` harness body — types, adapter, runCase, driver)

C's PR rebases over A's + B's at merge time; only T027 (driver wiring to skipList) and T013 (driver consuming the harness) need final synchronisation.

---

## Notes

- **No new public API on `mos6502`** (FR-010) — every new symbol is unexported and lives in `_test.go` or build-ignored `gen.go`. Reviewers should `git diff` the package's exported surface area before merge and confirm it is unchanged.
- **No new runtime dependencies** — only stdlib (`encoding/json`, `testing`, `os`, `path/filepath`, `bytes`, `fmt`, `os/exec` in `gen.go`). Confirm `go.mod`/`go.sum` show no additions.
- **Tests-of-tests**: T008, T009, T026, T022 are harness-validation tests. Not optional. They are the self-check that catches typos in the harness before they silently pass real CPU bugs.
- **SC-006 self-check (T033)** is the meta-validation that the harness actually fails when it should. Do not skip.
- **Corpus is gitignored** (FR-005) — `git status` after `go generate` should show clean. Verify in T023 and T037.
- Avoid: editing `mos6502/cpu.go` or other production files except when fixing a divergence the corpus surfaces (in which case it's a real bug fix and goes in the same phase per Assumptions).
