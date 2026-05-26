# Phase 0 Research — CPU Bus-Cycle Validation

All `NEEDS CLARIFICATION` markers from spec were resolved during `/speckit-clarify` (5 Q/A pairs). This document captures the remaining open implementation-level questions, the best-practice survey behind each, and the locked decision.

---

## R1 — Pinned upstream commit SHA

**Decision**: Pin the latest `master` commit of `SingleStepTests/ProcessorTests` at phase-execute time. Capture it once in `mos6502/processortests_test.go` as `const pinnedCorpusSHA = "<40-hex>"` and mirror it into `mos6502/gen.go`. Discover the SHA via `git ls-remote https://github.com/SingleStepTests/ProcessorTests.git refs/heads/master` during the first execute task; record the result in the implementation-phase commit.

**Rationale**: Tom Harte's repo updates infrequently and never re-rewrites history on `master` (file additions only). Latest-at-pin is the simplest reproducibility story and avoids us picking a stale revision the upstream may have superseded with a fix. Recording the SHA in two places (test + generator) keeps each file self-describing and lets either fail loudly if they drift.

**Alternatives considered**:
- A *tagged* upstream release — upstream does not publish tags for the 6502 corpus, so this is not available.
- A *git submodule* — would commit a SHA into our tree without committing the corpus payload itself. Adds the friction of "did you `git submodule update --init --recursive`?" to every fresh clone. Sparse-checkout via `git clone --depth 1` solves the same problem with no submodule cost.
- Fork the corpus into a `gobeeb/processortests-mirror` repo and pin the fork — adds maintenance + a permanently-divergent supply chain. Defer until upstream-availability becomes a real concern.

---

## R2 — Sparse-memory adapter shape

**Decision**: `type ramMap map[uint16]uint8` implementing `mos6502.Memory`. Reads of unmapped addresses return `0x00`. Writes record into the map. Created per case; the map's allocation cost is amortised across the case's bus cycles (typically ≤ 10 RAM entries in `initial.ram`, ≤ ~10 writes during the cycle sequence). Capacity hint: `make(ramMap, 32)` per case.

**Rationale**:
- Tom Harte cases set `initial.ram` as scattered `[addr, value]` pairs across a tiny working set (PC, stack page, indirect pointers, effective addresses) — typically fewer than 16 unique addresses per case. A 64 KB `[65536]byte` per case would allocate 64 KB × ~15 parallel goroutines = ~1 MB live + heavy GC churn through 1.5M cases; a `map[uint16]uint8` allocates ~1–2 KB per case (Go map bucket math) and is friendlier to the allocator on millions of cases.
- The map naturally captures `final.ram` comparison: iterate `final.ram` and assert `map[addr] == value`; iterate the map and assert every key that's not in `final.ram` matches the address's `initial.ram` value (unchanged). This makes RMW double-write semantics observable in the trace but invisible in the final-RAM diff (which is correct — the trace assertion catches the dummy write, not the RAM-state diff).
- Implementing `Memory` (two methods, no error) is trivial.

**Alternatives considered**:
- Flat 64 KB `[65536]byte` (mirror `flatRAM` from `functional_test.go`) — simpler, but ~1.5M × 64 KB = ~100 GB total allocation if not reused. Reuse via `sync.Pool` is possible but adds harness complexity that earns nothing because the map shape matches the JSON `initial.ram` shape verbatim.
- A sparse-by-page (`map[uint8][256]byte`) hybrid — wins only for cases with high spatial locality, which Tom Harte cases don't have. Skip.
- Slice + bitmap of mapped flags — micro-optimised, hides intent, no measurable win on this scale.

---

## R3 — Trace capacity & reuse strategy

**Decision**: One `*mos6502.Trace` per test goroutine (i.e., per opcode subtest), pre-allocated to capacity 16, reset between cases via the existing `Trace.Reset()`. Per-case allocation reduces to the slice that `Trace.Snapshot()` returns (necessary; consumed by the assertion code).

**Rationale**:
- NMOS 6502 documented opcodes top out at 8 cycles (`RTI`, `RTS`-with-page-cross). 16 gives 2× headroom and matches the order-of-magnitude already used by `golden_trace_test.go` (`NewTrace(64)`).
- `Trace.Reset()` already exists (`trace.go:78`) and clears without re-allocating the backing buffer — exactly what's needed.
- Per-goroutine (not per-case) ownership keeps the trace allocation outside the inner loop and matches the parallelism story (each subtest goroutine owns its own trace; no contention).

**Alternatives considered**:
- One trace per case — adds 16-byte-struct × 1.5M allocations, sub-microsecond cost each but unnecessary.
- A `Trace` pool via `sync.Pool` — earns nothing on top of per-goroutine ownership.

---

## R4 — JSON parsing strategy & per-case allocation budget

**Decision**: Parse each opcode's `XX.json` file once per subtest into a `[]processorCase` slice via `encoding/json` standard unmarshal. Iterate. Do not stream. Do not cache across subtests. Do not optimise parsing further unless the 5-minute full-run budget is missed.

**Rationale**:
- Each opcode file is ~few MB (10,000 cases × ~hundreds of bytes each). Parsed once per subtest goroutine — trivial.
- Streaming via `json.Decoder` saves transient memory but costs source clarity. Per-subtest parse caps live JSON memory at GOMAXPROCS × single-file size = manageable on any dev box.
- Cross-subtest case caching (e.g., load all files at `TestProcessorTests` entry) would force loading ~hundreds of MB up-front and lengthen `-short` warm-up. Per-subtest beats it.
- A custom faster JSON path (e.g., handwritten state machine, `json.RawMessage` deferral) is premature — re-evaluate only if SC-008 fails on a representative machine. Reasoned budget: ~38 s of pure-CPU `Step` time at ~5 ns/cycle × 1.51M cases × ~5 cycles/case + JSON parse + assertion ≈ 4 min on a 4-core box. Comfortable on 5 min.

**Alternatives considered**:
- `bytedance/sonic`, `goccy/go-json`, `mailru/easyjson` — drag in a runtime dep, violates FR-010 spirit (zero new deps). Defer until proven needed.
- Cache parsed cases on disk in gob format — adds a second representation to maintain. No.

---

## R5 — Skip-list canonical form

**Decision**: `var skipList = map[uint8]string{ … }` in `processortests_test.go`. Key = opcode byte. Value = the canonical undocumented mnemonic + addressing-mode tag (e.g., `0x07: "SLO_zp"`, `0xA3: "LAX_indx"`, `0x02: "KIL"`). 105 entries. Validated at test startup by `TestSkipListInvariants` — checks length == 105, every key is not present in the documented-opcode set (cross-referenced from `mos6502/opcodes.go`), and every value is non-empty.

**Rationale**:
- Map-by-opcode is the lookup the harness needs (single byte → "skip?"). Mnemonic values make the skip list grep-able and self-documenting when the future undocumented-opcode phase lands.
- Validation invariants catch typos at startup (length drift, accidental overlap with documented opcodes) rather than letting the suite quietly accept a wrong skip set.

**Alternatives considered**:
- Bitmask `[32]uint8` — micro-optimised, completely opaque. No.
- Slice of structs — works but loses the O(1) lookup ergonomics.

---

## R6 — Fetch script: pure Go vs shell

**Decision**: Pure Go program at `mos6502/gen.go` guarded by `//go:build ignore` so it's invoked as `go run gen.go` (not compiled into the package). It shells out to `git` for the four operations (`init`, `remote add`, `sparse-checkout`, `fetch --depth 1` + `checkout`). Pure Go (vs a bash script) is portable across Linux/macOS/Windows and matches the project's Go-only ethos.

**Rationale**:
- Shelling to `git` is mandated by the Q1 clarification (`git clone --depth 1` + sparse-checkout). The wrapper around those shell-outs being itself in Go gets us cross-platform paths, idempotency checks, and a clear `os.Exit(1)` failure surface without inventing a `Makefile`/`bash` dependency on Windows dev boxes.
- `//go:build ignore` keeps the generator file from polluting normal `go build`/`go test` of the `mos6502` package while still letting `go run mos6502/gen.go` work via `//go:generate` from inside the package.

**Idempotency check**: Generator reads `mos6502/testdata/processortests/.fetched-sha` (a tiny single-line marker file the generator writes after a successful checkout). If the file's content equals `pinnedCorpusSHA` and `6502/v1/` is non-empty, exit success without touching the network. Otherwise, do the full re-fetch (rm -rf the directory, re-clone). This handles partial-download recovery (Q3 edge case) without inventing complicated repair logic.

**Alternatives considered**:
- Shell script (`scripts/fetch-processortests.sh`) — adds a per-platform branch (bash vs PowerShell) the project doesn't otherwise need. Reject.
- Pure Go using `go-git` library — adds a heavy runtime dep solely for the generator, violating FR-010 spirit. The shell-out is simpler.
- Just-trust-the-user with a documented `git clone` invocation — fails the SC-004 "two commands from clean checkout" criterion.

---

## R7 — Parallelism & determinism

**Decision**: Each opcode subtest calls `t.Parallel()`. Within a subtest, cases run serially (no per-case parallelism). Trace and sparse-memory adapter are subtest-goroutine-local, so no shared mutable state. No `t.Parallel()` on the top-level `TestProcessorTests` (Go semantics — parent runs to completion before child parallelism resolves; this is fine).

**Rationale**:
- 151 parallel goroutines on a typical 8-to-16-core dev box saturates CPU without overwhelming the scheduler.
- Per-case parallelism inside a subtest would force shared-state synchronisation around the trace and complicate failure attribution (which case index failed?).
- Determinism: case iteration order within a file is the file's order; failure messages cite the file's `name` field, which makes them reproducible across runs.

**Alternatives considered**:
- `t.Parallel()` per case — orders-of-magnitude more goroutines, worse failure messages, no throughput win after CPU saturation.
- No parallelism, single goroutine — misses the 5-min full-run budget on multi-core machines; user explicitly chose Option B in Q5 with parallelism as the implementation lever.

---

## R8 — Failure-message localisation (FR-008)

**Decision**: Per-case failure formatter prints:
1. The `name` field from the JSON case.
2. If register diff: a single multi-line block with all 6 fields (`A`, `X`, `Y`, `SP`, `PC`, `P`) showing got vs want.
3. If RAM diff: list of `(addr, got, want)` triples for divergent cells only.
4. If trace diff: a unified diff of the cycle sequences, with the first divergent cycle index, both expected and observed `{addr, value, kind}`.

Use `t.Errorf` (not `t.Fatalf`) inside the case loop so one bad case doesn't blind us to the rest of the opcode's failure modes. Cap reported failures per subtest at 5 (after which subtest moves to `t.Fatalf` with a "skipped remaining N cases" message) to keep failure noise bounded.

**Rationale**: Matches the "localise to divergent cycle index" requirement (FR-008, SC-006). The 5-failure cap is borrowed from the existing `golden_trace_test.go` style of one-shot trace mismatch — Tom Harte's volume forces the cap.

---

## R9 — Status-line / progress reporting

**Decision**: None. Rely on Go's default `t.Parallel()` test output (one `PASS`/`FAIL` line per opcode subtest as it completes). No custom progress printer. Outstanding item from spec coverage; deferred per spec note as low-impact.

**Rationale**: A custom printer would either spam (per-case) or require coordination across parallel goroutines for sensible progress aggregation. Go's default subtest output is sufficient and matches the rest of `mos6502/`. If a future maintainer wants a single-line "X/151 opcodes done" indicator, that's an additive change to the harness, not a blocker for this phase.

---

## R10 — Roadmap status update on phase exit

**Decision**: Single-line edit to `docs/roadmap.md` flipping Phase 003 row from `🟡 Planned` → `✅ Complete` and appending a one-line completion summary (test counts, runtime achieved, coverage delta) at the end of the Phase 003 section. Done as the final commit of the phase, not by the test harness itself.

**Rationale**: Mirrors the Phase 002 completion-line precedent already in the roadmap. Keeps the roadmap a single source of phase truth without coupling it to test-run side effects.

---

## Open items intentionally NOT decided here

- Whether to implement the 105 undocumented opcodes — explicitly OOS-001 in the spec. The skip list is the contract; this phase ships skipped, the next phase ships green.
- Whether to also consume the `nes6502/v1/` subdir (NES-decimal-disabled 2A03 variant) — OOS-002 covers 65C02/65816; the NES variant is implicit OOS (we are not modelling NES).
- Whether to mirror the corpus to a gobeeb-owned bucket as a fallback — defer until upstream-availability becomes a real concern (R1 alternatives).

---

**Phase 0 status**: All NEEDS CLARIFICATION resolved (none remained after `/speckit-clarify`). All implementation-level questions above have a locked decision with stated rationale. Ready for Phase 1 design artifacts.
