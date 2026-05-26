# Phase 1 Data Model — CPU Bus-Cycle Validation

This phase introduces NO persistent runtime data and NO new public types on the `mos6502` package. All entities below are **test-only** Go types living in `mos6502/processortests_test.go` (unexported) plus one tiny on-disk marker file owned by the generator.

---

## E1 — `processorCase` (in-memory, test-only)

One parsed Tom Harte JSON case. Carries the full initial state, expected final state, and the expected per-cycle bus trace.

```go
// All fields unexported; struct itself unexported. encoding/json maps via tags.
type processorCase struct {
    Name    string         `json:"name"`
    Initial processorState `json:"initial"`
    Final   processorState `json:"final"`
    Cycles  []busCycle     `json:"cycles"`
}

type processorState struct {
    PC  uint16     `json:"pc"`
    S   uint8      `json:"s"`
    A   uint8      `json:"a"`
    X   uint8      `json:"x"`
    Y   uint8      `json:"y"`
    P   uint8      `json:"p"`
    RAM []ramEntry `json:"ram"`
}

type ramEntry struct {
    Addr  uint16
    Value uint8
}

// Custom UnmarshalJSON on ramEntry — upstream encodes as a 2-element JSON
// array [addr, value], not as an object. Same trick for busCycle below.
func (r *ramEntry) UnmarshalJSON(data []byte) error { … }

type busCycle struct {
    Addr  uint16
    Value uint8
    Kind  mos6502.BusEventKind // BusRead | BusWrite
}

// Custom UnmarshalJSON — upstream encodes as [addr, value, "read"|"write"].
func (c *busCycle) UnmarshalJSON(data []byte) error { … }
```

**Lifetime**: parsed once per opcode-subtest from `XX.json`, iterated, then garbage-collected when the subtest exits.

**Validation**: implicit. If JSON unmarshal fails, that's a corpus-format regression (or a wrong-SHA fetch) and the subtest fails via `t.Fatalf`.

**Why not reuse `mos6502.Registers`?** `Registers` has a `Cycles` field the corpus doesn't carry. Mapping is one trivial helper:

```go
func (s processorState) toRegisters() mos6502.Registers {
    return mos6502.Registers{
        A: s.A, X: s.X, Y: s.Y, SP: s.S, PC: s.PC, P: s.P, Cycles: 0,
    }
}
```

---

## E2 — `ramMap` (sparse memory adapter)

Implements `mos6502.Memory`. One instance per case.

```go
type ramMap map[uint16]uint8

func (r ramMap) Read(addr uint16) uint8 {
    return r[addr] // missing key returns zero — matches "implicit zero" semantics
}

func (r ramMap) Write(addr uint16, value uint8) {
    r[addr] = value
}

// Populate from a case's initial RAM.
func newRAMMap(initial []ramEntry) ramMap {
    m := make(ramMap, len(initial)*2) // 2× headroom for writes
    for _, e := range initial {
        m[e.Addr] = e.Value
    }
    return m
}
```

**Invariant**: any read that misses the map returns `0x00`. This matches the corpus's implicit-zero semantics — Tom Harte cases only enumerate non-zero RAM in `initial.ram`.

**Final-state comparison**:
- Iterate `final.ram`: assert `m[addr] == value` for every entry.
- Iterate `m`: for every key `k` not in `final.ram`, assert `m[k]` is unchanged from `initial.ram[k]` (or, if `k` is also not in `initial.ram`, that `m[k] == 0` — i.e., a spurious write to a never-mapped address occurred and we should flag it).

---

## E3 — `skipList` (constant; governance contract)

Static map of undocumented opcodes deferred from this phase.

```go
var skipList = map[uint8]string{
    0x02: "KIL",
    0x03: "SLO_indx",
    0x04: "DOP_zp",
    0x07: "SLO_zp",
    0x0B: "ANC_imm",
    0x0C: "TOP_abs",
    0x0F: "SLO_abs",
    // … 105 entries total …
    0xFB: "ISB_aby",
    0xFF: "ISB_abx",
}
```

**Invariants** (enforced by `TestSkipListInvariants`):
1. `len(skipList) == 105`.
2. No key in `skipList` appears in the documented-opcode set (cross-checked against an authoritative list — see below).
3. Every value is non-empty.

**Authoritative documented-opcode source**: derived once at test startup by introspecting `mos6502.opcodeTable` — any opcode byte whose handler is the canonical `illegalOpcode` (or equivalent NOP-stub function — confirmed during execute phase by reading `mos6502/opcodes.go` and `mos6502/illegal.go`) is "undocumented"; the complement is "documented". This gives us a single source of truth without hand-maintaining two parallel lists.

**Drift detection**: if `mos6502/opcodes.go` later changes which opcodes are documented (e.g., the future undocumented-opcode phase adds real implementations), `TestSkipListInvariants` will fail on invariant 2 and the maintainer is forced to update the skip list. This is the explicit contract between Phase 003 and the future implementation phase.

---

## E4 — `pinnedCorpusSHA` (constant; reproducibility contract)

```go
const pinnedCorpusSHA = "<40-char-hex>" // determined during execute; recorded in commit
```

**Mirrored in**: `mos6502/gen.go` (the generator) as a `const` of the same name. Both files reference the same upstream commit; drift between them is caught by inspection during code review.

**Why two declarations, not a shared one?** `mos6502/gen.go` has `//go:build ignore` and is not part of the `mos6502` package at normal compile time. Sharing a symbol would require either an `internal/` package or a `//go:embed`'d constants file. Both add ceremony for two-line constants that change once per pin update. Live with the duplication; it's noisy enough to be caught.

---

## E5 — `.fetched-sha` (on-disk marker; generator state)

Single-line text file written by `gen.go` after a successful corpus checkout:

```text
mos6502/testdata/processortests/.fetched-sha
```

Content: the 40-character SHA that was successfully checked out.

**Purpose**: idempotency. On generator re-invocation, compared against `pinnedCorpusSHA`:
- equal AND `6502/v1/` non-empty → exit success without touching the network
- not equal OR `6502/v1/` empty → wipe `testdata/processortests/` and re-fetch

**Lifecycle**: created by `gen.go`, never edited by tests. Gitignored alongside the rest of `testdata/processortests/`.

---

## Relationships

```text
processorCase  ──(holds)──>  processorState (initial, final)
                  │                  │
                  │                  └──(holds list of)──>  ramEntry
                  │
                  └──(holds list of)──>  busCycle  ───(compares against)──>  mos6502.BusEvent
                                                                              (via Trace.Snapshot)

ramMap  ───(implements)──>  mos6502.Memory
        ───(populated from)──>  processorState.RAM (initial)
        ───(asserted against)──>  processorState.RAM (final)

skipList  ───(validated against)──>  mos6502.opcodeTable / mos6502.illegal
pinnedCorpusSHA  ───(matched against)──>  .fetched-sha (on disk)
```

---

## State transitions

The only "lifecycle" in this phase is the per-case test execution:

```text
[load JSON file] → [for each case]:
    new ramMap ← initial.ram
    new CPU ← sparse adapter
    cpu.SetRegisters(initial)
    cpu.SetTrace(trace.Reset())
    cpu.Step()
    assert cpu.Registers() == final
    assert ramMap matches final.ram
    assert trace.Snapshot() matches case.Cycles
```

No persistent state across cases. No cross-test state (each opcode subtest is independent under `t.Parallel()`).

---

## Out of model

- No new public API on `mos6502` (FR-010).
- No persisted test artefacts (golden files, snapshots, etc.).
- No configuration file (the harness is fully self-contained between `processortests_test.go`, `gen.go`, and the on-disk `.fetched-sha` marker).
