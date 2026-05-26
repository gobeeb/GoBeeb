# Quickstart — CPU Bus-Cycle Validation

Two commands from a clean checkout get you to a green run.

---

## Prerequisites

- Go 1.22+ (`go version` to confirm). Repo uses mise; `mise install` if you have it configured.
- `git` on `PATH` (the corpus fetcher shells out to it). Verify: `git --version`.
- Network access to `github.com` for the first `go generate`. The actual `go test` invocation is fully offline.

---

## First time on a machine

```sh
cd /path/to/agent_gobeeb
go generate ./mos6502/
go test ./mos6502/
```

The first command sparse-checks-out `SingleStepTests/ProcessorTests/6502/v1/` at the pinned commit SHA into `mos6502/testdata/processortests/` (gitignored). The second command runs the full suite — Klaus Dormann functional ROM + golden traces + unit tests + the new Tom Harte corpus (10,000 cases per documented opcode).

Expected output:

```text
=== RUN   TestProcessorTests
=== RUN   TestProcessorTests/0x69_ADC_imm
=== PAUSE TestProcessorTests/0x69_ADC_imm
=== RUN   TestProcessorTests/0xA9_LDA_imm
=== PAUSE TestProcessorTests/0xA9_LDA_imm
…
=== CONT  TestProcessorTests/0x69_ADC_imm
--- PASS: TestProcessorTests/0x69_ADC_imm (3.21s)
…
--- PASS: TestProcessorTests (≤ 5 min on amd64)
PASS
ok    github.com/gobeeb/GoBeeb/mos6502
```

---

## Day-to-day (fast loop)

```sh
go test -short ./mos6502/
```

Runs the first 100 cases per documented opcode (~15,100 cases). Single-digit seconds on a modern dev box. Use this for the inner edit-run loop.

---

## Already fetched? Re-running `go generate`

```sh
go generate ./mos6502/
# → processortests corpus already at <pinnedSHA>; skipping fetch
```

Idempotent. Only re-fetches if `mos6502/testdata/processortests/.fetched-sha` is missing or doesn't match `pinnedCorpusSHA`.

To force a re-fetch (e.g., suspected partial download):

```sh
rm -rf mos6502/testdata/processortests
go generate ./mos6502/
```

---

## Updating the pinned corpus SHA

1. `git ls-remote https://github.com/SingleStepTests/ProcessorTests.git refs/heads/master` to discover the latest commit.
2. Update both `pinnedCorpusSHA` constants — in `mos6502/processortests_test.go` and `mos6502/gen.go`.
3. `rm -rf mos6502/testdata/processortests && go generate ./mos6502/`.
4. `go test ./mos6502/` — full corpus run on the new SHA.
5. If anything diverges, that's either a real CPU bug (fix in `mos6502/`) or an upstream-format change (re-examine the JSON schema contract).

---

## Running a single opcode

```sh
go test -run 'TestProcessorTests/0xA9_LDA_imm' ./mos6502/
```

Subtest naming convention: `0x<hex>_<MNEMONIC>_<addr_mode>` — e.g., `0xA9_LDA_imm`, `0x69_ADC_imm`, `0x6D_ADC_abs`, `0xFE_INC_abx`. See the `subtestName` function in `processortests_test.go` for the canonical naming source.

---

## Adding verbose failure output

`go test -v ./mos6502/ -run TestProcessorTests/0xA9_LDA_imm` prints subtest progress and the full failure message on diff. Trace failures include the first divergent cycle index plus expected and observed `{addr, value, kind}`.

---

## What "PASS" means

- All 151 documented NMOS 6502 opcodes match a real chip cycle-for-cycle across their 10,000 cases each (full run) or 100 cases each (`-short` run).
- All 105 undocumented opcodes are reported as SKIP (cross-referenced in `skipList`).
- All pre-existing `mos6502/` tests still pass (Klaus Dormann functional ROM, golden traces, unit suites).
- `mos6502/` line coverage stays at or above 99.3%.

---

## CI integration

The CI default runs `go test -short ./mos6502/...` — the corpus is fetched in a CI prep step (`go generate ./mos6502/`) so the short suite never needs network. The full run is intended for nightly / pre-release; gating it on `master` push would inflate CI time.

If the corpus directory is missing when `go test` runs (i.e., someone forgot `go generate`), the harness hard-fails (`t.Fatal`) with the remediation command:

```text
--- FAIL: TestProcessorTests
    processortests_test.go:NN: corpus not found at mos6502/testdata/processortests/6502/v1/; run `go generate ./mos6502/` to fetch
```

Loud is correct. Silent skip would defeat the validation purpose.

---

## Troubleshooting

| Symptom | Cause | Fix |
|---|---|---|
| `git not found on PATH` from `go generate` | system `git` missing | install `git` |
| `processortests corpus fetch failed: pinnedCorpusSHA unreachable` | SHA rewritten upstream or network blocked | confirm `pinnedCorpusSHA`; check network; re-pin if upstream changed history (unlikely) |
| `TestProcessorTests/0xXX_…` fails with a register diff | real CPU divergence | inspect the named case (`name` field); reproduce with `golden_trace_test.go` style; fix in `mos6502/` |
| `TestProcessorTests/0xXX_…` fails with a trace diff at cycle N | bus-order bug — extra/missing/swapped cycle | inspect the cycle index N; trace through addressing-mode code in `mos6502/addressing.go` / `rmw.go` |
| `TestSkipListInvariants` fails | skip list drifted from documented-opcode set | reconcile `skipList` against `mos6502.opcodeTable` |
| Full run exceeds 5 min on your box | machine slower than typical amd64 | use `-short` for dev loop; full run remains correctness oracle, no SLA on slow hardware |
