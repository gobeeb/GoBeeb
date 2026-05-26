# Contract — SingleStepTests/ProcessorTests JSON Schema

**Status**: external contract owned by upstream `SingleStepTests/ProcessorTests`. This file documents the shape we depend on at the pinned SHA. Drift from this shape by a future upstream revision is a corpus-format break that the harness MUST detect (JSON unmarshal failure) rather than silently mis-parse.

**Upstream**: <https://github.com/SingleStepTests/ProcessorTests>

**Pinned subtree**: `6502/v1/`

**Per-opcode file**: `XX.json` where `XX` is the 2-digit lowercase hex opcode byte (e.g., `a9.json` for `LDA #imm`). 256 files in the subtree (one per opcode byte, documented + undocumented).

**Per-file shape**: a top-level JSON array of test-case objects, exactly 10,000 entries per file (except the historical first revision, which is the only one this phase considers stable).

---

## Case object shape

```json
{
  "name": "a9 a3 c2",
  "initial": {
    "pc": 35714,
    "s":  248,
    "a":   45,
    "x":   54,
    "y":   23,
    "p":   34,
    "ram": [
      [35714, 169],
      [35715, 163]
    ]
  },
  "final": {
    "pc": 35716,
    "s":  248,
    "a": 163,
    "x":  54,
    "y":  23,
    "p":  160,
    "ram": [
      [35714, 169],
      [35715, 163]
    ]
  },
  "cycles": [
    [35714, 169, "read"],
    [35715, 163, "read"]
  ]
}
```

### Field contract

| Field           | Type                                       | Notes                                                                                                       |
|-----------------|--------------------------------------------|-------------------------------------------------------------------------------------------------------------|
| `name`          | string                                     | Human-readable. Used verbatim in failure messages. Format varies by case; treat as opaque.                  |
| `initial.pc`    | uint16 (JSON number)                       | Program counter at start.                                                                                   |
| `initial.s`     | uint8                                      | Stack pointer (live stack address = `0x0100 \| S`).                                                          |
| `initial.a/x/y` | uint8                                      | Architectural registers.                                                                                    |
| `initial.p`     | uint8                                      | Processor status. Bit layout `NV-BDIZC`; matches `mos6502.Flag*` constants.                                  |
| `initial.ram`   | array of `[uint16, uint8]` 2-element arrays | Sparse RAM pre-state. Addresses absent from this list are implicit zero.                                    |
| `final.*`       | same shape as `initial`                    | Expected post-`Step()` architectural state and RAM mutations.                                               |
| `cycles`        | array of `[uint16, uint8, string]` 3-tuples | One entry per bus cycle in chronological order. String is exactly `"read"` or `"write"` (case-sensitive). |

### Decoding subtleties

- `ram` and `cycles` entries are JSON **arrays**, not objects. Custom `UnmarshalJSON` on the Go side decodes the 2- or 3-tuple shape — standard struct-field tags do not handle this.
- All numeric values are non-negative integers within the byte/word ranges; the harness assumes upstream never emits floats or negatives. A range overflow during unmarshal is a corpus-format break and MUST surface as a `t.Fatalf` in the test.
- The `cycles` count equals the cycle count for that opcode/operand combination on a real NMOS. It is NOT a "claimed cycle budget" — equality with `Trace.Snapshot()` length is part of the assertion.
- Order matters: the `cycles` array reflects the exact bus order issued by a real NMOS, including dummy reads on page-cross and the dummy write during RMW. The harness asserts exact equality, not subset equality.

---

## Consumer contract (this phase)

The harness:

1. MUST parse every `XX.json` in `6502/v1/` whose `XX` is not in `skipList`.
2. MUST treat each case's `initial` as the complete starting state. Addresses not listed in `initial.ram` are zero in the sparse adapter; no other pre-population is performed.
3. MUST execute exactly one `cpu.Step()`. No multi-step cases. No interrupts injected (this corpus does not cover RESET/IRQ/NMI sequences — those remain in `interrupts_test.go`).
4. MUST assert against `final` and `cycles` exactly. No tolerance, no near-match, no swallowed mismatches.
5. MUST NOT skip a documented opcode silently. Every opcode not in `skipList` MUST be exercised; failures MUST surface.

---

## What this phase does NOT consume from upstream

- `nes6502/` (NES 2A03, no decimal mode).
- `wdc65c02/` (CMOS variant).
- `65816/` (16-bit variant).
- The `v2/` test-corpus directories if they exist at the pinned SHA — only `6502/v1/` is in scope. Upstream is stable on `v1`; `v2` is intentionally OOS to avoid open-set scope creep.
